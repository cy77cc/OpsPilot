// Package approval 实现审批过期扫描器。
package approval

import (
	"context"
	"time"

	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
)

const expirerDefaultPollInterval = 2 * time.Second

// Expirer 审批过期扫描器。
type Expirer struct {
	logic *Logic
	now   func() time.Time
}

// NewExpirer 创建审批过期扫描器实例。
func NewExpirer(l *Logic) *Expirer {
	return &Expirer{logic: l, now: time.Now}
}

// WithClock 设置自定义时钟。
func (e *Expirer) WithClock(now func() time.Time) *Expirer {
	if now != nil {
		e.now = now
	}
	return e
}

// RunLoop 运行主循环。
func (e *Expirer) RunLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = expirerDefaultPollInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		claimed, _ := e.RunOnce(ctx)
		if claimed {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RunOnce 执行一次过期扫描。
func (e *Expirer) RunOnce(ctx context.Context) (bool, error) {
	if e == nil || e.logic == nil || e.logic.SvcCtx == nil || e.logic.SvcCtx.DB == nil || e.logic.ApprovalDAO == nil {
		return false, nil
	}
	now := e.now().UTC()
	tasks, err := e.logic.ApprovalDAO.ListPending(ctx, 200)
	if err != nil {
		return false, err
	}
	expiredAny := false
	for i := range tasks {
		task := tasks[i]
		if task.ExpiresAt == nil || !task.ExpiresAt.Before(now) {
			continue
		}
		if err := e.expireTask(ctx, &task, now); err != nil {
			return expiredAny, err
		}
		expiredAny = true
	}
	return expiredAny, nil
}

func (e *Expirer) expireTask(ctx context.Context, snapshot *ai.AIApprovalTask, now time.Time) error {
	if snapshot == nil {
		return nil
	}
	if e == nil || e.logic == nil || e.logic.SvcCtx == nil || e.logic.SvcCtx.DB == nil {
		return nil
	}
	// Use the passed `now` for consistent time handling; ExpireApproval uses its own time.Now()
	// internally but we pass the scanner's time for logging consistency.
	_, err := NewWriteModel(e.logic.SvcCtx.DB).ExpireApproval(ctx, snapshot.ApprovalID)
	return err
}
