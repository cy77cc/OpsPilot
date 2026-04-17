# Implement Eino Metrics Callback Handler Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement an Eino callback handler to capture usage metrics automatically in the `feat/ai-dashboard-metrics` worktree.

**Architecture:** A `MetricsHandler` will implement Eino's `callbacks.Handler` interface. it will capture `AITraceSpan` and `AIUsageLog` data from `OnChatModelEnd` and `OnError` callbacks and persist them to the database. It uses `runtimectx.AIMetadataFrom(ctx)` for session metadata.

**Tech Stack:** Go, GORM, Eino (CloudWeGo)

---

### Task 1: Initialize Directory and Test

**Files:**
- Create: `.worktrees/feat/ai-dashboard-metrics/internal/modules/ai/logic/metrics/handler_test.go`

- [ ] **Step 1: Create the directory in the worktree**
Run: `mkdir -p .worktrees/feat/ai-dashboard-metrics/internal/modules/ai/logic/metrics/`

- [ ] **Step 2: Write the failing test**

```go
package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

func TestMetricsHandler(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&aimodel.AITraceSpan{}, &aimodel.AIUsageLog{})

	handler := NewMetricsHandler(db)
	ctx := context.Background()
	ctx = runtimectx.WithAIMetadata(ctx, runtimectx.AIMetadata{
		SessionID: "session-1",
		RunID:     "run-1",
		UserID:    123,
		Scene:     "chat",
	})

	t.Run("Record Success", func(t *testing.T) {
		// Mock Start
		ctx, _ = handler.OnChatModelStart(ctx, &model.ChatModelInput{})
		
		// Mock End
		out := &model.ChatModelOutput{
			Usage: &model.Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}
		_, _ = handler.OnChatModelEnd(ctx, &model.ChatModelInput{}, out)

		// Wait for async persistence
		time.Sleep(100 * time.Millisecond)

		var span aimodel.AITraceSpan
		err := db.First(&span, "run_id = ?", "run-1").Error
		assert.NoError(t, err)
		assert.Equal(t, "success", span.Status)
		assert.Equal(t, int64(30), span.Tokens)

		var log aimodel.AIUsageLog
		err = db.First(&log, "run_id = ?", "run-1").Error
		assert.NoError(t, err)
		assert.Equal(t, int64(10), log.PromptTokens)
		assert.Equal(t, int64(20), log.CompletionTokens)
	})
}
```

- [ ] **Step 3: Run test to verify it fails**
Run: `go test -v .worktrees/feat/ai-dashboard-metrics/internal/modules/ai/logic/metrics/handler_test.go`
Expected: FAIL (NewMetricsHandler not defined)

### Task 2: Implement MetricsHandler

**Files:**
- Create: `.worktrees/feat/ai-dashboard-metrics/internal/modules/ai/logic/metrics/handler.go`

- [ ] **Step 1: Write the implementation**

```go
package metrics

import (
	"context"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/google/uuid"
	"gorm.io/gorm"

	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

// MetricsHandler implements Eino callbacks to capture AI metrics.
type MetricsHandler struct {
	db *gorm.DB
}

// NewMetricsHandler creates a new MetricsHandler.
func NewMetricsHandler(db *gorm.DB) *MetricsHandler {
	return &MetricsHandler{db: db}
}

var _ callbacks.Handler = (*MetricsHandler)(nil)

// OnChatModelStart is called when the chat model starts.
func (h *MetricsHandler) OnChatModelStart(ctx context.Context, in *model.ChatModelInput) (context.Context, error) {
	return context.WithValue(ctx, startTimeKey{}, time.Now()), nil
}

// OnChatModelEnd is called when the chat model ends successfully.
func (h *MetricsHandler) OnChatModelEnd(ctx context.Context, in *model.ChatModelInput, out *model.ChatModelOutput) (context.Context, error) {
	h.recordMetrics(ctx, out, nil)
	return ctx, nil
}

// OnError is called when an error occurs during chat model execution.
func (h *MetricsHandler) OnError(ctx context.Context, err error) (context.Context, error) {
	h.recordMetrics(ctx, nil, err)
	return ctx, nil
}

type startTimeKey struct{}

func (h *MetricsHandler) recordMetrics(ctx context.Context, out *model.ChatModelOutput, err error) {
	startTime, ok := ctx.Value(startTimeKey{}).(time.Time)
	if !ok {
		startTime = time.Now()
	}
	duration := time.Since(startTime).Milliseconds()
	meta := runtimectx.AIMetadataFrom(ctx)

	status := "success"
	if err != nil {
		status = "error"
	}

	var promptTokens, completionTokens, totalTokens int64
	if out != nil && out.Usage != nil {
		promptTokens = int64(out.Usage.PromptTokens)
		completionTokens = int64(out.Usage.CompletionTokens)
		totalTokens = int64(out.Usage.TotalTokens)
	}

	span := &aimodel.AITraceSpan{
		ID:         uuid.NewString(),
		RunID:      meta.RunID,
		SessionID:  meta.SessionID,
		Scene:      meta.Scene,
		Status:     status,
		Tokens:     totalTokens,
		DurationMS: duration,
		StartTime:  startTime,
	}

	usageLog := &aimodel.AIUsageLog{
		RunID:            meta.RunID,
		SessionID:        meta.SessionID,
		UserID:           meta.UserID,
		Scene:            meta.Scene,
		Status:           status,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
	}

	// Persist metrics asynchronously to avoid blocking the main flow.
	go func() {
		_ = h.db.Create(span).Error
		_ = h.db.Create(usageLog).Error
	}()
}
```

- [ ] **Step 2: Run test to verify it passes**
Run: `go test -v .worktrees/feat/ai-dashboard-metrics/internal/modules/ai/logic/metrics/...`
Expected: PASS

- [ ] **Step 3: Commit**
Run: `cd .worktrees/feat/ai-dashboard-metrics && git add internal/modules/ai/logic/metrics/ && git commit -m "feat(ai): implement Eino metrics callback handler"`

### Task 3: Final Verification

- [ ] **Step 1: Run all tests in the module**
Run: `go test -v .worktrees/feat/ai-dashboard-metrics/internal/modules/ai/...`

- [ ] **Step 2: Verify files exist in worktree**
Run: `ls -l .worktrees/feat/ai-dashboard-metrics/internal/modules/ai/logic/metrics/`
