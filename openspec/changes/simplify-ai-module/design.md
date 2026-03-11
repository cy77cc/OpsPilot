## Context

`internal/ai/` 是 k8s-manage 平台的 AI 编排核心，包含 Rewrite → Planner → Executor → Summarizer 四阶段流水线。当前代码有三处结构性低效：

1. `executor.EventMeta` 与 `events.EventMeta` 字段完全相同，在 orchestrator 中需要手工逐字段转换。
2. `ResolvedResources` 同时保存平铺字段（`ServiceID int`、`ClusterID int` 等）和等价列表字段（`Services []ResourceRef` 等），`canonicalizePlan` 因此需要 11 个 fallback 块。
3. `planner/planner.go` 1302 行，混合了类型定义、解析、规范化、校验、资源收集五种职责。

变更范围严格限于 `internal/ai/planner/` 和 `internal/ai/executor/` 包内部及 `orchestrator.go` 胶水层。不涉及 API contract、Redis schema 或任何对外行为。

## Goals / Non-Goals

**Goals:**
- 删除 `executor.EventMeta`，executor 直接复用 `events.EventMeta`，消除 orchestrator 中的手工转换。
- 去掉 `ResolvedResources` 的平铺冗余字段，仅保留列表字段，`canonicalizePlan` 从 11 个 fallback 块缩减至 5 个。
- 将 `planner/planner.go` 拆分为 5 个职责单一的文件，总行数不变，每文件不超过 400 行。

**Non-Goals:**
- 不改变状态机逻辑（`scheduler.go` 中 `validTransition` / `advanceScheduler` 保持不动）。
- 不改变流水线执行顺序（仍为串行依赖调度）。
- 不修改 SSE 事件结构或前端协议。
- 不改变 Redis 存储的 `ExecutionState` / `SessionSnapshot` schema（删除 `ResolvedResources` 平铺字段是内部 planner 类型，不影响 `runtime.ExecutionState`）。

## Decisions

### Decision 1: executor 直接 import events 包

- **Choice**: 删除 `executor.EventMeta`，`executor.EventEmitter` 的 `meta` 参数类型改为 `events.EventMeta`。
- **Rationale**: 两个结构体字段完全相同，只因包边界而重复定义。依赖方向 executor → events 是合法的（events 包无依赖，位于依赖图叶端）。
- **Alternative**: 将 `EventMeta` 移到第三方共享包。复杂度更高，收益相同，不采用。

### Decision 2: ResolvedResources 只保留列表字段

- **Choice**: 从 `planner.ResolvedResources` 删除 `ServiceName`、`ServiceID`、`ClusterName`、`ClusterID`、`HostNames []string`、`HostIDs []int`、`PodName` 七个平铺字段，增加包级 helper：

  ```go
  func primaryID(refs []ResourceRef) int       // refs[0].ID，空则返回 0
  func primaryName(refs []ResourceRef) string  // refs[0].Name，空则返回 ""
  func allIDs(refs []ResourceRef) []int        // 提取所有 ref.ID > 0
  ```

- **Rationale**: 平铺字段是列表字段的子集，同步维护是 bug 风险来源。解析阶段（`parseResolvedResources`）继续接受 LLM 输出的两种格式（平铺 + 列表），解析完成后统一归入列表字段。
- **Risk**: 已存 Redis 中的旧 `ExecutionPlan`（内嵌在 ExecutionState）包含平铺字段，反序列化时这些字段会被忽略（Go JSON 的 unknown fields 默认丢弃），不会 panic，但平铺字段的信息丢失。由于 ExecutionState 的 TTL 是 24 小时，过渡期可接受。如需完全兼容，可在 `parseResolvedResources` 读取旧字段时回填列表字段（本次采用此策略）。
- **Alternative**: 保留平铺字段但隐藏为内部字段（小写）。引入 JSON 自定义序列化复杂度，不采用。

### Decision 3: planner.go 拆分策略

- **Choice**: 纯文件拆分，同一 Go package（`package planner`），不引入新的 package 边界。

  | 文件 | 内容 |
  |------|------|
  | `planner.go` | `Planner` struct、`New`、`Plan`、`PlanStream`、`plan` |
  | `types.go` | 所有导出类型 + `PlanningError` |
  | `parse.go` | `ParseDecision`、`parseExecutionPlan`、`parseResolvedResources`、`looseStringValue`、`looseIntValue` 等 |
  | `normalize.go` | `normalizeDecision`、`canonicalizePlan`、`populateStepInput`、`validatePlanPrerequisites`、所有 `requires*`/`resolved*` helpers |
  | `collect.go` | `buildBasePlanContext`、`collectHostNames`、`collectPods`、`detectScope`、`dedupe*` 等 |

- **Rationale**: 同 package 拆分零风险，测试文件不需修改。每个文件职责单一，新增功能时定位明确。
- **Alternative**: 拆成子包（`planner/parse`、`planner/normalize`）。引入循环依赖风险且收益不对称，不采用。

## Risks / Trade-offs

| 风险 | 概率 | 缓解措施 |
|------|------|----------|
| 旧 Redis ExecutionState 中 ResolvedResources 平铺字段信息丢失 | 低（TTL 24h，正常情况下无跨发布的存活状态） | `parseResolvedResources` 读取平铺字段并回填到列表字段，确保 `Load` 出的旧数据仍可用 |
| executor → events 引入新依赖导致循环 | 极低（events 无任何反向依赖） | 编译器保证，`go build ./...` 验证 |
| 文件拆分后测试引用路径失效 | 无（同 package，Go 不区分同包内文件） | `go test ./internal/ai/planner/...` 验证 |

## Migration Plan

无数据迁移。所有修改限于 Go 源码，不产生数据库变更或 API schema 变更。
