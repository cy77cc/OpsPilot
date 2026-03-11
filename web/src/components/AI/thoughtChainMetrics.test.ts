import { describe, expect, it } from 'vitest';
import { computeThoughtChainEventCompleteness, computeThoughtChainRenderConsistency } from './thoughtChainMetrics';
import type { ThoughtStageItem } from './types';

describe('thoughtChainMetrics', () => {
  it('computes event completeness from stage deltas', () => {
    const result = computeThoughtChainEventCompleteness([
      { type: 'rewrite_result' },
      { type: 'stage_delta', data: { stage: 'rewrite' } },
      { type: 'planner_state' },
      { type: 'stage_delta', data: { stage: 'plan' } },
      { type: 'step_update' },
      { type: 'summary' },
      { type: 'stage_delta', data: { stage: 'summary' } },
    ]);

    expect(result.expectedKeys).toEqual(['rewrite', 'plan', 'execute']);
    expect(result.deliveredKeys).toEqual(['rewrite', 'plan']);
    expect(result.missingKeys).toEqual(['execute']);
    expect(result.completenessRate).toBeCloseTo(2 / 3);
  });

  it('treats clarify as a delivered user action stage', () => {
    const result = computeThoughtChainEventCompleteness([
      { type: 'rewrite_result' },
      { type: 'stage_delta', data: { stage: 'rewrite' } },
      { type: 'planner_state' },
      { type: 'stage_delta', data: { stage: 'plan' } },
      { type: 'clarify_required', data: { message: 'need more info' } },
    ]);

    expect(result.expectedKeys).toEqual(['rewrite', 'plan', 'user_action']);
    expect(result.deliveredKeys).toEqual(['rewrite', 'plan', 'user_action']);
    expect(result.missingKeys).toEqual([]);
    expect(result.completenessRate).toBe(1);
  });

  it('computes render consistency against rendered thought chain items', () => {
    const rendered: ThoughtStageItem[] = [
      { key: 'rewrite', title: '理解你的问题', status: 'success' },
      { key: 'plan', title: '整理排查计划', status: 'success' },
      { key: 'summary', title: '生成结论', status: 'success' },
    ];
    const result = computeThoughtChainRenderConsistency([
      { type: 'rewrite_result' },
      { type: 'planner_state' },
      { type: 'step_update' },
      { type: 'summary' },
    ], rendered);

    expect(result.expectedKeys).toEqual(['rewrite', 'plan', 'execute']);
    expect(result.renderedKeys).toEqual(['rewrite', 'plan']);
    expect(result.missingKeys).toEqual(['execute']);
    expect(result.consistencyRate).toBeCloseTo(2 / 3);
  });
});
