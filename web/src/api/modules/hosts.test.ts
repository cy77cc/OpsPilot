import { beforeEach, describe, expect, it, vi } from 'vitest';
import { scopeStore } from '../../app/scope/scopeStore';
import { hostApi } from './hosts';

describe('hostApi.downloadFile', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    scopeStore.clearScope();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: {
          get: () => 'application/octet-stream',
        },
        json: vi.fn(),
        blob: vi.fn().mockResolvedValue(new Blob(['ok'])),
      })
    );
  });

  it('uses contextual fetch without Authorization header', async () => {
    scopeStore.setScope({ projectId: '42' });

    await hostApi.downloadFile('7', '/tmp/app.yaml');

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/hosts/7/files/download'),
      expect.objectContaining({
        credentials: 'include',
        headers: expect.objectContaining({
          'X-Project-ID': '42',
        }),
      })
    );
    expect(
      (fetch as unknown as ReturnType<typeof vi.fn>).mock.calls[0][1].headers
    ).not.toHaveProperty('Authorization');
  });
});
