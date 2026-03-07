# AI Assistant Enhancement - Tasks

## Phase 1: Core Enhancements (Day 1-2)

### Task 1.1: Think 组件替换
**Priority**: P0 | **Estimate**: 2h | **Dependencies**: None

- [x] 读取 `@ant-design/x` Think 组件文档，确认 API
- [x] 修改 `Copilot.tsx`，替换 `ThinkingBlock` 为 `<Think>` 组件
- [x] 处理流式状态 (`loading` / `done`)
- [x] 测试思考过程展示效果

**Files**:
- `web/src/components/AI/Copilot.tsx` (修改)

---

### Task 1.2: 历史对话恢复 Hook
**Priority**: P0 | **Estimate**: 3h | **Dependencies**: None

- [x] 创建 `useConversationRestore` hook
- [x] 实现 `getCurrentSession` → `getSessions` fallback 逻辑
- [x] 添加 `isRestoring` 状态
- [x] 在 `Copilot.tsx` 中集成 hook

**Files**:
- `web/src/components/AI/hooks/useConversationRestore.ts` (新增)
- `web/src/components/AI/Copilot.tsx` (修改)

---

### Task 1.3: Copilot 组件加载逻辑
**Priority**: P0 | **Estimate**: 2h | **Dependencies**: Task 1.2

- [x] 添加组件挂载时的会话恢复逻辑
- [x] 处理恢复成功/失败状态
- [x] 添加 loading 骨架屏
- [x] 测试页面刷新后恢复流程

**Files**:
- `web/src/components/AI/Copilot.tsx` (修改)

---

## Phase 2: Tool Call Enhancement (Day 2-3)

### Task 2.1: 扩展工具类型定义
**Priority**: P0 | **Estimate**: 1h | **Dependencies**: None

- [x] 扩展 `ToolExecution` 接口，添加 `params` 和 `result` 字段
- [x] 更新相关类型引用

**Files**:
- `web/src/components/AI/types.ts` (修改)

---

### Task 2.2: SSE 工具数据提取
**Priority**: P0 | **Estimate**: 2h | **Dependencies**: Task 2.1

- [x] 在 SSE 处理中提取 `tool_call` 的 `payload` 参数
- [x] 在 SSE 处理中提取 `tool_result` 的完整结果
- [x] 更新消息状态中的工具数据

**Files**:
- `web/src/components/AI/Copilot.tsx` (修改)
- `web/src/components/AI/hooks/useSSEAdapter.ts` (修改)

---

### Task 2.3: 增强工具卡片组件
**Priority**: P1 | **Estimate**: 3h | **Dependencies**: Task 2.1, 2.2

- [x] 添加展开/收起功能
- [x] 展示工具调用参数 (JSON 格式化)
- [x] 展示工具执行结果
- [x] 处理错误状态展示
- [x] 添加样式优化

**Files**:
- `web/src/components/AI/components/ToolCard.tsx` (修改)
- `web/src/components/AI/AIAssistantDrawer.css` (修改)

---

## Phase 3: Recommendations (Day 3-4)

### Task 3.1: 消息类型扩展
**Priority**: P1 | **Estimate**: 1h | **Dependencies**: None

- [x] 扩展 `ChatMessage` 类型，添加 `recommendations` 字段
- [x] 定义 `EmbeddedRecommendation` 类型复用

**Files**:
- `web/src/components/AI/types.ts` (修改)

---

### Task 3.2: SSE done 事件处理
**Priority**: P1 | **Estimate**: 1h | **Dependencies**: Task 3.1

- [x] 从 `done` 事件提取 `turn_recommendations`
- [x] 更新最后一条消息的 `recommendations` 字段

**Files**:
- `web/src/components/AI/Copilot.tsx` (修改)

---

### Task 3.3: 推荐卡片组件
**Priority**: P1 | **Estimate**: 2h | **Dependencies**: Task 3.1, 3.2

- [x] 创建 `RecommendationCard` 组件
- [x] 实现点击发送 `followup_prompt`
- [x] 添加样式和动画效果

**Files**:
- `web/src/components/AI/components/RecommendationCard.tsx` (新增)
- `web/src/components/AI/AIAssistantDrawer.css` (修改)

---

### Task 3.4: 集成推荐卡片
**Priority**: P1 | **Estimate**: 1h | **Dependencies**: Task 3.3

- [x] 在 `AssistantMessage` 中渲染 `RecommendationCard`
- [x] 处理点击后的消息发送

**Files**:
- `web/src/components/AI/Copilot.tsx` (修改)

---

## Phase 4: Scene Prompts (Day 4-5)

### Task 4.1: 数据库迁移
**Priority**: P1 | **Estimate**: 1h | **Dependencies**: None

- [x] 创建 `ai_scene_prompts` 表
- [x] 添加初始数据

**Files**:
- `internal/model/ai_scene_prompt.go` (新增)
- Migration SQL (新增)

---

### Task 4.2: 后端场景提示处理器
**Priority**: P1 | **Estimate**: 3h | **Dependencies**: Task 4.1

- [x] 创建 `scenePromptHandler`
- [x] 实现数据库查询逻辑
- [x] 实现 LLM 生成 fallback (可选)
- [x] 注册路由

**Files**:
- `internal/service/ai/handler/scene_prompt_handler.go` (新增)
- `internal/service/ai/routes.go` (修改)

---

### Task 4.3: 更新场景配置
**Priority**: P2 | **Estimate**: 1h | **Dependencies**: None

- [x] 在 `scene_mappings.yaml` 添加 prompts 配置
- [x] 更新加载逻辑

**Files**:
- `configs/scene_mappings.yaml` (修改)
- `internal/service/ai/handler/scene_context.go` (修改)

---

### Task 4.4: 前端 API 方法
**Priority**: P1 | **Estimate**: 1h | **Dependencies**: Task 4.2

- [x] 添加 `getScenePrompts` API 方法
- [x] 定义返回类型

**Files**:
- `web/src/api/modules/ai.ts` (修改)

---

### Task 4.5: 场景提示 Hook
**Priority**: P1 | **Estimate**: 2h | **Dependencies**: Task 4.4

- [x] 创建 `useScenePrompts` hook
- [x] 实现 API 调用和缓存
- [x] 添加默认 fallback

**Files**:
- `web/src/components/AI/hooks/useScenePrompts.ts` (新增)

---

### Task 4.6: 集成场景提示到欢迎页
**Priority**: P1 | **Estimate**: 2h | **Dependencies**: Task 4.5

- [x] 在 `Copilot.tsx` 欢迎页使用 `useScenePrompts`
- [x] 替换静态 `DEFAULT_PROMPTS`
- [x] 处理场景切换时的重新加载

**Files**:
- `web/src/components/AI/Copilot.tsx` (修改)

---

## Phase 5: Message Actions (Day 2-3)

### Task 5.1: 修复复制按钮功能
**Priority**: P0 | **Estimate**: 1h | **Dependencies**: None

- [x] 分析 `@ant-design/x` Bubble 组件的 footer 上下文传递方式
- [x] 实现消息内容复制到剪贴板 (`navigator.clipboard.writeText`)
- [x] 添加复制成功的 toast 提示
- [x] 处理复制失败的情况

**Files**:
- `web/src/components/AI/Copilot.tsx` (修改)

---

### Task 5.2: 消息操作组件
**Priority**: P1 | **Estimate**: 2h | **Dependencies**: Task 5.1

- [x] 创建 `MessageActions` 组件，封装操作按钮
- [x] 实现复制、点赞、点踩按钮
- [x] 添加分隔线和样式优化
- [x] 集成到消息渲染中

**Files**:
- `web/src/components/AI/components/MessageActions.tsx` (新增)
- `web/src/components/AI/Copilot.tsx` (修改)

---

### Task 5.3: 重新生成功能实现
**Priority**: P1 | **Estimate**: 3h | **Dependencies**: Task 5.2

- [x] 在 `MessageActions` 添加"重新生成"按钮
- [x] 实现 `handleRegenerate` 逻辑
  - 找到对应的用户消息
  - 删除当前助手消息
  - 重新发送请求
- [x] 添加重新生成中的 loading 状态
- [x] 处理边界情况（首条消息、连续点击等）

**Files**:
- `web/src/components/AI/Copilot.tsx` (修改)
- `web/src/components/AI/components/MessageActions.tsx` (修改)

---

### Task 5.4: 重新生成样式优化
**Priority**: P2 | **Estimate**: 1h | **Dependencies**: Task 5.3

- [x] 添加重新生成按钮的 hover 效果
- [x] 添加重新生成中的动画
- [x] 确保按钮在消息底部正确对齐

**Files**:
- `web/src/components/AI/AIAssistantDrawer.css` (修改)

---

## Testing Checklist

### Functional Tests
- [ ] 页面刷新后自动恢复上次对话
- [ ] Think 组件正确展示思考过程
- [ ] 工具调用展示参数和结果
- [ ] 点击推荐卡片可发送消息
- [ ] 不同场景显示不同快捷指令
- [ ] **点击复制按钮可将消息内容复制到剪贴板**
- [ ] **复制成功显示提示**
- [ ] **点击重新生成按钮可重新生成 AI 回复**
- [ ] **重新生成时显示 loading 状态**

### Edge Cases
- [ ] 无历史会话时的处理
- [ ] 工具调用失败的展示
- [ ] 网络错误时的 fallback
- [ ] 空推荐列表的处理
- [ ] **复制失败时的错误处理**
- [ ] **首条消息无法重新生成**
- [ ] **连续快速点击重新生成按钮的防抖处理**

### Performance
- [ ] 会话恢复时间 < 1s
- [ ] 工具卡片展开动画流畅
- [ ] 大量工具结果时的渲染性能
- [ ] **复制操作响应 < 100ms**

---

## Rollback Plan

1. **前端回滚**: 恢复 `Copilot.tsx` 到之前版本
2. **后端回滚**: 移除新路由，保留数据库表不影响
3. **数据回滚**: `DROP TABLE ai_scene_prompts` (如需要)
