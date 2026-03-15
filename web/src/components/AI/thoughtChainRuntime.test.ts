import { describe, expect, it } from 'vitest';
import {
  createToolChainState,
  reduceToolChainEvent,
  toThoughtChainRuntimeState,
  toolChainStateFromReplayTurn,
} from './thoughtChainRuntime';
import type { ChatTurn } from './types';

describe('thoughtChainRuntime', () => {
  it('handles tool_call event and creates a running tool node', () => {
    const state = reduceToolChainEvent(undefined, {
      type: 'tool_call',
      data: {
        call_id: 'call-1',
        tool_name: 'get_pods',
        tool_display_name: '获取 Pod 列表',
        arguments: '{"namespace":"default"}',
      },
    });

    expect(state.nodes).toHaveLength(1);
    expect(state.nodes[0]).toEqual(expect.objectContaining({
      id: 'call-1',
      kind: 'tool',
      toolName: 'get_pods',
      toolDisplayName: '获取 Pod 列表',
      status: 'running',
    }));
    expect(state.activeNodeId).toBe('call-1');
  });

  it('handles tool_approval event and creates a waiting approval node', () => {
    const state = reduceToolChainEvent(undefined, {
      type: 'tool_approval',
      data: {
        call_id: 'call-2',
        tool_name: 'delete_pod',
        tool_display_name: '删除 Pod',
        risk: 'high',
        summary: '即将删除 pod/nginx-deployment-xxx',
        arguments_json: '{"name":"nginx-deployment-xxx"}',
        approval_id: 'approval-1',
        checkpoint_id: 'cp-1',
        plan_id: 'plan-1',
        step_id: 'step-1',
      },
    });

    expect(state.nodes).toHaveLength(1);
    expect(state.nodes[0]).toEqual(expect.objectContaining({
      id: 'call-2',
      kind: 'approval',
      toolName: 'delete_pod',
      toolDisplayName: '删除 Pod',
      status: 'waiting_approval',
      approval: expect.objectContaining({
        id: 'approval-1',
        title: '删除 Pod',
        risk: 'high',
        description: '即将删除 pod/nginx-deployment-xxx',
        status: 'waiting_user',
      }),
    }));
    expect(state.activeNodeId).toBe('call-2');
  });

  it('handles tool_result event and updates node status', () => {
    let state = reduceToolChainEvent(undefined, {
      type: 'tool_call',
      data: {
        call_id: 'call-3',
        tool_name: 'get_pods',
      },
    });

    state = reduceToolChainEvent(state, {
      type: 'tool_result',
      data: {
        call_id: 'call-3',
        tool_name: 'get_pods',
        result: '{"items":[],"ok":true}',
      },
    });

    expect(state.nodes).toHaveLength(1);
    expect(state.nodes[0].status).toBe('success');
    expect(state.nodes[0].result).toEqual(expect.objectContaining({
      ok: true,
      data: expect.objectContaining({ items: [] }),
    }));
    expect(state.activeNodeId).toBeUndefined();
  });

  it('handles tool_result with error status', () => {
    let state = reduceToolChainEvent(undefined, {
      type: 'tool_call',
      data: {
        call_id: 'call-4',
        tool_name: 'get_pods',
      },
    });

    state = reduceToolChainEvent(state, {
      type: 'tool_result',
      data: {
        call_id: 'call-4',
        result: '{"error":"connection refused"}',
      },
    });

    expect(state.nodes[0].status).toBe('error');
    expect(state.nodes[0].result?.error).toBeTruthy();
  });

  it('updates existing node when receiving tool_call for same call_id', () => {
    let state = reduceToolChainEvent(undefined, {
      type: 'tool_call',
      data: {
        call_id: 'call-5',
        tool_name: 'get_pods',
      },
    });

    expect(state.nodes[0].status).toBe('running');

    state = reduceToolChainEvent(state, {
      type: 'tool_call',
      data: {
        call_id: 'call-5',
        tool_name: 'get_pods',
      },
    });

    expect(state.nodes).toHaveLength(1);
    expect(state.activeNodeId).toBe('call-5');
  });

  it('builds tool chain from replay turn with tool blocks', () => {
    const turn: ChatTurn = {
      id: 'turn-1',
      role: 'assistant',
      status: 'completed',
      blocks: [
        {
          id: 'tool-1',
          type: 'tool',
          title: 'get_pods',
          position: 0,
          status: 'success',
          data: {
            params: { namespace: 'default' },
            result: { ok: true, data: { items: [] } },
          },
        },
        {
          id: 'approval-1',
          type: 'approval',
          title: '删除 Pod',
          position: 1,
          status: 'waiting_approval',
          data: {
            tool_name: 'delete_pod',
            tool_display_name: '删除 Pod',
            risk: 'high',
            summary: '即将删除 pod',
            plan_id: 'plan-1',
            step_id: 'step-1',
            checkpoint_id: 'cp-1',
            arguments_json: '{"name":"nginx"}',
          },
        },
      ],
      createdAt: '2026-03-15T00:00:00Z',
      updatedAt: '2026-03-15T00:00:00Z',
    };

    const state = toolChainStateFromReplayTurn(turn);

    expect(state).not.toBeUndefined();
    expect(state!.nodes).toHaveLength(2);
    expect(state!.nodes[0]).toEqual(expect.objectContaining({
      id: 'tool-1',
      kind: 'tool',
      toolName: 'get_pods',
      status: 'success',
    }));
    expect(state!.nodes[1]).toEqual(expect.objectContaining({
      id: 'approval-1',
      kind: 'approval',
      toolName: 'delete_pod',
      status: 'waiting_approval',
      approval: expect.objectContaining({
        risk: 'high',
        title: '删除 Pod',
      }),
    }));
  });

  it('returns undefined for user replay turns', () => {
    const turn: ChatTurn = {
      id: 'turn-user',
      role: 'user',
      status: 'completed',
      blocks: [],
      createdAt: '2026-03-15T00:00:00Z',
      updatedAt: '2026-03-15T00:00:00Z',
    };

    const state = toolChainStateFromReplayTurn(turn);

    expect(state).toBeUndefined();
  });

  it('returns undefined for assistant turns with no tool blocks', () => {
    const turn: ChatTurn = {
      id: 'turn-simple',
      role: 'assistant',
      status: 'completed',
      blocks: [
        {
          id: 'text-1',
          type: 'text',
          position: 0,
          content: 'Hello!',
        },
      ],
      createdAt: '2026-03-15T00:00:00Z',
      updatedAt: '2026-03-15T00:00:00Z',
    };

    const state = toolChainStateFromReplayTurn(turn);

    expect(state).toBeUndefined();
  });

  it('ignores unknown event types', () => {
    const state = reduceToolChainEvent(undefined, {
      type: 'unknown_event' as any,
      data: {},
    });

    expect(state.nodes).toHaveLength(0);
  });

  it('maps running tool status to active in runtime compatibility state', () => {
    const state = reduceToolChainEvent(undefined, {
      type: 'tool_call',
      data: {
        call_id: 'call-runtime-1',
        tool_name: 'get_nodes',
      },
    });

    const runtime = toThoughtChainRuntimeState(state);

    expect(runtime).toBeDefined();
    expect(runtime?.nodes).toHaveLength(1);
    expect(runtime?.nodes[0]?.status).toBe('active');
  });
});
