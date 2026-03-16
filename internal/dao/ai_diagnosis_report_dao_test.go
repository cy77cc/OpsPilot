package dao

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/testutil"
	"github.com/google/uuid"
)

func TestAIDiagnosisReportDAO_CreateAndLookup(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)

	ctx := context.Background()
	reportDAO := NewAIDiagnosisReportDAO(suite.DB)

	run := &model.AIRun{
		ID:            uuid.NewString(),
		SessionID:     uuid.NewString(),
		UserMessageID: uuid.NewString(),
		IntentType:    "assist",
		AssistantType: "qa",
		RiskLevel:     "medium",
		Status:        "created",
		TraceID:       uuid.NewString(),
	}
	testutil.RequireNoError(t, suite.DB.Create(run).Error)

	report := &model.AIDiagnosisReport{
		ID:                  uuid.NewString(),
		RunID:               run.ID,
		SessionID:           run.SessionID,
		Summary:             "summary",
		ImpactScope:         "impact",
		SuspectedRootCauses: "cause",
		Evidence:            "evidence",
		Recommendations:     "actions",
		RawToolRefs:         "refs",
		Status:              "final",
	}
	testutil.RequireNoError(t, reportDAO.CreateReport(ctx, report))

	created, err := reportDAO.GetReport(ctx, report.ID)
	testutil.RequireNoError(t, err)
	testutil.AssertEqual(t, report.ID, created.ID)
	testutil.AssertEqual(t, report.Status, created.Status)

	byRun, err := reportDAO.GetReportByRunID(ctx, run.ID)
	testutil.RequireNoError(t, err)
	testutil.AssertEqual(t, report.ID, byRun.ID)
	testutil.AssertEqual(t, created.RunID, byRun.RunID)
}
