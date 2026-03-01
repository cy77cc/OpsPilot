package ai

import (
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

func (h *handler) capabilities(c *gin.Context) {
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

func (h *handler) previewTool(c *gin.Context) {
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
		t := h.store.newApproval(uid, approvalTicket{
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

func (h *handler) executeTool(c *gin.Context) {
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
	rec := &executionRecord{
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
	h.store.saveExecution(rec)
	httpx.OK(c, rec)
}

func (h *handler) getExecution(c *gin.Context) {
	if _, ok := uidFromContext(c); !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	rec, ok := h.store.getExecution(c.Param("id"))
	if !ok {
		httpx.Fail(c, xcode.NotFound, "execution not found")
		return
	}
	httpx.OK(c, rec)
}

func (h *handler) createApproval(c *gin.Context) {
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
	t := h.store.newApproval(uid, approvalTicket{
		Tool:   meta.Name,
		Params: req.Params,
		Risk:   meta.Risk,
		Mode:   meta.Mode,
		Meta:   meta,
	})
	httpx.OK(c, t)
}

func (h *handler) confirmApproval(c *gin.Context) {
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
	t, ok := h.store.getApproval(id)
	if !ok {
		httpx.Fail(c, xcode.NotFound, "approval not found")
		return
	}
	if time.Now().After(t.ExpiresAt) {
		_, _ = h.store.setApprovalStatus(id, "expired", uid)
		httpx.Fail(c, xcode.ParamError, "approval expired")
		return
	}
	status := "rejected"
	if req.Approve {
		status = "approved"
	}
	out, _ := h.store.setApprovalStatus(id, status, uid)
	httpx.OK(c, out)
}

func (h *handler) findMeta(name string) (tools.ToolMeta, bool) {
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
