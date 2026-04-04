import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import DeploymentListPage from './DeploymentListPage';

const mockNavigate = vi.fn();
const mockGetReleasesByRuntime = vi.fn();
const mockGetServiceList = vi.fn();
const mockListTargets = vi.fn();

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

vi.mock('../../api', () => ({
  Api: {
    deployment: {
      getReleasesByRuntime: (...args: unknown[]) => mockGetReleasesByRuntime(...args),
      listTargets: (...args: unknown[]) => mockListTargets(...args),
      rollbackRelease: vi.fn(),
    },
    services: {
      getList: (...args: unknown[]) => mockGetServiceList(...args),
    },
  },
}));

vi.mock('../../components/Motion', () => ({
  StaggerList: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  StaggerItem: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

describe('DeploymentListPage loading behavior', () => {
  it('keeps stale content visible during refresh', async () => {
    const unresolvedSecondLoad = new Promise(() => undefined);

    mockGetReleasesByRuntime
      .mockResolvedValueOnce({
        data: {
          list: [
            {
              id: 1,
              service_id: 10,
              target_id: 20,
              runtime_type: 'k8s',
              status: 'succeeded',
              created_at: '2026-04-04T00:00:00Z',
            },
          ],
        },
      })
      .mockReturnValueOnce(unresolvedSecondLoad);
    mockGetServiceList.mockResolvedValue({ data: { list: [] } });
    mockListTargets.mockResolvedValue({ data: { list: [] } });

    render(
      <MemoryRouter>
        <DeploymentListPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText('Release #1')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /刷新/ }));

    expect(screen.getByText('Release #1')).toBeInTheDocument();
    expect(screen.queryByTestId('card-grid-skeleton')).not.toBeInTheDocument();
  });
});
