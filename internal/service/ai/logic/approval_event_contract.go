// Package logic 实现 AI 模块的业务逻辑层。
//
// 本文件定义审批事件的类型和信封结构，用于事件溯源和 Outbox 模式。
package logic

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// 审批事件类型常量。
const (
	// ApprovalEventTypeRequested 审批请求事件
	ApprovalEventTypeRequested = "ai.approval.requested"
	// ApprovalEventTypeDecided 审批决定事件
	ApprovalEventTypeDecided = "ai.approval.decided"
	// ApprovalEventTypeExpired 审批过期事件
	ApprovalEventTypeExpired = "ai.approval.expired"
	// RunEventTypeResuming 运行恢复中事件
	RunEventTypeResuming = "ai.run.resuming"
	// RunEventTypeResumed 运行已恢复事件
	RunEventTypeResumed = "ai.run.resumed"
	// RunEventTypeResumeFailed 恢复失败事件
	RunEventTypeResumeFailed = "ai.run.resume_failed"
	// RunEventTypeCompleted 运行完成事件
	RunEventTypeCompleted = "ai.run.completed"
)

// ApprovalEventEnvelope 审批事件信封。
//
// 包含事件的所有元数据和负载，用于事件存储和传输。
type ApprovalEventEnvelope struct {
	EventID     string    `json:"event_id"`
	EventType   string    `json:"event_type"`
	OccurredAt  time.Time `json:"occurred_at"`
	Sequence    int64     `json:"sequence"`
	Version     int       `json:"version"`
	RunID       string    `json:"run_id"`
	SessionID   string    `json:"session_id"`
	ApprovalID  string    `json:"approval_id"`
	ToolCallID  string    `json:"tool_call_id"`
	AggregateID string    `json:"aggregate_id"`
	PayloadJSON string    `json:"payload_json"`
}

type ApprovalRequestedInput struct {
	EventID     string
	OccurredAt   time.Time
	Sequence    int64
	Version     int
	RunID       string
	SessionID   string
	ApprovalID  string
	ToolCallID  string
	AggregateID string
	Payload     any
}

type ApprovalDecidedInput struct {
	EventID     string
	OccurredAt   time.Time
	Sequence    int64
	Version     int
	RunID       string
	SessionID   string
	ApprovalID  string
	ToolCallID  string
	AggregateID string
	Payload     any
}

type ApprovalExpiredInput struct {
	EventID     string
	OccurredAt   time.Time
	Sequence    int64
	Version     int
	RunID       string
	SessionID   string
	ApprovalID  string
	ToolCallID  string
	AggregateID string
	Payload     any
}

type RunResumingInput struct {
	EventID     string
	OccurredAt   time.Time
	Sequence    int64
	Version     int
	RunID       string
	SessionID   string
	ApprovalID  string
	ToolCallID  string
	AggregateID string
	Payload     any
}

type RunResumedInput struct {
	EventID     string
	OccurredAt   time.Time
	Sequence    int64
	Version     int
	RunID       string
	SessionID   string
	ApprovalID  string
	ToolCallID  string
	AggregateID string
	Payload     any
}

type RunResumeFailedInput struct {
	EventID     string
	OccurredAt   time.Time
	Sequence    int64
	Version     int
	RunID       string
	SessionID   string
	ApprovalID  string
	ToolCallID  string
	AggregateID string
	Payload     any
}

type RunCompletedInput struct {
	EventID     string
	OccurredAt   time.Time
	Sequence    int64
	Version     int
	RunID       string
	SessionID   string
	ApprovalID  string
	ToolCallID  string
	AggregateID string
	Payload     any
}

func NewApprovalRequestedEnvelope(input ApprovalRequestedInput) (*ApprovalEventEnvelope, error) {
	return newApprovalEventEnvelope(ApprovalEventTypeRequested, input.EventID, input.OccurredAt, input.Sequence, input.Version, input.RunID, input.SessionID, input.ApprovalID, input.ToolCallID, input.AggregateID, input.Payload)
}

func NewApprovalDecidedEnvelope(input ApprovalDecidedInput) (*ApprovalEventEnvelope, error) {
	return newApprovalEventEnvelope(ApprovalEventTypeDecided, input.EventID, input.OccurredAt, input.Sequence, input.Version, input.RunID, input.SessionID, input.ApprovalID, input.ToolCallID, input.AggregateID, input.Payload)
}

func NewApprovalExpiredEnvelope(input ApprovalExpiredInput) (*ApprovalEventEnvelope, error) {
	return newApprovalEventEnvelope(ApprovalEventTypeExpired, input.EventID, input.OccurredAt, input.Sequence, input.Version, input.RunID, input.SessionID, input.ApprovalID, input.ToolCallID, input.AggregateID, input.Payload)
}

func NewRunResumingEnvelope(input RunResumingInput) (*ApprovalEventEnvelope, error) {
	return newApprovalEventEnvelope(RunEventTypeResuming, input.EventID, input.OccurredAt, input.Sequence, input.Version, input.RunID, input.SessionID, input.ApprovalID, input.ToolCallID, input.AggregateID, input.Payload)
}

func NewRunResumedEnvelope(input RunResumedInput) (*ApprovalEventEnvelope, error) {
	return newApprovalEventEnvelope(RunEventTypeResumed, input.EventID, input.OccurredAt, input.Sequence, input.Version, input.RunID, input.SessionID, input.ApprovalID, input.ToolCallID, input.AggregateID, input.Payload)
}

func NewRunResumeFailedEnvelope(input RunResumeFailedInput) (*ApprovalEventEnvelope, error) {
	return newApprovalEventEnvelope(RunEventTypeResumeFailed, input.EventID, input.OccurredAt, input.Sequence, input.Version, input.RunID, input.SessionID, input.ApprovalID, input.ToolCallID, input.AggregateID, input.Payload)
}

func NewRunCompletedEnvelope(input RunCompletedInput) (*ApprovalEventEnvelope, error) {
	return newApprovalEventEnvelope(RunEventTypeCompleted, input.EventID, input.OccurredAt, input.Sequence, input.Version, input.RunID, input.SessionID, input.ApprovalID, input.ToolCallID, input.AggregateID, input.Payload)
}

func newApprovalEventEnvelope(eventType, eventID string, occurredAt time.Time, sequence int64, version int, runID, sessionID, approvalID, toolCallID, aggregateID string, payload any) (*ApprovalEventEnvelope, error) {
	if eventType == "" {
		return nil, fmt.Errorf("event_type is required")
	}
	if sequence <= 0 {
		return nil, fmt.Errorf("sequence is required")
	}
	if version <= 0 {
		version = 1
	}
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if approvalID == "" {
		return nil, fmt.Errorf("approval_id is required")
	}
	if toolCallID == "" {
		return nil, fmt.Errorf("tool_call_id is required")
	}
	if aggregateID == "" {
		aggregateID = runID
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	} else {
		occurredAt = occurredAt.UTC()
	}
	if eventID == "" {
		eventID = uuid.NewString()
	}
	payloadJSON, err := marshalApprovalEventPayload(payload)
	if err != nil {
		return nil, err
	}
	if payloadJSON == "" {
		payloadJSON = "{}"
	}
	return &ApprovalEventEnvelope{
		EventID:     eventID,
		EventType:   eventType,
		OccurredAt:  occurredAt,
		Sequence:    sequence,
		Version:     version,
		RunID:       runID,
		SessionID:   sessionID,
		ApprovalID:  approvalID,
		ToolCallID:  toolCallID,
		AggregateID: aggregateID,
		PayloadJSON: payloadJSON,
	}, nil
}

func marshalApprovalEventPayload(payload any) (string, error) {
	if payload == nil {
		return "", fmt.Errorf("payload is required")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
