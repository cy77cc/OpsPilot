package handler

import (
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/gin-gonic/gin"
)

// HTTPHandler exposes the existing AI HTTP surface for route composition.
type HTTPHandler struct {
	inner *AIHandler
}

// NewHTTPHandler creates a new HTTPHandler
func NewHTTPHandler(svcCtx *svc.ServiceContext) *HTTPHandler {
	return &HTTPHandler{inner: NewAIHandler(svcCtx)}
}

func (h *HTTPHandler) Chat(c *gin.Context)                   { h.inner.chat(c) }
func (h *HTTPHandler) ChatRespond(c *gin.Context)            { h.inner.handleApprovalResponse(c) }
func (h *HTTPHandler) ListSessions(c *gin.Context)           { h.inner.listSessions(c) }
func (h *HTTPHandler) GetSession(c *gin.Context)             { h.inner.getSession(c) }
func (h *HTTPHandler) DeleteSession(c *gin.Context)          { h.inner.deleteSession(c) }
func (h *HTTPHandler) SubmitFeedback(c *gin.Context)         { h.inner.submitFeedback(c) }
func (h *HTTPHandler) HandleApprovalResponse(c *gin.Context) { h.inner.handleApprovalResponse(c) }
func (h *HTTPHandler) ResumeADKApproval(c *gin.Context)      { h.inner.resumeADKApproval(c) }
func (h *HTTPHandler) CurrentSession(c *gin.Context)         { h.inner.currentSession(c) }
func (h *HTTPHandler) BranchSession(c *gin.Context)          { h.inner.branchSession(c) }
func (h *HTTPHandler) UpdateSessionTitle(c *gin.Context)     { h.inner.updateSessionTitle(c) }
