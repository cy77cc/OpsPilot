import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { Modal } from 'antd';
import ClusterDetailPage from './ClusterDetailPage';

const mockApi = vi.hoisted(() => ({
  cluster: {
    getClusterDetail: vi.fn(),
    getClusterNodes: vi.fn(),
    getNamespaces: vi.fn(),
    getDeployments: vi.fn(),
    getStatefulSets: vi.fn(),
    getDaemonSets: vi.fn(),
    getPods: vi.fn(),
    getServices: vi.fn(),
    getConfigMaps: vi.fn(),
    getSecrets: vi.fn(),
    getPVCs: vi.fn(),
    getPVs: vi.fn(),
    getClusterServices: vi.fn(),
    getEvents: vi.fn(),
    getHPAs: vi.fn(),
    getResourceQuotas: vi.fn(),
    getLimitRanges: vi.fn(),
    getClusterVersion: vi.fn(),
    getCertificates: vi.fn(),
    getUpgradePlan: vi.fn(),
    syncClusterNodes: vi.fn(),
    testCluster: vi.fn(),
    updateCluster: vi.fn(),
    deleteCluster: vi.fn(),
    addClusterNodes: vi.fn(),
    cordonNode: vi.fn(),
    uncordonNode: vi.fn(),
    drainNode: vi.fn(),
    removeClusterNode: vi.fn(),
    upsertNodeLabel: vi.fn(),
    removeNodeLabel: vi.fn(),
    upsertNodeTaint: vi.fn(),
    removeNodeTaint: vi.fn(),
  },
}));

vi.mock('../../../api', () => ({
  Api: mockApi,
}));

const defaultCluster = {
  id: 42,
  name: 'phase1-cluster',
  description: 'test cluster',
  status: 'active',
  source: 'platform_managed',
  type: 'kubernetes',
  node_count: 1,
  created_at: '2026-04-04T00:00:00Z',
  updated_at: '2026-04-04T00:00:00Z',
};

const defaultNodes = [
  {
    id: 1,
    cluster_id: 42,
    name: 'worker-1',
    ip: '10.0.0.11',
    role: 'worker',
    status: 'ready',
    kubelet_version: 'v1.31.0',
    container_runtime: 'containerd://1.7.0',
    os_image: 'Ubuntu',
    kernel_version: '6.8.0',
    allocatable_cpu: '4',
    allocatable_mem: '8Gi',
    labels: {},
    taints: [],
    created_at: '2026-04-04T00:00:00Z',
    updated_at: '2026-04-04T00:00:00Z',
  },
];

function mockBaselineLoads() {
  mockApi.cluster.getClusterDetail.mockResolvedValue({ data: defaultCluster });
  mockApi.cluster.getClusterNodes.mockResolvedValue({ data: { list: defaultNodes } });
  mockApi.cluster.getNamespaces.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getDeployments.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getStatefulSets.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getDaemonSets.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getPods.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getServices.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getConfigMaps.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getSecrets.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getPVCs.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getPVs.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getClusterServices.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getEvents.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getHPAs.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getResourceQuotas.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getLimitRanges.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getClusterVersion.mockResolvedValue({ data: null });
  mockApi.cluster.getCertificates.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getUpgradePlan.mockResolvedValue({ data: null });
  mockApi.cluster.cordonNode.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: '节点已隔离',
      audit_id: 101,
    },
  });
  mockApi.cluster.uncordonNode.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: '节点已恢复',
      audit_id: 102,
    },
  });
  mockApi.cluster.drainNode.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: '节点已排空',
      audit_id: 103,
    },
  });
  mockApi.cluster.removeClusterNode.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: '节点已移除',
      audit_id: 104,
    },
  });
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/deployment/infrastructure/clusters/42']}>
      <Routes>
        <Route path="/deployment/infrastructure/clusters/:id" element={<ClusterDetailPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

async function openNodeActions(user: ReturnType<typeof userEvent.setup>) {
  const actionTextNodes = await screen.findAllByText(/^操作$/);
  const actionButton = actionTextNodes
    .map((node) => node.closest('button'))
    .find((node): node is HTMLButtonElement => node instanceof HTMLButtonElement);
  if (!actionButton) {
    throw new Error('node action button not found');
  }
  await user.click(actionButton);
}

describe('ClusterDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockBaselineLoads();
  });

  afterEach(() => {
    cleanup();
  });

  it('shows a detail skeleton during first load', async () => {
    mockApi.cluster.getClusterDetail.mockImplementation(
      () => new Promise(() => undefined),
    );

    const { container } = renderPage();

    expect(await screen.findByTestId('detail-skeleton')).toBeInTheDocument();
    expect(container.querySelector('.ant-spin')).toBeNull();
    expect(screen.queryByText('集群不存在')).not.toBeInTheDocument();
  });

  it('runs a row action and renders row-level audit feedback', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findAllByText('worker-1');
    await openNodeActions(user);
    await user.click(await screen.findByText('Cordon'));

    await waitFor(() => {
      expect(mockApi.cluster.cordonNode).toHaveBeenCalledWith(42, 'worker-1', { approval_token: undefined });
    });

    expect(await screen.findByText('节点隔离')).toBeInTheDocument();
    const auditLink = await screen.findByRole('link', { name: '审计' });
    expect(auditLink).toHaveAttribute('href', '/deployment/infrastructure/clusters/42/operations?audit_id=101');
  });

  it('opens the approval modal for approval_required actions and retries with approval_token', async () => {
    const user = userEvent.setup();
    mockApi.cluster.drainNode
      .mockResolvedValueOnce({
        data: {
          state: 'approval_required',
          success: false,
          code: 'approval_required',
          message: '需要审批',
          audit_id: 201,
          approval: {
            required: true,
            ticket: 'ticket-201',
          },
        },
      })
      .mockResolvedValueOnce({
        data: {
          state: 'completed',
          success: true,
          code: 'success',
          message: '节点已排空',
          audit_id: 202,
        },
      });

    renderPage();

    await screen.findAllByText('worker-1');
    await openNodeActions(user);
    await user.click(await screen.findByText('Drain'));

    expect(await screen.findByRole('dialog', { name: '审批确认' })).toBeInTheDocument();
    expect(screen.getByText('ticket-201')).toBeInTheDocument();

    await user.type(screen.getByLabelText('approval_token'), 'approved-token');
    await user.click(screen.getByRole('button', { name: '提交审批' }));

    await waitFor(() => {
      expect(mockApi.cluster.drainNode).toHaveBeenNthCalledWith(2, 42, 'worker-1', {
        approval_token: 'approved-token',
        ignore_daemonsets: true,
        delete_emptydir_data: false,
        force: false,
        grace_period_seconds: 30,
        timeout_seconds: 300,
      });
    });

    expect(await screen.findByRole('link', { name: '审计' })).toHaveAttribute(
      'href',
      '/deployment/infrastructure/clusters/42/operations?audit_id=202',
    );
  });

  it('confirms remove before calling the remove node API', async () => {
    const user = userEvent.setup();
    const confirmSpy = vi.spyOn(Modal, 'confirm');
    renderPage();

    await screen.findAllByText('worker-1');
    await openNodeActions(user);
    await user.click(await screen.findByText('Remove'));

    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalled();
    });

    const latestConfig = confirmSpy.mock.calls.at(-1)?.[0];
    expect(latestConfig?.title).toBe('移除节点');
    await latestConfig?.onOk?.();

    await waitFor(() => {
      expect(mockApi.cluster.removeClusterNode).toHaveBeenCalledWith(42, 'worker-1', { approval_token: undefined });
    });

    confirmSpy.mockRestore();
  });
});
