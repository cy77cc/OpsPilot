import apiService from '../../../api/api';
import type { ApiResponse } from '../../../api/api';

export interface AILLMProvider {
  id: number;
  name: string;
  provider: string;
  model: string;
  base_url: string;
  api_key_masked?: string;
  temperature: number;
  thinking: boolean;
  is_default: boolean;
  is_enabled: boolean;
  sort_order: number;
  created_at?: string;
  updated_at?: string;
}

export interface AILLMProviderCreatePayload {
  name: string;
  provider: string;
  model: string;
  base_url: string;
  api_key: string;
  temperature: number;
  thinking: boolean;
  is_default: boolean;
  is_enabled: boolean;
  sort_order: number;
}

export interface AILLMProviderUpdatePayload {
  name?: string;
  provider?: string;
  model?: string;
  base_url?: string;
  api_key?: string;
  temperature?: number;
  thinking?: boolean;
  is_default?: boolean;
  is_enabled?: boolean;
  sort_order?: number;
}

export interface AILLMProviderImportPayload {
  replace_all: boolean;
  providers: AILLMProviderCreatePayload[];
}

export interface AILLMProviderImportPreview {
  providers: AILLMProvider[];
}

export interface AILLMProviderImportResult {
  created: number;
  updated: number;
  providers: AILLMProvider[];
}

export const adminModelApi = {
  async listAdminModels(): Promise<ApiResponse<{ list: AILLMProvider[]; total: number }>> {
    return apiService.get('/admin/ai/models');
  },

  async getAdminModel(id: number): Promise<ApiResponse<AILLMProvider>> {
    return apiService.get(`/admin/ai/models/${id}`);
  },

  async createAdminModel(payload: AILLMProviderCreatePayload): Promise<ApiResponse<AILLMProvider>> {
    return apiService.post('/admin/ai/models', payload);
  },

  async updateAdminModel(id: number, payload: AILLMProviderUpdatePayload): Promise<ApiResponse<AILLMProvider>> {
    return apiService.put(`/admin/ai/models/${id}`, payload);
  },

  async setAdminDefaultModel(id: number): Promise<ApiResponse<null>> {
    return apiService.put(`/admin/ai/models/${id}/default`);
  },

  async deleteAdminModel(id: number): Promise<ApiResponse<null>> {
    return apiService.delete(`/admin/ai/models/${id}`);
  },

  async previewAdminModelImport(payload: AILLMProviderImportPayload): Promise<ApiResponse<AILLMProviderImportPreview>> {
    return apiService.post('/admin/ai/models/import/preview', payload);
  },

  async importAdminModels(payload: AILLMProviderImportPayload): Promise<ApiResponse<AILLMProviderImportResult>> {
    return apiService.post('/admin/ai/models/import', payload);
  },
};