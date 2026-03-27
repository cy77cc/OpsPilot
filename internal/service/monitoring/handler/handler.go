// Package handler 提供监控告警服务的 HTTP 处理器。
//
// 本文件包含监控模块的核心处理器实现，包括:
//   - 告警事件查询和管理
//   - 告警规则的 CRUD 操作
//   - 指标数据查询
//   - 通知渠道管理
//   - 告警投递记录查询
//   - Alertmanager Webhook 接收
package handler

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	monitoringlogic "github.com/cy77cc/OpsPilot/internal/service/monitoring/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// Handler 是监控服务的 HTTP 处理器。
//
// 聚合了告警规则同步、通知网关和业务逻辑层，
// 提供统一的监控告警 API 入口。
type Handler struct {
	logic     *monitoringlogic.Logic  // 业务逻辑层
	svcCtx    *svc.ServiceContext     // 服务上下文
	ruleSync  *RuleSyncService        // 规则同步服务
	webhookGW *NotificationGateway    // 通知网关
}

// NewHandler 创建监控服务处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接、配置等依赖
//
// 返回: 初始化完成的 Handler 实例
func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{
		logic:     monitoringlogic.NewLogic(svcCtx),
		svcCtx:    svcCtx,
		ruleSync:  NewRuleSyncService(svcCtx.DB),
		webhookGW: NewNotificationGateway(svcCtx),
	}
}

// StartRuleSync 启动规则同步服务。
//
// 执行初始规则同步，并启动定期同步定时任务。
// 默认每 5 分钟同步一次告警规则到 Prometheus。
func (h *Handler) StartRuleSync() {
	rootCtx := runtimectx.WithServices(context.Background(), h.svcCtx)
	_, _ = h.ruleSync.SyncRules(rootCtx)
	h.ruleSync.StartPeriodic(rootCtx, 5*time.Minute)
}

// ReceiveWebhook 接收 Alertmanager Webhook 请求。
//
// @Summary 接收 Alertmanager Webhook
// @Description 接收来自 Prometheus Alertmanager 的告警通知，自动创建或更新告警事件
// @Tags 监控告警
// @Accept json
// @Produce json
// @Param request body AlertmanagerWebhook true "Alertmanager Webhook 载荷"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /alerts/receiver [post]
func (h *Handler) ReceiveWebhook(c *gin.Context) {
	var req AlertmanagerWebhook
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	processed, err := h.webhookGW.HandleWebhook(c.Request.Context(), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"status":    "success",
		"processed": processed,
	})
}

// ListAlerts 获取告警事件列表。
//
// @Summary 获取告警事件列表
// @Description 分页查询告警事件，支持按严重级别和状态筛选
// @Tags 监控告警
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param severity query string false "严重级别 (critical/warning/info)"
// @Param status query string false "状态 (firing/resolved)"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /alerts [get]
func (h *Handler) ListAlerts(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:read") {
		return
	}
	alerts, total, err := h.logic.ListAlerts(
		c.Request.Context(),
		strings.TrimSpace(c.Query("severity")),
		strings.TrimSpace(c.Query("status")),
		intFromQuery(c, "page", 1),
		intFromQuery(c, "page_size", 20),
	)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": alerts, "total": total})
}

// ListRules 获取告警规则列表。
//
// @Summary 获取告警规则列表
// @Description 分页查询告警规则配置
// @Tags 监控告警
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(50)
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /alert-rules [get]
func (h *Handler) ListRules(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:read") {
		return
	}
	rules, total, err := h.logic.ListRules(c.Request.Context(), intFromQuery(c, "page", 1), intFromQuery(c, "page_size", 50))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": rules, "total": total})
}

// CreateRule 创建告警规则。
//
// @Summary 创建告警规则
// @Description 创建新的告警规则配置，并同步到 Prometheus
// @Tags 监控告警
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body object true "告警规则参数"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /alert-rules [post]
func (h *Handler) CreateRule(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:write") {
		return
	}
	var req struct {
		Name           string  `json:"name" binding:"required"`
		Metric         string  `json:"metric" binding:"required"`
		Operator       string  `json:"operator"`
		Threshold      float64 `json:"threshold"`
		Severity       string  `json:"severity"`
		Enabled        *bool   `json:"enabled"`
		WindowSec      int     `json:"window_sec"`
		GranularitySec int     `json:"granularity_sec"`
		DimensionsJSON string  `json:"dimensions_json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	rule, err := h.logic.CreateRule(c.Request.Context(), model.AlertRule{
		Name:           req.Name,
		Metric:         req.Metric,
		Operator:       defaultIfEmpty(req.Operator, "gt"),
		Threshold:      req.Threshold,
		Severity:       defaultIfEmpty(req.Severity, "warning"),
		Enabled:        req.Enabled == nil || *req.Enabled,
		WindowSec:      positiveOr(req.WindowSec, 3600),
		GranularitySec: positiveOr(req.GranularitySec, 60),
		DimensionsJSON: strings.TrimSpace(req.DimensionsJSON),
		Source:         "custom",
		Scope:          "global",
	})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if _, err := h.ruleSync.SyncRules(c.Request.Context()); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, rule)
}

// UpdateRule 更新告警规则。
//
// @Summary 更新告警规则
// @Description 更新指定 ID 的告警规则配置，并同步到 Prometheus
// @Tags 监控告警
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "规则 ID"
// @Param request body object true "更新参数"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /alert-rules/{id} [put]
func (h *Handler) UpdateRule(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:write") {
		return
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Name           string  `json:"name"`
		Operator       string  `json:"operator"`
		Threshold      float64 `json:"threshold"`
		Severity       string  `json:"severity"`
		Enabled        *bool   `json:"enabled"`
		WindowSec      int     `json:"window_sec"`
		GranularitySec int     `json:"granularity_sec"`
		DimensionsJSON *string `json:"dimensions_json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	payload := map[string]any{}
	if strings.TrimSpace(req.Name) != "" {
		payload["name"] = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.Operator) != "" {
		payload["operator"] = strings.TrimSpace(req.Operator)
	}
	if req.Threshold > 0 {
		payload["threshold"] = req.Threshold
	}
	if strings.TrimSpace(req.Severity) != "" {
		payload["severity"] = strings.TrimSpace(req.Severity)
	}
	if req.Enabled != nil {
		payload["enabled"] = *req.Enabled
	}
	if req.WindowSec > 0 {
		payload["window_sec"] = req.WindowSec
	}
	if req.GranularitySec > 0 {
		payload["granularity_sec"] = req.GranularitySec
	}
	if req.DimensionsJSON != nil {
		payload["dimensions_json"] = strings.TrimSpace(*req.DimensionsJSON)
	}
	rule, err := h.logic.UpdateRule(c.Request.Context(), uint(id), payload)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if _, err := h.ruleSync.SyncRules(c.Request.Context()); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, rule)
}

// EnableRule 启用告警规则。
//
// @Summary 启用告警规则
// @Description 启用指定 ID 的告警规则
// @Tags 监控告警
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "规则 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /alert-rules/{id}/enable [post]
func (h *Handler) EnableRule(c *gin.Context) {
	h.setRuleEnabled(c, true)
}

// DisableRule 禁用告警规则。
//
// @Summary 禁用告警规则
// @Description 禁用指定 ID 的告警规则
// @Tags 监控告警
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "规则 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /alert-rules/{id}/disable [post]
func (h *Handler) DisableRule(c *gin.Context) {
	h.setRuleEnabled(c, false)
}

// setRuleEnabled 设置告警规则的启用状态。
//
// 内部方法，用于启用或禁用告警规则。
//
// 参数:
//   - c: Gin 上下文
//   - enabled: 目标启用状态
func (h *Handler) setRuleEnabled(c *gin.Context, enabled bool) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:write") {
		return
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	rule, err := h.logic.SetRuleEnabled(c.Request.Context(), uint(id), enabled)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if _, err := h.ruleSync.SyncRules(c.Request.Context()); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, rule)
}

// SyncRules 手动同步告警规则。
//
// @Summary 同步告警规则
// @Description 将数据库中的告警规则同步到 Prometheus 配置文件
// @Tags 监控告警
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /alerts/rules/sync [post]
func (h *Handler) SyncRules(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:write") {
		return
	}
	n, err := h.ruleSync.SyncRules(c.Request.Context())
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"status":       "success",
		"synced_count": n,
		"synced_at":    time.Now().UTC(),
	})
}

// GetMetrics 查询指标数据。
//
// @Summary 查询指标数据
// @Description 从 Prometheus 查询指定指标的时间序列数据
// @Tags 监控告警
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param metric query string true "指标名称"
// @Param start_time query string false "开始时间 (RFC3339)"
// @Param end_time query string false "结束时间 (RFC3339)"
// @Param granularity_sec query int false "采样粒度 (秒)" default(60)
// @Param source query string false "数据来源"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /metrics [get]
func (h *Handler) GetMetrics(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:read") {
		return
	}
	metric := strings.TrimSpace(c.Query("metric"))
	if metric == "" {
		httpx.Fail(c, xcode.ParamError, "metric is required")
		return
	}
	start, err := parseTime(defaultIfEmpty(c.Query("start_time"), time.Now().Add(-24*time.Hour).Format(time.RFC3339)))
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid start_time")
		return
	}
	end, err := parseTime(defaultIfEmpty(c.Query("end_time"), time.Now().Format(time.RFC3339)))
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid end_time")
		return
	}
	out, err := h.logic.GetMetrics(c.Request.Context(), monitoringlogic.MetricQuery{
		Metric:         metric,
		Start:          start,
		End:            end,
		GranularitySec: intFromQuery(c, "granularity_sec", 60),
		Source:         strings.TrimSpace(c.Query("source")),
	})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, out)
}

// ListChannels 获取通知渠道列表。
//
// @Summary 获取通知渠道列表
// @Description 查询所有告警通知渠道配置
// @Tags 监控告警
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /alert-channels [get]
func (h *Handler) ListChannels(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:read") {
		return
	}
	items, err := h.logic.ListChannels(c.Request.Context())
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}

// CreateChannel 创建通知渠道。
//
// @Summary 创建通知渠道
// @Description 创建新的告警通知渠道配置
// @Tags 监控告警
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body object true "通知渠道参数"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /alert-channels [post]
func (h *Handler) CreateChannel(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:write") {
		return
	}
	var req struct {
		Name       string `json:"name" binding:"required"`
		Type       string `json:"type"`
		Provider   string `json:"provider"`
		Target     string `json:"target"`
		ConfigJSON string `json:"config_json"`
		Enabled    *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	item, err := h.logic.CreateChannel(c.Request.Context(), model.AlertNotificationChannel{
		Name:       strings.TrimSpace(req.Name),
		Type:       strings.TrimSpace(req.Type),
		Provider:   strings.TrimSpace(req.Provider),
		Target:     strings.TrimSpace(req.Target),
		ConfigJSON: strings.TrimSpace(req.ConfigJSON),
		Enabled:    req.Enabled == nil || *req.Enabled,
	})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, item)
}

// UpdateChannel 更新通知渠道。
//
// @Summary 更新通知渠道
// @Description 更新指定 ID 的通知渠道配置
// @Tags 监控告警
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "渠道 ID"
// @Param request body object true "更新参数"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /alert-channels/{id} [put]
func (h *Handler) UpdateChannel(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:write") {
		return
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Name       string  `json:"name"`
		Type       string  `json:"type"`
		Provider   string  `json:"provider"`
		Target     string  `json:"target"`
		ConfigJSON *string `json:"config_json"`
		Enabled    *bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	payload := map[string]any{}
	if strings.TrimSpace(req.Name) != "" {
		payload["name"] = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.Type) != "" {
		payload["type"] = strings.TrimSpace(req.Type)
	}
	if strings.TrimSpace(req.Provider) != "" {
		payload["provider"] = strings.TrimSpace(req.Provider)
	}
	if strings.TrimSpace(req.Target) != "" {
		payload["target"] = strings.TrimSpace(req.Target)
	}
	if req.ConfigJSON != nil {
		payload["config_json"] = strings.TrimSpace(*req.ConfigJSON)
	}
	if req.Enabled != nil {
		payload["enabled"] = *req.Enabled
	}
	item, err := h.logic.UpdateChannel(c.Request.Context(), uint(id), payload)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, item)
}

// ListDeliveries 获取告警投递记录列表。
//
// @Summary 获取告警投递记录列表
// @Description 分页查询告警通知投递记录
// @Tags 监控告警
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param alert_id query int false "告警 ID"
// @Param channel_type query string false "渠道类型"
// @Param status query string false "投递状态"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /alert-deliveries [get]
func (h *Handler) ListDeliveries(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:read") {
		return
	}
	alertID := uint(intFromQuery(c, "alert_id", 0))
	items, total, err := h.logic.ListDeliveries(
		c.Request.Context(),
		alertID,
		strings.TrimSpace(c.Query("channel_type")),
		strings.TrimSpace(c.Query("status")),
		intFromQuery(c, "page", 1),
		intFromQuery(c, "page_size", 20),
	)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": items, "total": total})
}

// parseTime 解析 RFC3339 格式的时间字符串。
//
// 参数:
//   - raw: RFC3339 格式的时间字符串
//
// 返回: 解析后的时间对象和可能的错误
func parseTime(raw string) (time.Time, error) {
	return time.Parse(time.RFC3339, raw)
}

// intFromQuery 从查询参数获取整数值。
//
// 参数:
//   - c: Gin 上下文
//   - key: 参数名
//   - def: 默认值
//
// 返回: 解析后的整数值，解析失败时返回默认值
func intFromQuery(c *gin.Context, key string, def int) int {
	v, err := strconv.Atoi(strings.TrimSpace(c.Query(key)))
	if err != nil || v < 0 {
		return def
	}
	if v == 0 {
		return def
	}
	return v
}

// defaultIfEmpty 返回非空字符串或默认值。
//
// 参数:
//   - v: 待检查的字符串
//   - d: 默认值
//
// 返回: 如果 v 为空则返回 d，否则返回 trim 后的 v
func defaultIfEmpty(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return strings.TrimSpace(v)
}

// positiveOr 返回正整数或默认值。
//
// 参数:
//   - v: 待检查的值
//   - d: 默认值
//
// 返回: 如果 v 大于 0 则返回 v，否则返回 d
func positiveOr(v, d int) int {
	if v > 0 {
		return v
	}
	return d
}
