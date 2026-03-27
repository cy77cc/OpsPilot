import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import ServiceListPage from './ServiceListPage';
import ServiceProvisionPage from './ServiceProvisionPage';

const mockGetList = vi.fn();
const mockProjectsList = vi.fn();
const mockCheckPermission = vi.fn();
const mockPreview = vi.fn();

vi.mock('@monaco-editor/react', () => ({
  default: (props: { value?: string }) => <pre data-testid="monaco-mock">{props.value || ''}</pre>,
}));

vi.mock('../../api', () => ({
  Api: {
    services: {
      getList: (...args: unknown[]) => mockGetList(...args),
      preview: (...args: unknown[]) => mockPreview(...args),
      transform: vi.fn(),
      create: vi.fn(),
    },
    projects: {
      list: (...args: unknown[]) => mockProjectsList(...args),
    },
    rbac: {
      checkPermission: (...args: unknown[]) => mockCheckPermission(...args),
    },
  },
}));

describe('Service pages interaction regression coverage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    mockPreview.mockResolvedValue({
      data: {
        resolved_yaml: 'apiVersion: v1\nkind: ConfigMap',
        diagnostics: [],
        detected_vars: [],
        unresolved_vars: [],
      },
    });
  });

  it('renders table row stop action and sortable service name column', async () => {
    localStorage.setItem('service-list-view-mode', 'list');
    mockGetList.mockResolvedValue({
      data: {
        list: [
          { id: '1', name: 'z-service', env: 'staging', runtimeType: 'k8s', owner: 'ops', status: 'running', labels: [] },
          { id: '2', name: 'a-service', env: 'staging', runtimeType: 'k8s', owner: 'ops', status: 'draft', labels: [] },
        ],
      },
    });

    render(
      <MemoryRouter initialEntries={['/services']}>
        <Routes>
          <Route path="/services" element={<ServiceListPage />} />
        </Routes>
      </MemoryRouter>
    );

    await screen.findByText('服务管理');
    const stopActions = await screen.findAllByText('停止');
    expect(stopActions.length).toBeGreaterThan(0);

    const nameHeader = screen.getByRole('columnheader', { name: '服务名' });
    fireEvent.click(nameHeader);
    await waitFor(() => {
      expect(nameHeader).toHaveAttribute('aria-sort');
    });
  });

  it('shows localized environment labels on provision page', async () => {
    mockProjectsList.mockResolvedValue({ data: { list: [{ id: '1', name: '项目A' }] } });
    mockCheckPermission.mockResolvedValue({ data: { hasPermission: true } });

    render(
      <MemoryRouter initialEntries={['/services/provision']}>
        <Routes>
          <Route path="/services/provision" element={<ServiceProvisionPage />} />
        </Routes>
      </MemoryRouter>
    );

    await screen.findByText('服务工作室');
    const envInput = screen.getByLabelText('环境');
    fireEvent.mouseDown(envInput);

    await screen.findByText('开发环境');
    expect(screen.getAllByText('测试环境').length).toBeGreaterThan(0);
    expect(screen.getAllByText('生产环境').length).toBeGreaterThan(0);
  });
});
