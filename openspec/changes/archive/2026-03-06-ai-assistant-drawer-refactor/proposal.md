# Proposal: AI 助手抽屉式重构

## 概述

将现有的 AI 聊天页面重构为抽屉式 AI 助手组件，使用 Ant Design X 组件库实现现代化的聊天体验。支持全局和场景双模式，提供更好的用户体验和代码可维护性。

## 动机

### 当前问题

1. **未使用 Ant Design X**: 已安装 `@ant-design/x@2.3.0` 但完全未使用，浪费了成熟的聊天组件
2. **手写 SSE 处理**: `useSSEConnection` 有 250+ 行手写代码，维护困难，错误处理分散
3. **错误提示简陋**: 只显示 `streamError` 字符串，用户看不到详细信息，无可操作性
4. **独立页面模式**: AI 助手作为独立页面，用户需要离开当前工作上下文
5. **样式硬编码**: 大量 CSS 变量硬编码，主题切换困难

### 目标

1. **抽屉式交互**: AI 助手从右侧滑出，用户可在当前页面上下文中使用
2. **场景感知**: 根据当前路由自动切换场景，提供上下文相关的 AI 能力
3. **现代化组件**: 使用 Ant Design X 的 Conversations、Bubble、Sender、XMarkdown 组件
4. **统一错误处理**: 使用 Toast 提示，提供用户友好的错误信息

## 范围

### 包含

- 创建 `AIAssistantDrawer` 抽屉组件
- 创建 `AIAssistantButton` 触发按钮
- 实现 `useSSEAdapter` 将 SSE 适配到 useXChat
- 实现场景检测和路由映射
- 创建简化版工具执行卡片
- 添加全局快捷键支持
- 删除现有 AIChat 页面

### 不包含

- 移动端适配（明确排除）
- 后端 API 修改
- AI 模型相关修改

## 方案

### 架构变更

```
现有架构:
  /ai 路由 → ChatPage → ConversationSidebar + ChatMain
                               ↓
                        useSSEConnection (手写)

新架构:
  Header 按钮 → AIAssistantDrawer
                    ├── Conversations (会话管理)
                    ├── Bubble.List (消息列表 + XMarkdown)
                    ├── ToolCard (简化版工具卡片)
                    └── Sender (输入框)
                               ↓
                        useXChat + XRequest (Ant Design X)
```

### 场景模式

- **全局模式**: `scene="global"`，所有页面可用，通过 `Cmd+/` 打开
- **场景模式**: 根据路由自动检测，通过 `Cmd+Shift+/` 打开

### 可变宽度

- 最小宽度: 480px
- 最大宽度: 800px
- 默认宽度: 520px
- 支持拖拽调整

## 影响

### 用户影响

- 正面: 无需离开当前页面即可使用 AI 助手，场景感知提供更精准的帮助
- 注意: 现有 `/ai` 页面将被移除，用户需要适应新的交互方式

### 开发者影响

- 需要学习 Ant Design X 组件使用
- 新的场景检测机制需要维护
- 删除大量现有代码

### 系统影响

- 无后端变更
- 前端包大小可能略有增加（使用 Ant Design X 组件）

## 风险

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| Ant Design X 组件不满足需求 | 中 | 高 | 提前调研，准备自定义组件方案 |
| SSE 适配复杂度超预期 | 中 | 中 | 保留现有 API 层，仅替换 UI 层 |
| 用户习惯变更抵触 | 低 | 低 | 提供快捷键，保持高效访问 |

## 验收标准

- [ ] 抽屉可从 Header 按钮或快捷键打开
- [ ] 支持全局和场景两种模式
- [ ] 抽屉宽度可拖拽调整 (480-800px)
- [ ] 使用 Conversations 组件管理会话
- [ ] 使用 Bubble.List + XMarkdown 渲染消息
- [ ] 使用 Sender 组件作为输入框
- [ ] 错误提示使用 Toast (antd message)
- [ ] 工具执行卡片显示工具名、状态、耗时
- [ ] `/ai` 路由已移除
- [ ] 现有 AIChat 页面代码已删除
