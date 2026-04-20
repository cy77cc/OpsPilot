import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { cleanup, fireEvent, renderWithProviders, screen, waitFor } from '../../test/utils/render';
import AIModelSettingsPage from './AIModelSettingsPage';

const mockApi = vi.hoisted(() => ({
  ai: {
    listAdminModels: vi.fn(),
    createAdminModel: vi.fn(),
    updateAdminModel: vi.fn(),
    setAdminDefaultModel: vi.fn(),
    deleteAdminModel: vi.fn(),
    previewAdminModelImport: vi.fn(),
    importAdminModels: vi.fn(),
  },
}));

vi.mock('../../api', () => ({
  Api: mockApi,
}));

describe('AIModelSettingsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.ai.listAdminModels.mockResolvedValue({ data: { list: [] } });
  });

  afterEach(() => {
    cleanup();
  });

  it('shows base URL guidance in the create drawer', async () => {
    const user = userEvent.setup();

    renderWithProviders(<AIModelSettingsPage />);
    await screen.findByText('AI 模型配置中心');

    await user.click(screen.getByRole('button', { name: /新增模型/ }));

    const baseUrlInput = await screen.findByLabelText('Base URL');
    fireEvent.focus(baseUrlInput);

    expect(screen.getByText('这里填什么')).toBeInTheDocument();
    expect(screen.getByText('填写模型供应商实际提供的接口根地址。')).toBeInTheDocument();

    fireEvent.blur(baseUrlInput);

    await waitFor(() => {
      expect(screen.queryByText('填写模型供应商实际提供的接口根地址。')).not.toBeInTheDocument();
    });
  });
});
