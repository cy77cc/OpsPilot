package logic

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/ai/agents"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/google/uuid"
)


type Logic struct {
	svcCtx             *svc.ServiceContext
	ChatDAO            *aidao.AIChatDAO
	RunDAO             *aidao.AIRunDAO
	DiagnosisReportDAO *aidao.AIDiagnosisReportDAO
	AIRouter           adk.ResumableAgent
}

func NewAILogic(svcCtx *svc.ServiceContext) *Logic {
	if svcCtx == nil || svcCtx.DB == nil {
		return &Logic{}
	}
	aiRouter, err := agents.NewRouter(svcCtx.GetContext())
	if err != nil {
		return &Logic{}
	}
	return &Logic{
		svcCtx:             svcCtx,
		ChatDAO:            aidao.NewAIChatDAO(svcCtx.DB),
		RunDAO:             aidao.NewAIRunDAO(svcCtx.DB),
		DiagnosisReportDAO: aidao.NewAIDiagnosisReportDAO(svcCtx.DB),
		AIRouter:           aiRouter,
	}
}


func (l *Logic) Chat(ctx context.Context, input ChatInput, emit EventEmitter) error {
	if l.ChatDAO == nil || l.RunDAO == nil || l.DiagnosisReportDAO == nil {
		return nil
	}

	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = uuid.NewString()
	}

	if session, err := l.ChatDAO.GetSession(ctx, sessionID, input.UserID); err != nil {
		return err
	} else if session == nil {
		if err := l.ChatDAO.CreateSession(ctx, &model.AIChatSession{
			ID:     sessionID,
			UserID: input.UserID,
			Scene:  "ai",
			Title:  buildSessionTitle(input.Message),
		}); err != nil {
			return err
		}
	}

	userMessage := &model.AIChatMessage{
		ID:        uuid.NewString(),
		SessionID: sessionID,
		Role:      "user",
		Content:   input.Message,
		Status:    "done",
	}
	if err := l.ChatDAO.CreateMessage(ctx, userMessage); err != nil {
		return err
	}

	run := &model.AIRun{
		ID:            uuid.NewString(),
		SessionID:     sessionID,
		UserMessageID: userMessage.ID,
		Status:        "running",
	}
	if err := l.RunDAO.CreateRun(ctx, run); err != nil {
		return err
	}

	emit("init", map[string]any{"session_id": sessionID, "run_id": run.ID})

	decision, err := l.IntentRouter.Route(ctx, input.Message)
	if err != nil {
		_ = l.RunDAO.UpdateRunStatus(ctx, run.ID, aidao.AIRunStatusUpdate{Status: "failed", ErrorMessage: err.Error()})
		emit("error", map[string]any{"message": err.Error()})
		return nil
	}
	emit("intent", map[string]any{"intent_type": decision.IntentType, "assistant_type": decision.AssistantType, "risk_level": decision.RiskLevel})
	emit("status", map[string]any{"status": "running"})

	assistantMessage := &model.AIChatMessage{
		ID:        uuid.NewString(),
		SessionID: sessionID,
		Role:      "assistant",
		Status:    "streaming",
	}
	if err := l.ChatDAO.CreateMessage(ctx, assistantMessage); err != nil {
		return err
	}

	switch decision.IntentType {
	case IntentTypeDiagnosis:
		result, err := l.DiagnosisAgent.Diagnose(ctx, DiagnosisRequest{Message: input.Message})
		if err != nil {
			_ = l.RunDAO.UpdateRunStatus(ctx, run.ID, aidao.AIRunStatusUpdate{Status: "failed", ErrorMessage: err.Error(), AssistantMessageID: assistantMessage.ID})
			emit("error", map[string]any{"message": err.Error()})
			return nil
		}
		for _, progress := range result.Progress {
			emit("progress", map[string]any{"summary": progress})
		}
		report := &model.AIDiagnosisReport{
			ID:                  uuid.NewString(),
			RunID:               run.ID,
			SessionID:           sessionID,
			Summary:             result.Report.Summary,
			EvidenceJSON:        mustJSON(result.Report.Evidence),
			RootCausesJSON:      mustJSON(result.Report.RootCauses),
			RecommendationsJSON: mustJSON(result.Report.Recommendations),
			GeneratedAt:         time.Now().UTC(),
		}
		if err := l.DiagnosisReportDAO.CreateReport(ctx, report); err != nil {
			return err
		}
		if err := l.ChatDAO.UpdateMessage(ctx, assistantMessage.ID, map[string]any{"content": result.Report.Summary, "status": "done"}); err != nil {
			return err
		}
		if err := l.RunDAO.UpdateRunStatus(ctx, run.ID, aidao.AIRunStatusUpdate{
			Status:             "completed",
			AssistantMessageID: assistantMessage.ID,
			ProgressSummary:    result.Report.Summary,
		}); err != nil {
			return err
		}
		emit("report_ready", map[string]any{"report_id": report.ID, "summary": report.Summary})
	default:
		result, err := l.QAAgent.Answer(ctx, QARequest{Message: input.Message})
		if err != nil {
			_ = l.RunDAO.UpdateRunStatus(ctx, run.ID, aidao.AIRunStatusUpdate{Status: "failed", ErrorMessage: err.Error(), AssistantMessageID: assistantMessage.ID})
			emit("error", map[string]any{"message": err.Error()})
			return nil
		}
		if err := l.ChatDAO.UpdateMessage(ctx, assistantMessage.ID, map[string]any{"content": result.Text, "status": "done"}); err != nil {
			return err
		}
		if err := l.RunDAO.UpdateRunStatus(ctx, run.ID, aidao.AIRunStatusUpdate{
			Status:             "completed",
			AssistantMessageID: assistantMessage.ID,
			ProgressSummary:    result.Text,
		}); err != nil {
			return err
		}
		emit("delta", map[string]any{"content": result.Text})
	}

	emit("done", map[string]any{"run_id": run.ID, "status": "completed"})
	return nil
}

func (l *Logic) CreateSession(ctx context.Context, userID uint64, title, scene string) (*model.AIChatSession, error) {
	s := &model.AIChatSession{
		ID:     uuid.NewString(),
		UserID: userID,
		Title:  title,
		Scene:  scene,
	}
	if l.ChatDAO != nil {
		if err := l.ChatDAO.CreateSession(ctx, s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (l *Logic) ListSessions(ctx context.Context, userID uint64, scene string) ([]model.AIChatSession, map[string][]model.AIChatMessage, error) {
	if l.ChatDAO == nil {
		return []model.AIChatSession{}, map[string][]model.AIChatMessage{}, nil
	}
	rows, err := l.ChatDAO.ListSessions(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	filtered := make([]model.AIChatSession, 0, len(rows))
	if strings.TrimSpace(scene) == "" {
		filtered = rows
	} else {
		for _, row := range rows {
			if row.Scene == scene {
				filtered = append(filtered, row)
			}
		}
	}
	messagesBySession := make(map[string][]model.AIChatMessage, len(filtered))
	for _, session := range filtered {
		messages, err := l.ChatDAO.ListMessagesBySession(ctx, session.ID)
		if err != nil {
			return nil, nil, err
		}
		messagesBySession[session.ID] = messages
	}
	return filtered, messagesBySession, nil
}

func (l *Logic) GetSession(ctx context.Context, userID uint64, sessionID string) (*model.AIChatSession, []model.AIChatMessage, error) {
	if l.ChatDAO == nil {
		return nil, nil, nil
	}
	session, err := l.ChatDAO.GetSession(ctx, sessionID, userID)
	if err != nil || session == nil {
		return session, nil, err
	}
	messages, err := l.ChatDAO.ListMessagesBySession(ctx, session.ID)
	if err != nil {
		return nil, nil, err
	}
	return session, messages, nil
}

func (l *Logic) DeleteSession(ctx context.Context, userID uint64, sessionID string) (bool, error) {
	if l.ChatDAO == nil {
		return false, nil
	}
	session, err := l.ChatDAO.GetSession(ctx, sessionID, userID)
	if err != nil {
		return false, err
	}
	if session == nil {
		return false, nil
	}
	if err := l.ChatDAO.DeleteSession(ctx, session.ID, userID); err != nil {
		return false, err
	}
	return true, nil
}

func (l *Logic) GetRun(ctx context.Context, userID uint64, runID string) (*model.AIRun, *model.AIDiagnosisReport, error) {
	if l.RunDAO == nil {
		return nil, nil, nil
	}
	run, err := l.RunDAO.GetRun(ctx, runID)
	if err != nil || run == nil {
		return run, nil, err
	}
	if l.ChatDAO != nil {
		session, err := l.ChatDAO.GetSession(ctx, run.SessionID, userID)
		if err != nil {
			return nil, nil, err
		}
		if session == nil {
			return nil, nil, nil
		}
	}
	if l.DiagnosisReportDAO == nil {
		return run, nil, nil
	}
	report, err := l.DiagnosisReportDAO.GetReportByRunID(ctx, run.ID)
	if err != nil {
		return nil, nil, err
	}
	return run, report, nil
}

func (l *Logic) GetDiagnosisReport(ctx context.Context, userID uint64, reportID string) (*model.AIDiagnosisReport, error) {
	if l.DiagnosisReportDAO == nil {
		return nil, nil
	}
	report, err := l.DiagnosisReportDAO.GetReport(ctx, reportID)
	if err != nil || report == nil {
		return report, err
	}
	if l.ChatDAO != nil {
		session, err := l.ChatDAO.GetSession(ctx, report.SessionID, userID)
		if err != nil {
			return nil, err
		}
		if session == nil {
			return nil, nil
		}
	}
	return report, nil
}

type defaultIntentRouter struct{}

func (defaultIntentRouter) Route(_ context.Context, message string) (Decision, error) {
	lower := strings.ToLower(strings.TrimSpace(message))
	if lower == "" {
		return Decision{IntentType: IntentTypeQA, AssistantType: "qa", RiskLevel: "low"}, nil
	}
	if strings.Contains(lower, "诊断") || strings.Contains(lower, "排查") || strings.Contains(lower, "日志") || strings.Contains(lower, "异常") || strings.Contains(lower, "故障") {
		return Decision{IntentType: IntentTypeDiagnosis, AssistantType: "diagnosis", RiskLevel: "medium"}, nil
	}
	if strings.Contains(lower, "扩容") || strings.Contains(lower, "缩容") || strings.Contains(lower, "重启") || strings.Contains(lower, "回滚") || strings.Contains(lower, "删除") || strings.Contains(lower, "scale") || strings.Contains(lower, "restart") || strings.Contains(lower, "rollback") || strings.Contains(lower, "delete") {
		return Decision{IntentType: IntentTypeChange, AssistantType: "change", RiskLevel: "high"}, nil
	}
	return Decision{IntentType: IntentTypeQA, AssistantType: "qa", RiskLevel: "low"}, nil
}

type defaultQAAgent struct{}

func (defaultQAAgent) Answer(_ context.Context, req QARequest) (QAResult, error) {
	msg := strings.TrimSpace(req.Message)
	if msg == "" {
		msg = "your question"
	}
	return QAResult{Text: "Phase 1 QA answer: " + msg}, nil
}

type defaultDiagnosisAgent struct{}

func (defaultDiagnosisAgent) Diagnose(_ context.Context, req DiagnosisRequest) (DiagnosisResult, error) {
	msg := strings.TrimSpace(req.Message)
	if msg == "" {
		msg = "cluster issue"
	}
	return DiagnosisResult{
		Progress: []string{"Collecting diagnostic evidence", "Analyzing root causes"},
		Report: DiagnosisReport{
			Summary:         "Preliminary diagnosis for: " + msg,
			Evidence:        []string{"No live probe integrated in fallback diagnosis agent"},
			RootCauses:      []string{"Insufficient runtime integration in fallback path"},
			Recommendations: []string{"Enable full runtime-based diagnosis agent integration"},
		},
	}, nil
}

func buildSessionTitle(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "New AI session"
	}
	if len(trimmed) > 48 {
		return trimmed[:48]
	}
	return trimmed
}

func mustJSON(value any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
