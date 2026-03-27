# AI 模块前后端契约收口设计（Contract-First）

- 日期：2026-03-24
- 范围：AI 模块前后端链路（接口契约、SSE 事件协议、UI 消费状态机、测试与发布）
- 策略：契约优先（以当前后端已实现契约为准，前端立即收敛）

## 1. 背景与目标

当前 AI 模块存在前端 API 声明与后端已注册路由不一致的问题，导致链路存在“静默降级掩盖的断链风险”（例如 UI 请求未实现接口，依赖前端 fallback 不显错）。

本方案目标：

1. 建立单一真实契约源，消除前后端漂移。
2. 固化 SSE 协议与断点续流语义，保证中断可恢复或可预期降级。
3. 收敛 UI 状态机消费边界，减少隐式兼容与不确定行为。
4. 用自动化测试把契约一致性纳入 CI 门禁。

## 2. 架构边界

### 2.1 契约层（唯一真相）

- 后端：`api/ai/v1` + `internal/service/ai/routes.go` 定义可用接口白名单。
- 前端：`web/src/api/modules/ai.ts` 必须与白名单一一对应。
- 规则：禁止“前端先定义、后端未注册”的可调用 API 暴露。

### 2.2 事件层（SSE 协议）

- 固定语义：
  - `id`：事件游标，用于续流
  - `event`：事件类型，用于状态分发
  - `data`：业务载荷
- 统一事件白名单：`meta`、`agent_handoff`、`plan`、`replan`、`delta`、`tool_call`、`tool_approval`、`tool_result`、`done`、`error`、`ai.run.*`、`ai.approval.expired`。
- 统一必填字段与错误码定义（含 `AI_STREAM_CURSOR_EXPIRED`）。

### 2.3 消费层（UI 状态机）

- `PlatformChatProvider + replyRuntime` 仅消费协议内事件。
- 未知事件：进入“可观测降级”（日志/指标）并忽略，不驱动主状态更新。

## 3. 收口实施方案

### 阶段 A：冻结与对齐（当天）

1. 生成并固化后端已注册路由清单（白名单）。
2. 前端对未实现接口改为显式不可用（抛 `NotImplementedByBackend`），避免静默 404。
3. 修复已确认真实断链：
   - `GET /ai/scene/:scene/prompts` 当前前端会调用，但后端无路由。
   - 处理方式：切换到已实现数据源或从当前 UI 流程移除该依赖。

### 阶段 B：契约收口（1-2 天）

1. 前端 API 模块仅保留后端存在且有测试覆盖的接口。
2. 对漂移接口执行二选一：
   - 下线（默认）
   - 若产品确认必须保留，则补后端实现并纳入白名单。
3. 新增“契约一致性测试”：前端接口表与后端路由表自动比对，CI 失败阻断合并。

### 阶段 C：续流与稳定性加固（1 天）

1. 请求链路透传 `lastEventId -> last_event_id`，打通断点续流。
2. 前端统一处理 `AI_STREAM_CURSOR_EXPIRED`：
   - 清理游标
   - 拉取最新 projection
   - 重建可恢复上下文并重发
3. 增加关键 e2e：中断重连、审批恢复、超时与错误分支。

## 4. 目标契约清单

### 4.1 当前保留（后端已实现）

- `POST /ai/chat`
- `GET /ai/sessions`
- `POST /ai/sessions`
- `GET /ai/sessions/:id`
- `DELETE /ai/sessions/:id`
- `GET /ai/runs/:runId`
- `GET /ai/runs/:runId/projection`
- `GET /ai/run-contents/:id`
- `GET /ai/diagnosis/:reportId`
- `GET /ai/approvals/pending`
- `GET /ai/approvals/:id`
- `POST /ai/approvals/:id/submit`

### 4.2 当前前端存在但需收口处理

以下接口需“下线或补后端后再开放”：

- `/ai/sessions/current`
- `/ai/sessions/:id/branch`
- `PATCH /ai/sessions/:id`
- `/ai/capabilities`
- `/ai/tools/*`
- `/ai/executions/:id`
- `/ai/feedback`
- `/ai/confirmations/:id/confirm`
- `/ai/scene/:scene/tools`
- `/ai/scene/:scene/prompts`
- `/ai/usage/stats`
- `/ai/usage/logs`

## 5. 数据流与错误处理

### 5.1 标准数据流

1. 发送：`CopilotSurface -> PlatformChatProvider -> aiApi.chatStream`。
2. 请求字段：有值即透传 `session_id/client_request_id/last_event_id`。
3. 流式接收：按 SSE `id/event/data` 解析，并保存最新 `id` 作为续流游标。
4. 历史恢复：`run projection + run content` 与实时流共享 runtime 投影模型。

### 5.2 错误策略

- `AI_STREAM_CURSOR_EXPIRED`：执行快照刷新与重建。
- `tool_timeout_soft`：提示并继续等待。
- `tool_timeout_hard`：终止当轮，标记可恢复。
- 审批冲突（400/409）：刷新审批状态，不重复提交。
- 非契约错误：统一 terminal error，并附 `run_id` 便于排查。

## 6. 测试与验收

### 6.1 测试矩阵

1. 契约一致性测试：前端 API 映射 vs 后端路由白名单。
2. 协议测试：事件枚举、必填字段、事件游标重放语义。
3. e2e 测试：正常流、断线续流、审批通过/拒绝/过期、工具超时。

### 6.2 验收标准

1. UI 不再请求后端未实现接口（无新增 404 漂移）。
2. 断线后可基于游标恢复或按预期降级。
3. 审批状态转换可复现、可追踪、可验证。

### 6.3 实施说明

- CI 门禁落点使用 `.gitea/workflows/ci.yaml`，因为当前仓库未使用 `.github/workflows`。

## 7. 风险与回滚

### 7.1 风险

- 下线漂移接口可能暴露既有隐藏依赖。
- 续流逻辑上线后可能引入“重复事件消费”边界问题。

### 7.2 控制与回滚

- 阶段化发布：先开契约检测告警，再开启阻断。
- 保留旧行为开关（短期）用于紧急回退到“禁续流 + 全量重拉”。
- 关键路径异常时可快速回退到“仅保留 chat + projection + approval 基线链路”。

## 8. 非目标（本期不做）

1. 不在本期扩展新业务能力（如新增工具系统或新审批模型）。
2. 不做与契约收口无关的大规模重构。
3. 不引入新的协议版本分叉（本期为单轨收口）。
