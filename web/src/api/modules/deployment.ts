import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';

export interface DeployTargetNode {
  host_id: number;
  role: string;
  weight: number;
}

export interface DeployTarget {
  id: number;
  name: string;
  target_type: 'k8s' | 'compose';
  runtime_type: 'k8s' | 'compose';
  cluster_id: number;
  cluster_source?: 'platform_managed' | 'external_managed';
  credential_id?: number;
  bootstrap_job_id?: string;
  project_id: number;
  team_id: number;
  env: string;
  environment?: string; // 环境类型：development, staging, production
  status: string;
  readiness_status?: string;
  cluster_name?: string;
  namespace?: string;
  nodes?: Array<{
    host_id: number;
    name: string;
    ip: string;
    status: string;
    role: string;
    weight: number;
  }>;
  created_at: string;
  updated_at: string;
}

export interface DeployRelease {
  id: number;
  unified_release_id?: number;
  service_id: number;
  target_id: number;
  namespace_or_project: string;
  runtime_type: 'k8s' | 'compose';
  strategy: string;
  trigger_source?: 'manual' | 'ci' | string;
  trigger_context_json?: string;
  ci_run_id?: number;
  revision_id: number;
  status: string;
  state?: string; // 发布状态：pending_approval, applied, failed, rejected, rolled_back
  lifecycle_state?: string;
  diagnostics_json?: string;
  verification_json?: string;
  source_release_id?: number;
  target_revision?: string;
  service_name?: string;
  target_name?: string;
  phase?: string;
  progress?: number;
  pods?: Array<{
    name: string;
    status: string;
    ready: boolean;
  }>;
  health_probes?: Array<{
    name: string;
    type: string;
    status: string;
  }>;
  logs?: string[];
  approval_info?: {
    ticket_id?: string;
    requester?: string;
    requester_email?: string;
    reason?: string;
    created_at: string;
    approved_by?: string;
    approved_at?: string;
    rejected_by?: string;
    rejected_at?: string;
    comment?: string;
  };
  created_at: string;
}

export interface DeployReleaseTimelineEvent {
  id: number;
  release_id: number;
  action: string;
  actor: number;
  detail: Record<string, any> | null;
  created_at: string;
}

export interface Inspection {
  id: number;
  release_id: number;
  target_id: number;
  service_id: number;
  stage: string;
  summary: string;
  status: string;
  created_at: string;
}

export interface ClusterBootstrapTask {
  id: string;
  name: string;
  control_plane_host_id: number;
  worker_ids_json: string;
  cni: string;
  status: string;
  result_json?: string;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export interface EnvironmentBootstrapJob {
  id: string;
  name: string;
  runtime_type: 'k8s' | 'compose';
  target_env: string;
  status: string;
  package_version: string;
  package_path: string;
  error_message?: string;
  result_json?: string;
  created_at: string;
  updated_at: string;
}

export interface ClusterCredential {
  id: number;
  name: string;
  runtime_type: 'k8s' | 'compose';
  source: 'platform_managed' | 'external_managed';
  cluster_id: number;
  endpoint: string;
  auth_method: string;
  status: string;
  last_test_at?: string;
  last_test_status?: string;
  last_test_message?: string;
  created_at: string;
  updated_at: string;
}

export interface AuditLog {
  id: number;
  action_type: string;
  resource_type: string;
  resource_id: number;
  actor_id: number;
  actor_name: string;
  detail: Record<string, any>;
  created_at: string;
}

export interface MetricsSummary {
  total_releases: number;
  success_rate: number;
  avg_duration_seconds: number;
  by_environment: Record<string, { total: number; success_rate: number }>;
  by_status: Record<string, number>;
  recent_failures: number;
  recent_releases: number;
}

export interface MetricsTrend {
  date: string;
  deployment_count: number;
  success_count: number;
  failure_count: number;
  success_rate: number;
}

export interface TopologyService {
  id: number;
  name: string;
  environment: string;
  status: string;
  last_deployment?: string;
  target_id: number;
  target_name?: string;
  runtime_type?: string;
}

export interface TopologyConnection {
  source_id: number;
  target_id: number;
  type: string;
}

export interface DeploymentTopology {
  services: TopologyService[];
  connections: TopologyConnection[];
}

export interface Policy {
  id: number;
  name: string;
  type: 'traffic' | 'resilience' | 'access' | 'slo';
  target_id: number;
  config: Record<string, any>;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export const deploymentApi = {
  getTargets(): Promise<ApiResponse<PaginatedResponse<DeployTarget>>> {
    return apiService.get('/deploy/targets');
  },
  listTargets(params?: { environment?: string; runtime_type?: string }): Promise<ApiResponse<PaginatedResponse<DeployTarget>>> {
    return apiService.get('/deploy/targets', { params });
  },
  getTargetDetail(id: number): Promise<ApiResponse<DeployTarget>> {
    return apiService.get(`/deploy/targets/${id}`);
  },
  listClusters(): Promise<ApiResponse<PaginatedResponse<any>>> {
    return apiService.get('/deploy/clusters');
  },
  createTarget(payload: {
    name: string;
    target_type: 'k8s' | 'compose';
    cluster_id?: number;
    cluster_source?: 'platform_managed' | 'external_managed';
    credential_id?: number;
    bootstrap_job_id?: string;
    project_id?: number;
    team_id?: number;
    env?: string;
    environment?: string;
    nodes?: DeployTargetNode[];
  }): Promise<ApiResponse<DeployTarget>> {
    return apiService.post('/deploy/targets', payload);
  },
  updateTarget(id: number, payload: Partial<{
    name: string;
    target_type: 'k8s' | 'compose';
    cluster_id: number;
    cluster_source: 'platform_managed' | 'external_managed';
    credential_id: number;
    bootstrap_job_id: string;
    project_id: number;
    team_id: number;
    env: string;
    environment: string;
    nodes: DeployTargetNode[];
  }>): Promise<ApiResponse<DeployTarget>> {
    return apiService.put(`/deploy/targets/${id}`, payload);
  },
  deleteTarget(id: number): Promise<ApiResponse<void>> {
    return apiService.delete(`/deploy/targets/${id}`);
  },
  previewRelease(payload: {
    service_id: number;
    target_id: number;
    env?: string;
    strategy?: string;
    variables?: Record<string, string>;
  }): Promise<ApiResponse<{ resolved_manifest: string; checks: Array<{ code: string; message: string; level: string }>; warnings: Array<{ code: string; message: string; level: string }>; runtime: string; preview_token?: string; preview_expires_at?: string }>> {
    return apiService.post('/deploy/releases/preview', payload);
  },
  applyRelease(payload: {
    service_id: number;
    target_id: number;
    env?: string;
    strategy?: string;
    variables?: Record<string, string>;
    preview_token?: string;
  }): Promise<ApiResponse<{ release_id: number; unified_release_id?: number; status: string; runtime_type: string; trigger_source?: string; trigger_context?: Record<string, any>; ci_run_id?: number; approval_required?: boolean; approval_ticket?: string; lifecycle_state?: string; reason_code?: string }>> {
    return apiService.post('/deploy/releases/apply', payload);
  },
  approveRelease(id: number, payload?: { comment?: string }): Promise<ApiResponse<{ release_id: number; unified_release_id?: number; status: string; runtime_type: string; trigger_source?: string; trigger_context?: Record<string, any>; ci_run_id?: number; lifecycle_state?: string }>> {
    return apiService.post(`/deploy/releases/${id}/approve`, payload || {});
  },
  rejectRelease(id: number, payload?: { comment?: string }): Promise<ApiResponse<{ release_id: number; unified_release_id?: number; status: string; runtime_type: string; trigger_source?: string; trigger_context?: Record<string, any>; ci_run_id?: number; lifecycle_state?: string }>> {
    return apiService.post(`/deploy/releases/${id}/reject`, payload || {});
  },
  rollbackRelease(id: number): Promise<ApiResponse<{ release_id: number; unified_release_id?: number; status: string; trigger_source?: string; trigger_context?: Record<string, any>; ci_run_id?: number }>> {
    return apiService.post(`/deploy/releases/${id}/rollback`);
  },
  getReleases(params?: { service_id?: number; target_id?: number }): Promise<ApiResponse<PaginatedResponse<DeployRelease>>> {
    return apiService.get('/deploy/releases', { params });
  },
  getReleaseDetail(id: number): Promise<ApiResponse<DeployRelease>> {
    return apiService.get(`/deploy/releases/${id}`);
  },
  getReleaseTimeline(id: number): Promise<ApiResponse<PaginatedResponse<DeployReleaseTimelineEvent>>> {
    return apiService.get(`/deploy/releases/${id}/timeline`);
  },
  getReleasesByRuntime(params?: { service_id?: number; target_id?: number; runtime_type?: 'k8s' | 'compose' }): Promise<ApiResponse<PaginatedResponse<DeployRelease>>> {
    return apiService.get('/deploy/releases', { params });
  },
  getGovernance(serviceId: number, env?: string): Promise<ApiResponse<any>> {
    return apiService.get(`/services/${serviceId}/governance`, { params: { env } });
  },
  putGovernance(serviceId: number, payload: any): Promise<ApiResponse<any>> {
    return apiService.put(`/services/${serviceId}/governance`, payload);
  },
  runInspection(payload: { release_id?: number; target_id?: number; service_id?: number; stage: 'pre' | 'post' | 'periodic' }): Promise<ApiResponse<Inspection>> {
    return apiService.post('/aiops/inspections/run', payload);
  },
  listInspections(params?: { service_id?: number; target_id?: number }): Promise<ApiResponse<PaginatedResponse<Inspection>>> {
    return apiService.get('/aiops/inspections', { params });
  },
  previewClusterBootstrap(payload: {
    name: string;
    control_plane_host_id: number;
    worker_host_ids?: number[];
    cni?: string;
  }): Promise<ApiResponse<{ name: string; control_plane_host_id: number; worker_host_ids: number[]; cni: string; steps: string[]; expected_endpoint: string }>> {
    return apiService.post('/deploy/clusters/bootstrap/preview', payload);
  },
  applyClusterBootstrap(payload: {
    name: string;
    control_plane_host_id: number;
    worker_host_ids?: number[];
    cni?: string;
  }): Promise<ApiResponse<{ task_id: string; status: string; cluster_id?: number; target_id?: number }>> {
    return apiService.post('/deploy/clusters/bootstrap/apply', payload);
  },
  getClusterBootstrapTask(taskId: string): Promise<ApiResponse<ClusterBootstrapTask>> {
    return apiService.get(`/deploy/clusters/bootstrap/${encodeURIComponent(taskId)}`);
  },
  startEnvironmentBootstrap(payload: {
    name: string;
    runtime_type: 'k8s' | 'compose';
    package_version: string;
    env?: string;
    target_id?: number;
    cluster_id?: number;
    control_plane_host_id?: number;
    worker_host_ids?: number[];
    node_ids?: number[];
  }): Promise<ApiResponse<{ job_id: string; status: string; runtime_type: string; package_version: string; target_id?: number }>> {
    return apiService.post('/deploy/environments/bootstrap', payload);
  },
  getEnvironmentBootstrapJob(jobId: string): Promise<ApiResponse<EnvironmentBootstrapJob>> {
    return apiService.get(`/deploy/environments/bootstrap/${encodeURIComponent(jobId)}`);
  },
  registerPlatformCredential(payload: {
    cluster_id: number;
    name?: string;
    runtime_type?: 'k8s' | 'compose';
  }): Promise<ApiResponse<ClusterCredential>> {
    return apiService.post('/deploy/credentials/platform/register', payload);
  },
  importExternalCredential(payload: {
    name: string;
    runtime_type?: 'k8s' | 'compose';
    auth_method?: 'kubeconfig' | 'cert';
    endpoint?: string;
    kubeconfig?: string;
    ca_cert?: string;
    cert?: string;
    key?: string;
    token?: string;
  }): Promise<ApiResponse<ClusterCredential>> {
    return apiService.post('/deploy/credentials/import', payload);
  },
  testCredential(id: number): Promise<ApiResponse<{ credential_id: number; connected: boolean; message: string; latency_ms?: number }>> {
    return apiService.post(`/deploy/credentials/${id}/test`);
  },
  listCredentials(params?: { runtime_type?: 'k8s' | 'compose' }): Promise<ApiResponse<PaginatedResponse<ClusterCredential>>> {
    return apiService.get('/deploy/credentials', { params });
  },

  // 审计日志
  getAuditLogs(params?: { action_type?: string; resource_type?: string; page?: number; page_size?: number }): Promise<ApiResponse<PaginatedResponse<AuditLog>>> {
    return apiService.get('/deploy/audit-logs', { params });
  },

  // 指标统计
  getMetricsSummary(): Promise<ApiResponse<MetricsSummary>> {
    return apiService.get('/deploy/metrics/summary');
  },
  getMetricsTrends(params?: { range?: 'daily' | 'weekly' | 'monthly' }): Promise<ApiResponse<MetricsTrend[]>> {
    return apiService.get('/deploy/metrics/trends', { params });
  },

  // 部署拓扑
  getTopology(params?: { environment?: string }): Promise<ApiResponse<DeploymentTopology>> {
    return apiService.get('/deploy/topology', { params });
  },

  // 策略管理
  getPolicies(params?: { type?: string; target_id?: number }): Promise<ApiResponse<PaginatedResponse<Policy>>> {
    return apiService.get('/deploy/policies', { params });
  },
  getPolicy(id: number): Promise<ApiResponse<Policy>> {
    return apiService.get(`/deploy/policies/${id}`);
  },
  createPolicy(payload: { name: string; type: string; target_id?: number; config?: Record<string, any>; enabled?: boolean }): Promise<ApiResponse<Policy>> {
    return apiService.post('/deploy/policies', payload);
  },
  updatePolicy(id: number, payload: Partial<{ name: string; type: string; config: Record<string, any>; enabled: boolean }>): Promise<ApiResponse<Policy>> {
    return apiService.put(`/deploy/policies/${id}`, payload);
  },
  deletePolicy(id: number): Promise<ApiResponse<void>> {
    return apiService.delete(`/deploy/policies/${id}`);
  },
};
