# AI 审批链路全量事件总线化设计

## 背景与目标

当前审批链路已经具备 checkpoint 持久化、审批提交与后台恢复能力，但在“审批请求事件实时可见”“多端一致状态”“恢复过程可观测”上存在语义分散和实现耦合问题。为避免 `checkpoint 已保存但审批事件不可见` 这类问题反复出现，方案将审批与恢复流程统一为事件驱动架构。

目标如下：

- 审批请求强实时可见：发起审批后 `<1s` 内出现审批卡片。
- 审批决策与恢复全程可追踪：同一 `approval_id` 可串联请求、决策、恢复、终态。
- 前端交互满足产品要求：乐观更新；批准后展示“已批准，恢复中”；失败不回退为待审批。
- 多端一致：会话窗口与审批中心消费同一事件语义。

非目标：

- 本次不重写 AI Agent 规划/执行逻辑。
- 本次不改动审批策略判定算法（仅改传输与状态流）。

## 约束与成功标准

### 约束

- 兼容现有 `submitApproval` API 语义。
- 与现有 `Run/Session/ApprovalTask` 数据结构共存，分阶段迁移。
- 所有事件处理必须幂等，可重试。
- 事件总线实现在本期冻结为“DB outbox + 进程内发布订阅总线”，不引入外部 MQ。
- SSE 回放游标在本期冻结为 `last_event_id`，不采用 `since_ts`。

### 成功标准

- 发起审批后，前端 `<1s` 可见 `waiting_approval`。
- 点击批准后，前端立即进入 `approved_resuming`。
- 恢复成功/失败可实时或准实时展示；失败时保持“已批准”事实。
- 事件链路可通过 `event_id` / `approval_id` 端到端排障。

## 总体架构

采用“业务写模型 + 事务 outbox + 统一事件总线 + 投影/推送消费者”四层结构：

1. 业务写模型层
- 负责审批任务状态迁移（`pending -> approved/rejected/expired -> resuming -> resumed/failed`）。
- 在同一事务内写入 `approval_task` 与 `outbox_event`。

2. 事务 outbox 层
- 作为唯一可靠事件源（source of truth for delivery）。
- 保证事件至少一次投递基础。

3. 事件总线层
- 统一承载审批与恢复领域事件。
- 为前端 SSE 桥接、投影构建、通知中心提供同源事件。

4. 读模型/推送层
- SSE 桥接器：按会话或审批主题推送事件。
- 运行态投影器：构建 run projection 与历史回放视图。
- 审批中心投影器：构建 pending/processing/completed 列表。

## 统一事件模型

统一信封字段：

- `event_id`：全局唯一事件 ID。
- `event_type`：领域事件类型。
- `occurred_at`：事件发生时间（UTC）。
- `sequence`：同一 `run_id` 内的单调递增序号（由服务端分配）。
- `version`：事件 schema 版本。
- `run_id`、`session_id`、`approval_id`、`tool_call_id`：关联键。
- `payload`：业务载荷。

核心事件类型：

- `ai.approval.requested`
- `ai.approval.decided`
- `ai.approval.expired`
- `ai.run.resuming`
- `ai.run.resumed`
- `ai.run.resume_failed`
- `ai.run.completed`

语义要求：

- 事件可重放：消费者仅基于 `last_event_id` 回放缺失事件。
- 幂等消费：以 `(event_id)` 或 `(approval_id + event_type + version)` 去重。
- 有序性：同一 `approval_id` 内按 `occurred_at + sequence` 保序。

## 审批状态机

审批任务状态：

- `pending`
- `approved`
- `rejected`
- `expired`
- `resuming`
- `resumed`
- `resume_failed_retryable`
- `resume_failed_terminal`

转换规则：

- `pending -> approved|rejected|expired`
- `approved -> resuming`
- `resuming -> resumed|resume_failed_retryable|resume_failed_terminal`

并发与冲突：

- 同一 `approval_id` 仅允许一次决策写入；重复提交返回当前快照。
- 决策后不可回到 `pending`。
- 已批准后恢复失败不可被展示为“待审批”。

## 前端交互与状态投影

前端统一状态：

- `waiting_approval`
- `submitting`
- `approved_resuming`
- `approved_retrying`
- `approved_done`
- `approved_failed_terminal`
- `rejected`
- `expired`

交互策略：

- 点击批准：本地 `submitting -> approved_resuming`（乐观更新）。
- 后端冲突响应：用服务端快照覆盖。
- 收到 `ai.run.resuming/resumed/resume_failed`：驱动状态推进。
- “已批准”事实不可逆，失败仅改变恢复子状态。
- `submitApproval` 请求超时/5xx/网络失败：保持 `submitting` 最长 3 秒，超时后进入 `approved_resuming` 并标注“结果确认中”；随后必须以服务端事件或轮询快照收敛到最终状态。

状态进入/退出条件：

- `approved_retrying`：收到 `ai.run.resume_failed` 且 `retryable=true` 时进入。
- `approved_retrying -> approved_resuming`：收到下一次 `ai.run.resuming`。
- `approved_retrying -> approved_failed_terminal`：收到 `ai.run.resume_failed` 且 `retryable=false` 或达到最大重试次数。
- `approved_resuming -> approved_done`：收到 `ai.run.resumed` 或 `ai.run.completed` 且结果成功。

多端一致：

- 会话窗口与审批中心订阅同一事件语义。
- 任一端决策后其他端 `<1s` 同步。

## 数据流设计

### A. 审批请求

1. 工具触发审批中断。
2. 后端创建 `approval_task(pending)`。
3. 同事务写入 `outbox(ai.approval.requested)`。
4. dispatcher 派发到总线。
5. SSE 桥接器推送前端，展示审批卡片。

### B. 审批决策

1. 前端调用 `submitApproval`。
2. 后端原子更新任务状态并写入 `outbox(ai.approval.decided)`。
3. dispatcher 派发；前端收到后对齐状态。

### C. 恢复执行

1. Worker 抢占租约并标记 `resuming`，发 `ai.run.resuming`。
2. 执行 `ResumeWithParams`。
3. 成功：写 `resumed` + `ai.run.resumed`，最终写 `ai.run.completed`。
4. 可重试失败：写 `resume_failed_retryable` + `ai.run.resume_failed`（附重试计划）。
5. 终止失败：写 `resume_failed_terminal` + `ai.run.resume_failed`。

### D. 审批过期

1. 过期扫描器发现 `pending && expires_at < now`。
2. 原子更新为 `expired` 并写入 `outbox(ai.approval.expired)`。
3. dispatcher 派发后，前端审批卡片转为 `expired` 且不可操作。

## 接口契约

### Outbox Dispatcher 接口

- 输入：`ai_approval_outbox_events` 中 `status in (pending,processing-stale)` 的记录。
- 输出：发布统一事件到进程内总线；成功后标记 `done`，失败写 `retry_count/next_retry_at`。
- 幂等键：`event_id`。

### Event Bus 接口

- `Publish(event Envelope) error`
- `Subscribe(topic string, consumer Consumer)`
- 保证：至少一次投递；并且同一 `aggregate_id`（`approval_id` 或 `run_id`）内按 `sequence` 投递。

### Projection Upsert 规则

- key：`(projection_type, aggregate_id)`，例如 `(run_projection, run_id)`、`(approval_projection, approval_id)`。
- 冲突策略：仅接受 `sequence` 更大的事件；旧事件丢弃。

### SSE 回放契约

- 请求参数：`last_event_id`（可选）。
- 响应行为：
- `last_event_id` 缺失：先返回当前审批/运行快照，再从连接确认时刻开始实时推送。
- `last_event_id` 存在：先补发缺失事件，再切换实时流。
- 错误码：
- `AI_STREAM_CURSOR_EXPIRED`：游标过旧无法回放，客户端需全量刷新快照。

## 错误处理与可靠性

- 派发失败：outbox 保持 `pending/processing`，指数退避重试。
- 消费重复：消费者幂等去重。
- Worker 崩溃：租约超时后可被其他实例接管。
- SSE 短断线：前端仅以 `last_event_id` 回放补齐。
- 数据不一致保护：写模型状态迁移采用条件更新（CAS）。

## 观测与审计

指标：

- `approval_request_to_visible_latency_ms`
- `approval_decision_to_resuming_latency_ms`
- `resume_success_rate`
- `resume_retry_count`
- `outbox_lag_seconds`

日志与追踪：

- 每个关键节点打印 `event_id/run_id/session_id/approval_id`。
- 审批审计日志记录决策人、决策时间、原因、结果。

## 测试策略

后端：

- 事务原子测试：状态更新与 outbox 双写一致。
- 幂等测试：重复派发、重复消费、重复 submit。
- 顺序测试：同 `approval_id` 事件序列不乱序。
- 故障恢复测试：dispatcher/worker 重启后续跑无丢失。

前端：

- 乐观更新状态流。
- 冲突覆盖逻辑。
- “已批准，恢复中/重试中/终止失败”展示。
- 多端同步可见性。

端到端：

- 从 `tool_approval` 到 `submitApproval` 到恢复完成/失败全链路回归。

## 迁移计划

里程碑 1（写模型与事件信封）

- 引入统一事件 schema 与 outbox 扩展字段。
- 落地 `sequence` 分配与状态迁移函数。

里程碑 2（分发与 SSE 桥接）

- 保留旧路径，同时写新领域事件。
- 新增 dispatcher、总线发布订阅、`last_event_id` 回放。

里程碑 3（前端状态机收敛）

- 接入 `approved_resuming/retrying/failed_terminal` 明确状态迁移。
- 保留旧事件兼容读取作为兜底。

里程碑 4（切换与收敛）

- SSE 与审批中心完全基于新事件。
- 观测稳定后下线旧直推逻辑与兼容分支。

## 风险与缓解

- 风险：双写期事件重复。
  缓解：消费端幂等键 + 去重缓存。

- 风险：状态机边界不清导致非法迁移。
  缓解：集中状态迁移函数 + 单元测试覆盖全部迁移。

- 风险：多端时钟漂移影响顺序。
  缓解：服务端 sequence 字段作为排序主键。

## 决策记录

- 本期事件总线：`DB outbox + 进程内总线`。
- 本期回放游标：`last_event_id`。
- 外部 MQ 与 `since_ts` 方案留作后续优化议题，不阻塞本期计划。
