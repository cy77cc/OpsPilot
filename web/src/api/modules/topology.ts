import apiService from '../api';
import type { ApiResponse } from '../api';

export const topologyApi = {
  async getServiceTopology(id: string): Promise<ApiResponse<any>> {
    return apiService.get(`/topology/services/${id}`);
  },
  async getHostServices(id: string): Promise<ApiResponse<any[]>> {
    return apiService.get(`/topology/hosts/${id}/services`);
  },
  async getClusterServices(id: string): Promise<ApiResponse<any[]>> {
    return apiService.get(`/topology/clusters/${id}/services`);
  },
};

