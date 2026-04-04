import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import Authorized from './Authorized';

const mockUsePermission = vi.fn();

vi.mock('./PermissionContext', () => ({
  usePermission: () => mockUsePermission(),
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
});
