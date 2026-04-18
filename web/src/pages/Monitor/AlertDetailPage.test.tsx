import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import AlertDetailPage from './AlertDetailPage';

const mockApi = vi.hoisted(() => ({
  monitoring: {
    getAlertList: vi.fn(),
  },
  aiAlertHeal: {
    listByAlert: vi.fn(),
    listGlobalPendingApprovals: vi.fn(),
    retryJob: vi.fn(),
  },
}));

vi.mock('../../api', () => ({
  Api: mockApi,
}));

describe('AlertDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.monitoring.getAlertList.mockResolvedValue({
      data: {
        list: [
          {
            id: 'alert-1',
            title: 'CPU usage too high',
            severity: 'critical',
            source: 'cpu_usage',
            status: 'resolved',
            createdAt: '2026-04-18T00:00:00Z',
          },
        ],
        total: 1,
      },
    });
    mockApi.aiAlertHeal.listByAlert.mockResolvedValue({
      data: {
        list: [
          {
            id: 'job-1',
            status: 'failed_manual',
            latest_run_id: 'run-1',
            last_error: 'fix failed',
          },
        ],
        total: 1,
      },
    });
    mockApi.aiAlertHeal.listGlobalPendingApprovals.mockResolvedValue({
      data: { list: [], total: 0 },
    });
    mockApi.aiAlertHeal.retryJob.mockResolvedValue({ data: { ok: true } });
  });

  it('disables 手动重试 when alert is resolved', async () => {
    render(
      <MemoryRouter initialEntries={['/monitor/alerts/alert-1']}>
        <Routes>
          <Route path="/monitor/alerts/:alertId" element={<AlertDetailPage />} />
        </Routes>
      </MemoryRouter>,
    );

    const retryButton = (await screen.findAllByRole('button', { name: '手动重试' })).at(-1);
    expect(retryButton).toBeTruthy();
    expect(retryButton!).toBeDisabled();
  });

  it('disables 手动重试 when latest job is pending', async () => {
    mockApi.monitoring.getAlertList.mockResolvedValueOnce({
      data: {
        list: [
          {
            id: 'alert-1',
            title: 'CPU usage too high',
            severity: 'critical',
            source: 'cpu_usage',
            status: 'firing',
            createdAt: '2026-04-18T00:00:00Z',
          },
        ],
        total: 1,
      },
    });
    mockApi.aiAlertHeal.listByAlert.mockResolvedValueOnce({
      data: {
        list: [
          {
            id: 'job-1',
            status: 'pending',
            latest_run_id: 'run-1',
            last_error: '',
            updated_at: '2026-04-18T00:10:00Z',
          },
        ],
        total: 1,
      },
    });

    render(
      <MemoryRouter initialEntries={['/monitor/alerts/alert-1']}>
        <Routes>
          <Route path="/monitor/alerts/:alertId" element={<AlertDetailPage />} />
        </Routes>
      </MemoryRouter>,
    );

    const retryButton = (await screen.findAllByRole('button', { name: '手动重试' })).at(-1);
    expect(retryButton).toBeTruthy();
    expect(retryButton!).toBeDisabled();
  });

  it('shows matching global approvals inline when 查看审批 is clicked', async () => {
    mockApi.monitoring.getAlertList.mockResolvedValueOnce({
      data: {
        list: [
          {
            id: 'alert-1',
            title: 'CPU usage too high',
            severity: 'critical',
            source: 'cpu_usage',
            status: 'firing',
            createdAt: '2026-04-18T00:00:00Z',
          },
        ],
        total: 1,
      },
    });
    mockApi.aiAlertHeal.listByAlert.mockResolvedValueOnce({
      data: {
        list: [
          {
            id: 'job-1',
            status: 'waiting_approval',
            latest_run_id: 'run-approval-1',
            last_error: '',
            updated_at: '2026-04-18T00:10:00Z',
          },
        ],
        total: 1,
      },
    });
    mockApi.aiAlertHeal.listGlobalPendingApprovals.mockResolvedValueOnce({
      data: {
        list: [
          {
            approval_id: 'approval-1',
            run_id: 'run-approval-1',
            tool_name: 'exec_command',
            status: 'pending',
            created_at: '2026-04-18T00:09:00Z',
          },
        ],
        total: 1,
      },
    });

    render(
      <MemoryRouter initialEntries={['/monitor/alerts/alert-1']}>
        <Routes>
          <Route path="/monitor/alerts/:alertId" element={<AlertDetailPage />} />
        </Routes>
      </MemoryRouter>,
    );

    fireEvent.click(await screen.findByRole('button', { name: '查看审批' }));

    expect(await screen.findByText('approval-1')).toBeInTheDocument();
    expect(mockApi.aiAlertHeal.listGlobalPendingApprovals).toHaveBeenCalledWith(1, 20);
  });
});
