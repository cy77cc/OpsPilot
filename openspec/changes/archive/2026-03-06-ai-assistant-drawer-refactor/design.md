# Design: AI 助手抽屉式重构

## 组件架构

### 文件结构

```
web/src/components/AI/
├── index.ts                    # 导出入口
├── AIAssistantDrawer.tsx       # 主抽屉组件
├── AIAssistantButton.tsx       # Header 触发按钮
├── SceneDetector.tsx           # 场景检测 Hook
│
├── components/
│   ├── ConversationsPanel.tsx  # 会话列表面板
│   ├── MessageList.tsx         # 消息列表 (Bubble.List)
│   ├── MessageBubble.tsx       # 消息气泡 (带 XMarkdown)
│   ├── ToolCard.tsx            # 简化版工具卡片
│   ├── ConfirmationPanel.tsx   # 审批确认面板
│   └── ChatInput.tsx           # 输入框 (Sender)
│
├── hooks/
│   ├── useAIChat.ts            # useXChat 封装
│   ├── useSSEAdapter.ts        # SSE → useXChat 适配器
│   └── useResizableDrawer.ts   # 可变宽度 Drawer
│
├── providers/
│   └── AIChatProvider.tsx      # Context Provider
│
├── constants/
│   └── sceneMapping.ts         # 路由→场景映射
│
└── types.ts                    # 类型定义
```

## 核心设计

### 1. AIAssistantDrawer

主抽屉组件，负责渲染整个 AI 助手界面。

```tsx
interface AIAssistantDrawerProps {
  open: boolean;
  onClose: () => void;
  scene: string;  // 'global' | 'deployment:hosts' | ...
}

function AIAssistantDrawer({ open, onClose, scene }: AIAssistantDrawerProps) {
  const [width, setWidth] = useState(520);
  const { messages, isLoading, sendMessage, ... } = useAIChat({ scene });

  return (
    <Drawer
      open={open}
      onClose={onClose}
      width={width}
      placement="right"
      closable={false}
      styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column' } }}
    >
      {/* 拖拽调整宽度 */}
      <ResizeHandle onResize={setWidth} />

      {/* 会话列表 */}
      <ConversationsPanel scene={scene} />

      {/* 消息列表 */}
      <MessageList messages={messages} isLoading={isLoading} />

      {/* 输入框 */}
      <ChatInput onSend={sendMessage} isLoading={isLoading} />
    </Drawer>
  );
}
```

### 2. AIAssistantButton

Header 中的触发按钮，支持全局和场景两种模式。

```tsx
function AIAssistantButton() {
  const { scene, hasSceneSupport } = useSceneDetector();
  const [globalOpen, setGlobalOpen] = useState(false);
  const [sceneOpen, setSceneOpen] = useState(false);

  return (
    <>
      {/* 全局助手按钮 */}
      <Tooltip title="AI 助手 (Cmd+/)">
        <Button icon={<RobotOutlined />} onClick={() => setGlobalOpen(true)}>
          AI 助手
        </Button>
      </Tooltip>

      {/* 场景助手按钮 (条件渲染) */}
      {hasSceneSupport && (
        <Tooltip title={`${scene} 助手 (Cmd+Shift+/)`}>
          <Button type="primary" onClick={() => setSceneOpen(true)}>
            {sceneLabel(scene)}
          </Button>
        </Tooltip>
      )}

      <AIAssistantDrawer open={globalOpen} onClose={() => setGlobalOpen(false)} scene="global" />
      <AIAssistantDrawer open={sceneOpen} onClose={() => setSceneOpen(false)} scene={scene} />
    </>
  );
}
```

### 3. useSSEAdapter

将后端 SSE 接口适配到 Ant Design X 的 useXChat。

```tsx
interface UseSSEAdapterOptions {
  scene: string;
  sessionId?: string;
}

function useSSEAdapter(options: UseSSEAdapterOptions) {
  const { scene, sessionId } = options;

  // 使用 XRequest 处理 SSE
  const request = useCallback(async (message: string) => {
    const response = await fetch('/api/v1/ai/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message, sessionId, context: { scene } }),
    });

    return {
      // 返回 SSE 流适配器
      [Symbol.asyncIterator]: async function* () {
        const reader = response.body!.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const events = parseSSEEvents(buffer);

          for (const event of events) {
            yield transformToXChatMessage(event);
          }
        }
      },
    };
  }, [scene, sessionId]);

  return { request };
}
```

### 4. MessageList

使用 Ant Design X 的 Bubble.List 渲染消息，支持 XMarkdown。

```tsx
function MessageList({ messages, isLoading }: MessageListProps) {
  const roles = {
    user: { placement: 'end' as const },
    assistant: { placement: 'start' as const },
  };

  const items = messages.map(msg => ({
    key: msg.id,
    role: msg.role,
    content: (
      <>
        {/* 工具执行卡片 */}
        {msg.tools?.map(tool => (
          <ToolCard key={tool.id} tool={tool} />
        ))}

        {/* 消息内容 (XMarkdown) */}
        <Bubble content={msg.content} />

        {/* 确认面板 */}
        {msg.confirmation && (
          <ConfirmationPanel confirmation={msg.confirmation} />
        )}
      </>
    ),
  }));

  return (
    <Bubble.List
      items={items}
      roles={roles}
      style={{ flex: 1, overflow: 'auto' }}
    />
  );
}
```

### 5. ToolCard

简化版工具执行卡片，显示工具名、状态、耗时。

```tsx
interface ToolCardProps {
  tool: {
    name: string;
    status: 'running' | 'success' | 'error';
    duration?: number;
  };
}

function ToolCard({ tool }: ToolCardProps) {
  const statusIcon = {
    running: <LoadingOutlined spin />,
    success: <CheckCircleOutlined style={{ color: '#52c41a' }} />,
    error: <CloseCircleOutlined style={{ color: '#ff4d4f' }} />,
  };

  return (
    <div className="tool-card">
      <span className="tool-icon">🔧</span>
      <span className="tool-name">{tool.name}</span>
      <span className="tool-status">{statusIcon[tool.status]}</span>
      {tool.duration && (
        <span className="tool-duration">{tool.duration}s</span>
      )}
    </div>
  );
}
```

### 6. 场景检测

根据路由自动检测当前场景。

```tsx
const SCENE_MAPPING: Record<string, string> = {
  '/deployment/infrastructure/clusters': 'deployment:clusters',
  '/deployment/infrastructure/hosts': 'deployment:hosts',
  '/deployment/targets': 'deployment:targets',
  '/deployment/overview': 'deployment:releases',
  '/deployment/approvals': 'deployment:approvals',
  '/services': 'services:list',
  '/services/deploy': 'services:deploy',
  '/governance/users': 'governance:users',
  '/monitor': 'deployment:metrics',
};

function useSceneDetector() {
  const location = useLocation();

  const scene = useMemo(() => {
    for (const [prefix, sceneKey] of Object.entries(SCENE_MAPPING)) {
      if (location.pathname.startsWith(prefix)) {
        return sceneKey;
      }
    }
    return null;
  }, [location.pathname]);

  return {
    scene,
    hasSceneSupport: scene !== null,
  };
}
```

## 错误处理

### 错误类型映射

```tsx
const ERROR_MESSAGES: Record<string, string> = {
  network: '网络连接失败，请检查网络后重试',
  timeout: '请求超时，点击重试',
  auth: '登录已过期，请重新登录',
  tool: '工具执行失败',
  unknown: '发生未知错误，请稍后重试',
};

function handleSSEError(error: SSEError) {
  const errorType = classifyError(error);
  const message = ERROR_MESSAGES[errorType] || ERROR_MESSAGES.unknown;

  message.error({
    content: message,
    duration: 3,
  });
}
```

## 快捷键

```tsx
// 在 AppLayout 中注册快捷键
useEffect(() => {
  const handler = (e: KeyboardEvent) => {
    // Cmd/Ctrl + / : 全局助手
    if (e.key === '/' && (e.metaKey || e.ctrlKey) && !e.shiftKey) {
      e.preventDefault();
      setGlobalAIOpen(true);
    }
    // Cmd/Ctrl + Shift + / : 场景助手
    if (e.key === '/' && (e.metaKey || e.ctrlKey) && e.shiftKey) {
      e.preventDefault();
      setSceneAIOpen(true);
    }
    // Escape : 关闭 Drawer
    if (e.key === 'Escape') {
      setGlobalAIOpen(false);
      setSceneAIOpen(false);
    }
  };

  document.addEventListener('keydown', handler);
  return () => document.removeEventListener('keydown', handler);
}, []);
```

## 可变宽度 Drawer

```tsx
function useResizableDrawer(defaultWidth = 520, minWidth = 480, maxWidth = 800) {
  const [width, setWidth] = useState(defaultWidth);
  const [isResizing, setIsResizing] = useState(false);

  const handleMouseDown = useCallback(() => {
    setIsResizing(true);
  }, []);

  const handleMouseMove = useCallback((e: MouseEvent) => {
    if (!isResizing) return;
    const newWidth = window.innerWidth - e.clientX;
    setWidth(clamp(newWidth, minWidth, maxWidth));
  }, [isResizing, minWidth, maxWidth]);

  const handleMouseUp = useCallback(() => {
    setIsResizing(false);
  }, []);

  useEffect(() => {
    if (isResizing) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
    }
    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isResizing, handleMouseMove, handleMouseUp]);

  return { width, handleMouseDown, isResizing };
}
```

## 删除范围

重构完成后删除以下文件:

```
web/src/pages/AIChat/
├── ChatPage.tsx
├── components/
│   ├── ChatMain.tsx
│   └── ConversationSidebar.tsx
├── hooks/
│   ├── useSSEConnection.ts
│   ├── useChatSession.ts
│   ├── useConfirmation.ts
│   └── useAIChatShortcuts.ts
├── types.ts
├── index.ts
└── ai-chat.css

web/src/pages/AI/ (如果存在)
└── AICommandCenterPage.tsx
```

同时更新 `App.tsx` 移除 `/ai` 路由。
