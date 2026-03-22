# AI 助手风险命令审批链路审计报告

- 日期: 2026-03-21
- 审计范围: AI 助手中“高风险工具调用 -> 人工审批 -> 恢复执行”全链路
- 结论摘要: 设计完整，但实现当前存在多处断点与状态一致性问题，尚未达到可依赖的生产闭环

## 1. 链路目标与期望行为

### 1.1 期望主流程

1. Change Agent 执行到高风险工具时，被审批中间件拦截并触发 interrupt。
2. 运行时向前端发送 `tool_approval`（以及 `run_state(waiting_approval)`）。
3. 后端持久化审批任务（approval_id、checkpoint_id、tool_call_id、session/run/user 等）。
4. 前端展示审批卡片，用户批准/拒绝。
5. 后端写入审批结果（approved/rejected/expired）并进行权限审计。
6. 批准时调用 resume，使用 checkpoint + tool_call_id 恢复执行。
7. 恢复后事件持续写入 run_event/projection，最终 run/message 状态收敛。

### 1.2 风险控制基线

- 未审批不得执行高风险工具。
- 已拒绝不得通过后续请求绕过。
- 审批结果必须与恢复执行入参一致，不可被客户端覆盖。
- 全链路可审计（事件、状态、操作者、时间）。

## 2. 代码路径梳理

### 2.1 中间件层（工具侧）

- 审批中间件: `internal/ai/tools/middleware/approval.go`
- 高风险工具判定: `DefaultNeedsApproval`
- interrupt 触发: `tool.StatefulInterrupt(...)`

### 2.2 运行时投影层

- interrupt 投影 `tool_approval` + `run_state(waiting_approval)`: `internal/ai/runtime/project.go`

### 2.3 服务层（AI 业务逻辑）

- 对话主流程: `internal/service/ai/logic/logic.go#Chat`
- 审批提交: `SubmitApproval`
- 审批恢复: `ResumeApproval`

### 2.4 DAO / 数据模型

- 审批任务 DAO: `internal/dao/ai/approval_dao.go`
- 审批任务模型: `internal/model/ai.go` (`AIApprovalTask`)
- 表结构: `storage/migrations/20260317_0003_create_ai_approval_tasks.sql`

### 2.5 API 路由与前端调用

- 后端 AI 路由: `internal/service/ai/routes.go`
- 前端 AI API: `web/src/api/modules/ai.ts`
- 聊天页面: `web/src/components/AI/CopilotSurface.tsx`

## 3. 发现的问题（按严重级别）

## 3.1 高危问题

### F1. 中断后可能被错误收尾为 `done/completed`

- 现象:
  - 投影层在 interrupt 时把状态置为 `waiting_approval`。
  - 但 `Chat` 主流程循环结束后无条件 `Finish()` + `done` + run `completed`。
- 风险:
  - 审批尚未发生，run 却可能被标记“已完成”，形成状态欺骗，后续审批与恢复语义错乱。
- 证据:
  - `internal/ai/runtime/project.go:102-131`
  - `internal/service/ai/logic/logic.go:258-280`

### F2. `ResumeApproval` 可绕过“已拒绝”决策继续执行

- 现象:
  - `ResumeApproval` 仅在 `task.Status == pending` 时更新状态。
  - 但实际恢复参数使用 `input.Approved` 直接构造 `approvalResult`，不强制使用已持久化决策。
- 风险:
  - 攻击或误操作场景下，可先拒绝后再以 `approved=true` 调用恢复执行，突破审批约束。
- 证据:
  - `internal/service/ai/logic/logic.go:1304-1313`
  - `internal/service/ai/logic/logic.go:1316-1322`

### F3. `SubmitApproval` 缺少会话归属权限校验

- 现象:
  - `SubmitApproval` 查询审批单后直接更新状态。
  - 未校验 `approval.session_id` 是否属于当前登录用户。
- 风险:
  - 若审批 ID 泄露，可能被非所属用户改写审批状态。
- 证据:
  - `internal/service/ai/logic/logic.go:1214-1261`
  - 对比 `ResumeApproval` 的 session 校验: `internal/service/ai/logic/logic.go:1293-1301`

### F4. 恢复执行事件未可靠持久化，run/message 未收敛

- 现象:
  - `ResumeApproval` 使用了包级 `consumeProjectedEvents(...)`，内部创建空 `Logic`，不会写 `RunEventDAO`。
  - 结束后未执行与 `Chat` 对应的 run/message/projection 收敛逻辑。
- 风险:
  - 回放、审计、历史展示会缺失恢复阶段关键事件，状态长期不一致。
- 证据:
  - 调用点: `internal/service/ai/logic/logic.go:1387`, `:1392`
  - 包级函数: `internal/service/ai/logic/logic.go:516-520`
  - `ResumeApproval` 结束逻辑: `internal/service/ai/logic/logic.go:1401-1404`

### F5. 审批任务创建链路缺失（闭环断点）

- 现象:
  - 存在审批 DAO `Create`，但在 AI 服务/运行时代码中无实际调用点。
- 风险:
  - interrupt 之后可能没有审批任务可查，`/ai/approvals/:id/*` 无法真正工作。
- 证据:
  - `internal/dao/ai/approval_dao.go:24-27`
  - 全局检索未发现 `ApprovalDAO.Create(...)` 被调用。

## 3.2 中危问题

### F6. 前后端审批 API 契约不一致

- 现象:
  - 后端路由提供: `/ai/approvals/pending`, `/:id`, `/:id/submit`, `/:id/resume`。
  - 前端 API 主要调用: `/ai/approvals`, `/ai/approvals/:id/confirm`, `/ai/chains/:chainId/.../decision`。
- 风险:
  - 前端按钮触发后命中 404/无处理器，审批流程无法真正落地。
- 证据:
  - 后端: `internal/service/ai/routes.go:27-30`
  - 前端: `web/src/api/modules/ai.ts:748-804`

### F7. `tool_approval` / `run_state` 未入 run_event

- 现象:
  - `marshalProjectedEvent` 没有 `tool_approval` 与 `run_state` 分支，导致 appendRunEvent 时被忽略。
- 风险:
  - 审计链缺关键节点，无法完整还原“谁审批、何时等待审批、何时恢复”。
- 证据:
  - `internal/service/ai/logic/logic.go:571-636`

### F8. Resume SSE 响应已开始后仍可能写 JSON 错误包

- 现象:
  - `ResumeApproval` handler 先写 SSE header，再在错误分支调用 `httpx.ServerErr`。
- 风险:
  - 客户端收到混合协议响应（SSE + JSON 错包），导致解析异常。
- 证据:
  - `internal/service/ai/handler/approval.go:72-90`

## 3.3 低危问题

### F9. 审批链路测试覆盖缺口明显

- 现象:
  - 未发现 `SubmitApproval` / `ResumeApproval` 的 handler/logic 单测。
  - 未发现审批中间件（NeedsApproval、interrupt/resume）测试。
- 风险:
  - 回归容易引入“可恢复但不持久化”等隐性错误，且难以及时发现。
- 证据:
  - `internal/service/ai/handler` 与 `internal/service/ai/logic` 无对应测试命中
  - `internal/ai/tools/middleware` 无审批中间件测试

## 4. 当前链路可用性判定

- 判定: **不可作为可靠生产闭环使用**
- 主要原因:
  - 状态机不闭合（waiting_approval 与 completed 冲突）
  - 权限与决策一致性不足（可绕过）
  - 前后端 API 契约不一致（功能不可达）
  - 恢复阶段持久化缺失（审计不完整）

## 5. 修复优先级建议（执行顺序）

### P0（必须先做）

1. 修复 `Chat` 中断收尾逻辑:
   - 检测 interrupt/`waiting_approval` 时不得 `Finish()`/`done`/`completed`。
   - run 状态应进入 `waiting_approval`（或等价状态）。
2. 修复 `ResumeApproval` 决策来源:
   - 恢复参数必须来自任务持久化状态，不信任 `input.Approved` 覆盖。
   - 已 `rejected/expired` 一律禁止恢复执行。
3. 为 `SubmitApproval` 增加 session/user 归属校验。

### P1（紧随其后）

1. 补齐审批任务创建链路:
   - interrupt 首次产生时创建 `ai_approval_tasks`（含 checkpoint/tool_call/session/run/user）。
2. 统一前后端审批 API 契约:
   - 要么后端补齐前端使用的 `/confirm` 与 `/chains/.../decision`；
   - 要么前端切到后端现有 `/submit` + `/resume`。
3. 修复恢复阶段持久化:
   - 复用实例方法 `l.consumeProjectedEvents(...)`；
   - 恢复结束后执行 run/message/projection 收敛。

### P2（完善与稳态）

1. 把 `tool_approval` 与 `run_state` 纳入 `run_event` 序列化持久化。
2. 修复 SSE handler 错误输出策略（流内发 `error` 事件，不走 JSON 包装）。
3. 补齐审批链路测试（中间件 + handler + logic + API 契约）。

## 6. 建议新增测试清单

1. `Chat` 在 interrupt 场景不发 `done`，run 进入 `waiting_approval`。
2. `SubmitApproval` 非会话归属用户提交应被拒绝。
3. `ResumeApproval` 在 `task.status=rejected/expired` 时必须拒绝。
4. `ResumeApproval` 事件应写入 run_event，并最终收敛 run/message 状态。
5. 前后端审批接口契约测试（至少一条端到端）。
6. 审批中间件测试:
   - 高风险工具触发 interrupt。
   - 审批通过后执行原工具；拒绝后返回拒绝消息。

## 7. 附：关键证据索引

- `internal/service/ai/logic/logic.go`
  - Chat 主流程收尾: 258-280
  - 投影事件持久化分发: 479-506
  - 事件序列化缺口: 571-636
  - SubmitApproval: 1214-1261
  - ResumeApproval: 1278-1404
- `internal/ai/runtime/project.go`
  - interrupt -> waiting_approval/tool_approval: 102-131
- `internal/service/ai/routes.go`
  - 后端审批路由: 27-30
- `web/src/api/modules/ai.ts`
  - 前端审批相关调用: 748-804
- `internal/dao/ai/approval_dao.go`
  - Create/UpdateStatus DAO: 24-27, 60-73

---

如需继续推进，下一步建议直接按 P0 项逐个落地改造，并在每项完成后补对应测试与回归验证。
