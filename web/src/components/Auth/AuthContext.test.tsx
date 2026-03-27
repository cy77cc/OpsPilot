import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import React from 'react';

// Mock the auth API module
const mockGetMe = vi.fn();

vi.mock('../../api/modules/auth', () => ({
  authApi: {
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    getMe: () => mockGetMe(),
  },
}));

// Mock the api service
vi.mock('../../api/api', () => ({
  default: {
    refreshAccessToken: vi.fn(),
  },
  TOKEN_EVENTS: {
    REFRESHED: 'tokenRefreshed',
    EXPIRED: 'tokenExpired',
    NEEDS_REFRESH: 'tokenNeedsRefresh',
  },
}));

// Mock tokenManager functions
const mockStartTokenExpiryCheck = vi.fn();
const mockStopTokenExpiryCheck = vi.fn();

vi.mock('../../utils/tokenManager', () => ({
  startTokenExpiryCheck: () => mockStartTokenExpiryCheck(),
  stopTokenExpiryCheck: () => mockStopTokenExpiryCheck(),
  dispatchTokenRefreshed: vi.fn(),
  dispatchTokenExpired: vi.fn(),
  TOKEN_EVENTS: {
    REFRESHED: 'tokenRefreshed',
    EXPIRED: 'tokenExpired',
    NEEDS_REFRESH: 'tokenNeedsRefresh',
  },
}));

// Import after mocks are set up
import { AuthProvider, useAuth } from './AuthContext';

// Simple test component
const TokenDisplay = () => {
  const { token } = useAuth();
  return <div data-testid="token-display">{token || 'no-token'}</div>;
};

const AuthStatusDisplay = () => {
  const { isAuthenticated } = useAuth();
  return <div data-testid="auth-status">{isAuthenticated ? 'authenticated' : 'not-authenticated'}</div>;
};

describe('AuthContext Token Refresh Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    sessionStorage.clear();

    // Default mock implementations
    mockGetMe.mockResolvedValue({
      data: {
        id: 1,
        username: 'testuser',
        name: 'Test User',
        email: 'test@example.com',
        status: 'active',
        roles: ['user'],
        permissions: [],
      },
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    sessionStorage.clear();
  });

  describe('Token Check Lifecycle', () => {
    it('starts token check on mount when token exists', async () => {
      localStorage.setItem('token', 'existing-token');

      await act(async () => {
        render(
          <AuthProvider>
            <TokenDisplay />
          </AuthProvider>
        );
      });

      // Should start token check on mount with existing token
      expect(mockStartTokenExpiryCheck).toHaveBeenCalled();
    });

    it('does not start token check when no token exists', async () => {
      await act(async () => {
        render(
          <AuthProvider>
            <TokenDisplay />
          </AuthProvider>
        );
      });

      // Should not start token check without token
      expect(mockStartTokenExpiryCheck).not.toHaveBeenCalled();
    });

    it('stops token check on unmount', async () => {
      localStorage.setItem('token', 'existing-token');

      const { unmount } = render(
        <AuthProvider>
          <TokenDisplay />
        </AuthProvider>
      );

      await act(async () => {
        // Wait for initial render
        await new Promise((resolve) => setTimeout(resolve, 100));
      });

      await act(async () => {
        unmount();
      });

      expect(mockStopTokenExpiryCheck).toHaveBeenCalled();
    });
  });

  describe('Initial State', () => {
    it('shows not authenticated when no token', async () => {
      await act(async () => {
        render(
          <AuthProvider>
            <AuthStatusDisplay />
          </AuthProvider>
        );
      });

      await waitFor(() => {
        expect(screen.getByTestId('auth-status').textContent).toBe('not-authenticated');
      });
    });
  });

  describe('Event System', () => {
    it('can dispatch and listen to tokenRefreshed event', async () => {
      const handler = vi.fn();
      window.addEventListener('tokenRefreshed', handler);

      window.dispatchEvent(
        new CustomEvent('tokenRefreshed', {
          detail: { token: 'new-token' },
        })
      );

      expect(handler).toHaveBeenCalled();

      window.removeEventListener('tokenRefreshed', handler);
    });

    it('can dispatch and listen to tokenExpired event', async () => {
      const handler = vi.fn();
      window.addEventListener('tokenExpired', handler);

      window.dispatchEvent(new CustomEvent('tokenExpired'));

      expect(handler).toHaveBeenCalled();

      window.removeEventListener('tokenExpired', handler);
    });

    it('can dispatch and listen to tokenNeedsRefresh event', async () => {
      const handler = vi.fn();
      window.addEventListener('tokenNeedsRefresh', handler);

      window.dispatchEvent(new CustomEvent('tokenNeedsRefresh'));

      expect(handler).toHaveBeenCalled();

      window.removeEventListener('tokenNeedsRefresh', handler);
    });
  });
});
