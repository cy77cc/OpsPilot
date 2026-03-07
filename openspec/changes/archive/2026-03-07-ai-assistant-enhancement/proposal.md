# AI Assistant Enhancement Proposal

## Summary

增强 AI Copilot 助手的用户体验，包括思考过程展示、历史对话持久化、下一步提示、工具调用详情展示、场景感知快捷指令、重新生成功能和复制修复。

## Motivation

当前 AI 助手存在以下问题：

1. **思考过程展示不够优雅** - 使用自定义 Collapse 组件，样式与 Ant Design X 不一致
2. **历史对话不持久化** - 页面刷新后对话丢失，用户体验差
3. **缺少下一步提示** - 对话结束后无引导，用户不知道如何继续
4. **工具调用信息不完整** - 只显示工具名和状态，缺少参数和结果详情
5. **快捷指令固定** - 欢迎页提示词静态，无法根据场景动态调整
6. **缺少重新生成功能** - 用户对输出不满意时无法重新生成
7. **复制按钮无效** - 消息操作栏的复制按钮没有绑定事件，点击无效果

## Goals

### Primary Goals

- 使用 `@ant-design/x` 的 `Think` 组件替换自定义思考过程展示
- 实现历史对话自动恢复，页面刷新后可继续上次对话
- 展示工具调用的参数和执行结果
- 在对话末尾显示 AI 生成的下一步建议
- **修复复制按钮功能**，实现消息内容复制到剪贴板

### Secondary Goals

- 后端动态生成场景感知的快捷指令
- 优化工具调用卡片 UI，支持展开查看详情
- **实现重新生成功能**，允许用户重新生成不满意的回复

### Non-Goals

- 修改后端 AI 推理逻辑
- 改变对话 API 接口结构

## Proposed Solution

### 1. Think 组件替换

将 `Copilot.tsx` 中的自定义 `ThinkingBlock` 组件替换为 `@ant-design/x` 的 `Think` 组件。

**变更点**：
- `web/src/components/AI/Copilot.tsx` - 使用 `<Think>` 组件

### 2. 历史对话持久化

在组件挂载时加载历史会话，自动恢复上次对话。

**变更点**：
- `web/src/components/AI/Copilot.tsx` - 添加 `useEffect` 调用加载逻辑
- 新增 `useConversationRestore` hook - 封装恢复逻辑

### 3. 下一步提示展示

从 SSE `done` 事件的 `turn_recommendations` 字段提取建议，在消息末尾展示可点击的推荐卡片。

**变更点**：
- `web/src/components/AI/types.ts` - 扩展消息类型，添加 `recommendations` 字段
- `web/src/components/AI/Copilot.tsx` - 处理 `done` 事件中的推荐
- 新增 `web/src/components/AI/components/RecommendationCard.tsx` - 推荐卡片组件

### 4. 工具调用详情展示

扩展工具数据结构，在 ToolCard 中展示参数和结果。

**变更点**：
- `web/src/components/AI/types.ts` - 扩展 `ToolExecution` 类型
- `web/src/components/AI/components/ToolCard.tsx` - 增强展示
- `web/src/components/AI/Copilot.tsx` - 从 SSE 事件提取完整数据

### 5. 场景感知快捷指令

新增后端模型和 API，动态生成场景专属提示词。

**变更点**：
- 新增数据库表 `ai_scene_prompts`
- 新增 `internal/model/ai_scene_prompt.go`
- 新增 `internal/service/ai/handler/scene_prompt_handler.go`
- 新增 API `GET /api/v1/ai/scene/:scene/prompts`
- 更新 `configs/scene_mappings.yaml` - 添加 prompts 配置
- 新增 `web/src/api/modules/ai.ts` - `getScenePrompts()` 方法
- 新增 `web/src/components/AI/hooks/useScenePrompts.ts` - 场景提示 hook

### 6. 复制功能修复

修复消息操作栏中复制按钮无效的问题。

**问题分析**：
- 当前实现 (`Copilot.tsx:141`) 只渲染了按钮 UI，没有 `onClick` 处理
- `Bubble.List` 的 `role` 配置中 `footer` 是静态 JSX，无法访问消息内容

**解决方案**：
- 使用 `@ant-design/x` 提供的消息上下文或自定义渲染
- 实现复制到剪贴板功能 (`navigator.clipboard.writeText`)
- 添加复制成功的视觉反馈

**变更点**：
- `web/src/components/AI/Copilot.tsx` - 修复 `createRoleConfig` 中的复制逻辑

### 7. 重新生成功能

为助手消息添加"重新生成"按钮，允许用户重新生成不满意的回复。

**功能设计**：
- 在助手消息操作栏添加"重新生成"按钮
- 点击后删除当前助手回复，基于相同上下文重新请求
- 可选择保留或清除之前的工具调用

**实现要点**：
- 重新生成需要携带相同的对话上下文
- 从消息历史中找到对应的用户问题
- 删除当前助手消息后重新发送请求

**变更点**：
- `web/src/components/AI/Copilot.tsx` - 添加重新生成按钮和处理逻辑
- `web/src/components/AI/types.ts` - 可能需要扩展消息操作类型

## Impact Analysis

### Frontend Changes

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `Copilot.tsx` | 修改 | Think组件、加载历史、处理推荐、复制功能、重新生成 |
| `types.ts` | 修改 | 扩展消息和工具类型 |
| `ToolCard.tsx` | 修改 | 增强工具详情展示 |
| `RecommendationCard.tsx` | 新增 | 推荐卡片组件 |
| `useScenePrompts.ts` | 新增 | 场景提示 hook |
| `useConversationRestore.ts` | 新增 | 会话恢复 hook |
| `ai.ts` (API) | 修改 | 新增 getScenePrompts |

### Backend Changes

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `ai_scene_prompt.go` | 新增 | 数据模型 |
| `scene_prompt_handler.go` | 新增 | API 处理器 |
| `routes.go` | 修改 | 注册新路由 |
| `scene_mappings.yaml` | 修改 | 添加 prompts 配置 |

### Database Changes

```sql
CREATE TABLE ai_scene_prompts (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  scene VARCHAR(128) NOT NULL,
  prompt_text TEXT NOT NULL,
  prompt_type VARCHAR(32) DEFAULT 'quick_action',
  display_order INT DEFAULT 0,
  is_active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_scene (scene)
);
```

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Think 组件 API 变化 | Low | Low | 检查 @ant-design/x 文档，使用稳定 API |
| 历史会话加载慢 | Medium | Medium | 添加 loading 状态，分页加载 |
| 工具结果数据量大 | Medium | Medium | 折叠显示，限制高度 |
| LLM 生成提示词不稳定 | Medium | Low | 使用固定 prompt 模板，添加 fallback |

## Timeline

- **Phase 1** (Day 1-2): Think 组件替换、历史对话持久化、复制功能修复
- **Phase 2** (Day 2-3): 工具调用详情展示、重新生成功能
- **Phase 3** (Day 3-4): 下一步提示展示
- **Phase 4** (Day 4-5): 场景感知快捷指令（后端 + 前端）

## Success Criteria

1. 页面刷新后自动恢复上次对话
2. 思考过程使用 Think 组件展示
3. 工具调用可展开查看参数和结果
4. 对话结束后显示可点击的下一步建议
5. 不同场景显示不同的快捷指令
6. 点击复制按钮可将消息内容复制到剪贴板
7. 点击重新生成按钮可重新生成 AI 回复
