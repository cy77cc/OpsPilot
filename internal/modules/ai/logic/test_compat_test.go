package logic

import "github.com/cy77cc/OpsPilot/internal/modules/ai/logic/event"

const (
	ApprovalEventTypeDecided = event.ApprovalEventTypeDecided
	ApprovalEventTypeExpired = event.ApprovalEventTypeExpired
	RunEventTypeResuming     = event.RunEventTypeResuming
	RunEventTypeResumed      = event.RunEventTypeResumed
	RunEventTypeResumeFailed = event.RunEventTypeResumeFailed
	RunEventTypeCompleted    = event.RunEventTypeCompleted
)
