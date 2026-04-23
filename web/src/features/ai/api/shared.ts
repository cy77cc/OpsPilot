import { ApiRequestError } from '../../../api/api';
import apiService from '../../../api/api';
import type { ApiResponse } from '../../../api/api';

export { ApiRequestError, apiService };
export type { ApiResponse };

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
  trace_id?: string;
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
  type: 'agent_handoff' | 'plan' | 'replan' | 'executor' | 'error' | 'delegation.node' | string;
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

export interface A2UIDelegationNodeEvent {
  delegation_id: string;
  agent_name: string;
  status: string;
  title: string;
  summary: string;
  intent?: string;
  risk_level?: string;
}

export interface A2UIOpsPlanUpdatedEvent {
  run_id?: string;
  session_id?: string;
  runtime?: Record<string, unknown>;
  snapshot?: Record<string, unknown>;
  todos?: Array<Record<string, unknown>>;
}

export interface A2UIDoneEvent {
  run_id: string;
  status: 'completed';
  iterations?: number;
  summary?: string;
}

export interface A2UIErrorEvent {
  message: string;
  code?: string;
  error_code?: string;
  recoverable?: boolean;
  retryable?: boolean;
  run_id?: string;
}

export interface A2UIUnknownStreamEvent {
  eventType: string;
  payload: unknown;
  eventId?: string;
  runId?: string | number;
  userId?: string | number;
  tenantId?: string;
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
  onDelegationNode?: (payload: A2UIDelegationNodeEvent) => void;
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

function normalizeTimestampCompat<T extends TimestampCompatFields>(item: T | null | undefined): T | null | undefined {
  if (!item) {return item;}
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

export function normalizeSession(session: AISession): AISession {
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
  if (!report) {return report ?? undefined;}
  const reportId = report.report_id ?? report.id;
  if (reportId === undefined) {return { ...report };}
  return { ...report, id: reportId, report_id: reportId };
}

export function normalizeRun(run: AIRun): AIRun {
  return { ...run, report: normalizeRunReport(run.report) };
}

export function normalizeSessionResponse(response: ApiResponse<AISession>): ApiResponse<AISession> {
  return { ...response, data: normalizeSession(response.data) };
}

export function normalizeSessionListResponse(response: ApiResponse<AISession[]>): ApiResponse<AISession[]> {
  return {
    ...response,
    data: Array.isArray(response.data) ? response.data.map((session) => normalizeSession(session)) : response.data,
  };
}

export function normalizeRunResponse(response: ApiResponse<AIRun>): ApiResponse<AIRun> {
  return { ...response, data: normalizeRun(response.data) };
}

function normalizeErrorEvent(payload: unknown): A2UIErrorEvent {
  const errorPayload = { ...((typeof payload === 'object' && payload ? payload : {}) as A2UIErrorEvent) };
  if (!errorPayload.code && errorPayload.error_code) {errorPayload.code = errorPayload.error_code;}
  if (errorPayload.code === 'AI_STREAM_CURSOR_EXPIRED' || errorPayload.error_code === 'AI_STREAM_CURSOR_EXPIRED') {
    errorPayload.recoverable = true;
  }
  return errorPayload;
}

function readStreamEventTag(payload: unknown, keys: string[]): string | number | undefined {
  if (!payload || typeof payload !== 'object') {return undefined;}
  const record = payload as Record<string, unknown>;
  for (const key of keys) {
    const value = record[key];
    if (typeof value === 'string' || typeof value === 'number') {return value;}
  }
  return undefined;
}

function buildUnknownStreamEvent(eventType: string, payload: unknown, eventId?: string): A2UIUnknownStreamEvent {
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
      ? { tenantId: String(readStreamEventTag(payload, ['tenant_id', 'tenantId'])) }
      : {}),
  };
}

export function dispatchAIStreamEvent(segment: string, handlers: A2UIStreamHandlers): void {
  if (!segment.trim()) {return;}
  const lines = segment.split('\n');
  let eventType = '';
  let data = '';
  let eventId = '';
  lines.forEach((line) => {
    if (line.startsWith('event:')) {eventType = line.slice(6).trim();}
    else if (line.startsWith('data:')) {data += line.slice(5).trim();}
    else if (line.startsWith('id:')) {eventId = line.slice(3).trim();}
  });
  if (eventId) {handlers.onEventId?.(eventId);}
  const payload = data ? JSON.parse(data) : {};
  if (eventType === 'meta') {handlers.onMeta?.(payload as A2UIMetaEvent);}
  else if (eventType === 'agent_handoff') {handlers.onAgentHandoff?.(payload as A2UIAgentHandoffEvent);}
  else if (eventType === 'plan') {handlers.onPlan?.(payload as A2UIPlanEvent);}
  else if (eventType === 'replan') {handlers.onReplan?.(payload as A2UIReplanEvent);}
  else if (eventType === 'delta') {handlers.onDelta?.(payload as A2UIDeltaEvent);}
  else if (eventType === 'tool_call') {handlers.onToolCall?.(payload as A2UIToolCallEvent);}
  else if (eventType === 'tool_approval') {handlers.onToolApproval?.(payload as A2UIToolApprovalEvent);}
  else if (eventType === 'tool_result') {handlers.onToolResult?.(payload as A2UIToolResultEvent);}
  else if (eventType === 'delegation_node') {handlers.onDelegationNode?.(payload as A2UIDelegationNodeEvent);}
  else if (eventType === 'ops.plan.updated' || eventType === 'ops_plan_updated') {handlers.onOpsPlanUpdated?.(payload as A2UIOpsPlanUpdatedEvent);}
  else if (eventType === 'ai.run.resuming') {handlers.onRunResuming?.(payload as A2UIRunResumingEvent);}
  else if (eventType === 'ai.run.resumed') {handlers.onRunResumed?.(payload as A2UIRunResumedEvent);}
  else if (eventType === 'ai.run.resume_failed') {handlers.onRunResumeFailed?.(payload as A2UIRunResumeFailedEvent);}
  else if (eventType === 'ai.run.completed') {handlers.onRunCompleted?.(payload as A2UIRunCompletedEvent);}
  else if (eventType === 'run_state') {handlers.onRunState?.(payload as A2UIRunStateEvent);}
  else if (eventType === 'ai.approval.expired') {handlers.onApprovalExpired?.(payload as A2UIApprovalExpiredEvent);}
  else if (eventType === 'done') {handlers.onDone?.(payload as A2UIDoneEvent);}
  else if (eventType === 'error') {handlers.onError?.(normalizeErrorEvent(payload));}
  else {handlers.onUnknownEvent?.(buildUnknownStreamEvent(eventType, payload, eventId || undefined));}
}

export async function consumeAIStream(response: Response, handlers: A2UIStreamHandlers): Promise<void> {
  if (!response.ok || !response.body) {throw new Error(`请求失败: ${response.status}`);}
  const reader = response.body.getReader();
  const decoder = new TextDecoder('utf-8');
  let buffer = '';
  while (true) {
    const { done, value } = await reader.read();
    if (done) {break;}
    buffer += decoder.decode(value, { stream: true }).replace(/\r/g, '');
    const segments = buffer.split('\n\n');
    buffer = segments.pop() || '';
    segments.forEach((segment) => dispatchAIStreamEvent(segment, handlers));
  }
  if (buffer.trim()) {dispatchAIStreamEvent(buffer, handlers);}
}

export function generateIdempotencyKey(): string {
  if (typeof globalThis !== 'undefined' && globalThis.crypto && typeof globalThis.crypto.randomUUID === 'function') {
    return globalThis.crypto.randomUUID();
  }
  return `approval-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export function isApprovalConflictError(error: unknown): boolean {
  if (error instanceof ApiRequestError) {
    if (error.statusCode !== 400 && error.statusCode !== 409) {return false;}
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

export function isApprovalNotFoundError(error: unknown): boolean {
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
