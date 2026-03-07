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
  ReloadOutlined,
} from '@ant-design/icons';
import {
  Bubble,
  Conversations,
  Prompts,
  Sender,
  Welcome,
} from '@ant-design/x';
import type { BubbleListRef, BubbleProps } from '@ant-design/x/es/bubble';
import { Button, message, Popover, Select, Space, Tooltip, theme, Skeleton } from 'antd';
import dayjs from 'dayjs';
import { getSceneLabel } from './constants/sceneMapping';
import type { ChatMessage, EmbeddedRecommendation } from './types';
import type { SceneOption } from './hooks/useAutoScene';
import { useConversationRestore, type RestoredConversation } from './hooks/useConversationRestore';
import { useScenePrompts } from './hooks/useScenePrompts';
import { MessageActions } from './components/MessageActions';
import { AssistantMessageBlocks } from './components/AssistantMessageBlocks';
import { normalizeAssistantMessage } from './messageBlocks';

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

// 扩展消息类型，包含 thinking 和 recommendations
interface ExtendedChatMessage extends ChatMessage {
  thinking?: string;
  recommendations?: EmbeddedRecommendation[];
}

// 助手消息渲染组件
const AssistantMessage: React.FC<{
  content: string;
  thinking?: string;
  recommendations?: EmbeddedRecommendation[];
  isStreaming?: boolean;
  showActions?: boolean;
  onRegenerate?: () => void;
  onRecommendationSelect?: (prompt: string) => void;
  isLoading?: boolean;
}> = ({ content, thinking, recommendations, isStreaming, showActions = true, onRegenerate, onRecommendationSelect, isLoading }) => {
  const { token } = theme.useToken();
  const blocks = useMemo(() => normalizeAssistantMessage({
    content,
    thinking,
    recommendations,
    isStreaming,
  }), [content, thinking, recommendations, isStreaming]);

  return (
    <div>
      {blocks.length > 0 ? (
        <AssistantMessageBlocks
          blocks={blocks}
          onRecommendationSelect={onRecommendationSelect}
        />
      ) : isStreaming ? (
        <span style={{ color: token.colorTextSecondary }}>正在输入...</span>
      ) : null}

      {/* 消息操作按钮 */}
      {showActions && !isStreaming && content && (
        <MessageActions
          content={content}
          messageId=""
          isLoading={isLoading}
          onRegenerate={onRegenerate}
        />
      )}
    </div>
  );
};

// 消息渲染配置
const createRoleConfig = () => ({
  assistant: {
    placement: 'start' as const,
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

  // 恢复会话的回调
  const handleRestoreConversation = useCallback((restored: RestoredConversation) => {
    // 创建恢复的会话
    const restoredItem: ConversationItem = {
      key: restored.id,
      label: restored.title,
      group: '最近',
      messages: restored.messages.map(m => ({
        ...m,
        createdAt: m.createdAt || new Date().toISOString(),
      })),
    };
    setConversations([restoredItem]);
    setActiveKey(restored.id);
  }, []);

  // 使用会话恢复 hook
  const { isRestoring } = useConversationRestore({
    scene,
    enabled: open,
    onRestore: handleRestoreConversation,
  });

  // 使用场景提示词 hook
  const { prompts: scenePrompts } = useScenePrompts({
    scene,
    enabled: open,
  });

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
    let assistantRecommendations: EmbeddedRecommendation[] | undefined;

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
              // 提取推荐
              if (data.turn_recommendations) {
                assistantRecommendations = data.turn_recommendations as EmbeddedRecommendation[];
              }
              setIsLoading(false);
              break;

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
                  recommendations: assistantRecommendations,
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

  // 重新生成消息
  const handleRegenerate = useCallback(async (assistantMsgId: string) => {
    if (isLoading) return;

    // 找到当前助手消息之前的用户消息
    const msgIndex = messages.findIndex(m => m.id === assistantMsgId);
    if (msgIndex <= 0) {
      message.warning('无法重新生成：找不到对应的用户问题');
      return;
    }

    const userMessage = messages[msgIndex - 1];
    if (userMessage.role !== 'user' || !userMessage.content) {
      message.warning('无法重新生成：用户问题为空');
      return;
    }

    // 移除当前助手消息
    setConversations(prev => prev.map(c => {
      if (c.key !== activeKey) return c;
      return {
        ...c,
        messages: c.messages.filter(m => m.id !== assistantMsgId),
      };
    }));

    // 重新发送用户消息
    await handleSubmit(userMessage.content);
  }, [messages, activeKey, isLoading, handleSubmit]);

  // 处理推荐点击
  const handleRecommendationSelect = useCallback((prompt: string) => {
    handleSubmit(prompt);
  }, [handleSubmit]);

  // 渲染消息内容
  const renderMessageContent = useCallback((msg: ExtendedChatMessage, isCurrentStreaming: boolean) => {
    if (msg.role === 'user') {
      return msg.content;
    }

    // 助手消息
    const isStreaming = isCurrentStreaming && !msg.content && !msg.thinking;

    // 只有当消息内容正在生成时（内容为空）才显示 loading
    // 如果消息已经有内容了（即使正在生成推荐），重新生成按钮不显示 loading
    const showLoading = isLoading && isCurrentStreaming && !msg.content;

    return (
      <AssistantMessage
        content={msg.content}
        thinking={msg.thinking}
        recommendations={msg.recommendations}
        isStreaming={isStreaming || (isCurrentStreaming && !!msg.thinking && !msg.content)}
        onRegenerate={() => handleRegenerate(msg.id)}
        onRecommendationSelect={handleRecommendationSelect}
        isLoading={showLoading}
      />
    );
  }, [handleRegenerate, handleRecommendationSelect, isLoading]);

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
        {/* 恢复会话中的加载状态 */}
        {isRestoring ? (
          <div style={{ padding: 16 }}>
            <Skeleton active paragraph={{ rows: 4 }} />
          </div>
        ) : messages.length > 0 ? (
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
              items={scenePrompts.length > 0 ? scenePrompts : [{ key: 'default', description: '有什么可以帮助你的？' }]}
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
