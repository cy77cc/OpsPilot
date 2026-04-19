package handler

import (
	"bytes"
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	monitoringmodel "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestReceiveWebhook_FanoutToAIIngestQueue(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:receiver-fanout?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&monitoringmodel.AlertEvent{},
		&monitoringmodel.AlertNotificationChannel{},
		&monitoringmodel.AlertRuleChannelBinding{},
		&monitoringmodel.AlertSeverityRoute{},
		&aimodel.AIAlertIngestEvent{},
		&aimodel.AIAlertHealJob{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	origSecret := config.CFG.Prometheus.WebhookSecret
	config.CFG.Prometheus.WebhookSecret = "test-prom-secret"
	t.Cleanup(func() {
		config.CFG.Prometheus.WebhookSecret = origSecret
	})

	h := NewHandler(&svc.ServiceContext{DB: db})
	h.aiFanout = &stubAIFanout{db: db}
	r := gin.New()
	r.POST("/alerts/receiver", h.ReceiveWebhook)

	payload := []byte(`{"alerts":[{"status":"firing","fingerprint":"fp-123","labels":{"alertname":"CPU"}}]}`)
	sig := signWebhookBody("test-prom-secret", payload)
	req := httptest.NewRequest(http.MethodPost, "/alerts/receiver", bytes.NewReader(payload))
	req.Header.Set("X-OpsPilot-Signature", hex.EncodeToString(sig))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var count int64
	if err := db.Table("ai_alert_heal_jobs").Count(&count).Error; err != nil {
		t.Fatalf("count ai jobs: %v", err)
	}
	if count == 0 {
		t.Fatal("expected ai alert-heal job to be enqueued")
	}
}

type stubAIFanout struct {
	db *gorm.DB
}

func (s *stubAIFanout) HandleAlertmanager(ctx context.Context, payload AlertmanagerWebhook) error {
	_ = ctx
	if len(payload.Alerts) == 0 {
		return nil
	}
	event := aimodel.AIAlertIngestEvent{
		ID:          "evt-fanout",
		Source:      "alertmanager",
		Protocol:    "alertmanager",
		Fingerprint: payload.Alerts[0].Fingerprint,
		Status:      payload.Alerts[0].Status,
		DedupeKey:   "alertmanager:" + payload.Alerts[0].Fingerprint + ":" + payload.Alerts[0].Status,
		Title:       "fanout",
		ReceivedAt:  time.Now().UTC(),
	}
	if err := s.db.Create(&event).Error; err != nil {
		return err
	}
	return s.db.Create(&aimodel.AIAlertHealJob{
		ID:      "job-fanout",
		EventID: event.ID,
		Scene:   "alert_self_heal",
		Status:  "pending",
	}).Error
}
