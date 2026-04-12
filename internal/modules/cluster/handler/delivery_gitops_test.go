package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	clusterintegration "github.com/cy77cc/OpsPilot/internal/modules/cluster/integration"
	clustermodel "github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
	governance "github.com/cy77cc/OpsPilot/internal/modules/governance"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHandlerPhase3GitOps_SyncCallsArgoAndRecordsRelease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, db := newPhase3GitOpsTestHandler(t)
	handler.argocd = stubArgoCDClient{result: clusterintegration.ArgoSyncResult{Status: "succeeded", Revision: "rev-sync-1"}}

	body := strings.NewReader(`{"environment":"prod","approval_token":"` + mustIssuePhase3GitOpsApproval(t, db, "payments") + `"}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/clusters/42/apps/payments/sync", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "name", Value: "payments"}}
	ctx.Set("uid", uint64(1001))

	handler.SyncGitOpsApp(ctx)

	env := decodeOperationEnvelope(t, recorder)
	if env.State != OperationStateCompleted {
		t.Fatalf("expected completed state, got %q", env.State)
	}
	if env.Code != OperationCodeSuccess {
		t.Fatalf("expected success code, got %q", env.Code)
	}

	var count int64
	if err := db.Table("gitops_app_releases").Where("cluster_id = ? AND app_name = ? AND sync_result = ?", 42, "payments", "succeeded").Count(&count).Error; err != nil {
		t.Fatalf("count gitops release records: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected at least one succeeded release record")
	}
}

func TestHandlerPhase3GitOps_TripsCircuitBreakerOnConsecutiveFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, db := newPhase3GitOpsTestHandler(t)
	handler.argocd = stubArgoCDClient{err: fmt.Errorf("argocd unreachable")}

	seedGitOpsRelease(t, db, 42, "payments", "prod", "rev-a", "failed")
	seedGitOpsRelease(t, db, 42, "payments", "prod", "rev-b", "failed")
	seedGitOpsRelease(t, db, 42, "payments", "prod", "rev-c", "failed")

	body := strings.NewReader(`{"environment":"prod","approval_token":"` + mustIssuePhase3GitOpsApproval(t, db, "payments") + `"}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/clusters/42/apps/payments/sync", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "name", Value: "payments"}}
	ctx.Set("uid", uint64(1001))

	handler.SyncGitOpsApp(ctx)

	env := decodeOperationEnvelope(t, recorder)
	if env.State != OperationStateFailed {
		t.Fatalf("expected failed state for open circuit breaker, got %q", env.State)
	}
	if env.Code != "circuit_open" {
		t.Fatalf("expected circuit_open code, got %q", env.Code)
	}
}

func newPhase3GitOpsTestHandler(t *testing.T) (*Handler, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:cluster-phase3-gitops-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&clustermodel.Cluster{},
		&clustermodel.OperationApproval{},
		&clustermodel.OperationAudit{},
	); err != nil {
		t.Fatalf("migrate base tables: %v", err)
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
	if err := db.Create(&clustermodel.Cluster{
		ID:      42,
		Name:    "phase3-gitops-test-cluster",
		Status:  "active",
		Type:    "kubernetes",
		Source:  ClusterModePlatformManaged,
		EnvType: "production",
	}).Error; err != nil {
		t.Fatalf("seed cluster: %v", err)
	}

	handler := &Handler{
		svcCtx: &svc.ServiceContext{DB: db},
		repo:   NewRepository(db),
	}
	return handler, db
}

func mustIssuePhase3GitOpsApproval(t *testing.T, db *gorm.DB, appName string) string {
	t.Helper()
	h := &Handler{svcCtx: &svc.ServiceContext{DB: db}}
	decision, err := h.phase3Preflight(context.Background(), governance.OperationIntent{
		OperatorID: 1001,
		OccurredAt: time.Now().UTC(),
		Scope: governance.Scope{
			Domain:     "cluster",
			ClusterID:  42,
			Resource:   "gitops",
			ResourceID: appName,
			Action:     "gitops.sync",
			Context: map[string]any{
				"phase3_domain": "gitops",
				"intent":        "sync",
			},
		},
	})
	if err != nil {
		t.Fatalf("issue gitops approval: %v", err)
	}
	if decision.Approval == nil || decision.Approval.Ticket == "" {
		t.Fatalf("expected approval ticket")
	}
	if err := db.Model(&clustermodel.OperationApproval{}).Where("ticket = ?", decision.Approval.Ticket).Updates(map[string]any{"status": "approved", "review_by": 2001}).Error; err != nil {
		t.Fatalf("approve gitops ticket: %v", err)
	}
	return decision.Approval.Ticket
}

type stubArgoCDClient struct {
	result clusterintegration.ArgoSyncResult
	err    error
}

func (s stubArgoCDClient) Sync(_ context.Context, _ string) (clusterintegration.ArgoSyncResult, error) {
	if s.err != nil {
		return clusterintegration.ArgoSyncResult{}, s.err
	}
	return s.result, nil
}

func seedGitOpsRelease(t *testing.T, db *gorm.DB, clusterID uint, app, env, revision, result string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO gitops_app_releases (cluster_id, app_name, environment, git_revision, sync_result, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		clusterID, app, env, revision, result, time.Now().UTC(), time.Now().UTC()).Error; err != nil {
		t.Fatalf("seed gitops release: %v", err)
	}
}
