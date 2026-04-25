import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import ClusterOperationCenterPage from './ClusterOperationCenterPage';

const mockApi = vi.hoisted(() => ({
  cluster: {
    getClusterDetail: vi.fn(),
    getClusterOperations: vi.fn(),
    getClusterOperationDetail: vi.fn(),
  },
}));

vi.mock('../../../api', () => ({
  Api: mockApi,
}));

const defaultCluster = {
  id: 42,
  name: 'phase1-cluster',
  description: 'test cluster',
  status: 'active',
  source: 'platform_managed',
  type: 'kubernetes',
  node_count: 1,
  created_at: '2026-04-04T00:00:00Z',
  updated_at: '2026-04-04T00:00:00Z',
};

const historyItem = {
  audit_id: 501,
  action: 'node.drain',
  resource: 'node',
  resource_type: 'node',
  resource_name: 'worker-1',
  target: 'worker-1',
  status: 'approval_required',
  operator: 'alice',
  message: '需要审批',
  created_at: '2026-04-05T10:00:00Z',
  updated_at: '2026-04-05T10:10:00Z',
  namespace: '',
};

const policyReleaseHistoryItem = {
  audit_id: 701,
  action: 'policy.release.apply',
  resource: 'policy_release',
  resource_type: 'policy_release',
  resource_name: 'allow-api',
  resource_id: '501',
  target: 'candidate-v2',
  status: 'completed',
  operator: 'alice',
  message: 'policy release applied',
  created_at: '2026-04-05T11:00:00Z',
  updated_at: '2026-04-05T11:05:00Z',
  namespace: 'prod',
};

function mockBaselineLoads() {
  mockApi.cluster.getClusterDetail.mockResolvedValue({ data: defaultCluster });
  mockApi.cluster.getClusterOperations.mockResolvedValue({
    data: {
      list: [historyItem],
      total: 1,
      page: 1,
      page_size: 20,
      total_pages: 1,
    },
  });
  mockApi.cluster.getClusterOperationDetail.mockResolvedValue({
    data: {
      ...historyItem,
      approval: {
        required: true,
        ticket: 'ticket-501',
        status: 'approved',
        expires_at: '2026-04-05T12:00:00Z',
        reason: 'production node drain',
        consumed_at: '2026-04-05T10:05:00Z',
        replay_count: 1,
        replay_code: 'approval_token_replayed',
        replay_message: 'duplicate consume blocked',
      },
      request: {
        node: 'worker-1',
        timeout_seconds: 300,
        approval_token: 'masked',
      },
      response: {
        code: 'approval_required',
        message: '需要审批',
      },
      diagnostics: [
        {
          level: 'warning',
          message: 'Drain waiting for pod eviction',
        },
      ],
      timeline: [],
    },
  });
}

function renderPage(initialEntry = '/resources/clusters/42/operations') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/resources/clusters/:id/operations" element={<ClusterOperationCenterPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('ClusterOperationCenterPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockBaselineLoads();
  });

  afterEach(() => {
    cleanup();
  });

  it('loads the deep-linked audit detail and shows approval and summary sections', async () => {
    renderPage('/resources/clusters/42/operations?audit_id=501');

    await waitFor(() => {
      expect(mockApi.cluster.getClusterOperationDetail).toHaveBeenCalledWith(42, '501');
    });
    expect(await screen.findByText('审批状态')).toBeInTheDocument();
    expect(await screen.findByText('approved')).toBeInTheDocument();
    expect((await screen.findAllByText('请求摘要')).length).toBeGreaterThan(0);
    expect((await screen.findAllByText('响应摘要')).length).toBeGreaterThan(0);
    expect((await screen.findAllByText('诊断摘要')).length).toBeGreaterThan(0);
    expect(await screen.findByText(/approval_token_replayed/)).toBeInTheDocument();
  });

  it('uses the server-corrected page when refreshing after a pagination boundary response', async () => {
    mockApi.cluster.getClusterOperations
      .mockResolvedValueOnce({
        data: {
          list: [historyItem],
          total: 21,
          page: 2,
          page_size: 20,
          total_pages: 2,
        },
      })
      .mockResolvedValue({
        data: {
          list: [historyItem],
          total: 21,
          page: 2,
          page_size: 20,
          total_pages: 2,
        },
      });

    const user = userEvent.setup();
    renderPage();

    await screen.findByText('501');
    await user.click(screen.getByRole('button', { name: /刷新/ }));

    await waitFor(() => {
      expect(mockApi.cluster.getClusterOperations).toHaveBeenNthCalledWith(2, 42, expect.objectContaining({
        page: 2,
        page_size: 20,
      }));
    });
  });

  it('boots into policy release trace mode from query params and keeps release linkage in the detail share link', async () => {
    mockApi.cluster.getClusterOperations.mockResolvedValueOnce({
      data: {
        list: [policyReleaseHistoryItem],
        total: 1,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });
    mockApi.cluster.getClusterOperationDetail.mockResolvedValueOnce({
      data: {
        ...policyReleaseHistoryItem,
        request: {
          release: {
            release_id: 501,
            version: 'candidate-v2',
            policy: {
              name: 'allow-api',
              namespace: 'prod',
            },
          },
        },
        response: {
          release: {
            release_id: 501,
            version: 'candidate-v2',
          },
        },
        diagnostics: [],
        timeline: [],
      },
    });

    renderPage('/resources/clusters/42/operations?resource=policy_release&release_id=501');

    await waitFor(() => {
      expect(mockApi.cluster.getClusterOperations).toHaveBeenCalledWith(42, expect.objectContaining({
        resource: 'policy_release',
      }));
    });

    expect(await screen.findByText('allow-api')).toBeInTheDocument();

    const shareLink = await screen.findByRole('button', { name: '复制/共享当前审计链接' });
    expect(await screen.findByText('#501')).toBeInTheDocument();
    expect(shareLink).toBeInTheDocument();
  });
});
