import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';

export interface ConfigApp {
  id: string;
  appId: string;
  name: string;
  description?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface Config {
  id: string;
  appId: string;
  key: string;
  value: string;
  version: number;
  status: string;
  env: string;
  description?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface ConfigVersion {
  id: string;
  configId: string;
  appId: string;
  env: string;
  key: string;
  value: string;
  version: number;
  operator: string;
  action: string;
  createdAt: string;
  description?: string;
}

export interface ConfigListParams {
  page?: number;
  pageSize?: number;
  env?: string;
  appId?: string;
}

const normalizeApp = (item: any): ConfigApp => ({
  id: String(item.id),
  appId: item.app_id || '',
  name: item.name || '',
  description: item.description || '',
  createdAt: item.created_at || '',
  updatedAt: item.updated_at || '',
});

const normalizeConfig = (item: any): Config => ({
  id: String(item.id),
  appId: item.app_id || '',
  key: item.key || '',
  value: item.value || '',
  version: Number(item.version || 1),
  status: item.status || 'active',
  env: item.env || 'dev',
  description: item.description || '',
  createdAt: item.created_at || '',
  updatedAt: item.updated_at || '',
});

const normalizeHistory = (item: any): ConfigVersion => ({
  id: String(item.id),
  configId: String(item.config_id),
  appId: item.app_id || '',
  env: item.env || '',
  key: item.key || '',
  value: item.value || '',
  version: Number(item.version || 1),
  operator: item.operator || 'system',
  action: item.action || 'update',
  createdAt: item.created_at || '',
  description: item.description || '',
});

export const configApi = {
  async getAppList(params?: { page?: number; pageSize?: number }): Promise<ApiResponse<PaginatedResponse<ConfigApp>>> {
    const response = await apiService.get<any[]>('/apps', {
      params: {
        page: params?.page,
        page_size: params?.pageSize,
      },
    });

    return {
      ...response,
      data: {
        list: (response.data || []).map(normalizeApp),
        total: response.total || 0,
      },
    };
  },

  async createApp(data: { app_id: string; name: string; description?: string }): Promise<ApiResponse<ConfigApp>> {
    const response = await apiService.post<any>('/apps', data);
    return {
      ...response,
      data: normalizeApp(response.data),
    };
  },

  async getConfigList(params?: ConfigListParams): Promise<ApiResponse<PaginatedResponse<Config>>> {
    const response = await apiService.get<any[]>('/configs', {
      params: {
        page: params?.page,
        page_size: params?.pageSize,
        app_id: params?.appId,
        env: params?.env,
      },
    });

    return {
      ...response,
      data: {
        list: (response.data || []).map(normalizeConfig),
        total: response.total || 0,
      },
    };
  },

  async getConfigDetail(id: string): Promise<ApiResponse<Config>> {
    const response = await apiService.get<any>(`/configs/${id}`);
    return {
      ...response,
      data: normalizeConfig(response.data),
    };
  },

  async getConfigByKey(params: { appId: string; env: string; key: string }): Promise<ApiResponse<Config>> {
    const response = await apiService.get<any>('/configs/by-key', {
      params: {
        app_id: params.appId,
        env: params.env,
        key: params.key,
      },
    });

    return {
      ...response,
      data: normalizeConfig(response.data),
    };
  },

  async createConfig(data: Partial<Config>): Promise<ApiResponse<Config>> {
    const payload = {
      app_id: data.appId,
      key: data.key,
      value: data.value,
      env: data.env,
      status: data.status || 'active',
      description: data.description || '',
    };
    const response = await apiService.post<any>('/configs', payload);
    return {
      ...response,
      data: normalizeConfig(response.data),
    };
  },

  async updateConfig(id: string, data: Partial<Config>): Promise<ApiResponse<Config>> {
    const payload = {
      app_id: data.appId,
      key: data.key,
      value: data.value,
      env: data.env,
      status: data.status || 'active',
      description: data.description || '',
    };
    const response = await apiService.put<any>(`/configs/${id}`, payload);
    return {
      ...response,
      data: normalizeConfig(response.data),
    };
  },

  async deleteConfig(id: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/configs/${id}`);
  },

  async getConfigVersions(id: string, params?: { page?: number; pageSize?: number }): Promise<ApiResponse<PaginatedResponse<ConfigVersion>>> {
    const response = await apiService.get<any[]>(`/configs/${id}/history`, {
      params: {
        page: params?.page,
        page_size: params?.pageSize,
      },
    });

    return {
      ...response,
      data: {
        list: (response.data || []).map(normalizeHistory),
        total: response.total || 0,
      },
    };
  },
};
