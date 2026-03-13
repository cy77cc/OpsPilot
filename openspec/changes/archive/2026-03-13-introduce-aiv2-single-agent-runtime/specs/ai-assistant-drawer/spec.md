## MODIFIED Requirements

### Requirement: 消息渲染

系统 SHALL 使用独立的消息块渲染机制渲染消息，并在富渲染失败时安全降级。随着后端演进到 AIOps control-plane 和 `aiv2` single-agent runtime，抽屉 MUST 以 turn-and-block 状态模型消费聊天与平台事件，而不再仅依赖单条纯文本消息输出。

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
- **AND** 支持 Markdown、工具、审批、证据、思考过程、推荐内容等独立块渲染
- **AND** assistant turn MUST use a stable information hierarchy so process, actions, approvals, and conclusion do not compete for the same visual role

#### Scenario: aiv2 execution is rendered as tool-call chain
- **GIVEN** assistant turn 来自 `aiv2` single-agent runtime
- **WHEN** 抽屉渲染工具调用过程
- **THEN** 执行过程 MUST 优先展示为工具调用链
- **AND** 不得再依赖“专家执行”文本日志作为主要执行视图
- **AND** 同一工具调用的重复更新 MUST 合并为单条持续更新记录

#### Scenario: historical replay stays summary-first
- **GIVEN** 用户查看恢复出来的历史 assistant 消息
- **WHEN** 该消息含有当前运行时采集的中间块
- **THEN** 默认视图 MUST 优先展示最终回答正文
- **AND** 历史视图 MAY 隐藏实时过程块和思维链
- **AND** Markdown 正文样式 MUST 与当前对话保持一致

#### Scenario: approval is rendered as a tool-centric gate
- **GIVEN** `aiv2` 在具体工具调用前触发审批
- **WHEN** 抽屉收到 approval block
- **THEN** UI MUST 将其渲染为针对当前工具动作的确认交互
- **AND** 批准或取消后等待确认态 MUST 立即退出
- **AND** 后续工具链和最终回答 MUST 在同一 turn 中继续显示
