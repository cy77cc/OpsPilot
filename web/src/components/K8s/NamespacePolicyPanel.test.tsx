import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { scopeStore } from '../../app/scope/scopeStore';
import HPAEditor from './HPAEditor';
import NamespacePolicyPanel from './NamespacePolicyPanel';
import QuotaEditor from './QuotaEditor';

const mockApi = vi.hoisted(() => ({
  kubernetes: {
    getClusterNamespaces: vi.fn(),
    getNamespaceBindings: vi.fn(),
    putNamespaceBindings: vi.fn(),
    createNamespace: vi.fn(),
    deleteNamespace: vi.fn(),
    listQuotas: vi.fn(),
    listLimitRanges: vi.fn(),
    applyQuota: vi.fn(),
    createLimitRange: vi.fn(),
    deleteQuota: vi.fn(),
    listHPA: vi.fn(),
    updateHPA: vi.fn(),
    createHPA: vi.fn(),
    deleteHPA: vi.fn(),
  },
}));

vi.mock('../../api', () => ({ Api: mockApi }));

afterEach(() => {
  cleanup();
});

describe('NamespacePolicyPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    scopeStore.clearScope();
    scopeStore.setScope({ teamId: '7' });
    mockApi.kubernetes.getClusterNamespaces.mockResolvedValue({ data: { list: [] } });
    mockApi.kubernetes.getNamespaceBindings.mockResolvedValue({ data: { list: [] } });
    mockApi.kubernetes.putNamespaceBindings.mockResolvedValue({ data: {} });
    mockApi.kubernetes.listQuotas.mockResolvedValue({ data: { list: [] } });
    mockApi.kubernetes.listLimitRanges.mockResolvedValue({ data: { list: [] } });
    mockApi.kubernetes.listHPA.mockResolvedValue({ data: { list: [] } });
  });

  it('reads and updates team scope through ScopeStore', async () => {
    render(<NamespacePolicyPanel clusterId="1" />);

    await waitFor(() => {
      expect(mockApi.kubernetes.getNamespaceBindings).toHaveBeenCalledWith('1', '7');
    });

    fireEvent.click(screen.getByRole('button', { name: '管理 Team 绑定' }));
    const dialog = await screen.findByRole('dialog', { name: '更新 Team Namespace 绑定' });
    fireEvent.change(screen.getByRole('spinbutton', { name: 'Team ID' }), { target: { value: '11' } });
    const namespacesInput = screen.getByPlaceholderText('逗号分隔: default,dev,staging');
    fireEvent.change(namespacesInput, { target: { value: 'dev' } });
    fireEvent.blur(namespacesInput);
    fireEvent.click(within(dialog).getByRole('button', { name: /^(OK|确定)$/ }));

    await waitFor(() => {
      expect(mockApi.kubernetes.putNamespaceBindings).toHaveBeenCalledWith('1', '11', [{ namespace: 'dev' }]);
      expect(scopeStore.getSnapshot().teamId).toBe('11');
    });
  });

  it('delegates binding saves to injected actions instead of calling the api inline', async () => {
    const actions = {
      load: vi.fn().mockResolvedValue({ namespaces: [], bindings: [] }),
      createNamespace: vi.fn(),
      saveBindings: vi.fn().mockResolvedValue(undefined),
      removeNamespace: vi.fn(),
    };

    render(<NamespacePolicyPanel clusterId="1" actions={actions as any} />);

    fireEvent.click(screen.getByRole('button', { name: '管理 Team 绑定' }));
    const dialog = await screen.findByRole('dialog', { name: '更新 Team Namespace 绑定' });

    fireEvent.change(screen.getByRole('spinbutton', { name: 'Team ID' }), { target: { value: '11' } });
    const namespacesInput = screen.getByPlaceholderText('逗号分隔: default,dev,staging');
    fireEvent.change(namespacesInput, { target: { value: 'dev,staging' } });
    fireEvent.blur(namespacesInput);
    fireEvent.click(within(dialog).getByRole('button', { name: /^(OK|确定)$/ }));

    await waitFor(() => {
      expect(actions.saveBindings).toHaveBeenCalledWith({
        clusterId: '1',
        teamId: '11',
        namespaces: ['dev', 'staging'],
      });
    });

    expect(mockApi.kubernetes.putNamespaceBindings).not.toHaveBeenCalled();
  });
});

describe('QuotaEditor', () => {
  it('delegates quota saves to injected actions instead of calling the api inline', async () => {
    const actions = {
      load: vi.fn().mockResolvedValue({ quotas: [], limits: [] }),
      saveQuota: vi.fn().mockResolvedValue(undefined),
      saveLimit: vi.fn().mockResolvedValue(undefined),
      removeQuota: vi.fn().mockResolvedValue(undefined),
    };

    render(<QuotaEditor clusterId="1" actions={actions as any} />);

    fireEvent.click(screen.getByRole('button', { name: '新增/更新 Quota' }));
    const dialog = await screen.findByRole('dialog', { name: 'Quota' });
    const textboxes = within(dialog).getAllByRole('textbox');
    fireEvent.change(textboxes[0], { target: { value: 'dev' } });
    fireEvent.change(textboxes[1], { target: { value: 'team-quota' } });
    fireEvent.change(textboxes[2], { target: { value: 'limits.cpu=4\npods=20' } });
    fireEvent.click(within(dialog).getByRole('button', { name: /^(OK|确定)$/ }));

    await waitFor(() => {
      expect(actions.saveQuota).toHaveBeenCalledWith({
        clusterId: '1',
        quota: {
          namespace: 'dev',
          name: 'team-quota',
          hard: {
            'limits.cpu': '4',
            pods: '20',
          },
        },
      });
    });

    expect(mockApi.kubernetes.applyQuota).not.toHaveBeenCalled();
  });
});

describe('HPAEditor', () => {
  it('delegates hpa saves to injected actions instead of calling the api inline', async () => {
    const actions = {
      load: vi.fn().mockResolvedValue({ list: [] }),
      save: vi.fn().mockResolvedValue(undefined),
      remove: vi.fn().mockResolvedValue(undefined),
    };

    render(<HPAEditor clusterId="1" actions={actions as any} />);

    fireEvent.click(screen.getByRole('button', { name: '新增/更新 HPA' }));
    const dialog = await screen.findByRole('dialog', { name: 'HPA 策略' });
    const textboxes = within(dialog).getAllByRole('textbox');
    fireEvent.change(textboxes[0], { target: { value: 'dev' } });
    fireEvent.change(textboxes[1], { target: { value: 'web' } });
    fireEvent.change(textboxes[2], { target: { value: 'Deployment' } });
    fireEvent.change(textboxes[3], { target: { value: 'frontend' } });
    const spinbuttons = within(dialog).getAllByRole('spinbutton');
    fireEvent.change(spinbuttons[0], { target: { value: '2' } });
    fireEvent.change(spinbuttons[1], { target: { value: '5' } });
    fireEvent.change(spinbuttons[2], { target: { value: '65' } });
    fireEvent.change(spinbuttons[3], { target: { value: '70' } });
    fireEvent.click(within(dialog).getByRole('button', { name: /^(OK|确定)$/ }));

    await waitFor(() => {
      expect(actions.save).toHaveBeenCalledWith({
        clusterId: '1',
        existing: [],
        hpa: {
          namespace: 'dev',
          name: 'web',
          target_ref_kind: 'Deployment',
          target_ref_name: 'frontend',
          min_replicas: 2,
          max_replicas: 5,
          cpu_utilization: 65,
          memory_utilization: 70,
        },
      });
    });

    expect(mockApi.kubernetes.createHPA).not.toHaveBeenCalled();
    expect(mockApi.kubernetes.updateHPA).not.toHaveBeenCalled();
  });
});
