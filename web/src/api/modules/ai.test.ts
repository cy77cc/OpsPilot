import { afterEach, describe, expect, it, vi } from 'vitest';
import { aiApi } from './ai';

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
        'event: stage_delta\ndata: {"stage":"rewrite","status":"loading","content_chunk":"正在理解"}\n\n',
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
    expect(onStageDelta).toHaveBeenCalledWith(expect.objectContaining({ stage: 'rewrite', status: 'loading', content_chunk: '正在理解' }));
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

  it('streams approval resume events on the dedicated resume endpoint', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: turn_started\ndata: {"turn_id":"turn-1","phase":"execute","status":"streaming"}\n\n',
        'event: block_delta\ndata: {"turn_id":"turn-1","block_id":"text:final","patch":{"content_chunk":"继续执行"}}\n\n',
        'event: done\ndata: {"stream_state":"ok"}\n\n',
      ]),
    } as Response);

    const onTurnStarted = vi.fn();
    const onBlockDelta = vi.fn();
    const onDone = vi.fn();

    await aiApi.respondApprovalStream(
      { session_id: 'sess-1', plan_id: 'plan-1', step_id: 'step-1', approved: true },
      { onTurnStarted, onBlockDelta, onDone },
    );

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/ai/resume/step/stream'),
      expect.objectContaining({ method: 'POST' }),
    );
    expect(onTurnStarted).toHaveBeenCalledWith(expect.objectContaining({ turn_id: 'turn-1', phase: 'execute' }));
    expect(onBlockDelta).toHaveBeenCalledWith(expect.objectContaining({ block_id: 'text:final' }));
    expect(onDone).toHaveBeenCalled();
  });
});
