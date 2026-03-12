## MODIFIED Requirements

### Requirement: 消息渲染

系统 SHALL 使用独立的消息块渲染机制渲染消息，并在富渲染失败时安全降级。随着后端演进到 AIOps control-plane，抽屉 MUST 以 turn-and-block 状态模型消费聊天与平台事件，而不再仅依赖单条纯文本消息输出。

#### Scenario: 用户消息渲染
- **GIVEN** 用户发送消息
- **WHEN** 消息添加到列表
- **THEN** 用户消息靠右显示
- **AND** 显示用户头像

#### Scenario: AI 消息渲染
- **GIVEN** AI 返回消息
- **WHEN** assistant turn 添加到列表
- **THEN** AI turn 靠左显示
- **AND** turn MUST 先被规范化为可渲染的消息块集合
- **AND** 支持 Markdown、状态、计划、工具、审批、证据、思考过程、推荐内容等独立块渲染
- **AND** assistant turn MUST use a stable information hierarchy so process, actions, approvals, and conclusion do not compete for the same visual role

#### Scenario: 富渲染块失败时安全降级
- **GIVEN** AI turn 包含富渲染块
- **WHEN** 某个块的渲染器执行失败
- **THEN** 仅失败块降级为安全文本或回退视图
- **AND** 同一 turn 中的其他块继续渲染
- **AND** AI 助手抽屉保持可用

#### Scenario: 流式输出
- **GIVEN** AI 正在生成回复
- **WHEN** 收到 turn 或 block 增量事件
- **THEN** 抽屉 MUST 实时更新对应 turn 中的目标块
- **AND** 最终回答文本块 MUST 显示打字效果
- **AND** 状态、工具和审批类块 MUST 使用状态更新而非逐字打印

#### Scenario: approval is rendered as pre-execution confirmation
- **GIVEN** assistant turn 命中一个需要审批的执行前 gate
- **WHEN** 抽屉收到 approval block
- **THEN** UI MUST 将其渲染为执行前确认交互，而不是“执行中断点”
- **AND** 用户在批准或取消后，等待确认态 MUST 立即退出
- **AND** 后续执行与总结内容 MUST 在同一 turn 中继续显示

#### Scenario: assistant turn uses four-layer information hierarchy
- **GIVEN** assistant turn 同时包含过程反馈、审批、执行状态和最终回答
- **WHEN** 抽屉渲染该 turn
- **THEN** 信息 MUST 按 `thought chain -> approval gate -> execution cards -> final answer` 的顺序展示
- **AND** 每一层 MUST 有明确职责，不得把相同内容在多个层级重复堆叠

#### Scenario: 历史对话中的 thought chain 默认折叠
- **GIVEN** 抽屉恢复一段已完成的历史 AI 对话
- **WHEN** assistant message 含有 thought chain
- **THEN** 历史消息中的 thought chain MUST 默认折叠
- **AND** 用户可以按消息粒度手动展开查看

#### Scenario: 当前流式对话展开活跃阶段
- **GIVEN** 当前 assistant turn 正在流式执行
- **WHEN** 抽屉渲染 thought chain
- **THEN** 当前活跃阶段 SHOULD 默认展开
- **AND** 已完成阶段 MAY 保持折叠以减少噪音

#### Scenario: thought chain 展示结构化执行细节
- **GIVEN** assistant turn 收到 step_update、tool_call 或 tool_result
- **WHEN** 抽屉更新 execute 阶段
- **THEN** thought chain MUST 展示步骤摘要以及结构化的工具调用与结果细节
- **AND** 同一 step/tool 的重复更新 MUST 合并为单条持续更新记录
- **AND** 不得将同一事件简单重复堆叠为多条文本日志

#### Scenario: 历史思维链不抢占结论阅读
- **GIVEN** 用户正在查看恢复出来的历史 assistant 消息
- **WHEN** thought chain 存在多阶段和多条细节
- **THEN** 默认视图 MUST 优先让用户先看到最终回答
- **AND** thought chain MUST 以折叠摘要形式出现，而不是完整展开占据主要阅读空间

## ADDED Requirements

### Requirement: 审批交互块 MUST be short-lived and coordinated with execution flow

审批交互块 MUST 在等待决策时提供 CTA，并在用户作出决策后让出主展示位置。

#### Scenario: approval CTA exits waiting state after decision
- **WHEN** 用户点击确认执行或取消
- **THEN** 审批交互块 MUST 立即进入提交中或终止反馈状态
- **AND** 成功后 MUST 不再保留可重复点击的等待确认 CTA
- **AND** 批准后的主内容区域 MUST 由执行状态和最终总结接管

#### Scenario: approval gate uses coordinated bubble styling
- **WHEN** 抽屉渲染等待审批的门控交互
- **THEN** UI MUST 使用与聊天块协调的浅色气泡样式
- **AND** 风险信息 MUST 通过 badge 和文案表达，而不是依赖大面积警报色背景
- **AND** 审批块视觉权重 MUST 低于最终回答但高于普通状态文案

#### Scenario: approval request failure remains actionable without restoring the old waiting card
- **WHEN** 审批结果提交失败
- **THEN** UI MUST 给出轻量失败反馈和可重试动作
- **AND** 不得恢复成无限可点击的初始等待审批表单

### Requirement: 重新生成 MUST 保持原用户消息时序

“重新生成” MUST 视为对同一用户问题的 assistant 重答，而不是新的用户提问。

#### Scenario: regenerate does not append duplicate user message
- **WHEN** 用户对某条 assistant 回答点击重新生成
- **THEN** 抽屉 MUST 保留原用户消息位置与内容
- **AND** 不得在消息列表中追加一条重复的用户消息
- **AND** 新的回答 MUST 作为原问题上下文下的再次作答显示

#### Scenario: regenerate preserves conversational continuity
- **WHEN** assistant 回答进入重新生成流程
- **THEN** UI MUST 维持原消息顺序不变
- **AND** 原回答 MAY 被原地替换或标记为上一版
- **AND** 默认视图 MUST 只强调最新生成的回答

### Requirement: 链路展示 MUST prioritize process clarity over raw log volume

流程展示 MUST 明确区分阶段状态、执行细节和最终结论，避免内部噪音覆盖用户理解。

#### Scenario: final answer, thought chain, and execution cards have clear roles
- **WHEN** assistant turn 同时包含阶段状态、执行细节和最终回答
- **THEN** thought chain MUST 主要承担过程反馈
- **AND** 工具/执行卡片 MUST 承担动作状态和结果展示
- **AND** 最终回答 MUST 承担结论与建议
- **AND** 相同信息不得在三个区域中重复堆叠

#### Scenario: final answer stays concise after rich execution flow
- **WHEN** 上方已经展示了完整的审批和执行过程
- **THEN** 最终回答 MUST 优先输出结论、关键事实和下一步建议
- **AND** 不得再次逐条复述完整的工具日志或审批交互文案
