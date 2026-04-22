import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ProjectSwitcher from './ProjectSwitcher';

const mockApi = vi.hoisted(() => ({
  projects: {
    list: vi.fn(),
  },
}));

vi.mock('../../api', () => ({
  Api: mockApi,
}));

describe('ProjectSwitcher', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    mockApi.projects.list.mockResolvedValue({
      data: {
        list: [
          { id: 'platform-id', name: 'Platform', key: 'platform', ownerUserId: 1, status: 'active' },
          { id: 'ops-id', name: 'Operations', key: 'ops', ownerUserId: 1, status: 'active' },
        ],
      },
    });
  });

  it('persists project selection through the shared scope key', async () => {
    render(<ProjectSwitcher />);

    await waitFor(() => {
      expect(mockApi.projects.list).toHaveBeenCalledTimes(1);
    });

    await waitFor(() => {
      expect(JSON.parse(localStorage.getItem('opspilot.scope') || '{}')).toEqual({ projectId: 'platform-id' });
    });

    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(await screen.findByText('Operations'));

    expect(JSON.parse(localStorage.getItem('opspilot.scope') || '{}')).toEqual({ projectId: 'ops-id' });
  });
});
