## MODIFIED Requirements

### Requirement: AI Chat Stream Events (Core)

系统 MUST 为 AI 聊天主路径发送 ThoughtChain 原生 SSE 事件，并且用户可见的执行语义 MUST 由 chain/node 生命周期直接表达，而不是依赖 legacy `stage_delta`、`turn/block`、`phase/step` 或脱离链路的审批事件族进行重建。

主路径必需事件类型：
- `chain_started`
- `chain_meta`
- `node_open`
- `node_delta`
- `node_replace`
- `node_close`
- `chain_paused`
- `chain_resumed`
- `chain_completed`
- `chain_error`
- `heartbeat`

其中节点载荷 MUST 覆盖 `plan`、`step`、`tool`、`approval`、`replan`、`answer` 等用户可见语义。

#### Scenario: 助手过程语义通过 chain/node 生命周期表达
- **WHEN** 运行时开始一次 AI 聊天助手响应
- **THEN** 系统 MUST 发出 `chain_started`
- **AND** 规划、执行、工具、审批、重规划和最终答案 MUST 通过 node 生命周期事件表示
- **AND** 消费者 MUST 能仅通过 ThoughtChain 事件还原用户可见的完整过程

#### Scenario: 内部结构化载荷不泄露到助手文本
- **WHEN** 模型或运行时产生内部结构化载荷，如工具参数、计划 JSON 或重规划上下文
- **THEN** 该载荷 MUST 通过结构化 chain/node 事件路由或被过滤
- **AND** 用户可见的答案内容 MUST NOT 将原始内部结构化载荷当作普通文本暴露

#### Scenario: SSE 解析保持 markdown 原始空白
- **WHEN** 主路径以 SSE 发送用户可见 markdown 内容、表格、空行或缩进文本
- **THEN** 系统 MUST 保留 `data:` 行中的用户可见空白和换行语义
- **AND** 消费侧 MUST NOT 通过逐行 trim 改写 markdown 内容
- **AND** 仅允许移除协议前缀而不允许破坏正文布局

#### Scenario: 可见 chunk 仅解包完整 envelope
- **WHEN** 流式内容包含 `{"response": ...}` 之类的协议 envelope
- **THEN** 系统 MAY 仅在确认收到完整 envelope 后解包其中的 `response`
- **AND** 对于部分 JSON 片段、非 envelope 文本或混合内容 MUST 原样透传
- **AND** 系统 MUST NOT 因为激进归一化而吞掉用户可见 markdown 分隔

#### Scenario: 主路径不依赖 legacy 事件族
- **WHEN** 主聊天链路向前端流式输出
- **THEN** 系统 MUST NOT 要求前端依赖 `phase_started`、`phase_complete`、`plan_generated`、`step_started`、`step_complete`、`replan_triggered`、`approval_required`、`turn_started`、`block_open` 等 legacy 主路径事件恢复 ThoughtChain

#### Scenario: 心跳维持连接
- **WHEN** 流式会话持续超过 10 秒
- **THEN** 系统 MUST 每 10 秒发出 `heartbeat` 事件，包含时间戳

### Requirement: Checkpoint-based Resume

系统 MUST 使用 ThoughtChain 审批节点标识来恢复被审批暂停的执行，并在同一条 chain 上继续后续节点生命周期。

恢复输入 MUST 基于：
- `chain_id`
- `approval node_id`
- 审批决定

#### Scenario: 审批后在同一条 chain 上恢复
- **WHEN** 用户为等待中的审批节点提交批准决定
- **THEN** 系统 MUST 恢复特定 `chain_id + approval node_id` 的执行上下文
- **AND** 恢复的事件 MUST 继续在原始 chain 上发出
- **AND** 如果执行成功完成，恢复后的流 MUST 继续到后续 tool、replan 或 answer 节点再完成

#### Scenario: 拒绝后终止同一条 chain
- **WHEN** 用户为等待中的审批节点提交拒绝决定
- **THEN** 系统 MUST 不执行被门控的操作
- **AND** 原始 chain MUST 进入终止的拒绝或取消状态

## REMOVED Requirements

### Requirement: Phase Lifecycle Events
**Reason**: `phase_started` 和 `phase_complete` 不再是 ThoughtChain 主路径的协议来源，阶段语义改由 chain/node 生命周期直接表达。
**Migration**: 使用 `chain_started`、`node_open`、`node_delta`、`node_close` 和相关 node kind 来表达 planning、execution、replan、approval、answer 语义。

### Requirement: Plan Generated Event
**Reason**: 独立 `plan_generated` 事件会再次引入与 ThoughtChain 并行的计划协议。
**Migration**: 使用 `plan` 节点的 `node_replace` 或 `node_delta` 载荷传递计划步骤和摘要。

### Requirement: Step Lifecycle Events
**Reason**: 独立 `step_started` 和 `step_complete` 事件会重复表达已存在于 ThoughtChain 节点生命周期中的步骤状态。
**Migration**: 使用 `step` 节点和 `tool` 子语义的 node 生命周期表达步骤开始、执行结果和状态更新。

### Requirement: Replan Triggered Event
**Reason**: 独立 `replan_triggered` 事件会造成重规划语义脱离主 ThoughtChain 模型。
**Migration**: 使用 `replan` 节点表达重规划触发原因、旧计划关联和新计划摘要。

### Requirement: Enhanced Tool Events
**Reason**: 独立工具事件不再作为主路径叙事协议，工具语义应落入 ThoughtChain 节点模型。
**Migration**: 使用 `tool` 节点载荷表达工具参数摘要、结果摘要、时长和状态。
