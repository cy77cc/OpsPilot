// Package logic 实现 AI 模块的业务逻辑层。
//
// 本文件实现审批事件的内存发布订阅总线，用于进程内事件分发。
package logic

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// ApprovalEventHandler 审批事件处理函数类型。
type ApprovalEventHandler func(context.Context, ApprovalEventEnvelope) error

// approvalEventSubscription 审批事件订阅。
type approvalEventSubscription struct {
	pattern string
	handler ApprovalEventHandler
}

// ApprovalEventBus 审批事件总线。
//
// 提供进程内的发布订阅功能，支持模式匹配订阅。
// 用于审批事件的多处理器分发场景。
type ApprovalEventBus struct {
	mu            sync.RWMutex
	subscriptions []approvalEventSubscription
}

// NewApprovalEventBus 创建审批事件总线实例。
func NewApprovalEventBus() *ApprovalEventBus {
	return &ApprovalEventBus{}
}

// Subscribe 订阅审批事件。
//
// 支持模式匹配:
//   - "*" 匹配所有事件
//   - "approval_*" 匹配前缀
//   - 精确匹配事件类型
//
// 参数:
//   - pattern: 事件类型模式
//   - handler: 事件处理函数
func (b *ApprovalEventBus) Subscribe(pattern string, handler ApprovalEventHandler) error {
	if b == nil {
		return fmt.Errorf("approval event bus is nil")
	}
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return fmt.Errorf("subscription pattern is required")
	}
	if handler == nil {
		return fmt.Errorf("subscription handler is required")
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscriptions = append(b.subscriptions, approvalEventSubscription{
		pattern: pattern,
		handler: handler,
	})
	return nil
}

// Publish 发布审批事件。
//
// 将事件分发给所有匹配的订阅者。
func (b *ApprovalEventBus) Publish(ctx context.Context, event ApprovalEventEnvelope) error {
	if b == nil {
		return fmt.Errorf("approval event bus is nil")
	}

	b.mu.RLock()
	subs := make([]approvalEventSubscription, len(b.subscriptions))
	copy(subs, b.subscriptions)
	b.mu.RUnlock()

	for _, sub := range subs {
		if !approvalEventPatternMatches(sub.pattern, event.EventType) {
			continue
		}
		if err := sub.handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func approvalEventPatternMatches(pattern, eventType string) bool {
	pattern = strings.TrimSpace(pattern)
	eventType = strings.TrimSpace(eventType)
	if pattern == "" || eventType == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(eventType, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == eventType
}
