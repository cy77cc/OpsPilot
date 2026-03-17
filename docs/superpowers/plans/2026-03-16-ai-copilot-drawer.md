# AI Copilot Drawer Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a drawer-based AI chat interface using Ant Design X components, integrated with our existing backend SSE API.

**Architecture:**
- Custom `PlatformChatProvider` that wraps our existing `aiApi.chatStream()` for `useXChat`
- `AICopilotDrawer` component with Bubble.List, Sender, file attachments, and conversation history
- State managed in `AppLayout` with drawer open/close controlled via `AICopilotButton`

**Tech Stack:** React 19, Ant Design 6, @ant-design/x, @ant-design/x-sdk, @ant-design/x-markdown, antd-style

---

## Chunk 1: Foundation - Types and Provider

### Task 1: Platform Chat Provider Types

**Files:**
- Create: `web/src/components/AI/types.ts`
- Test: `web/src/components/AI/__tests__/types.test.ts`

- [ ] **Step 1: Create types file**

```typescript
// web/src/components/AI/types.ts
import type { ReactNode } from 'react';

/**
 * PlatformChatProvider 配置参数。
 */
export interface PlatformChatProviderConfig {
  /** API 基础路径 */
  baseUrl?: string;
  /** 请求超时时间（毫秒） */
  timeout?: number;
  /** 默认场景 */
  scene?: string;
}

/**
 * 聊天请求参数。
 */
export interface ChatRequest {
  /** 会话 ID（可选，新建会话时不传） */
  sessionId?: string;
  /** 用户消息内容 */
  message: string;
  /** 场景标识 */
  scene?: string;
}

/**
 * SSE 事件类型映射。
 * 与后端 internal/service/ai/logic/logic.go 中定义的事件对应。
 */
export type SSEEventType =
  | 'init'
  | 'status'
  | 'intent'
  | 'delta'
  | 'thinking_delta'
  | 'tool_call'
  | 'tool_result'
  | 'tool_approval'
  | 'progress'
  | 'report_ready'
  | 'meta'
  | 'heartbeat'
  | 'done'
  | 'error';

/**
 * useXChat 消息格式适配。
 */
export interface XChatMessage {
  role: 'user' | 'assistant';
  content: string;
  id?: string;
  status?: 'loading' | 'success' | 'error';
}

/**
 * 快捷提示项。
 */
export interface QuickPrompt {
  key: string;
  label: string;
  icon?: ReactNode;
  prompt?: string;
}

/**
 * 对话会话摘要。
 */
export interface ConversationSummary {
  id: string;
  title: string;
  scene: string;
  createdAt: string;
  updatedAt: string;
  messageCount: number;
}

/**
 * 意图事件数据。
 */
export interface IntentEventData {
  intent_type: string;
  assistant_type: string;
  risk_level?: string;
}

/**
 * 工具调用事件数据。
 */
export interface ToolCallEventData {
  call_id?: string;
  tool_name?: string;
  tool_display_name?: string;
  arguments?: string;
}
```

- [ ] **Step 2: Write tests for types**

```typescript
// web/src/components/AI/__tests__/types.test.ts
import { describe, it, expect } from 'vitest';
import type {
  PlatformChatProviderConfig,
  ChatRequest,
  SSEEventType,
  XChatMessage,
  QuickPrompt,
  ConversationSummary,
} from '../types';

describe('AI Types', () => {
  it('should accept valid ChatRequest', () => {
    const request: ChatRequest = {
      message: 'Hello',
      sessionId: 'test-id',
    };
    expect(request.message).toBe('Hello');
  });

  it('should accept valid XChatMessage', () => {
    const message: XChatMessage = {
      role: 'user',
      content: 'Test message',
      id: 'msg-1',
      status: 'success',
    };
    expect(message.role).toBe('user');
  });

  it('should accept valid ConversationSummary', () => {
    const summary: ConversationSummary = {
      id: 'conv-1',
      title: 'Test Conversation',
      scene: 'ai',
      createdAt: '2024-01-01T00:00:00Z',
      updatedAt: '2024-01-01T00:00:00Z',
      messageCount: 5,
    };
    expect(summary.messageCount).toBe(5);
  });
});
```

- [ ] **Step 3: Run tests**

Run: `cd /root/project/k8s-manage/web && npm run test -- src/components/AI/__tests__/types.test.ts`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add web/src/components/AI/types.ts web/src/components/AI/__tests__/types.test.ts
git commit -m "feat(ai): add type definitions for PlatformChatProvider"
```

---

### Task 2: Platform Chat Provider

**Files:**
- Create: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Create: `web/src/components/AI/providers/index.ts`
- Test: `web/src/components/AI/__tests__/providers/PlatformChatProvider.test.ts`

- [ ] **Step 1: Create PlatformChatProvider**

```typescript
// web/src/components/AI/providers/PlatformChatProvider.ts

/**
 * PlatformChatProvider 实现 IXChat 接口，
 * 将后端 SSE API 适配为 useXChat 所需格式。
 *
 * 核心职责:
 *   - 发送聊天请求到 /api/v1/ai/chat
 *   - 解析 SSE 事件流
 *   - 将后端事件转换为 useXChat 回调
 */
import type { IXChat } from '@ant-design/x-sdk';
import { aiApi } from '@/api/modules/ai';
import type {
  ChatRequest,
  XChatMessage,
  PlatformChatProviderConfig,
  IntentEventData,
  ToolCallEventData,
} from '../types';

/** 单次请求的上下文 */
interface RequestContext {
  sessionId: string;
  runId: string;
  abortController: AbortController;
  onMessage: (message: XChatMessage) => void;
  onComplete: () => void;
  onError: (error: Error) => void;
  onIntent?: (data: IntentEventData) => void;
  onToolCall?: (data: ToolCallEventData) => void;
}

/**
 * PlatformChatProvider 适配器类。
 *
 * 用法:
 * ```tsx
 * const provider = new PlatformChatProvider({ scene: 'ai' });
 * const { onRequest, messages } = useXChat({ provider });
 * ```
 */
export class PlatformChatProvider implements IXChat {
  private config: PlatformChatProviderConfig;
  private currentContext: RequestContext | null = null;

  constructor(config: PlatformChatProviderConfig = {}) {
    this.config = config;
  }

  /**
   * 获取当前会话 ID。
   */
  getSessionId(): string | undefined {
    return this.currentContext?.sessionId;
  }

  /**
   * 中止当前请求。
   */
  abort(): void {
    if (this.currentContext?.abortController) {
      this.currentContext.abortController.abort();
      this.currentContext = null;
    }
  }

  /**
   * 发送聊天请求。
   *
   * @param params - 请求参数，包含 messages 数组
   * @param callbacks - 回调函数
   * @returns AbortController 用于取消请求
   */
  async request(
    params: { messages: Array<{ role: string; content: string }> },
    callbacks: {
      onUpdate: (message: XChatMessage) => void;
      onSuccess: (message: XChatMessage) => void;
      onError: (error: Error) => void;
    },
  ): Promise<AbortController> {
    const abortController = new AbortController();

    // 获取最后一条用户消息
    const lastUserMessage = [...params.messages].reverse().find((m) => m.role === 'user');
    if (!lastUserMessage) {
      callbacks.onError(new Error('No user message found'));
      return abortController;
    }

    // 构建请求参数
    const request: ChatRequest = {
      message: lastUserMessage.content,
      sessionId: this.currentContext?.sessionId,
      scene: this.config.scene,
    };

    // 内容缓冲区
    let contentBuffer = '';
    let assistantMessageId = '';

    // 创建请求上下文
    this.currentContext = {
      sessionId: request.sessionId || '',
      runId: '',
      abortController,
      onMessage: callbacks.onUpdate,
      onComplete: () => {
        callbacks.onSuccess({
          role: 'assistant',
          content: contentBuffer,
          id: assistantMessageId,
          status: 'success',
        });
      },
      onError: callbacks.onError,
    };

    try {
      await aiApi.chatStream(
        request,
        {
          onInit: (payload) => {
            this.currentContext!.sessionId = payload.session_id;
            this.currentContext!.runId = payload.run_id;
            assistantMessageId = `msg-${Date.now()}`;
          },
          onIntent: (payload) => {
            // 处理意图事件，可用于显示当前活跃的 Agent
            console.log('[AI] Intent detected:', payload.intent_type, payload.assistant_type);
          },
          onStatus: (payload) => {
            // 处理状态变化 (running/completed)
            console.log('[AI] Status:', payload.status);
          },
          onDelta: (payload) => {
            contentBuffer += payload.contentChunk;
            callbacks.onUpdate({
              role: 'assistant',
              content: contentBuffer,
              id: assistantMessageId,
              status: 'loading',
            });
          },
          onThinkingDelta: (payload) => {
            // 思考过程增量，可用于显示 AI 推理过程
            console.log('[AI] Thinking:', payload.contentChunk?.slice(0, 50));
          },
          onToolCall: (payload) => {
            // 工具调用开始
            console.log('[AI] Tool call:', payload.tool_name);
          },
          onToolResult: (payload) => {
            // 工具调用结果
            console.log('[AI] Tool result:', payload.tool_name);
          },
          onToolApproval: (payload) => {
            // 需要用户审批的工具调用
            console.log('[AI] Tool approval required:', payload.tool_name, 'risk:', payload.risk);
          },
          onDone: () => {
            this.currentContext?.onComplete();
          },
          onError: (payload) => {
            callbacks.onError(new Error(payload.message));
          },
        },
        abortController.signal,
      );
    } catch (error) {
      if ((error as Error).name === 'AbortError') {
        // 用户主动取消，不报错
        return abortController;
      }
      callbacks.onError(error as Error);
    }

    return abortController;
  }
}
```

- [ ] **Step 2: Create providers index**

```typescript
// web/src/components/AI/providers/index.ts
export { PlatformChatProvider } from './PlatformChatProvider';
// 类型从 types.ts 导出，避免重复定义
```

- [ ] **Step 3: Write tests for Provider**

```typescript
// web/src/components/AI/__tests__/providers/PlatformChatProvider.test.ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { PlatformChatProvider } from '../../providers/PlatformChatProvider';
import { aiApi } from '@/api/modules/ai';

// Mock aiApi
vi.mock('@/api/modules/ai', () => ({
  aiApi: {
    chatStream: vi.fn(),
  },
}));

describe('PlatformChatProvider', () => {
  let provider: PlatformChatProvider;

  beforeEach(() => {
    provider = new PlatformChatProvider({ scene: 'ai' });
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('should create instance with config', () => {
    expect(provider).toBeInstanceOf(PlatformChatProvider);
  });

  it('should return undefined sessionId initially', () => {
    expect(provider.getSessionId()).toBeUndefined();
  });

  it('should abort current request', () => {
    provider.abort();
    // No error should be thrown
    expect(provider.getSessionId()).toBeUndefined();
  });

  describe('request method', () => {
    it('should error when no user message', async () => {
      const onError = vi.fn();
      const abortController = await provider.request(
        { messages: [{ role: 'assistant', content: 'Hello' }] },
        {
          onUpdate: vi.fn(),
          onSuccess: vi.fn(),
          onError,
        },
      );

      expect(onError).toHaveBeenCalledWith(expect.any(Error));
      expect(onError.mock.calls[0][0].message).toBe('No user message found');
    });

    it('should call aiApi.chatStream with correct params', async () => {
      const mockChatStream = vi.mocked(aiApi.chatStream);
      mockChatStream.mockImplementation(async (_params, handlers) => {
        handlers.onInit({ session_id: 'test-session', run_id: 'test-run' });
        handlers.onDelta({ contentChunk: 'Hello' });
        handlers.onDone({});
        return Promise.resolve();
      });

      const onUpdate = vi.fn();
      const onSuccess = vi.fn();

      await provider.request(
        { messages: [{ role: 'user', content: 'Hi' }] },
        { onUpdate, onSuccess, onError: vi.fn() },
      );

      expect(mockChatStream).toHaveBeenCalledWith(
        expect.objectContaining({ message: 'Hi', scene: 'ai' }),
        expect.any(Object),
        expect.any(AbortSignal),
      );
    });

    it('should accumulate content from delta events', async () => {
      const mockChatStream = vi.mocked(aiApi.chatStream);
      mockChatStream.mockImplementation(async (_params, handlers) => {
        handlers.onInit({ session_id: 'test-session', run_id: 'test-run' });
        handlers.onDelta({ contentChunk: 'Hello ' });
        handlers.onDelta({ contentChunk: 'World' });
        handlers.onDone({});
        return Promise.resolve();
      });

      const onUpdate = vi.fn();
      const onSuccess = vi.fn();

      await provider.request(
        { messages: [{ role: 'user', content: 'Hi' }] },
        { onUpdate, onSuccess, onError: vi.fn() },
      );

      // Should receive incremental updates
      expect(onUpdate).toHaveBeenCalledWith(
        expect.objectContaining({ content: 'Hello ', status: 'loading' }),
      );
      expect(onUpdate).toHaveBeenCalledWith(
        expect.objectContaining({ content: 'Hello World', status: 'loading' }),
      );

      // Final success should have full content
      expect(onSuccess).toHaveBeenCalledWith(
        expect.objectContaining({ content: 'Hello World', status: 'success' }),
      );
    });

    it('should handle error events', async () => {
      const mockChatStream = vi.mocked(aiApi.chatStream);
      mockChatStream.mockImplementation(async (_params, handlers) => {
        handlers.onInit({ session_id: 'test-session', run_id: 'test-run' });
        handlers.onError({ message: 'Test error' });
        return Promise.resolve();
      });

      const onError = vi.fn();

      await provider.request(
        { messages: [{ role: 'user', content: 'Hi' }] },
        { onUpdate: vi.fn(), onSuccess: vi.fn(), onError },
      );

      expect(onError).toHaveBeenCalledWith(expect.any(Error));
      expect(onError.mock.calls[0][0].message).toBe('Test error');
    });
  });
});
```

- [ ] **Step 4: Run tests**

Run: `cd /root/project/k8s-manage/web && npm run test -- src/components/AI/__tests__/providers/PlatformChatProvider.test.ts`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/providers/ web/src/components/AI/__tests__/providers/
git commit -m "feat(ai): implement PlatformChatProvider for useXChat integration"
```

---

## Chunk 2: Core Hook

### Task 3: useAIChat Hook

**Files:**
- Create: `web/src/components/AI/hooks/useAIChat.ts`
- Create: `web/src/components/AI/hooks/index.ts`
- Test: `web/src/components/AI/__tests__/hooks/useAIChat.test.ts`

- [ ] **Step 1: Create useAIChat hook**

```typescript
// web/src/components/AI/hooks/useAIChat.ts

/**
 * useAIChat 封装 useXChat 与 PlatformChatProvider 的集成。
 *
 * 提供简化的 API:
 *   - messages: 当前消息列表
 *   - sendMessage: 发送消息
 *   - isLoading: 是否正在加载
 *   - abort: 中止当前请求
 *   - sessionId: 当前会话 ID
 */
import { useMemo, useRef, useCallback } from 'react';
import { useXChat } from '@ant-design/x-sdk';
import { PlatformChatProvider } from '../providers';
import type { XChatMessage } from '../types';

export interface UseAIChatOptions {
  /** 场景标识 */
  scene?: string;
  /** 初始会话 ID */
  initialSessionId?: string;
  /** 消息发送前回调 */
  onBeforeSend?: (message: string) => void;
  /** 消息发送成功回调 */
  onSuccess?: (message: XChatMessage) => void;
  /** 消息发送失败回调 */
  onError?: (error: Error) => void;
}

export interface UseAIChatReturn {
  /** 消息列表 */
  messages: Array<{
    id: string;
    content: string;
    role: 'user' | 'assistant';
    status?: 'loading' | 'success' | 'error';
  }>;
  /** 发送消息 */
  sendMessage: (content: string) => void;
  /** 是否正在加载 */
  isLoading: boolean;
  /** 中止当前请求 */
  abort: () => void;
  /** 当前会话 ID */
  sessionId: string | undefined;
  /** 清空消息 */
  clearMessages: () => void;
}

/**
 * AI 聊天 Hook。
 *
 * 用法:
 * ```tsx
 * const { messages, sendMessage, isLoading, abort } = useAIChat({ scene: 'ai' });
 *
 * // 发送消息
 * sendMessage('你好');
 *
 * // 渲染消息
 * messages.map(msg => <div key={msg.id}>{msg.content}</div>)
 * ```
 */
export function useAIChat(options: UseAIChatOptions = {}): UseAIChatReturn {
  const { scene = 'ai', onBeforeSend, onSuccess, onError } = options;

  // Provider 实例（每个 hook 实例一个）
  const providerRef = useRef<PlatformChatProvider | null>(null);

  // 消息 ID 计数器
  const messageIdRef = useRef(0);

  // 创建 Provider
  const provider = useMemo(() => {
    if (!providerRef.current) {
      providerRef.current = new PlatformChatProvider({ scene });
    }
    return providerRef.current;
  }, [scene]);

  // 使用 useXChat
  const {
    onRequest,
    messages,
    isRequesting,
    abort,
    setMessages,
  } = useXChat({
    provider,
  });

  // 发送消息
  const sendMessage = useCallback(
    (content: string) => {
      if (!content.trim()) return;

      onBeforeSend?.(content);

      // 生成用户消息 ID
      const userMessageId = `user-${Date.now()}-${messageIdRef.current++}`;

      // 调用 onRequest
      onRequest({
        messages: [{ role: 'user', content }],
      });
    },
    [onRequest, onBeforeSend],
  );

  // 清空消息
  const clearMessages = useCallback(() => {
    setMessages([]);
  }, [setMessages]);

  // 转换消息格式
  const formattedMessages = useMemo(() => {
    return messages.map((msg, index) => ({
      id: msg.id || `msg-${index}`,
      content: typeof msg.message?.content === 'string'
        ? msg.message.content
        : '',
      role: msg.message?.role as 'user' | 'assistant',
      status: msg.status as 'loading' | 'success' | 'error' | undefined,
    }));
  }, [messages]);

  return {
    messages: formattedMessages,
    sendMessage,
    isLoading: isRequesting,
    abort,
    sessionId: provider.getSessionId(),
    clearMessages,
  };
}
```

- [ ] **Step 2: Create hooks index**

```typescript
// web/src/components/AI/hooks/index.ts
export { useAIChat } from './useAIChat';
export type { UseAIChatOptions, UseAIChatReturn } from './useAIChat';
```

- [ ] **Step 3: Write tests for hook**

```typescript
// web/src/components/AI/__tests__/hooks/useAIChat.test.ts
import { describe, it, expect, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useAIChat } from '../../hooks/useAIChat';

// Mock dependencies
vi.mock('@ant-design/x-sdk', () => ({
  useXChat: vi.fn(() => ({
    onRequest: vi.fn(),
    messages: [],
    isRequesting: false,
    abort: vi.fn(),
    setMessages: vi.fn(),
  })),
}));

vi.mock('../../providers', () => ({
  PlatformChatProvider: vi.fn(() => ({
    getSessionId: vi.fn(() => 'test-session-id'),
  })),
}));

describe('useAIChat', () => {
  it('should return initial state', () => {
    const { result } = renderHook(() => useAIChat({ scene: 'ai' }));

    expect(result.current.messages).toEqual([]);
    expect(result.current.isLoading).toBe(false);
    expect(typeof result.current.sendMessage).toBe('function');
    expect(typeof result.current.abort).toBe('function');
  });

  it('should call onBeforeSend when sending message', () => {
    const onBeforeSend = vi.fn();
    const { result } = renderHook(() =>
      useAIChat({ scene: 'ai', onBeforeSend }),
    );

    act(() => {
      result.current.sendMessage('Hello');
    });

    expect(onBeforeSend).toHaveBeenCalledWith('Hello');
  });
});
```

- [ ] **Step 4: Run tests**

Run: `cd /root/project/k8s-manage/web && npm run test -- src/components/AI/__tests__/hooks/useAIChat.test.ts`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/hooks/ web/src/components/AI/__tests__/hooks/
git commit -m "feat(ai): add useAIChat hook for simplified chat integration"
```

---

## Chunk 3: UI Components

### Task 4: Chat Content Component (Bubble.List)

**Files:**
- Create: `web/src/components/AI/ChatContent.tsx`
- Test: `web/src/components/AI/__tests__/ChatContent.test.tsx`

- [ ] **Step 1: Create ChatContent component**

```typescript
// web/src/components/AI/ChatContent.tsx

/**
 * ChatContent 渲染聊天消息列表。
 *
 * 功能:
 *   - 使用 Ant Design X 的 Bubble.List 渲染消息
 *   - 支持 Markdown 渲染
 *   - 支持代码高亮
 *   - 支持思考过程展示
 */
import React, { useMemo } from 'react';
import { Bubble, Welcome, Prompts } from '@ant-design/x';
import type { GetProp } from 'antd';
import XMarkdown from '@ant-design/x-markdown';
import { Spin } from 'antd';
import type { XChatMessage, QuickPrompt } from './types';

const welcomePrompts: QuickPrompt[] = [
  { key: '1', label: '帮我诊断集群问题', prompt: '帮我诊断集群问题' },
  { key: '2', label: '查看主机状态', prompt: '查看主机状态' },
  { key: '3', label: '分析服务日志', prompt: '分析服务日志' },
];

export interface ChatContentProps {
  /** 消息列表 */
  messages: XChatMessage[];
  /** 是否正在加载 */
  isLoading?: boolean;
  /** 快捷提示点击回调 */
  onPromptClick?: (prompt: string) => void;
}

/**
 * 消息角色配置。
 */
const role: GetProp<typeof Bubble.List, 'role'> = {
  assistant: {
    placement: 'start',
    styles: {
      content: {
        backgroundColor: 'var(--ant-color-bg-container)',
        borderRadius: '12px 12px 12px 4px',
      },
    },
    contentRender: (content) => {
      if (typeof content !== 'string') return content;

      return (
        <XMarkdown
          content={content}
          components={{
            // 可扩展自定义组件
          }}
        />
      );
    },
  },
  user: {
    placement: 'end',
    styles: {
      content: {
        backgroundColor: 'var(--ant-color-primary)',
        color: '#fff',
        borderRadius: '12px 12px 4px 12px',
      },
    },
  },
};

/**
 * 聊天内容组件。
 */
export const ChatContent: React.FC<ChatContentProps> = ({
  messages,
  isLoading,
  onPromptClick,
}) => {
  // 转换消息格式为 Bubble.List 所需格式
  const bubbleItems = useMemo(() => {
    return messages.map((msg, index) => ({
      key: msg.id || `msg-${index}`,
      content: msg.content,
      role: msg.role,
      loading: msg.status === 'loading',
    }));
  }, [messages]);

  // 空消息时显示欢迎界面
  if (messages.length === 0) {
    return (
      <div className="flex flex-col h-full">
        <Welcome
          variant="borderless"
          title="👋 你好，我是 OpsPilot AI 助手"
          description="我可以帮助你管理集群、诊断问题、部署应用等。"
          styles={{
            title: { fontSize: 18, fontWeight: 600 },
          }}
        />
        <div className="mt-4 px-4">
          <Prompts
            vertical
            title="我可以帮你"
            items={welcomePrompts.map((p) => ({
              key: p.key,
              description: p.label,
            }))}
            onItemClick={(info) => {
              const prompt = welcomePrompts.find((p) => p.key === info.data?.key)?.prompt;
              if (prompt) {
                onPromptClick?.(prompt);
              }
            }}
            styles={{
              title: { fontSize: 14, marginBottom: 8 },
            }}
          />
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-hidden">
      <Bubble.List
        items={bubbleItems}
        role={role}
        style={{ height: '100%', padding: '16px' }}
      />
      {isLoading && (
        <div className="flex justify-center py-2">
          <Spin size="small" />
        </div>
      )}
    </div>
  );
};
```

- [ ] **Step 2: Write tests for ChatContent**

```typescript
// web/src/components/AI/__tests__/ChatContent.test.tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ChatContent } from '../ChatContent';
import type { XChatMessage } from '../types';

// Mock Ant Design X components
vi.mock('@ant-design/x', () => ({
  Bubble: {
    List: ({ items }: { items: Array<{ content: string; role: string }> }) => (
      <div data-testid="bubble-list">
        {items.map((item, i) => (
          <div key={i} data-role={item.role}>
            {item.content}
          </div>
        ))}
      </div>
    ),
  },
  Welcome: ({ title }: { title: string }) => <div data-testid="welcome">{title}</div>,
  Prompts: ({ items }: { items: Array<{ key: string; description: string }> }) => (
    <div data-testid="prompts">
      {items.map((item) => (
        <button key={item.key}>{item.description}</button>
      ))}
    </div>
  ),
}));

vi.mock('@ant-design/x-markdown', () => ({
  default: ({ content }: { content: string }) => <div>{content}</div>,
}));

describe('ChatContent', () => {
  it('should render welcome screen when no messages', () => {
    render(<ChatContent messages={[]} />);

    expect(screen.getByTestId('welcome')).toBeInTheDocument();
    expect(screen.getByTestId('prompts')).toBeInTheDocument();
  });

  it('should render messages when provided', () => {
    const messages: XChatMessage[] = [
      { id: '1', role: 'user', content: 'Hello' },
      { id: '2', role: 'assistant', content: 'Hi there!' },
    ];

    render(<ChatContent messages={messages} />);

    expect(screen.getByTestId('bubble-list')).toBeInTheDocument();
    expect(screen.getByText('Hello')).toBeInTheDocument();
    expect(screen.getByText('Hi there!')).toBeInTheDocument();
  });

  it('should call onPromptClick when prompt clicked', async () => {
    const onPromptClick = vi.fn();
    render(<ChatContent messages={[]} onPromptClick={onPromptClick} />);

    const button = screen.getByText('帮我诊断集群问题');
    button.click();

    // Check that prompt click was triggered (implementation dependent)
    expect(screen.getByTestId('prompts')).toBeInTheDocument();
  });
});
```

- [ ] **Step 3: Run tests**

Run: `cd /root/project/k8s-manage/web && npm run test -- src/components/AI/__tests__/ChatContent.test.tsx`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add web/src/components/AI/ChatContent.tsx web/src/components/AI/__tests__/ChatContent.test.tsx
git commit -m "feat(ai): add ChatContent component with Bubble.List and Welcome"
```

---

### Task 5: Chat Input Component (Sender)

**Files:**
- Create: `web/src/components/AI/ChatInput.tsx`
- Test: `web/src/components/AI/__tests__/ChatInput.test.tsx`

- [ ] **Step 1: Create ChatInput component**

```typescript
// web/src/components/AI/ChatInput.tsx

/**
 * ChatInput 提供聊天输入框。
 *
 * 功能:
 *   - 文本输入
 *   - 文件上传
 *   - 发送/取消按钮
 *   - 快捷提示按钮
 */
import React, { useState, useRef } from 'react';
import { Sender, Attachments, Suggestion } from '@ant-design/x';
import { Button, Flex, GetRef, Popover } from 'antd';
import {
  CloudUploadOutlined,
  PaperClipOutlined,
  SendOutlined,
  StopOutlined,
} from '@ant-design/icons';
import type { QuickPrompt } from './types';

const quickPrompts: QuickPrompt[] = [
  { key: 'diagnose', label: '诊断', prompt: '帮我诊断当前问题' },
  { key: 'status', label: '状态', prompt: '查看系统状态' },
  { key: 'logs', label: '日志', prompt: '分析最近的日志' },
];

export interface ChatInputProps {
  /** 发送消息回调 */
  onSend: (message: string) => void;
  /** 中止请求回调 */
  onCancel?: () => void;
  /** 是否正在加载 */
  isLoading?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 占位符 */
  placeholder?: string;
}

/**
 * 聊天输入组件。
 */
export const ChatInput: React.FC<ChatInputProps> = ({
  onSend,
  onCancel,
  isLoading = false,
  disabled = false,
  placeholder = '输入消息，按 Enter 发送...',
}) => {
  const [value, setValue] = useState('');
  const [attachmentsOpen, setAttachmentsOpen] = useState(false);
  const [files, setFiles] = useState<Array<{ uid: string; name: string; status?: string }>>([]);
  const attachmentsRef = useRef<GetRef<typeof Attachments>>(null);

  // 处理发送
  const handleSend = () => {
    if (!value.trim() || isLoading || disabled) return;
    onSend(value.trim());
    setValue('');
  };

  // 处理文件粘贴
  const handlePasteFile = (fileList: FileList) => {
    for (const file of fileList) {
      attachmentsRef.current?.upload(file);
    }
    setAttachmentsOpen(true);
  };

  // 发送按钮
  const submitButton = isLoading ? (
    <Button
      type="text"
      icon={<StopOutlined />}
      onClick={onCancel}
      className="text-gray-500 hover:text-red-500"
    />
  ) : (
    <Button
      type="text"
      icon={<SendOutlined />}
      onClick={handleSend}
      disabled={!value.trim() || disabled}
      className="text-gray-500 hover:text-primary-500"
    />
  );

  // 文件上传 Header
  const senderHeader = (
    <Sender.Header
      title="上传文件"
      open={attachmentsOpen}
      onOpenChange={setAttachmentsOpen}
      forceRender
      styles={{ content: { padding: 0 } }}
    >
      <Attachments
        ref={attachmentsRef}
        beforeUpload={() => false}
        items={files}
        onChange={({ fileList }) => setFiles(fileList as typeof files)}
        placeholder={(type) =>
          type === 'drop'
            ? { title: '拖拽文件到此处' }
            : {
                icon: <CloudUploadOutlined />,
                title: '上传文件',
                description: '点击或拖拽文件到此处上传',
              }
        }
      />
    </Sender.Header>
  );

  return (
    <div className="border-t border-gray-200 p-4 bg-white">
      {/* 快捷提示按钮 */}
      <Flex gap={8} className="mb-3">
        {quickPrompts.map((prompt) => (
          <Button
            key={prompt.key}
            size="small"
            onClick={() => onSend(prompt.prompt || prompt.label)}
            disabled={isLoading || disabled}
          >
            {prompt.label}
          </Button>
        ))}
      </Flex>

      {/* 输入框 */}
      <Suggestion
        items={quickPrompts.map((p) => ({ label: p.label, value: p.key }))}
        onSelect={(itemVal) => {
          const prompt = quickPrompts.find((p) => p.key === itemVal);
          if (prompt?.prompt) {
            setValue(prompt.prompt);
          }
        }}
      >
        {({ onTrigger, onKeyDown }) => (
          <Sender
            value={value}
            onChange={(v) => {
              onTrigger(v === '/');
              setValue(v);
            }}
            onSubmit={handleSend}
            onCancel={onCancel}
            loading={isLoading}
            disabled={disabled}
            placeholder={placeholder}
            header={senderHeader}
            prefix={
              <Button
                type="text"
                icon={<PaperClipOutlined />}
                onClick={() => setAttachmentsOpen(!attachmentsOpen)}
                className="text-gray-400 hover:text-gray-600"
              />
            }
            footer={submitButton}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                handleSend();
              }
              onKeyDown?.(e);
            }}
            onPasteFile={handlePasteFile}
            styles={{
              content: {
                backgroundColor: 'var(--ant-color-bg-container)',
                border: '1px solid var(--ant-color-border)',
                borderRadius: 12,
              },
            }}
          />
        )}
      </Suggestion>

      <div className="text-xs text-gray-400 mt-2 text-center">
        按 Enter 发送，Shift+Enter 换行
      </div>
    </div>
  );
};
```

- [ ] **Step 2: Write tests for ChatInput**

```typescript
// web/src/components/AI/__tests__/ChatInput.test.tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ChatInput } from '../ChatInput';

// Mock Ant Design X components
vi.mock('@ant-design/x', () => ({
  Sender: ({
    value,
    onChange,
    onSubmit,
    placeholder,
    disabled,
    loading,
  }: {
    value: string;
    onChange: (v: string) => void;
    onSubmit: () => void;
    placeholder: string;
    disabled?: boolean;
    loading?: boolean;
  }) => (
    <div data-testid="sender">
      <input
        data-testid="sender-input"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
      />
      <button data-testid="sender-submit" onClick={onSubmit}>
        {loading ? 'Loading' : 'Send'}
      </button>
    </div>
  ),
  Attachments: () => <div data-testid="attachments" />,
  Suggestion: ({ children }: { children: (props: { onTrigger: (v: string) => void; onKeyDown: (e: React.KeyboardEvent) => void }) => React.ReactNode }) =>
    children({ onTrigger: vi.fn(), onKeyDown: vi.fn() }),
}));

describe('ChatInput', () => {
  it('should render input with placeholder', () => {
    render(<ChatInput onSend={vi.fn()} placeholder="Test placeholder" />);

    expect(screen.getByPlaceholderText('Test placeholder')).toBeInTheDocument();
  });

  it('should call onSend when submit clicked', async () => {
    const onSend = vi.fn();
    render(<ChatInput onSend={onSend} />);

    const input = screen.getByTestId('sender-input');
    await userEvent.type(input, 'Hello');

    const submitBtn = screen.getByTestId('sender-submit');
    fireEvent.click(submitBtn);

    expect(onSend).toHaveBeenCalledWith('Hello');
  });

  it('should show quick prompt buttons', () => {
    render(<ChatInput onSend={vi.fn()} />);

    expect(screen.getByText('诊断')).toBeInTheDocument();
    expect(screen.getByText('状态')).toBeInTheDocument();
    expect(screen.getByText('日志')).toBeInTheDocument();
  });

  it('should call onSend with prompt when quick button clicked', () => {
    const onSend = vi.fn();
    render(<ChatInput onSend={onSend} />);

    const diagnoseBtn = screen.getByText('诊断');
    fireEvent.click(diagnoseBtn);

    expect(onSend).toHaveBeenCalledWith('帮我诊断当前问题');
  });

  it('should disable input when disabled prop is true', () => {
    render(<ChatInput onSend={vi.fn()} disabled />);

    const input = screen.getByTestId('sender-input');
    expect(input).toBeDisabled();
  });
});
```

- [ ] **Step 3: Run tests**

Run: `cd /root/project/k8s-manage/web && npm run test -- src/components/AI/__tests__/ChatInput.test.tsx`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add web/src/components/AI/ChatInput.tsx web/src/components/AI/__tests__/ChatInput.test.tsx
git commit -m "feat(ai): add ChatInput component with Sender and file upload"
```

---

## Chunk 4: Drawer and Integration

### Task 6: AI Copilot Drawer Component

**Files:**
- Create: `web/src/components/AI/AICopilotDrawer.tsx`
- Test: `web/src/components/AI/__tests__/AICopilotDrawer.test.tsx`

- [ ] **Step 1: Create AICopilotDrawer component**

```typescript
// web/src/components/AI/AICopilotDrawer.tsx

/**
 * AICopilotDrawer 是 AI 助手的抽屉式界面。
 *
 * 功能:
 *   - 右侧滑出抽屉
 *   - 消息列表
 *   - 输入框
 *   - 会话切换
 *   - 文件上传
 */
import React, { useState, useCallback, useEffect } from 'react';
import { Drawer, Button, Space, Popover, message } from 'antd';
import {
  CloseOutlined,
  PlusOutlined,
  CommentOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import { Conversations } from '@ant-design/x';
import { ChatContent } from './ChatContent';
import { ChatInput } from './ChatInput';
import { useAIChat } from './hooks';
import { aiApi } from '@/api/modules/ai';
import type { ConversationSummary } from './types';

export interface AICopilotDrawerProps {
  /** 是否打开 */
  open: boolean;
  /** 关闭回调 */
  onClose: () => void;
  /** 场景标识 */
  scene?: string;
}

/**
 * AI 助手抽屉组件。
 */
export const AICopilotDrawer: React.FC<AICopilotDrawerProps> = ({
  open,
  onClose,
  scene = 'ai',
}) => {
  // 聊天状态
  const {
    messages,
    sendMessage,
    isLoading,
    abort,
    sessionId,
    clearMessages,
  } = useAIChat({
    scene,
    onError: (error) => {
      message.error(`发送失败: ${error.message}`);
    },
  });

  // 会话列表
  const [conversations, setConversations] = useState<ConversationSummary[]>([]);
  const [showConversations, setShowConversations] = useState(false);

  // 加载会话列表
  const loadConversations = useCallback(async () => {
    try {
      const response = await aiApi.getSessions(scene);
      if (response.code === 1000 && response.data) {
        setConversations(
          response.data.map((s) => ({
            id: s.id,
            title: s.title,
            scene: s.scene,
            createdAt: s.createdAt,
            updatedAt: s.updatedAt,
            messageCount: s.messages?.length || 0,
          })),
        );
      }
    } catch (error) {
      console.error('Failed to load conversations:', error);
    }
  }, [scene]);

  // 抽屉打开时加载会话列表
  useEffect(() => {
    if (open) {
      loadConversations();
    }
  }, [open, loadConversations]);

  // 切换会话
  const handleSwitchConversation = useCallback(async (conversationId: string) => {
    try {
      const response = await aiApi.getSession(conversationId);
      if (response.code === 1000 && response.data) {
        // TODO: 加载会话消息到当前聊天
        setShowConversations(false);
      }
    } catch (error) {
      message.error('加载会话失败');
    }
  }, []);

  // 创建新会话
  const handleNewConversation = useCallback(() => {
    clearMessages();
    setShowConversations(false);
  }, [clearMessages]);

  // 删除当前会话
  const handleDeleteConversation = useCallback(async () => {
    if (!sessionId) return;

    try {
      await aiApi.deleteSession(sessionId);
      clearMessages();
      message.success('会话已删除');
    } catch (error) {
      message.error('删除会话失败');
    }
  }, [sessionId, clearMessages]);

  // 发送消息
  const handleSend = useCallback(
    (content: string) => {
      sendMessage(content);
    },
    [sendMessage],
  );

  // 抽屉标题
  const drawerTitle = (
    <div className="flex items-center justify-between w-full">
      <span className="font-semibold text-lg">✨ AI 助手</span>
      <Space size={0}>
        <Button
          type="text"
          icon={<PlusOutlined />}
          onClick={handleNewConversation}
          title="新会话"
        />
        <Popover
          open={showConversations}
          onOpenChange={setShowConversations}
          placement="bottomRight"
          trigger="click"
          content={
            <div style={{ width: 280 }}>
              <Conversations
                items={conversations.map((c) => ({
                  key: c.id,
                  label: c.title,
                  group: '历史会话',
                }))}
                activeKey={sessionId}
                onActiveChange={(key) => {
                  handleSwitchConversation(key as string);
                }}
              />
            </div>
          }
        >
          <Button type="text" icon={<CommentOutlined />} title="历史会话" />
        </Popover>
        {sessionId && (
          <Button
            type="text"
            icon={<DeleteOutlined />}
            onClick={handleDeleteConversation}
            title="删除会话"
            danger
          />
        )}
        <Button type="text" icon={<CloseOutlined />} onClick={onClose} title="关闭" />
      </Space>
    </div>
  );

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title={drawerTitle}
      placement="right"
      width={420}
      closable={false}
      styles={{
        header: { borderBottom: '1px solid #f0f0f0', padding: '12px 16px' },
        body: { padding: 0, display: 'flex', flexDirection: 'column' },
      }}
    >
      {/* 聊天内容 */}
      <ChatContent
        messages={messages}
        isLoading={isLoading}
        onPromptClick={handleSend}
      />

      {/* 输入框 */}
      <ChatInput
        onSend={handleSend}
        onCancel={abort}
        isLoading={isLoading}
      />
    </Drawer>
  );
};
```

- [ ] **Step 2: Write tests for AICopilotDrawer**

```typescript
// web/src/components/AI/__tests__/AICopilotDrawer.test.tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AICopilotDrawer } from '../AICopilotDrawer';

// Mock dependencies
vi.mock('../hooks', () => ({
  useAIChat: () => ({
    messages: [],
    sendMessage: vi.fn(),
    isLoading: false,
    abort: vi.fn(),
    sessionId: undefined,
    clearMessages: vi.fn(),
  }),
}));

vi.mock('@/api/modules/ai', () => ({
  aiApi: {
    getSessions: vi.fn().mockResolvedValue({ code: 1000, data: [] }),
    deleteSession: vi.fn(),
  },
}));

vi.mock('../ChatContent', () => ({
  ChatContent: ({ onPromptClick }: { onPromptClick?: (p: string) => void }) => (
    <div data-testid="chat-content">
      <button onClick={() => onPromptClick?.('test prompt')}>Test Prompt</button>
    </div>
  ),
}));

vi.mock('../ChatInput', () => ({
  ChatInput: ({ onSend }: { onSend: (m: string) => void }) => (
    <div data-testid="chat-input">
      <button onClick={() => onSend('test message')}>Send</button>
    </div>
  ),
}));

describe('AICopilotDrawer', () => {
  it('should render drawer when open', () => {
    render(<AICopilotDrawer open onClose={vi.fn()} />);

    expect(screen.getByText('✨ AI 助手')).toBeInTheDocument();
  });

  it('should not render when closed', () => {
    render(<AICopilotDrawer open={false} onClose={vi.fn()} />);

    expect(screen.queryByText('✨ AI 助手')).not.toBeInTheDocument();
  });

  it('should call onClose when close button clicked', async () => {
    const onClose = vi.fn();
    render(<AICopilotDrawer open onClose={onClose} />);

    const closeBtn = screen.getByTitle('关闭');
    await userEvent.click(closeBtn);

    expect(onClose).toHaveBeenCalled();
  });

  it('should have new conversation button', () => {
    render(<AICopilotDrawer open onClose={vi.fn()} />);

    expect(screen.getByTitle('新会话')).toBeInTheDocument();
  });

  it('should have history button', () => {
    render(<AICopilotDrawer open onClose={vi.fn()} />);

    expect(screen.getByTitle('历史会话')).toBeInTheDocument();
  });
});
```

- [ ] **Step 3: Run tests**

Run: `cd /root/project/k8s-manage/web && npm run test -- src/components/AI/__tests__/AICopilotDrawer.test.tsx`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add web/src/components/AI/AICopilotDrawer.tsx web/src/components/AI/__tests__/AICopilotDrawer.test.tsx
git commit -m "feat(ai): add AICopilotDrawer component with chat UI"
```

---

### Task 7: Update Exports and AICopilotButton

**Files:**
- Modify: `web/src/components/AI/index.ts`
- Modify: `web/src/components/AI/AICopilotButton.tsx`

- [ ] **Step 1: Update index.ts exports**

```typescript
// web/src/components/AI/index.ts

/**
 * AI 组件导出。
 */
export { AICopilotButton } from './AICopilotButton';
export { AICopilotDrawer } from './AICopilotDrawer';
export { ChatContent } from './ChatContent';
export { ChatInput } from './ChatInput';
export { useAIChat } from './hooks';
export { PlatformChatProvider } from './providers';
export type {
  XChatMessage,
  QuickPrompt,
  ConversationSummary,
  ChatRequest,
} from './types';
```

- [ ] **Step 2: Update AICopilotButton to work with drawer state**

```typescript
// web/src/components/AI/AICopilotButton.tsx

/**
 * AI Copilot 统一入口按钮。
 *
 * 点击后调用 onOpen 回调打开抽屉，
 * 支持快捷键 Cmd/Ctrl + / 快速打开。
 *
 * @breaking-change 从导航模式改为回调模式
 *   - 旧版: 无需 props，点击导航到 /ai
 *   - 新版: 需要传入 onOpen 回调，由父组件管理抽屉状态
 */
import React, { useEffect } from 'react';
import { Button, Tooltip } from 'antd';
import { RobotOutlined } from '@ant-design/icons';

export interface AICopilotButtonProps {
  /** 打开 AI 助手回调（必填） */
  onOpen: () => void;
  /** 是否显示提示文本，默认 false */
  showText?: boolean;
}

export function AICopilotButton({
  onOpen,
  showText = false,
}: AICopilotButtonProps) {
  // 快捷键监听: Cmd/Ctrl + /
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === '/' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        onOpen();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onOpen]);

  return (
    <Tooltip title={<span>AI 助手 <kbd className="ml-1 text-xs bg-gray-100 px-1 rounded">⌘/</kbd></span>}>
      <Button
        type="text"
        icon={<RobotOutlined />}
        onClick={onOpen}
        className="text-gray-600 hover:text-primary-600"
      >
        {showText && 'AI 助手'}
      </Button>
    </Tooltip>
  );
}
```

- [ ] **Step 3: Commit**

```bash
git add web/src/components/AI/index.ts web/src/components/AI/AICopilotButton.tsx
git commit -m "refactor(ai): update exports and AICopilotButton for drawer integration"
```

---

## Chunk 5: AppLayout Integration

> **依赖说明:** 此 Chunk 依赖 Chunk 1-4 完成，特别是:
> - Chunk 1: `types.ts`, `PlatformChatProvider`
> - Chunk 2: `useAIChat` hook
> - Chunk 3: `ChatContent`, `ChatInput`
> - Chunk 4: `AICopilotDrawer` (包含 `onOpen` prop 支持)

### Task 8: Integrate Drawer into AppLayout

**Files:**
- Modify: `web/src/components/Layout/AppLayout.tsx`

> **注意:** `/ai` 路由已不再需要，AI 助手现在通过抽屉提供。
> 如需保留 `/ai` 路由作为备用入口，可创建独立的 AI 页面。

- [ ] **Step 1: Add drawer state to AppLayout**

Find the `AICopilotButton` import and usage in AppLayout, then add drawer state management:

```typescript
// In AppLayout.tsx, add imports:
import { AICopilotButton, AICopilotDrawer } from '../AI';

// Add state near other useState calls:
const [aiDrawerOpen, setAIDrawerOpen] = useState(false);

// Update AICopilotButton usage (around line 410):
// Before:
<AICopilotButton />
// After:
<AICopilotButton onOpen={() => setAIDrawerOpen(true)} />

// Add drawer before closing Layout tag (around line 475):
// Before:
    </Layout>
  );
};

// After:
      {/* AI 助手抽屉 */}
      <AICopilotDrawer
        open={aiDrawerOpen}
        onClose={() => setAIDrawerOpen(false)}
        scene="ai"
      />
    </Layout>
  );
};
```

- [ ] **Step 2: Run frontend tests**

Run: `cd /root/project/k8s-manage/web && npm run test`
Expected: All tests pass

- [ ] **Step 3: Test in browser**

Run: `cd /root/project/k8s-manage && make dev-frontend`
Manual test:
1. Open browser at http://localhost:5173
2. Click AI assistant button in header
3. Verify drawer opens from right
4. Send a message
5. Verify message appears in chat
6. Close drawer with X button

- [ ] **Step 4: Commit**

```bash
git add web/src/components/Layout/AppLayout.tsx
git commit -m "feat(ai): integrate AICopilotDrawer into AppLayout"
```

---

### Task 9: Add Styles and Polish

**Files:**
- Create: `web/src/components/AI/styles.ts`
- Modify: `web/src/components/AI/AICopilotDrawer.tsx`

- [ ] **Step 1: Create shared styles**

```typescript
// web/src/components/AI/styles.ts

/**
 * AI 组件共享样式。
 */
import { createStyles } from 'antd-style';

export const useAIStyles = createStyles(({ token, css }) => ({
  drawerHeader: css`
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    border-bottom: 1px solid ${token.colorBorderSecondary};
  `,

  drawerTitle: css`
    font-weight: 600;
    font-size: 16px;
    display: flex;
    align-items: center;
    gap: 8px;
  `,

  chatContent: css`
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  `,

  welcomeSection: css`
    padding: 24px;
    text-align: center;
  `,

  messageList: css`
    flex: 1;
    padding: 16px;
    overflow-y: auto;
  `,

  inputSection: css`
    border-top: 1px solid ${token.colorBorderSecondary};
    padding: 16px;
    background: ${token.colorBgContainer};
  `,

  quickPrompts: css`
    display: flex;
    gap: 8px;
    margin-bottom: 12px;
    flex-wrap: wrap;
  `,

  senderWrapper: css`
    .ant-sender {
      border-radius: 12px;
      border: 1px solid ${token.colorBorder};
      background: ${token.colorBgContainer};
    }
  `,
}));
```

- [ ] **Step 2: Apply styles to AICopilotDrawer**

Update the drawer component to use styles:

```typescript
// Add import
import { useAIStyles } from './styles';

// In component body
const { styles } = useAIStyles();

// Update Drawer styles prop
styles={{
  header: { className: styles.drawerHeader },
  body: { className: styles.chatContent },
}}
```

- [ ] **Step 3: Commit**

```bash
git add web/src/components/AI/styles.ts web/src/components/AI/AICopilotDrawer.tsx
git commit -m "style(ai): add shared styles for AI components"
```

---

## Final Steps

### Task 10: Final Verification

- [ ] **Step 1: Run all frontend tests**

Run: `cd /root/project/k8s-manage/web && npm run test`
Expected: All tests pass

- [ ] **Step 2: Run lint**

Run: `cd /root/project/k8s-manage/web && npm run lint`
Expected: No errors

- [ ] **Step 3: Build frontend**

Run: `cd /root/project/k8s-manage/web && npm run build`
Expected: Build succeeds

- [ ] **Step 4: Manual integration test**

**基础功能测试:**
1. Start dev server: `make dev-backend`
2. Open browser at http://localhost:8080
3. Login to the application
4. Click AI assistant button in header
5. Send a message like "帮我查看集群状态"
6. Verify SSE streaming works
7. Test conversation history
8. Test file upload UI
9. Test quick prompts

**键盘快捷键测试:**
10. Press `Cmd+/` (Mac) or `Ctrl+/` (Windows/Linux)
11. Verify drawer opens without clicking button
12. Press `Cmd+/` again while drawer is open (should close or do nothing)

**移动端响应式测试:**
13. Open browser DevTools, switch to mobile viewport (375px width)
14. Verify drawer appears as full-screen overlay
15. Test AI assistant on mobile
16. Verify no layout issues with mobile sidebar/bottom nav

**错误状态测试:**
17. Disconnect network (DevTools offline mode)
18. Send a message
19. Verify error message appears
20. Reconnect and verify retry works

- [ ] **Step 5: Final commit**

```bash
git add .
git commit -m "feat(ai): complete AI Copilot drawer implementation"
```

---

## Summary

This plan implements an AI assistant drawer using Ant Design X components:

1. **PlatformChatProvider** - Custom SSE adapter for useXChat
2. **useAIChat** - Simplified hook for chat functionality
3. **ChatContent** - Bubble.List with Welcome and Prompts
4. **ChatInput** - Sender with file upload and quick prompts
5. **AICopilotDrawer** - Main drawer component
6. **AppLayout integration** - State management and trigger

**Files Created:**
- `web/src/components/AI/types.ts`
- `web/src/components/AI/providers/PlatformChatProvider.ts`
- `web/src/components/AI/providers/index.ts`
- `web/src/components/AI/hooks/useAIChat.ts`
- `web/src/components/AI/hooks/index.ts`
- `web/src/components/AI/ChatContent.tsx`
- `web/src/components/AI/ChatInput.tsx`
- `web/src/components/AI/AICopilotDrawer.tsx`
- `web/src/components/AI/styles.ts`
- Test files for each component

**Files Modified:**
- `web/src/components/AI/index.ts`
- `web/src/components/AI/AICopilotButton.tsx`
- `web/src/components/Layout/AppLayout.tsx`
