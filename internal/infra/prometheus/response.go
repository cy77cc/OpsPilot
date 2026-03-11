// Package prometheus 提供 Prometheus HTTP API 客户端实现。
//
// 本文件定义 API 响应结构和解析函数。
package prometheus

import (
	"encoding/json"
	"fmt"
)

// queryEnvelope 是 Prometheus API 响应信封。
type queryEnvelope struct {
	Status    string          `json:"status"`    // 响应状态 (success/error)
	Data      json.RawMessage `json:"data"`      // 响应数据
	ErrorType string          `json:"errorType"` // 错误类型
	Error     string          `json:"error"`     // 错误消息
}

// queryData 是即时查询结果数据。
type queryData struct {
	ResultType string        `json:"resultType"` // 结果类型 (vector/matrix/scalar/string)
	Result     []VectorPoint `json:"result"`     // 向量数据点
}

// rangeData 是范围查询结果数据。
type rangeData struct {
	ResultType string        `json:"resultType"` // 结果类型
	Result     []MatrixPoint `json:"result"`     // 矩阵数据点
}

// VectorPoint 是即时向量数据点。
type VectorPoint struct {
	Metric map[string]string `json:"metric"` // 指标标签
	Value  []any             `json:"value"`  // [timestamp, value]
}

// MatrixPoint 是范围向量数据点。
type MatrixPoint struct {
	Metric map[string]string `json:"metric"` // 指标标签
	Values [][]any           `json:"values"` // [[timestamp, value], ...]
}

// QueryResult 是统一的查询结果。
type QueryResult struct {
	ResultType string         // 结果类型
	Vector     []VectorPoint  // 即时向量结果
	Matrix     []MatrixPoint  // 范围向量结果
}

// MetadataItem 是指标元数据项。
type MetadataItem struct {
	Metric string `json:"metric"` // 指标名称
	Type   string `json:"type"`   // 指标类型 (counter/gauge/histogram/summary)
	Help   string `json:"help"`   // 帮助文本
	Unit   string `json:"unit"`   // 单位
}

// parseQueryEnvelope 解析 Prometheus API 响应信封。
func parseQueryEnvelope(body []byte) (*queryEnvelope, error) {
	var env queryEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	if env.Status != "success" {
		if env.Error != "" {
			return nil, fmt.Errorf("prometheus %s: %s", env.ErrorType, env.Error)
		}
		return nil, fmt.Errorf("prometheus status: %s", env.Status)
	}
	return &env, nil
}
