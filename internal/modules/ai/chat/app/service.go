package app

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	routing "github.com/cy77cc/OpsPilot/internal/modules/ai/routing"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

type RouteService interface {
	Decide(ctx context.Context, input routing.RouteInput) (routing.RouteDecision, error)
}

// Service provides chat/session/run/diagnosis use cases.
type Service struct {
	logic       *logic.Logic
	RunDAO      *aidao.AIRunDAO
	RunEventDAO *aidao.AIRunEventDAO
	router      RouteService
}

func NewService(svcCtx *svc.ServiceContext) *Service {
	l := logic.NewAILogic(svcCtx)
	return NewServiceWithLogicAndRouter(l, routing.NewService())
}

func NewServiceWithLogic(l *logic.Logic) *Service {
	return NewServiceWithLogicAndRouter(l, routing.NewService())
}

func NewServiceWithLogicAndRouter(l *logic.Logic, routeSvc RouteService) *Service {
	if routeSvc == nil {
		routeSvc = routing.NewService()
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
	return NewServiceWithLogicAndRouter(l, routing.NewService())
}

func (s *Service) Chat(ctx context.Context, input logic.ChatInput, emit logic.EventEmitter) error {
	decision, err := s.decideRoute(ctx, input.Message)
	if err != nil {
		return err
	}
	if decision.Mode == routing.ModeConversation {
		if emit != nil {
			emit("status", map[string]any{
				"mode":    string(routing.ModeConversation),
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

func (s *Service) decideRoute(ctx context.Context, message string) (routing.RouteDecision, error) {
	if s == nil || s.router == nil {
		return routing.RouteDecision{}, nil
	}
	decision, err := s.router.Decide(ctx, routing.RouteInput{
		Message: message,
	})
	if err != nil {
		return routing.RouteDecision{}, err
	}
	if err := decision.Validate(); err != nil {
		return routing.RouteDecision{}, err
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
