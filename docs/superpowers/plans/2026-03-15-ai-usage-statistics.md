# AI 使用统计功能实现计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 AI 模块添加 token 使用量、审批统计、工具调用统计等指标，支持 Prometheus 监控、数据库持久化和前端仪表盘展示。

**Architecture:** 新增 `ai_usage_logs` 表记录请求级别的汇总统计，在 `orchestrator.go` 中从 LLM ResponseMeta 提取 token 数据并累加，通过 DAO 层持久化，提供 REST API 供前端查询展示。

**Tech Stack:** Go 1.26, GORM, Gin, Prometheus, React 19, Ant Design 6, @ant-design/charts

---

## 文件结构

```
internal/
├── model/
│   └── ai_usage_log.go           # 新增：数据模型
├── dao/
│   └── ai_usage_log.go           # 新增：数据访问层
├── ai/
│   ├── orchestrator.go           # 修改：添加统计采集和写入
│   └── observability/
│       └── metrics.go            # 修改：新增 first_token_seconds 指标
└── service/ai/
    ├── handler.go                # 修改：添加 UsageLogDAO 依赖
    ├── usage_handler.go          # 新增：统计接口处理器
    └── routes.go                 # 修改：注册统计路由

storage/migration/
└── dev_auto.go                   # 修改：添加 AIUsageLog 迁移

web/src/
├── api/modules/ai/
│   └── usage.ts                  # 新增：API 客户端
└── pages/AI/Usage/
    ├── index.tsx                 # 新增：页面入口
    ├── components/
    │   ├── StatCards.tsx         # 新增：总量卡片
    │   ├── UsageTrendChart.tsx   # 新增：趋势图
    │   ├── ScenePieChart.tsx     # 新增：场景分布
    │   ├── ApprovalChart.tsx     # 新增：审批统计
    │   └── UsageTable.tsx        # 新增：请求列表
    └── hooks/
        └── useUsageStats.ts      # 新增：数据获取 hook
```

---

## Chunk 1: 数据模型与 DAO 层

### Task 1: 创建 AIUsageLog 模型

**Files:**
- Create: `internal/model/ai_usage_log.go`
- Modify: `storage/migration/dev_auto.go:88`

- [ ] **Step 1: 创建数据模型**

```go
// internal/model/ai_usage_log.go

// Package model 提供数据模型定义。
package model

import "time"

// AIUsageLog 记录单次 AI 执行的统计数据。
//
// 与 ai_executions 表的区别：
//   - ai_executions: 工具执行级别，每次 Tool 调用一条
//   - ai_usage_logs: 请求级别，每次 Run/Resume 一条汇总
type AIUsageLog struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	TraceID  string `gorm:"column:trace_id;type:varchar(36);not null"`
	SessionID string `gorm:"column:session_id;type:varchar(36);not null;index:idx_ai_usage_logs_session_id"`
	PlanID    string `gorm:"column:plan_id;type:varchar(36);not null"`
	TurnID    string `gorm:"column:turn_id;type:varchar(36)"`
	UserID    uint64 `gorm:"column:user_id;index:idx_ai_usage_logs_user_created"`

	Scene     string `gorm:"column:scene;type:varchar(64);index:idx_ai_usage_logs_scene"`
	Operation string `gorm:"column:operation;type:varchar(32)"`
	Status    string `gorm:"column:status;type:varchar(32);index:idx_ai_usage_logs_status"`

	PromptTokens     int     `gorm:"column:prompt_tokens;default:0"`
	CompletionTokens int     `gorm:"column:completion_tokens;default:0"`
	TotalTokens      int     `gorm:"column:total_tokens;default:0"`
	EstimatedCostUSD float64 `gorm:"column:estimated_cost_usd;type:decimal(10,6)"`
	ModelName        string  `gorm:"column:model_name;type:varchar(128)"`

	DurationMs      int     `gorm:"column:duration_ms"`
	FirstTokenMs    int     `gorm:"column:first_token_ms"`
	TokensPerSecond float64 `gorm:"column:tokens_per_second;type:decimal(10,2)"`

	ApprovalCount  int    `gorm:"column:approval_count;default:0"`
	ApprovalStatus string `gorm:"column:approval_status;type:varchar(32);default:'none'"`
	ApprovalWaitMs int    `gorm:"column:approval_wait_ms;default:0"`

	ToolCallCount  int `gorm:"column:tool_call_count;default:0"`
	ToolErrorCount int `gorm:"column:tool_error_count;default:0"`

	ErrorType    string `gorm:"column:error_type;type:varchar(64)"`
	ErrorMessage string `gorm:"column:error_message;type:text"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;index:idx_ai_usage_logs_created_at"`
}

// TableName 返回表名。
func (AIUsageLog) TableName() string {
	return "ai_usage_logs"
}
```

- [ ] **Step 2: 添加到迁移列表**

在 `storage/migration/dev_auto.go` 的 `RunDevAutoMigrate` 函数末尾添加：

```go
		&model.RiskFinding{},
		&model.Anomaly{},
		&model.Suggestion{},
		&model.AIUsageLog{}, // 新增
	)
```

- [ ] **Step 3: 验证编译通过**

Run: `go build ./...`
Expected: 编译成功，无错误

- [ ] **Step 4: 提交**

```bash
git add internal/model/ai_usage_log.go storage/migration/dev_auto.go
git commit -m "feat(ai): add AIUsageLog model for usage statistics"
```

---

### Task 2: 创建 UsageLogDAO

**Files:**
- Create: `internal/dao/ai_usage_log.go`

- [ ] **Step 1: 创建 DAO 文件**

```go
// internal/dao/ai_usage_log.go

// Package dao 提供数据访问对象。
package dao

import (
	"context"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// UsageLogDAO 是 AI 使用统计的数据访问对象。
type UsageLogDAO struct {
	db *gorm.DB
}

// NewUsageLogDAO 创建 UsageLogDAO 实例。
func NewUsageLogDAO(db *gorm.DB) *UsageLogDAO {
	return &UsageLogDAO{db: db}
}

// Create 创建使用统计记录。
func (d *UsageLogDAO) Create(ctx context.Context, log *model.AIUsageLog) error {
	return d.db.WithContext(ctx).Create(log).Error
}

// GetBySessionID 按 SessionID 查询使用统计列表。
func (d *UsageLogDAO) GetBySessionID(ctx context.Context, sessionID string) ([]model.AIUsageLog, error) {
	var logs []model.AIUsageLog
	err := d.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}

// StatsQuery 统计查询参数。
type StatsQuery struct {
	StartDate time.Time
	EndDate   time.Time
	Scene     string
	UserID    uint64
}

// StatsResult 统计结果。
type StatsResult struct {
	TotalRequests         int64   `json:"total_requests"`
	TotalTokens           int64   `json:"total_tokens"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	TotalCostUSD          float64 `json:"total_cost_usd"`
	AvgFirstTokenMs       float64 `json:"avg_first_token_ms"`
	AvgTokensPerSecond    float64 `json:"avg_tokens_per_second"`
	ApprovalRate          float64 `json:"approval_rate"`
	ApprovalPassRate      float64 `json:"approval_pass_rate"`
	ToolErrorRate         float64 `json:"tool_error_rate"`
}

// GetStats 获取统计数据。
func (d *UsageLogDAO) GetStats(ctx context.Context, query StatsQuery) (*StatsResult, error) {
	var result StatsResult
	q := d.db.WithContext(ctx).Model(&model.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate)

	if query.Scene != "" {
		q = q.Where("scene = ?", query.Scene)
	}
	if query.UserID > 0 {
		q = q.Where("user_id = ?", query.UserID)
	}

	if err := q.Select(
		"COUNT(*) as total_requests",
		"COALESCE(SUM(total_tokens), 0) as total_tokens",
		"COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens",
		"COALESCE(SUM(completion_tokens), 0) as total_completion_tokens",
		"COALESCE(SUM(estimated_cost_usd), 0) as total_cost_usd",
		"COALESCE(AVG(first_token_ms), 0) as avg_first_token_ms",
		"COALESCE(AVG(tokens_per_second), 0) as avg_tokens_per_second",
	).Scan(&result).Error; err != nil {
		return nil, err
	}

	// 审批统计
	var approvalTotal, approvalPassed int64
	approvalQ := d.db.WithContext(ctx).Model(&model.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate)
	approvalQ.Where("approval_count > 0").Count(&approvalTotal)
	approvalQ.Where("approval_status = ?", "approved").Count(&approvalPassed)

	if result.TotalRequests > 0 {
		result.ApprovalRate = float64(approvalTotal) / float64(result.TotalRequests)
	}
	if approvalTotal > 0 {
		result.ApprovalPassRate = float64(approvalPassed) / float64(approvalTotal)
	}

	// 工具错误率
	var toolCalls, toolErrors int64
	d.db.WithContext(ctx).Model(&model.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate).
		Select("COALESCE(SUM(tool_call_count), 0)").Scan(&toolCalls)
	d.db.WithContext(ctx).Model(&model.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate).
		Select("COALESCE(SUM(tool_error_count), 0)").Scan(&toolErrors)
	if toolCalls > 0 {
		result.ToolErrorRate = float64(toolErrors) / float64(toolCalls)
	}

	return &result, nil
}

// SceneStats 场景统计。
type SceneStats struct {
	Scene  string `json:"scene"`
	Count  int64  `json:"count"`
	Tokens int64  `json:"tokens"`
}

// GetByScene 按场景分组统计。
func (d *UsageLogDAO) GetByScene(ctx context.Context, query StatsQuery) ([]SceneStats, error) {
	var result []SceneStats
	q := d.db.WithContext(ctx).Model(&model.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate)
	if query.UserID > 0 {
		q = q.Where("user_id = ?", query.UserID)
	}
	err := q.Select("scene as scene, COUNT(*) as count, COALESCE(SUM(total_tokens), 0) as tokens").
		Group("scene").
		Order("count DESC").
		Find(&result).Error
	return result, err
}

// DateStats 日期统计。
type DateStats struct {
	Date     string `json:"date"`
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
}

// GetByDate 按日期分组统计。
func (d *UsageLogDAO) GetByDate(ctx context.Context, query StatsQuery) ([]DateStats, error) {
	var result []DateStats
	q := d.db.WithContext(ctx).Model(&model.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate)
	if query.Scene != "" {
		q = q.Where("scene = ?", query.Scene)
	}
	if query.UserID > 0 {
		q = q.Where("user_id = ?", query.UserID)
	}
	err := q.Select(
		"DATE(created_at) as date, COUNT(*) as requests, COALESCE(SUM(total_tokens), 0) as tokens",
	).Group("DATE(created_at)").
		Order("date ASC").
		Find(&result).Error
	return result, err
}

// ListQuery 列表查询参数。
type ListQuery struct {
	StartDate time.Time
	EndDate   time.Time
	Scene     string
	Status    string
	UserID    uint64
	Page      int
	PageSize  int
}

// ListResult 列表结果。
type ListResult struct {
	Total int64             `json:"total"`
	Items []model.AIUsageLog `json:"items"`
}

// List 分页查询使用日志列表。
func (d *UsageLogDAO) List(ctx context.Context, query ListQuery) (*ListResult, error) {
	var result ListResult
	q := d.db.WithContext(ctx).Model(&model.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate)

	if query.Scene != "" {
		q = q.Where("scene = ?", query.Scene)
	}
	if query.Status != "" {
		q = q.Where("status = ?", query.Status)
	}
	if query.UserID > 0 {
		q = q.Where("user_id = ?", query.UserID)
	}

	if err := q.Count(&result.Total).Error; err != nil {
		return nil, err
	}

	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	offset := (query.Page - 1) * query.PageSize

	err := q.Order("created_at DESC").
		Limit(query.PageSize).
		Offset(offset).
		Find(&result.Items).Error
	return &result, err
}
```

- [ ] **Step 2: 验证编译通过**

Run: `go build ./...`
Expected: 编译成功，无错误

- [ ] **Step 3: 提交**

```bash
git add internal/dao/ai_usage_log.go
git commit -m "feat(ai): add UsageLogDAO for usage statistics queries"
```

---

## Chunk 2: 数据采集与 Prometheus 指标

### Task 3: 新增首 Token 延迟指标

**Files:**
- Modify: `internal/ai/observability/metrics.go`

- [ ] **Step 1: 添加 firstTokenLatency 字段**

在 `Metrics` 结构体中添加字段（约 line 49-61）：

```go
type Metrics struct {
	toolExecutions      *prometheus.CounterVec
	toolDuration        *prometheus.HistogramVec
	agentExecutions     *prometheus.CounterVec
	agentDuration       *prometheus.HistogramVec
	tokenUsage          *prometheus.CounterVec
	costUsage           *prometheus.CounterVec
	thoughtChains       *prometheus.CounterVec
	thoughtChainLatency *prometheus.HistogramVec
	thoughtChainNodes   *prometheus.CounterVec
	thoughtNodeLatency  *prometheus.HistogramVec
	thoughtApprovals    *prometheus.CounterVec
	approvalWaitLatency *prometheus.HistogramVec
	firstTokenLatency   *prometheus.HistogramVec // 新增
}
```

- [ ] **Step 2: 在 newMetrics 中初始化**

在 `newMetrics` 函数中添加初始化（约 line 146）：

```go
	approvalWaitLatency: registerHistogramVec(prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "opspilot_ai_thoughtchain_approval_wait_seconds",
		Help:    "Wait time for thoughtChain approvals in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"scene", "status"})),
	firstTokenLatency: registerHistogramVec(prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "opspilot_ai_first_token_seconds",
		Help:    "Time to first token in seconds.",
		Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"scene"})),
}
```

- [ ] **Step 3: 添加 ObserveFirstToken 方法**

在 `Metrics` 结构体方法区域添加：

```go
// ObserveFirstToken 记录首 token 延迟。
func (m *Metrics) ObserveFirstToken(scene string, duration time.Duration) {
	if m == nil || duration <= 0 {
		return
	}
	m.firstTokenLatency.WithLabelValues(normalizeLabel(scene)).Observe(duration.Seconds())
}
```

- [ ] **Step 4: 添加全局函数**

```go
// ObserveFirstToken 记录首 token 延迟。
func ObserveFirstToken(scene string, duration time.Duration) {
	DefaultMetrics().ObserveFirstToken(scene, duration)
}
```

- [ ] **Step 5: 验证编译通过**

Run: `go build ./...`
Expected: 编译成功，无错误

- [ ] **Step 6: 提交**

```bash
git add internal/ai/observability/metrics.go
git commit -m "feat(ai): add first_token_seconds Prometheus metric"
```

---

### Task 4: 实现统计采集器

**Files:**
- Modify: `internal/ai/orchestrator.go`

- [ ] **Step 1: 添加 sync 到 import 块**

在 `internal/ai/orchestrator.go` 的 import 块中添加 `"sync"`：

```go
import (
	"context"
	"fmt"
	"io"
	"regexp"  // 新增
	"strings"
	"sync"    // 新增
	"time"

	"github.com/cloudwego/eino/adk"
	// ... 其他 imports
)
```

- [ ] **Step 2: 添加 executionStats 结构体**

在 import 块后添加内部结构体：

```go
// executionStats 累加单次执行的统计数据。
type executionStats struct {
	startedAt     time.Time
	firstTokenAt  time.Time
	firstTokenOnce sync.Once

	promptTokens     int64
	completionTokens int64
	totalTokens      int64

	toolCallCount  int
	toolErrorCount int

	approvalCount  int
	approvalWaitMs int64
}

func newExecutionStats() *executionStats {
	return &executionStats{
		startedAt: time.Now().UTC(),
	}
}

func (s *executionStats) recordTokens(prompt, completion, total int64) {
	s.promptTokens += prompt
	s.completionTokens += completion
	s.totalTokens += total
}

func (s *executionStats) recordFirstToken() {
	s.firstTokenOnce.Do(func() {
		s.firstTokenAt = time.Now().UTC()
	})
}

func (s *executionStats) firstTokenMs() int {
	if s.firstTokenAt.IsZero() {
		return 0
	}
	return int(s.firstTokenAt.Sub(s.startedAt).Milliseconds())
}

func (s *executionStats) tokensPerSecond() float64 {
	if s.firstTokenAt.IsZero() {
		return 0
	}
	elapsed := time.Since(s.firstTokenAt).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(s.completionTokens) / elapsed
}
```

- [ ] **Step 2: 添加 extractTokenUsage 函数**

```go
// extractTokenUsage 从 AgentEvent 提取 token 使用量。
func extractTokenUsage(event *adk.AgentEvent) (prompt, completion, total int64) {
	if event == nil || event.Output == nil || event.Output.MessageOutput == nil {
		return 0, 0, 0
	}
	msg := event.Output.MessageOutput.Message
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return 0, 0, 0
	}
	usage := msg.ResponseMeta.Usage
	return int64(usage.PromptTokens), int64(usage.CompletionTokens), int64(usage.TotalTokens)
}
```

- [ ] **Step 3: 添加 isToolError 函数**

```go
// isToolError 检查工具输出是否包含错误。
func isToolError(event *adk.AgentEvent) bool {
	if event == nil || event.Output == nil || event.Output.MessageOutput == nil {
		return false
	}
	// 检查消息内容是否包含错误标识
	content := strings.TrimSpace(event.Output.MessageOutput.Content())
	if strings.Contains(content, `"error"`) || strings.Contains(content, `"failed"`) {
		return true
	}
	return false
}
```

- [ ] **Step 3: 添加 sanitizeErrorMessage 函数**

在 import 中添加 `"regexp"`，然后添加：

```go
// sanitizeErrorMessage 脱敏错误信息。
func sanitizeErrorMessage(msg string) string {
	if len(msg) > 500 {
		msg = msg[:500]
	}
	msg = regexp.MustCompile(`(?i)(api[_-]?key|token|secret)[\s:=]*[\w-]{20,}`).ReplaceAllString(msg, "[REDACTED]")
	msg = regexp.MustCompile(`/[\w/.-]+`).ReplaceAllString(msg, "[PATH]")
	return msg
}
```

- [ ] **Step 4: 验证编译通过**

Run: `go build ./...`
Expected: 编译成功，无错误

- [ ] **Step 5: 提交**

```bash
git add internal/ai/orchestrator.go
git commit -m "feat(ai): add executionStats accumulator and token extraction"
```

---

### Task 5: 集成统计采集到 streamExecution

**Files:**
- Modify: `internal/ai/orchestrator.go`

- [ ] **Step 1: 在 PlatformDeps 中添加 UsageLogDAO 字段**

修改 `internal/ai/tools/common/common.go`，在 `PlatformDeps` 结构体中添加：

```go
// PlatformDeps 携带平台级依赖。
type PlatformDeps struct {
	DB          *gorm.DB         // 数据库连接
	Prometheus  prominfra.Client // Prometheus HTTP API 客户端
	UsageLogDAO UsageLogDAOInterface // 使用统计 DAO（可选）
}

// UsageLogDAOInterface 定义 UsageLogDAO 的接口，便于测试时 mock。
type UsageLogDAOInterface interface {
	Create(ctx context.Context, log *model.AIUsageLog) error
}
```

添加 import：
```go
import "github.com/cy77cc/OpsPilot/internal/model"
```

- [ ] **Step 2: 修改 Orchestrator 结构体使用 deps.UsageLogDAO**

在 `Orchestrator` 结构体中，通过 `deps.UsageLogDAO` 访问：

```go
type Orchestrator struct {
	runner      *adk.Runner
	checkpoints *airuntime.CheckpointStore
	executions  *airuntime.ExecutionStore
	converter   *airuntime.SSEConverter
	approvals   *airuntime.ApprovalDecisionMaker
	summaries   *approvaltools.SummaryRenderer
	usageLogDAO UsageLogDAOInterface // 从 deps 获取
	initErr     error
	runQuery    func(context.Context, string, ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent]
}
```

在文件开头添加 import：
```go
import "github.com/cy77cc/OpsPilot/internal/model"
```

- [ ] **Step 3: 修改 NewOrchestrator 从 deps 获取 usageLogDAO**

修改 `NewOrchestrator` 函数实现（签名不变）：

```go
func NewOrchestrator(_ any, executionStore *airuntime.ExecutionStore, deps common.PlatformDeps) *Orchestrator {
	// ... 现有初始化代码 ...

	agent, err := agents.NewAgent(ctx, agents.Deps{
		PlatformDeps:  deps,
		DecisionMaker: approvals,
	})
	if err != nil {
		return &Orchestrator{
			executions:    executionStore,
			checkpoints:   checkpointStore,
			converter:     airuntime.NewSSEConverter(),
			approvals:     approvals,
			summaries:     summaries,
			usageLogDAO:   deps.UsageLogDAO, // 新增
			initErr:       err,
		}
	}

	return &Orchestrator{
		runner: adk.NewRunner(ctx, adk.RunnerConfig{
			Agent:           agent,
			CheckPointStore: checkpointStore,
			EnableStreaming: true,
		}),
		checkpoints:   checkpointStore,
		executions:    executionStore,
		converter:     airuntime.NewSSEConverter(),
		approvals:     approvals,
		summaries:     summaries,
		usageLogDAO:   deps.UsageLogDAO, // 新增
		runQuery:      nil,
	}
}
```

	// 在返回时添加 usageLogDAO
	return &Orchestrator{
		runner:        adk.NewRunner(ctx, adk.RunnerConfig{...}),
		checkpoints:   checkpointStore,
		executions:    executionStore,
		converter:     airuntime.NewSSEConverter(),
		approvals:     approvals,
		summaries:     summaries,
		usageLogDAO:   usageLogDAO, // 新增
		runQuery:      nil,
	}
}
```

- [ ] **Step 3: 在 streamExecution 中采集统计数据**

在 `streamExecution` 方法的 for 循环中添加统计采集：

```go
func (o *Orchestrator) streamExecution(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent], state *airuntime.ExecutionState, emit airuntime.StreamEmitter) (*airuntime.ResumeResult, error) {
	// 初始化统计累加器
	stats := newExecutionStats()

	// ... 现有代码 ...

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}

		// 提取 token 使用量（新增）
		prompt, completion, total := extractTokenUsage(event)
		if total > 0 {
			stats.recordTokens(prompt, completion, total)
		}

		if event.Err != nil {
			// 写入失败记录（新增）
			o.writeUsageLog(ctx, state, stats, "failed", "model_error", event.Err)
			// ... 现有错误处理代码 ...
		}

		// ... 现有文本处理代码 ...
		for _, content := range contents {
			if strings.TrimSpace(content) != "" {
				stats.recordFirstToken() // 新增
			}
			// ... 发送 delta ...
		}

		if isToolOutputEvent(event) {
			stats.toolCallCount++ // 新增
			// ... 现有工具输出处理 ...
		}

		if event.Action != nil && event.Action.Interrupted != nil {
			stats.approvalCount++ // 新增
			return o.handleInterrupt(ctx, event, state, emit, stats, chainStartedAt)
		}
	}

	// 循环结束，写入成功记录（新增）
	o.writeUsageLog(ctx, state, stats, "completed", "", nil)

	// ... 现有完成处理代码 ...
}
```

- [ ] **Step 4: 添加 writeUsageLog 方法**

```go
// writeUsageLog 写入使用统计记录。
func (o *Orchestrator) writeUsageLog(ctx context.Context, state *airuntime.ExecutionState, stats *executionStats, status, errorType string, err error) {
	if o.usageLogDAO == nil {
		return
	}

	log := &model.AIUsageLog{
		TraceID:           state.TraceID,
		SessionID:         state.SessionID,
		PlanID:            state.PlanID,
		TurnID:            state.TurnID,
		UserID:            0, // TODO: 从 context 获取
		Scene:             state.Scene,
		Operation:         "run",
		Status:            status,
		PromptTokens:      int(stats.promptTokens),
		CompletionTokens:  int(stats.completionTokens),
		TotalTokens:       int(stats.totalTokens),
		DurationMs:        int(time.Since(stats.startedAt).Milliseconds()),
		FirstTokenMs:      stats.firstTokenMs(),
		TokensPerSecond:   stats.tokensPerSecond(),
		ApprovalCount:     stats.approvalCount,
		ApprovalStatus:    "none",
		ToolCallCount:     stats.toolCallCount,
		ToolErrorCount:    stats.toolErrorCount,
		ErrorType:         errorType,
	}

	if err != nil {
		log.ErrorMessage = sanitizeErrorMessage(err.Error())
	}

	if err := o.usageLogDAO.Create(ctx, log); err != nil {
		// 记录日志但不影响主流程
		if l := logger.L(); l != nil {
			l.Warn("failed to write usage log", logger.Error(err))
		}
	}

	// 上报 Prometheus 指标
	if stats.firstTokenMs() > 0 {
		aiobs.ObserveFirstToken(state.Scene, time.Duration(stats.firstTokenMs())*time.Millisecond)
	}
}
```

需要添加 import：
```go
import "github.com/cy77cc/OpsPilot/internal/logger"
```

- [ ] **Step 5: 验证编译通过**

Run: `go build ./...`
Expected: 编译成功，无错误

- [ ] **Step 6: 提交**

```bash
git add internal/ai/orchestrator.go
git commit -m "feat(ai): integrate usage statistics collection into streamExecution"
```

---

## Chunk 3: API 接口

### Task 6: 创建统计 Handler

**Files:**
- Create: `internal/service/ai/usage_handler.go`

- [ ] **Step 1: 创建 usage_handler.go**

```go
// internal/service/ai/usage_handler.go

// Package ai 提供 AI 编排服务的 HTTP 接口。
package ai

import (
	"fmt"
	"time"

	"github.com/cy77cc/OpsPilot/internal/dao"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/gin-gonic/gin"
)

// GetUsageStats 获取使用统计概览。
func (h *HTTPHandler) GetUsageStats(c *gin.Context) {
	if h.usageLogDAO == nil {
		httpx.Fail(c, xcode.ServerError, "usage statistics not available")
		return
	}

	startDate, endDate := parseDateRange(c)
	userID := httpx.UIDFromCtx(c)
	isAdmin := httpx.IsAdmin(h.svcCtx.DB, userID)

	query := dao.StatsQuery{
		StartDate: startDate,
		EndDate:   endDate,
		Scene:     c.Query("scene"),
	}

	if !isAdmin {
		query.UserID = userID
	}

	stats, err := h.usageLogDAO.GetStats(c.Request.Context(), query)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	byScene, err := h.usageLogDAO.GetByScene(c.Request.Context(), query)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	byDate, err := h.usageLogDAO.GetByDate(c.Request.Context(), query)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{
		"total_requests":           stats.TotalRequests,
		"total_tokens":             stats.TotalTokens,
		"total_prompt_tokens":      stats.TotalPromptTokens,
		"total_completion_tokens":  stats.TotalCompletionTokens,
		"total_cost_usd":           stats.TotalCostUSD,
		"avg_first_token_ms":       stats.AvgFirstTokenMs,
		"avg_tokens_per_second":    stats.AvgTokensPerSecond,
		"approval_rate":            stats.ApprovalRate,
		"approval_pass_rate":       stats.ApprovalPassRate,
		"tool_error_rate":          stats.ToolErrorRate,
		"by_scene":                 byScene,
		"by_date":                  byDate,
	})
}

// GetUsageLogs 获取使用日志列表。
func (h *HTTPHandler) GetUsageLogs(c *gin.Context) {
	if h.usageLogDAO == nil {
		httpx.Fail(c, xcode.ServerError, "usage statistics not available")
		return
	}

	startDate, endDate := parseDateRange(c)
	userID := httpx.UIDFromCtx(c)
	isAdmin := httpx.IsAdmin(h.svcCtx.DB, userID)

	query := dao.ListQuery{
		StartDate: startDate,
		EndDate:   endDate,
		Scene:     c.Query("scene"),
		Status:    c.Query("status"),
		Page:      parseInt(c.Query("page"), 1),
		PageSize:  parseInt(c.Query("page_size"), 20),
	}

	if !isAdmin {
		query.UserID = userID
	}

	result, err := h.usageLogDAO.List(c.Request.Context(), query)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, result)
}

// parseDateRange 解析日期范围参数。
func parseDateRange(c *gin.Context) (start, end time.Time) {
	now := time.Now()

	startDate := c.Query("start_date")
	if startDate != "" {
		start, _ = time.Parse("2006-01-02", startDate)
	} else {
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	}

	endDate := c.Query("end_date")
	if endDate != "" {
		end, _ = time.Parse("2006-01-02", endDate)
	} else {
		end = start.AddDate(0, 0, 1)
	}

	return start, end
}

func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	var v int
	_, _ = fmt.Sscanf(s, "%d", &v)
	if v <= 0 {
		return defaultValue
	}
	return v
}
```

- [ ] **Step 2: 修改 HTTPHandler 结构体添加 usageLogDAO**

在 `internal/service/ai/handler.go` 中修改：

```go
type HTTPHandler struct {
	svcCtx       *svc.ServiceContext
	sessions     *aistate.SessionState
	chatStore    *aistate.ChatStore
	orchestrator aiRuntime
	registry     *aitools.Registry
	approvals    *runtime.ApprovalDecisionMaker
	summaries    *approvaltools.SummaryRenderer
	hintResolver *HintResolver
	usageLogDAO  *dao.UsageLogDAO // 新增
}
```

添加 import：
```go
import "github.com/cy77cc/OpsPilot/internal/dao"
```

- [ ] **Step 3: 修改 NewHTTPHandler 初始化 usageLogDAO**

```go
func NewHTTPHandler(svcCtx *svc.ServiceContext) *HTTPHandler {
	sessionState := aistate.NewSessionState(svcCtx.Rdb, "ai:session:")
	executionStore := runtime.NewExecutionStore(svcCtx.Rdb, "ai:execution:")
	registry := aitools.NewRegistry(common.PlatformDeps{
		DB:         svcCtx.DB,
		Prometheus: svcCtx.Prometheus,
	})

	// 创建 usageLogDAO 一次，复用于 orchestrator 和 handler
	usageLogDAO := dao.NewUsageLogDAO(svcCtx.DB)

	handler := &HTTPHandler{
		svcCtx:    svcCtx,
		sessions:  sessionState,
		chatStore: aistate.NewChatStore(svcCtx.DB),
		orchestrator: coreai.NewOrchestrator(sessionState, executionStore, common.PlatformDeps{
			DB:          svcCtx.DB,
			Prometheus:  svcCtx.Prometheus,
			UsageLogDAO: usageLogDAO, // 通过 PlatformDeps 传递
		}),
		registry:     registry,
		summaries:    approvaltools.NewSummaryRenderer(),
		hintResolver: NewHintResolver(common.PlatformDeps{DB: svcCtx.DB, Prometheus: svcCtx.Prometheus}),
		usageLogDAO:  usageLogDAO, // 复用同一个实例
	}
	// ... 现有 approvals 初始化代码 ...
	return handler
}
```

需要添加 import：
```go
import "github.com/cy77cc/OpsPilot/internal/dao"
```

- [ ] **Step 4: 验证编译通过**

Run: `go build ./...`
Expected: 编译成功，无错误

- [ ] **Step 5: 提交**

```bash
git add internal/service/ai/usage_handler.go internal/service/ai/handler.go
git commit -m "feat(ai): add usage statistics HTTP handlers"
```

---

### Task 7: 注册统计路由

**Files:**
- Modify: `internal/service/ai/routes.go`

- [ ] **Step 1: 添加统计路由**

在 `registerHandlers` 函数中添加：

```go
func registerHandlers(g *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := NewHTTPHandler(svcCtx)
	g.Use(h.SceneContextMiddleware())
	// ... 现有路由 ...
	g.DELETE("/sessions/:id", h.DeleteSession)

	// 使用统计接口
	g.GET("/usage/stats", h.GetUsageStats)
	g.GET("/usage/logs", h.GetUsageLogs)
}
```

- [ ] **Step 2: 验证编译通过**

Run: `go build ./...`
Expected: 编译成功，无错误

- [ ] **Step 3: 提交**

```bash
git add internal/service/ai/routes.go
git commit -m "feat(ai): register usage statistics routes"
```

---

## Chunk 4: 前端页面

### Task 8: 创建 API 客户端

**Files:**
- Create: `web/src/api/modules/ai/usage.ts`

- [ ] **Step 1: 创建 API 客户端**

```typescript
// web/src/api/modules/ai/usage.ts

import { request } from '@/api/request';

export interface UsageStats {
  total_requests: number;
  total_tokens: number;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  total_cost_usd: number;
  avg_first_token_ms: number;
  avg_tokens_per_second: number;
  approval_rate: number;
  approval_pass_rate: number;
  tool_error_rate: number;
  by_scene: SceneStats[];
  by_date: DateStats[];
}

export interface SceneStats {
  scene: string;
  count: number;
  tokens: number;
}

export interface DateStats {
  date: string;
  requests: number;
  tokens: number;
}

export interface UsageLog {
  id: number;
  trace_id: string;
  session_id: string;
  scene: string;
  status: string;
  total_tokens: number;
  duration_ms: number;
  created_at: string;
}

export interface UsageLogsResult {
  total: number;
  items: UsageLog[];
}

export interface UsageStatsParams {
  start_date?: string;
  end_date?: string;
  scene?: string;
}

export interface UsageLogsParams extends UsageStatsParams {
  status?: string;
  page?: number;
  page_size?: number;
}

// 获取使用统计概览
export async function getUsageStats(params?: UsageStatsParams): Promise<UsageStats> {
  const response = await request.get('/api/v1/ai/usage/stats', { params });
  return response.data.data;
}

// 获取使用日志列表
export async function getUsageLogs(params?: UsageLogsParams): Promise<UsageLogsResult> {
  const response = await request.get('/api/v1/ai/usage/logs', { params });
  return response.data.data;
}
```

- [ ] **Step 2: 验证前端编译**

Run: `cd web && npm run build`
Expected: 编译成功，无错误

- [ ] **Step 3: 提交**

```bash
git add web/src/api/modules/ai/usage.ts
git commit -m "feat(web): add AI usage statistics API client"
```

---

### Task 9: 创建统计页面组件

**Files:**
- Create: `web/src/pages/AI/Usage/index.tsx`
- Create: `web/src/pages/AI/Usage/components/StatCards.tsx`
- Create: `web/src/pages/AI/Usage/components/UsageTrendChart.tsx`
- Create: `web/src/pages/AI/Usage/components/ScenePieChart.tsx`
- Create: `web/src/pages/AI/Usage/components/ApprovalChart.tsx`
- Create: `web/src/pages/AI/Usage/components/UsageTable.tsx`
- Create: `web/src/pages/AI/Usage/hooks/useUsageStats.ts`

- [ ] **Step 1: 创建 useUsageStats hook**

```typescript
// web/src/pages/AI/Usage/hooks/useUsageStats.ts

import { useRequest } from 'ahooks';
import { getUsageStats, getUsageLogs, UsageStats, UsageLogsResult, UsageStatsParams, UsageLogsParams } from '@/api/modules/ai/usage';
import { useState } from 'react';

export function useUsageStats(defaultParams?: UsageStatsParams) {
  const [params, setParams] = useState<UsageStatsParams>(defaultParams || {});

  const { data, loading, error, refresh } = useRequest(
    () => getUsageStats(params),
    { refreshDeps: [params] }
  );

  return {
    stats: data,
    loading,
    error,
    params,
    setParams,
    refresh,
  };
}

export function useUsageLogs(defaultParams?: UsageLogsParams) {
  const [params, setParams] = useState<UsageLogsParams>(defaultParams || { page: 1, page_size: 20 });

  const { data, loading, error, refresh } = useRequest(
    () => getUsageLogs(params),
    { refreshDeps: [params] }
  );

  return {
    result: data,
    loading,
    error,
    params,
    setParams,
    refresh,
  };
}
```

- [ ] **Step 2: 创建 StatCards 组件**

```typescript
// web/src/pages/AI/Usage/components/StatCards.tsx

import { Card, Col, Row, Statistic } from 'antd';
import { TransactionOutlined, ClockCircleOutlined, ApiOutlined, DollarOutlined } from '@ant-design/icons';
import { UsageStats } from '@/api/modules/ai/usage';

interface StatCardsProps {
  stats?: UsageStats;
  loading: boolean;
}

export default function StatCards({ stats, loading }: StatCardsProps) {
  return (
    <Row gutter={16}>
      <Col span={6}>
        <Card loading={loading}>
          <Statistic
            title="总请求数"
            value={stats?.total_requests || 0}
            prefix={<ApiOutlined />}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card loading={loading}>
          <Statistic
            title="总 Token"
            value={stats?.total_tokens || 0}
            prefix={<TransactionOutlined />}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card loading={loading}>
          <Statistic
            title="总费用"
            value={stats?.total_cost_usd || 0}
            precision={2}
            prefix={<DollarOutlined />}
            suffix="USD"
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card loading={loading}>
          <Statistic
            title="平均延迟"
            value={stats?.avg_first_token_ms || 0}
            precision={0}
            prefix={<ClockCircleOutlined />}
            suffix="ms"
          />
        </Card>
      </Col>
    </Row>
  );
}
```

- [ ] **Step 3: 创建 UsageTrendChart 组件**

```typescript
// web/src/pages/AI/Usage/components/UsageTrendChart.tsx

import { Card } from 'antd';
import { Line } from '@ant-design/charts';
import { DateStats } from '@/api/modules/ai/usage';

interface UsageTrendChartProps {
  data?: DateStats[];
  loading: boolean;
}

export default function UsageTrendChart({ data, loading }: UsageTrendChartProps) {
  const config = {
    data: data || [],
    xField: 'date',
    yField: 'tokens',
    smooth: true,
    point: {
      size: 4,
      shape: 'circle',
    },
    tooltip: {
      fields: ['date', 'tokens', 'requests'],
    },
  };

  return (
    <Card title="使用趋势" loading={loading} style={{ marginTop: 16 }}>
      <Line {...config} />
    </Card>
  );
}
```

- [ ] **Step 4: 创建 ScenePieChart 组件**

```typescript
// web/src/pages/AI/Usage/components/ScenePieChart.tsx

import { Card } from 'antd';
import { Pie } from '@ant-design/charts';
import { SceneStats } from '@/api/modules/ai/usage';

interface ScenePieChartProps {
  data?: SceneStats[];
  loading: boolean;
}

export default function ScenePieChart({ data, loading }: ScenePieChartProps) {
  const config = {
    data: data || [],
    angleField: 'count',
    colorField: 'scene',
    radius: 0.8,
    label: {
      type: 'outer',
      content: '{name} {percentage}',
    },
    legend: {
      position: 'bottom',
    },
  };

  return (
    <Card title="场景分布" loading={loading} style={{ marginTop: 16 }}>
      <Pie {...config} />
    </Card>
  );
}
```

- [ ] **Step 5: 创建 ApprovalChart 组件**

```typescript
// web/src/pages/AI/Usage/components/ApprovalChart.tsx

import { Card, Progress, Row, Col, Typography } from 'antd';
import { UsageStats } from '@/api/modules/ai/usage';

interface ApprovalChartProps {
  stats?: UsageStats;
  loading: boolean;
}

export default function ApprovalChart({ stats, loading }: ApprovalChartProps) {
  const passRate = stats?.approval_pass_rate || 0;

  return (
    <Card title="审批统计" loading={loading} style={{ marginTop: 16 }}>
      <Row gutter={16}>
        <Col span={12}>
          <div style={{ textAlign: 'center' }}>
            <Typography.Text type="secondary">审批通过率</Typography.Text>
            <Progress
              type="circle"
              percent={passRate * 100}
              format={(percent) => `${percent?.toFixed(1)}%`}
            />
          </div>
        </Col>
        <Col span={12}>
          <div style={{ textAlign: 'center' }}>
            <Typography.Text type="secondary">审批触发率</Typography.Text>
            <Progress
              type="circle"
              percent={(stats?.approval_rate || 0) * 100}
              format={(percent) => `${percent?.toFixed(1)}%`}
              strokeColor="#faad14"
            />
          </div>
        </Col>
      </Row>
    </Card>
  );
}
```

- [ ] **Step 6: 创建 UsageTable 组件**

```typescript
// web/src/pages/AI/Usage/components/UsageTable.tsx

import { Table, Tag } from 'antd';
import { UsageLog } from '@/api/modules/ai/usage';
import dayjs from 'dayjs';

interface UsageTableProps {
  data?: UsageLog[];
  total: number;
  loading: boolean;
  page: number;
  pageSize: number;
  onPageChange: (page: number, pageSize: number) => void;
}

const statusColors: Record<string, string> = {
  completed: 'green',
  failed: 'red',
  rejected: 'orange',
  waiting_approval: 'blue',
};

export default function UsageTable({ data, total, loading, page, pageSize, onPageChange }: UsageTableProps) {
  const columns = [
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '场景',
      dataIndex: 'scene',
      key: 'scene',
      width: 120,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (v: string) => <Tag color={statusColors[v] || 'default'}>{v}</Tag>,
    },
    {
      title: 'Tokens',
      dataIndex: 'total_tokens',
      key: 'total_tokens',
      width: 100,
      render: (v: number) => v.toLocaleString(),
    },
    {
      title: '耗时',
      dataIndex: 'duration_ms',
      key: 'duration_ms',
      width: 100,
      render: (v: number) => `${v}ms`,
    },
  ];

  return (
    <Table
      columns={columns}
      dataSource={data}
      rowKey="id"
      loading={loading}
      pagination={{
        current: page,
        pageSize,
        total,
        onChange: onPageChange,
        showSizeChanger: true,
        showTotal: (total) => `共 ${total} 条`,
      }}
    />
  );
}
```

- [ ] **Step 7: 创建页面入口**

```typescript
// web/src/pages/AI/Usage/index.tsx

import { Card, Row, Col, DatePicker, Select, Space } from 'antd';
import { useState } from 'react';
import dayjs from 'dayjs';
import StatCards from './components/StatCards';
import UsageTrendChart from './components/UsageTrendChart';
import ScenePieChart from './components/ScenePieChart';
import ApprovalChart from './components/ApprovalChart';
import UsageTable from './components/UsageTable';
import { useUsageStats, useUsageLogs } from './hooks/useUsageStats';

const { RangePicker } = DatePicker;

export default function AIUsagePage() {
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().startOf('day'),
    dayjs().add(1, 'day').startOf('day'),
  ]);
  const [scene, setScene] = useState<string>();

  const { stats, loading: statsLoading, setParams: setStatsParams } = useUsageStats({
    start_date: dateRange[0].format('YYYY-MM-DD'),
    end_date: dateRange[1].format('YYYY-MM-DD'),
  });

  const { result, loading: logsLoading, params: logsParams, setParams: setLogsParams } = useUsageLogs({
    start_date: dateRange[0].format('YYYY-MM-DD'),
    end_date: dateRange[1].format('YYYY-MM-DD'),
    page: 1,
    page_size: 20,
  });

  const handleDateRangeChange = (dates: [dayjs.Dayjs | null, dayjs.Dayjs | null] | null) => {
    if (dates && dates[0] && dates[1]) {
      setDateRange([dates[0], dates[1]]);
      const params = {
        start_date: dates[0].format('YYYY-MM-DD'),
        end_date: dates[1].format('YYYY-MM-DD'),
      };
      setStatsParams(params);
      setLogsParams({ ...logsParams, ...params, page: 1 });
    }
  };

  const handleSceneChange = (value?: string) => {
    setScene(value);
    setStatsParams({ scene: value });
    setLogsParams({ ...logsParams, scene: value, page: 1 });
  };

  const handlePageChange = (page: number, pageSize: number) => {
    setLogsParams({ ...logsParams, page, page_size: pageSize });
  };

  const sceneOptions = stats?.by_scene?.map((s) => ({ label: s.scene, value: s.scene })) || [];

  return (
    <div style={{ padding: 24 }}>
      <Card>
        <Space style={{ marginBottom: 16 }}>
          <RangePicker
            value={dateRange}
            onChange={handleDateRangeChange}
            format="YYYY-MM-DD"
          />
          <Select
            allowClear
            placeholder="筛选场景"
            style={{ width: 200 }}
            options={sceneOptions}
            value={scene}
            onChange={handleSceneChange}
          />
        </Space>

        <StatCards stats={stats} loading={statsLoading} />
        <UsageTrendChart data={stats?.by_date} loading={statsLoading} />

        <Row gutter={16}>
          <Col span={12}>
            <ScenePieChart data={stats?.by_scene} loading={statsLoading} />
          </Col>
          <Col span={12}>
            <ApprovalChart stats={stats} loading={statsLoading} />
          </Col>
        </Row>

        <Card title="请求列表" style={{ marginTop: 16 }}>
          <UsageTable
            data={result?.items}
            total={result?.total || 0}
            loading={logsLoading}
            page={logsParams.page || 1}
            pageSize={logsParams.page_size || 20}
            onPageChange={handlePageChange}
          />
        </Card>
      </Card>
    </div>
  );
}
```

- [ ] **Step 8: 验证前端编译**

Run: `cd web && npm run build`
Expected: 编译成功，无错误

- [ ] **Step 9: 提交**

```bash
git add web/src/pages/AI/Usage/
git commit -m "feat(web): add AI usage statistics dashboard page"
```

---

### Task 10: 注册前端路由

**Files:**
- Modify: `web/src/routes.tsx` (或对应路由配置文件)

- [ ] **Step 1: 添加路由配置**

在路由配置中添加：

```typescript
import AIUsagePage from '@/pages/AI/Usage';

// 在路由数组中添加
{
  path: '/ai/usage',
  element: <AIUsagePage />,
  meta: { title: 'AI 使用统计' },
}
```

- [ ] **Step 2: 验证前端编译**

Run: `cd web && npm run build`
Expected: 编译成功，无错误

- [ ] **Step 3: 提交**

```bash
git add web/src/routes.tsx
git commit -m "feat(web): register AI usage statistics route"
```

---

## 验证清单

- [ ] 运行后端测试: `make test`
- [ ] 运行前端测试: `make web-test`
- [ ] 启动开发服务器验证页面可访问
- [ ] 验证 Prometheus 指标端点: `curl http://localhost:8080/metrics | grep opspilot_ai`

---

## 清理旧代码

无需要清理的旧代码。此功能为新增模块，不涉及现有功能的废弃。
