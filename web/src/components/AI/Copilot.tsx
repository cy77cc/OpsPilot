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
import { XMarkdown } from '@ant-design/x-markdown';
import type { BubbleListRef } from '@ant-design/x/es/bubble';
import { Button, message, Popover, Segmented, Select, Space, Tooltip, theme, Skeleton } from 'antd';
import dayjs from 'dayjs';
import { aiApi } from '../../api/modules/ai';
import type {
  ApprovalRequiredEvent,
  SSEChainNodeEvent,
  SSEChainStartedEvent,
  SSEDoneEvent,
  SSEFinalAnswerEvent,
  SSEPhaseCompleteEvent,
  SSEPhaseStartedEvent,
  SSEPlanGeneratedEvent,
  SSEReplanTriggeredEvent,
  SSEStageDeltaEvent,
  SSEStepCompleteEvent,
  SSEStepStartedEvent,
  SSEStepUpdateEvent,
} from '../../api/modules/ai';
import { getSceneLabel } from './constants/sceneMapping';
import type { ChatMessage, ChatTurn, ConfirmationRequest, EmbeddedRecommendation, ThoughtChainRuntimeState, ThoughtStageDetailItem, ThoughtStageItem, ThoughtStageStatus } from './types';
import type { SceneOption } from './hooks/useAutoScene';
import {
  loadRestoredConversationDetail,
  useConversationRestore,
  type RestoredConversation,
  type RestoredConversationState,
} from './hooks/useConversationRestore';
import { useScenePrompts } from './hooks/useScenePrompts';
import { MessageActions } from './components/MessageActions';
import { ConfirmationPanel } from './components/ConfirmationPanel';
import FinalAnswerStream from './components/FinalAnswerStream';
import { RecommendedActions } from './components/RecommendedActions';
import RuntimeThoughtChain from './components/RuntimeThoughtChain';
import { ToolCard } from './components/ToolCard';
import AssistantMessageBlocks from './components/AssistantMessageBlocks';
import { mergeAssistantBlocks, normalizeAssistantMessage, normalizeTurnBlocks } from './messageBlocks';
import { createThoughtChainRuntimeState, reduceThoughtChainRuntimeEvent } from './thoughtChainRuntime';
import {
  applyBlockClose,
  applyBlockDelta,
  applyBlockOpen,
  applyBlockReplace,
  applyPhaseComplete,
  applyPhaseStarted,
  applyPlanGenerated,
  applyReplanTriggered,
  applyStepComplete,
  applyStepStarted,
  applyTurnDone,
  applyTurnStarted,
  applyTurnState,
  createAssistantTurn,
  getTurnBlocksForDisplay,
  projectTurnSummary,
  type DisplayMode,
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
      return status === 'success' ? '已确认目标对象和约束' : '正在识别目标对象和执行边界';
    case 'plan':
      return status === 'success' ? '已生成可执行步骤' : '正在整理执行步骤与影响范围';
    case 'execute':
      if (status === 'error') {
        return '工具调用出现异常';
      }
      return status === 'success' ? '工具调用已完成' : '正在调用工具处理当前步骤';
    case 'summary':
      return status === 'success' ? '已生成最终结论' : '正在整理最终结论';
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
        return '正在识别目标对象与约束条件。';
      case 'plan':
        return '正在整理执行步骤与所需上下文。';
      default:
        return undefined;
    }
  }
  switch (stage) {
    case 'rewrite':
      return '已识别目标对象，准备进入规划。';
    case 'plan':
      return '已形成执行步骤。';
    case 'execute':
      return typeof source?.summary === 'string' && source.summary.trim()
        ? source.summary.trim()
        : '正在调用工具处理当前步骤。';
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
  const order: Record<ThoughtStageItem['key'], number> = {
    rewrite: 10,
    plan: 20,
    user_action: 30,
    execute: 40,
    summary: 50,
  };
  return [...(stages || [])]
    .filter((item) => item.key !== 'summary')
    .sort((a, b) => (order[a.key] || 99) - (order[b.key] || 99));
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

function buildToolDetailID(data: Record<string, unknown>): string {
  const callID = String(data.call_id || '').trim();
  if (callID) {
    return callID;
  }
  const stepID = String(data.step_id || '').trim();
  const toolName = String(data.tool_name || data.tool || data.expert || 'tool').trim();
  return ['tool', stepID || 'step', toolName || 'tool'].join(':');
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
    content: nextContent,
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
    content: nextContent,
    blink: patch.blink ?? next.status === 'loading',
  };
  return stages.map((item, itemIndex) => (itemIndex === index ? merged : item));
}


function buildExecutionStageSummary(data: Record<string, unknown>): string | undefined {
  const title = String(data.title || data.summary || data.user_visible_summary || '').trim();
  const toolName = String(data.tool_name || data.tool || data.expert || '').trim();
  const error = String(data.error || '').trim();
  if (error) {
    return error;
  }
  if (title) {
    return `当前步骤: ${title}`;
  }
  if (toolName) {
    return `正在调用工具: ${toolName}`;
  }
  return undefined;
}

function resolveDefaultExpandedThoughtKeys(
  stages: ThoughtStageItem[],
  opts: { restored?: boolean; streaming?: boolean },
): string[] {
  if (opts.restored) {
    return [];
  }
  if (!opts.streaming) {
    return stages.map((item) => item.key);
  }
  const active = stages.filter((item) => item.status === 'loading').map((item) => item.key);
  if (active.length > 0) {
    return [active[active.length - 1]];
  }
  return stages.map((item) => item.key);
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
      return '识别目标与约束';
    case 'planning':
    case 'plan':
      return '整理执行步骤';
    case 'replanning':
      return '动态调整计划';
    case 'execute':
      return '工具调用链';
    case 'summary':
      return '整理最终结论';
    default:
      return '处理中';
  }
}

// 助手消息渲染组件
const AssistantMessage: React.FC<{
  content: string;
  thinking?: string;
  confirmation?: ConfirmationRequest;
  recommendations?: EmbeddedRecommendation[];
  thoughtChain?: ThoughtStageItem[];
  turn?: ChatTurn;
  runtime?: ThoughtChainRuntimeState;
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
  confirmation,
  recommendations,
  thoughtChain,
  turn,
  runtime,
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
  const isRestoredAssistant = Boolean(restored);
  const turnBlocks = useMemo(
    () => normalizeTurnBlocks(getTurnBlocksForDisplay(turn, displayMode, reducedMotion)),
    [turn, displayMode, reducedMotion],
  );
  const fallbackBlocks = useMemo(
    () => normalizeAssistantMessage({
      content,
      thinking,
      showThinking: !isRestoredAssistant,
      rawEvidence: displayMode === 'debug' ? rawEvidence : undefined,
      recommendations,
      isStreaming,
    }),
    [content, thinking, isRestoredAssistant, displayMode, rawEvidence, recommendations, isStreaming],
  );
  const assistantBlocks = useMemo(
    () => mergeAssistantBlocks(turnBlocks, fallbackBlocks),
    [turnBlocks, fallbackBlocks],
  );
  const hasApprovalBlock = assistantBlocks.some((block) => block.type === 'approval');
  const chainItems = useMemo(
    () => (isRestoredAssistant ? [] : visibleThoughtChain(thoughtChain)),
    [isRestoredAssistant, thoughtChain],
  );
  const defaultExpandedKeys = useMemo(
    () => resolveDefaultExpandedThoughtKeys(chainItems, { restored, streaming: Boolean(isStreaming) }),
    [chainItems, isStreaming, restored],
  );
  const showThinking = useMemo(
    () => !isRestoredAssistant && (Boolean((thinking || '').trim()) || Boolean((thoughtChain || []).some((item) => item.key === 'summary' && item.status === 'loading'))),
    [isRestoredAssistant, thinking, thoughtChain],
  );
  const effectiveConfirmation = useMemo(() => {
    if (!confirmation) {
      return null;
    }
    if (!onApprovalDecision) {
      return confirmation;
    }
    return {
      ...confirmation,
      onConfirm: () => onApprovalDecision(confirmation.details || {}, true),
      onCancel: () => onApprovalDecision(confirmation.details || {}, false),
    };
  }, [confirmation, onApprovalDecision]);
  const hasRuntimeChain = Boolean(runtime && (runtime.nodes.length > 0 || runtime.finalAnswer.visible || runtime.finalAnswer.content));
  const useTurnBlockRendering = Boolean(turn) && assistantBlocks.length > 0;

  return (
    <div>
      {hasRuntimeChain ? (
        <>
          {runtime?.nodes.length ? (
            <RuntimeThoughtChain
              nodes={runtime.nodes}
              isCollapsed={runtime.isCollapsed}
              onApprovalDecision={onApprovalDecision}
            />
          ) : null}
          <FinalAnswerStream
            content={runtime?.finalAnswer.content || content}
            visible={Boolean(runtime?.finalAnswer.visible)}
            streaming={runtime?.finalAnswer.streaming}
            reducedMotion={reducedMotion}
          />
          {showActions && !isStreaming && (runtime?.finalAnswer.content || content) ? (
            <MessageActions
              content={runtime?.finalAnswer.content || content}
              messageId=""
              isLoading={isLoading}
              onRegenerate={onRegenerate}
            />
          ) : null}
        </>
      ) : useTurnBlockRendering ? (
        <>
          <AssistantMessageBlocks
            blocks={assistantBlocks}
            onRecommendationSelect={onRecommendationSelect}
            onApprovalDecision={onApprovalDecision}
          />
          {effectiveConfirmation && !hasApprovalBlock ? (
            <ConfirmationPanel confirmation={effectiveConfirmation} />
          ) : null}
          {showActions && !isStreaming && content && (
            <MessageActions
              content={content}
              messageId=""
              isLoading={isLoading}
              onRegenerate={onRegenerate}
            />
          )}
        </>
      ) : isRestoredAssistant ? (
        <>
          {content ? (
            <div className="ai-markdown-content">
              <XMarkdown>{content}</XMarkdown>
            </div>
          ) : null}
          {showActions && !isStreaming && content && (
            <MessageActions
              content={content}
              messageId=""
              isLoading={isLoading}
              onRegenerate={onRegenerate}
            />
          )}
        </>
      ) : (
        <>
          {chainItems.length > 0 && (
            <div style={{ marginBottom: 12 }}>
              <ThoughtChain
                items={chainItems.map((item) => ({
                  key: item.key,
                  title: item.title,
                  description: item.description,
                  content: (
                    <div>
                      {item.content ? (
                        <div style={{ marginBottom: item.details?.length ? 8 : 0, whiteSpace: 'pre-wrap' }}>
                          {item.content}
                        </div>
                      ) : null}
                      {(item.details || []).map((detail) => {
                        if (detail.kind === 'tool' || detail.tool || detail.params || detail.result) {
                          return (
                            <ToolCard
                              key={detail.id}
                              tool={{
                                id: detail.id,
                                name: detail.tool || detail.label,
                                status: detail.status === 'error' ? 'error' : detail.status === 'success' ? 'success' : 'running',
                                summary: detail.content,
                                params: detail.params,
                                result: detail.result
                                  ? {
                                      ok: detail.result.ok !== false,
                                      data: detail.result.data,
                                      error: detail.result.error,
                                      latency_ms: detail.result.latency_ms,
                                    }
                                  : undefined,
                              }}
                            />
                          );
                        }
                        return (
                          <div key={detail.id} style={{ marginTop: 8, fontSize: 12, lineHeight: 1.6 }}>
                            <div style={{ fontWeight: 500 }}>{detail.label}</div>
                            {detail.content ? <div>{detail.content}</div> : null}
                          </div>
                        );
                      })}
                    </div>
                  ),
                  footer: item.footer,
                  status: item.status,
                  collapsible: item.collapsible,
                  blink: item.blink,
                }))}
                defaultExpandedKeys={defaultExpandedKeys}
              />
            </div>
          )}
          {showThinking ? (
            <div style={{ marginBottom: 12, whiteSpace: 'pre-wrap', color: token.colorTextSecondary }}>
              {thinking}
            </div>
          ) : null}
          {content ? (
            <div className="ai-markdown-content" style={{ marginBottom: confirmation || recommendations?.length || (displayMode === 'debug' && rawEvidence?.length) ? 12 : 0 }}>
              <XMarkdown>{content}</XMarkdown>
            </div>
          ) : isStreaming ? (
            <span style={{ color: token.colorTextSecondary }}>正在输入...</span>
          ) : null}
          {effectiveConfirmation ? (
            <ConfirmationPanel confirmation={effectiveConfirmation} />
          ) : null}
          {recommendations && recommendations.length > 0 && onRecommendationSelect ? (
            <RecommendedActions recommendations={recommendations} onSelect={onRecommendationSelect} />
          ) : null}
          {displayMode === 'debug' && rawEvidence && rawEvidence.length > 0 ? (
            <pre style={{ whiteSpace: 'pre-wrap', marginTop: 12 }}>
              {rawEvidence.map((item) => `- ${item}`).join('\n')}
            </pre>
          ) : null}

          {showActions && !isStreaming && content && (
            <MessageActions
              content={content}
              messageId=""
              isLoading={isLoading}
              onRegenerate={onRegenerate}
            />
          )}
        </>
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
  restored?: boolean;
  hydrated?: boolean;
  updatedAt?: string;
}

interface ConversationState {
  conversations: ConversationItem[];
  activeKey: string;
}

type ConversationAction =
  | { type: 'reset' }
  | { type: 'restore'; conversations: ConversationItem[]; activeKey: string }
  | { type: 'merge_conversation'; conversation: ConversationItem; activate?: boolean }
  | { type: 'set_active'; key: string }
  | { type: 'new'; key: string }
  | { type: 'append_messages'; key: string; label?: string; messages: ExtendedChatMessage[] }
  | { type: 'update_message'; key: string; messageId: string; updater: (message: ExtendedChatMessage) => ExtendedChatMessage }
  | { type: 'remove_message'; key: string; messageId: string };

const DEFAULT_CONVERSATION: ConversationItem = { key: 'default', label: '新对话', group: '今天', messages: [] };

function buildConversationItem(restored: RestoredConversation, hydrated: boolean): ConversationItem {
  return {
    key: restored.id,
    label: restored.title,
    group: '最近',
    restored: true,
    hydrated,
    updatedAt: restored.messages[restored.messages.length - 1]?.createdAt,
    messages: normalizeConversationMessages(restored.messages).map((message) => ({
      ...message,
      createdAt: message.createdAt || new Date().toISOString(),
    })),
  };
}

function normalizeConversationMessages(messages: RestoredConversation['messages']): ExtendedChatMessage[] {
  return [...messages].sort((a, b) => {
    const timeDiff = new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime();
    if (timeDiff !== 0) {
      return timeDiff;
    }
    const aParent = a.turn?.parentTurnId;
    const bParent = b.turn?.parentTurnId;
    if (a.id === bParent) {
      return -1;
    }
    if (b.id === aParent) {
      return 1;
    }
    if (a.role !== b.role) {
      return a.role === 'user' ? -1 : 1;
    }
    return a.id.localeCompare(b.id);
  });
}

function conversationReducer(state: ConversationState, action: ConversationAction): ConversationState {
  switch (action.type) {
    case 'reset':
      return {
        conversations: [DEFAULT_CONVERSATION],
        activeKey: DEFAULT_CONVERSATION.key,
      };
    case 'restore':
      return {
        conversations: action.conversations.length > 0 ? action.conversations : [DEFAULT_CONVERSATION],
        activeKey: action.activeKey,
      };
    case 'merge_conversation': {
      const existingIndex = state.conversations.findIndex((conversation) => conversation.key === action.conversation.key);
      const nextConversations = existingIndex === -1
        ? [action.conversation, ...state.conversations]
        : state.conversations.map((conversation, index) => (
            index === existingIndex
              ? {
                  ...conversation,
                  ...action.conversation,
                  messages: action.conversation.messages.length > 0 ? action.conversation.messages : conversation.messages,
                }
              : conversation
          ));
      return {
        conversations: nextConversations,
        activeKey: action.activate ? action.conversation.key : state.activeKey,
      };
    }
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
  const handleRestoreConversation = useCallback((restoredState: RestoredConversationState) => {
    const summaryItems: ConversationItem[] = restoredState.conversations.map((item) => ({
      key: item.id,
      label: item.title,
      group: '最近',
      messages: [],
      restored: true,
      hydrated: false,
      updatedAt: item.updatedAt,
    }));
    const active = restoredState.activeConversation
      ? buildConversationItem(restoredState.activeConversation, true)
      : undefined;
    const merged = active
      ? summaryItems.map((item) => (item.key === active.key ? active : item))
      : summaryItems;
    const fallbackActiveKey = active?.key || merged[0]?.key || DEFAULT_CONVERSATION.key;
    dispatch({
      type: 'restore',
      conversations: merged.length > 0 ? merged : [DEFAULT_CONVERSATION],
      activeKey: fallbackActiveKey,
    });
    if (active?.key) {
      setSessionId(active.key);
    }
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

  useEffect(() => {
    if (!open) {
      return;
    }
    if (!activeConversation || !activeConversation.restored || activeConversation.hydrated || activeConversation.key === DEFAULT_CONVERSATION.key) {
      return;
    }
    let cancelled = false;
    void (async () => {
      try {
        const restored = await loadRestoredConversationDetail(activeConversation.key, scene);
        if (!restored || cancelled) {
          return;
        }
        dispatch({
          type: 'merge_conversation',
          conversation: buildConversationItem(restored, true),
        });
        setSessionId(restored.id);
      } catch (err) {
        console.error('Failed to hydrate restored conversation:', err);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [activeConversation, open, scene]);

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
    ): ExtendedChatMessage => {
      const nextTurn = overrides.turn !== undefined ? overrides.turn : message.turn;
      const projected = projectTurnSummary(nextTurn);
      return {
        ...message,
        ...overrides,
        turn: nextTurn,
        runtime: overrides.runtime !== undefined ? overrides.runtime : message.runtime,
        content: overrides.content !== undefined
          ? overrides.content
          : (projected.content || message.content || assistantContent),
        thinking: overrides.thinking !== undefined
          ? overrides.thinking
          : (projected.thinking || message.thinking || assistantThinking || undefined),
        rawEvidence: overrides.rawEvidence !== undefined
          ? overrides.rawEvidence
          : (projected.rawEvidence || message.rawEvidence),
        recommendations: overrides.recommendations
          ?? projected.recommendations
          ?? message.recommendations
          ?? assistantRecommendations,
        traceId: overrides.traceId ?? assistantTraceId ?? message.traceId,
      };
    };

    return {
      onMeta: (data: { sessionId?: string; traceId?: string; turn_id?: string }) => {
        if (data.sessionId) {
          setSessionId(data.sessionId);
        }
        if (data.traceId) {
          assistantTraceId = String(data.traceId);
        }
      },
      onChainStarted: (data: SSEChainStartedEvent) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          runtime: reduceThoughtChainRuntimeEvent(message.runtime || createThoughtChainRuntimeState(), {
            type: 'chain_started',
            data,
          }),
        }));
      },
      onChainNodeOpen: (data: SSEChainNodeEvent) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          runtime: reduceThoughtChainRuntimeEvent(message.runtime || createThoughtChainRuntimeState(), {
            type: 'chain_node_open',
            data,
          }),
        }));
      },
      onChainNodePatch: (data: SSEChainNodeEvent) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          runtime: reduceThoughtChainRuntimeEvent(message.runtime || createThoughtChainRuntimeState(), {
            type: 'chain_node_patch',
            data,
          }),
        }));
      },
      onChainNodeClose: (data: SSEChainNodeEvent) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          runtime: reduceThoughtChainRuntimeEvent(message.runtime || createThoughtChainRuntimeState(), {
            type: 'chain_node_close',
            data,
          }),
        }));
      },
      onChainCollapsed: (data: SSEChainStartedEvent) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          runtime: reduceThoughtChainRuntimeEvent(message.runtime || createThoughtChainRuntimeState(), {
            type: 'chain_collapsed',
            data,
          }),
        }));
      },
      onFinalAnswerStarted: (data: SSEFinalAnswerEvent) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          runtime: reduceThoughtChainRuntimeEvent(message.runtime || createThoughtChainRuntimeState(), {
            type: 'final_answer_started',
            data,
          }),
        }));
      },
      onFinalAnswerDelta: (data: SSEFinalAnswerEvent) => {
        assistantContent = `${assistantContent}${String(data.chunk || '')}`;
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          runtime: reduceThoughtChainRuntimeEvent(message.runtime || createThoughtChainRuntimeState(), {
            type: 'final_answer_delta',
            data,
          }),
          content: assistantContent,
        }));
      },
      onFinalAnswerDone: (data: SSEFinalAnswerEvent) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          runtime: reduceThoughtChainRuntimeEvent(message.runtime || createThoughtChainRuntimeState(), {
            type: 'final_answer_done',
            data,
          }),
          content: message.runtime?.finalAnswer.content || assistantContent || message.content,
        }));
      },
      onTurnStarted: (data: { turn_id: string; phase?: string; status?: string; role?: string }) => {
        refreshAnnouncement(`${resolveThoughtStageTitle(data.phase)} ${data.status || ''}`);
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyTurnStarted(message.turn, data, assistantTraceId),
        }));
      },
      onBlockOpen: (data: { turn_id: string; block_id: string; block_type: string; position?: number; status?: string; title?: string; payload?: Record<string, unknown> }) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyBlockOpen(message.turn, data),
        }));
      },
      onBlockDelta: (data: { turn_id: string; block_id: string; block_type?: string; patch?: Record<string, unknown> }) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyBlockDelta(message.turn, data),
        }));
      },
      onBlockReplace: (data: { turn_id: string; block_id: string; block_type?: string; payload?: Record<string, unknown> }) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyBlockReplace(message.turn, data),
        }));
      },
      onBlockClose: (data: { turn_id: string; block_id: string; status?: string }) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyBlockClose(message.turn, data),
        }));
      },
      onTurnState: (data: { status?: string; phase?: string }) => {
        refreshAnnouncement(`${resolveThoughtStageTitle(data.phase)} ${data.status || ''}`);
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyTurnState(message.turn, {
            turn_id: message.turn?.id || assistantId,
            status: data.status,
            phase: data.phase,
          }),
        }));
      },
      onTurnDone: (data: { turn_id: string; status?: string; phase?: string }) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyTurnDone(message.turn, data),
        }));
      },
      onPhaseStarted: (data: SSEPhaseStartedEvent) => {
        refreshAnnouncement(data.title || resolveThoughtStageTitle(data.phase));
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyPhaseStarted(message.turn, data),
        }));
      },
      onPhaseComplete: (data: SSEPhaseCompleteEvent) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyPhaseComplete(message.turn, data),
        }));
      },
      onPlanGenerated: (data: SSEPlanGeneratedEvent) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyPlanGenerated(message.turn, data),
        }));
      },
      onStageDelta: (data: SSEStageDeltaEvent) => {
        const stageKey = String(data.stage || '').trim() as ThoughtStageItem['key'];
        if (!stageKey) {
          return;
        }
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          const stageSummary = String(
            data.contentChunk
            || data.content_chunk
            || data.detail
            || data.summary
            || buildStageMilestone(stageKey, 'delta', data as unknown as Record<string, unknown>)
            || '',
          ).trim();
          const phase = stageKey === 'plan'
            ? 'planning'
            : stageKey === 'execute'
              ? 'executing'
              : stageKey === 'summary'
                ? 'summary'
                : stageKey;
          const nextTurn = normalizeThoughtStatus(data.status as string | undefined, 'loading') === 'success'
            ? applyPhaseComplete(message.turn, {
                turn_id: message.turn?.id || assistantId,
                phase,
                status: String(data.status || 'success'),
                title: resolveThoughtStageTitle(stageKey),
                summary: stageSummary,
              })
            : applyPhaseStarted(message.turn, {
                turn_id: message.turn?.id || assistantId,
                phase,
                status: String(data.status || 'loading'),
                title: resolveThoughtStageTitle(stageKey),
                summary: stageSummary,
              });
          return syncMessageFromBuffers(message, { turn: nextTurn });
        });
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
                String(data.contentChunk || data.content_chunk || data.detail || data.summary || buildStageMilestone(stageKey, 'delta', data as unknown as Record<string, unknown>) || '').trim(),
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
            title: resolveThoughtStageTitle('rewrite'),
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
            title: resolveThoughtStageTitle('plan'),
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
            title: resolveThoughtStageTitle('plan'),
            status: 'success',
            description: buildStageDescription('plan', 'success'),
            content: appendStageContent(
              (message.thoughtChain || []).find((item) => item.key === 'plan')?.content,
              buildStageMilestone('plan', 'event', data),
            ),
          }),
        }));
      },
      onStepStarted: (data: SSEStepStartedEvent) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyStepStarted(message.turn, data),
        }));
      },
      onStepComplete: (data: SSEStepCompleteEvent) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyStepComplete(message.turn, data),
        }));
      },
      onStepUpdate: (data: SSEStepUpdateEvent) => {
        const rawStatus = String(data.status || '').trim();
        const stepData = data as unknown as Record<string, unknown>;
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: rawStatus === 'success' || rawStatus === 'completed' || rawStatus === 'error' || rawStatus === 'failed'
            ? applyStepComplete(message.turn, data)
            : applyStepStarted(message.turn, data),
        }));
        if (rawStatus === 'waiting_approval') {
          patchAssistantMessage(conversationKey, assistantId, (message) => ({
            ...message,
            thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
              key: 'user_action',
              title: '等待你确认',
              status: 'loading',
              description: String(data.title || data.user_visible_summary || '当前步骤需要确认后继续执行'),
              content: String(data.user_visible_summary || data.title || '当前步骤需要确认后继续执行'),
            }),
          }));
          return;
        }
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          const nextStatus = normalizeThoughtStatus(data.status as string | undefined, 'loading');
          const detailID = buildToolDetailID(stepData);
          const currentStage = (message.thoughtChain || []).find((item) => item.key === 'execute');
          const details = [...(currentStage?.details || [])];
          const detailIndex = details.findIndex((item) => item.id === detailID);
          const nextDetail: ThoughtStageDetailItem = {
            id: detailID,
            kind: 'tool',
            label: String(data.title || data.tool_name || data.tool || data.expert || data.step_id || '执行步骤'),
            tool: String(data.tool_name || data.tool || data.expert || '').trim() || undefined,
            status: nextStatus,
            content: String(data.user_visible_summary || data.summary || data.error || '').trim() || undefined,
            params: data.params,
            result: data.result,
            session_id: data.session_id,
            plan_id: data.plan_id,
            step_id: data.step_id,
            checkpoint_id: data.checkpoint_id,
          };
          if (detailIndex === -1) {
            details.push(nextDetail);
          } else {
            details[detailIndex] = { ...details[detailIndex], ...nextDetail };
          }
          return {
            ...message,
            confirmation: undefined,
            thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
              key: 'execute',
              title: '调用专家执行',
              status: nextStatus,
              description: buildStageDescription('execute', nextStatus, stepData),
              content: buildExecutionStageSummary(stepData) || buildStageMilestone('execute', 'event', stepData),
              details,
            }),
          };
        });
      },
      onToolCall: (data: Record<string, unknown>) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          return {
            ...message,
            thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
              key: 'execute',
              title: '调用专家执行',
              status: 'loading',
              description: buildStageDescription('execute', 'loading', data),
              content: buildExecutionStageSummary(data),
            }),
          };
        });
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyStepStarted(message.turn, {
            turn_id: String(data.turn_id || message.turn?.id || assistantId),
            step_id: typeof data.step_id === 'string' ? data.step_id : undefined,
            title: typeof data.title === 'string' ? data.title : undefined,
            tool_name: typeof data.tool_name === 'string' ? data.tool_name : typeof data.tool === 'string' ? data.tool : undefined,
            tool: typeof data.tool === 'string' ? data.tool : undefined,
            status: 'running',
            summary: typeof data.summary === 'string' ? data.summary : undefined,
            user_visible_summary: typeof data.user_visible_summary === 'string' ? data.user_visible_summary : undefined,
            params: data.params as Record<string, unknown> | undefined,
          }),
        }));
      },
      onToolResult: (data: Record<string, unknown>) => {
        const result = data.result as Record<string, unknown> | undefined;
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          const status = data.status === 'error' || result?.ok === false ? 'error' : 'success';
          return {
            ...message,
            thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
              key: 'execute',
              title: '调用专家执行',
              status: status === 'error' ? 'error' : 'loading',
              description: buildStageDescription('execute', status === 'error' ? 'error' : 'loading', data),
              content: buildExecutionStageSummary(data),
            }),
          };
        });
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyStepComplete(message.turn, {
            turn_id: String(data.turn_id || message.turn?.id || assistantId),
            step_id: typeof data.step_id === 'string' ? data.step_id : undefined,
            title: typeof data.title === 'string' ? data.title : undefined,
            tool_name: typeof data.tool_name === 'string' ? data.tool_name : typeof data.tool === 'string' ? data.tool : undefined,
            tool: typeof data.tool === 'string' ? data.tool : undefined,
            status: data.status === 'error' || result?.ok === false ? 'error' : 'success',
            summary: typeof data.summary === 'string' ? data.summary : undefined,
            user_visible_summary: typeof data.user_visible_summary === 'string' ? data.user_visible_summary : undefined,
            result,
            error: typeof data.error === 'string' ? data.error : undefined,
            params: data.params as Record<string, unknown> | undefined,
          }),
        }));
      },
      onDelta: (data: { contentChunk: string }) => {
        assistantContent += data.contentChunk || '';
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          content: assistantContent,
          turn: applyBlockDelta(message.turn, {
            turn_id: message.turn?.id || assistantId,
            block_id: 'text:main',
            block_type: 'text',
            patch: {
              content_chunk: data.contentChunk,
            },
          }),
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
          turn: applyBlockDelta(message.turn, {
            turn_id: message.turn?.id || assistantId,
            block_id: 'thinking:main',
            block_type: 'thinking',
            patch: {
              content_chunk: data.contentChunk,
            },
          }),
          thoughtChain: upsertThoughtStage(finalizeThoughtStage(
            message.thoughtChain || [],
            'execute',
            'success',
            buildStageDescription('execute', 'success'),
          ), {
            key: 'summary',
            title: resolveThoughtStageTitle('summary'),
            status: 'loading',
            description: buildStageDescription('summary', 'loading'),
            content: assistantThinking,
          }),
        }));
      },
      onApprovalRequired: (data: ApprovalRequiredEvent) => {
        const confirmation: ConfirmationRequest = {
          id: String(data.id || data.step_id || data.checkpoint_id || assistantId),
          title: String(data.title || data.tool_name || data.tool || '等待确认'),
          description: String(data.user_visible_summary || data.title || '当前步骤需要确认后继续执行'),
          risk: (data.risk || data.risk_level || 'high') as 'low' | 'medium' | 'high',
          status: 'waiting_user',
          details: data as unknown as Record<string, unknown>,
          onConfirm: () => {},
          onCancel: () => {},
        };
        patchAssistantMessage(conversationKey, assistantId, (message) => {
          return syncMessageFromBuffers(message, {
            confirmation,
            runtime: reduceThoughtChainRuntimeEvent(message.runtime || createThoughtChainRuntimeState(), {
              type: 'chain_node_open',
              data: {
                turn_id: data.turn_id || message.turn?.id || assistantId,
                node_id: `approval:${String(data.id || data.step_id || data.checkpoint_id || assistantId)}`,
                kind: 'approval',
                title: String(data.title || data.tool_name || data.tool || '等待确认'),
                status: 'waiting',
                summary: String(data.user_visible_summary || data.title || '当前步骤需要确认后继续执行'),
                approval: data as unknown as Record<string, unknown>,
              },
            }),
            turn: applyBlockReplace(message.turn, {
              turn_id: String(data.turn_id || message.turn?.id || assistantId),
              block_id: `approval:${String(data.id || data.step_id || data.checkpoint_id || assistantId)}`,
              block_type: 'approval',
              payload: data as unknown as Record<string, unknown>,
            }),
            thoughtChain: upsertThoughtStage(message.thoughtChain || [], {
              key: 'user_action',
              title: '等待你确认',
              status: 'loading',
              description: String(data.title || '当前步骤需要确认后继续执行'),
              content: String(data.user_visible_summary || ''),
            }),
          });
        });
        setIsLoading(false);
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
        setIsLoading(false);
      },
      onReplanTriggered: (data: SSEReplanTriggeredEvent) => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyReplanTriggered(message.turn, data),
        }));
      },
      onSummary: () => {
        patchAssistantMessage(conversationKey, assistantId, (message) => syncMessageFromBuffers(message, {
          turn: applyPhaseComplete(message.turn, {
            turn_id: message.turn?.id || assistantId,
            phase: 'summary',
            status: 'success',
            title: '整理最终结论',
            summary: '已整理出最终结论与建议',
          }),
          thoughtChain: finalizeThoughtStage(
            finalizeThoughtStage(
              finalizeThoughtStage(
                message.thoughtChain || [],
                'execute',
                'success',
                buildStageDescription('execute', 'success'),
              ),
              'summary',
              'success',
              '已整理出最终结论与建议',
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
            content: message.runtime?.finalAnswer.content || assistantContent || message.content,
            thinking: assistantThinking || undefined,
            recommendations: assistantRecommendations || message.recommendations,
            traceId: assistantTraceId || message.traceId,
            turn: message.turn
              ? applyTurnDone(message.turn, {
                  turn_id: data.turn_id || message.turn.id,
                  status: data.stream_state === 'failed' ? 'error' : message.turn.status,
                  phase: message.turn.phase,
                })
              : message.turn,
          }),
          thoughtChain: (message.thoughtChain || []).map((item) => ({
            ...item,
            blink: false,
            status: item.key === 'user_action' && message.confirmation
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
          return syncMessageFromBuffers(message, {
            content: message.content || errorText,
            turn: applyBlockReplace(message.turn, {
              turn_id: message.turn?.id || assistantId,
              block_id: `error:${stageKey || 'main'}`,
              block_type: 'error',
              payload: {
                title: stageKey ? resolveThoughtStageTitle(stageKey) : '执行错误',
                content: errorText,
                stage: stageKey || undefined,
              },
            }),
            thoughtChain: nextThoughtChain,
            confirmation: message.confirmation ? { ...message.confirmation, status: 'failed', errorMessage: errorText } : message.confirmation,
          });
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
          runtime: createThoughtChainRuntimeState(),
          turn: createAssistantTurn(assistantId),
          thoughtChain: [],
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
    patchAssistantMessage(activeKey, assistantId, (message) => {
      return {
        ...message,
        confirmation: {
          id: String(payload.id || payload.step_id || payload.checkpoint_id || assistantId),
          title: String(payload.title || '正在提交'),
          description: approved ? '正在提交确认，马上继续执行。' : '正在提交取消请求。',
          risk: (payload.risk || payload.risk_level || 'high') as 'low' | 'medium' | 'high',
          status: 'submitting',
          details: payload,
          onConfirm: () => {},
          onCancel: () => {},
        },
        turn: applyBlockReplace(message.turn, {
          turn_id: message.turn?.id || assistantId,
          block_id: `approval:${String(payload.id || payload.step_id || payload.checkpoint_id || assistantId)}`,
          block_type: 'approval',
          payload: {
            ...payload,
            status: 'submitting',
          },
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
          checkpoint_id: payload.checkpoint_id ? String(payload.checkpoint_id) : undefined,
          approved: false,
        });
        patchAssistantMessage(activeKey, assistantId, (message) => ({
          ...message,
          confirmation: undefined,
          turn: applyBlockReplace(message.turn, {
            turn_id: message.turn?.id || assistantId,
            block_id: `approval:${String(payload.id || payload.step_id || payload.checkpoint_id || assistantId)}`,
            block_type: 'approval',
            payload: {
              ...payload,
              status: 'rejected',
            },
          }),
          content: message.content || '已取消该操作。',
        }));
        setIsLoading(false);
        return;
      }

      patchAssistantMessage(activeKey, assistantId, (message) => ({
        ...message,
        confirmation: {
          id: String(payload.id || payload.step_id || payload.checkpoint_id || assistantId),
          title: '已确认，继续执行',
          description: '审批已通过，正在继续执行当前步骤。',
          risk: (payload.risk || payload.risk_level || 'high') as 'low' | 'medium' | 'high',
          status: 'submitting',
          details: payload,
          onConfirm: () => {},
          onCancel: () => {},
        },
        turn: applyBlockReplace(message.turn, {
          turn_id: message.turn?.id || assistantId,
          block_id: `approval:${String(payload.id || payload.step_id || payload.checkpoint_id || assistantId)}`,
          block_type: 'approval',
          payload: {
            ...payload,
            status: 'submitting',
          },
        }),
      }));

      await aiApi.respondApprovalStream(
        {
          session_id: String(payload.session_id || sessionId || ''),
          plan_id: payload.plan_id ? String(payload.plan_id) : undefined,
          step_id: payload.step_id ? String(payload.step_id) : undefined,
          checkpoint_id: payload.checkpoint_id ? String(payload.checkpoint_id) : undefined,
          approved: true,
        },
        createStreamHandlers(activeKey, assistantId),
      );
    } catch {
      patchAssistantMessage(activeKey, assistantId, (message) => ({
        ...message,
        confirmation: {
          id: String(payload.id || payload.step_id || payload.checkpoint_id || assistantId),
          title: String(payload.title || '等待你确认'),
          description: String(payload.user_visible_summary || payload.title || '当前步骤需要确认后继续执行'),
          risk: (payload.risk || payload.risk_level || 'high') as 'low' | 'medium' | 'high',
          status: 'failed',
          errorMessage: '审批结果提交失败，请重试。',
          details: payload,
          onConfirm: () => {},
          onCancel: () => {},
        },
        turn: applyBlockReplace(message.turn, {
          turn_id: message.turn?.id || assistantId,
          block_id: `approval:${String(payload.id || payload.step_id || payload.checkpoint_id || assistantId)}`,
          block_type: 'approval',
          payload: {
            ...payload,
            status: 'failed',
            error_message: '审批结果提交失败，请重试。',
          },
        }),
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
      confirmation: undefined,
      recommendations: undefined,
      rawEvidence: undefined,
      runtime: createThoughtChainRuntimeState(),
      turn: createAssistantTurn(message.turn?.id || assistantMsgId),
      thoughtChain: [],
      traceId: undefined,
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
    const hasVisibleAssistantState = Boolean(msg.content || msg.thinking || hasThoughtChain || msg.confirmation);
    const isStreaming = isCurrentStreaming && !hasVisibleAssistantState;

    // 只有当消息内容正在生成时（内容为空）才显示 loading
    // 如果消息已经有内容了（即使正在生成推荐），重新生成按钮不显示 loading
    const showLoading = isLoading && isCurrentStreaming && !msg.content && !hasThoughtChain;

    return (
      <AssistantMessage
        content={msg.content}
        thinking={msg.thinking}
        confirmation={msg.confirmation}
        recommendations={msg.recommendations}
        thoughtChain={msg.thoughtChain}
        turn={msg.turn}
        runtime={msg.runtime}
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
                  && !m.confirmation,
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
