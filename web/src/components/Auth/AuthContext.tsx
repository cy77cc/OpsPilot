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

  // 处理 token 刷新成功事件
  const handleTokenRefreshed = useCallback(
    (_event: Event) => {
      console.log('[Auth] Token refreshed successfully');
      void refreshUser().catch(() => {
        sessionStore.clearSession();
      });
    },
    [refreshUser]
  );

  // 处理 token 过期事件
  const handleTokenExpired = useCallback(() => {
    console.log('[Auth] Token expired, redirecting to login');

    // 清除状态
    sessionStore.clearSession();

    // 保存当前路径用于登录后重定向
    const currentPath = window.location.pathname + window.location.search;
    if (currentPath && !currentPath.includes('/login')) {
      sessionStorage.setItem('redirectAfterLogin', currentPath);
    }

    // 跳转到登录页
    window.location.href = '/login';
  }, []);

  // 初始化和事件监听
  useEffect(() => {
    void refreshUser().catch(() => undefined);

    // 监听 token 事件
    window.addEventListener(TOKEN_EVENTS.REFRESHED, handleTokenRefreshed);
    window.addEventListener(TOKEN_EVENTS.EXPIRED, handleTokenExpired);

    return () => {
      // 清理事件监听
      window.removeEventListener(TOKEN_EVENTS.REFRESHED, handleTokenRefreshed);
      window.removeEventListener(TOKEN_EVENTS.EXPIRED, handleTokenExpired);
    };
  }, [handleTokenRefreshed, handleTokenExpired, refreshUser]);

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
