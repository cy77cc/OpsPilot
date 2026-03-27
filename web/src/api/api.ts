import axios from 'axios';
import type { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import {
  TOKEN_EVENTS,
  dispatchTokenRefreshed,
  dispatchTokenExpired,
} from '../utils/tokenManager';

export class ApiRequestError extends Error {
  statusCode?: number;
  businessCode?: number;

  constructor(message: string, statusCode?: number, businessCode?: number) {
    super(message);
    this.name = 'ApiRequestError';
    this.statusCode = statusCode;
    this.businessCode = businessCode;
  }
}

// 响应数据结构
export interface ApiResponse<T = unknown> {
  success: boolean;
  data: T;
  message?: string;
  messageKey?: string;
  dataSource?: string;
  total?: number;
  error?: {
    code: string;
    message: string;
  };
}

// 分页响应结构
export interface PaginatedResponse<T> {
  total: number;
  list: T[];
}

// API服务类
class ApiService {
  private instance: AxiosInstance;
  private refreshPromise: Promise<boolean> | null = null;

  constructor() {
    this.instance = axios.create({
      baseURL: import.meta.env.VITE_API_BASE || '/api/v1',
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // 请求拦截器
    this.instance.interceptors.request.use(
      (config) => {
        // 添加认证token
        const token = localStorage.getItem('token');
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        const projectId = localStorage.getItem('projectId');
        if (projectId) {
          config.headers['X-Project-ID'] = projectId;
        }
        const teamId = localStorage.getItem('teamId');
        if (teamId) {
          config.headers['X-Team-ID'] = teamId;
        }
        return config;
      },
      (error) => {
        return Promise.reject(error);
      }
    );

    // 响应拦截器
    this.instance.interceptors.response.use(
      (response: AxiosResponse<any>) => {
        const payload = response.data;
        const originalConfig = response.config as AxiosRequestConfig & { _retry?: boolean };
        const requestURL = String(originalConfig.url || '');
        // 兼容后端统一结构：{ code, msg/message, data, total }
        if (typeof payload?.code === 'number') {
          if ((payload.code === 4005 || payload.code === 4006) && !requestURL.includes('/auth/refresh') && !originalConfig._retry) {
            originalConfig._retry = true;
            return this.tryRefreshAndRetry(originalConfig);
          }
          if (payload.code !== 1000 && payload.code !== 200) {
            return Promise.reject(new ApiRequestError(payload.msg || payload.message || '请求失败', response.status, payload.code));
          }
          response.data = {
            success: true,
            message: payload.msg || payload.message,
            messageKey: payload.message_key,
            dataSource: payload.data_source,
            data: payload.data,
            total: payload.total,
          } as ApiResponse;
          return response;
        }

        if (!payload?.success) {
          return Promise.reject(new ApiRequestError(payload?.error?.message || payload?.message || '请求失败', response.status));
        }
        return response;
      },
      (error) => {
        const originalConfig = (error.config || {}) as AxiosRequestConfig & { _retry?: boolean };
        const requestURL = String(originalConfig.url || '');
        if (error.response?.status === 401 && !requestURL.includes('/auth/refresh') && !originalConfig._retry) {
          originalConfig._retry = true;
          return this.tryRefreshAndRetry(originalConfig);
        }
        const message = error.response?.data?.message || error.response?.data?.error?.message || error.message || '网络错误';
        const businessCode = typeof error.response?.data?.code === 'number' ? error.response.data.code : undefined;
        return Promise.reject(new ApiRequestError(message, error.response?.status, businessCode));
      }
    );
  }

  // GET请求
  async get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await this.instance.get<ApiResponse<T>>(url, config);
    return response.data;
  }

  // POST请求
  async post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await this.instance.post<ApiResponse<T>>(url, data, config);
    return response.data;
  }

  // PUT请求
  async put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await this.instance.put<ApiResponse<T>>(url, data, config);
    return response.data;
  }

  // PATCH请求
  async patch<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await this.instance.patch<ApiResponse<T>>(url, data, config);
    return response.data;
  }

  // DELETE请求
  async delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await this.instance.delete<ApiResponse<T>>(url, config);
    return response.data;
  }

  private async tryRefreshAndRetry(config: AxiosRequestConfig): Promise<AxiosResponse<any>> {
    const refreshed = await this.refreshAccessToken();
    if (!refreshed) {
      return Promise.reject(new Error('登录已过期，请重新登录'));
    }
    const nextConfig: AxiosRequestConfig = {
      ...config,
      headers: {
        ...(config.headers || {}),
        Authorization: `Bearer ${localStorage.getItem('token') || ''}`,
      },
    };
    return this.instance.request<ApiResponse<any>>(nextConfig);
  }

  /**
   * 刷新 accessToken
   * 公开方法，支持主动刷新和事件通知
   * @returns 刷新是否成功
   */
  async refreshAccessToken(): Promise<boolean> {
    // 复用进行中的刷新请求，确保并发刷新只发起一次
    if (this.refreshPromise) {
      return this.refreshPromise;
    }

    const refreshToken = localStorage.getItem('refreshToken');
    if (!refreshToken) {
      dispatchTokenExpired();
      return false;
    }

    this.refreshPromise = (async () => {
      try {
        const response = await this.instance.post<ApiResponse<any>>('/auth/refresh', { refreshToken });
        const payload = response.data;
        const token = payload?.data?.accessToken || payload?.data?.token;
        const nextRefreshToken = payload?.data?.refreshToken;

        if (!token) {
          dispatchTokenExpired();
          return false;
        }

        // 更新 localStorage
        localStorage.setItem('token', token);
        if (nextRefreshToken) {
          localStorage.setItem('refreshToken', nextRefreshToken);
        }

        // 触发刷新成功事件
        dispatchTokenRefreshed({
          token,
          refreshToken: nextRefreshToken,
        });

        return true;
      } catch {
        // 清除状态
        localStorage.removeItem('token');
        localStorage.removeItem('refreshToken');

        // 触发刷新失败事件
        dispatchTokenExpired();

        return false;
      } finally {
        this.refreshPromise = null;
      }
    })();

    return this.refreshPromise;
  }
}

export const apiService = new ApiService();
export { TOKEN_EVENTS };
export default apiService;
