package testsupport

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/adk"
	approvalcompat "github.com/cy77cc/OpsPilot/internal/modules/ai/approval"
	chatcompat "github.com/cy77cc/OpsPilot/internal/modules/ai/chat"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type HandlerHarness struct {
	Logic           *logic.Logic
	ChatHandler     *chatcompat.HTTPHandler
	ApprovalHandler *approvalcompat.HTTPHandler
}

func NewAIHandlerTestHarness(db *gorm.DB) *HandlerHarness {
	svcCtx := &svc.ServiceContext{DB: db}
	l := logic.NewAILogic(svcCtx)
	if db != nil {
		l = logic.NewLogicWithDB(db, &noopAgent{})
	}
	return &HandlerHarness{
		Logic:           l,
		ChatHandler:     chatcompat.NewHTTPHandler(chatcompat.NewServiceWithLogic(l)),
		ApprovalHandler: approvalcompat.NewHTTPHandler(approvalcompat.NewServiceWithLogic(l)),
	}
}

func (h *HandlerHarness) Chat(c *gin.Context)             { h.ChatHandler.Chat(c) }
func (h *HandlerHarness) CreateSession(c *gin.Context)    { h.ChatHandler.CreateSession(c) }
func (h *HandlerHarness) ListSessions(c *gin.Context)     { h.ChatHandler.ListSessions(c) }
func (h *HandlerHarness) GetSession(c *gin.Context)       { h.ChatHandler.GetSession(c) }
func (h *HandlerHarness) DeleteSession(c *gin.Context)    { h.ChatHandler.DeleteSession(c) }
func (h *HandlerHarness) GetRun(c *gin.Context)           { h.ChatHandler.GetRun(c) }
func (h *HandlerHarness) GetRunProjection(c *gin.Context) { h.ChatHandler.GetRunProjection(c) }
func (h *HandlerHarness) GetRunContent(c *gin.Context)    { h.ChatHandler.GetRunContent(c) }
func (h *HandlerHarness) GetDiagnosisReport(c *gin.Context) {
	h.ChatHandler.GetDiagnosisReport(c)
}
func (h *HandlerHarness) SubmitApproval(c *gin.Context) { h.ApprovalHandler.SubmitApproval(c) }
func (h *HandlerHarness) RetryResumeApproval(c *gin.Context) {
	h.ApprovalHandler.RetryResumeApproval(c)
}
func (h *HandlerHarness) GetApproval(c *gin.Context) { h.ApprovalHandler.GetApproval(c) }
func (h *HandlerHarness) ListPendingApprovals(c *gin.Context) {
	h.ApprovalHandler.ListPendingApprovals(c)
}
func (h *HandlerHarness) StartApprovalWorker(ctx context.Context) {
	h.ApprovalHandler.StartApprovalWorker(ctx)
}
func (h *HandlerHarness) StartApprovalExpirer(ctx context.Context) {
	h.ApprovalHandler.StartApprovalExpirer(ctx)
}

func NewAIHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&ai.AIChatSession{},
		&ai.AIChatMessage{},
		&ai.AIRun{},
		&ai.AIRunEvent{},
		&ai.AIRunProjection{},
		&ai.AIRunContent{},
		&ai.AIDiagnosisReport{},
		&ai.AIScenePrompt{},
		&ai.AISceneConfig{},
		&ai.AIApprovalTask{},
		&ai.AIApprovalOutboxEvent{},
	); err != nil {
		t.Fatalf("auto migrate ai handler tables: %v", err)
	}
	return db
}

func RegisterAIHandlersForTest(v1 *gin.RouterGroup) {
	h := NewAIHandlerTestHarness(nil)
	g := v1.Group("/ai")
	{
		g.POST("/chat", h.Chat)
		g.GET("/sessions", h.ListSessions)
		g.POST("/sessions", h.CreateSession)
		g.GET("/sessions/:id", h.GetSession)
		g.DELETE("/sessions/:id", h.DeleteSession)
		g.GET("/runs/:runId", h.GetRun)
		g.GET("/runs/:runId/projection", h.GetRunProjection)
		g.GET("/run-contents/:id", h.GetRunContent)
		g.GET("/diagnosis/:reportId", h.GetDiagnosisReport)
		g.GET("/approvals/pending", h.ListPendingApprovals)
		g.GET("/approvals/:id", h.GetApproval)
		g.POST("/approvals/:id/submit", h.SubmitApproval)
		g.POST("/approvals/:id/retry-resume", h.RetryResumeApproval)
	}
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
