import { useCallback, useMemo, useRef, useState } from 'react';
import { message } from 'antd';
import type { ChatMessage, Conversation, ConfirmationRequest, ToolExecution } from '../types';
import { aiApi } from '../../../api/modules/ai';
import type { ApprovalTicket } from '../../../api/modules/ai';

interface UseAIChatOptions {
  scene: string;
  sessionId?: string;
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
    (payload: ApprovalTicket & { turn_id?: string; approval_required?: boolean; previewDiff?: string }) => {
      const confirmation: ConfirmationRequest = {
        id: String(payload.id || payload.turn_id || Date.now()),
        title: String(payload.tool || '待确认操作'),
        description: '高风险操作需要审批后继续执行',
        risk: payload.risk || 'high',
        details: payload as unknown as Record<string, unknown>,
        onConfirm: () => {
          void confirmApproval(true);
        },
        onCancel: () => {
          void confirmApproval(false);
        },
      };

      setPendingConfirmation(confirmation);
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
        checkpoint_id: details.checkpoint_id as string,
        session_id: details.session_id as string,
        target: (details.interrupt_targets as string[])?.[0] || '',
        approved,
      });

      setPendingConfirmation(null);

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
      const res = await aiApi.getSessionDetail(id);
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
