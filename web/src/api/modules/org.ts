import apiService from '../api';
import type { ApiResponse } from '../api';

export interface Department {
  id: string;
  name: string;
  parentId?: string;
  order?: number;
  children?: Department[];
  createdAt?: string;
  updatedAt?: string;
}

export interface Member {
  id: string;
  username: string;
  name: string;
  email: string;
  departmentId: string;
  status: string;
  joinedAt: string;
}

export interface CreateDepartmentParams {
  name: string;
  parentId?: string;
  order?: number;
}

export interface UpdateDepartmentParams {
  name?: string;
  parentId?: string;
  order?: number;
}

export interface TransferMemberParams {
  memberIds: string[];
  targetDepartmentId: string;
}

export const orgApi = {
  async getDepartmentTree(): Promise<ApiResponse<Department[]>> {
    return apiService.get<Department[]>('/org/departments/tree');
  },

  async createDepartment(data: CreateDepartmentParams): Promise<ApiResponse<Department>> {
    return apiService.post<Department>('/org/departments', data);
  },

  async updateDepartment(id: string, data: UpdateDepartmentParams): Promise<ApiResponse<Department>> {
    return apiService.put<Department>(`/org/departments/${id}`, data);
  },

  async deleteDepartment(id: string): Promise<ApiResponse<void>> {
    return apiService.delete<void>(`/org/departments/${id}`);
  },

  async getDepartmentMembers(departmentId: string): Promise<ApiResponse<Member[]>> {
    return apiService.get<Member[]>(`/org/departments/${departmentId}/members`);
  },

  async transferMember(data: TransferMemberParams): Promise<ApiResponse<void>> {
    return apiService.post<void>('/org/members/transfer', data);
  },
};
