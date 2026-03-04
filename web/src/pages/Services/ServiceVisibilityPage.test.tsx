import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import ServiceVisibilityPage from './ServiceVisibilityPage';

const mockGetDetail = vi.fn();
const mockUpdateVisibility = vi.fn();
const mockUpdateGrantedTeams = vi.fn();

vi.mock('../../api', () => ({
  Api: {
    services: {
      getDetail: (...args: unknown[]) => mockGetDetail(...args),
      updateVisibility: (...args: unknown[]) => mockUpdateVisibility(...args),
      updateGrantedTeams: (...args: unknown[]) => mockUpdateGrantedTeams(...args),
    },
  },
}));

describe('ServiceVisibilityPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetDetail.mockResolvedValue({ data: { visibility: 'team', grantedTeams: [2, 3] } });
    mockUpdateVisibility.mockResolvedValue({ data: {} });
    mockUpdateGrantedTeams.mockResolvedValue({ data: {} });
  });

  it('loads and submits visibility settings', async () => {
    render(
      <MemoryRouter initialEntries={['/services/8/visibility']}>
        <Routes>
          <Route path="/services/:id/visibility" element={<ServiceVisibilityPage />} />
        </Routes>
      </MemoryRouter>
    );

    await screen.findByText('服务可见性设置');
    fireEvent.change(screen.getByPlaceholderText('例如: 2,3,5'), { target: { value: '7,8' } });
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => {
      expect(mockUpdateVisibility).toHaveBeenCalled();
      expect(mockUpdateGrantedTeams).toHaveBeenCalledWith('8', [7, 8]);
    });
  });
});
