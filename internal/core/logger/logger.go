// Package logger 提供统一的日志接口和实现。
//
// 本文件定义 Logger 接口，支持结构化日志记录。
// 提供 Debug/Info/Warn/Error 级别的日志方法。
package logger

import (
	"context"
)

// Logger 是统一的结构化日志接口。
type Logger interface {
	Debug(msg string, fields ...Field)                               // 记录调试级别日志
	Debugf(format string, a []any, fields ...Field)                   // 记录格式化调试日志
	Info(msg string, fields ...Field)                                 // 记录信息级别日志
	Infof(format string, a []any, fields ...Field)                    // 记录格式化信息日志
	Warn(msg string, fields ...Field)                                 // 记录警告级别日志
	Warnf(msg string, a []any, fields ...Field)                       // 记录格式化警告日志
	Error(msg string, fields ...Field)                                // 记录错误级别日志
	Errorf(msg string, a []any, fields ...Field)                      // 记录格式化错误日志
	With(fields ...Field) Logger                                      // 创建带有固定字段的子 Logger
	WithContext(ctx context.Context) Logger                           // 创建带有上下文信息的子 Logger
}

// std 是全局 Logger 实例。
var std Logger

// Init 初始化全局 Logger。
func Init(l Logger) {
	std = l
}

// L 返回全局 Logger 实例。
func L() Logger {
	return std
}
