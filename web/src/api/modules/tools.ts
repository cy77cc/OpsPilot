import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';

export interface ToolItem {
  id: string;
  name: string;
  type: string;
  path: string;
  description?: string;
  enabled: boolean;
  createdAt?: string;
}

export interface ToolExecution {
  id: string;
  toolId: string;
  toolName: string;
  status: string;
  output?: string;
  error?: string;
  startTime?: string;
  endTime?: string;
}

export const toolApi = {
  async getToolList(params?: { page?: number; pageSize?: number }): Promise<ApiResponse<PaginatedResponse<ToolItem>>> {
    const response = await apiService.get<any[]>('/tools', {
      params: { page: params?.page, page_size: params?.pageSize },
    });
    const list = (response.data || []).map((item: any) => ({
      id: String(item.id),
      name: item.name,
      type: item.type,
      path: item.path,
      description: item.description,
      enabled: item.enabled,
      createdAt: item.created_at || item.createdAt,
    }));
    return {
      ...response,
      data: {
        list,
        total: response.total || 0,
      },
    };
  },

  async executeTool(id: string, params = ''): Promise<ApiResponse<ToolExecution>> {
    const response = await apiService.post<any>(`/tools/${id}/execute`, { params });
    return {
      ...response,
      data: {
        id: String(response.data?.id),
        toolId: String(response.data?.tool_id),
        toolName: response.data?.tool_name || '',
        status: response.data?.status || '',
        output: response.data?.output,
        error: response.data?.error,
        startTime: response.data?.start_time,
        endTime: response.data?.end_time,
      },
    };
  },

  async listExecutions(params?: { toolId?: string; page?: number; pageSize?: number }): Promise<ApiResponse<PaginatedResponse<ToolExecution>>> {
    const response = await apiService.get<any[]>('/executions', {
      params: {
        tool_id: params?.toolId,
        page: params?.page,
        page_size: params?.pageSize,
      },
    });
    const list = (response.data || []).map((item: any) => ({
      id: String(item.id),
      toolId: String(item.tool_id),
      toolName: item.tool_name || '',
      status: item.status,
      output: item.output,
      error: item.error,
      startTime: item.start_time,
      endTime: item.end_time,
    }));
    return {
      ...response,
      data: { list, total: response.total || 0 },
    };
  },
};

