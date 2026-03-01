package handler

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/cy77cc/k8s-manage/internal/utils"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct{ svcCtx *svc.ServiceContext }

func NewHandler(svcCtx *svc.ServiceContext) *Handler { return &Handler{svcCtx: svcCtx} }

type codeValidationError struct {
	field string
	codes []string
}

func (e *codeValidationError) Error() string {
	return fmt.Sprintf("invalid %s values: %s", e.field, strings.Join(e.codes, ","))
}

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

func hasPermission(perms []string, code string, resource string) bool {
	resourceWildcard := resource + ":*"
	for _, p := range perms {
		if p == code || p == resourceWildcard || p == "*:*" {
			return true
		}
	}
	return false
}

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

func toStatusText(status int8) string {
	if status == 1 {
		return "active"
	}
	return "disabled"
}

func toStatusInt(status string) int8 {
	if strings.EqualFold(strings.TrimSpace(status), "disabled") {
		return 0
	}
	return 1
}
