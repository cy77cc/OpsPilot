import React, { createContext, useContext, useEffect, useMemo, useState, useCallback } from 'react';
import type { ReactNode } from 'react';
import { authApi } from '../../api/modules/auth';
import type { AuthUser, LoginParams, RegisterParams } from '../../api/modules/auth';
import apiService, { TOKEN_EVENTS } from '../../api/api';
import {
  startTokenExpiryCheck,
  stopTokenExpiryCheck,
} from '../../utils/tokenManager';

interface AuthContextType {
  user: AuthUser | null;
  token: string | null;
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
  const [token, setToken] = useState<string | null>(localStorage.getItem('token'));
  const [loading, setLoading] = useState(true);

  const persistSession = (nextToken: string, nextUser: AuthUser, permissions?: string[]) => {
    localStorage.setItem('token', nextToken);
    localStorage.setItem('user', JSON.stringify(nextUser));
    if (permissions) {
      localStorage.setItem('permissions', JSON.stringify(permissions));
    }
    setToken(nextToken);
    setUser(nextUser);
  };

  const clearSession = useCallback(() => {
    localStorage.removeItem('token');
    localStorage.removeItem('refreshToken');
    localStorage.removeItem('user');
    localStorage.removeItem('permissions');
    setToken(null);
    setUser(null);
  }, []);

  const refreshUser = async () => {
    if (!localStorage.getItem('token')) {
      clearSession();
      return;
    }
    const res = await authApi.getMe();
    setUser(res.data);
    localStorage.setItem('user', JSON.stringify(res.data));
    if (res.data.permissions) {
      localStorage.setItem('permissions', JSON.stringify(res.data.permissions));
    }
  };

  const login = async (payload: LoginParams) => {
    const res = await authApi.login(payload);
    if (res.data.refreshToken) {
      localStorage.setItem('refreshToken', res.data.refreshToken);
    }
    persistSession(res.data.token, res.data.user, res.data.permissions);
    // 登录成功后启动 token 过期检查
    startTokenExpiryCheck();
  };

  const register = async (payload: RegisterParams) => {
    const res = await authApi.register(payload);
    if (res.data.refreshToken) {
      localStorage.setItem('refreshToken', res.data.refreshToken);
    }
    persistSession(res.data.token, res.data.user, res.data.permissions);
    // 注册成功后启动 token 过期检查
    startTokenExpiryCheck();
  };

  const logout = useCallback(() => {
    const refreshToken = localStorage.getItem('refreshToken') || undefined;
    void authApi.logout(refreshToken).catch(() => undefined);
    // 登出时停止 token 过期检查
    stopTokenExpiryCheck();
    clearSession();
  }, [clearSession]);

  // 处理 token 刷新成功事件
  const handleTokenRefreshed = useCallback(
    (event: Event) => {
      const customEvent = event as CustomEvent<{ token: string; refreshToken?: string }>;
      const { token: newToken } = customEvent.detail;

      // 更新状态
      setToken(newToken);

      // localStorage 已在 ApiService 中更新
      console.log('[Auth] Token refreshed successfully');
    },
    []
  );

  // 处理 token 过期事件
  const handleTokenExpired = useCallback(() => {
    console.log('[Auth] Token expired, redirecting to login');

    // 停止检查
    stopTokenExpiryCheck();

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

  // 处理需要刷新 token 事件（主动刷新）
  const handleNeedsRefresh = useCallback(async () => {
    console.log('[Auth] Token needs refresh, triggering refresh');
    await apiService.refreshAccessToken();
  }, []);

  // 初始化和事件监听
  useEffect(() => {
    let mounted = true;

    const bootstrap = async () => {
      try {
        const storedToken = localStorage.getItem('token');
        if (!storedToken) {
          clearSession();
          return;
        }
        const userText = localStorage.getItem('user');
        if (userText && mounted) {
          setUser(JSON.parse(userText) as AuthUser);
        }
        await refreshUser();

        // 初始化成功后启动 token 过期检查
        if (mounted) {
          startTokenExpiryCheck();
        }
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
    window.addEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handleNeedsRefresh);

    return () => {
      mounted = false;

      // 清理事件监听
      window.removeEventListener(TOKEN_EVENTS.REFRESHED, handleTokenRefreshed);
      window.removeEventListener(TOKEN_EVENTS.EXPIRED, handleTokenExpired);
      window.removeEventListener(TOKEN_EVENTS.NEEDS_REFRESH, handleNeedsRefresh);

      // 停止 token 检查
      stopTokenExpiryCheck();
    };
  }, [clearSession, handleTokenRefreshed, handleTokenExpired, handleNeedsRefresh]);

  const value = useMemo(
    () => ({
      user,
      token,
      loading,
      isAuthenticated: Boolean(token && user),
      login,
      register,
      logout,
      refreshUser,
    }),
    [user, token, loading, logout]
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
