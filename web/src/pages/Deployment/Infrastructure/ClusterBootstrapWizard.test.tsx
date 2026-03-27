import { afterEach, describe, expect, it, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import ClusterBootstrapWizard from './ClusterBootstrapWizard';
import { cleanup, renderWithProviders, screen } from '../../../test/utils/render';

const mockApi = vi.hoisted(() => ({
  hosts: {
    getHostList: vi.fn(),
  },
  cluster: {
    getBootstrapVersions: vi.fn(),
    getBootstrapProfiles: vi.fn(),
    getBootstrapTask: vi.fn(),
    previewBootstrap: vi.fn(),
    applyBootstrap: vi.fn(),
  },
}));

vi.mock('../../../api', () => ({ Api: mockApi }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('ClusterBootstrapWizard', () => {
  it('enables next button after filling cluster name in basic info step', async () => {
    const user = userEvent.setup();
    mockApi.hosts.getHostList.mockResolvedValue({ data: { list: [] } });
    mockApi.cluster.getBootstrapVersions.mockResolvedValue({ data: { default_channel: 'stable-1', list: [{ version: '1.28.0', channel: 'local-supported', status: 'supported' }] } });
    mockApi.cluster.getBootstrapProfiles.mockResolvedValue({ data: { list: [], total: 0 } });

    renderWithProviders(<ClusterBootstrapWizard />);

    const nextButton = screen.getByRole('button', { name: '下一步' });
    expect(nextButton).toBeDisabled();

    await user.type(screen.getByLabelText('集群名称'), 'prod-k8s');

    expect(nextButton).toBeEnabled();
  });

  it('loads dynamic versions and profiles on mount', async () => {
    mockApi.hosts.getHostList.mockResolvedValue({ data: { list: [] } });
    mockApi.cluster.getBootstrapVersions.mockResolvedValue({ data: { default_channel: 'stable-1', list: [{ version: '1.28.0', channel: 'stable-1', status: 'supported' }] } });
    mockApi.cluster.getBootstrapProfiles.mockResolvedValue({ data: { list: [{ id: 1, name: 'prod-default', repo_mode: 'mirror', endpoint_mode: 'vip', etcd_mode: 'stacked' }], total: 1 } });

    renderWithProviders(<ClusterBootstrapWizard />);

    expect(mockApi.cluster.getBootstrapVersions).toHaveBeenCalledTimes(1);
    expect(mockApi.cluster.getBootstrapProfiles).toHaveBeenCalledTimes(1);
  });
});
