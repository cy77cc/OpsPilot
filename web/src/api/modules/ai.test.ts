import { afterEach, describe, expect, it, vi } from 'vitest';
import { aiApi, normalizeVisibleStreamChunk } from './ai';
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

  it('maps message events to onDelta content', async () => {
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

  it('dispatches tool_call, tool_approval, and tool_result events', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: tool_call\ndata: {"call_id":"call-1","tool_name":"get_pods","arguments":"{\\"namespace\\":\\"default\\"}"}\n\n',
        'event: tool_approval\ndata: {"call_id":"call-1","tool_name":"delete_pod","risk":"high","summary":"即将删除 pod"}\n\n',
        'event: tool_result\ndata: {"call_id":"call-1","result":"{\\"ok\\":true}"}\n\n',
      ]),
    } as Response);

    const onToolCall = vi.fn();
    const onToolApproval = vi.fn();
    const onToolResult = vi.fn();

    await aiApi.chatStream(
      { message: 'hi', context: { scene: 'global' } },
      { onToolCall, onToolApproval, onToolResult },
    );

    expect(onToolCall).toHaveBeenCalledWith(expect.objectContaining({
      call_id: 'call-1',
      tool_name: 'get_pods',
    }));
    expect(onToolApproval).toHaveBeenCalledWith(expect.objectContaining({
      call_id: 'call-1',
      tool_name: 'delete_pod',
      risk: 'high',
    }));
    expect(onToolResult).toHaveBeenCalledWith(expect.objectContaining({
      call_id: 'call-1',
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
        'event: tool_call\ndata: {"call_id":"call-after-approval","tool_name":"execute_step"}\n\n',
        'event: tool_result\ndata: {"call_id":"call-after-approval","result":"{\\"ok\\":true}"}\n\n',
        'event: delta\ndata: {"contentChunk":"已继续执行"}\n\n',
      ]),
    } as Response);

    const onToolCall = vi.fn();
    const onToolResult = vi.fn();
    const onDelta = vi.fn();

    await aiApi.decideChainApprovalStream(
      'plan-1',
      'approval:step-1',
      true,
      { onToolCall, onToolResult, onDelta },
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
    expect(onToolCall).toHaveBeenCalledWith(expect.objectContaining({ call_id: 'call-after-approval' }));
    expect(onToolResult).toHaveBeenCalledWith(expect.objectContaining({ call_id: 'call-after-approval' }));
    expect(onDelta).toHaveBeenCalledWith(expect.objectContaining({ contentChunk: '已继续执行' }));
  });

  it('unwraps only complete response envelopes', () => {
    expect(normalizeVisibleStreamChunk('{"response":" ok\\n\\n| a | b |\\n"}')).toBe(' ok\n\n| a | b |\n');
    expect(normalizeVisibleStreamChunk('{"response":')).toBe('{"response":');
    expect(normalizeVisibleStreamChunk('\n\n| a | b |\n')).toBe('\n\n| a | b |\n');
  });
});
