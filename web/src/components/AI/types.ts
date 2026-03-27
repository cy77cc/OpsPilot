export interface SceneContext {
  route?: string;
  resourceType?: string;
  resourceId?: string;
  resourceName?: string;
  [key: string]: unknown;
}

export interface ChatRequest {
  message: string;
  sessionId?: string;
  clientRequestId?: string;
  lastEventId?: string;
  scene?: string;
  context?: SceneContext;
}

export type AssistantReplyPhase =
  | 'preparing'
  | 'identifying'
  | 'planning'
  | 'executing'
  | 'summarizing'
  | 'completed'
  | 'interrupted';

export type AssistantReplyActivityKind =
  | 'agent_handoff'
  | 'plan'
  | 'replan'
  | 'tool'
  | 'tool_approval'
  | 'hint'
  | 'error';

export type AssistantReplyActivityStatus = 'pending' | 'active' | 'done' | 'error';
export type AssistantReplyApprovalState =
  | 'waiting-approval'
  | 'submitting'
  | 'approved_resuming'
  | 'approved_retrying'
  | 'approved_failed_terminal'
  | 'approved_done'
  | 'expired'
  | 'approved'
  | 'rejected'
  | 'refresh-needed';

export type PendingRunStatus =
  | 'waiting_approval'
  | 'resuming'
  | 'running'
  | 'resume_failed_retryable';

export interface PendingRunMetadata {
  runId: string;
  sessionId?: string;
  clientRequestId?: string;
  latestEventId?: string;
  approvalId?: string;
  approvalCallId?: string;
  status: PendingRunStatus;
  resumable: boolean;
  messageId?: string;
  updatedAt?: string;
}

// 瘦身后的 executor block（减少内存占用，只保留懒加载必须字段）
export interface SlimExecutorBlock {
  id: string;
  items: Array<{
    type: string;
    content_id?: string;
    tool_call_id?: string;
    tool_name?: string;
    arguments?: Record<string, unknown>;
    result?: {
      status?: string;
      preview?: string;
      result_content_id?: string;
    };
  }>;
}

export interface AssistantReplyActivity {
  id: string;
  kind: AssistantReplyActivityKind;
  label: string;
  detail?: string;
  status?: AssistantReplyActivityStatus;
  approvalId?: string;
  approvalState?: AssistantReplyApprovalState;
  approvalMessage?: string;
  stepIndex?: number;
  createdAt?: string;
  arguments?: Record<string, unknown>;  // 工具调用参数
  rawContent?: string;                  // 完整结果内容
}

export interface AssistantReplyTodo {
  id: string;
  content: string;
  activeForm?: string;
  status: string;
  cluster?: string;
  namespace?: string;
  resourceType?: string;
  riskLevel?: string;
  requiresApproval?: boolean;
  estimatedDuration?: string;
  dependsOn?: string[];
}

export interface AssistantReplyPlanStep {
  id: string;
  title: string;
  status: 'pending' | 'active' | 'done';
  content?: string;
  segments?: AssistantReplySegment[];
  loaded?: boolean;  // 标记内容是否已加载
  sourceBlockIndex?: number;
  sourceStepIndex?: number;
  unresolved?: boolean;
}

export interface AssistantReplySegment {
  type: 'text' | 'tool_ref';
  text?: string;
  callId?: string;
}

export interface AssistantReplyPlan {
  steps: AssistantReplyPlanStep[];
  activeStepIndex?: number;
}

export type AssistantSummaryTone = 'default' | 'success' | 'warning' | 'danger';

export interface AssistantReplySummary {
  title?: string;
  items?: Array<{
    label: string;
    value: string;
    tone?: AssistantSummaryTone;
  }>;
}

export type AssistantReplyStatusKind =
  | 'streaming'
  | 'completed'
  | 'soft-timeout'
  | 'error'
  | 'waiting_approval'
  | 'resuming'
  | 'resume_failed_retryable'
  | 'failed'
  | 'interrupted'
  | 'approved_resuming'
  | 'approved_retrying'
  | 'approved_failed_terminal'
  | 'approved_done'
  | 'expired';

export interface AssistantReplyRuntimeStatus {
  kind: AssistantReplyStatusKind;
  label: string;
}

export interface AssistantReplyRuntime {
  phase?: AssistantReplyPhase;
  phaseLabel?: string;
  activities: AssistantReplyActivity[];
  plan?: AssistantReplyPlan;
  summary?: AssistantReplySummary;
  status?: AssistantReplyRuntimeStatus;
  pendingRun?: PendingRunMetadata;
  todos?: AssistantReplyTodo[];
  _executorBlocks?: SlimExecutorBlock[];  // 存储瘦身后的 executor blocks 引用
}

export interface XChatMessage {
  id?: string; // 消息 ID，用于懒加载 runtime
  role: 'user' | 'assistant';
  content: string;
  runtime?: AssistantReplyRuntime;
  hasRuntime?: boolean; // 标记是否有 runtime 数据
}

export interface ConversationSummary {
  key: string;
  label: string;
  scene: string;
  updatedAt?: string;
}

export interface PlatformStreamChunk {
  content: string;
  mode?: 'replace' | 'append';
  runtime?: AssistantReplyRuntime;
}
