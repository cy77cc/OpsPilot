// Package handler 提供 RBAC 服务的 HTTP 处理器。
//
// 本文件实现基于角色的访问控制 (RBAC) 相关的 HTTP 接口，包括：
//   - 用户管理: CRUD 操作
//   - 角色管理: CRUD 操作
//   - 权限管理: 查询操作
//   - 权限检查: 单点权限验证
//   - 迁移事件记录: 审计日志
package handler

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/utils"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler 是 RBAC 服务的 HTTP 处理器。
//
// 依赖:
//   - svcCtx: 服务上下文，包含数据库、Casbin 执行器等依赖
type Handler struct {
	svcCtx *svc.ServiceContext
}

// NewHandler 创建 RBAC 处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文
//
// 返回: RBAC 处理器实例
func NewHandler(svcCtx *svc.ServiceContext) *Handler { return &Handler{svcCtx: svcCtx} }

// codeValidationError 是代码验证错误。
//
// 用于在同步角色/权限时报告无效的代码值。
type codeValidationError struct {
	field string   // 字段名 (roles/permissions)
	codes []string // 无效的代码列表
}

// Error 实现错误接口。
func (e *codeValidationError) Error() string {
	return fmt.Sprintf("invalid %s values: %s", e.field, strings.Join(e.codes, ","))
}

// MyPermissions 获取当前用户权限列表。
//
// @Summary 获取我的权限
// @Description 获取当前登录用户的所有权限代码列表
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=[]string}
// @Failure 401 {object} httpx.Response
// @Router /rbac/me/permissions [get]
func (h *Handler) MyPermissions(c *gin.Context) {
	uid, ok := c.Get("uid")
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	userID := httpx.ToUint64(uid)
	perms, _ := h.fetchPermissionsByUserID(userID)
	if httpx.IsAdmin(h.svcCtx.DB, userID) {
		perms = mergePermissions(perms, adminPermissionSet()...)
	}
	httpx.OK(c, perms)
}

// Check 检查权限。
//
// @Summary 检查权限
// @Description 检查当前用户是否拥有指定资源和操作的权限
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body object{resource=string,action=string} true "权限检查请求"
// @Success 200 {object} httpx.Response{data=object{hasPermission=bool}}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /rbac/check [post]
func (h *Handler) Check(c *gin.Context) {
	var req struct{ Resource, Action string }
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	code := req.Resource + ":" + req.Action
	uid, _ := c.Get("uid")
	userID := httpx.ToUint64(uid)
	perms, _ := h.fetchPermissionsByUserID(userID)
	if httpx.IsAdmin(h.svcCtx.DB, userID) {
		perms = mergePermissions(perms, adminPermissionSet()...)
	}
	has := hasPermission(perms, code, req.Resource)
	httpx.OK(c, gin.H{"hasPermission": has})
}

// ListUsers 获取用户列表。
//
// @Summary 获取用户列表
// @Description 获取系统中所有用户及其角色信息
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=object{list=[]object,total=int}}
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /rbac/users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	var users []model.User
	if err := h.svcCtx.DB.Find(&users).Error; err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	list := make([]gin.H, 0, len(users))
	for _, u := range users {
		roles, _ := h.getRoleCodesByUserID(uint64(u.ID))
		list = append(list, gin.H{
			"id":        u.ID,
			"username":  u.Username,
			"name":      u.Username,
			"email":     u.Email,
			"roles":     roles,
			"status":    toStatusText(u.Status),
			"createdAt": time.Unix(u.CreateTime, 0),
			"updatedAt": time.Unix(u.UpdateTime, 0),
		})
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

// GetUser 获取用户详情。
//
// @Summary 获取用户详情
// @Description 根据 ID 获取用户详细信息及其角色
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "用户 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /rbac/users/{id} [get]
func (h *Handler) GetUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	var u model.User
	if err := h.svcCtx.DB.First(&u, id).Error; err != nil {
		httpx.Fail(c, xcode.NotFound, "user not found")
		return
	}
	roles, _ := h.getRoleCodesByUserID(id)
	httpx.OK(c, gin.H{
		"id":        u.ID,
		"username":  u.Username,
		"name":      u.Username,
		"email":     u.Email,
		"roles":     roles,
		"status":    toStatusText(u.Status),
		"createdAt": time.Unix(u.CreateTime, 0),
		"updatedAt": time.Unix(u.UpdateTime, 0),
	})
}

// CreateUser 创建用户。
//
// @Summary 创建用户
// @Description 创建新用户并分配角色
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body object{username=string,name=string,email=string,password=string,roles=[]string,status=string} true "用户创建请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /rbac/users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	var req struct {
		Username string   `json:"username" binding:"required"`
		Name     string   `json:"name"`
		Email    string   `json:"email"`
		Password string   `json:"password" binding:"required"`
		Roles    []string `json:"roles"`
		Status   string   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, "hash password failed")
		return
	}

	now := time.Now().Unix()
	u := model.User{Username: req.Username, PasswordHash: hashed, Email: req.Email, CreateTime: now, UpdateTime: now, Status: toStatusInt(req.Status)}
	if err := h.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&u).Error; err != nil {
			return err
		}
		return h.syncUserRolesTx(tx, uint64(u.ID), req.Roles)
	}); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	roles, _ := h.getRoleCodesByUserID(uint64(u.ID))
	httpx.OK(c, gin.H{
		"id":        u.ID,
		"username":  u.Username,
		"name":      u.Username,
		"email":     u.Email,
		"roles":     roles,
		"status":    toStatusText(u.Status),
		"createdAt": time.Unix(u.CreateTime, 0),
		"updatedAt": time.Unix(u.UpdateTime, 0),
	})
}

// UpdateUser 更新用户。
//
// @Summary 更新用户
// @Description 更新用户信息、密码或角色
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "用户 ID"
// @Param body body object{name=string,email=string,password=string,roles=[]string,status=string} true "用户更新请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /rbac/users/{id} [put]
func (h *Handler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	var req struct {
		Name     *string  `json:"name"`
		Email    *string  `json:"email"`
		Password *string  `json:"password"`
		Roles    []string `json:"roles"`
		Status   *string  `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	if err := h.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"update_time": time.Now().Unix()}
		if req.Email != nil {
			updates["email"] = strings.TrimSpace(*req.Email)
		}
		if req.Status != nil {
			updates["status"] = toStatusInt(*req.Status)
		}
		if req.Password != nil && strings.TrimSpace(*req.Password) != "" {
			hashed, err := utils.HashPassword(*req.Password)
			if err != nil {
				return err
			}
			updates["password_hash"] = hashed
		}
		if err := tx.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}
		if req.Roles != nil {
			if err := h.syncUserRolesTx(tx, id, req.Roles); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		var validationErr *codeValidationError
		if errors.As(err, &validationErr) {
			httpx.Fail(c, xcode.ParamError, validationErr.Error())
			return
		}
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}

	uid, _ := c.Get("uid")
	log.Printf("rbac update user actor=%d target=%d timestamp=%s", httpx.ToUint64(uid), id, time.Now().UTC().Format(time.RFC3339))
	h.GetUser(c)
}

// DeleteUser 删除用户。
//
// @Summary 删除用户
// @Description 删除指定用户及其角色关联
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "用户 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /rbac/users/{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	if err := h.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&model.UserRole{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.User{}, id).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

// ListRoles 获取角色列表。
//
// @Summary 获取角色列表
// @Description 获取系统中所有角色及其权限信息
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=object{list=[]object,total=int}}
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /rbac/roles [get]
func (h *Handler) ListRoles(c *gin.Context) {
	var roles []model.Role
	if err := h.svcCtx.DB.Find(&roles).Error; err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	list := make([]gin.H, 0, len(roles))
	for _, r := range roles {
		permissions, _ := h.getPermissionCodesByRoleID(uint64(r.ID))
		list = append(list, gin.H{"id": r.ID, "name": r.Name, "code": r.Code, "description": r.Description, "permissions": permissions, "createdAt": time.Unix(r.CreateTime, 0), "updatedAt": time.Unix(r.UpdateTime, 0)})
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

// GetRole 获取角色详情。
//
// @Summary 获取角色详情
// @Description 根据 ID 获取角色详细信息及其权限
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "角色 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /rbac/roles/{id} [get]
func (h *Handler) GetRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	var r model.Role
	if err := h.svcCtx.DB.First(&r, id).Error; err != nil {
		httpx.Fail(c, xcode.NotFound, "role not found")
		return
	}
	permissions, _ := h.getPermissionCodesByRoleID(id)
	httpx.OK(c, gin.H{"id": r.ID, "name": r.Name, "code": r.Code, "description": r.Description, "permissions": permissions, "createdAt": time.Unix(r.CreateTime, 0), "updatedAt": time.Unix(r.UpdateTime, 0)})
}

// CreateRole 创建角色。
//
// @Summary 创建角色
// @Description 创建新角色并分配权限
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body object{name=string,description=string,permissions=[]string} true "角色创建请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /rbac/roles [post]
func (h *Handler) CreateRole(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	now := time.Now().Unix()
	code := strings.TrimSpace(req.Name)
	r := model.Role{Name: req.Name, Code: code, Description: req.Description, Status: 1, CreateTime: now, UpdateTime: now}
	if err := h.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&r).Error; err != nil {
			return err
		}
		return h.syncRolePermissionsTx(tx, uint64(r.ID), req.Permissions)
	}); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	permissions, _ := h.getPermissionCodesByRoleID(uint64(r.ID))
	httpx.OK(c, gin.H{"id": r.ID, "name": r.Name, "code": r.Code, "description": r.Description, "permissions": permissions, "createdAt": time.Unix(r.CreateTime, 0), "updatedAt": time.Unix(r.UpdateTime, 0)})
}

// UpdateRole 更新角色。
//
// @Summary 更新角色
// @Description 更新角色名称、描述或权限
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "角色 ID"
// @Param body body object{name=string,description=string,permissions=[]string} true "角色更新请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /rbac/roles/{id} [put]
func (h *Handler) UpdateRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	var req struct {
		Name        *string  `json:"name"`
		Description *string  `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	if err := h.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"update_time": time.Now().Unix()}
		if req.Name != nil {
			updates["name"] = strings.TrimSpace(*req.Name)
			updates["code"] = strings.TrimSpace(*req.Name)
		}
		if req.Description != nil {
			updates["description"] = strings.TrimSpace(*req.Description)
		}
		if err := tx.Model(&model.Role{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}
		if req.Permissions != nil {
			if err := h.syncRolePermissionsTx(tx, id, req.Permissions); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		var validationErr *codeValidationError
		if errors.As(err, &validationErr) {
			httpx.Fail(c, xcode.ParamError, validationErr.Error())
			return
		}
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}

	uid, _ := c.Get("uid")
	log.Printf("rbac update role actor=%d target=%d timestamp=%s", httpx.ToUint64(uid), id, time.Now().UTC().Format(time.RFC3339))
	h.GetRole(c)
}

// DeleteRole 删除角色。
//
// @Summary 删除角色
// @Description 删除指定角色及其权限关联、用户关联
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "角色 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /rbac/roles/{id} [delete]
func (h *Handler) DeleteRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	if err := h.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", id).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", id).Delete(&model.UserRole{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Role{}, id).Error
	}); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

// ListPermissions 获取权限列表。
//
// @Summary 获取权限列表
// @Description 获取系统中所有权限
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=object{list=[]object,total=int}}
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /rbac/permissions [get]
func (h *Handler) ListPermissions(c *gin.Context) {
	var permissions []model.Permission
	if err := h.svcCtx.DB.Find(&permissions).Error; err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	list := make([]gin.H, 0, len(permissions))
	for _, p := range permissions {
		list = append(list, gin.H{"id": p.ID, "name": p.Name, "code": p.Code, "description": p.Description, "category": p.Resource, "createdAt": time.Unix(p.CreateTime, 0)})
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

// GetPermission 获取权限详情。
//
// @Summary 获取权限详情
// @Description 根据 ID 获取权限详细信息
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "权限 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /rbac/permissions/{id} [get]
func (h *Handler) GetPermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	var p model.Permission
	if err := h.svcCtx.DB.First(&p, id).Error; err != nil {
		httpx.Fail(c, xcode.NotFound, "permission not found")
		return
	}
	httpx.OK(c, gin.H{"id": p.ID, "name": p.Name, "code": p.Code, "description": p.Description, "category": p.Resource, "createdAt": time.Unix(p.CreateTime, 0)})
}

// RecordMigrationEvent 记录迁移事件。
//
// @Summary 记录迁移事件
// @Description 记录前端权限迁移相关事件用于审计
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body object{eventType=string,fromPath=string,toPath=string,action=string,status=string,durationMs=int} true "迁移事件请求"
// @Success 200 {object} httpx.Response{data=object{accepted=bool}}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /rbac/migration/events [post]
func (h *Handler) RecordMigrationEvent(c *gin.Context) {
	uid, ok := c.Get("uid")
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	var req struct {
		EventType  string `json:"eventType" binding:"required"`
		FromPath   string `json:"fromPath"`
		ToPath     string `json:"toPath"`
		Action     string `json:"action"`
		Status     string `json:"status"`
		DurationMs int64  `json:"durationMs"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	userID := httpx.ToUint64(uid)
	timestamp := time.Now().UTC().Format(time.RFC3339)
	log.Printf("rbac migration event=%s actor=%d from=%s to=%s action=%s status=%s duration_ms=%d timestamp=%s",
		strings.TrimSpace(req.EventType),
		userID,
		strings.TrimSpace(req.FromPath),
		strings.TrimSpace(req.ToPath),
		strings.TrimSpace(req.Action),
		strings.TrimSpace(req.Status),
		req.DurationMs,
		timestamp,
	)
	httpx.OK(c, gin.H{"accepted": true})
}

// fetchPermissionsByUserID 根据用户 ID 获取权限代码列表。
//
// 参数:
//   - userID: 用户 ID
//
// 返回:
//   - []string: 权限代码列表
//   - error: 查询失败时返回错误
func (h *Handler) fetchPermissionsByUserID(userID uint64) ([]string, error) {
	type row struct {
		Code string `gorm:"column:code"`
	}
	var rows []row
	err := h.svcCtx.DB.Table("permissions").Select("permissions.code").Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").Where("user_roles.user_id = ?", userID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Code)
	}
	return out, nil
}

// hasPermission 检查权限列表中是否包含指定权限。
//
// 参数:
//   - perms: 权限代码列表
//   - code: 目标权限代码 (如 "host:read")
//   - resource: 资源名称 (如 "host")
//
// 返回: 是否拥有权限
//
// 支持通配符匹配:
//   - "host:*" 匹配 host 资源的所有操作
//   - "*:*" 匹配所有资源的所有操作
func hasPermission(perms []string, code string, resource string) bool {
	resourceWildcard := resource + ":*"
	for _, p := range perms {
		if p == code || p == resourceWildcard || p == "*:*" {
			return true
		}
	}
	return false
}

// mergePermissions 合并权限列表并去重。
//
// 参数:
//   - base: 基础权限列表
//   - extras: 额外权限列表
//
// 返回: 合并后的权限列表 (已去重)
func mergePermissions(base []string, extras ...string) []string {
	seen := make(map[string]struct{}, len(base)+len(extras))
	merged := make([]string, 0, len(base)+len(extras))
	for _, p := range base {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		merged = append(merged, p)
	}
	for _, p := range extras {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		merged = append(merged, p)
	}
	return merged
}

// adminPermissionSet 返回管理员权限集合。
//
// 管理员拥有所有资源的完全访问权限。
//
// 返回: 管理员权限代码列表
func adminPermissionSet() []string {
	return []string{
		"*:*",
		"host:read", "host:write", "host:*",
		"task:read", "task:write", "task:*",
		"kubernetes:read", "kubernetes:write", "kubernetes:*",
		"monitoring:read", "monitoring:write", "monitoring:*",
		"config:read", "config:write", "config:*",
		"rbac:read", "rbac:write", "rbac:*",
		"automation:*",
		"cicd:*",
		"cmdb:*",
	}
}

// getRoleCodesByUserID 根据用户 ID 获取角色代码列表。
//
// 参数:
//   - userID: 用户 ID
//
// 返回:
//   - []string: 角色代码列表 (已去重)
//   - error: 查询失败时返回错误
func (h *Handler) getRoleCodesByUserID(userID uint64) ([]string, error) {
	type row struct {
		Code string `gorm:"column:code"`
	}
	var rows []row
	err := h.svcCtx.DB.Table("roles").
		Select("roles.code").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		code := strings.TrimSpace(r.Code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out, nil
}

// getPermissionCodesByRoleID 根据角色 ID 获取权限代码列表。
//
// 参数:
//   - roleID: 角色 ID
//
// 返回:
//   - []string: 权限代码列表 (已去重)
//   - error: 查询失败时返回错误
func (h *Handler) getPermissionCodesByRoleID(roleID uint64) ([]string, error) {
	type row struct {
		Code string `gorm:"column:code"`
	}
	var rows []row
	err := h.svcCtx.DB.Table("permissions").
		Select("permissions.code").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		code := strings.TrimSpace(r.Code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out, nil
}

// syncUserRolesTx 在事务中同步用户角色。
//
// 参数:
//   - tx: 数据库事务
//   - userID: 用户 ID
//   - roleCodes: 角色代码列表
//
// 返回:
//   - error: 同步失败时返回错误 (包括角色不存在的情况)
//
// 流程:
//  1. 删除用户现有角色关联
//  2. 验证角色代码是否有效
//  3. 创建新的用户角色关联
func (h *Handler) syncUserRolesTx(tx *gorm.DB, userID uint64, roleCodes []string) error {
	if err := tx.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
		return err
	}
	cleanCodes := make([]string, 0, len(roleCodes))
	seen := make(map[string]struct{}, len(roleCodes))
	for _, code := range roleCodes {
		v := strings.TrimSpace(code)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		cleanCodes = append(cleanCodes, v)
	}
	if len(cleanCodes) == 0 {
		return nil
	}
	var roles []model.Role
	if err := tx.Where("code IN ?", cleanCodes).Find(&roles).Error; err != nil {
		return err
	}
	if len(roles) != len(cleanCodes) {
		found := make(map[string]struct{}, len(roles))
		for _, role := range roles {
			found[strings.TrimSpace(role.Code)] = struct{}{}
		}
		missing := make([]string, 0)
		for _, code := range cleanCodes {
			if _, ok := found[code]; !ok {
				missing = append(missing, code)
			}
		}
		return &codeValidationError{field: "roles", codes: missing}
	}
	for _, role := range roles {
		if err := tx.Create(&model.UserRole{UserID: int64(userID), RoleID: int64(role.ID)}).Error; err != nil {
			return err
		}
	}
	return nil
}

// syncRolePermissionsTx 在事务中同步角色权限。
//
// 参数:
//   - tx: 数据库事务
//   - roleID: 角色 ID
//   - permissionCodes: 权限代码列表
//
// 返回:
//   - error: 同步失败时返回错误 (包括权限不存在的情况)
//
// 流程:
//  1. 删除角色现有权限关联
//  2. 验证权限代码是否有效
//  3. 创建新的角色权限关联
func (h *Handler) syncRolePermissionsTx(tx *gorm.DB, roleID uint64, permissionCodes []string) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; err != nil {
		return err
	}
	cleanCodes := make([]string, 0, len(permissionCodes))
	seen := make(map[string]struct{}, len(permissionCodes))
	for _, code := range permissionCodes {
		v := strings.TrimSpace(code)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		cleanCodes = append(cleanCodes, v)
	}
	if len(cleanCodes) == 0 {
		return nil
	}
	var perms []model.Permission
	if err := tx.Where("code IN ?", cleanCodes).Find(&perms).Error; err != nil {
		return err
	}
	if len(perms) != len(cleanCodes) {
		found := make(map[string]struct{}, len(perms))
		for _, permission := range perms {
			found[strings.TrimSpace(permission.Code)] = struct{}{}
		}
		missing := make([]string, 0)
		for _, code := range cleanCodes {
			if _, ok := found[code]; !ok {
				missing = append(missing, code)
			}
		}
		return &codeValidationError{field: "permissions", codes: missing}
	}
	for _, perm := range perms {
		if err := tx.Create(&model.RolePermission{RoleID: int64(roleID), PermissionID: int64(perm.ID)}).Error; err != nil {
			return err
		}
	}
	return nil
}

// toStatusText 将状态码转换为文本。
//
// 参数:
//   - status: 状态码 (0=禁用, 1=启用)
//
// 返回: 状态文本 ("disabled"/"active")
func toStatusText(status int8) string {
	if status == 1 {
		return "active"
	}
	return "disabled"
}

// toStatusInt 将状态文本转换为状态码。
//
// 参数:
//   - status: 状态文本 ("disabled"/"active")
//
// 返回: 状态码 (0=禁用, 1=启用)
func toStatusInt(status string) int8 {
	if strings.EqualFold(strings.TrimSpace(status), "disabled") {
		return 0
	}
	return 1
}
