package handler

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	usermodel "github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handler) ListRoles(c *gin.Context) {
	var roles []usermodel.Role
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
	var r usermodel.Role
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
	r := usermodel.Role{Name: req.Name, Code: code, Description: req.Description, Status: 1, CreateTime: now, UpdateTime: now}
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
		if err := tx.Model(&usermodel.Role{}).Where("id = ?", id).Updates(updates).Error; err != nil {
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
		if err := tx.Where("role_id = ?", id).Delete(&usermodel.RolePermission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", id).Delete(&usermodel.UserRole{}).Error; err != nil {
			return err
		}
		return tx.Delete(&usermodel.Role{}, id).Error
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
