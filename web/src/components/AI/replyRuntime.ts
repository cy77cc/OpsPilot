import type {
  AssistantReplyActivity,
  AssistantReplyRuntime,
  XChatMessage,
} from './types';

const PLACEHOLDER_CONTENT = '[准备中]';
const SOFT_TIMEOUT_MESSAGE = '工具执行较慢，正在继续等待结果…';
const MAX_ACTIVITIES = 50;

export function createEmptyAssistantRuntime(): AssistantReplyRuntime {
  return {
    activities: [],
    phase: undefined,
    phaseLabel: undefined,
    summary: undefined,
    status: undefined,
  };
}

function trimActivities(activities: AssistantReplyActivity[]): AssistantReplyActivity[] {
  if (activities.length <= MAX_ACTIVITIES) {
    return activities;
  }
  return activities.slice(activities.length - MAX_ACTIVITIES);
}

function upsertActivity(
  runtime: AssistantReplyRuntime,
  nextActivity: AssistantReplyActivity,
  match: (activity: AssistantReplyActivity) => boolean,
): AssistantReplyRuntime {
  const index = runtime.activities.findIndex(match);
  if (index === -1) {
    return {
      ...runtime,
      activities: trimActivities([...runtime.activities, nextActivity]),
    };
  }

  const activities = [...runtime.activities];
  activities[index] = {
    ...activities[index],
    ...nextActivity,
  };

  return {
    ...runtime,
    activities,
  };
}

export function applyDelta(
  message: Pick<XChatMessage, 'content' | 'runtime'>,
  payload: { content: string },
): Pick<XChatMessage, 'content' | 'runtime'> {
  const chunk = payload.content || '';
  const current = message.content || '';
  const nextContent =
    !current || current === PLACEHOLDER_CONTENT ? chunk : `${current}${chunk}`;

  return {
    content: nextContent,
    runtime: message.runtime,
  };
}

export function applyToolCall(
  runtime: AssistantReplyRuntime,
  payload: { call_id: string; tool_name: string; arguments?: Record<string, unknown> },
): AssistantReplyRuntime {
  return upsertActivity(
    runtime,
    {
      id: payload.call_id,
      kind: 'tool_call',
      label: payload.tool_name,
      status: 'active',
    },
    (item) => item.id === payload.call_id,
  );
}

export function applyMeta(runtime: AssistantReplyRuntime): AssistantReplyRuntime {
  return {
    ...runtime,
    phase: 'preparing',
    phaseLabel: '准备中',
  };
}

export function applySoftTimeout(runtime: AssistantReplyRuntime): AssistantReplyRuntime {
  return {
    ...runtime,
    status: {
      kind: 'soft-timeout',
      label: SOFT_TIMEOUT_MESSAGE,
    },
  };
}

export function applyAgentHandoff(
  runtime: AssistantReplyRuntime,
  payload: { from: string; to: string; intent: string },
): AssistantReplyRuntime {
  return {
    ...upsertActivity(
      runtime,
      {
        id: `handoff:${payload.to}`,
        kind: 'agent_handoff',
        label: payload.to,
        detail: payload.intent,
        status: 'done',
      },
      (item) => item.id === `handoff:${payload.to}`,
    ),
    phase: 'executing',
    phaseLabel: `${payload.to} 开始处理`,
  };
}

export function applyPlan(
  runtime: AssistantReplyRuntime,
  payload: { steps: string[]; iteration: number },
): AssistantReplyRuntime {
  return {
    ...upsertActivity(
      runtime,
      {
        id: 'planning',
        kind: 'plan',
        label: '规划处理步骤',
        detail: payload.steps.join(' -> '),
        status: 'done',
      },
      (item) => item.id === 'planning',
    ),
    phase: 'planning',
    phaseLabel: '正在规划处理方式',
  };
}

export function applyReplan(
  runtime: AssistantReplyRuntime,
  payload: { steps: string[]; completed: number; iteration: number; is_final: boolean },
): AssistantReplyRuntime {
  return {
    ...upsertActivity(
      runtime,
      {
        id: 'planning',
        kind: 'replan',
        label: '更新处理计划',
        detail: payload.steps.join(' -> '),
        status: payload.is_final ? 'done' : 'active',
      },
      (item) => item.id === 'planning',
    ),
    phase: 'planning',
    phaseLabel: '正在调整处理计划',
  };
}

export function applyToolApproval(
  runtime: AssistantReplyRuntime,
  payload: {
    approval_id: string;
    call_id: string;
    tool_name: string;
    preview: Record<string, unknown>;
    timeout_seconds: number;
  },
): AssistantReplyRuntime {
  return upsertActivity(
    runtime,
    {
      id: payload.call_id,
      kind: 'tool_approval',
      label: payload.tool_name,
      detail: `等待审批 ${payload.timeout_seconds}s`,
      status: 'pending',
    },
    (item) => item.id === payload.call_id,
  );
}

export function applyToolResult(
  runtime: AssistantReplyRuntime,
  payload: { call_id: string; tool_name: string; content: string },
): AssistantReplyRuntime {
  return upsertActivity(
    runtime,
    {
      id: payload.call_id,
      kind: 'tool_result',
      label: payload.tool_name,
      detail: payload.content,
      status: 'done',
    },
    (item) => item.id === payload.call_id,
  );
}

export function applyDone(runtime: AssistantReplyRuntime): AssistantReplyRuntime {
  return {
    ...runtime,
    phase: 'completed',
    phaseLabel: runtime.phaseLabel || '已完成',
    status: {
      kind: 'completed',
      label: '已生成',
    },
  };
}

export function applyRecoverableError(
  message: Pick<XChatMessage, 'content' | 'runtime'>,
  payload: { message: string },
): Pick<XChatMessage, 'content' | 'runtime'> {
  return {
    content: message.content,
    runtime: {
      ...(message.runtime || createEmptyAssistantRuntime()),
      status: {
        kind: 'soft-timeout',
        label: payload.message || SOFT_TIMEOUT_MESSAGE,
      },
    },
  };
}

export function applyTerminalError(
  message: Pick<XChatMessage, 'content' | 'runtime'>,
  payload: { message: string },
): Pick<XChatMessage, 'content' | 'runtime'> {
  return {
    content: message.content,
    runtime: {
      ...(message.runtime || createEmptyAssistantRuntime()),
      status: {
        kind: 'error',
        label: payload.message || 'Request failed',
      },
    },
  };
}
