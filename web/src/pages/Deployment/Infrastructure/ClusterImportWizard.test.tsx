import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import ClusterImportWizard from './ClusterImportWizard';
import { cleanup, fireEvent, renderWithProviders, screen, waitFor } from '../../../test/utils/render';

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
  beforeEach(() => {
    vi.clearAllMocks();
  });

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

    await user.click(screen.getByRole('radio', { name: /API 地址 \+ 证书/ }));
    await user.click(screen.getByRole('button', { name: '下一步' }));

    fireEvent.change(screen.getByLabelText('API Server 地址'), { target: { value: 'https://k8s.example.com:6443' } });
    fireEvent.change(screen.getByLabelText('CA 证书'), { target: { value: 'ca' } });
    fireEvent.change(screen.getByLabelText('客户端证书'), { target: { value: 'cert' } });
    fireEvent.change(screen.getByLabelText('客户端私钥'), { target: { value: 'key' } });
    await user.click(screen.getByRole('button', { name: '下一步' }));

    await user.click(screen.getByRole('button', { name: /测试连接/ }));

    expect(mockApi.cluster.validateImport).toHaveBeenCalledWith({
      name: 'prod-k8s',
      auth_method: 'certificate',
      endpoint: 'https://k8s.example.com:6443',
      ca_cert: 'ca',
      cert: 'cert',
      key: 'key',
    });
  }, 90000);

  it('includes skip_tls_verify when importing with token auth', async () => {
    const user = userEvent.setup();
    mockApi.cluster.validateImport.mockResolvedValue({
      data: { valid: true, message: 'ok', endpoint: 'https://k8s.example.com:6443', version: 'v1.28.0' },
    });
    mockApi.cluster.importCluster.mockResolvedValue({
      data: { id: 1, name: 'prod-k8s' },
    });

    renderWithProviders(<ClusterImportWizard />);

    await user.type(screen.getByLabelText('集群名称'), 'prod-k8s');
    await user.click(screen.getByRole('button', { name: '下一步' }));

    await user.click(screen.getByRole('radio', { name: /ServiceAccount Token/ }));
    await user.click(screen.getByRole('button', { name: '下一步' }));

    fireEvent.change(screen.getByLabelText('API Server 地址'), { target: { value: 'https://k8s.example.com:6443' } });
    fireEvent.change(screen.getByLabelText('Bearer Token'), { target: { value: 'token-value' } });
    await user.click(screen.getByRole('checkbox', { name: '跳过 TLS 证书验证（不推荐）' }));
    await user.click(screen.getByRole('button', { name: '下一步' }));

    await user.click(screen.getByRole('button', { name: /测试连接/ }));
    await waitFor(() => expect(mockApi.cluster.validateImport).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByRole('button', { name: '下一步' })).toBeEnabled());
    await user.click(screen.getByRole('button', { name: '下一步' }));
    await user.click(screen.getByRole('button', { name: '确认导入' }));

    await waitFor(() => expect(mockApi.cluster.importCluster).toHaveBeenCalledWith({
      name: 'prod-k8s',
      description: undefined,
      auth_method: 'token',
      endpoint: 'https://k8s.example.com:6443',
      ca_cert: undefined,
      token: 'token-value',
      skip_tls_verify: true,
    }));
  }, 90000);

  it('uses kubeconfig payload when validating connection', async () => {
    const user = userEvent.setup();
    mockApi.cluster.validateImport.mockResolvedValue({
      data: { valid: true, message: 'ok' },
    });

    renderWithProviders(<ClusterImportWizard />);

    await user.type(screen.getByLabelText('集群名称'), 'prod-k8s');
    await user.click(screen.getByRole('button', { name: '下一步' }));
    await user.click(screen.getByRole('button', { name: '下一步' }));
    fireEvent.change(screen.getByLabelText('Kubeconfig 内容'), { target: { value: 'apiVersion: v1' } });
    await user.click(screen.getByRole('button', { name: '下一步' }));
    await user.click(screen.getByRole('button', { name: /测试连接/ }));
    await waitFor(() => expect(mockApi.cluster.validateImport).toHaveBeenCalled());

    expect(mockApi.cluster.validateImport).toHaveBeenCalledWith({
      name: 'prod-k8s',
      auth_method: 'kubeconfig',
      kubeconfig: 'apiVersion: v1',
    });
  });

  it('uses kubeconfig payload when confirming import', async () => {
    const user = userEvent.setup();
    mockApi.cluster.validateImport.mockResolvedValue({
      data: { valid: true, message: 'ok', endpoint: 'https://k8s.example.com:6443', version: 'v1.28.0' },
    });
    mockApi.cluster.importCluster.mockResolvedValue({
      data: { id: 2, name: 'prod-k8s' },
    });

    renderWithProviders(<ClusterImportWizard />);

    await user.type(screen.getByLabelText('集群名称'), 'prod-k8s');
    await user.click(screen.getByRole('button', { name: '下一步' }));
    await user.click(screen.getByRole('button', { name: '下一步' }));
    fireEvent.change(screen.getByLabelText('Kubeconfig 内容'), { target: { value: 'apiVersion: v1' } });
    await user.click(screen.getByRole('button', { name: '下一步' }));
    await user.click(screen.getByRole('button', { name: /测试连接/ }));
    await waitFor(() => expect(mockApi.cluster.validateImport).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByRole('button', { name: '下一步' })).toBeEnabled());
    await user.click(screen.getByRole('button', { name: '下一步' }));
    await user.click(screen.getByRole('button', { name: '确认导入' }));

    await waitFor(() => expect(mockApi.cluster.importCluster).toHaveBeenCalledWith({
      name: 'prod-k8s',
      description: undefined,
      auth_method: 'kubeconfig',
      kubeconfig: 'apiVersion: v1',
    }));
  }, 30000);

  it('shows endpoint guidance when certificate auth fields receive focus', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ClusterImportWizard />);

    await user.type(screen.getByLabelText('集群名称'), 'prod-k8s');
    await user.click(screen.getByRole('button', { name: '下一步' }));
    await user.click(screen.getByRole('radio', { name: /API 地址 \+ 证书/ }));
    await user.click(screen.getByRole('button', { name: '下一步' }));

    const endpointInput = screen.getByLabelText('API Server 地址');
    fireEvent.focus(endpointInput);

    expect(screen.getByText('这里填什么')).toBeInTheDocument();
    expect(screen.getByText('填写目标集群 Kubernetes API Server 的完整 HTTPS 地址。')).toBeInTheDocument();

    fireEvent.blur(endpointInput);

    await waitFor(() => {
      expect(screen.queryByText('填写目标集群 Kubernetes API Server 的完整 HTTPS 地址。')).not.toBeInTheDocument();
    });
  });

  it('shows insecure TLS guidance when the token checkbox receives focus', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ClusterImportWizard />);

    await user.type(screen.getByLabelText('集群名称'), 'prod-k8s');
    await user.click(screen.getByRole('button', { name: '下一步' }));
    await user.click(screen.getByRole('radio', { name: /ServiceAccount Token/ }));
    await user.click(screen.getByRole('button', { name: '下一步' }));

    const skipTlsCheckbox = screen.getByRole('checkbox', { name: '跳过 TLS 证书验证（不推荐）' });
    fireEvent.focus(skipTlsCheckbox);

    expect(screen.getByText('只在测试环境或临时排障时启用。')).toBeInTheDocument();
    expect(screen.getByText('开启后虽然可能绕过证书问题，但也会放大中间人攻击风险。')).toBeInTheDocument();

    fireEvent.blur(skipTlsCheckbox);

    await waitFor(() => {
      expect(screen.queryByText('只在测试环境或临时排障时启用。')).not.toBeInTheDocument();
    });
  });

  it('includes explicit skip_tls_verify false when importing with token auth and checkbox untouched', async () => {
    const user = userEvent.setup();
    mockApi.cluster.validateImport.mockResolvedValue({
      data: { valid: true, message: 'ok', endpoint: 'https://k8s.example.com:6443', version: 'v1.28.0' },
    });
    mockApi.cluster.importCluster.mockResolvedValue({
      data: { id: 3, name: 'prod-k8s' },
    });

    renderWithProviders(<ClusterImportWizard />);

    await user.type(screen.getByLabelText('集群名称'), 'prod-k8s');
    await user.click(screen.getByRole('button', { name: '下一步' }));

    await user.click(screen.getByRole('radio', { name: /ServiceAccount Token/ }));
    await user.click(screen.getByRole('button', { name: '下一步' }));

    fireEvent.change(screen.getByLabelText('API Server 地址'), { target: { value: 'https://k8s.example.com:6443' } });
    fireEvent.change(screen.getByLabelText('Bearer Token'), { target: { value: 'token-value' } });
    await user.click(screen.getByRole('button', { name: '下一步' }));

    await user.click(screen.getByRole('button', { name: /测试连接/ }));
    await waitFor(() => expect(mockApi.cluster.validateImport).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByRole('button', { name: '下一步' })).toBeEnabled());
    await user.click(screen.getByRole('button', { name: '下一步' }));
    await user.click(screen.getByRole('button', { name: '确认导入' }));

    await waitFor(() => expect(mockApi.cluster.importCluster).toHaveBeenCalledWith({
      name: 'prod-k8s',
      description: undefined,
      auth_method: 'token',
      endpoint: 'https://k8s.example.com:6443',
      ca_cert: undefined,
      token: 'token-value',
      skip_tls_verify: false,
    }));
  }, 90000);
});
