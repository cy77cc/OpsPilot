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

  removeClusterNode(id: number, nodeName: string): Promise<ApiResponse<{ message: string }>> {
    return apiService.delete(`/clusters/${id}/nodes/${encodeURIComponent(nodeName)}`);
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

  upgradeCluster(id: number, targetVersion: string): Promise<ApiResponse<{
    cluster_id: number;
    from_version: string;
    to_version: string;
    status: string;
    message: string;
    upgrade_steps: string[];
  }>> {
    return apiService.post(`/clusters/${id}/upgrade`, { target_version: targetVersion });
  },

  renewCertificates(id: number): Promise<ApiResponse<{
    cluster_id: number;
    results: Array<{
      node_name: string;
      host_name?: string;
      success: boolean;
      message: string;
    }>;
    message: string;
  }>> {
    return apiService.post(`/clusters/${id}/certificates/renew`);
  },
};
