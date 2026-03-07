# ai-assistant-drawer Specification

## Purpose

定义 AI 助手抽屉式组件的功能规格，实现现代化的聊天体验和场景感知能力。

## Requirements

### Requirement: 抽屉式交互

系统 SHALL 提供抽屉式 AI 助手，从页面右侧滑出，不离开当前页面上下文。

#### Scenario: 从 Header 打开抽屉

- **GIVEN** 用户在任意页面
- **WHEN** 用户点击 Header 的 "AI 助手" 按钮
- **THEN** 抽屉从右侧滑出
- **AND** 当前页面内容保持不变

#### Scenario: 使用快捷键打开

- **GIVEN** 用户在任意页面
- **WHEN** 用户按下 `Cmd+/` (Mac) 或 `Ctrl+/` (Windows)
- **THEN** 全局 AI 助手抽屉打开
- **AND** 场景为 "global"

#### Scenario: 关闭抽屉

- **GIVEN** AI 助手抽屉已打开
- **WHEN** 用户点击抽屉外部区域或按下 Escape 键
- **THEN** 抽屉关闭
- **AND** 当前会话状态保留

---

### Requirement: 可变宽度抽屉

系统 SHALL 支持用户拖拽调整抽屉宽度。

#### Scenario: 拖拽调整宽度

- **GIVEN** AI 助手抽屉已打开
- **WHEN** 用户拖拽抽屉左边缘
- **THEN** 抽屉宽度随拖拽变化
- **AND** 宽度限制在 480px 至 800px 之间

#### Scenario: 记住宽度

- **GIVEN** 用户调整了抽屉宽度
- **WHEN** 用户关闭并重新打开抽屉
- **THEN** 抽屉保持上次调整的宽度

---

### Requirement: 场景感知模式

系统 SHALL 根据当前路由自动检测场景，提供上下文相关的 AI 能力。

#### Scenario: 全局模式

- **GIVEN** 用户点击 "AI 助手" 按钮或使用 `Cmd+/`
- **WHEN** 抽屉打开
- **THEN** 场景设置为 "global"
- **AND** 显示全局会话列表

#### Scenario: 场景模式

- **GIVEN** 用户在主机管理页面 (`/deployment/infrastructure/hosts`)
- **WHEN** 用户点击场景助手按钮或使用 `Cmd+Shift+/`
- **THEN** 场景设置为 "deployment:hosts"
- **AND** 显示该场景的会话列表和推荐工具

#### Scenario: 场景按钮显示

- **GIVEN** 当前路由有对应的场景映射
- **WHEN** Header 渲染
- **THEN** 显示场景助手按钮
- **AND** 按钮显示场景名称

#### Scenario: 无场景路由

- **GIVEN** 当前路由无场景映射（如首页）
- **WHEN** Header 渲染
- **THEN** 仅显示全局助手按钮
- **AND** 不显示场景助手按钮

---

### Requirement: 会话管理

系统 SHALL 使用 Ant Design X 的 Conversations 组件管理会话。

#### Scenario: 显示会话列表

- **GIVEN** AI 助手抽屉已打开
- **WHEN** 组件加载完成
- **THEN** 左侧显示会话列表
- **AND** 列表按更新时间倒序排列

#### Scenario: 切换会话

- **GIVEN** 会话列表显示多个会话
- **WHEN** 用户点击某个会话
- **THEN** 加载该会话的消息历史
- **AND** 消息列表滚动到底部

#### Scenario: 新建会话

- **GIVEN** 用户在 AI 助手中
- **WHEN** 用户点击 "新建会话" 按钮
- **THEN** 清空当前消息
- **AND** 创建新会话

#### Scenario: 删除会话

- **GIVEN** 用户在会话列表中
- **WHEN** 用户删除某个会话
- **THEN** 从列表中移除该会话
- **AND** 如果是当前会话，切换到新会话状态

---

### Requirement: AI surface failures MUST be isolated from the application shell

The system MUST keep the main application shell usable when the AI assistant drawer or its lazy-loaded surface fails to initialize.

#### Scenario: AI drawer initialization failure

- **WHEN** the AI assistant drawer surface throws during initialization
- **THEN** the main application shell remains interactive
- **AND** the AI entry shows a local fallback state instead of blanking the page

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

---

### Requirement: 工具执行卡片

系统 SHALL 显示简化版工具执行卡片，包含工具名、状态、耗时。

#### Scenario: 工具开始执行

- **GIVEN** AI 调用工具
- **WHEN** 收到 tool_call 事件
- **THEN** 显示工具卡片
- **AND** 状态显示为 "执行中"
- **AND** 显示加载动画

#### Scenario: 工具执行成功

- **GIVEN** 工具正在执行
- **WHEN** 收到 tool_result 事件且成功
- **THEN** 状态更新为 "成功"
- **AND** 显示执行耗时
- **AND** 图标变为绿色勾号

#### Scenario: 工具执行失败

- **GIVEN** 工具正在执行
- **WHEN** 收到 tool_result 事件且失败
- **THEN** 状态更新为 "失败"
- **AND** 图标变为红色叉号

---

### Requirement: 错误处理

系统 SHALL 使用 Toast (antd message) 显示用户友好的错误提示。

#### Scenario: 网络错误

- **GIVEN** 用户发送消息
- **WHEN** 网络请求失败
- **THEN** 显示 Toast 提示 "网络连接失败，请检查网络后重试"
- **AND** 提示 3 秒后自动消失

#### Scenario: 超时错误

- **GIVEN** 用户发送消息
- **WHEN** 请求超时
- **THEN** 显示 Toast 提示 "请求超时，点击重试"
- **AND** 提示可点击重试

#### Scenario: 认证错误

- **GIVEN** 用户发送消息
- **WHEN** 返回 401 错误
- **THEN** 显示 Toast 提示 "登录已过期，请重新登录"
- **AND** 跳转到登录页

#### Scenario: 工具错误

- **GIVEN** 工具执行失败
- **WHEN** 工具返回错误
- **THEN** 显示 Toast 提示 "工具执行失败：{错误详情}"
- **AND** 消息中显示简化错误信息

---

### Requirement: 输入框

系统 SHALL 使用 Ant Design X 的 Sender 组件作为输入框。

#### Scenario: 正常输入

- **GIVEN** 用户在输入框输入文本
- **WHEN** 用户点击发送或按 Enter
- **THEN** 发送消息
- **AND** 清空输入框

#### Scenario: 输入时加载中

- **GIVEN** AI 正在生成回复
- **WHEN** 用户查看输入框
- **THEN** 发送按钮禁用
- **AND** 显示加载状态

#### Scenario: Shift+Enter 换行

- **GIVEN** 用户在输入框输入
- **WHEN** 用户按 Shift+Enter
- **THEN** 输入框换行
- **AND** 不发送消息

---

### Requirement: 审批确认

系统 SHALL 显示审批确认面板，支持高风险操作的确认和取消。

#### Scenario: 高风险操作触发确认

- **GIVEN** AI 执行高风险工具
- **WHEN** 收到 approval_required 事件
- **THEN** 显示确认面板
- **AND** 显示操作描述、风险等级
- **AND** 显示确认和取消按钮

#### Scenario: 用户确认执行

- **GIVEN** 确认面板显示
- **WHEN** 用户点击 "确认"
- **THEN** 发送确认请求
- **AND** 继续执行工具

#### Scenario: 用户取消操作

- **GIVEN** 确认面板显示
- **WHEN** 用户点击 "取消"
- **THEN** 发送取消请求
- **AND** 终止工具执行

---

### Requirement: 路由清理

系统 SHALL 移除独立的 AI 聊天页面路由。

#### Scenario: 访问旧路由

- **GIVEN** 用户访问 `/ai` 路由
- **WHEN** 路由匹配
- **THEN** 重定向到首页
- **AND** 显示提示 "AI 助手已移至右上角"
