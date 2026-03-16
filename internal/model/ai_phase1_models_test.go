package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAIPhase1Models_AutoMigrateCreatesRunAndReportTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&AIChatSession{}, &AIChatMessage{}, &AIRun{}, &AIDiagnosisReport{}); err != nil {
		t.Fatalf("auto migrate phase 1 models: %v", err)
	}

	for _, table := range []string{"ai_chat_sessions", "ai_chat_messages", "ai_runs", "ai_diagnosis_reports"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table %s to exist", table)
		}
	}
}

func TestAIRun_PersistsPhase1Fields(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&AIRun{}); err != nil {
		t.Fatalf("auto migrate ai run: %v", err)
	}

	run := AIRun{
		ID:                 "run-1",
		SessionID:          "session-1",
		UserMessageID:      "msg-user-1",
		AssistantMessageID: "msg-assistant-1",
		IntentType:         "diagnosis",
		AssistantType:      "diagnosis",
		RiskLevel:          "low",
		Status:             "running",
		TraceID:            "trace-1",
		ErrorMessage:       "",
		ProgressSummary:    "Collecting evidence",
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatalf("create ai run: %v", err)
	}

	var got AIRun
	if err := db.First(&got, "id = ?", run.ID).Error; err != nil {
		t.Fatalf("load ai run: %v", err)
	}

	if got.SessionID != run.SessionID {
		t.Fatalf("expected session id %q, got %q", run.SessionID, got.SessionID)
	}
	if got.UserMessageID != run.UserMessageID {
		t.Fatalf("expected user message id %q, got %q", run.UserMessageID, got.UserMessageID)
	}
	if got.AssistantMessageID != run.AssistantMessageID {
		t.Fatalf("expected assistant message id %q, got %q", run.AssistantMessageID, got.AssistantMessageID)
	}
	if got.IntentType != run.IntentType {
		t.Fatalf("expected intent type %q, got %q", run.IntentType, got.IntentType)
	}
	if got.AssistantType != run.AssistantType {
		t.Fatalf("expected assistant type %q, got %q", run.AssistantType, got.AssistantType)
	}
	if got.RiskLevel != run.RiskLevel {
		t.Fatalf("expected risk level %q, got %q", run.RiskLevel, got.RiskLevel)
	}
	if got.Status != run.Status {
		t.Fatalf("expected status %q, got %q", run.Status, got.Status)
	}
	if got.TraceID != run.TraceID {
		t.Fatalf("expected trace id %q, got %q", run.TraceID, got.TraceID)
	}
	if got.ProgressSummary != run.ProgressSummary {
		t.Fatalf("expected progress summary %q, got %q", run.ProgressSummary, got.ProgressSummary)
	}
}

func TestAIDiagnosisReport_PersistsRunScopedFields(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&AIDiagnosisReport{}); err != nil {
		t.Fatalf("auto migrate diagnosis report: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	report := AIDiagnosisReport{
		ID:                 "report-1",
		RunID:              "run-1",
		SessionID:          "session-1",
		Summary:            "Cluster rollout blocked by quota exhaustion.",
		EvidenceJSON:       `["quota usage at 100%"]`,
		RootCausesJSON:     `["namespace quota exhausted"]`,
		RecommendationsJSON:`["increase quota or free resources"]`,
		GeneratedAt:        now,
	}
	if err := db.Create(&report).Error; err != nil {
		t.Fatalf("create diagnosis report: %v", err)
	}

	var got AIDiagnosisReport
	if err := db.First(&got, "id = ?", report.ID).Error; err != nil {
		t.Fatalf("load diagnosis report: %v", err)
	}

	if got.RunID != report.RunID {
		t.Fatalf("expected run id %q, got %q", report.RunID, got.RunID)
	}
	if got.SessionID != report.SessionID {
		t.Fatalf("expected session id %q, got %q", report.SessionID, got.SessionID)
	}
	if got.Summary != report.Summary {
		t.Fatalf("expected summary %q, got %q", report.Summary, got.Summary)
	}
	if got.EvidenceJSON != report.EvidenceJSON {
		t.Fatalf("expected evidence json %q, got %q", report.EvidenceJSON, got.EvidenceJSON)
	}
	if got.RootCausesJSON != report.RootCausesJSON {
		t.Fatalf("expected root causes json %q, got %q", report.RootCausesJSON, got.RootCausesJSON)
	}
	if got.RecommendationsJSON != report.RecommendationsJSON {
		t.Fatalf("expected recommendations json %q, got %q", report.RecommendationsJSON, got.RecommendationsJSON)
	}
}
