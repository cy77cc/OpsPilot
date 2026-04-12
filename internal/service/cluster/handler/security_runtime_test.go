package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/governance"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHandlerPhase3Runtime_IngestFalcoEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, db := newPhase3RuntimeTestHandler(t, ClusterModePlatformManaged)

	body := strings.NewReader(`{"source":"falco","payload":{"rule":"Terminal shell in container","priority":"Critical","output_fields":{"k8s.ns.name":"prod","k8s.pod.name":"api-1"}}}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/clusters/42/security/events/ingest", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "42"}}
	ctx.Set("uid", uint64(1001))

	handler.IngestRuntimeEvent(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected http 200, got %d", recorder.Code)
	}
	var count int64
	if err := db.Model(&model.RuntimeSecurityEvent{}).Where("cluster_id = ? AND source = ?", 42, model.SecurityEventSourceFalco).Count(&count).Error; err != nil {
		t.Fatalf("count runtime events: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one ingested runtime event, got %d", count)
	}
}

func TestHandlerPhase3Runtime_ListAlertsSupportsSeverityAndPageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, db := newPhase3RuntimeTestHandler(t, ClusterModePlatformManaged)
	seedRuntimeEvent(t, db, 42, "prod", "api-critical", "rule-1", model.SecuritySeverityCritical, model.SecurityEventSourceFalco)
	seedRuntimeEvent(t, db, 42, "prod", "api-high-1", "rule-2", model.SecuritySeverityHigh, model.SecurityEventSourceFalco)
	seedRuntimeEvent(t, db, 42, "prod", "api-high-2", "rule-3", model.SecuritySeverityHigh, model.SecurityEventSourceFalco)
	seedRuntimeEvent(t, db, 42, "prod", "api-low", "rule-4", model.SecuritySeverityLow, model.SecurityEventSourceFalco)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/clusters/42/security/alerts?severity=high&page_size=1", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "42"}}

	handler.ListRuntimeAlerts(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected http 200, got %d", recorder.Code)
	}
	payload := decodeRuntimeAlertsResponse(t, recorder)
	if len(payload.Data.List) != 1 {
		t.Fatalf("expected one alert by page_size=1, got %d", len(payload.Data.List))
	}
	if strings.ToLower(strings.TrimSpace(payload.Data.List[0].Severity)) != model.SecuritySeverityHigh {
		t.Fatalf("expected severity high, got %q", payload.Data.List[0].Severity)
	}
	if payload.Data.Total != 1 {
		t.Fatalf("expected total to reflect filtered page size of 1, got %d", payload.Data.Total)
	}
}

func TestHandlerPhase3Runtime_ContainPlatformManagedAuto(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, db := newPhase3RuntimeTestHandler(t, ClusterModePlatformManaged)
	eventID := seedRuntimeEvent(t, db, 42, "prod", "api-1", "rule-x", model.SecuritySeverityHigh, model.SecurityEventSourceFalco)

	body := strings.NewReader(`{"approval_token":"` + mustIssuePhase3RuntimeApproval(t, db, eventID) + `"}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/clusters/42/security/alerts/%d/contain", eventID), body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "alert_id", Value: fmt.Sprintf("%d", eventID)}}
	ctx.Set("uid", uint64(1001))

	handler.ContainRuntimeAlert(ctx)

	env := decodeOperationEnvelope(t, recorder)
	if env.State != OperationStateCompleted {
		t.Fatalf("expected completed state, got %q", env.State)
	}
	contain, ok := env.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected contain result map in envelope data")
	}
	if contain["mode"] != model.DisposalModeAuto {
		t.Fatalf("expected auto mode, got %v", contain["mode"])
	}
}

func TestHandlerPhase3Runtime_ContainExternalManagedSuggestOnlyWithAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, db := newPhase3RuntimeTestHandler(t, ClusterModeExternalManaged)
	eventID := seedRuntimeEvent(t, db, 42, "prod", "api-1", "rule-x", model.SecuritySeverityHigh, model.SecurityEventSourceFalco)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/clusters/42/security/alerts/%d/contain", eventID), strings.NewReader(`{}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "alert_id", Value: fmt.Sprintf("%d", eventID)}}
	ctx.Set("uid", uint64(1001))

	handler.ContainRuntimeAlert(ctx)

	env := decodeOperationEnvelope(t, recorder)
	if env.State != OperationStateCompleted {
		t.Fatalf("expected completed state, got %q", env.State)
	}
	contain, ok := env.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected contain result map in envelope data")
	}
	if contain["mode"] != model.DisposalModeSuggestOnly {
		t.Fatalf("expected suggest_only mode, got %v", contain["mode"])
	}
	if env.AuditID == 0 {
		t.Fatalf("expected non-zero audit id for suggest_only path")
	}
}

func newPhase3RuntimeTestHandler(t *testing.T, source string) (*Handler, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:cluster-phase3-runtime-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Cluster{},
		&model.OperationApproval{},
		&model.OperationAudit{},
		&model.RuntimeSecurityEvent{},
	); err != nil {
		t.Fatalf("migrate runtime tables: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS runtime_disposal_actions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id INTEGER NOT NULL,
		action VARCHAR(64) NOT NULL,
		mode VARCHAR(32) NOT NULL,
		approval_id INTEGER NOT NULL DEFAULT 0,
		audit_id INTEGER NOT NULL DEFAULT 0,
		result VARCHAR(32) NOT NULL DEFAULT 'pending',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`).Error; err != nil {
		t.Fatalf("create runtime_disposal_actions: %v", err)
	}
	if err := db.Create(&model.Cluster{
		ID:      42,
		Name:    "phase3-runtime-test-cluster",
		Status:  "active",
		Type:    "kubernetes",
		Source:  source,
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

func seedRuntimeEvent(t *testing.T, db *gorm.DB, clusterID uint, namespace, workload, ruleID, severity, src string) uint {
	t.Helper()
	rec := model.RuntimeSecurityEvent{
		ClusterID:      clusterID,
		Namespace:      namespace,
		Workload:       workload,
		RuleID:         ruleID,
		Severity:       severity,
		Source:         src,
		RawPayloadJSON: "{}",
		DisposeStatus:  "pending",
	}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("seed runtime event: %v", err)
	}
	return rec.ID
}

func mustIssuePhase3RuntimeApproval(t *testing.T, db *gorm.DB, eventID uint) string {
	t.Helper()
	h := &Handler{svcCtx: &svc.ServiceContext{DB: db}}
	decision, err := h.phase3Preflight(context.Background(), governance.OperationIntent{
		OperatorID: 1001,
		OccurredAt: time.Now().UTC(),
		Scope: governance.Scope{
			Domain:     "cluster",
			ClusterID:  42,
			Resource:   "runtime",
			ResourceID: fmt.Sprintf("%d", eventID),
			Action:     "runtime.contain",
			Context: map[string]any{
				"phase3_domain": "runtime",
				"intent":        "contain",
			},
		},
	})
	if err != nil {
		t.Fatalf("issue runtime approval: %v", err)
	}
	if decision.Approval == nil || decision.Approval.Ticket == "" {
		t.Fatalf("expected runtime approval ticket")
	}
	if err := db.Model(&model.OperationApproval{}).Where("ticket = ?", decision.Approval.Ticket).Updates(map[string]any{"status": "approved", "review_by": 2001}).Error; err != nil {
		t.Fatalf("approve runtime ticket: %v", err)
	}
	return decision.Approval.Ticket
}

type runtimeAlertsHTTPResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		List  []model.RuntimeSecurityEvent `json:"list"`
		Total int                          `json:"total"`
	} `json:"data"`
}

func decodeRuntimeAlertsResponse(t *testing.T, recorder *httptest.ResponseRecorder) runtimeAlertsHTTPResponse {
	t.Helper()
	var payload runtimeAlertsHTTPResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime alerts response: %v", err)
	}
	return payload
}
