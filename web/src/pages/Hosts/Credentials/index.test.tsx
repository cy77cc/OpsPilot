import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import CredentialsPage from './index';

const mockHostApi = vi.hoisted(() => ({
  getCredentialStats: vi.fn(),
  getCredentials: vi.fn(),
  getCredentialDetail: vi.fn(),
  getCredentialUsageRecords: vi.fn(),
  getCredentialPermissions: vi.fn(),
  listCredentialTemplates: vi.fn(),
  deleteSSHKey: vi.fn(),
  deleteCredentialTemplate: vi.fn(),
}));

vi.mock('../../../api/modules/hosts', () => ({
  hostApi: mockHostApi,
}));

describe('CredentialsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockHostApi.getCredentialStats.mockResolvedValue({
      success: true,
      data: {
        total: 2,
        available: 2,
        expiringSoon: 0,
        expired: 0,
        recentUpdate: '2024-05-12',
        recentUpdateBy: 'admin',
      },
    });
    mockHostApi.getCredentials.mockResolvedValue({
      success: true,
      data: {
        list: [
          {
            id: 'key-1',
            name: 'prod-ssh-key',
            description: '生产环境 SSH 密钥',
            type: 'ssh_key',
            authMethod: 'SSH Key',
            hostCount: 120,
            tags: ['生产', '核心'],
            status: 'available',
            expireAt: '2025-05-20',
            updatedAt: '2024-05-12',
            updatedBy: 'admin',
          },
          {
            id: 'tpl-2',
            name: 'dev-password',
            description: '开发环境密码',
            type: 'password',
            authMethod: '用户名密码',
            hostCount: 45,
            tags: ['开发'],
            status: 'available',
            expireAt: '2025-08-15',
            updatedAt: '2024-05-10',
            updatedBy: 'admin',
          },
        ],
      },
    });
    mockHostApi.getCredentialDetail.mockResolvedValue({
      success: true,
      data: {
        id: 'key-1',
        name: 'prod-ssh-key',
        description: '生产环境服务器访问密钥',
        type: 'ssh_key',
        authMethod: 'SSH Key',
        hostCount: 120,
        tags: ['生产', '核心'],
        status: 'available',
        expireAt: '2025-05-20',
        updatedAt: '2024-05-12 15:20:30',
        updatedBy: 'admin',
        createdAt: '2024-04-01 10:30:00',
        createdBy: 'admin',
        secret: 'ssh-ed25519 AAAATEST',
        usageCount: 1024,
        successCount: 1012,
        failureCount: 12,
        successRate: 98.8,
        recentUsage: '2024-05-12 14:30:22',
      },
    });
    mockHostApi.getCredentialUsageRecords.mockResolvedValue({ success: true, data: { list: [], total: 0 } });
    mockHostApi.getCredentialPermissions.mockResolvedValue({ success: true, data: { list: [], total: 0 } });
    mockHostApi.listCredentialTemplates.mockResolvedValue({ success: true, data: [] });
  });

  it('selects the first credential by default and renders a persistent detail side panel', async () => {
    render(
      <MemoryRouter initialEntries={['/resources/hosts/credentials']}>
        <Routes>
          <Route path="/resources/hosts/credentials" element={<CredentialsPage />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('prod-ssh-key')).toBeInTheDocument();
    });

    await waitFor(() => {
      expect(mockHostApi.getCredentialDetail).toHaveBeenCalledWith('key-1');
    });

    expect(screen.getByText('凭证详情')).toBeInTheDocument();
    expect(screen.getByText('基本信息')).toBeInTheDocument();
    expect(screen.getByText('生产环境服务器访问密钥')).toBeInTheDocument();
  });
});
