package logic

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestConsistency_DetectsDriftAndSchedulesRemediation(t *testing.T) {
	db := openGitOpsConsistencyDB(t)
	seedGitOpsRelease(t, db, 42, "payments", "prod", "rev-actual", "succeeded")

	result, err := EvaluateDrift(context.Background(), db, 42, "payments", "rev-desired")
	if err != nil {
		t.Fatalf("evaluate drift: %v", err)
	}
	if !result.Drifted {
		t.Fatalf("expected drift to be detected")
	}
	if !result.RemediationScheduled {
		t.Fatalf("expected remediation to be scheduled")
	}
	if result.ActualRevision != "rev-actual" || result.DesiredRevision != "rev-desired" {
		t.Fatalf("unexpected revision comparison: %#v", result)
	}
}

func TestConsistency_CircuitBreakerTripsOnConsecutiveFailures(t *testing.T) {
	db := openGitOpsConsistencyDB(t)
	seedGitOpsRelease(t, db, 42, "payments", "prod", "rev-a", "failed")
	seedGitOpsRelease(t, db, 42, "payments", "prod", "rev-b", "failed")
	seedGitOpsRelease(t, db, 42, "payments", "prod", "rev-c", "failed")

	trip, consecutive, err := ShouldTripGitOpsCircuitBreaker(context.Background(), db, 42, "payments", 3)
	if err != nil {
		t.Fatalf("evaluate circuit breaker: %v", err)
	}
	if !trip {
		t.Fatalf("expected circuit breaker trip")
	}
	if consecutive != 3 {
		t.Fatalf("expected 3 consecutive failures, got %d", consecutive)
	}
}

func openGitOpsConsistencyDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:cluster-phase3-consistency-" + time.Now().UTC().Format("20060102150405.000000000") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS gitops_app_releases (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cluster_id INTEGER NOT NULL,
		app_name VARCHAR(191) NOT NULL,
		environment VARCHAR(32) NOT NULL,
		git_revision VARCHAR(128) NOT NULL,
		sync_result VARCHAR(32) NOT NULL,
		rollback_ref VARCHAR(128) NOT NULL DEFAULT '',
		audit_id INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`).Error; err != nil {
		t.Fatalf("create gitops_app_releases: %v", err)
	}
	return db
}

func seedGitOpsRelease(t *testing.T, db *gorm.DB, clusterID uint, app, env, revision, result string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO gitops_app_releases (cluster_id, app_name, environment, git_revision, sync_result, rollback_ref, audit_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, '', 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, clusterID, app, env, revision, result).Error; err != nil {
		t.Fatalf("seed gitops release: %v", err)
	}
}
