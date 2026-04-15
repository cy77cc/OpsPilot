import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import Authorized from './Authorized';

const mockUsePermission = vi.fn();

vi.mock('./PermissionContext', () => ({
  usePermission: () => mockUsePermission(),
  PermissionProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

describe('Authorized', () => {
  it('shows a page skeleton while permissions are loading', () => {
    mockUsePermission.mockReturnValue({
      loading: true,
      permissions: [],
      hasPermission: vi.fn(),
      refreshPermissions: vi.fn(),
    });

    const { container } = render(
      <Authorized resource="cluster" action="read">
        <div>secret content</div>
      </Authorized>
    );

    expect(screen.getByTestId('page-skeleton')).toBeInTheDocument();
    expect(container.querySelector('.ant-spin')).toBeNull();
    expect(screen.queryByText('secret content')).not.toBeInTheDocument();
  });

  it('does not export the removed hard-coded checkPermission helper', async () => {
    const authorizedModule = await import('./Authorized');
    const rbacBarrelModule = await import('./index');

    expect('checkPermission' in authorizedModule).toBe(false);
    expect('checkPermission' in rbacBarrelModule).toBe(false);
    expect('checkPermission' in rbacBarrelModule.default).toBe(false);
  });
});
