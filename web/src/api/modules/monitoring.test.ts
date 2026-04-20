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

  it('calls delete alert rule endpoint', async () => {
    const deleteMock = vi.spyOn(apiService, 'delete').mockResolvedValue({
      success: true,
      data: { deleted: true },
    } as any);

    await monitoringApi.deleteAlertRule('7');

    expect(deleteMock).toHaveBeenCalledWith('/alert-rules/7');
  });

  it('calls delete alert channel endpoint', async () => {
    const deleteMock = vi.spyOn(apiService, 'delete').mockResolvedValue({
      success: true,
      data: { deleted: true },
    } as any);

    await monitoringApi.deleteAlertChannel('1001');

    expect(deleteMock).toHaveBeenCalledWith('/alert-channels/1001');
  });

  it('calls alert channels list endpoint with scope params', async () => {
    const getMock = vi.spyOn(apiService, 'get').mockResolvedValue({
      success: true,
      data: { list: [], total: 0 },
    } as any);

    await monitoringApi.listAlertChannels({ projectId: '42' });

    expect(getMock).toHaveBeenCalledWith('/alert-channels', {
      params: { project_id: 42 },
    });
  });

  it('calls create severity route endpoint with mapped payload', async () => {
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({
      success: true,
      data: { id: 31 },
    } as any);

    await monitoringApi.createSeverityRoute({
      projectId: '42',
      scope: 'project',
      severity: 'critical',
      channelIds: ['1001', '1.5', 'bad', '0', '-1', '1002'],
      enabled: false,
    });

    expect(postMock).toHaveBeenCalledWith('/alert-routing/severity', {
      project_id: 42,
      scope: 'project',
      severity: 'critical',
      channel_ids: [1001, 1002],
      enabled: false,
    });
  });

  it('normalizes create alert channel project_id to positive integer', async () => {
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({
      success: true,
      data: { id: 1 },
    } as any);

    await monitoringApi.createAlertChannel({ name: 'invalid-project', projectId: 'abc' });
    await monitoringApi.createAlertChannel({ name: 'valid-project', projectId: '42' });

    expect(postMock).toHaveBeenNthCalledWith(1, '/alert-channels', {
      name: 'invalid-project',
      type: undefined,
      provider: undefined,
      target: undefined,
      enabled: undefined,
      config_json: undefined,
      project_id: undefined,
    });
    expect(postMock).toHaveBeenNthCalledWith(2, '/alert-channels', {
      name: 'valid-project',
      type: undefined,
      provider: undefined,
      target: undefined,
      enabled: undefined,
      config_json: undefined,
      project_id: 42,
    });
  });

  it('calls update alert channel endpoint with mapped payload', async () => {
    const putMock = vi.spyOn(apiService, 'put').mockResolvedValue({
      success: true,
      data: { id: 1001 },
    } as any);

    await monitoringApi.updateAlertChannel('1001', {
      name: 'ops-webhook',
      type: 'webhook',
      provider: 'webhook',
      target: 'https://example.com/hook',
      enabled: true,
      configJson: '{"a":1}',
      projectId: '42',
    });

    expect(putMock).toHaveBeenCalledWith('/alert-channels/1001', {
      name: 'ops-webhook',
      type: 'webhook',
      provider: 'webhook',
      target: 'https://example.com/hook',
      enabled: true,
      config_json: '{"a":1}',
      project_id: 42,
    });
  });

  it('calls update severity route by id endpoint with mapped payload', async () => {
    const putMock = vi.spyOn(apiService, 'put').mockResolvedValue({
      success: true,
      data: { id: 31 },
    } as any);

    await monitoringApi.updateSeverityRouteByID('31', {
      severity: 'warning',
      channelIds: ['1001'],
    });

    expect(putMock).toHaveBeenCalledWith('/alert-routing/severity/31', {
      project_id: undefined,
      scope: undefined,
      severity: 'warning',
      channel_ids: [1001],
      enabled: undefined,
    });
  });

  it('drops non-integer channel ids in update rule channels payload', async () => {
    const putMock = vi.spyOn(apiService, 'put').mockResolvedValue({
      success: true,
      data: { ok: true },
    } as any);

    await monitoringApi.updateRuleChannels('7', ['1001', '1.5', 'bad', '0', '-2']);

    expect(putMock).toHaveBeenCalledWith('/alert-rules/7/channels', {
      channel_ids: [1001],
      project_id: undefined,
    });
  });

  it('calls delete severity route endpoint with project scope params', async () => {
    const deleteMock = vi.spyOn(apiService, 'delete').mockResolvedValue({
      success: true,
      data: { deleted: true },
    } as any);

    await monitoringApi.deleteSeverityRoute('31', '42');

    expect(deleteMock).toHaveBeenCalledWith('/alert-routing/severity/31', {
      params: { project_id: 42 },
    });
  });

  it('omits invalid project id from delete severity route params', async () => {
    const deleteMock = vi.spyOn(apiService, 'delete').mockResolvedValue({
      success: true,
      data: { deleted: true },
    } as any);

    await monitoringApi.deleteSeverityRoute('31', 'bad');

    expect(deleteMock).toHaveBeenCalledWith('/alert-routing/severity/31', {
      params: { project_id: undefined },
    });
  });

  it('calls create rule-channel binding endpoint with mapped payload', async () => {
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({
      success: true,
      data: { ok: true },
    } as any);

    await monitoringApi.createRuleChannelBinding('7', {
      projectId: '42',
      channelId: '1001',
      priority: 2,
      enabled: false,
    });

    expect(postMock).toHaveBeenCalledWith('/alert-rules/7/channels', {
      project_id: 42,
      channel_id: 1001,
      priority: 2,
      enabled: false,
    });
  });

  it('omits invalid channel id in create rule-channel binding payload', async () => {
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({
      success: true,
      data: { ok: true },
    } as any);

    await monitoringApi.createRuleChannelBinding('7', {
      projectId: '42',
      channelId: 'NaN',
      priority: 2,
      enabled: true,
    });

    expect(postMock).toHaveBeenCalledWith('/alert-rules/7/channels', {
      project_id: 42,
      channel_id: undefined,
      priority: 2,
      enabled: true,
    });
  });

  it('calls update rule-channel binding endpoint with project scope payload', async () => {
    const putMock = vi.spyOn(apiService, 'put').mockResolvedValue({
      success: true,
      data: { ok: true },
    } as any);

    await monitoringApi.updateRuleChannelBinding('7', '1001', {
      projectId: '42',
      priority: 3,
      enabled: true,
    });

    expect(putMock).toHaveBeenCalledWith('/alert-rules/7/channels/1001', {
      project_id: 42,
      priority: 3,
      enabled: true,
    });
  });

  it('calls delete rule-channel binding endpoint with project scope params', async () => {
    const deleteMock = vi.spyOn(apiService, 'delete').mockResolvedValue({
      success: true,
      data: { deleted: true },
    } as any);

    await monitoringApi.deleteRuleChannelBinding('7', '1001', '42');

    expect(deleteMock).toHaveBeenCalledWith('/alert-rules/7/channels/1001', {
      params: { project_id: 42 },
    });
  });
});
