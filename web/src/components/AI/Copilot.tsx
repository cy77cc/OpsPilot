/**
 * Copilot 组件
 * 使用 @ant-design/x 组件实现的 AI 助手
 */
import React, { useState, useRef, useCallback, useMemo } from 'react';
import {
  CloseOutlined,
  CommentOutlined,
  GlobalOutlined,
  EnvironmentOutlined,
  PlusOutlined,
  CopyOutlined,
  LikeOutlined,
  DislikeOutlined,
  BulbOutlined,
} from '@ant-design/icons';
import {
  Bubble,
  Conversations,
  Prompts,
  Sender,
  Welcome,
  Think,
} from '@ant-design/x';
import type { BubbleListRef, BubbleProps } from '@ant-design/x/es/bubble';
import XMarkdown from '@ant-design/x-markdown';
import { Button, message, Popover, Select, Space, Tooltip, theme, Collapse } from 'antd';
import dayjs from 'dayjs';
import { getSceneLabel } from './constants/sceneMapping';
import type { ChatMessage } from './types';
import type { SceneOption } from './hooks/useAutoScene';

const { useToken } = theme;

// SSE 事件类型
type SSEEventType =
  | 'meta'
  | 'delta'
  | 'thinking_delta'
  | 'tool_call'
  | 'tool_result'
  | 'approval_required'
  | 'confirmation_required'
  | 'done'
  | 'error'
  | 'heartbeat';

// 扩展消息类型，包含 thinking
interface ExtendedChatMessage extends ChatMessage {
  thinking?: string;
}

// 默认提示词
const DEFAULT_PROMPTS = [
  { key: 'deploy', description: '如何部署一个新服务？' },
  { key: 'monitor', description: '如何查看服务监控指标？' },
  { key: 'troubleshoot', description: '帮我排查服务异常问题' },
];

// 思考过程渲染组件
const ThinkingBlock: React.FC<{ content: string; isStreaming?: boolean }> = ({ content, isStreaming }) => {
  const { token } = theme.useToken();

  if (!content) return null;

  return (
    <Collapse
      defaultActiveKey={['thinking']}
      size="small"
      style={{
        marginBottom: 12,
        background: token.colorBgTextHover,
        border: `1px solid ${token.colorBorderSecondary}`,
        borderRadius: token.borderRadius,
      }}
      items={[
        {
          key: 'thinking',
          label: (
            <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <BulbOutlined style={{ color: token.colorWarning }} />
              <span>{isStreaming ? '思考中...' : '思考过程'}</span>
            </span>
          ),
          children: (
            <div style={{
              fontSize: 13,
              color: token.colorTextSecondary,
              whiteSpace: 'pre-wrap',
              maxHeight: 300,
              overflow: 'auto',
            }}>
              {content}
            </div>
          ),
        },
      ]}
    />
  );
};

// Markdown 内容渲染组件
const MarkdownContent: React.FC<{ content: string }> = ({ content }) => {
  return (
    <div className="ai-markdown-content">
      <XMarkdown content={content} />
    </div>
  );
};

// 助手消息渲染组件
const AssistantMessage: React.FC<{
  content: string;
  thinking?: string;
  isStreaming?: boolean;
}> = ({ content, thinking, isStreaming }) => {
  const { token } = theme.useToken();

  return (
    <div>
      {/* 思考过程 */}
      {thinking && <ThinkingBlock content={thinking} isStreaming={isStreaming} />}

      {/* 主要内容 */}
      {content ? (
        <MarkdownContent content={content} />
      ) : isStreaming ? (
        <span style={{ color: token.colorTextSecondary }}>正在输入...</span>
      ) : null}
    </div>
  );
};

// 消息渲染配置
const createRoleConfig = () => ({
  assistant: {
    placement: 'start' as const,
    footer: (
      <div style={{ display: 'flex', marginTop: 4 }}>
        <Tooltip title="复制">
          <Button type="text" size="small" icon={<CopyOutlined />} />
        </Tooltip>
        <Tooltip title="有帮助">
          <Button type="text" size="small" icon={<LikeOutlined />} />
        </Tooltip>
        <Tooltip title="无帮助">
          <Button type="text" size="small" icon={<DislikeOutlined />} />
        </Tooltip>
      </div>
    ),
  },
  user: { placement: 'end' as const },
});

// 会话类型
interface ConversationItem {
  key: string;
  label: string;
  group: string;
  messages: ExtendedChatMessage[];
}

// 发送 SSE 请求
async function sendChatMessage(
  scene: string,
  sessionId: string | undefined,
  content: string,
  onChunk: (chunk: { type: SSEEventType; data: Record<string, unknown> }) => void,
  signal?: AbortSignal
): Promise<void> {
  const token = localStorage.getItem('token');
  const projectId = localStorage.getItem('projectId');

  const response = await fetch('/api/v1/ai/chat', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(projectId ? { 'X-Project-ID': projectId } : {}),
    },
    body: JSON.stringify({
      sessionId,
      message: content,
      context: { scene },
    }),
    signal,
  });

  if (!response.ok || !response.body) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`请求失败: ${response.status} ${errorText}`);
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });

    // 按双换行分割事件
    const events = buffer.split('\n\n');
    buffer = events.pop() || '';

    for (const eventBlock of events) {
      if (!eventBlock.trim()) continue;

      // 解析事件块
      const lines = eventBlock.split('\n');
      let eventType: string | null = null;
      let eventData: string | null = null;

      for (const line of lines) {
        if (line.startsWith('event:')) {
          eventType = line.slice(6).trim();
        } else if (line.startsWith('data:')) {
          eventData = line.slice(5).trim();
        }
      }

      if (eventType && eventData) {
        let data: Record<string, unknown> = {};
        try {
          data = JSON.parse(eventData);
        } catch {
          data = { message: eventData };
        }
        onChunk({ type: eventType as SSEEventType, data });
      }
    }
  }
}

interface CopilotProps {
  /** 是否打开 */
  open?: boolean;
  /** 关闭回调 */
  onClose?: () => void;
  /** 当前场景（用于 API 调用） */
  scene: string;
  /** 用于 Select 显示的值 */
  selectValue?: string;
  /** 场景切换回调 */
  onSceneChange?: (scene: string) => void;
  /** 可用场景列表 */
  availableScenes?: SceneOption[];
  /** 是否自动模式 */
  isAuto?: boolean;
}

/**
 * Copilot 主组件
 */
export const Copilot: React.FC<CopilotProps> = ({
  open = true,
  onClose,
  scene,
  selectValue,
  onSceneChange,
  availableScenes = [{ key: 'global', label: '全局助手' }],
  isAuto = true,
}) => {
  const { token } = useToken();
  const listRef = useRef<BubbleListRef>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  // 输入状态
  const [inputValue, setInputValue] = useState('');

  // 会话状态
  const [conversations, setConversations] = useState<ConversationItem[]>([
    { key: 'default', label: '新对话', group: '今天', messages: [] },
  ]);
  const [activeKey, setActiveKey] = useState('default');

  // 当前会话
  const activeConversation = useMemo(() => {
    return conversations.find(c => c.key === activeKey) || conversations[0];
  }, [conversations, activeKey]);

  // 当前会话的消息
  const messages = activeConversation.messages;

  // 是否正在请求
  const [isLoading, setIsLoading] = useState(false);

  // 会话 ID
  const [sessionId, setSessionId] = useState<string | undefined>();

  // 发送消息
  const handleSubmit = useCallback(async (val: string) => {
    if (!val.trim() || isLoading) return;

    // 添加用户消息
    const userMessage: ExtendedChatMessage = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: val,
      createdAt: new Date().toISOString(),
    };

    setConversations(prev => prev.map(c => {
      if (c.key !== activeKey) return c;
      const newLabel = c.label === '新对话' ? val.slice(0, 20) + (val.length > 20 ? '...' : '') : c.label;
      return { ...c, label: newLabel, messages: [...c.messages, userMessage] };
    }));

    setIsLoading(true);

    // 创建助手消息占位
    const assistantId = `assistant-${Date.now()}`;
    let assistantContent = '';
    let assistantThinking = '';

    // 添加助手消息占位
    setConversations(prev => prev.map(c => {
      if (c.key !== activeKey) return c;
      return {
        ...c,
        messages: [...c.messages, {
          id: assistantId,
          role: 'assistant',
          content: '',
          createdAt: new Date().toISOString(),
        }],
      };
    }));

    // 创建 AbortController
    abortControllerRef.current = new AbortController();

    try {
      await sendChatMessage(
        scene,
        sessionId,
        val,
        (chunk) => {
          const { type, data } = chunk;

          switch (type) {
            case 'meta':
              if (data.sessionId) {
                setSessionId(data.sessionId as string);
              }
              break;

            case 'delta':
              assistantContent += (data.contentChunk as string) || '';
              break;

            case 'thinking_delta':
              assistantThinking += (data.contentChunk as string) || '';
              break;

            case 'done':
            case 'error':
              setIsLoading(false);
              break;
          }

          // 更新助手消息
          setConversations(prev => prev.map(c => {
            if (c.key !== activeKey) return c;
            return {
              ...c,
              messages: c.messages.map(m => {
                if (m.id !== assistantId) return m;
                return {
                  ...m,
                  content: assistantContent,
                  thinking: assistantThinking || undefined,
                };
              }),
            };
          }));
        },
        abortControllerRef.current.signal
      );
    } catch (error) {
      if ((error as Error).name !== 'AbortError') {
        message.error('请求失败，请稍后重试');
      }
      setIsLoading(false);
    }

    // 延迟滚动，等待渲染
    setTimeout(() => {
      listRef.current?.scrollTo({ top: 'bottom' });
    }, 100);
  }, [scene, sessionId, activeKey, isLoading]);

  // 中止请求
  const handleAbort = useCallback(() => {
    abortControllerRef.current?.abort();
    setIsLoading(false);
  }, []);

  // 新建会话
  const handleNewConversation = useCallback(() => {
    const timeNow = dayjs().valueOf().toString();
    setConversations(prev => [
      { key: timeNow, label: '新对话', group: '今天', messages: [] },
      ...prev,
    ]);
    setActiveKey(timeNow);
    setSessionId(undefined);
  }, []);

  // 场景选择器
  const sceneSelector = useMemo(() => {
    const sceneLabel = getSceneLabel(scene);
    const displayValue = selectValue || scene;

    if (!onSceneChange || availableScenes.length <= 1) {
      return (
        <span style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 4,
          padding: '2px 8px',
          background: token.colorPrimaryBg,
          borderRadius: token.borderRadiusSM,
          fontSize: 12,
          color: token.colorPrimary,
        }}>
          {scene === 'global' ? <GlobalOutlined /> : <EnvironmentOutlined />}
          {isAuto ? `自动: ${sceneLabel}` : sceneLabel}
        </span>
      );
    }

    return (
      <Select
        value={displayValue}
        onChange={onSceneChange}
        size="small"
        style={{ width: 160 }}
        optionLabelProp="label"
        popupMatchSelectWidth={false}
      >
        {availableScenes.map((s) => (
          <Select.Option key={s.key} value={s.key} label={s.label}>
            <Space>
              {s.key === '__auto__' ? (
                <GlobalOutlined style={{ color: token.colorPrimary }} />
              ) : s.key === 'global' ? (
                <GlobalOutlined />
              ) : (
                <EnvironmentOutlined />
              )}
              {s.label}
            </Space>
          </Select.Option>
        ))}
      </Select>
    );
  }, [scene, selectValue, isAuto, onSceneChange, availableScenes, token]);

  // 角色配置
  const role = useMemo(() => createRoleConfig(), []);

  // 渲染消息内容
  const renderMessageContent = useCallback((msg: ExtendedChatMessage, isCurrentStreaming: boolean) => {
    if (msg.role === 'user') {
      return msg.content;
    }

    // 助手消息
    const isStreaming = isCurrentStreaming && !msg.content && !msg.thinking;

    return (
      <AssistantMessage
        content={msg.content}
        thinking={msg.thinking}
        isStreaming={isStreaming || (isCurrentStreaming && !!msg.thinking && !msg.content)}
      />
    );
  }, []);

  if (!open) return null;

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      height: '100%',
      background: token.colorBgContainer,
      color: token.colorText,
    }}>
      {/* 头部 */}
      <div style={{
        height: 52,
        boxSizing: 'border-box',
        borderBottom: `1px solid ${token.colorBorder}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '0 12px 0 16px',
        flexShrink: 0,
      }}>
        <div style={{
          fontWeight: 600,
          fontSize: 15,
          display: 'flex',
          alignItems: 'center',
          gap: 8,
        }}>
          {sceneSelector}
          <span>AI Copilot</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <Tooltip title="新建对话">
            <Button
              type="text"
              icon={<PlusOutlined />}
              onClick={handleNewConversation}
            />
          </Tooltip>
          <Popover
            placement="bottomRight"
            styles={{ container: { padding: 0, maxHeight: 400 } }}
            content={
              <Conversations
                items={conversations.map(i =>
                  i.key === activeKey ? { ...i, label: `[当前] ${i.label}` } : i
                )}
                activeKey={activeKey}
                groupable
                onActiveChange={setActiveKey}
                styles={{ item: { padding: '0 8px' } }}
                style={{ width: 280, maxHeight: 400, overflowY: 'auto' }}
              />
            }
          >
            <Tooltip title="会话列表">
              <Button type="text" icon={<CommentOutlined />} />
            </Tooltip>
          </Popover>
          {onClose && (
            <Tooltip title="关闭">
              <Button
                type="text"
                icon={<CloseOutlined />}
                onClick={onClose}
              />
            </Tooltip>
          )}
        </div>
      </div>

      {/* 消息列表 */}
      <div style={{
        flex: 1,
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
        minHeight: 0,
      }}>
        {messages.length > 0 ? (
          <Bubble.List
            ref={listRef}
            style={{ paddingInline: 16, height: '100%' }}
            items={messages.map(m => ({
              key: m.id,
              content: renderMessageContent(m, isLoading && messages[messages.length - 1]?.id === m.id),
              role: m.role,
              loading: m.role === 'assistant' && isLoading && !m.content && !m.thinking,
            }))}
            role={role}
          />
        ) : (
          <>
            <Welcome
              variant="borderless"
              title="👋 你好，我是 AI Copilot"
              description="我可以帮助你进行部署管理、服务治理、监控运维等操作"
              style={{
                margin: 16,
                padding: 16,
                borderRadius: 8,
                background: token.colorBgTextHover,
              }}
            />
            <Prompts
              vertical
              title="我可以帮你："
              items={DEFAULT_PROMPTS}
              onItemClick={(info) => handleSubmit(info?.data?.description as string)}
              style={{ margin: '0 16px 16px' }}
              styles={{
                title: { fontSize: 14 },
              }}
            />
          </>
        )}
      </div>

      {/* 输入框 */}
      <div style={{
        padding: '12px 16px',
        borderTop: `1px solid ${token.colorBorder}`,
        background: token.colorBgContainer,
        flexShrink: 0,
      }}>
        <Sender
          loading={isLoading}
          value={inputValue}
          onChange={setInputValue}
          onSubmit={() => {
            handleSubmit(inputValue);
            setInputValue('');
          }}
          onCancel={handleAbort}
          placeholder="输入消息或使用 / 触发快捷命令"
          allowSpeech
        />
      </div>
    </div>
  );
};

export default Copilot;
