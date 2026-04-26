import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import HostDetailPage from './index';

const mockHost = {
  id: '1',
  name: 'test-host',
  ip: '192.168.1.10',
  status: 'online',
  cpu: 4,
  memory: 8192,
  disk: 200,
  tags: ['web', 'prod'],
  os: 'Ubuntu',
  osVersion: '22.04',
  lastActive: new Date().toISOString(),
  createdAt: new Date().toISOString(),
};

const mockApi = vi.hoisted(() => ({
  hosts: {
    getHostDetail: vi.fn(),
    getHostMetrics: vi.fn(),
    listSSHKeys: vi.fn(),
    hostAction: vi.fn(),
    runHealthCheck: vi.fn(),
    updateHost: vi.fn(),
    updateCredentials: vi.fn(),
    sshCheck: vi.fn(),
  },
}));

vi.mock('../../../api', () => ({
  Api: mockApi,
}));

// Mock ResizeObserver which is needed by recharts
global.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
};

describe('HostDetailPage New Design', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.hosts.getHostDetail.mockResolvedValue({ data: mockHost });
    mockApi.hosts.getHostMetrics.mockResolvedValue({ data: [] });
    mockApi.hosts.listSSHKeys.mockResolvedValue({ data: [] });
  });

  const renderPage = () => {
    return render(
      <MemoryRouter initialEntries={['/hosts/1']}>
        <Routes>
          <Route path="/hosts/:id" element={<HostDetailPage />} />
        </Routes>
      </MemoryRouter>
    );
  };

  it('renders the new header structure', async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getAllByText('test-host').length).toBeGreaterThan(0);
      expect(screen.getByText('运行中')).toBeInTheDocument();
      expect(screen.getAllByText('192.168.1.10').length).toBeGreaterThan(0);
      expect(screen.getByText('返回主机列表')).toBeInTheDocument();
    });
  });

  it('renders all designed tabs', async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getAllByText('概览').length).toBeGreaterThan(0);
      expect(screen.getAllByText('监控').length).toBeGreaterThan(0);
      expect(screen.getAllByText('进程').length).toBeGreaterThan(0);
      expect(screen.getAllByText('服务').length).toBeGreaterThan(0);
      expect(screen.getAllByText('磁盘').length).toBeGreaterThan(0);
      expect(screen.getAllByText('网络').length).toBeGreaterThan(0);
      expect(screen.getAllByText('软件包').length).toBeGreaterThan(0);
      expect(screen.getAllByText('配置').length).toBeGreaterThan(0);
      expect(screen.getAllByText('告警').length).toBeGreaterThan(0);
      expect(screen.getAllByText('操作记录').length).toBeGreaterThan(0);
    });
  });

  it('renders basic info card content in overview tab', async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getAllByText('基本信息').length).toBeGreaterThan(0);
      expect(screen.getAllByText('Ubuntu').length).toBeGreaterThan(0);
      expect(screen.getAllByText('生产集群').length).toBeGreaterThan(0);
    });
  });

  it('renders resource summary card in overview tab', async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getAllByText('资源使用率').length).toBeGreaterThan(0);
      expect(screen.getAllByText('CPU 使用率').length).toBeGreaterThan(0);
      expect(screen.getAllByText('内存使用率').length).toBeGreaterThan(0);
      expect(screen.getAllByText('磁盘使用率').length).toBeGreaterThan(0);
    });
  });

  it('switches to other tabs and shows placeholder', async () => {
    renderPage();
    await waitFor(() => {
      const monitorTabs = screen.getAllByText('监控');
      // Find the one in the tab list (it usually has role="tab" or is inside a tab-btn)
      const monitorTab = monitorTabs.find(el => el.getAttribute('role') === 'tab' || el.classList.contains('ant-tabs-tab-btn'));
      if (monitorTab) {
        fireEvent.click(monitorTab);
      }
    });
    expect(await screen.findByText('监控 模块正在开发中')).toBeInTheDocument();
  });
});
