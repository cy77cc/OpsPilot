import apiService from '../api';
import type { ApiResponse } from '../api';
import { normalizeClusterOperationResponse, type ClusterOperationResponse } from './cluster';

export type Phase3UIState = 'success' | 'warning' | 'pending' | 'error';

export interface Phase3SecurityAlert {
  id: number;
  cluster_id: number;
  namespace?: string;
  workload?: string;
  severity?: string;
  source?: string;
  dispose_status?: string;
  created_at?: string;
}

export interface Phase3OperationResponse<T = unknown> extends ClusterOperationResponse<T> {
  ui_state: Phase3UIState;
}

export function normalizePhase3OperationResponse<T = unknown>(payload: unknown): Phase3OperationResponse<T> {
  const normalized = normalizeClusterOperationResponse<T>(payload);
  const result = (normalized.result ?? null) as Record<string, unknown> | null;
  const mode = typeof result?.mode === 'string' ? result.mode : undefined;

  let uiState: Phase3UIState = 'error';
  if (mode === 'suggest_only') {
    uiState = 'warning';
  } else if (normalized.state === 'completed') {
    uiState = 'success';
  } else if (normalized.state === 'approval_required') {
    uiState = 'pending';
  }

  return {
    ...normalized,
    ui_state: uiState,
  };
}

export const clusterPhase3Api = {
  listSecurityAlerts(clusterId: number): Promise<ApiResponse<{ list: Phase3SecurityAlert[]; total: number }>> {
    return apiService.get(`/clusters/${clusterId}/security/alerts`);
  },

  containAlert(clusterId: number, alertId: number, approvalToken?: string): Promise<ApiResponse<Phase3OperationResponse>> {
    return apiService
      .post(`/clusters/${clusterId}/security/alerts/${alertId}/contain`, {
        ...(approvalToken ? { approval_token: approvalToken } : {}),
      })
      .then((res: ApiResponse<unknown>) => ({
        ...res,
        data: normalizePhase3OperationResponse(res.data),
      }));
  },
};

export default clusterPhase3Api;
