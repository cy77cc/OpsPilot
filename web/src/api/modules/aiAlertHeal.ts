import type { ApiResponse } from '../api';
import { apiService } from '../api';

export interface AlertHealJob {
  id: string;
  event_id: string;
  status: string;
  decision: string;
  retry_count: number;
  max_retry: number;
  last_error: string;
  latest_run_id: string;
  updated_at: string;
}

export interface AlertHealApprovalTask {
  approval_id: string;
  run_id: string;
  tool_name: string;
  status: string;
  created_at?: string;
}

export const aiAlertHealApi = {
  listByAlert(alertId: string): Promise<ApiResponse<{ list: AlertHealJob[]; total: number }>> {
    return apiService.get<{ list: AlertHealJob[]; total: number }>('/ai/alert-heal/jobs', {
      params: { alert_id: alertId },
    });
  },
  getJob(jobId: string): Promise<ApiResponse<AlertHealJob>> {
    return apiService.get<AlertHealJob>(`/ai/alert-heal/jobs/${jobId}`);
  },
  retryJob(jobId: string): Promise<ApiResponse<unknown>> {
    return apiService.post(`/ai/alert-heal/jobs/${jobId}/retry`, {});
  },
  listGlobalPendingApprovals(page = 1, pageSize = 20): Promise<ApiResponse<{ list: AlertHealApprovalTask[]; total: number }>> {
    return apiService.get<{ list: AlertHealApprovalTask[]; total: number }>('/ai/approvals/pending/global', {
      params: { page, page_size: pageSize },
    });
  },
};

export default aiAlertHealApi;
