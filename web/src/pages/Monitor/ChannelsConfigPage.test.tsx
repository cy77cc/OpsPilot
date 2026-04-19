import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { message, Modal } from 'antd';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import ChannelsConfigPage from './ChannelsConfigPage';

const mockApi = vi.hoisted(() => ({
  monitoring: {
    listAlertChannels: vi.fn(),
    createAlertChannel: vi.fn(),
    updateAlertChannel: vi.fn(),
    deleteAlertChannel: vi.fn(),
    testAlertChannel: vi.fn(),
  },
}));

vi.mock('../../api', () => ({
  Api: mockApi,
}));

describe('ChannelsConfigPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(message, 'success').mockImplementation(() => undefined as any);
    vi.spyOn(message, 'error').mockImplementation(() => undefined as any);
    vi.spyOn(Modal, 'error').mockImplementation(() => ({ destroy: vi.fn(), update: vi.fn() }) as any);
    mockApi.monitoring.listAlertChannels.mockResolvedValue({
      data: {
        list: [
          {
            id: 1001,
            name: 'Ops Webhook',
            type: 'webhook',
            provider: 'webhook',
            target: 'https://example.com/hook',
            enabled: true,
            config_json: '{}',
          },
        ],
      },
    });
    mockApi.monitoring.createAlertChannel.mockResolvedValue({ data: { id: 1002 } });
    mockApi.monitoring.updateAlertChannel.mockResolvedValue({ data: { id: 1001 } });
    mockApi.monitoring.deleteAlertChannel.mockResolvedValue({ data: { deleted: true } });
    mockApi.monitoring.testAlertChannel.mockResolvedValue({
      data: { status: 'sent' },
    });
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('calls test channel API on test send', async () => {
    render(<ChannelsConfigPage />);
    await screen.findByText('Ops Webhook');

    fireEvent.click(await screen.findByRole('button', { name: '测试发送' }));

    await waitFor(() => {
      expect(mockApi.monitoring.testAlertChannel).toHaveBeenCalled();
    });
  });

  it('creates a channel and reloads list', async () => {
    render(<ChannelsConfigPage />);
    await screen.findByText('Ops Webhook');

    fireEvent.click(screen.getByRole('button', { name: '新增渠道' }));
    const dialog = await screen.findByRole('dialog', { name: '新增渠道' });
    fireEvent.change(within(dialog).getByLabelText('名称'), { target: { value: 'Ops Email' } });
    fireEvent.change(within(dialog).getByLabelText('Provider'), { target: { value: 'email' } });
    fireEvent.change(within(dialog).getByLabelText('目标地址'), { target: { value: 'ops@example.com' } });
    fireEvent.change(within(dialog).getByLabelText('配置 JSON'), { target: { value: '{"from":"ops@example.com"}' } });
    fireEvent.click(within(dialog).getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => {
      expect(mockApi.monitoring.createAlertChannel).toHaveBeenCalledWith({
        name: 'Ops Email',
        provider: 'email',
        target: 'ops@example.com',
        configJson: '{"from":"ops@example.com"}',
      });
      expect(mockApi.monitoring.listAlertChannels).toHaveBeenCalledTimes(2);
    });
  });

  it('updates a channel and reloads list', async () => {
    render(<ChannelsConfigPage />);
    await screen.findByText('Ops Webhook');

    fireEvent.click(screen.getByRole('button', { name: '编辑' }));
    const dialog = await screen.findByRole('dialog', { name: '编辑渠道' });
    fireEvent.change(within(dialog).getByLabelText('名称'), { target: { value: 'Ops Webhook Updated' } });
    fireEvent.change(within(dialog).getByLabelText('目标地址'), { target: { value: 'https://example.com/v2/hook' } });
    fireEvent.click(within(dialog).getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => {
      expect(mockApi.monitoring.updateAlertChannel).toHaveBeenCalledWith('1001', {
        name: 'Ops Webhook Updated',
        provider: 'webhook',
        target: 'https://example.com/v2/hook',
        configJson: '{}',
      });
      expect(mockApi.monitoring.listAlertChannels).toHaveBeenCalledTimes(2);
    });
  });

  it('deletes a channel and reloads list', async () => {
    render(<ChannelsConfigPage />);
    await screen.findByText('Ops Webhook');

    fireEvent.click(screen.getByRole('button', { name: '删除' }));
    const confirm = await screen.findByRole('tooltip', { name: /确定删除此渠道/ });
    fireEvent.click(within(confirm).getByRole('button', { name: /^(OK|确定)$/ }));

    await waitFor(() => {
      expect(mockApi.monitoring.deleteAlertChannel).toHaveBeenCalledWith('1001');
      expect(mockApi.monitoring.listAlertChannels).toHaveBeenCalledTimes(2);
    });
  });

  it('shows blockers on 409 delete response', async () => {
    mockApi.monitoring.deleteAlertChannel.mockRejectedValueOnce({
      code: 409,
      data: {
        blockers: [
          { type: 'severity_routes', count: 2 },
          { type: 'rule_bindings', count: 1 },
        ],
      },
    });
    render(<ChannelsConfigPage />);
    await screen.findByText('Ops Webhook');

    fireEvent.click(screen.getByRole('button', { name: '删除' }));
    const confirm = await screen.findByRole('tooltip', { name: /确定删除此渠道/ });
    fireEvent.click(within(confirm).getByRole('button', { name: /^(OK|确定)$/ }));

    await waitFor(() => {
      expect(Modal.error).toHaveBeenCalled();
      expect(mockApi.monitoring.listAlertChannels).toHaveBeenCalledTimes(1);
    });
  });
});
