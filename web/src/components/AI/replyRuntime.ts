import type {
  AssistantReplyActivity,
  PendingRunMetadata,
  AssistantReplyPlan,
  AssistantReplyPlanStep,
  AssistantReplyRuntime,
  AssistantReplyRuntimeStatus,
  AssistantReplySegment,
  AssistantReplyTodo,
  XChatMessage,
} from './types';

const PLACEHOLDER_CONTENT = '[准备中]';
const SOFT_TIMEOUT_MESSAGE = '工具执行较慢，正在继续等待结果…';
const INTERRUPTED_TOOL_MESSAGE = '执行未完成';
const MAX_ACTIVITIES = 50;
const APPROVAL_SUMMARY_MAX_ROWS = 8;
const APPROVAL_SUMMARY_MAX_VALUE_LENGTH = 80;

type ApprovalPreviewSummaryRow = NonNullable<AssistantReplyActivity['approvalPreviewSummary']>[number];

const APPROVAL_SUMMARY_PREFERRED_FIELDS: Array<{
  key: string;
  aliases: string[];
}> = [
  { key: 'cluster', aliases: ['cluster'] },
  { key: 'namespace', aliases: ['namespace'] },
  { key: 'resource', aliases: ['resource', 'resourceType'] },
  { key: 'kind', aliases: ['kind'] },
  { key: 'name', aliases: ['name'] },
  { key: 'action', aliases: ['action'] },
  { key: 'risk', aliases: ['risk', 'riskLevel'] },
];

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function truncatePreviewValue(value: string): string {
  if (value.length <= APPROVAL_SUMMARY_MAX_VALUE_LENGTH) {
    return value;
  }
  return `${value.slice(0, APPROVAL_SUMMARY_MAX_VALUE_LENGTH - 1)}…`;
}

function formatApprovalPreviewValue(value: unknown): string | undefined {
  if (value === null || value === undefined) {
    return undefined;
  }

  if (typeof value === 'string') {
    const trimmed = value.trim();
    return trimmed ? truncatePreviewValue(trimmed) : undefined;
  }

  if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
    return String(value);
  }

  if (Array.isArray(value)) {
    if (value.length === 0) {
      return undefined;
    }
    const serialized = value.map((item) => formatApprovalPreviewValue(item) || '').filter(Boolean).join(', ');
    return serialized ? truncatePreviewValue(serialized) : truncatePreviewValue(JSON.stringify(value));
  }

  if (isRecord(value)) {
    const keys = Object.keys(value).sort();
    if (keys.length === 0) {
      return undefined;
    }
    return truncatePreviewValue(JSON.stringify(value));
  }

  return truncatePreviewValue(String(value));
}

function flattenApprovalPreview(
  value: Record<string, unknown>,
  prefix = '',
): Array<{ key: string; value: unknown }> {
  return Object.keys(value)
    .sort((left, right) => left.localeCompare(right))
    .flatMap((key) => {
      const nextKey = prefix ? `${prefix}.${key}` : key;
      const nextValue = value[key];
      if (isRecord(nextValue)) {
        const nested = flattenApprovalPreview(nextValue, nextKey);
        if (nested.length > 0) {
          return nested;
        }
      }
      return [{ key: nextKey, value: nextValue }];
    });
}

function deriveApprovalPreviewSummary(preview: Record<string, unknown>): ApprovalPreviewSummaryRow[] {
  if (!isRecord(preview) || Object.keys(preview).length === 0) {
    return [];
  }

  const rows: ApprovalPreviewSummaryRow[] = [];
  const consumedTopLevelKeys = new Set<string>();

  APPROVAL_SUMMARY_PREFERRED_FIELDS.forEach(({ key, aliases }) => {
    const matchedAlias = aliases.find((alias) => Object.prototype.hasOwnProperty.call(preview, alias));
    if (!matchedAlias) {
      return;
    }
    const formattedValue = formatApprovalPreviewValue(preview[matchedAlias]);
    if (!formattedValue) {
      return;
    }
    rows.push({
      key,
      label: key,
      value: formattedValue,
    });
    consumedTopLevelKeys.add(matchedAlias);
  });

  if (rows.length >= APPROVAL_SUMMARY_MAX_ROWS) {
    return rows.slice(0, APPROVAL_SUMMARY_MAX_ROWS);
  }

  const fallbackRows = flattenApprovalPreview(preview)
    .filter(({ key }) => !consumedTopLevelKeys.has(key.split('.')[0] || key))
    .map(({ key, value }) => {
      const formattedValue = formatApprovalPreviewValue(value);
      if (!formattedValue) {
        return null;
      }
      return {
        key,
        label: key,
        value: formattedValue,
      };
    })
    .filter((row): row is ApprovalPreviewSummaryRow => Boolean(row));

  return [...rows, ...fallbackRows].slice(0, APPROVAL_SUMMARY_MAX_ROWS);
}

function mergePendingRun(
  runtime: AssistantReplyRuntime,
  partial?: Partial<PendingRunMetadata> | null,
): AssistantReplyRuntime {
  if (!partial) {
    return runtime;
  }

  const nextPendingRun: PendingRunMetadata = {
    runId: partial.runId || runtime.pendingRun?.runId || '',
    status: partial.status || runtime.pendingRun?.status || 'running',
    resumable: partial.resumable ?? runtime.pendingRun?.resumable ?? true,
    sessionId: partial.sessionId ?? runtime.pendingRun?.sessionId,
    clientRequestId: partial.clientRequestId ?? runtime.pendingRun?.clientRequestId,
    latestEventId: partial.latestEventId ?? runtime.pendingRun?.latestEventId,
    approvalId: partial.approvalId ?? runtime.pendingRun?.approvalId,
    messageId: partial.messageId ?? runtime.pendingRun?.messageId,
    updatedAt: partial.updatedAt ?? runtime.pendingRun?.updatedAt,
  };

  if (!nextPendingRun.runId) {
    return {
      ...runtime,
      pendingRun: undefined,
    };
  }

  return {
    ...runtime,
    pendingRun: nextPendingRun,
  };
}

function clearPendingRun(runtime: AssistantReplyRuntime): AssistantReplyRuntime {
  if (!runtime.pendingRun) {
    return runtime;
  }
  return {
    ...runtime,
    pendingRun: undefined,
  };
}

function updateApprovalActivities(
  runtime: AssistantReplyRuntime,
  approvalState: AssistantReplyActivity['approvalState'],
  approvalMessage?: string,
): AssistantReplyRuntime {
  return {
    ...runtime,
    activities: runtime.activities.map((activity) => {
      if (activity.kind !== 'tool_approval') {
        return activity;
      }
      return {
        ...activity,
        approvalState,
        approvalMessage: approvalMessage ?? activity.approvalMessage,
      };
    }),
  };
}

function buildApprovalRuntimeStatus(
  kind: 'approved_resuming' | 'approved_retrying' | 'approved_failed_terminal' | 'approved_done' | 'expired',
  label: string,
): AssistantReplyRuntimeStatus {
  return { kind, label };
}

function buildPlanStepId(index: number): string {
  return `plan-step-${index}`;
}

function normalizeTodos(todos?: AssistantReplyTodo[] | null): AssistantReplyTodo[] {
  if (!Array.isArray(todos)) {
    return [];
  }

  return todos.map((todo, index) => ({
    id: todo.id || `todo-${index}`,
    content: todo.content || '',
    activeForm: todo.activeForm,
    status: todo.status || 'pending',
    cluster: todo.cluster,
    namespace: todo.namespace,
    resourceType: todo.resourceType,
    riskLevel: todo.riskLevel,
    requiresApproval: todo.requiresApproval,
    estimatedDuration: todo.estimatedDuration,
    dependsOn: Array.isArray(todo.dependsOn) ? [...todo.dependsOn] : undefined,
  }));
}

export function createEmptyAssistantRuntime(): AssistantReplyRuntime {
  return {
    activities: [],
    plan: undefined,
    phase: undefined,
    phaseLabel: undefined,
    pendingRun: undefined,
    summary: undefined,
    status: undefined,
    todos: [],
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

export function applyRunResuming(runtime: AssistantReplyRuntime): AssistantReplyRuntime {
  return mergePendingRun(updateApprovalActivities({
    ...runtime,
    phase: 'executing',
    phaseLabel: '已批准，恢复中',
    status: buildApprovalRuntimeStatus('approved_resuming', '已批准，恢复中'),
  }, 'approved_resuming'), {
    status: 'resuming',
  });
}

export function applyRunResumed(runtime: AssistantReplyRuntime): AssistantReplyRuntime {
  return mergePendingRun(updateApprovalActivities({
    ...runtime,
    phase: 'executing',
    phaseLabel: '已批准，继续执行',
    status: buildApprovalRuntimeStatus('approved_resuming', '已批准，继续执行'),
  }, 'approved_done'), {
    status: 'running',
    resumable: true,
  });
}

export function applyRunResumeFailed(
  runtime: AssistantReplyRuntime,
  payload: { retryable?: boolean; message?: string },
): AssistantReplyRuntime {
  if (payload.retryable) {
    return mergePendingRun(updateApprovalActivities({
      ...runtime,
      phase: 'executing',
      phaseLabel: '已批准，重试恢复中',
      status: buildApprovalRuntimeStatus('approved_retrying', payload.message || '恢复失败，正在重试'),
    }, 'approved_retrying', payload.message), {
      status: 'resume_failed_retryable',
    });
  }
  return clearPendingRun(updateApprovalActivities({
    ...runtime,
    phase: 'interrupted',
    phaseLabel: '已批准，恢复失败',
    status: buildApprovalRuntimeStatus('approved_failed_terminal', payload.message || '恢复失败，等待人工处理'),
  }, 'approved_failed_terminal', payload.message));
}

export function applyApprovalExpired(runtime: AssistantReplyRuntime): AssistantReplyRuntime {
  return clearPendingRun(updateApprovalActivities({
    ...runtime,
    phase: 'interrupted',
    phaseLabel: '审批已过期',
    status: buildApprovalRuntimeStatus('expired', '审批已过期'),
  }, 'expired'));
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
  const approvalPreview = isRecord(payload.preview) ? { ...payload.preview } : {};
  return mergePendingRun(upsertActivity(
    runtime,
    {
      id: payload.call_id,
      kind: 'tool_approval',
      label: payload.tool_name,
      detail: `等待审批 ${payload.timeout_seconds}s`,
      status: 'pending',
      approvalId: payload.approval_id,
      approvalState: 'waiting-approval',
      approvalPreview,
      approvalPreviewSummary: deriveApprovalPreviewSummary(approvalPreview),
      approvalTimeoutSeconds: payload.timeout_seconds,
      stepIndex: activeStepIndex,
    },
    (item) => item.id === payload.call_id,
  ), {
    approvalId: payload.approval_id,
    status: 'waiting_approval',
    resumable: true,
  });
}

export function applyRuntimeSnapshot(
  _runtime: AssistantReplyRuntime,
  snapshot: Partial<AssistantReplyRuntime> & { todos?: AssistantReplyTodo[] },
): AssistantReplyRuntime {
  return {
    activities: Array.isArray(snapshot.activities) ? [...snapshot.activities] : [],
    plan: snapshot.plan ? { ...snapshot.plan } : undefined,
    phase: snapshot.phase,
    phaseLabel: snapshot.phaseLabel,
    pendingRun: snapshot.pendingRun ? { ...snapshot.pendingRun } : undefined,
    summary: snapshot.summary ? { ...snapshot.summary } : undefined,
    status: snapshot.status ? { ...snapshot.status } : undefined,
    todos: normalizeTodos(snapshot.todos),
    _executorBlocks: snapshot._executorBlocks ? [...snapshot._executorBlocks] : undefined,
  };
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

export function applyRunState(
  runtime: AssistantReplyRuntime,
  payload: { run_id: string; status: string; agent?: string; summary?: string },
): AssistantReplyRuntime {
  switch (payload.status) {
    case 'waiting_approval':
      return mergePendingRun({
        ...runtime,
        phase: 'interrupted',
        phaseLabel: '等待审批',
        status: {
          kind: 'waiting_approval',
          label: '等待审批',
        },
      }, {
        runId: payload.run_id,
        status: 'waiting_approval',
        resumable: true,
      });
    case 'resuming':
      return mergePendingRun({
        ...runtime,
        phase: 'executing',
        phaseLabel: '恢复中',
        status: {
          kind: 'resuming',
          label: '恢复中',
        },
      }, {
        runId: payload.run_id,
        status: 'resuming',
        resumable: true,
      });
    case 'running':
      return mergePendingRun({
        ...runtime,
        phase: 'executing',
        phaseLabel: '执行中',
        status: {
          kind: 'streaming',
          label: '执行中',
        },
      }, {
        runId: payload.run_id,
        status: 'running',
        resumable: true,
      });
    case 'resume_failed_retryable':
      return mergePendingRun({
        ...runtime,
        phase: 'interrupted',
        phaseLabel: '恢复失败，可重试',
        status: {
          kind: 'resume_failed_retryable',
          label: '恢复失败，可重试',
        },
      }, {
        runId: payload.run_id,
        status: 'resume_failed_retryable',
        resumable: true,
      });
    case 'completed':
    case 'completed_with_tool_errors':
      return clearPendingRun({
        ...runtime,
        phase: 'completed',
        phaseLabel: '已完成',
        status: {
          kind: 'completed',
          label: '已完成',
        },
      });
    case 'failed':
    case 'failed_runtime':
    case 'cancelled':
    case 'expired':
      return clearPendingRun({
        ...runtime,
        phase: 'interrupted',
        phaseLabel: '运行失败',
        status: {
          kind: 'failed',
          label: '运行失败',
        },
      });
    default:
      return runtime;
  }
}

export function applyDone(runtime: AssistantReplyRuntime): AssistantReplyRuntime {
  const approvalFlow = runtime.status?.kind && runtime.status.kind.startsWith('approved_');
  const terminalFailure = runtime.status?.kind === 'failed';
  return clearPendingRun({
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
    phase: terminalFailure ? 'interrupted' : 'completed',
    phaseLabel: terminalFailure ? (runtime.phaseLabel || '运行失败') : (runtime.phaseLabel || '已完成'),
    status: terminalFailure
      ? runtime.status
      : {
          kind: approvalFlow ? 'approved_done' : 'completed',
          label: approvalFlow ? '已批准，恢复完成' : `已生成 ${new Date().toLocaleString('zh-CN')}`,
        },
  });
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
    runtime: clearPendingRun({
      ...(message.runtime || createEmptyAssistantRuntime()),
      status: {
        kind: 'failed',
        label: payload.message || 'Request failed',
      },
    }),
  };
}
