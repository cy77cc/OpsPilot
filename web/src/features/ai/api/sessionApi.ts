import { apiService, normalizeSessionListResponse, normalizeSessionResponse, type AISession, type ApiResponse } from './shared';

export async function getSessions(scene?: string): Promise<ApiResponse<AISession[]>> {
  const response = await apiService.get<AISession[]>('/ai/sessions', scene ? { params: { scene } } : undefined);
  return normalizeSessionListResponse(response);
}

export async function createSession(params: { title: string; scene: string }): Promise<ApiResponse<AISession>> {
  const response = await apiService.post<AISession>('/ai/sessions', params);
  return normalizeSessionResponse(response);
}

export async function getSession(id: string): Promise<ApiResponse<AISession>> {
  const response = await apiService.get<AISession>(`/ai/sessions/${id}`);
  return normalizeSessionResponse(response);
}

export async function deleteSession(id: string): Promise<ApiResponse<void>> {
  return apiService.delete(`/ai/sessions/${id}`);
}

export const sessionApi = { getSessions, createSession, getSession, deleteSession };
