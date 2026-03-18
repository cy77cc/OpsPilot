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

type Handler struct {
	logic     *monitoringlogic.Logic
	svcCtx    *svc.ServiceContext
	ruleSync  *RuleSyncService
	webhookGW *NotificationGateway
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{
		logic:     monitoringlogic.NewLogic(svcCtx),
		svcCtx:    svcCtx,
		ruleSync:  NewRuleSyncService(svcCtx.DB),
		webhookGW: NewNotificationGateway(svcCtx),
	}
}

func (h *Handler) StartRuleSync() {
	rootCtx := runtimectx.WithServices(context.Background(), h.svcCtx)
	_, _ = h.ruleSync.SyncRules(rootCtx)
	h.ruleSync.StartPeriodic(rootCtx, 5*time.Minute)
}

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

func (h *Handler) EnableRule(c *gin.Context) {
	h.setRuleEnabled(c, true)
}

func (h *Handler) DisableRule(c *gin.Context) {
	h.setRuleEnabled(c, false)
}

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

func parseTime(raw string) (time.Time, error) {
	return time.Parse(time.RFC3339, raw)
}

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

func defaultIfEmpty(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return strings.TrimSpace(v)
}

func positiveOr(v, d int) int {
	if v > 0 {
		return v
	}
	return d
}
