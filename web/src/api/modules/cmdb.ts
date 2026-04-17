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
  ownerId?: number;
  env?: string;
  region?: string;
  projectId?: number;
  teamId?: number;
  tagsJson?: string;
  attrsJson?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface CMDBTreeNode {
  id: string;
  name: string;
  type: string;
  parentId?: string;
  children?: CMDBTreeNode[];
  isLeaf?: boolean;
}

export interface CMDBTopologyData {
  nodes: Array<{
    id: string;
    label: string;
    type: string;
    status?: string;
    [key: string]: any;
  }>;
  edges: Array<{
    id: string;
    source: string;
    target: string;
    label?: string;
    type?: string;
    [key: string]: any;
  }>;
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
    const res = await apiService.get<any>('/cmdb/assets', {
      params: {
        asset_type: params?.assetType,
        status: params?.status,
        keyword: params?.keyword,
        page: params?.page,
        page_size: params?.pageSize,
      },
    });
    const list = res.data?.list || [];
    return {
      ...res,
      data: list.map((x: any) => ({
        id: String(x.id),
        ciUid: x.ci_uid,
        assetType: x.ci_type || x.asset_type,
        source: x.source,
        name: x.name,
        status: x.status,
        owner: x.owner,
        ownerId: x.owner_id,
        env: x.env,
        region: x.region,
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
    const res = await apiService.get<any>('/cmdb/relations', {
      params: { asset_id: params?.assetId },
    });
    const list = res.data?.list || [];
    return {
      ...res,
      data: list.map((x: any) => ({
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
    const res = await apiService.get<any>('/cmdb/changes', { params: { asset_id: params?.assetId } });
    return {
      ...res,
      data: res.data?.list || [],
    };
  },

  async listAudits(assetId: string | number): Promise<ApiResponse<any[]>> {
    const res = await apiService.get<any>(`/cmdb/assets/${assetId}/audits`);
    return {
      ...res,
      data: res.data?.list || [],
    };
  },

  async getTree(params: { parentId?: number; viewType?: string }): Promise<ApiResponse<CMDBTreeNode[]>> {
    const res = await apiService.get<any>('/cmdb/tree', {
      params: {
        parent_id: params.parentId,
        view_type: params.viewType,
      },
    });
    return {
      ...res,
      data: res.data?.nodes || [],
    };
  },

  async getTopologySubgraph(params: { rootId: number; depth?: number; relTypes?: string }): Promise<ApiResponse<CMDBTopologyData>> {
    return apiService.get('/cmdb/topology/subgraph', {
      params: {
        root_id: params.rootId,
        depth: params.depth,
        rel_types: params.relTypes,
      },
    });
  },

  async getAsset(id: string): Promise<ApiResponse<CMDBAsset>> {
    const res = await apiService.get<any>(`/cmdb/assets/${id}`);
    if (res.data) {
      res.data = {
        id: String(res.data.id),
        ciUid: res.data.ci_uid,
        assetType: res.data.ci_type || res.data.asset_type,
        source: res.data.source,
        name: res.data.name,
        status: res.data.status,
        owner: res.data.owner,
        ownerId: res.data.owner_id,
        env: res.data.env,
        region: res.data.region,
        projectId: res.data.project_id,
        teamId: res.data.team_id,
        tagsJson: res.data.tags_json,
        attrsJson: res.data.attrs_json,
        createdAt: res.data.created_at,
        updatedAt: res.data.updated_at,
      };
    }
    return res;
  },
};
