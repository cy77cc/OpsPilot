import { afterEach, describe, expect, it, vi } from 'vitest';
import apiService from '../api';
import { aiAlertHealApi } from './aiAlertHeal';
import { normalizeHealStatus } from '../../pages/Monitor/monitorAlertHealStatus';

describe('normalizeHealStatus', () => {
  it('maps waiting_approval to 待人工 + 转人工审批', () => {
    expect(normalizeHealStatus('waiting_approval')).toEqual({
      processing: '待人工',
      healing: '转人工审批',
      processingColor: 'orange',
      healingColor: 'volcano',
    });
  });
});

describe('aiAlertHealApi', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('calls alert-heal list endpoint with alert_id params', async () => {
    const getMock = vi.spyOn(apiService, 'get').mockResolvedValue({
      success: true,
      data: { list: [], total: 0 },
    });

    await aiAlertHealApi.listByAlert('alert-1');

    expect(getMock).toHaveBeenCalledWith('/ai/alert-heal/jobs', {
      params: { alert_id: 'alert-1' },
    });
  });

  it('calls global pending approvals endpoint with pagination params', async () => {
    const getMock = vi.spyOn(apiService, 'get').mockResolvedValue({
      success: true,
      data: { list: [], total: 0 },
    });

    await aiAlertHealApi.listGlobalPendingApprovals(2, 50);

    expect(getMock).toHaveBeenCalledWith('/ai/approvals/pending/global', {
      params: { page: 2, page_size: 50 },
    });
  });

  it('calls retry endpoint for a specific alert-heal job', async () => {
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({
      success: true,
      data: { ok: true },
    });

    await aiAlertHealApi.retryJob('job-1');

    expect(postMock).toHaveBeenCalledWith('/ai/alert-heal/jobs/job-1/retry', {});
  });
});
