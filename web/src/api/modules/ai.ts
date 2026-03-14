import apiService from '../api';
import type { ApiResponse } from '../api';

// AI对话消息数据结构
export interface AIMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  turnId?: string;
  content: string;
  runtime?: Record<string, unknown>;
  thinking?: string;
  rawEvidence?: string[];
  traces?: ToolTrace[];
  recommendations?: EmbeddedRecommendation[];
  thoughtChain?: Array<Record<string, any>>;
  traceId?: string;
  status?: string;
  timestamp: string;
}

// AI对话会话数据结构
export interface AISession {
  id: string;
  title: string;
  messages: AIMessage[];
  turns?: AIReplayTurn[];
  createdAt: string;
  updatedAt: string;
}

export interface AIReplayBlock {
  id: string;
  blockType: string;
  position: number;
  status?: string;
  title?: string;
  contentText?: string;
  contentJson?: Record<string, any>;
  streaming?: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface AIReplayTurn {
  id: string;
  role: 'user' | 'assistant';
  status?: string;
  phase?: string;
  traceId?: string;
  parentTurnId?: string;
  blocks: AIReplayBlock[];
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}

export interface EmbeddedRecommendation {
  id: string;
  type: string;
  title: string;
  content: string;
  followup_prompt?: string;
  reasoning?: string;
  relevance: number;
}

export interface ToolTrace {
  id: string;
  type: 'tool_call' | 'tool_result' | 'tool_missing';
  payload: Record<string, any>;
  timestamp: string;
}

export interface AIKnowledgeFeedbackPayload {
  session_id?: string;
  namespace?: string;
  is_effective: boolean;
  comment?: string;
  question?: string;
  answer?: string;
}

// AI对话请求参数
export interface AIChatParams {
  sessionId?: string;
  message: string;
  context?: any;
}

export interface AISceneToolsPayload {
  scene: string;
  description: string;
  keywords: string[];
  context_hints: string[];
  tools: AICapability[];
}

export interface AIScenePromptItem {
  id: number;
  prompt_text: string;
  prompt_type: string;
  display_order: number;
}

export interface AIScenePromptsPayload {
  scene: string;
  prompts: AIScenePromptItem[];
}

interface SSEMetaEvent {
  sessionId: string;
  createdAt: string;
  turn_id?: string;
}

export interface SSEPlanStep {
  [key: string]: unknown;
  id?: string;
  content?: string;
  title?: string;
  tool_hint?: string;
  status?: string;
  summary?: string;
}

export interface SSEChainStartedEvent {
  turn_id?: string;
}

export interface SSEChainNodeEvent {
  turn_id?: string;
  node_id?: string;
  kind?: 'plan' | 'execute' | 'tool' | 'replan' | 'approval';
  title?: string;
  status?: string;
  headline?: string;
  body?: string;
  structured?: Record<string, unknown>;
  raw?: unknown;
  summary?: string;
  details?: unknown[];
  approval?: Record<string, unknown>;
}

export interface SSEFinalAnswerEvent {
  turn_id?: string;
  chunk?: string;
}

interface SSEDeltaEvent {
  contentChunk: string;
  turn_id?: string;
}

export interface SSEClarifyRequiredEvent {
  kind?: 'clarify';
  title?: string;
  message?: string;
  candidates?: Array<Record<string, unknown>>;
}

export interface SSEReplanStartedEvent {
  reason?: string;
  previous_plan_id?: string;
}

function toContentChunk(payload: unknown): string {
  if (!payload || typeof payload !== 'object') {
    return '';
  }
  const data = payload as Record<string, unknown>;
  const direct = data.contentChunk ?? data.content ?? data.message;
  if (typeof direct === 'string') {
    return direct;
  }
  if (direct == null) {
    return '';
  }
  try {
    return JSON.stringify(direct);
  } catch {
    return String(direct);
  }
}

// normalizeVisibleStreamChunk 将模型内部协议 JSON 转换为用户可见文本。
// - {"steps": [...]} 视为内部计划，不透传
// - {"response": "..."} 解包 response 文本
// - 其他内容原样透传
export function normalizeVisibleStreamChunk(rawChunk: string): string {
  const chunk = typeof rawChunk === 'string' ? rawChunk : '';
  if (!chunk) {
    return '';
  }
  const trimmedForJSON = chunk.trim();
  if (!trimmedForJSON) {
    return '';
  }
  if (!trimmedForJSON.startsWith('{') || !trimmedForJSON.endsWith('}')) {
    return chunk;
  }

  let payload: Record<string, unknown>;
  try {
    payload = JSON.parse(trimmedForJSON) as Record<string, unknown>;
  } catch {
    return chunk;
  }
  if (!payload || typeof payload !== 'object') {
    return chunk;
  }

  const keys = Object.keys(payload);
  const hasOnlyResponseEnvelope = keys.length > 0
    && keys.every((key) => key === 'response' || key === 'reasoning' || key === 'metadata');
  if (hasOnlyResponseEnvelope && typeof payload.response === 'string') {
    return payload.response;
  }

  const hasOnlyStepsEnvelope = keys.length > 0
    && keys.every((key) => key === 'steps' || key === 'plan' || key === 'reasoning' || key === 'metadata');
  if (hasOnlyStepsEnvelope && Array.isArray(payload.steps)) {
    return '';
  }

  return chunk;
}

function normalizeFinalAnswerEventPayload(payload: unknown): SSEFinalAnswerEvent {
  const eventPayload = (typeof payload === 'object' && payload ? payload : {}) as SSEFinalAnswerEvent;
  const normalizedChunk = normalizeVisibleStreamChunk(String(eventPayload.chunk || ''));
  return {
    ...eventPayload,
    ...(normalizedChunk ? { chunk: normalizedChunk } : {}),
  };
}

function normalizeErrorEvent(payload: unknown): SSEErrorEvent {
  const errorPayload = { ...((typeof payload === 'object' && payload ? payload : {}) as SSEErrorEvent) };
  if (!errorPayload.code && errorPayload.error_code) {
    errorPayload.code = errorPayload.error_code;
  }
  return errorPayload;
}

export interface SSEDoneEvent {
  session: AISession;
  stream_state?: 'ok' | 'partial' | 'failed';
  turn_recommendations?: EmbeddedRecommendation[];
  tool_summary?: {
    calls: number;
    results: number;
    missing?: string[];
    missing_call_ids?: string[];
  };
  turn_id?: string;
}

interface SSEErrorEvent {
  message: string;
  code?: string;
  error_code?: string;
  stage?: string;
  recoverable?: boolean;
  tool_summary?: {
    calls: number;
    results: number;
    missing?: string[];
    missing_call_ids?: string[];
  };
  turn_id?: string;
}
interface SSEThinkingEvent {
  contentChunk: string;
  turn_id?: string;
}

export interface AIChatStreamHandlers {
  onChainStarted?: (payload: SSEChainStartedEvent) => void;
  onChainNodeOpen?: (payload: SSEChainNodeEvent) => void;
  onChainNodePatch?: (payload: SSEChainNodeEvent) => void;
  onChainNodeReplace?: (payload: SSEChainNodeEvent) => void;
  onChainNodeClose?: (payload: SSEChainNodeEvent) => void;
  onChainCollapsed?: (payload: SSEChainStartedEvent) => void;
  onFinalAnswerStarted?: (payload: SSEFinalAnswerEvent) => void;
  onFinalAnswerDelta?: (payload: SSEFinalAnswerEvent) => void;
  onFinalAnswerDone?: (payload: SSEFinalAnswerEvent) => void;
  onMeta?: (payload: SSEMetaEvent) => void;
  onDelta?: (payload: SSEDeltaEvent) => void;
  onClarifyRequired?: (payload: SSEClarifyRequiredEvent) => void;
  onReplanStarted?: (payload: SSEReplanStartedEvent) => void;
  onDone?: (payload: SSEDoneEvent) => void;
  onError?: (payload: SSEErrorEvent) => void;
  onThinkingDelta?: (payload: SSEThinkingEvent) => void;
  onToolCall?: (payload: { turn_id?: string; call_id?: string; tool?: string; payload?: Record<string, any>; ts?: string; tool_calls?: Array<{ function?: { name?: string; arguments?: string } }> }) => void;
  onToolResult?: (payload: { turn_id?: string; call_id?: string; tool?: string; payload?: Record<string, any>; result?: { ok: boolean; data?: any; error?: string; error_code?: string; source?: string; latency_ms?: number }; ts?: string }) => void;
  onHeartbeat?: (payload: { turn_id?: string; status?: string }) => void;
}

function dispatchAIStreamEvent(
  chunk: string,
  handlers: AIChatStreamHandlers,
  options?: { normalizeVisibleDelta?: boolean },
) {
  const lines = chunk.split('\n');
  let eventType = 'message';
  const dataLines: string[] = [];

  lines.forEach((line) => {
    if (line.startsWith('event:')) {
      eventType = line.slice(6).trim();
      return;
    }
    if (line.startsWith('data:')) {
      const value = line.slice(5);
      dataLines.push(value.startsWith(' ') ? value.slice(1) : value);
    }
  });

  if (dataLines.length === 0) {
    return;
  }

  const rawData = dataLines.join('\n');
  let payload: unknown = rawData;
  try {
    payload = JSON.parse(rawData);
  } catch {
    payload = { message: rawData };
  }

  const normalizeVisibleDelta = options?.normalizeVisibleDelta ?? false;
  if (eventType === 'meta') {
    handlers.onMeta?.(payload as SSEMetaEvent);
  } else if (eventType === 'chain_started') {
    handlers.onChainStarted?.(payload as SSEChainStartedEvent);
  } else if (eventType === 'chain_node_open') {
    handlers.onChainNodeOpen?.(payload as SSEChainNodeEvent);
  } else if (eventType === 'chain_node_patch') {
    handlers.onChainNodePatch?.(payload as SSEChainNodeEvent);
  } else if (eventType === 'chain_node_replace') {
    handlers.onChainNodeReplace?.(payload as SSEChainNodeEvent);
  } else if (eventType === 'chain_node_close') {
    handlers.onChainNodeClose?.(payload as SSEChainNodeEvent);
  } else if (eventType === 'chain_collapsed') {
    handlers.onChainCollapsed?.(payload as SSEChainStartedEvent);
  } else if (eventType === 'final_answer_started') {
    handlers.onFinalAnswerStarted?.(payload as SSEFinalAnswerEvent);
  } else if (eventType === 'final_answer_delta') {
    const normalized = normalizeFinalAnswerEventPayload(payload);
    if (typeof normalized.chunk === 'string' && normalized.chunk.length > 0) {
      handlers.onFinalAnswerDelta?.(normalized);
    }
  } else if (eventType === 'final_answer_done') {
    handlers.onFinalAnswerDone?.(payload as SSEFinalAnswerEvent);
  } else if (eventType === 'delta' || eventType === 'message') {
    const contentChunk = normalizeVisibleDelta
      ? normalizeVisibleStreamChunk(toContentChunk(payload))
      : toContentChunk(payload);
    if (contentChunk) {
      handlers.onDelta?.({
        ...(typeof payload === 'object' && payload ? payload as Record<string, unknown> : {}),
        contentChunk,
      } as SSEDeltaEvent);
    }
  } else if (eventType === 'done') {
    handlers.onDone?.(payload as SSEDoneEvent);
  } else if (eventType === 'error') {
    handlers.onError?.(normalizeErrorEvent(payload));
  } else if (eventType === 'thinking_delta') {
    handlers.onThinkingDelta?.(payload as SSEThinkingEvent);
  } else if (eventType === 'tool_call') {
    handlers.onToolCall?.(payload as any);
  } else if (eventType === 'tool_result') {
    handlers.onToolResult?.(payload as any);
  } else if (eventType === 'clarify_required') {
    handlers.onClarifyRequired?.(payload as SSEClarifyRequiredEvent);
  } else if (eventType === 'replan_started') {
    handlers.onReplanStarted?.(payload as SSEReplanStartedEvent);
  } else if (eventType === 'heartbeat') {
    handlers.onHeartbeat?.(payload as { turn_id?: string; status?: string });
  }
}

async function consumeAIStream(response: Response, handlers: AIChatStreamHandlers, options?: { normalizeVisibleDelta?: boolean }): Promise<void> {
  if (!response.ok || !response.body) {
    throw new Error(`请求失败: ${response.status}`);
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder('utf-8');
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    buffer += decoder.decode(value, { stream: true }).replace(/\r/g, '');
    const segments = buffer.split('\n\n');
    buffer = segments.pop() || '';
    segments.forEach((segment) => dispatchAIStreamEvent(segment, handlers, options));
  }

  if (buffer.trim()) {
    dispatchAIStreamEvent(buffer, handlers, options);
  }
}

export type RiskLevel = 'low' | 'medium' | 'high';

export interface AICapability {
  name: string;
  description: string;
  mode: 'readonly' | 'mutating';
  risk: RiskLevel;
  provider: 'local' | 'mcp';
  schema?: Record<string, any>;
  permission?: string;
  required?: string[];
  enum_sources?: Record<string, string>;
  param_hints?: Record<string, string>;
  related_tools?: string[];
  scene_scope?: string[];
}

export interface AIToolParamHintValue {
  value: any;
  label: string;
}

export interface AIToolParamHintItem {
  type?: string;
  required: boolean;
  default?: any;
  hint?: string;
  enum_source?: string;
  values?: AIToolParamHintValue[];
}

export interface AIToolParamHints {
  tool: string;
  params: Record<string, AIToolParamHintItem>;
}

export interface ToolCallTrace {
  tool: string;
  params?: Record<string, any>;
  at?: string;
}

export interface ApprovalTicket {
  id: string;
  session_id?: string;
  plan_id?: string;
  step_id?: string;
  checkpoint_id?: string;
  tool?: string;
  tool_name?: string;
  params?: Record<string, any>;
  params_json?: string;
  risk?: RiskLevel;
  risk_level?: RiskLevel;
  mode?: 'readonly' | 'mutating';
  status: 'pending' | 'approved' | 'rejected' | 'expired' | 'executed' | 'failed';
  createdAt?: string;
  created_at?: string;
  expiresAt?: string;
  expires_at?: string;
  approval_token?: string;
  target_resource_type?: string;
  target_resource_id?: string;
  target_resource_name?: string;
  task_detail_json?: string;
  resume?: {
    session_id?: string;
    plan_id?: string;
    step_id?: string;
    checkpoint_id?: string;
  };
}

export interface KnowledgeEntry {
  id: string;
  source: 'user_input' | 'feedback';
  namespace: string;
  question: string;
  answer: string;
  created_at?: string;
}

export interface ConfirmationTicket {
  id: string;
  request_user_id: number;
  tool_name: string;
  tool_mode: string;
  risk_level: string;
  status: 'pending' | 'confirmed' | 'cancelled' | 'expired';
  expires_at: string;
  confirmed_at?: string;
  cancelled_at?: string;
}

export interface AIToolExecution {
  id: string;
  tool: string;
  params: Record<string, any>;
  mode: 'readonly' | 'mutating';
  status: 'running' | 'succeeded' | 'failed';
  approvalId?: string;
  createdAt: string;
  finishedAt?: string;
  error?: string;
  result?: {
    ok: boolean;
    data?: any;
    error?: string;
    source: string;
    latency_ms: number;
  };
}

export interface AIHostExecutionPlan {
  execution_id: string;
  command_id?: string;
  host_ids: number[];
  mode: 'command' | 'script';
  command?: string;
  script_path?: string;
  risk: 'low' | 'medium' | 'high';
}

export interface AIHostExecutionResult {
  execution_id: string;
  host_id: number;
  host_ip: string;
  host_name: string;
  status: 'running' | 'succeeded' | 'failed';
  stdout: string;
  stderr: string;
  exit_code: number;
  started_at?: string;
  finished_at?: string;
}

export interface AISessionBranchParams {
  messageId?: string;
  title?: string;
}

// AI功能API
export const aiApi = {
  // AI对话（SSE流式）
  async chatStream(params: AIChatParams, handlers: AIChatStreamHandlers, signal?: AbortSignal): Promise<void> {
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
      if (softTimeoutTimer !== null) {
        window.clearTimeout(softTimeoutTimer);
        softTimeoutTimer = null;
      }
      if (hardTimeoutTimer !== null) {
        window.clearTimeout(hardTimeoutTimer);
        hardTimeoutTimer = null;
      }
      softWarned = false;
    };

    const armToolTimeout = () => {
      clearToolTimer();
      softTimeoutTimer = window.setTimeout(() => {
        if (softWarned) {
          return;
        }
        softWarned = true;
        handlers.onError?.({
          code: 'tool_timeout_soft',
          recoverable: true,
          message: '工具执行较慢，正在继续等待结果…',
        });
      }, 25000);
      hardTimeoutTimer = window.setTimeout(() => {
        timedOut = true;
        handlers.onError?.({
          code: 'tool_timeout_hard',
          recoverable: true,
          message: '工具调用超时，请重试本轮对话。',
        });
        controller.abort();
      }, 55000);
    };

    const touchActivity = () => {
      if (toolPending) {
        armToolTimeout();
      }
    };

    const abortFromCaller = () => controller.abort();
    signal?.addEventListener('abort', abortFromCaller, { once: true });

    const response = await fetch(`${base}/ai/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...(projectId ? { 'X-Project-ID': projectId } : {}),
      },
      signal: controller.signal,
      body: JSON.stringify(params),
    });

    const wrappedHandlers: AIChatStreamHandlers = {
      ...handlers,
      onDone: (payload) => {
        handlers.onDone?.(payload);
        toolPending = false;
        clearToolTimer();
      },
      onError: (payload) => {
        handlers.onError?.(payload);
        if (payload.code !== 'tool_timeout_soft') {
          toolPending = false;
          clearToolTimer();
        }
      },
      onThinkingDelta: (payload) => {
        handlers.onThinkingDelta?.(payload);
        touchActivity();
      },
      onToolCall: (payload) => {
        handlers.onToolCall?.(payload);
        toolPending = true;
        armToolTimeout();
      },
      onToolResult: (payload) => {
        handlers.onToolResult?.(payload);
        toolPending = false;
        clearToolTimer();
      },
      onClarifyRequired: (payload) => {
        handlers.onClarifyRequired?.(payload);
        toolPending = false;
        clearToolTimer();
      },
      onHeartbeat: (payload) => {
        handlers.onHeartbeat?.(payload);
        touchActivity();
      },
    };

    try {
      await consumeAIStream(response, wrappedHandlers, { normalizeVisibleDelta: true });
    } catch (err) {
      if (!timedOut) {
        throw err;
      }
    } finally {
      clearToolTimer();
      signal?.removeEventListener('abort', abortFromCaller);
    }
  },

  // 获取对话会话列表
  async getSessions(scene?: string): Promise<ApiResponse<AISession[]>> {
    return apiService.get('/ai/sessions', scene ? { params: { scene } } : undefined);
  },

  async getCurrentSession(scene?: string): Promise<ApiResponse<AISession | null>> {
    return apiService.get('/ai/sessions/current', scene ? { params: { scene } } : undefined);
  },

  // 获取对话会话详情
  async getSessionDetail(id: string, scene?: string): Promise<ApiResponse<AISession>> {
    return apiService.get(`/ai/sessions/${id}`, scene ? { params: { scene } } : undefined);
  },

  // 从指定消息创建会话分支
  async branchSession(id: string, params?: AISessionBranchParams): Promise<ApiResponse<AISession>> {
    return apiService.post(`/ai/sessions/${id}/branch`, params || {});
  },

  // 删除对话会话
  async deleteSession(id: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/ai/sessions/${id}`);
  },

  // 重命名对话会话
  async updateSessionTitle(id: string, title: string): Promise<ApiResponse<AISession>> {
    return apiService.patch(`/ai/sessions/${id}`, { title });
  },

  async getCapabilities(): Promise<ApiResponse<AICapability[]>> {
    return apiService.get('/ai/capabilities');
  },

  async getToolParamHints(name: string): Promise<ApiResponse<AIToolParamHints>> {
    return apiService.get(`/ai/tools/${name}/params/hints`);
  },

  async previewTool(params: { tool: string; params?: Record<string, any> }): Promise<ApiResponse<Record<string, any>>> {
    return apiService.post('/ai/tools/preview', params);
  },

  async executeTool(params: { tool: string; params?: Record<string, any>; approval_token?: string; checkpoint_id?: string }): Promise<ApiResponse<AIToolExecution>> {
    return apiService.post('/ai/tools/execute', params);
  },

  async getExecution(id: string): Promise<ApiResponse<AIToolExecution>> {
    return apiService.get(`/ai/executions/${id}`);
  },

  async createApproval(params: { tool: string; params?: Record<string, any> }): Promise<ApiResponse<ApprovalTicket>> {
    return apiService.post('/ai/approvals', params);
  },

  async confirmApproval(id: string, approve: boolean): Promise<ApiResponse<ApprovalTicket>> {
    return apiService.post(`/ai/approvals/${id}/confirm`, { approve });
  },

  async decideChainApproval(chainId: string, nodeId: string, approved: boolean, reason?: string): Promise<ApiResponse<{ approval?: Record<string, unknown>; execution?: Record<string, unknown> }>> {
    return apiService.post(`/ai/chains/${chainId}/approvals/${nodeId}/decision`, {
      approved,
      ...(reason ? { reason } : {}),
    });
  },

  async decideChainApprovalStream(
    chainId: string,
    nodeId: string,
    approved: boolean,
    handlers: AIChatStreamHandlers,
    reason?: string,
  ): Promise<void> {
    const base = import.meta.env.VITE_API_BASE || '/api/v1';
    const token = localStorage.getItem('token');
    const projectId = localStorage.getItem('projectId');

    const response = await fetch(`${base}/ai/chains/${chainId}/approvals/${nodeId}/decision`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...(projectId ? { 'X-Project-ID': projectId } : {}),
      },
      body: JSON.stringify({
        approved,
        ...(reason ? { reason } : {}),
      }),
    });

    await consumeAIStream(response, handlers, { normalizeVisibleDelta: true });
  },

  async listApprovals(status?: string): Promise<ApiResponse<ApprovalTicket[]>> {
    return apiService.get('/ai/approvals', status ? { params: { status } } : undefined);
  },

  async getApproval(id: string): Promise<ApiResponse<ApprovalTicket>> {
    return apiService.get(`/ai/approvals/${id}`);
  },

  async submitFeedback(payload: AIKnowledgeFeedbackPayload): Promise<ApiResponse<KnowledgeEntry | null>> {
    return apiService.post('/ai/feedback', payload);
  },

  async confirmConfirmation(id: string, approve: boolean): Promise<ApiResponse<ConfirmationTicket>> {
    return apiService.post(`/ai/confirmations/${id}/confirm`, { approve });
  },

  async getSceneTools(scene: string): Promise<ApiResponse<AISceneToolsPayload>> {
    return apiService.get(`/ai/scene/${scene}/tools`);
  },

  async getScenePrompts(scene: string): Promise<ApiResponse<AIScenePromptsPayload>> {
    return apiService.get(`/ai/scene/${scene}/prompts`);
  },

};
