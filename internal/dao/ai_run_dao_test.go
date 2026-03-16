package dao

import (
	"context"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/testutil"
	"github.com/google/uuid"
)

func TestAIRunDAO_StatusUpdates(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)

	ctx := context.Background()
	dao := NewAIRunDAO(suite.DB)

	run := &model.AIRun{
		ID:            uuid.NewString(),
		SessionID:     uuid.NewString(),
		UserMessageID: uuid.NewString(),
		IntentType:    "diagnosis",
		AssistantType: "auto",
		RiskLevel:     "low",
		Status:        "created",
		TraceID:       uuid.NewString(),
	}
	testutil.RequireNoError(t, dao.CreateRun(ctx, run))

	initial, err := dao.GetRun(ctx, run.ID)
	testutil.RequireNoError(t, err)
	testutil.AssertEqual(t, run.ID, initial.ID)
	testutil.AssertEqual(t, "created", initial.Status)

	startedAt := time.Now().UTC()
	testutil.RequireNoError(t, dao.UpdateRunStatus(ctx, run.ID, RunStatusUpdate{
		Status:    "running",
		StartedAt: &startedAt,
	}))

	finalErr := "something failed"
	finishedAt := startedAt.Add(90 * time.Second)
	testutil.RequireNoError(t, dao.UpdateRunStatus(ctx, run.ID, RunStatusUpdate{
		Status:       "failed",
		FinishedAt:   &finishedAt,
		ErrorMessage: &finalErr,
	}))

	stored, err := dao.GetRun(ctx, run.ID)
	testutil.RequireNoError(t, err)
	testutil.AssertEqual(t, "failed", stored.Status)
	testutil.AssertEqual(t, finalErr, stored.ErrorMessage)
	testutil.AssertNotNil(t, stored.StartedAt)
	testutil.AssertNotNil(t, stored.FinishedAt)
	testutil.AssertTrue(t, stored.StartedAt.Equal(startedAt), "started timestamp should round-trip through the DAO")
	testutil.AssertTrue(t, stored.FinishedAt.Equal(finishedAt), "finished timestamp should round-trip through the DAO")
}
