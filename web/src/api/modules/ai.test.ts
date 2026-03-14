import { afterEach, describe, expect, it, vi } from 'vitest';
import { aiApi } from './ai';
import apiService from '../api';

function buildStream(chunks: string[]) {
  const encoder = new TextEncoder();
  return new ReadableStream<Uint8Array>({
    start(controller) {
      chunks.forEach((chunk) => controller.enqueue(encoder.encode(chunk)));
      controller.close();
    },
  });
}

describe('aiApi.chatStream', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it('maps legacy message events to onDelta content', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: meta\ndata: {"sessionId":"sess-1"}\n\n',
        'event: message\ndata: {"content":"hello from backend"}\n\n',
        'event: done\ndata: {"stream_state":"ok"}\n\n',
      ]),
    } as Response);

    const onMeta = vi.fn();
    const onDelta = vi.fn();
    const onDone = vi.fn();

    await aiApi.chatStream(
      { message: 'hi', context: { scene: 'global' } },
      { onMeta, onDelta, onDone }
    );

    expect(fetchMock).toHaveBeenCalled();
    expect(onMeta).toHaveBeenCalledWith(expect.objectContaining({ sessionId: 'sess-1' }));
    expect(onDelta).toHaveBeenCalledWith(expect.objectContaining({ contentChunk: 'hello from backend' }));
    expect(onDone).toHaveBeenCalled();
  });

  it('dispatches high-level orchestration events', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: rewrite_result\ndata: {"user_visible_summary":"rewrite ok"}\n\n',
        'event: stage_delta\ndata: {"stage":"plan","status":"loading","title":"整理执行步骤","description":"正在根据你的需求整理执行步骤","steps":["检查告警","查看副本数"],"content_chunk":"正在理解"}\n\n',
        'event: plan_created\ndata: {"user_visible_summary":"plan ok"}\n\n',
        'event: step_update\ndata: {"step_id":"step-1","status":"running","user_visible_summary":"executing"}\n\n',
        'event: thinking_delta\ndata: {"contentChunk":"thinking"}\n\n',
        'event: summary\ndata: {"status":"success"}\n\n',
      ]),
    } as Response);

    const onRewriteResult = vi.fn();
    const onStageDelta = vi.fn();
    const onPlanCreated = vi.fn();
    const onStepUpdate = vi.fn();
    const onThinkingDelta = vi.fn();
    const onSummary = vi.fn();

    await aiApi.chatStream(
      { message: 'hi', context: { scene: 'global' } },
      { onRewriteResult, onStageDelta, onPlanCreated, onStepUpdate, onThinkingDelta, onSummary }
    );

    expect(onRewriteResult).toHaveBeenCalledWith(expect.objectContaining({ user_visible_summary: 'rewrite ok' }));
    expect(onStageDelta).toHaveBeenCalledWith(expect.objectContaining({
      stage: 'plan',
      status: 'loading',
      title: '整理执行步骤',
      description: '正在根据你的需求整理执行步骤',
      steps: ['检查告警', '查看副本数'],
      content_chunk: '正在理解',
    }));
    expect(onPlanCreated).toHaveBeenCalledWith(expect.objectContaining({ user_visible_summary: 'plan ok' }));
    expect(onStepUpdate).toHaveBeenCalledWith(expect.objectContaining({ step_id: 'step-1', status: 'running' }));
    expect(onThinkingDelta).toHaveBeenCalledWith(expect.objectContaining({ contentChunk: 'thinking' }));
    expect(onSummary).toHaveBeenCalledWith(expect.objectContaining({ status: 'success' }));
  });

  it('dispatches native turn and block lifecycle events', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: turn_started\ndata: {"turn_id":"turn-1","phase":"rewrite","status":"streaming"}\n\n',
        'event: block_open\ndata: {"turn_id":"turn-1","block_id":"status:rewrite","block_type":"status","position":1}\n\n',
        'event: block_delta\ndata: {"turn_id":"turn-1","block_id":"status:rewrite","patch":{"content_chunk":"理解问题"}}\n\n',
        'event: block_replace\ndata: {"turn_id":"turn-1","block_id":"plan:main","payload":{"summary":"已生成计划"}}\n\n',
        'event: block_close\ndata: {"turn_id":"turn-1","block_id":"status:rewrite","status":"success"}\n\n',
        'event: turn_state\ndata: {"turn_id":"turn-1","phase":"execute","status":"streaming"}\n\n',
        'event: turn_done\ndata: {"turn_id":"turn-1","phase":"done","status":"completed"}\n\n',
      ]),
    } as Response);

    const onTurnStarted = vi.fn();
    const onBlockOpen = vi.fn();
    const onBlockDelta = vi.fn();
    const onBlockReplace = vi.fn();
    const onBlockClose = vi.fn();
    const onTurnState = vi.fn();
    const onTurnDone = vi.fn();

    await aiApi.chatStream(
      { message: 'hi', context: { scene: 'global' } },
      { onTurnStarted, onBlockOpen, onBlockDelta, onBlockReplace, onBlockClose, onTurnState, onTurnDone }
    );

    expect(onTurnStarted).toHaveBeenCalledWith(expect.objectContaining({ turn_id: 'turn-1', phase: 'rewrite' }));
    expect(onBlockOpen).toHaveBeenCalledWith(expect.objectContaining({ block_id: 'status:rewrite', block_type: 'status' }));
    expect(onBlockDelta).toHaveBeenCalledWith(expect.objectContaining({ block_id: 'status:rewrite' }));
    expect(onBlockReplace).toHaveBeenCalledWith(expect.objectContaining({ block_id: 'plan:main' }));
    expect(onBlockClose).toHaveBeenCalledWith(expect.objectContaining({ block_id: 'status:rewrite', status: 'success' }));
    expect(onTurnState).toHaveBeenCalledWith(expect.objectContaining({ turn_id: 'turn-1', phase: 'execute' }));
    expect(onTurnDone).toHaveBeenCalledWith(expect.objectContaining({ turn_id: 'turn-1', status: 'completed' }));
  });

  it('does not dispatch legacy phase lifecycle events on the primary stream path', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: phase_started\ndata: {"phase":"planning","status":"loading","title":"整理执行步骤"}\n\n',
        'event: chain_started\ndata: {"turn_id":"turn-1"}\n\n',
      ]),
    } as Response);

    const onPhaseStarted = vi.fn();
    const onChainStarted = vi.fn();

    await aiApi.chatStream(
      { message: 'hi', context: { scene: 'global' } },
      { onPhaseStarted, onChainStarted } as any,
    );

    expect(onChainStarted).toHaveBeenCalledWith(expect.objectContaining({ turn_id: 'turn-1' }));
    expect(onPhaseStarted).not.toHaveBeenCalled();
  });

  it('preserves stage-aware error payloads', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: error\ndata: {"message":"AI 规划模块当前不可用，请稍后重试或手动在页面中执行操作。","error_code":"planner_runner_unavailable","stage":"plan","recoverable":true}\n\n',
      ]),
    } as Response);

    const onError = vi.fn();

    await aiApi.chatStream(
      { message: 'hi', context: { scene: 'global' } },
      { onError }
    );

    expect(onError).toHaveBeenCalledWith(expect.objectContaining({
      message: 'AI 规划模块当前不可用，请稍后重试或手动在页面中执行操作。',
      code: 'planner_runner_unavailable',
      stage: 'plan',
      recoverable: true,
    }));
  });

  it('posts unified chain approval decisions to the canonical endpoint', async () => {
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({
      success: true,
      data: { approval: { id: 'approval-1', status: 'approved' } },
    });

    await aiApi.decideChainApproval('plan-1', 'approval:step-1', true, 'looks safe');

    expect(postMock).toHaveBeenCalledWith(
      '/ai/chains/plan-1/approvals/approval:step-1/decision',
      { approved: true, reason: 'looks safe' },
    );
  });

  it('streams unified chain approval decisions from the canonical endpoint', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: chain_started\ndata: {"turn_id":"turn-approval"}\n\n',
        'event: chain_node_open\ndata: {"node_id":"tool:step-1","kind":"tool","title":"执行审批后的步骤"}\n\n',
        'event: final_answer_delta\ndata: {"turn_id":"turn-approval","chunk":"已继续执行"}\n\n',
      ]),
    } as Response);

    const onChainStarted = vi.fn();
    const onChainNodeOpen = vi.fn();
    const onFinalAnswerDelta = vi.fn();

    await aiApi.decideChainApprovalStream(
      'plan-1',
      'approval:step-1',
      true,
      { onChainStarted, onChainNodeOpen, onFinalAnswerDelta },
      'looks safe',
    );

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/ai/chains/plan-1/approvals/approval:step-1/decision'),
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
          Accept: 'text/event-stream',
        }),
        body: JSON.stringify({ approved: true, reason: 'looks safe' }),
      }),
    );
    expect(onChainStarted).toHaveBeenCalledWith(expect.objectContaining({ turn_id: 'turn-approval' }));
    expect(onChainNodeOpen).toHaveBeenCalledWith(expect.objectContaining({ node_id: 'tool:step-1', kind: 'tool' }));
    expect(onFinalAnswerDelta).toHaveBeenCalledWith(expect.objectContaining({ chunk: '已继续执行' }));
  });
});
