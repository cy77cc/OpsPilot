// Package cloud 提供云厂商主机导入的统一接口和适配器管理。
package cloud

import (
	"context"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// RetryConfig 重试配置。
type RetryConfig struct {
	// MaxRetries 最大重试次数。
	MaxRetries int

	// InitialDelay 初始延迟。
	InitialDelay time.Duration

	// MaxDelay 最大延迟。
	MaxDelay time.Duration

	// Multiplier 退避乘数。
	Multiplier float64
}

// DefaultRetryConfig 默认重试配置。
var DefaultRetryConfig = RetryConfig{
	MaxRetries:   3,
	InitialDelay: 500 * time.Millisecond,
	MaxDelay:     5 * time.Second,
	Multiplier:   2.0,
}

// RetryableErrors 可重试的错误码映射。
//
// 键为云厂商标识，值为可重试的错误码列表。
var RetryableErrors = map[string][]string{
	"alicloud":  {"Throttling", "ServiceUnavailable", "InternalError"},
	"volcengine": {"RequestLimitExceeded", "ServiceUnavailable"},
	"ucloud":    {"172", "5000"}, // 172: 请求频率限制, 5000: 服务内部错误
}

// DoWithRetry 执行带重试的操作。
//
// 参数:
//   - ctx: 上下文
//   - provider: 云厂商标识
//   - config: 重试配置
//   - op: 操作名称（用于日志）
//   - fn: 实际操作函数
//
// 返回:
//   - 成功返回结果
//   - 重试耗尽后返回最后一次错误
func DoWithRetry[T any](ctx context.Context, provider string, config RetryConfig, op string, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error
	delay := config.InitialDelay

	for i := 0; i <= config.MaxRetries; i++ {
		result, lastErr = fn()
		if lastErr == nil {
			return result, nil
		}

		// 检查是否可重试
		if !isRetryableError(provider, lastErr) {
			return result, lastErr
		}

		// 最后一次重试失败，不再等待
		if i == config.MaxRetries {
			break
		}

		// 等待后重试
		logrus.Debugf("云 API 请求失败，%s 后重试 (第 %d 次): %s", delay, i+1, op)
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
		}

		// 指数退避
		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return result, lastErr
}

// isRetryableError 检查错误是否可重试。
func isRetryableError(provider string, err error) bool {
	codes, ok := RetryableErrors[provider]
	if !ok {
		return false
	}

	errStr := err.Error()
	for _, code := range codes {
		if strings.Contains(errStr, code) {
			return true
		}
	}
	return false
}
