package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	governanceapproval "github.com/cy77cc/OpsPilot/internal/service/governance/approval"
	"gorm.io/gorm"
)

func TestPolicyReleaseStateMachine_TracksReleaseIDAndStableLifecycle(t *testing.T) {
	now := time.Date(2026, time.April, 5, 10, 0, 0, 0, time.UTC)
	release := newPolicyReleaseForTest(t, now)

	if release.ReleaseID != 101 {
		t.Fatalf("expected release_id 101, got %d", release.ReleaseID)
	}
	if release.PreviousStableVersion != "stable-v1" {
		t.Fatalf("expected previous stable version stable-v1, got %q", release.PreviousStableVersion)
	}
	if release.Status.Phase != PolicyReleaseStateSimulationPassed {
		t.Fatalf("expected initial phase %q, got %q", PolicyReleaseStateSimulationPassed, release.Status.Phase)
	}

	if err := release.MarkApplied(now); err == nil {
		t.Fatal("expected applying to be required before applied")
	}
	if err := release.MarkApplying(now); err != nil {
		t.Fatalf("mark applying: %v", err)
	}
	if release.Status.Phase != PolicyReleaseStateApplying {
		t.Fatalf("expected phase %q, got %q", PolicyReleaseStateApplying, release.Status.Phase)
	}

	appliedAt := now.Add(2 * time.Minute)
	if err := release.MarkApplied(appliedAt); err != nil {
		t.Fatalf("mark applied: %v", err)
	}
	if release.Status.Phase != PolicyReleaseStateApplied {
		t.Fatalf("expected phase %q, got %q", PolicyReleaseStateApplied, release.Status.Phase)
	}
	if release.Audit.AppliedAt == nil || !release.Audit.AppliedAt.Equal(appliedAt) {
		t.Fatalf("expected applied_at %v, got %#v", appliedAt, release.Audit.AppliedAt)
	}

	if err := release.MarkActive(); err != nil {
		t.Fatalf("mark active: %v", err)
	}
	if release.Status.Phase != PolicyReleaseStateActive {
		t.Fatalf("expected phase %q, got %q", PolicyReleaseStateActive, release.Status.Phase)
	}
}

func TestPolicyReleaseApproval_MissingTokenTransitionsToApprovalRequired(t *testing.T) {
	_, db := newHighRiskApprovalTestHandler(t)
	now := time.Date(2026, time.April, 5, 10, 15, 0, 0, time.UTC)
	release := newPolicyReleaseForTest(t, now)

	if err := release.EnsureApproval(context.Background(), db, "", 1001, now); err != nil {
		t.Fatalf("ensure approval: %v", err)
	}
	if release.Status.Phase != PolicyReleaseStateApprovalRequired {
		t.Fatalf("expected phase %q, got %q", PolicyReleaseStateApprovalRequired, release.Status.Phase)
	}
	if !release.Approval.Required {
		t.Fatal("expected approval to be required")
	}
	if release.Approval.ApprovalToken == "" {
		t.Fatal("expected approval ticket to be issued")
	}

	var approval model.OperationApproval
	if err := db.Where("ticket = ?", release.Approval.ApprovalToken).First(&approval).Error; err != nil {
		t.Fatalf("load approval: %v", err)
	}
	if approval.Action != PolicyReleaseApprovalActionApply {
		t.Fatalf("expected approval action %q, got %q", PolicyReleaseApprovalActionApply, approval.Action)
	}
	if approval.Resource != PolicyReleaseApprovalResource {
		t.Fatalf("expected approval resource %q, got %q", PolicyReleaseApprovalResource, approval.Resource)
	}
	if approval.ResourceID != "101" {
		t.Fatalf("expected approval resource id 101, got %q", approval.ResourceID)
	}
}

func TestPolicyReleaseApproval_ApprovedTokenTransitionsToApplying(t *testing.T) {
	_, db := newHighRiskApprovalTestHandler(t)
	now := time.Date(2026, time.April, 5, 10, 30, 0, 0, time.UTC)
	release := newPolicyReleaseForTest(t, now)
	token := issuePolicyReleaseTicket(t, db, release, 1001, now.Add(30*time.Minute), true)

	if err := release.EnsureApproval(context.Background(), db, token, 1001, now); err != nil {
		t.Fatalf("ensure approval with approved token: %v", err)
	}
	if release.Status.Phase != PolicyReleaseStateApplying {
		t.Fatalf("expected phase %q, got %q", PolicyReleaseStateApplying, release.Status.Phase)
	}
	if release.Approval.ApprovalToken != token {
		t.Fatalf("expected approval token %q, got %q", token, release.Approval.ApprovalToken)
	}
	if release.Approval.ApprovedAt == nil || !release.Approval.ApprovedAt.Equal(now) {
		t.Fatalf("expected approved_at %v, got %#v", now, release.Approval.ApprovedAt)
	}
}

func TestPolicyReleaseApprovalTokenValidation_ExpiredTokenKeepsReleaseAwaitingApproval(t *testing.T) {
	_, db := newHighRiskApprovalTestHandler(t)
	now := time.Date(2026, time.April, 5, 10, 45, 0, 0, time.UTC)
	release := newPolicyReleaseForTest(t, now)
	token := issuePolicyReleaseTicket(t, db, release, 1001, now.Add(-1*time.Minute), true)

	err := release.EnsureApproval(context.Background(), db, token, 1001, now)
	approvalErr, ok := IsApprovalError(err)
	if !ok {
		t.Fatalf("expected approval error, got %v", err)
	}
	if approvalErr.Code != approvalTokenExpiredCode {
		t.Fatalf("expected code %q, got %q", approvalTokenExpiredCode, approvalErr.Code)
	}
	if release.Status.Phase != PolicyReleaseStateApprovalRequired {
		t.Fatalf("expected phase %q after expired token, got %q", PolicyReleaseStateApprovalRequired, release.Status.Phase)
	}
	if release.LastErrorCode != approvalTokenExpiredCode {
		t.Fatalf("expected last error code %q, got %q", approvalTokenExpiredCode, release.LastErrorCode)
	}
}

func TestPolicyReleaseApproval_RejectedTokenTransitionsToApprovalRejected(t *testing.T) {
	_, db := newHighRiskApprovalTestHandler(t)
	now := time.Date(2026, time.April, 5, 11, 0, 0, 0, time.UTC)
	release := newPolicyReleaseForTest(t, now)
	token := issuePolicyReleaseTicket(t, db, release, 1001, now.Add(30*time.Minute), false)

	if err := release.EnsureApproval(context.Background(), db, token, 1001, now); err != nil {
		t.Fatalf("ensure approval with rejected token: %v", err)
	}
	if release.Status.Phase != PolicyReleaseStateApprovalRejected {
		t.Fatalf("expected phase %q, got %q", PolicyReleaseStateApprovalRejected, release.Status.Phase)
	}
}

func TestPolicyReleaseRollbackEndToEnd_ApplyFailedUsesPreviousStableVersion(t *testing.T) {
	now := time.Date(2026, time.April, 5, 11, 15, 0, 0, time.UTC)
	release := newPolicyReleaseForTest(t, now)

	if err := release.MarkApplying(now); err != nil {
		t.Fatalf("mark applying: %v", err)
	}
	if err := release.MarkApplyFailed(PolicyErrorApplyValidationFailed, "adapter rejected manifest"); err != nil {
		t.Fatalf("mark apply failed: %v", err)
	}
	if release.Status.Phase != PolicyReleaseStateApplyFailed {
		t.Fatalf("expected phase %q, got %q", PolicyReleaseStateApplyFailed, release.Status.Phase)
	}
	if release.LastErrorCode != PolicyErrorApplyValidationFailed {
		t.Fatalf("expected last error code %q, got %q", PolicyErrorApplyValidationFailed, release.LastErrorCode)
	}

	rollbackAt := now.Add(3 * time.Minute)
	if err := release.Rollback("", rollbackAt); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if release.Status.Phase != PolicyReleaseStateRollbackApplied {
		t.Fatalf("expected phase %q, got %q", PolicyReleaseStateRollbackApplied, release.Status.Phase)
	}
	if release.RollbackTargetVersion != "stable-v1" {
		t.Fatalf("expected rollback target stable-v1, got %q", release.RollbackTargetVersion)
	}
	if release.Audit.RollbackAt == nil || !release.Audit.RollbackAt.Equal(rollbackAt) {
		t.Fatalf("expected rollback_at %v, got %#v", rollbackAt, release.Audit.RollbackAt)
	}
}

func newPolicyReleaseForTest(t *testing.T, now time.Time) *PolicyReleaseRecord {
	t.Helper()

	release, err := NewPolicyRelease(PolicyReleaseCreateInput{
		ReleaseID:             101,
		Version:               "candidate-v2",
		PreviousStableVersion: "stable-v1",
		Policy: PolicyReference{
			APIVersion: PolicyDefinitionAPIVersion,
			Kind:       PolicyDefinitionKind,
			Name:       "allow-api",
			Namespace:  "prod",
		},
		TargetCluster: PolicyTargetCluster{
			ClusterID:  42,
			CNIType:    "cilium",
			CNIVersion: "1.17.0",
		},
		SimulationResult: PolicySimulationResult{
			Passed: true,
			PolicySimulationStatus: PolicySimulationStatus{
				PassedAt: &now,
			},
			PolicyReleaseStatus: PolicyReleaseStatus{
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

	return release
}

func issuePolicyReleaseTicket(t *testing.T, db *gorm.DB, release *PolicyReleaseRecord, requestedBy uint, expiresAt time.Time, approved bool) string {
	t.Helper()

	rec, err := IssuePolicyReleaseApproval(context.Background(), db, release.TargetCluster.ClusterID, release.Policy.Namespace, release.ReleaseID, PolicyReleaseApprovalActionApply, requestedBy, expiresAt)
	if err != nil {
		t.Fatalf("issue policy release approval: %v", err)
	}
	if !expiresAt.IsZero() {
		if err := db.Model(&model.OperationApproval{}).Where("ticket = ?", rec.Ticket).Update("expires_at", expiresAt.UTC()).Error; err != nil {
			t.Fatalf("pin policy release approval expiry: %v", err)
		}
	}

	svc := governanceapproval.NewService(db)
	if err := svc.Confirm(context.Background(), rec.Ticket, 9001, approved, "reviewed"); err != nil {
		t.Fatalf("confirm policy release approval: %v", err)
	}

	return rec.Ticket
}
