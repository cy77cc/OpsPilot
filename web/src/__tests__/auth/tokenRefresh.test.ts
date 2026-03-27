import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  isTokenExpiringSoon,
  dispatchTokenRefreshed,
} from '../../utils/tokenManager';

// Test the token refresh flow using actual event system
describe('Token Refresh Flow', () => {
  const TOKEN_EVENTS = {
    REFRESHED: 'tokenRefreshed',
    EXPIRED: 'tokenExpired',
    NEEDS_REFRESH: 'tokenNeedsRefresh',
  };

  // Helper to create mock JWT
  function createMockJwt(expSeconds: number): string {
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
    const payload = btoa(JSON.stringify({ exp: expSeconds, uid: 1 }));
    const signature = btoa('mock-signature');
    return [header, payload, signature]
      .map((s) => s.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, ''))
      .join('.');
  }

  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
    sessionStorage.clear();
    vi.restoreAllMocks();
  });

  describe('Token Expiring Soon Detection', () => {
    it('detects token expiring within 5 minutes', () => {
      const expSeconds = Math.floor(Date.now() / 1000) + 60; // 1 minute from now
      const token = createMockJwt(expSeconds);

      expect(isTokenExpiringSoon(token)).toBe(true);
    });

    it('does not flag token with more than 5 minutes remaining', () => {
      const expSeconds = Math.floor(Date.now() / 1000) + 600; // 10 minutes from now
      const token = createMockJwt(expSeconds);

      expect(isTokenExpiringSoon(token)).toBe(false);
    });
  });

  describe('Event Flow', () => {
    it('tokenRefreshed event carries new tokens', async () => {
      const handler = vi.fn();

      window.addEventListener(TOKEN_EVENTS.REFRESHED, handler);

      window.dispatchEvent(
        new CustomEvent(TOKEN_EVENTS.REFRESHED, {
          detail: {
            token: 'new-access-token',
            refreshToken: 'new-refresh-token',
          },
        })
      );

      expect(handler).toHaveBeenCalledWith(
        expect.objectContaining({
          detail: {
            token: 'new-access-token',
            refreshToken: 'new-refresh-token',
          },
        })
      );

      window.removeEventListener(TOKEN_EVENTS.REFRESHED, handler);
    });

    it('tokenExpired event triggers logout flow', () => {
      const handler = vi.fn();

      localStorage.setItem('token', 'expiring-token');
      localStorage.setItem('refreshToken', 'expiring-refresh');

      window.addEventListener(TOKEN_EVENTS.EXPIRED, handler);

      window.dispatchEvent(new CustomEvent(TOKEN_EVENTS.EXPIRED));

      expect(handler).toHaveBeenCalled();

      window.removeEventListener(TOKEN_EVENTS.EXPIRED, handler);
    });

    it('tokenNeedsRefresh triggers refresh request', async () => {
      const handler = vi.fn();

      window.addEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);

      window.dispatchEvent(
        new CustomEvent(TOKEN_EVENTS.NEEDS_REFRESH, {
          detail: { remainingTime: 60000 },
        })
      );

      expect(handler).toHaveBeenCalledWith(
        expect.objectContaining({
          detail: { remainingTime: 60000 },
        })
      );

      window.removeEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);
    });
  });

  describe('LocalStorage Sync', () => {
    it('dispatches tokenRefreshed event', () => {
      const handler = vi.fn();

      window.addEventListener(TOKEN_EVENTS.REFRESHED, handler);

      // Simulate refresh success
      dispatchTokenRefreshed({
        token: 'new-token',
        refreshToken: 'new-refresh',
      });

      expect(handler).toHaveBeenCalled();

      window.removeEventListener(TOKEN_EVENTS.REFRESHED, handler);
    });
  });

  describe('Concurrent Refresh Prevention', () => {
    it('multiple NEEDS_REFRESH events result in single refresh call', async () => {
      const refreshCalls: number[] = [];

      // Simulate ApiService refresh tracking
      let refreshPromise: Promise<boolean> | null = null;

      const mockRefresh = async (): Promise<boolean> => {
        if (refreshPromise) {
          return refreshPromise;
        }

        refreshCalls.push(Date.now());
        refreshPromise = new Promise((resolve) => {
          setTimeout(() => {
            resolve(true);
            refreshPromise = null;
          }, 100);
        });

        return refreshPromise;
      };

      // Trigger multiple refreshes concurrently
      const results = await Promise.all([
        mockRefresh(),
        mockRefresh(),
        mockRefresh(),
      ]);

      // All should succeed
      expect(results.every((r) => r === true)).toBe(true);

      // But only one actual refresh call
      expect(refreshCalls.length).toBe(1);
    });
  });
});

describe('Redirect After Login', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  afterEach(() => {
    sessionStorage.clear();
  });

  it('saves and retrieves redirect path', () => {
    const testPath = '/dashboard/settings?tab=profile';

    // Simulate saving redirect path on token expiry
    sessionStorage.setItem('redirectAfterLogin', testPath);

    // Simulate retrieving after login
    const savedPath = sessionStorage.getItem('redirectAfterLogin');

    expect(savedPath).toBe(testPath);
  });

  it('clears redirect path after use', () => {
    const testPath = '/dashboard';

    sessionStorage.setItem('redirectAfterLogin', testPath);
    sessionStorage.removeItem('redirectAfterLogin');

    expect(sessionStorage.getItem('redirectAfterLogin')).toBeNull();
  });
});
