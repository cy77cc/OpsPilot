import { useState, useEffect, useCallback } from 'react';
import { aiApi } from '../../../api/modules/ai';
import type { AISession } from '../../../api/modules/ai';
import type { ChatTurn, EmbeddedRecommendation, ThoughtChainRuntimeState, ThoughtStageItem } from '../types';
import { projectTurnSummary, turnFromReplay } from '../turnLifecycle';
import { runtimeStateFromReplayTurn } from '../thoughtChainRuntime';

export interface RestoredConversation {
  id: string;
  title: string;
  messages: Array<{
    id: string;
    role: 'user' | 'assistant';
    content: string;
    thinking?: string;
    traceId?: string;
    thoughtChain?: ThoughtStageItem[];
    recommendations?: EmbeddedRecommendation[];
    rawEvidence?: string[];
    status?: string;
    turn?: ChatTurn;
    runtime?: ThoughtChainRuntimeState;
    restored?: boolean;
    createdAt: string;
  }>;
}

export interface RestoredConversationSummary {
  id: string;
  title: string;
  createdAt: string;
  updatedAt: string;
}

export interface RestoredConversationState {
  conversations: RestoredConversationSummary[];
  activeConversation: RestoredConversation | null;
}

interface UseConversationRestoreOptions {
  scene: string;
  enabled?: boolean;
  onRestore?: (state: RestoredConversationState) => void;
}

interface UseConversationRestoreResult {
  isRestoring: boolean;
  error: string | null;
  restoredSessionId: string | null;
  restore: () => Promise<void>;
}

/**
 * 会话恢复 Hook
 * 页面刷新后自动恢复最近的对话会话
 */
export function useConversationRestore(options: UseConversationRestoreOptions): UseConversationRestoreResult {
  const { scene, enabled = true, onRestore } = options;

  const [isRestoring, setIsRestoring] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [restoredSessionId, setRestoredSessionId] = useState<string | null>(null);

  const restore = useCallback(async () => {
    if (!enabled) return;

    setIsRestoring(true);
    setError(null);

    try {
      const listRes = await aiApi.getSessions(scene);
      const summaries = normalizeConversationSummaries(listRes.data || []);

      // 1. 尝试获取当前活跃会话
      const currentRes = await aiApi.getCurrentSession(scene);
      if (currentRes.data) {
        const session = currentRes.data;
        setRestoredSessionId(session.id);
        onRestore?.({
          conversations: summaries,
          activeConversation: toRestoredConversation(session),
        });
        return;
      }

      // 2. 如果没有当前会话，尝试获取最近的会话列表
      if (summaries.length > 0) {
        const recentSession = summaries[0];
        const detailRes = await aiApi.getSessionDetail(recentSession.id, scene);
        if (detailRes.data) {
          const session = detailRes.data;
          setRestoredSessionId(session.id);
          onRestore?.({
            conversations: summaries,
            activeConversation: toRestoredConversation(session),
          });
        } else {
          onRestore?.({
            conversations: summaries,
            activeConversation: null,
          });
        }
        return;
      }

      onRestore?.({
        conversations: summaries,
        activeConversation: null,
      });
    } catch (err) {
      console.error('Failed to restore conversation:', err);
      setError((err as Error).message || '恢复会话失败');
    } finally {
      setIsRestoring(false);
    }
  }, [scene, enabled, onRestore]);

  // 组件挂载时自动恢复
  useEffect(() => {
    restore();
  }, [restore]);

  return {
    isRestoring,
    error,
    restoredSessionId,
    restore,
  };
}

export async function loadRestoredConversationDetail(id: string, scene: string): Promise<RestoredConversation | null> {
  const detailRes = await aiApi.getSessionDetail(id, scene);
  if (!detailRes.data) {
    return null;
  }
  return toRestoredConversation(detailRes.data);
}

function toRestoredConversation(session: AISession): RestoredConversation {
  if (session.turns && session.turns.length > 0) {
    const runtimeByTurnId = new Map(
      normalizeLegacyMessages(session.messages || [])
        .filter((message) => message.role === 'assistant' && typeof message.turnId === 'string' && message.turnId.trim())
        .map((message) => [String(message.turnId), normalizePersistedRuntime(message.runtime)]),
    );
    const replayMessages = normalizeReplayTurns(session.turns).map((turn) => {
      const hydratedTurn = turnFromReplay(turn);
      const runtime = turn.role === 'assistant'
        ? (runtimeByTurnId.get(turn.id) || runtimeStateFromReplayTurn(hydratedTurn))
        : undefined;
      const summary = projectTurnSummary(hydratedTurn);
      const fallbackContent = turn.role === 'assistant'
        ? resolveReplayAssistantContent(turn, runtime, summary.content)
        : resolveUserTurnContent(turn, summary.content);
      return {
        id: turn.id,
        role: turn.role,
        content: fallbackContent,
        thinking: turn.role === 'assistant' ? summary.thinking : undefined,
        traceId: hydratedTurn.traceId,
        recommendations: turn.role === 'assistant' ? summary.recommendations : undefined,
        rawEvidence: turn.role === 'assistant' ? summary.rawEvidence : undefined,
        status: hydratedTurn.status,
        turn: hydratedTurn,
        runtime,
        restored: true,
        createdAt: turn.createdAt,
      };
    });
    return {
      id: session.id,
      title: session.title || 'AI Session',
      messages: mergeReplayMessages(session.messages || [], replayMessages),
    };
  }

  return {
    id: session.id,
    title: session.title || 'AI Session',
    messages: normalizeLegacyMessages(session.messages || []).map(m => {
      const thoughtChain = ((m.thoughtChain || []) as ThoughtStageItem[]);
      const summaryStage = thoughtChain.find((item) => item.key === 'summary');
      const content = resolveLegacyAssistantContent(m.content, summaryStage);
      return {
        id: m.id,
        role: m.role as 'user' | 'assistant',
        content,
        thinking: undefined,
        traceId: m.traceId,
        thoughtChain: thoughtChain.filter((item) => item.key !== 'summary'),
        recommendations: (m.recommendations || []) as EmbeddedRecommendation[],
        rawEvidence: (m.rawEvidence || []) as string[],
        status: m.status,
        restored: true,
        createdAt: m.timestamp,
      };
    }),
  };
}

function mergeReplayMessages(
  legacyMessages: AISession['messages'],
  replayMessages: RestoredConversation['messages'],
): RestoredConversation['messages'] {
  const replayHasUserTurn = replayMessages.some((message) => message.role === 'user');
  const legacyUserMessages = replayHasUserTurn
    ? []
    : normalizeLegacyMessages(legacyMessages).filter((message) => message.role === 'user').map((message) => ({
        id: String(message.id || ''),
        role: 'user' as const,
        content: String(message.content || ''),
        status: typeof message.status === 'string' ? message.status : undefined,
        turn: undefined,
        restored: true,
        createdAt: String(message.timestamp || ''),
      }));

  return [...legacyUserMessages, ...replayMessages].sort((a, b) => {
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

function normalizeConversationSummaries(sessions: AISession[]): RestoredConversationSummary[] {
  return [...sessions]
    .map((session) => ({
      id: session.id,
      title: session.title || 'AI Session',
      createdAt: session.createdAt,
      updatedAt: session.updatedAt,
    }))
    .sort((a, b) => (
      new Date(b.updatedAt || b.createdAt).getTime() - new Date(a.updatedAt || a.createdAt).getTime()
    ));
}

function normalizeReplayTurns(turns: NonNullable<AISession['turns']>) {
  return [...turns].sort((a, b) => {
    const timeDiff = new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime();
    if (timeDiff !== 0) {
      return timeDiff;
    }
    if (a.parentTurnId && a.parentTurnId === b.id) {
      return 1;
    }
    if (b.parentTurnId && b.parentTurnId === a.id) {
      return -1;
    }
    if (a.role !== b.role) {
      return a.role === 'user' ? -1 : 1;
    }
    return a.id.localeCompare(b.id);
  });
}

function normalizeLegacyMessages(messages: AISession['messages']) {
  return [...messages].sort((a, b) => {
    const timeDiff = new Date(String(a.timestamp || '')).getTime() - new Date(String(b.timestamp || '')).getTime();
    if (timeDiff !== 0) {
      return timeDiff;
    }
    if (a.role !== b.role) {
      return a.role === 'user' ? -1 : 1;
    }
    return String(a.id || '').localeCompare(String(b.id || ''));
  });
}

function normalizePersistedRuntime(raw: Record<string, unknown> | undefined): ThoughtChainRuntimeState | undefined {
  if (!raw || typeof raw !== 'object') {
    return undefined;
  }

  const finalAnswerRaw = asRecord(raw.final_answer);
  const nodes = asArray(raw.nodes).map((item) => asRecord(item)).filter(Boolean).map((node) => ({
    nodeId: asString(node.node_id) || asString(node.nodeId) || `node:${Date.now()}`,
    kind: normalizeNodeKind(asString(node.kind)),
    title: asString(node.title) || '执行步骤',
    status: normalizeNodeStatus(asString(node.status)),
    headline: asString(node.headline),
    body: asString(node.body),
    structured: asRecord(node.structured),
    raw: node.raw,
    summary: asString(node.summary),
    details: asArray(node.details),
    approval: asRecord(node.approval) as ThoughtChainRuntimeState['nodes'][number]['approval'],
  }));

  return {
    turnId: asString(raw.turn_id) || asString(raw.turnId),
    nodes,
    activeNodeId: asString(raw.active_node_id) || asString(raw.activeNodeId) || undefined,
    isCollapsed: Boolean(raw.is_collapsed ?? raw.isCollapsed),
    collapsePhase: Boolean(raw.is_collapsed ?? raw.isCollapsed) ? 'collapsed' : 'expanded',
    finalAnswer: {
      visible: Boolean(finalAnswerRaw?.visible),
      streaming: Boolean(finalAnswerRaw?.streaming),
      content: asString(finalAnswerRaw?.content),
      revealState: normalizeRevealState(asString(finalAnswerRaw?.reveal_state) || asString(finalAnswerRaw?.revealState)),
    },
  };
}

function resolveUserTurnContent(turn: NonNullable<AISession['turns']>[number], fallback: string): string {
  return firstNonEmpty(
    turn.blocks.find((block) => block.blockType === 'text')?.contentText,
    fallback,
  );
}

function resolveReplayAssistantContent(
  turn: NonNullable<AISession['turns']>[number],
  runtime: ThoughtChainRuntimeState | undefined,
  fallback: string,
): string {
  if (runtime?.finalAnswer.content?.trim()) {
    return runtime.finalAnswer.content;
  }

  const textBlock = turn.blocks.find((block) => block.blockType === 'text' && block.contentText?.trim());
  if (textBlock?.contentText?.trim()) {
    return textBlock.contentText;
  }

  if (fallback.trim()) {
    return fallback;
  }

  const statusCandidate = [...turn.blocks]
    .filter((block) => block.blockType === 'status' && block.contentText?.trim())
    .sort((a, b) => b.position - a.position)
    .find((block) => looksLikeMarkdownAnswer(block.contentText || ''));
  return statusCandidate?.contentText || '';
}

function looksLikeMarkdownAnswer(content: string): boolean {
  const text = content.trim();
  if (!text) {
    return false;
  }
  return /(^|\n)\s{0,3}#{1,6}\s+\S/.test(text)
    || /(^|\n)\s*[-*]\s+\S/.test(text)
    || /(^|\n)\s*\d+\.\s+\S/.test(text)
    || /```/.test(text)
    || /\|.+\|/.test(text);
}

function resolveLegacyAssistantContent(rawContent: string, summaryStage: ThoughtStageItem | undefined): string {
  const direct = (rawContent || '').trim();
  if (direct) {
    return rawContent;
  }
  const summaryContent = summaryStage?.content || summaryStage?.description || '';
  if (summaryContent.trim()) {
    return summaryContent;
  }
  return rawContent;
}

function firstNonEmpty(...values: Array<string | undefined>): string {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) {
      return value;
    }
  }
  return '';
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? value as Record<string, unknown>
    : undefined;
}

function asArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

function asString(value: unknown): string {
  return typeof value === 'string' ? value : '';
}

function normalizeNodeKind(kind: string): ThoughtChainRuntimeState['nodes'][number]['kind'] {
  switch (kind) {
    case 'plan':
    case 'execute':
    case 'tool':
    case 'replan':
    case 'approval':
      return kind;
    default:
      return 'execute';
  }
}

function normalizeNodeStatus(status: string): ThoughtChainRuntimeState['nodes'][number]['status'] {
  switch (status) {
    case 'pending':
      return 'pending';
    case 'waiting':
    case 'waiting_approval':
      return 'waiting';
    case 'done':
    case 'success':
    case 'completed':
      return 'done';
    case 'error':
    case 'failed':
      return 'error';
    default:
      return 'active';
  }
}

function normalizeRevealState(state: string): ThoughtChainRuntimeState['finalAnswer']['revealState'] {
  switch (state) {
    case 'primed':
    case 'revealing':
    case 'complete':
      return state;
    default:
      return 'hidden';
  }
}
