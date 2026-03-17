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
