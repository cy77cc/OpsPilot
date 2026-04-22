import { StrictMode } from 'react';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { message } from 'antd';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import RoutingConfigPage from './RoutingConfigPage';
import { scopeStore } from '../../app/scope/scopeStore';

const mockApi = vi.hoisted(() => ({
  monitoring: {
    getSeverityRoutes: vi.fn(),
    createSeverityRoute: vi.fn(),
    updateSeverityRouteByID: vi.fn(),
    deleteSeverityRoute: vi.fn(),
  },
}));

vi.mock('../../api', () => ({
  Api: mockApi,
}));

describe('RoutingConfigPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    scopeStore.clearScope();
    vi.spyOn(message, 'success').mockImplementation(() => undefined as any);
    vi.spyOn(message, 'error').mockImplementation(() => undefined as any);
    mockApi.monitoring.getSeverityRoutes.mockResolvedValue({
      data: {
        list: [{ id: 1, scope: 'global', severity: 'critical', channel_ids_json: '[1001]', enabled: true }],
        total: 1,
      },
    });
    mockApi.monitoring.createSeverityRoute.mockResolvedValue({ data: { id: 2 } });
    mockApi.monitoring.updateSeverityRouteByID.mockResolvedValue({ data: { id: 1 } });
    mockApi.monitoring.deleteSeverityRoute.mockResolvedValue({ data: { deleted: true } });
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('renders severity routes', async () => {
    render(<RoutingConfigPage />);

    expect(await screen.findByText('critical')).toBeInTheDocument();
    expect(screen.getByText('[1001]')).toBeInTheDocument();
  });

  it('loads global scope routes in StrictMode without getting stuck', async () => {
    render(
      <StrictMode>
        <RoutingConfigPage />
      </StrictMode>
    );

    expect(await screen.findByText('critical')).toBeInTheDocument();
  });

  it('passes project scope to route list request after switching scope', async () => {
    render(<RoutingConfigPage />);
    await screen.findByText('critical');

    fireEvent.click(screen.getByRole('radio', { name: '项目' }));
    fireEvent.change(screen.getByPlaceholderText('项目ID'), { target: { value: '42' } });

    await waitFor(() => {
      expect(mockApi.monitoring.getSeverityRoutes).toHaveBeenCalledWith({ projectId: '42' });
    });
  });

  it('clears the rendered rows when switching to project scope without project id', async () => {
    render(<RoutingConfigPage />);
    await screen.findByText('critical');

    fireEvent.click(screen.getByRole('radio', { name: '项目' }));

    await waitFor(() => {
      expect(screen.queryByText('critical')).not.toBeInTheDocument();
      expect(mockApi.monitoring.getSeverityRoutes).toHaveBeenCalledTimes(1);
    });
  });

  it('creates a route and reloads list with scope-aware payload', async () => {
    render(<RoutingConfigPage />);
    await screen.findByText('critical');

    fireEvent.click(screen.getByRole('radio', { name: '项目' }));
    fireEvent.change(screen.getByPlaceholderText('项目ID'), { target: { value: '42' } });
    await waitFor(() => {
      expect(mockApi.monitoring.getSeverityRoutes).toHaveBeenCalledWith({ projectId: '42' });
    });

    fireEvent.click(screen.getByRole('button', { name: '新增路由' }));
    const dialog = await screen.findByRole('dialog', { name: '新增路由' });
    fireEvent.change(within(dialog).getByLabelText('作用域'), { target: { value: 'project' } });
    fireEvent.change(within(dialog).getByLabelText('级别'), { target: { value: 'warning' } });
    fireEvent.change(within(dialog).getByLabelText('渠道ID'), { target: { value: '1001,1002' } });
    fireEvent.click(within(dialog).getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => {
      expect(mockApi.monitoring.createSeverityRoute).toHaveBeenCalledWith({
        projectId: '42',
        scope: 'project',
        severity: 'warning',
        channelIds: ['1001', '1002'],
        enabled: true,
      });
      expect(mockApi.monitoring.getSeverityRoutes).toHaveBeenCalledTimes(3);
    });
  });

  it('updates a route and reloads list with mapped payload', async () => {
    render(<RoutingConfigPage />);
    await screen.findByText('critical');

    fireEvent.click(screen.getByRole('button', { name: '编辑' }));
    const dialog = await screen.findByRole('dialog', { name: '编辑路由' });
    fireEvent.change(within(dialog).getByLabelText('级别'), { target: { value: 'info' } });
    fireEvent.change(within(dialog).getByLabelText('渠道ID'), { target: { value: '1003' } });
    fireEvent.click(within(dialog).getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => {
      expect(mockApi.monitoring.updateSeverityRouteByID).toHaveBeenCalledWith('1', {
        projectId: undefined,
        scope: 'global',
        severity: 'info',
        channelIds: ['1003'],
        enabled: true,
      });
      expect(mockApi.monitoring.getSeverityRoutes).toHaveBeenCalledTimes(2);
    });
  });

  it('deletes a route and reloads list with scope-aware params', async () => {
    scopeStore.setScope({ projectId: '42' });
    mockApi.monitoring.getSeverityRoutes.mockResolvedValueOnce({
      data: {
        list: [{ id: 1, scope: 'project', severity: 'critical', channel_ids_json: '[1001]', enabled: true }],
        total: 1,
      },
    });
    render(<RoutingConfigPage />);
    await screen.findByText('critical');

    fireEvent.click(screen.getByRole('button', { name: '删除' }));
    const confirm = await screen.findByRole('tooltip', { name: /确定删除此路由/ });
    fireEvent.click(within(confirm).getByRole('button', { name: /^(OK|确定)$/ }));

    await waitFor(() => {
      expect(mockApi.monitoring.deleteSeverityRoute).toHaveBeenCalledWith('1', '42');
      expect(mockApi.monitoring.getSeverityRoutes).toHaveBeenCalledTimes(2);
    });
  });

  it('blocks project-scoped route delete when project id is missing', async () => {
    mockApi.monitoring.getSeverityRoutes.mockResolvedValueOnce({
      data: {
        list: [{ id: 1, scope: 'project', severity: 'critical', channel_ids_json: '[1001]', enabled: true }],
        total: 1,
      },
    });
    render(<RoutingConfigPage />);
    await screen.findByText('critical');

    fireEvent.click(screen.getByRole('button', { name: '删除' }));
    const confirm = await screen.findByRole('tooltip', { name: /确定删除此路由/ });
    fireEvent.click(within(confirm).getByRole('button', { name: /^(OK|确定)$/ }));

    await waitFor(() => {
      expect(message.error).toHaveBeenCalledWith('项目作用域操作需要先选择项目ID');
      expect(mockApi.monitoring.deleteSeverityRoute).not.toHaveBeenCalled();
    });
  });
});
