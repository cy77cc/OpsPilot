package logic

import "testing"

func TestOperationResponseConstructors(t *testing.T) {
	t.Run("completed response uses canonical code", func(t *testing.T) {
		resp := NewCompletedOperationResponse(12, map[string]any{"ok": true})
		if resp.State != OperationStateCompleted {
			t.Fatalf("expected state %q, got %q", OperationStateCompleted, resp.State)
		}
		if resp.Code != OperationCodeSuccess {
			t.Fatalf("expected code %q, got %q", OperationCodeSuccess, resp.Code)
		}
		if resp.AuditID != 12 {
			t.Fatalf("expected audit id 12, got %d", resp.AuditID)
		}
	})

	t.Run("approval required response uses canonical code", func(t *testing.T) {
		resp := NewApprovalRequiredOperationResponse(&OperationApproval{Ticket: "appr-1"}, 7, "needs approval", nil)
		if resp.State != OperationStateApprovalRequired {
			t.Fatalf("expected state %q, got %q", OperationStateApprovalRequired, resp.State)
		}
		if resp.Code != OperationCodeApprovalRequired {
			t.Fatalf("expected code %q, got %q", OperationCodeApprovalRequired, resp.Code)
		}
		if resp.Approval == nil || resp.Approval.Ticket != "appr-1" {
			t.Fatalf("expected approval ticket appr-1")
		}
	})

	t.Run("rejected response uses canonical code", func(t *testing.T) {
		resp := NewRejectedOperationResponse(9, "rejected", nil)
		if resp.State != OperationStateRejected {
			t.Fatalf("expected state %q, got %q", OperationStateRejected, resp.State)
		}
		if resp.Code != OperationCodeApprovalRejected {
			t.Fatalf("expected code %q, got %q", OperationCodeApprovalRejected, resp.Code)
		}
	})

	t.Run("failed response defaults code", func(t *testing.T) {
		resp := NewFailedOperationResponse(3, "", "failed", nil)
		if resp.State != OperationStateFailed {
			t.Fatalf("expected state %q, got %q", OperationStateFailed, resp.State)
		}
		if resp.Code != OperationCodeFailed {
			t.Fatalf("expected code %q, got %q", OperationCodeFailed, resp.Code)
		}
	})

	t.Run("failed response keeps explicit code", func(t *testing.T) {
		resp := NewFailedOperationResponse(3, "approval_token_replayed", "failed", nil)
		if resp.Code != "approval_token_replayed" {
			t.Fatalf("expected custom failure code preserved, got %q", resp.Code)
		}
	})
}
