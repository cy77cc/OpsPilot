package handler

import (
	"context"
	"io"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/service/ai/approval"
	"github.com/cy77cc/OpsPilot/internal/service/ai/chat"
	"github.com/cy77cc/OpsPilot/internal/service/ai/logic"
	aimodel "github.com/cy77cc/OpsPilot/internal/service/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler is a compatibility facade. New code should use domain handlers directly.
type Handler struct {
	svcCtx *svc.ServiceContext
	logic  *logic.Logic

	workerMu     sync.Mutex
	workerStart  bool
	workerCancel context.CancelFunc

	expirerMu     sync.Mutex
	expirerStart  bool
	expirerCancel context.CancelFunc

	chatHandler     *chat.HTTPHandler
	approvalHandler *approval.HTTPHandler
}

func NewAIHandler(svcCtx *svc.ServiceContext) *Handler {
	l := logic.NewAILogic(svcCtx)
	return &Handler{
		svcCtx:          svcCtx,
		logic:           l,
		chatHandler:     chat.NewHTTPHandler(chat.NewServiceWithLogic(l)),
		approvalHandler: approval.NewHTTPHandler(approval.NewServiceWithLogic(l)),
	}
}

func NewAIHandlerWithDB(db *gorm.DB) *Handler {
	svcCtx := &svc.ServiceContext{DB: db}
	h := NewAIHandler(svcCtx)
	h.logic = logic.NewLogicWithDB(db, &noopAgent{})
	h.chatHandler = chat.NewHTTPHandler(chat.NewServiceWithLogic(h.logic))
	h.approvalHandler = approval.NewHTTPHandler(approval.NewServiceWithLogic(h.logic))
	return h
}

// noopAgent is used in tests.
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

func (h *Handler) Chat(c *gin.Context)                 { h.chatHandler.Chat(c) }
func (h *Handler) CreateSession(c *gin.Context)        { h.chatHandler.CreateSession(c) }
func (h *Handler) ListSessions(c *gin.Context)         { h.chatHandler.ListSessions(c) }
func (h *Handler) GetSession(c *gin.Context)           { h.chatHandler.GetSession(c) }
func (h *Handler) DeleteSession(c *gin.Context)        { h.chatHandler.DeleteSession(c) }
func (h *Handler) GetRun(c *gin.Context)               { h.chatHandler.GetRun(c) }
func (h *Handler) GetRunProjection(c *gin.Context)     { h.chatHandler.GetRunProjection(c) }
func (h *Handler) GetRunContent(c *gin.Context)        { h.chatHandler.GetRunContent(c) }
func (h *Handler) GetDiagnosisReport(c *gin.Context)   { h.chatHandler.GetDiagnosisReport(c) }
func (h *Handler) SubmitApproval(c *gin.Context)       { h.approvalHandler.SubmitApproval(c) }
func (h *Handler) RetryResumeApproval(c *gin.Context)  { h.approvalHandler.RetryResumeApproval(c) }
func (h *Handler) GetApproval(c *gin.Context)          { h.approvalHandler.GetApproval(c) }
func (h *Handler) ListPendingApprovals(c *gin.Context) { h.approvalHandler.ListPendingApprovals(c) }
func (h *Handler) StartApprovalWorker(ctx context.Context) {
	h.approvalHandler.StartApprovalWorker(ctx)
}
func (h *Handler) StartApprovalExpirer(ctx context.Context) {
	h.approvalHandler.StartApprovalExpirer(ctx)
}

// LLMProviderHandler compatibility type.
type LLMProviderHandler = aimodel.HTTPHandler

func NewLLMProviderHandler(svcCtx *svc.ServiceContext) *LLMProviderHandler {
	return aimodel.NewHTTPHandler(svcCtx)
}

func NewLLMProviderHandlerWithDB(db *gorm.DB) *LLMProviderHandler {
	return aimodel.NewHTTPHandlerWithDB(db)
}

// SSE writer compatibility exports.
type SSEWriter = chat.SSEWriter

func NewSSEWriter(writer io.Writer) *SSEWriter {
	return chat.NewSSEWriter(writer)
}
