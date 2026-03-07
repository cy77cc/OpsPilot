import type { EmbeddedRecommendation } from './types';

export interface AssistantMessageInput {
  content?: string;
  thinking?: string;
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
}

export interface RecommendationsBlock extends BaseBlock {
  type: 'recommendations';
  recommendations: EmbeddedRecommendation[];
}

export interface FallbackBlock extends BaseBlock {
  type: 'fallback';
  content: string;
}

export type AssistantMessageBlock =
  | ThinkingBlock
  | MarkdownBlock
  | RecommendationsBlock
  | FallbackBlock;

export function normalizeAssistantMessage(input: AssistantMessageInput): AssistantMessageBlock[] {
  const blocks: AssistantMessageBlock[] = [];

  if (input.thinking) {
    blocks.push({
      id: 'thinking',
      type: 'thinking',
      content: input.thinking,
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

  return blocks;
}
