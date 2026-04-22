import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockPost = vi.fn();

vi.mock('../api', () => ({
  __esModule: true,
  default: {
    post: (...args: unknown[]) => mockPost(...args),
  },
}));

describe('serviceApi.create', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    mockPost.mockResolvedValue({
      success: true,
      data: { id: 1, name: 'payments', env: 'staging', owner: 'platform' },
    });
  });

  it('does not fall back to projectId from localStorage when payload omits it', async () => {
    localStorage.setItem('projectId', '999');
    const { serviceApi } = await import('./services');

    await serviceApi.create({
      name: 'payments',
      env: 'staging',
      owner: 'platform',
      runtime_type: 'k8s',
      config_mode: 'standard',
      service_kind: 'business',
      visibility: 'team',
      service_type: 'stateless',
      render_target: 'k8s',
      labels: [],
      standard_config: {
        image: 'ghcr.io/example/payments:1.0.0',
        replicas: 1,
        ports: [],
        envs: [],
        resources: { cpu: '500m', memory: '512Mi' },
      },
      status: 'draft',
    });

    expect(mockPost).toHaveBeenCalledWith(
      '/services',
      expect.not.objectContaining({ project_id: 999 })
    );
  });
});
