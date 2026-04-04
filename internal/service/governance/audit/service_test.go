package audit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/governance"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestService_RecordRedactsSensitivePayloads(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:gov-audit?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.OperationAudit{}); err != nil {
		t.Fatalf("migrate audits: %v", err)
	}

	current := time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)
	svc := NewServiceWithClock(db, nil, func() time.Time { return current })
	ctx := governance.WithOperationContext(context.Background(), governance.OperationContext{
		TeamID:      8,
		Environment: "production",
		Values:      map[string]any{"region": "ap-southeast-1"},
	})

	started := current.Add(-1500 * time.Millisecond)
	in := governance.FinalizeInput{
		Intent: governance.OperationIntent{
			OperatorID:    55,
			ApprovalToken: "gov-appr-9",
			Scope: governance.Scope{
				Domain:      "cluster",
				ClusterID:   3,
				TeamID:      8,
				Namespace:   "prod",
				Environment: "production",
				Resource:    "node",
				ResourceID:  "node-1",
				Action:      "drain",
				Context:     map[string]any{"trace_id": "abc-123"},
			},
			RequestSummary: map[string]any{
				"authorization": "Bearer secret-token",
				"nested": map[string]any{
					"api_key": "secret-key",
				},
				"plain": "keep-me",
			},
		},
		Decision: governance.Decision{
			Allowed: true,
			State:   governance.StateCompleted,
			Code:    governance.CodeSuccess,
			Message: "completed",
		},
		Result: map[string]any{
			"token":   "super-secret",
			"changed": true,
		},
		Diagnostics: map[string]any{
			"password": "p@ssword",
			"status":   "ok",
		},
		StartedAt:  started,
		FinishedAt: current,
	}

	id, err := svc.Record(ctx, in)
	if err != nil {
		t.Fatalf("record audit: %v", err)
	}

	var rec model.OperationAudit
	if err := db.First(&rec, id).Error; err != nil {
		t.Fatalf("reload audit record: %v", err)
	}
	if rec.Status != "completed" || rec.Code != governance.CodeSuccess {
		t.Fatalf("unexpected status/code: %#v", rec)
	}
	if rec.LatencyMS != 1500 {
		t.Fatalf("expected latency 1500ms, got %d", rec.LatencyMS)
	}
	if rec.ApprovalTicket != "gov-appr-9" {
		t.Fatalf("expected approval ticket to be stored, got %q", rec.ApprovalTicket)
	}

	for _, secret := range []string{"secret-token", "secret-key", "super-secret", "p@ssword"} {
		if strings.Contains(rec.RequestSummaryJSON, secret) || strings.Contains(rec.ResultSummaryJSON, secret) || strings.Contains(rec.DiagnosticsJSON, secret) {
			t.Fatalf("expected secret %q to be redacted: %#v", secret, rec)
		}
	}
	for _, redacted := range []string{"***", "keep-me", "changed", "ok"} {
		if !strings.Contains(rec.RequestSummaryJSON, redacted) && !strings.Contains(rec.ResultSummaryJSON, redacted) && !strings.Contains(rec.DiagnosticsJSON, redacted) {
			t.Fatalf("expected payloads to retain %q, got %#v", redacted, rec)
		}
	}

	var request map[string]any
	if err := json.Unmarshal([]byte(rec.RequestSummaryJSON), &request); err != nil {
		t.Fatalf("unmarshal request summary: %v", err)
	}
	if request["authorization"] != "***" {
		t.Fatalf("expected authorization to be redacted, got %#v", request)
	}
}
