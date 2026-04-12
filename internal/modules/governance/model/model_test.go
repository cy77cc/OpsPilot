package model

import "testing"

func TestOperationGovernance_TableNames(t *testing.T) {
	if got := (OperationApproval{}).TableName(); got != "operation_approvals" {
		t.Fatalf("expected operation_approvals, got %q", got)
	}
	if got := (OperationAudit{}).TableName(); got != "operation_audits" {
		t.Fatalf("expected operation_audits, got %q", got)
	}
}
