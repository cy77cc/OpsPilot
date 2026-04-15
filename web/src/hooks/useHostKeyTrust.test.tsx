import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useHostKeyTrust } from './useHostKeyTrust';

const mockApi = vi.hoisted(() => ({
  hosts: {
    trustHostKey: vi.fn(),
  },
}));

vi.mock('../api', () => ({
  Api: mockApi,
}));

describe('useHostKeyTrust', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.hosts.trustHostKey.mockResolvedValue({ data: {} });
  });

  it('calls trustHostKey then retries original operation exactly once', async () => {
    const original = vi.fn()
      .mockRejectedValueOnce({
        businessCode: 2000,
        message: 'ssh host key verification failed',
        response: {
          data: {
            data: {
              error_type: 'ssh_host_key_unknown',
              host_key: {
                host: '118.193.38.89',
                port: 13012,
                fingerprint_sha256: 'SHA256:test',
                algorithm: 'ssh-ed25519',
                public_key: 'ssh-ed25519 AAAATEST',
              },
            },
          },
        },
      })
      .mockResolvedValueOnce({ data: { reachable: true } });

    const { result } = renderHook(() => useHostKeyTrust('10'));

    await act(async () => {
      await expect(result.current.runWithTrustRetry(original)).rejects.toBeTruthy();
    });
    expect(result.current.pendingTrust?.errorType).toBe('ssh_host_key_unknown');
    expect(result.current.pendingTrust?.hostKey.fingerprintSha256).toBe('SHA256:test');

    await act(async () => {
      await result.current.confirmTrustAndRetry(async () => {
        await result.current.runWithTrustRetry(original);
      });
    });

    expect(mockApi.hosts.trustHostKey).toHaveBeenCalledTimes(1);
    expect(mockApi.hosts.trustHostKey).toHaveBeenCalledWith('10', expect.objectContaining({
      host: '118.193.38.89',
      port: 13012,
      fingerprintSha256: 'SHA256:test',
      replaceExisting: false,
    }));
    expect(original).toHaveBeenCalledTimes(2);
    expect(result.current.pendingTrust).toBeNull();
  });
});
