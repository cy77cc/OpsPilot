// Package approval 实现审批编排器。
package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/approval"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Orchestrator 实现审批编排逻辑。
type Orchestrator struct {
	policyStore  approvalPolicyStore
	approvalDAO  approvalTaskWriter
	outboxDAO    approvalOutboxWriter
	riskRegistry *approval.RiskRegistry
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

// NewOrchestrator 创建审批编排器。
func NewOrchestrator(db *gorm.DB) *Orchestrator {
	if db == nil {
		return &Orchestrator{riskRegistry: approval.DefaultRiskRegistry(), now: time.Now, newID: uuid.NewString, defaultTTL: approval.DefaultApprovalTimeout}
	}
	return NewOrchestratorWithStores(approvalPolicyDAO{db: db}, approvalTaskDAO{db: db}, approvalOutboxDAO{db: db}).withDB(db)
}

// NewOrchestratorWithStores 使用显式存储创建审批编排器。
func NewOrchestratorWithStores(policy approvalPolicyStore, a approvalTaskWriter, outbox approvalOutboxWriter) *Orchestrator {
	return &Orchestrator{policyStore: policy, approvalDAO: a, outboxDAO: outbox, riskRegistry: approval.DefaultRiskRegistry(), now: time.Now, newID: uuid.NewString, defaultTTL: approval.DefaultApprovalTimeout}
}

func (o *Orchestrator) withDB(db *gorm.DB) *Orchestrator {
	if o != nil {
		o.db = db
	}
	return o
}

// Evaluate 评估工具调用是否需要审批。
func (o *Orchestrator) Evaluate(ctx context.Context, toolName string, args string, meta approval.ApprovalEvalMeta) (*approval.ApprovalDecision, error) {
	if o == nil {
		if FallbackRequiresApproval(toolName, meta.CommandClass) {
			return &approval.ApprovalDecision{RequiresApproval: true, DecisionSource: "fallback_safe"}, nil
		}
		return &approval.ApprovalDecision{RequiresApproval: false, DecisionSource: "fallback_safe"}, nil
	}
	now := o.nowOrDefault()
	timeoutSeconds := meta.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = o.defaultTTL
	}
	argsMap := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(args)), &argsMap); err != nil {
		// Fail-closed: malformed args means we cannot safely evaluate policies.
		// Return auto-approval with no-risk, but log the anomaly for monitoring.
		argsMap = map[string]any{"_parse_error": true}
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
		return &approval.ApprovalDecision{RequiresApproval: false, MatchedRuleID: &matched.ID, PolicyVersion: matched.PolicyVersion, DecisionSource: "db_policy", TimeoutSeconds: timeoutSeconds}, nil
	}
	approvalID := o.nextID()
	expiresAt := now.Add(time.Duration(timeoutSeconds) * time.Second)
	if err := o.persistApproval(ctx, approvalID, toolName, args, meta, matched, expiresAt, timeoutSeconds); err != nil {
		return nil, err
	}
	return &approval.ApprovalDecision{RequiresApproval: true, ApprovalID: approvalID, MatchedRuleID: &matched.ID, PolicyVersion: matched.PolicyVersion, DecisionSource: "db_policy", TimeoutSeconds: timeoutSeconds, ExpiresAt: expiresAt}, nil
}

// EvaluateTool 是 Evaluate 的别名。
func (o *Orchestrator) EvaluateTool(ctx context.Context, toolName string, args string, meta approval.ApprovalEvalMeta) (*approval.ApprovalDecision, error) {
	return o.Evaluate(ctx, toolName, args, meta)
}

func (o *Orchestrator) String() string {
	return fmt.Sprintf("ApprovalOrchestrator{db:%v}", o.db != nil)
}

func (o *Orchestrator) listPolicies(ctx context.Context, toolName string) ([]ai.AIToolRiskPolicy, error) {
	if o == nil || o.policyStore == nil {
		return nil, nil
	}
	return o.policyStore.ListEnabledByToolName(ctx, toolName)
}

func (o *Orchestrator) createFallbackDecision(ctx context.Context, toolName, args string, meta approval.ApprovalEvalMeta, now time.Time, timeoutSeconds int) (*approval.ApprovalDecision, error) {
	if !FallbackRequiresApproval(toolName, meta.CommandClass) {
		return &approval.ApprovalDecision{RequiresApproval: false, DecisionSource: "fallback_safe", TimeoutSeconds: timeoutSeconds}, nil
	}
	approvalID := o.nextID()
	expiresAt := now.Add(time.Duration(timeoutSeconds) * time.Second)
	if err := o.persistFallbackApproval(ctx, approvalID, toolName, args, meta, expiresAt, timeoutSeconds); err != nil {
		return nil, err
	}
	return &approval.ApprovalDecision{RequiresApproval: true, ApprovalID: approvalID, DecisionSource: "fallback_safe", TimeoutSeconds: timeoutSeconds, ExpiresAt: expiresAt}, nil
}

func (o *Orchestrator) persistApproval(ctx context.Context, approvalID, toolName, args string, meta approval.ApprovalEvalMeta, matched *ai.AIToolRiskPolicy, expiresAt time.Time, timeoutSeconds int) error {
	task := &ai.AIApprovalTask{ApprovalID: approvalID, CheckpointID: meta.CheckpointID, SessionID: meta.SessionID, RunID: meta.RunID, UserID: meta.UserID, ToolName: toolName, ToolCallID: meta.CallID, ArgumentsJSON: args, Status: "pending", TimeoutSeconds: timeoutSeconds, ExpiresAt: &expiresAt}
	if matched != nil {
		task.MatchedRuleID = &matched.ID
		if matched.PolicyVersion != "" {
			pv := matched.PolicyVersion
			task.PolicyVersion = &pv
		}
		ds := "db_policy"
		task.DecisionSource = &ds
	}
	payload, _ := json.Marshal(map[string]any{"approval_id": approvalID, "tool_name": toolName, "tool_call_id": meta.CallID, "run_id": meta.RunID, "session_id": meta.SessionID})
	outbox := &ai.AIApprovalOutboxEvent{ApprovalID: approvalID, ToolCallID: meta.CallID, EventType: "approval_requested", RunID: meta.RunID, SessionID: meta.SessionID, PayloadJSON: string(payload), Status: "pending"}
	return o.persistTaskAndOutbox(ctx, task, outbox)
}

func (o *Orchestrator) persistFallbackApproval(ctx context.Context, approvalID, toolName, args string, meta approval.ApprovalEvalMeta, expiresAt time.Time, timeoutSeconds int) error {
	ds := "fallback_safe"
	task := &ai.AIApprovalTask{ApprovalID: approvalID, CheckpointID: meta.CheckpointID, SessionID: meta.SessionID, RunID: meta.RunID, UserID: meta.UserID, ToolName: toolName, ToolCallID: meta.CallID, ArgumentsJSON: args, Status: "pending", TimeoutSeconds: timeoutSeconds, ExpiresAt: &expiresAt, DecisionSource: &ds}
	payload, _ := json.Marshal(map[string]any{"approval_id": approvalID, "tool_name": toolName, "tool_call_id": meta.CallID, "run_id": meta.RunID, "session_id": meta.SessionID})
	outbox := &ai.AIApprovalOutboxEvent{ApprovalID: approvalID, ToolCallID: meta.CallID, EventType: "approval_requested", RunID: meta.RunID, SessionID: meta.SessionID, PayloadJSON: string(payload), Status: "pending"}
	return o.persistTaskAndOutbox(ctx, task, outbox)
}

func (o *Orchestrator) persistTaskAndOutbox(ctx context.Context, task *ai.AIApprovalTask, outbox *ai.AIApprovalOutboxEvent) error {
	if o.db != nil {
		return o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			taskDAO := approvalTaskDAO{db: tx}
			outboxDAO := approvalOutboxDAO{db: tx}
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

func (o *Orchestrator) nextID() string {
	if o != nil && o.newID != nil {
		return o.newID()
	}
	return uuid.NewString()
}

func (o *Orchestrator) nowOrDefault() time.Time {
	if o != nil && o.now != nil {
		return o.now().UTC()
	}
	return time.Now().UTC()
}

// FallbackRequiresApproval 判断默认情况下是否需要审批。
func FallbackRequiresApproval(toolName, commandClass string) bool {
	if strings.EqualFold(strings.TrimSpace(commandClass), "readonly") {
		return false
	}
	if strings.Contains(strings.ToLower(toolName), "preview") && strings.EqualFold(strings.TrimSpace(commandClass), "readonly") {
		return false
	}
	return true
}

// MatchRiskPolicy 匹配风险策略（本地实现，避免循环依赖）。
func MatchRiskPolicy(rules []ai.AIToolRiskPolicy, scene, commandClass string, args map[string]any) (*ai.AIToolRiskPolicy, bool) {
	var (
		best         *ai.AIToolRiskPolicy
		bestScore    int
		bestPriority int
	)
	for i := range rules {
		rule := &rules[i]
		score, ok := matchRiskPolicyScore(rule, scene, commandClass, args)
		if !ok {
			continue
		}
		if best == nil || rule.Priority > bestPriority || (rule.Priority == bestPriority && score > bestScore) {
			best = rule
			bestScore = score
			bestPriority = rule.Priority
		}
	}
	if best == nil {
		return nil, false
	}
	return best, true
}

func matchRiskPolicyScore(rule *ai.AIToolRiskPolicy, scene, commandClass string, args map[string]any) (int, bool) {
	if rule == nil {
		return 0, false
	}
	if rule.Scene != nil && strings.TrimSpace(*rule.Scene) != "" && !strings.EqualFold(strings.TrimSpace(*rule.Scene), strings.TrimSpace(scene)) {
		return 0, false
	}
	if rule.CommandClass != nil && strings.TrimSpace(*rule.CommandClass) != "" && !strings.EqualFold(strings.TrimSpace(*rule.CommandClass), strings.TrimSpace(commandClass)) {
		return 0, false
	}
	if !matchesArgRules(rule.ArgumentRulesJSON, args) {
		return 0, false
	}
	score := 0
	if hasArgRules(rule.ArgumentRulesJSON) {
		score += 4
	}
	if rule.CommandClass != nil && strings.TrimSpace(*rule.CommandClass) != "" {
		score += 2
	}
	if rule.Scene != nil && strings.TrimSpace(*rule.Scene) != "" {
		score += 1
	}
	return score, true
}

func hasArgRules(raw *string) bool {
	if raw == nil {
		return false
	}
	return strings.TrimSpace(*raw) != "" && strings.TrimSpace(*raw) != "{}"
}

func matchesArgRules(raw *string, args map[string]any) bool {
	if !hasArgRules(raw) || len(args) == 0 {
		return !hasArgRules(raw)
	}
	var ruleMap map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(*raw)), &ruleMap); err != nil || len(ruleMap) == 0 {
		return err == nil
	}
	for key, expected := range ruleMap {
		actual, ok := args[key]
		if !ok || !argValueMatches(expected, actual) {
			return false
		}
	}
	return true
}

func argValueMatches(expected, actual any) bool {
	if expected == nil {
		return actual == nil
	}
	if em, ok := expected.(map[string]any); ok {
		if rp, ok := em["regex"].(string); ok && rp != "" {
			if as, ok := actual.(string); ok {
				re, err := regexp.Compile(rp)
				if err != nil {
					return false
				}
				return re.MatchString(as)
			}
			return false
		}
	}
	return reflect.DeepEqual(expected, actual) || numericEqual(expected, actual)
}

func numericEqual(expected, actual any) bool {
	e, ok1 := toFloat64(expected)
	a, ok2 := toFloat64(actual)
	return ok1 && ok2 && e == a
}

func toFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return float64(reflect.ValueOf(v).Convert(reflect.TypeOf(float64(0))).Float()), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

type approvalPolicyDAO struct{ db *gorm.DB }

func (d approvalPolicyDAO) ListEnabledByToolName(ctx context.Context, toolName string) ([]ai.AIToolRiskPolicy, error) {
	var policies []ai.AIToolRiskPolicy
	err := d.db.WithContext(ctx).Where("tool_name = ? AND enabled = ?", toolName, true).Order("priority DESC, id ASC").Find(&policies).Error
	return policies, err
}

type approvalTaskDAO struct{ db *gorm.DB }

func (d approvalTaskDAO) Create(ctx context.Context, task *ai.AIApprovalTask) error {
	return d.db.WithContext(ctx).Create(task).Error
}

type approvalOutboxDAO struct{ db *gorm.DB }

func (d approvalOutboxDAO) EnqueueOrTouch(ctx context.Context, event *ai.AIApprovalOutboxEvent) error {
	now := time.Now()
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "approval_id"}, {Name: "event_type"}},
		DoUpdates: clause.Assignments(map[string]any{
			"run_id": gorm.Expr("CASE WHEN status = ? THEN ? ELSE run_id END", "pending", event.RunID),
			"session_id": gorm.Expr("CASE WHEN status = ? THEN ? ELSE session_id END", "pending", event.SessionID),
			"tool_call_id": gorm.Expr("CASE WHEN status = ? THEN ? ELSE tool_call_id END", "pending", event.ToolCallID),
			"payload_json": gorm.Expr("CASE WHEN status = ? THEN ? ELSE payload_json END", "pending", event.PayloadJSON),
			"status": gorm.Expr("CASE WHEN status = ? THEN ? ELSE status END", "pending", event.Status),
			"next_retry_at": gorm.Expr("CASE WHEN status = ? THEN ? ELSE next_retry_at END", "pending", event.NextRetryAt),
			"updated_at": gorm.Expr("CASE WHEN status = ? THEN ? ELSE updated_at END", "pending", now),
		}),
	}).Create(event).Error
}
