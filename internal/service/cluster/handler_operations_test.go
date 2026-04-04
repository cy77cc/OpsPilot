package cluster

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	governanceapproval "github.com/cy77cc/OpsPilot/internal/service/governance/approval"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRequireHighRiskApproval_MissingTokenCreatesApprovalAndAudit(t *testing.T) {
	handler, db := newHighRiskApprovalTestHandler(t)
	scope := ApprovalScope{
		ClusterID:  42,
		Action:     "node.drain",
		Resource:   "node",
		ResourceID: "node-1",
	}

	result := handler.requireHighRiskApproval(context.Background(), scope.ClusterID, scope.Namespace, scope.Action, scope.Resource, scope.ResourceID, "", 1001)
	if result.Allowed {
		t.Fatalf("expected missing token to require approval")
	}
	if result.Code != clusterOperationCodeApprovalRequired {
		t.Fatalf("expected code %q, got %q", clusterOperationCodeApprovalRequired, result.Code)
	}
	if result.ApprovalTicket == "" {
		t.Fatalf("expected approval ticket to be issued")
	}
	if result.AuditID == 0 {
		t.Fatalf("expected pending audit to be recorded")
	}

	resp := operationResponseFromGate(scope.ClusterID, scope.Resource, scope.ResourceID, result)
	if resp.State != OperationStateApprovalRequired {
		t.Fatalf("expected response state %q, got %q", OperationStateApprovalRequired, resp.State)
	}

	var approval model.OperationApproval
	if err := db.Where("ticket = ?", result.ApprovalTicket).First(&approval).Error; err != nil {
		t.Fatalf("expected approval record: %v", err)
	}
	if approval.Status != "pending" {
		t.Fatalf("expected pending approval status, got %q", approval.Status)
	}

	var audit model.OperationAudit
	if err := db.First(&audit, result.AuditID).Error; err != nil {
		t.Fatalf("expected audit record: %v", err)
	}
	if audit.Status != OperationStateApprovalRequired {
		t.Fatalf("expected audit status %q, got %q", OperationStateApprovalRequired, audit.Status)
	}
	if audit.Code != clusterOperationCodeApprovalRequired {
		t.Fatalf("expected audit code %q, got %q", clusterOperationCodeApprovalRequired, audit.Code)
	}
}

func TestRequireHighRiskApproval_ExpiredTokenReturnsStableFailure(t *testing.T) {
	handler, db := newHighRiskApprovalTestHandler(t)
	scope := ApprovalScope{
		ClusterID:  42,
		Action:     "node.drain",
		Resource:   "node",
		ResourceID: "node-1",
	}
	token := issueApprovalTicket(t, db, scope, 1001, time.Now().UTC().Add(-1*time.Minute), true)

	result := handler.requireHighRiskApproval(context.Background(), scope.ClusterID, scope.Namespace, scope.Action, scope.Resource, scope.ResourceID, token, 1001)
	if result.Allowed {
		t.Fatalf("expected expired token to be rejected")
	}
	if result.Code != approvalTokenExpiredCode {
		t.Fatalf("expected code %q, got %q", approvalTokenExpiredCode, result.Code)
	}
	if result.AuditID == 0 {
		t.Fatalf("expected failure audit to be recorded")
	}

	resp := operationResponseFromGate(scope.ClusterID, scope.Resource, scope.ResourceID, result)
	if resp.State != OperationStateFailed {
		t.Fatalf("expected response state %q, got %q", OperationStateFailed, resp.State)
	}

	assertAuditRecord(t, db, result.AuditID, OperationStateFailed, approvalTokenExpiredCode)
}

func TestRequireHighRiskApproval_ReplayedTokenReturnsStableFailure(t *testing.T) {
	handler, db := newHighRiskApprovalTestHandler(t)
	scope := ApprovalScope{
		ClusterID:  42,
		Action:     "node.drain",
		Resource:   "node",
		ResourceID: "node-1",
	}
	token := issueApprovalTicket(t, db, scope, 1001, time.Now().UTC().Add(5*time.Minute), true)

	first := handler.requireHighRiskApproval(context.Background(), scope.ClusterID, scope.Namespace, scope.Action, scope.Resource, scope.ResourceID, token, 1001)
	if !first.Allowed {
		t.Fatalf("expected approved token to pass on first consume, got %+v", first)
	}

	second := handler.requireHighRiskApproval(context.Background(), scope.ClusterID, scope.Namespace, scope.Action, scope.Resource, scope.ResourceID, token, 1001)
	if second.Allowed {
		t.Fatalf("expected replayed token to fail")
	}
	if second.Code != ApprovalTokenReplayedCode {
		t.Fatalf("expected code %q, got %q", ApprovalTokenReplayedCode, second.Code)
	}

	resp := operationResponseFromGate(scope.ClusterID, scope.Resource, scope.ResourceID, second)
	if resp.State != OperationStateFailed {
		t.Fatalf("expected response state %q, got %q", OperationStateFailed, resp.State)
	}

	var approval model.OperationApproval
	if err := db.Where("ticket = ?", token).First(&approval).Error; err != nil {
		t.Fatalf("expected approval record: %v", err)
	}
	if approval.ReplayCount != 1 {
		t.Fatalf("expected replay count 1, got %d", approval.ReplayCount)
	}
	if approval.ReplayCode != ApprovalTokenReplayedCode {
		t.Fatalf("expected replay code %q, got %q", ApprovalTokenReplayedCode, approval.ReplayCode)
	}

	assertAuditRecord(t, db, second.AuditID, OperationStateFailed, ApprovalTokenReplayedCode)
}

func TestRequireHighRiskApproval_RejectedTokenReturnsRejectedState(t *testing.T) {
	handler, db := newHighRiskApprovalTestHandler(t)
	scope := ApprovalScope{
		ClusterID:  42,
		Action:     "node.drain",
		Resource:   "node",
		ResourceID: "node-1",
	}
	token := issueApprovalTicket(t, db, scope, 1001, time.Now().UTC().Add(5*time.Minute), false)

	result := handler.requireHighRiskApproval(context.Background(), scope.ClusterID, scope.Namespace, scope.Action, scope.Resource, scope.ResourceID, token, 1001)
	if result.Allowed {
		t.Fatalf("expected rejected approval to block operation")
	}
	if result.Code != clusterOperationCodeApprovalRejected {
		t.Fatalf("expected code %q, got %q", clusterOperationCodeApprovalRejected, result.Code)
	}

	resp := operationResponseFromGate(scope.ClusterID, scope.Resource, scope.ResourceID, result)
	if resp.State != OperationStateRejected {
		t.Fatalf("expected response state %q, got %q", OperationStateRejected, resp.State)
	}

	assertAuditRecord(t, db, result.AuditID, OperationStateRejected, clusterOperationCodeApprovalRejected)
}

func TestRequireHighRiskApproval_BypassesApprovalForApprovePermissions(t *testing.T) {
	cases := []struct {
		name           string
		username       string
		roleCode       string
		permissionCode string
	}{
		{name: "exact approve permission", username: "approver1", roleCode: "cluster-approver", permissionCode: "k8s:approve"},
		{name: "wildcard approve permission", username: "approver2", roleCode: "cluster-operator", permissionCode: "k8s:*"},
		{name: "admin role", username: "adminusr", roleCode: "admin", permissionCode: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler, db := newHighRiskApprovalTestHandler(t)
			userID := seedRBACPermission(t, db, tc.username, tc.roleCode, tc.permissionCode)

			result := handler.requireHighRiskApproval(context.Background(), 42, "", "node.drain", "node", "node-1", "", userID)
			if !result.Allowed {
				t.Fatalf("expected approve-capable operator to bypass approval gate, got %+v", result)
			}
			if result.Code != clusterOperationCodeSuccess {
				t.Fatalf("expected code %q, got %q", clusterOperationCodeSuccess, result.Code)
			}

			var approvalCount int64
			if err := db.Model(&model.OperationApproval{}).Count(&approvalCount).Error; err != nil {
				t.Fatalf("count approvals: %v", err)
			}
			if approvalCount != 0 {
				t.Fatalf("expected no approval tickets to be created, got %d", approvalCount)
			}

			var auditCount int64
			if err := db.Model(&model.OperationAudit{}).Count(&auditCount).Error; err != nil {
				t.Fatalf("count audits: %v", err)
			}
			if auditCount != 0 {
				t.Fatalf("expected no audits for direct bypass, got %d", auditCount)
			}
		})
	}
}

func newHighRiskApprovalTestHandler(t *testing.T) (*Handler, *gorm.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:cluster-approval-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.OperationApproval{},
		&model.OperationAudit{},
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.UserRole{},
		&model.RolePermission{},
	); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	if err := db.Create(&model.User{
		ID:           1001,
		Username:     "operator",
		PasswordHash: "hash",
		Status:       1,
	}).Error; err != nil {
		t.Fatalf("seed operator user: %v", err)
	}

	return &Handler{
		svcCtx: &svc.ServiceContext{DB: db},
	}, db
}

func issueApprovalTicket(t *testing.T, db *gorm.DB, scope ApprovalScope, requestedBy uint, expiresAt time.Time, approved bool) string {
	t.Helper()

	rec, err := IssueClusterDeployApproval(context.Background(), db, scope, requestedBy, expiresAt)
	if err != nil {
		t.Fatalf("issue approval: %v", err)
	}
	if !expiresAt.IsZero() {
		if err := db.Model(&model.OperationApproval{}).Where("ticket = ?", rec.Ticket).Update("expires_at", expiresAt.UTC()).Error; err != nil {
			t.Fatalf("pin approval expiry: %v", err)
		}
	}

	svc := governanceapproval.NewService(db)
	if err := svc.Confirm(context.Background(), rec.Ticket, 9001, approved, "reviewed"); err != nil {
		t.Fatalf("confirm approval: %v", err)
	}

	var approval model.OperationApproval
	if err := db.Where("ticket = ?", rec.Ticket).First(&approval).Error; err != nil {
		t.Fatalf("reload issued approval: %v", err)
	}
	if approval.Domain != "cluster" {
		t.Fatalf("expected domain cluster, got %q", approval.Domain)
	}
	if approval.ScopeClusterID == nil || *approval.ScopeClusterID != scope.ClusterID {
		t.Fatalf("expected cluster scope %d, got %+v", scope.ClusterID, approval.ScopeClusterID)
	}
	if approval.Action != scope.Action {
		t.Fatalf("expected action %q, got %q", scope.Action, approval.Action)
	}
	if approval.Namespace != scope.Namespace {
		t.Fatalf("expected namespace %q, got %q", scope.Namespace, approval.Namespace)
	}
	if approval.Resource != scope.Resource {
		t.Fatalf("expected resource %q, got %q", scope.Resource, approval.Resource)
	}
	if approval.ResourceID != scope.ResourceID {
		t.Fatalf("expected resource id %q, got %q", scope.ResourceID, approval.ResourceID)
	}
	if approval.ContextJSON != "{\"approval_scope\":\"cluster\"}" {
		t.Fatalf("expected cluster approval context, got %q", approval.ContextJSON)
	}

	return rec.Ticket
}

func assertAuditRecord(t *testing.T, db *gorm.DB, auditID uint, wantStatus, wantCode string) {
	t.Helper()

	if auditID == 0 {
		t.Fatalf("expected audit id to be set")
	}
	var audit model.OperationAudit
	if err := db.First(&audit, auditID).Error; err != nil {
		t.Fatalf("load audit record: %v", err)
	}
	if audit.Status != wantStatus {
		t.Fatalf("expected audit status %q, got %q", wantStatus, audit.Status)
	}
	if audit.Code != wantCode {
		t.Fatalf("expected audit code %q, got %q", wantCode, audit.Code)
	}
}

func seedRBACPermission(t *testing.T, db *gorm.DB, username, roleCode, permissionCode string) uint64 {
	t.Helper()

	user := model.User{
		Username:     username,
		PasswordHash: "hash",
		Status:       1,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	role := model.Role{
		Name:   roleCode,
		Code:   roleCode,
		Status: 1,
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := db.Create(&model.UserRole{UserID: int64(user.ID), RoleID: int64(role.ID)}).Error; err != nil {
		t.Fatalf("create user role: %v", err)
	}

	if permissionCode == "" {
		return uint64(user.ID)
	}

	permission := model.Permission{
		Name:   permissionCode,
		Code:   permissionCode,
		Type:   1,
		Status: 1,
	}
	if err := db.Create(&permission).Error; err != nil {
		t.Fatalf("create permission: %v", err)
	}
	if err := db.Create(&model.RolePermission{RoleID: int64(role.ID), PermissionID: int64(permission.ID)}).Error; err != nil {
		t.Fatalf("create role permission: %v", err)
	}
	return uint64(user.ID)
}
