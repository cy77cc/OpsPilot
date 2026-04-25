import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import ClusterPolicyCenterPage from './ClusterPolicyCenterPage';

const mockApi = vi.hoisted(() => ({
  cluster: {
    getClusterDetail: vi.fn(),
    simulatePolicy: vi.fn(),
    createPolicyRelease: vi.fn(),
    applyPolicyRelease: vi.fn(),
    rollbackPolicyRelease: vi.fn(),
  },
}));

vi.mock('../../../api', () => ({
  Api: mockApi,
}));

const defaultCluster = {
  id: 42,
  name: 'phase2-cluster',
  description: 'policy center test cluster',
  status: 'active',
  source: 'platform_managed',
  type: 'kubernetes',
  node_count: 6,
  endpoint: 'https://cluster.example.com',
  created_at: '2026-04-05T00:00:00Z',
  updated_at: '2026-04-05T00:00:00Z',
};

const createdRelease = {
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
        release: createdRelease,
      },
    },
  });
  mockApi.cluster.simulatePolicy.mockResolvedValue({
    data: {
      passed: true,
      blocking_issues: [],
      warnings: [
        {
          code: 'L7_SIMPLIFIED',
          message: '已降级为 L4 规则',
        },
      ],
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
          ...createdRelease,
          status: {
            phase: 'applied',
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
          ...createdRelease,
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

function renderPage(initialEntry = '/resources/clusters/42/policies') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/resources/clusters/:id/policies" element={<ClusterPolicyCenterPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

async function createDraftRelease(user: ReturnType<typeof userEvent.setup>) {
  await user.click(await screen.findByRole('button', { name: '创建发布单' }));
  await waitFor(() => {
    expect(mockApi.cluster.createPolicyRelease).toHaveBeenCalledWith(42, 'prod', 'allow-api', {
      version: 'candidate-v2',
      previous_stable_version: 'stable-v1',
    });
  });
}

describe('ClusterPolicyCenterPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockBaselineLoads();
  });

  afterEach(() => {
    cleanup();
  });

  it('renders policy center shell', async () => {
    renderPage();

    expect(await screen.findByRole('heading', { name: '集群策略中心' })).toBeInTheDocument();
    expect(await screen.findByText('策略拓扑')).toBeInTheDocument();
    expect(await screen.findByText('仿真 Diff')).toBeInTheDocument();
    expect(await screen.findByText('发布操作')).toBeInTheDocument();
    expect(await screen.findByText('phase2-cluster')).toBeInTheDocument();
  });

  it('blocks release apply when simulation returns blocking issues', async () => {
    const user = userEvent.setup();
    mockApi.cluster.simulatePolicy.mockResolvedValueOnce({
      data: {
        passed: false,
        blocking_issues: [
          {
            code: 'SIMULATION_BLOCKING_CONFLICT',
            message: 'critical namespace would be blocked',
            severity: 'BLOCKING',
            suggestion: 'add an allow rule',
          },
        ],
        warnings: [],
        impact_summary: {
          affected_pods: 12,
          affected_namespaces: ['prod', 'payments'],
          new_denied_flows: ['api -> kube-dns'],
        },
        risk_score: 91,
        risk_level: 'CRITICAL',
      },
    });

    renderPage();
    await createDraftRelease(user);

    await user.click(screen.getByRole('button', { name: '运行仿真' }));
    expect((await screen.findAllByText('critical namespace would be blocked')).length).toBeGreaterThan(0);

    await user.click(screen.getByRole('button', { name: '执行发布' }));

    expect(await screen.findByText('仿真存在阻断项，禁止发布')).toBeInTheDocument();
    expect(mockApi.cluster.applyPolicyRelease).not.toHaveBeenCalled();
  });

  it('shows approval-required feedback when apply enters approval flow', async () => {
    const user = userEvent.setup();
    mockApi.cluster.applyPolicyRelease.mockResolvedValueOnce({
      data: {
        state: 'approval_required',
        success: false,
        code: 'approval_required',
        message: '需要审批后才能发布',
        approval: {
          required: true,
          ticket: 'ticket-501',
        },
        result: {
          release: {
            ...createdRelease,
            status: {
              phase: 'approval_required',
              risk_score: 35,
              risk_level: 'MEDIUM',
            },
            approval: {
              required: true,
              approval_token: 'ticket-501',
            },
          },
        },
      },
    });

    renderPage();
    await createDraftRelease(user);
    await user.click(screen.getByRole('button', { name: '运行仿真' }));

    await waitFor(() => {
      expect(mockApi.cluster.simulatePolicy).toHaveBeenCalledWith(42, 'prod', 'allow-api', {
        base_version: 'stable-v1',
        candidate_version: 'candidate-v2',
        cluster: {
          namespaces: ['prod'],
        },
      });
    });

    await user.click(screen.getByRole('button', { name: '执行发布' }));

    expect(await screen.findByText('发布进入审批')).toBeInTheDocument();
    expect(await screen.findByText('ticket-501')).toBeInTheDocument();
  });

  it('shows rollback success feedback after rollback completes', async () => {
    const user = userEvent.setup();

    renderPage();
    await createDraftRelease(user);

    await user.click(screen.getByRole('button', { name: '回滚到 stable-v1' }));

    await waitFor(() => {
      expect(mockApi.cluster.rollbackPolicyRelease).toHaveBeenCalledWith(42, 501, {
        rollback_target_version: 'stable-v1',
      });
    });

    expect(await screen.findByText('回滚已提交')).toBeInTheDocument();
    expect(await screen.findByText('已回滚到 stable-v1')).toBeInTheDocument();
  });
});
