import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';

export interface Task {
  id: string;
  name: string;
  type: string;
  status: string;
  schedule: string;
  command?: string;
  hostIds?: string;
  description?: string;
  timeout?: number;
  priority?: number;
  lastRun?: string;
  nextRun?: string;
  duration?: number;
  createdAt: string;
  updatedAt?: string;
}

export interface TaskExecution {
  id: string;
  jobId: string;
  hostId?: string;
  hostIp?: string;
  status: string;
  exitCode?: number;
  output?: string;
  startTime?: string;
  endTime?: string;
  createdAt?: string;
}

export interface TaskLog {
  id: string;
  taskId: string;
  executionId?: string;
  timestamp: string;
  level: string;
  message: string;
}

export interface TaskListParams {
  page?: number;
  pageSize?: number;
}

export interface TaskCreateParams {
  name: string;
  type: string;
  schedule: string;
  command?: string;
  hostIds?: string;
  description?: string;
  timeout?: number;
  priority?: number;
}

export interface TaskUpdateParams {
  name?: string;
  type?: string;
  schedule?: string;
  status?: string;
  command?: string;
  hostIds?: string;
  description?: string;
  timeout?: number;
  priority?: number;
}

export interface TaskLogParams {
  page?: number;
  pageSize?: number;
  level?: string;
}

const normalizeTask = (item: any): Task => ({
  id: String(item.id),
  name: item.name || '',
  type: item.type || '',
  status: item.status || 'pending',
  schedule: item.cron || item.schedule || '',
  command: item.command || '',
  hostIds: item.host_ids || '',
  description: item.description || '',
  timeout: item.timeout || 0,
  priority: item.priority || 0,
  createdAt: item.created_at || item.createdAt || '',
  updatedAt: item.updated_at || item.updatedAt || '',
  lastRun: item.last_run || '',
  nextRun: item.next_run || '',
  duration: item.duration || 0,
});

const normalizeExecution = (item: any): TaskExecution => ({
  id: String(item.id),
  jobId: String(item.job_id || item.jobId || ''),
  hostId: item.host_id ? String(item.host_id) : '',
  hostIp: item.host_ip || '',
  status: item.status || 'pending',
  exitCode: item.exit_code,
  output: item.output || '',
  startTime: item.start_time || '',
  endTime: item.end_time || '',
  createdAt: item.created_at || '',
});

const normalizeLog = (item: any): TaskLog => ({
  id: String(item.id),
  taskId: String(item.job_id || item.taskId || ''),
  executionId: item.execution_id ? String(item.execution_id) : '',
  timestamp: item.created_at || item.timestamp || '',
  level: item.level || 'info',
  message: item.message || '',
});

export const taskApi = {
  async getTaskList(params?: TaskListParams): Promise<ApiResponse<PaginatedResponse<Task>>> {
    const response = await apiService.get<any[]>('/jobs', {
      params: {
        page: params?.page,
        page_size: params?.pageSize,
      },
    });

    return {
      ...response,
      data: {
        list: (response.data || []).map(normalizeTask),
        total: response.total || 0,
      },
    };
  },

  async getTaskDetail(id: string): Promise<ApiResponse<Task>> {
    const response = await apiService.get<any>(`/jobs/${id}`);
    return {
      ...response,
      data: normalizeTask(response.data),
    };
  },

  async createTask(data: TaskCreateParams): Promise<ApiResponse<Task>> {
    const response = await apiService.post<any>('/jobs', {
      name: data.name,
      type: data.type,
      command: data.command || `echo running ${data.name}`,
      host_ids: data.hostIds || '1',
      cron: data.schedule,
      status: 'pending',
      timeout: data.timeout || 300,
      priority: data.priority || 0,
      description: data.description || '',
    });

    return {
      ...response,
      data: normalizeTask(response.data),
    };
  },

  async updateTask(id: string, data: TaskUpdateParams): Promise<ApiResponse<Task>> {
    const current = await this.getTaskDetail(id);
    const merged = {
      ...current.data,
      ...data,
    };

    const response = await apiService.put<any>(`/jobs/${id}`, {
      name: merged.name,
      type: merged.type,
      command: merged.command,
      host_ids: merged.hostIds,
      cron: merged.schedule,
      status: merged.status,
      timeout: merged.timeout,
      priority: merged.priority,
      description: merged.description,
    });

    return {
      ...response,
      data: normalizeTask(response.data),
    };
  },

  async deleteTask(id: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/jobs/${id}`);
  },

  async startTask(id: string): Promise<ApiResponse<void>> {
    return apiService.post(`/jobs/${id}/start`);
  },

  async stopTask(id: string): Promise<ApiResponse<void>> {
    return apiService.post(`/jobs/${id}/stop`);
  },

  async getTaskExecutions(id: string, params?: TaskLogParams): Promise<ApiResponse<PaginatedResponse<TaskExecution>>> {
    const response = await apiService.get<any[]>(`/jobs/${id}/executions`, {
      params: {
        page: params?.page,
        page_size: params?.pageSize,
      },
    });

    return {
      ...response,
      data: {
        list: (response.data || []).map(normalizeExecution),
        total: response.total || 0,
      },
    };
  },

  async getTaskLogs(id: string, params?: TaskLogParams): Promise<ApiResponse<PaginatedResponse<TaskLog>>> {
    const response = await apiService.get<any[]>(`/jobs/${id}/logs`, {
      params: {
        page: params?.page,
        page_size: params?.pageSize,
        level: params?.level,
      },
    });

    return {
      ...response,
      data: {
        list: (response.data || []).map(normalizeLog),
        total: response.total || 0,
      },
    };
  },
};
