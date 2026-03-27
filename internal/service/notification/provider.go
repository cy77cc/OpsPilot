// Package notification 提供通知管理服务。
//
// provider.go 实现可插拔的通知发送提供者。
// 支持的通道类型:
//   - log: 日志记录（开发调试）
//   - dingtalk: 钉钉机器人
//   - wecom: 企业微信机器人
//   - email: 邮件通知
//   - sms: 短信通知
package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
)

// Provider 是可插拔的通知发送接口。
//
// 所有通知通道（钉钉、企业微信、邮件等）都需实现此接口。
type Provider interface {
	// Name 返回提供者名称，用于注册和查找。
	Name() string
	// Send 发送告警通知到指定通道。
	Send(ctx context.Context, alert *model.AlertEvent, channel model.AlertNotificationChannel) error
	// ValidateConfig 验证通道配置是否有效。
	ValidateConfig(config map[string]any) error
}

// ProviderRegistry 是通知提供者注册中心。
//
// 管理所有已注册的通知发送提供者，支持并发安全访问。
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewProviderRegistry 创建空的提供者注册中心。
//
// 返回: 空的注册中心实例
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{providers: make(map[string]Provider)}
}

// NewDefaultProviderRegistry 创建预注册默认提供者的注册中心。
//
// 默认注册: log, dingtalk, wecom, email, sms。
//
// 返回: 包含默认提供者的注册中心实例
func NewDefaultProviderRegistry() *ProviderRegistry {
	r := NewProviderRegistry()
	r.Register(&LogProvider{})
	r.Register(&DingTalkProvider{client: &http.Client{Timeout: 5 * time.Second}})
	r.Register(&WeComProvider{client: &http.Client{Timeout: 5 * time.Second}})
	r.Register(&EmailProvider{})
	r.Register(&SMSProvider{})
	return r
}

// Register 注册通知提供者。
//
// 提供者名称不区分大小写，前后空格会被去除。
//
// 参数:
//   - p: 通知提供者实例
func (r *ProviderRegistry) Register(p Provider) {
	if r == nil || p == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[strings.ToLower(strings.TrimSpace(p.Name()))] = p
}

// Get 根据名称获取通知提供者。
//
// 参数:
//   - name: 提供者名称（不区分大小写）
//
// 返回: 提供者实例和是否找到标志
func (r *ProviderRegistry) Get(name string) (Provider, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[strings.ToLower(strings.TrimSpace(name))]
	return p, ok
}

// ParseChannelConfig 解析通道配置 JSON。
//
// 将 ConfigJSON 字段解析为 map 结构，解析失败返回空 map。
//
// 参数:
//   - channel: 告警通知通道模型
//
// 返回: 配置键值对
func ParseChannelConfig(channel model.AlertNotificationChannel) map[string]any {
	out := map[string]any{}
	if strings.TrimSpace(channel.ConfigJSON) == "" {
		return out
	}
	_ = json.Unmarshal([]byte(channel.ConfigJSON), &out)
	return out
}

// LogProvider 是日志通知提供者（开发调试用）。
//
// 实际不发送通知，仅用于开发环境测试。
type LogProvider struct{}

// Name 返回提供者名称 "log"。
func (p *LogProvider) Name() string { return "log" }

// ValidateConfig 日志提供者无需配置，始终返回 nil。
func (p *LogProvider) ValidateConfig(_ map[string]any) error { return nil }

// Send 日志提供者不实际发送，直接返回 nil。
func (p *LogProvider) Send(_ context.Context, alert *model.AlertEvent, channel model.AlertNotificationChannel) error {
	_ = alert
	_ = channel
	return nil
}

// DingTalkProvider 是钉钉机器人通知提供者。
//
// 通过钉钉机器人 Webhook 发送 Markdown 格式告警通知。
type DingTalkProvider struct{ client *http.Client }

// Name 返回提供者名称 "dingtalk"。
func (p *DingTalkProvider) Name() string { return "dingtalk" }

// ValidateConfig 验证钉钉配置。
//
// 必填字段: webhook。
//
// 参数:
//   - config: 配置键值对
//
// 返回: 配置无效返回错误
func (p *DingTalkProvider) ValidateConfig(config map[string]any) error {
	webhook := strings.TrimSpace(fmt.Sprintf("%v", config["webhook"]))
	if webhook == "" {
		return fmt.Errorf("dingtalk webhook is required")
	}
	return nil
}

// Send 发送钉钉告警通知。
//
// 发送 Markdown 格式消息，包含告警标题、状态、级别、指标、当前值和阈值。
// webhook 优先从配置读取，其次从 channel.Target 读取。
//
// 参数:
//   - ctx: 上下文
//   - alert: 告警事件模型
//   - channel: 告警通知通道模型
//
// 返回: 发送失败返回错误
func (p *DingTalkProvider) Send(ctx context.Context, alert *model.AlertEvent, channel model.AlertNotificationChannel) error {
	cfg := ParseChannelConfig(channel)
	if err := p.ValidateConfig(cfg); err != nil {
		return err
	}
	webhook := strings.TrimSpace(fmt.Sprintf("%v", cfg["webhook"]))
	if webhook == "" {
		webhook = strings.TrimSpace(channel.Target)
	}
	payload := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]any{
			"title": alert.Title,
			"text":  fmt.Sprintf("### %s\n\n- 状态: %s\n- 级别: %s\n- 指标: %s\n- 当前值: %.2f\n- 阈值: %.2f", alert.Title, alert.Status, alert.Severity, alert.Metric, alert.Value, alert.Threshold),
		},
	}
	return postJSON(ctx, p.client, webhook, payload)
}

// WeComProvider 是企业微信机器人通知提供者。
//
// 通过企业微信机器人 Webhook 发送 Markdown 格式告警通知。
type WeComProvider struct{ client *http.Client }

// Name 返回提供者名称 "wecom"。
func (p *WeComProvider) Name() string { return "wecom" }

// ValidateConfig 验证企业微信配置。
//
// 必填字段: webhook。
//
// 参数:
//   - config: 配置键值对
//
// 返回: 配置无效返回错误
func (p *WeComProvider) ValidateConfig(config map[string]any) error {
	webhook := strings.TrimSpace(fmt.Sprintf("%v", config["webhook"]))
	if webhook == "" {
		return fmt.Errorf("wecom webhook is required")
	}
	return nil
}

// Send 发送企业微信告警通知。
//
// 发送 Markdown 格式消息，包含告警标题、状态、级别、指标、当前值和阈值。
// webhook 优先从配置读取，其次从 channel.Target 读取。
//
// 参数:
//   - ctx: 上下文
//   - alert: 告警事件模型
//   - channel: 告警通知通道模型
//
// 返回: 发送失败返回错误
func (p *WeComProvider) Send(ctx context.Context, alert *model.AlertEvent, channel model.AlertNotificationChannel) error {
	cfg := ParseChannelConfig(channel)
	if err := p.ValidateConfig(cfg); err != nil {
		return err
	}
	webhook := strings.TrimSpace(fmt.Sprintf("%v", cfg["webhook"]))
	if webhook == "" {
		webhook = strings.TrimSpace(channel.Target)
	}
	payload := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]any{
			"content": fmt.Sprintf("**%s**\n> 状态: %s\n> 级别: %s\n> 指标: %s\n> 当前值: %.2f\n> 阈值: %.2f", alert.Title, alert.Status, alert.Severity, alert.Metric, alert.Value, alert.Threshold),
		},
	}
	return postJSON(ctx, p.client, webhook, payload)
}

// EmailProvider 是邮件通知提供者。
//
// 通过 SMTP 发送邮件告警通知（待实现）。
type EmailProvider struct{}

// Name 返回提供者名称 "email"。
func (p *EmailProvider) Name() string { return "email" }

// ValidateConfig 验证邮件配置。
//
// 必填字段: smtp_host。
//
// 参数:
//   - config: 配置键值对
//
// 返回: 配置无效返回错误
func (p *EmailProvider) ValidateConfig(config map[string]any) error {
	if strings.TrimSpace(fmt.Sprintf("%v", config["smtp_host"])) == "" {
		return fmt.Errorf("email smtp_host is required")
	}
	return nil
}

// Send 发送邮件告警通知（待实现）。
//
// 参数:
//   - _: 上下文（未使用）
//   - alert: 告警事件模型
//   - _: 通知通道模型（未使用）
//
// 返回: 当前直接返回 nil
func (p *EmailProvider) Send(_ context.Context, alert *model.AlertEvent, _ model.AlertNotificationChannel) error {
	_ = alert
	return nil
}

// SMSProvider 是短信通知提供者。
//
// 通过短信网关发送告警通知（待实现）。
type SMSProvider struct{}

// Name 返回提供者名称 "sms"。
func (p *SMSProvider) Name() string { return "sms" }

// ValidateConfig 短信提供者暂无必填配置，返回 nil。
func (p *SMSProvider) ValidateConfig(_ map[string]any) error { return nil }

// Send 发送短信告警通知（待实现）。
//
// 参数:
//   - _: 上下文（未使用）
//   - alert: 告警事件模型
//   - _: 通知通道模型（未使用）
//
// 返回: 当前直接返回 nil
func (p *SMSProvider) Send(_ context.Context, alert *model.AlertEvent, _ model.AlertNotificationChannel) error {
	_ = alert
	return nil
}

// postJSON 发送 JSON HTTP POST 请求。
//
// 用于向 Webhook 端点发送通知消息。
//
// 参数:
//   - ctx: 上下文
//   - c: HTTP 客户端
//   - endpoint: 目标 URL
//   - payload: 请求体 JSON 数据
//
// 返回: 发送失败返回错误
func postJSON(ctx context.Context, c *http.Client, endpoint string, payload map[string]any) error {
	if strings.TrimSpace(endpoint) == "" {
		return fmt.Errorf("notification endpoint is empty")
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("provider response status: %d", resp.StatusCode)
	}
	return nil
}
