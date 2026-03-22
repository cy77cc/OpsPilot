import apiService from '../api';
import type { ApiResponse } from '../api';

// AI对话消息数据结构
export interface AIMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  run_id?: string;
  turnId?: string;
  content?: string;
  runtime?: Record<string, unknown>;
  thinking?: string;
  rawEvidence?: string[];
  traces?: ToolTrace[];
  recommendations?: EmbeddedRecommendation[];
  thoughtChain?: Array<Record<string, any>>;
  traceId?: string;
  status?: string;
  error_message?: string;
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
  session_id?: string;
  clientRequestId?: string;
  message: string;
  scene?: string;
  context?: any;
}

export interface AIRun {
  run_id: string;
  status: string;
  assistant_type?: string;
  intent_type?: string;
  progress_summary?: string;
  report?: {
    report_id: string;
    summary?: string;
  };
}

export interface AIRunProjectionSummary {
  title?: string;
  content_mode?: string;
  content?: string;
}

export interface AIRunProjectionToolResult {
  event_id?: string;
  status?: string;
  preview?: string;
  result_content_id?: string;
}

export interface AIRunProjectionExecutorItem {
  id: string;
  type: 'content' | 'tool_call';
  content_id?: string;
  start_event_id?: string;
  end_event_id?: string;
  tool_call_id?: string;
  tool_name?: string;
  event_id?: string;
  arguments?: Record<string, unknown>;
  arguments_content_id?: string;
  result?: AIRunProjectionToolResult;
}

export interface AIRunProjectionBlock {
  id: string;
  type: 'agent_handoff' | 'plan' | 'replan' | 'executor' | 'error';
  title: string;
  agent?: string;
  event_ids?: string[];
  steps?: string[];
  data?: Record<string, unknown>;
  items?: AIRunProjectionExecutorItem[];
}

export interface AIRunProjection {
  version: number;
  run_id: string;
  session_id: string;
  status: string;
  summary?: AIRunProjectionSummary;
  blocks: AIRunProjectionBlock[];
}

export interface AIRunContent {
  id: string;
  run_id: string;
  session_id: string;
  content_kind: string;
  encoding: string;
  summary_text?: string;
  body_text?: string;
  body_json?: string;
  size_bytes?: number;
  created_at?: string;
}

export interface AIDiagnosisReport {
  report_id: string;
  run_id?: string;
  session_id?: string;
  summary?: string;
  evidence?: string[];
  root_causes?: string[];
  recommendations?: string[];
  generated_at?: string;
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

export interface A2UIMetaEvent {
  session_id: string;
  run_id: string;
  turn: number;
}

export interface A2UIAgentHandoffEvent {
  from: string;
  to: string;
  intent: 'diagnosis' | 'change' | 'qa' | 'unknown';
}

export interface A2UIPlanEvent {
  steps: string[];
  iteration: number;
}

export interface A2UIReplanEvent {
  steps: string[];
  completed: number;
  iteration: number;
  is_final: boolean;
}

export interface A2UIDeltaEvent {
  content: string;
  agent?: string;
}

export interface A2UIToolCallEvent {
  call_id: string;
  tool_name: string;
  arguments: Record<string, unknown>;
}

export interface A2UIToolApprovalEvent {
  approval_id: string;
  call_id: string;
  tool_name: string;
  preview: Record<string, unknown>;
  timeout_seconds: number;
}

export interface A2UIToolResultEvent {
  call_id: string;
  tool_name: string;
  content: string;
}

function normalizeErrorEvent(payload: unknown): A2UIErrorEvent {
  const errorPayload = { ...((typeof payload === 'object' && payload ? payload : {}) as A2UIErrorEvent) };
  if (!errorPayload.code && errorPayload.error_code) {
    errorPayload.code = errorPayload.error_code;
  }
  return errorPayload;
}

export interface A2UIDoneEvent {
  run_id: string;
  status: 'completed';
  iterations: number;
}

export interface A2UIErrorEvent {
  message: string;
  code?: string;
  error_code?: string;
  recoverable?: boolean;
  run_id?: string;
}

export interface A2UIStreamHandlers {
  onMeta?: (payload: A2UIMetaEvent) => void;
  onAgentHandoff?: (payload: A2UIAgentHandoffEvent) => void;
  onPlan?: (payload: A2UIPlanEvent) => void;
  onReplan?: (payload: A2UIReplanEvent) => void;
  onDelta?: (payload: A2UIDeltaEvent) => void;
  onToolCall?: (payload: A2UIToolCallEvent) => void;
  onToolApproval?: (payload: A2UIToolApprovalEvent) => void;
  onToolResult?: (payload: A2UIToolResultEvent) => void;
  onDone?: (payload: A2UIDoneEvent) => void;
  onError?: (payload: A2UIErrorEvent) => void;
}

function dispatchAIStreamEvent(
  chunk: string,
  handlers: A2UIStreamHandlers,
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

  if (eventType === 'meta') {
    handlers.onMeta?.(payload as A2UIMetaEvent);
  } else if (eventType === 'agent_handoff') {
    handlers.onAgentHandoff?.(payload as A2UIAgentHandoffEvent);
  } else if (eventType === 'plan') {
    handlers.onPlan?.(payload as A2UIPlanEvent);
  } else if (eventType === 'replan') {
    handlers.onReplan?.(payload as A2UIReplanEvent);
  } else if (eventType === 'delta') {
    handlers.onDelta?.(payload as A2UIDeltaEvent);
  } else if (eventType === 'done') {
    handlers.onDone?.(payload as A2UIDoneEvent);
  } else if (eventType === 'error') {
    handlers.onError?.(normalizeErrorEvent(payload));
  } else if (eventType === 'tool_call') {
    handlers.onToolCall?.(payload as A2UIToolCallEvent);
  } else if (eventType === 'tool_approval') {
    handlers.onToolApproval?.(payload as A2UIToolApprovalEvent);
  } else if (eventType === 'tool_result') {
    handlers.onToolResult?.(payload as A2UIToolResultEvent);
  }
}

async function consumeAIStream(response: Response, handlers: A2UIStreamHandlers): Promise<void> {
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
    segments.forEach((segment) => dispatchAIStreamEvent(segment, handlers));
  }

  if (buffer.trim()) {
    dispatchAIStreamEvent(buffer, handlers);
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

export interface SubmitApprovalPayload {
  approved: boolean;
  disapprove_reason?: string;
  comment?: string;
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

export interface UsageStats {
  total_requests: number;
  total_tokens: number;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  total_cost_usd: number;
  avg_first_token_ms: number;
  avg_tokens_per_second: number;
  approval_rate: number;
  approval_pass_rate: number;
  tool_error_rate: number;
  by_scene: SceneStats[];
  by_date: DateStats[];
}

export interface SceneStats {
  scene: string;
  count: number;
  tokens: number;
}

export interface DateStats {
  date: string;
  requests: number;
  tokens: number;
}

export interface UsageLog {
  id: number;
  trace_id: string;
  session_id: string;
  scene: string;
  status: string;
  total_tokens: number;
  duration_ms: number;
  created_at: string;
}

export interface UsageLogsResult {
  total: number;
  items: UsageLog[];
}

export interface UsageStatsParams {
  start_date?: string;
  end_date?: string;
  scene?: string;
}

export interface UsageLogsParams extends UsageStatsParams {
  status?: string;
  page?: number;
  page_size?: number;
}

// AI功能API
export const aiApi = {
  // AI对话（SSE流式）
  async chatStream(params: AIChatParams, handlers: A2UIStreamHandlers, signal?: AbortSignal): Promise<void> {
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
      body: JSON.stringify({
        ...params,
        ...(params.sessionId && !params.session_id ? { session_id: params.sessionId } : {}),
        ...(params.clientRequestId ? { client_request_id: params.clientRequestId } : {}),
      }),
    });

    const wrappedHandlers: A2UIStreamHandlers = {
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
    };

    try {
      await consumeAIStream(response, wrappedHandlers);
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

  async createSession(params: { title: string; scene: string }): Promise<ApiResponse<AISession>> {
    return apiService.post('/ai/sessions', params);
  },

  async getSession(id: string): Promise<ApiResponse<AISession>> {
    return apiService.get(`/ai/sessions/${id}`);
  },

  async getRunStatus(runId: string): Promise<ApiResponse<AIRun>> {
    return apiService.get(`/ai/runs/${runId}`);
  },

  async getRunProjection(runId: string): Promise<ApiResponse<AIRunProjection>> {
    return apiService.get(`/ai/runs/${runId}/projection`);
  },

  async getRunContent(id: string): Promise<ApiResponse<AIRunContent>> {
    return apiService.get(`/ai/run-contents/${id}`);
  },

  async getDiagnosisReport(reportId: string): Promise<ApiResponse<AIDiagnosisReport>> {
    return apiService.get(`/ai/diagnosis/${reportId}`);
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

  async listPendingApprovals(): Promise<ApiResponse<ApprovalTicket[]>> {
    return apiService.get('/ai/approvals/pending');
  },

  async submitApproval(id: string, payload: SubmitApprovalPayload): Promise<ApiResponse<ApprovalTicket>> {
    return apiService.post(`/ai/approvals/${id}/submit`, payload);
  },

  // Deprecated: legacy callers should migrate to submitApproval.
  async confirmApproval(id: string, approve: boolean): Promise<ApiResponse<ApprovalTicket>> {
    return this.submitApproval(id, { approved: approve });
  },

  // Deprecated: prefer listPendingApprovals for the submit-only approval flow.
  async listApprovals(status?: string): Promise<ApiResponse<ApprovalTicket[]>> {
    if (status && status !== 'pending') {
      throw new Error('Only pending approval listing is supported by the current AI approval flow');
    }
    return this.listPendingApprovals();
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

  // 获取使用统计概览
  async getUsageStats(params?: UsageStatsParams): Promise<ApiResponse<UsageStats>> {
    return apiService.get('/ai/usage/stats', params ? { params } : undefined);
  },

  // 获取使用日志列表
  async getUsageLogs(params?: UsageLogsParams): Promise<ApiResponse<UsageLogsResult>> {
    return apiService.get('/ai/usage/logs', params ? { params } : undefined);
  },

};
