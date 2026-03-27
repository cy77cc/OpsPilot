package chat

import (
	"context"

	"github.com/cloudwego/eino/adk"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

// Service provides chat/session/run/diagnosis use cases.
type Service struct {
	logic       *logic.Logic
	RunDAO      *aidao.AIRunDAO
	RunEventDAO *aidao.AIRunEventDAO
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

func NewServiceWithDB(db *gorm.DB, router adk.ResumableAgent) *Service {
	l := logic.NewLogicWithDB(db, router)
	return NewServiceWithLogic(l)
}

func (s *Service) Chat(ctx context.Context, input logic.ChatInput, emit logic.EventEmitter) error {
	return s.logic.Chat(ctx, input, emit)
}

func (s *Service) CreateSession(ctx context.Context, userID uint64, title, scene string) (*model.AIChatSession, error) {
	return s.logic.CreateSession(ctx, userID, title, scene)
}

func (s *Service) ListSessions(ctx context.Context, userID uint64, scene string) ([]logic.SessionSummary, error) {
	return s.logic.ListSessions(ctx, userID, scene)
}

func (s *Service) GetSession(ctx context.Context, userID uint64, scene, sessionID string) (*model.AIChatSession, []model.AIChatMessage, error) {
	return s.logic.GetSession(ctx, userID, scene, sessionID)
}

func (s *Service) DeleteSession(ctx context.Context, userID uint64, sessionID string) (bool, error) {
	return s.logic.DeleteSession(ctx, userID, sessionID)
}

func (s *Service) GetRun(ctx context.Context, userID uint64, runID string) (*model.AIRun, *model.AIDiagnosisReport, error) {
	return s.logic.GetRun(ctx, userID, runID)
}

func (s *Service) BuildResumableCredentials(ctx context.Context, run *model.AIRun) (*logic.ResumableCredentials, error) {
	if s == nil || s.logic == nil {
		return nil, nil
	}
	return s.logic.BuildResumableCredentials(ctx, run)
}

func (s *Service) GetRunProjectionPayload(ctx context.Context, userID uint64, runID string, query logic.RunProjectionQuery) (any, error) {
	return s.logic.GetRunProjectionPayload(ctx, userID, runID, query)
}

func (s *Service) GetRunContent(ctx context.Context, userID uint64, contentID string) (*model.AIRunContent, error) {
	return s.logic.GetRunContent(ctx, userID, contentID)
}

func (s *Service) GetDiagnosisReport(ctx context.Context, userID uint64, reportID string) (*model.AIDiagnosisReport, error) {
	return s.logic.GetDiagnosisReport(ctx, userID, reportID)
}
