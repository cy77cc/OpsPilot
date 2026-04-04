import type { ReactNode } from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';

const mockUseAuth = vi.hoisted(() => vi.fn());

vi.mock('./components/Auth/AuthContext', () => ({
  useAuth: mockUseAuth,
}));

vi.mock('./contexts/NotificationContext', () => ({
  NotificationProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock('./components/RBAC', () => ({
  PermissionProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
  Authorized: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock('./components/Motion', () => ({
  PageTransition: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock('./components/Auth/AccessDeniedPage', () => ({
  default: () => <div>access-denied</div>,
}));

vi.mock('./components/Auth/LegacyGovernanceRedirect', () => ({
  default: () => <div>legacy-governance-redirect</div>,
}));

vi.mock('./components/Layout/AppLayout', () => ({
  default: ({ children }: { children: ReactNode }) => (
    <div>
      <div data-testid="app-shell">shell</div>
      <main>{children}</main>
    </div>
  ),
}));

vi.mock('./pages/Dashboard/Dashboard', async () => {
  await new Promise((resolve) => setTimeout(resolve, 200));
  return { default: () => <div data-testid="dashboard-content">dashboard-content</div> };
});

import ProtectedApp from './ProtectedApp';

describe('ProtectedApp loading boundary', () => {
  it('keeps the app shell mounted while route content is suspended', async () => {
    mockUseAuth.mockReturnValue({ user: { id: 'user-1' } });

    render(
      <MemoryRouter initialEntries={['/']}>
        <ProtectedApp />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('app-shell')).toBeInTheDocument();
      expect(screen.getByTestId('page-skeleton')).toBeInTheDocument();
    });

    expect(await screen.findByTestId('dashboard-content')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByTestId('app-shell')).toBeInTheDocument();
      expect(screen.queryByTestId('page-skeleton')).not.toBeInTheDocument();
    });
  });
});
