import { describe, expect, it } from 'vitest';
import {
  createThoughtChainRuntimeState,
  reduceThoughtChainRuntimeEvent,
} from './thoughtChainRuntime';

describe('thoughtChainRuntime', () => {
  it('ignores legacy phase events for the primary chain reducer', () => {
    const legacyPhaseEvent = {
      type: 'phase_started',
      data: { phase: 'planning', status: 'loading' },
    } as unknown as Parameters<typeof reduceThoughtChainRuntimeEvent>[1];

    const state = reduceThoughtChainRuntimeEvent(undefined, legacyPhaseEvent);

    expect(state.nodes).toHaveLength(0);
    expect(state.turnId).toBeUndefined();
  });

  it('opens, patches, closes, and collapses native chain nodes before final answer becomes visible', () => {
    let state = createThoughtChainRuntimeState();

    state = reduceThoughtChainRuntimeEvent(state, {
      type: 'chain_started',
      data: { turn_id: 'turn-1' },
    });
    state = reduceThoughtChainRuntimeEvent(state, {
      type: 'chain_node_open',
      data: {
        turn_id: 'turn-1',
        node_id: 'plan-1',
        kind: 'plan',
        title: '正在整理执行计划',
        status: 'loading',
        summary: '准备检查集群状态',
      },
    });
    state = reduceThoughtChainRuntimeEvent(state, {
      type: 'chain_node_patch',
      data: {
        turn_id: 'turn-1',
        node_id: 'plan-1',
        summary: '已整理出 2 个执行步骤',
        details: ['检查集群状态', '确认 deployment 副本数'],
      },
    });
    state = reduceThoughtChainRuntimeEvent(state, {
      type: 'chain_node_close',
      data: {
        turn_id: 'turn-1',
        node_id: 'plan-1',
        status: 'done',
      },
    });
    state = reduceThoughtChainRuntimeEvent(state, {
      type: 'chain_collapsed',
      data: { turn_id: 'turn-1' },
    });
    state = reduceThoughtChainRuntimeEvent(state, {
      type: 'final_answer_started',
      data: { turn_id: 'turn-1' },
    });
    state = reduceThoughtChainRuntimeEvent(state, {
      type: 'final_answer_delta',
      data: { turn_id: 'turn-1', chunk: '扩容已完成' },
    });
    state = reduceThoughtChainRuntimeEvent(state, {
      type: 'final_answer_done',
      data: { turn_id: 'turn-1' },
    });

    expect(state.turnId).toBe('turn-1');
    expect(state.nodes).toEqual([
      expect.objectContaining({
        nodeId: 'plan-1',
        kind: 'plan',
        status: 'done',
        summary: '已整理出 2 个执行步骤',
        details: ['检查集群状态', '确认 deployment 副本数'],
      }),
    ]);
    expect(state.isCollapsed).toBe(true);
    expect(state.collapsePhase).toBe('collapsed');
    expect(state.finalAnswer).toEqual(expect.objectContaining({
      visible: true,
      streaming: false,
      content: '扩容已完成',
    }));
  });

  it('maps approval payload into the active approval node', () => {
    let state = createThoughtChainRuntimeState();

    state = reduceThoughtChainRuntimeEvent(state, {
      type: 'chain_node_open',
      data: {
        node_id: 'approval-1',
        kind: 'approval',
        title: '扩容 nginx 需要确认',
        status: 'waiting',
        summary: '该步骤会修改工作负载副本数',
        approval: {
          request_id: 'approval-1',
          title: '扩容 nginx 需要确认',
          risk: 'medium',
          details: {
            step_id: 'step-1',
          },
        },
      },
    });

    expect(state.activeNodeId).toBe('approval-1');
    expect(state.nodes[0]).toEqual(expect.objectContaining({
      nodeId: 'approval-1',
      kind: 'approval',
      approval: expect.objectContaining({
        requestId: 'approval-1',
        title: '扩容 nginx 需要确认',
        risk: 'medium',
      }),
    }));
  });
});
