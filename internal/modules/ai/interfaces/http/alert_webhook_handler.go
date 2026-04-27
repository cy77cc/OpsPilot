package http

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	alertheal "github.com/cy77cc/OpsPilot/internal/modules/ai/logic/alertheal"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/gin-gonic/gin"
)

const maxAlertWebhookPayloadBytes = 1 << 20 // 1 MiB
const maxAlertWebhookTimestampAge = 5 * time.Minute

type AlertWebhookIngestor interface {
	Ingest(ctx context.Context, protocol string, raw []byte) ([]model.AIAlertIngestEvent, error)
}

type AlertWebhookEnqueuer interface {
	EnqueueBatch(ctx context.Context, events []model.AIAlertIngestEvent) (string, error)
}

type AlertWebhookHandler struct {
	ingestor AlertWebhookIngestor
	enqueuer AlertWebhookEnqueuer
}

func NewAlertWebhookHandler(ingestor AlertWebhookIngestor, enqueuer AlertWebhookEnqueuer) *AlertWebhookHandler {
	return &AlertWebhookHandler{ingestor: ingestor, enqueuer: enqueuer}
}

func (h *AlertWebhookHandler) Handle(c *gin.Context) {
	secret := strings.TrimSpace(config.CFG.AI.AlertWebhookSecret)
	if !isValidAlertWebhookSecret(secret) {
		writeWebhookError(c, http.StatusInternalServerError, "webhook secret is not configured")
		return
	}

	signature := strings.TrimSpace(c.GetHeader("X-OpsPilot-Signature"))
	if signature == "" {
		writeWebhookError(c, http.StatusUnauthorized, "missing webhook signature")
		return
	}

	// Validate timestamp to prevent replay attacks
	timestampStr := strings.TrimSpace(c.GetHeader("X-OpsPilot-Timestamp"))
	if timestampStr != "" {
		ts, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			writeWebhookError(c, http.StatusUnauthorized, "invalid webhook timestamp")
			return
		}
		tsTime := time.Unix(ts, 0)
		if time.Since(tsTime) > maxAlertWebhookTimestampAge {
			writeWebhookError(c, http.StatusUnauthorized, "webhook timestamp expired")
			return
		}
	}

	body, err := readAlertWebhookPayload(c.Request.Body)
	if err != nil {
		if err == errAlertWebhookPayloadTooLarge {
			writeWebhookError(c, http.StatusBadRequest, "webhook payload too large")
			return
		}
		writeWebhookError(c, http.StatusInternalServerError, "failed to read webhook payload")
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	if !verifyAlertWebhookSignature(secret, signature, body) {
		writeWebhookError(c, http.StatusUnauthorized, "invalid webhook signature")
		return
	}

	if h == nil || h.ingestor == nil || h.enqueuer == nil {
		writeWebhookError(c, http.StatusInternalServerError, "webhook handler not initialized")
		return
	}

	events, err := h.ingestor.Ingest(c.Request.Context(), detectAlertWebhookProtocol(body), body)
	if err != nil {
		if errors.Is(err, alertheal.ErrInvalidPayload) {
			writeWebhookError(c, http.StatusBadRequest, "invalid webhook payload")
			return
		}
		writeWebhookError(c, http.StatusInternalServerError, "failed to ingest webhook payload")
		return
	}
	if len(events) == 0 {
		writeWebhookError(c, http.StatusBadRequest, "invalid webhook payload")
		return
	}

	firstJobID, enqueueErr := h.enqueuer.EnqueueBatch(c.Request.Context(), events)
	if enqueueErr != nil {
		writeWebhookError(c, http.StatusInternalServerError, "failed to enqueue alert heal jobs")
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"accepted": true,
		"event_id": events[0].ID,
		"job_id":   firstJobID,
	})
}

var errAlertWebhookPayloadTooLarge = errors.New("webhook payload too large")

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
	expected := signAlertWebhookBody(secret, body)
	if expected == nil {
		return false
	}
	actual, err := decodeAlertWebhookSignature(signature)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, actual)
}

func signAlertWebhookBody(secret string, body []byte) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write(body); err != nil {
		return nil
	}
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

func writeWebhookError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func isValidAlertWebhookSecret(secret string) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	if strings.HasPrefix(secret, "${") && strings.HasSuffix(secret, "}") {
		return false
	}
	return true
}
