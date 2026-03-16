package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/dao"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestRunHandler_GetRun(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := New(Dependencies{
		RunDAO:             dao.NewAIRunDAO(db),
		DiagnosisReportDAO: dao.NewAIDiagnosisReportDAO(db),
	})

	run := &model.AIRun{
		ID:                 "run-1",
		SessionID:          "session-1",
		UserMessageID:      "message-1",
		AssistantMessageID: "message-2",
		IntentType:         "diagnosis",
		AssistantType:      "diagnosis",
		Status:             "completed",
		ProgressSummary:    "Diagnosis complete",
	}
	if err := h.deps.RunDAO.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	report := &model.AIDiagnosisReport{
		ID:                  "report-1",
		RunID:               run.ID,
		SessionID:           run.SessionID,
		Summary:             "Quota exhaustion blocked scheduling",
		EvidenceJSON:        `["pod pending due to quota"]`,
		RootCausesJSON:      `["namespace quota exhausted"]`,
		RecommendationsJSON: `["increase quota"]`,
		GeneratedAt:         time.Now().UTC().Truncate(time.Second),
	}
	if err := h.deps.DiagnosisReportDAO.CreateReport(context.Background(), report); err != nil {
		t.Fatalf("seed diagnosis report: %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "runId", Value: run.ID}}
	c.Request = httptest.NewRequest(http.MethodGet, "/runs/"+run.ID, nil)

	h.GetRun(c)

	var resp struct {
		Data map[string]map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	got := resp.Data["run"]
	if got["run_id"] != run.ID {
		t.Fatalf("expected run id %q, got %#v", run.ID, got["run_id"])
	}
	if got["status"] != run.Status {
		t.Fatalf("expected status %q, got %#v", run.Status, got["status"])
	}
	reportData, ok := got["report"].(map[string]any)
	if !ok {
		t.Fatalf("expected report summary, got %#v", got["report"])
	}
	if reportData["report_id"] != report.ID {
		t.Fatalf("expected report id %q, got %#v", report.ID, reportData["report_id"])
	}
}

func NewAIRunDAOForTest(db *gorm.DB) *dao.AIRunDAO {
	return dao.NewAIRunDAO(db)
}
