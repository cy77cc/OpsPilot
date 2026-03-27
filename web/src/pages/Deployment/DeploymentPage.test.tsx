import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import DeploymentPage from './DeploymentPage';

const mockApi = vi.hoisted(() => ({
  deployment: {
    getTargets: vi.fn(),
    getReleasesByRuntime: vi.fn(),
    getReleases: vi.fn(),
    listInspections: vi.fn(),
    getClusterBootstrapTask: vi.fn(),
    createTarget: vi.fn(),
    previewRelease: vi.fn(),
    applyRelease: vi.fn(),
    approveRelease: vi.fn(),
    rejectRelease: vi.fn(),
    rollbackRelease: vi.fn(),
    getReleaseTimeline: vi.fn(),
    runInspection: vi.fn(),
    getGovernance: vi.fn(),
    putGovernance: vi.fn(),
    previewClusterBootstrap: vi.fn(),
    applyClusterBootstrap: vi.fn(),
    listCredentials: vi.fn(),
    getEnvironmentBootstrapJob: vi.fn(),
  },
  kubernetes: {
    getClusterList: vi.fn(),
    createCluster: vi.fn(),
  },
  hosts: {
    getHostList: vi.fn(),
  },
  services: {
    getList: vi.fn(),
  },
}));

vi.mock('../../api', () => ({ Api: mockApi }));

const seed = () => {
  mockApi.deployment.getTargets.mockResolvedValue({ data: { list: [{ id: 11, name: 'k8s-target', target_type: 'k8s', runtime_type: 'k8s', cluster_id: 1, project_id: 1, team_id: 1, env: 'staging', status: 'active', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' }] } });
  mockApi.deployment.getReleasesByRuntime.mockResolvedValue({
    data: {
      list: [{
        id: 21,
        service_id: 101,
        target_id: 11,
        namespace_or_project: 'staging',
        runtime_type: 'k8s',
        strategy: 'rolling',
        revision_id: 1,
        status: 'failed',
        diagnostics_json: '[{\"runtime\":\"k8s\",\"summary\":\"apply failed\"}]',
        verification_json: '{\"passed\":false}',
        created_at: '2026-01-01T00:00:00Z',
      }],
    },
  });
  mockApi.deployment.listInspections.mockResolvedValue({ data: { list: [] } });
  mockApi.kubernetes.getClusterList.mockResolvedValue({ data: { list: [{ id: 1, name: 'c1' }] } });
  mockApi.hosts.getHostList.mockResolvedValue({ data: { list: [{ id: 9, name: 'node-a', ip: '10.0.0.1' }] } });
  mockApi.services.getList.mockResolvedValue({ data: { list: [{ id: 101, name: 'svc-a' }] } });
  mockApi.deployment.createTarget.mockResolvedValue({ data: {} });
  mockApi.deployment.previewRelease.mockResolvedValue({ data: { resolved_manifest: '', checks: [], warnings: [], runtime: 'k8s' } });
  mockApi.deployment.applyRelease.mockResolvedValue({ data: { release_id: 1, status: 'applied', runtime_type: 'k8s' } });
  mockApi.deployment.getReleaseTimeline.mockResolvedValue({ data: { list: [] } });
  mockApi.deployment.listCredentials.mockResolvedValue({ data: { list: [] } });
  mockApi.deployment.getEnvironmentBootstrapJob.mockResolvedValue({ data: { id: 'job-1', status: 'completed' } });
};

describe('DeploymentPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    seed();
  });

  it('renders runtime_type in target list', async () => {
    render(<DeploymentPage />);
    await screen.findByText('Targets (1)');
    expect(screen.getAllByText('k8s').length).toBeGreaterThan(0);
  });

});
