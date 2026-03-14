import apiService from '../api';
import type { ApiResponse } from '../api';

// AI对话消息数据结构
export interface AIMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
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
  type: 'tool_call' | 'tool_result' | 'approval_required' | 'tool_missing';
  payload: Record<string, any>;
  timestamp: string;
}

export interface AIInterruptApprovalResponse {
  session_id?: string;
  plan_id?: string;
  step_id?: string;
  checkpoint_id?: string;
  approved: boolean;
  reason?: string;
}

export interface AIInterruptApprovalResult {
  resumed: boolean;
  interrupted?: boolean;
  content?: string;
  sessionId?: string;
  session_id?: string;
  plan_id?: string;
  step_id?: string;
  checkpoint_id?: string;
  status?: string;
  interrupt_targets?: string[];
  interrupt_contexts?: any[];
  approval_required?: boolean;
  review_required?: boolean;
  tool?: string;
  arguments?: string;
  risk?: string;
  preview?: Record<string, any>;
  message?: string;
  interrupt_error?: string;
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

export interface SSETurnStartedEvent {
  turn_id: string;
  role?: string;
  phase?: string;
  status?: string;
}

export interface SSEBlockOpenEvent {
  turn_id: string;
  block_id: string;
  block_type: string;
  position?: number;
  status?: string;
  phase?: string;
  title?: string;
  payload?: Record<string, unknown>;
}

export interface SSEBlockDeltaEvent {
  turn_id: string;
  block_id: string;
  block_type?: string;
  patch?: Record<string, unknown>;
}

export interface SSEBlockReplaceEvent {
  turn_id: string;
  block_id: string;
  block_type?: string;
  payload?: Record<string, unknown>;
}

export interface SSEBlockCloseEvent {
  turn_id: string;
  block_id: string;
  status?: string;
}

export interface SSETurnStateEvent {
  turn_id: string;
  status?: string;
  phase?: string;
}

export interface SSETurnDoneEvent {
  turn_id: string;
  status?: string;
  phase?: string;
}

export interface SSEPhaseStartedEvent {
  [key: string]: unknown;
  turn_id?: string;
  phase?: string;
  status?: string;
  title?: string;
  summary?: string;
  user_visible_summary?: string;
  message?: string;
}

export interface SSEPhaseCompleteEvent extends SSEPhaseStartedEvent {}

export interface SSEPlanStep {
  [key: string]: unknown;
  id?: string;
  content?: string;
  title?: string;
  tool_hint?: string;
  status?: string;
  summary?: string;
}

export interface SSEPlanGeneratedEvent {
  [key: string]: unknown;
  turn_id?: string;
  session_id?: string;
  plan_id?: string;
  title?: string;
  summary?: string;
  user_visible_summary?: string;
  total?: number;
  steps?: SSEPlanStep[];
  plan?: Record<string, unknown>;
}

export interface SSEStepStartedEvent extends SSEStepUpdateEvent {}

export interface SSEStepCompleteEvent extends SSEStepUpdateEvent {}

export interface SSEReplanTriggeredEvent {
  [key: string]: unknown;
  turn_id?: string;
  session_id?: string;
  plan_id?: string;
  previous_plan_id?: string;
  reason?: string;
  title?: string;
  summary?: string;
  user_visible_summary?: string;
  completed_steps?: number;
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

export interface SSERewriteResultEvent {
  rewrite?: Record<string, unknown>;
  user_visible_summary?: string;
}

export interface SSEPlannerStateEvent {
  status?: string;
  user_visible_summary?: string;
}

export interface SSEPlanCreatedEvent {
  plan?: Record<string, unknown>;
  user_visible_summary?: string;
}

export interface SSEStageDeltaEvent {
  stage?: string;
  status?: string;
  session_id?: string;
  plan_id?: string;
  step_id?: string;
  checkpoint_id?: string;
  content_chunk?: string;
  contentChunk?: string;
  content?: string;
  detail?: string;
  details?: string[];
  summary?: string;
  user_visible_summary?: string;
  message?: string;
  title?: string;
  description?: string;
  steps?: string[];
  replace?: boolean;
}

export interface SSEStepUpdateEvent {
  session_id?: string;
  plan_id?: string;
  step_id?: string;
  checkpoint_id?: string;
  status?: string;
  title?: string;
  expert?: string;
  tool?: string;
  tool_name?: string;
  params?: Record<string, unknown>;
  result?: {
    ok?: boolean;
    data?: unknown;
    error?: string;
    latency_ms?: number;
  };
  error?: string;
  summary?: string;
  user_visible_summary?: string;
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

export interface SSESummaryEvent {
  summary?: string;
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
  const trimmed = chunk.trim();
  if (!trimmed) {
    return '';
  }
  if (!trimmed.startsWith('{') || !trimmed.endsWith('}')) {
    return chunk;
  }

  let payload: Record<string, unknown>;
  try {
    payload = JSON.parse(trimmed) as Record<string, unknown>;
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

function normalizeResumeIdentity<T extends Record<string, unknown>>(payload: T): T {
  const resume = (payload.resume || {}) as Record<string, unknown>;
  const sessionID = payload.session_id ?? resume.session_id;
  const planID = payload.plan_id ?? resume.plan_id;
  const stepID = payload.step_id ?? resume.step_id;
  const checkpointID = payload.checkpoint_id ?? resume.checkpoint_id;

  return {
    ...payload,
    ...(sessionID ? { session_id: String(sessionID) } : {}),
    ...(planID ? { plan_id: String(planID) } : {}),
    ...(stepID ? { step_id: String(stepID) } : {}),
    ...(checkpointID ? { checkpoint_id: String(checkpointID) } : {}),
  };
}

function normalizeStageDeltaEvent(payload: unknown): SSEStageDeltaEvent {
  const base = typeof payload === 'object' && payload ? normalizeResumeIdentity(payload as Record<string, unknown>) : {};
  const contentChunk = toContentChunk(base);
  return {
    ...(base as SSEStageDeltaEvent),
    ...(contentChunk ? { contentChunk, content_chunk: typeof base.content_chunk === 'string' ? base.content_chunk : contentChunk } : {}),
  };
}

function normalizeStepUpdateEvent(payload: unknown): SSEStepUpdateEvent {
  return normalizeResumeIdentity((typeof payload === 'object' && payload ? payload : {}) as Record<string, unknown>) as SSEStepUpdateEvent;
}

function normalizeApprovalRequiredEvent(payload: unknown): ApprovalRequiredEvent {
  return normalizeResumeIdentity((typeof payload === 'object' && payload ? payload : {}) as Record<string, unknown>) as unknown as ApprovalRequiredEvent;
}

function normalizePhaseEvent<T extends Record<string, unknown>>(payload: unknown): T {
  return normalizeResumeIdentity((typeof payload === 'object' && payload ? payload : {}) as Record<string, unknown>) as T;
}

function normalizePlanGeneratedEvent(payload: unknown): SSEPlanGeneratedEvent {
  return normalizePhaseEvent<SSEPlanGeneratedEvent>(payload);
}

function normalizeReplanTriggeredEvent(payload: unknown): SSEReplanTriggeredEvent {
  return normalizePhaseEvent<SSEReplanTriggeredEvent>(payload);
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
  onChainNodeClose?: (payload: SSEChainNodeEvent) => void;
  onChainCollapsed?: (payload: SSEChainStartedEvent) => void;
  onFinalAnswerStarted?: (payload: SSEFinalAnswerEvent) => void;
  onFinalAnswerDelta?: (payload: SSEFinalAnswerEvent) => void;
  onFinalAnswerDone?: (payload: SSEFinalAnswerEvent) => void;
  onMeta?: (payload: SSEMetaEvent) => void;
  onTurnStarted?: (payload: SSETurnStartedEvent) => void;
  onBlockOpen?: (payload: SSEBlockOpenEvent) => void;
  onBlockDelta?: (payload: SSEBlockDeltaEvent) => void;
  onBlockReplace?: (payload: SSEBlockReplaceEvent) => void;
  onBlockClose?: (payload: SSEBlockCloseEvent) => void;
  onTurnState?: (payload: SSETurnStateEvent) => void;
  onTurnDone?: (payload: SSETurnDoneEvent) => void;
  onPhaseStarted?: (payload: SSEPhaseStartedEvent) => void;
  onPhaseComplete?: (payload: SSEPhaseCompleteEvent) => void;
  onRewriteResult?: (payload: SSERewriteResultEvent) => void;
  onPlanGenerated?: (payload: SSEPlanGeneratedEvent) => void;
  onPlannerState?: (payload: SSEPlannerStateEvent) => void;
  onPlanCreated?: (payload: SSEPlanCreatedEvent) => void;
  onStageDelta?: (payload: SSEStageDeltaEvent) => void;
  onStepStarted?: (payload: SSEStepStartedEvent) => void;
  onStepComplete?: (payload: SSEStepCompleteEvent) => void;
  onStepUpdate?: (payload: SSEStepUpdateEvent) => void;
  onDelta?: (payload: SSEDeltaEvent) => void;
  onClarifyRequired?: (payload: SSEClarifyRequiredEvent) => void;
  onReplanTriggered?: (payload: SSEReplanTriggeredEvent) => void;
  onReplanStarted?: (payload: SSEReplanStartedEvent) => void;
  onSummary?: (payload: SSESummaryEvent) => void;
  onDone?: (payload: SSEDoneEvent) => void;
  onError?: (payload: SSEErrorEvent) => void;
  onThinkingDelta?: (payload: SSEThinkingEvent) => void;
  onToolCall?: (payload: { turn_id?: string; call_id?: string; tool?: string; payload?: Record<string, any>; ts?: string; tool_calls?: Array<{ function?: { name?: string; arguments?: string } }> }) => void;
  onToolResult?: (payload: { turn_id?: string; call_id?: string; tool?: string; payload?: Record<string, any>; result?: { ok: boolean; data?: any; error?: string; error_code?: string; source?: string; latency_ms?: number }; ts?: string }) => void;
  onApprovalRequired?: (payload: ApprovalRequiredEvent) => void;
  onHeartbeat?: (payload: { turn_id?: string; status?: string }) => void;
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

export interface ApprovalRequiredEvent extends ApprovalTicket {
  turn_id?: string;
  approval_required?: boolean;
  previewDiff?: string;
  title?: string;
  user_visible_summary?: string;
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

    if (!response.ok || !response.body) {
      throw new Error(`请求失败: ${response.status}`);
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder('utf-8');
    let buffer = '';

    const dispatchEvent = (chunk: string) => {
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
      let payload: unknown = rawData;
      try {
        payload = JSON.parse(rawData);
      } catch {
        payload = { message: rawData };
      }

      if (eventType === 'meta') {
        handlers.onMeta?.(payload as SSEMetaEvent);
      } else if (eventType === 'chain_started') {
        handlers.onChainStarted?.(payload as SSEChainStartedEvent);
      } else if (eventType === 'chain_node_open') {
        handlers.onChainNodeOpen?.(payload as SSEChainNodeEvent);
      } else if (eventType === 'chain_node_patch') {
        handlers.onChainNodePatch?.(payload as SSEChainNodeEvent);
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
      } else if (eventType === 'turn_started') {
        handlers.onTurnStarted?.(payload as SSETurnStartedEvent);
      } else if (eventType === 'block_open') {
        handlers.onBlockOpen?.(payload as SSEBlockOpenEvent);
      } else if (eventType === 'block_delta') {
        handlers.onBlockDelta?.(payload as SSEBlockDeltaEvent);
      } else if (eventType === 'block_replace') {
        handlers.onBlockReplace?.(payload as SSEBlockReplaceEvent);
      } else if (eventType === 'block_close') {
        handlers.onBlockClose?.(payload as SSEBlockCloseEvent);
      } else if (eventType === 'turn_state') {
        handlers.onTurnState?.(payload as SSETurnStateEvent);
      } else if (eventType === 'turn_done') {
        handlers.onTurnDone?.(payload as SSETurnDoneEvent);
      } else if (eventType === 'rewrite_result') {
        handlers.onRewriteResult?.(payload as SSERewriteResultEvent);
      } else if (eventType === 'planner_state') {
        handlers.onPlannerState?.(payload as SSEPlannerStateEvent);
      } else if (eventType === 'plan_created') {
        handlers.onPlanCreated?.(payload as SSEPlanCreatedEvent);
      } else if (eventType === 'stage_delta') {
        handlers.onStageDelta?.(normalizeStageDeltaEvent(payload));
      } else if (eventType === 'step_update') {
        handlers.onStepUpdate?.(normalizeStepUpdateEvent(payload));
      } else if (eventType === 'delta' || eventType === 'message') {
        const contentChunk = normalizeVisibleStreamChunk(toContentChunk(payload));
        if (contentChunk) {
          handlers.onDelta?.({
            ...(typeof payload === 'object' && payload ? payload as Record<string, unknown> : {}),
            contentChunk,
          } as SSEDeltaEvent);
        }
      } else if (eventType === 'done') {
        handlers.onDone?.(payload as SSEDoneEvent);
        toolPending = false;
        clearToolTimer();
      } else if (eventType === 'error') {
        const errorPayload = normalizeErrorEvent(payload);
        handlers.onError?.(errorPayload);
        const err = errorPayload;
        if (err.code !== 'tool_timeout_soft') {
          toolPending = false;
          clearToolTimer();
        }
      } else if (eventType === 'thinking_delta') {
        handlers.onThinkingDelta?.(payload as SSEThinkingEvent);
        touchActivity();
      } else if (eventType === 'tool_call') {
        handlers.onToolCall?.(payload as { tool: string; params?: Record<string, any> });
        toolPending = true;
        armToolTimeout();
      } else if (eventType === 'tool_result') {
        handlers.onToolResult?.(payload as { tool?: string; result?: { ok: boolean; data?: any; error?: string; source?: string; latency_ms?: number } });
        toolPending = false;
        clearToolTimer();
      } else if (eventType === 'clarify_required') {
        handlers.onClarifyRequired?.(payload as SSEClarifyRequiredEvent);
        toolPending = false;
        clearToolTimer();
      } else if (eventType === 'replan_started') {
        handlers.onReplanStarted?.(payload as SSEReplanStartedEvent);
      } else if (eventType === 'summary') {
        handlers.onSummary?.(payload as SSESummaryEvent);
      } else if (eventType === 'heartbeat') {
        handlers.onHeartbeat?.(payload as { turn_id?: string; status?: string });
        touchActivity();
      }
    };

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) {
          break;
        }

        buffer += decoder.decode(value, { stream: true }).replace(/\r/g, '');
        const segments = buffer.split('\n\n');
        buffer = segments.pop() || '';
        segments.forEach(dispatchEvent);
        touchActivity();
      }
    } catch (err) {
      if (!timedOut) {
        throw err;
      }
    } finally {
      clearToolTimer();
      signal?.removeEventListener('abort', abortFromCaller);
    }

    if (buffer.trim()) {
      dispatchEvent(buffer);
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

  async listApprovals(status?: string): Promise<ApiResponse<ApprovalTicket[]>> {
    return apiService.get('/ai/approvals', status ? { params: { status } } : undefined);
  },

  async getApproval(id: string): Promise<ApiResponse<ApprovalTicket>> {
    return apiService.get(`/ai/approvals/${id}`);
  },

  async approveApproval(id: string, reason?: string): Promise<ApiResponse<{ task: ApprovalTicket; execution?: AIToolExecution }>> {
    return apiService.post(`/ai/approvals/${id}/approve`, reason ? { reason } : {});
  },

  async rejectApproval(id: string, reason?: string): Promise<ApiResponse<{ task: ApprovalTicket; execution?: AIToolExecution }>> {
    return apiService.post(`/ai/approvals/${id}/reject`, reason ? { reason } : {});
  },

  async submitFeedback(payload: AIKnowledgeFeedbackPayload): Promise<ApiResponse<KnowledgeEntry | null>> {
    return apiService.post('/ai/feedback', payload);
  },

  async respondApproval(params: AIInterruptApprovalResponse): Promise<ApiResponse<AIInterruptApprovalResult>> {
    return apiService.post('/ai/resume/step', params);
  },

  async respondApprovalStream(params: AIInterruptApprovalResponse, handlers: AIChatStreamHandlers): Promise<void> {
    const base = import.meta.env.VITE_API_BASE || '/api/v1';
    const token = localStorage.getItem('token');
    const projectId = localStorage.getItem('projectId');

    const response = await fetch(`${base}/ai/resume/step/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...(projectId ? { 'X-Project-ID': projectId } : {}),
      },
      body: JSON.stringify(params),
    });

    if (!response.ok || !response.body) {
      throw new Error(`请求失败: ${response.status}`);
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder('utf-8');
    let buffer = '';

    const dispatchEvent = (chunk: string) => {
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
      let payload: unknown = rawData;
      try {
        payload = JSON.parse(rawData);
      } catch {
        payload = { message: rawData };
      }

      if (eventType === 'meta') {
        handlers.onMeta?.(payload as SSEMetaEvent);
      } else if (eventType === 'turn_started') {
        handlers.onTurnStarted?.(payload as SSETurnStartedEvent);
      } else if (eventType === 'block_open') {
        handlers.onBlockOpen?.(payload as SSEBlockOpenEvent);
      } else if (eventType === 'block_delta') {
        handlers.onBlockDelta?.(payload as SSEBlockDeltaEvent);
      } else if (eventType === 'block_replace') {
        handlers.onBlockReplace?.(payload as SSEBlockReplaceEvent);
      } else if (eventType === 'block_close') {
        handlers.onBlockClose?.(payload as SSEBlockCloseEvent);
      } else if (eventType === 'turn_state') {
        handlers.onTurnState?.(payload as SSETurnStateEvent);
      } else if (eventType === 'turn_done') {
        handlers.onTurnDone?.(payload as SSETurnDoneEvent);
      } else if (eventType === 'stage_delta') {
        handlers.onStageDelta?.(normalizeStageDeltaEvent(payload));
      } else if (eventType === 'step_update') {
        handlers.onStepUpdate?.(normalizeStepUpdateEvent(payload));
      } else if (eventType === 'tool_call') {
        handlers.onToolCall?.(payload as any);
      } else if (eventType === 'tool_result') {
        handlers.onToolResult?.(payload as any);
      } else if (eventType === 'thinking_delta') {
        handlers.onThinkingDelta?.(payload as SSEThinkingEvent);
      } else if (eventType === 'delta' || eventType === 'message') {
        const contentChunk = toContentChunk(payload);
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
      }
    };

    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      buffer += decoder.decode(value, { stream: true }).replace(/\r/g, '');
      const segments = buffer.split('\n\n');
      buffer = segments.pop() || '';
      segments.forEach(dispatchEvent);
    }

    if (buffer.trim()) {
      dispatchEvent(buffer);
    }
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
