package http

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	alertheal "github.com/cy77cc/OpsPilot/internal/modules/ai/logic/alertheal"
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

func (s *stubAlertWebhookEnqueuer) EnqueueBatch(_ context.Context, _ []model.AIAlertIngestEvent) (string, error) {
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

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAlertWebhook_RejectsInvalidSignature(t *testing.T) {
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
	req.Header.Set("X-OpsPilot-Signature", "deadbeef")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAlertWebhook_RejectsOversizedPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origSecret := config.CFG.AI.AlertWebhookSecret
	config.CFG.AI.AlertWebhookSecret = "test-ai-webhook-secret"
	defer func() {
		config.CFG.AI.AlertWebhookSecret = origSecret
	}()

	payload := oversizedAlertWebhookPayload()
	signature := signAlertWebhookPayload(config.CFG.AI.AlertWebhookSecret, payload)

	handler := NewAlertWebhookHandler(
		&stubAlertWebhookIngestor{},
		&stubAlertWebhookEnqueuer{},
	)

	r := gin.New()
	r.POST("/api/v1/ai/alerts/webhook", handler.Handle)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/alerts/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OpsPilot-Signature", signature)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
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

func TestAlertWebhook_IngestInvalidPayloadReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	payload := []byte(`{"alerts":[]}`)
	rec := invokeAlertWebhook(
		t,
		"test-ai-webhook-secret",
		payload,
		signAlertWebhookPayload("test-ai-webhook-secret", payload),
		&stubAlertWebhookIngestor{err: alertheal.ErrInvalidPayload},
		&stubAlertWebhookEnqueuer{},
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestAlertWebhook_IngestInternalErrorReturns500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	payload := []byte(`{"alerts":[]}`)
	rec := invokeAlertWebhook(
		t,
		"test-ai-webhook-secret",
		payload,
		signAlertWebhookPayload("test-ai-webhook-secret", payload),
		&stubAlertWebhookIngestor{err: errors.New("db boom")},
		&stubAlertWebhookEnqueuer{},
	)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}
}

func TestAlertWebhook_EnqueueFailureReturns500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	payload := []byte(`{"kind":"opspilot.alert.v1","alerts":[{"fingerprint":"fp-1","status":"firing"}]}`)
	rec := invokeAlertWebhook(
		t,
		"test-ai-webhook-secret",
		payload,
		signAlertWebhookPayload("test-ai-webhook-secret", payload),
		&stubAlertWebhookIngestor{events: []model.AIAlertIngestEvent{{ID: "evt-1"}}},
		&stubAlertWebhookEnqueuer{err: errors.New("enqueue boom")},
	)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}
}

func TestAlertWebhook_RejectsMissingSecretConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	payload := []byte(`{"kind":"opspilot.alert.v1","alerts":[{"fingerprint":"fp-1","status":"firing"}]}`)
	rec := invokeAlertWebhook(
		t,
		"",
		payload,
		signAlertWebhookPayload("test-ai-webhook-secret", payload),
		&stubAlertWebhookIngestor{events: []model.AIAlertIngestEvent{{ID: "evt-1"}}},
		&stubAlertWebhookEnqueuer{jobID: "job-1"},
	)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}
}

func TestAlertWebhook_RejectsPlaceholderSecretConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	payload := []byte(`{"kind":"opspilot.alert.v1","alerts":[{"fingerprint":"fp-1","status":"firing"}]}`)
	rec := invokeAlertWebhook(
		t,
		"${AI_ALERT_WEBHOOK_SECRET}",
		payload,
		signAlertWebhookPayload("test-ai-webhook-secret", payload),
		&stubAlertWebhookIngestor{events: []model.AIAlertIngestEvent{{ID: "evt-1"}}},
		&stubAlertWebhookEnqueuer{jobID: "job-1"},
	)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}
}

func invokeAlertWebhook(
	t *testing.T,
	secret string,
	payload []byte,
	signature string,
	ingestor AlertWebhookIngestor,
	enqueuer AlertWebhookEnqueuer,
) *httptest.ResponseRecorder {
	t.Helper()

	origSecret := config.CFG.AI.AlertWebhookSecret
	config.CFG.AI.AlertWebhookSecret = secret
	defer func() {
		config.CFG.AI.AlertWebhookSecret = origSecret
	}()

	handler := NewAlertWebhookHandler(ingestor, enqueuer)
	r := gin.New()
	r.POST("/api/v1/ai/alerts/webhook", handler.Handle)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/alerts/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if signature != "" {
		req.Header.Set("X-OpsPilot-Signature", signature)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func signAlertWebhookPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func oversizedAlertWebhookPayload() []byte {
	pad := bytes.Repeat([]byte("a"), maxAlertWebhookPayloadBytes+32)
	body := make([]byte, 0, len(`{"alerts":[],"pad":""}`)+len(pad))
	body = append(body, []byte(`{"alerts":[],"pad":"`)...)
	body = append(body, pad...)
	body = append(body, []byte(`"}`)...)
	return body
}
