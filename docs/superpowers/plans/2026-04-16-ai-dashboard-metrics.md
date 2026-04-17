# AI Assistant Dashboard Metrics and Session Sorting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement AI metrics collection via Eino callbacks and fix dashboard session sorting.

**Architecture:** A global Eino callback handler will capture metrics (tokens, duration, status) from all AI interactions and persist them to `ai_trace_spans` and `ai_usage_logs`. The dashboard logic will be updated to sort sessions by activity time.

**Tech Stack:** Go, GORM, Eino (CloudWeGo)

---

### Task 1: Fix Recent Conversations Sorting

**Files:**
- Modify: `internal/modules/dashboard/logic/logic.go`

- [ ] **Step 1: Update session query sorting**

Modify the `getAIActivity` function to sort by `updated_at` instead of `created_at`.

```go
// internal/modules/dashboard/logic/logic.go

// In getAIActivity function:
	// 查询最近的 AI 会话
	sessions := make([]aimodel.AIChatSession, 0, 5)
	if err := l.svcCtx.DB.WithContext(ctx).
		Order("updated_at DESC"). // Changed from created_at DESC
		Limit(5).
		Find(&sessions).Error; err != nil {
		return out, err
	}
```

- [ ] **Step 2: Commit**

```bash
git add internal/modules/dashboard/logic/logic.go
git commit -m "fix(dashboard): sort recent AI sessions by updated_at"
```

---

### Task 2: Implement Eino Metrics Callback Handler

**Files:**
- Create: `internal/modules/ai/logic/metrics/handler.go`
- Create: `internal/modules/ai/logic/metrics/handler_test.go`

- [ ] **Step 1: Create the metrics handler**

Implement the Eino callback handler to capture metrics.

```go
// internal/modules/ai/logic/metrics/handler.go
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

type MetricsHandler struct {
	db *gorm.DB
}

func NewMetricsHandler(db *gorm.DB) *MetricsHandler {
	return &MetricsHandler{db: db}
}

// OnChatModelStart 记录开始时间
func (h *MetricsHandler) OnChatModelStart(ctx context.Context, in *model.ChatModelInput) (context.Context, error) {
	return context.WithValue(ctx, "start_time", time.Now()), nil
}

// OnChatModelEnd 记录指标并持久化
func (h *MetricsHandler) OnChatModelEnd(ctx context.Context, in *model.ChatModelInput, out *model.ChatModelOutput) (context.Context, error) {
	startTime, ok := ctx.Value("start_time").(time.Time)
	if !ok {
		startTime = time.Now()
	}
	duration := time.Since(startTime).Milliseconds()
	meta := runtimectx.AIMetadataFrom(ctx)

	span := &aimodel.AITraceSpan{
		ID:         uuid.NewString(),
		RunID:      meta.RunID,
		SessionID:  meta.SessionID,
		Scene:      meta.Scene,
		Status:     "success",
		DurationMS: duration,
		StartTime:  startTime,
	}

	if out != nil && out.Usage != nil {
		span.Tokens = int64(out.Usage.TotalTokens)
	}

	// 异步保存以避免阻塞主流程
	go func() {
		_ = h.db.Create(span).Error
		// 还可以同时创建 AIUsageLog
	}()

	return ctx, nil
}
```

- [ ] **Step 2: Write tests for the handler**

- [ ] **Step 3: Run tests**

Run: `go test ./internal/modules/ai/logic/metrics/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/modules/ai/logic/metrics/
git commit -m "feat(ai): implement Eino metrics callback handler"
```

---

### Task 3: Register Global Metrics Callback

**Files:**
- Modify: `internal/svc/ai_runtime.go`
- Modify: `internal/svc/app_context.go` (if needed to pass DB)

- [ ] **Step 1: Update initAIRuntime to register metrics**

```go
// internal/svc/ai_runtime.go

func initAIMetricsCallback(db *gorm.DB) {
    handler := metrics.NewMetricsHandler(db)
    callbacks.AppendGlobalHandlers(handler)
}

// Update initAIRuntime to accept DB or call it from MustNewServiceContext
```

- [ ] **Step 2: Verify registration in ServiceContext**

- [ ] **Step 3: Commit**

```bash
git add internal/svc/
git commit -m "feat(ai): register global metrics callback"
```

---

### Task 4: Verify Dashboard Integration

- [ ] **Step 1: Check getAIActivity in logic.go**

Ensure it correctly queries `ai_trace_spans`.

- [ ] **Step 2: Final verification**
