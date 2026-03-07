# 技术设计

## 架构变更

### 前端架构

```
当前架构:
┌─────────────────────────────────────┐
│  [AI 助手] [场景名(条件)]            │  ← 两个按钮
│       ↓        ↓                    │
│  <Drawer> <Drawer>                  │  ← 两个独立抽屉
│  scene="global" scene={scene}       │
└─────────────────────────────────────┘

目标架构:
┌─────────────────────────────────────┐
│  [✨ AI Copilot]                    │  ← 单一入口
│       ↓                             │
│  <Drawer>                           │
│  ┌─────────────────────────────────┐│
│  │ 场景: [全局 ▼] [集群管理] [...]  ││  ← 场景切换
│  └─────────────────────────────────┘│
│  自动感知当前页面场景                │
└─────────────────────────────────────┘
```

### 后端架构

```
internal/ai/ (工具层):
├── agent/                    # Agent 核心逻辑
│   ├── agent.go
│   ├── agent_types.go
│   ├── hybrid_agent.go
│   └── runner.go
├── modes/                    # 运行模式
│   ├── agentic_mode.go
│   └── simple_chat_mode.go
├── store/                    # 存储层
│   ├── store.go
│   └── checkpoint_store.go
├── classifier/               # 意图分类
│   └── classifier.go
└── tools/                    # 工具实现
    ├── registry/
    ├── host/
    ├── k8s/
    └── ...

internal/service/ai/ (服务层):
├── handler/                  # HTTP 处理器
│   ├── chat_handler.go
│   ├── capability_handler.go
│   ├── session_handler.go
│   └── ...
├── logic/                    # 业务逻辑
│   ├── session_store.go
│   ├── store.go
│   └── ...
├── events/                   # SSE 事件
└── knowledge/                # 知识库
```

## 组件设计

### 前端组件

#### 1. AICopilotButton (新)

```tsx
// 替代 AIAssistantButton，单一入口
export function AICopilotButton() {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button icon={<Sparkles />} onClick={() => setOpen(true)}>
        AI Copilot
      </Button>
      <AICopilotDrawer open={open} onClose={() => setOpen(false)} />
    </>
  );
}
```

#### 2. AICopilotDrawer (重构)

```tsx
// 使用 @ant-design/x-sdk
import { useXChat, useXConversations } from '@ant-design/x-sdk';
import { Bubble, Sender, Conversations } from '@ant-design/x';

export function AICopilotDrawer({ open, onClose }) {
  // 场景状态
  const [scene, setScene] = useAutoScene();

  // 使用 x-sdk hooks
  const { conversations, activeKey, setActiveKey } = useXConversations();
  const { messages, onRequest, isLoading } = useXChat({
    provider: new PlatformChatProvider(scene),
  });

  return (
    <Drawer open={open} onClose={onClose}>
      <SceneSelector value={scene} onChange={setScene} />
      <Conversations items={conversations} activeKey={activeKey} />
      <Bubble.List items={messages} />
      <Sender onSend={onRequest} loading={isLoading} />
    </Drawer>
  );
}
```

#### 3. PlatformChatProvider (新)

```tsx
// 自定义 Provider 适配后端 SSE API
import { AbstractChatProvider, XStream } from '@ant-design/x-sdk';

export class PlatformChatProvider extends AbstractChatProvider<Message, Request, SSEOutput> {
  transformParams(params) {
    return {
      sessionId: this.sessionId,
      message: params.message,
      context: { scene: this.scene },
    };
  }

  transformLocalMessage(params) {
    return { id: nanoid(), role: 'user', content: params.message };
  }

  transformMessage({ chunk }) {
    // 解析 SSE 事件: meta, delta, tool_call, tool_result, done, error
    if (chunk.event === 'delta') {
      return { id: chunk.turn_id, content: chunk.data };
    }
    // ...
  }
}
```

### 后端目录变更

#### 删除文件
```
internal/service/ai/approval_notifier.go
internal/service/ai/approval_notifier_test.go
internal/service/ai/permission_checker.go
internal/service/ai/permission_checker_test.go
internal/service/ai/handler/ (空目录)
internal/service/ai/logic/ (空目录)
```

#### 合并文件
```
internal/ai/tools/cicd.go
internal/ai/tools/config.go
internal/ai/tools/deployment.go
internal/ai/tools/governance.go
internal/ai/tools/inventory.go
internal/ai/tools/k8s.go
internal/ai/tools/monitor.go
internal/ai/tools/ops.go
internal/ai/tools/service.go
→ internal/ai/tools/category.go
```

#### 移动文件

| 原路径 | 新路径 |
|--------|--------|
| internal/ai/agent.go | internal/ai/agent/agent.go |
| internal/ai/agent_types.go | internal/ai/agent/types.go |
| internal/ai/hybrid_agent.go | internal/ai/agent/hybrid.go |
| internal/ai/runner.go | internal/ai/agent/runner.go |
| internal/ai/agentic_mode.go | internal/ai/modes/agentic.go |
| internal/ai/simple_chat_mode.go | internal/ai/modes/simple.go |
| internal/ai/store.go | internal/ai/store/store.go |
| internal/ai/checkpoint_store.go | internal/ai/store/checkpoint.go |
| internal/ai/classifier.go | internal/ai/classifier/classifier.go |
| internal/service/ai/chat_handler.go | internal/service/ai/handler/chat.go |
| internal/service/ai/capability_handler.go | internal/service/ai/handler/capability.go |
| internal/service/ai/session_store.go | internal/service/ai/logic/session_store.go |

## API 兼容性

所有 HTTP API 端点保持不变，前端重构不涉及后端接口变更。

## 测试策略

1. 后端：确保现有测试全部通过
2. 前端：为新增组件编写单元测试
3. 集成：E2E 测试验证 AI Copilot 完整流程
