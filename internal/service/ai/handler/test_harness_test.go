package handler

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/service/ai/approval"
	"github.com/cy77cc/OpsPilot/internal/service/ai/chat"
	"github.com/cy77cc/OpsPilot/internal/service/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type aiHandlerTestHarness struct {
	logic           *logic.Logic
	chatHandler     *chat.HTTPHandler
	approvalHandler *approval.HTTPHandler
}

func newAIHandlerTestHarness(db *gorm.DB) *aiHandlerTestHarness {
	svcCtx := &svc.ServiceContext{DB: db}
	l := logic.NewAILogic(svcCtx)
	if db != nil {
		l = logic.NewLogicWithDB(db, &noopAgent{})
	}
	return &aiHandlerTestHarness{
		logic:           l,
		chatHandler:     chat.NewHTTPHandler(chat.NewServiceWithLogic(l)),
		approvalHandler: approval.NewHTTPHandler(approval.NewServiceWithLogic(l)),
	}
}

func (h *aiHandlerTestHarness) Chat(c *gin.Context)             { h.chatHandler.Chat(c) }
func (h *aiHandlerTestHarness) CreateSession(c *gin.Context)    { h.chatHandler.CreateSession(c) }
func (h *aiHandlerTestHarness) ListSessions(c *gin.Context)     { h.chatHandler.ListSessions(c) }
func (h *aiHandlerTestHarness) GetSession(c *gin.Context)       { h.chatHandler.GetSession(c) }
func (h *aiHandlerTestHarness) DeleteSession(c *gin.Context)    { h.chatHandler.DeleteSession(c) }
func (h *aiHandlerTestHarness) GetRun(c *gin.Context)           { h.chatHandler.GetRun(c) }
func (h *aiHandlerTestHarness) GetRunProjection(c *gin.Context) { h.chatHandler.GetRunProjection(c) }
func (h *aiHandlerTestHarness) GetRunContent(c *gin.Context)    { h.chatHandler.GetRunContent(c) }
func (h *aiHandlerTestHarness) GetDiagnosisReport(c *gin.Context) {
	h.chatHandler.GetDiagnosisReport(c)
}
func (h *aiHandlerTestHarness) SubmitApproval(c *gin.Context) { h.approvalHandler.SubmitApproval(c) }
func (h *aiHandlerTestHarness) RetryResumeApproval(c *gin.Context) {
	h.approvalHandler.RetryResumeApproval(c)
}
func (h *aiHandlerTestHarness) GetApproval(c *gin.Context) { h.approvalHandler.GetApproval(c) }
func (h *aiHandlerTestHarness) ListPendingApprovals(c *gin.Context) {
	h.approvalHandler.ListPendingApprovals(c)
}
func (h *aiHandlerTestHarness) StartApprovalWorker(ctx context.Context) {
	h.approvalHandler.StartApprovalWorker(ctx)
}
func (h *aiHandlerTestHarness) StartApprovalExpirer(ctx context.Context) {
	h.approvalHandler.StartApprovalExpirer(ctx)
}

type noopAgent struct{}

func (n *noopAgent) Name(_ context.Context) string        { return "NoopAgent" }
func (n *noopAgent) Description(_ context.Context) string { return "No-op agent for testing" }
func (n *noopAgent) Run(_ context.Context, _ *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Close()
	return iter
}
func (n *noopAgent) Resume(_ context.Context, _ *adk.ResumeInfo, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Close()
	return iter
}
