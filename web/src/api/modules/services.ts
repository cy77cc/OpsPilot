import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';

export type ServiceRuntimeType = 'k8s' | 'compose' | 'helm';
export type ServiceConfigMode = 'standard' | 'custom';
export type ServiceVisibility = 'private' | 'team' | 'team-granted' | 'public';
export type ServiceKind = 'middleware' | 'business' | string;

export interface LabelKV {
  key: string;
  value: string;
}

export interface EnvKV {
  key: string;
  value: string;
}

export interface PortConfig {
  name?: string;
  protocol?: string;
  container_port: number;
  service_port: number;
}

export interface StandardServiceConfig {
  image: string;
  replicas: number;
  ports: PortConfig[];
  envs: EnvKV[];
  resources?: Record<string, string>;
}

export interface ServiceItem {
  id: string;
  projectId: string;
  teamId: string;
  name: string;
  env: string;
  owner: string;
  runtimeType: ServiceRuntimeType;
  visibility?: ServiceVisibility;
  grantedTeams?: number[];
  icon?: string;
  tags?: string[];
  deployCount?: number;
  renderTarget?: 'k8s' | 'compose' | 'helm';
  configMode: ServiceConfigMode;
  serviceKind: string;
  serviceType?: 'stateless' | 'stateful';
  status: string;
  labels: LabelKV[];
  standardConfig?: StandardServiceConfig;
  customYaml?: string;
  renderedYaml?: string;
  lastRevisionId?: string;
  defaultTargetId?: string;
  templateEngineVersion?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface TemplateVar {
  name: string;
  required: boolean;
  default?: string;
  description?: string;
  source_path?: string;
}

export interface VariableValueSet {
  service_id: number;
  env: string;
  values: Record<string, string>;
  secret_keys?: string[];
  updated_at?: string;
}

export interface ServiceRevision {
  id: number;
  service_id: number;
  revision_no: number;
  config_mode: ServiceConfigMode;
  render_target: 'k8s' | 'compose' | 'helm';
  variable_schema?: TemplateVar[];
  created_by: number;
  created_at: string;
}

export interface ServiceDeployTarget {
  id: number;
  service_id: number;
  cluster_id: number;
  namespace: string;
  deploy_target: ServiceRuntimeType;
  policy?: Record<string, any>;
  is_default: boolean;
  updated_at: string;
}

export interface ServiceReleaseRecord {
  id: number;
  unified_release_id?: number;
  service_id: number;
  revision_id: number;
  cluster_id: number;
  namespace: string;
  env: string;
  deploy_target: ServiceRuntimeType;
  status: string;
  trigger_source?: 'manual' | 'ci' | string;
  ci_run_id?: number;
  error?: string;
  created_at: string;
}

export interface ServiceEvent {
  id: string;
  serviceId: string;
  type: string;
  level: string;
  message: string;
  createdAt: string;
}

export interface ServiceQuota {
  cpuLimit: number;
  memoryLimit: number;
  cpuUsed: number;
  memoryUsed: number;
}

export interface ServiceMgmtListParams {
  page?: number;
  pageSize?: number;
  projectId?: string;
  teamId?: string;
  runtimeType?: ServiceRuntimeType | 'all';
  env?: string;
  labelSelector?: string;
  q?: string;
  serviceKind?: ServiceKind | 'all';
  visibility?: ServiceVisibility | 'all';
}

export interface ServiceCreateParams {
  project_id?: number;
  team_id?: number;
  name: string;
  env: string;
  owner: string;
  runtime_type: ServiceRuntimeType;
  config_mode: ServiceConfigMode;
  service_kind: string;
  visibility?: ServiceVisibility;
  granted_teams?: number[];
  icon?: string;
  tags?: string[];
  service_type: 'stateless' | 'stateful';
  render_target: 'k8s' | 'compose';
  labels?: LabelKV[];
  standard_config?: StandardServiceConfig;
  custom_yaml?: string;
  source_template_version?: string;
  status?: string;
}

export interface RenderPreviewReq {
  mode: ServiceConfigMode;
  target: 'k8s' | 'compose';
  service_name: string;
  service_type: 'stateless' | 'stateful';
  standard_config?: StandardServiceConfig;
  custom_yaml?: string;
  variables?: Record<string, string>;
  validate_only?: boolean;
}

export interface RenderPreviewResp {
  rendered_yaml: string;
  resolved_yaml?: string;
  unresolved_vars?: string[];
  detected_vars?: TemplateVar[];
  ast_summary?: Record<string, any>;
  diagnostics: Array<{ level: string; code: string; message: string }>;
  normalized_config?: StandardServiceConfig;
}

const mapService = (item: any): ServiceItem => ({
  id: String(item.id),
  projectId: String(item.project_id || ''),
  teamId: String(item.team_id || ''),
  name: item.name || '',
  env: item.env || 'staging',
  owner: item.owner || 'system',
  runtimeType: (item.runtime_type || 'k8s') as ServiceRuntimeType,
  renderTarget: (item.render_target || item.runtime_type || 'k8s') as 'k8s' | 'compose' | 'helm',
  configMode: (item.config_mode || 'standard') as ServiceConfigMode,
  serviceKind: item.service_kind || 'business',
  visibility: (item.visibility || 'team') as ServiceVisibility,
  grantedTeams: Array.isArray(item.granted_teams) ? item.granted_teams.map((x: any) => Number(x)) : [],
  icon: item.icon || '',
  tags: Array.isArray(item.tags) ? item.tags.map((x: any) => String(x)) : [],
  deployCount: Number(item.deploy_count || 0),
  serviceType: (item.service_type || 'stateless') as 'stateless' | 'stateful',
  status: item.status || 'draft',
  labels: Array.isArray(item.labels) ? item.labels.map((x: any) => ({ key: x.key || '', value: x.value || '' })) : [],
  standardConfig: item.standard_config,
  customYaml: item.custom_yaml,
  renderedYaml: item.rendered_yaml || item.yaml_content,
  lastRevisionId: item.last_revision_id ? String(item.last_revision_id) : '',
  defaultTargetId: item.default_target_id ? String(item.default_target_id) : '',
  templateEngineVersion: item.template_engine_version || 'v1',
  createdAt: item.created_at,
  updatedAt: item.updated_at,
});

export const serviceApi = {
  async preview(data: RenderPreviewReq): Promise<ApiResponse<RenderPreviewResp>> {
    return apiService.post('/services/render/preview', data);
  },

  async transform(data: { standard_config: StandardServiceConfig; target: 'k8s' | 'compose'; service_name: string; service_type: 'stateless' | 'stateful' }): Promise<ApiResponse<{ custom_yaml: string; source_hash: string; detected_vars?: TemplateVar[] }>> {
    return apiService.post('/services/transform', data);
  },

  async getList(params?: ServiceMgmtListParams): Promise<ApiResponse<PaginatedResponse<ServiceItem>>> {
    const response = await apiService.get<any>('/services', {
      params: {
        page: params?.page,
        page_size: params?.pageSize,
        project_id: params?.projectId,
        team_id: params?.teamId,
        runtime_type: params?.runtimeType && params.runtimeType !== 'all' ? params.runtimeType : undefined,
        env: params?.env && params.env !== 'all' ? params.env : undefined,
        label_selector: params?.labelSelector,
        q: params?.q,
        service_kind: params?.serviceKind && params.serviceKind !== 'all' ? params.serviceKind : undefined,
        visibility: params?.visibility && params.visibility !== 'all' ? params.visibility : undefined,
      },
    });
    const payload = response.data || {};
    const list = (payload.list || []).map(mapService);
    return {
      ...response,
      data: { list, total: Number(payload.total || list.length) },
    };
  },

  async getDetail(id: string): Promise<ApiResponse<ServiceItem>> {
    const response = await apiService.get<any>(`/services/${id}`);
    return { ...response, data: mapService(response.data) };
  },

  async create(data: ServiceCreateParams): Promise<ApiResponse<ServiceItem>> {
    const projectId = data.project_id ?? Number(localStorage.getItem('projectId') || '0');
    const payload: Record<string, unknown> = { ...data };
    if (projectId > 0) {
      payload.project_id = projectId;
    }
    if (typeof data.team_id === 'number' && data.team_id > 0) {
      payload.team_id = data.team_id;
    }
    const response = await apiService.post<any>('/services', payload);
    return { ...response, data: mapService(response.data) };
  },

  async update(id: string, data: Partial<ServiceCreateParams>): Promise<ApiResponse<ServiceItem>> {
    const response = await apiService.put<any>(`/services/${id}`, data);
    return { ...response, data: mapService(response.data) };
  },

  async remove(id: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/services/${id}`);
  },

  async deploy(id: string, payload?: { deploy_target?: ServiceRuntimeType; cluster_id?: number; namespace?: string; env?: string; variables?: Record<string, string>; approval_token?: string }): Promise<ApiResponse<{ release_record_id: number; unified_release_id?: number; trigger_source?: string }>> {
    return apiService.post(`/services/${id}/deploy`, payload || {});
  },

  async updateVisibility(id: string, visibility: ServiceVisibility): Promise<ApiResponse<ServiceItem>> {
    const response = await apiService.put<any>(`/services/${id}/visibility`, { visibility });
    return { ...response, data: mapService(response.data) };
  },

  async updateGrantedTeams(id: string, grantedTeams: number[]): Promise<ApiResponse<ServiceItem>> {
    const response = await apiService.put<any>(`/services/${id}/grant-teams`, { granted_teams: grantedTeams });
    return { ...response, data: mapService(response.data) };
  },

  async deployPreview(id: string, payload: { env?: string; cluster_id?: number; namespace?: string; deploy_target?: ServiceRuntimeType; variables?: Record<string, string> }): Promise<ApiResponse<{ resolved_yaml: string; checks: Array<{ level: string; code: string; message: string }>; warnings: Array<{ level: string; code: string; message: string }>; target: ServiceDeployTarget; target_id?: number; preview_token?: string; preview_expires_at?: string }>> {
    return apiService.post(`/services/${id}/deploy/preview`, payload);
  },

  async extractVariables(payload: { standard_config?: StandardServiceConfig; custom_yaml?: string; render_target: 'k8s' | 'compose'; service_name?: string; service_type?: 'stateless' | 'stateful' }): Promise<ApiResponse<{ vars: TemplateVar[] }>> {
    return apiService.post('/services/variables/extract', payload);
  },

  async getVariableSchema(id: string): Promise<ApiResponse<{ vars: TemplateVar[] }>> {
    return apiService.get(`/services/${id}/variables/schema`);
  },

  async getVariableValues(id: string, env: string): Promise<ApiResponse<VariableValueSet>> {
    return apiService.get(`/services/${id}/variables/values`, { params: { env } });
  },

  async upsertVariableValues(id: string, payload: { env: string; values: Record<string, string>; secret_keys?: string[] }): Promise<ApiResponse<VariableValueSet>> {
    return apiService.put(`/services/${id}/variables/values`, payload);
  },

  async listRevisions(id: string): Promise<ApiResponse<PaginatedResponse<ServiceRevision>>> {
    return apiService.get(`/services/${id}/revisions`);
  },

  async createRevision(id: string, payload: { config_mode: ServiceConfigMode; render_target: 'k8s' | 'compose'; standard_config?: StandardServiceConfig; custom_yaml?: string; variable_schema?: TemplateVar[] }): Promise<ApiResponse<ServiceRevision>> {
    return apiService.post(`/services/${id}/revisions`, payload);
  },

  async upsertDeployTarget(id: string, payload: { cluster_id: number; namespace: string; deploy_target: ServiceRuntimeType; policy?: Record<string, any> }): Promise<ApiResponse<ServiceDeployTarget>> {
    return apiService.put(`/services/${id}/deploy-target`, payload);
  },

  async listReleases(id: string): Promise<ApiResponse<PaginatedResponse<ServiceReleaseRecord>>> {
    return apiService.get(`/services/${id}/releases`);
  },

  async rollback(id: string): Promise<ApiResponse<void>> {
    return apiService.post(`/services/${id}/rollback`);
  },

  async getEvents(id: string): Promise<ApiResponse<PaginatedResponse<ServiceEvent>>> {
    const response = await apiService.get<any[]>(`/services/${id}/events`);
    const list = (response.data || []).map((item: any) => ({
      id: String(item.id),
      serviceId: String(item.service_id || item.serviceId),
      type: item.type,
      level: item.level,
      message: item.message,
      createdAt: item.created_at || item.createdAt,
    }));
    return { ...response, data: { list, total: list.length } };
  },

  async getQuota(): Promise<ApiResponse<ServiceQuota>> {
    return apiService.get('/services/quota');
  },

  async helmImport(payload: { service_id: number; chart_name: string; chart_version?: string; chart_ref?: string; values_yaml?: string; rendered_yaml?: string }): Promise<ApiResponse<any>> {
    return apiService.post('/services/helm/import', payload);
  },

  async helmRender(payload: { release_id?: number; chart_name?: string; chart_ref?: string; values_yaml?: string; rendered_yaml?: string }): Promise<ApiResponse<{ rendered_yaml: string; diagnostics: Array<{ level: string; code: string; message: string }> }>> {
    return apiService.post('/services/helm/render', payload);
  },

  async deployHelm(id: string): Promise<ApiResponse<void>> {
    return apiService.post(`/services/${id}/deploy/helm`);
  },
};
