# AI 对话 Runtime 持久化与懒加载设计

**日期**: 2026-03-18
**状态**: Draft
**相关文件**:
- `internal/model/ai.go`
- `internal/ai/runtime/project.go` (新增 `PersistedState`)
- `internal/service/ai/logic/logic.go`
- `internal/service/ai/handler/session.go`
- `web/src/components/AI/AssistantReply.tsx`
- `web/src/components/AI/historyRuntime.ts`
- `web/src/api/modules/ai.ts`

## 背景与问题

### 当前状态

前端 `AssistantReply` 组件依赖 `content` + `runtime` 两个字段来展示完整的对话效果：
- **实时流式对话**：通过 SSE 事件动态构建 `AssistantReplyRuntime`，包含步骤、活动、摘要等
- **历史对话**：后端 `AIChatMessage` 只存储 `content`，没有持久化 runtime 状态

### 问题

历史对话加载后：
1. 没有步骤折叠效果
2. 没有活动记录显示
3. 没有摘要信息
4. 显示效果与实时对话不一致

## 目标

1. 持久化消息的 runtime 状态到数据库
2. 支持按需懒加载 runtime 数据
3. 历史对话默认全部折叠，点击展开时加载详情
4. 减少历史会话加载的数据量

## 设计方案

### 1. 数据模型

在 `AIChatMessage` 表增加 `runtime_json` 字段：

```go
// AIChatMessage stores the final persisted messages for a session.
type AIChatMessage struct {
    ID           string         `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
    SessionID    string         `gorm:"column:session_id;type:varchar(64);not null" json:"session_id"`
    SessionIDNum int            `gorm:"column:session_id_num;not null;default:0" json:"session_id_num"`
    Role         string         `gorm:"column:role;type:varchar(16);not null" json:"role"`
    Content      string         `gorm:"column:content;type:longtext;not null" json:"content"`
    Status       string         `gorm:"column:status;type:varchar(16);not null" json:"status"`
    RuntimeJSON  string         `gorm:"column:runtime_json;type:longtext" json:"-"` // 新增，不返回给前端
    CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
    DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}
```

### 2. Go 类型定义

在 `internal/ai/runtime/` 新增持久化类型：

```go
// Package runtime 定义 AI 运行时类型。
package runtime

// PersistedRuntime 存储到数据库的运行时状态。
//
// 与前端 AssistantReplyRuntime 类型保持一致。
// 字段说明:
//   - Phase: 执行阶段枚举（preparing/planning/executing/completed）
//   - PhaseLabel: 阶段显示文本（如"正在规划"）
//   - Plan: 步骤计划信息
//   - Activities: 工具调用活动记录
//   - Summary: 执行摘要
//   - Status: 运行时状态（streaming/completed/error）
type PersistedRuntime struct {
    Phase      string               `json:"phase,omitempty"`
    PhaseLabel string               `json:"phaseLabel,omitempty"`
    Plan       *PersistedPlan       `json:"plan,omitempty"`
    Activities []PersistedActivity  `json:"activities,omitempty"`
    Summary    *PersistedSummary    `json:"summary,omitempty"`
    Status     *PersistedStatus     `json:"status,omitempty"`
}

// PersistedPlan 步骤计划。
type PersistedPlan struct {
    Steps           []PersistedStep `json:"steps,omitempty"`
    ActiveStepIndex int             `json:"activeStepIndex,omitempty"` // 当前活动步骤索引，-1 表示无活动步骤
}

// PersistedPlanStep 单个步骤。
type PersistedStep struct {
    ID      string `json:"id"`
    Title   string `json:"title"`
    Status  string `json:"status"` // pending, active, done
    Content string `json:"content,omitempty"`
}

// PersistedActivity 工具调用活动。
type PersistedActivity struct {
    ID        string `json:"id"`
    Kind      string `json:"kind"`      // tool_call, tool_result, tool_approval
    Label     string `json:"label"`     // 工具名称
    Detail    string `json:"detail,omitempty"`
    Status    string `json:"status,omitempty"`
    StepIndex int    `json:"stepIndex,omitempty"`
}

// PersistedSummary 执行摘要。
type PersistedSummary struct {
    Title string              `json:"title,omitempty"`
    Items []PersistedSummaryItem `json:"items,omitempty"`
}

// PersistedSummaryItem 摘要项。
type PersistedSummaryItem struct {
    Label string `json:"label"`
    Value string `json:"value"`
    Tone  string `json:"tone,omitempty"` // default, success, warning, danger
}

// PersistedStatus 运行时状态。
type PersistedStatus struct {
    Kind  string `json:"kind"`           // streaming, completed, error, interrupted
    Label string `json:"label,omitempty"` // 状态显示文本
}
```

### 3. 扩展 ProjectionState

修改 `internal/ai/runtime/project.go`，在 `ProjectionState` 中添加持久化字段：

```go
// ProjectionState 跟踪流式投影状态。
type ProjectionState struct {
    // 现有字段（解析状态）
    TotalPlanSteps         int
    ReplanIteration        int
    RunPhase               string
    PendingPlannerJSON     string
    PendingReplannerJSON   string
    ReplannerResponseState *ResponseExtractState

    // 新增：持久化状态跟踪
    Persisted *PersistedRuntime // 累积的运行时状态
}

// 初始化时创建 Persisted
func NewStreamProjector() *StreamProjector {
    return &StreamProjector{
        buffer: NewDeltaBuffer(DeltaBufferConfig{
            MinChunkSize: 50,
            MaxWaitMs:    100,
        }),
        state: ProjectionState{
            Persisted: &PersistedRuntime{},
        },
    }
}

// GetPersistedState 获取累积的持久化状态。
//
// 在流结束时调用此方法获取完整的 runtime 数据用于存储。
func (p *StreamProjector) GetPersistedState() *PersistedRuntime {
    return p.state.Persisted
}
```

### 4. 持久化状态累积

在 `projectNormalizedEvent` 中同步更新 `Persisted`：

```go
func projectNormalizedEvent(event NormalizedEvent, state *ProjectionState) []PublicStreamEvent {
    // 现有逻辑先生成 SSE 事件
    events := make([]PublicStreamEvent, 0)

    switch event.Kind {
    case NormalizedKindHandoff:
        if event.Handoff != nil {
            // 更新持久化状态
            state.Persisted.Phase = "executing"
            state.Persisted.PhaseLabel = fmt.Sprintf("%s 开始处理", event.Handoff.To)
            // 记录 agent_handoff 活动
            state.Persisted.Activities = append(state.Persisted.Activities, PersistedActivity{
                ID:     fmt.Sprintf("handoff:%s", event.Handoff.To),
                Kind:   "agent_handoff",
                Label:  event.Handoff.To,
                Status: "done",
            })
        }

    case NormalizedKindInterrupt:
        if event.Interrupt != nil {
            state.Persisted.Phase = "waiting_approval"
            state.Persisted.PhaseLabel = "等待审批"
            state.Persisted.Activities = append(state.Persisted.Activities, PersistedActivity{
                ID:     event.Interrupt.CallID,
                Kind:   "tool_approval",
                Label:  event.Interrupt.ToolName,
                Detail: fmt.Sprintf("等待审批 %ds", event.Interrupt.TimeoutSeconds),
                Status: "pending",
            })
        }

    case NormalizedKindToolCall:
        if event.Tool != nil {
            state.Persisted.Activities = append(state.Persisted.Activities, PersistedActivity{
                ID:        event.Tool.CallID,
                Kind:      "tool_call",
                Label:     event.Tool.ToolName,
                Status:    "active",
                StepIndex: state.Persisted.Plan.ActiveStepIndex,
            })
        }

    case NormalizedKindToolResult:
        if event.Tool != nil {
            // 更新对应 activity 状态
            for i := range state.Persisted.Activities {
                if state.Persisted.Activities[i].ID == event.Tool.CallID {
                    state.Persisted.Activities[i].Status = "done"
                    state.Persisted.Activities[i].Detail = truncateString(event.Tool.Content, 200)
                }
            }
        }

    case NormalizedKindMessage:
        trimmedAgent := strings.TrimSpace(event.AgentName)

        // planner 发送 plan 事件
        if trimmedAgent == "planner" {
            raw := appendAgentJSONBuffer(state.PendingPlannerJSON, event.Message.Content)
            if steps, ok := decodeStepsEnvelope(strings.TrimSpace(raw)); ok {
                state.PendingPlannerJSON = ""
                state.TotalPlanSteps = len(steps)
                // 更新持久化 plan
                state.Persisted.Plan = buildPersistedPlanFromSteps(steps, 0)
                state.Persisted.Phase = "planning"
                state.Persisted.PhaseLabel = "正在规划处理方式"
            } else if shouldBufferAgentEnvelope(state.PendingPlannerJSON, raw) {
                state.PendingPlannerJSON = raw
            }
        }

        // replanner 发送 replan 事件
        if trimmedAgent == "replanner" {
            raw := appendAgentJSONBuffer(state.PendingReplannerJSON, event.Message.Content)
            state.PendingReplannerJSON = raw

            if steps, ok := decodeStepsEnvelope(strings.TrimSpace(raw)); ok {
                state.PendingReplannerJSON = ""
                state.ReplanIteration++
                completed := state.TotalPlanSteps - len(steps)
                if completed < 0 {
                    completed = 0
                }
                // 更新持久化 plan
                state.Persisted.Plan = buildPersistedPlanFromSteps(steps, completed)
                state.Persisted.Phase = "planning"
                state.Persisted.PhaseLabel = "正在调整处理计划"
            }
        }
    }

    return events
}

// buildPersistedPlanFromSteps 从步骤数组构建 PersistedPlan。
func buildPersistedPlanFromSteps(steps []string, completedCount int) *PersistedPlan {
    result := &PersistedPlan{}
    for i, title := range steps {
        status := "pending"
        if i < completedCount {
            status = "done"
        } else if i == completedCount {
            status = "active"
        }
        result.Steps = append(result.Steps, PersistedStep{
            ID:     fmt.Sprintf("plan-step-%d", i),
            Title:  title,
            Status: status,
        })
    }
    if completedCount < len(steps) {
        result.ActiveStepIndex = completedCount
    }
    return result
}

// Finish 完成流式处理，设置最终状态。
func (p *StreamProjector) Finish(runID string) PublicStreamEvent {
    // 设置最终状态
    p.state.Persisted.Phase = "completed"
    p.state.Persisted.PhaseLabel = "已完成"
    p.state.Persisted.Status = &PersistedStatus{
        Kind:  "completed",
        Label: "已生成",
    }
    // 清除活动步骤索引
    if p.state.Persisted.Plan != nil {
        p.state.Persisted.Plan.ActiveStepIndex = -1
        // 标记所有步骤为完成
        for i := range p.state.Persisted.Plan.Steps {
            p.state.Persisted.Plan.Steps[i].Status = "done"
        }
    }

    return doneEvent(runID, p.state.ReplanIteration)
}
```

### 5. API 设计

#### 新增接口：获取消息 Runtime

**请求：**
```
GET /api/v1/ai/messages/:id/runtime
```

**响应：**
```json
{
  "code": 1000,
  "msg": "请求成功",
  "data": {
    "message_id": "xxx",
    "runtime": {
      "phase": "completed",
      "plan": { "steps": [...] },
      "activities": [...],
      "summary": { ... }
    }
  }
}
```

**错误情况：**
- 消息不存在：`{ "code": 2005, "msg": "消息不存在" }`
- 无权限：`{ "code": 2003, "msg": "无权限访问" }`
- 无 runtime：`{ "code": 1000, "data": { "message_id": "xxx", "runtime": null } }`

#### 修改 GetSession 接口

返回的消息列表不包含 `runtime_json`，但增加 `has_runtime` 标志：

```json
{
  "id": "xxx",
  "content": "...",
  "status": "done",
  "has_runtime": true
}
```

### 6. 后端 Handler 实现

```go
// MessageRuntimeResponse 消息 runtime 响应结构。
type MessageRuntimeResponse struct {
    MessageID string                 `json:"message_id"`
    Runtime   map[string]any         `json:"runtime,omitempty"`
}

// GetMessageRuntime 获取单条消息的运行时状态。
//
// 权限验证：检查消息所属会话是否属于当前用户。
func (h *Handler) GetMessageRuntime(c *gin.Context) {
    messageID := c.Param("id")
    userID := httpx.UIDFromCtx(c)

    // 获取消息并验证权限
    message, err := h.logic.GetMessageWithOwnership(c.Request.Context(), userID, messageID)
    if err != nil {
        httpx.ServerErr(c, err)
        return
    }
    if message == nil {
        httpx.NotFound(c, "消息不存在或无权限访问")
        return
    }

    // 解析 runtime JSON
    var runtime map[string]any
    if message.RuntimeJSON != "" {
        if err := json.Unmarshal([]byte(message.RuntimeJSON), &runtime); err != nil {
            // JSON 解析失败，返回 null
            runtime = nil
        }
    }

    httpx.OK(c, MessageRuntimeResponse{
        MessageID: messageID,
        Runtime:   runtime,
    })
}
```

### 7. Logic 层实现

```go
// GetMessageWithOwnership 获取消息并验证所有权。
func (l *Logic) GetMessageWithOwnership(ctx context.Context, userID uint64, messageID string) (*model.AIChatMessage, error) {
    if l.ChatDAO == nil {
        return nil, nil
    }

    // 获取消息
    message, err := l.ChatDAO.GetMessage(ctx, messageID)
    if err != nil || message == nil {
        return nil, err
    }

    // 验证会话所有权
    session, err := l.ChatDAO.GetSession(ctx, message.SessionID, userID, "")
    if err != nil {
        return nil, err
    }
    if session == nil {
        return nil, nil // 无权限
    }

    return message, nil
}
```

### 7.1 DAO 层新增方法

在 `internal/dao/ai/chat.go` 新增：

```go
// GetMessage 根据 ID 获取单条消息。
func (d *AIChatDAO) GetMessage(ctx context.Context, messageID string) (*model.AIChatMessage, error) {
    var message model.AIChatMessage
    if err := d.db.WithContext(ctx).Where("id = ?", messageID).First(&message).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, err
    }
    return &message, nil
}
```

### 8. 前端实现

#### 8.1 API 模块

```typescript
// web/src/api/modules/ai.ts
export const aiApi = {
  // ... 现有方法

  // 获取单条消息的 runtime 数据
  async getMessageRuntime(id: string): Promise<ApiResponse<{
    message_id: string;
    runtime: AssistantReplyRuntime | null;
  }>> {
    return apiService.get(`/ai/messages/${id}/runtime`);
  },
};
```

#### 8.2 类型定义扩展

```typescript
// web/src/components/AI/types.ts

// 扩展 XChatMessage，增加 id 字段
export interface XChatMessage {
  id?: string;  // 新增：消息 ID，用于懒加载
  role: 'user' | 'assistant';
  content: string;
  runtime?: AssistantReplyRuntime;
  hasRuntime?: boolean;  // 新增：标记是否有 runtime 数据
}
```

#### 8.3 历史消息处理

```typescript
// web/src/components/AI/historyRuntime.ts

const MAX_CACHE_SIZE = 50; // 最大缓存数量

const runtimeCache = new Map<string, AssistantReplyRuntime>();
const cacheOrder: string[] = []; // LRU 顺序

function evictOldest() {
  if (cacheOrder.length > MAX_CACHE_SIZE) {
    const oldest = cacheOrder.shift();
    if (oldest) {
      runtimeCache.delete(oldest);
    }
  }
}

export async function loadMessageRuntime(messageId: string): Promise<AssistantReplyRuntime | null> {
  // 缓存命中
  if (runtimeCache.has(messageId)) {
    // 更新 LRU 顺序
    const index = cacheOrder.indexOf(messageId);
    if (index > -1) {
      cacheOrder.splice(index, 1);
      cacheOrder.push(messageId);
    }
    return runtimeCache.get(messageId)!;
  }

  try {
    const response = await aiApi.getMessageRuntime(messageId);
    if (response.data?.runtime) {
      runtimeCache.set(messageId, response.data.runtime);
      cacheOrder.push(messageId);
      evictOldest();
      return response.data.runtime;
    }
  } catch (error) {
    console.error('Failed to load runtime:', error);
  }
  return null;
}

// 修改 hydrateAssistantHistoryMessage
export function hydrateAssistantHistoryMessage(message: AIMessage): XChatMessage {
  return {
    id: message.id,  // 传递 ID
    role: message.role === 'assistant' ? 'assistant' : 'user',
    content: message.content || '',
    hasRuntime: Boolean(message.has_runtime),  // 从 API 获取
    // 不再尝试合成 runtime
  };
}
```

#### 8.4 AssistantReply 组件

```typescript
interface AssistantReplyProps {
  content: string;
  runtime?: AssistantReplyRuntime;
  status?: string;
  messageId?: string;
  hasRuntime?: boolean;
  onLoadRuntime?: (messageId: string) => Promise<AssistantReplyRuntime | null>;
}

export function AssistantReply({
  content,
  runtime,
  status,
  messageId,
  hasRuntime,
  onLoadRuntime
}: AssistantReplyProps) {
  const [localRuntime, setLocalRuntime] = useState<AssistantReplyRuntime | null>(null);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState(false);

  // 显示的 runtime：优先使用已加载的，其次使用传入的
  const displayRuntime = localRuntime || runtime;

  // 有 runtime 时直接显示
  if (displayRuntime) {
    return <AssistantReplyContent content={content} runtime={displayRuntime} status={status} />;
  }

  // 无 runtime 且没有懒加载能力
  if (!messageId || !hasRuntime || !onLoadRuntime) {
    return <SimpleMarkdownContent content={content} />;
  }

  // 懒加载场景
  const handleExpand = async () => {
    if (loading) return;
    setLoading(true);
    setExpanded(true);
    const loaded = await onLoadRuntime(messageId);
    if (loaded) {
      setLocalRuntime(loaded);
    }
    setLoading(false);
  };

  if (loading) {
    return (
      <div>
        <Skeleton active paragraph={{ rows: 3 }} />
      </div>
    );
  }

  if (!expanded) {
    return (
      <div>
        <SimpleMarkdownContent content={content} />
        <Button type="link" onClick={handleExpand}>
          展开详情
        </Button>
      </div>
    );
  }

  return <AssistantReplyContent content={content} runtime={localRuntime!} status={status} />;
}
```

### 9. 用户体验流程

```
用户打开历史会话
    ↓
显示消息列表
  - 有 has_runtime=true 的消息显示"展开详情"按钮
  - 只显示 content（markdown）
    ↓
用户点击"展开详情"按钮
    ↓
显示 loading skeleton
    ↓
调用 getMessageRuntime(messageId)
    ↓
渲染完整 runtime（步骤折叠、活动、摘要）
```

### 10. 边界情况处理

| 场景 | 处理方式 |
|------|----------|
| `runtime_json` 为空 | API 返回 `runtime: null`，前端不显示展开按钮 |
| `runtime_json` JSON 格式错误 | 后端解析失败返回 `null`，前端降级为纯文本 |
| 消息不存在 | 返回 404 错误 |
| 无权限访问 | 返回 403 错误 |
| 缓存已满 | LRU 淘汰最旧的条目 |
| 网络请求失败 | 显示错误提示，保留展开按钮供重试 |

### 11. 迁移策略

由于用户表示无需向后兼容，不需要数据迁移：
1. 旧消息没有 `runtime_json`，`has_runtime` 为 false，保持原有显示方式
2. 新消息自动存储 runtime
3. 用户可删除旧会话

## 实现步骤

1. **后端 Model 改动**
   - 增加 `RuntimeJSON` 字段
   - 数据库迁移：项目使用 GORM AutoMigrate，字段会自动添加

   手动迁移 SQL（如需要）：
   ```sql
   -- storage/migration/20260319_add_runtime_json_to_messages.sql
   ALTER TABLE ai_chat_messages ADD COLUMN runtime_json LONGTEXT AFTER status;
   ```

2. **后端 Runtime 类型定义**
   - 新增 `PersistedRuntime` 等类型到 `internal/ai/runtime/types.go`
   - 扩展 `ProjectionState` 添加 `Persisted` 字段

3. **后端状态累积**
   - 修改 `projectNormalizedEvent` 同步更新 `Persisted`
   - 添加 `GetPersistedState()` 方法
   - 修改 `Finish()` 设置最终状态

4. **后端 Logic 改动**
   - 在 `logic.Chat` Step 7 提取并保存 runtime：
   ```go
   runtimeJSON := ""
   if persisted := projector.GetPersistedState(); persisted != nil && len(persisted.Activities) > 0 {
       if data, err := json.Marshal(persisted); err == nil {
           runtimeJSON = string(data)
       }
   }
   if err := l.ChatDAO.UpdateMessage(ctx, assistantMessage.ID, map[string]any{
       "content":      finalContent,
       "status":       finalStatus,
       "runtime_json": runtimeJSON,
   }); err != nil {
       return fmt.Errorf("update assistant message: %w", err)
   }
   ```

5. **后端 API 新增**
   - 实现 `GetMessageWithOwnership` Logic 方法
   - 实现 `GetMessageRuntime` handler
   - 修改 `GetSession` 返回 `has_runtime` 标志
   - 在 `routes.go` 注册路由：`messages.GET("/:id/runtime", h.GetMessageRuntime)`

6. **前端 API 模块**
   - 增加 `getMessageRuntime` API

7. **前端类型扩展**
   - 扩展 `XChatMessage` 类型

8. **前端组件改造**
   - 修改 `historyRuntime.ts` 支持懒加载和 LRU 缓存
   - 修改 `AssistantReply.tsx` 支持 loading 状态和懒加载

## 测试要点

### 功能测试

1. 新对话完成后，检查 `runtime_json` 是否正确存储
2. 历史对话加载时，确认不返回 runtime
3. `has_runtime` 标志正确标识消息是否有 runtime
4. 懒加载 API 返回正确的 runtime 数据
5. 前端正确显示 loading 状态和展开后的 runtime

### 边界测试

1. **空 runtime**：消息无 runtime 时不显示展开按钮
2. **JSON 解析失败**：后端返回 null，前端降级显示
3. **权限验证**：访问他人消息返回 403
4. **缓存 LRU**：超过 50 条消息时淘汰最旧缓存

### 性能测试

1. 大型会话（100+ 消息）加载性能
2. 连续展开多条消息的加载速度
3. 缓存命中/未命中场景
