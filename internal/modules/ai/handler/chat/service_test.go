package chathandler

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
)

func TestService_NilLogicReturnsInitializationError(t *testing.T) {
	svc := NewServiceWithLogic(nil)
	ctx := context.Background()

	if _, err := svc.CreateSession(ctx, 1, "title", "ai"); err == nil {
		t.Fatal("expected CreateSession to fail when logic is nil")
	}
	if _, err := svc.ListSessions(ctx, 1, "ai"); err == nil {
		t.Fatal("expected ListSessions to fail when logic is nil")
	}
	if _, _, err := svc.GetSession(ctx, 1, "ai", "session-1"); err == nil {
		t.Fatal("expected GetSession to fail when logic is nil")
	}
	if _, err := svc.DeleteSession(ctx, 1, "session-1"); err == nil {
		t.Fatal("expected DeleteSession to fail when logic is nil")
	}
	if _, _, err := svc.GetRun(ctx, 1, "run-1"); err == nil {
		t.Fatal("expected GetRun to fail when logic is nil")
	}
	if _, err := svc.GetRunProjectionPayload(ctx, 1, "run-1", logic.RunProjectionQuery{}); err == nil {
		t.Fatal("expected GetRunProjectionPayload to fail when logic is nil")
	}
	if _, err := svc.GetRunContent(ctx, 1, "content-1"); err == nil {
		t.Fatal("expected GetRunContent to fail when logic is nil")
	}
	if _, err := svc.GetDiagnosisReport(ctx, 1, "report-1"); err == nil {
		t.Fatal("expected GetDiagnosisReport to fail when logic is nil")
	}
}
