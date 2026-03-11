// Package logger 提供统一的日志接口和实现。
//
// 本文件实现上下文相关的日志记录，支持从 context 中提取 trace_id。
package logger

import "context"

// ctxKey 是上下文键类型。
type ctxKey string

// traceIDKey 是 trace_id 在上下文中的键。
const traceIDKey ctxKey = "trace_id"

// WithTraceID 将 trace_id 存入上下文。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// WithContext 创建带有上下文信息的子 Logger。
//
// 从上下文中提取 trace_id 并添加到日志字段中。
func (z *zapLogger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return z
	}

	if v := ctx.Value(traceIDKey); v != nil {
		return z.With(String("trace_id", v.(string)))
	}
	return z
}
