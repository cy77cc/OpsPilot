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
      data: { list: [], total: 0 },
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
});
