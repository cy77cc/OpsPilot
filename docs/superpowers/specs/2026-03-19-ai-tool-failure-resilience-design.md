# AI 工具失败韧性设计

Date: 2026-03-19

## Goal

让 AI 助手在工具调用失败时保持会话连续性。

核心原则：

- 除非是模型或 runtime 自身出现致命故障，否则不能因为单个 tool 调用失败而终止当前会话
- 普通工具失败必须作为结构化结果返回给模型和前端，而不是直接把 run 判定为失败
- 工具层不再尝试穷举命令安全性，尤其不再依赖“只读命令白名单”这类不可维护的策略

## Scope

In scope:

- AI tool 调用失败的语义重构
- AI runtime 对 tool error 与 fatal runtime error 的分类
- SSE 事件与持久化状态对“带错误完成”场景的表达
- 主机命令类工具的契约调整
- Agent prompt 对工具选择、失败恢复和降级回答的约束增强

Out of scope:

- 新增大量业务工具
- 重做审批系统
- 完整重写前端 AI 抽屉
- 对所有危险命令做完备静态检测

## Problem Statement

当前实现存在两个根问题。

第一，部分工具在能力边界上限制过死。例如主机命令工具把“只读”命令写成固定白名单，导致模型即使提出合理命令，也可能因为未被枚举而直接失败。

第二，runtime 对错误的终止语义过于激进。工具错误、参数错误、环境错误、审批拒绝等都容易被升级成整轮 run 失败，进而让本应可恢复的对话被提前打断。

这两个问题叠加后，会出现以下坏结果：

- 模型无法通过重试、换工具或调整策略来自恢复
- 前端只能看到 run 失败，而不是“某个工具失败但回答仍继续”
- 历史回放无法准确表达“失败发生在哪里，模型后来如何处理”
- 工具规则越补越多，但永远覆盖不完真实命令空间

## Product Decision

选定方向是：`tool failure becomes data`。

这意味着：

- 普通工具失败必须转成结构化 `tool_result` 错误结果
- 模型必须能够读取这些错误结果，并继续判断下一步动作
- 只有系统级致命错误才允许结束当前 run
- 会话本身不因单轮 run 失败而失效，用户始终可以继续下一条消息

## Failure Model

### 1. Recoverable Tool Error

以下错误属于可恢复的工具错误：

- 参数缺失或参数格式不匹配
- 工具执行返回非零退出码
- SSH、HTTP、数据库、K8s API 等下游依赖失败
- 目标资源不存在
- 权限不足
- 审批被拒绝
- 工具能力不足或输入前提不满足
- 命令执行成功但结果为空或不足以支持当前判断

这些错误都不应直接终止 run，而应转成结构化 tool result，继续进入模型上下文。

### 2. Fatal Runtime Error

只有以下错误属于致命 runtime 错误：

- 模型不可用或模型流中断且无法恢复
- checkpoint 或 resume 状态损坏
- 关键持久化失败导致消息或运行状态不一致
- 事件合同解析损坏，无法继续投影
- 内部 panic 或无法恢复的编排错误

这类错误可以结束当前 run，但不应让 session 作废。用户仍然可以继续发送下一条消息。

## Tool Contract Design

### 1. Execute-and-Report Instead of Allow-or-Block

工具层职责从“预先判断用户是否允许这么做”调整为“执行能力并报告结果”。

对命令类工具尤其如此。系统不再依赖具体命令白名单来判断一个命令是否安全，因为这在语义上不可完备，像 `cat`、`find`、`sed`、`python` 都可能同时具备读写或副作用能力。

因此，命令选择的主判断责任交给模型，系统只保留极窄的底线兜底：

- 明确会破坏平台宿主环境或 AI runtime 自身状态的行为仍可阻断
- 审批策略继续负责高风险写操作的人类确认
- 审计日志继续记录命令、目标、结果和审批链路

### 2. Result Shape

工具返回值应统一支持成功与失败两种结果，并尽量避免通过 Go error 直接中断流程。

推荐结果字段：

```json
{
  "ok": false,
  "error_type": "permission_denied",
  "retryable": false,
  "summary": "ssh authentication failed",
  "exit_code": 255,
  "stdout": "",
  "stderr": "permission denied",
  "raw": {}
}
```

约束：

- `ok=true/false` 是模型最先读取的分支信号
- `error_type` 必须稳定，可用于 prompt 指导恢复策略
- `retryable` 帮助模型决定是否重试、换工具或直接降级回答
- `summary` 提供简洁、可复述的错误摘要
- `raw` 允许保留工具特定上下文，但不能替代标准字段

### 3. Approval as a Tool Outcome

审批拒绝不是 fatal error，而是普通可恢复结果的一种。

模型拿到审批拒绝后，应继续回答，例如：

- 说明操作未获批准，因此没有执行
- 提供只读替代调查路径
- 请求用户改用低风险方案

## Prompt Strategy

模型 prompt 需要从“默认信任工具层规则”转成“模型主责判断 + runtime 保证连续性”。

### 1. Tool Selection Rules

prompt 必须明确要求模型：

- 优先选择最小影响、最容易验证的动作
- 优先探测，再变更
- 对命令类工具自行判断风险，不要依赖系统替你理解命令语义
- 不确定时先选择更保守、更可观测的命令

### 2. Recovery Rules

prompt 必须明确要求模型在工具失败后继续工作，而不是停止回答：

- 先判断错误属于参数问题、权限问题、环境问题还是目标不存在
- 能修正参数就修正参数
- 能换工具就换工具
- 无法继续执行时，基于已有证据给出带限制说明的答复
- 除非收到 fatal runtime failure 信号，否则不能把本轮会话当成终止

### 3. User-Facing Rules

prompt 必须要求模型在失败后保持可解释性：

- 说明哪个步骤失败
- 说明失败原因和影响范围
- 说明是否已经尝试替代路径
- 给出后续建议，而不是只输出“失败了”

## Runtime Design

### 1. Error Classification Layer

runtime 必须新增统一的错误分类层，将 agent 事件中的错误明确分流为：

- `recoverable_tool_error`
- `fatal_runtime_error`

分类结果决定后续行为：

- `recoverable_tool_error` -> 投影为 `tool_result(status=error)`，流继续
- `fatal_runtime_error` -> 投影为 `error`，当前 run 终止

### 2. Run State Semantics

run 不应再只有 `completed` 和 `failed` 两种粗粒度语义。

建议最少区分：

- `completed`
- `completed_with_tool_errors`
- `failed_runtime`

这能避免把“模型已成功兜底回答，但中间某个工具失败”误判为整轮失败。

### 3. Session Continuity

session 与 run 的边界必须明确：

- run 表示单轮 assistant 处理过程
- session 表示整个会话上下文

因此：

- run 可以失败
- session 不能因为单个 run 失败而进入不可用状态
- 当前轮 assistant message 即使标记为 error，后续新消息仍可正常追加到同一 session

## Stream Contract

前端和持久化必须接受“带错误完成”的流。

推荐事件语义：

- `tool_call`: 发起工具调用
- `tool_result`: 工具结果，允许 `status=success|error`
- `run_state`: 表示 run 当前状态，如 `running`, `waiting_approval`, `degraded`
- `done`: 本轮结束
- `error`: 仅用于 fatal runtime error

关键规则：

- 有 `tool_result(status=error)` 不代表 run 已结束
- `error` 才表示当前 run 发生致命错误
- `done` 可以出现在带 tool error 的 run 末尾

## Frontend Behavior

前端需要把“工具失败”与“run 致命失败”分开渲染。

预期表现：

- 工具失败显示在活动流中，状态为 error
- assistant 主回答仍可继续流出
- 历史回放可以看到失败工具及其错误摘要
- 当前 run 若为 `completed_with_tool_errors`，界面应显示“已完成，但有步骤失败”而不是整段红色失败态

## Data and Observability

### 1. Persistence

持久化层需要保存：

- tool error 次数
- run 最终状态
- 结构化 runtime 活动记录
- fatal runtime error 摘要

### 2. Metrics

监控应区分：

- tool call 总数
- tool error 总数
- fatal runtime failure 总数
- completed_with_tool_errors 总数

否则系统会误判整体稳定性。

## Architecture Boundaries

建议按职责拆成四个清晰单元：

1. Tool contract layer
负责工具输入输出契约、结构化结果和最小底线阻断。

2. Prompt policy layer
负责告诉模型如何选工具、如何处理失败、如何向用户解释。

3. Runtime classification layer
负责把事件中的错误分类成 recoverable tool error 或 fatal runtime error。

4. Projection and presentation layer
负责把结构化结果映射成 SSE、持久化 runtime 和前端展示状态。

每个单元都应能单独理解和测试，不应把策略、执行、投影揉进同一层。

## Risks

- 如果 prompt 不够强，模型可能在工具失败后反复撞同一个错误
- 如果工具仍大量返回原始 Go error，runtime 仍会被迫走中断路径
- 如果前端把任意 error 事件都当成终止信号，用户体验仍然会被打断
- 如果指标不拆分，运维会继续把 tool error 当成整体 AI 不可用

## Validation

设计完成后，后续实现计划至少要覆盖以下验证：

- 工具参数错误时，模型能读取错误结果并继续回答
- SSH 或外部 API 失败时，run 最终为 `completed_with_tool_errors` 而不是 `failed_runtime`
- 审批拒绝后，模型能输出替代方案
- fatal runtime error 仍能正确终止当前 run，并保留 session 可继续
- 历史回放能看到失败工具和最终回答同时存在

## Open Questions Resolved

- 是否继续依赖只读命令白名单：否
- 是否让模型看到工具失败：是
- 是否允许普通工具失败终止当前会话：否
- 哪些错误才允许终止当前 run：仅 fatal runtime error
