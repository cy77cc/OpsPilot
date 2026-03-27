import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import UsersPage from './UsersPage';

const mockApi = vi.hoisted(() => ({
  rbac: {
    getUserList: vi.fn(),
    getRoleList: vi.fn(),
    createUser: vi.fn(),
    updateUser: vi.fn(),
    deleteUser: vi.fn(),
    recordMigrationEvent: vi.fn(),
  },
}));
const mockCanWrite = vi.hoisted(() => ({ value: true }));

vi.mock('../../api', () => ({ Api: mockApi }));
vi.mock('../../components/RBAC/PermissionContext', () => ({
  usePermission: () => ({ hasPermission: (_resource: string, action: string) => (action === 'write' ? mockCanWrite.value : true) }),
}));

describe('UsersPage accessibility interactions', () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();
    mockCanWrite.value = true;
    mockApi.rbac.getUserList.mockResolvedValue({
      data: {
        list: [
          {
            id: '1',
            username: 'alice',
            name: 'Alice',
            email: 'alice@example.com',
            roles: ['admin'],
            status: 'active',
            createdAt: '2026-01-01T00:00:00Z',
            updatedAt: '2026-01-01T00:00:00Z',
          },
        ],
      },
    });
    mockApi.rbac.getRoleList.mockResolvedValue({
      data: {
        list: [{ id: '10', name: '管理员', code: 'admin', description: '', permissions: [], createdAt: '', updatedAt: '' }],
      },
    });
  });

  it('opens detail drawer with keyboard Enter on interactive row', async () => {
    render(<UsersPage />);

    await waitFor(() => expect(screen.getAllByText('alice').length).toBeGreaterThan(0));

    const rowTrigger = screen.getAllByLabelText('查看用户 alice 详情').find((element) => element.tagName.toLowerCase() !== 'button');
    expect(rowTrigger).toBeDefined();
    if (!rowTrigger) {
      throw new Error('row trigger not found');
    }
    fireEvent.keyDown(rowTrigger, { key: 'Enter' });

    expect(await screen.findByText('用户详情')).toBeInTheDocument();
    const dialogs = screen.getAllByRole('dialog');
    const detailDialog = dialogs[dialogs.length - 1];
    expect(within(detailDialog).getByText('alice@example.com')).toBeInTheDocument();
  });

  it('shows explicit edit action in user operations', async () => {
    render(<UsersPage />);

    await waitFor(() => expect(screen.getAllByText('alice').length).toBeGreaterThan(0));

    expect(screen.getAllByLabelText('编辑用户 alice').length).toBeGreaterThan(0);
  });

  it('hides write actions for read-only users', async () => {
    mockCanWrite.value = false;
    render(<UsersPage />);

    await waitFor(() => expect(screen.getAllByText('alice').length).toBeGreaterThan(0));

    const rowTrigger = screen.getAllByLabelText('查看用户 alice 详情').find((element) => element.tagName.toLowerCase() !== 'button');
    expect(rowTrigger).toBeDefined();
    if (!rowTrigger) {
      throw new Error('row trigger not found');
    }
    const row = rowTrigger.closest('tr');
    expect(row).toBeTruthy();
    if (!row) {
      throw new Error('row not found');
    }
    expect(within(row).queryByText('编辑')).not.toBeInTheDocument();
    expect(within(row).queryByText('删除')).not.toBeInTheDocument();
  });
});
