import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';

export interface Cluster {
  id: number;
  name: string;
  description?: string;
  version?: string;
  k8s_version?: string;
  status: string;
  source: string;
  type: string;
  node_count: number;
  endpoint?: string;
  pod_cidr?: string;
  service_cidr?: string;
  management_mode?: string;
  credential_id?: number;
  last_sync_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ClusterNode {
  id: number;
  cluster_id: number;
  host_id?: number;
  host_name?: string;
  name: string;
  ip: string;
  role: string;
  status: string;
  kubelet_version?: string;
  kube_proxy_version?: string;
  container_runtime?: string;
  os_image?: string;
  kernel_version?: string;
  allocatable_cpu?: string;
  allocatable_mem?: string;
  allocatable_pods?: number;
  labels?: Record<string, string>;
  taints?: Taint[];
  conditions?: NodeCondition[];
  joined_at?: string;
  last_seen_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Taint {
  key: string;
  value: string;
  effect: string;
}

export interface NodeCondition {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  last_transition_time?: string;
}

export interface BootstrapPreviewReq {
  name: string;
  profile_id?: number;
  control_plane_host_id: number;
  worker_host_ids?: number[];
  k8s_version?: string;
  version_channel?: string;
  cni?: string;
  pod_cidr?: string;
  service_cidr?: string;
  repo_mode?: 'online' | 'mirror';
  repo_url?: string;
  image_repository?: string;
  endpoint_mode?: 'nodeIP' | 'vip' | 'lbDNS';
  control_plane_endpoint?: string;
  vip_provider?: 'kube-vip' | 'keepalived';
  etcd_mode?: 'stacked' | 'external';
  external_etcd?: {
    endpoints?: string[];
    ca_cert?: string;
    cert?: string;
    key?: string;
  };
}

export interface BootstrapValidationIssue {
  field: string;
  code: string;
  domain?: string;
  message: string;
  remediation?: string;
}

export interface BootstrapProfile {
  id: number;
  name: string;
  description?: string;
  version_channel: string;
  k8s_version: string;
  repo_mode: 'online' | 'mirror';
  repo_url?: string;
  image_repository?: string;
  endpoint_mode: 'nodeIP' | 'vip' | 'lbDNS';
  control_plane_endpoint?: string;
  vip_provider?: 'kube-vip' | 'keepalived';
  etcd_mode: 'stacked' | 'external';
  external_etcd?: {
    endpoints?: string[];
    ca_cert?: string;
    cert?: string;
    key?: string;
  };
  created_at: string;
  updated_at: string;
}

export interface BootstrapVersionItem {
  version: string;
  channel: string;
  status: 'supported' | 'blocked';
  reason?: string;
}

export interface BootstrapPreviewResp {
  name: string;
  control_plane_host_id: number;
  worker_host_ids: number[];
  k8s_version: string;
  version_channel: string;
  cni: string;
  pod_cidr: string;
  service_cidr: string;
  repo_mode: string;
  repo_url: string;
  image_repository: string;
  endpoint_mode: string;
  control_plane_endpoint: string;
  vip_provider: string;
  etcd_mode: string;
  steps: string[];
  expected_endpoint: string;
  warnings?: string[];
  validation_issues?: BootstrapValidationIssue[];
  diagnostics?: Record<string, unknown>;
}

export interface BootstrapTask {
  id: string;
  name: string;
  cluster_id?: number;
  k8s_version: string;
  version_channel: string;
  repo_mode: string;
  repo_url: string;
  image_repository: string;
  endpoint_mode: string;
  control_plane_endpoint: string;
  vip_provider: string;
  etcd_mode: string;
  cni: string;
  pod_cidr: string;
  service_cidr: string;
  status: string;
  steps: BootstrapStepStatus[];
  current_step: number;
  error_message?: string;
  resolved_config_json?: string;
  diagnostics_json?: string;
  created_at: string;
  updated_at: string;
}

export interface BootstrapStepStatus {
  name: string;
  status: string;
  message?: string;
  started_at?: string;
  finished_at?: string;
  host_id?: number;
  output?: string;
}

export interface ClusterImportReq {
  name: string;
  description?: string;
  kubeconfig?: string;
  endpoint?: string;
  ca_cert?: string;
  cert?: string;
  key?: string;
  token?: string;
  skip_tls_verify?: boolean;
  auth_method?: string;
}

export interface ClusterTestResp {
  cluster_id: number;
  connected: boolean;
  message: string;
  version?: string;
  latency_ms?: number;
}

export interface AddNodeReq {
  host_ids: number[];
  role?: string;
}

export type ClusterOperationState = 'completed' | 'approval_required' | 'rejected' | 'failed';

export interface ClusterOperationApproval {
  required: boolean;
  ticket?: string;
  expires_at?: string;
  reason?: string;
}

export interface ClusterOperationResponse<T = unknown> {
  state: ClusterOperationState;
  success: boolean;
  message: string;
  audit_id?: string | number;
  approval?: ClusterOperationApproval;
  error_code?: string;
  diagnostics?: string[];
  result?: T;
  raw?: Record<string, unknown>;
}

export interface ClusterNodeApprovalPayload {
  approval_token?: string;
}

export interface ClusterNodeDrainPayload extends ClusterNodeApprovalPayload {
  delete_emptydir_data?: boolean;
  force?: boolean;
  ignore_daemonsets?: boolean;
  grace_period_seconds?: number;
  timeout_seconds?: number;
}

export interface ClusterNodeTaintPayload extends ClusterNodeApprovalPayload {
  key: string;
  value?: string;
  effect?: string;
}

export interface ClusterNodeLabelPayload extends ClusterNodeApprovalPayload {
  key: string;
  value?: string;
}

export interface ClusterUpgradePayload extends ClusterNodeApprovalPayload {
  target_version: string;
}

export interface ClusterCertificateRenewPayload extends ClusterNodeApprovalPayload {}

export interface ClusterOperationHistoryQuery {
  page?: number;
  page_size?: number;
  resource?: string;
  status?: string;
  operator?: string;
  from?: string;
  to?: string;
}

export interface ClusterOperationHistoryItem {
  audit_id: string | number;
  resource_type?: string;
  resource_name?: string;
  resource?: string;
  action: string;
  status: string;
  operator?: string;
  message?: string;
  approval_required?: boolean;
  approval_ticket?: string;
  created_at: string;
  updated_at?: string;
  namespace?: string;
  target?: string;
}

export interface ClusterOperationDetail extends ClusterOperationHistoryItem {
  approval?: ClusterOperationApproval & {
    approver?: string;
    approved_at?: string;
    rejected_by?: string;
    rejected_at?: string;
    ticket?: string;
  };
  request?: Record<string, unknown>;
  response?: Record<string, unknown>;
  diagnostics?: unknown[];
  timeline?: Array<{
    at?: string;
    message?: string;
    status?: string;
    level?: string;
  }>;
}

// Resource types
export interface NamespaceInfo {
  name: string;
  status: string;
  labels?: Record<string, string>;
  created_at: string;
}

export interface PodInfo {
  name: string;
  namespace: string;
  status: string;
  pod_ip: string;
  node_name: string;
  ready: string;
  restarts: number;
  age: string;
  labels?: Record<string, string>;
  created_at: string;
}

export interface DeploymentInfo {
  name: string;
  namespace: string;
  replicas: number;
  ready: number;
  updated: number;
  available: number;
  age: string;
  created_at: string;
}

export interface StatefulSetInfo {
  name: string;
  namespace: string;
  replicas: number;
  ready: number;
  age: string;
  created_at: string;
}

export interface DaemonSetInfo {
  name: string;
  namespace: string;
  desired: number;
  ready: number;
  age: string;
  created_at: string;
}

export interface JobInfo {
  name: string;
  namespace: string;
  completions: number;
  succeeded: number;
  failed: number;
  status: string;
  age: string;
  created_at: string;
}

export interface ServiceInfo {
  name: string;
  namespace: string;
  type: string;
  cluster_ip: string;
  ports: ServicePort[];
  selector?: Record<string, string>;
  age: string;
  created_at: string;
}

export interface ServicePort {
  name: string;
  port: number;
  target_port: string;
  protocol: string;
}

export interface IngressInfo {
  name: string;
  namespace: string;
  hosts: IngressHost[];
  age: string;
  created_at: string;
}

export interface IngressHost {
  host: string;
  paths: string[];
}

export interface ConfigMapInfo {
  name: string;
  namespace: string;
  data_keys: string[];
  age: string;
  created_at: string;
}

export interface SecretInfo {
  name: string;
  namespace: string;
  type: string;
  data_keys: string[];
  age: string;
  created_at: string;
}

export interface PVCInfo {
  name: string;
  namespace: string;
  status: string;
  capacity: string;
  access_modes: string;
  storage_class: string;
  volume_name: string;
  age: string;
  created_at: string;
}

export interface PVInfo {
  name: string;
  status: string;
  capacity: string;
  access_modes: string;
  storage_class: string;
  claim_ref: string;
  age: string;
  created_at: string;
}

export interface ClusterServiceInfo {
  id: number;
  name: string;
  project_name: string;
  team_name: string;
  env: string;
  last_deploy_at: string;
  status: string;
}

// Advanced operation types
export interface EventInfo {
  name: string;
  namespace: string;
  type: string;
  reason: string;
  message: string;
  source: string;
  count: number;
  age: string;
  first_seen: string;
  last_seen: string;
}

export interface HPAInfo {
  name: string;
  namespace: string;
  reference: string;
  min_replicas: number;
  max_replicas: number;
  current_cpu: string;
  target_cpu: string;
  current_mem: string;
  target_mem: string;
  replicas: number;
  metrics: HPAMetricInfo[];
  age: string;
  created_at: string;
}

export interface HPAMetricInfo {
  name: string;
  type: string;
  current: string;
  target: string;
}

export interface ResourceQuotaInfo {
  name: string;
  namespace: string;
  hard: Record<string, string>;
  used: Record<string, string>;
  age: string;
  created_at: string;
}

export interface LimitRangeInfo {
  name: string;
  namespace: string;
  type: string;
  limits: LimitRangeItem[];
  age: string;
  created_at: string;
}

export interface LimitRangeItem {
  type: string;
  max: Record<string, string>;
  min: Record<string, string>;
  default: Record<string, string>;
  default_request: Record<string, string>;
}

export interface ClusterVersionInfo {
  kubernetes_version: string;
  git_version: string;
  platform: string;
  go_version: string;
}

export interface ClusterUpgradePlan {
  current_version: string;
  target_version: string;
  upgradable: boolean;
  steps: string[];
  warnings: string[];
}

export interface CertificateInfo {
  name: string;
  expires_at: string;
  days_left: number;
  ca: boolean;
  alternate_names: string[];
}

const clusterOperationStates: ClusterOperationState[] = ['completed', 'approval_required', 'rejected', 'failed'];

const isPlainObject = (value: unknown): value is Record<string, any> => (
  Boolean(value) && typeof value === 'object' && !Array.isArray(value)
);

const coerceStringArray = (value: unknown): string[] | undefined => {
  if (Array.isArray(value)) {
    return value
      .map((item) => (typeof item === 'string' ? item : JSON.stringify(item)))
      .filter((item) => item.length > 0);
  }
  if (typeof value === 'string' && value.trim()) {
    try {
      const parsed = JSON.parse(value);
      if (Array.isArray(parsed)) {
        return parsed.map((item) => (typeof item === 'string' ? item : JSON.stringify(item)));
      }
    } catch {
      return [value];
    }
    return [value];
  }
  return undefined;
};

const normalizeApprovalPayload = (payload: unknown): ClusterOperationApproval | undefined => {
  if (!isPlainObject(payload)) {
    return undefined;
  }
  const hasApprovalSignals = Object.prototype.hasOwnProperty.call(payload, 'approval')
    || Object.prototype.hasOwnProperty.call(payload, 'approval_required')
    || Object.prototype.hasOwnProperty.call(payload, 'required')
    || Object.prototype.hasOwnProperty.call(payload, 'is_required')
    || Object.prototype.hasOwnProperty.call(payload, 'ticket')
    || Object.prototype.hasOwnProperty.call(payload, 'ticket_id')
    || Object.prototype.hasOwnProperty.call(payload, 'approval_ticket')
    || Object.prototype.hasOwnProperty.call(payload, 'approval_reason');
  const required = Boolean(payload.required ?? payload.approval_required ?? payload.is_required);
  const ticket = payload.ticket ?? payload.ticket_id ?? payload.approval_ticket;
  const expires_at = payload.expires_at ?? payload.expiresAt ?? payload.approval_expires_at;
  const reason = payload.reason ?? payload.approval_reason;
  if (!hasApprovalSignals) {
    return undefined;
  }
  if (!required && !ticket && !expires_at && !reason) {
    return undefined;
  }
  return {
    required: required || Boolean(ticket || expires_at || reason),
    ticket: typeof ticket === 'string' ? ticket : undefined,
    expires_at: typeof expires_at === 'string' ? expires_at : undefined,
    reason: typeof reason === 'string' ? reason : undefined,
  };
};

export function normalizeClusterOperationResponse<T = unknown>(payload: unknown): ClusterOperationResponse<T> {
  if (!isPlainObject(payload)) {
    return {
      state: 'completed',
      success: true,
      message: typeof payload === 'string' ? payload : '操作已完成',
      result: payload as T,
      raw: isPlainObject(payload) ? payload : undefined,
    };
  }

  const raw = payload;
  const stateCandidate = raw.state ?? raw.status ?? raw.result_state;
  const approval = normalizeApprovalPayload(raw.approval) ?? normalizeApprovalPayload(raw);
  const diagnostics = coerceStringArray(raw.diagnostics ?? raw.diagnostic_messages ?? raw.errors);
  let state: ClusterOperationState = 'completed';

  if (stateCandidate === 'approval_required' || approval?.required || raw.approval_required) {
    state = 'approval_required';
  } else if (stateCandidate === 'rejected' || raw.rejected === true || raw.approval_rejected === true) {
    state = 'rejected';
  } else if (
    stateCandidate === 'failed'
    || raw.success === false
    || raw.failed === true
    || raw.error_code
    || raw.error
    || raw.error_message
  ) {
    state = 'failed';
  } else if (stateCandidate && clusterOperationStates.includes(stateCandidate as ClusterOperationState)) {
    state = stateCandidate as ClusterOperationState;
  }

  const message = typeof raw.message === 'string'
    ? raw.message
    : typeof raw.msg === 'string'
      ? raw.msg
      : typeof raw.error_message === 'string'
        ? raw.error_message
        : state === 'approval_required'
          ? '操作需要审批'
          : state === 'rejected'
            ? '操作已拒绝'
            : state === 'failed'
              ? '操作失败'
              : '操作已完成';

  const auditId = raw.audit_id ?? raw.auditId ?? raw.operation_id ?? raw.operationId;
  const result = raw.result ?? raw.data ?? raw.payload ?? raw.response ?? raw.details;
  return {
    state,
    success: state === 'completed',
    message,
    audit_id: typeof auditId === 'string' || typeof auditId === 'number' ? auditId : undefined,
    approval,
    error_code: typeof raw.error_code === 'string' ? raw.error_code : typeof raw.reason_code === 'string' ? raw.reason_code : undefined,
    diagnostics,
    result: result as T,
    raw,
  };
}

const wrapOperationResponse = async <T = unknown>(promise: Promise<ApiResponse<unknown>>): Promise<ApiResponse<ClusterOperationResponse<T>>> => {
  const response = await promise;
  return {
    ...response,
    data: normalizeClusterOperationResponse<T>(response.data),
  };
};

export const clusterApi = {
  // Cluster CRUD
  getClusters(params?: { status?: string; source?: string }): Promise<ApiResponse<PaginatedResponse<Cluster>>> {
    return apiService.get('/clusters', { params });
  },

  getClusterDetail(id: number): Promise<ApiResponse<Cluster>> {
    return apiService.get(`/clusters/${id}`);
  },

  createCluster(data: ClusterImportReq): Promise<ApiResponse<Cluster>> {
    return apiService.post('/clusters', data);
  },

  updateCluster(id: number, data: { name?: string; description?: string }): Promise<ApiResponse<{ id: number; message: string }>> {
    return apiService.put(`/clusters/${id}`, data);
  },

  deleteCluster(id: number): Promise<ApiResponse<{ id: number; message: string }>> {
    return apiService.delete(`/clusters/${id}`);
  },

  testCluster(id: number): Promise<ApiResponse<ClusterTestResp>> {
    return apiService.post(`/clusters/${id}/test`);
  },

  // Cluster nodes
  getClusterNodes(id: number): Promise<ApiResponse<PaginatedResponse<ClusterNode>>> {
    return apiService.get(`/clusters/${id}/nodes`);
  },

  syncClusterNodes(id: number): Promise<ApiResponse<PaginatedResponse<ClusterNode>>> {
    return apiService.post(`/clusters/${id}/nodes/sync`);
  },

  getNodeDetail(clusterId: number, nodeName: string): Promise<ApiResponse<ClusterNode>> {
    return apiService.get(`/clusters/${clusterId}/nodes/${encodeURIComponent(nodeName)}`);
  },

  addClusterNodes(id: number, data: AddNodeReq): Promise<ApiResponse<{ results: any[]; message: string }>> {
    return apiService.post(`/clusters/${id}/nodes`, data);
  },

  removeClusterNode(id: number, nodeName: string, payload?: ClusterNodeApprovalPayload): Promise<ApiResponse<ClusterOperationResponse>> {
    return wrapOperationResponse(apiService.delete(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}`, { data: payload || {} }));
  },

  cordonNode(id: number, nodeName: string, payload?: ClusterNodeApprovalPayload): Promise<ApiResponse<ClusterOperationResponse>> {
    return wrapOperationResponse(apiService.post(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}/cordon`, payload || {}));
  },

  uncordonNode(id: number, nodeName: string, payload?: ClusterNodeApprovalPayload): Promise<ApiResponse<ClusterOperationResponse>> {
    return wrapOperationResponse(apiService.post(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}/uncordon`, payload || {}));
  },

  drainNode(id: number, nodeName: string, payload?: ClusterNodeDrainPayload): Promise<ApiResponse<ClusterOperationResponse>> {
    return wrapOperationResponse(apiService.post(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}/drain`, payload || {}));
  },

  upsertNodeTaint(id: number, nodeName: string, payload: ClusterNodeTaintPayload): Promise<ApiResponse<ClusterOperationResponse>> {
    return wrapOperationResponse(apiService.post(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}/taints`, payload));
  },

  removeNodeTaint(id: number, nodeName: string, payload: ClusterNodeTaintPayload): Promise<ApiResponse<ClusterOperationResponse>> {
    return wrapOperationResponse(apiService.delete(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}/taints`, { data: payload }));
  },

  upsertNodeLabel(id: number, nodeName: string, payload: ClusterNodeLabelPayload): Promise<ApiResponse<ClusterOperationResponse>> {
    return wrapOperationResponse(apiService.post(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}/labels`, payload));
  },

  removeNodeLabel(id: number, nodeName: string, payload: ClusterNodeLabelPayload): Promise<ApiResponse<ClusterOperationResponse>> {
    return wrapOperationResponse(apiService.delete(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}/labels`, { data: payload }));
  },

  // Namespaces
  getNamespaces(id: number): Promise<ApiResponse<PaginatedResponse<NamespaceInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces`);
  },

  // Workloads
  getPods(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<PodInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/pods`);
  },

  getDeployments(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<DeploymentInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/deployments`);
  },

  getStatefulSets(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<StatefulSetInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/statefulsets`);
  },

  getDaemonSets(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<DaemonSetInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/daemonsets`);
  },

  getJobs(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<JobInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/jobs`);
  },

  // Services and networking
  getServices(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<ServiceInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/services`);
  },

  getIngresses(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<IngressInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/ingresses`);
  },

  // Config
  getConfigMaps(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<ConfigMapInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/configmaps`);
  },

  getSecrets(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<SecretInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/secrets`);
  },

  // Storage
  getPVs(id: number): Promise<ApiResponse<PaginatedResponse<PVInfo>>> {
    return apiService.get(`/clusters/${id}/pvs`);
  },

  getPVCs(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<PVCInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/pvcs`);
  },

  // Deployed services
  getClusterServices(id: number): Promise<ApiResponse<PaginatedResponse<ClusterServiceInfo>>> {
    return apiService.get(`/clusters/${id}/services`);
  },

  // Bootstrap (self-hosted cluster)
  getBootstrapVersions(): Promise<ApiResponse<{ default_channel: string; list: BootstrapVersionItem[] }>> {
    return apiService.get('/clusters/bootstrap/versions');
  },

  getBootstrapProfiles(): Promise<ApiResponse<{ list: BootstrapProfile[]; total: number }>> {
    return apiService.get('/clusters/bootstrap/profiles');
  },

  createBootstrapProfile(data: Omit<BootstrapProfile, 'id' | 'created_at' | 'updated_at'>): Promise<ApiResponse<BootstrapProfile>> {
    return apiService.post('/clusters/bootstrap/profiles', data);
  },

  updateBootstrapProfile(id: number, data: Partial<Omit<BootstrapProfile, 'id' | 'name' | 'created_at' | 'updated_at'>>): Promise<ApiResponse<BootstrapProfile>> {
    return apiService.put(`/clusters/bootstrap/profiles/${id}`, data);
  },

  deleteBootstrapProfile(id: number): Promise<ApiResponse<{ id: number; deleted: boolean }>> {
    return apiService.delete(`/clusters/bootstrap/profiles/${id}`);
  },

  previewBootstrap(data: BootstrapPreviewReq): Promise<ApiResponse<BootstrapPreviewResp>> {
    return apiService.post('/clusters/bootstrap/preview', data);
  },

  applyBootstrap(data: BootstrapPreviewReq): Promise<ApiResponse<{ task_id: string; status: string }>> {
    return apiService.post('/clusters/bootstrap/apply', data);
  },

  getBootstrapTask(taskId: string): Promise<ApiResponse<BootstrapTask>> {
    return apiService.get(`/clusters/bootstrap/${encodeURIComponent(taskId)}`);
  },

  // Import external cluster
  importCluster(data: ClusterImportReq): Promise<ApiResponse<Cluster>> {
    return apiService.post('/clusters/import', data);
  },

  validateImport(data: {
    name?: string;
    kubeconfig?: string;
    endpoint?: string;
    ca_cert?: string;
    cert?: string;
    key?: string;
    token?: string;
    skip_tls_verify?: boolean;
  }): Promise<ApiResponse<{
    valid: boolean;
    message: string;
    endpoint?: string;
    version?: string;
    auth_method?: string;
  }>> {
    return apiService.post('/clusters/import/validate', data);
  },

  // Advanced operations
  getEvents(id: number, namespace?: string): Promise<ApiResponse<PaginatedResponse<EventInfo>>> {
    const params: Record<string, string> = {};
    if (namespace) params.namespace = namespace;
    return apiService.get(`/clusters/${id}/events`, { params });
  },

  getHPAs(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<HPAInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/hpas`);
  },

  getResourceQuotas(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<ResourceQuotaInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/resourcequotas`);
  },

  getLimitRanges(id: number, namespace: string): Promise<ApiResponse<PaginatedResponse<LimitRangeInfo>>> {
    return apiService.get(`/clusters/${id}/namespaces/${encodeURIComponent(namespace)}/limitranges`);
  },

  getClusterVersion(id: number): Promise<ApiResponse<ClusterVersionInfo>> {
    return apiService.get(`/clusters/${id}/version`);
  },

  getCertificates(id: number): Promise<ApiResponse<PaginatedResponse<CertificateInfo>>> {
    return apiService.get(`/clusters/${id}/certificates`);
  },

  getUpgradePlan(id: number): Promise<ApiResponse<ClusterUpgradePlan>> {
    return apiService.get(`/clusters/${id}/upgrade-plan`);
  },

  upgradeCluster(id: number, payload: ClusterUpgradePayload | string): Promise<ApiResponse<ClusterOperationResponse<{
    cluster_id?: number;
    from_version?: string;
    to_version?: string;
    status?: string;
    message?: string;
    upgrade_steps?: string[];
  }>>> {
    const requestBody = typeof payload === 'string' ? { target_version: payload } : payload;
    return wrapOperationResponse(apiService.post(`/clusters/${id}/upgrade`, requestBody));
  },

  renewCertificates(id: number, payload?: ClusterCertificateRenewPayload): Promise<ApiResponse<ClusterOperationResponse<{
    cluster_id?: number;
    results?: Array<{
      node_name: string;
      host_name?: string;
      success: boolean;
      message: string;
    }>;
    message?: string;
  }>>> {
    return wrapOperationResponse(apiService.post(`/clusters/${id}/certificates/renew`, payload || {}));
  },

  getClusterOperations(id: number, query?: ClusterOperationHistoryQuery): Promise<ApiResponse<PaginatedResponse<ClusterOperationHistoryItem>>> {
    return apiService.get(`/clusters/${id}/operations/history`, { params: query });
  },

  getClusterOperationDetail(id: number, auditId: string | number): Promise<ApiResponse<ClusterOperationDetail>> {
    return apiService.get(`/clusters/${id}/operations/${encodeURIComponent(String(auditId))}`);
  },
};
