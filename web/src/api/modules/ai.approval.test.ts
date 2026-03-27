import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiRequestError } from '../api';
import { aiApi, isApprovalConflictError, resolveApprovalTicket } from './ai';
import apiService from '../api';

describe('aiApi approval contract', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('uses /ai/approvals/:id/submit for decisions', async () => {
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({
      success: true,
      data: { approval_id: 'approval-1', status: 'approved', message: 'ok' },
    });
    const submitApproval = (aiApi as any).submitApproval;

    expect(typeof submitApproval).toBe('function');

    const response = await submitApproval('approval-1', { approved: true, comment: 'ship it' });

    expect(postMock).toHaveBeenCalledWith(
      '/ai/approvals/approval-1/submit',
      {
        approved: true,
        comment: 'ship it',
      },
      expect.objectContaining({
        headers: expect.objectContaining({
          'Idempotency-Key': expect.any(String),
        }),
      }),
    );
    expect(response.data).toEqual({
      approval_id: 'approval-1',
      status: 'approved',
      message: 'ok',
    });
  });

  it('does not expose legacy approval wrapper helpers', () => {
    expect((aiApi as any).confirmApproval).toBeUndefined();
    expect((aiApi as any).listApprovals).toBeUndefined();
    expect((aiApi as any).decideChainApproval).toBeUndefined();
    expect((aiApi as any).decideChainApprovalStream).toBeUndefined();
  });

  it('uses /ai/approvals/:id/retry-resume for retryable resume requeues', async () => {
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({
      success: true,
      data: { approval_id: 'approval-1', status: 'requeued', message: 'ok' },
    });

    const response = await (aiApi as any).retryResumeApproval('approval-1', { trigger_id: 'trigger-1' });

    expect(postMock).toHaveBeenCalledWith(
      '/ai/approvals/approval-1/retry-resume',
      { trigger_id: 'trigger-1' },
    );
    expect(response.data).toEqual({
      approval_id: 'approval-1',
      status: 'requeued',
      message: 'ok',
    });
  });

  it('treats already-processed submit errors as approval conflicts', () => {
    expect(isApprovalConflictError(new ApiRequestError('already approved', 409))).toBe(true);
    expect(isApprovalConflictError(new ApiRequestError('already approved', 400))).toBe(true);
    expect(isApprovalConflictError(new ApiRequestError('bad request', 400))).toBe(false);
  });

  it('refreshes approval state with getApproval and falls back to pending approvals', async () => {
    const getApprovalMock = vi.spyOn(aiApi, 'getApproval').mockRejectedValueOnce(new ApiRequestError('conflict', 409));
    const listPendingMock = vi.spyOn(aiApi, 'listPendingApprovals').mockResolvedValueOnce({
      success: true,
      data: [
        {
          approval_id: 'approval-1',
          checkpoint_id: 'checkpoint-1',
          session_id: 'session-1',
          run_id: 'run-1',
          tool_name: 'kubectl_apply',
          tool_call_id: 'call-1',
          arguments_json: '{}',
          preview_json: '{}',
          status: 'pending',
        },
      ],
    } as any);

    const ticket = await resolveApprovalTicket('approval-1');

    expect(getApprovalMock).toHaveBeenCalledWith('approval-1');
    expect(listPendingMock).toHaveBeenCalled();
    expect(ticket).toEqual(expect.objectContaining({
      approval_id: 'approval-1',
      tool_call_id: 'call-1',
      status: 'pending',
    }));
  });

  it('resolveApprovalTicket matches pending records by tool_call_id fallback', async () => {
    vi.spyOn(aiApi, 'getApproval').mockRejectedValueOnce(new ApiRequestError('not found', 404));
    vi.spyOn(aiApi, 'listPendingApprovals').mockResolvedValueOnce({
      success: true,
      data: [
        {
          approval_id: 'approval-99',
          checkpoint_id: 'checkpoint-1',
          session_id: 'session-1',
          run_id: 'run-1',
          tool_name: 'host_exec',
          tool_call_id: 'call_7dd6640999ee4875836a0256',
          arguments_json: '{}',
          preview_json: '{}',
          status: 'pending',
        },
      ],
    } as any);

    const ticket = await resolveApprovalTicket('call_7dd6640999ee4875836a0256');
    expect(ticket?.approval_id).toBe('approval-99');
  });

  it('submitApproval retries with resolved approval_id when id is a call_id alias', async () => {
    const submitApproval = (aiApi as any).submitApproval;
    const postMock = vi.spyOn(apiService, 'post')
      .mockRejectedValueOnce(new ApiRequestError('approval "call_7dd6640999ee4875836a0256" not found', 404))
      .mockResolvedValueOnce({
        success: true,
        data: { approval_id: 'approval-99', status: 'approved', message: 'ok' },
      });
    vi.spyOn(aiApi, 'getApproval').mockRejectedValueOnce(new ApiRequestError('not found', 404));
    vi.spyOn(aiApi, 'listPendingApprovals').mockResolvedValueOnce({
      success: true,
      data: [
        {
          approval_id: 'approval-99',
          checkpoint_id: 'checkpoint-1',
          session_id: 'session-1',
          run_id: 'run-1',
          tool_name: 'host_exec',
          tool_call_id: 'call_7dd6640999ee4875836a0256',
          arguments_json: '{}',
          preview_json: '{}',
          status: 'pending',
        },
      ],
    } as any);

    const result = await submitApproval('call_7dd6640999ee4875836a0256', { approved: true });

    expect(postMock).toHaveBeenNthCalledWith(
      1,
      '/ai/approvals/call_7dd6640999ee4875836a0256/submit',
      { approved: true },
      expect.any(Object),
    );
    expect(postMock).toHaveBeenNthCalledWith(
      2,
      '/ai/approvals/approval-99/submit',
      { approved: true },
      expect.any(Object),
    );
    expect(result.data).toEqual(expect.objectContaining({
      approval_id: 'approval-99',
      status: 'approved',
    }));
  });

  it('submitApproval retries with resolved approval_id on business not-found code 2005', async () => {
    const submitApproval = (aiApi as any).submitApproval;
    const postMock = vi.spyOn(apiService, 'post')
      .mockRejectedValueOnce(new ApiRequestError('approval "call_409a77c6a4be45d6bc324218" not found', 200, 2005))
      .mockResolvedValueOnce({
        success: true,
        data: { approval_id: 'approval-100', status: 'approved', message: 'ok' },
      });
    vi.spyOn(aiApi, 'getApproval').mockRejectedValueOnce(new ApiRequestError('approval "call_409a77c6a4be45d6bc324218" not found', 200, 2005));
    vi.spyOn(aiApi, 'listPendingApprovals').mockResolvedValueOnce({
      success: true,
      data: [
        {
          approval_id: 'approval-100',
          checkpoint_id: 'checkpoint-1',
          session_id: 'session-1',
          run_id: 'run-1',
          tool_name: 'host_exec',
          tool_call_id: 'call_409a77c6a4be45d6bc324218',
          arguments_json: '{}',
          preview_json: '{}',
          status: 'pending',
        },
      ],
    } as any);

    const result = await submitApproval('call_409a77c6a4be45d6bc324218', { approved: true });

    expect(postMock).toHaveBeenNthCalledWith(
      1,
      '/ai/approvals/call_409a77c6a4be45d6bc324218/submit',
      { approved: true },
      expect.any(Object),
    );
    expect(postMock).toHaveBeenNthCalledWith(
      2,
      '/ai/approvals/approval-100/submit',
      { approved: true },
      expect.any(Object),
    );
    expect(result.data).toEqual(expect.objectContaining({
      approval_id: 'approval-100',
      status: 'approved',
    }));
  });
});
