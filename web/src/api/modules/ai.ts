import apiService from '../api';
import type { ApiResponse } from '../api';

// AI对话消息数据结构
export interface AIMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  thinking?: string;
  traces?: ToolTrace[];
  recommendations?: EmbeddedRecommendation[];
  timestamp: string;
}

// AI对话会话数据结构
export interface AISession {
  id: string;
  title: string;
  messages: AIMessage[];
  createdAt: string;
  updatedAt: string;
}

// AI推荐数据结构
export interface AIRecommendation {
  id: string;
  type: string;
  title: string;
  content: string;
  followup_prompt?: string;
  reasoning?: string;
  relevance: number;
  action?: string;
  params?: Record<string, any>;
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
  type: 'tool_call' | 'tool_result' | 'approval_required' | 'confirmation_required' | 'tool_missing';
  payload: Record<string, any>;
  timestamp: string;
}

export interface AIInterruptApprovalResponse {
  checkpoint_id?: string;
  session_id?: string;
  target: string;
  approved: boolean;
  reason?: string;
}

export interface AIInterruptApprovalResult {
  resumed: boolean;
  interrupted?: boolean;
  content?: string;
  sessionId?: string;
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

// AI分析结果数据结构
export interface AIAnalysis {
  id: string;
  type: string;
  title: string;
  summary: string;
  details: any;
  createdAt: string;
}

// AI对话请求参数
export interface AIChatParams {
  sessionId?: string;
  message: string;
  context?: any;
}

// AI分析请求参数
export interface AIAnalysisParams {
  type: string;
  data: any;
  context?: any;
}

// AI推荐请求参数
export interface AIRecommendationParams {
  type: string;
  context: any;
  limit?: number;
}

export interface AIActionPreviewParams {
  action: string;
  params: Record<string, any>;
}

export interface AIActionExecuteParams {
  action?: string;
  approval_token: string;
}

export interface AICommandPreviewParams {
  command: string;
  scene?: string;
  params?: Record<string, any>;
}

export interface AICommandExecuteParams {
  command_id?: string;
  command?: string;
  scene?: string;
  params?: Record<string, any>;
  confirm: boolean;
  approval_token?: string;
}

export interface AICommandResult {
  status: 'previewed' | 'blocked' | 'succeeded' | 'failed';
  summary: string;
  artifacts: Record<string, any>;
  trace_id: string;
  next_actions: string[];
  plan?: Record<string, any>;
  missing?: string[];
  prompts?: Record<string, string>;
  risk: 'readonly' | 'low' | 'high';
}

export interface AICommandHistoryItem {
  id: string;
  command: string;
  intent: string;
  status: string;
  risk: string;
  trace_id: string;
  plan_hash: string;
  execution_summary: string;
  created_at: string;
  plan?: Record<string, any>;
  result?: Record<string, any>;
}

export interface AICommandSuggestion {
  command: string;
  hint?: string;
  category?: string;
  source?: string;
}

export interface AISceneToolsPayload {
  scene: string;
  description: string;
  keywords: string[];
  context_hints: string[];
  tools: AICapability[];
}

export interface AICommandAliasPayload {
  scene: string;
  aliases: Record<string, string>;
  builtin?: Record<string, string>;
}

export interface AICommandTemplatePayload {
  scene: string;
  templates: Record<string, Record<string, any>>;
}

interface SSEMetaEvent {
  sessionId: string;
  createdAt: string;
  turn_id?: string;
}

interface SSEDeltaEvent {
  contentChunk: string;
  turn_id?: string;
}

interface SSEDoneEvent {
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

export interface SSEExpertProgressEvent {
  expert: string;
  status: 'running' | 'done';
  task?: string;
  duration_ms?: number;
  turn_id?: string;
}

export interface AIChatStreamHandlers {
  onMeta?: (payload: SSEMetaEvent) => void;
  onDelta?: (payload: SSEDeltaEvent) => void;
  onDone?: (payload: SSEDoneEvent) => void;
  onError?: (payload: SSEErrorEvent) => void;
  onThinkingDelta?: (payload: SSEThinkingEvent) => void;
  onToolCall?: (payload: { turn_id?: string; call_id?: string; tool?: string; payload?: Record<string, any>; ts?: string; tool_calls?: Array<{ function?: { name?: string; arguments?: string } }> }) => void;
  onToolResult?: (payload: { turn_id?: string; call_id?: string; tool?: string; payload?: Record<string, any>; result?: { ok: boolean; data?: any; error?: string; error_code?: string; source?: string; latency_ms?: number }; ts?: string }) => void;
  onApprovalRequired?: (payload: ApprovalTicket & { turn_id?: string; approval_required?: boolean; previewDiff?: string }) => void;
  onConfirmationRequired?: (payload: { turn_id?: string; tool?: string; confirmation_token?: string; expiresAt?: string; preview?: Record<string, any>; message?: string }) => void;
  onToolIntentUnresolved?: (payload: { turn_id?: string; tool?: string; message?: string }) => void;
  onHeartbeat?: (payload: { turn_id?: string; status?: string }) => void;
  onExpertProgress?: (payload: SSEExpertProgressEvent) => void;
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
  tool: string;
  params: Record<string, any>;
  risk: RiskLevel;
  mode: 'readonly' | 'mutating';
  status: 'pending' | 'approved' | 'rejected' | 'expired';
  createdAt: string;
  expiresAt: string;
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
  async chatStream(params: AIChatParams, handlers: AIChatStreamHandlers): Promise<void> {
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
      } else if (eventType === 'delta') {
        handlers.onDelta?.(payload as SSEDeltaEvent);
      } else if (eventType === 'done') {
        handlers.onDone?.(payload as SSEDoneEvent);
        toolPending = false;
        clearToolTimer();
      } else if (eventType === 'error') {
        handlers.onError?.(payload as SSEErrorEvent);
        const err = payload as SSEErrorEvent;
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
      } else if (eventType === 'approval_required') {
        handlers.onApprovalRequired?.(payload as ApprovalTicket & { approval_required?: boolean; previewDiff?: string });
        toolPending = false;
        clearToolTimer();
      } else if (eventType === 'confirmation_required') {
        handlers.onConfirmationRequired?.(payload as { turn_id?: string; tool?: string; confirmation_token?: string; expiresAt?: string; preview?: Record<string, any>; message?: string });
        toolPending = false;
        clearToolTimer();
      } else if (eventType === 'tool_intent_unresolved') {
        handlers.onToolIntentUnresolved?.(payload as { turn_id?: string; tool?: string; message?: string });
      } else if (eventType === 'heartbeat') {
        handlers.onHeartbeat?.(payload as { turn_id?: string; status?: string });
        touchActivity();
      } else if (eventType === 'expert_progress') {
        handlers.onExpertProgress?.(payload as SSEExpertProgressEvent);
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
  async getSessionDetail(id: string): Promise<ApiResponse<AISession>> {
    return apiService.get(`/ai/sessions/${id}`);
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

  // AI分析
  async analyze(params: AIAnalysisParams): Promise<ApiResponse<AIAnalysis>> {
    return apiService.post('/ai/analyze', params);
  },

  // AI推荐
  async getRecommendations(params: AIRecommendationParams): Promise<ApiResponse<AIRecommendation[]>> {
    return apiService.post('/ai/recommendations', params);
  },

  // AI辅助功能
  async getAssistance(params: {
    type: string;
    context: any;
  }): Promise<ApiResponse<{
    assistance: string;
    actions?: Array<{
      label: string;
      value: string;
    }>;
  }>> {
    return apiService.post('/ai/assist', params);
  },

  async previewAction(params: AIActionPreviewParams): Promise<ApiResponse<{
    approval_token: string;
    intent: string;
    risk: string;
    params: Record<string, any>;
    previewDiff: string;
  }>> {
    return apiService.post('/ai/actions/preview', params);
  },

  async executeAction(params: AIActionExecuteParams): Promise<ApiResponse<Record<string, any>>> {
    return apiService.post('/ai/actions/execute', params);
  },

  async k8sAnalyze(params: { cluster_id: number; namespace?: string; question?: string; context?: Record<string, any> }): Promise<ApiResponse<{
    insights: string[];
    risks: string[];
    recommended_actions: Array<{ action: string; params?: Record<string, any>; reason?: string }>;
  }>> {
    return apiService.post('/ai/k8s/analyze', params);
  },

  async previewK8sAction(params: AIActionPreviewParams): Promise<ApiResponse<any>> {
    return apiService.post('/ai/k8s/actions/preview', params);
  },

  async executeK8sAction(params: AIActionExecuteParams): Promise<ApiResponse<Record<string, any>>> {
    return apiService.post('/ai/k8s/actions/execute', params);
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

  async executeTool(params: { tool: string; params?: Record<string, any>; approval_token?: string }): Promise<ApiResponse<AIToolExecution>> {
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

  async respondApproval(params: AIInterruptApprovalResponse): Promise<ApiResponse<AIInterruptApprovalResult>> {
    return apiService.post('/ai/approval/respond', params);
  },

  async confirmConfirmation(id: string, approve: boolean): Promise<ApiResponse<ConfirmationTicket>> {
    return apiService.post(`/ai/confirmations/${id}/confirm`, { approve });
  },

  async getSceneTools(scene: string): Promise<ApiResponse<AISceneToolsPayload>> {
    return apiService.get(`/ai/scene/${scene}/tools`);
  },

  async getCommandSuggestions(params?: { scene?: string; q?: string }): Promise<ApiResponse<AICommandSuggestion[]>> {
    return apiService.get('/ai/commands/suggestions', { params });
  },

  async getCommandAliases(scene?: string): Promise<ApiResponse<AICommandAliasPayload>> {
    return apiService.get('/ai/commands/aliases', { params: { scene } });
  },

  async saveCommandAlias(params: { scene?: string; alias: string; command: string }): Promise<ApiResponse<AICommandAliasPayload>> {
    return apiService.post('/ai/commands/aliases', params);
  },

  async deleteCommandAlias(alias: string, scene?: string): Promise<ApiResponse<AICommandAliasPayload>> {
    return apiService.delete(`/ai/commands/aliases/${alias}`, { params: { scene } });
  },

  async getCommandTemplates(scene?: string): Promise<ApiResponse<AICommandTemplatePayload>> {
    return apiService.get('/ai/commands/templates', { params: { scene } });
  },

  async saveCommandTemplate(params: { scene?: string; name: string; params: Record<string, any> }): Promise<ApiResponse<AICommandTemplatePayload>> {
    return apiService.post('/ai/commands/templates', params);
  },

  async previewCommand(params: AICommandPreviewParams): Promise<ApiResponse<AICommandResult>> {
    return apiService.post('/ai/commands/preview', params);
  },

  async executeCommand(params: AICommandExecuteParams): Promise<ApiResponse<AICommandResult>> {
    return apiService.post('/ai/commands/execute', params);
  },

  async getCommandHistory(limit = 20): Promise<ApiResponse<{ list: AICommandHistoryItem[]; total: number }>> {
    return apiService.get('/ai/commands/history', { params: { limit } });
  },

  async getCommandHistoryDetail(id: string): Promise<ApiResponse<{ record: AICommandHistoryItem; audit_events: any[] }>> {
    return apiService.get(`/ai/commands/history/${id}`);
  },
};
