import { apiService, generateIdempotencyKey, isApprovalNotFoundError, type ApprovalTicket, type ApiResponse, type RetryResumeApprovalPayload, type RetryResumeApprovalResult, type SubmitApprovalOptions, type SubmitApprovalPayload, type SubmitApprovalResult } from './shared';

export async function listPendingApprovals(): Promise<ApiResponse<ApprovalTicket[]>> {
  return apiService.get('/ai/approvals/pending');
}

export async function getApproval(id: string): Promise<ApiResponse<ApprovalTicket>> {
  return apiService.get(`/ai/approvals/${id}`);
}

export async function submitApproval(id: string, payload: SubmitApprovalPayload, options?: SubmitApprovalOptions): Promise<ApiResponse<SubmitApprovalResult>> {
  const idempotencyKey = options?.idempotencyKey || generateIdempotencyKey();
  const requestConfig = {
    headers: {
      'Idempotency-Key': idempotencyKey,
    },
  };
  try {
    return await apiService.post(`/ai/approvals/${id}/submit`, payload, requestConfig);
  } catch (error) {
    if (!isApprovalNotFoundError(error)) {throw error;}
    const aliasTicket = await resolveApprovalTicket(id);
    const canonicalID = aliasTicket?.approval_id;
    if (!canonicalID || canonicalID === id) {throw error;}
    return apiService.post(`/ai/approvals/${canonicalID}/submit`, payload, requestConfig);
  }
}

export async function retryResumeApproval(id: string, payload: RetryResumeApprovalPayload): Promise<ApiResponse<RetryResumeApprovalResult>> {
  return apiService.post(`/ai/approvals/${id}/retry-resume`, payload);
}

export async function resolveApprovalTicket(approvalId: string): Promise<ApprovalTicket | null> {
  if (!approvalId) {return null;}
  try {
    const response = await getApproval(approvalId);
    return response.data || null;
  } catch {
    try {
      const response = await listPendingApprovals();
      return response.data?.find((ticket) =>
        ticket.approval_id === approvalId ||
        ticket.tool_call_id === approvalId ||
        String(ticket.id ?? '') === approvalId
      ) || null;
    } catch {
      return null;
    }
  }
}

export { isApprovalConflictError } from './shared';

export const approvalApi = { listPendingApprovals, getApproval, submitApproval, retryResumeApproval };
