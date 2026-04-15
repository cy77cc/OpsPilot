import React, { createContext, useContext, useEffect, useMemo, useState, useCallback } from 'react';
import type { ReactNode } from 'react';
import { authApi } from '../../api/modules/auth';
import type { AuthUser, LoginParams, RegisterParams } from '../../api/modules/auth';
import { TOKEN_EVENTS } from '../../api/api';

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
  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(true);

  const persistSession = (nextUser: AuthUser, permissions?: string[]) => {
    localStorage.setItem('user', JSON.stringify(nextUser));
    if (permissions) {
      localStorage.setItem('permissions', JSON.stringify(permissions));
    }
    setUser(nextUser);
  };

  const clearSession = useCallback(() => {
    localStorage.removeItem('user');
    localStorage.removeItem('permissions');
    setUser(null);
  }, []);

  const refreshUser = useCallback(async () => {
    const res = await authApi.getMe();
    setUser(res.data);
    localStorage.setItem('user', JSON.stringify(res.data));
    if (res.data.permissions) {
      localStorage.setItem('permissions', JSON.stringify(res.data.permissions));
    }
  }, []);

  const login = async (payload: LoginParams) => {
    const res = await authApi.login(payload);
    persistSession(res.data.user, res.data.permissions);
  };

  const register = async (payload: RegisterParams) => {
    const res = await authApi.register(payload);
    persistSession(res.data.user, res.data.permissions);
  };

  const logout = useCallback(() => {
    void authApi.logout().catch(() => undefined);
    clearSession();
  }, [clearSession]);

  // 处理 token 刷新成功事件
  const handleTokenRefreshed = useCallback(
    (_event: Event) => {
      console.log('[Auth] Token refreshed successfully');
      void refreshUser().catch(() => clearSession());
    },
    [clearSession, refreshUser]
  );

  // 处理 token 过期事件
  const handleTokenExpired = useCallback(() => {
    console.log('[Auth] Token expired, redirecting to login');

    // 清除状态
    clearSession();

    // 保存当前路径用于登录后重定向
    const currentPath = window.location.pathname + window.location.search;
    if (currentPath && !currentPath.includes('/login')) {
      sessionStorage.setItem('redirectAfterLogin', currentPath);
    }

    // 跳转到登录页
    window.location.href = '/login';
  }, [clearSession]);

  // 初始化和事件监听
  useEffect(() => {
    let mounted = true;

    const bootstrap = async () => {
      try {
        const userText = localStorage.getItem('user');
        if (userText && mounted) {
          setUser(JSON.parse(userText) as AuthUser);
        }
        await refreshUser();
      } catch {
        clearSession();
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    };
    bootstrap();

    // 监听 token 事件
    window.addEventListener(TOKEN_EVENTS.REFRESHED, handleTokenRefreshed);
    window.addEventListener(TOKEN_EVENTS.EXPIRED, handleTokenExpired);

    return () => {
      mounted = false;

      // 清理事件监听
      window.removeEventListener(TOKEN_EVENTS.REFRESHED, handleTokenRefreshed);
      window.removeEventListener(TOKEN_EVENTS.EXPIRED, handleTokenExpired);
    };
  }, [clearSession, handleTokenRefreshed, handleTokenExpired, refreshUser]);

  const value = useMemo(
    () => ({
      user,
      loading,
      isAuthenticated: Boolean(user),
      login,
      register,
      logout,
      refreshUser,
    }),
    [user, loading, logout, refreshUser]
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
