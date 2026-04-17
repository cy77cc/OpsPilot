# Enhanced AI Assistant Dashboard Metrics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement detailed token consumption statistics (total, average interaction, average session) on the AI assistant dashboard.

**Architecture:** Update the backend API to calculate aggregate token metrics from `ai_trace_spans`, update the frontend API client, and enhance the dashboard UI card to display these new metrics.

**Tech Stack:** Go (GORM), React (TypeScript, Ant Design, Tailwind CSS)

---

### Task 1: Update Backend API and Logic

**Files:**
- Modify: `api/dashboard/v1/dashboard.go`
- Modify: `internal/modules/dashboard/logic/logic.go`

- [ ] **Step 1: Update AIStatsSummary struct**

Add average interaction and average session token fields.

```go
// api/dashboard/v1/dashboard.go

type AIStatsSummary struct {
	SessionCount            int64   `json:"sessionCount"`
	TokenCount              int64   `json:"tokenCount"`              // Total Tokens
	AvgTokenPerInteraction int64   `json:"avgTokenPerInteraction"` // New
	AvgTokenPerSession     int64   `json:"avgTokenPerSession"`     // New
	AvgDurationMs           int64   `json:"avgDurationMs"`
	SuccessRate             float64 `json:"successRate"`
	PreviousChange          string  `json:"previousChange"`
}
```

- [ ] **Step 2: Update aggregation logic**

Calculate the new metrics in `getAIActivity`.

```go
// internal/modules/dashboard/logic/logic.go

func (l *Logic) getAIActivity(ctx context.Context, since, now time.Time) (dashboardv1.AIActivity, error) {
    // ... existing stats query ...
    
    // Total interactions and unique sessions for token averages
    type tokenStats struct {
        TotalTokens         int64
        InteractionCount    int64
        UniqueSessionCount  int64
    }
    var ts tokenStats
    if err := l.svcCtx.DB.WithContext(ctx).
        Model(&aimodel.AITraceSpan{}).
        Where("start_time >= ? AND start_time <= ?", since, now).
        Select("COALESCE(SUM(tokens), 0) as total_tokens, COUNT(*) as interaction_count, COUNT(DISTINCT session_id) as unique_session_count").
        Scan(&ts).Error; err != nil {
        return out, err
    }

    var avgPerInteraction, avgPerSession int64
    if ts.InteractionCount > 0 {
        avgPerInteraction = ts.TotalTokens / ts.InteractionCount
    }
    if ts.UniqueSessionCount > 0 {
        avgPerSession = ts.TotalTokens / ts.UniqueSessionCount
    }

    out.Stats = dashboardv1.AIStatsSummary{
        SessionCount:            stats.TotalCount,
        TokenCount:              ts.TotalTokens,
        AvgTokenPerInteraction: avgPerInteraction,
        AvgTokenPerSession:     avgPerSession,
        AvgDurationMs:           avgDuration,
        SuccessRate:             successRate,
    }
    // ... rest of the function ...
}
```

- [ ] **Step 2.1: Run tests**
Run: `go test ./internal/modules/dashboard/logic/...`

- [ ] **Step 3: Commit**

```bash
git add api/dashboard/v1/dashboard.go internal/modules/dashboard/logic/logic.go
git commit -m "feat(dashboard): add average token metrics to backend"
```

---

### Task 2: Update Frontend API Client

**Files:**
- Modify: `web/src/api/modules/dashboard.ts`

- [ ] **Step 1: Update AIStatsSummary interface**

```typescript
// web/src/api/modules/dashboard.ts

export interface AIStatsSummary {
  sessionCount: number;
  tokenCount: number;
  avgTokenPerInteraction: number; // New
  avgTokenPerSession: number;     // New
  avgDurationMs: number;
  successRate: number;
  previousChange?: string;
}
```

- [ ] **Step 2: Update normalizeAIStats**

```typescript
// web/src/api/modules/dashboard.ts

const normalizeAIStats = (data: any): AIStatsSummary => ({
  sessionCount: Number(data?.sessionCount || 0),
  tokenCount: Number(data?.tokenCount || 0),
  avgTokenPerInteraction: Number(data?.avgTokenPerInteraction || 0), // New
  avgTokenPerSession: Number(data?.avgTokenPerSession || 0),         // New
  avgDurationMs: Number(data?.avgDurationMs || 0),
  successRate: Number(data?.successRate || 0),
  previousChange: String(data?.previousChange || ''),
});
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/dashboard.ts
git commit -m "feat(dashboard): update frontend API client for enhanced AI metrics"
```

---

### Task 3: Update Dashboard UI

**Files:**
- Modify: `web/src/components/Dashboard/AIActivityCard.tsx`

- [ ] **Step 1: Update AIActivityCard component**

Enhance the metrics display to show total, average interaction, and average session tokens.

```tsx
// web/src/components/Dashboard/AIActivityCard.tsx

// Modify the Statistics display area to include new metrics.
// Use formatTokens(stats.avgTokenPerSession) and formatTokens(stats.avgTokenPerInteraction)
```

- [ ] **Step 2: Commit**

```bash
git add web/src/components/Dashboard/AIActivityCard.tsx
git commit -m "feat(dashboard): enhance AI metrics display in UI"
```

---

### Task 4: Final Verification

- [ ] **Step 1: Build and run the project**

Run: `go build ./cmd/opspilot` and check the web interface.

- [ ] **Step 2: Verify metrics display**

Confirm that "Token 消耗 (总)", "Token (平均/对话)", and "Token (平均/请求)" are displayed with correct values and formatting.
