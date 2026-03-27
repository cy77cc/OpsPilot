import apiService from '../api';
import type { ApiResponse } from '../api';

export interface AutomationInventory {
  id: string;
  name: string;
  hostsJson: string;
}

export interface AutomationPlaybook {
  id: string;
  name: string;
  contentYml: string;
  riskLevel: string;
}

export interface AutomationRun {
  id: string;
  action: string;
  status: string;
  resultJson: string;
}

export const automationApi = {
  async listInventories(): Promise<ApiResponse<AutomationInventory[]>> {
    return apiService.get('/automation/inventories');
  },
  async createInventory(payload: { name: string; hostsJson: string }): Promise<ApiResponse<AutomationInventory>> {
    return apiService.post('/automation/inventories', {
      name: payload.name,
      hosts_json: payload.hostsJson,
    });
  },
  async listPlaybooks(): Promise<ApiResponse<AutomationPlaybook[]>> {
    return apiService.get('/automation/playbooks');
  },
  async createPlaybook(payload: { name: string; contentYml: string; riskLevel?: string }): Promise<ApiResponse<AutomationPlaybook>> {
    return apiService.post('/automation/playbooks', {
      name: payload.name,
      content_yml: payload.contentYml,
      risk_level: payload.riskLevel,
    });
  },
  async previewRun(payload: { action: string; params?: Record<string, any> }): Promise<ApiResponse<any>> {
    return apiService.post('/automation/runs/preview', payload);
  },
  async executeRun(payload: { approval_token: string }): Promise<ApiResponse<any>> {
    return apiService.post('/automation/runs/execute', payload);
  },
  async getRun(id: string): Promise<ApiResponse<AutomationRun>> {
    return apiService.get(`/automation/runs/${id}`);
  },
  async getRunLogs(id: string): Promise<ApiResponse<any[]>> {
    return apiService.get(`/automation/runs/${id}/logs`);
  },
};
