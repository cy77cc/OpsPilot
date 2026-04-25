import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import ClusterWorkloadsPage from './ClusterWorkloadsPage';

const mockApi = vi.hoisted(() => ({
  cluster: {
    getClusterDetail: vi.fn(),
    getNamespaces: vi.fn(),
    getDeployments: vi.fn(),
    getStatefulSets: vi.fn(),
    getPods: vi.fn(),
  },
}));

vi.mock('../../../api', () => ({
  Api: mockApi,
}));

function renderPage(initialEntry = '/resources/clusters/42/workloads') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/resources/clusters/:id/workloads" element={<ClusterWorkloadsPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('ClusterWorkloadsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.cluster.getClusterDetail.mockResolvedValue({
      data: { id: 42, name: 'phase1-cluster' },
    });
    mockApi.cluster.getNamespaces.mockResolvedValue({
      data: { list: [{ name: 'default', status: 'Active', created_at: '2026-04-04T00:00:00Z' }] },
    });
    mockApi.cluster.getDeployments.mockResolvedValue({
      data: { list: [{ name: 'web', replicas: 3, ready: 3, age: '1d' }] },
    });
    mockApi.cluster.getStatefulSets.mockResolvedValue({
      data: { list: [] },
    });
    mockApi.cluster.getPods.mockResolvedValue({
      data: { list: [] },
    });
  });

  afterEach(() => {
    cleanup();
  });

  it('renders namespace filter toolbar and workload tabs', async () => {
    renderPage();

    expect(await screen.findByLabelText('Namespace')).toBeInTheDocument();
    expect(screen.getByText('Deployments')).toBeInTheDocument();
    expect(screen.getByText('StatefulSets')).toBeInTheDocument();
    expect(screen.getByText('Pods')).toBeInTheDocument();
    expect(screen.getByText('web')).toBeInTheDocument();
  });
});
