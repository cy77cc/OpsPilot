/**
 * AI 助手组件类型定义
 */

// 消息角色
export type MessageRole = 'user' | 'assistant' | 'system';

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

// 审批请求
export interface AskRequest {
  id: string;
  kind?: 'approval' | 'confirmation' | 'review' | 'interrupt';
  title: string;
  description?: string;
  risk?: string;
  status?: 'pending' | 'approved' | 'rejected' | 'confirmed' | 'cancelled' | 'expired';
  details?: Record<string, unknown>;
}

// 确认请求
export interface ConfirmationRequest {
  id: string;
  title: string;
  description: string;
  risk: RiskLevel;
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

// 聊天消息
export interface ChatMessage {
  id: string;
  role: MessageRole;
  content: string;
  thinking?: string;
  tools?: ToolExecution[];
  confirmation?: ConfirmationRequest;
  // 新增: 下一步推荐
  recommendations?: EmbeddedRecommendation[];
  createdAt: string;
  updatedAt?: string;
}

// 会话
export interface Conversation {
  id: string;
  title: string;
  scene: string;
  messages: ChatMessage[];
  createdAt: string;
  updatedAt: string;
}

// 场景信息
export interface SceneInfo {
  key: string;
  label: string;
  description?: string;
  tools?: string[];
}

// AI 聊天上下文
export interface AIChatContextValue {
  // 状态
  messages: ChatMessage[];
  isLoading: boolean;
  currentConversation: Conversation | null;
  conversations: Conversation[];

  // 操作
  sendMessage: (content: string) => Promise<void>;
  createConversation: () => void;
  switchConversation: (id: string) => void;
  deleteConversation: (id: string) => Promise<void>;
  clearMessages: () => void;

  // 确认操作
  confirmAction: (id: string, approved: boolean) => Promise<void>;
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
  | 'thinking_delta'
  | 'tool_call'
  | 'tool_result'
  | 'approval_required'
  | 'confirmation_required'
  | 'done'
  | 'error'
  | 'heartbeat';

// SSE 事件载荷
export interface SSEEventPayload {
  type: SSEEventType;
  data: Record<string, unknown>;
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
