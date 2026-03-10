package handler

import (
	"net/http"
	"strings"
	"time"

	aitools "github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/service/ai/events"
	"github.com/cy77cc/k8s-manage/internal/service/ai/logic"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

func (h *AIHandler) chatWithADK(c *gin.Context, req ChatRequest, uid uint64, msg string) {
	if h.ai == nil {
		httpx.Fail(c, xcode.ServerError, "ai agent not initialized")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		httpx.Fail(c, xcode.ServerError, "streaming not supported")
		return
	}

	turnID := "turn-" + strconvFormatInt(time.Now().UnixNano())
	writer := events.NewSSEWriter(c, flusher, turnID)
	emit := func(event string, payload gin.H) bool {
		for _, projected := range events.ProjectCompatibilityEvents(event, payload) {
			if !writer.Emit(projected.Name, projected.Payload) {
				return false
			}
		}
		return true
	}
	defer writer.Close()

	stopHeartbeat := make(chan struct{})
	go events.HeartbeatLoop(stopHeartbeat, emit)
	defer close(stopHeartbeat)

	scene := logic.NormalizeScene(logic.ToString(req.Context["scene"]))
	session := h.sessions.Ensure(uid, scene)
	if req.SessionID != "" {
		session.ID = req.SessionID
		h.sessions.Put(session)
	}
	h.runtime.RememberContext(uid, session.Scene, req.Context)

	// 发送 meta 事件
	emit("meta", gin.H{"sessionId": session.ID})

	_, err := h.ai.Stream(c.Request.Context(), session.ID, msg, uid, req.Context, func(event string, payload map[string]any) bool {
		return emit(event, gin.H(payload))
	})
	if err != nil {
		_ = emit("error", gin.H{"message": err.Error()})
		return
	}

	// 检查中断状态在 AIAgent.Stream 内部已经处理或通过事件发送了
	emit("done", gin.H{"stream_state": "ok"})
}

func (h *AIHandler) resumeADKApproval(c *gin.Context) {
	var req struct {
		CheckpointID string `json:"checkpoint_id" binding:"required"`
		Target       string `json:"target" binding:"required"`
		Data         any    `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	checkpointID := strings.TrimSpace(req.CheckpointID)
	targets := map[string]any{
		strings.TrimSpace(req.Target): req.Data,
	}

	payload, err := h.ai.ResumePayload(c.Request.Context(), checkpointID, targets)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, payload)
}

func (h *AIHandler) handleApprovalResponse(c *gin.Context) {
	var req struct {
		CheckpointID string `json:"checkpoint_id"`
		SessionID    string `json:"session_id"`
		Target       string `json:"target"`
		Approved     *bool  `json:"approved"`
		Reason       string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	checkpointID := strings.TrimSpace(req.CheckpointID)
	if checkpointID == "" {
		checkpointID = strings.TrimSpace(req.SessionID)
	}
	target := strings.TrimSpace(req.Target)
	if checkpointID == "" || target == "" || req.Approved == nil {
		httpx.Fail(c, xcode.ParamError, "checkpoint_id/session_id, target and approved are required")
		return
	}

	var data any
	if *req.Approved {
		data = &aitools.ApprovalResult{Approved: true}
	} else {
		reason := strings.TrimSpace(req.Reason)
		payload := &aitools.ApprovalResult{Approved: false}
		if reason != "" {
			payload.DisapproveReason = &reason
		}
		data = payload
	}

	payload, err := h.ai.ResumePayload(c.Request.Context(), checkpointID, map[string]any{target: data})
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, payload)
}

func (h *AIHandler) streamApprovals(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		httpx.Fail(c, xcode.ServerError, "streaming not supported")
		return
	}

	writer := events.NewSSEWriter(c, flusher, "approval-stream")
	defer writer.Close()

	emit := func(event string, payload gin.H) bool {
		return writer.Emit(event, payload)
	}

	ch, unsubscribe := events.DefaultApprovalHub().Subscribe(uid, 16)
	defer unsubscribe()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	_ = emit("ready", gin.H{"user_id": uid})
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case update, ok := <-ch:
			if !ok {
				return
			}
			if !emit("approval_update", gin.H{
				"id":               update.ID,
				"approval_token":   update.ApprovalToken,
				"tool_name":        update.ToolName,
				"status":           update.Status,
				"request_user_id":  update.RequestUserID,
				"approver_user_id": update.ApproverUserID,
				"execution":        update.Execution,
				"updated_at":       update.UpdatedAt,
			}) {
				return
			}
		case <-ticker.C:
			if !emit("heartbeat", gin.H{"status": "ok"}) {
				return
			}
		}
	}
}
