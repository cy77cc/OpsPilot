import { describe, expect, it } from 'vitest';
import {
  applyBlockDelta,
  applyBlockOpen,
  applyBlockReplace,
  applyTurnStarted,
  getTurnBlocksForDisplay,
  projectTurnSummary,
} from './turnLifecycle';
import { normalizeTurnBlocks } from './messageBlocks';

describe('turnLifecycle', () => {
  it('projects block deltas into the same assistant turn summary', () => {
    let turn = applyTurnStarted(undefined, { turn_id: 'turn-1', phase: 'summary', status: 'streaming' });
    turn = applyBlockOpen(turn, {
      turn_id: 'turn-1',
      block_id: 'text:final',
      block_type: 'text',
      position: 1,
    });
    turn = applyBlockDelta(turn, {
      turn_id: 'turn-1',
      block_id: 'text:final',
      patch: { content_chunk: '扩容完成' },
    });

    expect(projectTurnSummary(turn)).toMatchObject({
      content: '扩容完成',
    });
  });

  it('hides thinking blocks and strips raw tool details in normal mode', () => {
    const turn = {
      id: 'turn-2',
      role: 'assistant' as const,
      status: 'streaming' as const,
      phase: 'execute',
      blocks: [
        {
          id: 'thinking-1',
          type: 'thinking' as const,
          position: 1,
          content: 'internal reasoning',
        },
        {
          id: 'tool-1',
          type: 'tool' as const,
          position: 2,
          data: {
            tool_name: 'kubectl_scale',
            params: { replicas: 3 },
            result: { ok: true, data: { replicas: 3 }, latency_ms: 1200 },
          },
        },
      ],
      createdAt: '2026-03-12T00:00:00Z',
      updatedAt: '2026-03-12T00:00:00Z',
    };

    const blocks = getTurnBlocksForDisplay(turn, 'normal', false);
    expect(blocks).toHaveLength(1);
    expect(blocks[0]).toMatchObject({
      id: 'tool-1',
      data: {
        tool_name: 'kubectl_scale',
        result: { ok: true, latency_ms: 1200 },
      },
    });
    expect((blocks[0].data as Record<string, unknown>).params).toBeUndefined();
  });

  it('disables streaming animation flags when reduced motion is enabled', () => {
    const turn = {
      id: 'turn-3',
      role: 'assistant' as const,
      status: 'streaming' as const,
      phase: 'summary',
      blocks: [
        {
          id: 'text-1',
          type: 'text' as const,
          position: 1,
          content: 'final answer',
          streaming: true,
        },
      ],
      createdAt: '2026-03-12T00:00:00Z',
      updatedAt: '2026-03-12T00:00:00Z',
    };

    const blocks = getTurnBlocksForDisplay(turn, 'normal', true);
    expect(blocks[0].streaming).toBe(false);
  });

  it('keeps approval blocks visible in normal mode', () => {
    let turn = applyTurnStarted(undefined, { turn_id: 'turn-4', phase: 'execute', status: 'waiting_user' });
    turn = applyBlockReplace(turn, {
      turn_id: 'turn-4',
      block_id: 'approval:step-1',
      block_type: 'approval',
      payload: {
        title: '等待你确认',
        summary: '请确认创建定时任务',
        risk: 'high',
        status: 'waiting_user',
      },
    });

    const blocks = getTurnBlocksForDisplay(turn, 'normal', false);
    expect(blocks).toHaveLength(1);
    expect(blocks[0]).toMatchObject({
      id: 'approval:step-1',
      type: 'approval',
      data: {
        title: '等待你确认',
        summary: '请确认创建定时任务',
      },
    });
  });

  it('orders approval, execution, and final answer blocks by conversation hierarchy', () => {
    const turn = {
      id: 'turn-5',
      role: 'assistant' as const,
      status: 'streaming' as const,
      phase: 'execute',
      blocks: [
        {
          id: 'text-1',
          type: 'text' as const,
          position: 3,
          content: '任务已创建',
        },
        {
          id: 'tool-1',
          type: 'tool' as const,
          position: 2,
          data: { tool_name: 'host_batch_exec_apply' },
        },
        {
          id: 'approval-1',
          type: 'approval' as const,
          position: 1,
          data: { title: '执行前确认' },
        },
      ],
      createdAt: '2026-03-12T00:00:00Z',
      updatedAt: '2026-03-12T00:00:00Z',
    };

    const blocks = normalizeTurnBlocks(getTurnBlocksForDisplay(turn, 'normal', false));
    expect(blocks.map((block) => block.type)).toEqual(['approval', 'tool', 'markdown']);
  });
});
