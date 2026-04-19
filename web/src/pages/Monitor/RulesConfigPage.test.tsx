import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import RulesConfigPage from './RulesConfigPage';

const mockApi = vi.hoisted(() => ({
  monitoring: {
    getEffectiveRules: vi.fn(),
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
  });

  it('renders effective rules list', async () => {
    render(<RulesConfigPage />);

    expect(await screen.findByText('CPU High')).toBeInTheDocument();
    expect(screen.getByText('warning')).toBeInTheDocument();
  });
});
