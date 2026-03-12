/**
 * Copilot 组件
 * 使用 @ant-design/x 组件实现的 AI 助手
 */
import React, { useState, useRef, useCallback, useMemo, useEffect, useReducer } from 'react';
import {
  CloseOutlined,
  CommentOutlined,
  GlobalOutlined,
  EnvironmentOutlined,
  DownOutlined,
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
import type { BubbleListRef } from '@ant-design/x/es/bubble';
import { Button, message, Popover, Segmented, Select, Space, Tooltip, theme, Skeleton } from 'antd';
import dayjs from 'dayjs';
import { aiApi } from '../../api/modules/ai';
import type { ApprovalTicket, SSEDoneEvent, SSEStageDeltaEvent } from '../../api/modules/ai';
import { getSceneLabel } from './constants/sceneMapping';
import type { ChatMessage, ChatTurn, EmbeddedRecommendation, ThoughtStageDetailItem, ThoughtStageItem, ThoughtStageStatus } from './types';
import type { SceneOption } from './hooks/useAutoScene';
import { useConversationRestore, type RestoredConversation } from './hooks/useConversationRestore';
import { useScenePrompts } from './hooks/useScenePrompts';
import { MessageActions } from './components/MessageActions';
import { AssistantMessageBlocks } from './components/AssistantMessageBlocks';
import { normalizeAssistantMessage, normalizeTurnBlocks } from './messageBlocks';
import {
  applyBlockClose,
  applyBlockDelta,
  applyBlockOpen,
  applyBlockReplace,
  applyTurnDone,
  applyTurnStarted,
  applyTurnState,
  createAssistantTurn,
  getTurnBlocksForDisplay,
  type DisplayMode,
  projectTurnSummary,
} from './turnLifecycle';

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

function finalizeThoughtStage(
  stages: ThoughtStageItem[] | undefined,
  key: ThoughtStageItem['key'],
  status: ThoughtStageStatus,
  description?: string,
): ThoughtStageItem[] {
  return (stages || []).map((item) => (
    item.key === key
      ? {
          ...item,
          status,
          blink: false,
          description: description ?? item.description,
        }
      : item
  ));
}

function buildApprovalBlockID(payload: Record<string, unknown>): string {
  const resume = (payload.resume || {}) as Record<string, unknown>;
  const stepID = String(payload.step_id || resume.step_id || '').trim();
  if (stepID) {
    return `approval:${stepID}`;
  }
  const planID = String(payload.plan_id || resume.plan_id || '').trim();
  if (planID) {
    return `approval:${planID}`;
  }
  return 'approval:active';
}

function buildToolDetailID(data: Record<string, unknown>): string {
  const callID = String(data.call_id || '').trim();
  if (callID) {
    return callID;
  }
  const stepID = String(data.step_id || '').trim();
  const toolName = String(data.tool_name || data.tool || data.expert || 'tool').trim();
  return ['tool', stepID || 'step', toolName || 'tool'].join(':');
}

function buildStepDetailID(data: Record<string, unknown>): string {
  const stepID = String(data.step_id || '').trim();
  if (stepID) {
    return `step:${stepID}`;
  }
  const title = String(data.title || data.summary || '').trim();
  return `step:${title || 'current'}`;
}

function buildApprovalStatusPayload(
  base: Record<string, unknown>,
  title: string,
  summary: string,
  status: 'waiting_user' | 'submitting' | 'failed',
  errorMessage?: string,
): Record<string, unknown> {
  return {
    ...base,
    title,
    summary,
    status,
    error_message: errorMessage,
  };
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

function buildExecutionDetailContent(
  type: 'step' | 'tool_call' | 'tool_result',
  data: Record<string, unknown>,
): string | undefined {
  const lines: string[] = [];
  const title = String(data.title || data.summary || data.user_visible_summary || '').trim();
  const target = String(data.target || data.host_id || data.resource || '').trim();
  const toolName = String(data.tool_name || data.tool || '').trim();
  const expert = String(data.expert || '').trim();
  const status = String(data.status || '').trim();
  const error = String(data.error || '').trim();
  const result = data.result && typeof data.result === 'object' ? data.result as Record<string, unknown> : undefined;
  const latency = typeof result?.latency_ms === 'number' ? `${result.latency_ms}ms` : '';
  const resultError = typeof result?.error === 'string' ? result.error : '';

  if (type === 'step' && title) {
    lines.push(`步骤: ${title}`);
  }
  if (type !== 'step' && toolName) {
    lines.push(`工具: ${toolName}`);
  }
  if (target) {
    lines.push(`目标: ${target}`);
  }
  if (expert && type === 'step') {
    lines.push(`专家: ${expert}`);
  }
  if (status && type === 'step') {
    lines.push(`状态: ${status}`);
  }
  if (title && type !== 'step') {
    lines.push(title);
  }
  if (error) {
    lines.push(`结果: ${error}`);
  } else if (resultError) {
    lines.push(`结果: ${resultError}`);
  } else if (type === 'tool_result') {
    lines.push(`结果: ${result?.ok === false ? '执行失败' : '执行完成'}`);
  }
  if (latency) {
    lines.push(`耗时: ${latency}`);
  }

  return lines.filter(Boolean).join('\n') || undefined;
}

function resolveDefaultExpandedThoughtKeys(
  stages: ThoughtStageItem[],
  opts: { restored?: boolean; streaming?: boolean },
): string[] {
  if (opts.restored || !opts.streaming) {
    return [];
  }
  const active = stages.filter((item) => item.status === 'loading').map((item) => item.key);
  return active.length > 0 ? [active[active.length - 1]] : [];
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
  turn?: ChatTurn;
  recommendations?: EmbeddedRecommendation[];
  thoughtChain?: ThoughtStageItem[];
  rawEvidence?: string[];
  isStreaming?: boolean;
  restored?: boolean;
  showActions?: boolean;
  displayMode: DisplayMode;
  reducedMotion: boolean;
  onRegenerate?: () => void;
  onRecommendationSelect?: (prompt: string) => void;
  onApprovalDecision?: (payload: Record<string, unknown>, approved: boolean) => void;
  isLoading?: boolean;
}> = ({
  content,
  thinking,
  turn,
  recommendations,
  thoughtChain,
  rawEvidence,
  isStreaming,
  restored,
  showActions = true,
  displayMode,
  reducedMotion,
  onRegenerate,
  onRecommendationSelect,
  onApprovalDecision,
  isLoading,
}) => {
  const { token } = theme.useToken();
  const chainItems = useMemo(
    () => visibleThoughtChain(thoughtChain),
    [thoughtChain],
  );
  const defaultExpandedKeys = useMemo(
    () => resolveDefaultExpandedThoughtKeys(chainItems, { restored, streaming: Boolean(isStreaming) }),
    [chainItems, isStreaming, restored],
  );
  const showThinking = useMemo(
    () => Boolean((thinking || '').trim()) || Boolean((thoughtChain || []).some((item) => item.key === 'summary' && item.status === 'loading')),
    [thinking, thoughtChain],
  );
  const blocks = useMemo(() => {
    if (turn && turn.blocks.length > 0) {
      return normalizeTurnBlocks(getTurnBlocksForDisplay(turn, displayMode, reducedMotion));
    }
    return normalizeAssistantMessage({
      content,
      thinking,
      showThinking,
      rawEvidence: displayMode === 'debug' ? rawEvidence : undefined,
      recommendations,
      isStreaming,
    });
  }, [content, displayMode, isStreaming, rawEvidence, recommendations, reducedMotion, showThinking, thinking, turn]);

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
            defaultExpandedKeys={defaultExpandedKeys}
          />
        </div>
      )}
      {blocks.length > 0 ? (
        <AssistantMessageBlocks
          blocks={blocks}
          onRecommendationSelect={onRecommendationSelect}
          onApprovalDecision={onApprovalDecision}
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

interface ConversationState {
  conversations: ConversationItem[];
  activeKey: string;
}

type ConversationAction =
  | { type: 'reset' }
  | { type: 'restore'; conversation: ConversationItem }
  | { type: 'set_active'; key: string }
  | { type: 'new'; key: string }
  | { type: 'append_messages'; key: string; label?: string; messages: ExtendedChatMessage[] }
  | { type: 'update_message'; key: string; messageId: string; updater: (message: ExtendedChatMessage) => ExtendedChatMessage }
  | { type: 'remove_message'; key: string; messageId: string };

const DEFAULT_CONVERSATION: ConversationItem = { key: 'default', label: '新对话', group: '今天', messages: [] };

function conversationReducer(state: ConversationState, action: ConversationAction): ConversationState {
  switch (action.type) {
    case 'reset':
      return {
        conversations: [DEFAULT_CONVERSATION],
        activeKey: DEFAULT_CONVERSATION.key,
      };
    case 'restore':
      return {
        conversations: [action.conversation],
        activeKey: action.conversation.key,
      };
    case 'set_active':
      return {
        ...state,
        activeKey: action.key,
      };
    case 'new':
      return {
        conversations: [
          { key: action.key, label: '新对话', group: '今天', messages: [] },
          ...state.conversations,
        ],
        activeKey: action.key,
      };
    case 'append_messages':
      return {
        ...state,
        conversations: state.conversations.map((conversation) => (
          conversation.key !== action.key
            ? conversation
            : {
                ...conversation,
                label: action.label || conversation.label,
                messages: [...conversation.messages, ...action.messages],
              }
        )),
      };
    case 'update_message':
      return {
        ...state,
        conversations: state.conversations.map((conversation) => {
          if (conversation.key !== action.key) {
            return conversation;
          }
          return {
            ...conversation,
            messages: conversation.messages.map((message) => (
              message.id === action.messageId ? action.updater(message) : message
            )),
          };
        }),
      };
    case 'remove_message':
      return {
        ...state,
        conversations: state.conversations.map((conversation) => (
          conversation.key !== action.key
            ? conversation
            : {
                ...conversation,
                messages: conversation.messages.filter((message) => message.id !== action.messageId),
              }
        )),
      };
    default:
      return state;
  }
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
  const [conversationState, dispatch] = useReducer(conversationReducer, {
    conversations: [DEFAULT_CONVERSATION],
    activeKey: DEFAULT_CONVERSATION.key,
  });
  const conversations = conversationState.conversations;
  const activeKey = conversationState.activeKey;
  const [displayMode, setDisplayMode] = useState<DisplayMode>(() => {
    const stored = localStorage.getItem('ai.drawer.display_mode');
    return stored === 'debug' ? 'debug' : 'normal';
  });
  const [reducedMotion, setReducedMotion] = useState(false);
  const [isNearBottom, setIsNearBottom] = useState(true);
  const [showJumpToLatest, setShowJumpToLatest] = useState(false);
  const [liveAnnouncement, setLiveAnnouncement] = useState('');

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
    dispatch({ type: 'restore', conversation: restoredItem });
    setSessionId(restored.id);
  }, []);

  useEffect(() => {
    dispatch({ type: 'reset' });
    setSessionId(undefined);
    setIsLoading(false);
  }, [scene]);

  useEffect(() => {
    localStorage.setItem('ai.drawer.display_mode', displayMode);
  }, [displayMode]);

  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) {
      return undefined;
    }
    const media = window.matchMedia('(prefers-reduced-motion: reduce)');
    const update = () => setReducedMotion(media.matches);
    update();
    media.addEventListener?.('change', update);
    return () => media.removeEventListener?.('change', update);
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

  const patchAssistantMessage = useCallback((
    conversationKey: string,
    assistantId: string,
    updater: (message: ExtendedChatMessage) => ExtendedChatMessage,
  ) => {
    dispatch({ type: 'update_message', key: conversationKey, messageId: assistantId, updater });
  }, []);

  const createStreamHandlers = useCallback((
    conversationKey: string,
    assistantId: string,
  ) => {
    let assistantContent = '';
    let assistantThinking = '';
    let assistantRecommendations: EmbeddedRecommendation[] | undefined;
    let assistantTraceId: string | undefined;

    const refreshAnnouncement = (value: string) => {
      if (!value.trim()) {
        return;
      }
      setLiveAnnouncement(value.trim());
    };

    const syncMessageFromBuffers = (
      message: ExtendedChatMessage,
      overrides: Partial<ExtendedChatMessage> = {},
    ): ExtendedChatMessage => ({
      ...message,
      ...overrides,
      content: overrides.content !== undefined ? overrides.content : (message.content || assistantContent),
      thinking: overrides.thinking !== undefined ? overrides.thinking : (message.thinking || assistantThinking || undefined),
      recommendations: overrides.recommendations ?? message.recommendations ?? assistantRecommendations,
      traceId: overrides.traceId ?? assistantTraceId ?? message.traceId,
    });

    return {
      onMeta: (data: { sessionId?: string; traceId?: string; turn_id?: string }) => {
        if (data.sessionId) {
          setSessionId(data.sessionId);
        }
        if (data.traceId) {
          assistantTraceId = String(data.traceId);
        }
        if (data.turn_id) {
          patchAssistantMessage(conversationKey, assistantId, (message) => {
            const nextTurn = applyTurnStarted(message.turn, { turn_id: data.turn_id!, phase: 'rewrite', status: 'streaming' }, assistantTraceId);
            const summary = projectTurnSummary(nextTurn);
            return syncMessageFromBuffers(message, {
              turn: nextTurn,
              content: summary.content || message.content,
              thinking: summary.thinking || message.thinking,
              rawEvidence: summary.rawEvidence || message.rawEvidence,
              recommendations: summary.recommendations || message.recommendations,
            });
          });
        }
      },
      onTurnStarted: (data: { turn_id: string; phase?: string; status?: string; role?: string }) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          const nextTurn = applyTurnStarted(message.turn, data, assistantTraceId || message.traceId);
          const summary = projectTurnSummary(nextTurn);
          refreshAnnouncement(resolveThoughtStageTitle(nextTurn.phase));
            return syncMessageFromBuffers(message, {
              turn: nextTurn,
              content: summary.content || message.content,
              thinking: summary.thinking || message.thinking,
              rawEvidence: summary.rawEvidence,
              recommendations: summary.recommendations,
            });
        });
      },
      onBlockOpen: (data: { turn_id: string; block_id: string; block_type: string; position?: number; status?: string; title?: string; payload?: Record<string, unknown> }) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          const nextTurn = applyBlockOpen(message.turn, data);
          const summary = projectTurnSummary(nextTurn);
          return syncMessageFromBuffers(message, {
            turn: nextTurn,
            content: summary.content || message.content,
            thinking: summary.thinking || message.thinking,
            rawEvidence: summary.rawEvidence,
            recommendations: summary.recommendations,
          });
        });
      },
      onBlockDelta: (data: { turn_id: string; block_id: string; block_type?: string; patch?: Record<string, unknown> }) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          const nextTurn = applyBlockDelta(message.turn, data);
          const summary = projectTurnSummary(nextTurn);
          return syncMessageFromBuffers(message, {
            turn: nextTurn,
            content: summary.content || message.content,
            thinking: summary.thinking || message.thinking,
            rawEvidence: summary.rawEvidence,
            recommendations: summary.recommendations,
          });
        });
      },
      onBlockReplace: (data: { turn_id: string; block_id: string; block_type?: string; payload?: Record<string, unknown> }) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          const nextTurn = applyBlockReplace(message.turn, data);
          const summary = projectTurnSummary(nextTurn);
          return syncMessageFromBuffers(message, {
            turn: nextTurn,
            content: summary.content || message.content,
            thinking: summary.thinking || message.thinking,
            rawEvidence: summary.rawEvidence,
            recommendations: summary.recommendations,
          });
        });
      },
      onBlockClose: (data: { turn_id: string; block_id: string; status?: string }) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          const nextTurn = applyBlockClose(message.turn, data);
          const summary = projectTurnSummary(nextTurn);
          return syncMessageFromBuffers(message, {
            turn: nextTurn,
            content: summary.content || message.content,
            thinking: summary.thinking || message.thinking,
            rawEvidence: summary.rawEvidence,
            recommendations: summary.recommendations,
          });
        });
      },
      onTurnState: (data: { turn_id: string; status?: string; phase?: string }) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          let nextThoughtChain = message.thoughtChain || [];
          if (data.phase === 'summary') {
            nextThoughtChain = finalizeThoughtStage(
              nextThoughtChain,
              'execute',
              'success',
              buildStageDescription('execute', 'success'),
            );
          }
          return {
            ...message,
            turn: applyTurnState(message.turn, data),
            thoughtChain: nextThoughtChain,
          };
        });
        refreshAnnouncement(`${resolveThoughtStageTitle(data.phase)} ${data.status || ''}`);
      },
      onTurnDone: (data: { turn_id: string; status?: string; phase?: string }) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          const nextTurn = applyTurnDone(message.turn, data);
          const summary = projectTurnSummary(nextTurn);
          return syncMessageFromBuffers(message, {
            turn: nextTurn,
            content: summary.content || message.content,
            thinking: summary.thinking || message.thinking,
            rawEvidence: summary.rawEvidence,
            recommendations: summary.recommendations,
          });
        });
      },
      onStageDelta: (data: SSEStageDeltaEvent) => {
        const stageKey = String(data.stage || '').trim() as ThoughtStageItem['key'];
        if (!stageKey) {
          return;
        }
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          const currentStage = (message.thoughtChain || []).find((item) => item.key === stageKey);
          const status = normalizeThoughtStatus(data.status as string | undefined, 'loading');
          return syncMessageFromBuffers(message, {
            thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
              key: stageKey,
              title: resolveThoughtStageTitle(stageKey),
              status,
              description: buildStageDescription(stageKey, status, data as unknown as Record<string, unknown>) || currentStage?.description,
              content: appendStageContent(
                data.replace ? '' : currentStage?.content,
                buildStageMilestone(stageKey, 'delta', data as unknown as Record<string, unknown>),
              ),
            }),
          });
        });
      },
      onRewriteResult: (data: Record<string, unknown>) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => ({
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
        }));
      },
      onPlannerState: (data: Record<string, unknown>) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => ({
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
        }));
      },
      onPlanCreated: (data: Record<string, unknown>) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => ({
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
        }));
      },
      onStepUpdate: (data: Record<string, unknown>) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          const nextStatus = normalizeThoughtStatus(data.status as string | undefined, 'loading');
          const nextStage = upsertThoughtStage(message.thoughtChain || [], {
            key: 'execute',
            title: '调用专家执行',
            status: nextStatus,
            description: buildStageDescription('execute', nextStatus, data),
            content: appendStageContent(
              (message.thoughtChain || []).find((item) => item.key === 'execute')?.content,
              buildStageMilestone('execute', 'event', data),
            ),
          });
          return {
            ...message,
            thoughtChain: upsertThoughtDetail(nextStage, 'execute', {
              id: buildStepDetailID(data),
              label: String(data.title || '当前步骤'),
              status: nextStatus,
              content: buildExecutionDetailContent('step', data),
            }),
          };
        });
      },
      onToolCall: (data: Record<string, unknown>) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          const detailID = buildToolDetailID(data);
          const nextStages = upsertThoughtStage(message.thoughtChain || [], {
            key: 'execute',
            title: '调用专家执行',
            status: 'loading',
            description: buildStageDescription('execute', 'loading', data),
            content: appendStageContent(
              (message.thoughtChain || []).find((item) => item.key === 'execute')?.content,
              String(data.summary || data.user_visible_summary || ''),
            ),
          });
          return {
            ...message,
            thoughtChain: upsertThoughtDetail(nextStages, 'execute', {
              id: detailID,
              label: String(data.tool_name || data.expert || 'tool'),
              status: 'loading',
              content: buildExecutionDetailContent('tool_call', data),
            }),
          };
        });
      },
      onToolResult: (data: Record<string, unknown>) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          const detailID = buildToolDetailID(data);
          const result = data.result as Record<string, unknown> | undefined;
          const status = data.status === 'error' || result?.ok === false ? 'error' : 'success';
          return {
            ...message,
            thoughtChain: upsertThoughtDetail(message.thoughtChain || [], 'execute', {
              id: detailID,
              label: String(data.tool_name || data.expert || 'tool'),
              status,
              content: buildExecutionDetailContent('tool_result', data),
            }),
          };
        });
      },
      onDelta: (data: { contentChunk: string }) => {
        assistantContent += data.contentChunk || '';
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          content: assistantContent,
          thoughtChain: finalizeThoughtStage(
            finalizeThoughtStage(
              message.thoughtChain || [],
              'execute',
              'success',
              buildStageDescription('execute', 'success'),
            ),
            'summary',
            'loading',
          ),
        }));
      },
      onThinkingDelta: (data: { contentChunk: string }) => {
        assistantThinking += data.contentChunk || '';
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          thinking: assistantThinking,
          thoughtChain: upsertThoughtStage(finalizeThoughtStage(
            message.thoughtChain || [],
            'execute',
            'success',
            buildStageDescription('execute', 'success'),
          ), {
            key: 'summary',
            title: '整理最终回答',
            status: 'loading',
            description: '正在思考并整理最终回答',
            content: assistantThinking,
          }),
        }));
      },
      onApprovalRequired: (data: ApprovalTicket & {
        turn_id?: string;
        approval_required?: boolean;
        previewDiff?: string;
        title?: string;
        user_visible_summary?: string;
      }) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          const turnID = String(data.turn_id || message.turn?.id || `pending-${assistantId}`);
          const nextTurn = applyTurnState(
            applyBlockReplace(message.turn, {
              turn_id: turnID,
              block_id: buildApprovalBlockID(data as unknown as Record<string, unknown>),
              block_type: 'approval',
              payload: {
                ...data,
                status: 'waiting_user',
                title: String(data.title || '等待你确认'),
                summary: String(data.user_visible_summary || data.title || '当前步骤需要确认后继续执行'),
              },
            }),
            {
              turn_id: turnID,
              status: 'waiting_user',
              phase: 'execute',
            },
          );
          const summary = projectTurnSummary(nextTurn);
          return syncMessageFromBuffers(message, {
            turn: nextTurn,
            content: summary.content || message.content,
            thinking: summary.thinking || message.thinking,
            rawEvidence: summary.rawEvidence || message.rawEvidence,
            recommendations: summary.recommendations || message.recommendations,
            thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
              key: 'user_action',
              title: '等待你确认',
              status: 'loading',
              description: String(data.title || '当前步骤需要确认后继续执行'),
              content: String(data.user_visible_summary || ''),
            }),
          });
        });
        refreshAnnouncement('等待确认');
      },
      onClarifyRequired: (data: Record<string, unknown>) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => ({
          ...message,
          thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
            key: 'user_action',
            title: '等待你补充信息',
            status: 'loading',
            description: String(data.message || data.title || '当前目标仍有歧义'),
          }),
        }));
        assistantContent ||= String(data.message || '');
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          content: assistantContent,
        }));
      },
      onSummary: () => {
        patchAssistantMessage(conversationKey, assistantId, (message) => ({
          ...message,
          thoughtChain: finalizeThoughtStage(
            finalizeThoughtStage(
              (message.thoughtChain || []).filter((item) => item.key !== 'summary'),
              'execute',
              'success',
              buildStageDescription('execute', 'success'),
            ),
            'user_action',
            'success',
          ),
        }));
      },
      onDone: (data: SSEDoneEvent) => {
        if (data.turn_recommendations) {
          assistantRecommendations = data.turn_recommendations as EmbeddedRecommendation[];
        }
        if (data.session) {
          const sessionData = data.session as unknown as Record<string, unknown>;
          const finalAssistant = getLastAssistantMessage(sessionData);
          assistantContent ||= String(finalAssistant?.content || '');
          if (!assistantThinking) {
            assistantThinking = String(finalAssistant?.thinking || '');
          }
          assistantRecommendations ||= finalAssistant?.recommendations as EmbeddedRecommendation[] | undefined;
          if (sessionData.id) {
            setSessionId(String(sessionData.id));
          }
        }
        patchAssistantMessage(conversationKey, assistantId, (message) => ({
          ...syncMessageFromBuffers(message, {
          content: message.turn ? projectTurnSummary(message.turn).content || assistantContent : assistantContent,
          thinking: message.turn ? projectTurnSummary(message.turn).thinking || assistantThinking || undefined : assistantThinking || undefined,
          recommendations: assistantRecommendations || message.recommendations,
          traceId: assistantTraceId || message.traceId,
          }),
          thoughtChain: (message.thoughtChain || []).map((item) => ({
            ...item,
            blink: false,
            status: item.key === 'user_action' && message.turn?.status === 'waiting_user'
              ? item.status
              : item.status === 'loading' ? 'success' : item.status,
          })),
        }));
        setIsLoading(false);
      },
      onError: (data: { message?: string; stage?: string }) => {
        assistantContent ||= String(data.message || '当前 AI 阶段执行失败，请稍后重试。').trim();
        patchAssistantMessage(conversationKey, assistantId, (message) => {
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
            turn: message.turn ? applyTurnState(message.turn, { turn_id: message.turn.id, status: 'error', phase: message.turn.phase }) : message.turn,
          };
        });
        setIsLoading(false);
      },
    };
  }, [patchAssistantMessage]);

  const streamAssistantReply = useCallback(async (
    conversationKey: string,
    assistantId: string,
    messageText: string,
  ) => {
    abortControllerRef.current = new AbortController();
    try {
      await aiApi.chatStream(
        {
          sessionId,
          message: messageText,
          context: { scene },
        },
        createStreamHandlers(conversationKey, assistantId),
        abortControllerRef.current.signal,
      );
    } catch (error) {
      if ((error as Error).name !== 'AbortError') {
        message.error('请求失败，请稍后重试');
      }
      setIsLoading(false);
    }
  }, [createStreamHandlers, scene, sessionId]);

  // 发送消息
  const handleSubmit = useCallback(async (val: string) => {
    if (!val.trim() || isLoading) return;
    const trimmed = val.trim();
    const conversationKey = activeKey;

    // 添加用户消息
    const userMessage: ExtendedChatMessage = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: trimmed,
      createdAt: new Date().toISOString(),
    };
    const newLabel = activeConversation?.label === '新对话'
      ? trimmed.slice(0, 20) + (trimmed.length > 20 ? '...' : '')
      : activeConversation?.label;

    setIsLoading(true);

    // 创建助手消息占位
    const assistantId = `assistant-${Date.now()}`;
    dispatch({
      type: 'append_messages',
      key: conversationKey,
      label: newLabel,
      messages: [
        userMessage,
        {
          id: assistantId,
          role: 'assistant',
          content: '',
          thoughtChain: [],
          turn: createAssistantTurn(`pending-${assistantId}`, { status: 'streaming', phase: 'rewrite' }),
          createdAt: new Date().toISOString(),
        },
      ],
    });
    await streamAssistantReply(conversationKey, assistantId, trimmed);
  }, [activeConversation?.label, activeKey, isLoading, streamAssistantReply]);

  // 中止请求
  const handleAbort = useCallback(() => {
    abortControllerRef.current?.abort();
    setIsLoading(false);
  }, []);

  const handleApprovalDecision = useCallback(async (
    assistantId: string,
    payload: Record<string, unknown>,
    approved: boolean,
  ) => {
    setIsLoading(true);
    const turnID = String(payload.turn_id || '');
    const approvalBlockID = buildApprovalBlockID(payload);
    patchAssistantMessage(activeKey, assistantId, (message) => {
      if (!message.turn) {
        return message;
      }
      const submittingTurn = applyBlockReplace(message.turn, {
        turn_id: turnID || message.turn.id,
        block_id: approvalBlockID,
        block_type: 'approval',
        payload: buildApprovalStatusPayload(
          payload,
          String(payload.title || '正在提交'),
          approved ? '正在提交确认，马上继续执行。' : '正在提交取消请求。',
          'submitting',
        ),
      });
      return {
        ...message,
        turn: applyTurnState(submittingTurn, {
          turn_id: submittingTurn.id,
          status: approved ? 'streaming' : 'completed',
          phase: 'execute',
        }),
        thoughtChain: finalizeThoughtStage(message.thoughtChain || [], 'user_action', approved ? 'success' : 'abort', approved ? '已确认，继续执行' : '已取消执行'),
      };
    });

    try {
      if (!approved) {
        await aiApi.respondApproval({
          session_id: String(payload.session_id || sessionId || ''),
          plan_id: payload.plan_id ? String(payload.plan_id) : undefined,
          step_id: payload.step_id ? String(payload.step_id) : undefined,
          approved: false,
        });
        patchAssistantMessage(activeKey, assistantId, (message) => ({
          ...message,
          content: message.content || '已取消该操作。',
          turn: message.turn ? applyTurnDone(
            applyBlockReplace(message.turn, {
              turn_id: message.turn.id,
              block_id: approvalBlockID,
              block_type: 'status',
              payload: {
                title: '审批已取消',
                summary: '已取消执行，当前变更不会继续推进。',
                status: 'cancelled',
              },
            }),
            { turn_id: message.turn.id, status: 'completed', phase: 'done' },
          ) : message.turn,
        }));
        setIsLoading(false);
        return;
      }

      patchAssistantMessage(activeKey, assistantId, (message) => ({
        ...message,
        turn: message.turn ? applyBlockReplace(message.turn, {
          turn_id: message.turn.id,
          block_id: approvalBlockID,
          block_type: 'status',
          payload: {
            title: '已确认，继续执行',
            summary: '审批已通过，正在继续执行当前步骤。',
            status: 'approved',
          },
        }) : message.turn,
      }));

      await aiApi.respondApprovalStream(
        {
          session_id: String(payload.session_id || sessionId || ''),
          plan_id: payload.plan_id ? String(payload.plan_id) : undefined,
          step_id: payload.step_id ? String(payload.step_id) : undefined,
          approved: true,
        },
        createStreamHandlers(activeKey, assistantId),
      );
    } catch {
      patchAssistantMessage(activeKey, assistantId, (message) => ({
        ...message,
        turn: message.turn ? applyBlockReplace(message.turn, {
          turn_id: message.turn.id,
          block_id: approvalBlockID,
          block_type: 'approval',
          payload: buildApprovalStatusPayload(
            payload,
            String(payload.title || '等待你确认'),
            String(payload.user_visible_summary || payload.title || '当前步骤需要确认后继续执行'),
            'failed',
            '审批结果提交失败，请重试。',
          ),
        }) : message.turn,
        thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
          key: 'user_action',
          title: '等待你确认',
          status: 'error',
          description: '审批结果提交失败，请重试。',
        }),
      }));
      message.error('确认操作失败');
      setIsLoading(false);
    }
  }, [activeKey, createStreamHandlers, patchAssistantMessage, sessionId]);

  // 新建会话
  const handleNewConversation = useCallback(() => {
    const timeNow = dayjs().valueOf().toString();
    dispatch({ type: 'new', key: timeNow });
    setSessionId(undefined);
  }, []);

  useEffect(() => {
    const scrollBox = listRef.current?.scrollBoxNativeElement;
    if (!scrollBox) {
      return undefined;
    }
    const handleScroll = () => {
      const nearBottom = scrollBox.scrollHeight - scrollBox.scrollTop - scrollBox.clientHeight < 72;
      setIsNearBottom(nearBottom);
      setShowJumpToLatest(!nearBottom && isLoading);
    };
    handleScroll();
    scrollBox.addEventListener('scroll', handleScroll, { passive: true });
    return () => scrollBox.removeEventListener('scroll', handleScroll);
  }, [isLoading, messages.length]);

  useEffect(() => {
    if (!messages.length || !isLoading || !isNearBottom) {
      return;
    }
    listRef.current?.scrollTo({
      top: 'bottom',
      behavior: reducedMotion ? 'auto' : 'smooth',
    });
  }, [conversations, isLoading, isNearBottom, messages.length, reducedMotion]);

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

    setIsLoading(true);
    patchAssistantMessage(activeKey, assistantMsgId, (message) => ({
      ...message,
      content: '',
      thinking: undefined,
      recommendations: undefined,
      rawEvidence: undefined,
      thoughtChain: [],
      traceId: undefined,
      turn: createAssistantTurn(`pending-${assistantMsgId}`, { status: 'streaming', phase: 'rewrite' }),
      restored: false,
    }));

    await streamAssistantReply(activeKey, assistantMsgId, userMessage.content);
  }, [activeKey, isLoading, messages, patchAssistantMessage, streamAssistantReply]);

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
    const hasVisibleAssistantState = Boolean(msg.content || msg.thinking || hasThoughtChain || (msg.turn && msg.turn.blocks.length > 0));
    const isStreaming = isCurrentStreaming && !hasVisibleAssistantState;

    // 只有当消息内容正在生成时（内容为空）才显示 loading
    // 如果消息已经有内容了（即使正在生成推荐），重新生成按钮不显示 loading
    const showLoading = isLoading && isCurrentStreaming && !msg.content && !hasThoughtChain;

    return (
      <AssistantMessage
        content={msg.content}
        thinking={msg.thinking}
        turn={msg.turn}
        recommendations={msg.recommendations}
        thoughtChain={msg.thoughtChain}
        rawEvidence={msg.rawEvidence}
        restored={msg.restored}
        displayMode={displayMode}
        reducedMotion={reducedMotion}
        isStreaming={isStreaming || (isCurrentStreaming && !!msg.thinking && !msg.content)}
        onRegenerate={() => handleRegenerate(msg.id)}
        onRecommendationSelect={handleRecommendationSelect}
        onApprovalDecision={(payload, approved) => handleApprovalDecision(msg.id, payload, approved)}
        isLoading={showLoading}
      />
    );
  }, [displayMode, handleApprovalDecision, handleRegenerate, handleRecommendationSelect, isLoading, reducedMotion]);

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
          <Segmented
            size="small"
            aria-label="AI 展示模式"
            value={displayMode}
            onChange={(value) => setDisplayMode(value as DisplayMode)}
            options={[
              { label: '普通', value: 'normal' },
              { label: '调试', value: 'debug' },
            ]}
          />
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
                onActiveChange={(key) => dispatch({ type: 'set_active', key })}
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
          <div style={{ position: 'relative', height: '100%' }}>
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
                  && !(m.thoughtChain && m.thoughtChain.length > 0)
                  && !(m.turn && m.turn.blocks.length > 0),
              }))}
              role={role}
            />
            {showJumpToLatest && (
              <div
                style={{
                  position: 'absolute',
                  right: 16,
                  bottom: 18,
                  display: 'flex',
                  justifyContent: 'flex-end',
                }}
              >
                <button
                  type="button"
                  aria-label="跳转到最新消息"
                  onClick={() => {
                    listRef.current?.scrollTo({ top: 'bottom', behavior: reducedMotion ? 'auto' : 'smooth' });
                    setShowJumpToLatest(false);
                    setIsNearBottom(true);
                  }}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: 8,
                    padding: '8px 12px',
                    borderRadius: 18,
                    border: `1px solid ${token.colorBorderSecondary}`,
                    background: token.colorBgElevated,
                    color: token.colorText,
                    boxShadow: token.boxShadowSecondary,
                    backdropFilter: 'blur(12px)',
                    cursor: 'pointer',
                    fontSize: 12,
                    lineHeight: 1,
                  }}
                >
                  <span
                    style={{
                      width: 18,
                      height: 18,
                      borderRadius: 999,
                      display: 'inline-flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      background: token.colorPrimaryBg,
                      color: token.colorPrimary,
                      flexShrink: 0,
                    }}
                  >
                    <DownOutlined style={{ fontSize: 10 }} />
                  </span>
                  <span style={{ whiteSpace: 'nowrap' }}>有新消息</span>
                </button>
              </div>
            )}
          </div>
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
      <div
        aria-live="polite"
        aria-atomic="true"
        style={{
          position: 'absolute',
          width: 1,
          height: 1,
          padding: 0,
          margin: -1,
          overflow: 'hidden',
          clip: 'rect(0, 0, 0, 0)',
          whiteSpace: 'nowrap',
          border: 0,
        }}
      >
        {liveAnnouncement}
      </div>
    </div>
  );
};

export default Copilot;
