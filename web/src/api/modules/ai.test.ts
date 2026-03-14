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

  it('dispatches phase lifecycle events for planning and replanning', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: phase_started\ndata: {"phase":"planning","status":"loading","title":"整理执行步骤"}\n\n',
        'event: phase_complete\ndata: {"phase":"planning","status":"success"}\n\n',
        'event: phase_started\ndata: {"phase":"replanning","status":"loading","title":"动态调整计划"}\n\n',
      ]),
    } as Response);

    const onPhaseStarted = vi.fn();
    const onPhaseComplete = vi.fn();

    await aiApi.chatStream(
      { message: 'hi', context: { scene: 'global' } },
      { onPhaseStarted, onPhaseComplete } as any,
    );

    expect(onPhaseStarted).toHaveBeenNthCalledWith(1, expect.objectContaining({
      phase: 'planning',
      status: 'loading',
      title: '整理执行步骤',
    }));
    expect(onPhaseComplete).toHaveBeenCalledWith(expect.objectContaining({
      phase: 'planning',
      status: 'success',
    }));
    expect(onPhaseStarted).toHaveBeenNthCalledWith(2, expect.objectContaining({
      phase: 'replanning',
      status: 'loading',
      title: '动态调整计划',
    }));
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

  it('dispatches structured plan generation events with steps and total', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: plan_generated\ndata: {"plan_id":"plan-42","steps":[{"id":"step-1","content":"检查集群状态","tool_hint":"get_cluster_info"},{"id":"step-2","content":"获取 deployment 列表","tool_hint":"list_deployments"}],"total":2}\n\n',
      ]),
    } as Response);

    const onPlanGenerated = vi.fn();

    await aiApi.chatStream(
      { message: 'hi', context: { scene: 'global' } },
      { onPlanGenerated } as any,
    );

    expect(onPlanGenerated).toHaveBeenCalledWith(expect.objectContaining({
      plan_id: 'plan-42',
      total: 2,
      steps: [
        expect.objectContaining({ id: 'step-1', content: '检查集群状态', tool_hint: 'get_cluster_info' }),
        expect.objectContaining({ id: 'step-2', content: '获取 deployment 列表', tool_hint: 'list_deployments' }),
      ],
    }));
  });

  it('dispatches step lifecycle and replan events during execution pivots', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: step_started\ndata: {"step_id":"step-1","title":"检查集群状态","tool_name":"get_cluster_info","status":"running"}\n\n',
        'event: step_complete\ndata: {"step_id":"step-1","status":"success","summary":"集群状态正常"}\n\n',
        'event: replan_triggered\ndata: {"reason":"步骤执行失败，需要调整计划","completed_steps":1}\n\n',
      ]),
    } as Response);

    const onStepStarted = vi.fn();
    const onStepComplete = vi.fn();
    const onReplanTriggered = vi.fn();

    await aiApi.chatStream(
      { message: 'hi', context: { scene: 'global' } },
      { onStepStarted, onStepComplete, onReplanTriggered } as any,
    );

    expect(onStepStarted).toHaveBeenCalledWith(expect.objectContaining({
      step_id: 'step-1',
      title: '检查集群状态',
      tool_name: 'get_cluster_info',
      status: 'running',
    }));
    expect(onStepComplete).toHaveBeenCalledWith(expect.objectContaining({
      step_id: 'step-1',
      status: 'success',
      summary: '集群状态正常',
    }));
    expect(onReplanTriggered).toHaveBeenCalledWith(expect.objectContaining({
      reason: '步骤执行失败，需要调整计划',
      completed_steps: 1,
    }));
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
        'event: stage_delta\ndata: {"stage":"execute","status":"loading","summary":"继续执行审批后的步骤"}\n\n',
        'event: step_update\ndata: {"plan_id":"plan-1","step_id":"step-1","tool":"scale_deployment","status":"success","user_visible_summary":"扩容完成"}\n\n',
        'event: delta\ndata: {"content":"继续执行"}\n\n',
        'event: done\ndata: {"stream_state":"ok"}\n\n',
      ]),
    } as Response);

    const onStageDelta = vi.fn();
    const onStepUpdate = vi.fn();
    const onDelta = vi.fn();
    const onDone = vi.fn();

    await aiApi.respondApprovalStream(
      { session_id: 'sess-1', plan_id: 'plan-1', step_id: 'step-1', checkpoint_id: 'cp-1', approved: true },
      { onStageDelta, onStepUpdate, onDelta, onDone },
    );

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/ai/resume/step/stream'),
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          session_id: 'sess-1',
          plan_id: 'plan-1',
          step_id: 'step-1',
          checkpoint_id: 'cp-1',
          approved: true,
        }),
      }),
    );
    expect(onStageDelta).toHaveBeenCalledWith(expect.objectContaining({ stage: 'execute', status: 'loading' }));
    expect(onStepUpdate).toHaveBeenCalledWith(expect.objectContaining({ step_id: 'step-1', status: 'success' }));
    expect(onDelta).toHaveBeenCalledWith(expect.objectContaining({ contentChunk: '继续执行' }));
    expect(onDone).toHaveBeenCalled();
  });

  it('keeps checkpoint_id approval events while preserving canonical resume identity fields', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: approval_required\ndata: {"id":"approval-1","session_id":"sess-1","plan_id":"plan-1","step_id":"step-1","checkpoint_id":"cp-1","tool":"scale_deployment","status":"pending"}\n\n',
      ]),
    } as Response);

    const onApprovalRequired = vi.fn();

    await aiApi.chatStream(
      { message: 'hi', context: { scene: 'global' } },
      { onApprovalRequired }
    );

    expect(onApprovalRequired).toHaveBeenCalledWith(expect.objectContaining({
      id: 'approval-1',
      session_id: 'sess-1',
      plan_id: 'plan-1',
      step_id: 'step-1',
      checkpoint_id: 'cp-1',
      tool: 'scale_deployment',
      status: 'pending',
    }));
  });
});
