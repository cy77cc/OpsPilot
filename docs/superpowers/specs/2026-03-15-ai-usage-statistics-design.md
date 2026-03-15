# AI 模块统计功能设计

## 概述

为 AI 模块添加 token 使用量、审批统计、工具调用统计等指标，支持 Prometheus 监控、数据库持久化和前端仪表盘展示。

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

    -- 执行元信息
    scene VARCHAR(64) COMMENT '场景（k8s、host 等）',
    operation VARCHAR(32) COMMENT '操作类型（run、resume）',
    status VARCHAR(32) COMMENT '执行状态（completed、failed、rejected、waiting_approval）',

    -- Token 统计
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
    error_message TEXT COMMENT '错误摘要',

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

    INDEX idx_session_id (session_id),
    INDEX idx_created_at (created_at),
    INDEX idx_scene (scene),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT 'AI 使用统计日志';
```

### Go Model

```go
// internal/model/ai/usage_log.go

// AIUsageLog 记录单次 AI 执行的统计数据。
type AIUsageLog struct {
    ID     int64  `gorm:"primaryKey;autoIncrement"`
    TraceID string `gorm:"column:trace_id;type:varchar(36);not null;index:idx_session_id"`

    // 关联标识
    SessionID string `gorm:"column:session_id;type:varchar(36);not null;index:idx_session_id"`
    PlanID    string `gorm:"column:plan_id;type:varchar(36);not null"`
    TurnID    string `gorm:"column:turn_id;type:varchar(36)"`

    // 执行元信息
    Scene     string `gorm:"column:scene;type:varchar(64);index:idx_scene"`
    Operation string `gorm:"column:operation;type:varchar(32)"`
    Status    string `gorm:"column:status;type:varchar(32);index:idx_status"`

    // Token 统计
    PromptTokens     int      `gorm:"column:prompt_tokens;default:0"`
    CompletionTokens int      `gorm:"column:completion_tokens;default:0"`
    TotalTokens      int      `gorm:"column:total_tokens;default:0"`
    EstimatedCostUSD float64  `gorm:"column:estimated_cost_usd;type:decimal(10,6)"`
    ModelName        string   `gorm:"column:model_name;type:varchar(128)"`

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

    CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;index:idx_created_at"`
}

func (AIUsageLog) TableName() string {
    return "ai_usage_logs"
}
```

## 数据采集

### 采集位置

在 `internal/ai/orchestrator.go` 的 `streamExecution` 方法中采集数据。

### 数据流

```
AgentEvent
    └─> MessageVariant (Output.MessageOutput)
        └─> Message (schema.Message)
            └─> ResponseMeta
                └─> TokenUsage (PromptTokens, CompletionTokens, TotalTokens)
```

### 采集时机

1. **Token 数据** - 每次收到 `Assistant` 角色的消息时，从 `ResponseMeta.Usage` 提取并累加
2. **首 Token 延迟** - 首次收到文本内容时记录与执行开始时间的差值
3. **审批统计** - 在 `handleInterrupt` 和 `resume` 方法中更新
4. **工具统计** - 统计 `EventToolCall` 和 `EventToolResult` 事件
5. **最终写入** - 执行完成、失败或中断时，写入数据库并上报 Prometheus

### 执行统计累加器

在 `streamExecution` 中使用累加器结构：

```go
// executionStats 累加单次执行的统计数据。
type executionStats struct {
    startedAt       time.Time
    firstTokenAt    time.Time
    firstTokenOnce  sync.Once

    promptTokens     int64
    completionTokens int64
    totalTokens      int64
    modelName        string

    toolCallCount  int
    toolErrorCount int

    approvalCount  int
    approvalWaitMs int64
}
```

### 写入时机

| 场景 | 写入时机 | Status 值 |
|------|----------|-----------|
| 正常完成 | `streamExecution` 循环结束 | `completed` |
| 执行失败 | 收到 `event.Err` | `failed` |
| 等待审批 | `handleInterrupt` | `waiting_approval` |
| 审批拒绝 | `resume` 中 `!req.Approved` | `rejected` |

## API 接口

### 统计概览

```
GET /api/v1/ai/usage/stats
```

请求参数：
- `start_date` - 开始日期（YYYY-MM-DD）
- `end_date` - 结束日期（YYYY-MM-DD）
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

在配置文件中定义模型定价（美元/千 token）：

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

### 费用计算

```go
func calculateCost(modelName string, promptTokens, completionTokens int64, pricing map[string]ModelPricing) float64 {
    p, ok := pricing[modelName]
    if !ok {
        return 0
    }
    promptCost := float64(promptTokens) / 1000 * p.Prompt
    completionCost := float64(completionTokens) / 1000 * p.Completion
    return promptCost + completionCost
}
```

## Prometheus 指标

复用现有指标，补全数据采集：

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `opspilot_ai_tokens_total` | Counter | scope, name, scene, token_type, source | Token 使用量 |
| `opspilot_ai_cost_usd_total` | Counter | scope, name, scene, source | 费用累计 |
| `opspilot_ai_agent_executions_total` | Counter | operation, scene, status | 执行次数 |

新增指标：

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `opspilot_ai_first_token_seconds` | Histogram | scene | 首 Token 延迟 |
| `opspilot_ai_approvals_total` | Counter | scene, status | 审批统计 |

## 实现计划

1. **数据库迁移** - 创建 `ai_usage_logs` 表
2. **Model 层** - 新增 `AIUsageLog` 模型和 DAO
3. **数据采集** - 在 `orchestrator.go` 中实现统计累加和写入
4. **API 层** - 实现统计接口
5. **前端页面** - 实现统计仪表盘
6. **配置** - 添加模型定价配置
