package metrics

import (
	"context"
	"time"

	"github.com/cloudwego/eino/callbacks"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"gorm.io/gorm"

	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

type startTimeKey struct{}

// MetricsHandler 捕获 AI 模型的指标。
type MetricsHandler struct {
	db *gorm.DB
}

// NewMetricsHandler 创建一个新的 MetricsHandler 实例。
func NewMetricsHandler(db *gorm.DB) *MetricsHandler {
	return &MetricsHandler{db: db}
}

// OnStartFn 记录 AI 交互的开始。
func (h *MetricsHandler) OnStartFn(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	return context.WithValue(ctx, startTimeKey{}, time.Now())
}

// OnEndFn 记录 AI 交互的结束和指标。
func (h *MetricsHandler) OnEndFn(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	startTime, ok := ctx.Value(startTimeKey{}).(time.Time)
	if !ok {
		startTime = time.Now()
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime).Milliseconds()
	meta := runtimectx.AIMetadataFrom(ctx)

	span := &aimodel.AITraceSpan{
		ID:         uuid.NewString(),
		RunID:      meta.RunID,
		SessionID:  meta.SessionID,
		Scene:      meta.Scene,
		Status:     "success",
		DurationMS: duration,
		StartTime:  startTime,
		EndTime:    &endTime,
	}

	usageLog := &aimodel.AIUsageLog{
		RunID:     meta.RunID,
		SessionID: meta.SessionID,
		UserID:    meta.UserID,
		Scene:     meta.Scene,
		Status:    "success",
	}

	mo := einomodel.ConvCallbackOutput(output)
	if mo != nil && mo.TokenUsage != nil {
		span.PromptTokens = int64(mo.TokenUsage.PromptTokens)
		span.CompletionTokens = int64(mo.TokenUsage.CompletionTokens)
		span.Tokens = int64(mo.TokenUsage.TotalTokens)

		usageLog.PromptTokens = int64(mo.TokenUsage.PromptTokens)
		usageLog.CompletionTokens = int64(mo.TokenUsage.CompletionTokens)
		usageLog.TotalTokens = int64(mo.TokenUsage.TotalTokens)
	}

	// 异步持久化以避免阻塞主链路
	go func() {
		_ = h.db.Create(span).Error
		_ = h.db.Create(usageLog).Error
	}()

	return ctx
}

// OnEndWithStreamOutputFn 记录流式 AI 交互的结束和指标。
func (h *MetricsHandler) OnEndWithStreamOutputFn(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	// 必须异步读取私有流副本，否则会阻塞主输出流
	go func() {
		defer output.Close()

		var lastOutput callbacks.CallbackOutput
		for {
			chunk, err := output.Recv()
			if err != nil {
				break
			}
			lastOutput = chunk
		}

		// 获取最终累积的指标并持久化
		_ = h.OnEndFn(ctx, info, lastOutput)
	}()

	return ctx
}

// OnErrorFn 记录 AI 交互过程中的错误。
func (h *MetricsHandler) OnErrorFn(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	startTime, ok := ctx.Value(startTimeKey{}).(time.Time)
	if !ok {
		startTime = time.Now()
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime).Milliseconds()
	meta := runtimectx.AIMetadataFrom(ctx)

	span := &aimodel.AITraceSpan{
		ID:         uuid.NewString(),
		RunID:      meta.RunID,
		SessionID:  meta.SessionID,
		Scene:      meta.Scene,
		Status:     "error",
		DurationMS: duration,
		StartTime:  startTime,
		EndTime:    &endTime,
	}

	usageLog := &aimodel.AIUsageLog{
		RunID:     meta.RunID,
		SessionID: meta.SessionID,
		UserID:    meta.UserID,
		Scene:     meta.Scene,
		Status:    "error",
	}

	go func() {
		_ = h.db.Create(span).Error
		_ = h.db.Create(usageLog).Error
	}()

	return ctx
}

// Build 构造 Eino 回调处理器。
func (h *MetricsHandler) Build() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(h.OnStartFn).
		OnEndFn(h.OnEndFn).
		OnEndWithStreamOutputFn(h.OnEndWithStreamOutputFn).
		OnErrorFn(h.OnErrorFn).
		Build()
}
