import { describe, expect, expectTypeOf, it } from 'vitest';
import { createEmptyAssistantRuntime } from './replyRuntime';

describe('assistant reply runtime shape', () => {
  it('creates an empty runtime with append-ready activity state', () => {
    expect(createEmptyAssistantRuntime()).toEqual({
      activities: [],
      phase: undefined,
      phaseLabel: undefined,
      summary: undefined,
      status: undefined,
    });
  });

  it('defines spec-required phase, activity, summary, and status fields', () => {
    const runtime = createEmptyAssistantRuntime();

    expect(runtime.activities).toEqual([]);
    expectTypeOf(runtime.phase).toEqualTypeOf<
      | 'preparing'
      | 'identifying'
      | 'planning'
      | 'executing'
      | 'summarizing'
      | 'completed'
      | 'interrupted'
      | undefined
    >();
    expectTypeOf(runtime.summary).toEqualTypeOf<
      | {
          title?: string;
          items?: Array<{
            label: string;
            value: string;
            tone?: 'default' | 'success' | 'warning' | 'danger';
          }>;
        }
      | undefined
    >();
    expectTypeOf(runtime.status).toEqualTypeOf<
      | {
          kind: 'streaming' | 'completed' | 'soft-timeout' | 'error' | 'interrupted';
          label: string;
        }
      | undefined
    >();
    expectTypeOf(runtime.activities).toEqualTypeOf<
      Array<{
        id: string;
        kind:
          | 'agent_handoff'
          | 'plan'
          | 'replan'
          | 'tool_call'
          | 'tool_approval'
          | 'tool_result'
          | 'hint'
          | 'error';
        label: string;
        detail?: string;
        status?: 'pending' | 'active' | 'done' | 'error';
        createdAt?: string;
      }>
    >();
  });
});
