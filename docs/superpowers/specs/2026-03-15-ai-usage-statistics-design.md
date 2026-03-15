# AI 模块统计功能设计

## 概述

为 AI 模块添加 token 使用量、审批统计、工具调用统计等指标，支持 Prometheus 监控、数据库持久化和前端仪表盘展示。

## 与现有系统的关系

### 现有 ai_executions 表

`ai_executions` 表记录**工具执行级别**的详细信息：
- 每次 Tool 调用一条记录
- 包含 `ToolName`、`StepID`、`ParamsJSON`、`ResultJSON`
- 已有 token 字段，用于工具自身报告的 token 使用量

### 新增 ai_usage_logs 表

`ai_usage_logs` 表记录**请求级别**的汇总统计：
- 每次 Run/Resume 一条记录
- 汇总该次请求的所有 LLM 调用 token
- 包含审批统计、性能指标、错误追踪

**为什么需要两个表：**
1. 粒度不同：工具执行 vs 整体请求
2. 数据来源不同：工具自报告 vs LLM ResponseMeta
3. 查询效率：汇总查询无需聚合 step 级别数据

## 目标

1. **Prometheus 指标** - 复用现有 `opspilot_ai_tokens_total` 等指标，补全数据采集
2. **持久化存储** - 每次执行记录详细统计数据，支持时间范围查询
3. **前端展示** - 独立统计页面，展示总量、趋势、分布

## 数据模型

### 数据库表

新增 `ai_usage_logs` 表：

```sql
CREATE TABLE ai_usage_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,

    -- 关联标识
    trace_id VARCHAR(36) NOT NULL COMMENT '追踪 ID',
    session_id VARCHAR(36) NOT NULL COMMENT '会话 ID',
    plan_id VARCHAR(36) NOT NULL COMMENT '计划 ID',
    turn_id VARCHAR(36) COMMENT '轮次 ID',
    user_id BIGINT COMMENT '用户 ID',

    -- 执行元信息
    scene VARCHAR(64) COMMENT '场景（k8s、host 等）',
    operation VARCHAR(32) COMMENT '操作类型（run、resume）',
    status VARCHAR(32) COMMENT '执行状态（completed、failed、rejected、waiting_approval）',

    -- Token 统计（来自 LLM ResponseMeta）
    prompt_tokens INT DEFAULT 0 COMMENT '输入 token 数',
    completion_tokens INT DEFAULT 0 COMMENT '输出 token 数',
    total_tokens INT DEFAULT 0 COMMENT '总 token 数',
    estimated_cost_usd DECIMAL(10,6) COMMENT '预估费用（美元）',
    model_name VARCHAR(128) COMMENT '模型名称',

    -- 性能指标
    duration_ms INT COMMENT '执行耗时（毫秒）',
    first_token_ms INT COMMENT '首 token 延迟（TTFT）',
    tokens_per_second DECIMAL(10,2) COMMENT '生成速度',

    -- 审批统计
    approval_count INT DEFAULT 0 COMMENT '审批触发次数',
    approval_status VARCHAR(32) DEFAULT 'none' COMMENT '审批状态（none/pending/approved/rejected）',
    approval_wait_ms INT DEFAULT 0 COMMENT '审批等待总时长（毫秒）',

    -- 工具统计
    tool_call_count INT DEFAULT 0 COMMENT '工具调用次数',
    tool_error_count INT DEFAULT 0 COMMENT '工具调用失败次数',

    -- 错误追踪
    error_type VARCHAR(64) COMMENT '错误类型（model_error/tool_error/timeout/approval_rejected）',
    error_message TEXT COMMENT '错误摘要（脱敏处理）',

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

    INDEX idx_ai_usage_logs_session_id (session_id),
    INDEX idx_ai_usage_logs_user_created (user_id, created_at),
    INDEX idx_ai_usage_logs_created_at (created_at),
    INDEX idx_ai_usage_logs_scene (scene),
    INDEX idx_ai_usage_logs_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT 'AI 使用统计日志';
```

### Go Model

```go
// internal/model/ai_usage_log.go

package model

import "time"

// AIUsageLog 记录单次 AI 执行的统计数据。
//
// 与 ai_executions 表的区别：
//   - ai_executions: 工具执行级别，每次 Tool 调用一条
//   - ai_usage_logs: 请求级别，每次 Run/Resume 一条汇总
type AIUsageLog struct {
    ID     int64  `gorm:"primaryKey;autoIncrement"`
    TraceID string `gorm:"column:trace_id;type:varchar(36);not null"`

    // 关联标识
    SessionID string `gorm:"column:session_id;type:varchar(36);not null;index:idx_ai_usage_logs_session_id"`
    PlanID    string `gorm:"column:plan_id;type:varchar(36);not null"`
    TurnID    string `gorm:"column:turn_id;type:varchar(36)"`
    UserID    uint64 `gorm:"column:user_id;index:idx_ai_usage_logs_user_created"`

    // 执行元信息
    Scene     string `gorm:"column:scene;type:varchar(64);index:idx_ai_usage_logs_scene"`
    Operation string `gorm:"column:operation;type:varchar(32)"`
    Status    string `gorm:"column:status;type:varchar(32);index:idx_ai_usage_logs_status"`

    // Token 统计
    PromptTokens     int     `gorm:"column:prompt_tokens;default:0"`
    CompletionTokens int     `gorm:"column:completion_tokens;default:0"`
    TotalTokens      int     `gorm:"column:total_tokens;default:0"`
    EstimatedCostUSD float64 `gorm:"column:estimated_cost_usd;type:decimal(10,6)"`
    ModelName        string  `gorm:"column:model_name;type:varchar(128)"`

    // 性能指标
    DurationMs      int     `gorm:"column:duration_ms"`
    FirstTokenMs    int     `gorm:"column:first_token_ms"`
    TokensPerSecond float64 `gorm:"column:tokens_per_second;type:decimal(10,2)"`

    // 审批统计
    ApprovalCount  int    `gorm:"column:approval_count;default:0"`
    ApprovalStatus string `gorm:"column:approval_status;type:varchar(32);default:'none'"`
    ApprovalWaitMs int    `gorm:"column:approval_wait_ms;default:0"`

    // 工具统计
    ToolCallCount  int `gorm:"column:tool_call_count;default:0"`
    ToolErrorCount int `gorm:"column:tool_error_count;default:0"`

    // 错误追踪
    ErrorType    string `gorm:"column:error_type;type:varchar(64)"`
    ErrorMessage string `gorm:"column:error_message;type:text"`

    CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;index:idx_ai_usage_logs_created_at"`
}

func (AIUsageLog) TableName() string {
    return "ai_usage_logs"
}
```

### DAO 层

```go
// internal/dao/ai_usage_log.go

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
    TotalRequests      int64   `json:"total_requests"`
    TotalTokens        int64   `json:"total_tokens"`
    TotalPromptTokens  int64   `json:"total_prompt_tokens"`
    TotalCompletionTokens int64 `json:"total_completion_tokens"`
    TotalCostUSD       float64 `json:"total_cost_usd"`
    AvgFirstTokenMs    float64 `json:"avg_first_token_ms"`
    AvgTokensPerSecond float64 `json:"avg_tokens_per_second"`
    ApprovalRate       float64 `json:"approval_rate"`
    ApprovalPassRate   float64 `json:"approval_pass_rate"`
    ToolErrorRate      float64 `json:"tool_error_rate"`
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

    // 基础统计
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
    Page      int
    PageSize  int
}

// ListResult 列表结果。
type ListResult struct {
    Total int64            `json:"total"`
    Items []model.AIUsageLog  `json:"items"`
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

## 数据采集

### Token 数据提取

在 `internal/ai/orchestrator.go` 的 `streamExecution` 方法中，从 ADK 事件提取 token 使用量：

```go
// extractTokenUsage 从 AgentEvent 提取 token 使用量。
//
// 数据流: AgentEvent → Output.MessageOutput → Message → ResponseMeta → Usage
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

### 执行统计累加器

```go
// executionStats 累加单次执行的统计数据。
type executionStats struct {
    startedAt      time.Time
    firstTokenAt   time.Time
    firstTokenOnce sync.Once

    promptTokens     int64
    completionTokens int64
    totalTokens      int64

    toolCallCount  int
    toolErrorCount int

    approvalCount  int
    approvalWaitMs int64
}

// recordTokens 记录 token 使用量（累加）。
func (s *executionStats) recordTokens(prompt, completion, total int64) {
    s.promptTokens += prompt
    s.completionTokens += completion
    s.totalTokens += total
}

// recordFirstToken 记录首 token 时间（只记录一次）。
func (s *executionStats) recordFirstToken() {
    s.firstTokenOnce.Do(func() {
        s.firstTokenAt = time.Now().UTC()
    })
}

// firstTokenMs 返回首 token 延迟。
func (s *executionStats) firstTokenMs() int {
    if s.firstTokenAt.IsZero() {
        return 0
    }
    return int(s.firstTokenAt.Sub(s.startedAt).Milliseconds())
}

// tokensPerSecond 计算生成速度。
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

### 采集时机

在 `streamExecution` 的事件循环中：

```go
for {
    event, ok := iter.Next()
    if !ok {
        break
    }
    if event == nil {
        continue
    }

    // 提取 token 使用量
    prompt, completion, total := extractTokenUsage(event)
    if total > 0 {
        stats.recordTokens(prompt, completion, total)
    }

    // 处理文本内容
    contents, err := eventTextContents(event)
    if err != nil {
        return nil, fmt.Errorf("failed to parse agent message chunks: %w", err)
    }
    for _, content := range contents {
        if strings.TrimSpace(content) != "" {
            stats.recordFirstToken() // 首次收到文本时记录
        }
        // ... 发送 delta 事件
    }

    // 处理工具调用事件
    if isToolOutputEvent(event) {
        stats.toolCallCount++
        // 检查工具执行是否有错误
        if isToolError(event) {
            stats.toolErrorCount++
        }
    }

    // 处理中断事件
    if event.Action != nil && event.Action.Interrupted != nil {
        stats.approvalCount++
        return o.handleInterrupt(ctx, event, state, emit, stats, chainStartedAt)
    }
}
```

### 写入时机

| 场景 | 写入时机 | Status 值 | ErrorType |
|------|----------|-----------|-----------|
| 正常完成 | `streamExecution` 循环结束 | `completed` | 空 |
| 执行失败 | 收到 `event.Err` | `failed` | `model_error` 或 `tool_error` |
| 等待审批 | `handleInterrupt` | `waiting_approval` | 空 |
| 审批拒绝 | `resume` 中 `!req.Approved` | `rejected` | `approval_rejected` |

### 错误信息脱敏

写入 `error_message` 前进行脱敏处理：

```go
// sanitizeErrorMessage 脱敏错误信息。
//
// 移除可能包含的敏感信息：
//   - API Key
//   - 文件路径
//   - 数据库连接串
func sanitizeErrorMessage(msg string) string {
    // 限制长度
    if len(msg) > 500 {
        msg = msg[:500]
    }
    // 移除 API Key 模式
    msg = regexp.MustCompile(`(?i)(api[_-]?key|token|secret)[\s:=]*[\w-]{20,}`).ReplaceAllString(msg, "[REDACTED]")
    // 移除文件路径
    msg = regexp.MustCompile(`/[\w/.-]+`).ReplaceAllString(msg, "[PATH]")
    return msg
}
```

## API 接口

### 访问控制

所有统计接口需要登录认证，通过 `middleware.JWTAuth()` 验证：
- 普通用户只能查看自己的使用统计
- 管理员可以查看全部统计

### 统计概览

```
GET /api/v1/ai/usage/stats
```

请求参数：
- `start_date` - 开始日期（YYYY-MM-DD），默认今天
- `end_date` - 结束日期（YYYY-MM-DD），默认明天
- `scene` - 场景过滤（可选）

响应示例：
```json
{
  "code": 1000,
  "data": {
    "total_requests": 1234,
    "total_tokens": 567890,
    "total_prompt_tokens": 234567,
    "total_completion_tokens": 333323,
    "total_cost_usd": 12.34,
    "avg_first_token_ms": 523,
    "avg_tokens_per_second": 45.6,
    "approval_rate": 0.15,
    "approval_pass_rate": 0.92,
    "tool_error_rate": 0.03,
    "by_scene": [
      {"scene": "k8s", "count": 500, "tokens": 234567},
      {"scene": "host", "count": 300, "tokens": 123456}
    ],
    "by_date": [
      {"date": "2026-03-15", "requests": 42, "tokens": 12345}
    ]
  }
}
```

### 详情列表

```
GET /api/v1/ai/usage/logs
```

请求参数：
- `start_date` - 开始日期
- `end_date` - 结束日期
- `scene` - 场景过滤
- `status` - 状态过滤
- `page` - 页码
- `page_size` - 每页数量

响应示例：
```json
{
  "code": 1000,
  "data": {
    "total": 100,
    "items": [
      {
        "id": 1,
        "trace_id": "xxx",
        "session_id": "xxx",
        "scene": "k8s",
        "status": "completed",
        "total_tokens": 1234,
        "duration_ms": 5234,
        "created_at": "2026-03-15T10:30:00Z"
      }
    ]
  }
}
```

### 路由注册

在 `internal/service/ai/routes.go` 中添加：

```go
// 统计接口（需要登录）
usage := aiGroup.Group("/usage", middleware.JWTAuth())
{
    usage.GET("/stats", h.GetUsageStats)
    usage.GET("/logs", h.GetUsageLogs)
}
```

### Handler 实现

```go
// internal/service/ai/handler/usage.go

package handler

import (
    "time"

    "github.com/gin-gonic/gin"
    "github.com/cy77cc/OpsPilot/internal/dao/ai_usage_log"
    "github.com/cy77cc/OpsPilot/internal/httpx"
    "github.com/cy77cc/OpsPilot/internal/middleware"
)

// UsageHandler 处理使用统计相关请求。
type UsageHandler struct {
    dao *dao.UsageLogDAO
}

// NewUsageHandler 创建 UsageHandler 实例。
func NewUsageHandler(dao *dao.UsageLogDAO) *UsageHandler {
    return &UsageHandler{dao: dao}
}

// GetUsageStats 获取使用统计概览。
func (h *UsageHandler) GetUsageStats(c *gin.Context) {
    // 解析时间范围
    startDate, endDate := parseDateRange(c)

    // 获取当前用户
    userID := middleware.GetUserID(c)
    isAdmin := middleware.IsAdmin(c)

    // 构建查询
    query := dao.StatsQuery{
        StartDate: startDate,
        EndDate:   endDate,
        Scene:     c.Query("scene"),
    }

    // 非管理员只能查看自己的统计
    if !isAdmin {
        query.UserID = userID
    }

    // 获取统计数据
    stats, err := h.dao.GetStats(c.Request.Context(), query)
    if err != nil {
        httpx.ServerErr(c, err)
        return
    }

    // 获取场景分布
    byScene, err := h.dao.GetByScene(c.Request.Context(), query)
    if err != nil {
        httpx.ServerErr(c, err)
        return
    }

    // 获取日期趋势
    byDate, err := h.dao.GetByDate(c.Request.Context(), query)
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
func (h *UsageHandler) GetUsageLogs(c *gin.Context) {
    // 解析时间范围
    startDate, endDate := parseDateRange(c)

    // 获取当前用户
    userID := middleware.GetUserID(c)
    isAdmin := middleware.IsAdmin(c)

    // 构建查询
    query := dao.ListQuery{
        StartDate: startDate,
        EndDate:   endDate,
        Scene:     c.Query("scene"),
        Status:    c.Query("status"),
        Page:      parseInt(c.Query("page"), 1),
        PageSize:  parseInt(c.Query("page_size"), 20),
    }

    // 非管理员只能查看自己的日志
    if !isAdmin {
        // 注意：ListQuery 需要添加 UserID 字段
        // 这里假设已添加
    }

    // 获取列表
    result, err := h.dao.List(c.Request.Context(), query)
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

## 前端页面

### 页面位置

新增「AI 使用统计」菜单项，路由 `/ai/usage`。

### 页面布局

```
┌─────────────────────────────────────────────────────────────┐
│  AI 使用统计                              [今日 ▼] [导出]   │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ 总请求数 │  │ 总Token  │  │ 总费用   │  │ 平均延迟 │    │
│  │  1,234   │  │ 567,890  │  │ $12.34   │  │  523ms   │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   使用趋势图                          │   │
│  │          （按日/小时折线图）                          │   │
│  └─────────────────────────────────────────────────────┘   │
├────────────────────────┬────────────────────────────────────┤
│   场景分布（饼图）      │        审批统计（环形图）          │
│                        │                                    │
│    k8s: 40%            │   通过: 92%                        │
│    host: 25%           │   拒绝: 8%                         │
│    service: 20%        │   审批率: 15%                      │
│    other: 15%          │                                    │
├────────────────────────┴────────────────────────────────────┤
│  最近请求列表                                               │
│  ┌─────┬──────┬────────┬────────┬────────┬──────────────┐  │
│  │ 时间 │ 场景 │ 状态   │ Tokens │ 耗时   │ 审批         │  │
│  ├─────┼──────┼────────┼────────┼────────┼──────────────┤  │
│  │ ... │ ...  │ ...    │ ...    │ ...    │ ...          │  │
│  └─────┴──────┴────────┴────────┴────────┴──────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 组件结构

```
web/src/pages/AI/Usage/
├── index.tsx              # 页面入口
├── components/
│   ├── StatCards.tsx      # 总量卡片
│   ├── UsageTrendChart.tsx # 趋势图
│   ├── ScenePieChart.tsx  # 场景分布
│   ├── ApprovalChart.tsx  # 审批统计
│   └── UsageTable.tsx     # 请求列表
└── hooks/
    └── useUsageStats.ts   # 数据获取 hook
```

## 成本计算

### 模型定价配置

在配置文件 `configs/config.yaml` 中定义模型定价（美元/千 token）：

```yaml
ai:
  pricing:
    qwen-plus:
      prompt: 0.0004
      completion: 0.0012
    qwen-turbo:
      prompt: 0.0002
      completion: 0.0006
    gpt-4o:
      prompt: 0.0025
      completion: 0.01
```

### 费用计算服务

```go
// internal/service/ai/pricing.go

// PricingService 计算模型调用费用。
type PricingService struct {
    pricing map[string]ModelPricing
}

type ModelPricing struct {
    Prompt     float64 `yaml:"prompt"`     // 美元/千 token
    Completion float64 `yaml:"completion"` // 美元/千 token
}

// CalculateCost 计算 token 使用费用。
func (s *PricingService) CalculateCost(modelName string, promptTokens, completionTokens int64) float64 {
    p, ok := s.pricing[modelName]
    if !ok {
        return 0
    }
    promptCost := float64(promptTokens) / 1000 * p.Prompt
    completionCost := float64(completionTokens) / 1000 * p.Completion
    return promptCost + completionCost
}
```

### 模型名称获取

模型名称从配置中获取，存储在 `ExecutionState` 或全局配置中：

```go
// 在 orchestrator 中获取模型名称
func (o *Orchestrator) getModelName() string {
    // 从配置或 chatmodel 包获取当前使用的模型名称
    return chatmodel.CurrentModelName()
}
```

## Prometheus 指标

复用现有指标，补全数据采集：

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `opspilot_ai_tokens_total` | Counter | scope, name, scene, token_type, source | Token 使用量 |
| `opspilot_ai_cost_usd_total` | Counter | scope, name, scene, source | 费用累计 |
| `opspilot_ai_agent_executions_total` | Counter | operation, scene, status | 执行次数 |
| `opspilot_ai_thoughtchain_approvals_total` | Counter | scene, status | 审批统计（已有） |

新增指标：

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `opspilot_ai_first_token_seconds` | Histogram | scene | 首 Token 延迟 |

> **注意**：审批统计复用现有的 `opspilot_ai_thoughtchain_approvals_total`，无需新增。

新增指标实现：

```go
// internal/ai/observability/metrics.go

// 在 Metrics 结构体中添加
firstTokenLatency *prometheus.HistogramVec

// 在 newMetrics 中初始化
firstTokenLatency: registerHistogramVec(prometheus.NewHistogramVec(prometheus.HistogramOpts{
    Name:    "opspilot_ai_first_token_seconds",
    Help:    "Time to first token in seconds.",
    Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
}, []string{"scene"})),
```

## 数据库迁移

使用 GORM AutoMigrate 进行迁移，在 `storage/migration/dev_auto.go` 中添加：

```go
// 在 AutoMigrate 的 models 列表中添加
&model.AIUsageLog{},
```

## 实现计划

1. **Model 层** - 新增 `AIUsageLog` 模型
2. **DAO 层** - 新增 `UsageLogDAO`
3. **数据采集** - 在 `orchestrator.go` 中实现统计累加和写入
4. **配置** - 添加模型定价配置
5. **API 层** - 实现统计接口和路由注册
6. **前端页面** - 实现统计仪表盘
