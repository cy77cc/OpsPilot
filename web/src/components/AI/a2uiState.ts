import type {
  A2UIAgentHandoffEvent,
  A2UIDeltaEvent,
  A2UIDoneEvent,
  A2UIErrorEvent,
  A2UIMetaEvent,
  A2UIPlanEvent,
  A2UIReplanEvent,
  A2UIToolApprovalEvent,
  A2UIToolCallEvent,
  A2UIToolResultEvent,
} from '../../api/modules/ai';

export interface A2UIPlanItem {
  content: string;
  status: 'pending' | 'active' | 'done';
}

export interface A2UIToolState {
  callId: string;
  toolName: string;
  status: 'running' | 'waiting_approval' | 'done';
  content?: string;
}

export interface A2UIState {
  sessionId?: string;
  runId?: string;
  turn?: number;
  handoff?: A2UIAgentHandoffEvent;
  planItems: A2UIPlanItem[];
  tools: A2UIToolState[];
  approval?: A2UIToolApprovalEvent;
  content: string;
  done: boolean;
  error?: A2UIErrorEvent;
}

export type A2UIStateEvent =
  | { type: 'meta'; payload: A2UIMetaEvent }
  | { type: 'agent_handoff'; payload: A2UIAgentHandoffEvent }
  | { type: 'plan'; payload: A2UIPlanEvent }
  | { type: 'replan'; payload: A2UIReplanEvent }
  | { type: 'delta'; payload: A2UIDeltaEvent }
  | { type: 'tool_call'; payload: A2UIToolCallEvent }
  | { type: 'tool_approval'; payload: A2UIToolApprovalEvent }
  | { type: 'tool_result'; payload: A2UIToolResultEvent }
  | { type: 'done'; payload: A2UIDoneEvent }
  | { type: 'error'; payload: A2UIErrorEvent };

export const initialA2UIState: A2UIState = {
  planItems: [],
  tools: [],
  content: '',
  done: false,
};

export function reduceA2UIState(state: A2UIState, event: A2UIStateEvent): A2UIState {
  switch (event.type) {
    case 'meta':
      return {
        ...state,
        sessionId: event.payload.session_id,
        runId: event.payload.run_id,
        turn: event.payload.turn,
      };
    case 'agent_handoff':
      return {
        ...state,
        handoff: event.payload,
      };
    case 'plan':
      return {
        ...state,
        planItems: event.payload.steps.map((content, index) => ({
          content,
          status: index === 0 ? 'active' : 'pending',
        })),
      };
    case 'replan':
      return {
        ...state,
        planItems: event.payload.is_final
          ? state.planItems.map((item) => ({ ...item, status: 'done' }))
          : reconcilePlanItems(state.planItems, event.payload),
      };
    case 'delta':
      return {
        ...state,
        content: `${state.content}${event.payload.content}`,
      };
    case 'tool_call':
      return {
        ...state,
        tools: [
          ...state.tools.filter((tool) => tool.callId !== event.payload.call_id),
          {
            callId: event.payload.call_id,
            toolName: event.payload.tool_name,
            status: 'running',
          },
        ],
      };
    case 'tool_approval':
      return {
        ...state,
        approval: event.payload,
        tools: state.tools.map((tool) => (tool.callId === event.payload.call_id
          ? { ...tool, status: 'waiting_approval' }
          : tool)),
      };
    case 'tool_result':
      return {
        ...state,
        approval: state.approval?.call_id === event.payload.call_id ? undefined : state.approval,
        tools: state.tools.map((tool) => (tool.callId === event.payload.call_id
          ? { ...tool, status: 'done', content: event.payload.content }
          : tool)),
      };
    case 'done':
      return {
        ...state,
        done: true,
      };
    case 'error':
      return {
        ...state,
        error: event.payload,
      };
    default:
      return state;
  }
}

function reconcilePlanItems(previous: A2UIPlanItem[], payload: A2UIReplanEvent): A2UIPlanItem[] {
  const completed = Math.max(0, payload.completed);
  const remaining = payload.steps;
  const next: A2UIPlanItem[] = [];

  for (let i = 0; i < completed; i += 1) {
    next.push({
      content: previous[i]?.content || `completed-${i + 1}`,
      status: 'done',
    });
  }

  remaining.forEach((content, index) => {
    next.push({
      content,
      status: index === 0 ? 'active' : 'pending',
    });
  });

  return next;
}
