import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';

export interface Project {
  id: string;
  key: string;
  name: string;
  ownerUserId: number;
  status: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface ProjectMember {
  id: string;
  projectId: string;
  userId: string;
  role: string;
  createdAt?: string;
}

export const projectApi = {
  async list(): Promise<ApiResponse<PaginatedResponse<Project>>> {
    const response = await apiService.get<Project[]>('/projects');
    const list = (response.data || []).map((item: any) => ({
      id: String(item.id),
      key: item.key,
      name: item.name,
      ownerUserId: item.owner_user_id || item.ownerUserId || 0,
      status: item.status,
      createdAt: item.created_at || item.createdAt,
      updatedAt: item.updated_at || item.updatedAt,
    }));
    return { ...response, data: { list, total: response.total || list.length } };
  },

  async create(data: { key: string; name: string }): Promise<ApiResponse<Project>> {
    return apiService.post('/projects', data);
  },

  async getDetail(id: string): Promise<ApiResponse<Project>> {
    return apiService.get(`/projects/${id}`);
  },

  async update(id: string, data: { name?: string; status?: string }): Promise<ApiResponse<Project>> {
    return apiService.put(`/projects/${id}`, data);
  },

  async listMembers(projectId: string): Promise<ApiResponse<PaginatedResponse<ProjectMember>>> {
    const response = await apiService.get<ProjectMember[]>(`/projects/${projectId}/members`);
    const list = (response.data || []).map((item: any) => ({
      id: String(item.id),
      projectId: String(item.project_id || item.projectId),
      userId: String(item.user_id || item.userId),
      role: item.role,
      createdAt: item.created_at || item.createdAt,
    }));
    return { ...response, data: { list, total: response.total || list.length } };
  },

  async addMember(projectId: string, data: { userId: number; role: string }): Promise<ApiResponse<void>> {
    return apiService.post(`/projects/${projectId}/members`, data);
  },

  async removeMember(projectId: string, userId: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/projects/${projectId}/members/${userId}`);
  },
};
