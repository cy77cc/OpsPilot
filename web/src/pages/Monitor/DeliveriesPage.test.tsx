import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import DeliveriesPage from './DeliveriesPage';

const mockApi = vi.hoisted(() => ({
  monitoring: {
    listAlertDeliveries: vi.fn(),
  },
}));

vi.mock('../../api', () => ({
  Api: mockApi,
}));

describe('DeliveriesPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.monitoring.listAlertDeliveries.mockResolvedValue({
      data: {
        list: [
          {
            id: '1',
            channelType: 'webhook',
            target: 'https://example.com/hook',
            status: 'sent',
            deliveredAt: '2026-04-19T00:00:00Z',
          },
        ],
        total: 1,
      },
    });
  });

  it('renders delivery records', async () => {
    render(<DeliveriesPage />);

    expect(await screen.findByText('webhook')).toBeInTheDocument();
    expect(screen.getByText('https://example.com/hook')).toBeInTheDocument();
  });
});
