import { afterEach, describe, expect, it, vi } from 'vitest';
import { aiApi } from './ai';
import apiService from '../api';

describe('aiApi approval contract', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('uses /ai/approvals/:id/submit for decisions', async () => {
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({ success: true, data: {} });
    const submitApproval = (aiApi as any).submitApproval;

    expect(typeof submitApproval).toBe('function');

    await submitApproval('approval-1', { approved: true, comment: 'ship it' });

    expect(postMock).toHaveBeenCalledWith('/ai/approvals/approval-1/submit', {
      approved: true,
      comment: 'ship it',
    });
  });

  it('does not use legacy /confirm or /chains/.../decision path', async () => {
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({ success: true, data: {} });

    await aiApi.confirmApproval('approval-2', false);

    expect(postMock).toHaveBeenCalledWith('/ai/approvals/approval-2/submit', {
      approved: false,
    });
    expect(postMock.mock.calls.some(([path]) => String(path).includes('/confirm'))).toBe(false);
    expect(postMock.mock.calls.some(([path]) => String(path).includes('/decision'))).toBe(false);
    expect((aiApi as any).decideChainApproval).toBeUndefined();
    expect((aiApi as any).decideChainApprovalStream).toBeUndefined();
  });
});
