import type { EmbeddedRecommendation } from './types';

export interface AssistantMessageInput {
  content?: string;
  thinking?: string;
  summaryOutput?: Record<string, unknown>;
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

export interface SummaryOutputBlock extends BaseBlock {
  type: 'summary_output';
  headline?: string;
  conclusion?: string;
  narrative?: string;
  keyFindings?: string[];
  resourceSummaries?: string[];
  recommendations?: string[];
  nextActions?: string[];
}

export interface FallbackBlock extends BaseBlock {
  type: 'fallback';
  content: string;
}

export type AssistantMessageBlock =
  | ThinkingBlock
  | MarkdownBlock
  | RecommendationsBlock
  | SummaryOutputBlock
  | RawEvidenceBlock
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

  const summaryBlock = normalizeSummaryOutput(input.summaryOutput);
  if (summaryBlock) {
    blocks.push(summaryBlock);
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

function normalizeSummaryOutput(summaryOutput?: Record<string, unknown>): SummaryOutputBlock | null {
  if (!summaryOutput || Object.keys(summaryOutput).length === 0) {
    return null;
  }
  const headline = asString(summaryOutput.headline);
  const conclusion = asString(summaryOutput.conclusion);
  const narrative = asString(summaryOutput.narrative);
  const keyFindings = uniqueStrings(summaryOutput.key_findings);
  const resourceSummaries = uniqueStrings(summaryOutput.resource_summaries);
  const recommendations = uniqueStrings(summaryOutput.recommendations);
  const nextActions = uniqueStrings(summaryOutput.next_actions);

  if (!headline && !conclusion && !narrative && keyFindings.length === 0 && resourceSummaries.length === 0 && recommendations.length === 0 && nextActions.length === 0) {
    return null;
  }

  return {
    id: 'summary_output',
    type: 'summary_output',
    headline,
    conclusion,
    narrative,
    keyFindings,
    resourceSummaries,
    recommendations,
    nextActions,
  };
}

function asString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined;
}

function uniqueStrings(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  const seen = new Set<string>();
  const out: string[] = [];
  value.forEach((item) => {
    if (typeof item !== 'string') {
      return;
    }
    const text = item.trim();
    if (!text || seen.has(text)) {
      return;
    }
    seen.add(text);
    out.push(text);
  });
  return out;
}
