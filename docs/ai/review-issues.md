# AI 文档综合审查报告

> **审查范围**：`docs/ai/roadmap.md`、`docs/ai/phase1-phase2-technical-design.md`、`docs/ai/phase3-phase4-technical-design.md`
> **对照依据**：项目实际代码（`internal/model/`、`internal/ai/`、`storage/migrations/`、`web/src/`）
> **问题分级**：🔴 致命（会导致编译/运行失败）| 🟠 严重（逻辑错误/数据不一致）| 🟡 中等（文档矛盾/设计缺口）| 🔵 轻微（规范不统一/易修复）

---

## 目录

- [一、致命问题（共 6 项）](#一致命问题共-6-项)
- [二、严重问题（共 6 项）](#二严重问题共-6-项)
- [三、中等问题（共 8 项）](#三中等问题共-8-项)
- [四、轻微问题（共 5 项）](#四轻微问题共-5-项)
- [五、问题汇总一览表](#五问题汇总一览表)
- [六、修正建议优先级](#六修正建议优先级)

---

## 一、致命问题（共 6 项）

### 🔴 P1-1：所有模型 ID 类型与文档接口定义不一致

**所在文档**：`phase1-phase2-technical-design.md`（贯穿全文）

**问题描述**：

项目实际模型中，所有 AI 相关表的主键均为 `string` 类型（UUID）：

```go
// internal/model/ai_chat.go
type AIChatSession struct {
    ID string `gorm:"column:id;type:varchar(64);primaryKey"`  // UUID string
    ...
}

// internal/model/ai_approval_task.go
type AIApprovalTask struct {
    ID string `gorm:"column:id;type:varchar(64);primaryKey"`  // UUID string
    ...
}

// internal/model/ai_chat.go
type AIChatMessage struct {
    ID        string `gorm:"column:id;type:varchar(64);primaryKey"`  // UUID string
    SessionID string `gorm:"column:session_id;type:varchar(64)"`     // UUID string
    ...
}
```

但 Phase 1 & Phase 2 文档中的 API 响应体和 TypeScript 类型定义全部使用 `number`（int）：

```typescript
// 文档中（错误）
export interface AISession {
  id: number;       // ❌ 实际是 string (UUID)
  ...
}

export interface ApprovalItem {
  id: number;       // ❌ 实际是 string (UUID)
  session_id: number; // ❌ 实际是 string (UUID)
  ...
}
```

**影响范围**：Phase 1 全部 7 个 API 接口的响应、Phase 2 全部 5 个审批接口、`useSSEChat` Hook 中的 `sessionId: number` 参数、`InlineApprovalCard` 的 `approval_id: number`。

**修正方案**：
- 将所有文档中 `id: number` 改为 `id: string`
- `session_id: number` 改为 `session_id: string`
- `approval_id: number` 改为 `approval_id: string`
- Go 结构体中 `SessionID int64` 改为 `SessionID string`
- API 路径参数（如 `GET /api/v1/ai/sessions/:id`）的文档说明补充"UUID 字符串格式"

---

### 🔴 P1-2：`AIChatSession` 模型缺少 `cluster_name` 字段，但文档大量使用

**所在文档**：`phase1-phase2-technical-design.md`（1.2.1、1.2.5 节）

**问题描述**：

实际 `AIChatSession` 模型只有以下字段：

```go
type AIChatSession struct {
    ID        string    // UUID
    UserID    uint64
    Scene     string    // 场景标识（如: host, cluster, service, k8s）
    Title     string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

但 Phase 1 文档在多处使用了不存在的 `cluster_name` 字段：

- `CreateSessionParams` 包含 `cluster_name`
- 接口响应体中返回 `cluster_name: "prod-cluster"`
- 聊天请求体中携带 `cluster_name`
- TypeScript `AISession` 接口有 `cluster_name: string`

**影响**：如果直接按文档实现，GORM 映射会静默忽略此字段，`cluster_name` 无法持久化，前端传入的集群上下文会丢失。

**修正方案**：

有两种处理方式：

**方案 A（推荐）**：利用已有 `Scene` 字段存储集群标识，或在 `Scene` 字段语义上扩展，同时在消息的 `MetadataJSON` 中存储 `cluster_name`，无需加列。

**方案 B**：在下一个 migration 中给 `ai_chat_sessions` 加列：

```sql
ALTER TABLE ai_chat_sessions
    ADD COLUMN cluster_name VARCHAR(128) NOT NULL DEFAULT '' COMMENT '关联集群名称' AFTER scene;
```

文档必须选定一种方案，并统一描述，不能在没有 migration 的情况下直接使用该字段。

---

### 🔴 P1-3：Phase 1 Migration 文件路径错误

**所在文档**：`phase1-phase2-technical-design.md`（1.2.7 节）

**问题描述**：

Phase 1 文档将 migration 文件写在：

```
storage/migration/20250101_add_ai_indexes.sql
storage/migration/20250101_add_diagnosis_report.sql
```

但项目实际的 migration SQL 文件目录是 `storage/migrations/`（有 `s`）：

```
storage/migrations/20260224_000001_create_ai_chat_tables.sql
storage/migrations/20260312_000035_ai_chat_turn_blocks.sql
...（共 38 个文件）
```

`storage/migration/`（无 `s`）目录只包含 Go 代码（`dev_auto.go`、`runner.go`），不是 SQL 文件存放位置。将 SQL 放入此目录不会被 migration runner 识别，迁移将静默跳过。

**此外，命名规范也不符合**：现有文件格式为 `YYYYMMDD_######_description.sql`（含 6 位序号），但文档使用 `YYYYMMDD_description.sql`（无序号），且年份使用 `2025` 而非现有文件的 `2026` 系列，可能影响排序执行。

**修正方案**：
- 路径统一改为 `storage/migrations/`
- 文件命名遵循 `YYYYMMDD_######_description.sql` 规范，序号接续当前最大值（`000038` 之后）
- 日期与项目实际开发时间对齐

---

### 🔴 P1-4：Phase 1 Migration 中索引字段 `message_id` 不存在于 `ai_chat_blocks` 表

**所在文档**：`phase1-phase2-technical-design.md`（1.2.7 节）

**问题描述**：

Phase 1 migration 包含：

```sql
ALTER TABLE ai_chat_blocks ADD INDEX idx_message_id (message_id);
```

但实际 `AIChatBlock` 模型的外键字段是 `TurnID`，不是 `MessageID`：

```go
type AIChatBlock struct {
    ID        string
    TurnID    string  `gorm:"column:turn_id;index:idx_ai_block_turn_position"`  // ✅ 实际字段
    // 没有 message_id 字段
    ...
}
```

查阅 `storage/migrations/20260312_000035_ai_chat_turn_blocks.sql` 也可证实 `ai_chat_blocks` 表没有 `message_id` 列。该 migration 语句会因字段不存在而**执行失败**，阻断整个 migration 流程。

**修正方案**：删除该错误索引语句。如需加速 block 查询，正确写法是：

```sql
-- ai_chat_blocks 已有 (turn_id, position) 联合索引，查询已覆盖
-- 若需按 session 查询所有 blocks，可通过 turns 关联，无需在 blocks 上加 session 索引
```

---

### 🔴 P1-5：文档 `run_id` 概念与实际 `AIExecution` 模型主键不对应

**所在文档**：`phase1-phase2-technical-design.md`（1.2.5 节、1.2.6 节）

**问题描述**：

Phase 1 设计了 `GET /api/v1/ai/runs/:runId` 接口和 `AIRunInfo` 类型，并在 SSE 协议的 `init` 事件中返回 `run_id`。文档隐含的假设是 `run_id` 等同于 `AIExecution.ID`。

但实际 `AIExecution` 模型：

```go
type AIExecution struct {
    ID           string  // 主键，UUID（这才是 execution id）
    SessionID    string
    PlanID       string
    StepID       string
    CheckpointID string
    // 没有独立的 run_id 字段
    ...
}
```

`AIExecution` 是**每一步工具执行**的记录，一次 Agent 对话可能有多条 `AIExecution` 记录（每个工具调用一条）。文档将"Agent 整体运行"（run）与"单次工具执行"（execution）混为一谈。

**影响**：
- `GET /api/v1/ai/runs/:runId` 无法通过 `AIExecution` 直接实现
- SSE `init` 事件返回的 `run_id` 含义不清

**修正方案**：

需要明确区分两个概念并分开设计：
1. **AgentRun**（会话级 Agent 运行）：可以用 `AIChatTurn`（已有模型，有 `TraceID`）作为 run 的载体，或新增一个轻量级 `run` 记录表
2. **AIExecution**（工具调用级执行记录）：维持现有模型不变，作为详细日志

文档应以 `AIChatTurn.ID` 或新增 `ai_runs` 表的 `id` 作为 `run_id` 的来源，而非 `AIExecution.ID`。

---

### 🔴 P1-6：`DBCheckpointStore.Save` 使用 `FirstOrCreate` 无法更新已有 Checkpoint

**所在文档**：`phase1-phase2-technical-design.md`（2.2.3 节）

**问题描述**：

文档中 `DBCheckpointStore.Save` 实现：

```go
func (s *DBCheckpointStore) Save(ctx context.Context, key string, value []byte) error {
    cp := &model.AICheckPoint{
        Key:   key,
        Value: value,
    }
    return s.db.WithContext(ctx).
        Where(model.AICheckPoint{Key: key}).
        Assign(cp).
        FirstOrCreate(cp).Error  // ❌ 只有不存在时才创建，不会更新 value
}
```

`FirstOrCreate` 语义：若记录已存在（`key` 命中唯一索引），返回已有记录，**不执行更新**。而 Checkpoint 的核心用途是在每次 HITL 中断时**覆盖更新**同一 key 的 value（保存最新状态）。用 `FirstOrCreate` 会导致 Checkpoint value 永远停留在第一次保存的状态，Resume 时恢复的是旧状态。

**修正方案**：改用 `ON DUPLICATE KEY UPDATE` 语义：

```go
func (s *DBCheckpointStore) Save(ctx context.Context, key string, value []byte) error {
    cp := &model.AICheckPoint{Key: key, Value: value}
    return s.db.WithContext(ctx).
        Clauses(clause.OnConflict{
            Columns:   []clause.Column{{Name: "key"}},
            DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
        }).
        Create(cp).Error
}
```

---

## 二、严重问题（共 6 项）

### 🟠 P2-1：项目存在两张审批表，文档仅描述其中一张，关系未厘清

**所在文档**：`phase1-phase2-technical-design.md`（2.2 节、2.3 节）

**问题描述**：

查阅 `storage/migrations/` 可以发现项目实际存在**两张**审批相关表：

**表 1：`ai_approval_tickets`**（旧表）
- 来自 `storage/migrations/20260305_000029_ai_confirmation_and_approval.sql`
- 对应模型：`internal/model/ai_approval_task.go`（`AIApprovalTask`）
- 字段特征：`confirmation_id`、`approval_token`（用于邮件链接）、`target_resource_type/id/name`、`task_detail_json`、`tool_calls_json`

**表 2：`ai_approvals`**（新表）
- 来自 `storage/migrations/20260313_000036_ai_module_redesign_backend.sql`
- 当前无对应 GORM 模型文件
- 字段特征：`session_id`、`plan_id`、`step_id`、`checkpoint_id`、`approval_key`（唯一索引）

Phase 2 文档的 HITL 流程需要 `session_id`、`plan_id`、`checkpoint_id` 等字段，这些字段在 `ai_approvals`（新表）中存在，在 `ai_approval_tickets`（旧表）中**不存在**。但文档始终引用 `ai_approval_tickets` 和 `AIApprovalTask` 模型，从未提及 `ai_approvals`。

**影响**：开发者按文档实现时，会向不含必要字段的旧表写数据，且无法通过 `session_id`/`checkpoint_id` 关联 Agent 运行状态。

**修正方案**：需要在文档中明确：
1. Phase 2 HITL 审批流应以 `ai_approvals` 表（新表）为主，补充其 GORM 模型定义（`internal/model/ai_approval.go`）
2. 说明 `ai_approval_tickets`（旧表）与新流程的关系（复用还是废弃）
3. 若两张表都要保留，明确各自的使用场景

---

### 🟠 P2-2：乐观锁依赖的 `version` 字段在任何审批表中均不存在

**所在文档**：`phase1-phase2-technical-design.md`（2.2.5 节）

**问题描述**：

Phase 2 幂等控制章节描述了基于 `version` 字段的数据库乐观锁方案，但实际上：

- `ai_approval_tickets` 表：无 `version` 字段
- `ai_approvals` 表：无 `version` 字段

文档中的 `updateApprovalStatus` 函数实际上并没有用到 `version`——它用的是 `WHERE status = fromStatus` 条件判断（这本身是一种有效的乐观锁，但文档里叫"version 乐观锁"，名实不符）。

**修正方案**：
- **方案 A（无需改表）**：将文档中"乐观锁"的说明修正为"基于状态字段的条件更新"——`WHERE id = ? AND status = ?`，RowsAffected = 0 时代表状态已被其他请求修改，这已经能正确实现幂等
- **方案 B（加字段）**：若确实需要 version 乐观锁，在 migration 中添加 `version INT NOT NULL DEFAULT 0`，并在更新时 `WHERE id = ? AND version = ?` + `SET version = version + 1`

---

### 🟠 P2-3：`roadmap.md` 与 `phase3` 文档的风险评分模型完全不同

**所在文档**：`roadmap.md`（7.2 节）vs `phase3-phase4-technical-design.md`（3.3 节）

**问题描述**：

两个文档描述的是同一个风险评分引擎，但维度定义完全不同：

| | **roadmap.md（7.2 节）** | **phase3 文档（3.3 节）** |
|--|--|--|
| 维度 1 | 操作类型（30%）：scale=20/restart=30/... | 影响范围 ImpactScore（0-25）：受影响资源数×环境系数 |
| 维度 2 | 目标环境（25%）：dev=10/staging=40/prod=80 | 操作不可逆性 ReversibleScore（0-25）：可逆/不可逆 |
| 维度 3 | 影响范围（25%）：单Pod=20/单服务=40/... | 环境等级 EnvScore（0-25）：dev=5/staging=15/prod=25 |
| 维度 4 | **时间窗口（20%）**：变更窗口内=10/高峰期=80 | **当前健康状态 HealthScore（0-25）**：告警数×3+副本比 |

roadmap 有"时间窗口"维度，phase3 有"当前健康状态"维度，两者互不包含。roadmap 的操作类型直接给分数，phase3 用不可逆性分类。这是两套完全不同的设计。

**影响**：开发者无法确定应该实现哪个版本，两个文档都无法直接作为实现依据。

**修正方案**：统一成一个版本，建议采用 phase3 文档中的版本（影响范围 + 不可逆性 + 环境等级 + 健康状态），因为它有更完整的代码实现。同时在 roadmap.md 的 7.2 节删除旧的评分表格，改为引用 phase3 文档。

---

### 🟠 P2-4：`internal/ai/agents/router.go` 与 Phase 1 意图路由设计重叠，职责未厘清

**所在文档**：`phase1-phase2-technical-design.md`（1.2.2 节）

**问题描述**：

项目已有 `internal/ai/agents/router.go`，实现了一个基于 `ChatModelAgent` 的路由 Agent：

```go
func NewRouterAgent(ctx context.Context) (*adk.ChatModelAgent, error) {
    // 使用 ROUTERPROMPT，MaxIterations=3，Temp=0.2
    ...
}
```

Phase 1 文档设计了全新的 `internal/ai/agents/intent/` 目录，包含 `IntentRouter`、`ruleLayer`、`modelLayer`、`policyLayer`，与现有 `router.go` 功能完全重叠。

文档**没有**说明：
1. 现有 `router.go` 是否废弃
2. 新 `intent/` 模块是对 `router.go` 的扩展还是替换
3. `ROUTERPROMPT`（`internal/ai/agents/prompt/prompt.go`）是否继续使用

**影响**：开发者会不知道该修改哪个文件，极有可能产生两套并行的路由逻辑。

**修正方案**：文档必须明确说明：**新的 `IntentRouter` 是对 `router.go` 的重构替代**，`NewRouterAgent` 作为其中 `modelLayer` 的底层实现保留，整体 Router 入口改为 `IntentRouter`。同时在文档中指出 `ROUTERPROMPT` 需要同步更新以支持输出结构化的 `IntentResult`（JSON 格式）。

---

### 🟠 P2-5：Phase 2 审批接口路由权限中间件名称未在项目中验证

**所在文档**：`phase1-phase2-technical-design.md`（2.3 节）

**问题描述**：

文档路由注册代码：

```go
approvals := ai.Group("/approvals", middleware.RequirePermission("cluster:write"))
```

但项目 `internal/middleware/` 目录中并未确认存在 `RequirePermission` 这个中间件函数名。项目使用 Casbin/RBAC 的方式与此名称可能不符。

**影响**：直接复制文档代码会导致编译失败（函数未定义）。

**修正方案**：检查 `internal/middleware/` 实际暴露的权限校验中间件函数名，将文档中的 `middleware.RequirePermission(...)` 替换为实际函数名。若该中间件尚不存在，文档应补充"需要新增 `RequirePermission` 中间件"的说明，而非假设其已存在。

---

### 🟠 P2-6：Phase 3 的 `secondary_pending` 状态在 `AIApprovalTask` 的 Status 说明中缺失

**所在文档**：`phase3-phase4-technical-design.md`（3.4.5 节）

**问题描述**：

Phase 3 多级审批状态机图中新增了 `secondary_pending` 状态（Level 3 一审通过后等待二审），但：

1. `AIApprovalTask.Status` 字段的注释只列出了 `pending/approved/rejected/executed/expired` 五种状态
2. Phase 3 的 Migration（`20240810_extend_approval_tickets.sql`）只新增了 `secondary_status` 字段，没有在 `status` 字段增加 `secondary_pending` 枚举值说明
3. Phase 2 文档的 7 态状态机（`created/pending/approved/rejected/expired/executing/executed/failed/ended`）中也没有 `secondary_pending`

这导致两个阶段的状态机定义互相矛盾，`status` 字段的合法取值范围在三处文档中不一致。

**修正方案**：统一状态机定义。建议的完整状态集：

```
pending → approved → (Level 3 时) secondary_pending → executing → executed
                  → rejected（一审拒绝）
pending → expired
secondary_pending → rejected（二审拒绝）
executing → failed
```

并在所有文档中使用同一份状态机描述。`secondary_pending` 作为 `status` 字段的合法值，需在 model 注释和 migration 中同步更新。

---

## 三、中等问题（共 8 项）

### 🟡 P3-1：roadmap.md 和 phase1 文档的 SSE 事件类型定义不一致

**所在文档**：`roadmap.md`（5.2 节）vs `phase1-phase2-technical-design.md`（1.2.6 节）

**问题描述**：

| 事件类型 | roadmap.md | phase1 文档 |
|---------|-----------|------------|
| 文本内容 | `event: block` → 发送 AIChatBlock | `event: message` → LLM 输出 token |
| 状态更新 | `event: status` | 未单独定义 |
| Phase 2 审批 | 未提及 | `approval_required` |

roadmap 用 `block` 对齐数据库中的 `AIChatBlock` 模型（更合理），phase1 用 `message` 对齐更常见的 SSE 命名习惯。这两套命名前端实现时只能选其一。

**修正方案**：统一采用 `block` 作为发送 `AIChatBlock` 结构体的事件类型，`message` 作为纯文本 token 流的事件类型（两者语义不同可以共存），在 phase1 文档中补充说明两者区别。

---

### 🟡 P3-2：Phase 3 migration 文件命名和日期与项目规范不符

**所在文档**：`phase3-phase4-technical-design.md`（3.7 节）

**问题描述**：

Phase 3 文档给出的 migration 文件名：

```
storage/migrations/20240810_extend_approval_tickets.sql
storage/migrations/20240811_ai_change_snapshots.sql
```

但项目现有文件的命名规范是 `YYYYMMDD_######_description.sql`（日期 + 6 位序号），且现有文件的日期均为 `20260xxx`，下一个应该是 `20260316_000039_xxx.sql` 或更晚。

使用 `2024` 年的日期会导致 migration runner 按字典序排列时将 Phase 3 的 migration 插入到已有 38 个文件之前执行，可能引发依赖问题（Phase 3 的表扩展依赖 Phase 2 创建的表）。

**修正方案**：将所有 Phase 3 & Phase 4 的 migration 文件名修正为正确的日期格式和序号，例如：
- `storage/migrations/20260316_000039_extend_approval_tickets.sql`
- `storage/migrations/20260316_000040_ai_change_snapshots.sql`

---

### 🟡 P3-3：`ai_change_snapshots` 表日期时间精度与项目不一致

**所在文档**：`phase3-phase4-technical-design.md`（3.5.2 节）

**问题描述**：

Phase 3 的 `ai_change_snapshots` 表定义使用 `DATETIME` 类型：

```sql
`collected_at` DATETIME NOT NULL COMMENT '实际采集时间',
`created_at`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
```

但查阅项目现有 migration 文件，时间字段均使用 `TIMESTAMP` 或 `DATETIME(3)`（毫秒精度）。例如，`ai_approvals` 表：

```sql
approved_at TIMESTAMP NULL,
created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
```

对于指标快照这类时序数据，`DATETIME`（秒级精度）会导致同一秒内的两次采集无法区分，建议使用更高精度。

**修正方案**：将 `ai_change_snapshots` 的时间字段改为 `TIMESTAMP(3)` 或 `DATETIME(3)`，与项目其他时序数据表保持一致。

---

### 🟡 P3-4：Phase 3 Go 1.26 项目中自定义 `min()` 函数与内置函数冲突

**所在文档**：`phase3-phase4-technical-design.md`（3.3.2 节）

**问题描述**：

Phase 3 `engine.go` 末尾定义了：

```go
// min 返回两个整数中的较小值（Go 1.21+ 标准库已内置，此为兼容旧版本）。
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

注释本身已承认 Go 1.21+ 已内置，而项目使用 Go 1.26（`go.mod` 中指定）。**在 Go 1.21+ 中，`min` 是内置函数（builtin），自定义同名函数会导致编译错误**：`min redeclared in this block`。

**修正方案**：直接删除该自定义函数，在代码中直接使用内置 `min(a, b)`，Go 1.21+ 已原生支持泛型 `min`。

---

### 🟡 P3-5：`ai_diagnosis_reports` 新表与已有 `AIChatBlock` 的职责重叠，文档未说明取舍

**所在文档**：`phase1-phase2-technical-design.md`（1.2.7 节）

**问题描述**：

Phase 1 提议新建 `ai_diagnosis_reports` 表存储结构化诊断报告。但项目已有 `AIChatBlock` 模型，其 `content_json` 字段（`LONGTEXT`）已经可以存储任意结构化 JSON：

```go
type AIChatBlock struct {
    BlockType   string  // 可以是 "diagnosis_report"
    ContentText string  // 摘要文本
    ContentJSON string  // 完整 DiagnosisReport JSON
    ...
}
```

实际上，`storage/migrations/20260312_000035_ai_chat_turn_blocks.sql` 的设计意图就是将诊断报告、工具调用等各种块统一存入 `ai_chat_blocks`。

新建 `ai_diagnosis_reports` 表会导致：
- 诊断报告数据**双写**：`ai_chat_blocks` 一份，`ai_diagnosis_reports` 一份
- 后续查询逻辑分裂（按会话查是在 blocks，按 run_id 查是在 diagnosis_reports）

**修正方案**：取消新建 `ai_diagnosis_reports` 表。诊断报告以 `block_type = 'diagnosis_report'` 存入 `ai_chat_blocks.content_json`，同时在 `ai_chat_blocks` 上补充 `run_id` 索引（或通过 `turn_id → trace_id → run` 关联查询）来支持 `GET /api/v1/ai/diagnosis/:runId/report` 接口。

---

### 🟡 P3-6：Phase 4 `DiagnosisSink` 调用 `rag.Ingestion` 接口未验证其实际签名

**所在文档**：`phase3-phase4-technical-design.md`（4.4 节）

**问题描述**：

Phase 4 文档描述根因知识沉淀时写道"通过已有 `internal/rag/ingestion.go` 的 `Ingest()` 函数"写入 Milvus，但没有给出实际调用代码，也没有核对 `ingestion.go` 是否暴露了合适的对外接口（仅在 `internal/ai/knowledge/diagnosis_sink.go` 中引用）。

实际 `internal/rag/ingestion.go` 可能是面向定时批量导入设计的（`ScheduledIngestion`），其接口可能是接受文件路径或文档列表，而非单条实时写入。用异步 goroutine 调用批量导入接口会产生资源争用问题。

**修正方案**：Phase 4 文档应先确认 `ingestion.go` 的对外接口签名，再根据实际签名描述调用方式。若现有 `ingestion.go` 不支持单条实时写入，需要补充"新增 `internal/rag/realtime_ingest.go` 提供单条写入接口"的说明。

---

### 🟡 P3-7：Phase 4 自动巡检 Job 与已有 `internal/service/jobs/` 的集成方式未说明

**所在文档**：`phase3-phase4-technical-design.md`（4.3 节）

**问题描述**：

Phase 4 提议在 `internal/service/jobs/` 下新增 `daily_inspection_job.go`，但没有说明如何与现有 jobs 基础设施集成：
- 项目现有 Job 的注册/调度方式是什么（是否有统一的 cron scheduler）
- 巡检 Job 如何获取所有集群列表（`PlatformDeps` 注入？还是直接查 DB？）
- 巡检结果的 `cluster_ids_json` 字段说明不足（存储所有集群 ID 还是仅有告警的集群）

**修正方案**：补充说明如何向现有 jobs 注册新 Job，给出 `DailyInspectionJob` 结构体的完整依赖注入方式（应通过 `PlatformDeps` 获取 DB 和 K8s 客户端）。

---

### 🟡 P3-8：Phase 3 `PatchResourceTool.Invoke` 存在逻辑缺陷：审批通过后未使用可能被修改的参数

**所在文档**：`phase3-phase4-technical-design.md`（3.2.2 节）

**问题描述**：

`PatchResourceTool.Invoke` 在触发 HITL 中断并恢复后直接使用原始 `input.PatchContent` 执行实际 patch：

```go
// Step 9: 审批通过后执行实际 patch（Resume 后重新进入此处）
realResult, err := dynClient.Resource(gvr).Namespace(input.Namespace).
    Patch(ctx, input.Name, pt, []byte(input.PatchContent), metav1.PatchOptions{})
```

但根据 Phase 2 的 HITL 设计，审批人可以**修改参数**后批准（`modified_params`）。Phase 3 的 Invoke 实现没有处理"使用审批后修改过的 patch_content 执行"这个场景，直接忽略了审批人的修改。

相比之下，Phase 2 的 `ScaleDeploymentTool` 正确处理了这个场景（`mergeParams(input, resumeParams.ModifiedParams)`）。

**修正方案**：在 Step 9 之前，应从 Interrupter 的恢复上下文中获取最终参数：

```go
// Step 9: 从恢复上下文获取最终参数（可能被审批人修改）
resumeCtx := t.interrupter.GetResumeContext(ctx)
finalPatchContent := input.PatchContent
if resumeCtx.ModifiedParams != nil {
    if pc, ok := resumeCtx.ModifiedParams["patch_content"].(string); ok {
        finalPatchContent = pc
    }
}
realResult, err := dynClient.Resource(gvr).Namespace(input.Namespace).
    Patch(ctx, input.Name, pt, []byte(finalPatchContent), metav1.PatchOptions{})
```

`apply_manifest` 工具存在相同问题，需要同样修正。

---

## 四、轻微问题（共 5 项）

### 🔵 P4-1：`ErrOptimisticLock` 在 Phase 2 中引用但未定义

**所在文档**：`phase1-phase2-technical-design.md`（2.2.5 节）

**问题描述**：`updateApprovalStatus` 函数末尾返回 `ErrOptimisticLock`，但文档中未定义该错误变量，也未说明其所在包和定义方式。

**修正方案**：在文档中补充定义：

```go
// internal/ai/hitl/errors.go
var ErrOptimisticLock = errors.New("optimistic lock: approval status already changed")
```

---

### 🔵 P4-2：TypeScript `useSSEChat` Hook 中 `getToken()` 来源不明

**所在文档**：`phase1-phase2-technical-design.md`（1.3.2 节）

**问题描述**：

`useSSEChat` Hook 代码中调用了 `getToken()`：

```typescript
headers: {
    Authorization: `Bearer ${getToken()}`,
```

但文档未说明 `getToken` 的来源（是 `@/utils/auth`、`localStorage` 直接读取还是其他方式），也没有对应的 import 语句，无法直接使用。

**修正方案**：补充说明 `getToken` 的来源，或改写为项目实际使用的认证方式，例如通过 `axios` 拦截器统一注入 token，避免在 `fetch` 中手动处理。

---

### 🔵 P4-3：Phase 1 中 `IntentRequest.SessionID` 定义为 `int64` 但实际应为 `string`

**所在文档**：`phase1-phase2-technical-design.md`（1.2.2 节）

**问题描述**：

```go
type IntentRequest struct {
    UserID    int64  `json:"user_id"`
    SessionID int64  `json:"session_id"`  // ❌ 实际 session ID 是 UUID string
    ...
}
```

这是 P1-1（ID 类型不一致）问题在 Go 结构体中的具体体现，在意图路由层也需要修正。

**修正方案**：改为 `SessionID string`。

---

### 🔵 P4-4：Phase 3 `ApplyManifestTool` 中 `serverSideApplyDryRun` 和 `serverSideApply` 等辅助函数未定义

**所在文档**：`phase3-phase4-technical-design.md`（3.2.3 节）

**问题描述**：

`ApplyManifestTool.Invoke` 调用了多个辅助函数，但文档中均未给出实现：
- `splitYAMLDocuments(yaml string) ([]unstructured.Unstructured, error)`
- `serverSideApplyDryRun(ctx, client, doc, ns, fieldManager, force) (*ApplyResult, error)`
- `serverSideApply(ctx, client, doc, ns, fieldManager, force) error`
- `resolveDynamicClient(deps, clusterID int) (dynamic.Interface, error)`
- `resolveGVR(ctx, client, resourceType string) (schema.GroupVersionResource, error)`

这些都是非常复杂的函数（尤其是 `resolveDynamicClient` 涉及多集群 KubeConfig 管理），文档作为设计参考应至少给出函数签名和关键逻辑说明。

**修正方案**：在文档中补充上述辅助函数的接口签名和实现要点（如 `resolveDynamicClient` 如何从 `PlatformDeps.DB` 查询集群凭据、`splitYAMLDocuments` 如何处理空文档等），作为开发者的实现指引。

---

### 🔵 P4-5：三份文档的日期标注混乱，无法作为版本管理依据

**所在文档**：全部三份文档

**问题描述**：

- `roadmap.md`：标注版本 `v1.0`，日期 `2025-07`
- `phase1-phase2-technical-design.md`：标注版本 `v1.0`，无日期
- `phase3-phase4-technical-design.md`：标注版本 `v1.0`，无日期
- Phase 3 migration 文件使用 `2024` 年日期

文档之间的版本号相同但没有关联关系说明，未来若 Phase 1 文档更新到 v1.1，Phase 2 文档是否同步更新无从判断。

**修正方案**：
1. 在三份文档的头部添加统一的"文档关系说明"：`roadmap.md` 是总纲，phase 文档是细化
2. 统一填写创建/更新日期
3. migration 文件日期改用实际开发日期（`2026` 系列）

---

## 五、问题汇总一览表

| 编号 | 级别 | 所在文档 | 问题简述 | 修复成本 |
|------|------|---------|---------|---------|
| P1-1 | 🔴 致命 | phase1-2 | 所有 ID 类型：文档用 `number`，实际是 `string` UUID | 中（需全文替换） |
| P1-2 | 🔴 致命 | phase1-2 | `AIChatSession` 缺 `cluster_name` 字段，需 migration | 小（补 migration 或改方案） |
| P1-3 | 🔴 致命 | phase1-2 | Migration 路径 `storage/migration/` 应为 `storage/migrations/` | 小（改路径和命名） |
| P1-4 | 🔴 致命 | phase1-2 | Migration 索引 `ai_chat_blocks.message_id` 字段不存在 | 小（删除该语句） |
| P1-5 | 🔴 致命 | phase1-2 | `run_id` 概念混淆：`AIExecution` 无此字段，需明确 run 的来源 | 中（重新设计 run 模型） |
| P1-6 | 🔴 致命 | phase1-2 | `DBCheckpointStore.Save` 用 `FirstOrCreate` 无法更新已有值 | 小（改用 Upsert） |
| P2-1 | 🟠 严重 | phase1-2 | 两张审批表（`ai_approvals` vs `ai_approval_tickets`）混淆 | 大（需重新厘清职责） |
| P2-2 | 🟠 严重 | phase1-2 | 乐观锁 `version` 字段不存在，描述与实现不符 | 小（改描述或加字段） |
| P2-3 | 🟠 严重 | roadmap vs phase3 | 风险评分两套维度体系互相矛盾 | 中（统一删除一套） |
| P2-4 | 🟠 严重 | phase1-2 | 意图路由设计与已有 `router.go` 职责重叠未说明 | 小（补充说明） |
| P2-5 | 🟠 严重 | phase1-2 | `middleware.RequirePermission` 函数是否存在未验证 | 小（核查 middleware 包） |
| P2-6 | 🟠 严重 | phase3-4 | `secondary_pending` 状态未在 model 和 status 说明中同步 | 小（补充状态定义） |
| P3-1 | 🟡 中等 | roadmap vs phase1 | SSE event 类型：`block` vs `message` 不一致 | 小（统一命名） |
| P3-2 | 🟡 中等 | phase3-4 | Phase 3 migration 命名规范和日期不符合项目规范 | 小（改命名） |
| P3-3 | 🟡 中等 | phase3-4 | `ai_change_snapshots` 时间字段精度低于项目标准 | 小（改 DATETIME(3)） |
| P3-4 | 🟡 中等 | phase3-4 | Go 1.26 中自定义 `min()` 与内置函数冲突导致编译失败 | 小（删除自定义函数） |
| P3-5 | 🟡 中等 | phase1-2 | `ai_diagnosis_reports` 新表与 `ai_chat_blocks` 职责重叠 | 中（取消新表设计） |
| P3-6 | 🟡 中等 | phase3-4 | Phase 4 知识沉淀调用 `rag.Ingest()` 接口签名未验证 | 小（补充说明） |
| P3-7 | 🟡 中等 | phase3-4 | 巡检 Job 与现有 jobs 基础设施的集成方式未说明 | 小（补充说明） |
| P3-8 | 🟡 中等 | phase3-4 | `PatchResourceTool` 和 `ApplyManifestTool` 审批后忽略修改参数 | 中（补充 resumeCtx 逻辑） |
| P4-1 | 🔵 轻微 | phase1-2 | `ErrOptimisticLock` 引用但未定义 | 小 |
| P4-2 | 🔵 轻微 | phase1-2 | TypeScript `getToken()` 来源不明 | 小 |
| P4-3 | 🔵 轻微 | phase1-2 | `IntentRequest.SessionID` 应为 `string` 而非 `int64` | 小 |
| P4-4 | 🔵 轻微 | phase3-4 | `apply_manifest` 多个辅助函数未定义，缺乏实现指引 | 小（补充接口说明） |
| P4-5 | 🔵 轻微 | 全部 | 文档日期/版本标注混乱，migration 日期用 2024 年 | 小 |

**问题总计**：🔴 6 项 + 🟠 6 项 + 🟡 8 项 + 🔵 5 项 = **25 项**

---

## 六、修正建议优先级

按"阻塞开发进度"排序，建议按以下顺序修正文档：

### 第一优先级（开始编码前必须修正，否则写出来就要返工）

1. **P1-1**：统一所有 ID 类型为 `string`
2. **P1-3 + P1-4**：修正 migration 路径和错误的索引语句
3. **P1-2**：明确 `cluster_name` 的处理方案（加列 or 复用 `scene`）
4. **P1-5**：明确 `run_id` 的来源（基于 `AIChatTurn` 还是新建 `ai_runs` 表）
5. **P2-1**：厘清两张审批表的职责（`ai_approvals` vs `ai_approval_tickets`），确定 Phase 2 HITL 流程以哪张表为主
6. **P1-6**：修正 `DBCheckpointStore.Save` 使用 `FirstOrCreate` → `Upsert`
7. **P2-3**：删除 `roadmap.md` 中的旧风险评分表格，统一指向 phase3 版本

### 第二优先级（Phase 2 开发前必须修正）

8. **P2-4**：明确意图路由新设计与已有 `router.go` 的关系（扩展还是替换）
9. **P2-5**：核查 `middleware.RequirePermission` 是否存在，补充正确的中间件调用方式
10. **P2-6**：统一三份文档的审批状态机，明确 `secondary_pending` 的完整定义
11. **P2-2**：将"乐观锁 version 字段"描述修正为"基于状态字段的条件更新"
12. **P3-5**：取消 `ai_diagnosis_reports` 新表设计，改为复用 `AIChatBlock.content_json`
13. **P4-3**：修正 `IntentRequest.SessionID` 类型为 `string`

### 第三优先级（Phase 3 开发前必须修正）

14. **P3-1**：统一 SSE event 类型命名（`block` vs `message`）
15. **P3-2 + P3-3**：修正 Phase 3 migration 文件命名规范和时间字段精度
16. **P3-4**：删除 Phase 3 `engine.go` 中自定义 `min()` 函数
17. **P3-8**：在 `PatchResourceTool` 和 `ApplyManifestTool` 的 Invoke 中补充"审批后使用修改参数"逻辑

### 第四优先级（Phase 4 前或迭代中修正）

18. **P3-6**：补充 Phase 4 知识沉淀对 `rag.Ingestion` 接口的调用说明
19. **P3-7**：补充巡检 Job 与现有 jobs 基础设施的集成方式
20. **P4-1 ～ P4-5**：补充缺失的错误变量定义、`getToken` 来源、辅助函数说明，统一文档日期

---

*文档审查完毕。建议在修正上述问题后，将本文档归档到 `docs/ai/review-issues-resolved.md`，并在每份设计文档的头部添加"已按 review-issues.md 完成修正，版本 v1.1"的标注。*