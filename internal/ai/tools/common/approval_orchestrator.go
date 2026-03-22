package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ApprovalOrchestrator evaluates DB policies and persists approval snapshots when approval is required.
type ApprovalOrchestrator struct {
	policyStore approvalPolicyStore
	approvalDAO approvalTaskWriter
	outboxDAO   approvalOutboxWriter
	db          *gorm.DB
	now         func() time.Time
	newID       func() string
	defaultTTL  int
}

type approvalPolicyStore interface {
	ListEnabledByToolName(ctx context.Context, toolName string) ([]model.AIToolRiskPolicy, error)
}

type approvalTaskWriter interface {
	Create(ctx context.Context, task *model.AIApprovalTask) error
}

type approvalOutboxWriter interface {
	EnqueueOrTouch(ctx context.Context, event *model.AIApprovalOutboxEvent) error
}

// NewApprovalOrchestrator creates a DB-backed approval orchestrator.
func NewApprovalOrchestrator(db *gorm.DB) *ApprovalOrchestrator {
	if db == nil {
		return &ApprovalOrchestrator{
			now:        time.Now,
			newID:      uuid.NewString,
			defaultTTL: DefaultApprovalTimeout,
		}
	}
	return NewApprovalOrchestratorWithStores(
		aidao.NewAIToolRiskPolicyDAO(db),
		aidao.NewAIApprovalTaskDAO(db),
		aidao.NewAIApprovalOutboxDAO(db),
	).withDB(db)
}

// NewApprovalOrchestratorWithStores wires custom stores for tests or alternative integrations.
func NewApprovalOrchestratorWithStores(policy approvalPolicyStore, approval approvalTaskWriter, outbox approvalOutboxWriter) *ApprovalOrchestrator {
	return &ApprovalOrchestrator{
		policyStore: policy,
		approvalDAO: approval,
		outboxDAO:   outbox,
		now:         time.Now,
		newID:       uuid.NewString,
		defaultTTL:  DefaultApprovalTimeout,
	}
}

func (o *ApprovalOrchestrator) withDB(db *gorm.DB) *ApprovalOrchestrator {
	if o != nil {
		o.db = db
	}
	return o
}

// Evaluate resolves the DB policy decision for a tool call.
func (o *ApprovalOrchestrator) Evaluate(ctx context.Context, toolName string, args string, meta ApprovalEvalMeta) (*ApprovalDecision, error) {
	if o == nil {
		if fallbackRequiresApproval(toolName, meta.CommandClass) {
			return &ApprovalDecision{RequiresApproval: true, DecisionSource: "fallback_safe"}, nil
		}
		return &ApprovalDecision{RequiresApproval: false, DecisionSource: "fallback_safe"}, nil
	}

	now := o.nowOrDefault()
	timeoutSeconds := meta.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = o.defaultTTL
	}

	argsMap := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(args)), &argsMap); err != nil {
		argsMap = map[string]any{}
	}

	policies, err := o.listPolicies(ctx, toolName)
	if err != nil {
		return o.createFallbackDecision(ctx, toolName, args, meta, now, timeoutSeconds)
	}

	matched, ok := MatchRiskPolicy(policies, meta.Scene, meta.CommandClass, argsMap)
	if !ok || matched == nil {
		return o.createFallbackDecision(ctx, toolName, args, meta, now, timeoutSeconds)
	}
	if !matched.ApprovalRequired {
		return &ApprovalDecision{
			RequiresApproval: false,
			TimeoutSeconds:   timeoutSeconds,
		}, nil
	}

	return o.createApprovalDecision(ctx, toolName, args, meta, now, timeoutSeconds, matched, "db_policy")
}

func (o *ApprovalOrchestrator) createFallbackDecision(
	ctx context.Context,
	toolName, args string,
	meta ApprovalEvalMeta,
	now time.Time,
	timeoutSeconds int,
) (*ApprovalDecision, error) {
	if !fallbackRequiresApproval(toolName, meta.CommandClass) {
		return &ApprovalDecision{
			RequiresApproval: false,
			TimeoutSeconds:   timeoutSeconds,
			DecisionSource:   "fallback_safe",
		}, nil
	}
	return o.createApprovalDecision(ctx, toolName, args, meta, now, timeoutSeconds, nil, "fallback_safe")
}

func (o *ApprovalOrchestrator) createApprovalDecision(
	ctx context.Context,
	toolName, args string,
	meta ApprovalEvalMeta,
	now time.Time,
	timeoutSeconds int,
	matched *model.AIToolRiskPolicy,
	decisionSource string,
) (*ApprovalDecision, error) {
	if o.approvalDAO == nil || o.outboxDAO == nil {
		return nil, fmt.Errorf("approval persistence not initialized")
	}
	if strings.TrimSpace(meta.CheckpointID) == "" {
		return nil, fmt.Errorf("approval checkpoint id is empty")
	}

	approvalID := o.newIDOrDefault()
	expiresAt := now.Add(time.Duration(timeoutSeconds) * time.Second)
	preview := buildApprovalPreview(toolName, args, matched)

	task := &model.AIApprovalTask{
		ApprovalID:     approvalID,
		CheckpointID:   meta.CheckpointID,
		SessionID:      meta.SessionID,
		RunID:          meta.RunID,
		UserID:         meta.UserID,
		ToolName:       toolName,
		ToolCallID:     meta.CallID,
		ArgumentsJSON:  strings.TrimSpace(args),
		PreviewJSON:    mustJSONString(preview),
		Status:         "pending",
		TimeoutSeconds: timeoutSeconds,
		ExpiresAt:      &expiresAt,
		DecisionSource: ptrString(decisionSource),
	}
	if matched != nil {
		task.MatchedRuleID = &matched.ID
		task.PolicyVersion = ptrString(strings.TrimSpace(matched.PolicyVersion))
	}

	eventPayload := map[string]any{
		"approval_id":     approvalID,
		"call_id":         meta.CallID,
		"tool_name":       toolName,
		"session_id":      meta.SessionID,
		"run_id":          meta.RunID,
		"preview":         preview,
		"timeout_seconds": timeoutSeconds,
		"expires_at":      expiresAt.UTC().Format(time.RFC3339Nano),
		"decision_source": decisionSource,
	}
	if matched != nil {
		eventPayload["matched_rule_id"] = matched.ID
		eventPayload["policy_version"] = strings.TrimSpace(matched.PolicyVersion)
	}
	event := &model.AIApprovalOutboxEvent{
		ApprovalID:  approvalID,
		EventType:   "approval_requested",
		RunID:       meta.RunID,
		SessionID:   meta.SessionID,
		PayloadJSON: mustJSONString(eventPayload),
		Status:      "pending",
	}

	if o.db != nil {
		err := o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			txApproval := aidao.NewAIApprovalTaskDAO(tx)
			txOutbox := aidao.NewAIApprovalOutboxDAO(tx)
			if err := txApproval.Create(ctx, task); err != nil {
				return err
			}
			if err := txOutbox.EnqueueOrTouch(ctx, event); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		if err := o.approvalDAO.Create(ctx, task); err != nil {
			return nil, err
		}
		if err := o.outboxDAO.EnqueueOrTouch(ctx, event); err != nil {
			return nil, err
		}
	}

	return &ApprovalDecision{
		RequiresApproval: true,
		ApprovalID:       approvalID,
		Preview:          preview,
		TimeoutSeconds:   timeoutSeconds,
		MatchedRuleID:    task.MatchedRuleID,
		PolicyVersion:    valueOrEmpty(task.PolicyVersion),
		DecisionSource:   decisionSource,
		ExpiresAt:        expiresAt,
	}, nil
}

func (o *ApprovalOrchestrator) listPolicies(ctx context.Context, toolName string) ([]model.AIToolRiskPolicy, error) {
	if o.policyStore == nil {
		return nil, fmt.Errorf("risk policy store not initialized")
	}
	return o.policyStore.ListEnabledByToolName(ctx, toolName)
}

func (o *ApprovalOrchestrator) nowOrDefault() time.Time {
	if o != nil && o.now != nil {
		return o.now().UTC()
	}
	return time.Now().UTC()
}

func (o *ApprovalOrchestrator) newIDOrDefault() string {
	if o != nil && o.newID != nil {
		return o.newID()
	}
	return uuid.NewString()
}

func buildApprovalPreview(toolName, args string, matched *model.AIToolRiskPolicy) ApprovalPreview {
	preview := ApprovalPreview{
		Action:    toolName,
		RiskLevel: RiskLevelMedium,
	}

	if matched != nil && strings.TrimSpace(matched.RiskLevel) != "" {
		preview.RiskLevel = strings.TrimSpace(matched.RiskLevel)
	}

	var params map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(args)), &params); err == nil {
		if target, ok := params["target"].(string); ok && strings.TrimSpace(target) != "" {
			preview.Target = strings.TrimSpace(target)
		}
		if name, ok := params["name"].(string); ok && strings.TrimSpace(name) != "" && preview.Target == "" {
			preview.Target = strings.TrimSpace(name)
		}
		if action, ok := params["action"].(string); ok && strings.TrimSpace(action) != "" {
			preview.Action = strings.TrimSpace(action)
		}
		if cmd, ok := params["command"].(string); ok && strings.TrimSpace(cmd) != "" {
			preview.Action = strings.TrimSpace(cmd)
		}
	}

	return preview
}

func mustJSONString(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func ptrString(v string) *string {
	v = strings.TrimSpace(v)
	return &v
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

func fallbackRequiresApproval(toolName, commandClass string) bool {
	commandClass = strings.TrimSpace(strings.ToLower(commandClass))
	if commandClass != "" && commandClass != "readonly" {
		switch strings.TrimSpace(toolName) {
		case "host_batch", "host_exec_change", "host_batch_exec_apply", "host_batch_exec_preview", "host_batch_status_update":
			return true
		}
	}
	return defaultFallbackRequiresApproval(toolName)
}

func defaultFallbackRequiresApproval(toolName string) bool {
	toolName = strings.TrimSpace(toolName)
	switch toolName {
	case "host_batch", "host_exec_change", "host_batch_exec_apply", "host_batch_status_update",
		"k8s_scale_deployment", "k8s_restart_deployment", "k8s_delete_pod",
		"k8s_rollback_deployment", "k8s_delete_deployment",
		"cicd_trigger_pipeline", "cicd_cancel_pipeline",
		"service_restart", "service_scale", "service_update_config":
		return true
	default:
		return false
	}
}
