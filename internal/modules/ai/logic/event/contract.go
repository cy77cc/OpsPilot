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

type ApprovalEventInput = aiapproval.ApprovalEventInput

func NewApprovalRequestedEnvelope(input ApprovalEventInput) (*ApprovalEventEnvelope, error) {
	return aiapproval.NewApprovalRequestedEnvelope(input)
}

func NewApprovalDecidedEnvelope(input ApprovalEventInput) (*ApprovalEventEnvelope, error) {
	return aiapproval.NewApprovalDecidedEnvelope(input)
}

func NewApprovalExpiredEnvelope(input ApprovalEventInput) (*ApprovalEventEnvelope, error) {
	return aiapproval.NewApprovalExpiredEnvelope(input)
}

func NewRunResumingEnvelope(input ApprovalEventInput) (*ApprovalEventEnvelope, error) {
	return aiapproval.NewRunResumingEnvelope(input)
}

func NewRunResumedEnvelope(input ApprovalEventInput) (*ApprovalEventEnvelope, error) {
	return aiapproval.NewRunResumedEnvelope(input)
}

func NewRunResumeFailedEnvelope(input ApprovalEventInput) (*ApprovalEventEnvelope, error) {
	return aiapproval.NewRunResumeFailedEnvelope(input)
}

func NewRunCompletedEnvelope(input ApprovalEventInput) (*ApprovalEventEnvelope, error) {
	return aiapproval.NewRunCompletedEnvelope(input)
}
