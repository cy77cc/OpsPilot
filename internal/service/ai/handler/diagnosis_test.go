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
)

func TestDiagnosisHandler_GetDiagnosisReport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)

	// 先创建会话和用户
	session := &model.AIChatSession{
		ID:     "session-1",
		UserID: 1,
		Scene:  "ai",
		Title:  "Test Session",
	}
	chatDAO := aidao.NewAIChatDAO(db)
	if err := chatDAO.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	// 创建诊断报告
	diagnosisReportDAO := aidao.NewAIDiagnosisReportDAO(db)
	report := &model.AIDiagnosisReport{
		ID:                  "report-1",
		RunID:               "run-1",
		SessionID:           session.ID,
		Summary:             "Quota exhaustion blocked scheduling",
		EvidenceJSON:        `["pod pending due to quota"]`,
		RootCausesJSON:      `["namespace quota exhausted"]`,
		RecommendationsJSON: `["increase quota"]`,
		GeneratedAt:         time.Now().UTC().Truncate(time.Second),
	}
	if err := diagnosisReportDAO.CreateReport(context.Background(), report); err != nil {
		t.Fatalf("seed diagnosis report: %v", err)
	}

	h := newAIHandlerTestHarness(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(1))
	c.Params = gin.Params{{Key: "reportId", Value: report.ID}}
	c.Request = httptest.NewRequest(http.MethodGet, "/diagnosis/"+report.ID, nil)

	h.GetDiagnosisReport(c)

	var resp struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode diagnosis response: %v", err)
	}

	if resp.Code != 1000 {
		t.Fatalf("expected code 1000, got %d", resp.Code)
	}

	got := resp.Data
	if got["report_id"] != report.ID {
		t.Fatalf("expected report id %q, got %#v", report.ID, got["report_id"])
	}
	if got["run_id"] != report.RunID {
		t.Fatalf("expected run id %q, got %#v", report.RunID, got["run_id"])
	}
}

func TestGetDiagnosisReport_ReturnsNotFoundForNonexistentReport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := newAIHandlerTestHarness(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(1))
	c.Params = gin.Params{{Key: "reportId", Value: "nonexistent-report-id"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/diagnosis/nonexistent-report-id", nil)

	h.GetDiagnosisReport(c)

	// httpx.NotFound returns HTTP 200 with business error code 2005
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200 (httpx convention), got %d", recorder.Code)
	}

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// 2005 is the NotFound business code
	if resp.Code != 2005 {
		t.Fatalf("expected business code 2005 (NotFound), got %d", resp.Code)
	}
}
