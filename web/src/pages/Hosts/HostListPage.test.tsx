import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import HostListPage from './HostListPage';

vi.mock('@ant-design/charts', () => ({
  Pie: () => <div data-testid="mock-pie-chart" />,
  Line: () => <div data-testid="mock-line-chart" />,
}));

const mockApi = vi.hoisted(() => ({
  hosts: {
    getHostList: vi.fn(),
    getHostOverview: vi.fn(),
    getHostDistribution: vi.fn(),
    getHostUsageTrend: vi.fn(),
    getHostPendingAlerts: vi.fn(),
    batchUpdate: vi.fn(),
    runHealthCheck: vi.fn(),
    hostAction: vi.fn(),
    deleteHost: vi.fn(),
  },
}));

vi.mock('../../api', () => ({ Api: mockApi }));

describe('HostListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.hosts.getHostList.mockResolvedValue({
      data: {
        list: [
          {
            id: '1',
            name: 'prod-app-01',
            ip: '10.0.1.21',
            status: 'online',
            healthState: 'healthy',
            cpu: 28,
            memory: 45,
            disk: 62,
            tags: ['prod', 'cluster:prod-cluster', 'project:支付服务'],
            region: '华东',
            os: 'Ubuntu 22.04',
            createdAt: '2026-04-22T12:00:00Z',
            lastActive: '2026-04-23T11:59:00Z',
          },
          {
            id: '2',
            name: 'test-win-01',
            ip: '10.0.4.15',
            status: 'offline',
            healthState: 'critical',
            cpu: 18,
            memory: 33,
            disk: 36,
            tags: ['test', 'windows'],
            region: '华北',
            os: 'Windows Server 2019',
            createdAt: '2026-04-22T12:00:00Z',
            lastActive: '2026-04-23T11:58:00Z',
          },
        ],
      },
    });
    mockApi.hosts.getHostOverview.mockResolvedValue({
      data: {
        totalHosts: 2,
        onlineHosts: 1,
        abnormalHosts: 1,
        avgCpuUsage: 23,
        avgMemoryUsage: 39,
        todayAlertCount: 3,
        severeAlertCount: 1,
        warningAlertCount: 2,
        onlineRate: 50,
      },
    });
    mockApi.hosts.getHostDistribution.mockResolvedValue({
      data: [
        { name: '生产', value: 1, percent: 50 },
        { name: '测试', value: 1, percent: 50 },
      ],
    });
    mockApi.hosts.getHostUsageTrend.mockResolvedValue({
      data: [
        { time: '10:00', cpuUsage: 20, memoryUsage: 35 },
        { time: '11:00', cpuUsage: 23, memoryUsage: 39 },
      ],
    });
    mockApi.hosts.getHostPendingAlerts.mockResolvedValue({
      data: [
        { name: 'prod-app-01', level: 'warning', count: 2 },
        { name: 'test-win-01', level: 'critical', count: 1 },
      ],
    });
  });

  it('renders KPI cards, table rows and right-side charts', async () => {
    render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/hosts']}>
        <Routes>
          <Route path="/deployment/infrastructure/hosts" element={<HostListPage />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('主机总数')).toBeInTheDocument();
      expect(screen.getAllByText('prod-app-01').length).toBeGreaterThan(0);
      expect(screen.getAllByText('test-win-01').length).toBeGreaterThan(0);
    });

    expect(screen.getByTestId('mock-pie-chart')).toBeInTheDocument();
    expect(screen.getByTestId('mock-line-chart')).toBeInTheDocument();
  });

  it('filters table rows by search keyword', async () => {
    render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/hosts']}>
        <Routes>
          <Route path="/deployment/infrastructure/hosts" element={<HostListPage />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getAllByText('prod-app-01').length).toBeGreaterThan(0);
      expect(screen.getAllByText('test-win-01').length).toBeGreaterThan(0);
    });

    const searchInputs = screen.getAllByPlaceholderText('搜索主机名、IP 或标签');
    fireEvent.change(searchInputs[0], {
      target: { value: 'prod-app-01' },
    });

    await waitFor(() => {
      expect(searchInputs[0]).toHaveValue('prod-app-01');
      expect(screen.getAllByText('prod-app-01').length).toBeGreaterThan(0);
    });
  });
});
