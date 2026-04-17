package chathandler

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/adk"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

// Service provides chat/session/run/diagnosis use cases.
type Service struct {
	logic *logic.Logic
}

func LegacyRoutingEnabled() bool {
	return false
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
		logic: l,
	}
}

func NewServiceWithDB(db *gorm.DB, agentRouter adk.ResumableAgent) *Service {
	l := logic.NewLogicWithDB(db, agentRouter)
	return NewServiceWithLogic(l)
}

func (s *Service) Chat(ctx context.Context, input logic.ChatInput, emit logic.EventEmitter) error {
	if s == nil || s.logic == nil {
		return nil
	}
	return s.logic.Chat(ctx, input, emit)
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

func (s *Service) ValidateReplayCursor(ctx context.Context, sessionID, clientRequestID, lastEventID string) error {
	if strings.TrimSpace(lastEventID) == "" {
		return nil
	}
	if s == nil || s.logic == nil || s.logic.RunDAO == nil || s.logic.RunEventDAO == nil {
		return aidao.ErrRunEventCursorExpired
	}
	run, err := s.logic.RunDAO.FindByClientRequestID(ctx, sessionID, clientRequestID)
	if err != nil {
		return err
	}
	if run == nil {
		return aidao.ErrRunEventCursorExpired
	}
	cursor, err := s.logic.RunEventDAO.FindByEventID(ctx, run.ID, lastEventID)
	if err != nil {
		return err
	}
	if cursor == nil {
		return aidao.ErrRunEventCursorExpired
	}
	return nil
}

func (s *Service) RunByAssistantMessageID(ctx context.Context, sessionID string) map[string]*ai.AIRun {
	result := map[string]*ai.AIRun{}
	if s == nil || s.logic == nil || s.logic.RunDAO == nil {
		return result
	}
	runs, err := s.logic.RunDAO.ListBySession(ctx, sessionID)
	if err != nil {
		return result
	}
	for _, run := range runs {
		if strings.TrimSpace(run.AssistantMessageID) != "" {
			runCopy := run
			result[run.AssistantMessageID] = &runCopy
		}
	}
	return result
}

func (s *Service) RunBySessionAndAssistantMessageID(ctx context.Context, sessions []ai.AIChatSession) map[string]map[string]*ai.AIRun {
	result := map[string]map[string]*ai.AIRun{}
	if s == nil || s.logic == nil || s.logic.RunDAO == nil || len(sessions) == 0 {
		return result
	}
	sessionIDs := make([]string, 0, len(sessions))
	for _, session := range sessions {
		sessionIDs = append(sessionIDs, session.ID)
		result[session.ID] = map[string]*ai.AIRun{}
	}
	runs, err := s.logic.RunDAO.ListBySessionIDs(ctx, sessionIDs)
	if err != nil {
		return result
	}
	for _, run := range runs {
		if strings.TrimSpace(run.AssistantMessageID) == "" {
			continue
		}
		if _, ok := result[run.SessionID]; !ok {
			result[run.SessionID] = map[string]*ai.AIRun{}
		}
		runCopy := run
		result[run.SessionID][run.AssistantMessageID] = &runCopy
	}
	return result
}
