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
  onConfirm: () => void;
  onCancel: () => void;
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

// === Turn/block 主模型 ===

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
  turn?: ChatTurn;

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
  | 'turn_started'
  | 'block_open'
  | 'block_delta'
  | 'block_replace'
  | 'block_close'
  | 'turn_state'
  | 'turn_done'
  | 'rewrite_result'
  | 'planner_state'
  | 'plan_created'
  | 'phase_started'
  | 'phase_complete'
  | 'plan_generated'
  | 'stage_delta'
  | 'step_started'
  | 'step_complete'
  | 'step_update'
  | 'delta'
  | 'message'
  | 'thinking_delta'
  | 'tool_call'
  | 'tool_result'
  | 'approval_required'
  | 'clarify_required'
  | 'replan_triggered'
  | 'replan_started'
  | 'summary'
  | 'done'
  | 'error'
  | 'heartbeat';

// SSE 事件载荷
export interface SSEEventPayload {
  type: SSEEventType;
  data: Record<string, unknown>;
}

// === Legacy compatibility types ===

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
