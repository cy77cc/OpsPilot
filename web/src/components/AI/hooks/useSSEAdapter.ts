import { useCallback, useRef } from 'react';
import { message } from 'antd';
import type { ChatMessage, ErrorInfo, ErrorType } from '../types';

const API_BASE = import.meta.env.VITE_API_BASE || '/api/v1';

interface SSEAdapterOptions {
  scene: string;
  sessionId?: string;
  onMessage?: (message: ChatMessage) => void;
  onToolCall?: (tool: { id: string; name: string }) => void;
  onToolResult?: (tool: { id: string; name: string; status: string; duration?: number }) => void;
  onConfirmation?: (confirmation: Record<string, unknown>) => void;
  onDone?: (sessionId: string) => void;
  onError?: (error: ErrorInfo) => void;
}

/**
 * 错误消息映射
 */
const ERROR_MESSAGES: Record<ErrorType, string> = {
  network: '网络连接失败，请检查网络后重试',
  timeout: '请求超时，请重试',
  auth: '登录已过期，请重新登录',
  tool: '工具执行失败',
  unknown: '发生未知错误，请稍后重试',
};

/**
 * 分类错误类型
 */
function classifyError(error: unknown): ErrorType {
  if (error instanceof TypeError && error.message.includes('fetch')) {
    return 'network';
  }
  if (error instanceof Error) {
    if (error.message.includes('timeout') || error.message.includes('Timeout')) {
      return 'timeout';
    }
    if (error.message.includes('401') || error.message.includes('Unauthorized')) {
      return 'auth';
    }
  }
  return 'unknown';
}

/**
 * SSE 适配 Hook
 * 将后端 SSE 接口适配为消息流
 */
export function useSSEAdapter(options: SSEAdapterOptions) {
  const { scene, sessionId, onMessage, onToolCall, onToolResult, onConfirmation, onDone, onError } = options;

  const abortControllerRef = useRef<AbortController | null>(null);
  const assistantIdRef = useRef<string>('');
  const contentRef = useRef<string>('');
  const thinkingRef = useRef<string>('');
  const toolsRef = useRef<Array<{ id: string; name: string; status: string; duration?: number }>>([]);

  /**
   * 发送消息更新
   */
  const emitMessage = useCallback(() => {
    const msg: ChatMessage = {
      id: assistantIdRef.current,
      role: 'assistant',
      content: contentRef.current,
      thinking: thinkingRef.current || undefined,
      tools: toolsRef.current.map((t) => ({
        id: t.id,
        name: t.name,
        status: t.status as 'running' | 'success' | 'error',
        duration: t.duration,
      })),
      createdAt: new Date().toISOString(),
    };
    onMessage?.(msg);
  }, [onMessage]);

  /**
   * 处理单个 SSE 事件块
   */
  const dispatchEvent = useCallback(
    (chunk: string) => {
      const lines = chunk.split('\n');
      let eventType = 'message';
      const dataLines: string[] = [];

      lines.forEach((line) => {
        if (line.startsWith('event:')) {
          eventType = line.slice(6).trim();
          return;
        }
        if (line.startsWith('data:')) {
          dataLines.push(line.slice(5).trim());
        }
      });

      if (dataLines.length === 0) {
        return;
      }

      const rawData = dataLines.join('\n');
      let payload: Record<string, unknown>;
      try {
        payload = JSON.parse(rawData);
      } catch {
        payload = { message: rawData };
      }

      switch (eventType) {
        case 'meta':
          // 会话元数据
          break;

        case 'delta':
          // 文本增量
          contentRef.current += String(payload.contentChunk || '');
          emitMessage();
          break;

        case 'thinking_delta':
          // 思考过程增量
          thinkingRef.current += String(payload.contentChunk || '');
          emitMessage();
          break;

        case 'tool_call':
          // 工具调用开始
          const nestedPayload = payload.payload && typeof payload.payload === 'object' ? (payload.payload as Record<string, unknown>) : undefined;
          const toolName = String(payload.tool || nestedPayload?.tool || 'unknown');
          const toolId = String(payload.call_id || `tool-${Date.now()}`);
          toolsRef.current.push({ id: toolId, name: toolName, status: 'running' });
          onToolCall?.({ id: toolId, name: toolName });
          emitMessage();
          break;

        case 'tool_result':
          // 工具调用结果
          const resultToolId = String(payload.call_id || '');
          const resultToolName = String(payload.tool || 'unknown');
          const result = payload.result as Record<string, unknown> | undefined;
          const resultOk = result?.ok !== false;
          const duration = result?.latency_ms as number | undefined;

          toolsRef.current = toolsRef.current.map((t) =>
            t.id === resultToolId || t.name === resultToolName
              ? { ...t, status: resultOk ? 'success' : 'error', duration: duration ? duration / 1000 : undefined }
              : t
          );
          onToolResult?.({
            id: resultToolId,
            name: resultToolName,
            status: resultOk ? 'success' : 'error',
            duration: duration ? duration / 1000 : undefined,
          });
          emitMessage();
          break;

        case 'approval_required':
        case 'confirmation_required':
          // 需要确认
          onConfirmation?.(payload);
          break;

        case 'done':
          // 完成
          const sessionData = payload.session as Record<string, unknown> | undefined;
          const newSessionId = sessionData?.id as string | undefined;
          if (newSessionId) {
            onDone?.(newSessionId);
          }
          break;

        case 'error':
          // 错误
          const errorMsg = String(payload.message || payload.error || '发生错误');
          message.error(errorMsg);
          onError?.({ type: 'unknown', message: errorMsg });
          break;

        case 'heartbeat':
          // 心跳，忽略
          break;
      }
    },
    [emitMessage, onToolCall, onToolResult, onConfirmation, onDone, onError]
  );

  /**
   * 发送消息并处理 SSE 响应
   */
  const sendMessage = useCallback(
    async (content: string): Promise<void> => {
      const token = localStorage.getItem('token');
      const projectId = localStorage.getItem('projectId');

      // 重置状态
      assistantIdRef.current = `assistant-${Date.now()}`;
      contentRef.current = '';
      thinkingRef.current = '';
      toolsRef.current = [];

      // 创建 AbortController
      abortControllerRef.current = new AbortController();

      try {
        const response = await fetch(`${API_BASE}/ai/chat`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
            ...(projectId ? { 'X-Project-ID': projectId } : {}),
          },
          signal: abortControllerRef.current.signal,
          body: JSON.stringify({
            sessionId,
            message: content,
            context: { scene },
          }),
        });

        if (!response.ok) {
          if (response.status === 401) {
            const errorInfo: ErrorInfo = { type: 'auth', message: ERROR_MESSAGES.auth };
            message.error(errorInfo.message);
            onError?.(errorInfo);
            // 跳转登录
            window.location.href = '/login';
            return;
          }
          throw new Error(`HTTP ${response.status}`);
        }

        if (!response.body) {
          throw new Error('No response body');
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder('utf-8');
        let buffer = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          // 移除 \r 并追加到 buffer
          buffer += decoder.decode(value, { stream: true }).replace(/\r/g, '');

          // 用 \n\n 分割事件块
          const segments = buffer.split('\n\n');
          // 最后一个可能不完整，保留
          buffer = segments.pop() || '';

          // 处理完整的事件块
          segments.forEach(dispatchEvent);
        }

        // 处理剩余的 buffer
        if (buffer.trim()) {
          dispatchEvent(buffer);
        }
      } catch (error) {
        if ((error as Error).name === 'AbortError') {
          // 用户取消，不报错
          return;
        }

        const errorType = classifyError(error);
        const errorInfo: ErrorInfo = {
          type: errorType,
          message: ERROR_MESSAGES[errorType],
        };

        message.error(errorInfo.message);
        onError?.(errorInfo);
      }
    },
    [scene, sessionId, dispatchEvent, onError]
  );

  /**
   * 取消请求
   */
  const cancel = useCallback(() => {
    abortControllerRef.current?.abort();
    abortControllerRef.current = null;
  }, []);

  return {
    sendMessage,
    cancel,
  };
}
