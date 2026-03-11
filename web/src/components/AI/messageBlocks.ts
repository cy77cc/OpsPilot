import type { EmbeddedRecommendation } from './types';

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
