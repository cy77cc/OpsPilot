import { afterEach, describe, expect, it, vi } from 'vitest';
import apiService from '../api';
import { monitoringApi } from './monitoring';

describe('monitoringApi config endpoints', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('calls effective rules endpoint', async () => {
    const getMock = vi.spyOn(apiService, 'get').mockResolvedValue({
      success: true,
      data: { list: [], total: 0 },
    } as any);

    await monitoringApi.getEffectiveRules({ projectId: '42' });

    expect(getMock).toHaveBeenCalledWith('/alert-rules/effective', {
      params: { project_id: '42', page: undefined, page_size: undefined },
    });
  });

  it('calls channel test endpoint', async () => {
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({
      success: true,
      data: { status: 'sent' },
    } as any);

    await monitoringApi.testAlertChannel({ provider: 'webhook', target: 'https://example.com', configJson: '{}' });

    expect(postMock).toHaveBeenCalledWith('/alert-channels/test', {
      provider: 'webhook',
      target: 'https://example.com',
      config_json: '{}',
    });
  });
});
