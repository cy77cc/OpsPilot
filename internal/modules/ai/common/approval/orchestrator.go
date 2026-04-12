package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ApprovalOrchestrator evaluates DB policies and persists approval snapshots when approval is required.
type ApprovalOrchestrator struct {
	policyStore  approvalPolicyStore
	approvalDAO  approvalTaskWriter
	outboxDAO    approvalOutboxWriter
	riskRegistry *RiskRegistry
	db           *gorm.DB
	now          func() time.Time
	newID        func() string
	defaultTTL   int
}

type approvalPolicyStore interface {
	ListEnabledByToolName(ctx context.Context, toolName string) ([]ai.AIToolRiskPolicy, error)
}

type approvalTaskWriter interface {
	Create(ctx context.Context, task *approvalTaskRecord) error
}

type approvalOutboxWriter interface {
	EnqueueOrTouch(ctx context.Context, event *approvalOutboxEventRecord) error
}

// NewApprovalOrchestrator creates a DB-backed approval orchestrator.
func NewApprovalOrchestrator(db *gorm.DB) *ApprovalOrchestrator {
	if db == nil {
		return &ApprovalOrchestrator{
			riskRegistry: DefaultRiskRegistry(),
			now:          time.Now,
			newID:        uuid.NewString,
			defaultTTL:   DefaultApprovalTimeout,
		}
	}
	return NewApprovalOrchestratorWithStores(
		approvalPolicyDAO{db: db},
		approvalTaskDAO{db: db},
		approvalOutboxDAO{db: db},
	).withDB(db)
}

type approvalPolicyDAO struct {
	db *gorm.DB
}

func (d approvalPolicyDAO) ListEnabledByToolName(ctx context.Context, toolName string) ([]ai.AIToolRiskPolicy, error) {
	var policies []ai.AIToolRiskPolicy
	err := d.db.WithContext(ctx).
		Where("tool_name = ? AND enabled = ?", toolName, true).
		Order("priority DESC, id ASC").
		Find(&policies).Error
	return policies, err
}

type approvalTaskDAO struct {
	db *gorm.DB
}

func (d approvalTaskDAO) Create(ctx context.Context, task *approvalTaskRecord) error {
	return d.db.WithContext(ctx).Create(task).Error
}

type approvalOutboxDAO struct {
	db *gorm.DB
}

func (d approvalOutboxDAO) EnqueueOrTouch(ctx context.Context, event *approvalOutboxEventRecord) error {
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

// NewApprovalOrchestratorWithStores wires custom stores for tests or alternative integrations.
func NewApprovalOrchestratorWithStores(policy approvalPolicyStore, approval approvalTaskWriter, outbox approvalOutboxWriter) *ApprovalOrchestrator {
	return &ApprovalOrchestrator{
		policyStore:  policy,
		approvalDAO:  approval,
		outboxDAO:    outbox,
		riskRegistry: DefaultRiskRegistry(),
		now:          time.Now,
		newID:        uuid.NewString,
		defaultTTL:   DefaultApprovalTimeout,
	}
}

type approvalTaskRecord struct {
	ID             uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	ApprovalID     string     `gorm:"column:approval_id"`
	CheckpointID   string     `gorm:"column:checkpoint_id"`
	SessionID      string     `gorm:"column:session_id"`
	RunID          string     `gorm:"column:run_id"`
	UserID         uint64     `gorm:"column:user_id"`
	ToolName       string     `gorm:"column:tool_name"`
	ToolCallID     string     `gorm:"column:tool_call_id"`
	ArgumentsJSON  string     `gorm:"column:arguments_json"`
	PreviewJSON    string     `gorm:"column:preview_json"`
	Status         string     `gorm:"column:status"`
	TimeoutSeconds int        `gorm:"column:timeout_seconds"`
	ExpiresAt      *time.Time `gorm:"column:expires_at"`
	MatchedRuleID  *uint64    `gorm:"column:matched_rule_id"`
	PolicyVersion  *string    `gorm:"column:policy_version"`
	DecisionSource *string    `gorm:"column:decision_source"`
}

func (approvalTaskRecord) TableName() string { return "ai_approval_tasks" }

type approvalOutboxEventRecord struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	EventID     string     `gorm:"column:event_id"`
	Sequence    int64      `gorm:"column:sequence"`
	AggregateID string     `gorm:"column:aggregate_id"`
	OccurredAt  time.Time  `gorm:"column:occurred_at"`
	Version     int        `gorm:"column:version"`
	ApprovalID  string     `gorm:"column:approval_id"`
	ToolCallID  string     `gorm:"column:tool_call_id"`
	EventType   string     `gorm:"column:event_type"`
	RunID       string     `gorm:"column:run_id"`
	SessionID   string     `gorm:"column:session_id"`
	PayloadJSON string     `gorm:"column:payload_json"`
	Status      string     `gorm:"column:status"`
	RetryCount  int        `gorm:"column:retry_count"`
	NextRetryAt *time.Time `gorm:"column:next_retry_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
}

func (approvalOutboxEventRecord) TableName() string { return "ai_approval_outbox_events" }

func (e *approvalOutboxEventRecord) BeforeCreate(tx *gorm.DB) error {
	if e == nil {
		return nil
	}
	if e.EventID == "" {
		e.EventID = uuid.NewString()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	if e.Version <= 0 {
		e.Version = 1
	}
	if e.AggregateID == "" {
		e.AggregateID = e.RunID
	}
	if e.Sequence <= 0 && tx != nil && e.RunID != "" {
		var sequence int64
		if err := tx.Raw("SELECT COALESCE(MAX(sequence), 0) + 1 FROM ai_approval_outbox_events WHERE run_id = ?", e.RunID).Scan(&sequence).Error; err != nil {
			return err
		}
		e.Sequence = sequence
	}
	return nil
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
	if !o.requiresApprovalByFallbackPolicy(toolName, meta.CommandClass) {
		return &ApprovalDecision{
			RequiresApproval: false,
			TimeoutSeconds:   timeoutSeconds,
			DecisionSource:   "fallback_safe",
		}, nil
	}
	return o.createApprovalDecision(ctx, toolName, args, meta, now, timeoutSeconds, nil, "fallback_safe")
}

func (o *ApprovalOrchestrator) requiresApprovalByFallbackPolicy(toolName, commandClass string) bool {
	if o != nil && o.riskRegistry != nil {
		return o.riskRegistry.RequiresApproval(toolName, commandClass)
	}
	return fallbackRequiresApproval(toolName, commandClass)
}

func (o *ApprovalOrchestrator) createApprovalDecision(
	ctx context.Context,
	toolName, args string,
	meta ApprovalEvalMeta,
	now time.Time,
	timeoutSeconds int,
	matched *ai.AIToolRiskPolicy,
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

	task := &approvalTaskRecord{
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
		"approval_id":        approvalID,
		"call_id":            meta.CallID,
		"tool_name":          toolName,
		"session_id":         meta.SessionID,
		"run_id":             meta.RunID,
		"preview":            preview,
		"timeout_seconds":    timeoutSeconds,
		"expires_at":         expiresAt.UTC().Format(time.RFC3339Nano),
		"decision_source":    decisionSource,
		"approver_id":        "",
		"approval_timestamp": "",
		"reject_reason":      "",
	}
	if matched != nil {
		eventPayload["matched_rule_id"] = matched.ID
		eventPayload["policy_version"] = strings.TrimSpace(matched.PolicyVersion)
	}
	event := &approvalOutboxEventRecord{
		ApprovalID:  approvalID,
		EventType:   "approval_requested",
		RunID:       meta.RunID,
		SessionID:   meta.SessionID,
		PayloadJSON: mustJSONString(eventPayload),
		Status:      "pending",
	}

	if o.db != nil {
		err := o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			txApproval := approvalTaskDAO{db: tx}
			txOutbox := approvalOutboxDAO{db: tx}
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

func (o *ApprovalOrchestrator) listPolicies(ctx context.Context, toolName string) ([]ai.AIToolRiskPolicy, error) {
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

func buildApprovalPreview(toolName, args string, matched *ai.AIToolRiskPolicy) ApprovalPreview {
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
	return DefaultRiskRegistry().RequiresApproval(toolName, commandClass)
}
