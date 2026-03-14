// 本文件实现 AI 工具相关的 HTTP 处理器：
//   - 工具能力列表与参数提示
//   - 工具预览（dry-run）和直接执行
//   - 审批工单的创建、查询、审批、拒绝
//   - 场景配置的 CRUD
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	coreai "github.com/cy77cc/OpsPilot/internal/ai"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	aitools "github.com/cy77cc/OpsPilot/internal/ai/tools"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// toolActionRequest 是工具预览和直接执行接口的请求体。
type toolActionRequest struct {
	Tool         string         `json:"tool" binding:"required"`
	Params       map[string]any `json:"params"`
	Context      map[string]any `json:"context,omitempty"`
	Scene        string         `json:"scene,omitempty"`
	SessionID    string         `json:"session_id,omitempty"`
	PlanID       string         `json:"plan_id,omitempty"`
	StepID       string         `json:"step_id,omitempty"`
	CheckpointID string         `json:"checkpoint_id,omitempty"`
}

// createApprovalRequest 是创建审批工单接口的请求体。
type createApprovalRequest struct {
	SessionID    string         `json:"session_id,omitempty"`
	PlanID       string         `json:"plan_id,omitempty"`
	StepID       string         `json:"step_id,omitempty"`
	CheckpointID string         `json:"checkpoint_id,omitempty"`
	Context      map[string]any `json:"context,omitempty"`
	Scene        string         `json:"scene,omitempty"`
	Tool         string         `json:"tool" binding:"required"`
	Params       map[string]any `json:"params"`
	Summary      string         `json:"summary,omitempty"`
	Reason       string         `json:"reason,omitempty"`
}

type chainApprovalDecisionRequest struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}

// updateSceneConfigRequest 是更新场景配置接口的请求体。
type updateSceneConfigRequest struct {
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Constraints    []string       `json:"constraints"`
	AllowedTools   []string       `json:"allowed_tools"`
	BlockedTools   []string       `json:"blocked_tools"`
	Examples       []string       `json:"examples"`
	ApprovalConfig map[string]any `json:"approval_config"`
}

// Capabilities 返回当前场景下可用的工具能力列表，按 category+name 排序。
func (h *HTTPHandler) Capabilities(c *gin.Context) {
	scene := normalizedScene(c.Query("scene"))
	cfg, err := h.sceneConfig(c.Request.Context(), scene)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, sortedCapabilities(h.registry.FilterByScene(scene, cfg)))
}

// ToolParamHints 返回指定工具的参数智能提示（枚举选项、默认值、依赖关系）。
// 提示内容根据当前 RuntimeContext（用户选中资源、项目等）动态解析。
func (h *HTTPHandler) ToolParamHints(c *gin.Context) {
	name := c.Param("name")
	spec, ok := h.registry.Get(name)
	if !ok {
		httpx.Fail(c, xcode.NotFound, "tool not found")
		return
	}
	params := hintRequestParams(c)
	runtimeCtx := h.normalizeRuntimeContext(c, params)
	hints, err := h.hintResolver.Resolve(c.Request.Context(), spec, runtimeCtx, params)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"tool": name, "params": hints, "context": runtimeCtx})
}

func (h *HTTPHandler) PreviewTool(c *gin.Context) {
	var req toolActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	spec, ok := h.registry.Get(req.Tool)
	if !ok {
		httpx.Fail(c, xcode.NotFound, "tool not found")
		return
	}
	deps := common.PlatformDeps{DB: h.svcCtx.DB, Prometheus: h.svcCtx.Prometheus}
	if spec.Preview != nil {
		ctx, _ := h.contextWithRuntime(c, mergeScenePayload(req.Scene, req.Context))
		preview, err := spec.Preview(ctx, deps, req.Params)
		if err != nil {
			httpx.ServerErr(c, err)
			return
		}
		httpx.OK(c, gin.H{"tool": spec.Name, "preview": preview, "dry_run": true})
		return
	}
	httpx.OK(c, gin.H{"tool": spec.Name, "preview": gin.H{"params": req.Params, "mode": spec.Mode, "risk": spec.Risk}, "dry_run": true})
}

func (h *HTTPHandler) ExecuteTool(c *gin.Context) {
	var req toolActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	spec, ok := h.registry.Get(req.Tool)
	if !ok {
		httpx.Fail(c, xcode.NotFound, "tool not found")
		return
	}
	runtimeCtx := h.normalizeToolRuntimeContext(c, req.Context, req.Scene)
	decision, err := h.approvalDecision(c.Request.Context(), spec, runtimeCtx, req.Params)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	exec := model.AIExecution{
		ID:            uuid.NewString(),
		SessionID:     strings.TrimSpace(req.SessionID),
		PlanID:        strings.TrimSpace(req.PlanID),
		StepID:        firstNonEmpty(req.StepID, req.CheckpointID),
		CheckpointID:  firstNonEmpty(req.CheckpointID, req.StepID),
		RequestUserID: httpx.UIDFromCtx(c),
		ToolName:      spec.Name,
		ToolMode:      firstNonEmpty(decision.Tool.Mode, string(spec.Mode)),
		RiskLevel:     firstNonEmpty(decision.Tool.Risk, string(spec.Risk)),
		Scene:         normalizedScene(runtimeCtx.Scene),
		Status:        "running",
		ParamsJSON:    mustJSON(req.Params),
	}
	now := time.Now().UTC()
	exec.StartedAt = &now
	if decision.NeedApproval && strings.TrimSpace(req.SessionID) != "" && strings.TrimSpace(req.PlanID) != "" && strings.TrimSpace(firstNonEmpty(req.StepID, req.CheckpointID)) == "" {
		httpx.Fail(c, xcode.ParamError, "step_id is required when approval is needed")
		return
	}
	if decision.NeedApproval && strings.TrimSpace(req.CheckpointID) == "" && strings.TrimSpace(req.StepID) == "" {
		exec.Status = "pending_approval"
	}
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Create(&exec).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if exec.Status == "pending_approval" {
		httpx.OK(c, gin.H{"id": exec.ID, "tool": exec.ToolName, "status": exec.Status, "created_at": exec.CreatedAt})
		return
	}
	ctx, _ := h.contextWithRuntime(c, mergeScenePayload(req.Scene, req.Context))
	result, err := h.executeRegistryTool(ctx, spec, req.Params)
	if err != nil {
		finalizeExecutionRecord(c.Request.Context(), h.svcCtx.DB, exec, nil, err)
		httpx.ServerErr(c, err)
		return
	}
	finalizeExecutionRecord(c.Request.Context(), h.svcCtx.DB, exec, result, nil)
	httpx.OK(c, gin.H{"id": exec.ID, "tool": exec.ToolName, "status": "success", "result": result.Result, "created_at": exec.CreatedAt})
}

func (h *HTTPHandler) ExecutionStatus(c *gin.Context) {
	var row model.AIExecution
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Where("id = ?", c.Param("id")).First(&row).Error; err != nil {
		httpx.Fail(c, xcode.NotFound, "execution not found")
		return
	}
	httpx.OK(c, gin.H{
		"id":                 row.ID,
		"session_id":         row.SessionID,
		"plan_id":            row.PlanID,
		"step_id":            row.StepID,
		"checkpoint_id":      firstNonEmpty(row.CheckpointID, row.StepID),
		"tool":               row.ToolName,
		"mode":               row.ToolMode,
		"risk_level":         row.RiskLevel,
		"params":             decodeAnyJSON(row.ParamsJSON),
		"status":             row.Status,
		"result":             decodeAnyJSON(row.ResultJSON),
		"metadata":           decodeAnyJSON(row.MetadataJSON),
		"error":              row.ErrorMessage,
		"duration_ms":        row.DurationMs,
		"prompt_tokens":      row.PromptTokens,
		"completion_tokens":  row.CompletionTokens,
		"total_tokens":       row.TotalTokens,
		"estimated_cost_usd": row.EstimatedCostUSD,
		"started_at":         row.StartedAt,
		"finished_at":        row.FinishedAt,
		"created_at":         row.CreatedAt,
		"updated_at":         row.UpdatedAt,
	})
}

func (h *HTTPHandler) CreateApproval(c *gin.Context) {
	var req createApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	spec, ok := h.registry.Get(req.Tool)
	if !ok {
		httpx.Fail(c, xcode.NotFound, "tool not found")
		return
	}
	runtimeCtx := h.normalizeToolRuntimeContext(c, req.Context, req.Scene)
	decision, err := h.approvalDecision(c.Request.Context(), spec, runtimeCtx, req.Params)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	deps := common.PlatformDeps{DB: h.svcCtx.DB, Prometheus: h.svcCtx.Prometheus}
	preview := map[string]any(nil)
	if spec.Preview != nil {
		ctx, _ := h.contextWithRuntime(c, mergeScenePayload(req.Scene, req.Context))
		value, err := spec.Preview(ctx, deps, req.Params)
		if err != nil {
			httpx.ServerErr(c, err)
			return
		}
		if cast, ok := value.(map[string]any); ok {
			preview = cast
		} else {
			preview = map[string]any{"value": value}
		}
	}
	stepID := firstNonEmpty(req.StepID, req.CheckpointID)
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	approval := model.AIApproval{
		ID:              uuid.NewString(),
		SessionID:       strings.TrimSpace(req.SessionID),
		PlanID:          strings.TrimSpace(req.PlanID),
		StepID:          stepID,
		CheckpointID:    firstNonEmpty(req.CheckpointID, stepID),
		ApprovalKey:     approvalKey(strings.TrimSpace(req.SessionID), strings.TrimSpace(req.PlanID), stepID),
		RequestUserID:   httpx.UIDFromCtx(c),
		ToolName:        spec.Name,
		ToolDisplayName: firstNonEmpty(decision.Tool.DisplayName, spec.DisplayName, spec.Name),
		ToolMode:        firstNonEmpty(decision.Tool.Mode, string(spec.Mode)),
		RiskLevel:       firstNonEmpty(decision.Tool.Risk, string(spec.Risk)),
		Status:          "pending",
		Scene:           normalizedScene(runtimeCtx.Scene),
		Summary:         firstNonEmpty(strings.TrimSpace(req.Summary), h.renderApprovalSummary(decision, req.Params)),
		Reason:          strings.TrimSpace(req.Reason),
		ParamsJSON:      mustJSON(req.Params),
		PreviewJSON:     mustJSON(preview),
		ExpiresAt:       &expiresAt,
	}
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Create(&approval).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, approvalPayload(approval))
}

func (h *HTTPHandler) ListApprovals(c *gin.Context) {
	rows := make([]model.AIApproval, 0)
	q := h.svcCtx.DB.WithContext(c.Request.Context()).Order("created_at desc")
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Find(&rows).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, approvalPayload(row))
	}
	httpx.OK(c, gin.H{"list": out, "total": len(out)})
}

func (h *HTTPHandler) GetApproval(c *gin.Context) {
	row, err := findApproval(h.svcCtx.DB.WithContext(c.Request.Context()), c.Param("id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if row == nil {
		httpx.Fail(c, xcode.NotFound, "approval not found")
		return
	}
	httpx.OK(c, approvalPayload(*row))
}

func (h *HTTPHandler) DecideChainApproval(c *gin.Context) {
	var req chainApprovalDecisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	nodeID := strings.TrimSpace(c.Param("node_id"))
	stepID := strings.TrimPrefix(nodeID, "approval:")
	stepID = strings.TrimSpace(stepID)
	if stepID == "" {
		httpx.Fail(c, xcode.ParamError, "node_id is required")
		return
	}

	chainID := strings.TrimSpace(c.Param("chain_id"))
	query := h.svcCtx.DB.WithContext(c.Request.Context()).Order("created_at desc")
	if chainID != "" {
		query = query.Where("plan_id = ?", chainID)
	}
	var row model.AIApproval
	if err := query.Where("step_id = ?", stepID).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.Fail(c, xcode.NotFound, "approval not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}

	if req.Approved && wantsEventStream(c) {
		h.respondChainApprovalDecisionStream(c, row, req.Reason)
		return
	}
	h.respondApprovalDecision(c, row, req.Approved, req.Reason)
}

func (h *HTTPHandler) respondApprovalDecision(c *gin.Context, row model.AIApproval, approved bool, reason string) {
	if approved {
		execID, startedAt, err := h.markApprovalApproved(c, row, reason)
		if err != nil {
			httpx.ServerErr(c, err)
			return
		}
		res, err := h.resumeRuntime(c.Request.Context(), approvalResumeRequest(row, true, reason))
		approvalExec := model.AIExecution{
			ID:           execID,
			SessionID:    row.SessionID,
			PlanID:       row.PlanID,
			StepID:       row.StepID,
			CheckpointID: firstNonEmpty(row.CheckpointID, row.StepID),
			ApprovalID:   row.ID,
			ToolName:     row.ToolName,
			ToolMode:     row.ToolMode,
			RiskLevel:    row.RiskLevel,
			Scene:        row.Scene,
			Status:       executionStatusFromResume(res, err),
			ParamsJSON:   row.ParamsJSON,
			StartedAt:    startedAt,
		}
		if err != nil {
			finalizeExecutionRecord(c.Request.Context(), h.svcCtx.DB, approvalExec, nil, err)
		} else {
			finalizeExecutionRecord(c.Request.Context(), h.svcCtx.DB, approvalExec, &aitools.Execution{
				Result: res,
				Metadata: map[string]any{
					"token_accounting_status": "unavailable",
					"token_accounting_source": "resume_runtime_result",
				},
			}, nil)
		}
		httpx.OK(c, gin.H{
			"approval":  gin.H{"id": row.ID, "status": "approved"},
			"execution": gin.H{"id": execID, "status": firstNonEmpty(executionStatusFromResume(res, err), "running")},
		})
		return
	}

	now := time.Now().UTC()
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Model(&model.AIApproval{}).Where("id = ?", row.ID).Updates(map[string]any{
		"status":           "rejected",
		"reviewer_user_id": httpx.UIDFromCtx(c),
		"reason":           strings.TrimSpace(reason),
		"rejected_at":      now,
	}).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	_, _ = h.resumeRuntime(c.Request.Context(), coreai.ResumeRequest{
		SessionID:    row.SessionID,
		PlanID:       row.PlanID,
		StepID:       row.StepID,
		CheckpointID: row.CheckpointID,
		Approved:     false,
		Reason:       reason,
	})
	httpx.OK(c, gin.H{"id": row.ID, "status": "rejected", "reason": strings.TrimSpace(reason)})
}

func (h *HTTPHandler) respondChainApprovalDecisionStream(c *gin.Context, row model.AIApproval, reason string) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		httpx.Fail(c, xcode.ServerError, "streaming is not supported")
		return
	}

	execID, startedAt, err := h.markApprovalApproved(c, row, reason)
	if err != nil {
		httpx.ServerErr(c, err)
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
		if string(evt.Type) == "meta" {
			payload = attachRolloutMetadata(payload, rollout)
		}
		return writeSSE(c, flusher, string(evt.Type), payload)
	}

	res, streamErr := h.resumeRuntimeStream(c.Request.Context(), approvalResumeRequest(row, true, reason), emit)
	approvalExec := model.AIExecution{
		ID:           execID,
		SessionID:    row.SessionID,
		PlanID:       row.PlanID,
		StepID:       row.StepID,
		CheckpointID: firstNonEmpty(row.CheckpointID, row.StepID),
		ApprovalID:   row.ID,
		ToolName:     row.ToolName,
		ToolMode:     row.ToolMode,
		RiskLevel:    row.RiskLevel,
		Scene:        row.Scene,
		Status:       executionStatusFromResume(res, streamErr),
		ParamsJSON:   row.ParamsJSON,
		StartedAt:    startedAt,
	}
	if streamErr != nil {
		finalizeExecutionRecord(c.Request.Context(), h.svcCtx.DB, approvalExec, nil, streamErr)
		_ = writeSSE(c, flusher, "error", map[string]any{"message": streamErr.Error()})
		return
	}
	finalizeExecutionRecord(c.Request.Context(), h.svcCtx.DB, approvalExec, &aitools.Execution{
		Result: res,
		Metadata: map[string]any{
			"token_accounting_status": "unavailable",
			"token_accounting_source": "resume_runtime_result",
		},
	}, nil)
}

func (h *HTTPHandler) markApprovalApproved(c *gin.Context, row model.AIApproval, reason string) (string, *time.Time, error) {
	execID := uuid.NewString()
	now := time.Now().UTC()
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AIApproval{}).Where("id = ?", row.ID).Updates(map[string]any{
			"status":           "approved",
			"reviewer_user_id": httpx.UIDFromCtx(c),
			"reason":           firstNonEmpty(strings.TrimSpace(reason), row.Reason),
			"approved_at":      now,
			"execution_id":     execID,
		}).Error; err != nil {
			return err
		}
		return tx.Create(&model.AIExecution{
			ID:            execID,
			SessionID:     row.SessionID,
			PlanID:        row.PlanID,
			StepID:        row.StepID,
			CheckpointID:  firstNonEmpty(row.CheckpointID, row.StepID),
			ApprovalID:    row.ID,
			RequestUserID: row.RequestUserID,
			ToolName:      row.ToolName,
			ToolMode:      row.ToolMode,
			RiskLevel:     row.RiskLevel,
			Scene:         row.Scene,
			Status:        "running",
			ParamsJSON:    row.ParamsJSON,
			StartedAt:     &now,
		}).Error
	}); err != nil {
		return "", nil, err
	}
	return execID, &now, nil
}

func approvalResumeRequest(row model.AIApproval, approved bool, reason string) coreai.ResumeRequest {
	return coreai.ResumeRequest{
		SessionID:    row.SessionID,
		PlanID:       row.PlanID,
		StepID:       row.StepID,
		CheckpointID: row.CheckpointID,
		Approved:     approved,
		Reason:       reason,
	}
}

func wantsEventStream(c *gin.Context) bool {
	return strings.Contains(strings.ToLower(c.GetHeader("Accept")), "text/event-stream")
}

func (h *HTTPHandler) SceneTools(c *gin.Context) {
	scene := normalizedScene(c.Param("scene"))
	cfg, err := h.sceneConfig(c.Request.Context(), scene)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, sortedCapabilities(h.registry.FilterByScene(scene, cfg)))
}

func (h *HTTPHandler) ScenePrompts(c *gin.Context) {
	scene := normalizedScene(c.Param("scene"))
	rows := make([]model.AIScenePrompt, 0)
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Where("scene = ? AND is_active = ?", scene, true).Order("display_order asc, id asc").Find(&rows).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	prompts := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		prompts = append(prompts, gin.H{"id": row.ID, "scene": row.Scene, "text": row.PromptText, "type": row.PromptType, "display_order": row.DisplayOrder})
	}
	httpx.OK(c, prompts)
}

func (h *HTTPHandler) ListSceneConfigs(c *gin.Context) {
	rows := make([]model.AISceneConfig, 0)
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Order("scene asc").Find(&rows).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	out := make([]aitools.SceneConfig, 0, len(rows))
	for _, row := range rows {
		out = append(out, aitools.DecodeSceneConfig(row))
	}
	httpx.OK(c, out)
}

func (h *HTTPHandler) GetSceneConfig(c *gin.Context) {
	scene := normalizedScene(c.Param("scene"))
	var row model.AISceneConfig
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Where("scene = ?", scene).First(&row).Error; err != nil {
		httpx.Fail(c, xcode.NotFound, "scene config not found")
		return
	}
	httpx.OK(c, aitools.DecodeSceneConfig(row))
}

func (h *HTTPHandler) UpdateSceneConfig(c *gin.Context) {
	scene := normalizedScene(c.Param("scene"))
	var req updateSceneConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row := aitools.EncodeSceneConfig(aitools.SceneConfig{
		Scene:          scene,
		Name:           req.Name,
		Description:    req.Description,
		Constraints:    req.Constraints,
		AllowedTools:   req.AllowedTools,
		BlockedTools:   req.BlockedTools,
		Examples:       req.Examples,
		ApprovalConfig: req.ApprovalConfig,
	})
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Where("scene = ?", scene).Assign(row).FirstOrCreate(&model.AISceneConfig{}).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, aitools.DecodeSceneConfig(row))
}

func (h *HTTPHandler) DeleteSceneConfig(c *gin.Context) {
	scene := normalizedScene(c.Param("scene"))
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Where("scene = ?", scene).Delete(&model.AISceneConfig{}).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"scene": scene, "deleted": true})
}

func (h *HTTPHandler) executeRegistryTool(ctx context.Context, spec aitools.ToolSpec, params map[string]any) (*aitools.Execution, error) {
	if spec.Execute == nil {
		return nil, fmt.Errorf("tool %s does not support execution", spec.Name)
	}
	return spec.Execute(ctx, common.PlatformDeps{DB: h.svcCtx.DB, Prometheus: h.svcCtx.Prometheus}, params)
}

func approvalPayload(row model.AIApproval) gin.H {
	return gin.H{
		"id":            row.ID,
		"session_id":    row.SessionID,
		"plan_id":       row.PlanID,
		"step_id":       row.StepID,
		"checkpoint_id": firstNonEmpty(row.CheckpointID, row.StepID),
		"approval_key":  row.ApprovalKey,
		"tool":          row.ToolName,
		"tool_name":     row.ToolDisplayName,
		"mode":          row.ToolMode,
		"risk":          row.RiskLevel,
		"risk_level":    row.RiskLevel,
		"status":        row.Status,
		"scene":         row.Scene,
		"summary":       row.Summary,
		"params":        decodeAnyJSON(row.ParamsJSON),
		"preview":       decodeAnyJSON(row.PreviewJSON),
		"created_at":    row.CreatedAt,
		"updated_at":    row.UpdatedAt,
	}
}

func previewSummary(spec aitools.ToolSpec, params map[string]any) string {
	if len(params) == 0 {
		return fmt.Sprintf("%s requires approval", spec.Name)
	}
	return fmt.Sprintf("%s with params %s", spec.Name, mustJSON(params))
}

func (h *HTTPHandler) normalizeToolRuntimeContext(c *gin.Context, raw map[string]any, scene string) coreai.RuntimeContext {
	return h.normalizeRuntimeContext(c, mergeScenePayload(scene, raw))
}

func (h *HTTPHandler) approvalDecision(ctx context.Context, spec aitools.ToolSpec, runtimeCtx coreai.RuntimeContext, params map[string]any) (airuntime.ApprovalDecision, error) {
	if h == nil || h.approvals == nil {
		return airuntime.ApprovalDecision{
			NeedApproval: spec.Mode == aitools.ModeMutating,
			Reason:       "fallback mutating approval policy",
			Tool: airuntime.ApprovalToolSpec{
				Name:        spec.Name,
				DisplayName: spec.DisplayName,
				Description: spec.Description,
				Mode:        string(spec.Mode),
				Risk:        string(spec.Risk),
				Category:    spec.Category,
			},
		}, nil
	}
	return h.approvals.Decide(ctx, airuntime.ApprovalCheckRequest{
		ToolName:       spec.Name,
		Mode:           string(spec.Mode),
		Risk:           string(spec.Risk),
		Scene:          runtimeCtx.Scene,
		Environment:    stringify(firstNonNil(runtimeCtx.Metadata["environment"], runtimeCtx.Metadata["env"])),
		Namespace:      firstNonEmpty(stringify(params["namespace"]), selectedNamespace(runtimeCtx.SelectedResources)),
		Params:         params,
		RuntimeContext: runtimeCtx,
	})
}

func (h *HTTPHandler) renderApprovalSummary(decision airuntime.ApprovalDecision, params map[string]any) string {
	if h == nil || h.summaries == nil {
		return previewSummary(aitools.ToolSpec{Capability: aitools.Capability{Name: decision.Tool.Name}}, params)
	}
	return h.summaries.Render(decision, params)
}

func selectedNamespace(resources []coreai.SelectedResource) string {
	for _, resource := range resources {
		if strings.TrimSpace(resource.Namespace) != "" {
			return strings.TrimSpace(resource.Namespace)
		}
	}
	return ""
}

func decodeAnyJSON(raw string) any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return raw
	}
	return out
}

func mustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func executionStatusFromResume(res *coreai.ResumeResult, err error) string {
	if err != nil {
		return "failed"
	}
	if res == nil || strings.TrimSpace(res.Status) == "" {
		return "success"
	}
	switch strings.TrimSpace(strings.ToLower(res.Status)) {
	case "completed", "complete", "success", "succeeded":
		return "success"
	case "rejected", "abort", "aborted", "cancelled", "canceled":
		return "rejected"
	case "failed", "error":
		return "failed"
	default:
		return strings.TrimSpace(res.Status)
	}
}

func approvalKey(sessionID, planID, stepID string) string {
	sessionID = strings.TrimSpace(sessionID)
	planID = strings.TrimSpace(planID)
	stepID = strings.TrimSpace(stepID)
	if sessionID != "" && planID != "" && stepID != "" {
		return sessionID + ":" + planID + ":" + stepID
	}
	return uuid.NewString()
}
