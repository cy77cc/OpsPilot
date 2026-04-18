package alertheal

import (
	"context"
	"testing"

	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao"
	aidaoalertheal "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/alertheal"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
)

func TestNormalizePayload_SupportsAlertmanagerAndUnified(t *testing.T) {
	alertmanagerRaw := []byte(`{"alerts":[{"status":"firing","fingerprint":"fp-a","labels":{"alertname":"CPU"}}]}`)
	unifiedRaw := []byte(`{"kind":"opspilot.alert.v1","alerts":[{"status":"firing","fingerprint":"fp-b","title":"Disk Full"}]}`)

	amAlerts, err := NormalizePayload("alertmanager", alertmanagerRaw)
	if err != nil {
		t.Fatalf("normalize alertmanager payload: %v", err)
	}
	if len(amAlerts) != 1 {
		t.Fatalf("expected 1 alertmanager alert, got %d", len(amAlerts))
	}
	if amAlerts[0].Fingerprint != "fp-a" {
		t.Fatalf("expected fingerprint fp-a, got %q", amAlerts[0].Fingerprint)
	}

	unifiedAlerts, err := NormalizePayload("opspilot.alert.v1", unifiedRaw)
	if err != nil {
		t.Fatalf("normalize unified payload: %v", err)
	}
	if len(unifiedAlerts) != 1 {
		t.Fatalf("expected 1 unified alert, got %d", len(unifiedAlerts))
	}
	if unifiedAlerts[0].Fingerprint != "fp-b" {
		t.Fatalf("expected fingerprint fp-b, got %q", unifiedAlerts[0].Fingerprint)
	}
}

func TestIngestService_DeduplicatesBySourceFingerprintStatus(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertIngestEvent{})
	dao := aidaoalertheal.NewDAO(db)
	svc := NewService(dao)

	raw := []byte(`{"alerts":[{"status":"firing","fingerprint":"fp-a","labels":{"alertname":"CPU"}}]}`)

	first, err := svc.Ingest(context.Background(), "alertmanager", raw)
	if err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	second, err := svc.Ingest(context.Background(), "alertmanager", raw)
	if err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("expected one row each ingest, got first=%d second=%d", len(first), len(second))
	}

	var count int64
	dedupeKey := DedupeKey("alertmanager", "fp-a", "firing")
	if err := db.Model(&model.AIAlertIngestEvent{}).Where("dedupe_key = ?", dedupeKey).Count(&count).Error; err != nil {
		t.Fatalf("count dedupe rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one deduped row, got %d", count)
	}
}

func TestIngestService_EmptyAlertsReturnsInvalidPayload(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertIngestEvent{})
	dao := aidaoalertheal.NewDAO(db)
	svc := NewService(dao)

	_, err := svc.Ingest(context.Background(), "alertmanager", []byte(`{"alerts":[]}`))
	if err == nil {
		t.Fatal("expected ErrInvalidPayload, got nil")
	}
	if err != ErrInvalidPayload {
		t.Fatalf("expected ErrInvalidPayload, got %v", err)
	}
}

func TestIngestService_MixedInvalidBatchDoesNotPartiallyPersist(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertIngestEvent{})
	dao := aidaoalertheal.NewDAO(db)
	svc := NewService(dao)

	raw := []byte(`{"alerts":[{"status":"firing","fingerprint":"fp-a","labels":{"alertname":"CPU"}},{"status":"firing","fingerprint":"  ","labels":{"alertname":"MEM"}}]}`)

	_, err := svc.Ingest(context.Background(), "alertmanager", raw)
	if err == nil {
		t.Fatal("expected ErrInvalidPayload, got nil")
	}
	if err != ErrInvalidPayload {
		t.Fatalf("expected ErrInvalidPayload, got %v", err)
	}

	var count int64
	if err := db.Model(&model.AIAlertIngestEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected zero rows persisted on invalid batch, got %d", count)
	}
}
