// Package ai 提供 AI 编排服务的 HTTP 处理器实现。
//
// 本文件实现 AI 对话、会话管理和流程恢复等核心功能的 HTTP 接口。
// 主要职责包括：
//   - SSE 流式对话响应
//   - 会话持久化和查询
//   - 审批流程恢复
//   - 用户反馈收集
package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	v1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	coreai "github.com/cy77cc/OpsPilot/internal/ai"
	"github.com/cy77cc/OpsPilot/internal/ai/events"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
	aistate "github.com/cy77cc/OpsPilot/internal/ai/state"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HTTPHandler AI 服务的 HTTP 处理器。
type HTTPHandler struct {
	svcCtx       *svc.ServiceContext     // 服务上下文
	sessions     *aistate.SessionState   // 会话状态管理
	chatStore    *aistate.ChatStore      // 聊天记录存储
	orchestrator *coreai.Orchestrator    // AI 编排器
}

// approvalResponseRequest 审批响应请求结构。
type approvalResponseRequest struct {
	CheckpointID string `json:"checkpoint_id,omitempty"` // 检查点 ID（兼容旧版）
	SessionID    string `json:"session_id,omitempty"`    // 会话 ID
	PlanID       string `json:"plan_id,omitempty"`       // 计划 ID
	StepID       string `json:"step_id,omitempty"`       // 步骤 ID
	Target       string `json:"target,omitempty"`        // 目标标识
	Approved     bool   `json:"approved"`                // 是否批准
	Reason       string `json:"reason,omitempty"`        // 原因说明
}

// updateSessionTitleRequest 更新会话标题请求。
type updateSessionTitleRequest struct {
	Title string `json:"title" binding:"required"` // 新标题
}

// branchSessionRequest 分支会话请求。
type branchSessionRequest struct {
	Title string `json:"title,omitempty"` // 新会话标题
}

// feedbackRequest 用户反馈请求。
type feedbackRequest struct {
	SessionID   string `json:"session_id,omitempty"`   // 会话 ID
	Namespace   string `json:"namespace,omitempty"`     // 命名空间
	IsEffective bool   `json:"is_effective"`            // 是否有效反馈
	Comment     string `json:"comment,omitempty"`       // 反馈评论
	Question    string `json:"question,omitempty"`      // 问题（用于知识提取）
	Answer      string `json:"answer,omitempty"`        // 答案（用于知识提取）
}

// NewHTTPHandler 创建新的 HTTP 处理器实例。
func NewHTTPHandler(svcCtx *svc.ServiceContext) *HTTPHandler {
	sessionState := aistate.NewSessionState(svcCtx.Rdb, "ai:session:")
	executionStore := runtime.NewExecutionStore(svcCtx.Rdb, "ai:execution:")
	return &HTTPHandler{
		svcCtx:    svcCtx,
		sessions:  sessionState,
		chatStore: aistate.NewChatStore(svcCtx.DB),
		orchestrator: coreai.NewOrchestrator(sessionState, executionStore, common.PlatformDeps{
			DB: svcCtx.DB,
		}),
	}
}

// Chat 处理对话请求，返回 SSE 流式响应。
//
// 支持 AI 编排的完整流程：改写、规划、执行、总结。
func (h *HTTPHandler) Chat(c *gin.Context) {
	var req v1.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		httpx.Fail(c, xcode.ServerError, "streaming is not supported")
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	rollout := coreai.CurrentRolloutConfig()
	c.Writer.Header().Set("X-AI-Runtime-Mode", rollout.RuntimeMode())
	c.Writer.Header().Set("X-AI-Compatibility-Enabled", boolHeaderValue(rollout.CompatibilityEnabled()))
	c.Writer.Header().Set("X-AI-Model-First-Enabled", boolHeaderValue(rollout.ModelFirstEnabled()))
	c.Status(http.StatusOK)

	scene := normalizedScene(req.Context["scene"])
	recorder := newChatRecorder(h.chatStore, httpx.UIDFromCtx(c), scene, req.Message)
	emit := func(evt coreai.StreamEvent) bool {
		payload := evt.Data
		if evt.Type == events.Meta {
			payload = attachRolloutMetadata(cloneMap(payload), rollout)
		}
		if recorder != nil {
			payload = cloneMap(evt.Data)
			if evt.Type == events.Meta {
				payload = attachRolloutMetadata(payload, rollout)
			}
			recorder.HandleEvent(c.Request.Context(), evt.Type, payload)
			if evt.Type == events.Done {
				if sessionPayload := recorder.SessionPayload(c.Request.Context()); sessionPayload != nil {
					payload["session"] = sessionPayload
				}
			}
		}
		return writeSSE(c, flusher, string(evt.Type), payload)
	}

	runReq := coreai.RunRequest{
		SessionID:      req.SessionID,
		Message:        req.Message,
		RuntimeContext: h.normalizeRuntimeContext(c, req.Context),
	}
	if err := h.orchestrator.Run(c.Request.Context(), runReq, emit); err != nil {
		writeSSE(c, flusher, "error", map[string]any{
			"message": err.Error(),
		})
	}
}

// ResumeStep 处理步骤恢复请求。
func (h *HTTPHandler) ResumeStep(c *gin.Context) {
	h.handleResume(c, false)
}

// handleResume 统一处理恢复请求。
func (h *HTTPHandler) handleResume(c *gin.Context, legacyADK bool) {
	var req approvalResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	res, err := h.orchestrator.Resume(c.Request.Context(), buildResumeRequest(req))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, buildResumeResponse(res, legacyADK))
}

// ResumeADKApproval 处理 ADK 审批恢复请求（兼容旧版）。
func (h *HTTPHandler) ResumeADKApproval(c *gin.Context) {
	h.handleResume(c, true)
}

// buildResumeRequest 构建恢复请求对象。
func buildResumeRequest(req approvalResponseRequest) coreai.ResumeRequest {
	return coreai.ResumeRequest{
		SessionID: req.SessionID,
		PlanID:    req.PlanID,
		StepID:    firstNonEmpty(req.StepID, req.Target, req.CheckpointID),
		Target:    firstNonEmpty(req.Target, req.CheckpointID),
		Approved:  req.Approved,
		Reason:    req.Reason,
	}
}

func buildResumeResponse(res *coreai.ResumeResult, legacyADK bool) gin.H {
	if res == nil {
		res = &coreai.ResumeResult{}
	}
	payload := gin.H{
		"resumed":           res.Resumed,
		"interrupted":       res.Interrupted,
		"sessionId":         res.SessionID,
		"session_id":        res.SessionID,
		"plan_id":           res.PlanID,
		"step_id":           res.StepID,
		"message":           res.Message,
		"status":            res.Status,
		"interrupt_error":   "",
		"approval_required": false,
	}
	if legacyADK {
		payload["deprecated"] = true
		payload["compat_mode"] = "legacy_adk_resume"
		payload["message"] = legacyResumeMessage(res.Message)
	}
	return payload
}

func legacyResumeMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "旧版恢复请求已按 step resume 语义处理。"
	}
	if strings.Contains(message, "/api/v1/ai/resume/step") {
		return message
	}
	return message + " 请迁移到 /api/v1/ai/resume/step，并使用 session_id + plan_id + step_id。"
}

func (h *HTTPHandler) SubmitFeedback(c *gin.Context) {
	var req feedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":         uuid.NewString(),
		"source":     "feedback",
		"namespace":  strings.TrimSpace(req.Namespace),
		"question":   strings.TrimSpace(req.Question),
		"answer":     strings.TrimSpace(req.Answer),
		"created_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *HTTPHandler) ListSessions(c *gin.Context) {
	rows, err := h.chatStore.ListSessions(c.Request.Context(), httpx.UIDFromCtx(c), normalizedScene(c.Query("scene")))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	out := make([]v1.AISession, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAPISession(row, false))
	}
	httpx.OK(c, out)
}

func (h *HTTPHandler) CurrentSession(c *gin.Context) {
	row, err := h.chatStore.CurrentSession(c.Request.Context(), httpx.UIDFromCtx(c), normalizedScene(c.Query("scene")), true)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if row == nil {
		httpx.OK(c, nil)
		return
	}
	httpx.OK(c, toAPISession(*row, true))
}

func (h *HTTPHandler) GetSession(c *gin.Context) {
	row, err := h.chatStore.GetSession(c.Request.Context(), httpx.UIDFromCtx(c), strings.TrimSpace(c.Query("scene")), c.Param("id"), true)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if row == nil {
		httpx.Fail(c, xcode.NotFound, "session not found")
		return
	}
	httpx.OK(c, toAPISession(*row, true))
}

func (h *HTTPHandler) BranchSession(c *gin.Context) {
	var req branchSessionRequest
	_ = c.ShouldBindJSON(&req)
	row, err := h.chatStore.Clone(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("id"), uuid.NewString(), req.Title)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if row == nil {
		httpx.Fail(c, xcode.NotFound, "session not found")
		return
	}
	httpx.OK(c, toAPISession(*row, true))
}

func (h *HTTPHandler) UpdateSessionTitle(c *gin.Context) {
	var req updateSessionTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if err := h.chatStore.UpdateTitle(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("id"), req.Title); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	row, err := h.chatStore.GetSession(c.Request.Context(), httpx.UIDFromCtx(c), "", c.Param("id"), true)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if row == nil {
		httpx.Fail(c, xcode.NotFound, "session not found")
		return
	}
	httpx.OK(c, toAPISession(*row, true))
}

func (h *HTTPHandler) DeleteSession(c *gin.Context) {
	if err := h.chatStore.Delete(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("id")); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, nil)
}

func (h *HTTPHandler) normalizeRuntimeContext(c *gin.Context, raw map[string]any) coreai.RuntimeContext {
	ctx := coreai.RuntimeContext{
		Route:       c.FullPath(),
		ProjectID:   strings.TrimSpace(c.GetHeader("X-Project-ID")),
		CurrentPage: c.Request.Referer(),
		UserContext: map[string]any{
			"uid":   httpx.UIDFromCtx(c),
			"admin": httpx.IsAdmin(h.svcCtx.DB, httpx.UIDFromCtx(c)),
		},
		Metadata: map[string]any{},
	}
	if scene, ok := raw["scene"].(string); ok {
		ctx.Scene = strings.TrimSpace(scene)
	}
	if route, ok := raw["route"].(string); ok && strings.TrimSpace(route) != "" {
		ctx.Route = strings.TrimSpace(route)
	}
	if page, ok := raw["current_page"].(string); ok && strings.TrimSpace(page) != "" {
		ctx.CurrentPage = strings.TrimSpace(page)
	}
	if page, ok := raw["currentPage"].(string); ok && strings.TrimSpace(page) != "" {
		ctx.CurrentPage = strings.TrimSpace(page)
	}
	if resources, ok := raw["selected_resources"].([]any); ok {
		ctx.SelectedResources = toSelectedResources(resources)
	}
	if resources, ok := raw["selectedResources"].([]any); ok && len(ctx.SelectedResources) == 0 {
		ctx.SelectedResources = toSelectedResources(resources)
	}
	for key, value := range raw {
		ctx.Metadata[key] = value
	}
	return ctx
}

func toSelectedResources(items []any) []coreai.SelectedResource {
	out := make([]coreai.SelectedResource, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, coreai.SelectedResource{
			Type: stringify(row["type"]),
			ID:   stringify(firstNonNil(row["id"], row["value"])),
			Name: stringify(firstNonNil(row["name"], row["label"])),
		})
	}
	return out
}

func toAPISession(snapshot aistate.ChatSessionRecord, includeMessages bool) v1.AISession {
	msgs := make([]map[string]any, 0, len(snapshot.Messages))
	if includeMessages {
		for _, msg := range snapshot.Messages {
			payload := map[string]any{
				"id":        msg.ID,
				"role":      msg.Role,
				"content":   msg.Content,
				"thinking":  msg.Thinking,
				"status":    msg.Status,
				"timestamp": msg.CreatedAt.Format(time.RFC3339),
			}
			if msg.TraceID != "" {
				payload["traceId"] = msg.TraceID
			}
			if len(msg.ThoughtChain) > 0 {
				payload["thoughtChain"] = msg.ThoughtChain
			}
			if len(msg.Recommendations) > 0 {
				payload["recommendations"] = msg.Recommendations
			}
			if len(msg.SummaryOutput) > 0 {
				payload["summaryOutput"] = msg.SummaryOutput
			}
			if len(msg.RawEvidence) > 0 {
				payload["rawEvidence"] = msg.RawEvidence
			}
			msgs = append(msgs, map[string]any{
				"id":              payload["id"],
				"role":            payload["role"],
				"content":         payload["content"],
				"thinking":        payload["thinking"],
				"status":          payload["status"],
				"traceId":         payload["traceId"],
				"thoughtChain":    payload["thoughtChain"],
				"recommendations": payload["recommendations"],
				"summaryOutput":   payload["summaryOutput"],
				"rawEvidence":     payload["rawEvidence"],
				"timestamp":       payload["timestamp"],
			})
		}
	}
	return v1.AISession{
		ID:        snapshot.ID,
		Scene:     snapshot.Scene,
		Title:     firstNonEmpty(snapshot.Title, "新对话"),
		Messages:  msgs,
		CreatedAt: snapshot.CreatedAt,
		UpdatedAt: snapshot.UpdatedAt,
	}
}

func writeSSE(c *gin.Context, flusher http.Flusher, event string, payload map[string]any) bool {
	data, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return false
	}
	flusher.Flush()
	return true
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func boolHeaderValue(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func attachRolloutMetadata(payload map[string]any, rollout coreai.RolloutConfig) map[string]any {
	if payload == nil {
		payload = map[string]any{}
	}
	payload["runtime_mode"] = rollout.RuntimeMode()
	payload["model_first_enabled"] = rollout.ModelFirstEnabled()
	payload["compatibility_enabled"] = rollout.CompatibilityEnabled()
	return payload
}

func normalizedScene(raw any) string {
	if text, ok := raw.(string); ok {
		if strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return "global"
}

func stringify(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
