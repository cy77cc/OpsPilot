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
      handlers.onInit?.({ session_id: 'sess-1', run_id: 'run-1' });
      handlers.onIntent?.({ assistant_type: 'planner', intent_type: 'unknown' });
      handlers.onDelta?.({ contentChunk: 'hello ' });
      handlers.onDelta?.({ contentChunk: 'world' });
      handlers.onDone?.({ session: {} as any });
    });

    request.run({ message: 'hi', scene: 'ai' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenCalledTimes(4);
    expect(onUpdate).toHaveBeenNthCalledWith(
      1,
      { content: '[准备中]', mode: 'replace' },
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenNthCalledWith(
      2,
      { content: '[正在规划处理方式]', mode: 'replace' },
      expect.any(Headers),
    );
    expect(onSuccess).toHaveBeenCalledWith(
      [{ content: 'hello ', mode: 'replace' }, { content: 'world', mode: 'append' }],
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
      handlers.onInit?.({ session_id: 'sess-1', run_id: 'run-1' });
      handlers.onStatus?.({ status: 'running' });
      handlers.onDelta?.({ contentChunk: 'successfully transferred to agent [DiagnosisAgent]' });
      handlers.onIntent?.({ assistant_type: 'planner', intent_type: 'unknown' });
      handlers.onDelta?.({ contentChunk: '诊断完成' });
      handlers.onDone?.({ session: {} as any });
    });

    request.run({ message: 'hi', scene: 'ai' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenNthCalledWith(
      1,
      { content: '[准备中]', mode: 'replace' },
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenNthCalledWith(
      2,
      { content: '[识别任务]', mode: 'replace' },
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenNthCalledWith(
      3,
      { content: '[诊断助手开始处理]', mode: 'replace' },
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenNthCalledWith(
      4,
      { content: '诊断完成', mode: 'replace' },
      expect.any(Headers),
    );
    expect(onSuccess).toHaveBeenCalledWith(
      [{ content: '诊断完成', mode: 'replace' }],
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
      handlers.onIntent?.({ assistant_type: 'planner', intent_type: 'unknown' });
      handlers.onDelta?.({ contentChunk: '第一段' });
      handlers.onDelta?.({ contentChunk: '，第二段' });
      handlers.onDone?.({ session: {} as any });
    });

    request.run({ message: 'hi', scene: 'ai' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenNthCalledWith(
      1,
      { content: '[正在规划处理方式]', mode: 'replace' },
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenNthCalledWith(
      2,
      { content: '第一段', mode: 'replace' },
      expect.any(Headers),
    );
    expect(onUpdate).toHaveBeenNthCalledWith(
      3,
      { content: '，第二段', mode: 'append' },
      expect.any(Headers),
    );
    expect(onSuccess).toHaveBeenCalledWith(
      [
        { content: '第一段', mode: 'replace' },
        { content: '，第二段', mode: 'append' },
      ],
      expect.any(Headers),
    );
  });
});
