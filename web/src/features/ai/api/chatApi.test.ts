import { beforeEach, describe, expect, it, vi } from 'vitest';
import { scopeStore } from '../../../app/scope/scopeStore';
import { chatStream } from './chatApi';

const mockConsumeAIStream = vi.fn().mockResolvedValue(undefined);

vi.mock('./shared', () => ({
  consumeAIStream: (...args: unknown[]) => mockConsumeAIStream(...args),
  apiService: {},
}));

describe('chatStream', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    scopeStore.clearScope();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response('', {
          status: 200,
          headers: { 'Content-Type': 'text/event-stream' },
        })
      )
    );
  });

  it('sends cookie-based request context without localStorage bearer auth', async () => {
    scopeStore.setScope({ projectId: '42' });

    await chatStream({ message: 'hello' }, {});

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/ai/chat'),
      expect.objectContaining({
        credentials: 'include',
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
          'X-Project-ID': '42',
        }),
      })
    );
    expect(
      (fetch as unknown as ReturnType<typeof vi.fn>).mock.calls[0][1].headers
    ).not.toHaveProperty('Authorization');
    expect(mockConsumeAIStream).toHaveBeenCalledTimes(1);
  });
});
