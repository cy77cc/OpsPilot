package chathandler_test

import (
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	approvalhandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/approval"
	chathandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/chat"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type aiHandlerTestHarness struct {
	chat     *chathandler.HTTPHandler
	approval *approvalhandler.HTTPHandler
	logic    *logic.Logic
}

func newAIHandlerTestHarness(db *gorm.DB) *aiHandlerTestHarness {
	l := logic.NewLogicWithDB(db, nil)
	l.AIRouter = &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			adk.EventFromMessage(schema.AssistantMessage("ok", nil), nil, schema.Assistant, ""),
		},
	}
	chatSvc := chathandler.NewServiceWithLogic(l)
	approvalSvc := approvalhandler.NewServiceWithLogic(l)
	return &aiHandlerTestHarness{
		chat:     chathandler.NewHTTPHandler(chatSvc),
		approval: approvalhandler.NewHTTPHandler(approvalSvc),
		logic:    l,
	}
}

func (h *aiHandlerTestHarness) Chat(c *gin.Context) {
	h.chat.Chat(c)
}

func (h *aiHandlerTestHarness) ListSessions(c *gin.Context) {
	h.chat.ListSessions(c)
}

func (h *aiHandlerTestHarness) CreateSession(c *gin.Context) {
	h.chat.CreateSession(c)
}

func (h *aiHandlerTestHarness) GetSession(c *gin.Context) {
	h.chat.GetSession(c)
}

func (h *aiHandlerTestHarness) DeleteSession(c *gin.Context) {
	h.chat.DeleteSession(c)
}

func (h *aiHandlerTestHarness) GetRun(c *gin.Context) {
	h.chat.GetRun(c)
}

func (h *aiHandlerTestHarness) GetRunProjection(c *gin.Context) {
	h.chat.GetRunProjection(c)
}

func (h *aiHandlerTestHarness) GetRunContent(c *gin.Context) {
	h.chat.GetRunContent(c)
}

func (h *aiHandlerTestHarness) GetDiagnosisReport(c *gin.Context) {
	h.chat.GetDiagnosisReport(c)
}

func (h *aiHandlerTestHarness) ListPendingApprovals(c *gin.Context) {
	h.approval.ListPendingApprovals(c)
}

func (h *aiHandlerTestHarness) ListPendingApprovalsGlobal(c *gin.Context) {
	h.approval.ListPendingApprovalsGlobal(c)
}

func (h *aiHandlerTestHarness) GetApproval(c *gin.Context) {
	h.approval.GetApproval(c)
}

func (h *aiHandlerTestHarness) SubmitApproval(c *gin.Context) {
	h.approval.SubmitApproval(c)
}

func (h *aiHandlerTestHarness) RetryResumeApproval(c *gin.Context) {
	h.approval.RetryResumeApproval(c)
}
