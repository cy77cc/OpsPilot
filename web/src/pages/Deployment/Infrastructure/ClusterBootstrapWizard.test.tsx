import { afterEach, describe, expect, it, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import ClusterBootstrapWizard from './ClusterBootstrapWizard';
import { cleanup, renderWithProviders, screen } from '../../../test/utils/render';

const mockApi = vi.hoisted(() => ({
  hosts: {
    getHostList: vi.fn(),
  },
  cluster: {
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

    renderWithProviders(<ClusterBootstrapWizard />);

    const nextButton = screen.getByRole('button', { name: '下一步' });
    expect(nextButton).toBeDisabled();

    await user.type(screen.getByLabelText('集群名称'), 'prod-k8s');

    expect(nextButton).toBeEnabled();
  });
});
