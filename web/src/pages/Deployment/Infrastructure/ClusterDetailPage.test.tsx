import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { message, Modal } from 'antd';
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
    restartDeployment: vi.fn(),
    scaleDeployment: vi.fn(),
    deleteDeployment: vi.fn(),
    restartStatefulSet: vi.fn(),
    scaleStatefulSet: vi.fn(),
    deleteStatefulSet: vi.fn(),
    deletePod: vi.fn(),
    getServices: vi.fn(),
    createService: vi.fn(),
    updateService: vi.fn(),
    deleteService: vi.fn(),
    getIngresses: vi.fn(),
    createIngress: vi.fn(),
    updateIngress: vi.fn(),
    deleteIngress: vi.fn(),
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
    renewCertificates: vi.fn(),
    upgradeCluster: vi.fn(),
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

const HIGH_RISK_RUNBOOK_PATH = '/docs/runbooks/cluster-high-risk-operations.md';

function mockBaselineLoads() {
  mockApi.cluster.getClusterDetail.mockResolvedValue({ data: defaultCluster });
  mockApi.cluster.getClusterNodes.mockResolvedValue({ data: { list: defaultNodes } });
  mockApi.cluster.getNamespaces.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getDeployments.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getStatefulSets.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getDaemonSets.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getPods.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getServices.mockResolvedValue({ data: { list: [] } });
  mockApi.cluster.getIngresses.mockResolvedValue({ data: { list: [] } });
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
  mockApi.cluster.renewCertificates.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: '证书已续期',
      audit_id: 115,
    },
  });
  mockApi.cluster.upgradeCluster.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: '集群已升级',
      audit_id: 116,
    },
  });
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
  mockApi.cluster.upsertNodeLabel.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: '标签已更新',
      audit_id: 105,
    },
  });
  mockApi.cluster.removeNodeLabel.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: '标签已删除',
      audit_id: 106,
    },
  });
  mockApi.cluster.upsertNodeTaint.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: '污点已更新',
      audit_id: 107,
    },
  });
  mockApi.cluster.removeNodeTaint.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: '污点已删除',
      audit_id: 108,
    },
  });
  mockApi.cluster.createService.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: 'Service 已创建',
      audit_id: 109,
    },
  });
  mockApi.cluster.updateService.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: 'Service 已更新',
      audit_id: 110,
    },
  });
  mockApi.cluster.deleteService.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: 'Service 已删除',
      audit_id: 111,
    },
  });
  mockApi.cluster.createIngress.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: 'Ingress 已创建',
      audit_id: 112,
    },
  });
  mockApi.cluster.updateIngress.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: 'Ingress 已更新',
      audit_id: 113,
    },
  });
  mockApi.cluster.deleteIngress.mockResolvedValue({
    data: {
      state: 'completed',
      success: true,
      code: 'success',
      message: 'Ingress 已删除',
      audit_id: 114,
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

async function openServicesTab(user: ReturnType<typeof userEvent.setup>) {
  const tabs = await screen.findAllByRole('tab');
  const servicesTab = tabs.find((tab) => tab.textContent?.trim() === '服务');
  if (!(servicesTab instanceof HTMLElement)) {
    throw new Error('services tab not found');
  }
  await user.click(servicesTab);
}

async function openMaintenanceTab(user: ReturnType<typeof userEvent.setup>) {
  const tab = await screen.findByRole('tab', { name: /运维/i });
  await user.click(tab);
}

async function confirmLatestPopconfirm(
  user: ReturnType<typeof userEvent.setup>,
  title: string | RegExp,
) {
  await screen.findByText(title);
  const confirmButton = screen.getAllByRole('button', { name: /确\s*定/ }).at(-1);
  if (!(confirmButton instanceof HTMLButtonElement)) {
    throw new Error('popconfirm confirm button not found');
  }
  await user.click(confirmButton);
}

async function expectHighRiskRunbookGuidance(summary: string | RegExp) {
  expect(await screen.findByText(summary)).toBeInTheDocument();
  expect(screen.getAllByRole('link', { name: '查看运行手册' }).some((link) => (
    link.getAttribute('href') === HIGH_RISK_RUNBOOK_PATH
  ))).toBe(true);
}

describe('ClusterDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockBaselineLoads();
  });

  afterEach(() => {
    message.destroy();
    Modal.destroyAll();
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

  it('renders action-first overview and keeps base info in collapsed section', async () => {
    renderPage();

    expect(await screen.findByText('集群作战面板')).toBeInTheDocument();
    expect(screen.getByText('关键操作台')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '展开基础信息' })).toBeInTheDocument();
    expect(screen.queryByText('基本信息')).not.toBeInTheDocument();
  });

  it('renders quick links to security, policy, and operation centers', async () => {
    renderPage();

    expect(await screen.findByRole('link', { name: '进入安全中心' })).toHaveAttribute(
      'href',
      '/deployment/infrastructure/clusters/42/security',
    );
    expect(screen.getByRole('link', { name: '进入策略中心' })).toHaveAttribute(
      'href',
      '/deployment/infrastructure/clusters/42/policies',
    );
    expect(screen.getByRole('link', { name: '查看全部操作' })).toHaveAttribute(
      'href',
      '/deployment/infrastructure/clusters/42/operations',
    );
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

  it('runs uncordon action from row dropdown', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findAllByText('worker-1');
    await openNodeActions(user);
    await user.click(await screen.findByText('Uncordon'));

    await waitFor(() => {
      expect(mockApi.cluster.uncordonNode).toHaveBeenCalledWith(42, 'worker-1', { approval_token: undefined });
    });
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

  it('updates and removes node label/taint from node drawer', async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole('button', { name: 'worker-1' }));

    fireEvent.change(screen.getByPlaceholderText('app.kubernetes.io/name'), { target: { value: 'app' } });
    fireEvent.change(screen.getByPlaceholderText('frontend'), { target: { value: 'frontend' } });
    await user.click(screen.getByRole('button', { name: '保存标签' }));

    await waitFor(() => {
      expect(mockApi.cluster.upsertNodeLabel).toHaveBeenCalledWith(42, 'worker-1', {
        key: 'app',
        value: 'frontend',
        approval_token: undefined,
      });
    });

    fireEvent.change(screen.getByPlaceholderText('app.kubernetes.io/name'), { target: { value: 'app' } });
    fireEvent.change(screen.getByPlaceholderText('frontend'), { target: { value: 'frontend' } });
    await user.click(screen.getByRole('button', { name: '删除标签' }));
    await waitFor(() => {
      expect(mockApi.cluster.removeNodeLabel).toHaveBeenCalledWith(42, 'worker-1', {
        key: 'app',
        value: 'frontend',
        approval_token: undefined,
      });
    });

    fireEvent.change(screen.getByPlaceholderText('node-role.kubernetes.io/worker'), { target: { value: 'node-role' } });
    fireEvent.change(screen.getByPlaceholderText('value'), { target: { value: 'dedicated' } });
    await user.click(screen.getByRole('button', { name: '保存污点' }));
    await waitFor(() => {
      expect(mockApi.cluster.upsertNodeTaint).toHaveBeenCalledWith(42, 'worker-1', {
        key: 'node-role',
        value: 'dedicated',
        effect: undefined,
        approval_token: undefined,
      });
    });

    fireEvent.change(screen.getByPlaceholderText('node-role.kubernetes.io/worker'), { target: { value: 'node-role' } });
    fireEvent.change(screen.getByPlaceholderText('value'), { target: { value: 'dedicated' } });
    await user.click(screen.getByRole('button', { name: '删除污点' }));
    await waitFor(() => {
      expect(mockApi.cluster.removeNodeTaint).toHaveBeenCalledWith(42, 'worker-1', {
        key: 'node-role',
        value: 'dedicated',
        effect: undefined,
        approval_token: undefined,
      });
    });
  }, 90000);

  it('runs a deployment restart from the workloads tab and renders audit feedback', async () => {
    const user = userEvent.setup();
    mockApi.cluster.getNamespaces.mockResolvedValue({
      data: {
        list: [{ name: 'default', status: 'Active', created_at: '2026-04-04T00:00:00Z' }],
      },
    });
    mockApi.cluster.getDeployments.mockResolvedValue({
      data: {
        list: [{
          name: 'web',
          namespace: 'default',
          replicas: 3,
          ready: 3,
          updated: 3,
          available: 3,
          age: '1d',
          created_at: '2026-04-04T00:00:00Z',
        }],
      },
    });
    mockApi.cluster.restartDeployment.mockResolvedValue({
      data: {
        state: 'completed',
        success: true,
        code: 'success',
        message: 'Deployment 已重启',
        audit_id: 301,
      },
    });

    renderPage();

    await user.click(await screen.findByRole('tab', { name: /工作负载/i }));
    await screen.findByText('web');
    await user.click(screen.getByRole('button', { name: '重启 Deployment web' }));

    await waitFor(() => {
      expect(mockApi.cluster.restartDeployment).toHaveBeenCalledWith(42, 'default', 'web', { approval_token: undefined });
    });

    expect(await screen.findByText('Deployment 重启')).toBeInTheDocument();
    expect(await screen.findByText('审计')).toBeInTheDocument();
  }, 90000);

  it('opens the scale modal for statefulsets and retries with approval_token', async () => {
    const user = userEvent.setup();
    mockApi.cluster.getNamespaces.mockResolvedValue({
      data: {
        list: [{ name: 'default', status: 'Active', created_at: '2026-04-04T00:00:00Z' }],
      },
    });
    mockApi.cluster.getStatefulSets.mockResolvedValue({
      data: {
        list: [{
          name: 'db',
          namespace: 'default',
          replicas: 2,
          ready: 2,
          age: '1d',
          created_at: '2026-04-04T00:00:00Z',
        }],
      },
    });
    mockApi.cluster.scaleStatefulSet
      .mockResolvedValueOnce({
        data: {
          state: 'approval_required',
          success: false,
          code: 'approval_required',
          message: '需要审批',
          audit_id: 401,
          approval: {
            required: true,
            ticket: 'ticket-401',
          },
        },
      })
      .mockResolvedValueOnce({
        data: {
          state: 'completed',
          success: true,
          code: 'success',
          message: 'StatefulSet 已扩缩容',
          audit_id: 402,
        },
      });

    renderPage();

    await user.click(await screen.findByRole('tab', { name: /工作负载/i }));
    await screen.findByText('db');
    await user.click(screen.getByRole('button', { name: '扩缩容 StatefulSet db' }));

    expect(await screen.findByText('调整副本数')).toBeInTheDocument();
    fireEvent.change(screen.getByRole('spinbutton', { name: 'replicas' }), { target: { value: '5' } });
    await user.click(screen.getByRole('button', { name: '提交扩缩容' }));

    await waitFor(() => {
      expect(mockApi.cluster.scaleStatefulSet).toHaveBeenNthCalledWith(1, 42, 'default', 'db', {
        replicas: 5,
        approval_token: undefined,
      });
    });

    expect(await screen.findByLabelText('approval_token')).toBeInTheDocument();
    await user.type(screen.getByLabelText('approval_token'), 'approved-token');
    await user.click(screen.getByRole('button', { name: '提交审批' }));

    await waitFor(() => {
      expect(mockApi.cluster.scaleStatefulSet).toHaveBeenNthCalledWith(2, 42, 'default', 'db', {
        replicas: 5,
        approval_token: 'approved-token',
      });
    });
  }, 90000);

  it('deletes a service from the services tab and renders audit feedback', async () => {
    const user = userEvent.setup();
    mockApi.cluster.getNamespaces.mockResolvedValue({
      data: {
        list: [{ name: 'default', status: 'Active', created_at: '2026-04-04T00:00:00Z' }],
      },
    });
    mockApi.cluster.getServices.mockResolvedValue({
      data: {
        list: [{
          name: 'web',
          namespace: 'default',
          type: 'ClusterIP',
          cluster_ip: '10.96.0.20',
          selector: { app: 'web' },
          ports: [{ name: 'http', port: 80, target_port: '8080', protocol: 'TCP' }],
          age: '1d',
          created_at: '2026-04-04T00:00:00Z',
        }],
      },
    });

    renderPage();

    await openServicesTab(user);
    await screen.findByText('web');
    await user.click(screen.getByRole('button', { name: '删除 Service web' }));
    await confirmLatestPopconfirm(user, '确定删除此 Service？');

    await waitFor(() => {
      expect(mockApi.cluster.deleteService).toHaveBeenCalledWith(42, 'default', 'web', { approval_token: undefined });
    });

    expect(await screen.findByText('Service 删除')).toBeInTheDocument();
    expect(await screen.findByRole('link', { name: '审计' })).toHaveAttribute(
      'href',
      '/deployment/infrastructure/clusters/42/operations?audit_id=111',
    );
  }, 90000);

  it('creates an ingress from the services tab and retries with approval_token', async () => {
    const user = userEvent.setup();
    mockApi.cluster.getNamespaces.mockResolvedValue({
      data: {
        list: [{ name: 'default', status: 'Active', created_at: '2026-04-04T00:00:00Z' }],
      },
    });
    mockApi.cluster.getIngresses
      .mockResolvedValueOnce({ data: { list: [] } })
      .mockResolvedValue({
        data: {
          list: [{
            name: 'web',
            namespace: 'default',
            hosts: [{ host: 'app.example.com', paths: ['/'] }],
            age: '1d',
            created_at: '2026-04-04T00:00:00Z',
          }],
        },
      });
    mockApi.cluster.createIngress
      .mockResolvedValueOnce({
        data: {
          state: 'approval_required',
          success: false,
          code: 'approval_required',
          message: '需要审批',
          audit_id: 601,
          approval: {
            required: true,
            ticket: 'ticket-601',
          },
        },
      })
      .mockResolvedValueOnce({
        data: {
          state: 'completed',
          success: true,
          code: 'success',
          message: 'Ingress 已创建',
          audit_id: 602,
        },
      });

    renderPage();

    await openServicesTab(user);
    await user.click(await screen.findByRole('button', { name: '新建 Ingress' }));

    expect(await screen.findByLabelText('ingress_name')).toBeInTheDocument();
    await user.type(screen.getByLabelText('ingress_name'), 'web');
    await user.type(screen.getByLabelText('ingress_host'), 'app.example.com');
    fireEvent.change(screen.getByLabelText('ingress_path'), { target: { value: '/' } });
    await user.type(screen.getByLabelText('backend_service_name'), 'web');
    fireEvent.change(screen.getByRole('spinbutton', { name: 'backend_service_port' }), { target: { value: '80' } });
    await user.click(screen.getByRole('button', { name: '保存 Ingress' }));

    await waitFor(() => {
      expect(mockApi.cluster.createIngress).toHaveBeenNthCalledWith(1, 42, 'default', {
        name: 'web',
        ingress_class_name: undefined,
        rules: [{
          host: 'app.example.com',
          paths: [{
            path: '/',
            path_type: 'Prefix',
            service_name: 'web',
            service_port: 80,
          }],
        }],
        tls: undefined,
        approval_token: undefined,
      });
    });

    expect(await screen.findByLabelText('approval_token')).toBeInTheDocument();
    await user.type(screen.getByLabelText('approval_token'), 'approved-token');
    await user.click(screen.getByRole('button', { name: '提交审批' }));

    await waitFor(() => {
      expect(mockApi.cluster.createIngress).toHaveBeenNthCalledWith(2, 42, 'default', {
        name: 'web',
        ingress_class_name: undefined,
        rules: [{
          host: 'app.example.com',
          paths: [{
            path: '/',
            path_type: 'Prefix',
            service_name: 'web',
            service_port: 80,
          }],
        }],
        tls: undefined,
        approval_token: 'approved-token',
      });
    });

    expect(await screen.findByText('Ingress 创建')).toBeInTheDocument();
    expect(await screen.findByRole('link', { name: '审计' })).toHaveAttribute(
      'href',
      '/deployment/infrastructure/clusters/42/operations?audit_id=602',
    );
  }, 90000);

  it('creates a service from services tab', async () => {
    const user = userEvent.setup();
    mockApi.cluster.getNamespaces.mockResolvedValue({
      data: { list: [{ name: 'default', status: 'Active', created_at: '2026-04-04T00:00:00Z' }] },
    });
    mockApi.cluster.getServices.mockResolvedValue({ data: { list: [] } });

    renderPage();

    await openServicesTab(user);
    await user.click(await screen.findByRole('button', { name: '新建 Service' }));
    await user.type(screen.getByLabelText('service_name'), 'edge');
    await user.type(screen.getByLabelText('selector'), 'app=edge');
    fireEvent.change(screen.getByRole('spinbutton', { name: 'service_port' }), { target: { value: '80' } });
    await user.type(screen.getByLabelText('target_port'), '8080');
    await user.click(screen.getByRole('button', { name: '保存 Service' }));

    await waitFor(() => {
      expect(mockApi.cluster.createService).toHaveBeenCalledWith(42, 'default', expect.objectContaining({
        name: 'edge',
        approval_token: undefined,
      }));
    });
  }, 90000);

  it('updates and deletes an ingress from services tab', async () => {
    const user = userEvent.setup();
    mockApi.cluster.getNamespaces.mockResolvedValue({
      data: { list: [{ name: 'default', status: 'Active', created_at: '2026-04-04T00:00:00Z' }] },
    });
    mockApi.cluster.getServices.mockResolvedValue({ data: { list: [] } });
    mockApi.cluster.getIngresses.mockResolvedValue({
      data: {
        list: [{
          name: 'web',
          namespace: 'default',
          hosts: [{ host: 'old.example.com', paths: ['/'] }],
          age: '1d',
          created_at: '2026-04-04T00:00:00Z',
        }],
      },
    });

    renderPage();

    await openServicesTab(user);
    await user.click(await screen.findByRole('button', { name: '编辑 Ingress web' }));
    fireEvent.change(screen.getByLabelText('ingress_host'), { target: { value: 'new.example.com' } });
    await user.type(screen.getByLabelText('backend_service_name'), 'web');
    fireEvent.change(screen.getByRole('spinbutton', { name: 'backend_service_port' }), { target: { value: '8080' } });
    await user.click(screen.getByRole('button', { name: '保存 Ingress' }));

    await waitFor(() => {
      expect(mockApi.cluster.updateIngress).toHaveBeenCalledWith(42, 'default', 'web', expect.objectContaining({
        name: 'web',
        approval_token: undefined,
      }));
    });

    await user.click(await screen.findByRole('button', { name: '删除 Ingress web' }));
    await confirmLatestPopconfirm(user, '确定删除此 Ingress？');
    await waitFor(() => {
      expect(mockApi.cluster.deleteIngress).toHaveBeenCalledWith(42, 'default', 'web', { approval_token: undefined });
    });
  }, 90000);

  it('shows error feedback when service create fails', async () => {
    const user = userEvent.setup();
    const errorSpy = vi.spyOn(message, 'error').mockImplementation(() => null as any);
    mockApi.cluster.getNamespaces.mockResolvedValue({
      data: { list: [{ name: 'default', status: 'Active', created_at: '2026-04-04T00:00:00Z' }] },
    });
    mockApi.cluster.getServices.mockResolvedValue({ data: { list: [] } });
    mockApi.cluster.createService.mockRejectedValueOnce(new Error('create failed'));

    renderPage();

    await openServicesTab(user);
    await user.click(await screen.findByRole('button', { name: '新建 Service' }));
    await user.type(screen.getByLabelText('service_name'), 'edge');
    await user.type(screen.getByLabelText('selector'), 'app=edge');
    fireEvent.change(screen.getByRole('spinbutton', { name: 'service_port' }), { target: { value: '80' } });
    await user.type(screen.getByLabelText('target_port'), '8080');
    await user.click(screen.getByRole('button', { name: '保存 Service' }));

    await waitFor(() => {
      expect(errorSpy).toHaveBeenCalledWith('create failed');
    });
    errorSpy.mockRestore();
  }, 90000);

  it('renders an operation-center audit link after renewing certificates', async () => {
    const user = userEvent.setup();
    mockApi.cluster.getCertificates.mockResolvedValue({
      data: {
        list: [{
          name: 'apiserver',
          ca: false,
          expires_at: '2026-08-01T00:00:00Z',
          days_left: 90,
        }],
      },
    });

    renderPage();

    await openMaintenanceTab(user);
    await user.click(await screen.findByRole('button', { name: /续期证书/ }));
    await confirmLatestPopconfirm(user, '确定续期所有证书？此操作将重启控制平面组件。');

    await waitFor(() => {
      expect(mockApi.cluster.renewCertificates).toHaveBeenCalledWith(42, { approval_token: undefined });
    });

    expect(await screen.findByText('证书续期')).toBeInTheDocument();
    expect(await screen.findByRole('link', { name: '审计' })).toHaveAttribute(
      'href',
      '/deployment/infrastructure/clusters/42/operations?audit_id=115',
    );
  }, 90000);

  it('renders an operation-center audit link after cluster upgrade', async () => {
    const user = userEvent.setup();
    mockApi.cluster.getUpgradePlan.mockResolvedValue({
      data: {
        current_version: 'v1.28.0',
        upgradable: true,
        warnings: [],
        steps: [],
      },
    });

    renderPage();

    await openMaintenanceTab(user);
    await user.click(await screen.findByRole('button', { name: /升级集群/ }));
    await confirmLatestPopconfirm(user, '确定升级集群？建议先备份数据。');

    await waitFor(() => {
      expect(mockApi.cluster.upgradeCluster).toHaveBeenCalledWith(42, {
        target_version: '1.29.0',
        approval_token: undefined,
      });
    });

    expect(await screen.findByText('集群升级')).toBeInTheDocument();
    expect(await screen.findByRole('link', { name: '审计' })).toHaveAttribute(
      'href',
      '/deployment/infrastructure/clusters/42/operations?audit_id=116',
    );
  }, 90000);

  it('shows drain recovery guidance when node drain fails', async () => {
    const user = userEvent.setup();
    mockApi.cluster.drainNode.mockResolvedValueOnce({
      data: {
        state: 'failed',
        success: false,
        code: 'drain_failed',
        message: '节点排空失败：仍有 Pod 因 PDB 无法驱逐',
        audit_id: 701,
      },
    });

    renderPage();

    await screen.findAllByText('worker-1');
    await openNodeActions(user);
    await user.click(await screen.findByText('Drain'));

    await waitFor(() => {
      expect(mockApi.cluster.drainNode).toHaveBeenCalledWith(42, 'worker-1', {
        approval_token: undefined,
        ignore_daemonsets: true,
        delete_emptydir_data: false,
        force: false,
        grace_period_seconds: 30,
        timeout_seconds: 300,
      });
    });

    await expectHighRiskRunbookGuidance(/核对未驱逐 Pod、PDB 与 DaemonSet 阻塞/i);
  });

  it('shows remove recovery guidance when node removal fails', async () => {
    const user = userEvent.setup();
    const confirmSpy = vi.spyOn(Modal, 'confirm');
    mockApi.cluster.removeClusterNode.mockResolvedValueOnce({
      data: {
        state: 'failed',
        success: false,
        code: 'remove_failed',
        message: '节点移除失败：节点仍处于 Ready',
        audit_id: 702,
      },
    });

    renderPage();

    await screen.findAllByText('worker-1');
    await openNodeActions(user);
    await user.click(await screen.findByText('Remove'));

    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalled();
    });

    const latestConfig = confirmSpy.mock.calls.at(-1)?.[0];
    await latestConfig?.onOk?.();

    await waitFor(() => {
      expect(mockApi.cluster.removeClusterNode).toHaveBeenCalledWith(42, 'worker-1', { approval_token: undefined });
    });

    await expectHighRiskRunbookGuidance(/确认节点已完成 drain、已从流量与自动伸缩池摘除/i);
    confirmSpy.mockRestore();
  }, 90000);

  it('shows certificate renewal recovery guidance when renew fails', async () => {
    const user = userEvent.setup();
    mockApi.cluster.getCertificates.mockResolvedValue({
      data: {
        list: [{
          name: 'apiserver',
          ca: false,
          expires_at: '2026-08-01T00:00:00Z',
          days_left: 90,
        }],
      },
    });
    mockApi.cluster.renewCertificates.mockResolvedValueOnce({
      data: {
        state: 'failed',
        success: false,
        code: 'renew_failed',
        message: '证书续期失败：controller-manager 未恢复',
        audit_id: 703,
      },
    });

    renderPage();

    await openMaintenanceTab(user);
    await user.click(await screen.findByRole('button', { name: /续期证书/ }));
    await confirmLatestPopconfirm(user, '确定续期所有证书？此操作将重启控制平面组件。');

    await waitFor(() => {
      expect(mockApi.cluster.renewCertificates).toHaveBeenCalledWith(42, { approval_token: undefined });
    });

    await expectHighRiskRunbookGuidance(/逐项核对 apiserver、controller-manager、scheduler 证书与静态 Pod 重启情况/i);
  }, 90000);

  it('shows upgrade recovery guidance when cluster upgrade fails', async () => {
    const user = userEvent.setup();
    mockApi.cluster.getUpgradePlan.mockResolvedValue({
      data: {
        current_version: 'v1.28.0',
        upgradable: true,
        warnings: [],
        steps: [],
      },
    });
    mockApi.cluster.upgradeCluster.mockResolvedValueOnce({
      data: {
        state: 'failed',
        success: false,
        code: 'upgrade_failed',
        message: '集群升级失败：升级前检查未通过',
        audit_id: 704,
      },
    });

    renderPage();

    await openMaintenanceTab(user);
    await user.click(await screen.findByRole('button', { name: /升级集群/ }));
    await confirmLatestPopconfirm(user, '确定升级集群？建议先备份数据。');

    await waitFor(() => {
      expect(mockApi.cluster.upgradeCluster).toHaveBeenCalledWith(42, {
        target_version: '1.29.0',
        approval_token: undefined,
      });
    });

    await expectHighRiskRunbookGuidance(/先冻结变更并确认 etcd 与控制平面备份可恢复/i);
  }, 90000);
});
