import type { AIMessage, AIRunContent, AIRunProjection } from '../../api/modules/ai';
import { aiApi } from '../../api/modules/ai';
import { normalizeMarkdownContent } from './markdownContent';
import type { AssistantReplyActivity, AssistantReplyPlanStep, AssistantReplyRuntime, AssistantReplySegment, SlimExecutorBlock, XChatMessage } from './types';

const projectionCache = new Map<string, AIRunProjection | null>();
const contentCache = new Map<string, AIRunContent | null>();
const INTERRUPTED_TOOL_MESSAGE = '执行未完成';
export const PROJECTION_MISSING_SUMMARY_LABEL = 'projection missing summary';
export const PROJECTION_UNRECOVERABLE_PLACEHOLDER = '回答内容不可恢复';

function reconcileHistoricalPlan(
  previous: AssistantReplyPlanStep[],
  steps: string[],
  completed: number,
  isFinal: boolean,
): { steps: AssistantReplyPlanStep[]; activeStepIndex?: number } {
  const total = completed + steps.length;
  const nextSteps: AssistantReplyPlanStep[] = [];

  for (let index = 0; index < completed; index += 1) {
    const previousStep = previous[index];
    nextSteps.push({
      id: previousStep?.id || `historical-step-${index}`,
      title: previousStep?.title || `步骤 ${index + 1}`,
      status: 'done',
      loaded: false,
      sourceBlockIndex: previousStep?.sourceBlockIndex,
      sourceStepIndex: previousStep?.sourceStepIndex ?? index,
      unresolved: previousStep?.unresolved,
    });
  }

  steps.forEach((title, index) => {
    const nextIndex = index + completed;
    const previousStep = previous[nextIndex];
    nextSteps.push({
      id: previousStep?.id || `historical-step-${nextIndex}`,
      title,
      status: 'done',
      loaded: false,
      sourceBlockIndex: previousStep?.sourceBlockIndex,
      sourceStepIndex: previousStep?.sourceStepIndex ?? nextIndex,
      unresolved: previousStep?.unresolved,
    });
  });

  if (isFinal && previous.length > total) {
    previous.slice(total).forEach((step, index) => {
      nextSteps.push({
        ...step,
        id: step.id || `historical-step-${total + index}`,
        loaded: false,
      });
    });
  }

  return {
    steps: nextSteps,
    activeStepIndex: isFinal ? undefined : completed,
  };
}

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

/**
 * projectionToLazyRuntime 将 projection 转换为轻量级 runtime。
 * 只提取 steps 标题和 summary，不加载 executor 内容。
 * executor blocks 进行瘦身存储，只保留懒加载必须的字段。
 */
function projectionToLazyRuntime(projection: AIRunProjection): AssistantReplyRuntime {
  let steps: AssistantReplyPlanStep[] = [];
  let activeStepIndex: number | undefined;
  let executorIndex = 0;

  for (const block of projection.blocks) {
    if (block.type === 'plan') {
      const next = reconcileHistoricalPlan([], block.steps || [], 0, false);
      steps = next.steps;
      activeStepIndex = next.activeStepIndex;
      continue;
    }

    if (block.type === 'replan') {
      const completed = Number(block.data?.completed || 0);
      const isFinal = Boolean(block.data?.is_final);
      const next = reconcileHistoricalPlan(steps, block.steps || [], completed, isFinal);
      steps = next.steps;
      activeStepIndex = next.activeStepIndex;
      continue;
    }

    if (block.type === 'executor') {
      if (activeStepIndex !== undefined && steps[activeStepIndex] && steps[activeStepIndex].sourceBlockIndex === undefined) {
        steps[activeStepIndex] = {
          ...steps[activeStepIndex],
          sourceBlockIndex: executorIndex,
          unresolved: false,
        };
      }
      executorIndex += 1;
    }
  }

  if (steps.length > 0) {
    steps = steps.map((step, index) => ({
      ...step,
      id: step.id || `historical-step-${index}`,
      loaded: false,
      sourceStepIndex: step.sourceStepIndex ?? index,
      unresolved: step.unresolved ?? step.sourceBlockIndex === undefined,
    }));
  }

  // executor blocks 瘦身存储，只保留懒加载必须的字段
  const executorBlocks: SlimExecutorBlock[] = projection.blocks
    .filter(b => b.type === 'executor')
    .map(block => ({
      id: block.id,
      items: (block.items || []).map(item => ({
        type: item.type,
        content_id: item.content_id,
        tool_call_id: item.tool_call_id,
        tool_name: item.tool_name,
        arguments: item.arguments,
        result: item.result ? {
          status: item.result.status,
          preview: item.result.preview,
          result_content_id: item.result.result_content_id,
        } : undefined,
      })),
    }));

  return {
    activities: [],
    plan: steps.length > 0 ? { steps } : undefined,
    summary: projection.summary?.title ? { title: projection.summary.title } : undefined,
    status: {
      kind: projection.status === 'failed_runtime' ? 'error' : 'completed',
      label: projection.status,
    },
    _executorBlocks: executorBlocks,
  };
}

/**
 * loadStepContent 加载单个 step 的内容。
 * 根据 executor block 的 items 加载文本内容和工具调用信息。
 * 同时构建该 step 对应的 activities。
 */
export async function loadStepContent(
  block: SlimExecutorBlock,
  stepIndex: number,
): Promise<{
  content: string;
  segments: AssistantReplySegment[];
  activities: AssistantReplyActivity[];
}> {
  const segments: AssistantReplySegment[] = [];
  const activities: AssistantReplyActivity[] = [];
  let content = '';

  for (const item of block.items || []) {
    if (item.type === 'content' && item.content_id) {
      const runContent = await loadRunContent(item.content_id);
      const text = normalizeMarkdownContent(runContent?.body_text || '');
      if (text) {
        segments.push({ type: 'text', text });
        content += text;
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
        detail: item.result ? item.result.preview : INTERRUPTED_TOOL_MESSAGE,
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

  return { content, segments, activities };
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
    if (fallbackContent.trim()) {
      return {
        id: message.id,
        role: 'assistant',
        content: fallbackContent,
        runtime: {
          activities: [],
          status: {
            kind: 'error',
            label: message.error_message || '生成中断，请稍后重试',
          },
        },
      };
    }
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

  // 使用轻量级 runtime 转换，不加载 executor 内容
  const runtime = projectionToLazyRuntime(projection);
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
