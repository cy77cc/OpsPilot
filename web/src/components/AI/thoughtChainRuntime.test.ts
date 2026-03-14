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
        headline: '准备检查集群状态',
        body: '先确认集群可用性，再检查 deployment 副本数。',
        structured: {
          steps: ['检查集群状态'],
        },
      },
    });
    state = reduceThoughtChainRuntimeEvent(state, {
      type: 'chain_node_patch',
      data: {
        turn_id: 'turn-1',
        node_id: 'plan-1',
        headline: '已整理出 2 个执行步骤',
        structured: {
          steps: ['检查集群状态', '确认 deployment 副本数'],
        },
        raw: {
          source: 'planner',
        },
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
        headline: '已整理出 2 个执行步骤',
        body: '先确认集群可用性，再检查 deployment 副本数。',
        structured: {
          steps: ['检查集群状态', '确认 deployment 副本数'],
        },
        raw: {
          source: 'planner',
        },
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

  it('stores separated narrative and raw fields on tool nodes', () => {
    const state = reduceThoughtChainRuntimeEvent(undefined, {
      type: 'chain_node_open',
      data: {
        turn_id: 'turn-1',
        node_id: 'tool-1',
        kind: 'tool',
        title: 'host_list_inventory',
        status: 'loading',
        headline: '已获取 5 台主机',
        body: '当前所有主机均在线。',
        structured: {
          resource: 'hosts',
          rows: [{ id: 1, name: 'test', status: 'online' }],
        },
        raw: {
          total: 5,
        },
      },
    });

    expect(state.nodes[0]).toEqual(expect.objectContaining({
      nodeId: 'tool-1',
      headline: '已获取 5 台主机',
      body: '当前所有主机均在线。',
      structured: expect.objectContaining({
        resource: 'hosts',
      }),
      raw: expect.objectContaining({
        total: 5,
      }),
    }));
  });
});
