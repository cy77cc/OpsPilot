import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import AlertDetailPage from './AlertDetailPage';

const mockApi = vi.hoisted(() => ({
  monitoring: {
    getAlertList: vi.fn(),
  },
  aiAlertHeal: {
    listByAlert: vi.fn(),
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

    const retryButton = await screen.findByRole('button', { name: '手动重试' });
    expect(retryButton).toBeDisabled();
  });
});
