import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import CICDPage from './CICDPage';

const mockApi = vi.hoisted(() => ({
  services: {
    getList: vi.fn(),
  },
  deployment: {
    getTargets: vi.fn(),
  },
  cicd: {
    getServiceCIConfig: vi.fn(),
    putServiceCIConfig: vi.fn(),
    deleteServiceCIConfig: vi.fn(),
    listCIRuns: vi.fn(),
    triggerCIRun: vi.fn(),
    getDeploymentCDConfig: vi.fn(),
    putDeploymentCDConfig: vi.fn(),
    triggerRelease: vi.fn(),
    listReleases: vi.fn(),
    approveRelease: vi.fn(),
    rejectRelease: vi.fn(),
    rollbackRelease: vi.fn(),
    listApprovals: vi.fn(),
    getServiceTimeline: vi.fn(),
  },
}));

vi.mock('../../api', () => ({
  Api: mockApi,
}));

const setupApi = () => {
  mockApi.services.getList.mockResolvedValue({
    data: { list: [{ id: 101, name: 'svc-a' }] },
  });
  mockApi.deployment.getTargets.mockResolvedValue({
    data: { list: [{ id: 201, name: 'target-a', env: 'staging' }] },
  });
  mockApi.cicd.getServiceCIConfig.mockResolvedValue({
    data: {
      id: 1,
      service_id: 101,
      repo_url: 'https://git.example.com/a.git',
      branch: 'main',
      build_steps: ['npm ci', 'npm run build'],
      artifact_target: 'registry.example.com/a:v1',
      trigger_mode: 'manual',
      status: 'active',
      updated_by: 1,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    },
  });
  mockApi.cicd.listCIRuns.mockResolvedValue({
    data: { list: [{ id: 11, trigger_type: 'manual', status: 'queued', triggered_by: 1, triggered_at: '2026-01-01T00:00:00Z' }] },
  });
  mockApi.cicd.listReleases.mockResolvedValue({
    data: {
      list: [{ id: 31, unified_release_id: 31, service_id: 101, deployment_id: 201, env: 'staging', runtime_type: 'k8s', version: 'v1.0.0', strategy: 'rolling', status: 'pending_approval', trigger_source: 'ci' }],
    },
  });
  mockApi.cicd.listApprovals.mockResolvedValue({
    data: { list: [{ id: 41, release_id: 31, approver_id: 2, decision: 'approved', comment: 'ok', created_at: '2026-01-01T00:00:00Z' }] },
  });
  mockApi.cicd.getServiceTimeline.mockResolvedValue({
    data: { list: [{ id: 51, event_type: 'release.triggered', actor_id: 1, created_at: '2026-01-01T00:00:00Z' }] },
  });
  mockApi.cicd.putServiceCIConfig.mockResolvedValue({ data: {} });
  mockApi.cicd.approveRelease.mockResolvedValue({ data: {} });
};

describe('CICDPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupApi();
  });

  it('renders timeline and release rows', async () => {
    render(<CICDPage />);
    const auditTabs = await screen.findAllByRole('tab', { name: '审批与审计时间线' });
    fireEvent.click(auditTabs[0]);
    expect(await screen.findByText('release.triggered', {}, { timeout: 10000 })).toBeInTheDocument();
  }, 15000);

  it('saves service ci config', async () => {
    render(<CICDPage />);
    await screen.findAllByDisplayValue('https://git.example.com/a.git');
    const repoInput = screen.getByLabelText('仓库地址');
    const artifactInput = screen.getByLabelText('产物目标（镜像仓库）');
    fireEvent.change(repoInput, { target: { value: 'https://git.example.com/a.git' } });
    fireEvent.change(artifactInput, { target: { value: 'registry.example.com/a:v1' } });
    const saveButtons = await screen.findAllByRole('button', { name: '保存 CI 配置' });
    fireEvent.click(saveButtons[0]);
    await waitFor(() => expect(mockApi.cicd.putServiceCIConfig).toHaveBeenCalled());
    const call = mockApi.cicd.putServiceCIConfig.mock.calls[0];
    expect(call[0]).toBe(101);
    expect(call[1].repo_url).toBe('https://git.example.com/a.git');
    expect(call[1].artifact_target).toBe('registry.example.com/a:v1');
  });

  it('approves pending release', async () => {
    render(<CICDPage />);
    const deployTabs = await screen.findAllByRole('tab', { name: '部署 CD 配置与发布' });
    fireEvent.click(deployTabs[0]);
    await waitFor(() => expect(screen.getByText('Unified ID')).toBeInTheDocument());
    const approveButtons = await screen.findAllByTestId('approve-release-31');
    fireEvent.click(approveButtons[0]);
    await waitFor(() => expect(mockApi.cicd.approveRelease).toHaveBeenCalledWith(31, 'approved in UI'));
  });
});
