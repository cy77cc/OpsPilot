# Tasks: AI 助手抽屉式重构

## Phase 1: 基础组件搭建 (2 天)

### 1.1 创建组件目录结构
- [x] 创建 `web/src/components/AI/` 目录
- [x] 创建 `index.ts` 导出入口
- [x] 创建 `types.ts` 类型定义
- [x] 创建 `constants/sceneMapping.ts` 场景映射配置

### 1.2 实现核心 Hooks
- [x] 实现 `useResizableDrawer.ts` 可变宽度 Drawer Hook
- [x] 实现 `useSceneDetector.ts` 场景检测 Hook
- [x] 实现 `useSSEAdapter.ts` SSE 适配 Hook

### 1.3 创建 Provider
- [x] 实现 `AIChatProvider.tsx` Context Provider
- [x] 集成 `useXChat` 状态管理
- [x] 添加错误处理逻辑

---

## Phase 2: 抽屉组件开发 (3 天)

### 2.1 主抽屉组件
- [x] 实现 `AIAssistantDrawer.tsx` 主组件
- [x] 集成 antd Drawer 组件
- [x] 实现拖拽调整宽度功能
- [x] 添加最小/最大宽度限制

### 2.2 会话管理组件
- [x] 实现 `ConversationsPanel.tsx`
- [x] 使用 Ant Design X `Conversations` 组件
- [x] 实现会话列表、切换、新建、删除
- [x] 添加会话搜索功能

### 2.3 消息列表组件
- [x] 实现 `MessageList.tsx`
- [x] 使用 Ant Design X `Bubble.List` 组件
- [x] 实现 `MessageBubble.tsx` 消息气泡
- [x] 集成 `XMarkdown` 渲染
- [x] 实现流式输出效果

### 2.4 输入组件
- [x] 实现 `ChatInput.tsx`
- [x] 使用 Ant Design X `Sender` 组件
- [x] 实现发送、加载状态
- [x] 支持 Shift+Enter 换行

---

## Phase 3: 工具和审批组件 (1 天)

### 3.1 工具执行卡片
- [x] 实现 `ToolCard.tsx` 简化版工具卡片
- [x] 显示工具名、状态、耗时
- [x] 实现状态图标 (loading/success/error)

### 3.2 审批确认面板
- [x] 实现 `ConfirmationPanel.tsx`
- [x] 显示操作描述、风险等级
- [x] 实现确认/取消按钮交互
- [x] 处理审批响应

---

## Phase 4: Header 集成 (1 天)

### 4.1 AI 助手按钮
- [x] 实现 `AIAssistantButton.tsx`
- [x] 实现全局助手按钮
- [x] 实现场景助手按钮 (条件渲染)
- [x] 添加 Tooltip 提示

### 4.2 AppLayout 集成
- [x] 在 `AppLayout.tsx` Header 中添加按钮
- [x] 实现全局状态管理 (两个 Drawer 的 open 状态)
- [x] 添加快捷键监听 (`Cmd+/`, `Cmd+Shift+/`, `Escape`)

---

## Phase 5: 错误处理 (1 天)

### 5.1 错误分类和映射
- [x] 定义错误类型枚举
- [x] 实现错误消息映射
- [x] 实现错误分类函数

### 5.2 Toast 集成
- [x] 集成 antd `message` 组件
- [x] 实现网络错误提示
- [x] 实现超时错误提示
- [x] 实现认证错误处理 (跳转登录)
- [x] 实现工具错误提示

---

## Phase 6: 清理和迁移 (1 天)

### 6.1 删除旧代码
- [x] 删除 `web/src/pages/AIChat/` 目录
- [x] 删除 `web/src/pages/AI/` 目录 (如果存在)
- [x] 移除 `App.tsx` 中的 `/ai` 路由
- [x] 移除 `AppLayout.tsx` 中的菜单项

### 6.2 路由重定向
- [x] 添加 `/ai` → `/` 重定向
- [x] 显示迁移提示 Toast

---

## Phase 7: 测试和优化 (1 天)

### 7.1 功能测试
- [x] 测试抽屉打开/关闭
- [x] 测试宽度调整
- [x] 测试场景切换
- [x] 测试消息发送/接收
- [x] 测试工具执行显示
- [x] 测试审批确认流程

### 7.2 错误场景测试
- [x] 测试网络错误处理
- [x] 测试超时错误处理
- [x] 测试认证错误处理
- [x] 测试工具错误显示

### 7.3 性能优化
- [x] 检查组件渲染性能
- [x] 优化消息列表滚动
- [x] 检查内存泄漏

---

## 任务依赖关系

```
Phase 1 (基础组件)
├─ 1.1 目录结构 ─────┐
├─ 1.2 核心 Hooks ───┼──▶ Phase 2 (抽屉组件)
└─ 1.3 Provider ─────┘         │
                               ▼
                    Phase 3 (工具组件)
                               │
                               ▼
                    Phase 4 (Header 集成)
                               │
                               ▼
                    Phase 5 (错误处理)
                               │
                               ▼
                    Phase 6 (清理迁移)
                               │
                               ▼
                    Phase 7 (测试优化)
```

## 风险和阻塞点

| 风险 | 缓解措施 | 负责人 |
|------|----------|--------|
| Ant Design X 组件 API 不熟悉 | 提前阅读文档，参考官方示例 | 前端 |
| SSE 适配复杂度超预期 | 保留现有 API 层，渐进式替换 | 前端 |
| 宽度拖拽性能问题 | 使用 requestAnimationFrame 优化 | 前端 |
| 快捷键与其他功能冲突 | 合理设计快捷键，添加冲突检测 | 前端 |

## 验收标准

- [x] 抽屉可从 Header 按钮或快捷键打开
- [x] 支持全局和场景两种模式
- [x] 抽屉宽度可拖拽调整 (480-800px)
- [x] 使用 Conversations 组件管理会话
- [x] 使用 Bubble.List + XMarkdown 渲染消息
- [x] 使用 Sender 组件作为输入框
- [x] 错误提示使用 Toast (antd message)
- [x] 工具执行卡片显示工具名、状态、耗时
- [x] `/ai` 路由已移除并重定向到首页
- [x] 现有 AIChat 页面代码已删除
- [x] 所有功能测试通过
