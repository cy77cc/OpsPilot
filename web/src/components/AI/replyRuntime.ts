import type {
  AssistantReplyActivity,
  AssistantReplyPlan,
  AssistantReplyPlanStep,
  AssistantReplyRuntime,
  AssistantReplySegment,
  XChatMessage,
} from './types';

const PLACEHOLDER_CONTENT = '[准备中]';
const SOFT_TIMEOUT_MESSAGE = '工具执行较慢，正在继续等待结果…';
const INTERRUPTED_TOOL_MESSAGE = '执行未完成';
const MAX_ACTIVITIES = 50;

function buildPlanStepId(index: number): string {
  return `plan-step-${index}`;
}

export function createEmptyAssistantRuntime(): AssistantReplyRuntime {
  return {
    activities: [],
    plan: undefined,
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

function reconcilePlan(
  previous: AssistantReplyPlan | undefined,
  steps: string[],
  completed: number,
  isFinal: boolean,
): AssistantReplyPlan {
  const total = completed + steps.length;
  const nextSteps: AssistantReplyPlanStep[] = [];

  for (let index = 0; index < completed; index += 1) {
    nextSteps.push({
      id: buildPlanStepId(index),
      title: previous?.steps[index]?.title || `步骤 ${index + 1}`,
      status: 'done',
      content: previous?.steps[index]?.content,
    });
  }

  steps.forEach((title, index) => {
    nextSteps.push({
      id: buildPlanStepId(index + completed),
      title,
      status: isFinal ? 'done' : index === 0 ? 'active' : 'pending',
      content: previous?.steps[index + completed]?.content,
    });
  });

  if (isFinal && previous && previous.steps.length > total) {
    previous.steps.slice(total).forEach((step, index) => {
      nextSteps.push({
        id: buildPlanStepId(total + index),
        title: step.title,
        status: 'done',
        content: step.content,
      });
    });
  }

  return {
    activeStepIndex: isFinal ? undefined : completed,
    steps: nextSteps,
  };
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

function appendSegmentToActiveStep(
  runtime: AssistantReplyRuntime,
  segment: AssistantReplySegment,
): AssistantReplyRuntime {
  const stepIndex = runtime.plan?.activeStepIndex;
  if (stepIndex === undefined || !runtime.plan) {
    return runtime;
  }

  return {
    ...runtime,
    plan: {
      ...runtime.plan,
      steps: runtime.plan.steps.map((step, index) => {
        if (index !== stepIndex) {
          return step;
        }
        const segments = [...(step.segments || [])];
        if (segments.length === 0 && step.content) {
          segments.push({ type: 'text', text: step.content });
        }
        const previous = segments[segments.length - 1];
        if (segment.type === 'text' && previous?.type === 'text') {
          previous.text = `${previous.text || ''}${segment.text || ''}`;
        } else {
          segments.push(segment);
        }
        return {
          ...step,
          content: segment.type === 'text'
            ? `${step.content || ''}${segment.text || ''}`
            : step.content,
          segments,
        };
      }),
    },
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

export function applyStepDelta(
  runtime: AssistantReplyRuntime,
  payload: { content: string },
): AssistantReplyRuntime {
  return appendSegmentToActiveStep(runtime, {
    type: 'text',
    text: payload.content || '',
  });
}

export function applyToolCall(
  runtime: AssistantReplyRuntime,
  payload: { call_id: string; tool_name: string; arguments?: Record<string, unknown> },
): AssistantReplyRuntime {
  const activeStepIndex = runtime.plan?.activeStepIndex;
  const withActivity = upsertActivity(
    runtime,
    {
      id: payload.call_id,
      kind: 'tool',
      label: payload.tool_name,
      status: 'active',
      stepIndex: activeStepIndex,
      arguments: payload.arguments,
    },
    (item) => item.id === payload.call_id,
  );

  if (!payload.call_id) {
    return withActivity;
  }

  return appendSegmentToActiveStep(withActivity, {
    type: 'tool_ref',
    callId: payload.call_id,
  });
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
    plan: reconcilePlan(undefined, payload.steps, 0, false),
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
    plan: reconcilePlan(runtime.plan, payload.steps, payload.completed, payload.is_final),
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
  const activeStepIndex = runtime.plan?.activeStepIndex;
  return upsertActivity(
    runtime,
    {
      id: payload.call_id,
      kind: 'tool_approval',
      label: payload.tool_name,
      detail: `等待审批 ${payload.timeout_seconds}s`,
      status: 'pending',
      stepIndex: activeStepIndex,
    },
    (item) => item.id === payload.call_id,
  );
}

export function applyToolResult(
  runtime: AssistantReplyRuntime,
  payload: { call_id: string; tool_name: string; content: string; status?: string },
): AssistantReplyRuntime {
  const existing = runtime.activities.find((item) => item.id === payload.call_id);
  const detailContent = payload.content.length > 200
    ? payload.content.slice(0, 200)
    : payload.content;
  return upsertActivity(
    runtime,
    {
      id: payload.call_id,
      kind: 'tool',
      label: payload.tool_name,
      detail: detailContent,
      rawContent: payload.content,
      status: payload.status === 'error' ? 'error' : 'done',
      stepIndex: existing?.stepIndex ?? runtime.plan?.activeStepIndex,
      arguments: existing?.arguments,
    },
    (item) => item.id === payload.call_id,
  );
}

export function applyDone(runtime: AssistantReplyRuntime): AssistantReplyRuntime {
  return {
    ...runtime,
    activities: runtime.activities.map((activity) => {
      if (activity.kind === 'tool' && activity.status === 'active') {
        return {
          ...activity,
          status: 'error',
          detail: activity.detail || INTERRUPTED_TOOL_MESSAGE,
        };
      }
      return activity;
    }),
    plan: runtime.plan
      ? {
          ...runtime.plan,
          activeStepIndex: undefined,
          steps: runtime.plan.steps.map((step) => ({ ...step, status: 'done' })),
        }
      : runtime.plan,
    phase: 'completed',
    phaseLabel: runtime.phaseLabel || '已完成',
    status: {
      kind: 'completed',
      label: `已生成 ${new Date().toLocaleString('zh-CN')}`,
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
