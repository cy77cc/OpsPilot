import { consumeAIStream, type A2UIStreamHandlers, apiService, type AIChatParams } from './shared';

export async function chatStream(params: AIChatParams, handlers: A2UIStreamHandlers, signal?: AbortSignal): Promise<void> {
  const base = import.meta.env.VITE_API_BASE || '/api/v1';
  const token = localStorage.getItem('token');
  const projectId = localStorage.getItem('projectId');
  const controller = new AbortController();
  let timedOut = false;
  let toolPending = false;
  let softTimeoutTimer: number | null = null;
  let hardTimeoutTimer: number | null = null;
  let softWarned = false;

  const clearToolTimer = () => {
    if (softTimeoutTimer !== null) window.clearTimeout(softTimeoutTimer);
    if (hardTimeoutTimer !== null) window.clearTimeout(hardTimeoutTimer);
    softTimeoutTimer = null;
    hardTimeoutTimer = null;
    softWarned = false;
  };

  const armToolTimeout = () => {
    clearToolTimer();
    softTimeoutTimer = window.setTimeout(() => {
      if (softWarned) return;
      softWarned = true;
      handlers.onError?.({ code: 'tool_timeout_soft', recoverable: true, message: '工具执行较慢，正在继续等待结果…' });
    }, 25000);
    hardTimeoutTimer = window.setTimeout(() => {
      timedOut = true;
      handlers.onError?.({ code: 'tool_timeout_hard', recoverable: true, message: '工具调用超时，请重试本轮对话。' });
      controller.abort();
    }, 55000);
  };

  const abortFromCaller = () => controller.abort();
  signal?.addEventListener('abort', abortFromCaller, { once: true });

  const response = await fetch(`${base}/ai/chat`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(projectId ? { 'X-Project-ID': projectId } : {}),
      ...(params.lastEventId || params.last_event_id ? { 'Last-Event-ID': String(params.lastEventId || params.last_event_id) } : {}),
    },
    body: JSON.stringify({
      session_id: params.session_id ?? params.sessionId,
      client_request_id: params.clientRequestId,
      lastEventId: params.lastEventId ?? params.last_event_id,
      last_event_id: params.last_event_id ?? params.lastEventId,
      message: params.message,
      scene: params.scene,
      context: params.context,
    }),
    signal: controller.signal,
  });

  const wrappedHandlers: A2UIStreamHandlers = {
    ...handlers,
    onToolCall: (payload) => {
      handlers.onToolCall?.(payload);
      toolPending = true;
      armToolTimeout();
    },
    onToolApproval: (payload) => {
      handlers.onToolApproval?.(payload);
      toolPending = false;
      clearToolTimer();
    },
    onToolResult: (payload) => {
      handlers.onToolResult?.(payload);
      toolPending = false;
      clearToolTimer();
    },
    onDelta: (payload) => {
      handlers.onDelta?.(payload);
      if (toolPending) armToolTimeout();
    },
    onRunState: (payload) => {
      handlers.onRunState?.(payload);
      if (toolPending) armToolTimeout();
    },
  };

  try {
    await consumeAIStream(response, wrappedHandlers);
  } catch (err) {
    if (!timedOut) throw err;
  } finally {
    clearToolTimer();
    signal?.removeEventListener('abort', abortFromCaller);
  }
}

export const chatApi = { chatStream };
