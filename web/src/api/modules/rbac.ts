import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';

// 用户数据结构
export interface User {
  id: string;
  username: string;
  name: string;
  email: string;
  roles: string[];
  status: string;
  createdAt: string;
  updatedAt: string;
}

// 角色数据结构
export interface Role {
  id: string;
  name: string;
  code?: string;
  description: string;
  permissions: string[];
  createdAt: string;
  updatedAt: string;
}

// 权限数据结构
export interface Permission {
  id: string;
  name: string;
  code: string;
  description: string;
  category: string;
  createdAt: string;
}

// 用户列表请求参数
export interface UserListParams {
  page?: number;
  pageSize?: number;
  status?: string;
  role?: string;
}

// 角色列表请求参数
export interface RoleListParams {
  page?: number;
  pageSize?: number;
}

// 权限列表请求参数
export interface PermissionListParams {
  page?: number;
  pageSize?: number;
  category?: string;
}

// 用户创建/更新参数
export interface UserCreateParams {
  username: string;
  name: string;
  email: string;
  password: string;
  roles: string[];
  status: string;
}

export interface UserUpdateParams {
  name?: string;
  email?: string;
  roles?: string[];
  status?: string;
  password?: string;
}

// 角色创建/更新参数
export interface RoleCreateParams {
  name: string;
  description: string;
  permissions: string[];
}

export interface RoleUpdateParams {
  name?: string;
  description?: string;
  permissions?: string[];
}

export interface MigrationEventParams {
  eventType: 'legacy_redirect' | 'governance_task';
  fromPath?: string;
  toPath?: string;
  action?: string;
  status?: string;
  durationMs?: number;
}

// RBAC管理API
export const rbacApi = {
  async getMyPermissions(): Promise<ApiResponse<string[]>> {
    return apiService.get('/rbac/me/permissions');
  },

  // 用户管理
  async getUserList(params?: UserListParams): Promise<ApiResponse<PaginatedResponse<User>>> {
    return apiService.get('/rbac/users', { params });
  },

  async getUserDetail(id: string): Promise<ApiResponse<User>> {
    return apiService.get(`/rbac/users/${id}`);
  },

  async createUser(data: UserCreateParams): Promise<ApiResponse<User>> {
    return apiService.post('/rbac/users', data);
  },

  async updateUser(id: string, data: UserUpdateParams): Promise<ApiResponse<User>> {
    return apiService.put(`/rbac/users/${id}`, data);
  },

  async deleteUser(id: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/rbac/users/${id}`);
  },

  // 角色管理
  async getRoleList(params?: RoleListParams): Promise<ApiResponse<PaginatedResponse<Role>>> {
    return apiService.get('/rbac/roles', { params });
  },

  async getRoleDetail(id: string): Promise<ApiResponse<Role>> {
    return apiService.get(`/rbac/roles/${id}`);
  },

  async createRole(data: RoleCreateParams): Promise<ApiResponse<Role>> {
    return apiService.post('/rbac/roles', data);
  },

  async updateRole(id: string, data: RoleUpdateParams): Promise<ApiResponse<Role>> {
    return apiService.put(`/rbac/roles/${id}`, data);
  },

  async deleteRole(id: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/rbac/roles/${id}`);
  },

  // 权限管理
  async getPermissionList(params?: PermissionListParams): Promise<ApiResponse<PaginatedResponse<Permission>>> {
    return apiService.get('/rbac/permissions', { params });
  },

  async getPermissionDetail(id: string): Promise<ApiResponse<Permission>> {
    return apiService.get(`/rbac/permissions/${id}`);
  },

  // 权限验证
  async checkPermission(resource: string, action: string): Promise<ApiResponse<{ hasPermission: boolean }>> {
    return apiService.post('/rbac/check', { resource, action });
  },

  async recordMigrationEvent(data: MigrationEventParams): Promise<ApiResponse<{ accepted: boolean }>> {
    return apiService.post('/rbac/migration/events', data);
  },
};
