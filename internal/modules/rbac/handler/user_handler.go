package handler

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	usermodel "github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handler) MyPermissions(c *gin.Context) {
	uid, ok := c.Get("uid")
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	userID := httpx.ToUint64(uid)
	perms, _ := h.fetchPermissionsByUserID(userID)
	if httpx.IsAdmin(h.db, userID) {
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
	if httpx.IsAdmin(h.db, userID) {
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
	var users []usermodel.User
	if err := h.db.Find(&users).Error; err != nil {
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
	var u usermodel.User
	if err := h.db.First(&u, id).Error; err != nil {
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
	u := usermodel.User{Username: req.Username, PasswordHash: hashed, Email: req.Email, CreateTime: now, UpdateTime: now, Status: toStatusInt(req.Status)}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
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

	if err := h.db.Transaction(func(tx *gorm.DB) error {
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
		if err := tx.Model(&usermodel.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
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
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&usermodel.UserRole{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&usermodel.User{}, id).Error; err != nil {
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
