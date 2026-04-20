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
    await user.click(baseUrlInput);

    expect(screen.getByText('这里填什么')).toBeInTheDocument();
    expect(screen.getByText('填写模型供应商实际提供的接口根地址。')).toBeInTheDocument();

    await user.tab();

    await waitFor(() => {
      expect(screen.queryByText('填写模型供应商实际提供的接口根地址。')).not.toBeInTheDocument();
    });
  });

  it('submits create payload with guided required fields', async () => {
    const user = userEvent.setup();
    mockApi.ai.createAdminModel.mockResolvedValue({});

    renderWithProviders(<AIModelSettingsPage />);
    await screen.findByText('AI 模型配置中心');

    await user.click(screen.getByRole('button', { name: /新增模型/ }));

    fireEvent.change(await screen.findByLabelText('显示名称'), { target: { value: 'OpenAI 生产模型' } });
    fireEvent.change(screen.getByLabelText('模型标识'), { target: { value: 'gpt-4.1' } });
    fireEvent.change(screen.getByLabelText('Base URL'), { target: { value: 'https://api.openai.com/v1' } });
    fireEvent.change(screen.getByLabelText('API Key'), { target: { value: 'sk-create-test' } });

    await user.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => {
      expect(mockApi.ai.createAdminModel).toHaveBeenCalledTimes(1);
    });
    expect(mockApi.ai.createAdminModel).toHaveBeenCalledWith({
      name: 'OpenAI 生产模型',
      provider: 'qwen',
      model: 'gpt-4.1',
      base_url: 'https://api.openai.com/v1',
      api_key: 'sk-create-test',
      temperature: 0.7,
      thinking: false,
      is_default: false,
      is_enabled: true,
      sort_order: 0,
    });
  });

  it('submits edit payload without overwriting api_key when left blank', async () => {
    const user = userEvent.setup();
    mockApi.ai.listAdminModels.mockResolvedValue({
      data: {
        list: [
          {
            id: 101,
            name: 'Qwen 线上',
            provider: 'qwen',
            model: 'qwen-max',
            base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
            api_key: '',
            temperature: 0.6,
            thinking: true,
            is_default: true,
            is_enabled: true,
            sort_order: 5,
            created_at: '2026-04-20T00:00:00Z',
            updated_at: '2026-04-20T00:00:00Z',
          },
        ],
      },
    });
    mockApi.ai.updateAdminModel.mockResolvedValue({});

    renderWithProviders(<AIModelSettingsPage />);
    await screen.findByText('Qwen 线上');

    await user.click(screen.getByRole('button', { name: 'edit' }));
    await screen.findByText('编辑模型：Qwen 线上');

    const apiKeyInput = screen.getByLabelText('API Key（留空则不修改）');
    await user.clear(apiKeyInput);

    await user.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => {
      expect(mockApi.ai.updateAdminModel).toHaveBeenCalledTimes(1);
    });

    const [id, payload] = mockApi.ai.updateAdminModel.mock.calls[0];
    expect(id).toBe(101);
    expect(payload).toMatchObject({
      name: 'Qwen 线上',
      provider: 'qwen',
      model: 'qwen-max',
      base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
      temperature: 0.6,
      thinking: true,
      is_default: true,
      is_enabled: true,
      sort_order: 5,
    });
    expect(payload).not.toHaveProperty('api_key');
  });
});
