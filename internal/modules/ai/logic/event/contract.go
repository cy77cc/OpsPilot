// Package event keeps compatibility shims for approval event contracts.
//
// Canonical approval event ownership now lives under:
// internal/modules/ai/agent/shared/approval.
package event

import (
	aiapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/approval"
)

const (
	ApprovalEventTypeRequested = aiapproval.ApprovalEventTypeRequested
	ApprovalEventTypeDecided   = aiapproval.ApprovalEventTypeDecided
	ApprovalEventTypeExpired   = aiapproval.ApprovalEventTypeExpired
	RunEventTypeResuming       = aiapproval.RunEventTypeResuming
	RunEventTypeResumed        = aiapproval.RunEventTypeResumed
	RunEventTypeResumeFailed   = aiapproval.RunEventTypeResumeFailed
	RunEventTypeCompleted      = aiapproval.RunEventTypeCompleted
)

type ApprovalEventEnvelope = aiapproval.ApprovalEventEnvelope

type ApprovalRequestedInput = aiapproval.ApprovalRequestedInput
type ApprovalDecidedInput = aiapproval.ApprovalDecidedInput
type ApprovalExpiredInput = aiapproval.ApprovalExpiredInput
type RunResumingInput = aiapproval.RunResumingInput
type RunResumedInput = aiapproval.RunResumedInput
type RunResumeFailedInput = aiapproval.RunResumeFailedInput
type RunCompletedInput = aiapproval.RunCompletedInput

func NewApprovalRequestedEnvelope(input ApprovalRequestedInput) (*ApprovalEventEnvelope, error) {
	return aiapproval.NewApprovalRequestedEnvelope(input)
}

func NewApprovalDecidedEnvelope(input ApprovalDecidedInput) (*ApprovalEventEnvelope, error) {
	return aiapproval.NewApprovalDecidedEnvelope(input)
}

func NewApprovalExpiredEnvelope(input ApprovalExpiredInput) (*ApprovalEventEnvelope, error) {
	return aiapproval.NewApprovalExpiredEnvelope(input)
}

func NewRunResumingEnvelope(input RunResumingInput) (*ApprovalEventEnvelope, error) {
	return aiapproval.NewRunResumingEnvelope(input)
}

func NewRunResumedEnvelope(input RunResumedInput) (*ApprovalEventEnvelope, error) {
	return aiapproval.NewRunResumedEnvelope(input)
}

func NewRunResumeFailedEnvelope(input RunResumeFailedInput) (*ApprovalEventEnvelope, error) {
	return aiapproval.NewRunResumeFailedEnvelope(input)
}

func NewRunCompletedEnvelope(input RunCompletedInput) (*ApprovalEventEnvelope, error) {
	return aiapproval.NewRunCompletedEnvelope(input)
}
