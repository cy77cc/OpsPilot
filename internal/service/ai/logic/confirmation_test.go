package logic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cy77cc/k8s-manage/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.ConfirmationRequest{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestConfirmationServiceRequestAndConfirm(t *testing.T) {
	db := newTestDB(t)
	svc := NewConfirmationService(db)
	svc.pollInterval = 20 * time.Millisecond

	item, err := svc.RequestConfirmation(context.Background(), ConfirmationRequestInput{
		RequestUserID: 1,
		ToolName:      "host_batch_exec_apply",
		ToolMode:      "mutating",
		RiskLevel:     "high",
		Timeout:       2 * time.Second,
	})
	if err != nil {
		t.Fatalf("request confirmation: %v", err)
	}
	if item.Status != "pending" {
		t.Fatalf("expected pending status, got %s", item.Status)
	}

	confirmed, err := svc.Confirm(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if confirmed.Status != "confirmed" {
		t.Fatalf("expected confirmed status, got %s", confirmed.Status)
	}
	if confirmed.ConfirmedAt == nil {
		t.Fatalf("expected confirmed_at set")
	}
}

func TestConfirmationServiceWaitForConfirmation(t *testing.T) {
	db := newTestDB(t)
	svc := NewConfirmationService(db)
	svc.pollInterval = 20 * time.Millisecond

	item, err := svc.RequestConfirmation(context.Background(), ConfirmationRequestInput{
		RequestUserID: 1,
		ToolName:      "service_deploy_apply",
		Timeout:       2 * time.Second,
	})
	if err != nil {
		t.Fatalf("request confirmation: %v", err)
	}
	go func() {
		time.Sleep(60 * time.Millisecond)
		_, _ = svc.Confirm(context.Background(), item.ID)
	}()
	out, err := svc.WaitForConfirmation(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("wait for confirmation: %v", err)
	}
	if out.Status != "confirmed" {
		t.Fatalf("expected confirmed status, got %s", out.Status)
	}
}

func TestConfirmationServiceCancel(t *testing.T) {
	db := newTestDB(t)
	svc := NewConfirmationService(db)

	item, err := svc.RequestConfirmation(context.Background(), ConfirmationRequestInput{
		RequestUserID: 1,
		ToolName:      "host_batch_exec_apply",
		Timeout:       2 * time.Second,
	})
	if err != nil {
		t.Fatalf("request confirmation: %v", err)
	}
	out, err := svc.Cancel(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("cancel confirmation: %v", err)
	}
	if out.Status != "cancelled" {
		t.Fatalf("expected cancelled status, got %s", out.Status)
	}
}

func TestConfirmationServiceWaitCancelledByContext(t *testing.T) {
	db := newTestDB(t)
	svc := NewConfirmationService(db)

	item, err := svc.RequestConfirmation(context.Background(), ConfirmationRequestInput{
		RequestUserID: 1,
		ToolName:      "host_batch_exec_apply",
		Timeout:       2 * time.Second,
	})
	if err != nil {
		t.Fatalf("request confirmation: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	_, err = svc.WaitForConfirmation(ctx, item.ID)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}
