// Package logic 实现 AI 模块的业务逻辑层。
//
// 本文件实现审批事件分发器，负责从 Outbox 读取事件并发布到事件总线。
package logic

import (
	"context"
	"fmt"
	"time"

	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// approvalEventDispatcherRetryDelay 重试延迟。
const approvalEventDispatcherRetryDelay = 5 * time.Second

// ApprovalEventDispatcher 审批事件分发器。
//
// 从 Outbox 表读取待处理事件并发布到内存事件总线。
type ApprovalEventDispatcher struct {
	db         *gorm.DB
	bus        *ApprovalEventBus
	retryDelay time.Duration
	now        func() time.Time
}

// NewApprovalEventDispatcher 创建审批事件分发器实例。
func NewApprovalEventDispatcher(db *gorm.DB, bus *ApprovalEventBus) *ApprovalEventDispatcher {
	return &ApprovalEventDispatcher{
		db:         db,
		bus:        bus,
		retryDelay: approvalEventDispatcherRetryDelay,
		now:        time.Now,
	}
}

func (d *ApprovalEventDispatcher) RunOnce(ctx context.Context) (bool, error) {
	if d == nil || d.db == nil || d.bus == nil {
		return false, nil
	}

	outboxDAO := aidao.NewAIApprovalOutboxDAO(d.db)
	event, err := outboxDAO.ClaimPending(ctx)
	if err != nil {
		return false, err
	}
	if event == nil {
		return false, nil
	}

	envelope := ApprovalEventEnvelope{
		EventID:     event.EventID,
		EventType:   event.EventType,
		OccurredAt:  event.OccurredAt,
		Sequence:    event.Sequence,
		Version:     event.Version,
		RunID:       event.RunID,
		SessionID:   event.SessionID,
		ApprovalID:  event.ApprovalID,
		ToolCallID:  event.ToolCallID,
		AggregateID: event.AggregateID,
		PayloadJSON: event.PayloadJSON,
	}

	if err := d.bus.Publish(ctx, envelope); err != nil {
		if retryErr := outboxDAO.MarkRetry(ctx, event.ID, d.now().Add(d.retryDelay)); retryErr != nil {
			return true, fmt.Errorf("publish approval event: %w; mark retry: %v", err, retryErr)
		}
		return true, err
	}

	if err := outboxDAO.MarkDone(ctx, event.ID); err != nil {
		if retryErr := outboxDAO.MarkRetry(ctx, event.ID, d.now().Add(d.retryDelay)); retryErr != nil {
			return true, fmt.Errorf("mark approval event done: %w; mark retry: %v", err, retryErr)
		}
		return true, err
	}
	return true, nil
}

func (d *ApprovalEventDispatcher) Run(ctx context.Context) error {
	for {
		claimed, err := d.RunOnce(ctx)
		if err != nil {
			return err
		}
		if !claimed {
			return nil
		}
	}
}

func (d *ApprovalEventDispatcher) Deliver(ctx context.Context, event *model.AIApprovalOutboxEvent) error {
	if event == nil {
		return nil
	}
	return d.bus.Publish(ctx, ApprovalEventEnvelope{
		EventID:     event.EventID,
		EventType:   event.EventType,
		OccurredAt:  event.OccurredAt,
		Sequence:    event.Sequence,
		Version:     event.Version,
		RunID:       event.RunID,
		SessionID:   event.SessionID,
		ApprovalID:  event.ApprovalID,
		ToolCallID:  event.ToolCallID,
		AggregateID: event.AggregateID,
		PayloadJSON: event.PayloadJSON,
	})
}
