package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestDiagnosisHandler_GetDiagnosisReport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := New(Dependencies{
		DiagnosisReportDAO: NewAIDiagnosisReportDAOForTest(db),
	})

	report := &model.AIDiagnosisReport{
		ID:                  "report-1",
		RunID:               "run-1",
		SessionID:           "session-1",
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
	c.Params = gin.Params{{Key: "reportId", Value: report.ID}}
	c.Request = httptest.NewRequest(http.MethodGet, "/diagnosis/"+report.ID, nil)

	h.GetDiagnosisReport(c)

	var resp struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode diagnosis response: %v", err)
	}
	got := resp.Data
	if got["report_id"] != report.ID {
		t.Fatalf("expected report id %q, got %#v", report.ID, got["report_id"])
	}
	if got["run_id"] != report.RunID {
		t.Fatalf("expected run id %q, got %#v", report.RunID, got["run_id"])
	}
}

func NewAIDiagnosisReportDAOForTest(db *gorm.DB) *aidao.AIDiagnosisReportDAO {
	return aidao.NewAIDiagnosisReportDAO(db)
}
