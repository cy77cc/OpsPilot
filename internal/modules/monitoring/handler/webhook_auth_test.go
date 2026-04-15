package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	"github.com/gin-gonic/gin"
)

func TestReceiveWebhook_RejectsMissingSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	payload := []byte(`{"alerts":[]}`)
	resp := invokeReceiveWebhook(t, payload, "")
	if resp.Code != int(xcode.Unauthorized) {
		t.Fatalf("expected code %d, got %d, msg=%q", xcode.Unauthorized, resp.Code, resp.Msg)
	}
}

func TestReceiveWebhook_RejectsInvalidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	payload := []byte(`{"alerts":[]}`)
	resp := invokeReceiveWebhook(t, payload, "deadbeef")
	if resp.Code != int(xcode.Unauthorized) {
		t.Fatalf("expected code %d, got %d, msg=%q", xcode.Unauthorized, resp.Code, resp.Msg)
	}
}

func TestReceiveWebhook_AcceptsValidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	payload := []byte(`{"alerts":[]}`)
	sig := signWebhookPayload("test-webhook-secret", payload)
	resp := invokeReceiveWebhook(t, payload, sig)
	if resp.Code != int(xcode.Success) {
		t.Fatalf("expected code %d, got %d, msg=%q", xcode.Success, resp.Code, resp.Msg)
	}
}

func TestReceiveWebhook_BindFailsAfterValidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	payload := []byte(`{"alerts":[}`)
	sig := signWebhookPayload("test-webhook-secret", payload)
	resp := invokeReceiveWebhook(t, payload, sig)
	if resp.Code != int(xcode.ParamError) {
		t.Fatalf("expected code %d, got %d, msg=%q", xcode.ParamError, resp.Code, resp.Msg)
	}
}

func TestReceiveWebhook_RejectsOversizedPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	payload := oversizedWebhookPayload()
	sig := signWebhookPayload("test-webhook-secret", payload)
	resp := invokeReceiveWebhook(t, payload, sig)
	if resp.Code != int(xcode.ParamError) {
		t.Fatalf("expected code %d, got %d, msg=%q", xcode.ParamError, resp.Code, resp.Msg)
	}
}

type webhookResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func invokeReceiveWebhook(t *testing.T, payload []byte, signature string) webhookResponse {
	t.Helper()

	origSecret := config.CFG.Prometheus.WebhookSecret
	config.CFG.Prometheus.WebhookSecret = "test-webhook-secret"
	defer func() {
		config.CFG.Prometheus.WebhookSecret = origSecret
	}()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/alerts/receiver", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if signature != "" {
		req.Header.Set("X-OpsPilot-Signature", signature)
	}
	ctx.Request = req

	h := &Handler{webhookGW: &NotificationGateway{}}
	h.ReceiveWebhook(ctx)

	var resp webhookResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v, body=%s", err, recorder.Body.String())
	}
	return resp
}

func signWebhookPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func oversizedWebhookPayload() []byte {
	pad := bytes.Repeat([]byte("a"), maxWebhookPayloadBytes+32)
	body := make([]byte, 0, len(`{"alerts":[],"pad":""}`)+len(pad))
	body = append(body, []byte(`{"alerts":[],"pad":"`)...)
	body = append(body, pad...)
	body = append(body, []byte(`"}`)...)
	return body
}
