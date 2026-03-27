# AI 模型路由热更新设计（仅影响新请求）

## 1. 背景与问题

当前 AI 对话入口在服务启动时初始化 `Logic.AIRouter`。`AIRouter` 内部各 Agent 的 `ChatModel` 也在初始化阶段绑定。
这导致管理员在系统内修改默认模型后，不重启服务时，新请求仍可能继续使用旧模型实例。

已确认的目标语义：
- 仅影响新请求
- 不影响进行中的会话 / run
- 不要求重启服务

## 2. 目标与非目标

### 2.1 目标

1. 默认模型配置（增删改、设默认、导入）变化后，新进入的 `Chat` 请求自动使用新模型。
2. 进行中的请求继续使用进入时的 router/model 快照，不发生中途切换。
3. 失败可回退：新 router 构建失败时保持旧 router 可用，避免服务中断。
4. 在高并发下避免重复构建风暴。

### 2.2 非目标

1. 不实现“强制切换进行中请求到新模型”。
2. 不改动前端交互协议（SSE contract 不变）。
3. 不在本次引入复杂配置中心或分布式通知系统。

## 3. 总体方案

采用“请求前版本探测 + 懒重建 + 原子替换”的热更新机制。

### 3.1 核心流程

在 `Logic.Chat(...)` 入口增加 `ensureRouterFresh(ctx)`：

1. 读取模型配置版本（DB）。
2. 若版本与当前 router 版本一致：直接返回当前 router。
3. 若版本变化：构建新 `agents.NewRouter(...)`。
4. 构建成功后原子替换 `AIRouter` 与 `routerVersion`。
5. 当前请求使用进入时拿到的 router 快照。

### 3.2 配置版本定义

从 `ai_llm_providers`（`deleted_at IS NULL`）查询：
- `MAX(updated_at)`
- `COUNT(*)`

组合版本键：
- `<maxUpdatedAtUnixNano>:<count>`

说明：
- `COUNT(*)` 覆盖新增/删除变化。
- `MAX(updated_at)` 覆盖更新、设默认、导入更新。

精度与一致性约束：
- 优先保证 `updated_at` 具备毫秒或微秒精度（如 `DATETIME(3|6)`）。
- 若部署环境无法保证高精度时间戳，采用兜底版本策略：
  - 查询全部有效记录的 `(id, updated_at, is_default, is_enabled, sort_order, config_version)`；
  - 按 `id` 排序后拼接并计算 `CRC32/MD5` 作为版本指纹；
  - 最终版本键可采用 `<count>:<hash>`。

## 4. 组件与改动点

## 4.1 Logic 结构扩展

文件：`internal/service/ai/logic/logic.go`

新增字段（示意）：

```go
type Logic struct {
    // existing fields...
    AIRouter adk.ResumableAgent // 兼容字段，后续读路径以 routerState 为准

    // 原子发布不可变快照，避免多字结构并发读写 data race
    routerState atomic.Pointer[RouterState]

    // 仅用于构建阶段互斥，防止重复重建
    buildMu sync.Mutex

    // 合并并发版本查询，避免 TTL 过期瞬间查库风暴
    versionSF singleflight.Group
}

type RouterState struct {
    Router    adk.ResumableAgent
    Version   string
    VersionAt time.Time
}
```

新增方法（示意）：

```go
func (l *Logic) ensureRouterFresh(ctx context.Context) (adk.ResumableAgent, error)
func (l *Logic) currentModelConfigVersion(ctx context.Context) (string, error)
func (l *Logic) rebuildRouter(ctx context.Context) (adk.ResumableAgent, string, error)
```

## 4.2 Chat 入口接入

在 `Chat(...)` 开始阶段：

1. 调用 `ensureRouterFresh(ctx)` 拿到本次请求使用的 `router`。
2. `adk.NewRunner(...)` 时使用该 `router`。
3. 不再直接依赖可变全局 `l.AIRouter` 引用。

这样可保证“请求级快照”语义：
- 新请求可用新 router
- 进行中请求不受影响

## 4.3 并发控制策略

采用“原子快照 + DCL + singleflight”组合：

1. 快路径：原子加载 `routerState` 快照，版本相同直接返回。
2. 版本查询：通过 `singleflight` 合并同一时刻并发查询。
3. 慢路径：进入 `buildMu` 后二次比较版本，避免并发重复构建。
4. 仅当版本确实变化时构建一次新 router，并原子替换整个 `RouterState`。

## 4.4 失败回退策略

若版本变化但 `agents.NewRouter(...)` 失败：

1. 保持旧 router 不变。
2. 记录错误日志与指标，并触发高优告警（配置已提交但运行态未生效）。
3. 当前请求优先使用旧 router 继续服务（可用性优先）。

## 4.5 刷新节流

引入版本检查缓存 TTL（如 1~3 秒）：

- 间隔内重复请求复用已知版本结果，减少 DB 版本查询压力。
- 仅用于降低频率，不改变版本变化语义。

## 5. 数据流与时序

1. 管理员通过 `/admin/ai/models` 更新默认模型。
2. DB 行变更，`updated_at` 或 `count` 变化。
3. 下一次 `/ai/chat` 进入 `ensureRouterFresh`，检测到版本变化。
4. 重建 router，绑定新默认模型。
5. 本次及后续新请求使用新 router。
6. 旧请求继续使用其创建时 runner 持有的旧 router 快照。

## 6. 错误处理

1. 版本查询失败：
   - 打日志；继续使用当前 router。
2. router 重建失败：
   - 打日志 + 计数；继续旧 router。
3. 当前无可用 router（极端启动失败）：
   - 返回已有错误语义 `AI service not initialized`。
4. 冷启动期间若 router 构建连续失败：
   - 增加退避策略（如指数退避上限 30s）避免每请求都触发重建尝试。

## 7. 可观测性

建议增加：

1. 指标
   - `ai_router_reload_total{result="success|fail"}`
   - `ai_router_reload_duration_ms`
   - `ai_router_version_change_total`
   - `ai_router_reload_backoff_seconds`
2. 日志字段
   - `old_version`
   - `new_version`
   - `reload_result`
   - `error`
   - `fallback_to_old_router=true|false`

## 8. 测试策略

## 8.1 单元测试

1. 版本不变时不重建 router。
2. 版本变化时仅重建一次（并发场景下也仅一次）。
3. 重建失败时回退旧 router，`Chat` 仍可运行。
4. 节流窗口内不会重复查库/重建。

## 8.2 集成测试

1. 初始默认模型 A 发起请求，命中 A。
2. 修改默认模型为 B，不重启服务。
3. 新请求命中 B。
4. 在模型切换期间启动的旧请求不切到 B。

## 9. 风险与缓解

1. 风险：版本查询带来额外 DB 负载。
   - 缓解：版本检查缓存 TTL + singleflight 合并并发查询。
2. 风险：错误配置导致 router 重建反复失败。
   - 缓解：失败保留旧 router + 指标告警 + 重建退避。
3. 风险：并发下竞态导致重复构建。
   - 缓解：原子快照 + 双重检查锁。
4. 风险：旧 router 存在潜在资源泄漏。
   - 缓解：
     - 若 router/agent 内部为轻量无状态对象，依赖 GC 回收；
     - 若未来引入需显式释放资源（连接池、后台 goroutine），补充优雅退役机制（引用计数归零后调用 `Close`）。

## 10. 实施步骤

1. 在 `Logic` 增加 `RouterState` 原子快照、构建锁、singleflight 与 `ensureRouterFresh`。
2. 在 `Chat` 入口接入请求级 router 快照。
3. 实现版本查询（含高精度时间戳或 Hash 兜底策略）。
4. 增加失败回退日志和指标。
5. 增加冷启动失败退避策略。
6. 补充并发单测与集成测试。

## 11. 验收标准

1. 修改默认模型后，不重启服务，新请求可命中新默认模型。
2. 进行中请求不受切换影响。
3. 重建失败不影响可用性（继续旧 router）。
4. 并发压测下无重建风暴（重建次数符合预期）。
