package ai

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
)

func TestUpdateRunStatus_SetsFinishedAtForTerminalStates(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAIRunDAO(db)
	ctx := context.Background()

	run := &model.AIRun{
		ID:            "run-1",
		SessionID:     "session-1",
		UserMessageID: "msg-1",
		Status:        "running",
		TraceJSON:     "{}",
	}
	if err := dao.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	if err := dao.UpdateRunStatus(ctx, run.ID, AIRunStatusUpdate{
		Status:          "completed",
		ProgressSummary: "done",
	}); err != nil {
		t.Fatalf("update run status: %v", err)
	}

	refreshed, err := dao.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if refreshed == nil || refreshed.FinishedAt == nil || refreshed.FinishedAt.IsZero() {
		t.Fatalf("expected finished_at to be set for completed run, got %#v", refreshed)
	}
}

func TestUpdateRunStatus_DoesNotBlankExistingFieldsOnPartialUpdate(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAIRunDAO(db)
	ctx := context.Background()

	run := &model.AIRun{
		ID:            "run-2",
		SessionID:     "session-2",
		UserMessageID: "msg-2",
		Status:        "running",
		TraceJSON:     "{}",
	}
	if err := dao.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	if err := dao.UpdateRunStatus(ctx, run.ID, AIRunStatusUpdate{
		IntentType:    "diagnosis",
		AssistantType: "DiagnosisAgent",
	}); err != nil {
		t.Fatalf("partial update run status: %v", err)
	}

	refreshed, err := dao.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if refreshed == nil {
		t.Fatal("expected run to exist")
	}
	if refreshed.Status != "running" {
		t.Fatalf("expected status to remain running, got %q", refreshed.Status)
	}
	if refreshed.IntentType != "diagnosis" || refreshed.AssistantType != "DiagnosisAgent" {
		t.Fatalf("expected partial fields to be updated, got %#v", refreshed)
	}
}

func TestListBySessionIDs_ReturnsRunsAcrossSessions(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAIRunDAO(db)
	ctx := context.Background()

	for _, run := range []*model.AIRun{
		{ID: "run-a", SessionID: "session-a", UserMessageID: "msg-a", Status: "completed", TraceJSON: "{}"},
		{ID: "run-b", SessionID: "session-b", UserMessageID: "msg-b", Status: "completed", TraceJSON: "{}"},
		{ID: "run-c", SessionID: "session-c", UserMessageID: "msg-c", Status: "completed", TraceJSON: "{}"},
	} {
		if err := dao.CreateRun(ctx, run); err != nil {
			t.Fatalf("create run %s: %v", run.ID, err)
		}
	}

	runs, err := dao.ListBySessionIDs(ctx, []string{"session-a", "session-c"})
	if err != nil {
		t.Fatalf("list runs by session ids: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
	if runs[0].SessionID != "session-a" || runs[1].SessionID != "session-c" {
		t.Fatalf("unexpected runs returned: %#v", runs)
	}
}
