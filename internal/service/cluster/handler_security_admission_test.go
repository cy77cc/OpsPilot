package cluster

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	clusterintegration "github.com/cy77cc/OpsPilot/internal/service/cluster/integration"
	"github.com/cy77cc/OpsPilot/internal/service/governance"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPhase3Governance_PreflightApprovalRequired(t *testing.T) {
	handler, _ := newPhase3GovernanceTestHandler(t)
	decision, err := handler.phase3Preflight(context.Background(), governance.OperationIntent{
		OperatorID: 1001,
		Scope: governance.Scope{
			Domain:    "cluster",
			ClusterID: 42,
			Resource:  "admission",
			Action:    "admission.apply",
		},
	})
	if err != nil {
		t.Fatalf("preflight failed: %v", err)
	}
	if decision.State != governance.StateApprovalRequired {
		t.Fatalf("expected approval_required, got %s", decision.State)
	}
	if decision.Approval == nil || decision.Approval.Ticket == "" {
		t.Fatalf("expected approval ticket in decision")
	}
}

func TestPhase3Governance_FinalizeWritesAuditEnvelope(t *testing.T) {
	handler, db := newPhase3GovernanceTestHandler(t)
	now := time.Now().UTC()
	out, err := handler.phase3Finalize(context.Background(), governance.FinalizeInput{
		Intent: governance.OperationIntent{
			OperatorID: 1001,
			Scope: governance.Scope{
				Domain:    "cluster",
				ClusterID: 42,
				Resource:  "admission",
				Action:    "admission.apply",
			},
			OccurredAt: now,
		},
		Decision: governance.Decision{
			Allowed: true,
			State:   governance.StateCompleted,
			Code:    governance.CodeSuccess,
		},
		ExecutionCode: governance.CodeSuccess,
		ExecutionMsg:  "ok",
		Result: map[string]any{
			"policy": "deny-privileged",
		},
		StartedAt:  now.Add(-2 * time.Second),
		FinishedAt: now,
	})
	if err != nil {
		t.Fatalf("finalize failed: %v", err)
	}
	if out.AuditID == 0 {
		t.Fatalf("expected non-zero audit id")
	}

	var audit model.OperationAudit
	if err := db.First(&audit, out.AuditID).Error; err != nil {
		t.Fatalf("load audit record: %v", err)
	}
	if audit.Action != "admission.apply" {
		t.Fatalf("expected action admission.apply, got %s", audit.Action)
	}
}

func newPhase3GovernanceTestHandler(t *testing.T) (*Handler, *gorm.DB) {
	t.Helper()

	dsn := "file:cluster-phase3-governance?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.OperationApproval{}, &model.OperationAudit{}); err != nil {
		t.Fatalf("migrate governance tables: %v", err)
	}

	handler := &Handler{
		svcCtx: &svc.ServiceContext{DB: db},
		repo:   NewRepository(db),
	}
	return handler, db
}

func TestHandlerPhase3Admission_RegistersPolicyAndVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, db := newPhase3AdmissionTestHandler(t)
	handler.trivy = stubTrivyClient{result: clusterintegration.TrivyScanResult{Summary: clusterintegration.TrivySummary{Critical: 0, High: 1}}}

	body := strings.NewReader(`{"policy_name":"deny-privileged","version":"v1","image":"repo/api:1.2.3","approval_token":"` + mustIssuePhase3Approval(t, db, "admission.apply", "deny-privileged") + `"}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/clusters/42/admission/policies", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "42"}}
	ctx.Set("uid", uint64(1001))

	handler.UpsertAdmissionPolicy(ctx)

	env := decodeOperationEnvelope(t, recorder)
	if env.State != OperationStateCompleted {
		t.Fatalf("expected state %q, got %q, body=%s", OperationStateCompleted, env.State, recorder.Body.String())
	}
	if env.Code != OperationCodeSuccess {
		t.Fatalf("expected code %q, got %q", OperationCodeSuccess, env.Code)
	}

	var count int64
	if err := db.Model(&model.AdmissionPolicy{}).
		Where("cluster_id = ? AND policy_name = ? AND version = ?", 42, "deny-privileged", "v1").
		Count(&count).Error; err != nil {
		t.Fatalf("count admission policy: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one admission policy record, got %d", count)
	}
}

func TestHandlerPhase3Admission_BlocksCriticalVulnFromTrivy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, db := newPhase3AdmissionTestHandler(t)
	handler.trivy = stubTrivyClient{result: clusterintegration.TrivyScanResult{Summary: clusterintegration.TrivySummary{Critical: 2, High: 0}}}

	body := strings.NewReader(`{"policy_name":"deny-privileged","version":"v2","image":"repo/api:2.0.0","approval_token":"` + mustIssuePhase3Approval(t, db, "admission.apply", "deny-privileged") + `"}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/clusters/42/admission/policies", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "42"}}
	ctx.Set("uid", uint64(1001))

	handler.UpsertAdmissionPolicy(ctx)

	env := decodeOperationEnvelope(t, recorder)
	if env.State != OperationStateRejected {
		t.Fatalf("expected state %q, got %q", OperationStateRejected, env.State)
	}
	if env.Code != "admission_denied" {
		t.Fatalf("expected code admission_denied, got %q", env.Code)
	}

	var count int64
	if err := db.Model(&model.AdmissionPolicy{}).Where("cluster_id = ? AND policy_name = ?", 42, "deny-privileged").Count(&count).Error; err != nil {
		t.Fatalf("count admission policy: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no policy persisted when trivy blocks, got %d", count)
	}
}

func TestHandlerPhase3Admission_ExemptionRequiresApproval(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newPhase3AdmissionTestHandler(t)

	body := strings.NewReader(`{"scope_type":"namespace","scope_ref":"prod","reason":"temporary exception","expires_at":"2026-04-10T00:00:00Z"}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/clusters/42/admission/exemptions", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "42"}}
	ctx.Set("uid", uint64(1001))

	handler.CreateAdmissionExemption(ctx)

	env := decodeOperationEnvelope(t, recorder)
	if env.State != OperationStateApprovalRequired {
		t.Fatalf("expected state %q, got %q", OperationStateApprovalRequired, env.State)
	}
	if env.Code != OperationCodeApprovalRequired {
		t.Fatalf("expected code %q, got %q", OperationCodeApprovalRequired, env.Code)
	}
	if env.Approval == nil || env.Approval.Ticket == "" {
		t.Fatalf("expected approval ticket in response")
	}
}

func TestRouteAdmission_RegistersPhase3AdmissionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newPhase3AdmissionTestHandler(t)

	engine := gin.New()
	v1 := engine.Group("/api/v1")
	RegisterClusterHandlers(v1, handler.svcCtx)

	want := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/clusters/:id/admission/policies"},
		{method: http.MethodGet, path: "/api/v1/clusters/:id/admission/results"},
		{method: http.MethodPost, path: "/api/v1/clusters/:id/admission/exemptions"},
		{method: http.MethodPost, path: "/api/v1/clusters/:id/admission/exemptions/:exemption_id/revoke"},
	}

	routes := engine.Routes()
	for _, routeWant := range want {
		found := false
		for _, route := range routes {
			if route.Method == routeWant.method && route.Path == routeWant.path {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected route %s %s to be registered", routeWant.method, routeWant.path)
		}
	}
}

func newPhase3AdmissionTestHandler(t *testing.T) (*Handler, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:cluster-phase3-admission-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Cluster{},
		&model.OperationApproval{},
		&model.OperationAudit{},
		&model.AdmissionPolicy{},
		&model.AdmissionExemption{},
	); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS image_scan_reports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cluster_id INTEGER NOT NULL,
		image_digest VARCHAR(255) NOT NULL,
		scanner VARCHAR(32) NOT NULL,
		severity_summary_json TEXT NOT NULL DEFAULT '{}',
		sbom_ref VARCHAR(255) NOT NULL DEFAULT '',
		policy_decision VARCHAR(32) NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`).Error; err != nil {
		t.Fatalf("create image_scan_reports: %v", err)
	}

	if err := db.Create(&model.Cluster{
		ID:      42,
		Name:    "phase3-test-cluster",
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

func mustIssuePhase3Approval(t *testing.T, db *gorm.DB, action string, resourceID string) string {
	t.Helper()
	ctx := context.Background()
	handler := &Handler{svcCtx: &svc.ServiceContext{DB: db}}
	intent := governance.OperationIntent{
		OperatorID: 1001,
		Scope: governance.Scope{
			Domain:     "cluster",
			ClusterID:  42,
			Resource:   "admission",
			ResourceID: resourceID,
			Action:     action,
			Context: map[string]any{
				"phase3_domain": "admission",
				"intent":        "policy_upsert",
			},
		},
		OccurredAt: time.Now().UTC(),
	}
	decision, err := handler.phase3Preflight(ctx, intent)
	if err != nil {
		t.Fatalf("issue approval by preflight: %v", err)
	}
	if decision.Approval == nil || decision.Approval.Ticket == "" {
		t.Fatalf("expected approval ticket to be issued")
	}
	if err := db.Model(&model.OperationApproval{}).
		Where("ticket = ?", decision.Approval.Ticket).
		Updates(map[string]any{
			"status":    "approved",
			"review_by": 2001,
		}).Error; err != nil {
		t.Fatalf("approve issued ticket: %v", err)
	}
	return decision.Approval.Ticket
}

type stubTrivyClient struct {
	result clusterintegration.TrivyScanResult
	err    error
}

func (s stubTrivyClient) ScanImage(_ context.Context, _ string) (clusterintegration.TrivyScanResult, error) {
	return s.result, s.err
}
