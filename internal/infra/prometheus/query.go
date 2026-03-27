// Package prometheus 提供 Prometheus HTTP API 客户端实现。
//
// 本文件实现 PromQL 查询构建器，支持标签、范围向量和聚合函数。
package prometheus

import (
	"fmt"
	"sort"
	"strings"
)

// allowedAgg 是允许的聚合函数列表。
var allowedAgg = map[string]struct{}{
	"avg": {}, "sum": {}, "max": {}, "min": {}, "count": {}, "stddev": {}, "stdvar": {},
}

// QueryBuilder 是 PromQL 查询构建器。
type QueryBuilder struct {
	metric    string            // 指标名称
	labels    map[string]string // 标签键值对
	rangeExpr string            // 范围向量表达式（如 "5m"）
	aggFunc   string            // 聚合函数
}

// NewQueryBuilder 创建查询构建器。
func NewQueryBuilder(metric string) *QueryBuilder {
	return &QueryBuilder{metric: strings.TrimSpace(metric), labels: make(map[string]string)}
}

// WithLabel 添加标签选择器。
func (b *QueryBuilder) WithLabel(key, value string) *QueryBuilder {
	k := strings.TrimSpace(key)
	if k == "" {
		return b
	}
	b.labels[k] = strings.TrimSpace(value)
	return b
}

// WithRange 设置范围向量表达式（如 "5m"、"1h"）。
func (b *QueryBuilder) WithRange(expr string) *QueryBuilder {
	b.rangeExpr = strings.TrimSpace(expr)
	return b
}

// WithAggregation 设置聚合函数（如 "avg"、"sum"）。
func (b *QueryBuilder) WithAggregation(agg string) *QueryBuilder {
	a := strings.ToLower(strings.TrimSpace(agg))
	if _, ok := allowedAgg[a]; ok {
		b.aggFunc = a
	}
	return b
}

// Build 构建 PromQL 查询字符串。
func (b *QueryBuilder) Build() string {
	base := b.metric
	if len(b.labels) > 0 {
		keys := make([]string, 0, len(b.labels))
		for k := range b.labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=\"%s\"", k, b.labels[k]))
		}
		base = fmt.Sprintf("%s{%s}", base, strings.Join(parts, ","))
	}
	if b.rangeExpr != "" {
		base = fmt.Sprintf("%s[%s]", base, b.rangeExpr)
	}
	if b.aggFunc != "" {
		base = fmt.Sprintf("%s(%s)", b.aggFunc, base)
	}
	return base
}
