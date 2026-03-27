import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';

// 集群数据结构
export interface Cluster {
  id: string;
  name: string;
  version: string;
  status: string;
  server?: string;
  credentialRef?: string;
  lastConnectMessage?: string;
  nodes: number;
  pods: number;
  cpu: number;
  memory: number;
  createdAt: string;
}

// 节点数据结构
export interface Node {
  id: string;
  name: string;
  ip?: string;
  role: string;
  status: string;
  cpu: number;
  memory: number;
  pods: number;
  labels: Record<string, string>;
}

// Pod数据结构
export interface Pod {
  id: string;
  name: string;
  namespace: string;
  status: string;
  phase: string;
  node: string;
  cpu: number;
  memory: number;
  restarts: number;
  age: string;
  labels: Record<string, string>;
  containers: Container[];
  qosClass: string;
  createdAt?: string;
  startTime: string;
}

// 容器数据结构
export interface Container {
  name: string;
  image: string;
  status: string;
  cpu: number;
  memory: number;
  restarts: number;
}

// 服务数据结构
export interface Service {
  id: string;
  name: string;
  namespace: string;
  type: string;
  clusterIP: string;
  externalIP?: string;
  ports: ServicePort[];
  selector: Record<string, string>;
  age: string;
}

// 服务端口数据结构
export interface ServicePort {
  port: number;
  targetPort: number;
  protocol: string;
}

// Ingress数据结构
export interface Ingress {
  id: string;
  name: string;
  namespace: string;
  host: string;
  path: string;
  service: string;
  port: number;
  tls: boolean;
}

// 集群列表请求参数
export interface ClusterListParams {
  page?: number;
  pageSize?: number;
  status?: string;
}

export interface CreateClusterPayload {
  name: string;
  server: string;
  kubeconfig?: string;
  credential_ref?: string;
  description?: string;
}

// 节点列表请求参数
export interface NodeListParams {
  page?: number;
  pageSize?: number;
  status?: string;
}

// Pod列表请求参数
export interface PodListParams {
  page?: number;
  pageSize?: number;
  namespace?: string;
  status?: string;
}

// 服务列表请求参数
export interface ServiceListParams {
  page?: number;
  pageSize?: number;
  namespace?: string;
  type?: string;
}

// Ingress列表请求参数
export interface IngressListParams {
  page?: number;
  pageSize?: number;
  namespace?: string;
}

export interface NamespaceItem {
  name: string;
  status?: string;
  labels?: Record<string, string>;
  created_at?: string;
}

export interface NamespaceBinding {
  id?: number;
  cluster_id: number;
  team_id: number;
  namespace: string;
  env?: string;
  readonly?: boolean;
}

export interface RolloutItem {
  name: string;
  namespace: string;
  strategy: string;
  phase?: string;
  ready_replicas?: number;
  replicas?: number;
  created_at?: string;
}

export interface HPAInput {
  namespace: string;
  name: string;
  target_ref_kind: string;
  target_ref_name: string;
  min_replicas: number;
  max_replicas: number;
  cpu_utilization?: number;
  memory_utilization?: number;
}

export interface QuotaInput {
  namespace: string;
  name: string;
  hard: Record<string, string>;
}

export interface LimitRangeInput {
  namespace: string;
  name: string;
  default?: Record<string, string>;
  default_request?: Record<string, string>;
  min?: Record<string, string>;
  max?: Record<string, string>;
}

// Kubernetes管理API
export const kubernetesApi = {
  async createCluster(payload: CreateClusterPayload): Promise<ApiResponse<Cluster>> {
    const response = await apiService.post<any>('/clusters', payload);
    const item = response.data || {};
    return {
      ...response,
      data: {
        id: String(item.id),
        name: item.name,
        version: item.version,
        status: item.status,
        server: item.server,
        credentialRef: item.credential_ref,
        lastConnectMessage: item.last_connect_message,
        nodes: item.nodes ?? 0,
        pods: item.pods ?? 0,
        cpu: item.cpu ?? 0,
        memory: item.memory ?? 0,
        createdAt: item.created_at ?? item.createdAt,
      },
    };
  },

  // 获取集群列表
  async getClusterList(params?: ClusterListParams): Promise<ApiResponse<PaginatedResponse<Cluster>>> {
    const response = await apiService.get<Cluster[]>('/clusters', {
      params: {
        page: params?.page,
        page_size: params?.pageSize,
      },
    });
    const list = (response.data || []).map((item: any) => ({
      id: String(item.id),
      name: item.name,
      version: item.version,
      status: item.status,
      server: item.server,
      credentialRef: item.credential_ref,
      lastConnectMessage: item.last_connect_message,
      nodes: item.nodes ?? 0,
      pods: item.pods ?? 0,
      cpu: item.cpu ?? 0,
      memory: item.memory ?? 0,
      createdAt: item.created_at ?? item.createdAt,
    }));
    return {
      ...response,
      data: {
        list,
        total: response.total || 0,
      },
    };
  },

  // 获取集群详情
  async getClusterDetail(id: string): Promise<ApiResponse<Cluster>> {
    return apiService.get(`/clusters/${id}`);
  },

  // 获取集群节点列表
  async getClusterNodes(clusterId: string, params?: NodeListParams): Promise<ApiResponse<PaginatedResponse<Node>>> {
    const response = await apiService.get<Node[]>(`/clusters/${clusterId}/nodes`, {
      params: {
        page: params?.page,
        page_size: params?.pageSize,
      },
    });
    const list = (response.data || []).map((item: any) => ({
      id: String(item.id),
      name: item.name,
      role: item.roles || item.role || '',
      status: (item.status || '').toLowerCase(),
      cpu: item.cpu_cores ?? item.cpu ?? 0,
      memory: item.memory ?? 0,
      pods: item.pods ?? 0,
      labels: item.labels || {},
      ip: item.ip,
    }));
    return {
      ...response,
      data: {
        list,
        total: response.total || 0,
      },
    };
  },

  // 后端当前提供 namespaces / deployments
  async getClusterNamespaces(clusterId: string): Promise<ApiResponse<PaginatedResponse<NamespaceItem>>> {
    const response = await apiService.get<any>(`/clusters/${clusterId}/namespaces`);
    const payload = response.data || {};
    const list = (payload.list || []).map((x: any) => ({
      name: x.name || x.metadata?.name,
      status: x.status,
      labels: x.labels || x.metadata?.labels || {},
      created_at: x.created_at || x.createdAt,
    }));
    return { ...response, data: { list, total: Number(payload.total || list.length) } };
  },
  async getClusterDeployments(clusterId: string, namespace?: string): Promise<ApiResponse<any[]>> {
    return apiService.get(`/clusters/${clusterId}/deployments`, { params: { namespace } });
  },

  async getClusterPods(clusterId: string, namespace?: string): Promise<ApiResponse<any[]>> {
    return apiService.get(`/clusters/${clusterId}/pods`, { params: { namespace } });
  },

  async testClusterConnect(clusterId: string): Promise<ApiResponse<void>> {
    return apiService.post(`/clusters/${clusterId}/connect/test`);
  },

  async getClusterEvents(clusterId: string, namespace?: string): Promise<ApiResponse<any[]>> {
    return apiService.get(`/clusters/${clusterId}/events`, { params: { namespace } });
  },

  async getPodLogs(clusterId: string, pod: string, container?: string, namespace?: string): Promise<ApiResponse<{ logs: string }>> {
    return apiService.get(`/clusters/${clusterId}/logs`, { params: { pod, container, namespace } });
  },

  async getClusterServices(clusterId: string, namespace?: string): Promise<ApiResponse<any[]>> {
    const response = await apiService.get<any[]>(`/clusters/${clusterId}/services`, { params: { namespace } });
    const list = (response.data || []).map((item: any) => ({
      ...item,
      ports: typeof item.ports === 'string' ? JSON.parse(item.ports || '[]') : item.ports,
    }));
    return { ...response, data: list };
  },

  async getClusterIngresses(clusterId: string, namespace?: string): Promise<ApiResponse<any[]>> {
    return apiService.get(`/clusters/${clusterId}/ingresses`, { params: { namespace } });
  },

  async previewDeploy(clusterId: string, payload: { namespace: string; name: string; image: string; replicas: number }): Promise<ApiResponse<any>> {
    return apiService.post(`/clusters/${clusterId}/deploy/preview`, payload);
  },

  async applyDeploy(clusterId: string, payload: { namespace: string; name: string; image: string; replicas: number }): Promise<ApiResponse<void>> {
    return apiService.post(`/clusters/${clusterId}/deploy/apply`, payload);
  },

  async createNamespace(clusterId: string, payload: { name: string; env?: string; labels?: Record<string, string> }): Promise<ApiResponse<any>> {
    return apiService.post(`/clusters/${clusterId}/namespaces`, payload);
  },

  async deleteNamespace(clusterId: string, namespace: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/clusters/${clusterId}/namespaces/${encodeURIComponent(namespace)}`);
  },

  async getNamespaceBindings(clusterId: string, teamId?: string): Promise<ApiResponse<PaginatedResponse<NamespaceBinding>>> {
    const response = await apiService.get<any>(`/clusters/${clusterId}/namespaces/bindings`, { params: { team_id: teamId } });
    const payload = response.data || {};
    return { ...response, data: { list: payload.list || [], total: Number(payload.total || 0) } };
  },

  async putNamespaceBindings(clusterId: string, teamId: string, bindings: Array<{ namespace: string; env?: string; readonly?: boolean }>): Promise<ApiResponse<PaginatedResponse<NamespaceBinding>>> {
    const response = await apiService.put<any>(`/clusters/${clusterId}/namespaces/bindings/${teamId}`, { bindings });
    const payload = response.data || {};
    return { ...response, data: { list: payload.list || [], total: Number(payload.total || 0) } };
  },

  async listRollouts(clusterId: string, namespace?: string): Promise<ApiResponse<PaginatedResponse<RolloutItem>>> {
    const response = await apiService.get<any>(`/clusters/${clusterId}/rollouts`, { params: { namespace } });
    const payload = response.data || {};
    return { ...response, data: { list: payload.list || [], total: Number(payload.total || 0) } };
  },

  async previewRollout(clusterId: string, payload: any): Promise<ApiResponse<{ manifest: string; strategy: string }>> {
    return apiService.post(`/clusters/${clusterId}/rollouts/preview`, payload);
  },

  async applyRollout(clusterId: string, payload: any): Promise<ApiResponse<any>> {
    return apiService.post(`/clusters/${clusterId}/rollouts/apply`, payload);
  },

  async promoteRollout(clusterId: string, name: string, payload: { namespace: string; full?: boolean; approval_token?: string }): Promise<ApiResponse<any>> {
    return apiService.post(`/clusters/${clusterId}/rollouts/${encodeURIComponent(name)}/promote`, payload);
  },

  async abortRollout(clusterId: string, name: string, payload: { namespace: string; approval_token?: string }): Promise<ApiResponse<any>> {
    return apiService.post(`/clusters/${clusterId}/rollouts/${encodeURIComponent(name)}/abort`, payload);
  },

  async rollbackRollout(clusterId: string, name: string, payload: { namespace: string; approval_token?: string }): Promise<ApiResponse<any>> {
    return apiService.post(`/clusters/${clusterId}/rollouts/${encodeURIComponent(name)}/rollback`, payload);
  },

  async listHPA(clusterId: string, namespace?: string): Promise<ApiResponse<PaginatedResponse<any>>> {
    const response = await apiService.get<any>(`/clusters/${clusterId}/hpa`, { params: { namespace } });
    const payload = response.data || {};
    return { ...response, data: { list: payload.list || [], total: Number(payload.total || 0) } };
  },

  async createHPA(clusterId: string, payload: HPAInput): Promise<ApiResponse<any>> {
    return apiService.post(`/clusters/${clusterId}/hpa`, payload);
  },

  async updateHPA(clusterId: string, name: string, payload: HPAInput): Promise<ApiResponse<any>> {
    return apiService.put(`/clusters/${clusterId}/hpa/${encodeURIComponent(name)}`, payload);
  },

  async deleteHPA(clusterId: string, name: string, namespace: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/clusters/${clusterId}/hpa/${encodeURIComponent(name)}`, { params: { namespace } });
  },

  async listQuotas(clusterId: string, namespace?: string): Promise<ApiResponse<PaginatedResponse<any>>> {
    const response = await apiService.get<any>(`/clusters/${clusterId}/quotas`, { params: { namespace } });
    const payload = response.data || {};
    return { ...response, data: { list: payload.list || [], total: Number(payload.total || 0) } };
  },

  async applyQuota(clusterId: string, payload: QuotaInput): Promise<ApiResponse<any>> {
    return apiService.post(`/clusters/${clusterId}/quotas`, payload);
  },

  async deleteQuota(clusterId: string, name: string, namespace: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/clusters/${clusterId}/quotas/${encodeURIComponent(name)}`, { params: { namespace } });
  },

  async listLimitRanges(clusterId: string, namespace?: string): Promise<ApiResponse<PaginatedResponse<any>>> {
    const response = await apiService.get<any>(`/clusters/${clusterId}/limit-ranges`, { params: { namespace } });
    const payload = response.data || {};
    return { ...response, data: { list: payload.list || [], total: Number(payload.total || 0) } };
  },

  async createLimitRange(clusterId: string, payload: LimitRangeInput): Promise<ApiResponse<any>> {
    return apiService.post(`/clusters/${clusterId}/limit-ranges`, payload);
  },

  async createClusterApproval(clusterId: string, payload: { namespace: string; action: string }): Promise<ApiResponse<any>> {
    return apiService.post(`/clusters/${clusterId}/approvals`, payload);
  },

  async confirmClusterApproval(clusterId: string, ticket: string, status: 'approved' | 'rejected' = 'approved'): Promise<ApiResponse<any>> {
    return apiService.post(`/clusters/${clusterId}/approvals/${encodeURIComponent(ticket)}/confirm`, { status });
  },
};
