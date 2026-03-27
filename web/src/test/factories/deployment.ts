import type { DeployTarget, DeployRelease, DeployReleaseTimelineEvent, Inspection } from '../../api/modules/deployment';

/**
 * Factory for creating test deployment targets.
 */
export function createDeployTarget(overrides?: Partial<DeployTarget>): DeployTarget {
  return {
    id: 1,
    name: 'test-target',
    target_type: 'k8s',
    runtime_type: 'k8s',
    cluster_id: 1,
    cluster_source: 'platform_managed',
    credential_id: 0,
    bootstrap_job_id: '',
    project_id: 1,
    team_id: 1,
    env: 'staging',
    status: 'active',
    readiness_status: 'ready',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Factory for creating test deployment releases.
 */
export function createDeployRelease(overrides?: Partial<DeployRelease>): DeployRelease {
  return {
    id: 1,
    service_id: 1,
    target_id: 1,
    namespace_or_project: 'staging',
    runtime_type: 'k8s',
    strategy: 'rolling',
    revision_id: 1,
    status: 'applied',
    created_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Factory for creating test release timeline events.
 */
export function createReleaseTimelineEvent(
  overrides?: Partial<DeployReleaseTimelineEvent>
): DeployReleaseTimelineEvent {
  return {
    id: 1,
    release_id: 1,
    action: 'release.previewed',
    actor: 1,
    detail: null,
    created_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Factory for creating test inspections.
 */
export function createInspection(overrides?: Partial<Inspection>): Inspection {
  return {
    id: 1,
    release_id: 1,
    target_id: 1,
    service_id: 1,
    stage: 'pre',
    summary: 'Test inspection',
    status: 'completed',
    created_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}
