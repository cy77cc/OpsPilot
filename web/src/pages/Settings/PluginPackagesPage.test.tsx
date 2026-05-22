import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, renderWithProviders, screen } from '../../test/utils/render';
import PluginPackagesPage from './PluginPackagesPage';

const mockApi = vi.hoisted(() => ({
  hosts: {
    listPluginPackages: vi.fn(),
    uploadPluginPackage: vi.fn(),
    deletePluginPackage: vi.fn(),
  },
}));

vi.mock('../../api', () => ({
  Api: mockApi,
}));

describe('PluginPackagesPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.hosts.listPluginPackages.mockResolvedValue({ data: [] });
  });

  afterEach(() => {
    cleanup();
  });

  it('renders the page title', async () => {
    renderWithProviders(<PluginPackagesPage />);
    expect(await screen.findByText('插件包管理')).toBeInTheDocument();
  });

  it('renders upload button', async () => {
    renderWithProviders(<PluginPackagesPage />);
    expect(await screen.findByText('上传安装包')).toBeInTheDocument();
  });

  it('calls listPluginPackages on mount', async () => {
    renderWithProviders(<PluginPackagesPage />);
    await screen.findByText('插件包管理');
    expect(mockApi.hosts.listPluginPackages).toHaveBeenCalledTimes(1);
  });
});
