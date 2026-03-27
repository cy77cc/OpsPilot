// Package handler 提供监控告警服务的 HTTP 处理器。
//
// 本文件提供通知器的基础类型和接口定义，
// 用于抽象不同类型的通知发送逻辑。
package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/model"
)

// NotificationPayload 是通知发送的载荷结构。
//
// 包含告警的核心信息，用于通知发送时的参数传递。
type NotificationPayload struct {
	AlertID   uint    // 告警 ID
	RuleID    uint    // 规则 ID
	Title     string  // 告警标题
	Message   string  // 告警消息
	Severity  string  // 严重级别
	Metric    string  // 指标名称
	Value     float64 // 当前值
	Threshold float64 // 阈值
}

// DeliveryResult 是通知投递结果结构。
//
// 记录通知发送的最终状态和可能的错误信息。
type DeliveryResult struct {
	Status string // 投递状态 (sent/failed)
	Error  string // 错误信息
}

// Notifier 定义通知发送接口。
//
// 不同类型的通知渠道 (日志、Webhook、邮件等) 实现此接口，
// 提供统一的通知发送能力。
type Notifier interface {
	// Send 发送通知。
	//
	// 参数:
	//   - ctx: 上下文
	//   - channel: 通知渠道配置
	//   - payload: 通知载荷
	//
	// 返回: 投递结果
	Send(ctx context.Context, channel model.AlertNotificationChannel, payload NotificationPayload) DeliveryResult
}

// logNotifier 是日志类型的通知器实现。
//
// 将告警信息输出到日志，用于开发和测试环境。
type logNotifier struct{}

// Send 发送日志通知。
//
// 实际不做任何操作，仅返回成功状态。
// 用于测试环境或不需要真实发送的场景。
//
// 参数:
//   - _: 上下文 (未使用)
//   - _: 渠道配置 (未使用)
//   - _: 通知载荷 (未使用)
//
// 返回: 总是返回 sent 状态
func (n *logNotifier) Send(_ context.Context, _ model.AlertNotificationChannel, _ NotificationPayload) DeliveryResult {
	return DeliveryResult{Status: "sent"}
}

// webhookNotifier 是 Webhook 类型的通知器实现。
//
// 通过 HTTP 请求将告警信息发送到外部系统。
type webhookNotifier struct{}

// Send 发送 Webhook 通知。
//
// 验证目标 URL 的有效性，实际 HTTP 发送逻辑在完整实现中补充。
//
// 参数:
//   - _: 上下文 (未使用)
//   - channel: 渠道配置，包含目标 URL
//   - payload: 通知载荷 (当前未使用)
//
// 返回: 投递结果，URL 无效时返回失败状态
func (n *webhookNotifier) Send(_ context.Context, channel model.AlertNotificationChannel, payload NotificationPayload) DeliveryResult {
	target := strings.TrimSpace(channel.Target)
	if target == "" {
		return DeliveryResult{Status: "failed", Error: "webhook target is empty"}
	}
	// Skeleton adapter. Real HTTP delivery can replace this without changing handler contract.
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		return DeliveryResult{Status: "failed", Error: "invalid webhook url"}
	}
	_ = payload
	return DeliveryResult{Status: "sent"}
}

// buildNotifier 根据渠道类型构建通知器实例。
//
// 工厂方法，根据类型返回对应的通知器实现。
// 不支持的类型返回错误。
//
// 参数:
//   - channelType: 渠道类型 (log/webhook 等)
//
// 返回: 通知器实例和可能的错误
func buildNotifier(channelType string) (Notifier, error) {
	switch strings.ToLower(strings.TrimSpace(channelType)) {
	case "", "log":
		return &logNotifier{}, nil
	case "webhook":
		return &webhookNotifier{}, nil
	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}
}
