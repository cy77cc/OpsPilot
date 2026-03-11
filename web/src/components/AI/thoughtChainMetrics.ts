import type { SSEEventType, ThoughtStageItem, ThoughtStageKey } from './types';

export interface ThoughtChainEventRecord {
  type: SSEEventType;
  data?: Record<string, unknown>;
}

export interface ThoughtChainEventCompleteness {
  expectedKeys: ThoughtStageKey[];
  deliveredKeys: ThoughtStageKey[];
  missingKeys: ThoughtStageKey[];
  completenessRate: number;
}

export interface ThoughtChainRenderConsistency {
  expectedKeys: ThoughtStageKey[];
  renderedKeys: ThoughtStageKey[];
  missingKeys: ThoughtStageKey[];
  consistencyRate: number;
}

export function expectedThoughtChainStages(events: ThoughtChainEventRecord[]): ThoughtStageKey[] {
  const expected = new Set<ThoughtStageKey>();
  for (const event of events) {
    switch (event.type) {
      case 'rewrite_result':
        expected.add('rewrite');
        break;
      case 'planner_state':
      case 'plan_created':
      case 'replan_started':
        expected.add('plan');
        break;
      case 'step_update':
      case 'tool_call':
      case 'tool_result':
        expected.add('execute');
        break;
      case 'approval_required':
      case 'clarify_required':
        expected.add('user_action');
        break;
      default:
        break;
    }
  }
  return Array.from(expected);
}

export function deliveredThoughtChainStages(events: ThoughtChainEventRecord[]): ThoughtStageKey[] {
  const delivered = new Set<ThoughtStageKey>();
  for (const event of events) {
    if (event.type === 'stage_delta') {
      const stage = String(event.data?.stage || '').trim() as ThoughtStageKey;
      if (stage && stage !== 'summary') {
        delivered.add(stage);
      }
      continue;
    }
    if (event.type === 'approval_required' || event.type === 'clarify_required') {
      delivered.add('user_action');
    }
  }
  return Array.from(delivered);
}

export function computeThoughtChainEventCompleteness(events: ThoughtChainEventRecord[]): ThoughtChainEventCompleteness {
  const expectedKeys = expectedThoughtChainStages(events);
  const deliveredKeys = deliveredThoughtChainStages(events);
  const deliveredSet = new Set(deliveredKeys);
  const missingKeys = expectedKeys.filter((key) => !deliveredSet.has(key));
  return {
    expectedKeys,
    deliveredKeys,
    missingKeys,
    completenessRate: ratio(expectedKeys.length - missingKeys.length, expectedKeys.length),
  };
}

export function computeThoughtChainRenderConsistency(
  events: ThoughtChainEventRecord[],
  thoughtChain: ThoughtStageItem[] | undefined
): ThoughtChainRenderConsistency {
  const expectedKeys = expectedThoughtChainStages(events);
  const renderedKeys = Array.from(new Set((thoughtChain || []).map((item) => item.key).filter((key) => key !== 'summary')));
  const renderedSet = new Set(renderedKeys);
  const missingKeys = expectedKeys.filter((key) => !renderedSet.has(key));
  return {
    expectedKeys,
    renderedKeys,
    missingKeys,
    consistencyRate: ratio(expectedKeys.length - missingKeys.length, expectedKeys.length),
  };
}

function ratio(numerator: number, denominator: number): number {
  if (denominator <= 0) {
    return 0;
  }
  return numerator / denominator;
}
