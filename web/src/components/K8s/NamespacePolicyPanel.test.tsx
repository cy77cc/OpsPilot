import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { scopeStore } from '../../app/scope/scopeStore';
import NamespacePolicyPanel from './NamespacePolicyPanel';

const mockApi = vi.hoisted(() => ({
  kubernetes: {
    getClusterNamespaces: vi.fn(),
    getNamespaceBindings: vi.fn(),
    putNamespaceBindings: vi.fn(),
    createNamespace: vi.fn(),
    deleteNamespace: vi.fn(),
  },
}));

vi.mock('../../api', () => ({ Api: mockApi }));

describe('NamespacePolicyPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    scopeStore.clearScope();
    scopeStore.setScope({ teamId: '7' });
    mockApi.kubernetes.getClusterNamespaces.mockResolvedValue({ data: { list: [] } });
    mockApi.kubernetes.getNamespaceBindings.mockResolvedValue({ data: { list: [] } });
    mockApi.kubernetes.putNamespaceBindings.mockResolvedValue({ data: {} });
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
});
