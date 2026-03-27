// Package logger 提供统一的日志接口和实现。
//
// 本文件实现上下文相关的日志记录，支持从 context 中提取 trace_id。
package logger

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

// WithTraceID 将 trace_id 存入上下文。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return runtimectx.WithTraceID(ctx, traceID)
}

// WithContext 创建带有上下文信息的子 Logger。
//
// 从上下文中提取 trace_id 并添加到日志字段中。
func (z *zapLogger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return z
	}

	if traceID := runtimectx.TraceID(ctx); traceID != "" {
		return z.With(String("trace_id", traceID))
	}
	return z
}
