package http

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/gin-gonic/gin"
)

type stubAlertWebhookIngestor struct {
	events []model.AIAlertIngestEvent
	err    error
}

func (s *stubAlertWebhookIngestor) Ingest(_ context.Context, _ string, _ []byte) ([]model.AIAlertIngestEvent, error) {
	return s.events, s.err
}

type stubAlertWebhookEnqueuer struct {
	jobID string
	err   error
}

func (s *stubAlertWebhookEnqueuer) Enqueue(_ context.Context, _ model.AIAlertIngestEvent) (string, error) {
	return s.jobID, s.err
}

func TestAlertWebhook_RejectsMissingSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origSecret := config.CFG.AI.AlertWebhookSecret
	config.CFG.AI.AlertWebhookSecret = "test-ai-webhook-secret"
	defer func() {
		config.CFG.AI.AlertWebhookSecret = origSecret
	}()

	handler := NewAlertWebhookHandler(
		&stubAlertWebhookIngestor{},
		&stubAlertWebhookEnqueuer{},
	)

	r := gin.New()
	r.POST("/api/v1/ai/alerts/webhook", handler.Handle)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/alerts/webhook", bytes.NewReader([]byte(`{"alerts":[]}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v, body=%s", err, rec.Body.String())
	}
	if resp.Code != int(xcode.Unauthorized) {
		t.Fatalf("expected response code %d, got %d", xcode.Unauthorized, resp.Code)
	}
}

func TestAlertWebhook_AcceptsValidSignatureAndReturns202(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origSecret := config.CFG.AI.AlertWebhookSecret
	config.CFG.AI.AlertWebhookSecret = "test-ai-webhook-secret"
	defer func() {
		config.CFG.AI.AlertWebhookSecret = origSecret
	}()

	payload := []byte(`{"kind":"opspilot.alert.v1","alerts":[{"fingerprint":"fp-1","status":"firing"}]}`)
	signature := signAlertWebhookPayload(config.CFG.AI.AlertWebhookSecret, payload)

	handler := NewAlertWebhookHandler(
		&stubAlertWebhookIngestor{events: []model.AIAlertIngestEvent{{ID: "evt-1"}}},
		&stubAlertWebhookEnqueuer{jobID: "job-1"},
	)

	r := gin.New()
	r.POST("/api/v1/ai/alerts/webhook", handler.Handle)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/alerts/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OpsPilot-Signature", "sha256="+signature)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusAccepted, rec.Code, rec.Body.String())
	}

	var resp struct {
		Accepted bool   `json:"accepted"`
		EventID  string `json:"event_id"`
		JobID    string `json:"job_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v, body=%s", err, rec.Body.String())
	}
	if !resp.Accepted || resp.EventID != "evt-1" || resp.JobID != "job-1" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func signAlertWebhookPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
