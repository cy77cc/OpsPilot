import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import AlertsPage from './AlertsPage';

const mockApi = vi.hoisted(() => ({
  monitoring: {
    getAlertList: vi.fn(),
  },
  aiAlertHeal: {
    listByAlert: vi.fn(),
  },
}));

vi.mock('../../api', () => ({
  Api: mockApi,
}));

describe('AlertsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.monitoring.getAlertList.mockResolvedValue({
      data: {
        list: [
          {
            id: 'alert-1',
            title: 'CPU usage too high',
            severity: 'critical',
            source: 'alertmanager/fp-1',
            status: 'firing',
            createdAt: '2026-04-18T00:00:00Z',
            latestHealJobId: 'job-1',
            latestHealStatus: 'waiting_approval',
            latestHealUpdatedAt: '2026-04-18T00:10:00Z',
            latestHealRunId: 'run-1',
          },
        ],
        total: 1,
      },
    });
  });

  it('renders 处理状态 and 自愈状态 columns', async () => {
    render(
      <MemoryRouter>
        <AlertsPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText('处理状态')).toBeInTheDocument();
    expect(screen.getByText('自愈状态')).toBeInTheDocument();
  });

  it('uses latest heal summary from alert list without per-row heal requests', async () => {
    render(
      <MemoryRouter>
        <AlertsPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText('转人工审批')).toBeInTheDocument();
    expect(mockApi.aiAlertHeal.listByAlert).not.toHaveBeenCalled();
  });
});
