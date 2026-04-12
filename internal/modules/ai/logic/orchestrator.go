package logic

import (
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/approval"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	common "github.com/cy77cc/OpsPilot/internal/modules/ai/logic/shared/approval"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ApprovalOrchestrator struct {
	policyStore  approvalPolicyStore
	approvalDAO  approvalTaskWriter
	outboxDAO    approvalOutboxWriter
	riskRegistry *common.RiskRegistry
	db           *gorm.DB
	now          func() time.Time
	newID        func() string
	defaultTTL   int
}

type approvalPolicyStore interface {
	ListEnabledByToolName(ctx context.Context, toolName string) ([]ai.AIToolRiskPolicy, error)
}

type approvalTaskWriter interface {
	Create(ctx context.Context, task *ai.AIApprovalTask) error
}

type approvalOutboxWriter interface {
	EnqueueOrTouch(ctx context.Context, event *ai.AIApprovalOutboxEvent) error
}

func NewApprovalOrchestrator(db *gorm.DB) *ApprovalOrchestrator {
	if db == nil {
		return &ApprovalOrchestrator{
			riskRegistry: common.DefaultRiskRegistry(),
			now:          time.Now,
			newID:        uuid.NewString,
			defaultTTL:   common.DefaultApprovalTimeout,
		}
	}
	return NewApprovalOrchestratorWithStores(
		approvalPolicyDAO{db: db},
		approvalTaskDAO{db: db},
		approvalOutboxDAO{db: db},
	).withDB(db)
}

func NewApprovalOrchestratorWithStores(policy approvalPolicyStore, approval approvalTaskWriter, outbox approvalOutboxWriter) *ApprovalOrchestrator {
	return &ApprovalOrchestrator{
		policyStore:  policy,
		approvalDAO:  approval,
		outboxDAO:    outbox,
		riskRegistry: common.DefaultRiskRegistry(),
		now:          time.Now,
		newID:        uuid.NewString,
		defaultTTL:   common.DefaultApprovalTimeout,
	}
}

type approvalPolicyDAO struct{ db *gorm.DB }

func (d approvalPolicyDAO) ListEnabledByToolName(ctx context.Context, toolName string) ([]ai.AIToolRiskPolicy, error) {
	var policies []ai.AIToolRiskPolicy
	err := d.db.WithContext(ctx).
		Where("tool_name = ? AND enabled = ?", toolName, true).
		Order("priority DESC, id ASC").
		Find(&policies).Error
	return policies, err
}

type approvalTaskDAO struct{ db *gorm.DB }

func (d approvalTaskDAO) Create(ctx context.Context, task *ai.AIApprovalTask) error {
	return d.db.WithContext(ctx).Create(task).Error
}

type approvalOutboxDAO struct{ db *gorm.DB }

func (d approvalOutboxDAO) EnqueueOrTouch(ctx context.Context, event *ai.AIApprovalOutboxEvent) error {
	now := time.Now()
	return d.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "approval_id"}, {Name: "event_type"}},
			DoUpdates: clause.Assignments(map[string]any{
				"run_id":        gorm.Expr("CASE WHEN status = ? THEN ? ELSE run_id END", "pending", event.RunID),
				"session_id":    gorm.Expr("CASE WHEN status = ? THEN ? ELSE session_id END", "pending", event.SessionID),
				"tool_call_id":  gorm.Expr("CASE WHEN status = ? THEN ? ELSE tool_call_id END", "pending", event.ToolCallID),
				"payload_json":  gorm.Expr("CASE WHEN status = ? THEN ? ELSE payload_json END", "pending", event.PayloadJSON),
				"status":        gorm.Expr("CASE WHEN status = ? THEN ? ELSE status END", "pending", event.Status),
				"next_retry_at": gorm.Expr("CASE WHEN status = ? THEN ? ELSE next_retry_at END", "pending", event.NextRetryAt),
				"updated_at":    gorm.Expr("CASE WHEN status = ? THEN ? ELSE updated_at END", "pending", now),
			}),
		}).
		Create(event).Error
}

func (o *ApprovalOrchestrator) withDB(db *gorm.DB) *ApprovalOrchestrator {
	if o != nil {
		o.db = db
	}
	return o
}

func (o *ApprovalOrchestrator) Evaluate(ctx context.Context, toolName string, args string, meta common.ApprovalEvalMeta) (*common.ApprovalDecision, error) {
	if o == nil {
		if fallbackRequiresApproval(toolName, meta.CommandClass) {
			return &common.ApprovalDecision{RequiresApproval: true, DecisionSource: "fallback_safe"}, nil
		}
		return &common.ApprovalDecision{RequiresApproval: false, DecisionSource: "fallback_safe"}, nil
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
		return &common.ApprovalDecision{
			RequiresApproval: false,
			MatchedRuleID:    &matched.ID,
			PolicyVersion:    matched.PolicyVersion,
			DecisionSource:   "db_policy",
			TimeoutSeconds:   timeoutSeconds,
		}, nil
	}

	approvalID := o.nextID()
	expiresAt := now.Add(time.Duration(timeoutSeconds) * time.Second)
	if err := o.persistApproval(ctx, approvalID, toolName, args, meta, matched, expiresAt, timeoutSeconds); err != nil {
		return nil, err
	}

	return &common.ApprovalDecision{
		RequiresApproval: true,
		ApprovalID:       approvalID,
		MatchedRuleID:    &matched.ID,
		PolicyVersion:    matched.PolicyVersion,
		DecisionSource:   "db_policy",
		TimeoutSeconds:   timeoutSeconds,
		ExpiresAt:        expiresAt,
	}, nil
}

func (o *ApprovalOrchestrator) listPolicies(ctx context.Context, toolName string) ([]ai.AIToolRiskPolicy, error) {
	if o == nil || o.policyStore == nil {
		return nil, nil
	}
	return o.policyStore.ListEnabledByToolName(ctx, toolName)
}

func (o *ApprovalOrchestrator) createFallbackDecision(ctx context.Context, toolName, args string, meta common.ApprovalEvalMeta, now time.Time, timeoutSeconds int) (*common.ApprovalDecision, error) {
	if !fallbackRequiresApproval(toolName, meta.CommandClass) {
		return &common.ApprovalDecision{
			RequiresApproval: false,
			DecisionSource:   "fallback_safe",
			TimeoutSeconds:   timeoutSeconds,
		}, nil
	}
	approvalID := o.nextID()
	expiresAt := now.Add(time.Duration(timeoutSeconds) * time.Second)
	if err := o.persistFallbackApproval(ctx, approvalID, toolName, args, meta, expiresAt, timeoutSeconds); err != nil {
		return nil, err
	}
	return &common.ApprovalDecision{
		RequiresApproval: true,
		ApprovalID:       approvalID,
		DecisionSource:   "fallback_safe",
		TimeoutSeconds:   timeoutSeconds,
		ExpiresAt:        expiresAt,
	}, nil
}

func (o *ApprovalOrchestrator) persistApproval(ctx context.Context, approvalID, toolName, args string, meta common.ApprovalEvalMeta, matched *ai.AIToolRiskPolicy, expiresAt time.Time, timeoutSeconds int) error {
	task := &ai.AIApprovalTask{
		ApprovalID:     approvalID,
		CheckpointID:   meta.CheckpointID,
		SessionID:      meta.SessionID,
		RunID:          meta.RunID,
		UserID:         meta.UserID,
		ToolName:       toolName,
		ToolCallID:     meta.CallID,
		ArgumentsJSON:  args,
		Status:         "pending",
		TimeoutSeconds: timeoutSeconds,
		ExpiresAt:      &expiresAt,
	}
	if matched != nil {
		task.MatchedRuleID = &matched.ID
		if matched.PolicyVersion != "" {
			pv := matched.PolicyVersion
			task.PolicyVersion = &pv
		}
		ds := "db_policy"
		task.DecisionSource = &ds
	}

	payload, err := json.Marshal(map[string]any{
		"approval_id": approvalID,
		"tool_name":   toolName,
		"tool_call_id": meta.CallID,
		"run_id":      meta.RunID,
		"session_id":  meta.SessionID,
	})
	if err != nil {
		return err
	}
	outbox := &ai.AIApprovalOutboxEvent{
		ApprovalID:  approvalID,
		ToolCallID:  meta.CallID,
		EventType:   "approval_requested",
		RunID:       meta.RunID,
		SessionID:   meta.SessionID,
		PayloadJSON: string(payload),
		Status:      "pending",
	}
	return o.persistTaskAndOutbox(ctx, task, outbox)
}

func (o *ApprovalOrchestrator) persistFallbackApproval(ctx context.Context, approvalID, toolName, args string, meta common.ApprovalEvalMeta, expiresAt time.Time, timeoutSeconds int) error {
	ds := "fallback_safe"
	task := &ai.AIApprovalTask{
		ApprovalID:     approvalID,
		CheckpointID:   meta.CheckpointID,
		SessionID:      meta.SessionID,
		RunID:          meta.RunID,
		UserID:         meta.UserID,
		ToolName:       toolName,
		ToolCallID:     meta.CallID,
		ArgumentsJSON:  args,
		Status:         "pending",
		TimeoutSeconds: timeoutSeconds,
		ExpiresAt:      &expiresAt,
		DecisionSource: &ds,
	}
	payload, err := json.Marshal(map[string]any{
		"approval_id": approvalID,
		"tool_name":   toolName,
		"tool_call_id": meta.CallID,
		"run_id":      meta.RunID,
		"session_id":  meta.SessionID,
	})
	if err != nil {
		return err
	}
	outbox := &ai.AIApprovalOutboxEvent{
		ApprovalID:  approvalID,
		ToolCallID:  meta.CallID,
		EventType:   "approval_requested",
		RunID:       meta.RunID,
		SessionID:   meta.SessionID,
		PayloadJSON: string(payload),
		Status:      "pending",
	}
	return o.persistTaskAndOutbox(ctx, task, outbox)
}

func (o *ApprovalOrchestrator) persistTaskAndOutbox(ctx context.Context, task *ai.AIApprovalTask, outbox *ai.AIApprovalOutboxEvent) error {
	if o.db != nil {
		return o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			taskDAO := aidao.NewAIApprovalTaskDAO(tx)
			outboxDAO := aidao.NewAIApprovalOutboxDAO(tx)
			if err := taskDAO.Create(ctx, task); err != nil {
				return err
			}
			return outboxDAO.EnqueueOrTouch(ctx, outbox)
		})
	}
	if o.approvalDAO != nil {
		if err := o.approvalDAO.Create(ctx, task); err != nil {
			return err
		}
	}
	if o.outboxDAO != nil {
		if err := o.outboxDAO.EnqueueOrTouch(ctx, outbox); err != nil {
			return err
		}
	}
	return nil
}

func (o *ApprovalOrchestrator) nextID() string {
	if o != nil && o.newID != nil {
		return o.newID()
	}
	return uuid.NewString()
}

func (o *ApprovalOrchestrator) nowOrDefault() time.Time {
	if o != nil && o.now != nil {
		return o.now().UTC()
	}
	return time.Now().UTC()
}

func fallbackRequiresApproval(toolName, commandClass string) bool {
	if strings.EqualFold(strings.TrimSpace(commandClass), "readonly") {
		return false
	}
	if strings.Contains(strings.ToLower(toolName), "preview") && strings.EqualFold(strings.TrimSpace(commandClass), "readonly") {
		return false
	}
	return true
}

func MatchRiskPolicyOrFallback(rules []ai.AIToolRiskPolicy, scene, commandClass string, args map[string]any) (*ai.AIToolRiskPolicy, bool) {
	return MatchRiskPolicy(rules, scene, commandClass, args)
}

func (o *ApprovalOrchestrator) EvaluateTool(ctx context.Context, toolName string, args string, meta common.ApprovalEvalMeta) (*common.ApprovalDecision, error) {
	return o.Evaluate(ctx, toolName, args, meta)
}

func (o *ApprovalOrchestrator) String() string {
	return fmt.Sprintf("ApprovalOrchestrator{db:%v}", o.db != nil)
}
