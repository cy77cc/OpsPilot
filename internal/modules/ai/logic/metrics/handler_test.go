package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/eino/callbacks"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

func TestMetricsHandler(t *testing.T) {
	// 初始化内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&aimodel.AITraceSpan{}, &aimodel.AIUsageLog{})
	assert.NoError(t, err)

	handler := NewMetricsHandler(db)
	ctx := context.Background()

	// 注入 AI 元数据
	meta := runtimectx.AIMetadata{
		SessionID: uuid.NewString(),
		RunID:     uuid.NewString(),
		Scene:     "test",
		UserID:    1,
	}
	ctx = runtimectx.WithAIMetadata(ctx, meta)

	info := &callbacks.RunInfo{
		Name: "test_model",
	}

	// 测试 OnStartFn
	ctx = handler.OnStartFn(ctx, info, nil)
	assert.NotNil(t, ctx.Value(startTimeKey{}))

	// 模拟耗时
	time.Sleep(10 * time.Millisecond)

	// 测试 OnEndFn
	output := &einomodel.CallbackOutput{
		TokenUsage: &einomodel.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}
	ctx = handler.OnEndFn(ctx, info, output)

	// 等待异步写入完成
	time.Sleep(100 * time.Millisecond)

	// 验证 AITraceSpan
	var span aimodel.AITraceSpan
	err = db.Where("run_id = ?", meta.RunID).First(&span).Error
	assert.NoError(t, err)
	assert.Equal(t, meta.SessionID, span.SessionID)
	assert.Equal(t, "success", span.Status)
	assert.Equal(t, int64(30), span.Tokens)
	assert.True(t, span.DurationMS >= 10)

	// 验证 AIUsageLog
	var usageLog aimodel.AIUsageLog
	err = db.Where("run_id = ?", meta.RunID).First(&usageLog).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(10), usageLog.PromptTokens)
	assert.Equal(t, int64(20), usageLog.CompletionTokens)
	assert.Equal(t, int64(30), usageLog.TotalTokens)
}

func TestMetricsHandler_NoDBIsNoOp(t *testing.T) {
	handler := NewMetricsHandler(nil)
	ctx := runtimectx.WithAIMetadata(context.Background(), runtimectx.AIMetadata{
		SessionID: uuid.NewString(),
		RunID:     uuid.NewString(),
		Scene:     "test",
		UserID:    1,
	})

	ctx = handler.OnStartFn(ctx, &callbacks.RunInfo{Name: "test_model"}, nil)
	_ = handler.OnEndFn(ctx, &callbacks.RunInfo{Name: "test_model"}, &einomodel.CallbackOutput{})
	_ = handler.OnErrorFn(ctx, &callbacks.RunInfo{Name: "test_model"}, assert.AnError)
	time.Sleep(20 * time.Millisecond)
}
