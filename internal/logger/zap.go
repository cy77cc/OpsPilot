// Package logger 提供统一的日志接口和实现。
//
// 本文件实现基于 zap 的 Logger，支持 JSON 和文本格式输出。
// 配置从 config.CFG.Log 读取。
package logger

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// zapLogger 是基于 zap 的 Logger 实现。
type zapLogger struct {
	l *zap.Logger
}

// MustNewZapLogger 创建 zap Logger 实例，失败则 panic。
//
// 根据配置初始化日志级别、编码格式、输出路径等。
// 使用 AddCallerSkip(1) 确保调用者信息显示正确的文件和行号。
func MustNewZapLogger() Logger {

	if config.CFG.Log.File.Path != "" {
		os.Mkdir("log", 0644)
	}

	level := zap.NewAtomicLevel()
	levelStr := strings.ToLower(config.CFG.Log.Level)
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		log.Fatal(err)
		return nil
	}

	cfg := zap.Config{
		Level:       level,
		Development: config.CFG.App.Debug,
		Encoding:    config.CFG.Log.Format, // 生产推荐 json
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:      "ts",
			LevelKey:     "level",
			MessageKey:   "msg",
			CallerKey:    "caller",
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			EncodeLevel:  zapcore.LowercaseLevelEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout", config.CFG.Log.File.Path},
		ErrorOutputPaths: []string{"stderr", config.CFG.Log.File.Path},
	}

	// AddCallerSkip(1) 跳过一层调用栈，使日志显示的调用者为业务代码而非封装层
	logger, err := cfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		log.Fatal(err)
		return nil
	}

	return &zapLogger{l: logger}
}

// Info 记录信息级别日志。
func (z *zapLogger) Info(msg string, fields ...Field) {
	z.l.Info(msg, toZapFields(fields)...)
}

// Infof 记录格式化信息日志。
func (z *zapLogger) Infof(format string, a []any, fields ...Field) {
	z.l.Info(fmt.Sprintf(format, a...), toZapFields(fields)...)
}

// Error 记录错误级别日志。
func (z *zapLogger) Error(msg string, fields ...Field) {
	z.l.Error(msg, toZapFields(fields)...)
}

// Errorf 记录格式化错误日志。
func (z *zapLogger) Errorf(format string, a []any, fields ...Field) {
	z.l.Error(fmt.Sprintf(format, a...), toZapFields(fields)...)
}

// Debug 记录调试级别日志。
func (z *zapLogger) Debug(msg string, fields ...Field) {
	z.l.Debug(msg, toZapFields(fields)...)
}

// Debugf 记录格式化调试日志。
func (z *zapLogger) Debugf(format string, a []any, fields ...Field) {
	z.l.Debug(fmt.Sprintf(format, a...), toZapFields(fields)...)
}

// Warn 记录警告级别日志。
func (z *zapLogger) Warn(msg string, fields ...Field) {
	z.l.Warn(msg, toZapFields(fields)...)
}

// Warnf 记录格式化警告日志。
func (z *zapLogger) Warnf(format string, a []any, fields ...Field) {
	z.l.Warn(fmt.Sprintf(format, a...), toZapFields(fields)...)
}

// With 创建带有固定字段的子 Logger。
func (z *zapLogger) With(fields ...Field) Logger {
	return &zapLogger{
		l: z.l.With(toZapFields(fields)...),
	}
}

// toZapFields 将 Field 列表转换为 zap.Field 列表。
func toZapFields(fields []Field) []zap.Field {
	zfs := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zfs = append(zfs, zap.Any(f.Key, f.Value))
	}
	return zfs
}
