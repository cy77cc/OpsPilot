/**
 * 工具链运行时状态管理
 *
 * 处理简化后的事件流：tool_call -> tool_approval(可选) -> tool_result
 */
import type { SSEToolCallEvent, SSEToolApprovalEvent, SSEToolResultEvent } from '../../api/modules/ai';
import type {
  ChatTurn,
  ConfirmationRequest,
  RuntimeThoughtChainNodeStatus,
  ToolChainNode,
  ToolChainState,
  ToolChainNodeKind,
  ToolChainNodeStatus,
  ThoughtChainRuntimeState,
} from './types';

type ToolChainEvent =
  | { type: 'tool_call'; data: SSEToolCallEvent }
  | { type: 'tool_approval'; data: SSEToolApprovalEvent }
  | { type: 'tool_result'; data: SSEToolResultEvent };

type RuntimeState = ToolChainState | ThoughtChainRuntimeState;

/**
 * 创建初始工具链状态
 */
export function createToolChainState(): ToolChainState {
  return {
    nodes: [],
  };
}

/**
 * 将旧版 ThoughtChainRuntimeState 转换为 ToolChainState
 */
function toToolChainState(state: RuntimeState | undefined): ToolChainState {
  if (!state) {
    return createToolChainState();
  }
  // 如果已经是 ToolChainState 格式（没有 isCollapsed 字段）
  if (!('isCollapsed' in state)) {
    return state as ToolChainState;
  }
  // 转换旧格式
  const legacyState = state as ThoughtChainRuntimeState;
  return {
    nodes: legacyState.nodes.map((node) => ({
      id: node.nodeId,
      kind: (node.kind === 'tool' || node.kind === 'approval') ? node.kind : 'tool',
      toolName: node.title || 'tool',
      toolDisplayName: node.title,
      status: convertStatus(node.status),
      arguments: node.structured,
      result: node.raw ? { ok: true, data: node.raw } : undefined,
      approval: node.approval,
    })),
    activeNodeId: legacyState.activeNodeId,
  };
}

function convertStatus(status: string): ToolChainNodeStatus {
  switch (status) {
    case 'pending':
      return 'pending';
    case 'active':
      return 'running';
    case 'waiting':
      return 'waiting_approval';
    case 'done':
      return 'success';
    case 'error':
      return 'error';
    default:
      return 'pending';
  }
}

/**
 * 处理工具链事件，返回新状态
 */
export function reduceToolChainEvent(
  current: RuntimeState | undefined,
  event: ToolChainEvent,
): ToolChainState {
  const state = toToolChainState(current);

  switch (event.type) {
    case 'tool_call':
      return handleToolCall(state, event.data);
    case 'tool_approval':
      return handleToolApproval(state, event.data);
    case 'tool_result':
      return handleToolResult(state, event.data);
    default:
      return state;
  }
}

function handleToolCall(state: ToolChainState, data: SSEToolCallEvent): ToolChainState {
  const callId = data.call_id || `call-${Date.now()}`;
  const existingIndex = state.nodes.findIndex((n) => n.id === callId);

  if (existingIndex >= 0) {
    // 更新现有节点
    const nodes = [...state.nodes];
    nodes[existingIndex] = {
      ...nodes[existingIndex],
      status: 'running',
    };
    return { ...state, nodes, activeNodeId: callId };
  }

  // 创建新节点
  const node: ToolChainNode = {
    id: callId,
    kind: 'tool',
    toolName: data.tool_name || 'unknown',
    toolDisplayName: data.tool_display_name,
    status: 'running',
    arguments: parseArguments(data.arguments),
  };

  return {
    ...state,
    nodes: [...state.nodes, node],
    activeNodeId: callId,
  };
}

function handleToolApproval(state: ToolChainState, data: SSEToolApprovalEvent): ToolChainState {
  const callId = data.call_id || `approval-${Date.now()}`;
  const existingIndex = state.nodes.findIndex((n) => n.id === callId);

  const approval: Omit<ConfirmationRequest, 'onConfirm' | 'onCancel'> = {
    id: data.approval_id || callId,
    title: data.tool_display_name || data.tool_name || '待确认操作',
    description: data.summary || '此操作需要确认',
    risk: (data.risk || 'medium') as ConfirmationRequest['risk'],
    status: 'waiting_user',
    toolName: data.tool_name,
    toolDisplayName: data.tool_display_name,
    planId: data.plan_id,
    stepId: data.step_id,
    checkpointId: data.checkpoint_id,
    argumentsJson: data.arguments_json,
    editable: true,
  };

  if (existingIndex >= 0) {
    // 更新现有节点
    const nodes = [...state.nodes];
    nodes[existingIndex] = {
      ...nodes[existingIndex],
      status: 'waiting_approval',
      approval,
    };
    return { ...state, nodes, activeNodeId: callId };
  }

  // 创建新的审批节点
  const node: ToolChainNode = {
    id: callId,
    kind: 'approval',
    toolName: data.tool_name || 'unknown',
    toolDisplayName: data.tool_display_name,
    status: 'waiting_approval',
    approval,
    arguments: parseArguments(data.arguments_json),
  };

  return {
    ...state,
    nodes: [...state.nodes, node],
    activeNodeId: callId,
  };
}

function handleToolResult(state: ToolChainState, data: SSEToolResultEvent): ToolChainState {
  const callId = data.call_id;
  if (!callId) {
    return state;
  }

  const existingIndex = state.nodes.findIndex((n) => n.id === callId);
  if (existingIndex < 0) {
    return state;
  }

  const nodes = [...state.nodes];
  const existingNode = nodes[existingIndex];
  const result = parseResult(data.result);

  nodes[existingIndex] = {
    ...existingNode,
    status: result.ok ? 'success' : 'error',
    result,
  };

  // 如果有活跃节点且匹配，清除活跃状态
  const activeNodeId = state.activeNodeId === callId ? undefined : state.activeNodeId;

  return { ...state, nodes, activeNodeId };
}

function parseArguments(args?: string): Record<string, unknown> | undefined {
  if (!args) return undefined;
  try {
    return JSON.parse(args);
  } catch {
    return undefined;
  }
}

function parseResult(result?: string): { ok: boolean; data?: unknown; error?: string } {
  if (!result) {
    return { ok: true };
  }
  try {
    const parsed = JSON.parse(result);
    if (typeof parsed === 'object' && parsed !== null) {
      if ('error' in parsed && parsed.error) {
        return { ok: false, error: String(parsed.error), data: parsed };
      }
      return { ok: true, data: parsed };
    }
    return { ok: true, data: parsed };
  } catch {
    // 如果不是 JSON，直接作为文本返回
    if (result.includes('error') || result.includes('failed')) {
      return { ok: false, error: result };
    }
    return { ok: true, data: result };
  }
}

// === 兼容性函数：从回放数据重建状态 ===

/**
 * 从回放轮次数据重建工具链状态
 */
export function toolChainStateFromReplayTurn(turn: ChatTurn | undefined): ToolChainState | undefined {
  if (!turn || turn.role !== 'assistant') {
    return undefined;
  }

  const state = createToolChainState();

  for (const block of turn.blocks) {
    if (block.type === 'tool') {
      const callId = block.id || `tool-${Date.now()}`;
      state.nodes.push({
        id: callId,
        kind: 'tool',
        toolName: block.title || 'tool',
        toolDisplayName: block.title,
        status: normalizeBlockStatus(block.status, turn.status),
        result: block.data?.result as ToolChainNode['result'],
        arguments: block.data?.params as Record<string, unknown>,
      });
    } else if (block.type === 'approval') {
      const callId = block.id || `approval-${Date.now()}`;
      const data = block.data || {};
      state.nodes.push({
        id: callId,
        kind: 'approval',
        toolName: (data.tool_name as string) || 'unknown',
        toolDisplayName: data.tool_display_name as string,
        status: normalizeBlockStatus(block.status, turn.status, true),
        approval: {
          id: callId,
          title: block.title || '待确认操作',
          description: (data.summary as string) || '此操作需要确认',
          risk: (data.risk as ConfirmationRequest['risk']) || 'medium',
          status: 'waiting_user',
          toolName: data.tool_name as string,
          toolDisplayName: data.tool_display_name as string,
          planId: data.plan_id as string,
          stepId: data.step_id as string,
          checkpointId: data.checkpoint_id as string,
          argumentsJson: data.arguments_json as string,
          editable: true,
        },
      });
    }
  }

  if (state.nodes.length === 0) {
    return undefined;
  }

  state.activeNodeId = state.nodes.find((n) => n.status === 'running' || n.status === 'waiting_approval')?.id;
  return state;
}

function normalizeBlockStatus(
  status?: string,
  turnStatus?: string,
  isApproval = false,
): ToolChainNodeStatus {
  if (isApproval || status === 'waiting_approval') {
    return 'waiting_approval';
  }
  if (status === 'error' || turnStatus === 'error') {
    return 'error';
  }
  if (status === 'running' || status === 'streaming' || turnStatus === 'streaming') {
    return 'running';
  }
  if (status === 'success' || status === 'completed' || turnStatus === 'completed') {
    return 'success';
  }
  return 'pending';
}

function toRuntimeNodeStatus(status: ToolChainNodeStatus): RuntimeThoughtChainNodeStatus {
  if (status === 'waiting_approval') {
    return 'waiting';
  }
  if (status === 'success') {
    return 'done';
  }
  if (status === 'running') {
    return 'active';
  }
  return status;
}

/**
 * 将 ToolChainState 转换为 ThoughtChainRuntimeState
 * 用于向后兼容旧版渲染组件
 */
export function toThoughtChainRuntimeState(state: ToolChainState | undefined): ThoughtChainRuntimeState | undefined {
  if (!state || state.nodes.length === 0) {
    return undefined;
  }

  return {
    nodes: state.nodes.map((node) => ({
      nodeId: node.id,
      kind: node.kind,
      title: node.toolDisplayName || node.toolName,
      status: toRuntimeNodeStatus(node.status),
      summary: node.result?.error || node.result?.data ? String(node.result.data || node.result.error) : undefined,
      approval: node.approval,
    })),
    activeNodeId: state.activeNodeId,
    isCollapsed: false,
    collapsePhase: 'expanded' as const,
    finalAnswer: {
      visible: false,
      streaming: false,
      content: '',
      revealState: 'hidden' as const,
    },
  };
}

// === 旧版兼容：保留原有函数签名 ===

export function createThoughtChainRuntimeState() {
  return {
    nodes: [],
    isCollapsed: false,
    collapsePhase: 'expanded' as const,
    finalAnswer: {
      visible: false,
      streaming: false,
      content: '',
      revealState: 'hidden' as const,
    },
  };
}

export function runtimeStateFromReplayTurn(turn: ChatTurn | undefined) {
  const toolChain = toolChainStateFromReplayTurn(turn);
  if (!toolChain) {
    return undefined;
  }

  // 转换为旧格式
  return {
    turnId: turn?.id,
    nodes: toolChain.nodes.map((node) => ({
      nodeId: node.id,
      kind: node.kind,
      title: node.toolDisplayName || node.toolName,
      status: toRuntimeNodeStatus(node.status),
      summary: node.result?.error || node.result?.data ? String(node.result.data || node.result.error) : undefined,
      approval: node.approval,
    })),
    activeNodeId: toolChain.activeNodeId,
    isCollapsed: false,
    collapsePhase: 'expanded' as const,
    finalAnswer: {
      visible: false,
      streaming: false,
      content: '',
      revealState: 'hidden' as const,
    },
  };
}

// 辅助函数
function asString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined;
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined;
}
