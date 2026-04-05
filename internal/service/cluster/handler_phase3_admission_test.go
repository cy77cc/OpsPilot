package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/governance"
	"github.com/cy77cc/OpsPilot/internal/svc"
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
