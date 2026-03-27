// Package deployment 提供部署管理服务的策略处理器。
//
// 本文件包含部署策略的 CRUD 操作 HTTP 处理器实现。
package deployment

import (
	"context"
	"strconv"
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// PolicyHandler 是策略管理的 HTTP 处理器。
type PolicyHandler struct {
	svcCtx *svc.ServiceContext
}

// NewPolicyHandler 创建策略处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文
//
// 返回: PolicyHandler 实例
func NewPolicyHandler(svcCtx *svc.ServiceContext) *PolicyHandler {
	return &PolicyHandler{svcCtx: svcCtx}
}

// listPoliciesReq 是获取策略列表的请求参数。
type listPoliciesReq struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`     // 页码
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"` // 每页数量
	Type     string `form:"type"`                               // 策略类型
	TargetID uint   `form:"target_id"`                          // 目标 ID
}

// createPolicyReq 是创建策略的请求参数。
type createPolicyReq struct {
	Name     string                 `json:"name" binding:"required"`  // 策略名称
	Type     string                 `json:"type" binding:"required,oneof=traffic resilience access slo"` // 策略类型
	TargetID uint                   `json:"target_id"`                // 目标 ID
	Config   map[string]interface{} `json:"config"`                   // 策略配置
	Enabled  bool                   `json:"enabled"`                  // 是否启用
}

// updatePolicyReq 是更新策略的请求参数。
type updatePolicyReq struct {
	Name    string                 `json:"name"`    // 策略名称
	Type    string                 `json:"type"`    // 策略类型
	Config  map[string]interface{} `json:"config"`  // 策略配置
	Enabled *bool                  `json:"enabled"` // 是否启用
}

// ListPolicies 获取策略列表。
//
// @Summary 获取策略列表
// @Description 获取部署策略列表，支持按类型和目标筛选
// @Tags 策略管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param type query string false "策略类型"
// @Param target_id query int false "目标 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/policies [get]
func (h *PolicyHandler) ListPolicies(c *gin.Context) {
	var req listPoliciesReq
	if err := c.ShouldBindQuery(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	ctx := c.Request.Context()
	policies, total, err := h.listPolicies(ctx, req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{"list": policies, "total": total})
}

// GetPolicy 获取策略详情。
//
// @Summary 获取策略详情
// @Description 根据ID获取策略的详细信息
// @Tags 策略管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "策略 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/policies/{id} [get]
func (h *PolicyHandler) GetPolicy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}

	ctx := c.Request.Context()
	policy, err := h.getPolicy(ctx, uint(id))
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "policy not found")
		return
	}

	httpx.OK(c, policy)
}

// CreatePolicy 创建策略。
//
// @Summary 创建策略
// @Description 创建新的部署策略
// @Tags 策略管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body createPolicyReq true "策略信息"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/policies [post]
func (h *PolicyHandler) CreatePolicy(c *gin.Context) {
	var req createPolicyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	ctx := c.Request.Context()
	policy, err := h.createPolicy(ctx, req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, policy)
}

// UpdatePolicy 更新策略。
//
// @Summary 更新策略
// @Description 更新指定策略的信息
// @Tags 策略管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "策略 ID"
// @Param body body updatePolicyReq true "策略信息"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/policies/{id} [put]
func (h *PolicyHandler) UpdatePolicy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}

	var req updatePolicyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	ctx := c.Request.Context()
	policy, err := h.updatePolicy(ctx, uint(id), req)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "policy not found")
		return
	}

	httpx.OK(c, policy)
}

// DeletePolicy 删除策略。
//
// @Summary 删除策略
// @Description 删除指定的策略
// @Tags 策略管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "策略 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/policies/{id} [delete]
func (h *PolicyHandler) DeletePolicy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}

	ctx := c.Request.Context()
	if err := h.deletePolicy(ctx, uint(id)); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{"message": "deleted"})
}

// listPolicies 查询策略列表。
//
// 参数:
//   - ctx: 上下文
//   - req: 查询请求参数
//
// 返回: 策略列表和总数
func (h *PolicyHandler) listPolicies(ctx context.Context, req listPoliciesReq) ([]model.Policy, int64, error) {
	query := h.svcCtx.DB.WithContext(ctx).Model(&model.Policy{})

	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}
	if req.TargetID > 0 {
		query = query.Where("target_id = ?", req.TargetID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var policies []model.Policy
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("id desc").Offset(offset).Limit(req.PageSize).Find(&policies).Error; err != nil {
		return nil, 0, err
	}

	return policies, total, nil
}

// getPolicy 根据ID获取策略。
//
// 参数:
//   - ctx: 上下文
//   - id: 策略 ID
//
// 返回: 策略对象
func (h *PolicyHandler) getPolicy(ctx context.Context, id uint) (*model.Policy, error) {
	var policy model.Policy
	if err := h.svcCtx.DB.WithContext(ctx).First(&policy, id).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

// createPolicy 创建策略记录。
//
// 参数:
//   - ctx: 上下文
//   - req: 创建请求参数
//
// 返回: 创建的策略对象
func (h *PolicyHandler) createPolicy(ctx context.Context, req createPolicyReq) (*model.Policy, error) {
	policy := model.Policy{
		Name:     req.Name,
		Type:     req.Type,
		TargetID: req.TargetID,
		Config:   req.Config,
		Enabled:  req.Enabled,
	}

	if err := h.svcCtx.DB.WithContext(ctx).Create(&policy).Error; err != nil {
		return nil, err
	}

	return &policy, nil
}

// updatePolicy 更新策略记录。
//
// 参数:
//   - ctx: 上下文
//   - id: 策略 ID
//   - req: 更新请求参数
//
// 返回: 更新后的策略对象
func (h *PolicyHandler) updatePolicy(ctx context.Context, id uint, req updatePolicyReq) (*model.Policy, error) {
	var policy model.Policy
	if err := h.svcCtx.DB.WithContext(ctx).First(&policy, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	if req.Config != nil {
		updates["config"] = req.Config
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	if err := h.svcCtx.DB.WithContext(ctx).Model(&policy).Updates(updates).Error; err != nil {
		return nil, err
	}

	return h.getPolicy(ctx, id)
}

// deletePolicy 删除策略记录。
//
// 参数:
//   - ctx: 上下文
//   - id: 策略 ID
//
// 返回: 错误信息
func (h *PolicyHandler) deletePolicy(ctx context.Context, id uint) error {
	return h.svcCtx.DB.WithContext(ctx).Delete(&model.Policy{}, id).Error
}
