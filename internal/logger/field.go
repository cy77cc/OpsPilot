// Package logger 提供统一的日志接口和实现。
//
// 本文件定义结构化日志字段类型，用于键值对形式的日志记录。
package logger

// Field 是结构化日志字段。
type Field struct {
	Key   string // 字段名
	Value any    // 字段值
}

// String 创建字符串类型的日志字段。
func String(key, val string) Field {
	return Field{Key: key, Value: val}
}

// Int 创建整数类型的日志字段。
func Int(key string, val int) Field {
	return Field{Key: key, Value: val}
}

// Error 创建错误类型的日志字段。
//
// 固定使用 "error" 作为字段名。
func Error(err error) Field {
	return Field{Key: "error", Value: err}
}