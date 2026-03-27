// Package deployment 提供部署管理服务的审计处理器。
//
// 本文件包含审计日志的 HTTP 处理器实现。
package deployment

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// AuditHandler 是审计日志的 HTTP 处理器。
type AuditHandler struct {
	svcCtx *svc.ServiceContext
}

// NewAuditHandler 创建审计处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文
//
// 返回: AuditHandler 实例
func NewAuditHandler(svcCtx *svc.ServiceContext) *AuditHandler {
	return &AuditHandler{svcCtx: svcCtx}
}

// listAuditLogsReq 是获取审计日志列表的请求参数。
type listAuditLogsReq struct {
	Page         int    `form:"page" binding:"omitempty,min=1"`        // 页码
	PageSize     int    `form:"page_size" binding:"omitempty,min=1,max=100"` // 每页数量
	ActionType   string `form:"action_type"`                           // 操作类型
	ResourceType string `form:"resource_type"`                         // 资源类型
}

// ListAuditLogs 获取审计日志列表。
//
// @Summary 获取审计日志列表
// @Description 获取部署相关的审计日志，支持按操作类型和资源类型筛选
// @Tags 审计管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param action_type query string false "操作类型"
// @Param resource_type query string false "资源类型"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/audit-logs [get]
func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	var req listAuditLogsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		h.respondBadRequest(c, err)
		return
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	ctx := c.Request.Context()
	logs, total, err := h.listAuditLogs(ctx, req)
	if err != nil {
		h.respondInternalError(c, err)
		return
	}

	h.respondOK(c, gin.H{"list": logs, "total": total})
}

// listAuditLogs 查询审计日志列表。
//
// 参数:
//   - ctx: 上下文
//   - req: 查询请求参数
//
// 返回: 审计日志列表和总数
func (h *AuditHandler) listAuditLogs(ctx context.Context, req listAuditLogsReq) ([]model.AuditLog, int64, error) {
	query := h.svcCtx.DB.WithContext(ctx).Model(&model.AuditLog{})

	if req.ActionType != "" {
		query = query.Where("action_type = ?", req.ActionType)
	}
	if req.ResourceType != "" {
		query = query.Where("resource_type = ?", req.ResourceType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []model.AuditLog
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("id desc").Offset(offset).Limit(req.PageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// CreateAuditLog 创建审计日志（内部方法）。
//
// 参数:
//   - ctx: 上下文
//   - db: 服务上下文
//   - actionType: 操作类型
//   - resourceType: 资源类型
//   - resourceID: 资源 ID
//   - actorID: 操作人 ID
//   - actorName: 操作人名称
//   - detail: 详情
//
// 返回: 错误信息
func CreateAuditLog(ctx context.Context, db *svc.ServiceContext, actionType, resourceType string, resourceID, actorID uint, actorName string, detail map[string]interface{}) error {
	log := model.AuditLog{
		ActionType:   actionType,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ActorID:      actorID,
		ActorName:    actorName,
		Detail:       detail,
	}
	return db.DB.WithContext(ctx).Create(&log).Error
}

// respondOK 返回成功响应。
//
// 参数:
//   - c: Gin 上下文
//   - data: 响应数据
func (h *AuditHandler) respondOK(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{"code": 1000, "msg": "success", "data": data})
}

// respondBadRequest 返回参数错误响应。
//
// 参数:
//   - c: Gin 上下文
//   - err: 错误信息
func (h *AuditHandler) respondBadRequest(c *gin.Context, err error) {
	c.JSON(400, gin.H{"code": xcode.ParamError, "msg": err.Error()})
}

// respondInternalError 返回服务器错误响应。
//
// 参数:
//   - c: Gin 上下文
//   - err: 错误信息
func (h *AuditHandler) respondInternalError(c *gin.Context, err error) {
	c.JSON(500, gin.H{"code": xcode.ServerError, "msg": err.Error()})
}
