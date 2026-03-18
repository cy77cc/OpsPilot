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
  | 'tool_call'
  | 'tool_approval'
  | 'tool_result'
  | 'hint'
  | 'error';

export type AssistantReplyActivityStatus = 'pending' | 'active' | 'done' | 'error';

export interface AssistantReplyActivity {
  id: string;
  kind: AssistantReplyActivityKind;
  label: string;
  detail?: string;
  status?: AssistantReplyActivityStatus;
  stepIndex?: number;
  createdAt?: string;
}

export interface AssistantReplyPlanStep {
  id: string;
  title: string;
  status: 'pending' | 'active' | 'done';
  content?: string;
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
  | 'interrupted';

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
}

export interface XChatMessage {
  role: 'user' | 'assistant';
  content: string;
  runtime?: AssistantReplyRuntime;
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
