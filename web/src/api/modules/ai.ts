import { ApiRequestError } from '../api';
import apiService from '../api';
import type { ApiResponse } from '../api';

// AI对话消息数据结构
export interface AIMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  run_id?: string;
  client_request_id?: string;
  latest_event_id?: string;
  approval_id?: string;
  resumable?: boolean;
  createdAt?: string;
  updatedAt?: string;
  created_at?: string;
  updated_at?: string;
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

interface TimestampCompatFields {
  createdAt?: string;
  updatedAt?: string;
  created_at?: string;
  updated_at?: string;
}

// AI对话会话数据结构
export interface AISession {
  id: string;
  title: string;
  messages: AIMessage[];
  turns?: AIReplayTurn[];
  createdAt?: string;
  updatedAt?: string;
  created_at?: string;
  updated_at?: string;
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
  createdAt?: string;
  updatedAt?: string;
  created_at?: string;
  updated_at?: string;
}

export interface AIReplayTurn {
  id: string;
  role: 'user' | 'assistant';
  status?: string;
  phase?: string;
  traceId?: string;
  parentTurnId?: string;
  blocks: AIReplayBlock[];
  createdAt?: string;
  updatedAt?: string;
  created_at?: string;
  updated_at?: string;
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
  lastEventId?: string;
  last_event_id?: string;
  message: string;
  scene?: string;
  context?: any;
}

export interface AIRun {
  run_id: string;
  status: string;
  client_request_id?: string;
  latest_event_id?: string;
  approval_id?: string;
  resumable?: boolean;
  assistant_type?: string;
  intent_type?: string;
  progress_summary?: string;
  report?: AIRunReport;
}

export interface AIRunReport {
  id?: string;
  report_id?: string;
  summary?: string;
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
  has_more?: boolean;
  next_cursor?: string;
}

export interface AIRunProjectionQuery {
  cursor?: string;
  limit?: number;
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

export interface A2UIRunResumingEvent {
  approval_id?: string;
  run_id: string;
  session_id: string;
  event_id?: string;
  status?: string;
}

export interface A2UIRunResumedEvent extends A2UIRunResumingEvent {}

export interface A2UIRunResumeFailedEvent extends A2UIRunResumingEvent {
  retryable?: boolean;
  message?: string;
}

export interface A2UIRunCompletedEvent extends A2UIRunResumingEvent {
  status?: string;
}

export interface A2UIRunStateEvent {
  run_id: string;
  status: string;
  agent?: string;
  summary?: string;
  event_id?: string;
}

export interface A2UIApprovalExpiredEvent extends A2UIRunResumingEvent {
  expired?: boolean;
  expires_at?: string;
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

export interface A2UIOpsPlanUpdatedEvent {
  run_id?: string;
  session_id?: string;
  runtime?: Record<string, unknown>;
  snapshot?: Record<string, unknown>;
  todos?: Array<Record<string, unknown>>;
}

function normalizeErrorEvent(payload: unknown): A2UIErrorEvent {
  const errorPayload = { ...((typeof payload === 'object' && payload ? payload : {}) as A2UIErrorEvent) };
  if (!errorPayload.code && errorPayload.error_code) {
    errorPayload.code = errorPayload.error_code;
  }
  if (errorPayload.code === 'AI_STREAM_CURSOR_EXPIRED' || errorPayload.error_code === 'AI_STREAM_CURSOR_EXPIRED') {
    errorPayload.recoverable = true;
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

export interface A2UIUnknownStreamEvent {
  eventType: string;
  payload: unknown;
  eventId?: string;
  runId?: string | number;
  userId?: string | number;
  tenantId?: string | number;
}

export interface A2UIStreamHandlers {
  onEventId?: (eventId: string) => void;
  onMeta?: (payload: A2UIMetaEvent) => void;
  onAgentHandoff?: (payload: A2UIAgentHandoffEvent) => void;
  onPlan?: (payload: A2UIPlanEvent) => void;
  onReplan?: (payload: A2UIReplanEvent) => void;
  onDelta?: (payload: A2UIDeltaEvent) => void;
  onToolCall?: (payload: A2UIToolCallEvent) => void;
  onToolApproval?: (payload: A2UIToolApprovalEvent) => void;
  onToolResult?: (payload: A2UIToolResultEvent) => void;
  onOpsPlanUpdated?: (payload: A2UIOpsPlanUpdatedEvent) => void;
  onRunResuming?: (payload: A2UIRunResumingEvent) => void;
  onRunResumed?: (payload: A2UIRunResumedEvent) => void;
  onRunResumeFailed?: (payload: A2UIRunResumeFailedEvent) => void;
  onRunCompleted?: (payload: A2UIRunCompletedEvent) => void;
  onRunState?: (payload: A2UIRunStateEvent) => void;
  onApprovalExpired?: (payload: A2UIApprovalExpiredEvent) => void;
  onUnknownEvent?: (event: A2UIUnknownStreamEvent) => void;
  onDone?: (payload: A2UIDoneEvent) => void;
  onError?: (payload: A2UIErrorEvent) => void;
}

function readStreamEventTag(
  payload: unknown,
  keys: string[],
): string | number | undefined {
  if (!payload || typeof payload !== 'object') {
    return undefined;
  }

  const record = payload as Record<string, unknown>;
  for (const key of keys) {
    const value = record[key];
    if (typeof value === 'string' || typeof value === 'number') {
      return value;
    }
  }

  return undefined;
}

function buildUnknownStreamEvent(
  eventType: string,
  payload: unknown,
  eventId?: string,
): A2UIUnknownStreamEvent {
  return {
    eventType,
    payload,
    ...(eventId ? { eventId } : {}),
    ...(readStreamEventTag(payload, ['run_id', 'runId']) !== undefined
      ? { runId: readStreamEventTag(payload, ['run_id', 'runId']) }
      : {}),
    ...(readStreamEventTag(payload, ['user_id', 'userId']) !== undefined
      ? { userId: readStreamEventTag(payload, ['user_id', 'userId']) }
      : {}),
    ...(readStreamEventTag(payload, ['tenant_id', 'tenantId']) !== undefined
      ? { tenantId: readStreamEventTag(payload, ['tenant_id', 'tenantId']) }
      : {}),
  };
}

function dispatchAIStreamEvent(
  chunk: string,
  handlers: A2UIStreamHandlers,
) {
  const lines = chunk.split('\n');
  let eventType = 'message';
  let eventId = '';
  const dataLines: string[] = [];

  lines.forEach((line) => {
    if (line.startsWith('id:')) {
      eventId = line.slice(3).trim();
      return;
    }
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
    if (eventId) {
      handlers.onEventId?.(eventId);
    }
    return;
  }

  const rawData = dataLines.join('\n');
  let payload: unknown = rawData;
  try {
    payload = JSON.parse(rawData);
  } catch {
    payload = { message: rawData };
  }

  if (eventId) {
    handlers.onEventId?.(eventId);
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
  } else if (eventType === 'ops_plan_updated') {
    handlers.onOpsPlanUpdated?.(payload as A2UIOpsPlanUpdatedEvent);
  } else if (eventType === 'ai.run.resuming') {
    handlers.onRunResuming?.(payload as A2UIRunResumingEvent);
  } else if (eventType === 'ai.run.resumed') {
    handlers.onRunResumed?.(payload as A2UIRunResumedEvent);
  } else if (eventType === 'ai.run.resume_failed') {
    handlers.onRunResumeFailed?.(payload as A2UIRunResumeFailedEvent);
  } else if (eventType === 'ai.run.completed') {
    handlers.onRunCompleted?.(payload as A2UIRunCompletedEvent);
  } else if (eventType === 'run_state') {
    handlers.onRunState?.(payload as A2UIRunStateEvent);
  } else if (eventType === 'ai.approval.expired') {
    handlers.onApprovalExpired?.(payload as A2UIApprovalExpiredEvent);
  } else {
    handlers.onUnknownEvent?.(buildUnknownStreamEvent(eventType, payload, eventId || undefined));
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
  id?: number;
  approval_id: string;
  checkpoint_id: string;
  session_id: string;
  run_id: string;
  user_id?: number;
  tool_name: string;
  tool_call_id: string;
  arguments_json: string;
  preview_json: string;
  status: 'pending' | 'approved' | 'rejected' | 'expired';
  approved_by?: number;
  disapprove_reason?: string;
  comment?: string;
  timeout_seconds?: number;
  expires_at?: string;
  lock_expires_at?: string;
  matched_rule_id?: number;
  policy_version?: string;
  decision_source?: string;
  decided_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface SubmitApprovalPayload {
  approved: boolean;
  disapprove_reason?: string;
  comment?: string;
}

export interface SubmitApprovalOptions {
  idempotencyKey?: string;
}

export interface SubmitApprovalResult {
  approval_id: string;
  status: string;
  message?: string;
}

export interface RetryResumeApprovalPayload {
  trigger_id: string;
}

export interface RetryResumeApprovalResult {
  approval_id: string;
  status: string;
  message?: string;
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

export interface AILLMProvider {
  id: number;
  name: string;
  provider: 'qwen' | 'ark' | 'ollama' | 'openai' | 'minimax' | string;
  model: string;
  base_url: string;
  api_key_masked?: string;
  api_key_version: number;
  temperature: number;
  thinking: boolean;
  is_default: boolean;
  is_enabled: boolean;
  sort_order: number;
  config_version: number;
  created_at?: string;
  updated_at?: string;
}

export interface AILLMProviderCreatePayload {
  name: string;
  provider: string;
  model: string;
  base_url: string;
  api_key: string;
  temperature?: number;
  thinking?: boolean;
  is_default?: boolean;
  is_enabled?: boolean;
  sort_order?: number;
}

export interface AILLMProviderUpdatePayload {
  name?: string;
  provider?: string;
  model?: string;
  base_url?: string;
  api_key?: string;
  temperature?: number;
  thinking?: boolean;
  is_default?: boolean;
  is_enabled?: boolean;
  sort_order?: number;
}

export interface AILLMProviderImportPayload {
  replace_all?: boolean;
  providers: AILLMProviderCreatePayload[];
}

export interface AILLMProviderImportPreview {
  replace_all: boolean;
  total: number;
  providers: AILLMProvider[];
}

export interface AILLMProviderImportResult {
  replace_all: boolean;
  created: number;
  updated: number;
  providers: AILLMProvider[];
}

export class NotImplementedByBackendError extends Error {
  endpoint: string;

  constructor(operation: string, endpoint: string) {
    super(`NotImplementedByBackend: ${operation} is not exposed by the backend (${endpoint})`);
    this.name = 'NotImplementedByBackendError';
    this.endpoint = endpoint;
  }
}

function notImplementedByBackend(operation: string, endpoint: string): never {
  throw new NotImplementedByBackendError(operation, endpoint);
}

function normalizeTimestampCompat<T extends TimestampCompatFields>(item: T | null | undefined): T | null | undefined {
  if (!item) {
    return item;
  }

  const createdAt = item.createdAt ?? item.created_at;
  const updatedAt = item.updatedAt ?? item.updated_at;

  return {
    ...item,
    ...(createdAt !== undefined ? { createdAt, created_at: createdAt } : {}),
    ...(updatedAt !== undefined ? { updatedAt, updated_at: updatedAt } : {}),
  };
}

function normalizeReplayBlock(block: AIReplayBlock): AIReplayBlock {
  return normalizeTimestampCompat(block) as AIReplayBlock;
}

function normalizeReplayTurn(turn: AIReplayTurn): AIReplayTurn {
  const normalized = normalizeTimestampCompat(turn) as AIReplayTurn;
  return {
    ...normalized,
    blocks: Array.isArray(turn.blocks) ? turn.blocks.map((block) => normalizeReplayBlock(block)) : turn.blocks,
  };
}

function normalizeSession(session: AISession): AISession {
  const normalized = normalizeTimestampCompat(session) as AISession;
  return {
    ...normalized,
    messages: Array.isArray(session.messages)
      ? session.messages.map((message) => normalizeTimestampCompat(message as AIMessage) as AIMessage)
      : session.messages,
    turns: Array.isArray(session.turns) ? session.turns.map((turn) => normalizeReplayTurn(turn)) : session.turns,
  };
}

function normalizeRunReport(report?: AIRunReport | null): AIRunReport | undefined {
  if (!report) {
    return report ?? undefined;
  }

  const reportId = report.report_id ?? report.id;
  if (reportId === undefined) {
    return { ...report };
  }

  return {
    ...report,
    id: reportId,
    report_id: reportId,
  };
}

function normalizeRun(run: AIRun): AIRun {
  return {
    ...run,
    report: normalizeRunReport(run.report),
  };
}

function normalizeSessionResponse(response: ApiResponse<AISession>): ApiResponse<AISession> {
  return {
    ...response,
    data: normalizeSession(response.data),
  };
}

function normalizeSessionListResponse(response: ApiResponse<AISession[]>): ApiResponse<AISession[]> {
  return {
    ...response,
    data: Array.isArray(response.data) ? response.data.map((session) => normalizeSession(session)) : response.data,
  };
}

function normalizeRunResponse(response: ApiResponse<AIRun>): ApiResponse<AIRun> {
  return {
    ...response,
    data: normalizeRun(response.data),
  };
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
        ...(params.lastEventId && !params.last_event_id ? { last_event_id: params.lastEventId } : {}),
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
    const response = await apiService.get<AISession[]>('/ai/sessions', scene ? { params: { scene } } : undefined);
    return normalizeSessionListResponse(response);
  },

  async createSession(params: { title: string; scene: string }): Promise<ApiResponse<AISession>> {
    const response = await apiService.post<AISession>('/ai/sessions', params);
    return normalizeSessionResponse(response);
  },

  async getSession(id: string): Promise<ApiResponse<AISession>> {
    const response = await apiService.get<AISession>(`/ai/sessions/${id}`);
    return normalizeSessionResponse(response);
  },

  async getRunStatus(runId: string): Promise<ApiResponse<AIRun>> {
    const response = await apiService.get<AIRun>(`/ai/runs/${runId}`);
    return normalizeRunResponse(response);
  },

  async getRunProjection(runId: string, query?: AIRunProjectionQuery): Promise<ApiResponse<AIRunProjection>> {
    return apiService.get(`/ai/runs/${runId}/projection`, query ? { params: query } : undefined);
  },

  async getRunContent(id: string): Promise<ApiResponse<AIRunContent>> {
    return apiService.get(`/ai/run-contents/${id}`);
  },

  async getDiagnosisReport(reportId: string): Promise<ApiResponse<AIDiagnosisReport>> {
    return apiService.get(`/ai/diagnosis/${reportId}`);
  },

  async getCurrentSession(scene?: string): Promise<ApiResponse<AISession | null>> {
    notImplementedByBackend('getCurrentSession', '/ai/sessions/current');
  },

  // 获取对话会话详情
  async getSessionDetail(id: string, scene?: string): Promise<ApiResponse<AISession>> {
    const response = await apiService.get<AISession>(`/ai/sessions/${id}`, scene ? { params: { scene } } : undefined);
    return normalizeSessionResponse(response);
  },

  // 从指定消息创建会话分支
  async branchSession(id: string, params?: AISessionBranchParams): Promise<ApiResponse<AISession>> {
    notImplementedByBackend('branchSession', `/ai/sessions/${id}/branch`);
  },

  // 删除对话会话
  async deleteSession(id: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/ai/sessions/${id}`);
  },

  // 重命名对话会话
  async updateSessionTitle(id: string, title: string): Promise<ApiResponse<AISession>> {
    notImplementedByBackend('updateSessionTitle', `/ai/sessions/${id}`);
  },

  async getCapabilities(): Promise<ApiResponse<AICapability[]>> {
    notImplementedByBackend('getCapabilities', '/ai/capabilities');
  },

  async getToolParamHints(name: string): Promise<ApiResponse<AIToolParamHints>> {
    notImplementedByBackend('getToolParamHints', `/ai/tools/${name}/params/hints`);
  },

  async previewTool(params: { tool: string; params?: Record<string, any> }): Promise<ApiResponse<Record<string, any>>> {
    notImplementedByBackend('previewTool', '/ai/tools/preview');
  },

  async executeTool(params: { tool: string; params?: Record<string, any>; approval_token?: string; checkpoint_id?: string }): Promise<ApiResponse<AIToolExecution>> {
    notImplementedByBackend('executeTool', '/ai/tools/execute');
  },

  async getExecution(id: string): Promise<ApiResponse<AIToolExecution>> {
    notImplementedByBackend('getExecution', `/ai/executions/${id}`);
  },

  async listPendingApprovals(): Promise<ApiResponse<ApprovalTicket[]>> {
    return apiService.get('/ai/approvals/pending');
  },

  async submitApproval(id: string, payload: SubmitApprovalPayload, options?: SubmitApprovalOptions): Promise<ApiResponse<SubmitApprovalResult>> {
    const idempotencyKey = options?.idempotencyKey || generateIdempotencyKey();
    const requestConfig = {
      headers: {
        'Idempotency-Key': idempotencyKey,
      },
    };
    try {
      return await apiService.post(`/ai/approvals/${id}/submit`, payload, requestConfig);
    } catch (error) {
      if (!isApprovalNotFoundError(error)) {
        throw error;
      }
      const aliasTicket = await resolveApprovalTicket(id);
      const canonicalID = aliasTicket?.approval_id;
      if (!canonicalID || canonicalID === id) {
        throw error;
      }
      return apiService.post(`/ai/approvals/${canonicalID}/submit`, payload, requestConfig);
    }
  },

  async retryResumeApproval(id: string, payload: RetryResumeApprovalPayload): Promise<ApiResponse<RetryResumeApprovalResult>> {
    return apiService.post(`/ai/approvals/${id}/retry-resume`, payload);
  },

  async getApproval(id: string): Promise<ApiResponse<ApprovalTicket>> {
    return apiService.get(`/ai/approvals/${id}`);
  },

  async submitFeedback(payload: AIKnowledgeFeedbackPayload): Promise<ApiResponse<KnowledgeEntry | null>> {
    notImplementedByBackend('submitFeedback', '/ai/feedback');
  },

  async confirmConfirmation(id: string, approve: boolean): Promise<ApiResponse<ConfirmationTicket>> {
    notImplementedByBackend('confirmConfirmation', `/ai/confirmations/${id}/confirm`);
  },

  async getSceneTools(scene: string): Promise<ApiResponse<AISceneToolsPayload>> {
    notImplementedByBackend('getSceneTools', `/ai/scene/${scene}/tools`);
  },

  async getScenePrompts(scene: string): Promise<ApiResponse<AIScenePromptsPayload>> {
    notImplementedByBackend('getScenePrompts', `/ai/scene/${scene}/prompts`);
  },

  // 获取使用统计概览
  async getUsageStats(params?: UsageStatsParams): Promise<ApiResponse<UsageStats>> {
    notImplementedByBackend('getUsageStats', '/ai/usage/stats');
  },

  // 获取使用日志列表
  async getUsageLogs(params?: UsageLogsParams): Promise<ApiResponse<UsageLogsResult>> {
    notImplementedByBackend('getUsageLogs', '/ai/usage/logs');
  },

  async listAdminModels(): Promise<ApiResponse<{ list: AILLMProvider[]; total: number }>> {
    return apiService.get('/admin/ai/models');
  },

  async getAdminModel(id: number): Promise<ApiResponse<AILLMProvider>> {
    return apiService.get(`/admin/ai/models/${id}`);
  },

  async createAdminModel(payload: AILLMProviderCreatePayload): Promise<ApiResponse<AILLMProvider>> {
    return apiService.post('/admin/ai/models', payload);
  },

  async updateAdminModel(id: number, payload: AILLMProviderUpdatePayload): Promise<ApiResponse<AILLMProvider>> {
    return apiService.put(`/admin/ai/models/${id}`, payload);
  },

  async setAdminDefaultModel(id: number): Promise<ApiResponse<null>> {
    return apiService.put(`/admin/ai/models/${id}/default`);
  },

  async deleteAdminModel(id: number): Promise<ApiResponse<null>> {
    return apiService.delete(`/admin/ai/models/${id}`);
  },

  async previewAdminModelImport(payload: AILLMProviderImportPayload): Promise<ApiResponse<AILLMProviderImportPreview>> {
    return apiService.post('/admin/ai/models/import/preview', payload);
  },

  async importAdminModels(payload: AILLMProviderImportPayload): Promise<ApiResponse<AILLMProviderImportResult>> {
    return apiService.post('/admin/ai/models/import', payload);
  },

};

function generateIdempotencyKey(): string {
  if (typeof globalThis !== 'undefined' && globalThis.crypto && typeof globalThis.crypto.randomUUID === 'function') {
    return globalThis.crypto.randomUUID();
  }
  return `approval-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export function isApprovalConflictError(error: unknown): boolean {
  if (error instanceof ApiRequestError) {
    if (error.statusCode !== 400 && error.statusCode !== 409) {
      return false;
    }
    return /already\s+(approved|rejected|processed|handled)|conflict/i.test(error.message);
  }

  if (typeof error === 'object' && error && 'statusCode' in error) {
    const statusCode = Number((error as { statusCode?: unknown }).statusCode);
    const message = String((error as { message?: unknown }).message || '');
    return (statusCode === 400 || statusCode === 409) &&
      /already\s+(approved|rejected|processed|handled)|conflict/i.test(message);
  }

  return false;
}

function isApprovalNotFoundError(error: unknown): boolean {
  if (error instanceof ApiRequestError) {
    const statusMatched = error.statusCode === 404;
    const businessMatched = error.businessCode === 2005;
    return (statusMatched || businessMatched) && /approval.*not found/i.test(error.message);
  }

  if (typeof error === 'object' && error) {
    const statusCode = Number((error as { statusCode?: unknown }).statusCode);
    const businessCode = Number((error as { businessCode?: unknown; code?: unknown }).businessCode ?? (error as { code?: unknown }).code);
    const message = String((error as { message?: unknown }).message || '');
    return (statusCode === 404 || businessCode === 2005) && /approval.*not found/i.test(message);
  }

  return false;
}

export async function resolveApprovalTicket(approvalId: string): Promise<ApprovalTicket | null> {
  if (!approvalId) {
    return null;
  }

  try {
    const response = await aiApi.getApproval(approvalId);
    return response.data || null;
  } catch {
    try {
      const response = await aiApi.listPendingApprovals();
      return response.data?.find((ticket) =>
        ticket.approval_id === approvalId ||
        ticket.tool_call_id === approvalId ||
        String(ticket.id ?? '') === approvalId
      ) || null;
    } catch {
      return null;
    }
  }
}
