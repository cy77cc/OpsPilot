import type { DeployTarget, DeployRelease, DeployReleaseTimelineEvent, Inspection } from '../../api/modules/deployment';
import type { Cluster } from '../../api/modules/kubernetes';
import type { Host } from '../../api/modules/hosts';
import type { ServiceItem } from '../../api/modules/services';

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
    created_by: 1,
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
    source_release_id: 0,
    target_revision: '',
    preview_context_hash: '',
    preview_token_hash: '',
    status: 'applied',
    manifest_snapshot: '',
    runtime_context_json: '{}',
    checks_json: '[]',
    warnings_json: '[]',
    diagnostics_json: '[]',
    verification_json: '{}',
    operator: 1,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
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
    detail_json: '{}',
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
    findings_json: '[]',
    suggestions_json: '[]',
    status: 'done',
    created_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Factory for creating test clusters.
 */
export function createCluster(overrides?: Partial<Cluster>): Cluster {
  return {
    id: 1,
    name: 'test-cluster',
    description: 'Test cluster',
    version: '1.28.0',
    status: 'active',
    type: 'kubernetes',
    endpoint: 'https://127.0.0.1:6443',
    auth_method: 'token',
    management_mode: 'k8s-only',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Factory for creating test hosts.
 */
export function createHost(overrides?: Partial<Host>): Host {
  return {
    id: 1,
    name: 'test-host',
    hostname: 'test-host',
    ip: '10.0.0.1',
    port: 22,
    ssh_user: 'root',
    os: 'linux',
    arch: 'amd64',
    status: 'active',
    role: 'worker',
    cluster_id: 0,
    source: 'manual_ssh',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Factory for creating test services.
 */
export function createService(overrides?: Partial<ServiceItem>): ServiceItem {
  return {
    id: 1,
    name: 'test-service',
    env: 'staging',
    yaml_content: 'services:\n  app:\n    image: nginx:latest',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Creates a list of test deployment targets.
 */
export function createDeployTargets(count: number): DeployTarget[] {
  return Array.from({ length: count }, (_, i) =>
    createDeployTarget({
      id: i + 1,
      name: `test-target-${i + 1}`,
    })
  );
}

/**
 * Creates a list of test deployment releases.
 */
export function createDeployReleases(count: number): DeployRelease[] {
  return Array.from({ length: count }, (_, i) =>
    createDeployRelease({
      id: i + 1,
      service_id: i + 1,
      status: i % 3 === 0 ? 'applied' : i % 3 === 1 ? 'failed' : 'pending_approval',
    })
  );
}
