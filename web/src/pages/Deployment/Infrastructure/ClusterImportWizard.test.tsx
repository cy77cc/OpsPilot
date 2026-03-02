import { afterEach, describe, expect, it, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import ClusterImportWizard from './ClusterImportWizard';
import { cleanup, fireEvent, renderWithProviders, screen } from '../../../test/utils/render';

const mockApi = vi.hoisted(() => ({
  cluster: {
    validateImport: vi.fn(),
    importCluster: vi.fn(),
  },
}));

vi.mock('../../../api', () => ({ Api: mockApi }));

afterEach(() => {
  cleanup();
});

describe('ClusterImportWizard', () => {
  it('enables next button after filling basic cluster name', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ClusterImportWizard />);

    const nextButton = screen.getByRole('button', { name: '下一步' });
    expect(nextButton).toBeDisabled();

    await user.type(screen.getByLabelText('集群名称'), 'prod-k8s');

    expect(nextButton).toBeEnabled();
  });

  it('uses certificate payload when testing connection after selecting certificate auth', async () => {
    const user = userEvent.setup();
    mockApi.cluster.validateImport.mockResolvedValue({
      data: { valid: false, message: 'connect failed' },
    });

    renderWithProviders(<ClusterImportWizard />);

    await user.type(screen.getByLabelText('集群名称'), 'prod-k8s');
    await user.click(screen.getByRole('button', { name: '下一步' }));

    await user.click(screen.getByText('API 地址 + 证书'));
    await user.click(screen.getByRole('button', { name: '下一步' }));

    fireEvent.change(screen.getByLabelText('API Server 地址'), { target: { value: 'https://k8s.example.com:6443' } });
    fireEvent.change(screen.getByLabelText('CA 证书'), { target: { value: 'ca' } });
    fireEvent.change(screen.getByLabelText('客户端证书'), { target: { value: 'cert' } });
    fireEvent.change(screen.getByLabelText('客户端私钥'), { target: { value: 'key' } });
    await user.click(screen.getByRole('button', { name: '下一步' }));

    await user.click(screen.getByRole('button', { name: /测试连接/ }));

    expect(mockApi.cluster.validateImport).toHaveBeenCalledWith({
      name: 'prod-k8s',
      endpoint: 'https://k8s.example.com:6443',
      ca_cert: 'ca',
      cert: 'cert',
      key: 'key',
    });
  });
});
