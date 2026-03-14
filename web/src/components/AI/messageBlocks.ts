import type { EmbeddedRecommendation, PlanStep, TurnBlock } from './types';

export interface AssistantMessageInput {
  content?: string;
  thinking?: string;
  showThinking?: boolean;
  rawEvidence?: string[];
  recommendations?: EmbeddedRecommendation[];
  isStreaming?: boolean;
}

interface BaseBlock {
  id: string;
}

export interface ThinkingBlock extends BaseBlock {
  type: 'thinking';
  content: string;
  isStreaming?: boolean;
}

export interface MarkdownBlock extends BaseBlock {
  type: 'markdown';
  content: string;
  streaming?: boolean;
}

export interface StatusBlock extends BaseBlock {
  type: 'status';
  title?: string;
  content: string;
  status?: string;
  payload?: Record<string, unknown>;
}

export interface PlanBlock extends BaseBlock {
  type: 'plan';
  title?: string;
  content?: string;
  payload?: Record<string, unknown>;
  steps?: PlanStep[];
  total?: number;
}

export interface ToolExecutionBlock extends BaseBlock {
  type: 'tool';
  title?: string;
  status?: string;
  payload?: Record<string, unknown>;
}

export interface ApprovalBlock extends BaseBlock {
  type: 'approval';
  title?: string;
  payload?: Record<string, unknown>;
}

export interface EvidenceBlock extends BaseBlock {
  type: 'evidence';
  title?: string;
  items: string[];
  payload?: Record<string, unknown>;
}

export interface ErrorBlock extends BaseBlock {
  type: 'error';
  title?: string;
  content: string;
}

export interface RecommendationsBlock extends BaseBlock {
  type: 'recommendations';
  recommendations: EmbeddedRecommendation[];
}

export interface RawEvidenceBlock extends BaseBlock {
  type: 'raw_evidence';
  items: string[];
}

export interface FallbackBlock extends BaseBlock {
  type: 'fallback';
  content: string;
}

export type AssistantMessageBlock =
  | ThinkingBlock
  | MarkdownBlock
  | StatusBlock
  | PlanBlock
  | ToolExecutionBlock
  | ApprovalBlock
  | EvidenceBlock
  | ErrorBlock
  | RecommendationsBlock
  | RawEvidenceBlock
  | FallbackBlock;

export function normalizeAssistantMessage(input: AssistantMessageInput): AssistantMessageBlock[] {
  const blocks: AssistantMessageBlock[] = [];

  if (input.showThinking || input.thinking) {
    blocks.push({
      id: 'thinking',
      type: 'thinking',
      content: input.thinking || '',
      isStreaming: input.isStreaming && !input.content,
    });
  }

  if (input.content) {
    blocks.push({
      id: 'markdown',
      type: 'markdown',
      content: input.content,
    });
  }

  if (input.recommendations && input.recommendations.length > 0) {
    blocks.push({
      id: 'recommendations',
      type: 'recommendations',
      recommendations: input.recommendations,
    });
  }

  if (input.rawEvidence && input.rawEvidence.length > 0) {
    blocks.push({
      id: 'raw_evidence',
      type: 'raw_evidence',
      items: input.rawEvidence,
    });
  }

  return blocks;
}

export function mergeAssistantBlocks(
  primary: AssistantMessageBlock[],
  fallback: AssistantMessageBlock[],
): AssistantMessageBlock[] {
  if (primary.length === 0) {
    return fallback;
  }
  const merged = [...primary];
  const seenTypes = new Set(primary.map((block) => `${block.type}:${block.id}`));
  for (const block of fallback) {
    const typeKey = `${block.type}:${block.id}`;
    if (seenTypes.has(typeKey)) {
      continue;
    }
    if (block.type === 'markdown' && primary.some((item) => item.type === 'markdown')) {
      continue;
    }
    if (block.type === 'thinking' && primary.some((item) => item.type === 'thinking')) {
      continue;
    }
    if (block.type === 'recommendations' && primary.some((item) => item.type === 'recommendations')) {
      continue;
    }
    if (block.type === 'raw_evidence' && primary.some((item) => item.type === 'raw_evidence' || item.type === 'evidence')) {
      continue;
    }
    merged.push(block);
  }
  return merged;
}

export function normalizeTurnBlocks(turnBlocks: TurnBlock[] | undefined): AssistantMessageBlock[] {
  if (!turnBlocks || turnBlocks.length === 0) {
    return [];
  }

  type RenderableBlock = AssistantMessageBlock & {
    __renderOrder: number;
    __position: number;
  };

  return [...turnBlocks]
    .sort((a, b) => a.position - b.position)
    .map<RenderableBlock>((block) => {
      const renderOrder = blockRenderOrder(block.type);
      switch (block.type) {
        case 'text':
          return {
            id: block.id,
            type: 'markdown',
            content: block.content || '',
            streaming: block.streaming,
            __renderOrder: renderOrder,
            __position: block.position,
          };
        case 'thinking':
          return {
            id: block.id,
            type: 'thinking',
            content: block.content || '',
            isStreaming: block.streaming,
            __renderOrder: renderOrder,
            __position: block.position,
          };
        case 'status':
          return {
            id: block.id,
            type: 'status',
            title: block.title,
            content: block.content || stringifyBlockPayload(block.data),
            status: block.status,
            payload: block.data,
            __renderOrder: renderOrder,
            __position: block.position,
          };
        case 'plan':
          return {
            id: block.id,
            type: 'plan',
            title: block.title,
            content: block.content,
            payload: block.data,
            steps: extractPlanSteps(block.data),
            total: typeof block.data?.total === 'number' ? block.data.total : undefined,
            __renderOrder: renderOrder,
            __position: block.position,
          };
        case 'tool':
          return {
            id: block.id,
            type: 'tool',
            title: block.title,
            status: block.status,
            payload: block.data,
            __renderOrder: renderOrder,
            __position: block.position,
          };
        case 'approval':
          return {
            id: block.id,
            type: 'approval',
            title: block.title,
            payload: block.data,
            __renderOrder: renderOrder,
            __position: block.position,
          };
        case 'evidence':
          return {
            id: block.id,
            type: 'evidence',
            title: block.title,
            items: extractEvidenceItems(block),
            payload: block.data,
            __renderOrder: renderOrder,
            __position: block.position,
          };
        case 'error':
          return {
            id: block.id,
            type: 'error',
            title: block.title,
            content: block.content || stringifyBlockPayload(block.data),
            __renderOrder: renderOrder,
            __position: block.position,
          };
        case 'recommendations':
          return {
            id: block.id,
            type: 'recommendations',
            recommendations: ((block.data?.recommendations || block.data?.items || []) as EmbeddedRecommendation[]),
            __renderOrder: renderOrder,
            __position: block.position,
          };
        default:
          return {
            id: block.id,
            type: 'fallback',
            content: block.content || stringifyBlockPayload(block.data),
            __renderOrder: renderOrder,
            __position: block.position,
          };
      }
    })
    .sort((a, b) => (
      a.__renderOrder - b.__renderOrder
      || (a.__position - b.__position)
    ))
    .map((block) => {
      const next = { ...block };
      delete next.__renderOrder;
      delete next.__position;
      return next as AssistantMessageBlock;
    });
}

function stringifyBlockPayload(payload: Record<string, unknown> | undefined): string {
  if (!payload) {
    return '';
  }
  if (typeof payload.content_chunk === 'string') {
    return payload.content_chunk;
  }
  if (typeof payload.user_visible_summary === 'string') {
    return payload.user_visible_summary;
  }
  if (typeof payload.summary === 'string') {
    return payload.summary;
  }
  if (typeof payload.reason === 'string') {
    return payload.reason;
  }
  if (typeof payload.title === 'string') {
    return payload.title;
  }
  if (Array.isArray(payload.steps)) {
    return payload.steps
      .map((step, index) => `${index + 1}. ${String((step as Record<string, unknown>).content || (step as Record<string, unknown>).title || '步骤')}`)
      .join('\n');
  }
  try {
    return JSON.stringify(payload, null, 2);
  } catch {
    return String(payload);
  }
}

function extractPlanSteps(payload: Record<string, unknown> | undefined): PlanStep[] | undefined {
  if (!payload || !Array.isArray(payload.steps)) {
    return undefined;
  }
  const steps = payload.steps
    .filter((step): step is Record<string, unknown> => Boolean(step && typeof step === 'object'))
    .map((step) => ({
      id: typeof step.id === 'string' ? step.id : undefined,
      content: typeof step.content === 'string' ? step.content : undefined,
      title: typeof step.title === 'string' ? step.title : undefined,
      tool_hint: typeof step.tool_hint === 'string' ? step.tool_hint : undefined,
      status: typeof step.status === 'string' ? step.status : undefined,
      summary: typeof step.summary === 'string' ? step.summary : undefined,
    }));
  return steps.length > 0 ? steps : undefined;
}

function extractEvidenceItems(block: TurnBlock): string[] {
  const items = block.data?.items;
  if (Array.isArray(items)) {
    return items.map((item) => String(item));
  }
  if (block.content) {
    return [block.content];
  }
  const payload = stringifyBlockPayload(block.data);
  return payload ? [payload] : [];
}

function blockRenderOrder(type: TurnBlock['type']): number {
  switch (type) {
    case 'approval':
      return 10;
    case 'status':
    case 'plan':
      return 20;
    case 'tool':
      return 30;
    case 'evidence':
      return 40;
    case 'error':
      return 45;
    case 'text':
      return 50;
    case 'recommendations':
      return 60;
    case 'thinking':
      return 70;
    default:
      return 99;
  }
}
