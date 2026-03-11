// Package prometheus 提供 Prometheus HTTP API 客户端实现。
//
// 本文件定义客户端配置和规范化逻辑。
package prometheus

import (
	"fmt"
	"strings"
	"time"
)

// Config 是 Prometheus 客户端配置。
type Config struct {
	Address       string        `yaml:"address" json:"address"`             // Prometheus 地址 (如 http://prometheus:9090)
	Host          string        `yaml:"host" json:"host"`                   // Prometheus 主机
	Port          string        `yaml:"port" json:"port"`                   // Prometheus 端口
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`             // 请求超时
	MaxConcurrent int           `yaml:"max_concurrent" json:"max_concurrent"` // 最大并发数
	RetryCount    int           `yaml:"retry_count" json:"retry_count"`     // 重试次数
}

// Normalize 规范化配置，填充默认值。
//
// 默认值：
//   - 端口: 9090
//   - 超时: 10s
//   - 最大并发: 10
//   - 重试次数: 3
func (c Config) Normalize() Config {
	out := c
	if strings.TrimSpace(out.Address) == "" {
		h := strings.TrimSpace(out.Host)
		p := strings.TrimSpace(out.Port)
		if h != "" {
			if p == "" {
				p = "9090"
			}
			out.Address = fmt.Sprintf("http://%s:%s", h, p)
		}
	}
	if out.Timeout <= 0 {
		out.Timeout = 10 * time.Second
	}
	if out.MaxConcurrent <= 0 {
		out.MaxConcurrent = 10
	}
	if out.RetryCount <= 0 {
		out.RetryCount = 3
	}
	return out
}
