import { StrictMode } from 'react';
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
    getRuleChannels: vi.fn(),
    createRuleChannelBinding: vi.fn(),
    updateRuleChannelBinding: vi.fn(),
    deleteRuleChannelBinding: vi.fn(),
  },
}));

vi.mock('../../api', () => ({
  Api: mockApi,
}));

describe('RulesConfigPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
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
    mockApi.monitoring.getRuleChannels.mockResolvedValue({
      data: {
        list: [{ channel_id: 1001, priority: 1, enabled: true }],
      },
    });
    mockApi.monitoring.createRuleChannelBinding.mockResolvedValue({ data: { ok: true } });
    mockApi.monitoring.updateRuleChannelBinding.mockResolvedValue({ data: { ok: true } });
    mockApi.monitoring.deleteRuleChannelBinding.mockResolvedValue({ data: { ok: true } });
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

  it('loads global scope list in StrictMode without getting stuck', async () => {
    render(
      <StrictMode>
        <RulesConfigPage />
      </StrictMode>
    );

    expect(await screen.findByText('CPU High')).toBeInTheDocument();
  });

  it('passes project scope to effective-rules requests after switching scope', async () => {
    render(<RulesConfigPage />);
    await screen.findByText('CPU High');

    fireEvent.click(screen.getByRole('radio', { name: '项目' }));
    fireEvent.change(screen.getByPlaceholderText('项目ID'), { target: { value: '42' } });

    await waitFor(() => {
      expect(mockApi.monitoring.getEffectiveRules).toHaveBeenCalledWith(expect.objectContaining({ projectId: '42' }));
    });
  });

  it('keeps latest scope rows when an older request resolves late', async () => {
    const createDeferred = <T,>() => {
      let resolve!: (value: T) => void;
      const promise = new Promise<T>((res) => {
        resolve = res;
      });
      return { promise, resolve };
    };
    const globalRequest = createDeferred<any>();
    const projectRequest = createDeferred<any>();
    mockApi.monitoring.getEffectiveRules.mockReset();
    mockApi.monitoring.getEffectiveRules.mockImplementation((params?: { projectId?: string }) => {
      if (params?.projectId === '42') {
        return projectRequest.promise;
      }
      return globalRequest.promise;
    });

    render(<RulesConfigPage />);
    fireEvent.click(screen.getByRole('radio', { name: '项目' }));
    fireEvent.change(screen.getByPlaceholderText('项目ID'), { target: { value: '42' } });
    await waitFor(() => {
      expect(mockApi.monitoring.getEffectiveRules).toHaveBeenCalledTimes(2);
    });

    projectRequest.resolve({
      data: {
        list: [{ id: 2, name: 'Project Rule', severity: 'warning', threshold: 80, scope: 'project', inherit_key: 'project-rule' }],
        total: 1,
      },
    });
    expect(await screen.findByText('Project Rule')).toBeInTheDocument();

    globalRequest.resolve({
      data: {
        list: [{ id: 1, name: 'Global Rule', severity: 'warning', threshold: 90, scope: 'global', inherit_key: 'global-rule' }],
        total: 1,
      },
    });

    await waitFor(() => {
      expect(screen.queryByText('Global Rule')).not.toBeInTheDocument();
      expect(screen.getByText('Project Rule')).toBeInTheDocument();
    });
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

  it('creates a rule-channel binding and refreshes bindings', async () => {
    render(<RulesConfigPage />);
    await screen.findByText('CPU High');

    fireEvent.click(screen.getByRole('button', { name: '渠道绑定' }));
    const drawer = await screen.findByRole('dialog', { name: '规则渠道绑定' });
    fireEvent.change(within(drawer).getByLabelText('渠道ID'), { target: { value: '1002' } });
    fireEvent.change(within(drawer).getByRole('spinbutton', { name: '优先级' }), { target: { value: '5' } });
    fireEvent.click(within(drawer).getByRole('button', { name: '新增绑定' }));

    await waitFor(() => {
      expect(mockApi.monitoring.getRuleChannels).toHaveBeenCalledWith('1', { projectId: undefined });
      expect(mockApi.monitoring.createRuleChannelBinding).toHaveBeenCalledWith('1', {
        projectId: undefined,
        channelId: '1002',
        priority: 5,
        enabled: true,
      });
      expect(mockApi.monitoring.getRuleChannels).toHaveBeenCalledTimes(2);
    });
  });

  it('updates a rule-channel binding and refreshes bindings', async () => {
    render(<RulesConfigPage />);
    await screen.findByText('CPU High');

    fireEvent.click(screen.getByRole('button', { name: '渠道绑定' }));
    const drawer = await screen.findByRole('dialog', { name: '规则渠道绑定' });
    fireEvent.click(within(drawer).getByRole('button', { name: '编辑绑定' }));
    fireEvent.change(within(drawer).getByRole('spinbutton', { name: '优先级' }), { target: { value: '10' } });
    fireEvent.click(within(drawer).getByRole('button', { name: '更新绑定' }));

    await waitFor(() => {
      expect(mockApi.monitoring.updateRuleChannelBinding).toHaveBeenCalledWith('1', '1001', {
        projectId: undefined,
        priority: 10,
        enabled: true,
      });
      expect(mockApi.monitoring.getRuleChannels).toHaveBeenCalledTimes(2);
    });
  });

  it('deletes a rule-channel binding and refreshes bindings', async () => {
    render(<RulesConfigPage />);
    await screen.findByText('CPU High');

    fireEvent.click(screen.getByRole('button', { name: '渠道绑定' }));
    const drawer = await screen.findByRole('dialog', { name: '规则渠道绑定' });
    fireEvent.click(within(drawer).getByRole('button', { name: '删除绑定' }));
    const confirm = await screen.findByRole('tooltip', { name: /确定删除此绑定/ });
    fireEvent.click(within(confirm).getByRole('button', { name: /^(OK|确定)$/ }));

    await waitFor(() => {
      expect(mockApi.monitoring.deleteRuleChannelBinding).toHaveBeenCalledWith('1', '1001', undefined);
      expect(mockApi.monitoring.getRuleChannels).toHaveBeenCalledTimes(2);
    });
  });

  it('shows binding error and does not reload bindings when create binding fails', async () => {
    mockApi.monitoring.createRuleChannelBinding.mockRejectedValueOnce(new Error('create binding failed'));
    render(<RulesConfigPage />);
    await screen.findByText('CPU High');

    fireEvent.click(screen.getByRole('button', { name: '渠道绑定' }));
    const drawer = await screen.findByRole('dialog', { name: '规则渠道绑定' });
    fireEvent.change(within(drawer).getByLabelText('渠道ID'), { target: { value: '1002' } });
    fireEvent.change(within(drawer).getByRole('spinbutton', { name: '优先级' }), { target: { value: '5' } });
    fireEvent.click(within(drawer).getByRole('button', { name: '新增绑定' }));

    await waitFor(() => {
      expect(mockApi.monitoring.createRuleChannelBinding).toHaveBeenCalledWith('1', {
        projectId: undefined,
        channelId: '1002',
        priority: 5,
        enabled: true,
      });
      expect(message.error).toHaveBeenCalledWith('绑定创建失败');
    });
    expect(mockApi.monitoring.getRuleChannels).toHaveBeenCalledTimes(1);
  });
});
