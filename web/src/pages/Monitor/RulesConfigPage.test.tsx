import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
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
  });

  it('renders effective rules list', async () => {
    render(<RulesConfigPage />);

    expect(await screen.findByText('CPU High')).toBeInTheDocument();
    expect(screen.getByText('warning')).toBeInTheDocument();
  });

  it('creates a rule and reloads list', async () => {
    render(<RulesConfigPage />);
    await screen.findByText('CPU High');

    fireEvent.click(screen.getAllByRole('button', { name: '新增规则' })[0]);
    fireEvent.change(screen.getByLabelText('名称'), { target: { value: 'Memory High' } });
    fireEvent.change(screen.getByLabelText('指标'), { target: { value: 'memory_usage' } });
    fireEvent.change(screen.getByRole('spinbutton', { name: '阈值' }), { target: { value: '80' } });
    fireEvent.click(screen.getByRole('button', { name: 'OK' }));

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
    await screen.findAllByText('CPU High');

    fireEvent.click(screen.getByRole('button', { name: '编辑' }));
    fireEvent.change(screen.getByLabelText('名称'), { target: { value: 'CPU Critical' } });
    fireEvent.change(screen.getByRole('spinbutton', { name: '阈值' }), { target: { value: '95' } });
    fireEvent.click(screen.getByRole('button', { name: 'OK' }));

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
    await screen.findAllByText('CPU High');

    fireEvent.click(screen.getByRole('button', { name: '删除' }));
    fireEvent.click(screen.getByRole('button', { name: 'OK' }));

    await waitFor(() => {
      expect(mockApi.monitoring.deleteAlertRule).toHaveBeenCalledWith('1');
      expect(mockApi.monitoring.getEffectiveRules).toHaveBeenCalledTimes(2);
    });
  });
});
