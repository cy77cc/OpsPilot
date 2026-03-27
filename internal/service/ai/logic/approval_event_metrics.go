// Package logic 实现 AI 模块的业务逻辑层。
//
// 本文件定义审批事件的度量指标接口，用于监控审批流程的性能和可靠性。
package logic

import "time"

// ApprovalEventMetrics 审批事件度量指标记录器。
//
// 记录审批流程中的关键时间点和事件计数，用于监控和告警。
type ApprovalEventMetrics struct{}

// NewApprovalEventMetrics 创建度量指标记录器实例。
func NewApprovalEventMetrics() *ApprovalEventMetrics {
	return &ApprovalEventMetrics{}
}

func (m *ApprovalEventMetrics) RecordApprovalRequestToVisibleLatency(_ time.Duration) {}

func (m *ApprovalEventMetrics) RecordApprovalDecisionToResumingLatency(_ time.Duration) {}

func (m *ApprovalEventMetrics) RecordResumeSuccess() {}

func (m *ApprovalEventMetrics) RecordResumeRetry() {}

func (m *ApprovalEventMetrics) RecordOutboxLag(_ time.Duration) {}

func (m *ApprovalEventMetrics) RecordSSEReplaySuccess() {}

func (m *ApprovalEventMetrics) RecordSSECursorExpired() {}

func (m *ApprovalEventMetrics) RecordApprovalConflict() {}
