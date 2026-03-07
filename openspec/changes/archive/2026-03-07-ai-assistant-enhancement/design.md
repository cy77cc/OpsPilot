# AI Assistant Enhancement - Design Document

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Frontend Layer                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Copilot.tsx                                                               │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │  useConversationRestore()  ──── 自动恢复上次会话                      │   │
│   │  useScenePrompts(scene)    ──── 获取场景快捷指令                      │   │
│   │  useAIChat()               ──── 消息状态管理                          │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│   Components:                                                               │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │
│   │ Think       │  │ ToolCard    │  │ Recommend   │  │ Prompts     │      │
│   │ (思考过程)   │  │ (工具详情)   │  │ Card (推荐) │  │ (快捷指令)  │      │
│   └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      │ HTTP/SSE
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Backend Layer                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   API Routes:                                                               │
│   ├── GET  /ai/scene/:scene/prompts    ──── 获取场景快捷指令               │
│   ├── GET  /ai/sessions/current        ──── 获取当前会话                   │
│   ├── GET  /ai/sessions                ──── 获取会话列表                   │
│   └── POST /ai/chat                    ──── 对话 (SSE)                     │
│                                                                             │
│   SSE Events:                                                               │
│   ├── thinking_delta  ─→ 思考过程增量                                      │
│   ├── tool_call       ─→ { tool, call_id, payload: { params } }           │
│   ├── tool_result     ─→ { tool, call_id, result: { ok, data, error } }   │
│   └── done            ─→ { turn_recommendations: [...] }                  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Data Layer                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ai_chat_sessions      ──── 会话元数据                                     │
│   ai_chat_messages      ──── 消息记录                                       │
│   ai_scene_prompts      ──── 场景快捷指令 (新增)                            │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Component Design

### 1. Think Component Integration

**Current Implementation** (`Copilot.tsx:61-100`):
```tsx
const ThinkingBlock: React.FC<{ content: string; isStreaming?: boolean }> = ({ content, isStreaming }) => {
  // Uses Ant Design Collapse
  return (
    <Collapse
      items={[{ key: 'thinking', label: '...', children: <div>...</div> }]}
    />
  );
};
```

**New Implementation**:
```tsx
import { Think } from '@ant-design/x';

// In AssistantMessage component
{thinking && (
  <Think
    content={thinking}
    status={isStreaming ? 'loading' : 'done'}
  />
)}
```

### 2. Conversation Restore Hook

```tsx
// hooks/useConversationRestore.ts
interface UseConversationRestoreOptions {
  scene: string;
  onRestore: (session: AISession) => void;
}

export function useConversationRestore(options: UseConversationRestoreOptions) {
  const { scene, onRestore } = options;
  const [isRestoring, setIsRestoring] = useState(true);

  useEffect(() => {
    const restore = async () => {
      try {
        // 1. 尝试获取当前会话
        const currentRes = await aiApi.getCurrentSession(scene);
        if (currentRes.data) {
          onRestore(currentRes.data);
          return;
        }

        // 2. 如果没有当前会话，获取最近会话列表
        const listRes = await aiApi.getSessions(scene);
        if (listRes.data?.length > 0) {
          const recent = listRes.data[0];
          const detailRes = await aiApi.getSessionDetail(recent.id);
          if (detailRes.data) {
            onRestore(detailRes.data);
          }
        }
      } catch (error) {
        console.error('Failed to restore session:', error);
      } finally {
        setIsRestoring(false);
      }
    };

    restore();
  }, [scene, onRestore]);

  return { isRestoring };
}
```

### 3. Tool Execution Display Enhancement

**Type Extension**:
```typescript
// types.ts
export interface ToolExecution {
  id: string;
  name: string;
  status: ToolStatus;
  duration?: number;
  error?: string;
  // NEW FIELDS
  params?: Record<string, unknown>;
  result?: {
    ok: boolean;
    data?: unknown;
    error?: string;
    latency_ms?: number;
  };
}
```

**Enhanced ToolCard**:
```tsx
// components/ToolCard.tsx
export function ToolCard({ tool }: ToolCardProps) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="ai-tool-card enhanced">
      <div className="tool-header">
        <span className="tool-icon">🔧</span>
        <span className="tool-name">{formatToolName(tool.name)}</span>
        <span className={`tool-status ${tool.status}`}>
          {getStatusIcon(tool.status)}
        </span>
        {tool.duration && (
          <span className="tool-duration">{tool.duration.toFixed(1)}s</span>
        )}
        <Button
          type="text"
          size="small"
          icon={expanded ? <UpOutlined /> : <DownOutlined />}
          onClick={() => setExpanded(!expanded)}
        />
      </div>

      {expanded && (
        <div className="tool-details">
          {tool.params && (
            <div className="tool-params">
              <div className="detail-label">参数:</div>
              <pre>{JSON.stringify(tool.params, null, 2)}</pre>
            </div>
          )}
          {tool.result && (
            <div className="tool-result">
              <div className="detail-label">
                结果: {tool.result.ok ? '✅ 成功' : '❌ 失败'}
              </div>
              {tool.result.data && (
                <pre className="result-data">
                  {formatResult(tool.result.data)}
                </pre>
              )}
              {tool.result.error && (
                <div className="result-error">{tool.result.error}</div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
```

### 4. Recommendation Card Component

```tsx
// components/RecommendationCard.tsx
interface RecommendationCardProps {
  recommendations: EmbeddedRecommendation[];
  onSelect: (prompt: string) => void;
}

export function RecommendationCard({ recommendations, onSelect }: RecommendationCardProps) {
  if (!recommendations?.length) return null;

  return (
    <div className="recommendation-card">
      <div className="recommendation-header">
        <BulbOutlined /> 下一步建议
      </div>
      <div className="recommendation-list">
        {recommendations.map((rec) => (
          <div
            key={rec.id}
            className="recommendation-item"
            onClick={() => rec.followup_prompt && onSelect(rec.followup_prompt)}
          >
            <div className="rec-title">{rec.title}</div>
            {rec.content && (
              <div className="rec-content">{rec.content}</div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
```

### 5. Scene Prompts System

**Backend Model** (`internal/model/ai_scene_prompt.go`):
```go
package model

import "time"

type AIScenePrompt struct {
    ID            uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Scene         string    `gorm:"column:scene;type:varchar(128);index:idx_scene_prompts" json:"scene"`
    PromptText    string    `gorm:"column:prompt_text;type:text" json:"prompt_text"`
    PromptType    string    `gorm:"column:prompt_type;type:varchar(32);default:'quick_action'" json:"prompt_type"`
    DisplayOrder  int       `gorm:"column:display_order;default:0" json:"display_order"`
    IsActive      bool      `gorm:"column:is_active;default:true" json:"is_active"`
    CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AIScenePrompt) TableName() string { return "ai_scene_prompts" }
```

**Scene Prompts Handler** (`internal/service/ai/handler/scene_prompt_handler.go`):
```go
package handler

import (
    "context"
    "github.com/cy77cc/k8s-manage/internal/httpx"
    "github.com/cy77cc/k8s-manage/internal/xcode"
    "github.com/gin-gonic/gin"
)

func (h *AIHandler) scenePrompts(c *gin.Context) {
    scene := c.Param("scene")
    prompts := h.getScenePrompts(scene)
    httpx.OK(c, gin.H{
        "scene":   scene,
        "prompts": prompts,
    })
}

func (h *AIHandler) getScenePrompts(scene string) []gin.H {
    // 1. Check database cache
    // 2. If not found, generate from scene config + LLM
    // 3. Cache to database
    // 4. Return prompts
}
```

**Frontend Hook** (`hooks/useScenePrompts.ts`):
```tsx
export function useScenePrompts(scene: string) {
  const [prompts, setPrompts] = useState<PromptItem[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const loadPrompts = async () => {
      if (!scene) return;
      setLoading(true);
      try {
        const res = await aiApi.getScenePrompts(scene);
        setPrompts(res.data?.prompts || getDefaultPrompts(scene));
      } catch {
        setPrompts(getDefaultPrompts(scene));
      } finally {
        setLoading(false);
      }
    };
    loadPrompts();
  }, [scene]);

  return { prompts, loading };
}

function getDefaultPrompts(scene: string): PromptItem[] {
  // Fallback prompts based on scene
  const defaults: Record<string, PromptItem[]> = {
    'deployment:clusters': [
      { text: '查看所有集群状态' },
      { text: '帮我检查集群健康度' },
    ],
    // ... more defaults
  };
  return defaults[scene] || [
    { text: '有什么可以帮助你的？' },
  ];
}
```

## Data Flow

### Conversation Restore Flow

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Page Load    │────▶│ getCurrent   │────▶│ Session Found?│
│              │     │ Session()    │     └──────┬───────┘
└──────────────┘     └──────────────┘            │
                                           ┌─────┴─────┐
                                           │           │
                                          Yes          No
                                           │           │
                                           ▼           ▼
                                    ┌──────────┐ ┌──────────────┐
                                    │ Restore  │ │ getSessions()│
                                    │ Messages │ │              │
                                    └──────────┘ └──────┬───────┘
                                                        │
                                                        ▼
                                                 ┌──────────────┐
                                                 │ Load Most    │
                                                 │ Recent       │
                                                 └──────────────┘
```

### Tool Event Processing Flow

```
SSE Event: tool_call
    │
    ▼
┌─────────────────────────────┐
│ Extract:                    │
│ - tool name                 │
│ - call_id                   │
│ - payload (params)          │
└─────────────────────────────┘
    │
    ▼
Update ToolExecution in message

SSE Event: tool_result
    │
    ▼
┌─────────────────────────────┐
│ Extract:                    │
│ - tool name                 │
│ - call_id                   │
│ - result.ok                 │
│ - result.data               │
│ - result.error              │
│ - latency_ms                │
└─────────────────────────────┘
    │
    ▼
Update ToolExecution status & result
```

## UI Specifications

### Tool Card States

```
┌─ Running ──────────────────────────────────────────────┐
│ 🔧 host_list_inventory          ⏳ 执行中...          │
└────────────────────────────────────────────────────────┘

┌─ Success (Collapsed) ─────────────────────────────────┐
│ 🔧 host_list_inventory          ✅ 1.2s        [▼]   │
└────────────────────────────────────────────────────────┘

┌─ Success (Expanded) ──────────────────────────────────┐
│ 🔧 host_list_inventory          ✅ 1.2s        [▲]   │
│ ┌──────────────────────────────────────────────────┐ │
│ │ 参数:                                            │ │
│ │ { "keyword": "web-server" }                      │ │
│ ├──────────────────────────────────────────────────┤ │
│ │ 结果: ✅ 成功                                     │ │
│ │ [{ "id": 1, "name": "web-server-01", ... }]      │ │
│ └──────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────┘

┌─ Error ───────────────────────────────────────────────┐
│ 🔧 host_ssh_exec                ❌ 0.5s        [▼]   │
│ ┌──────────────────────────────────────────────────┐ │
│ │ 错误: SSH 连接超时                                │ │
│ └──────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────┘
```

### Recommendation Card

```
┌─ 下一步建议 ──────────────────────────────────────────┐
│ ┌──────────────────────────────────────────────────┐ │
│ │ 📌 查看服务健康状态                              │ │
│ │    检查服务的资源使用和运行状态                   │ │
│ └──────────────────────────────────────────────────┘ │
│ ┌──────────────────────────────────────────────────┐ │
│ │ 📊 查看监控指标                                  │ │
│ │    获取服务的实时监控数据                         │ │
│ └──────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────┘
```

## Configuration Changes

### scene_mappings.yaml Addition

```yaml
mappings:
  deployment:clusters:
    # ... existing fields
    prompts:
      - text: "查看所有集群状态"
        type: "quick_action"
      - text: "帮我检查集群健康度"
        type: "quick_action"
      - text: "部署应用到指定集群"
        type: "quick_action"
```

## Migration Script

```sql
-- Migration: Add ai_scene_prompts table
CREATE TABLE IF NOT EXISTS ai_scene_prompts (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  scene VARCHAR(128) NOT NULL COMMENT '场景标识',
  prompt_text TEXT NOT NULL COMMENT '提示词文本',
  prompt_type VARCHAR(32) DEFAULT 'quick_action' COMMENT '类型: quick_action, suggestion',
  display_order INT DEFAULT 0 COMMENT '显示顺序',
  is_active BOOLEAN DEFAULT TRUE COMMENT '是否启用',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_scene_prompts (scene)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='场景快捷提示词';

-- Initial data from config
INSERT INTO ai_scene_prompts (scene, prompt_text, prompt_type, display_order) VALUES
('deployment:clusters', '查看所有集群状态', 'quick_action', 1),
('deployment:clusters', '帮我检查集群健康度', 'quick_action', 2),
('deployment:clusters', '部署应用到指定集群', 'quick_action', 3),
('deployment:hosts', '查看主机列表', 'quick_action', 1),
('deployment:hosts', '执行主机健康检查', 'quick_action', 2),
('services:list', '查看服务目录', 'quick_action', 1),
('services:list', '搜索服务', 'quick_action', 2),
('global', '有什么可以帮助你的？', 'quick_action', 1);
```

## Copy Function Design

### Problem Analysis

Current implementation at `Copilot.tsx:141`:
```tsx
// Current broken code
const createRoleConfig = () => ({
  assistant: {
    placement: 'start' as const,
    footer: (
      <div style={{ display: 'flex', marginTop: 4 }}>
        <Tooltip title="复制">
          <Button type="text" size="small" icon={<CopyOutlined />} />
          {/* ↑ No onClick handler! */}
        </Tooltip>
        ...
      </div>
    ),
  },
});
```

**Root Cause**: The `footer` is static JSX that cannot access message content. Need to use dynamic rendering with message context.

### Solution Design

**Approach 1**: Use `Bubble.List` items with custom `footer` renderer

```tsx
// Enhanced message rendering with action handlers
const renderMessageFooter = useCallback((msg: ChatMessage) => {
  if (msg.role !== 'assistant') return null;

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(msg.content);
      message.success('已复制到剪贴板');
    } catch {
      message.error('复制失败');
    }
  };

  return (
    <div style={{ display: 'flex', marginTop: 4 }}>
      <Tooltip title="复制">
        <Button type="text" size="small" icon={<CopyOutlined />} onClick={handleCopy} />
      </Tooltip>
      <Tooltip title="有帮助">
        <Button type="text" size="small" icon={<LikeOutlined />} />
      </Tooltip>
      <Tooltip title="无帮助">
        <Button type="text" size="small" icon={<DislikeOutlined />} />
      </Tooltip>
    </div>
  );
}, []);
```

**Approach 2**: Use `@ant-design/x` Bubble component's built-in footer support

```tsx
// Using Bubble's typing to pass content
items={messages.map(m => ({
  key: m.id,
  content: m.content,
  role: m.role,
  // Bubble supports footer function with message context
  footer: m.role === 'assistant' ? (
    <MessageActions content={m.content} onRegenerate={handleRegenerate} />
  ) : undefined,
}))}
```

### UI Implementation

```
┌─────────────────────────────────────────────────────────┐
│ [AI 消息内容...]                                        │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ 📋 复制  │  👍 有帮助  │  👎 无帮助  │  🔄 重新生成 │ │
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

---

## Regenerate Function Design

### Feature Overview

Allow users to regenerate AI response when unsatisfied with the current output.

### Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                      Regenerate Flow                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  User clicks "重新生成"                                         │
│       │                                                         │
│       ▼                                                         │
│  ┌─────────────────────────────────────────┐                   │
│  │ 1. Find the user message that triggered │                   │
│  │    this assistant response              │                   │
│  │                                         │                   │
│  │ 2. Remove current assistant message     │                   │
│  │    from local state                     │                   │
│  │                                         │                   │
│  │ 3. Re-send the user message with        │                   │
│  │    same context (sessionId, scene)      │                   │
│  └─────────────────────────────────────────┘                   │
│       │                                                         │
│       ▼                                                         │
│  SSE stream starts, new assistant message appears              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Implementation

```tsx
// In Copilot.tsx
const handleRegenerate = useCallback(async (assistantMsgId: string) => {
  // 1. Find the user message before this assistant message
  const msgIndex = messages.findIndex(m => m.id === assistantMsgId);
  if (msgIndex <= 0) return;

  const userMessage = messages[msgIndex - 1];
  if (userMessage.role !== 'user') return;

  // 2. Remove the assistant message
  setConversations(prev => prev.map(c => {
    if (c.key !== activeKey) return c;
    return {
      ...c,
      messages: c.messages.filter(m => m.id !== assistantMsgId),
    };
  }));

  // 3. Re-send the user message
  await handleSubmit(userMessage.content);
}, [messages, activeKey, handleSubmit]);
```

### Message Actions Component

```tsx
// components/MessageActions.tsx
interface MessageActionsProps {
  message: ChatMessage;
  onCopy: () => void;
  onRegenerate: () => void;
  onLike?: () => void;
  onDislike?: () => void;
}

export function MessageActions({ message, onCopy, onRegenerate, onLike, onDislike }: MessageActionsProps) {
  return (
    <div className="message-actions">
      <Tooltip title="复制">
        <Button type="text" size="small" icon={<CopyOutlined />} onClick={onCopy} />
      </Tooltip>
      <Tooltip title="有帮助">
        <Button type="text" size="small" icon={<LikeOutlined />} onClick={onLike} />
      </Tooltip>
      <Tooltip title="无帮助">
        <Button type="text" size="small" icon={<DislikeOutlined />} onClick={onDislike} />
      </Tooltip>
      <Divider type="vertical" />
      <Tooltip title="重新生成">
        <Button type="text" size="small" icon={<ReloadOutlined />} onClick={onRegenerate} />
      </Tooltip>
    </div>
  );
}
```

### UI States

**Normal State**:
```
┌─────────────────────────────────────────────────────────┐
│ [AI 消息内容...]                                        │
│                                                         │
│ 📋 复制 │ 👍 │ 👎 │ ── │ 🔄 重新生成                    │
└─────────────────────────────────────────────────────────┘
```

**Regenerating State**:
```
┌─────────────────────────────────────────────────────────┐
│ [AI 消息内容...]                                        │
│                                                         │
│ 📋 复制 │ 👍 │ 👎 │ ── │ ⏳ 重新生成中...               │
└─────────────────────────────────────────────────────────┘
```

### Considerations

1. **Context Preservation**: When regenerating, should we:
   - Keep the same session context? ✅ (recommended)
   - Start a new context? (may lose conversation history)

2. **Tool Calls**: What happens to previous tool executions?
   - Clear them and re-execute? ✅ (cleaner)
   - Show cached results? (faster but may be stale)

3. **Rate Limiting**: Consider adding debounce to prevent spam regeneration

4. **History**: Should regeneration be tracked in conversation history?
   - Option A: Replace old response in history ✅
   - Option B: Keep all variants (more complex UI)

