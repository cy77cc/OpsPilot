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

  it('sends request context headers with cookie credentials and no Authorization header', async () => {
    scopeStore.setScope({ projectId: '42', teamId: 'team-7' });

    await chatStream({ message: 'hello' }, {});

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/ai/chat'),
      expect.objectContaining({
        credentials: 'include',
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
          'X-Project-ID': '42',
          'X-Team-ID': 'team-7',
        }),
      })
    );
    expect(
      (fetch as unknown as ReturnType<typeof vi.fn>).mock.calls[0][1].headers
    ).not.toHaveProperty('Authorization');
    expect(mockConsumeAIStream).toHaveBeenCalledTimes(1);
  });
});
