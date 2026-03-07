package handler

import (
	"net/http"
	"strings"
	"time"

	aitools "github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/service/ai/events"
	coreai "github.com/cy77cc/k8s-manage/internal/ai"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

func (h *AIHandler) chatWithADK(c *gin.Context, req ChatRequest, uid uint64, msg string) {
	if h.orchestrator == nil {
		httpx.Fail(c, xcode.ServerError, "ai orchestrator not initialized")
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
	emit := writer.Emit
	defer writer.Close()

	stopHeartbeat := make(chan struct{})
	go events.HeartbeatLoop(stopHeartbeat, emit)
	defer close(stopHeartbeat)

	err := h.orchestrator.ChatStream(c.Request.Context(), coreai.ChatStreamRequest{
		UserID:    uid,
		SessionID: req.SessionID,
		Message:   msg,
		Context:   req.Context,
	}, func(event string, payload map[string]any) bool {
		return emit(event, gin.H(payload))
	})
	if err != nil {
		_ = emit("error", gin.H{"message": err.Error()})
	}
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
	payload, err := h.orchestrator.ResumePayload(c.Request.Context(), strings.TrimSpace(req.CheckpointID), map[string]any{
		strings.TrimSpace(req.Target): req.Data,
	})
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

	payload, err := h.orchestrator.ResumePayload(c.Request.Context(), checkpointID, map[string]any{target: data})
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, payload)
}
