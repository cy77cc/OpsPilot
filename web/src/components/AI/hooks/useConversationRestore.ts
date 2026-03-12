import { useState, useEffect, useCallback } from 'react';
import { aiApi } from '../../../api/modules/ai';
import type { AISession } from '../../../api/modules/ai';
import type { ChatTurn, EmbeddedRecommendation, ThoughtStageItem } from '../types';
import { projectTurnSummary, turnFromReplay } from '../turnLifecycle';

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
    restored?: boolean;
    createdAt: string;
  }>;
}

interface UseConversationRestoreOptions {
  scene: string;
  enabled?: boolean;
  onRestore?: (conversation: RestoredConversation) => void;
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
      // 1. 尝试获取当前活跃会话
      const currentRes = await aiApi.getCurrentSession(scene);
      if (currentRes.data) {
        const session = currentRes.data;
        setRestoredSessionId(session.id);
        onRestore?.(toRestoredConversation(session));
        return;
      }

      // 2. 如果没有当前会话，尝试获取最近的会话列表
      const listRes = await aiApi.getSessions(scene);
      if (listRes.data && listRes.data.length > 0) {
        const recentSession = listRes.data[0];
        const detailRes = await aiApi.getSessionDetail(recentSession.id, scene);
        if (detailRes.data) {
          const session = detailRes.data;
          setRestoredSessionId(session.id);
          onRestore?.(toRestoredConversation(session));
        }
      }
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

function toRestoredConversation(session: AISession): RestoredConversation {
  if (session.turns && session.turns.length > 0) {
    return {
      id: session.id,
      title: session.title || 'AI Session',
      messages: session.turns.map((turn) => {
        const hydratedTurn = turnFromReplay(turn);
        const summary = projectTurnSummary(hydratedTurn);
        return {
          id: turn.id,
          role: turn.role,
          content: turn.role === 'user' ? summary.content : summary.content,
          thinking: turn.role === 'assistant' ? summary.thinking : undefined,
          traceId: hydratedTurn.traceId,
          recommendations: turn.role === 'assistant' ? summary.recommendations : undefined,
          rawEvidence: turn.role === 'assistant' ? summary.rawEvidence : undefined,
          status: hydratedTurn.status,
          turn: hydratedTurn,
          restored: true,
          createdAt: turn.createdAt,
        };
      }),
    };
  }

  return {
    id: session.id,
    title: session.title || 'AI Session',
    messages: (session.messages || []).map(m => {
      const thoughtChain = ((m.thoughtChain || []) as ThoughtStageItem[]);
      const summaryStage = thoughtChain.find((item) => item.key === 'summary');
      return {
        id: m.id,
        role: m.role as 'user' | 'assistant',
        content: m.content,
        thinking: m.thinking || summaryStage?.content || undefined,
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
