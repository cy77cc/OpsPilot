package chathandler

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

// Service provides chat/session/run/diagnosis use cases.
type Service struct {
	logic       *logic.Logic
	RunDAO      *aidao.AIRunDAO
	RunEventDAO *aidao.AIRunEventDAO
	router      RouteService
}

func NewService(svcCtx *svc.ServiceContext) *Service {
	l := logic.NewAILogic(svcCtx)
	return NewServiceWithLogic(l)
}

func NewServiceWithLogic(l *logic.Logic) *Service {
	if l == nil {
		return &Service{}
	}
	return &Service{
		logic:       l,
		RunDAO:      l.RunDAO,
		RunEventDAO: l.RunEventDAO,
	}
}

func NewServiceWithLogicAndRouter(l *logic.Logic, routeSvc RouteService) *Service {
	if routeSvc == nil {
		routeSvc = NewRouteService()
	}
	if l == nil {
		return &Service{router: routeSvc}
	}
	return &Service{
		logic:       l,
		RunDAO:      l.RunDAO,
		RunEventDAO: l.RunEventDAO,
		router:      routeSvc,
	}
}

func NewServiceWithDB(db *gorm.DB, agentRouter adk.ResumableAgent) *Service {
	l := logic.NewLogicWithDB(db, agentRouter)
	return NewServiceWithLogic(l)
}

func (s *Service) Chat(ctx context.Context, input logic.ChatInput, emit logic.EventEmitter) error {
	decision, err := s.decideRoute(ctx, input.Message)
	if err != nil {
		return err
	}
	if decision.Mode == ModeConversation {
		if emit != nil {
			emit("status", map[string]any{
				"mode":    string(ModeConversation),
				"state":   "deferred",
				"reason":  "not_implemented",
				"message": "conversation path is deferred until the dedicated copilot flow is implemented",
			})
		}
		return nil
	}
	if s == nil || s.logic == nil {
		return fmt.Errorf("AI service not initialized")
	}
	return s.logic.Chat(ctx, input, emit)
}

func (s *Service) decideRoute(ctx context.Context, message string) (RouteDecision, error) {
	if s == nil || s.router == nil {
		return RouteDecision{}, nil
	}
	decision, err := s.router.Decide(ctx, RouteInput{
		Message: message,
	})
	if err != nil {
		return RouteDecision{}, err
	}
	if err := decision.Validate(); err != nil {
		return RouteDecision{}, err
	}
	return decision, nil
}

func (s *Service) CreateSession(ctx context.Context, userID uint64, title, scene string) (*ai.AIChatSession, error) {
	return s.logic.CreateSession(ctx, userID, title, scene)
}

func (s *Service) ListSessions(ctx context.Context, userID uint64, scene string) ([]logic.SessionSummary, error) {
	return s.logic.ListSessions(ctx, userID, scene)
}

func (s *Service) GetSession(ctx context.Context, userID uint64, scene, sessionID string) (*ai.AIChatSession, []ai.AIChatMessage, error) {
	return s.logic.GetSession(ctx, userID, scene, sessionID)
}

func (s *Service) DeleteSession(ctx context.Context, userID uint64, sessionID string) (bool, error) {
	return s.logic.DeleteSession(ctx, userID, sessionID)
}

func (s *Service) GetRun(ctx context.Context, userID uint64, runID string) (*ai.AIRun, *ai.AIDiagnosisReport, error) {
	return s.logic.GetRun(ctx, userID, runID)
}

func (s *Service) BuildResumableCredentials(ctx context.Context, run *ai.AIRun) (*logic.ResumableCredentials, error) {
	if s == nil || s.logic == nil {
		return nil, nil
	}
	return s.logic.BuildResumableCredentials(ctx, run)
}

func (s *Service) GetRunProjectionPayload(ctx context.Context, userID uint64, runID string, query logic.RunProjectionQuery) (any, error) {
	return s.logic.GetRunProjectionPayload(ctx, userID, runID, query)
}

func (s *Service) GetRunContent(ctx context.Context, userID uint64, contentID string) (*ai.AIRunContent, error) {
	return s.logic.GetRunContent(ctx, userID, contentID)
}

func (s *Service) GetDiagnosisReport(ctx context.Context, userID uint64, reportID string) (*ai.AIDiagnosisReport, error) {
	return s.logic.GetDiagnosisReport(ctx, userID, reportID)
}
