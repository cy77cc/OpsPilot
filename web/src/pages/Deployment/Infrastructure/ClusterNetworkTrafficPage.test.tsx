import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import ClusterNetworkTrafficPage from './ClusterNetworkTrafficPage';

const mockApi = vi.hoisted(() => ({
  cluster: {
    getClusterDetail: vi.fn(),
    getNamespaces: vi.fn(),
    getServices: vi.fn(),
    getIngresses: vi.fn(),
  },
}));

vi.mock('../../../api', () => ({
  Api: mockApi,
}));

function renderPage(initialEntry = '/resources/clusters/42/network') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/resources/clusters/:id/network" element={<ClusterNetworkTrafficPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('ClusterNetworkTrafficPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.cluster.getClusterDetail.mockResolvedValue({
      data: { id: 42, name: 'phase1-cluster' },
    });
    mockApi.cluster.getNamespaces.mockResolvedValue({
      data: { list: [{ name: 'default', status: 'Active', created_at: '2026-04-04T00:00:00Z' }] },
    });
    mockApi.cluster.getServices.mockResolvedValue({
      data: { list: [{ name: 'web', type: 'ClusterIP', cluster_ip: '10.96.0.20', age: '1d' }] },
    });
    mockApi.cluster.getIngresses.mockResolvedValue({
      data: { list: [] },
    });
  });

  afterEach(() => {
    cleanup();
  });

  it('renders network route visibility and service/ingress sections', async () => {
    renderPage();

    expect(await screen.findByText('网络与流量')).toBeInTheDocument();
    expect(screen.getByText('Gateway API')).toBeInTheDocument();
    expect(screen.getByText('Services')).toBeInTheDocument();
    expect(screen.getByText('Ingresses (兼容)')).toBeInTheDocument();
    expect(screen.getByText('web')).toBeInTheDocument();
  });
});
