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
      handlers.onDelta?.({ contentChunk: 'hello ' });
      handlers.onDelta?.({ contentChunk: 'world' });
      handlers.onDone?.({ session: {} as any });
    });

    request.run({ message: 'hi', scene: 'ai' });
    await request.asyncHandler;

    expect(onUpdate).toHaveBeenCalledTimes(2);
    expect(onSuccess).toHaveBeenCalledWith(
      [{ content: 'hello ' }, { content: 'world' }],
      expect.any(Headers),
    );
    expect(onError).not.toHaveBeenCalled();
  });
});
