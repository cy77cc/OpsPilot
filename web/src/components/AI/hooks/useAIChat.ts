import { useCallback, useMemo, useRef, useState } from 'react';
import { message } from 'antd';
import type {
  ChatMessage,
  Conversation,
  ConfirmationRequest,
  ThoughtStageDetailItem,
  ThoughtStageItem,
  ThoughtStageKey,
  ThoughtStageStatus,
  ToolExecution,
} from '../types';
import { aiApi } from '../../../api/modules/ai';
import type { ApprovalRequiredEvent, SSEStageDeltaEvent, SSEStepUpdateEvent } from '../../../api/modules/ai';

interface UseAIChatOptions {
  scene: string;
  sessionId?: string;
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
    default:
      return fallback;
  }
}

function resolveStageTitle(stage: ThoughtStageKey): string {
  switch (stage) {
    case 'rewrite':
      return '识别目标与约束';
    case 'plan':
      return '整理执行步骤';
    case 'execute':
      return '工具调用链';
    case 'user_action':
      return '等待你确认';
    case 'summary':
      return '整理最终结论';
    default:
      return '处理中';
  }
}

function upsertThoughtStage(
  stages: ThoughtStageItem[] | undefined,
  patch: Partial<ThoughtStageItem> & Pick<ThoughtStageItem, 'key' | 'status'>
): ThoughtStageItem[] {
  const currentStages = stages || [];
  const index = currentStages.findIndex((item) => item.key === patch.key);
  const next: ThoughtStageItem = {
    key: patch.key,
    title: patch.title || resolveStageTitle(patch.key),
    status: patch.status,
    description: patch.description,
    content: patch.content,
    details: patch.details,
    footer: patch.footer,
    collapsible: patch.collapsible ?? true,
    blink: patch.blink ?? patch.status === 'loading',
  };

  if (index === -1) {
    return [...currentStages, next];
  }

  return currentStages.map((item, itemIndex) => (
    itemIndex === index
      ? {
          ...item,
          ...next,
          title: next.title || item.title,
          details: next.details ?? item.details,
          content: next.content ?? item.content,
        }
      : item
  ));
}

function upsertThoughtDetail(
  stages: ThoughtStageItem[] | undefined,
  detail: ThoughtStageDetailItem,
): ThoughtStageItem[] {
  const currentStages = stages || [];
  const executeStage = currentStages.find((item) => item.key === 'execute');
  const details = [...(executeStage?.details || [])];
  const index = details.findIndex((item) => item.id === detail.id);
  if (index === -1) {
    details.push(detail);
  } else {
    details[index] = { ...details[index], ...detail };
  }

  return upsertThoughtStage(currentStages, {
    key: 'execute',
    title: '工具调用链',
    status: detail.status === 'error' ? 'error' : 'loading',
    details,
  });
}

function buildDetailLabel(payload: SSEStepUpdateEvent): string {
  return String(payload.title || payload.tool_name || payload.tool || payload.expert || payload.step_id || '执行步骤');
}

function buildDetailContent(payload: SSEStepUpdateEvent): string | undefined {
  return String(payload.user_visible_summary || payload.summary || payload.error || '').trim() || undefined;
}

/**
 * AI 聊天状态管理 Hook
 * 使用现有的 aiApi.chatStream 处理 SSE
 */
export function useAIChat(options: UseAIChatOptions) {
  const { scene, sessionId: initialSessionId } = options;

  // 状态
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [currentConversation, setCurrentConversation] = useState<Conversation | null>(null);
  const [currentSessionId, setCurrentSessionId] = useState<string | undefined>(initialSessionId);
  const [isLoading, setIsLoading] = useState(false);
  const [pendingConfirmation, setPendingConfirmation] = useState<ConfirmationRequest | null>(null);

  // 引用 - 用于在流式回调中累积内容
  const activeAssistantIdRef = useRef<string>('');
  const contentRef = useRef<string>('');
  const thinkingRef = useRef<string>('');
  const toolsRef = useRef<ToolExecution[]>([]);

  /**
   * 更新助手消息（使用 ref 中的累积内容）
   */
  const emitAssistantMessage = useCallback(() => {
    const assistantId = activeAssistantIdRef.current;
    if (!assistantId) return;

    setMessages((prev) =>
      prev.map((item) => {
        if (item.id !== assistantId) {
          return item;
        }
        return {
          ...item,
          content: contentRef.current,
          thinking: thinkingRef.current || undefined,
          tools: toolsRef.current.length > 0 ? toolsRef.current : undefined,
        };
      })
    );
  }, []);

  /**
   * 处理审批请求
   */
  const handleApprovalRequired = useCallback(
    (payload: ApprovalRequiredEvent) => {
      const confirmation: ConfirmationRequest = {
        id: String(payload.id || payload.step_id || payload.checkpoint_id || Date.now()),
        title: String(payload.title || payload.tool_name || payload.tool || '待确认操作'),
        description: '高风险操作需要审批后继续执行',
        risk: payload.risk || payload.risk_level || 'high',
        details: payload as unknown as Record<string, unknown>,
        onConfirm: () => {
          void confirmApproval(true);
        },
        onCancel: () => {
          void confirmApproval(false);
        },
      };

      setPendingConfirmation(confirmation);
      setIsLoading(false);
      setMessages((prev) => prev.map((item) => (
        item.id !== activeAssistantIdRef.current
          ? item
          : {
              ...item,
              confirmation,
              thoughtChain: upsertThoughtStage(item.thoughtChain, {
                key: 'user_action',
                status: 'loading',
                title: '等待你确认',
                description: String(payload.title || payload.user_visible_summary || '当前步骤需要确认后继续执行'),
                content: String(payload.user_visible_summary || ''),
              }),
            }
      )));
    },
    []
  );

  /**
   * 确认审批
   */
  const confirmApproval = useCallback(async (approved: boolean) => {
    if (!pendingConfirmation) return;

    try {
      const details = pendingConfirmation.details || {};
      await aiApi.respondApproval({
        session_id: details.session_id as string,
        plan_id: details.plan_id as string | undefined,
        step_id: details.step_id as string | undefined,
        checkpoint_id: details.checkpoint_id as string | undefined,
        approved,
      });

      setPendingConfirmation(null);
      setMessages((prev) => prev.map((item) => (
        item.id !== activeAssistantIdRef.current
          ? item
          : {
              ...item,
              confirmation: undefined,
              thoughtChain: upsertThoughtStage(item.thoughtChain, {
                key: 'user_action',
                status: approved ? 'success' : 'abort',
                title: '等待你确认',
                description: approved ? '已确认，继续执行' : '已取消执行',
              }),
            }
      )));

      if (approved) {
        message.success('已确认，继续执行');
      } else {
        message.info('已取消操作');
        setIsLoading(false);
      }
    } catch (error) {
      message.error('确认操作失败');
    }
  }, [pendingConfirmation]);

  /**
   * 发送消息
   */
  const sendMessage = useCallback(
    async (content: string) => {
      const trimmed = content.trim();
      if (!trimmed || isLoading) return;

      const userMessageId = `user-${Date.now()}`;
      const assistantId = `assistant-${Date.now()}`;

      // 重置 ref
      activeAssistantIdRef.current = assistantId;
      contentRef.current = '';
      thinkingRef.current = '';
      toolsRef.current = [];

      // 添加用户消息
      const userMessage: ChatMessage = {
        id: userMessageId,
        role: 'user',
        content: trimmed,
        createdAt: new Date().toISOString(),
      };

      // 添加助手消息占位
      const assistantPlaceholder: ChatMessage = {
        id: assistantId,
        role: 'assistant',
        content: '',
        createdAt: new Date().toISOString(),
      };

      setMessages((prev) => [...prev, userMessage, assistantPlaceholder]);
      setIsLoading(true);

      try {
        await aiApi.chatStream(
          {
            sessionId: currentSessionId,
            message: trimmed,
            context: { scene },
          },
          {
            onMeta: (payload) => {
              if (payload.sessionId) {
                setCurrentSessionId(payload.sessionId);
              }
            },
            onDelta: (payload) => {
              // 追加到 ref
              contentRef.current += payload.contentChunk || '';
              emitAssistantMessage();
            },
            onStageDelta: (payload: SSEStageDeltaEvent) => {
              const stage = String(payload.stage || '').trim() as ThoughtStageKey;
              if (!stage) {
                return;
              }
              setMessages((prev) => prev.map((item) => (
                item.id !== assistantId
                  ? item
                  : {
                      ...item,
                      thoughtChain: upsertThoughtStage(item.thoughtChain, {
                        key: stage,
                        status: normalizeThoughtStatus(payload.status, 'loading'),
                        title: resolveStageTitle(stage),
                        description: String(payload.summary || payload.user_visible_summary || payload.message || '').trim() || item.thoughtChain?.find((entry) => entry.key === stage)?.description,
                        content: String(payload.contentChunk || payload.content_chunk || payload.detail || payload.summary || '').trim() || item.thoughtChain?.find((entry) => entry.key === stage)?.content,
                      }),
                    }
              )));
            },
            onStepUpdate: (payload: SSEStepUpdateEvent) => {
              const status = normalizeThoughtStatus(payload.status, 'loading');
              setMessages((prev) => prev.map((item) => (
                item.id !== assistantId
                  ? item
                  : {
                      ...item,
                      thoughtChain: upsertThoughtDetail(item.thoughtChain, {
                        id: String(payload.step_id || payload.plan_id || payload.tool || Date.now()),
                        label: buildDetailLabel(payload),
                        status,
                        content: buildDetailContent(payload),
                        tool: payload.tool_name || payload.tool,
                        params: payload.params,
                        result: payload.result,
                        risk: undefined,
                        checkpoint_id: payload.checkpoint_id,
                        session_id: payload.session_id,
                        plan_id: payload.plan_id,
                        step_id: payload.step_id,
                      }),
                    }
              )));
            },
            onThinkingDelta: (payload) => {
              thinkingRef.current += payload.contentChunk || '';
              emitAssistantMessage();
            },
            onToolCall: (payload) => {
              // 添加工具
              toolsRef.current.push({
                id: `tool-${Date.now()}`,
                name: payload.tool || 'unknown',
                status: 'running',
              });
              emitAssistantMessage();
            },
            onToolResult: (payload) => {
              // 更新工具状态
              const toolName = payload.tool || 'unknown';
              toolsRef.current = toolsRef.current.map((t) =>
                t.name === toolName
                  ? {
                      ...t,
                      status: payload.result?.ok !== false ? 'success' : 'error',
                      duration: payload.result?.latency_ms
                        ? payload.result.latency_ms / 1000
                        : undefined,
                    }
                  : t
              );
              emitAssistantMessage();
            },
            onApprovalRequired: handleApprovalRequired,
            onDone: (payload) => {
              setIsLoading(false);
              if (payload.session?.id) {
                setCurrentSessionId(payload.session.id);
              }
              setMessages((prev) => prev.map((item) => (
                item.id !== assistantId
                  ? item
                  : {
                      ...item,
                      thoughtChain: (item.thoughtChain || []).map((stage) => ({
                        ...stage,
                        blink: false,
                        status: stage.status === 'loading' ? 'success' : stage.status,
                        details: stage.details?.map((detail) => ({
                          ...detail,
                          status: detail.status === 'loading' ? 'success' : detail.status,
                        })),
                      })),
                    }
              )));
            },
            onError: (error) => {
              setIsLoading(false);
              message.error(error.message || '发生错误');
            },
          }
        );
      } catch (error) {
        setIsLoading(false);
        message.error('请求失败，请重试');
      }
    },
    [scene, currentSessionId, isLoading, emitAssistantMessage, handleApprovalRequired]
  );

  /**
   * 取消当前请求
   */
  const cancel = useCallback(() => {
    setIsLoading(false);
  }, []);

  /**
   * 创建新会话
   */
  const createConversation = useCallback(() => {
    setMessages([]);
    setCurrentSessionId(undefined);
    setCurrentConversation(null);
    setPendingConfirmation(null);
    // 重置 ref
    contentRef.current = '';
    thinkingRef.current = '';
    toolsRef.current = [];
  }, []);

  /**
   * 切换会话
   */
  const switchConversation = useCallback(async (id: string) => {
    try {
      const res = await aiApi.getSessionDetail(id, scene);
      const session = res.data;

      setCurrentSessionId(session.id);
      setCurrentConversation({
        id: session.id,
        title: session.title || 'AI Session',
        scene: scene,
        messages: [],
        createdAt: session.createdAt,
        updatedAt: session.updatedAt,
      });

      setMessages(
        (session.messages || []).map((m) => ({
          id: m.id,
          role: m.role as 'user' | 'assistant',
          content: m.content,
          thinking: m.thinking,
          createdAt: m.timestamp,
        }))
      );
    } catch (error) {
      message.error('加载会话失败');
    }
  }, [scene]);

  /**
   * 删除会话
   */
  const deleteConversation = useCallback(
    async (id: string) => {
      try {
        await aiApi.deleteSession(id);
        setConversations((prev) => prev.filter((c) => c.id !== id));
        if (currentSessionId === id) {
          createConversation();
        }
        message.success('会话已删除');
      } catch (error) {
        message.error('删除会话失败');
      }
    },
    [currentSessionId, createConversation]
  );

  /**
   * 加载会话列表
   */
  const loadConversations = useCallback(async () => {
    try {
      const res = await aiApi.getSessions(scene);
      setConversations(
        (res.data || []).map((s) => ({
          id: s.id,
          title: s.title || 'AI Session',
          scene: scene,
          messages: [],
          createdAt: s.createdAt,
          updatedAt: s.updatedAt,
        }))
      );
    } catch (error) {
      console.error('加载会话列表失败:', error);
    }
  }, [scene]);

  /**
   * 确认操作
   */
  const confirmAction = useCallback(
    async (id: string, approved: boolean) => {
      if (pendingConfirmation?.id === id) {
        await confirmApproval(approved);
      }
    },
    [pendingConfirmation, confirmApproval]
  );

  return useMemo(
    () => ({
      // 状态
      messages,
      isLoading,
      conversations,
      currentConversation,
      currentSessionId,
      pendingConfirmation,

      // 操作
      sendMessage,
      cancel,
      createConversation,
      switchConversation,
      deleteConversation,
      loadConversations,
      confirmAction,

      // 内部方法
      setMessages,
      setCurrentConversation,
    }),
    [
      messages,
      isLoading,
      conversations,
      currentConversation,
      currentSessionId,
      pendingConfirmation,
      sendMessage,
      cancel,
      createConversation,
      switchConversation,
      deleteConversation,
      loadConversations,
      confirmAction,
    ]
  );
}
