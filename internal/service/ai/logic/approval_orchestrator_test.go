package logic

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	common "github.com/cy77cc/OpsPilot/internal/ai/common/approval"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEvaluateRequiresApprovalByPolicy(t *testing.T) {
	db := newApprovalOrchestratorTestDB(t)
	if err := db.Create(&model.AIToolRiskPolicy{
		ToolName:          "k8s_scale_deployment",
		Scene:             strPtr("cluster"),
		CommandClass:      strPtr("write"),
		ArgumentRulesJSON: strPtr(`{"namespace":"prod"}`),
		ApprovalRequired:  true,
		PolicyVersion:     "policy-v1",
		Priority:          50,
		Enabled:           true,
	}).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}

	orchestrator := NewApprovalOrchestratorWithStores(
		aidao.NewAIToolRiskPolicyDAO(db),
		aidao.NewAIApprovalTaskDAO(db),
		aidao.NewAIApprovalOutboxDAO(db),
	)

	decision, err := orchestrator.Evaluate(context.Background(), "k8s_scale_deployment", `{"namespace":"prod","replicas":3}`, common.ApprovalEvalMeta{
		Scene:          "cluster",
		CommandClass:   "write",
		SessionID:      "session-1",
		RunID:          "run-1",
		CheckpointID:   "checkpoint-1",
		CallID:         "call-1",
		UserID:         42,
		TimeoutSeconds: 120,
	})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !decision.RequiresApproval {
		t.Fatal("expected approval to be required")
	}
	if decision.ApprovalID == "" {
		t.Fatal("expected approval id to be assigned")
	}
	if decision.MatchedRuleID == nil || *decision.MatchedRuleID == 0 {
		t.Fatalf("expected matched rule id, got %#v", decision.MatchedRuleID)
	}
	if decision.PolicyVersion != "policy-v1" {
		t.Fatalf("expected policy version policy-v1, got %q", decision.PolicyVersion)
	}
	if decision.DecisionSource != "db_policy" {
		t.Fatalf("expected db_policy decision source, got %q", decision.DecisionSource)
	}
	if decision.TimeoutSeconds != 120 {
		t.Fatalf("expected timeout 120, got %d", decision.TimeoutSeconds)
	}
	if decision.ExpiresAt.IsZero() {
		t.Fatal("expected expires_at to be set")
	}

	task, err := aidao.NewAIApprovalTaskDAO(db).GetByApprovalID(context.Background(), decision.ApprovalID)
	if err != nil {
		t.Fatalf("load approval task: %v", err)
	}
	if task == nil {
		t.Fatal("expected approval task to be created")
	}
	if task.CheckpointID != "checkpoint-1" {
		t.Fatalf("expected checkpoint id checkpoint-1, got %q", task.CheckpointID)
	}
	if task.Status != "pending" {
		t.Fatalf("expected pending task, got %q", task.Status)
	}
	if task.MatchedRuleID == nil || *task.MatchedRuleID != *decision.MatchedRuleID {
		t.Fatalf("expected task matched rule id to persist, got %#v", task.MatchedRuleID)
	}
	if task.PolicyVersion == nil || *task.PolicyVersion != "policy-v1" {
		t.Fatalf("expected task policy version policy-v1, got %#v", task.PolicyVersion)
	}
	if task.DecisionSource == nil || *task.DecisionSource != "db_policy" {
		t.Fatalf("expected task decision source db_policy, got %#v", task.DecisionSource)
	}
	if task.ExpiresAt == nil || task.ExpiresAt.IsZero() {
		t.Fatal("expected task expires_at to be set")
	}
	if task.ToolName != "k8s_scale_deployment" || task.ToolCallID != "call-1" {
		t.Fatalf("unexpected task identity fields: %#v", task)
	}

	var outbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", decision.ApprovalID, "approval_requested").First(&outbox).Error; err != nil {
		t.Fatalf("load approval_requested outbox: %v", err)
	}
	if outbox.Status != "pending" {
		t.Fatalf("expected pending outbox event, got %q", outbox.Status)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(outbox.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}
	if payload["approval_id"] != decision.ApprovalID {
		t.Fatalf("expected approval_id in outbox payload, got %#v", payload)
	}

	resumeTask, err := aidao.NewAIApprovalTaskDAO(db).GetByCheckpointID(context.Background(), "checkpoint-1")
	if err != nil {
		t.Fatalf("get by checkpoint id: %v", err)
	}
	if resumeTask == nil || resumeTask.ApprovalID != decision.ApprovalID {
		t.Fatalf("expected checkpoint lookup to find approval task, got %#v", resumeTask)
	}
}

func TestApprovalOrchestrator_EvaluateRequiresApprovalByPolicy(t *testing.T) {
	TestEvaluateRequiresApprovalByPolicy(t)
}

func TestEvaluateFallsBackSafeOnPolicyError(t *testing.T) {
	orchestrator := NewApprovalOrchestratorWithStores(
		riskPolicyStoreFunc(func(context.Context, string) ([]model.AIToolRiskPolicy, error) {
			return nil, errors.New("policy lookup failed")
		}),
		approvalTaskStoreFunc(func(context.Context, *model.AIApprovalTask) error { return nil }),
		outboxStoreFunc(func(context.Context, *model.AIApprovalOutboxEvent) error { return nil }),
	)

	decision, err := orchestrator.Evaluate(context.Background(), "k8s_scale_deployment", `{"namespace":"prod","replicas":3}`, common.ApprovalEvalMeta{
		Scene:          "cluster",
		CommandClass:   "write",
		SessionID:      "session-2",
		RunID:          "run-2",
		CheckpointID:   "checkpoint-2",
		CallID:         "call-2",
		UserID:         7,
		TimeoutSeconds: 90,
	})
	if err != nil {
		t.Fatalf("evaluate should fall back safely, got error: %v", err)
	}
	if !decision.RequiresApproval {
		t.Fatal("expected safe fallback to require approval")
	}
	if decision.DecisionSource != "fallback_safe" {
		t.Fatalf("expected fallback_safe decision source, got %q", decision.DecisionSource)
	}
	if decision.ApprovalID == "" {
		t.Fatal("expected approval id to be assigned on fallback")
	}
}

func TestEvaluateFallsBackSafeOnPolicyErrorLeavesSafeToolsOpen(t *testing.T) {
	orchestrator := NewApprovalOrchestratorWithStores(
		riskPolicyStoreFunc(func(context.Context, string) ([]model.AIToolRiskPolicy, error) {
			return nil, errors.New("policy lookup failed")
		}),
		approvalTaskStoreFunc(func(context.Context, *model.AIApprovalTask) error {
			t.Fatal("safe tool should not create approval task on policy error")
			return nil
		}),
		outboxStoreFunc(func(context.Context, *model.AIApprovalOutboxEvent) error {
			t.Fatal("safe tool should not enqueue outbox event on policy error")
			return nil
		}),
	)

	decision, err := orchestrator.Evaluate(context.Background(), "host_batch_exec_preview", `{"host_ids":[1],"command":"uptime"}`, common.ApprovalEvalMeta{
		Scene:          "cluster",
		CommandClass:   "readonly",
		SessionID:      "session-3",
		RunID:          "run-3",
		CheckpointID:   "checkpoint-3",
		CallID:         "call-3",
		UserID:         7,
		TimeoutSeconds: 90,
	})
	if err != nil {
		t.Fatalf("evaluate should keep safe tools open, got error: %v", err)
	}
	if decision.RequiresApproval {
		t.Fatal("expected safe fallback to leave readonly tool open")
	}
	if decision.DecisionSource != "fallback_safe" {
		t.Fatalf("expected fallback_safe decision source, got %q", decision.DecisionSource)
	}
}

func TestEvaluateFallsBackSafeOnPolicyErrorProtectsServiceControlBatch(t *testing.T) {
	orchestrator := NewApprovalOrchestratorWithStores(
		riskPolicyStoreFunc(func(context.Context, string) ([]model.AIToolRiskPolicy, error) {
			return nil, errors.New("policy lookup failed")
		}),
		approvalTaskStoreFunc(func(context.Context, *model.AIApprovalTask) error { return nil }),
		outboxStoreFunc(func(context.Context, *model.AIApprovalOutboxEvent) error { return nil }),
	)

	decision, err := orchestrator.Evaluate(context.Background(), "host_batch_exec_preview", `{"host_ids":[1],"command":"systemctl restart nginx"}`, common.ApprovalEvalMeta{
		Scene:          "cluster",
		CommandClass:   "service_control",
		SessionID:      "session-unsafe",
		RunID:          "run-unsafe",
		CheckpointID:   "checkpoint-unsafe",
		CallID:         "call-unsafe",
		UserID:         7,
		TimeoutSeconds: 90,
	})
	if err != nil {
		t.Fatalf("evaluate should use safe fallback, got error: %v", err)
	}
	if !decision.RequiresApproval {
		t.Fatal("expected safe fallback to require approval for service_control batch command")
	}
	if decision.DecisionSource != "fallback_safe" {
		t.Fatalf("expected fallback_safe decision source, got %q", decision.DecisionSource)
	}
}

func TestEvaluateFallsBackToLegacyGateWhenNoPolicyMatches(t *testing.T) {
	db := newApprovalOrchestratorTestDB(t)
	// Seed unrelated policy to ensure DB lookup succeeds but no rule matches the requested tool.
	if err := db.Create(&model.AIToolRiskPolicy{
		ToolName:         "host_batch_exec_preview",
		Scene:            strPtr("cluster"),
		CommandClass:     strPtr("readonly"),
		ApprovalRequired: false,
		PolicyVersion:    "policy-readonly",
		Priority:         10,
		Enabled:          true,
	}).Error; err != nil {
		t.Fatalf("seed unrelated policy: %v", err)
	}

	orchestrator := NewApprovalOrchestratorWithStores(
		aidao.NewAIToolRiskPolicyDAO(db),
		aidao.NewAIApprovalTaskDAO(db),
		aidao.NewAIApprovalOutboxDAO(db),
	)

	decision, err := orchestrator.Evaluate(context.Background(), "k8s_scale_deployment", `{"namespace":"prod","replicas":2}`, common.ApprovalEvalMeta{
		Scene:          "cluster",
		CommandClass:   "write",
		SessionID:      "session-fallback",
		RunID:          "run-fallback",
		CheckpointID:   "checkpoint-fallback",
		CallID:         "call-fallback",
		UserID:         99,
		TimeoutSeconds: 60,
	})
	if err != nil {
		t.Fatalf("evaluate no-match fallback: %v", err)
	}
	if !decision.RequiresApproval {
		t.Fatal("expected legacy fallback gate to require approval for high-risk tool")
	}
	if decision.DecisionSource != "fallback_safe" {
		t.Fatalf("expected fallback_safe source, got %q", decision.DecisionSource)
	}
}

func TestEvaluateDistinguishesReadonlyAndServiceControlBatchCalls(t *testing.T) {
	db := newApprovalOrchestratorTestDB(t)
	if err := db.Create(&model.AIToolRiskPolicy{
		ToolName:         "host_batch_exec_preview",
		Scene:            strPtr("cluster"),
		CommandClass:     strPtr("readonly"),
		ApprovalRequired: false,
		PolicyVersion:    "policy-readonly",
		Priority:         20,
		Enabled:          true,
	}).Error; err != nil {
		t.Fatalf("seed readonly policy: %v", err)
	}
	if err := db.Create(&model.AIToolRiskPolicy{
		ToolName:         "host_batch_exec_preview",
		Scene:            strPtr("cluster"),
		CommandClass:     strPtr("service_control"),
		ApprovalRequired: true,
		PolicyVersion:    "policy-service-control",
		Priority:         20,
		Enabled:          true,
	}).Error; err != nil {
		t.Fatalf("seed service_control policy: %v", err)
	}

	orchestrator := NewApprovalOrchestratorWithStores(
		aidao.NewAIToolRiskPolicyDAO(db),
		aidao.NewAIApprovalTaskDAO(db),
		aidao.NewAIApprovalOutboxDAO(db),
	)

	readonlyDecision, err := orchestrator.Evaluate(context.Background(), "host_batch_exec_preview", `{"host_ids":[1],"command":"uptime"}`, common.ApprovalEvalMeta{
		Scene:          "cluster",
		CommandClass:   "readonly",
		SessionID:      "session-4",
		RunID:          "run-4",
		CheckpointID:   "checkpoint-4",
		CallID:         "call-4",
		UserID:         7,
		TimeoutSeconds: 90,
	})
	if err != nil {
		t.Fatalf("readonly evaluate: %v", err)
	}
	if readonlyDecision.RequiresApproval {
		t.Fatalf("expected readonly batch call to be approved by policy without interruption")
	}

	serviceControlDecision, err := orchestrator.Evaluate(context.Background(), "host_batch_exec_preview", `{"host_ids":[1],"command":"systemctl restart nginx"}`, common.ApprovalEvalMeta{
		Scene:          "cluster",
		CommandClass:   "service_control",
		SessionID:      "session-5",
		RunID:          "run-5",
		CheckpointID:   "checkpoint-5",
		CallID:         "call-5",
		UserID:         7,
		TimeoutSeconds: 90,
	})
	if err != nil {
		t.Fatalf("service control evaluate: %v", err)
	}
	if !serviceControlDecision.RequiresApproval {
		t.Fatal("expected service_control batch call to require approval")
	}
	if serviceControlDecision.DecisionSource != "db_policy" {
		t.Fatalf("expected db_policy decision source, got %q", serviceControlDecision.DecisionSource)
	}
}

func TestApprovalOrchestrator_RollsBackTaskWhenOutboxEnqueueFails(t *testing.T) {
	db := newApprovalOrchestratorTestDB(t)
	if err := db.Create(&model.AIToolRiskPolicy{
		ToolName:         "k8s_scale_deployment",
		Scene:            strPtr("cluster"),
		CommandClass:     strPtr("write"),
		ApprovalRequired: true,
		PolicyVersion:    "policy-v1",
		Priority:         50,
		Enabled:          true,
	}).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}
	callbackName := "test:approval_outbox_create_failure"
	if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Schema == nil {
			return
		}
		if tx.Statement.Schema.Table != "ai_approval_outbox_events" {
			return
		}
		tx.AddError(errors.New("simulated outbox failure"))
	}); err != nil {
		t.Fatalf("register outbox failure callback: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Create().Remove(callbackName)
	})

	orchestrator := NewApprovalOrchestrator(db)
	decision, err := orchestrator.Evaluate(context.Background(), "k8s_scale_deployment", `{"namespace":"prod","replicas":3}`, common.ApprovalEvalMeta{
		Scene:          "cluster",
		CommandClass:   "write",
		SessionID:      "session-rollback",
		RunID:          "run-rollback",
		CheckpointID:   "checkpoint-rollback",
		CallID:         "call-rollback",
		UserID:         42,
		TimeoutSeconds: 120,
	})
	if err == nil {
		t.Fatalf("expected outbox failure to abort transaction, got decision %#v", decision)
	}

	var taskCount int64
	if err := db.Model(&model.AIApprovalTask{}).Where("checkpoint_id = ?", "checkpoint-rollback").Count(&taskCount).Error; err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if taskCount != 0 {
		t.Fatalf("expected approval task insert to roll back, found %d row(s)", taskCount)
	}
	var outboxCount int64
	if err := db.Model(&model.AIApprovalOutboxEvent{}).Where("run_id = ?", "run-rollback").Count(&outboxCount).Error; err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if outboxCount != 0 {
		t.Fatalf("expected outbox insert to roll back, found %d row(s)", outboxCount)
	}
}

func TestApprovalOrchestrator_EvaluateFallsBackSafeOnPolicyError(t *testing.T) {
	TestEvaluateFallsBackSafeOnPolicyError(t)
}

func newApprovalOrchestratorTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AIToolRiskPolicy{}, &model.AIApprovalTask{}, &model.AIApprovalOutboxEvent{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	return db
}

type riskPolicyStoreFunc func(context.Context, string) ([]model.AIToolRiskPolicy, error)

func (f riskPolicyStoreFunc) ListEnabledByToolName(ctx context.Context, toolName string) ([]model.AIToolRiskPolicy, error) {
	return f(ctx, toolName)
}

type approvalTaskStoreFunc func(context.Context, *model.AIApprovalTask) error

func (f approvalTaskStoreFunc) Create(ctx context.Context, task *model.AIApprovalTask) error {
	return f(ctx, task)
}

type outboxStoreFunc func(context.Context, *model.AIApprovalOutboxEvent) error

func (f outboxStoreFunc) EnqueueOrTouch(ctx context.Context, event *model.AIApprovalOutboxEvent) error {
	return f(ctx, event)
}

func strPtr(s string) *string { return &s }
