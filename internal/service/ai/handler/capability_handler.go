package handler

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/service/ai/logic"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

func (h *AIHandler) capabilities(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	if h.svcCtx.AI == nil {
		httpx.OK(c, []any{})
		return
	}
	all := h.svcCtx.AI.ToolMetas()
	out := make([]tools.ToolMeta, 0, len(all))
	for _, item := range all {
		if h.hasPermission(uid, item.Permission) {
			out = append(out, item)
		}
	}
	httpx.OK(c, out)
}

func (h *AIHandler) previewTool(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	var req struct {
		Tool   string         `json:"tool" binding:"required"`
		Params map[string]any `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	meta, ok := h.findMeta(req.Tool)
	if !ok {
		httpx.Fail(c, xcode.NotFound, "tool not found")
		return
	}
	req.Tool = meta.Name
	if h.svcCtx.AI == nil {
		httpx.Fail(c, xcode.ServerError, "ai agent not initialized")
		return
	}
	if !h.hasPermission(uid, meta.Permission) {
		httpx.Fail(c, xcode.Forbidden, "permission denied")
		return
	}
	data := gin.H{
		"tool":              meta.Name,
		"mode":              meta.Mode,
		"risk":              meta.Risk,
		"params":            req.Params,
		"approval_required": meta.Mode == tools.ToolModeMutating,
	}
	if meta.Mode == tools.ToolModeMutating {
		t := h.runtime.NewApproval(uid, ApprovalTicket{
			Tool:   meta.Name,
			Params: req.Params,
			Risk:   meta.Risk,
			Mode:   meta.Mode,
			Meta:   meta,
		})
		data["approval_token"] = t.ID
		data["expiresAt"] = t.ExpiresAt
		data["previewDiff"] = "Mutating operation requires approval."
	}
	httpx.OK(c, data)
}

func (h *AIHandler) executeTool(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	var req struct {
		Tool          string         `json:"tool" binding:"required"`
		Params        map[string]any `json:"params"`
		ApprovalToken string         `json:"approval_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	meta, ok := h.findMeta(req.Tool)
	if !ok {
		httpx.Fail(c, xcode.NotFound, "tool not found")
		return
	}
	req.Tool = meta.Name
	rec := &ExecutionRecord{
		ID:         "exe-" + strconvFormatInt(time.Now().UnixNano()),
		Tool:       req.Tool,
		Params:     req.Params,
		Mode:       meta.Mode,
		Status:     "running",
		RequestUID: uid,
		CreatedAt:  time.Now(),
	}
	start := time.Now()
	ctx := tools.WithToolUser(c.Request.Context(), uid, strings.TrimSpace(req.ApprovalToken))
	ctx = tools.WithToolPolicyChecker(ctx, h.toolPolicy)
	result, err := h.svcCtx.AI.RunTool(ctx, req.Tool, req.Params)
	finished := time.Now()
	rec.FinishedAt = &finished
	rec.Result = &result
	if err != nil {
		rec.Status = "failed"
		rec.Error = err.Error()
	} else if result.OK {
		rec.Status = "succeeded"
	} else {
		rec.Status = "failed"
		rec.Error = result.Error
	}
	if apErr, ok := tools.IsApprovalRequired(err); ok {
		rec.Status = "failed"
		rec.Error = apErr.Error()
	}
	if result.LatencyMS == 0 {
		rec.Result.LatencyMS = time.Since(start).Milliseconds()
	}
	h.runtime.SaveExecution(rec)
	httpx.OK(c, rec)
}

func (h *AIHandler) getExecution(c *gin.Context) {
	if _, ok := uidFromContext(c); !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	rec, ok := h.runtime.GetExecution(c.Param("id"))
	if !ok {
		httpx.Fail(c, xcode.NotFound, "execution not found")
		return
	}
	httpx.OK(c, rec)
}

func (h *AIHandler) createApproval(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	var req struct {
		Tool   string         `json:"tool" binding:"required"`
		Params map[string]any `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	meta, ok := h.findMeta(req.Tool)
	if !ok {
		httpx.Fail(c, xcode.NotFound, "tool not found")
		return
	}
	req.Tool = meta.Name
	if meta.Mode == tools.ToolModeReadonly {
		httpx.Fail(c, xcode.ParamError, "readonly tool does not require approval")
		return
	}
	t := h.runtime.NewApproval(uid, ApprovalTicket{
		Tool:   meta.Name,
		Params: req.Params,
		Risk:   meta.Risk,
		Mode:   meta.Mode,
		Meta:   meta,
	})
	httpx.OK(c, t)
}

func (h *AIHandler) confirmApproval(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	if !h.hasPermission(uid, "ai:approval:review") {
		httpx.Fail(c, xcode.Forbidden, "permission denied")
		return
	}
	var req struct {
		Approve bool `json:"approve"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	id := c.Param("id")
	t, ok := h.runtime.GetApproval(id)
	if !ok {
		httpx.Fail(c, xcode.NotFound, "approval not found")
		return
	}
	if time.Now().After(t.ExpiresAt) {
		_, _ = h.runtime.SetApprovalStatus(id, "expired", uid)
		httpx.Fail(c, xcode.ParamError, "approval expired")
		return
	}
	status := "rejected"
	if req.Approve {
		status = "approved"
	}
	out, _ := h.runtime.SetApprovalStatus(id, status, uid)
	httpx.OK(c, out)
}

func (h *AIHandler) confirmConfirmation(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	var req struct {
		Approve bool `json:"approve"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpx.Fail(c, xcode.ParamError, "confirmation id is required")
		return
	}
	svc := logic.NewConfirmationService(h.svcCtx.DB)
	item, err := svc.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, logic.ErrConfirmationNotFound) {
			httpx.Fail(c, xcode.NotFound, "confirmation not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}
	if item.RequestUserID != uid && !h.isAdmin(uid) {
		httpx.Fail(c, xcode.Forbidden, "permission denied")
		return
	}
	if time.Now().After(item.ExpiresAt) {
		_, _ = svc.ExpirePending(c.Request.Context(), time.Now())
		httpx.Fail(c, xcode.ParamError, "confirmation expired")
		return
	}

	if req.Approve {
		out, err := svc.Confirm(c.Request.Context(), id)
		if err != nil {
			httpx.Fail(c, xcode.ParamError, err.Error())
			return
		}
		httpx.OK(c, out)
		return
	}

	out, err := svc.Cancel(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	httpx.OK(c, out)
}

func (h *AIHandler) findMeta(name string) (tools.ToolMeta, bool) {
	if h.svcCtx.AI == nil {
		return tools.ToolMeta{}, false
	}
	normalized := tools.NormalizeToolName(name)
	for _, item := range h.svcCtx.AI.ToolMetas() {
		if item.Name == normalized {
			return item, true
		}
	}
	return tools.ToolMeta{}, false
}

func strconvFormatInt(v int64) string {
	return fmt.Sprintf("%d", v)
}

func (h *AIHandler) toolParamHints(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	name := strings.TrimSpace(c.Param("name"))
	meta, ok := h.findMeta(name)
	if !ok {
		httpx.Fail(c, xcode.NotFound, "tool not found")
		return
	}
	if !h.hasPermission(uid, meta.Permission) {
		httpx.Fail(c, xcode.Forbidden, "permission denied")
		return
	}
	resp := tools.ResolveToolParamHints(c.Request.Context(), tools.PlatformDeps{DB: h.svcCtx.DB}, meta)
	httpx.OK(c, resp)
}
