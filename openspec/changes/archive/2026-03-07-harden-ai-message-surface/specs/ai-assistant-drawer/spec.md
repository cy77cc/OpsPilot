## ADDED Requirements

### Requirement: AI surface failures MUST be isolated from the application shell
The system MUST keep the main application shell usable when the AI assistant drawer or its lazy-loaded surface fails to initialize.

#### Scenario: AI drawer initialization failure
- **WHEN** the AI assistant drawer surface throws during initialization
- **THEN** the main application shell remains interactive
- **AND** the AI entry shows a local fallback state instead of blanking the page

## MODIFIED Requirements

### Requirement: 消息渲染
系统 SHALL 使用独立的消息块渲染机制渲染消息，并在富渲染失败时安全降级。

#### Scenario: 用户消息渲染
- **GIVEN** 用户发送消息
- **WHEN** 消息添加到列表
- **THEN** 用户消息靠右显示
- **AND** 显示用户头像

#### Scenario: AI 消息渲染
- **GIVEN** AI 返回消息
- **WHEN** 消息添加到列表
- **THEN** AI 消息靠左显示
- **AND** 消息先被规范化为可渲染的消息块
- **AND** 支持 Markdown、代码、思考过程、推荐内容等独立块渲染

#### Scenario: 富渲染块失败时安全降级
- **GIVEN** AI 消息包含富渲染块
- **WHEN** 某个块的渲染器执行失败
- **THEN** 仅失败块降级为安全文本或回退视图
- **AND** 同一条消息中的其他块继续渲染
- **AND** AI 助手抽屉保持可用

#### Scenario: 流式输出
- **GIVEN** AI 正在生成回复
- **WHEN** 收到 SSE delta 事件
- **THEN** 实时更新消息内容
- **AND** 显示打字效果
