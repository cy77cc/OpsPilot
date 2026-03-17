import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import AppLayout from './AppLayout';

const mockUseAuth = vi.hoisted(() => vi.fn());
const mockUsePermission = vi.hoisted(() => vi.fn());
const mockUseI18n = vi.hoisted(() => vi.fn());
const mockAiApi = vi.hoisted(() => ({
  getSessions: vi.fn(async () => ({ data: [] })),
  getScenePrompts: vi.fn(async () => ({ data: { prompts: [] } })),
  getSession: vi.fn(async () => ({ data: { messages: [] } })),
  createSession: vi.fn(),
  chatStream: vi.fn(),
}));

vi.mock('../Auth/AuthContext', () => ({
  useAuth: mockUseAuth,
}));

vi.mock('../RBAC', () => ({
  usePermission: mockUsePermission,
}));

vi.mock('../../i18n', () => ({
  useI18n: mockUseI18n,
}));

vi.mock('../../api/modules/ai', () => ({
  aiApi: mockAiApi,
}));

vi.mock('../Project/ProjectSwitcher', () => ({
  default: () => <div data-testid="project-switcher" />,
}));

vi.mock('../Notification', () => ({
  NotificationBell: () => <div data-testid="notification-bell" />,
}));

describe('AppLayout governance menu', () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  const renderWithRouter = () => render(
    <MemoryRouter>
      <AppLayout>
        <div>content</div>
      </AppLayout>
    </MemoryRouter>,
  );

  it('shows governance menu for users with rbac read permission', () => {
    mockUseAuth.mockReturnValue({ logout: vi.fn() });
    mockUsePermission.mockReturnValue({ hasPermission: vi.fn(() => true) });
    mockUseI18n.mockReturnValue({
      t: (key: string) => key,
      lang: 'zh-CN',
      setLang: vi.fn(),
    });

    renderWithRouter();

    expect(screen.getByText('访问治理')).toBeInTheDocument();
  });

  it('hides governance menu for users without rbac read permission', () => {
    mockUseAuth.mockReturnValue({ logout: vi.fn() });
    mockUsePermission.mockReturnValue({ hasPermission: vi.fn(() => false) });
    mockUseI18n.mockReturnValue({
      t: (key: string) => key,
      lang: 'zh-CN',
      setLang: vi.fn(),
    });

    renderWithRouter();

    expect(screen.queryByText('访问治理')).not.toBeInTheDocument();
  });

  it('opens the copilot drawer from the shell entry', async () => {
    mockUseAuth.mockReturnValue({ logout: vi.fn() });
    mockUsePermission.mockReturnValue({ hasPermission: vi.fn(() => true) });
    mockUseI18n.mockReturnValue({
      t: (key: string) => key,
      lang: 'zh-CN',
      setLang: vi.fn(),
    });

    renderWithRouter();

    await userEvent.click(screen.getByRole('button', { name: /AI Assistant/i }));

    expect(await screen.findByText('AI Copilot')).toBeInTheDocument();
  });
});
