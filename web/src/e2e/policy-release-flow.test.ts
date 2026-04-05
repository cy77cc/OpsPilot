import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import ClusterPolicyCenterPage from '../pages/Deployment/Infrastructure/ClusterPolicyCenterPage';

const mockApi = vi.hoisted(() => ({
  cluster: {
    getClusterDetail: vi.fn(),
    simulatePolicy: vi.fn(),
    createPolicyRelease: vi.fn(),
    applyPolicyRelease: vi.fn(),
    rollbackPolicyRelease: vi.fn(),
  },
}));

vi.mock('../api', () => ({
  Api: mockApi,
}));

const defaultCluster = {
  id: 42,
  name: 'phase2-cluster',
  status: 'active',
  source: 'platform_managed',
  type: 'kubernetes',
  node_count: 6,
  endpoint: 'https://cluster.example.com',
  created_at: '2026-04-05T00:00:00Z',
  updated_at: '2026-04-05T00:00:00Z',
};

const releasePayload = {
  release_id: 501,
  version: 'candidate-v2',
  previous_stable_version: 'stable-v1',
  rollback_target_version: 'stable-v1',
  policy: {
    kind: 'NetworkPolicyDefinition',
    name: 'allow-api',
    namespace: 'prod',
  },
  target_cluster: {
    cluster_id: 42,
    cni_type: 'cilium',
    cni_version: '1.17.0',
  },
  status: {
    phase: 'draft',
    risk_score: 35,
    risk_level: 'MEDIUM',
  },
  simulation: {
    blocking_issues: [],
    warnings: [],
    impact_summary: {
      affected_pods: 8,
      affected_namespaces: ['prod'],
      new_denied_flows: ['api -> db'],
    },
  },
  approval: {
    required: false,
  },
};

function mockBaselineLoads() {
  mockApi.cluster.getClusterDetail.mockResolvedValue({ data: defaultCluster });
  mockApi.cluster.createPolicyRelease.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: '已创建发布单',
      result: {
        release: releasePayload,
      },
    },
  });
  mockApi.cluster.simulatePolicy.mockResolvedValue({
    data: {
      passed: true,
      blocking_issues: [],
      warnings: [],
      impact_summary: {
        affected_pods: 8,
        affected_namespaces: ['prod'],
        new_denied_flows: ['api -> db'],
      },
      risk_score: 35,
      risk_level: 'MEDIUM',
    },
  });
  mockApi.cluster.applyPolicyRelease.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: '发布已执行',
      result: {
        release: {
          ...releasePayload,
          status: {
            phase: 'active',
            risk_score: 35,
            risk_level: 'MEDIUM',
          },
        },
      },
    },
  });
  mockApi.cluster.rollbackPolicyRelease.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: '已回滚到 stable-v1',
      result: {
        release: {
          ...releasePayload,
          status: {
            phase: 'rollback_applied',
            risk_score: 20,
            risk_level: 'LOW',
          },
        },
      },
    },
  });
}

function renderPage() {
  const pageElement = React.createElement(ClusterPolicyCenterPage);
  const routeElement = React.createElement(Route, {
    path: '/deployment/infrastructure/clusters/:id/policies',
    element: pageElement,
  });
  const routesElement = React.createElement(Routes, null, routeElement);
  const routerElement = React.createElement(
    MemoryRouter,
    { initialEntries: ['/deployment/infrastructure/clusters/42/policies'] },
    routesElement,
  );

  return render(
    routerElement,
  );
}

async function createRelease(user: ReturnType<typeof userEvent.setup>) {
  await user.click(await screen.findByRole('button', { name: '创建发布单' }));
  await waitFor(() => {
    expect(mockApi.cluster.createPolicyRelease).toHaveBeenCalledWith(42, 'prod', 'allow-api', {
      version: 'candidate-v2',
      previous_stable_version: 'stable-v1',
    });
  });
}

describe('PolicyReleaseFlowE2E', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockBaselineLoads();
  });

  afterEach(() => {
    cleanup();
  });

  it('complete policy release flow', async () => {
    const user = userEvent.setup();
    renderPage();

    await createRelease(user);
    await user.click(screen.getByRole('button', { name: '运行仿真' }));
    await waitFor(() => {
      expect(mockApi.cluster.simulatePolicy).toHaveBeenCalledTimes(1);
    });

    await user.click(screen.getByRole('button', { name: '执行发布' }));
    expect(await screen.findByText('发布已提交')).toBeInTheDocument();
    expect(mockApi.cluster.applyPolicyRelease).toHaveBeenCalledWith(42, 501);
  }, 30000);

  it('simulation blocking UI feedback', async () => {
    const user = userEvent.setup();
    mockApi.cluster.simulatePolicy.mockResolvedValueOnce({
      data: {
        passed: false,
        blocking_issues: [
          {
            code: 'SIMULATION_BLOCKING_CONFLICT',
            message: 'critical namespace would be blocked',
            severity: 'BLOCKING',
            suggestion: 'add allowlist',
          },
        ],
        warnings: [],
        impact_summary: {
          affected_pods: 12,
          affected_namespaces: ['prod', 'kube-system'],
          new_denied_flows: ['api -> kube-dns'],
        },
        risk_score: 90,
        risk_level: 'CRITICAL',
      },
    });

    renderPage();
    await createRelease(user);
    await user.click(screen.getByRole('button', { name: '运行仿真' }));
    expect((await screen.findAllByText('critical namespace would be blocked')).length).toBeGreaterThan(0);

    await user.click(screen.getByRole('button', { name: '执行发布' }));
    expect(await screen.findByText('仿真存在阻断项，禁止发布')).toBeInTheDocument();
    expect(mockApi.cluster.applyPolicyRelease).not.toHaveBeenCalled();
  }, 30000);

  it('rollback success feedback', async () => {
    const user = userEvent.setup();
    renderPage();

    await createRelease(user);
    await user.click(screen.getByRole('button', { name: '回滚到 stable-v1' }));

    expect(await screen.findByText('回滚已提交')).toBeInTheDocument();
    expect(mockApi.cluster.rollbackPolicyRelease).toHaveBeenCalledWith(42, 501, expect.objectContaining({
      rollback_target_version: 'stable-v1',
    }));
  }, 30000);
});
