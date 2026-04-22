import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { scopeStore } from '../../app/scope/scopeStore';
import ServiceProvisionPage from './ServiceProvisionPage';

const mockApi = vi.hoisted(() => ({
  projects: {
    list: vi.fn(),
  },
  rbac: {
    checkPermission: vi.fn(),
  },
  services: {
    preview: vi.fn(),
    create: vi.fn(),
    upsertVariableValues: vi.fn(),
    transform: vi.fn(),
  },
}));

vi.mock('../../api', () => ({ Api: mockApi }));
vi.mock('@monaco-editor/react', () => ({
  default: () => null,
}));

describe('ServiceProvisionPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    scopeStore.clearScope();
    scopeStore.setScope({ projectId: '88' });
    mockApi.projects.list.mockResolvedValue({
      data: { list: [{ id: '88', name: 'Platform', key: 'platform', ownerUserId: 1, status: 'active' }] },
    });
    mockApi.rbac.checkPermission.mockResolvedValue({ data: { hasPermission: false } });
    mockApi.services.preview.mockResolvedValue({
      data: { resolved_yaml: '', diagnostics: [], detected_vars: [], unresolved_vars: [] },
    });
    mockApi.services.create.mockResolvedValue({ data: { id: 101 } });
    mockApi.services.upsertVariableValues.mockResolvedValue({ data: {} });
  });

  it('uses ScopeStore project selection when creating a service', async () => {
    render(
      <MemoryRouter>
        <ServiceProvisionPage />
      </MemoryRouter>
    );

    fireEvent.change(screen.getByLabelText('服务名'), { target: { value: 'payments' } });
    fireEvent.change(screen.getByLabelText('负责人'), { target: { value: 'platform' } });
    fireEvent.change(screen.getByLabelText('镜像'), { target: { value: 'ghcr.io/example/payments:1.0.0' } });
    fireEvent.click(screen.getByRole('button', { name: /创建服务/ }));

    await waitFor(() => {
      expect(mockApi.services.create).toHaveBeenCalledWith(expect.objectContaining({ project_id: 88 }));
    });
  });
});
