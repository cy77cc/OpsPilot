import type {
  ServiceCIConfig,
  CIRunRecord,
  DeploymentCDConfig,
  ReleaseRecord,
  ReleaseApproval,
  TimelineEvent,
} from '../../api/modules/cicd';

/**
 * Factory for creating test CI configs.
 */
export function createCIConfig(overrides?: Partial<ServiceCIConfig>): ServiceCIConfig {
  return {
    id: 1,
    service_id: 1,
    repo_url: 'https://github.com/test/repo.git',
    branch: 'main',
    build_steps: ['npm install', 'npm run build'],
    artifact_target: 'dist/',
    trigger_mode: 'manual',
    status: 'active',
    updated_by: 1,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Factory for creating test CI run records.
 */
export function createCIRun(overrides?: Partial<CIRunRecord>): CIRunRecord {
  return {
    id: 1,
    service_id: 1,
    ci_config_id: 1,
    trigger_type: 'manual',
    status: 'succeeded',
    reason: 'Manual trigger',
    triggered_by: 1,
    triggered_at: '2026-01-01T00:00:00Z',
    created_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Factory for creating test CD configs.
 */
export function createCDConfig(overrides?: Partial<DeploymentCDConfig>): DeploymentCDConfig {
  return {
    id: 1,
    deployment_id: 1,
    env: 'staging',
    runtime_type: 'k8s',
    strategy: 'rolling',
    strategy_config: {},
    approval_required: false,
    updated_by: 1,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Factory for creating test release records.
 */
export function createRelease(overrides?: Partial<ReleaseRecord>): ReleaseRecord {
  return {
    id: 1,
    service_id: 1,
    deployment_id: 1,
    env: 'staging',
    runtime_type: 'k8s',
    version: 'v1.0.0',
    strategy: 'rolling',
    status: 'succeeded',
    triggered_by: 1,
    approved_by: 0,
    approval_comment: '',
    rollback_from_release_id: 0,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Factory for creating test release approvals.
 */
export function createReleaseApproval(overrides?: Partial<ReleaseApproval>): ReleaseApproval {
  return {
    id: 1,
    release_id: 1,
    approver_id: 1,
    decision: 'approved',
    comment: 'LGTM',
    created_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Factory for creating test timeline events.
 */
export function createTimelineEvent(overrides?: Partial<TimelineEvent>): TimelineEvent {
  return {
    id: 1,
    service_id: 1,
    deployment_id: 1,
    release_id: 1,
    event_type: 'release.triggered',
    actor_id: 1,
    payload: {},
    created_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Creates a list of test CI runs with varied statuses.
 */
export function createCIRuns(count: number): CIRunRecord[] {
  const statuses = ['queued', 'running', 'succeeded', 'failed'];
  return Array.from({ length: count }, (_, i) =>
    createCIRun({
      id: i + 1,
      service_id: Math.floor(i / 2) + 1,
      status: statuses[i % statuses.length] as CIRunRecord['status'],
    })
  );
}

/**
 * Creates a list of test releases with varied statuses.
 */
export function createReleases(count: number): ReleaseRecord[] {
  const statuses: ReleaseRecord['status'][] = [
    'pending_approval',
    'approved',
    'executing',
    'succeeded',
    'failed',
  ];
  return Array.from({ length: count }, (_, i) =>
    createRelease({
      id: i + 1,
      service_id: Math.floor(i / 2) + 1,
      version: `v1.0.${i}`,
      status: statuses[i % statuses.length],
    })
  );
}
