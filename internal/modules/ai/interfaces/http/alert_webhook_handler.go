package http

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/gin-gonic/gin"
)

const maxAlertWebhookPayloadBytes = 1 << 20 // 1 MiB

type AlertWebhookIngestor interface {
	Ingest(ctx context.Context, protocol string, raw []byte) ([]model.AIAlertIngestEvent, error)
}

type AlertWebhookEnqueuer interface {
	Enqueue(ctx context.Context, event model.AIAlertIngestEvent) (string, error)
}

type AlertWebhookHandler struct {
	ingestor AlertWebhookIngestor
	enqueuer AlertWebhookEnqueuer
}

func NewAlertWebhookHandler(ingestor AlertWebhookIngestor, enqueuer AlertWebhookEnqueuer) *AlertWebhookHandler {
	return &AlertWebhookHandler{ingestor: ingestor, enqueuer: enqueuer}
}

func (h *AlertWebhookHandler) Handle(c *gin.Context) {
	signature := strings.TrimSpace(c.GetHeader("X-OpsPilot-Signature"))
	if signature == "" {
		httpx.Fail(c, xcode.Unauthorized, "missing webhook signature")
		return
	}

	body, err := readAlertWebhookPayload(c.Request.Body)
	if err != nil {
		if err == errAlertWebhookPayloadTooLarge {
			httpx.Fail(c, xcode.ParamError, "webhook payload too large")
			return
		}
		httpx.ServerErr(c, err)
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	if !verifyAlertWebhookSignature(config.CFG.AI.AlertWebhookSecret, signature, body) {
		httpx.Fail(c, xcode.Unauthorized, "invalid webhook signature")
		return
	}

	if h == nil || h.ingestor == nil || h.enqueuer == nil {
		httpx.ServerErr(c, xcode.NewErrCodeMsg(xcode.ServerError, "alert webhook handler not initialized"))
		return
	}

	events, err := h.ingestor.Ingest(c.Request.Context(), detectAlertWebhookProtocol(body), body)
	if err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	if len(events) == 0 {
		httpx.BadRequest(c, "invalid webhook payload")
		return
	}

	var firstEventID string
	var firstJobID string
	for i, event := range events {
		jobID, enqueueErr := h.enqueuer.Enqueue(c.Request.Context(), event)
		if enqueueErr != nil {
			httpx.ServerErr(c, enqueueErr)
			return
		}
		if i == 0 {
			firstEventID = event.ID
			firstJobID = jobID
		}
	}

	c.JSON(http.StatusAccepted, gin.H{
		"accepted": true,
		"event_id": firstEventID,
		"job_id":   firstJobID,
	})
}

var errAlertWebhookPayloadTooLarge = xcode.NewErrCodeMsg(xcode.ParamError, "webhook payload too large")

func readAlertWebhookPayload(body io.Reader) ([]byte, error) {
	limited := &io.LimitedReader{R: body, N: maxAlertWebhookPayloadBytes + 1}
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxAlertWebhookPayloadBytes {
		return nil, errAlertWebhookPayloadTooLarge
	}
	return raw, nil
}

func verifyAlertWebhookSignature(secret, signature string, body []byte) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}

	expected := signAlertWebhookBody(secret, body)
	actual, err := decodeAlertWebhookSignature(signature)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, actual)
}

func signAlertWebhookBody(secret string, body []byte) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return mac.Sum(nil)
}

func decodeAlertWebhookSignature(raw string) ([]byte, error) {
	signature := strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(signature), "sha256=") {
		signature = strings.TrimSpace(signature[len("sha256="):])
	}
	return hex.DecodeString(signature)
}

func detectAlertWebhookProtocol(raw []byte) string {
	var envelope struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "alertmanager"
	}
	if strings.EqualFold(strings.TrimSpace(envelope.Kind), "opspilot.alert.v1") {
		return "opspilot.alert.v1"
	}
	return "alertmanager"
}
