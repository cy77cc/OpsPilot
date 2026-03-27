import apiService from '../api';
import type { ApiResponse } from '../api';

export interface CMDBAsset {
  id: string;
  ciUid?: string;
  assetType: string;
  source: string;
  name: string;
  status: string;
  owner: string;
  projectId?: number;
  teamId?: number;
  tagsJson?: string;
  attrsJson?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface CMDBRelation {
  id: string;
  fromAssetId: string;
  toAssetId: string;
  relationType: string;
}

export interface CMDBTopology {
  nodes: Array<Record<string, any>>;
  edges: Array<Record<string, any>>;
}

export interface CMDBSyncJob {
  id: string;
  source: string;
  status: string;
  summaryJson?: string;
  errorMessage?: string;
  startedAt?: string;
  finishedAt?: string;
}

export const cmdbApi = {
  async listAssets(params?: { assetType?: string; status?: string; keyword?: string; page?: number; pageSize?: number }): Promise<ApiResponse<CMDBAsset[]>> {
    const res = await apiService.get<any[]>('/cmdb/assets', {
      params: {
        asset_type: params?.assetType,
        status: params?.status,
        keyword: params?.keyword,
        page: params?.page,
        page_size: params?.pageSize,
      },
    });
    return {
      ...res,
      data: (res.data || []).map((x: any) => ({
        id: String(x.id),
        ciUid: x.ci_uid,
        assetType: x.ci_type || x.asset_type,
        source: x.source,
        name: x.name,
        status: x.status,
        owner: x.owner,
        projectId: x.project_id,
        teamId: x.team_id,
        tagsJson: x.tags_json,
        attrsJson: x.attrs_json,
        createdAt: x.created_at,
        updatedAt: x.updated_at,
      })),
    };
  },

  async createAsset(payload: {
    assetType: string;
    name: string;
    source?: string;
    status?: string;
    owner?: string;
    attrsJson?: string;
    tagsJson?: string;
  }): Promise<ApiResponse<CMDBAsset>> {
    return apiService.post('/cmdb/assets', {
      ci_type: payload.assetType,
      name: payload.name,
      source: payload.source,
      status: payload.status,
      owner: payload.owner,
      attrs_json: payload.attrsJson,
      tags_json: payload.tagsJson,
    });
  },

  async updateAsset(id: string, payload: {
    name?: string;
    status?: string;
    owner?: string;
    attrsJson?: string;
    tagsJson?: string;
  }): Promise<ApiResponse<CMDBAsset>> {
    return apiService.put(`/cmdb/assets/${id}`, {
      name: payload.name,
      status: payload.status,
      owner: payload.owner,
      attrs_json: payload.attrsJson,
      tags_json: payload.tagsJson,
    });
  },

  async deleteAsset(id: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/cmdb/assets/${id}`);
  },

  async listRelations(params?: { assetId?: string }): Promise<ApiResponse<CMDBRelation[]>> {
    const res = await apiService.get<any[]>('/cmdb/relations', {
      params: { asset_id: params?.assetId },
    });
    return {
      ...res,
      data: (res.data || []).map((x: any) => ({
        id: String(x.id),
        fromAssetId: String(x.from_asset_id ?? x.from_ci_id),
        toAssetId: String(x.to_asset_id ?? x.to_ci_id),
        relationType: x.relation_type,
      })),
    };
  },

  async createRelation(payload: { fromAssetId: string; toAssetId: string; relationType: string }): Promise<ApiResponse<CMDBRelation>> {
    return apiService.post('/cmdb/relations', {
      from_ci_id: Number(payload.fromAssetId),
      to_ci_id: Number(payload.toAssetId),
      relation_type: payload.relationType,
    });
  },

  async deleteRelation(id: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/cmdb/relations/${id}`);
  },

  async getTopology(): Promise<ApiResponse<CMDBTopology>> {
    return apiService.get('/cmdb/topology');
  },

  async triggerSync(source = 'all'): Promise<ApiResponse<CMDBSyncJob>> {
    return apiService.post('/cmdb/sync/jobs', { source });
  },

  async getSyncJob(id: string): Promise<ApiResponse<CMDBSyncJob>> {
    return apiService.get(`/cmdb/sync/jobs/${id}`);
  },

  async retrySyncJob(id: string): Promise<ApiResponse<CMDBSyncJob>> {
    return apiService.post(`/cmdb/sync/jobs/${id}/retry`);
  },

  async listChanges(params?: { assetId?: string }): Promise<ApiResponse<any[]>> {
    return apiService.get('/cmdb/changes', { params: { asset_id: params?.assetId } });
  },
};
