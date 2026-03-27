import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import HostDetailPage from './HostDetailPage';

const mockApi = vi.hoisted(() => ({
  hosts: {
    getHostDetail: vi.fn(),
    getHostMetrics: vi.fn(),
    getHostAudits: vi.fn(),
    runHealthCheck: vi.fn(),
    hostAction: vi.fn(),
    updateHost: vi.fn(),
    updateCredentials: vi.fn(),
    sshCheck: vi.fn(),
    listSSHKeys: vi.fn(),
    createSSHKey: vi.fn(),
  },
}));

vi.mock('../../api', () => ({ Api: mockApi }));

describe('HostDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.hosts.getHostDetail.mockResolvedValue({
      data: {
        id: '1',
        name: 'node-01',
        ip: '10.0.0.1',
        status: 'maintenance',
        healthState: 'critical',
        maintenanceReason: 'disk replace',
        maintenanceStartedAt: new Date().toISOString(),
        username: 'root',
        port: 22,
        cpu: 4,
        memory: 8192,
        disk: 200,
        tags: ['prod'],
      },
    });
    mockApi.hosts.getHostMetrics.mockResolvedValue({
      data: [{ id: 1, cpu: 60, memory: 72, disk: 81, network: 10, createdAt: new Date().toISOString() }],
    });
    mockApi.hosts.getHostAudits.mockResolvedValue({ data: [] });
    mockApi.hosts.runHealthCheck.mockResolvedValue({
      data: {
        state: 'degraded',
        connectivityStatus: 'healthy',
        resourceStatus: 'degraded',
        systemStatus: 'healthy',
        latencyMs: 88,
      },
    });
  });

  it('shows health and maintenance metadata, and renders health-check result modal', async () => {
    render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/hosts/1']}>
        <Routes>
          <Route path="/deployment/infrastructure/hosts/:id" element={<HostDetailPage />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('健康状态: critical')).toBeInTheDocument();
      expect(screen.getByText('维护信息: disk replace')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: '健康检查' }));
    await waitFor(() => {
      expect(screen.getAllByText('健康检查结果').length).toBeGreaterThan(0);
      expect(screen.getByText('连通性')).toBeInTheDocument();
      expect(screen.getAllByText('healthy').length).toBeGreaterThan(0);
    });
  }, 15000);
});
