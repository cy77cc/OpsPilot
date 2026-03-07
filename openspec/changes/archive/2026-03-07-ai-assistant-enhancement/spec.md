# AI Assistant Enhancement - Specification

## Feature: AI Copilot 用户体验增强

### Overview

增强 AI Copilot 助手的用户体验，提供更智能的对话交互、完整的历史持久化和丰富的信息展示。

---

## Specifications

### SPEC-1: 思考过程展示 (Think Component)

**Requirement**: 使用 `@ant-design/x` 的 `Think` 组件展示 AI 思考过程。

**Behavior**:
- 思考内容在助手消息开头展示
- 流式输出时显示 loading 状态
- 完成后可展开/收起

**Acceptance Criteria**:
- [GIVEN] 助手消息包含 `thinking` 字段
- [WHEN] 渲染消息时
- [THEN] 使用 `<Think>` 组件展示思考过程

**Technical Notes**:
- 组件来源: `import { Think } from '@ant-design/x'`
- 属性: `content`, `status`, `title`

---

### SPEC-2: 历史对话持久化

**Requirement**: 页面刷新后自动恢复上次对话，用户可继续交互。

**Behavior**:
- 组件挂载时检查当前会话
- 优先恢复最近活跃会话
- 显示恢复中的 loading 状态

**Acceptance Criteria**:
- [GIVEN] 用户有过对话历史
- [WHEN] 刷新页面或重新打开 AI 助手
- [THEN] 自动加载最近的会话内容和消息列表

**API Usage**:
```
GET /api/v1/ai/sessions/current?scene={scene}
GET /api/v1/ai/sessions/{id}
```

---

### SPEC-3: 工具调用详情展示

**Requirement**: 展示工具调用的完整信息，包括参数和执行结果。

**Behavior**:
- 工具卡片默认显示: 名称、状态、耗时
- 展开后显示: 参数、结果数据
- 结果支持 JSON 格式化和折叠

**Data Structure**:
```typescript
interface ToolExecution {
  id: string;
  name: string;
  status: 'running' | 'success' | 'error';
  duration?: number;
  error?: string;
  params?: Record<string, unknown>;      // 调用参数
  result?: {                             // 执行结果
    ok: boolean;
    data?: unknown;
    error?: string;
    latency_ms?: number;
  };
}
```

**Acceptance Criteria**:
- [GIVEN] 工具被调用
- [WHEN] 消息包含工具执行记录
- [THEN] 展示工具名称、状态、可展开查看参数和结果

---

### SPEC-4: 下一步提示

**Requirement**: 对话结束后显示 AI 生成的下一步建议。

**Behavior**:
- 从 SSE `done` 事件获取 `turn_recommendations`
- 在消息末尾展示推荐卡片
- 点击推荐自动填充输入或发送消息

**Data Structure**:
```typescript
interface EmbeddedRecommendation {
  id: string;
  type: string;
  title: string;
  content: string;
  followup_prompt?: string;  // 点击后发送的内容
  reasoning?: string;
  relevance: number;
}
```

**Acceptance Criteria**:
- [GIVEN] 对话完成且返回了推荐
- [WHEN] 渲染助手消息
- [THEN] 在消息末尾显示可点击的推荐卡片

---

### SPEC-5: 场景感知快捷指令

**Requirement**: 根据当前场景动态显示快捷指令提示。

**Behavior**:
- 从后端 API 获取场景专属提示词
- 无数据时使用默认提示词
- 场景切换时重新加载

**API**:
```
GET /api/v1/ai/scene/:scene/prompts

Response:
{
  "scene": "deployment:clusters",
  "prompts": [
    { "id": "1", "prompt_text": "查看所有集群状态", "prompt_type": "quick_action" },
    { "id": "2", "prompt_text": "帮我检查集群健康度", "prompt_type": "quick_action" }
  ]
}
```

**Acceptance Criteria**:
- [GIVEN] 用户打开 AI 助手
- [WHEN] 处于特定场景页面
- [THEN] 欢迎页显示该场景相关的快捷指令

---

## API Contracts

### GET /api/v1/ai/scene/:scene/prompts

**Purpose**: 获取场景快捷提示词

**Request**:
- Method: GET
- Path: `/api/v1/ai/scene/:scene/prompts`
- Auth: Required

**Response**:
```json
{
  "code": 0,
  "data": {
    "scene": "deployment:clusters",
    "prompts": [
      {
        "id": "1",
        "prompt_text": "查看所有集群状态",
        "prompt_type": "quick_action"
      }
    ]
  }
}
```

**Error Codes**:
- `401`: 未授权
- `404`: 场景不存在

---

## Database Schema

### Table: ai_scene_prompts

| Column | Type | Description |
|--------|------|-------------|
| id | BIGINT | Primary Key |
| scene | VARCHAR(128) | 场景标识 |
| prompt_text | TEXT | 提示词文本 |
| prompt_type | VARCHAR(32) | 类型 (quick_action, suggestion) |
| display_order | INT | 显示顺序 |
| is_active | BOOLEAN | 是否启用 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

**Indexes**:
- `idx_scene_prompts` on `scene`

---

## UI Components

### Component: ToolCard (Enhanced)

```
┌─────────────────────────────────────────────────┐
│ 🔧 tool_name                    ✅ 1.2s   [▼] │
├─────────────────────────────────────────────────┤
│ [Expanded]                                      │
│ 参数:                                           │
│ ┌─────────────────────────────────────────────┐│
│ │ { "key": "value" }                          ││
│ └─────────────────────────────────────────────┘│
│ 结果: ✅ 成功                                   │
│ ┌─────────────────────────────────────────────┐│
│ │ { "result": "data" }                        ││
│ └─────────────────────────────────────────────┘│
└─────────────────────────────────────────────────┘
```

### Component: RecommendationCard

```
┌─────────────────────────────────────────────────┐
│ 💡 下一步建议                                    │
├─────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────┐│
│ │ 📌 标题                                      ││
│ │    描述内容                                  ││
│ └─────────────────────────────────────────────┘│
│ ┌─────────────────────────────────────────────┐│
│ │ 📊 标题                                      ││
│ │    描述内容                                  ││
│ └─────────────────────────────────────────────┘│
└─────────────────────────────────────────────────┘
```

---

## Error Handling

| Scenario | Error Code | User Message | Recovery |
|----------|------------|--------------|----------|
| 会话恢复失败 | NETWORK_ERROR | 加载历史对话失败 | 显示新建会话 |
| 工具结果过大 | N/A | 结果已截断 | 可展开查看完整 |
| 场景提示获取失败 | API_ERROR | 使用默认提示 | 显示通用提示 |

---

## Performance Requirements

| Metric | Target | Measurement |
|--------|--------|-------------|
| 会话恢复时间 | < 1s | 从挂载到渲染完成 |
| 工具卡片渲染 | < 100ms | 单个卡片渲染 |
| 推荐卡片渲染 | < 50ms | 单个卡片渲染 |
| 场景提示加载 | < 500ms | API 响应时间 |

---

## Security Considerations

- 所有 API 需要 Authorization header
- 工具参数可能包含敏感信息，前端展示时需脱敏处理
- 会话数据按用户隔离，后端验证 user_id

---

### SPEC-6: 消息复制功能

**Requirement**: 用户可将 AI 回复内容复制到剪贴板。

**Behavior**:
- 助手消息底部显示复制按钮
- 点击后复制纯文本内容（不含 Markdown 格式）
- 显示复制成功的 toast 提示

**Acceptance Criteria**:
- [GIVEN] 用户查看一条助手消息
- [WHEN] 点击复制按钮
- [THEN] 消息内容被复制到剪贴板，显示"已复制"提示

**Error Handling**:
- 权限被拒绝: 显示"复制失败，请检查浏览器权限"
- 空消息: 按钮禁用

---

### SPEC-7: 重新生成功能

**Requirement**: 用户可对不满意的 AI 回复进行重新生成。

**Behavior**:
- 助手消息底部显示"重新生成"按钮
- 点击后删除当前回复，重新发送相同问题
- 显示重新生成中的 loading 状态
- 新回复替换原位置

**Data Flow**:
```
用户点击"重新生成"
    → 定位该助手消息之前的用户消息
    → 从本地状态移除当前助手消息
    → 使用相同上下文重新发送请求
    → 新的助手消息渲染在原位置
```

**Acceptance Criteria**:
- [GIVEN] 存在一条助手消息及其对应的用户消息
- [WHEN] 用户点击"重新生成"
- [THEN] 删除当前助手消息，发送相同问题，显示新生成的回复

**Edge Cases**:
- 首条消息（无用户问题）: 不显示重新生成按钮
- 正在生成中: 禁用重新生成按钮
- 连续快速点击: 添加防抖，避免重复请求

---

## UI Components (Updated)

### Component: MessageActions

```
┌─────────────────────────────────────────────────────────┐
│ [AI 消息内容...]                                        │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ 📋 复制  │  👍 有帮助  │  👎 无帮助  │  │  🔄 重新生成 │ │
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘

States:
- Normal: 所有按钮可点击
- Loading: 重新生成按钮显示 loading spinner
- Disabled: 复制按钮在空消息时禁用
```

### Component: ToolCard (Enhanced)

```
┌─────────────────────────────────────────────────┐
│ 🔧 tool_name                    ✅ 1.2s   [▼] │
├─────────────────────────────────────────────────┤
│ [Expanded]                                      │
│ 参数:                                           │
│ ┌─────────────────────────────────────────────┐│
│ │ { "key": "value" }                          ││
│ │ └─────────────────────────────────────────────┘│
│ 结果: ✅ 成功                                   │
│ ┌─────────────────────────────────────────────┐│
│ │ { "result": "data" }                        ││
│ │ └─────────────────────────────────────────────┘│
└─────────────────────────────────────────────────┘
```

### Component: RecommendationCard

```
┌─────────────────────────────────────────────────┐
│ 💡 下一步建议                                    │
├─────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────┐│
│ │ 📌 标题                                      ││
│ │    描述内容                                  ││
│ │ └─────────────────────────────────────────────┘│
│ ┌─────────────────────────────────────────────┐│
│ │ 📊 标题                                      ││
│ │    描述内容                                  ││
│ │ └─────────────────────────────────────────────┘│
└─────────────────────────────────────────────────┘
```

---

## Error Handling (Updated)

| Scenario | Error Code | User Message | Recovery |
|----------|------------|--------------|----------|
| 会话恢复失败 | NETWORK_ERROR | 加载历史对话失败 | 显示新建会话 |
| 工具结果过大 | N/A | 结果已截断 | 可展开查看完整 |
| 场景提示获取失败 | API_ERROR | 使用默认提示 | 显示通用提示 |
| **复制权限被拒** | PERMISSION_DENIED | 复制失败，请检查浏览器权限 | 手动选择复制 |
| **重新生成失败** | NETWORK_ERROR | 重新生成失败，请重试 | 保留原消息 |

---

## Performance Requirements (Updated)

| Metric | Target | Measurement |
|--------|--------|-------------|
| 会话恢复时间 | < 1s | 从挂载到渲染完成 |
| 工具卡片渲染 | < 100ms | 单个卡片渲染 |
| 推荐卡片渲染 | < 50ms | 单个卡片渲染 |
| 场景提示加载 | < 500ms | API 响应时间 |
| **复制操作响应** | < 100ms | 从点击到剪贴板写入 |
| **重新生成请求启动** | < 200ms | 从点击到请求发出 |
