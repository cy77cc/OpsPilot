import { describe, expect, it } from 'vitest';
import { initialA2UIState, reduceA2UIState } from '../a2uiState';

describe('reduceA2UIState', () => {
  it('stores meta and handoff information', () => {
    let state = reduceA2UIState(initialA2UIState, {
      type: 'meta',
      payload: { session_id: 'sess-1', run_id: 'run-1', turn: 1 },
    });
    state = reduceA2UIState(state, {
      type: 'agent_handoff',
      payload: { from: 'OpsPilotAgent', to: 'DiagnosisAgent', intent: 'diagnosis' },
    });

    expect(state.sessionId).toBe('sess-1');
    expect(state.runId).toBe('run-1');
    expect(state.handoff?.to).toBe('DiagnosisAgent');
  });

  it('marks completed steps when replan arrives', () => {
    let state = reduceA2UIState(initialA2UIState, {
      type: 'plan',
      payload: { steps: ['inspect pods', 'check quota'], iteration: 0 },
    });

    state = reduceA2UIState(state, {
      type: 'replan',
      payload: { steps: ['check quota'], completed: 1, iteration: 1, is_final: false },
    });

    expect(state.planItems).toEqual([
      { content: 'inspect pods', status: 'done' },
      { content: 'check quota', status: 'active' },
    ]);
  });

  it('tracks tool lifecycle and approval state', () => {
    let state = reduceA2UIState(initialA2UIState, {
      type: 'tool_call',
      payload: { call_id: 'call-1', tool_name: 'host_exec', arguments: { host_id: 1 } },
    });

    state = reduceA2UIState(state, {
      type: 'tool_approval',
      payload: {
        approval_id: 'approval-1',
        call_id: 'call-1',
        tool_name: 'host_exec',
        preview: { command: 'uptime' },
        timeout_seconds: 300,
      },
    });

    state = reduceA2UIState(state, {
      type: 'tool_result',
      payload: { call_id: 'call-1', tool_name: 'host_exec', content: '{"stdout":"ok"}' },
    });

    expect(state.approval).toBeUndefined();
    expect(state.tools).toEqual([
      { callId: 'call-1', toolName: 'host_exec', status: 'done', content: '{"stdout":"ok"}' },
    ]);
  });

  it('accumulates content and terminal state', () => {
    let state = reduceA2UIState(initialA2UIState, {
      type: 'delta',
      payload: { content: 'hello ' },
    });
    state = reduceA2UIState(state, {
      type: 'delta',
      payload: { content: 'world' },
    });
    state = reduceA2UIState(state, {
      type: 'done',
      payload: { run_id: 'run-1', status: 'completed', iterations: 1 },
    });

    expect(state.content).toBe('hello world');
    expect(state.done).toBe(true);
  });
});
