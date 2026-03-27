# AI HITL 流恢复重写设计

日期：2026-03-26  
状态：Proposed  
范围：AI Chat HITL 审批链路（设计阶段，不改代码）

## 1. 背景与问题

当前 AI HITL 链路的核心用户问题不是“审批能力不存在”，而是“审批触发后当前聊天消息无法自然继续”。

调研当前实现后，确认已有以下事实：

1. Eino `interrupt -> checkpoint -> ResumeWithParams` 底层机制已接入。
2. AI chat 在命中审批后会进入 `waiting_approval`，当前 `/ai/chat` SSE 请求结束。
3. 审批通过后的恢复由 `ApprovalWorker` 在后台完成，恢复后的事件会写入 `ai_run_events`。
4. 前端当前缺少围绕同一 `run_id` 的自动续连与事件续播，因此用户体感为“流断了、没后续”。

这说明问题主要位于平台层协议与前端会话状态机，而不是 Eino 中断恢复机制本身。

## 2. 调研结论

### 2.1 Eino 官方模型

基于 Eino v0.8.4 文档与本地模块源码，确认其推荐模型为：

1. 组件在需要外部输入时触发 `Interrupt` / `StatefulInterrupt` / `CompositeInterrupt`。
2. `Runner` 捕获 interrupt 后保存 checkpoint，并结束当前执行流。
3. 外部系统使用 `ResumeWithParams` 进行 targeted resume。
4. 被明确 targeting 的 leaf interrupt 点使用 `GetResumeContext` 消费恢复数据。
5. 未被 targeting 的 leaf interrupt 点必须重新 interrupt 以保留状态。

该模型天然支持 HITL，但不保证“原 HTTP/SSE 连接一直保持到审批结束”。

### 2.2 对当前代码的影响

当前实现中：

1. 命中审批后结束当前 chat SSE，本身不违背 Eino 语义。
2. 真正缺失的是“审批后如何把恢复后的 run 事件重新送回当前 assistant 消息”。
3. 产品目标若要求“审批后当前聊天消息继续自然往下长出来”，必须在平台层补齐 run 级续播协议。

## 3. 目标

### 3.1 主目标

1. 审批通过后，当前 assistant 消息继续自然增长，不创建新的 assistant 消息。
2. 首次执行与恢复执行统一归属于同一个 `run_id`。
3. 当前 chat SSE 中断后，前端自动续连同一 run 并继续消费恢复后的事件。
4. 历史加载、刷新页面、切换会话后，仍能准确恢复 `waiting_approval` / `resuming` / `completed` 状态。
5. 清理当前存在但不再作为主路径的旧 HITL 恢复代码与旧协议心智。

### 3.2 非目标

1. 本阶段不重写 Eino 中间件或 checkpoint 存储实现。
2. 本阶段不设计新的审批中心产品界面。
3. 本阶段不引入新的 run stream HTTP 路由，优先复用 `/api/v1/ai/chat` 的 replay 语义。
4. 本阶段不做与 HITL 无关的 AI runtime UI 改造。

## 4. 方案选择

本设计采用：

`中断后结束当前 SSE + 前端自动续连同一 run`

而不采用：

1. 在原 chat SSE 连接中阻塞等待审批结束。
2. 新建独立 `/runs/:id/stream` 作为唯一恢复通道。

原因：

1. 与 Eino 原生 `interrupt/resume` 模型一致，避免逆着框架设计实现“悬挂连接等待审批”。
2. 能满足“同一条 assistant 消息继续长”的产品目标。
3. 相比新开 run stream 路由，改动面更可控，且可复用现有 replay 能力。
4. 服务重启、worker 异步恢复、审批跨端提交等场景下容错更高。

## 5. 设计原则

1. `run_id` 是恢复链路唯一主身份，审批前后不变。
2. `clientRequestId` 是 chat 续连身份，前端必须稳定持有直到 run 收敛。
3. `event_id` 是唯一去重游标，前端不得基于文本或事件类型做重放去重。
4. `tool_approval` 解释“为什么停”，`run_state` 表达“当前生命周期状态”，`delta/tool_result` 继续产出内容。
5. `waiting_approval` 不是终态；进入该状态时不得发送 `done`。

## 6. 目标架构

### 6.1 后端职责

后端分三层职责：

1. Eino 执行层  
   继续使用现有 `interrupt -> checkpoint -> ResumeWithParams`。

2. Run 事件层  
   首次执行和恢复执行都写入同一个 `run_id` 的 `ai_run_events`，事件序号严格单调追加。

3. Chat / replay 层  
   `/api/v1/ai/chat` 既负责首次发起，也负责在 `clientRequestId + lastEventId` 场景下 replay 并继续 tail 同一 run 的后续事件。

### 6.2 前端职责

前端分两层职责：

1. 消息层  
   当前 assistant 消息绑定到稳定的 `run_id + clientRequestId`，审批前后都更新同一个消息对象。

2. 连接层  
   `tool_approval` 到达后进入挂起态；无论审批是在当前端、另一端，还是通知中心中完成，前端都必须基于同一个 `run_id` 触发自动续连，并使用旧的 `clientRequestId + lastEventId` 发起续连。

### 6.3 历史恢复凭证

为支持“设备 A 挂起、设备 B 打开历史会话后继续自然恢复”的场景，恢复凭证不能只存在前端本地内存。

服务端必须持久化并向历史读取接口返回最小恢复凭证，至少包括：

1. `run_id`
2. `client_request_id`
3. `latest_event_id`
4. `approval_id`（若当前处于待审批或恢复失败态）
5. `run_status`
6. `resumable` / `can_retry_resume` 之类的显式布尔标记

约束：

1. 会话详情、历史消息、run projection 任一能恢复当前 assistant 消息状态的读取接口，都必须能够提供上述恢复凭证，不能要求前端依赖旧标签页内存。
2. 前端在加载历史会话时，若发现最后一条 assistant 消息处于 `waiting_approval` 或 `resume_failed_retryable`，必须使用服务端返回的恢复凭证重建续连上下文。

## 7. 生命周期状态机

用户可见状态收敛为：

1. `running`
2. `waiting_approval`
3. `resuming`
4. `completed`
5. `cancelled`
6. `resume_failed_retryable`
7. `failed`

状态迁移规则：

1. `running -> waiting_approval`
2. `waiting_approval -> resuming`
3. `resuming -> running`
4. `running -> completed`
5. `waiting_approval -> cancelled`
6. `resuming -> resume_failed_retryable`
7. `resume_failed_retryable -> resuming`
8. `running -> failed`
9. `resuming -> failed`

约束：

1. `waiting_approval` 期间 assistant 消息不得进入 `done` 或 `error` 终结渲染。
2. 审批拒绝或过期后必须显式进入终态，不允许悬空。
3. `resume_failed_retryable` 表示恢复链路失败，但原审批事实已存在，不得回退为 `pending`。
4. 终态事件序列必须确定，不允许同一终态在不同路径下出现不同组合。
5. 所有用户可见终态都必须有对应的终态 `run_state`，`error` 不能单独承担终态语义。

### 7.1 终态事件序列

为避免 reducer 与 replay 语义分叉，终态序列固定如下：

1. 批准后成功完成：
   - 先发送 `run_state=status=resuming`
   - 恢复后发送 `run_state=status=running`
   - 继续发送 `delta/tool_result`
   - 最后发送 `run_state=status=completed`
   - 最后发送 `done`
2. 审批拒绝：
   - 发送 `run_state=status=cancelled`
   - 不发送 `error`
   - 不发送 `done`
3. 审批过期：
   - 发送 `run_state=status=cancelled`
   - 可在 payload 中带 `reason=approval_expired`
   - 不发送 `error`
   - 不发送 `done`
4. 恢复失败但可重试：
   - 发送 `run_state=status=resume_failed_retryable`
   - 不发送 `done`
   - 仅在无法恢复消费链路且需要显式错误展示时发送 `error`
   - 后续仅允许进入 `run_state=status=resuming` 或人工终结的 `run_state=status=failed`
5. 非审批相关运行时致命错误：
   - 先发送 `run_state=status=failed`
   - 发送 `error`
   - 不发送 `done`
   - 不再发送新的非终态 `run_state`

## 8. 事件协议设计

### 8.1 主流程事件

保留以下对外主流程事件：

1. `meta`
2. `delta`
3. `tool_call`
4. `tool_approval`
5. `tool_result`
6. `done`
7. `error`

规则：

1. 命中审批时必须发送 `tool_approval`。
2. 进入 `waiting_approval` 时不得发送 `done`。
3. 恢复后继续使用 `delta/tool_result` 向同一 assistant 消息追加内容。
4. 只有 run 真正收敛时才发送 `done`。

### 8.2 生命周期事件

统一保留 `run_state` 作为状态同步事件，至少覆盖：

1. `running`
2. `waiting_approval`
3. `resuming`
4. `completed`
5. `cancelled`
6. `resume_failed_retryable`
7. `failed`

规则：

1. 前端消息状态优先由 `run_state` 驱动。
2. `tool_approval` 是审批原因事件，不替代 run 生命周期状态。
3. 已存在的 `ai.run.resuming / ai.run.resumed / ai.run.resume_failed / ai.run.completed` 可以短期保留，但对 chat UI 必须映射到同一套 `run_state` 心智。
4. `done` 只允许出现在 `run_state=status=completed` 之后。
5. `cancelled`、`failed` 与 `resume_failed_retryable` 为当前状态时，不允许继续沿用旧的 `running` UI 展示。
6. `cancelled` 与 `failed` 为终态时，不允许再追加 `delta/tool_result`。

### 8.3 replay 约束

1. 同一 `run_id` 下，恢复后的事件必须接续审批前最后一个 `seq/event_id`。
2. replay 不得重排，不得跨 run 混流。
3. 历史 replay 必须能重建 `waiting_approval` 对应的当前消息状态。
4. 基于 `last_event_id` 的 replay 必须是严格增量 replay，不得返回该游标之前的历史事件。
5. 前端 reducer 允许做事件去重兜底，但后端不得把重复下发旧事件当作常规行为。

### 8.4 reconnect / tail 语义

`POST /api/v1/ai/chat` 在 `clientRequestId + lastEventId` 场景下，不只是“查库回放当前快照”，而必须具备 run 级 tail 语义。

规则：

1. 若 `last_event_id` 之后已有新事件，接口先按序 replay 这些事件。
2. 若 `last_event_id` 之后暂时没有新事件，但该 run 仍处于非终态（尤其是 `waiting_approval`、`resuming`、`running`），接口不得立即返回终止响应。
3. 上述情况下，接口必须保持连接并阻塞等待，直到：
   - 有新事件可发送；
   - run 进入终态；
   - 或达到明确的 server-side idle timeout。
4. 若因 idle timeout 结束连接，前端必须可再次使用同一恢复凭证发起续连，且不会丢事件。

## 9. 前端续连设计

### 9.1 首次请求

首次发消息时：

1. 生成稳定的 `clientRequestId`。
2. 将 `clientRequestId` 与当前 assistant 消息对象绑定。
3. 在流式消费过程中持续记录最新 `event_id` 作为 `lastEventId`。
4. 将 `run_id`、`clientRequestId`、`lastEventId` 持久化到当前会话消息的本地状态，供刷新和跨组件恢复使用。

说明：

1. 前端本地状态只是加速层，不是恢复真源。
2. 真正的跨端/跨页面恢复真源是服务端返回的恢复凭证。

### 9.2 收到审批

收到 `tool_approval` 后：

1. 消息进入 `waiting_approval`。
2. 保留当前正文和 runtime，不创建新消息。
3. 当前请求对象结束后，不将消息视为失败或完成。
4. 在本地注册该 `run_id` 的待恢复监听状态。

### 9.3 审批通过后的自动续连

审批通过后：

1. 前端不得仅刷新历史列表。
2. 前端必须自动重新调用 `POST /api/v1/ai/chat`。
3. 重新调用时携带原始：
   - `sessionId`
   - `clientRequestId`
   - `lastEventId`
4. 新连接消费 replay + tail 后续事件，并继续更新当前 assistant 消息。

### 9.4 跨端审批触发

审批可能在以下位置完成：

1. 当前 chat 窗口
2. 同用户另一页面 / 另一浏览器标签页
3. 通知中心或审批中心

因此自动续连触发条件必须统一为：

1. 当前端本地审批提交成功；
2. 或当前端收到针对同一 `run_id` / `approval_id` 的外部审批状态变化信号；
3. 且当前消息仍处于 `waiting_approval`，本地尚未进入 `resuming/completed/cancelled`。

外部审批状态变化信号可以由既有通知或轮询机制承载，但对 chat 模块的语义要求固定为：

1. 旁路信号只负责触发自动续连。
2. 真正的恢复内容仍必须来自 `POST /api/v1/ai/chat` 的 replay + tail。

### 9.5 历史会话打开时的恢复

当用户重新打开历史会话时：

1. 若最后一条 assistant 消息为 `completed/cancelled/failed`，只渲染历史，不主动续连。
2. 若最后一条 assistant 消息为 `waiting_approval`，渲染挂起态，并监听审批变化；一旦审批在任意端完成，使用服务端恢复凭证自动续连。
3. 若最后一条 assistant 消息为 `resume_failed_retryable`，渲染可重试态，并允许显式触发恢复重试接口。
4. 若最后一条 assistant 消息为 `resuming/running` 且 run 未终结，应直接使用恢复凭证重新附着事件流。

### 9.6 去重与一致性

1. 所有 replay 与续播去重基于 `event_id`。
2. 消息 identity 必须绑定 `run_id`，不能只依赖“当前最后一条 assistant 消息”。
3. 审批期间切换会话、刷新页面、重新打开抽屉后，若当前 run 未收敛，重新载入时必须恢复挂起态和最新游标。
4. 若当前 run 已进入 `resume_failed_retryable`，前端只展示可重试状态，不自行发起无限自动重试。

## 10. 后端重写范围

### 10.1 保留的主链路

保留并强化如下主链路：

1. `POST /api/v1/ai/chat`
2. `SubmitApproval`
3. `ApprovalWorker`
4. `ResumeWithParams`
5. `ai_run_events` 持久化
6. `last_event_id` replay

### 10.2 需要补齐的能力

1. chat replay 从“仅回放历史”提升到“回放后可继续 tail 同一 run 的新事件”。
2. 恢复后的 worker 事件与 chat 续连消费闭环需要显式设计，而不是只写库不回流。
3. `waiting_approval -> resuming -> running` 的状态投影必须统一落在 projection 与当前消息 runtime 中。
4. `resume_failed_retryable` 的后续重试由后端 worker / 显式用户动作拥有，前端不拥有写侧重试职责。
5. 历史读取接口必须返回恢复凭证，支撑跨端、跨页面恢复。

### 10.3 `resume_failed_retryable` 重试归属

`resume_failed_retryable` 不是新的审批态，而是“审批已决但恢复执行失败，可再次尝试恢复”的中间失败态。

规则：

1. 重试所有权在后端，不在 chat 前端。
2. 后端可以基于既有 worker 重试机制再次尝试恢复，并在尝试开始时发送 `run_state=status=resuming`。
3. 前端不做无限自动重试；仅在收到新的状态变化或用户显式触发恢复动作时重新附着事件流。
4. 若后端判定不可再重试，必须从 `resume_failed_retryable` 收敛到 `failed` 或 `cancelled`，不得长期停留。

### 10.4 人工重试入口

若 UI 展示 `resume_failed_retryable`，则系统必须提供显式后端触发入口，供用户手动再次尝试恢复。

要求：

1. 后端必须暴露明确的 retry-resume 接口，语义上等价于“重新调度该 run 的恢复动作”。
2. 该接口不得创建新的 `run_id`，不得创建新的 assistant 消息。
3. 该接口的幂等键与运行幂等语义必须独立定义，避免重复触发多次恢复。
4. 成功受理后，run 进入 `resuming`，前端重新附着到同一事件流。
5. 若 retry 被拒绝（例如 run 已终态、权限不足、已在恢复中），接口必须返回明确业务错误。

本设计不强制该接口最终挂在哪个资源路径下，但必须是显式 API，而不是依赖前端伪造一次新的 chat 请求来触发 worker。

## 11. 需要清理的旧代码与旧心智

本次设计明确要求把以下内容列入清理范围。

### 11.1 旧恢复入口

`Logic.ResumeApproval(...)` 属于旧的“直接流式恢复”心智，不再作为主路径。

处理策略：

1. 在新自动续连主链路合入同一个变更后，立即标记为 deprecated，不允许新调用方接入。
2. 在后续一个 cleanup 变更中删除，退出条件是：主仓所有路由、前端调用与测试不再引用该入口。
3. 所有审批恢复统一走 `SubmitApproval -> worker -> ResumeWithParams`。

### 11.2 旧事件命名混用

文档和代码中混用的：

1. `approval_required`
2. `tool_approval`
3. `approval_decided`
4. `ai.approval.*`

处理策略：

1. 对 chat UI 公开契约收敛到一套主命名。
2. 兼容窗口仅覆盖在途 run 与历史 replay 读取，不允许新写入继续产出旧事件名。
3. cleanup 退出条件是：写入侧全部 canonical，消费侧测试仅保留 canonical 断言，旧事件名适配层被删除。

### 11.3 旧前端行为

当前审批后主要依赖：

1. 通知中心提交审批
2. `ai-approval-updated` 触发列表/历史刷新

这套逻辑只能作为辅助刷新机制，不得再承担恢复主链路。

处理策略：

1. 保留通知刷新作为旁路同步。
2. 主链路改为自动续连当前 run。
3. cleanup 退出条件是：审批成功后只刷新历史、不发起续连的主逻辑被删除；`ai-approval-updated` 仅用于旁路同步，不再作为恢复主链路。

### 11.4 旧状态误判

所有把以下情况误当终态的逻辑都应清理：

1. SSE 请求结束即消息结束
2. `waiting_approval` 视为 `done`
3. 无 `done/error` 时直接把消息置为失败

## 12. 风险与缓解

1. replay + tail 实现复杂，容易重复发事件  
缓解：强制 `event_id` 去重，测试覆盖重复 replay 与续连场景。

2. 前端消息 identity 绑定不稳定导致串消息  
缓解：统一以 `run_id + clientRequestId` 作为当前消息会话身份。

3. 旧兼容分支长期残留，协议继续漂移  
缓解：在实施计划中单列 cleanup 阶段与删除清单。

4. 恢复阶段若仅写库不通知前端，体验仍会断  
缓解：把“chat replay 后可继续 tail 新事件”列为硬约束，而不是可选增强。

## 13. 验收标准

1. 审批触发后，当前 assistant 消息进入 `waiting_approval`，但不结束。
2. 审批通过后，当前 assistant 消息继续自然增长，不创建新的 assistant 消息。
3. 审批拒绝或过期后，当前 assistant 消息进入明确终态，不悬空。
4. 刷新页面、切换会话或跨设备打开历史会话后，仍能恢复 pending approval / resumed run 的正确状态。
5. 恢复执行不会重复渲染 replay 事件。
6. 旧恢复入口和旧命名分支有明确退役计划，不再作为主路径。
7. `resume_failed_retryable` 有明确的用户可见重试动作和对应后端接口。

## 14. 测试计划

### 14.1 后端

1. interrupt 后 run 进入 `waiting_approval`，不发送 `done`。
2. worker 恢复后同一 `run_id` 继续追加事件序列。
3. `last_event_id` replay 后可以继续接收新事件，且在 run 非终态但暂无新事件时保持 tail 阻塞语义。
4. 拒绝/过期场景能正确收敛 run 状态。
5. 重复提交审批不会重复执行写操作。
6. 历史读取接口返回恢复凭证。
7. retry-resume 接口可以重新调度 `resume_failed_retryable` run。

### 14.2 前端

1. 收到 `tool_approval` 后当前消息进入挂起态。
2. 审批通过后自动续连，不创建新 assistant 消息。
3. replay 事件不会重复追加正文或 runtime activity。
4. 刷新页面或跨设备打开历史会话后，能利用服务端恢复凭证恢复当前消息的 `waiting_approval` / `resuming` / `completed` 状态。
5. `resume_failed_retryable` 显示明确重试动作，点击后调用显式后端接口。

### 14.3 端到端

1. 用户发起高风险工具调用。
2. 当前消息进入 `waiting_approval`。
3. 用户批准。
4. 当前消息进入 `resuming` 并继续增长正文。
5. 最终收到 `done` 并正常收敛。
6. 设备 A 挂起后关闭页面，设备 B 打开历史会话仍可恢复续连。

## 15. 实施顺序

1. 先收敛协议与状态机。
2. 再统一后端恢复主链路与 replay/tail 行为。
3. 然后改前端自动续连与消息 identity。
4. 最后清理旧恢复入口、旧事件命名、旧列表刷新主逻辑。
