import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import ClusterConfigStoragePage from './ClusterConfigStoragePage';

const mockApi = vi.hoisted(() => ({
  cluster: {
    getClusterDetail: vi.fn(),
    getNamespaces: vi.fn(),
    getConfigMaps: vi.fn(),
    getSecrets: vi.fn(),
    getPVCs: vi.fn(),
    getPVs: vi.fn(),
  },
}));

vi.mock('../../../api', () => ({
  Api: mockApi,
}));

function renderPage(initialEntry = '/resources/clusters/42/config-storage') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/resources/clusters/:id/config-storage" element={<ClusterConfigStoragePage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('ClusterConfigStoragePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.cluster.getClusterDetail.mockResolvedValue({
      data: { id: 42, name: 'phase1-cluster' },
    });
    mockApi.cluster.getNamespaces.mockResolvedValue({
      data: { list: [{ name: 'default', status: 'Active', created_at: '2026-04-04T00:00:00Z' }] },
    });
    mockApi.cluster.getConfigMaps.mockResolvedValue({
      data: { list: [{ name: 'app-config', data_keys: ['a', 'b'], age: '1d' }] },
    });
    mockApi.cluster.getSecrets.mockResolvedValue({
      data: { list: [] },
    });
    mockApi.cluster.getPVCs.mockResolvedValue({
      data: { list: [] },
    });
    mockApi.cluster.getPVs.mockResolvedValue({
      data: { list: [] },
    });
  });

  afterEach(() => {
    cleanup();
  });

  it('renders config and storage tabs with namespace toolbar', async () => {
    renderPage();

    expect(await screen.findByLabelText('Namespace')).toBeInTheDocument();
    expect(screen.getByText('ConfigMaps')).toBeInTheDocument();
    expect(screen.getByText('Secrets')).toBeInTheDocument();
    expect(screen.getByText('PVCs')).toBeInTheDocument();
    expect(screen.getByText('PVs')).toBeInTheDocument();
    expect(screen.getByText('app-config')).toBeInTheDocument();
  });
});
