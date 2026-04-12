package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	clustermodel "github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestCNIInfo_ReportsClusterCapabilityMatrixFromBootstrapState(t *testing.T) {
	handler, db := newOperationHistoryTestHandler(t)
	if err := db.AutoMigrate(&clustermodel.ClusterBootstrapTask{}); err != nil {
		t.Fatalf("migrate bootstrap task: %v", err)
	}

	if err := db.Create(&clustermodel.ClusterBootstrapTask{
		ID:                 "bootstrap-42",
		Name:               "cluster-42-bootstrap",
		ClusterID:          uintPtr(42),
		CNI:                "cilium",
		Status:             "success",
		ResolvedConfigJSON: `{"cniVersion":"1.17.0","flannel":{"netpol":{"enabled":true}}}`,
	}).Error; err != nil {
		t.Fatalf("create bootstrap task: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/clusters/42/cni-info", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "42"}}

	handler.GetCNIInfo(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var body struct {
		Data struct {
			ClusterID    uint            `json:"cluster_id"`
			CNIType      string          `json:"cni_type"`
			CNIVersion   string          `json:"cni_version"`
			Capabilities map[string]bool `json:"capabilities"`
			Constraints  map[string]any  `json:"constraints"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode cni info: %v", err)
	}
	_ = engine

	if body.Data.ClusterID != 42 {
		t.Fatalf("expected cluster id 42, got %d", body.Data.ClusterID)
	}
	if body.Data.CNIType != "cilium" {
		t.Fatalf("expected cni type cilium, got %q", body.Data.CNIType)
	}
	if body.Data.CNIVersion != "1.17.0" {
		t.Fatalf("expected cni version 1.17.0, got %q", body.Data.CNIVersion)
	}
	if !body.Data.Capabilities["l7"] {
		t.Fatalf("expected cilium l7 capability to be true")
	}
	if !body.Data.Capabilities["fqdn"] {
		t.Fatalf("expected cilium fqdn capability to be true")
	}
}

func TestHandlerPolicy_ApplyReleaseMissingApprovalReturnsUnifiedEnvelope(t *testing.T) {
	handler, db := newOperationHistoryTestHandler(t)
	releaseID := seedPolicyReleaseAudit(t, db, 42, "prod", "allow-api", "candidate-v2", "cilium", PolicyReleaseStateSimulationPassed)

	body := strings.NewReader(`{"version":"candidate-v2"}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/clusters/42/releases/"+releaseID+"/apply", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{
		{Key: "id", Value: "42"},
		{Key: "release_id", Value: releaseID},
	}
	ctx.Set("uid", uint64(1001))

	handler.ApplyPolicyRelease(ctx)

	payload := decodeOperationEnvelope(t, recorder)
	if payload.State != OperationStateApprovalRequired {
		t.Fatalf("expected state %q, got %q", OperationStateApprovalRequired, payload.State)
	}
	if payload.Code != OperationCodeApprovalRequired {
		t.Fatalf("expected code %q, got %q", OperationCodeApprovalRequired, payload.Code)
	}
	if payload.Approval == nil || payload.Approval.Ticket == "" {
		t.Fatalf("expected approval ticket in unified envelope")
	}
	if payload.AuditID == 0 {
		t.Fatalf("expected audit id in unified envelope")
	}
}

func TestReleaseVersion_ApplyReleaseRejectsMismatchedRequestVersion(t *testing.T) {
	handler, db := newOperationHistoryTestHandler(t)
	releaseID := seedPolicyReleaseAudit(t, db, 42, "prod", "allow-api", "candidate-v2", "cilium", PolicyReleaseStateSimulationPassed)

	body := strings.NewReader(`{"version":"candidate-v3"}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/clusters/42/releases/"+releaseID+"/apply", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{
		{Key: "id", Value: "42"},
		{Key: "release_id", Value: releaseID},
	}
	ctx.Set("uid", uint64(1001))

	handler.ApplyPolicyRelease(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var bodyResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &bodyResp); err != nil {
		t.Fatalf("decode mismatch response: %v", err)
	}
	if bodyResp.Code == 1000 {
		t.Fatalf("expected mismatched version to be rejected")
	}
	if !strings.Contains(bodyResp.Msg, "release version mismatch") {
		t.Fatalf("expected mismatch error message, got %q", bodyResp.Msg)
	}

	release, err := handler.repo.GetPolicyReleaseRecord(ctx.Request.Context(), 42, 501)
	if err != nil {
		t.Fatalf("reload policy release: %v", err)
	}
	if release.Version != "candidate-v2" {
		t.Fatalf("expected canonical version candidate-v2, got %q", release.Version)
	}

	var auditCount int64
	if err := db.Model(&clustermodel.OperationAudit{}).
		Where("domain = ? AND scope_cluster_id = ? AND resource = ? AND resource_id = ? AND action = ?", "cluster", 42, PolicyReleaseApprovalResource, "501", PolicyReleaseApprovalActionApply).
		Count(&auditCount).Error; err != nil {
		t.Fatalf("count apply audits: %v", err)
	}
	if auditCount != 0 {
		t.Fatalf("expected no apply audit to be recorded for mismatched version, got %d", auditCount)
	}
}

func TestPolicyAudit_OperationHistoryDetailIncludesApprovalTrace(t *testing.T) {
	handler, db := newOperationHistoryTestHandler(t)
	releaseID := seedPolicyReleaseAudit(t, db, 42, "prod", "allow-api", "candidate-v2", "cilium", PolicyReleaseStateSimulationPassed)

	body := strings.NewReader(`{"version":"candidate-v2"}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/clusters/42/releases/"+releaseID+"/apply", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{
		{Key: "id", Value: "42"},
		{Key: "release_id", Value: releaseID},
	}
	ctx.Set("uid", uint64(1001))

	handler.ApplyPolicyRelease(ctx)

	envelope := decodeOperationEnvelope(t, recorder)
	if envelope.AuditID == 0 {
		t.Fatalf("expected audit id to inspect operation history")
	}

	historyRecorder := httptest.NewRecorder()
	historyCtx, _ := gin.CreateTestContext(historyRecorder)
	historyCtx.Request = httptest.NewRequest(http.MethodGet, "/clusters/42/operations/history?resource=policy_release", nil)
	historyCtx.Params = gin.Params{{Key: "id", Value: "42"}}

	handler.ListOperationHistory(historyCtx)

	history := decodeOperationHistoryResponse(t, historyRecorder)
	if len(history.List) == 0 {
		t.Fatalf("expected policy release history item")
	}
	if history.List[0].ResourceType != PolicyReleaseApprovalResource {
		t.Fatalf("expected resource type %q, got %q", PolicyReleaseApprovalResource, history.List[0].ResourceType)
	}
	if history.List[0].ResourceName != "allow-api" {
		t.Fatalf("expected policy resource name allow-api, got %q", history.List[0].ResourceName)
	}
	if history.List[0].Status != OperationStateApprovalRequired {
		t.Fatalf("expected canonical approval_required status, got %q", history.List[0].Status)
	}

	detailRecorder := httptest.NewRecorder()
	detailCtx, _ := gin.CreateTestContext(detailRecorder)
	detailCtx.Request = httptest.NewRequest(http.MethodGet, "/clusters/42/operations/"+releaseID, nil)
	detailCtx.Params = gin.Params{
		{Key: "id", Value: "42"},
		{Key: "audit_id", Value: toString(history.List[0].AuditID)},
	}

	handler.GetOperationAudit(detailCtx)

	var detailBody struct {
		Data OperationAuditDetail `json:"data"`
	}
	if err := json.Unmarshal(detailRecorder.Body.Bytes(), &detailBody); err != nil {
		t.Fatalf("decode operation detail: %v", err)
	}
	if detailBody.Data.Approval == nil || detailBody.Data.Approval.Ticket == "" {
		t.Fatalf("expected approval trace on operation audit detail")
	}
	if detailBody.Data.ResourceName != "allow-api" {
		t.Fatalf("expected resource name allow-api on detail, got %q", detailBody.Data.ResourceName)
	}
}

func decodeOperationEnvelope(t *testing.T, recorder *httptest.ResponseRecorder) OperationResponse {
	t.Helper()

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var body struct {
		Data OperationResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode operation envelope: %v", err)
	}
	return body.Data
}

func seedPolicyReleaseAudit(t *testing.T, db *gorm.DB, clusterID uint, namespace, policyName, version, cniType string, phase PolicyReleaseState) string {
	t.Helper()

	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	release, err := NewPolicyRelease(PolicyReleaseCreateInput{
		ReleaseID:             501,
		Version:               version,
		PreviousStableVersion: "stable-v1",
		Policy: PolicyReference{
			APIVersion: PolicyDefinitionAPIVersion,
			Kind:       PolicyDefinitionKind,
			Name:       policyName,
			Namespace:  namespace,
		},
		TargetCluster: PolicyTargetCluster{
			ClusterID:  clusterID,
			CNIType:    cniType,
			CNIVersion: "1.17.0",
		},
		SimulationResult: PolicySimulationResult{
			Passed:                 true,
			PolicySimulationStatus: PolicySimulationStatus{PassedAt: &now},
			PolicyReleaseStatus: PolicyReleaseStatus{
				Phase:     phase,
				RiskScore: 35,
				RiskLevel: PolicyRiskLevelMedium,
			},
		},
		CreatedBy: 1001,
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("new policy release: %v", err)
	}
	release.Status.Phase = phase

	rec := clustermodel.OperationAudit{
		Domain:         "cluster",
		ScopeClusterID: &clusterID,
		Namespace:      namespace,
		Resource:       PolicyReleaseApprovalResource,
		ResourceID:     "501",
		Action:         "policy.release.create",
		OperatorID:     1001,
		Status:         string(OperationStateCompleted),
		Code:           OperationCodeSuccess,
		Message:        "policy release created",
		ResultSummaryJSON: mustJSON(t, map[string]any{
			"release": release,
		}),
	}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("seed policy release audit: %v", err)
	}
	return "501"
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()

	buf, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return string(buf)
}

func uintPtr(v uint) *uint { return &v }

func toString(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}
