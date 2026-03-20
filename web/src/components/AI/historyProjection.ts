import type { AIMessage, AIRunContent, AIRunProjection } from '../../api/modules/ai';
import { aiApi } from '../../api/modules/ai';
import { normalizeMarkdownContent } from './markdownContent';
import type { AssistantReplyActivity, AssistantReplyPlanStep, AssistantReplyRuntime, AssistantReplySegment, XChatMessage } from './types';

const projectionCache = new Map<string, AIRunProjection | null>();
const contentCache = new Map<string, AIRunContent | null>();
const INTERRUPTED_TOOL_MESSAGE = '执行未完成';
export const PROJECTION_MISSING_SUMMARY_LABEL = 'projection missing summary';
export const PROJECTION_UNRECOVERABLE_PLACEHOLDER = '回答内容不可恢复';

export function resetHistoryProjectionCache(): void {
  projectionCache.clear();
  contentCache.clear();
}

export async function loadRunProjection(runId: string): Promise<AIRunProjection | null> {
  if (!runId) return null;
  if (projectionCache.has(runId)) return projectionCache.get(runId) ?? null;
  try {
    const response = await aiApi.getRunProjection(runId);
    const projection = response.data || null;
    projectionCache.set(runId, projection);
    return projection;
  } catch {
    return null;
  }
}

export async function loadRunContent(contentId: string): Promise<AIRunContent | null> {
  if (!contentId) return null;
  if (contentCache.has(contentId)) return contentCache.get(contentId) ?? null;
  try {
    const response = await aiApi.getRunContent(contentId);
    const content = response.data || null;
    contentCache.set(contentId, content);
    return content;
  } catch {
    return null;
  }
}

export async function hydrateAssistantHistoryFromProjection(
  message: AIMessage,
): Promise<XChatMessage> {
  const fallbackContent = message.content || '';
  if (message.role !== 'assistant') {
    return {
      id: message.id,
      role: 'user',
      content: fallbackContent,
    };
  }

  const runId = (message as AIMessage & { run_id?: string }).run_id;
  if (!runId) {
    return {
      id: message.id,
      role: 'assistant',
      content: fallbackContent,
    };
  }

  const projection = await loadRunProjection(runId);
  if (!projection) {
    return {
      id: message.id,
      role: 'assistant',
      content: PROJECTION_UNRECOVERABLE_PLACEHOLDER,
      runtime: {
        activities: [],
        status: {
          kind: 'error',
          label: PROJECTION_MISSING_SUMMARY_LABEL,
        },
      },
    };
  }

  const summaryContent = normalizeMarkdownContent(projection.summary?.content || '').trim();
  if (!summaryContent) {
    return {
      id: message.id,
      role: 'assistant',
      content: PROJECTION_UNRECOVERABLE_PLACEHOLDER,
      runtime: {
        activities: [],
        status: {
          kind: 'error',
          label: PROJECTION_MISSING_SUMMARY_LABEL,
        },
      },
    };
  }

  const runtime = await projectionToRuntime(projection);
  return {
    id: message.id,
    role: 'assistant',
    content: summaryContent,
    runtime,
  };
}

export function isProjectionHydrationPending(message?: XChatMessage): boolean {
  return message?.role === 'assistant'
    && message.content === PROJECTION_UNRECOVERABLE_PLACEHOLDER
    && message.runtime?.status?.kind === 'error'
    && message.runtime.status.label === PROJECTION_MISSING_SUMMARY_LABEL;
}

async function projectionToRuntime(projection: AIRunProjection): Promise<AssistantReplyRuntime> {
  const activities: AssistantReplyActivity[] = [];
  const steps: AssistantReplyPlanStep[] = [];
  let nextExecutorStepIndex = 0;

  const ensureStep = (index: number, title: string): AssistantReplyPlanStep => {
    while (steps.length <= index) {
      steps.push({
        id: `history-step-${steps.length}`,
        title: `步骤 ${steps.length + 1}`,
        status: 'done',
      });
    }

    const current = steps[index];
    const nextTitle = title.trim() || current.title;
    const nextStep: AssistantReplyPlanStep = {
      ...current,
      id: current.id || `history-step-${index}`,
      title: nextTitle,
      status: 'done',
    };
    steps[index] = nextStep;
    return nextStep;
  };

  const syncPlanTitles = (titles: string[], completed: number) => {
    titles.forEach((title, offset) => {
      ensureStep(completed + offset, title);
    });
  };

  for (const block of projection.blocks) {
    if (block.type === 'agent_handoff') {
      activities.push({
        id: block.id,
        kind: 'agent_handoff',
        label: String(block.data?.to || block.title || 'handoff'),
        detail: String(block.data?.intent || ''),
        status: 'done',
      });
      continue;
    }
    if (block.type === 'plan') {
      if (block.steps?.length) {
        syncPlanTitles(block.steps, 0);
        nextExecutorStepIndex = 0;
      }
      activities.push({
        id: block.id,
        kind: 'plan',
        label: block.title,
        detail: block.steps?.join('\n'),
        status: 'done',
      });
      continue;
    }
    if (block.type === 'replan') {
      if (block.steps?.length) {
        const completed = Number(block.data?.completed || 0);
        syncPlanTitles(block.steps, completed);
        nextExecutorStepIndex = completed;
      }
      activities.push({
        id: block.id,
        kind: 'replan',
        label: block.title,
        detail: block.steps?.join('\n'),
        status: 'done',
      });
      continue;
    }
    if (block.type === 'error') {
      activities.push({
        id: block.id,
        kind: 'error',
        label: block.title,
        detail: String(block.data?.message || ''),
        status: 'error',
      });
    }
  }

  const executorBlocks = projection.blocks.filter((block) => block.type === 'executor');
  for (const block of executorBlocks) {
    const segments: AssistantReplySegment[] = [];
    let stepContent = '';
    const stepIndex = steps.length > nextExecutorStepIndex ? nextExecutorStepIndex : steps.length;
    const step = ensureStep(stepIndex, steps[stepIndex]?.title || block.title);

    for (const item of block.items || []) {
      if (item.type === 'content' && item.content_id) {
        const content = await loadRunContent(item.content_id);
        const text = normalizeMarkdownContent(content?.body_text || '');
        if (text) {
          segments.push({ type: 'text', text });
          stepContent += text;
        }
      }
      if (item.type === 'tool_call' && item.tool_call_id && item.tool_name) {
        const resultContent = item.result?.result_content_id
          ? await loadRunContent(item.result.result_content_id)
          : null;
        const rawContent = resultContent?.body_text || item.result?.preview;
        activities.push({
          id: item.tool_call_id,
          kind: 'tool',
          label: item.tool_name,
          detail: item.result
            ? item.result.preview
            : INTERRUPTED_TOOL_MESSAGE,
          rawContent,
          status: item.result
            ? item.result.status === 'done' ? 'done' : 'error'
            : 'error',
          stepIndex,
          arguments: item.arguments,
        });
        segments.push({ type: 'tool_ref', callId: item.tool_call_id });
      }
    }

    steps[stepIndex] = {
      ...step,
      id: step.id || block.id,
      status: 'done',
      content: stepContent || undefined,
      segments: segments.length > 0 ? segments : undefined,
    };
    nextExecutorStepIndex = stepIndex + 1;
  }

  return {
    activities,
    plan: steps.length > 0 ? { steps } : undefined,
    summary: projection.summary?.title ? {
      title: projection.summary.title,
    } : undefined,
    status: {
      kind: projection.status === 'failed_runtime' ? 'error' : 'completed',
      label: projection.status,
    },
  };
}
