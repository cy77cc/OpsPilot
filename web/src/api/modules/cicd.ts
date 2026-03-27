import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';

export type TriggerMode = 'manual' | 'source-event' | 'both';
export type TriggerType = 'manual' | 'source-event';
export type ReleaseStatus = 'previewed' | 'pending_approval' | 'approved' | 'applying' | 'applied' | 'failed' | 'rejected' | 'rollback' | 'rolled_back' | 'executing' | 'succeeded';

export interface ServiceCIConfig {
  id: number;
  service_id: number;
  repo_url: string;
  branch: string;
  build_steps: string[];
  artifact_target: string;
  trigger_mode: TriggerMode;
  status: string;
  updated_by: number;
  created_at: string;
  updated_at: string;
}

export interface CIRunRecord {
  id: number;
  service_id: number;
  ci_config_id: number;
  trigger_type: TriggerType;
  status: string;
  reason: string;
  triggered_by: number;
  triggered_at: string;
  created_at: string;
}

export interface DeploymentCDConfig {
  id: number;
  deployment_id: number;
  env: string;
  runtime_type: 'k8s' | 'compose';
  strategy: 'rolling' | 'blue-green' | 'canary';
  strategy_config: Record<string, any>;
  approval_required: boolean;
  updated_by: number;
  created_at: string;
  updated_at: string;
}

export interface ReleaseRecord {
  id: number;
  unified_release_id?: number;
  service_id: number;
  deployment_id: number;
  env: string;
  runtime_type: 'k8s' | 'compose';
  version: string;
  strategy: string;
  status: ReleaseStatus;
  triggered_by: number;
  approved_by: number;
  approval_comment: string;
  rollback_from_release_id: number;
  diagnostics?: any;
  trigger_source?: 'manual' | 'ci' | string;
  trigger_context?: any;
  ci_run_id?: number;
  started_at?: string;
  finished_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ReleaseApproval {
  id: number;
  release_id: number;
  approver_id: number;
  decision: 'approved' | 'rejected';
  comment: string;
  created_at: string;
}

export interface TimelineEvent {
  id: number;
  service_id: number;
  deployment_id: number;
  release_id: number;
  event_type: string;
  actor_id: number;
  payload: any;
  created_at: string;
}

export const cicdApi = {
  getServiceCIConfig(serviceId: number): Promise<ApiResponse<ServiceCIConfig>> {
    return apiService.get(`/cicd/services/${serviceId}/ci-config`);
  },

  putServiceCIConfig(serviceId: number, payload: {
    repo_url: string;
    branch?: string;
    build_steps?: string[];
    artifact_target: string;
    trigger_mode: TriggerMode;
  }): Promise<ApiResponse<ServiceCIConfig>> {
    return apiService.put(`/cicd/services/${serviceId}/ci-config`, payload);
  },

  deleteServiceCIConfig(serviceId: number): Promise<ApiResponse<void>> {
    return apiService.delete(`/cicd/services/${serviceId}/ci-config`);
  },

  triggerCIRun(serviceId: number, payload: { trigger_type: TriggerType; reason?: string }): Promise<ApiResponse<CIRunRecord>> {
    return apiService.post(`/cicd/services/${serviceId}/ci-runs/trigger`, payload);
  },

  listCIRuns(serviceId: number): Promise<ApiResponse<PaginatedResponse<CIRunRecord>>> {
    return apiService.get(`/cicd/services/${serviceId}/ci-runs`);
  },

  getDeploymentCDConfig(deploymentId: number, env?: string, runtimeType?: 'k8s' | 'compose'): Promise<ApiResponse<DeploymentCDConfig>> {
    return apiService.get(`/cicd/deployments/${deploymentId}/cd-config`, { params: { env, runtime_type: runtimeType } });
  },

  putDeploymentCDConfig(deploymentId: number, payload: {
    env: string;
    runtime_type?: 'k8s' | 'compose';
    strategy: 'rolling' | 'blue-green' | 'canary';
    strategy_config?: Record<string, any>;
    approval_required?: boolean;
  }): Promise<ApiResponse<DeploymentCDConfig>> {
    return apiService.put(`/cicd/deployments/${deploymentId}/cd-config`, payload);
  },

  triggerRelease(payload: {
    service_id: number;
    deployment_id: number;
    env: string;
    runtime_type?: 'k8s' | 'compose';
    version: string;
  }): Promise<ApiResponse<ReleaseRecord>> {
    return apiService.post('/cicd/releases', payload);
  },

  listReleases(params?: { service_id?: number; deployment_id?: number; runtime_type?: 'k8s' | 'compose' }): Promise<ApiResponse<PaginatedResponse<ReleaseRecord>>> {
    return apiService.get('/cicd/releases', { params });
  },

  approveRelease(id: number, comment?: string): Promise<ApiResponse<ReleaseRecord>> {
    return apiService.post(`/cicd/releases/${id}/approve`, { comment: comment || '' });
  },

  rejectRelease(id: number, comment?: string): Promise<ApiResponse<ReleaseRecord>> {
    return apiService.post(`/cicd/releases/${id}/reject`, { comment: comment || '' });
  },

  rollbackRelease(id: number, payload: { target_version: string; comment?: string }): Promise<ApiResponse<ReleaseRecord>> {
    return apiService.post(`/cicd/releases/${id}/rollback`, payload);
  },

  listApprovals(releaseId: number): Promise<ApiResponse<PaginatedResponse<ReleaseApproval>>> {
    return apiService.get(`/cicd/releases/${releaseId}/approvals`);
  },

  getServiceTimeline(serviceId: number): Promise<ApiResponse<PaginatedResponse<TimelineEvent>>> {
    return apiService.get(`/cicd/services/${serviceId}/timeline`);
  },
};
