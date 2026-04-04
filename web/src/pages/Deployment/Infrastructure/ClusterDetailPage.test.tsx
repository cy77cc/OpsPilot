import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import ClusterDetailPage from './ClusterDetailPage';

const mockGetClusterDetail = vi.fn();
const mockGetClusterNodes = vi.fn();
const mockGetNamespaces = vi.fn();
const mockGetDeployments = vi.fn();
const mockGetStatefulSets = vi.fn();
const mockGetDaemonSets = vi.fn();
const mockGetPods = vi.fn();
const mockGetServices = vi.fn();
const mockGetConfigMaps = vi.fn();
const mockGetSecrets = vi.fn();
const mockGetPVCs = vi.fn();
const mockGetPVs = vi.fn();
const mockGetClusterServices = vi.fn();
const mockGetEvents = vi.fn();
const mockGetHPAs = vi.fn();
const mockGetResourceQuotas = vi.fn();
const mockGetLimitRanges = vi.fn();
const mockGetClusterVersion = vi.fn();
const mockGetCertificates = vi.fn();
const mockGetUpgradePlan = vi.fn();

vi.mock('../../../api', () => ({
  Api: {
    cluster: {
      getClusterDetail: (...args: unknown[]) => mockGetClusterDetail(...args),
      getClusterNodes: (...args: unknown[]) => mockGetClusterNodes(...args),
      getNamespaces: (...args: unknown[]) => mockGetNamespaces(...args),
      getDeployments: (...args: unknown[]) => mockGetDeployments(...args),
      getStatefulSets: (...args: unknown[]) => mockGetStatefulSets(...args),
      getDaemonSets: (...args: unknown[]) => mockGetDaemonSets(...args),
      getPods: (...args: unknown[]) => mockGetPods(...args),
      getServices: (...args: unknown[]) => mockGetServices(...args),
      getConfigMaps: (...args: unknown[]) => mockGetConfigMaps(...args),
      getSecrets: (...args: unknown[]) => mockGetSecrets(...args),
      getPVCs: (...args: unknown[]) => mockGetPVCs(...args),
      getPVs: (...args: unknown[]) => mockGetPVs(...args),
      getClusterServices: (...args: unknown[]) => mockGetClusterServices(...args),
      getEvents: (...args: unknown[]) => mockGetEvents(...args),
      getHPAs: (...args: unknown[]) => mockGetHPAs(...args),
      getResourceQuotas: (...args: unknown[]) => mockGetResourceQuotas(...args),
      getLimitRanges: (...args: unknown[]) => mockGetLimitRanges(...args),
      getClusterVersion: (...args: unknown[]) => mockGetClusterVersion(...args),
      getCertificates: (...args: unknown[]) => mockGetCertificates(...args),
      getUpgradePlan: (...args: unknown[]) => mockGetUpgradePlan(...args),
      syncClusterNodes: vi.fn(),
      testCluster: vi.fn(),
      updateCluster: vi.fn(),
      deleteCluster: vi.fn(),
      addClusterNodes: vi.fn(),
      cordonNode: vi.fn(),
      uncordonNode: vi.fn(),
      drainNode: vi.fn(),
      deleteNode: vi.fn(),
      getNodeLabels: vi.fn(),
      updateNodeLabels: vi.fn(),
      getNodeTaints: vi.fn(),
      updateNodeTaints: vi.fn(),
      getNodeMetrics: vi.fn(),
      getNodeEvents: vi.fn(),
      getNodeConditions: vi.fn(),
      getNodeProcesses: vi.fn(),
      getNodeLogs: vi.fn(),
      getNodeContainers: vi.fn(),
      getNodePods: vi.fn(),
      getNodeMetricsSummary: vi.fn(),
    },
  },
}));

describe('ClusterDetailPage', () => {
  it('shows a detail skeleton during first load', async () => {
    mockGetClusterDetail.mockImplementation(
      () => new Promise(() => undefined)
    );
    mockGetClusterNodes.mockResolvedValue({ data: { list: [] } });
    mockGetNamespaces.mockResolvedValue({ data: { list: [] } });
    mockGetDeployments.mockResolvedValue({ data: { list: [] } });
    mockGetStatefulSets.mockResolvedValue({ data: { list: [] } });
    mockGetDaemonSets.mockResolvedValue({ data: { list: [] } });
    mockGetPods.mockResolvedValue({ data: { list: [] } });
    mockGetServices.mockResolvedValue({ data: { list: [] } });
    mockGetConfigMaps.mockResolvedValue({ data: { list: [] } });
    mockGetSecrets.mockResolvedValue({ data: { list: [] } });
    mockGetPVCs.mockResolvedValue({ data: { list: [] } });
    mockGetPVs.mockResolvedValue({ data: { list: [] } });
    mockGetClusterServices.mockResolvedValue({ data: { list: [] } });
    mockGetEvents.mockResolvedValue({ data: { list: [] } });
    mockGetHPAs.mockResolvedValue({ data: { list: [] } });
    mockGetResourceQuotas.mockResolvedValue({ data: { list: [] } });
    mockGetLimitRanges.mockResolvedValue({ data: { list: [] } });
    mockGetClusterVersion.mockResolvedValue({ data: null });
    mockGetCertificates.mockResolvedValue({ data: { list: [] } });
    mockGetUpgradePlan.mockResolvedValue({ data: null });

    const { container } = render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/clusters/42']}>
        <Routes>
          <Route path="/deployment/infrastructure/clusters/:id" element={<ClusterDetailPage />} />
        </Routes>
      </MemoryRouter>
    );

    expect(await screen.findByTestId('detail-skeleton')).toBeInTheDocument();
    expect(container.querySelector('.ant-spin')).toBeNull();
    expect(screen.queryByText('集群不存在')).not.toBeInTheDocument();
  });
});
