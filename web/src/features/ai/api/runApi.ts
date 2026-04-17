import { apiService, normalizeRunResponse, type AIDiagnosisReport, type AIRun, type AIRunContent, type AIRunProjection, type AIRunProjectionQuery, type ApiResponse } from './shared';

export async function getRunStatus(runId: string): Promise<ApiResponse<AIRun>> {
  const response = await apiService.get<AIRun>(`/ai/runs/${runId}`);
  return normalizeRunResponse(response);
}

export async function getRunProjection(runId: string, query?: AIRunProjectionQuery): Promise<ApiResponse<AIRunProjection>> {
  return apiService.get(`/ai/runs/${runId}/projection`, query ? { params: query } : undefined);
}

export async function getRunContent(id: string): Promise<ApiResponse<AIRunContent>> {
  return apiService.get(`/ai/run-contents/${id}`);
}

export async function getDiagnosisReport(reportId: string): Promise<ApiResponse<AIDiagnosisReport>> {
  return apiService.get(`/ai/diagnosis/${reportId}`);
}

export function listUnsupportedMethods(): string[] {
  return [];
}

export const runApi = { getRunStatus, getRunProjection, getRunContent, getDiagnosisReport, listUnsupportedMethods };
