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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	v1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	coreai "github.com/cy77cc/OpsPilot/internal/ai"
	"github.com/cy77cc/OpsPilot/internal/ai/events"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
	aistate "github.com/cy77cc/OpsPilot/internal/ai/state"
	aitools "github.com/cy77cc/OpsPilot/internal/ai/tools"
	approvaltools "github.com/cy77cc/OpsPilot/internal/ai/tools/approval"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/logger"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// HTTPHandler AI 服务的 HTTP 处理器。
type HTTPHandler struct {
	svcCtx       *svc.ServiceContext   // 服务上下文
	sessions     *aistate.SessionState // 会话状态管理
	chatStore    *aistate.ChatStore    // 聊天记录存储
	orchestrator aiRuntime             // AI 编排器
	registry     *aitools.Registry
	approvals    *runtime.ApprovalDecisionMaker
	summaries    *approvaltools.SummaryRenderer
	hintResolver *HintResolver
}

// aiRuntime 是对 Orchestrator 的最小接口抽象，便于单元测试时替换实现。
type aiRuntime interface {
	Run(ctx context.Context, req coreai.RunRequest, emit coreai.StreamEmitter) error
	Resume(ctx context.Context, req coreai.ResumeRequest) (*coreai.ResumeResult, error)
	ResumeStream(ctx context.Context, req coreai.ResumeRequest, emit coreai.StreamEmitter) (*coreai.ResumeResult, error)
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
	SessionID   string `json:"session_id,omitempty"` // 会话 ID
	Namespace   string `json:"namespace,omitempty"`  // 命名空间
	IsEffective bool   `json:"is_effective"`         // 是否有效反馈
	Comment     string `json:"comment,omitempty"`    // 反馈评论
	Question    string `json:"question,omitempty"`   // 问题（用于知识提取）
	Answer      string `json:"answer,omitempty"`     // 答案（用于知识提取）
}

// NewHTTPHandler 创建新的 HTTP 处理器实例。
func NewHTTPHandler(svcCtx *svc.ServiceContext) *HTTPHandler {
	sessionState := aistate.NewSessionState(svcCtx.Rdb, "ai:session:")
	executionStore := runtime.NewExecutionStore(svcCtx.Rdb, "ai:execution:")
	registry := aitools.NewRegistry(common.PlatformDeps{
		DB:         svcCtx.DB,
		Prometheus: svcCtx.Prometheus,
	})
	handler := &HTTPHandler{
		svcCtx:    svcCtx,
		sessions:  sessionState,
		chatStore: aistate.NewChatStore(svcCtx.DB),
		orchestrator: coreai.NewOrchestrator(sessionState, executionStore, common.PlatformDeps{
			DB:         svcCtx.DB,
			Prometheus: svcCtx.Prometheus,
		}),
		registry:     registry,
		summaries:    approvaltools.NewSummaryRenderer(),
		hintResolver: NewHintResolver(common.PlatformDeps{DB: svcCtx.DB, Prometheus: svcCtx.Prometheus}),
	}
	handler.approvals = runtime.NewApprovalDecisionMaker(runtime.ApprovalDecisionMakerOptions{
		ResolveScene: handler.resolveApprovalScene,
		LookupTool: func(name string) (runtime.ApprovalToolSpec, bool) {
			spec, ok := registry.Get(name)
			if !ok {
				return runtime.ApprovalToolSpec{}, false
			}
			return runtime.ApprovalToolSpec{
				Name:        spec.Name,
				DisplayName: spec.DisplayName,
				Description: spec.Description,
				Mode:        string(spec.Mode),
				Risk:        string(spec.Risk),
				Category:    spec.Category,
			}, true
		},
	})
	return handler
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
	c.Writer.Header().Set("X-AI-Runtime-Mode", h.runtimeModeHeader(rollout))
	c.Writer.Header().Set("X-AI-Compatibility-Enabled", boolHeaderValue(rollout.CompatibilityEnabled()))
	c.Writer.Header().Set("X-AI-Model-First-Enabled", boolHeaderValue(rollout.ModelFirstEnabled()))
	c.Writer.Header().Set("X-AI-Turn-Block-Streaming-Enabled", boolHeaderValue(rollout.TurnBlockStreamingEnabled()))
	c.Status(http.StatusOK)

	scene := normalizedScene(req.Context["scene"])
	recorder := newChatRecorder(h.chatStore, httpx.UIDFromCtx(c), scene, req.Message)
	if recorder == nil {
		if l := logger.L(); l != nil {
			l.Warn("ai chat recorder unavailable", logger.String("scene", scene))
		}
	}
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
	if err := h.runRuntime(c.Request.Context(), runReq, emit); err != nil {
		writeSSE(c, flusher, "error", map[string]any{
			"message": err.Error(),
		})
	}
}

// ResumeStep 处理步骤恢复请求。
func (h *HTTPHandler) ResumeStep(c *gin.Context) {
	h.handleResume(c, false)
}

// ResumeStepStream 处理流式步骤恢复请求。
func (h *HTTPHandler) ResumeStepStream(c *gin.Context) {
	var req approvalResponseRequest
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
	c.Writer.Header().Set("X-AI-Runtime-Mode", h.runtimeModeHeader(rollout))
	c.Writer.Header().Set("X-AI-Compatibility-Enabled", boolHeaderValue(rollout.CompatibilityEnabled()))
	c.Writer.Header().Set("X-AI-Model-First-Enabled", boolHeaderValue(rollout.ModelFirstEnabled()))
	c.Writer.Header().Set("X-AI-Turn-Block-Streaming-Enabled", boolHeaderValue(rollout.TurnBlockStreamingEnabled()))
	c.Status(http.StatusOK)

	emit := func(evt coreai.StreamEvent) bool {
		payload := cloneMap(evt.Data)
		if evt.Type == events.Meta {
			payload = attachRolloutMetadata(payload, rollout)
		}
		return writeSSE(c, flusher, string(evt.Type), payload)
	}
	if _, err := h.resumeRuntimeStream(c.Request.Context(), buildResumeRequest(req), emit); err != nil {
		writeSSE(c, flusher, "error", map[string]any{
			"message": err.Error(),
		})
	}
}

// handleResume 统一处理恢复请求。
func (h *HTTPHandler) handleResume(c *gin.Context, legacyADK bool) {
	var req approvalResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	res, err := h.resumeRuntime(c.Request.Context(), buildResumeRequest(req))
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
		SessionID:    req.SessionID,
		PlanID:       req.PlanID,
		StepID:       firstNonEmpty(req.StepID, req.Target, req.CheckpointID),
		CheckpointID: req.CheckpointID,
		Target:       firstNonEmpty(req.Target, req.CheckpointID),
		Approved:     req.Approved,
		Reason:       req.Reason,
	}
}

func (h *HTTPHandler) runtimeModeHeader(rollout coreai.RolloutConfig) string {
	return rollout.RuntimeMode()
}

func (h *HTTPHandler) runRuntime(ctx context.Context, req coreai.RunRequest, emit coreai.StreamEmitter) error {
	return h.orchestrator.Run(ctx, req, emit)
}

func (h *HTTPHandler) resumeRuntime(ctx context.Context, req coreai.ResumeRequest) (*coreai.ResumeResult, error) {
	return h.orchestrator.Resume(ctx, req)
}

func (h *HTTPHandler) resumeRuntimeStream(ctx context.Context, req coreai.ResumeRequest, emit coreai.StreamEmitter) (*coreai.ResumeResult, error) {
	return h.orchestrator.ResumeStream(ctx, req, emit)
}

// buildResumeResponse 将 ResumeResult 序列化为 HTTP 响应 payload。
// legacyADK=true 时附加废弃标记和迁移提示，用于兼容旧版客户端。
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
		"checkpoint_id":     res.StepID,
		"turn_id":           res.TurnID,
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

// legacyResumeMessage 在旧版 resume 消息中追加迁移提示（若尚未包含新接口路径）。
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

// SubmitFeedback 接收用户对 AI 回答的反馈，当前仅做基础记录（未持久化）。
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

// ListSessions 返回当前用户在指定场景下的所有会话列表（不含消息详情）。
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

// CurrentSession 返回当前用户最近的活跃会话（含消息详情），不存在时返回 null。
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

// GetSession 按 ID 返回指定会话（含消息详情）。
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

// BranchSession 克隆指定会话（含历史消息），生成一个独立的新会话分支。
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

// UpdateSessionTitle 修改会话标题并返回更新后的会话详情。
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

// DeleteSession 软删除指定会话。
func (h *HTTPHandler) DeleteSession(c *gin.Context) {
	if err := h.chatStore.Delete(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("id")); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, nil)
}

// normalizeRuntimeContext 将请求体 context 字段与中间件捕获的基础 RuntimeContext 合并。
// 请求体字段优先级更高，支持 camelCase 和 snake_case 两种键名格式。
func (h *HTTPHandler) normalizeRuntimeContext(c *gin.Context, raw map[string]any) coreai.RuntimeContext {
	ctx := h.baseRuntimeContext(c)
	if ctx.UserContext == nil {
		ctx.UserContext = map[string]any{}
	}
	if ctx.Metadata == nil {
		ctx.Metadata = map[string]any{}
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
	if projectID, ok := raw["project_id"].(string); ok && strings.TrimSpace(projectID) != "" {
		ctx.ProjectID = strings.TrimSpace(projectID)
		ctx.ProjectName = h.lookupProjectName(c.Request.Context(), ctx.ProjectID)
	}
	if projectID, ok := raw["projectId"].(string); ok && strings.TrimSpace(projectID) != "" {
		ctx.ProjectID = strings.TrimSpace(projectID)
		ctx.ProjectName = h.lookupProjectName(c.Request.Context(), ctx.ProjectID)
	}
	if projectName, ok := raw["project_name"].(string); ok && strings.TrimSpace(projectName) != "" {
		ctx.ProjectName = strings.TrimSpace(projectName)
	}
	if projectName, ok := raw["projectName"].(string); ok && strings.TrimSpace(projectName) != "" {
		ctx.ProjectName = strings.TrimSpace(projectName)
	}
	if resources, ok := raw["selected_resources"].([]any); ok {
		ctx.SelectedResources = toSelectedResources(resources)
	}
	if resources, ok := raw["selectedResources"].([]any); ok && len(ctx.SelectedResources) == 0 {
		ctx.SelectedResources = toSelectedResources(resources)
	}
	if ctx.Scene == "" {
		ctx.Scene = "global"
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
			Type:      stringify(row["type"]),
			ID:        stringify(firstNonNil(row["id"], row["value"])),
			Name:      stringify(firstNonNil(row["name"], row["label"])),
			Namespace: stringify(firstNonNil(row["namespace"], row["ns"])),
		})
	}
	return out
}

func mergeScenePayload(scene string, raw map[string]any) map[string]any {
	if len(raw) == 0 && strings.TrimSpace(scene) == "" {
		return nil
	}
	out := make(map[string]any, len(raw)+1)
	for key, value := range raw {
		out[key] = value
	}
	if strings.TrimSpace(scene) != "" {
		out["scene"] = strings.TrimSpace(scene)
	}
	return out
}

// toAPISession 将内部 ChatSessionRecord 转换为 API 响应结构 v1.AISession。
// includeMessages=true 时同时填充消息列表和 turn 列表。
func toAPISession(snapshot aistate.ChatSessionRecord, includeMessages bool) v1.AISession {
	msgs := make([]map[string]any, 0, len(snapshot.Messages))
	turns := make([]v1.AIReplayTurn, 0, len(snapshot.Turns))
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
				"rawEvidence":     payload["rawEvidence"],
				"timestamp":       payload["timestamp"],
			})
		}
		for _, turn := range snapshot.Turns {
			blocks := make([]v1.AIReplayBlock, 0, len(turn.Blocks))
			for _, block := range turn.Blocks {
				blocks = append(blocks, v1.AIReplayBlock{
					ID:          block.ID,
					BlockType:   block.BlockType,
					Position:    block.Position,
					Status:      block.Status,
					Title:       block.Title,
					ContentText: block.ContentText,
					ContentJSON: block.ContentJSON,
					Streaming:   block.Streaming,
					CreatedAt:   block.CreatedAt,
					UpdatedAt:   block.UpdatedAt,
				})
			}
			turns = append(turns, v1.AIReplayTurn{
				ID:           turn.ID,
				Role:         turn.Role,
				Status:       turn.Status,
				Phase:        turn.Phase,
				TraceID:      turn.TraceID,
				ParentTurnID: turn.ParentTurnID,
				Blocks:       blocks,
				CreatedAt:    turn.CreatedAt,
				UpdatedAt:    turn.UpdatedAt,
				CompletedAt:  turn.CompletedAt,
			})
		}
	}
	return v1.AISession{
		ID:        snapshot.ID,
		Scene:     snapshot.Scene,
		Title:     firstNonEmpty(snapshot.Title, "新对话"),
		Messages:  msgs,
		Turns:     turns,
		CreatedAt: snapshot.CreatedAt,
		UpdatedAt: snapshot.UpdatedAt,
	}
}

// writeSSE 向客户端写入一个 SSE 事件帧并立即 Flush。
// 返回 false 表示写入失败（客户端已断开）。
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

// attachRolloutMetadata 将当前灰度配置的各开关状态附加到 meta 事件 payload 中，
// 使前端能感知当前运行时模式，从而渲染对应的 UI 风格。
func attachRolloutMetadata(payload map[string]any, rollout coreai.RolloutConfig) map[string]any {
	if payload == nil {
		payload = map[string]any{}
	}
	payload["runtime_mode"] = rollout.RuntimeMode()
	payload["model_first_enabled"] = rollout.ModelFirstEnabled()
	payload["compatibility_enabled"] = rollout.CompatibilityEnabled()
	payload["turn_block_streaming_enabled"] = rollout.TurnBlockStreamingEnabled()
	return payload
}

// loadSceneConfigs 从数据库加载所有场景配置，以 scene key 为索引返回 map。
func (h *HTTPHandler) loadSceneConfigs(ctx context.Context) (map[string]aitools.SceneConfig, error) {
	rows := make([]model.AISceneConfig, 0)
	if err := h.svcCtx.DB.WithContext(ctx).Order("scene asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]aitools.SceneConfig, len(rows))
	for _, row := range rows {
		cfg := aitools.DecodeSceneConfig(row)
		out[cfg.Scene] = cfg
	}
	return out, nil
}

// sceneConfig 获取指定场景的配置，场景不存在时降级返回 global 配置。
func (h *HTTPHandler) sceneConfig(ctx context.Context, scene string) (*aitools.SceneConfig, error) {
	scene = normalizedScene(scene)
	configs, err := h.loadSceneConfigs(ctx)
	if err != nil {
		return nil, err
	}
	if cfg, ok := configs[scene]; ok {
		return &cfg, nil
	}
	if cfg, ok := configs["global"]; ok {
		return &cfg, nil
	}
	return nil, nil
}

// resolveApprovalScene 将静态场景配置与数据库中的动态配置合并，供审批决策使用。
// 数据库配置（工具白/黑名单、审批策略）覆盖静态默认值。
func (h *HTTPHandler) resolveApprovalScene(scene string) runtime.ResolvedScene {
	resolved := runtime.NewSceneConfigResolver(nil).Resolve(scene)
	if h == nil {
		return resolved
	}
	cfg, err := h.sceneConfig(context.Background(), scene)
	if err != nil || cfg == nil {
		return resolved
	}
	resolved.SceneKey = normalizedScene(scene)
	resolved.SceneConfig.Name = firstNonEmpty(cfg.Name, resolved.SceneConfig.Name)
	resolved.SceneConfig.Description = firstNonEmpty(cfg.Description, resolved.SceneConfig.Description)
	resolved.SceneConfig.Constraints = append([]string(nil), cfg.Constraints...)
	resolved.SceneConfig.AllowedTools = append([]string(nil), cfg.AllowedTools...)
	resolved.SceneConfig.BlockedTools = append([]string(nil), cfg.BlockedTools...)
	resolved.SceneConfig.Examples = append([]string(nil), cfg.Examples...)
	resolved.AllowedTools = append([]string(nil), cfg.AllowedTools...)
	resolved.BlockedTools = append([]string(nil), cfg.BlockedTools...)
	resolved.Constraints = append([]string(nil), cfg.Constraints...)
	resolved.SceneConfig.ApprovalConfig = decodeSceneApprovalConfig(cfg.ApprovalConfig)
	return resolved
}

func decodeSceneApprovalConfig(raw map[string]any) *runtime.SceneApprovalConfig {
	if len(raw) == 0 {
		return nil
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var config runtime.SceneApprovalConfig
	if err := json.Unmarshal(payload, &config); err != nil {
		return nil
	}
	return &config
}

func sortedCapabilities(items []aitools.Capability) []aitools.Capability {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Category == items[j].Category {
			return items[i].Name < items[j].Name
		}
		return items[i].Category < items[j].Category
	})
	return items
}

func findApproval(db *gorm.DB, id string) (*model.AIApproval, error) {
	var row model.AIApproval
	if err := db.Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
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
