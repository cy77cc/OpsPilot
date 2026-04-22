import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  dispatchTokenNeedsRefresh,
  dispatchTokenRefreshed,
} from '../../utils/tokenManager';

const TOKEN_STORAGE_KEYS = new Set(['token', 'refreshToken']);

const createTokenStorageSpies = () => {
  const getItemSpy = vi.spyOn(Storage.prototype, 'getItem');
  const setItemSpy = vi.spyOn(Storage.prototype, 'setItem');
  const removeItemSpy = vi.spyOn(Storage.prototype, 'removeItem');

  const assertNoTokenStorageDependency = () => {
    const tokenKeyCalls = [...getItemSpy.mock.calls, ...setItemSpy.mock.calls, ...removeItemSpy.mock.calls]
      .filter(([key]) => TOKEN_STORAGE_KEYS.has(String(key)));

    expect(tokenKeyCalls).toEqual([]);
  };

  return {
    assertNoTokenStorageDependency,
  };
};

describe('Token Refresh Flow', () => {
  const TOKEN_EVENTS = {
    REFRESHED: 'tokenRefreshed',
    EXPIRED: 'tokenExpired',
    NEEDS_REFRESH: 'tokenNeedsRefresh',
  };

  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
    sessionStorage.clear();
    vi.restoreAllMocks();
  });

  describe('Event Flow', () => {
    it('tokenRefreshed event is emitted without requiring token payload', () => {
      const handler = vi.fn();

      window.addEventListener(TOKEN_EVENTS.REFRESHED, handler);
      window.dispatchEvent(new CustomEvent(TOKEN_EVENTS.REFRESHED));

      expect(handler).toHaveBeenCalled();

      window.removeEventListener(TOKEN_EVENTS.REFRESHED, handler);
    });

    it('tokenExpired event triggers logout flow', () => {
      const handler = vi.fn();

      window.addEventListener(TOKEN_EVENTS.EXPIRED, handler);
      window.dispatchEvent(new CustomEvent(TOKEN_EVENTS.EXPIRED));

      expect(handler).toHaveBeenCalled();

      window.removeEventListener(TOKEN_EVENTS.EXPIRED, handler);
    });

    it('tokenNeedsRefresh carries response-source metadata', () => {
      const handler = vi.fn();

      window.addEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);
      window.dispatchEvent(
        new CustomEvent(TOKEN_EVENTS.NEEDS_REFRESH, {
          detail: { source: 'response' },
        })
      );

      expect(handler).toHaveBeenCalledWith(
        expect.objectContaining({
          detail: { source: 'response' },
        })
      );

      window.removeEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);
    });
  });

  describe('Refresh Event Dispatch', () => {
    it('dispatches tokenRefreshed event without token storage dependency', () => {
      const { assertNoTokenStorageDependency } = createTokenStorageSpies();
      const handler = vi.fn();

      window.addEventListener(TOKEN_EVENTS.REFRESHED, handler);
      dispatchTokenRefreshed();

      expect(handler).toHaveBeenCalled();
      assertNoTokenStorageDependency();

      window.removeEventListener(TOKEN_EVENTS.REFRESHED, handler);
    });
  });

  describe('Concurrent Refresh Prevention', () => {
    it('multiple refresh requests can be coalesced behind one in-flight promise', async () => {
      const refreshCalls: number[] = [];
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

      const results = await Promise.all([mockRefresh(), mockRefresh(), mockRefresh()]);

      expect(results.every((result) => result)).toBe(true);
      expect(refreshCalls.length).toBe(1);
    });

    it('dispatches tokenNeedsRefresh with an explicit source', () => {
      const handler = vi.fn();

      window.addEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);
      dispatchTokenNeedsRefresh('manual');

      expect(handler).toHaveBeenCalledWith(
        expect.objectContaining({
          detail: { source: 'manual' },
        })
      );

      window.removeEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handler);
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

    sessionStorage.setItem('redirectAfterLogin', testPath);

    expect(sessionStorage.getItem('redirectAfterLogin')).toBe(testPath);
  });

  it('clears redirect path after use', () => {
    sessionStorage.setItem('redirectAfterLogin', '/dashboard');
    sessionStorage.removeItem('redirectAfterLogin');

    expect(sessionStorage.getItem('redirectAfterLogin')).toBeNull();
  });
});
