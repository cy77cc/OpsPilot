import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { message } from 'antd';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import RulesConfigPage from './RulesConfigPage';

const mockApi = vi.hoisted(() => ({
  monitoring: {
    getEffectiveRules: vi.fn(),
    createAlertRule: vi.fn(),
    updateAlertRule: vi.fn(),
    deleteAlertRule: vi.fn(),
  },
}));

vi.mock('../../api', () => ({
  Api: mockApi,
}));

describe('RulesConfigPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(message, 'success').mockImplementation(() => undefined as any);
    vi.spyOn(message, 'error').mockImplementation(() => undefined as any);
    vi.spyOn(message, 'warning').mockImplementation(() => undefined as any);
    mockApi.monitoring.getEffectiveRules.mockResolvedValue({
      data: {
        list: [{ id: 1, name: 'CPU High', severity: 'warning', threshold: 90, scope: 'global', inherit_key: 'cpu-high' }],
        total: 1,
      },
    });
    mockApi.monitoring.createAlertRule.mockResolvedValue({ data: { id: '2' } });
    mockApi.monitoring.updateAlertRule.mockResolvedValue({ data: { id: '1' } });
    mockApi.monitoring.deleteAlertRule.mockResolvedValue({ data: { ok: true } });
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('renders effective rules list', async () => {
    render(<RulesConfigPage />);

    expect(await screen.findByText('CPU High')).toBeInTheDocument();
    expect(screen.getByText('warning')).toBeInTheDocument();
  });

  it('creates a rule and reloads list', async () => {
    render(<RulesConfigPage />);
    await screen.findByText('CPU High');

    fireEvent.click(screen.getByRole('button', { name: '新增规则' }));
    const dialog = await screen.findByRole('dialog', { name: '新增规则' });
    fireEvent.change(within(dialog).getByLabelText('名称'), { target: { value: 'Memory High' } });
    fireEvent.change(within(dialog).getByLabelText('指标'), { target: { value: 'memory_usage' } });
    fireEvent.change(within(dialog).getByRole('spinbutton', { name: '阈值' }), { target: { value: '80' } });
    fireEvent.click(within(dialog).getByRole('button', { name: /^(OK|确定)$/ }));

    await waitFor(() => {
      expect(mockApi.monitoring.createAlertRule).toHaveBeenCalledWith({
        name: 'Memory High',
        metric: 'memory_usage',
        threshold: 80,
        severity: 'warning',
      });
      expect(mockApi.monitoring.getEffectiveRules).toHaveBeenCalledTimes(2);
    });
  });

  it('updates a rule and reloads list', async () => {
    render(<RulesConfigPage />);
    await screen.findByText('CPU High');

    fireEvent.click(screen.getByRole('button', { name: '编辑' }));
    const dialog = await screen.findByRole('dialog', { name: '编辑规则' });
    fireEvent.change(within(dialog).getByLabelText('名称'), { target: { value: 'CPU Critical' } });
    fireEvent.change(within(dialog).getByRole('spinbutton', { name: '阈值' }), { target: { value: '95' } });
    fireEvent.click(within(dialog).getByRole('button', { name: /^(OK|确定)$/ }));

    await waitFor(() => {
      expect(mockApi.monitoring.updateAlertRule).toHaveBeenCalledWith('1', {
        name: 'CPU Critical',
        threshold: 95,
        severity: 'warning',
      });
      expect(mockApi.monitoring.getEffectiveRules).toHaveBeenCalledTimes(2);
    });
  });

  it('deletes a rule and reloads list', async () => {
    render(<RulesConfigPage />);
    await screen.findByText('CPU High');

    fireEvent.click(screen.getByRole('button', { name: '删除' }));
    const confirm = await screen.findByRole('tooltip', { name: /确定删除此规则/ });
    fireEvent.click(within(confirm).getByRole('button', { name: /^(OK|确定)$/ }));

    await waitFor(() => {
      expect(mockApi.monitoring.deleteAlertRule).toHaveBeenCalledWith('1');
      expect(mockApi.monitoring.getEffectiveRules).toHaveBeenCalledTimes(2);
    });
  });

  it('shows error and does not reload on mutation failure', async () => {
    mockApi.monitoring.deleteAlertRule.mockRejectedValueOnce(new Error('delete failed'));
    render(<RulesConfigPage />);
    await screen.findByText('CPU High');

    fireEvent.click(screen.getByRole('button', { name: '删除' }));
    const confirm = await screen.findByRole('tooltip', { name: /确定删除此规则/ });
    fireEvent.click(within(confirm).getByRole('button', { name: /^(OK|确定)$/ }));

    await waitFor(() => {
      expect(message.error).toHaveBeenCalledWith('规则删除失败');
    });
    expect(mockApi.monitoring.getEffectiveRules).toHaveBeenCalledTimes(1);
  });

  it('shows success then warning when reload fails after mutation success', async () => {
    mockApi.monitoring.getEffectiveRules
      .mockResolvedValueOnce({
        data: {
          list: [{ id: 1, name: 'CPU High', severity: 'warning', threshold: 90, scope: 'global', inherit_key: 'cpu-high' }],
          total: 1,
        },
      })
      .mockRejectedValueOnce(new Error('reload failed'));

    render(<RulesConfigPage />);
    await screen.findByText('CPU High');

    fireEvent.click(screen.getByRole('button', { name: '删除' }));
    const confirm = await screen.findByRole('tooltip', { name: /确定删除此规则/ });
    fireEvent.click(within(confirm).getByRole('button', { name: /^(OK|确定)$/ }));

    await waitFor(() => {
      expect(message.success).toHaveBeenCalledWith('规则删除成功');
      expect(message.warning).toHaveBeenCalledWith('规则删除成功，但列表刷新失败');
    });
    expect(message.error).not.toHaveBeenCalledWith('规则删除失败');
  });
});
