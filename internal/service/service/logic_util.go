// Package service 提供服务管理模块的通用工具函数。
//
// 本文件包含 JSON 序列化、字符串处理、配置规范化等工具函数。
package service

import (
	"encoding/json"
	"strings"
)

// mustJSON 将任意值序列化为 JSON 字符串。
//
// 忽略序列化错误，返回空字符串或序列化结果。
//
// 参数:
//   - v: 任意值
//
// 返回: JSON 字符串
func mustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

// normalizeStringMap 规范化字符串映射。
//
// 去除键值两端的空格，过滤空键。
//
// 参数:
//   - in: 输入映射
//
// 返回: 规范化后的映射
func normalizeStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(v)
	}
	return out
}

// buildLegacyEnvs 构建旧版环境变量 JSON 字符串。
//
// 用于兼容旧版数据结构。
//
// 参数:
//   - cfg: 标准服务配置
//
// 返回: JSON 字符串
func buildLegacyEnvs(cfg *StandardServiceConfig) string {
	if cfg == nil {
		return ""
	}
	b, _ := json.Marshal(cfg.Envs)
	return string(b)
}

// buildLegacyResources 构建旧版资源配置 JSON 字符串。
//
// 用于兼容旧版数据结构。
//
// 参数:
//   - cfg: 标准服务配置
//
// 返回: JSON 字符串
func buildLegacyResources(cfg *StandardServiceConfig) string {
	if cfg == nil {
		return ""
	}
	b, _ := json.Marshal(map[string]any{"limits": cfg.Resources})
	return string(b)
}

// truncateStr 截断字符串到指定最大长度。
//
// 参数:
//   - v: 输入字符串
//   - max: 最大长度
//
// 返回: 截断后的字符串
func truncateStr(v string, max int) string {
	s := strings.TrimSpace(v)
	if len(s) <= max || max <= 0 {
		return s
	}
	return s[:max]
}

// ensureStandardConfig 确保标准配置非空并填充默认值。
//
// 若配置为 nil，创建默认配置；否则填充缺失的默认值。
//
// 参数:
//   - cfg: 标准服务配置
//
// 返回: 填充默认值后的配置
func ensureStandardConfig(cfg *StandardServiceConfig) *StandardServiceConfig {
	if cfg == nil {
		cfg = &StandardServiceConfig{
			Image:    "nginx:latest",
			Replicas: 1,
			Resources: map[string]string{
				"cpu":    "500m",
				"memory": "512Mi",
			},
		}
	}
	if strings.TrimSpace(cfg.Image) == "" {
		cfg.Image = "nginx:latest"
	}
	if cfg.Replicas <= 0 {
		cfg.Replicas = 1
	}
	if len(cfg.Ports) == 0 {
		cfg.Ports = []PortConfig{{
			Name:          "http",
			Protocol:      "TCP",
			ContainerPort: 8080,
			ServicePort:   80,
		}}
	}
	return cfg
}
