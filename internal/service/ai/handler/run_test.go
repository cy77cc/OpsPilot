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

func TestRunHandler_GetRun(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)

	// 创建会话
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

	// 创建 Run
	runDAO := aidao.NewAIRunDAO(db)
	run := &model.AIRun{
		ID:                 "run-1",
		SessionID:          session.ID,
		UserMessageID:      "message-1",
		AssistantMessageID: "message-2",
		IntentType:         "diagnosis",
		AssistantType:      "diagnosis",
		Status:             "completed",
		ProgressSummary:    "Diagnosis complete",
	}
	if err := runDAO.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("seed run: %v", err)
	}

	// 创建诊断报告
	diagnosisReportDAO := aidao.NewAIDiagnosisReportDAO(db)
	report := &model.AIDiagnosisReport{
		ID:                  "report-1",
		RunID:               run.ID,
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

	h := NewAIHandlerWithDB(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(1))
	c.Params = gin.Params{{Key: "runId", Value: run.ID}}
	c.Request = httptest.NewRequest(http.MethodGet, "/runs/"+run.ID, nil)

	h.GetRun(c)

	var resp struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode run response: %v", err)
	}

	if resp.Code != 1000 {
		t.Fatalf("expected code 1000, got %d", resp.Code)
	}

	got := resp.Data
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
	if reportData["id"] != report.ID {
		t.Fatalf("expected report id %q, got %#v", report.ID, reportData["id"])
	}
}

func TestGetRun_ReturnsNotFoundForNonexistentRun(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(1))
	c.Params = gin.Params{{Key: "runId", Value: "nonexistent-run-id"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/runs/nonexistent-run-id", nil)

	h.GetRun(c)

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
