import { describe, expect, it, vi } from 'vitest';
import { PlatformChatProvider, PlatformChatRequest } from '../providers/PlatformChatProvider';
import { aiApi } from '../../../api/modules/ai';

vi.mock('../../../api/modules/ai', () => ({
  aiApi: {
    chatStream: vi.fn(),
  },
}));

describe('PlatformChatProvider', () => {
  it('merges scene, session, and context into request params', () => {
    const provider = new PlatformChatProvider({
      scene: 'cluster',
      getSessionId: () => 'sess-1',
      getSceneContext: () => ({ route: '/deployment/infrastructure/clusters/42', resourceId: '42' }),
    });

    const params = provider.transformParams(
      { message: 'check health' },
      { params: undefined } as any,
    );

    expect(params).toEqual({
      message: 'check health',
      sessionId: 'sess-1',
      scene: 'cluster',
      context: {
        route: '/deployment/infrastructure/clusters/42',
        resourceId: '42',
      },
    });
  });

  it('streams delta chunks through request callbacks', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    const onSuccess = vi.fn();
    const onError = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess,
      onError,
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
      handlers.onPlan?.({ steps: ['inspect pods'], iteration: 0 });
      handlers.onDelta?.({ content: 'hello ' });
      handlers.onDelta?.({ content: 'world' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'ai' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenCalledTimes(5);
    expect(onUpdate).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ content: '[准备中]', mode: 'replace' }),
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ content: '[正在规划处理方式]', mode: 'replace' }),
      expect.any(Headers),
    );
    expect(onSuccess).toHaveBeenCalledWith(
      [
        expect.objectContaining({ content: 'hello ', mode: 'replace' }),
        expect.objectContaining({ content: 'world', mode: 'append' }),
      ],
      expect.any(Headers),
    );
    expect(onError).not.toHaveBeenCalled();
  });

  it('shows localized status updates and withholds visible content until intent', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    const onSuccess = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess,
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
      handlers.onAgentHandoff?.({ from: 'OpsPilotAgent', to: 'DiagnosisAgent', intent: 'diagnosis' });
      handlers.onDelta?.({ content: '诊断完成' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'ai' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ content: '[准备中]', mode: 'replace' }),
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ content: '[诊断助手开始处理]', mode: 'replace' }),
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenNthCalledWith(
      3,
      expect.objectContaining({ content: '诊断完成', mode: 'replace' }),
      expect.any(Headers),
    );
    expect(onSuccess).toHaveBeenCalledWith(
      [expect.objectContaining({ content: '诊断完成', mode: 'replace' })],
      expect.any(Headers),
    );
  });

  it('appends later visible chunks after replacing the placeholder', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    const onSuccess = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess,
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onPlan?.({ steps: ['step one'], iteration: 0 });
      handlers.onDelta?.({ content: '第一段' });
      handlers.onDelta?.({ content: '，第二段' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'ai' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ content: '[正在规划处理方式]', mode: 'replace' }),
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ content: '第一段', mode: 'replace' }),
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenNthCalledWith(
      3,
      expect.objectContaining({ content: '，第二段', mode: 'append' }),
      expect.any(Headers),
    );
    expect(onSuccess).toHaveBeenCalledWith(
      [
        expect.objectContaining({ content: '第一段', mode: 'replace' }),
        expect.objectContaining({ content: '，第二段', mode: 'append' }),
      ],
      expect.any(Headers),
    );
  });

  it('preserves runtime activities while streaming visible markdown', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess: vi.fn(),
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onToolCall?.({ call_id: 'call-1', tool_name: 'kubectl_get', arguments: {} });
      handlers.onDelta?.({ content: '巡检完成' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'cluster' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        runtime: expect.objectContaining({
          activities: [expect.objectContaining({ id: 'call-1' })],
        }),
      }),
      expect.any(Headers),
    );
  });

  it('projects replan, approval, recoverable error, handoff, tool result, and terminal error states', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    const onError = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess: vi.fn(),
      onError,
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
      handlers.onAgentHandoff?.({ from: 'OpsPilotAgent', to: 'DiagnosisAgent', intent: 'diagnosis' });
      handlers.onPlan?.({ steps: ['检查节点'], iteration: 0 });
      handlers.onReplan?.({ steps: ['检查节点', '汇总异常'], completed: 1, iteration: 1, is_final: false });
      handlers.onToolCall?.({ call_id: 'call-1', tool_name: 'kubectl_get', arguments: {} });
      handlers.onToolApproval?.({
        approval_id: 'approval-1',
        call_id: 'call-1',
        tool_name: 'kubectl_get',
        preview: {},
        timeout_seconds: 300,
      });
      handlers.onDelta?.({ content: '已经拿到部分结果' });
      handlers.onToolResult?.({ call_id: 'call-1', tool_name: 'kubectl_get', content: 'node-1 ok' });
      handlers.onError?.({ message: '工具执行较慢，正在继续等待结果…', code: 'tool_timeout_soft', recoverable: true });
      handlers.onError?.({ message: 'stream failed', code: 'stream_failed', recoverable: false });
    });

    request.run({ message: 'hi', scene: 'cluster' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        content: '已经拿到部分结果',
        runtime: expect.objectContaining({
          activities: expect.arrayContaining([
            expect.objectContaining({ kind: 'agent_handoff' }),
            expect.objectContaining({ kind: 'replan' }),
            expect.objectContaining({ kind: 'tool_result', id: 'call-1' }),
          ]),
          status: expect.objectContaining({ kind: 'soft-timeout' }),
        }),
      }),
      expect.any(Headers),
    );
    expect(onError).toHaveBeenCalledWith(
      expect.any(Error),
      expect.anything(),
      expect.any(Headers),
    );
  });

  it('buffers planner and replanner delta envelopes before exposing them', async () => {
    const request = new PlatformChatRequest();
    const onUpdate = vi.fn();
    const onSuccess = vi.fn();
    request.options.callbacks = {
      onUpdate,
      onSuccess,
      onError: vi.fn(),
    };

    vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
      handlers.onDelta?.({ agent: 'planner', content: '{"steps":["获取服务器列表",' });
      handlers.onDelta?.({ agent: 'planner', content: '"批量执行健康检查"]}' });
      handlers.onToolCall?.({ call_id: 'call-1', tool_name: 'host_list_inventory', arguments: {} });
      handlers.onDelta?.({ agent: 'replanner', content: '{"response":"## 检查完成' });
      handlers.onDelta?.({ agent: 'replanner', content: '\\n\\n全部正常"}' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
    });

    request.run({ message: 'hi', scene: 'host' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        content: '## 检查完成\n\n全部正常',
        runtime: expect.objectContaining({
          plan: expect.objectContaining({
            steps: [
              expect.objectContaining({ title: '获取服务器列表' }),
              expect.objectContaining({ title: '批量执行健康检查' }),
            ],
          }),
        }),
      }),
      expect.any(Headers),
    );
    expect(onSuccess).toHaveBeenCalledWith(
      [expect.objectContaining({ content: '## 检查完成\n\n全部正常', mode: 'replace' })],
      expect.any(Headers),
    );
  });
});
