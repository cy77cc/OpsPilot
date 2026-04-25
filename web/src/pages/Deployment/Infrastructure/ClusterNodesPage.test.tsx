import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import ClusterNodesPage from './ClusterNodesPage';

const mockApi = vi.hoisted(() => ({
  cluster: {
    getClusterDetail: vi.fn(),
    getClusterNodes: vi.fn(),
  },
}));

vi.mock('../../../api', () => ({
  Api: mockApi,
}));

function renderPage(initialEntry = '/resources/clusters/42/nodes') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/resources/clusters/:id/nodes" element={<ClusterNodesPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('ClusterNodesPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.cluster.getClusterDetail.mockResolvedValue({
      data: { id: 42, name: 'phase1-cluster', source: 'platform_managed' },
    });
    mockApi.cluster.getClusterNodes.mockResolvedValue({
      data: {
        list: [
          {
            id: 1,
            cluster_id: 42,
            name: 'worker-1',
            ip: '10.0.0.11',
            role: 'worker',
            status: 'ready',
            allocatable_cpu: '4',
            allocatable_mem: '8Gi',
            kubelet_version: 'v1.31.0',
            created_at: '2026-04-04T00:00:00Z',
            updated_at: '2026-04-04T00:00:00Z',
          },
        ],
        total: 1,
      },
    });
  });

  afterEach(() => {
    cleanup();
  });

  it('registers /deployment/infrastructure/clusters/:id/nodes route', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/app/routes/resource.routes.tsx'), 'utf8');
    expect(source).toContain('/resources/clusters/:id/nodes');
  });

  it('loads node list with compact table and namespace-independent toolbar', async () => {
    renderPage();

    expect(await screen.findByText('节点与容量')).toBeInTheDocument();
    expect(screen.getByText('worker-1')).toBeInTheDocument();
    expect(screen.getByRole('table')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '进入操作中心' })).toHaveAttribute(
      'href',
      '/resources/clusters/42/operations',
    );
  });
});
