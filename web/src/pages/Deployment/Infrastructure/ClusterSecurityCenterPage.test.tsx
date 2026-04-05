import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import ClusterSecurityCenterPage from './ClusterSecurityCenterPage';

const mockApi = vi.hoisted(() => ({
  cluster: {
    getClusterDetail: vi.fn(),
  },
  phase3: {
    listSecurityAlerts: vi.fn(),
    containAlert: vi.fn(),
  },
}));

vi.mock('../../../api', () => ({
  Api: mockApi,
}));

function renderPage(initialEntry = '/deployment/infrastructure/clusters/42/security') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/deployment/infrastructure/clusters/:id/security" element={<ClusterSecurityCenterPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('ClusterSecurityCenterPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.cluster.getClusterDetail.mockResolvedValue({
      data: {
        id: 42,
        name: 'phase3-external-cluster',
        source: 'external_managed',
      },
    });
    mockApi.phase3.listSecurityAlerts.mockResolvedValue({
      data: {
        list: [
          {
            id: 1001,
            cluster_id: 42,
            namespace: 'prod',
            workload: 'api-1',
            severity: 'high',
            dispose_status: 'pending',
          },
        ],
        total: 1,
      },
    });
    mockApi.phase3.containAlert.mockResolvedValue({
      data: {
        state: 'completed',
        code: 'success',
        ui_state: 'warning',
        result: {
          event_id: 1001,
          mode: 'suggest_only',
          status: 'suggested',
        },
      },
    });
  });

  afterEach(() => {
    cleanup();
  });

  it('renders external_managed containment as manual suggestion', async () => {
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText('external_managed')).toBeInTheDocument();
    expect(await screen.findByText('api-1')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Contain' }));

    await waitFor(() => {
      expect(mockApi.phase3.containAlert).toHaveBeenCalledWith(42, 1001);
    });

    expect(await screen.findByTestId('contain-mode')).toHaveTextContent('suggest_only');
    expect(await screen.findByTestId('manual-suggestion')).toHaveTextContent('manual suggestion');
  });
});
