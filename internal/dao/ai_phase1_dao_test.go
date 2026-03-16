package dao

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAIPhase1DAOTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.AIChatSession{},
		&model.AIChatMessage{},
		&model.AIRun{},
		&model.AIDiagnosisReport{},
	); err != nil {
		t.Fatalf("auto migrate ai phase1 tables: %v", err)
	}
	return db
}

func TestAIChatDAO_CreateListGetDeleteSession(t *testing.T) {
	ctx := context.Background()
	db := newAIPhase1DAOTestDB(t)
	dao := NewAIChatDAO(db)

	session := &model.AIChatSession{
		ID:     "session-1",
		UserID: 42,
		Scene:  "cluster",
		Title:  "Cluster diagnosis",
	}
	if err := dao.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	sessions, err := dao.ListSessions(ctx, 42)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected one session, got %d", len(sessions))
	}

	got, err := dao.GetSession(ctx, "session-1", 42)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got == nil || got.Title != session.Title {
		t.Fatalf("unexpected session: %#v", got)
	}

	if err := dao.DeleteSession(ctx, "session-1", 42); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	got, err = dao.GetSession(ctx, "session-1", 42)
	if err != nil {
		t.Fatalf("get deleted session: %v", err)
	}
	if got != nil {
		t.Fatalf("expected deleted session to be nil, got %#v", got)
	}
}

func TestAIChatDAO_CreateAndListMessagesBySession(t *testing.T) {
	ctx := context.Background()
	db := newAIPhase1DAOTestDB(t)
	chatDAO := NewAIChatDAO(db)

	if err := chatDAO.CreateSession(ctx, &model.AIChatSession{
		ID:     "session-1",
		UserID: 7,
		Scene:  "cluster",
		Title:  "Session 1",
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	userMsg := &model.AIChatMessage{
		ID:        "message-1",
		SessionID: "session-1",
		Role:      "user",
		Content:   "What is wrong with the rollout?",
		Status:    "done",
	}
	assistantMsg := &model.AIChatMessage{
		ID:        "message-2",
		SessionID: "session-1",
		Role:      "assistant",
		Content:   "Investigating",
		Status:    "streaming",
	}

	if err := chatDAO.CreateMessage(ctx, userMsg); err != nil {
		t.Fatalf("create user message: %v", err)
	}
	if err := chatDAO.CreateMessage(ctx, assistantMsg); err != nil {
		t.Fatalf("create assistant message: %v", err)
	}

	messages, err := chatDAO.ListMessagesBySession(ctx, "session-1")
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected two messages, got %d", len(messages))
	}
	if messages[0].ID != userMsg.ID || messages[1].ID != assistantMsg.ID {
		t.Fatalf("unexpected message ordering: %#v", messages)
	}
}

func TestAIRunDAO_CreateUpdateAndGetRun(t *testing.T) {
	ctx := context.Background()
	db := newAIPhase1DAOTestDB(t)
	runDAO := NewAIRunDAO(db)

	run := &model.AIRun{
		ID:            "run-1",
		SessionID:     "session-1",
		UserMessageID: "message-1",
		IntentType:    "diagnosis",
		AssistantType: "diagnosis",
		RiskLevel:     "low",
		Status:        "queued",
	}
	if err := runDAO.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	if err := runDAO.UpdateRunStatus(ctx, "run-1", AIRunStatusUpdate{
		Status:          "completed",
		AssistantMessageID: "message-2",
		ProgressSummary: "Diagnosis complete",
	}); err != nil {
		t.Fatalf("update run status: %v", err)
	}

	got, err := runDAO.GetRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got == nil {
		t.Fatalf("expected run to exist")
	}
	if got.Status != "completed" {
		t.Fatalf("expected completed status, got %q", got.Status)
	}
	if got.AssistantMessageID != "message-2" {
		t.Fatalf("expected assistant message id to update, got %q", got.AssistantMessageID)
	}
	if got.ProgressSummary != "Diagnosis complete" {
		t.Fatalf("expected progress summary to update, got %q", got.ProgressSummary)
	}
}

func TestAIDiagnosisReportDAO_CreateAndLookupByIDs(t *testing.T) {
	ctx := context.Background()
	db := newAIPhase1DAOTestDB(t)
	reportDAO := NewAIDiagnosisReportDAO(db)

	report := &model.AIDiagnosisReport{
		ID:                  "report-1",
		RunID:               "run-1",
		SessionID:           "session-1",
		Summary:             "Rollout blocked by quota",
		EvidenceJSON:        `["quota: hard limit reached"]`,
		RootCausesJSON:      `["namespace quota exhausted"]`,
		RecommendationsJSON: `["increase quota"]`,
	}
	if err := reportDAO.CreateReport(ctx, report); err != nil {
		t.Fatalf("create report: %v", err)
	}

	gotByID, err := reportDAO.GetReport(ctx, "report-1")
	if err != nil {
		t.Fatalf("get report by id: %v", err)
	}
	if gotByID == nil || gotByID.RunID != report.RunID {
		t.Fatalf("unexpected report by id: %#v", gotByID)
	}

	gotByRunID, err := reportDAO.GetReportByRunID(ctx, "run-1")
	if err != nil {
		t.Fatalf("get report by run id: %v", err)
	}
	if gotByRunID == nil || gotByRunID.ID != report.ID {
		t.Fatalf("unexpected report by run id: %#v", gotByRunID)
	}
}
