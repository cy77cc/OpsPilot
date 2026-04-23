package handler

import (
	"strconv"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	usermodel "github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/gin-gonic/gin"
)

func (h *Handler) ListPermissions(c *gin.Context) {
	var permissions []usermodel.Permission
	if err := h.db.Find(&permissions).Error; err != nil {
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
	var p usermodel.Permission
	if err := h.db.First(&p, id).Error; err != nil {
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
