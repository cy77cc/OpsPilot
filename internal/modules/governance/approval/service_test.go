package approval

import (
	"context"
	"testing"
	"time"

	governance "github.com/cy77cc/OpsPilot/internal/modules/governance"
	model "github.com/cy77cc/OpsPilot/internal/modules/governance/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestService_ConsumeEnforcesScopeReplayAndExpiry(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:gov-approval?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.OperationApproval{}); err != nil {
		t.Fatalf("migrate approvals: %v", err)
	}

	current := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	svc := NewServiceWithOptions(db, func() time.Time { return current }, time.Minute)

	ctx := context.Background()
	intent := governance.OperationIntent{
		OperatorID: 7,
		Scope: governance.Scope{
			Domain:      "cluster",
			ClusterID:   1,
			TeamID:      2,
			Environment: "production",
			Namespace:   "prod",
			Resource:    "node",
			ResourceID:  "node-1",
			Action:      "cordon",
			Context:     map[string]any{"tier": "critical"},
		},
	}

	info, err := svc.Issue(ctx, intent, "production cordon requires approval")
	if err != nil {
		t.Fatalf("issue approval: %v", err)
	}
	if info.Ticket == "" {
		t.Fatalf("expected ticket to be generated")
	}

	if err := svc.Confirm(ctx, info.Ticket, 99, true, "approved"); err != nil {
		t.Fatalf("confirm approval: %v", err)
	}

	intent.ApprovalToken = info.Ticket
	if err := svc.Consume(ctx, intent); err != nil {
		t.Fatalf("consume approval: %v", err)
	}

	err = svc.Consume(ctx, intent)
	if !governance.IsCode(err, governance.CodeApprovalTokenReplay) {
		t.Fatalf("expected replay error, got %v", err)
	}

	var rec model.OperationApproval
	if err := db.Where("ticket = ?", info.Ticket).First(&rec).Error; err != nil {
		t.Fatalf("reload approval record: %v", err)
	}
	if rec.ConsumedAt == nil || rec.ConsumedBy != intent.OperatorID {
		t.Fatalf("expected consumed record, got %#v", rec)
	}
	if rec.ReplayCount != 1 || rec.ReplayBy != intent.OperatorID {
		t.Fatalf("expected replay metadata to be recorded, got %#v", rec)
	}

	badScope := intent
	badScope.Scope.Environment = "staging"
	if err := svc.Consume(ctx, badScope); !governance.IsCode(err, governance.CodeApprovalScopeMismatch) {
		t.Fatalf("expected scope mismatch, got %v", err)
	}

	expiredIntent := intent
	current = current.Add(2 * time.Minute)
	expiredInfo, err := svc.Issue(ctx, expiredIntent, "production cordon requires approval")
	if err != nil {
		t.Fatalf("issue expired approval: %v", err)
	}
	if err := svc.Confirm(ctx, expiredInfo.Ticket, 99, true, "approved"); err != nil {
		t.Fatalf("confirm expired approval: %v", err)
	}
	current = current.Add(2 * time.Minute)
	expiredIntent.ApprovalToken = expiredInfo.Ticket
	if err := svc.Consume(ctx, expiredIntent); !governance.IsCode(err, governance.CodeApprovalTokenExpired) {
		t.Fatalf("expected expiry error, got %v", err)
	}
}

func TestService_ConfirmReturnsNotFoundErrorForMissingTicket(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:gov-approval-notfound?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.OperationApproval{}); err != nil {
		t.Fatalf("migrate approvals: %v", err)
	}
	svc := NewService(db)
	err = svc.Confirm(context.Background(), "missing", 1, true, "")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !governance.IsCode(err, governance.CodeApprovalTokenInvalid) {
		t.Fatalf("expected invalid ticket error, got %v", err)
	}
}
