/**
 * AI 助手组件类型定义
 */

// 消息角色
export type MessageRole = 'user' | 'assistant' | 'system';
export type ChatTurnStatus = 'streaming' | 'waiting_user' | 'completed' | 'error';
export type ChatTurnPhase = 'rewrite' | 'plan' | 'planning' | 'execute' | 'replanning' | 'summary' | 'done' | 'user_message' | 'streaming';

// 工具执行状态
export type ToolStatus = 'running' | 'success' | 'error';

// 风险等级
export type RiskLevel = 'low' | 'medium' | 'high';

// 工具追踪类型
export type ToolTraceType = 'tool_call' | 'tool_result';

// 消息内容片段
export interface ContentPart {
  type: 'text' | 'tool_card' | 'confirmation';
  text?: string;
  tool?: ToolExecution;
  confirmation?: ConfirmationRequest;
}

// 工具执行信息
export interface ToolExecution {
  id: string;
  name: string;
  status: ToolStatus;
  duration?: number;
  summary?: string;
  target?: string;
  error?: string;
  // 新增: 工具调用参数
  params?: Record<string, unknown>;
  // 新增: 工具执行结果
  result?: {
    ok: boolean;
    data?: unknown;
    error?: string;
    latency_ms?: number;
  };
}

// 工具追踪
export interface ToolTrace {
  id: string;
  type: ToolTraceType;
  tool: string;
  callId?: string;
  timestamp?: string;
  payload?: Record<string, unknown>;
  retry?: boolean;
}

// 确认请求（审批面板使用）
export interface ConfirmationRequest {
  id: string;
  title: string;
  description: string;
  risk: RiskLevel;
  status?: 'waiting_user' | 'submitting' | 'failed';
  errorMessage?: string;
  details?: Record<string, unknown>;
  // 工具信息
  toolName?: string;
  toolDisplayName?: string;
  // 恢复身份字段
  planId?: string;
  stepId?: string;
  checkpointId?: string;
  target?: string;
  // 参数编辑支持
  argumentsJson?: string;
  editable?: boolean;
  // 回调
  onConfirm: (editedArgs?: string) => void;
  onCancel: (reason?: string) => void;
}

// 推荐建议
export interface EmbeddedRecommendation {
  id: string;
  type: string;
  title: string;
  content: string;
  followup_prompt?: string;
  reasoning?: string;
  relevance: number;
}

export interface PlanStep {
  id?: string;
  content?: string;
  title?: string;
  tool_hint?: string;
  status?: string;
  summary?: string;
}

export type RuntimeThoughtChainNodeKind = 'plan' | 'execute' | 'tool' | 'replan' | 'approval';
export type RuntimeThoughtChainNodeStatus = 'pending' | 'active' | 'done' | 'error' | 'waiting';
export type FinalAnswerRevealState = 'hidden' | 'primed' | 'revealing' | 'complete';

export interface RuntimeThoughtChainNode {
  nodeId: string;
  kind: RuntimeThoughtChainNodeKind;
  title: string;
  status: RuntimeThoughtChainNodeStatus;
  headline?: string;
  body?: string;
  structured?: Record<string, unknown>;
  raw?: unknown;
  summary?: string;
  details?: unknown[];
  approval?: Omit<ConfirmationRequest, 'onConfirm' | 'onCancel'> & {
    requestId?: string;
    details?: Record<string, unknown>;
  };
}

export interface FinalAnswerState {
  visible: boolean;
  streaming: boolean;
  content: string;
  revealState: FinalAnswerRevealState;
}

export interface ThoughtChainRuntimeState {
  turnId?: string;
  nodes: RuntimeThoughtChainNode[];
  activeNodeId?: string;
  isCollapsed: boolean;
  collapsePhase: 'expanded' | 'collapsing' | 'collapsed';
  finalAnswer: FinalAnswerState;
}

// === Replay compatibility structures ===
// These remain only for persisted history projection and transitional UI fallback.
// They are not the canonical live-runtime contract.

export type TurnBlockType =
  | 'status'
  | 'text'
  | 'plan'
  | 'tool'
  | 'approval'
  | 'evidence'
  | 'thinking'
  | 'error'
  | 'recommendations';

export interface TurnBlock {
  id: string;
  type: TurnBlockType;
  status?: string;
  title?: string;
  position: number;
  streaming?: boolean;
  content?: string;
  data?: Record<string, unknown>;
}

export interface ChatTurn {
  id: string;
  role: Extract<MessageRole, 'assistant' | 'user'>;
  status: ChatTurnStatus;
  phase?: ChatTurnPhase | string;
  traceId?: string;
  parentTurnId?: string;
  blocks: TurnBlock[];
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}

// 聊天消息
export interface ChatMessage {
  id: string;
  role: MessageRole;
  content: string;
  // Compatibility replay snapshot for block-based history projection.
  turn?: ChatTurn;
  // Canonical live/runtime-first thoughtChain state.
  runtime?: ThoughtChainRuntimeState;

  // Legacy compatibility for the pre-turn/block rendering path.
  thinking?: string;
  rawEvidence?: string[];
  tools?: ToolExecution[];
  confirmation?: ConfirmationRequest;
  recommendations?: EmbeddedRecommendation[];
  thoughtChain?: ThoughtStageItem[];
  traceId?: string;
  restored?: boolean;
  createdAt: string;
  updatedAt?: string;
}

// 场景信息
export interface SceneInfo {
  key: string;
  label: string;
  description?: string;
  tools?: string[];
}

// 抽屉宽度设置
export interface DrawerWidthConfig {
  default: number;
  min: number;
  max: number;
}

// SSE 事件类型
export type SSEEventType =
  | 'meta'
  | 'delta'
  | 'message'
  | 'thinking_delta'
  | 'tool_call'
  | 'tool_result'
  | 'clarify_required'
  | 'replan_started'
  | 'done'
  | 'error'
  | 'heartbeat';

// SSE 事件载荷
export interface SSEEventPayload {
  type: SSEEventType;
  data: Record<string, unknown>;
}

// === Legacy compatibility types ===
// Legacy thought-stage structures are preserved only for restoration and narrow test bridges.

export type ThoughtStageKey = 'rewrite' | 'plan' | 'execute' | 'user_action' | 'summary';

export type ThoughtStageStatus = 'loading' | 'success' | 'error' | 'abort';

export interface ThoughtStageDetailItem {
  id: string;
  label: string;
  status: ThoughtStageStatus;
  content?: string;
  kind?: 'tool' | 'approval' | 'note';
  tool?: string;
  params?: Record<string, unknown>;
  result?: {
    ok?: boolean;
    data?: unknown;
    error?: string;
    latency_ms?: number;
  };
  risk?: RiskLevel;
  session_id?: string;
  plan_id?: string;
  step_id?: string;
  checkpoint_id?: string;
  metadata?: Record<string, unknown>;
}

export interface ThoughtStageItem {
  key: ThoughtStageKey;
  title: string;
  description?: string;
  content?: string;
  footer?: string;
  details?: ThoughtStageDetailItem[];
  status: ThoughtStageStatus;
  collapsible?: boolean;
  blink?: boolean;
}

// 错误类型
export type ErrorType = 'network' | 'timeout' | 'auth' | 'tool' | 'unknown';

// 错误信息
export interface ErrorInfo {
  type: ErrorType;
  message: string;
  code?: string;
  recoverable?: boolean;
  retry?: () => void;
}
