import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import RoutingConfigPage from './RoutingConfigPage';

const mockApi = vi.hoisted(() => ({
  monitoring: {
    getSeverityRoutes: vi.fn(),
  },
}));

vi.mock('../../api', () => ({
  Api: mockApi,
}));

describe('RoutingConfigPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.monitoring.getSeverityRoutes.mockResolvedValue({
      data: {
        list: [{ id: 1, scope: 'global', severity: 'critical', channel_ids_json: '[1001]', enabled: true }],
        total: 1,
      },
    });
  });

  it('renders severity routes', async () => {
    render(<RoutingConfigPage />);

    expect(await screen.findByText('critical')).toBeInTheDocument();
    expect(screen.getByText('[1001]')).toBeInTheDocument();
  });
});
