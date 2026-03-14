import type { SSEChainNodeEvent, SSEChainStartedEvent, SSEFinalAnswerEvent } from '../../api/modules/ai';
import type {
  ChatTurn,
  ConfirmationRequest,
  ThoughtChainRuntimeState,
  RuntimeThoughtChainNode,
  RuntimeThoughtChainNodeKind,
  RuntimeThoughtChainNodeStatus,
} from './types';

type RuntimeEvent =
  | { type: 'chain_started'; data: SSEChainStartedEvent }
  | { type: 'chain_node_open'; data: SSEChainNodeEvent }
  | { type: 'chain_node_patch'; data: SSEChainNodeEvent }
  | { type: 'chain_node_replace'; data: SSEChainNodeEvent }
  | { type: 'chain_node_close'; data: SSEChainNodeEvent }
  | { type: 'chain_collapsed'; data: SSEChainStartedEvent }
  | { type: 'final_answer_started'; data: SSEFinalAnswerEvent }
  | { type: 'final_answer_delta'; data: SSEFinalAnswerEvent }
  | { type: 'final_answer_done'; data: SSEFinalAnswerEvent };

const defaultFinalAnswer = {
  visible: false,
  streaming: false,
  content: '',
  revealState: 'hidden',
} as const;

export function createThoughtChainRuntimeState(): ThoughtChainRuntimeState {
  return {
    nodes: [],
    isCollapsed: false,
    collapsePhase: 'expanded',
    finalAnswer: { ...defaultFinalAnswer },
  };
}

export function reduceThoughtChainRuntimeEvent(
  current: ThoughtChainRuntimeState | undefined,
  event: RuntimeEvent,
): ThoughtChainRuntimeState {
  const state = current || createThoughtChainRuntimeState();
  switch (event.type) {
    case 'chain_started':
      return {
        ...state,
        turnId: event.data.turn_id || state.turnId,
        isCollapsed: false,
        collapsePhase: 'expanded',
        finalAnswer: { ...defaultFinalAnswer },
      };
    case 'chain_node_open':
      return upsertNode(state, toRuntimeNode(event.data, 'active'), asString(event.data.turn_id));
    case 'chain_node_patch':
      return patchNode(state, event.data);
    case 'chain_node_replace':
      return replaceNode(state, event.data);
    case 'chain_node_close':
      return closeNode(state, event.data);
    case 'chain_collapsed':
      return {
        ...state,
        turnId: event.data.turn_id || state.turnId,
        isCollapsed: true,
        collapsePhase: 'collapsed',
      };
    case 'final_answer_started':
      return {
        ...state,
        turnId: event.data.turn_id || state.turnId,
        finalAnswer: {
          ...state.finalAnswer,
          visible: true,
          streaming: true,
          revealState: 'revealing',
        },
      };
    case 'final_answer_delta':
      return {
        ...state,
        turnId: event.data.turn_id || state.turnId,
        finalAnswer: {
          ...state.finalAnswer,
          visible: true,
          streaming: true,
          revealState: 'revealing',
          content: `${state.finalAnswer.content}${event.data.chunk || ''}`,
        },
      };
    case 'final_answer_done':
      return {
        ...state,
        turnId: event.data.turn_id || state.turnId,
        finalAnswer: {
          ...state.finalAnswer,
          visible: true,
          streaming: false,
          revealState: 'complete',
        },
      };
    default:
      return state;
  }
}

function upsertNode(
  state: ThoughtChainRuntimeState,
  node: RuntimeThoughtChainNode,
  turnID?: string,
): ThoughtChainRuntimeState {
  const nodes = [...state.nodes];
  const index = nodes.findIndex((item) => item.nodeId === node.nodeId);
  if (index >= 0) {
    nodes[index] = { ...nodes[index], ...node };
  } else {
    nodes.push(node);
  }
  return {
    ...state,
    turnId: turnID || state.turnId,
    nodes,
    activeNodeId: node.status === 'done' ? state.activeNodeId : node.nodeId,
    isCollapsed: false,
    collapsePhase: 'expanded',
  };
}

function patchNode(state: ThoughtChainRuntimeState, payload: SSEChainNodeEvent): ThoughtChainRuntimeState {
  const nodeID = asString(payload.node_id);
  if (!nodeID) {
    return state;
  }
  const nodes = state.nodes.map((node) => (
    node.nodeId === nodeID
      ? {
          ...node,
          title: asString(payload.title) || node.title,
          headline: asString(payload.headline) || node.headline,
          body: asString(payload.body) || node.body,
          structured: asRecord(payload.structured) || node.structured,
          raw: payload.raw ?? node.raw,
          summary: asString(payload.summary) || node.summary,
          details: Array.isArray(payload.details) ? payload.details : node.details,
          approval: payload.approval ? approvalFromPayload(payload.approval as Record<string, unknown>) : node.approval,
          status: normalizeNodeStatus(asString(payload.status)) || node.status,
        }
      : node
  ));
  return { ...state, nodes };
}

function closeNode(state: ThoughtChainRuntimeState, payload: SSEChainNodeEvent): ThoughtChainRuntimeState {
  const nodeID = asString(payload.node_id);
  if (!nodeID) {
    return state;
  }
  const nodes = state.nodes.map((node) => (
    node.nodeId === nodeID
      ? {
          ...node,
          status: normalizeNodeStatus(asString(payload.status)) || 'done',
        }
      : node
  ));
  const activeNode = state.activeNodeId === nodeID ? undefined : state.activeNodeId;
  return { ...state, nodes, activeNodeId: activeNode };
}

function replaceNode(state: ThoughtChainRuntimeState, payload: SSEChainNodeEvent): ThoughtChainRuntimeState {
  return upsertNode(state, {
    ...toRuntimeNode(payload, 'active'),
    status: normalizeNodeStatus(asString(payload.status)) || 'active',
  }, asString(payload.turn_id));
}

function toRuntimeNode(payload: SSEChainNodeEvent, fallbackStatus: RuntimeThoughtChainNodeStatus): RuntimeThoughtChainNode {
  return {
    nodeId: asString(payload.node_id) || `node:${Date.now()}`,
    kind: normalizeNodeKind(asString(payload.kind)),
    title: asString(payload.title) || '执行步骤',
    status: normalizeNodeStatus(asString(payload.status)) || fallbackStatus,
    headline: asString(payload.headline),
    body: asString(payload.body),
    structured: asRecord(payload.structured),
    raw: payload.raw,
    summary: asString(payload.summary),
    details: Array.isArray(payload.details) ? payload.details : undefined,
    approval: payload.approval ? approvalFromPayload(payload.approval as Record<string, unknown>) : undefined,
  };
}

function approvalFromPayload(payload: Record<string, unknown>): Omit<ConfirmationRequest, 'onConfirm' | 'onCancel'> & { requestId?: string } {
  return {
    id: asString(payload.request_id || payload.id) || 'approval',
    requestId: asString(payload.request_id || payload.id) || 'approval',
    title: asString(payload.title) || 'Approval required',
    description: asString(payload.summary || payload.description) || '当前步骤需要确认后继续执行',
    risk: (asString(payload.risk || payload.risk_level) || 'medium') as ConfirmationRequest['risk'],
    status: 'waiting_user',
    details: payload.details as Record<string, unknown> | undefined,
  };
}

function normalizeNodeKind(kind: string): RuntimeThoughtChainNodeKind {
  switch (kind) {
    case 'plan':
    case 'execute':
    case 'tool':
    case 'replan':
    case 'approval':
      return kind;
    default:
      return 'execute';
  }
}

function normalizeNodeStatus(status: string): RuntimeThoughtChainNodeStatus | undefined {
  switch (status) {
    case 'loading':
    case 'running':
      return 'active';
    case 'waiting':
    case 'waiting_approval':
      return 'waiting';
    case 'success':
    case 'done':
    case 'completed':
      return 'done';
    case 'error':
    case 'failed':
      return 'error';
    case 'pending':
      return 'pending';
    default:
      return undefined;
  }
}

export function runtimeStateFromReplayTurn(turn: ChatTurn | undefined): ThoughtChainRuntimeState | undefined {
  if (!turn || turn.role !== 'assistant') {
    return undefined;
  }
  const runtime = createThoughtChainRuntimeState();
  runtime.turnId = turn.id;

  for (const block of turn.blocks) {
    if (block.type === 'plan') {
      runtime.nodes.push({
        nodeId: block.id,
        kind: 'plan',
        title: block.title || '正在整理执行计划',
        status: normalizeReplayBlockStatus(block.status, turn.status),
        headline: block.content,
        structured: asRecord(block.data),
        summary: block.content,
        details: block.data?.steps as unknown[] | undefined,
      });
    } else if (block.type === 'tool') {
      runtime.nodes.push({
        nodeId: block.id,
        kind: 'tool',
        title: block.title || '正在调用工具',
        status: normalizeReplayBlockStatus(block.status, turn.status),
        headline: block.content || asString(block.data?.summary),
        structured: asRecord(block.data),
        raw: block.data,
        summary: block.content || asString(block.data?.summary),
        details: [block.data].filter(Boolean) as unknown[],
      });
    } else if (block.type === 'approval') {
      runtime.nodes.push({
        nodeId: block.id,
        kind: 'approval',
        title: block.title || '等待你确认',
        status: normalizeReplayBlockStatus(block.status, turn.status, true),
        headline: block.content || asString(block.data?.summary),
        summary: block.content || asString(block.data?.summary),
        approval: approvalFromApprovalEvent(block.data as Record<string, unknown>),
      });
    } else if (block.type === 'status' && (block.data?.phase === 'replanning' || block.data?.phase === 'replan')) {
      runtime.nodes.push({
        nodeId: block.id,
        kind: 'replan',
        title: block.title || '发现新信息，正在调整计划',
        status: normalizeReplayBlockStatus(block.status, turn.status),
        headline: block.content,
        summary: block.content,
      });
    }
  }

  const finalAnswerBlock = [...turn.blocks]
    .filter((block) => block.type === 'text' && (block.content || '').trim())
    .sort((a, b) => b.position - a.position)[0];

  if (finalAnswerBlock) {
    runtime.finalAnswer = {
      visible: true,
      streaming: finalAnswerBlock.streaming === true,
      content: finalAnswerBlock.content || '',
      revealState: finalAnswerBlock.streaming ? 'revealing' : 'complete',
    };
  }

  if (turn.status === 'completed' || turn.completedAt) {
    runtime.isCollapsed = runtime.nodes.length > 0;
    runtime.collapsePhase = runtime.isCollapsed ? 'collapsed' : 'expanded';
    runtime.nodes = runtime.nodes.map((node) => ({
      ...node,
      status: node.status === 'error' ? 'error' : node.status === 'waiting' ? 'waiting' : 'done',
    }));
  }

  if (runtime.nodes.length === 0 && !runtime.finalAnswer.content) {
    return undefined;
  }

  runtime.activeNodeId = runtime.nodes.find((node) => node.status === 'active' || node.status === 'waiting')?.nodeId;
  return runtime;
}

function normalizeReplayBlockStatus(status?: string, turnStatus?: string, waiting = false): RuntimeThoughtChainNodeStatus {
  if (waiting || status === 'waiting_approval') {
    return 'waiting';
  }
  if (status === 'error' || turnStatus === 'error') {
    return 'error';
  }
  if (status === 'running' || status === 'streaming' || turnStatus === 'streaming') {
    return 'active';
  }
  return 'done';
}

function approvalFromApprovalEvent(data: Record<string, unknown> | undefined) {
  if (!data) {
    return undefined;
  }
  return approvalFromPayload(data);
}

function asString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined;
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? value as Record<string, unknown>
    : undefined;
}
