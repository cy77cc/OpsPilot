/**
 * Copilot 组件
 * 使用 @ant-design/x 组件实现的 AI 助手
 */
import React, { useState, useRef, useCallback, useMemo, useEffect } from 'react';
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
  ThoughtChain,
  Welcome,
} from '@ant-design/x';
import type { BubbleListRef, BubbleProps } from '@ant-design/x/es/bubble';
import { Button, message, Popover, Select, Space, Tooltip, theme, Skeleton } from 'antd';
import dayjs from 'dayjs';
import { getSceneLabel } from './constants/sceneMapping';
import type { ChatMessage, EmbeddedRecommendation, SSEEventType, ThoughtStageDetailItem, ThoughtStageItem, ThoughtStageStatus } from './types';
import type { SceneOption } from './hooks/useAutoScene';
import { useConversationRestore, type RestoredConversation } from './hooks/useConversationRestore';
import { useScenePrompts } from './hooks/useScenePrompts';
import { MessageActions } from './components/MessageActions';
import { AssistantMessageBlocks } from './components/AssistantMessageBlocks';
import { normalizeAssistantMessage } from './messageBlocks';

const { useToken } = theme;

// 扩展消息类型，包含 thinking 和 recommendations
interface ExtendedChatMessage extends ChatMessage {
  thinking?: string;
  recommendations?: EmbeddedRecommendation[];
}

function buildStageDescription(
  stage: ThoughtStageItem['key'],
  status: ThoughtStageStatus,
  source?: Record<string, unknown>,
): string | undefined {
  if (stage === 'user_action') {
    return typeof source?.message === 'string'
      ? source.message
      : typeof source?.title === 'string'
        ? source.title
        : undefined;
  }
  switch (stage) {
    case 'rewrite':
      return status === 'success' ? '已完成问题理解与目标提炼' : '正在理解问题并提炼目标';
    case 'plan':
      return status === 'success' ? '已整理出执行计划与边界' : '正在整理执行计划与边界';
    case 'execute':
      if (status === 'error') {
        return '执行过程中出现异常';
      }
      return status === 'success' ? '已完成计划执行并收集结果' : '正在调用专家执行计划';
    case 'summary':
      return status === 'success' ? '已生成最终结论' : '正在生成最终结论';
    default:
      return undefined;
  }
}

function buildStageMilestone(
  stage: ThoughtStageItem['key'],
  event: 'event' | 'delta',
  source?: Record<string, unknown>,
): string | undefined {
  if (event === 'delta') {
    switch (stage) {
      case 'rewrite':
        return '正在提炼目标、资源和约束条件。';
      case 'plan':
        return '正在整理步骤、执行边界和所需上下文。';
      default:
        return undefined;
    }
  }
  switch (stage) {
    case 'rewrite':
      return '已识别用户目标，准备进入规划。';
    case 'plan':
      return '已形成可执行计划。';
    case 'execute':
      if (typeof source?.title === 'string' && source.title.trim()) {
        return `当前步骤：${source.title.trim()}`;
      }
      return '正在按计划推进执行。';
    case 'user_action':
      return typeof source?.user_visible_summary === 'string'
        ? source.user_visible_summary
        : undefined;
    default:
      return undefined;
  }
}

function appendStageContent(current: string | undefined, next: string | undefined): string | undefined {
  const normalizedCurrent = (current || '').trim();
  const normalizedNext = (next || '').trim();
  if (!normalizedNext) {
    return normalizedCurrent || undefined;
  }
  if (!normalizedCurrent) {
    return normalizedNext;
  }
  const lines = normalizedCurrent.split('\n').map((line) => line.trim()).filter(Boolean);
  if (lines.includes(normalizedNext)) {
    return normalizedCurrent;
  }
  return `${normalizedCurrent}\n${normalizedNext}`.trim();
}

function visibleThoughtChain(stages: ThoughtStageItem[] | undefined): ThoughtStageItem[] {
  return (stages || []).filter((item) => item.key !== 'summary');
}

function updateAssistantMessage(
  conversations: ConversationItem[],
  conversationKey: string,
  assistantId: string,
  updater: (message: ExtendedChatMessage) => ExtendedChatMessage
): ConversationItem[] {
  return conversations.map((conversation) => {
    if (conversation.key !== conversationKey) {
      return conversation;
    }
    return {
      ...conversation,
      messages: conversation.messages.map((message) => (
        message.id === assistantId ? updater(message) : message
      )),
    };
  });
}

function upsertThoughtStage(
  stages: ThoughtStageItem[],
  patch: Partial<ThoughtStageItem> & Pick<ThoughtStageItem, 'key' | 'title' | 'status'>
): ThoughtStageItem[] {
  const index = stages.findIndex((item) => item.key === patch.key);
  const current = index >= 0 ? stages[index] : undefined;
  const nextDetails = patch.details ?? current?.details;
  const nextContent = patch.content ?? current?.content;
  const next: ThoughtStageItem = {
    key: patch.key,
    title: patch.title,
    status: patch.status,
    description: patch.description ?? current?.description,
    content: renderThoughtContent(nextContent, nextDetails),
    footer: patch.footer ?? current?.footer,
    details: nextDetails,
    collapsible: patch.collapsible ?? true,
    blink: patch.blink ?? patch.status === 'loading',
  };
  if (index === -1) {
    return [...stages, next];
  }
  const merged = {
    ...current,
    ...next,
    content: renderThoughtContent(nextContent, nextDetails),
    blink: patch.blink ?? next.status === 'loading',
  };
  return stages.map((item, itemIndex) => (itemIndex === index ? merged : item));
}

function renderThoughtContent(content?: string, details?: ThoughtStageDetailItem[]): string | undefined {
  const summary = (content || '').trim();
  const detailLines = (details || []).map((detail) => {
    const prefix = detail.status === 'error' ? '[失败]' : detail.status === 'success' ? '[完成]' : '[执行中]';
    const body = detail.content?.trim();
    return body ? `${prefix} ${detail.label}: ${body}` : `${prefix} ${detail.label}`;
  });
  const segments = [summary, ...detailLines].filter(Boolean);
  return segments.length > 0 ? segments.join('\n') : undefined;
}

function upsertThoughtDetail(
  stages: ThoughtStageItem[],
  stageKey: ThoughtStageItem['key'],
  detail: ThoughtStageDetailItem
): ThoughtStageItem[] {
  return stages.map((item) => {
    if (item.key !== stageKey) {
      return item;
    }
    const details = [...(item.details || [])];
    const index = details.findIndex((candidate) => candidate.id === detail.id);
    if (index === -1) {
      details.push(detail);
    } else {
      details[index] = { ...details[index], ...detail };
    }
    return {
      ...item,
      details,
      content: renderThoughtContent(item.content, details),
    };
  });
}

function normalizeThoughtStatus(status: string | undefined, fallback: ThoughtStageStatus = 'loading'): ThoughtStageStatus {
  switch (status) {
    case 'completed':
    case 'success':
      return 'success';
    case 'failed':
    case 'error':
    case 'blocked':
      return 'error';
    case 'cancelled':
    case 'rejected':
      return 'abort';
    case 'running':
    case 'waiting_approval':
    case 'planning':
    case 'replanning':
      return 'loading';
    default:
      return fallback;
  }
}

function resolveThoughtStageTitle(stage: string | undefined): ThoughtStageItem['title'] {
  switch (stage) {
    case 'rewrite':
      return '理解你的问题';
    case 'plan':
      return '整理排查计划';
    case 'execute':
      return '调用专家执行';
    case 'summary':
      return '生成结论';
    default:
      return '处理中';
  }
}

// 助手消息渲染组件
const AssistantMessage: React.FC<{
  content: string;
  thinking?: string;
  recommendations?: EmbeddedRecommendation[];
  thoughtChain?: ThoughtStageItem[];
  rawEvidence?: string[];
  isStreaming?: boolean;
  showActions?: boolean;
  onRegenerate?: () => void;
  onRecommendationSelect?: (prompt: string) => void;
  isLoading?: boolean;
}> = ({ content, thinking, recommendations, thoughtChain, rawEvidence, isStreaming, showActions = true, onRegenerate, onRecommendationSelect, isLoading }) => {
  const { token } = theme.useToken();
  const chainItems = useMemo(() => visibleThoughtChain(thoughtChain), [thoughtChain]);
  const blocks = useMemo(() => normalizeAssistantMessage({
    content,
    thinking,
    rawEvidence,
    recommendations,
    isStreaming,
  }), [content, thinking, rawEvidence, recommendations, isStreaming]);

  return (
    <div>
      {chainItems.length > 0 && (
        <div style={{ marginBottom: 12 }}>
          <ThoughtChain
            items={chainItems.map((item) => ({
              key: item.key,
              title: item.title,
              description: item.description,
              content: item.content,
              footer: item.footer,
              status: item.status,
              collapsible: item.collapsible,
              blink: item.blink,
            }))}
            defaultExpandedKeys={[]}
          />
        </div>
      )}
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

function getLastAssistantMessage(session: Record<string, unknown> | undefined): Record<string, unknown> | undefined {
  const messages = Array.isArray(session?.messages) ? (session?.messages as Record<string, unknown>[]) : [];
  for (let i = messages.length - 1; i >= 0; i -= 1) {
    if (messages[i]?.role === 'assistant') {
      return messages[i];
    }
  }
  return undefined;
}

function resolveStreamContent(data: Record<string, unknown>): string {
  const value = data.contentChunk ?? data.content_chunk ?? data.content ?? data.message;
  if (typeof value === 'string') {
    return value;
  }
  if (value == null) {
    return '';
  }
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
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
    setSessionId(restored.id);
  }, []);

  useEffect(() => {
    setConversations([{ key: 'default', label: '新对话', group: '今天', messages: [] }]);
    setActiveKey('default');
    setSessionId(undefined);
    setIsLoading(false);
  }, [scene]);

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
    let assistantTraceId: string | undefined;

    // 添加助手消息占位
    setConversations(prev => prev.map(c => {
      if (c.key !== activeKey) return c;
      return {
        ...c,
        messages: [...c.messages, {
          id: assistantId,
          role: 'assistant',
          content: '',
          thoughtChain: [],
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
              if (data.traceId) {
                assistantTraceId = String(data.traceId);
              }
              break;

            case 'rewrite_result':
              setConversations((prev) => updateAssistantMessage(prev, activeKey, assistantId, (message) => ({
                ...message,
                thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
                  key: 'rewrite',
                  title: '理解你的问题',
                  status: 'success',
                  description: buildStageDescription('rewrite', 'success'),
                  content: appendStageContent(
                    (message.thoughtChain || []).find((item) => item.key === 'rewrite')?.content,
                    buildStageMilestone('rewrite', 'event', data),
                  ),
                }),
              })));
              break;

            case 'planner_state':
              setConversations((prev) => updateAssistantMessage(prev, activeKey, assistantId, (message) => ({
                ...message,
                thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
                  key: 'plan',
                  title: '整理排查计划',
                  status: normalizeThoughtStatus(data.status as string | undefined, 'loading'),
                  description: buildStageDescription('plan', normalizeThoughtStatus(data.status as string | undefined, 'loading')),
                  content: appendStageContent(
                    (message.thoughtChain || []).find((item) => item.key === 'plan')?.content,
                    buildStageMilestone('plan', 'delta', data),
                  ),
                }),
              })));
              break;

            case 'plan_created':
              setConversations((prev) => updateAssistantMessage(prev, activeKey, assistantId, (message) => ({
                ...message,
                thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
                  key: 'plan',
                  title: '整理排查计划',
                  status: 'success',
                  description: buildStageDescription('plan', 'success'),
                  content: appendStageContent(
                    (message.thoughtChain || []).find((item) => item.key === 'plan')?.content,
                    buildStageMilestone('plan', 'event', data),
                  ),
                }),
              })));
              break;

            case 'stage_delta': {
              const stageKey = String(data.stage || '') as ThoughtStageItem['key'];
              if (!stageKey) {
                break;
              }
              const chunkText = resolveStreamContent(data);
              const replace = Boolean(data.replace);
              setConversations((prev) => updateAssistantMessage(prev, activeKey, assistantId, (message) => {
                if (stageKey === 'summary') {
                  const previousThinking = message.thinking || '';
                  const nextThinking = replace ? chunkText : `${previousThinking}${chunkText}`.trim();
                  return {
                    ...message,
                    thinking: nextThinking,
                  };
                }
                const currentStage = (message.thoughtChain || []).find((item) => item.key === stageKey);
                const status = normalizeThoughtStatus(data.status as string | undefined, 'loading');
                const milestone = buildStageMilestone(stageKey, 'delta', data);
                const nextContent = appendStageContent(replace ? '' : currentStage?.content, milestone || chunkText);
                return {
                  ...message,
                  thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
                    key: stageKey,
                    title: resolveThoughtStageTitle(stageKey),
                    status,
                    description: buildStageDescription(stageKey, status, data) || currentStage?.description,
                    content: nextContent,
                  }),
                };
              }));
              break;
            }

            case 'step_update':
              setConversations((prev) => updateAssistantMessage(prev, activeKey, assistantId, (message) => ({
                ...message,
                thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
                  key: 'execute',
                  title: '调用专家执行',
                  status: normalizeThoughtStatus(data.status as string | undefined, 'loading'),
                  description: buildStageDescription('execute', normalizeThoughtStatus(data.status as string | undefined, 'loading'), data),
                  content: appendStageContent(
                    (message.thoughtChain || []).find((item) => item.key === 'execute')?.content,
                    buildStageMilestone('execute', 'event', data),
                  ),
                }),
              })));
              break;

            case 'tool_call':
              setConversations((prev) => updateAssistantMessage(prev, activeKey, assistantId, (message) => {
                const nextStages = upsertThoughtStage(message.thoughtChain || [], {
                  key: 'execute',
                  title: '调用专家执行',
                  status: 'loading',
                  description: buildStageDescription('execute', 'loading', data),
                  content: appendStageContent(
                    (message.thoughtChain || []).find((item) => item.key === 'execute')?.content,
                    String(data.summary || ''),
                  ),
                });
                return {
                  ...message,
                  thoughtChain: upsertThoughtDetail(nextStages, 'execute', {
                    id: String(data.call_id || data.tool_name || Date.now()),
                    label: String(data.tool_name || data.expert || 'tool'),
                    status: 'loading',
                    content: String(data.summary || ''),
                  }),
                };
              }));
              break;

            case 'tool_result':
              setConversations((prev) => updateAssistantMessage(prev, activeKey, assistantId, (message) => {
                const result = data.result as Record<string, unknown> | undefined;
                const status = data.status === 'error' || result?.ok === false ? 'error' : 'success';
                return {
                  ...message,
                  thoughtChain: upsertThoughtDetail(message.thoughtChain || [], 'execute', {
                    id: String(data.call_id || data.tool_name || Date.now()),
                    label: String(data.tool_name || data.expert || 'tool'),
                    status,
                    content: String(data.error || data.summary || ''),
                  }),
                };
              }));
              break;

            case 'delta':
            case 'message':
              assistantContent += resolveStreamContent(data);
              break;

            case 'thinking_delta':
              assistantThinking += (data.contentChunk as string) || '';
              break;

            case 'approval_required':
              setConversations((prev) => updateAssistantMessage(prev, activeKey, assistantId, (message) => ({
                ...message,
                thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
                  key: 'user_action',
                  title: '等待你确认',
                  status: 'loading',
                  description: String(data.title || '当前步骤需要确认后继续执行'),
                  content: String(data.user_visible_summary || ''),
                }),
              })));
              break;

            case 'clarify_required':
              setConversations((prev) => updateAssistantMessage(prev, activeKey, assistantId, (message) => ({
                ...message,
                thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
                  key: 'user_action',
                  title: '等待你补充信息',
                  status: 'loading',
                  description: String(data.message || data.title || '当前目标仍有歧义'),
                }),
              })));
              assistantContent ||= String(data.message || '');
              break;

            case 'replan_started':
              setConversations((prev) => updateAssistantMessage(prev, activeKey, assistantId, (message) => ({
                ...message,
                thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
                  key: 'plan',
                  title: '整理排查计划',
                  status: 'loading',
                  description: '正在开始新一轮规划',
                }),
              })));
              break;

            case 'summary': {
              setConversations((prev) => updateAssistantMessage(prev, activeKey, assistantId, (message) => ({
                ...message,
                thinking: message.thinking || String(data.summary || ''),
              })));
              break;
            }

            case 'done':
              // 提取推荐
              if (data.turn_recommendations) {
                assistantRecommendations = data.turn_recommendations as EmbeddedRecommendation[];
              }
              if (data.session && typeof data.session === 'object') {
                const session = data.session as Record<string, unknown>;
                const finalAssistant = getLastAssistantMessage(session);
                assistantContent ||= String(finalAssistant?.content || '');
                if (!assistantThinking) {
                  assistantThinking = String(finalAssistant?.thinking || '');
                  if (!assistantThinking) {
                    const finalThoughtChain = Array.isArray(finalAssistant?.thoughtChain)
                      ? (finalAssistant?.thoughtChain as ThoughtStageItem[])
                      : [];
                    const summaryStage = finalThoughtChain.find((item) => item.key === 'summary');
                    assistantThinking = String(summaryStage?.content || '');
                  }
                }
                assistantRecommendations ||= (finalAssistant?.recommendations as EmbeddedRecommendation[] | undefined);
                const finalRawEvidence = (finalAssistant?.rawEvidence as string[] | undefined) || undefined;
                setConversations((prev) => updateAssistantMessage(prev, activeKey, assistantId, (message) => ({
                  ...message,
                  rawEvidence: finalRawEvidence || message.rawEvidence,
                })));
                if (session.id) {
                  setSessionId(session.id as string);
                }
              }
              setConversations((prev) => updateAssistantMessage(prev, activeKey, assistantId, (message) => ({
                ...message,
                thoughtChain: (message.thoughtChain || []).map((item) => ({
                  ...item,
                  blink: false,
                  status: item.status === 'loading' ? 'success' : item.status,
                })),
              })));
              setIsLoading(false);
              break;

            case 'error':
              assistantContent ||= String(data.message || '当前 AI 阶段执行失败，请稍后重试。').trim();
              setConversations((prev) => updateAssistantMessage(prev, activeKey, assistantId, (message) => {
                const stageKey = String(data.stage || '').trim() as ThoughtStageItem['key'];
                const errorText = String(data.message || '当前 AI 阶段执行失败，请稍后重试。').trim();
                const nextThoughtChain = stageKey
                  ? upsertThoughtStage(message.thoughtChain || [], {
                    key: stageKey,
                    title: resolveThoughtStageTitle(stageKey),
                    status: 'error',
                    description: errorText,
                    content: errorText,
                    blink: false,
                  })
                  : (message.thoughtChain || []).map((item, index, items) => (
                    index === items.length - 1 ? { ...item, status: 'error' as ThoughtStageStatus, blink: false, content: item.content || errorText } : item
                  ));
                return {
                  ...message,
                  content: message.content || errorText,
                  thoughtChain: nextThoughtChain,
                };
              }));
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
                  traceId: assistantTraceId,
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
    const hasThoughtChain = visibleThoughtChain(msg.thoughtChain).length > 0;
    const hasVisibleAssistantState = Boolean(msg.content || msg.thinking || hasThoughtChain);
    const isStreaming = isCurrentStreaming && !hasVisibleAssistantState;

    // 只有当消息内容正在生成时（内容为空）才显示 loading
    // 如果消息已经有内容了（即使正在生成推荐），重新生成按钮不显示 loading
    const showLoading = isLoading && isCurrentStreaming && !msg.content && !hasThoughtChain;

    return (
      <AssistantMessage
        content={msg.content}
        thinking={msg.thinking}
        recommendations={msg.recommendations}
        thoughtChain={msg.thoughtChain}
        rawEvidence={msg.rawEvidence}
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
              loading: m.role === 'assistant'
                && isLoading
                && !m.content
                && !m.thinking
                && !(m.thoughtChain && m.thoughtChain.length > 0),
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
