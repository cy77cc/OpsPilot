import React, { createContext, useContext, useEffect, useMemo, useCallback, useSyncExternalStore } from 'react';
import type { ReactNode } from 'react';
import type { AuthUser, LoginParams, RegisterParams } from '../../api/modules/auth';
import { TOKEN_EVENTS } from '../../api/api';
import { sessionStore } from '../../app/session/sessionStore';

interface AuthContextType {
  user: AuthUser | null;
  loading: boolean;
  isAuthenticated: boolean;
  login: (payload: LoginParams) => Promise<void>;
  register: (payload: RegisterParams) => Promise<void>;
  logout: () => void;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const session = useSyncExternalStore(sessionStore.subscribe, sessionStore.getSnapshot, sessionStore.getSnapshot);

  const refreshUser = useCallback(async () => {
    await sessionStore.bootstrap();
  }, []);

  const login = async (payload: LoginParams) => {
    await sessionStore.login(payload);
  };

  const register = async (payload: RegisterParams) => {
    await sessionStore.register(payload);
  };

  const logout = useCallback(() => {
    void sessionStore.logout();
  }, []);

  const handleSessionRefreshed = useCallback(
    (_event: Event) => {
      void refreshUser().catch(() => {
        sessionStore.clearSession();
      });
    },
    [refreshUser]
  );

  const handleSessionExpired = useCallback(() => {
    const currentPath = window.location.pathname + window.location.search;

    if (!currentPath.includes('/login')) {
      sessionStorage.setItem('redirectAfterLogin', currentPath);
      sessionStore.clearSession();
      window.location.href = '/login';
    } else {
      // Already on login page — just clear session state, no reload needed
      sessionStore.clearSession();
    }
  }, []);

  useEffect(() => {
    void refreshUser().catch(() => undefined);

    window.addEventListener(TOKEN_EVENTS.REFRESHED, handleSessionRefreshed);
    window.addEventListener(TOKEN_EVENTS.EXPIRED, handleSessionExpired);

    return () => {
      window.removeEventListener(TOKEN_EVENTS.REFRESHED, handleSessionRefreshed);
      window.removeEventListener(TOKEN_EVENTS.EXPIRED, handleSessionExpired);
    };
  }, [handleSessionRefreshed, handleSessionExpired, refreshUser]);

  const value = useMemo(
    () => ({
      user: session.user,
      loading: session.loading,
      isAuthenticated: session.isAuthenticated,
      login,
      register,
      logout,
      refreshUser,
    }),
    [session, logout, refreshUser]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
};
