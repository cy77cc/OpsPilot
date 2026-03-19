import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';

export interface Host {
  id: string;
  name: string;
  ip: string;
  status: string;
  healthState?: 'healthy' | 'degraded' | 'critical' | 'unknown';
  cpu: number;
  memory: number;
  disk: number;
  network: number;
  tags: string[];
  region: string;
  createdAt: string;
  lastActive: string;
  os?: string;
  username?: string;
  port?: number;
  description?: string;
  source?: string;
  provider?: string;
  providerInstanceId?: string;
  parentHostId?: string;
  sshKeyId?: number;
  maintenanceReason?: string;
  maintenanceBy?: number;
  maintenanceStartedAt?: string;
  maintenanceUntil?: string;
}

export interface HostListParams {
  page?: number;
  pageSize?: number;
  status?: string;
  region?: string;
  tags?: string[];
}

export interface HostCreateParams {
  probeToken?: string;
  name: string;
  ip: string;
  status?: string;
  tags?: string[];
  region?: string;
  description?: string;
  role?: string;
  clusterId?: number;
  force?: boolean;
  authType?: 'password' | 'key';
  username?: string;
  port?: number;
  password?: string;
  sshKeyId?: number;
  source?: 'manual_ssh' | 'cloud_import' | 'kvm_provision';
  provider?: string;
  providerInstanceId?: string;
  parentHostId?: number;
}

export interface HostUpdateParams {
  name?: string;
  status?: string;
  tags?: string[];
  region?: string;
  description?: string;
}

export interface HostBatchParams {
  hostIds: string[];
  action: string;
  tags?: string[];
  groupId?: number;
}

export interface HostMetricPoint {
  id: string;
  cpu: number;
  memory: number;
  disk: number;
  network: number;
  createdAt: string;
  healthState?: 'healthy' | 'degraded' | 'critical' | 'unknown';
  latencyMs?: number;
  errorMessage?: string;
}

export interface HostAuditItem {
  id: string;
  action: string;
  operator: string;
  detail: string;
  createdAt: string;
}

export interface HostHealthSnapshot {
  id: string;
  hostId: string;
  state: 'healthy' | 'degraded' | 'critical' | 'unknown';
  connectivityStatus: string;
  resourceStatus: string;
  systemStatus: string;
  latencyMs: number;
  cpuLoad: number;
  memoryUsedMB: number;
  memoryTotalMB: number;
  diskUsedPct: number;
  inodeUsedPct: number;
  summaryJson?: string;
  errorMessage?: string;
  checkedAt: string;
}

export interface SSHExecResult {
  stdout: string;
  stderr: string;
  exit_code: number;
}

export interface HostTerminalSession {
  session_id: string;
  status: string;
  ws_path: string;
  created_at: string;
  expires_at: string;
}

export interface HostFileItem {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  mode: string;
  updated_at: string;
}

export interface HostProbeParams {
  name: string;
  ip: string;
  port?: number;
  authType: 'password' | 'key';
  username: string;
  password?: string;
  sshKeyId?: number;
}

export interface HostProbeResult {
  probeToken: string;
  reachable: boolean;
  latencyMs: number;
  facts: {
    hostname?: string;
    os?: string;
    arch?: string;
    kernel?: string;
    cpuCores?: number;
    memoryMB?: number;
    diskGB?: number;
  };
  warnings: string[];
  errorCode?: string;
  message?: string;
  expiresAt: string;
}

export interface SSHKeyItem {
  id: string;
  name: string;
  publicKey: string;
  fingerprint: string;
  algorithm: string;
  encrypted: boolean;
  usageCount: number;
  createdAt: string;
}

export interface CloudAccount {
  id: string;
  provider: string;
  accountName: string;
  accessKeyId: string;
  regionDefault: string;
  status: string;
}

export interface CloudProviderInfo {
  name: string;
  displayName: string;
}

export interface CloudInstance {
  instanceId: string;
  name: string;
  ip: string;
  region: string;
  status: string;
  os: string;
  cpu: number;
  memoryMB: number;
  diskGB: number;
}

const parseLabels = (labels: any): string[] => {
  if (Array.isArray(labels)) {
    return labels.map((x) => String(x).trim()).filter(Boolean);
  }
  const raw = String(labels || '').trim();
  if (!raw) {
    return [];
  }
  if (raw.startsWith('[')) {
    try {
      const arr = JSON.parse(raw);
      if (Array.isArray(arr)) {
        return arr.map((x) => String(x).trim()).filter(Boolean);
      }
    } catch {
      // fallback to csv parser
    }
  }
  return raw.split(',').map((x) => x.trim()).filter(Boolean);
};

export const hostApi = {
  async getHostList(params?: HostListParams): Promise<ApiResponse<PaginatedResponse<Host>>> {
    const response = await apiService.get<Host[]>('/hosts', {
      params: {
        page: params?.page,
        page_size: params?.pageSize,
      },
    });
    const rawData = response.data as any;
    const items = Array.isArray(rawData) ? rawData : (rawData?.list || []);
    const list = items.map((item: any) => ({
      id: String(item.id),
      name: item.name,
      ip: item.ip,
      status: item.status,
      healthState: item.health_state || 'unknown',
      cpu: item.cpu_cores ?? item.cpu ?? 0,
      memory: item.memory_mb ?? item.memory ?? 0,
      disk: item.disk_gb ?? item.disk ?? 0,
      network: item.network ?? 0,
      tags: item.tags ?? parseLabels(item.labels),
      region: item.region ?? '',
      source: item.source,
      provider: item.provider,
      providerInstanceId: item.provider_instance_id,
      parentHostId: item.parent_host_id ? String(item.parent_host_id) : undefined,
      maintenanceReason: item.maintenance_reason || '',
      maintenanceBy: Number(item.maintenance_by || 0),
      maintenanceStartedAt: item.maintenance_started_at || undefined,
      maintenanceUntil: item.maintenance_until || undefined,
      createdAt: item.created_at ?? item.createdAt,
      lastActive: item.updated_at ?? item.lastActive,
    }));
    return {
      ...response,
      data: {
        list,
        total: response.total || 0,
      },
    };
  },

  async getHostDetail(id: string): Promise<ApiResponse<Host>> {
    const response = await apiService.get<any>(`/hosts/${id}`);
    const item = response.data || {};
    return {
      ...response,
      data: {
        id: String(item.id),
        name: item.name,
        ip: item.ip,
        status: item.status,
        healthState: item.health_state || 'unknown',
        cpu: item.cpu_cores ?? item.cpu ?? 0,
        memory: item.memory_mb ?? item.memory ?? 0,
        disk: item.disk_gb ?? item.disk ?? 0,
        network: item.network ?? 0,
        tags: item.tags ?? parseLabels(item.labels),
        region: item.region ?? '',
        createdAt: item.created_at ?? item.createdAt,
        lastActive: item.updated_at ?? item.lastActive,
        os: item.os,
        username: item.ssh_user ?? item.username,
        port: item.port,
        description: item.description,
        source: item.source,
        provider: item.provider,
        providerInstanceId: item.provider_instance_id,
        parentHostId: item.parent_host_id ? String(item.parent_host_id) : undefined,
        sshKeyId: item.ssh_key_id ? Number(item.ssh_key_id) : undefined,
        maintenanceReason: item.maintenance_reason || '',
        maintenanceBy: Number(item.maintenance_by || 0),
        maintenanceStartedAt: item.maintenance_started_at || undefined,
        maintenanceUntil: item.maintenance_until || undefined,
      },
    };
  },

  async createHost(data: HostCreateParams): Promise<ApiResponse<Host>> {
    return apiService.post('/hosts', {
      probe_token: data.probeToken,
      name: data.name,
      ip: data.ip,
      status: data.status || 'offline',
      username: data.username || 'root',
      auth_type: data.authType || 'password',
      password: data.password,
      ssh_key_id: data.sshKeyId,
      port: data.port || 22,
      labels: data.tags || [],
      role: data.role || '',
      cluster_id: data.clusterId || 0,
      source: data.source || 'manual_ssh',
      provider: data.provider || '',
      provider_instance_id: data.providerInstanceId || '',
      parent_host_id: data.parentHostId || undefined,
      force: !!data.force,
      description: data.description || `${data.region || ''} ${(data.tags || []).join(',')}`.trim(),
    });
  },

  async probeHost(data: HostProbeParams): Promise<ApiResponse<HostProbeResult>> {
    const res = await apiService.post<any>('/hosts/probe', {
      name: data.name,
      ip: data.ip,
      port: data.port || 22,
      auth_type: data.authType,
      username: data.username,
      password: data.password,
      ssh_key_id: data.sshKeyId,
    });
    const d = res.data || {};
    return {
      ...res,
      data: {
        probeToken: d.probe_token,
        reachable: !!d.reachable,
        latencyMs: Number(d.latency_ms || 0),
        facts: {
          hostname: d.facts?.hostname,
          os: d.facts?.os,
          arch: d.facts?.arch,
          kernel: d.facts?.kernel,
          cpuCores: d.facts?.cpu_cores,
          memoryMB: d.facts?.memory_mb,
          diskGB: d.facts?.disk_gb,
        },
        warnings: d.warnings || [],
        errorCode: d.error_code,
        message: d.message,
        expiresAt: d.expires_at,
      },
    };
  },

  async updateCredentials(id: string, data: { authType: 'password' | 'key'; username: string; password?: string; sshKeyId?: number; port?: number }): Promise<ApiResponse<any>> {
    return apiService.put(`/hosts/${id}/credentials`, {
      auth_type: data.authType,
      username: data.username,
      password: data.password,
      ssh_key_id: data.sshKeyId,
      port: data.port || 22,
    });
  },

  async updateHost(id: string, data: HostUpdateParams): Promise<ApiResponse<Host>> {
    const payload: Record<string, any> = {
      name: data.name,
      status: data.status,
      region: data.region,
      description: data.description,
    };
    if (Array.isArray(data.tags)) {
      payload.labels = JSON.stringify(data.tags.map((x) => String(x).trim()).filter(Boolean));
    }
    return apiService.put(`/hosts/${id}`, payload);
  },

  async deleteHost(id: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/hosts/${id}`);
  },

  async batchUpdate(data: HostBatchParams): Promise<ApiResponse<void>> {
    return apiService.post('/hosts/batch', {
      host_ids: data.hostIds.map((x) => Number(x)),
      action: data.action,
      tags: data.tags || [],
      group_id: data.groupId || 0,
    });
  },

  async getHostMetrics(id: string): Promise<ApiResponse<HostMetricPoint[]>> {
    const response = await apiService.get<any[]>(`/hosts/${id}/metrics`);
    return {
      ...response,
      data: (response.data || []).map((m: any) => ({
        id: String(m.id),
        cpu: Number(m.cpu || 0),
        memory: Number(m.memory || 0),
        disk: Number(m.disk || 0),
        network: Number(m.network || 0),
        createdAt: m.created_at ?? m.createdAt,
        healthState: m.health_state || undefined,
        latencyMs: Number(m.latency_ms || 0),
        errorMessage: m.error_message || undefined,
      })),
    };
  },

  async getHostAudits(id: string): Promise<ApiResponse<HostAuditItem[]>> {
    const response = await apiService.get<any[]>(`/hosts/${id}/audits`);
    return {
      ...response,
      data: (response.data || []).map((a: any) => ({
        id: String(a.id),
        action: a.action,
        operator: a.operator,
        detail: a.detail,
        createdAt: a.created_at ?? a.createdAt,
      })),
    };
  },

  async hostAction(id: string, action: string, opts?: { reason?: string; until?: string }): Promise<ApiResponse<void>> {
    return apiService.post(`/hosts/${id}/actions`, { action, reason: opts?.reason || '', until: opts?.until });
  },

  async runHealthCheck(id: string, deep?: boolean): Promise<ApiResponse<HostHealthSnapshot>> {
    const response = await apiService.post<any>(`/hosts/${id}/health/check`, { deep: !!deep });
    const d = response.data || {};
    return {
      ...response,
      data: {
        id: String(d.id || ''),
        hostId: String(d.host_id || id),
        state: (d.state || 'unknown') as HostHealthSnapshot['state'],
        connectivityStatus: d.connectivity_status || 'unknown',
        resourceStatus: d.resource_status || 'unknown',
        systemStatus: d.system_status || 'unknown',
        latencyMs: Number(d.latency_ms || 0),
        cpuLoad: Number(d.cpu_load || 0),
        memoryUsedMB: Number(d.memory_used_mb || 0),
        memoryTotalMB: Number(d.memory_total_mb || 0),
        diskUsedPct: Number(d.disk_used_pct || 0),
        inodeUsedPct: Number(d.inode_used_pct || 0),
        summaryJson: d.summary_json || '',
        errorMessage: d.error_message || '',
        checkedAt: d.checked_at || new Date().toISOString(),
      },
    };
  },

  async sshCheck(id: string): Promise<ApiResponse<{ reachable: boolean; message?: string }>> {
    return apiService.post(`/hosts/${id}/ssh/check`);
  },

  async sshExec(id: string, command: string): Promise<ApiResponse<SSHExecResult>> {
    return apiService.post(`/hosts/${id}/ssh/exec`, { command });
  },

  async createTerminalSession(id: string): Promise<ApiResponse<HostTerminalSession>> {
    return apiService.post(`/hosts/${id}/terminal/sessions`);
  },

  async getTerminalSession(id: string, sessionId: string): Promise<ApiResponse<any>> {
    return apiService.get(`/hosts/${id}/terminal/sessions/${sessionId}`);
  },

  async closeTerminalSession(id: string, sessionId: string): Promise<ApiResponse<any>> {
    return apiService.delete(`/hosts/${id}/terminal/sessions/${sessionId}`);
  },

  async listFiles(id: string, dirPath: string): Promise<ApiResponse<{ path: string; list: HostFileItem[]; total: number }>> {
    return apiService.get(`/hosts/${id}/files`, { params: { path: dirPath } });
  },

  async readFile(id: string, filePath: string): Promise<ApiResponse<{ path: string; content: string }>> {
    return apiService.get(`/hosts/${id}/files/content`, { params: { path: filePath } });
  },

  async writeFile(id: string, filePath: string, content: string): Promise<ApiResponse<{ path: string; size: number }>> {
    return apiService.put(`/hosts/${id}/files/content`, { path: filePath, content });
  },

  async uploadFile(id: string, dirPath: string, file: File): Promise<ApiResponse<{ path: string }>> {
    const form = new FormData();
    form.append('file', file);
    return apiService.post(`/hosts/${id}/files/upload`, form, {
      params: { path: dirPath },
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },

  async downloadFile(id: string, filePath: string): Promise<Blob> {
    const base = import.meta.env.VITE_API_BASE || '/api/v1';
    const token = localStorage.getItem('token');
    const resp = await fetch(`${base}/hosts/${id}/files/download?path=${encodeURIComponent(filePath)}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    });
    if (!resp.ok) {
      throw new Error(`下载失败: ${resp.status}`);
    }
    return await resp.blob();
  },

  async mkdir(id: string, dirPath: string): Promise<ApiResponse<{ path: string }>> {
    return apiService.post(`/hosts/${id}/files/mkdir`, { path: dirPath });
  },

  async renamePath(id: string, oldPath: string, newPath: string): Promise<ApiResponse<{ old_path: string; new_path: string }>> {
    return apiService.post(`/hosts/${id}/files/rename`, { old_path: oldPath, new_path: newPath });
  },

  async deletePath(id: string, targetPath: string): Promise<ApiResponse<{ path: string }>> {
    return apiService.delete(`/hosts/${id}/files`, { params: { path: targetPath } });
  },

  async batchExec(hostIds: string[], command: string): Promise<ApiResponse<Record<string, SSHExecResult>>> {
    return apiService.post('/hosts/batch/exec', { host_ids: hostIds.map((x) => Number(x)), command });
  },

  async getFacts(id: string): Promise<ApiResponse<any>> {
    return apiService.get(`/hosts/${id}/facts`);
  },

  async listTags(id: string): Promise<ApiResponse<string[]>> {
    return apiService.get(`/hosts/${id}/tags`);
  },

  async addTag(id: string, tag: string): Promise<ApiResponse<void>> {
    return apiService.post(`/hosts/${id}/tags`, { tag });
  },

  async removeTag(id: string, tag: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/hosts/${id}/tags/${encodeURIComponent(tag)}`);
  },

  async listSSHKeys(): Promise<ApiResponse<SSHKeyItem[]>> {
    const res = await apiService.get<any>('/credentials/ssh_keys');
    // 后端返回 { list: [...], total: N } 结构
    const rawList = Array.isArray(res.data) ? res.data : (res.data?.list || []);
    return {
      ...res,
      data: rawList.map((x: any) => ({
        id: String(x.id),
        name: x.name,
        publicKey: x.public_key || '',
        fingerprint: x.fingerprint || '',
        algorithm: x.algorithm || '',
        encrypted: !!x.encrypted,
        usageCount: Number(x.usage_count || 0),
        createdAt: x.created_at,
      })),
    };
  },

  async createSSHKey(payload: { name: string; privateKey: string; passphrase?: string }): Promise<ApiResponse<SSHKeyItem>> {
    const res = await apiService.post<any>('/credentials/ssh_keys', {
      name: payload.name,
      private_key: payload.privateKey,
      passphrase: payload.passphrase || '',
    });
    const x = res.data || {};
    return {
      ...res,
      data: {
        id: String(x.id),
        name: x.name,
        publicKey: x.public_key || '',
        fingerprint: x.fingerprint || '',
        algorithm: x.algorithm || '',
        encrypted: !!x.encrypted,
        usageCount: Number(x.usage_count || 0),
        createdAt: x.created_at,
      },
    };
  },

  async verifySSHKey(id: string, payload: { ip: string; port?: number; username?: string }): Promise<ApiResponse<any>> {
    return apiService.post(`/credentials/ssh_keys/${id}/verify`, {
      ip: payload.ip,
      port: payload.port || 22,
      username: payload.username || 'root',
    });
  },

  async deleteSSHKey(id: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/credentials/ssh_keys/${id}`);
  },

  async listCloudAccounts(provider?: string): Promise<ApiResponse<CloudAccount[]>> {
    const res = await apiService.get<any[]>('/hosts/cloud/accounts', { params: { provider } });
    return {
      ...res,
      data: (res.data || []).map((x: any) => ({
        id: String(x.id),
        provider: x.provider,
        accountName: x.account_name,
        accessKeyId: x.access_key_id,
        regionDefault: x.region_default,
        status: x.status,
      })),
    };
  },

  async listCloudProviders(): Promise<ApiResponse<CloudProviderInfo[]>> {
    const res = await apiService.get<any[]>('/hosts/cloud/providers');
    return {
      ...res,
      data: (res.data || []).map((x: any) => ({
        name: x.name,
        displayName: x.display_name,
      })),
    };
  },

  async createCloudAccount(payload: { provider: string; accountName: string; accessKeyId: string; accessKeySecret: string; regionDefault?: string }): Promise<ApiResponse<CloudAccount>> {
    const res = await apiService.post<any>('/hosts/cloud/accounts', {
      provider: payload.provider,
      account_name: payload.accountName,
      access_key_id: payload.accessKeyId,
      access_key_secret: payload.accessKeySecret,
      region_default: payload.regionDefault || '',
    });
    const x = res.data || {};
    return {
      ...res,
      data: {
        id: String(x.id),
        provider: x.provider,
        accountName: x.account_name,
        accessKeyId: x.access_key_id,
        regionDefault: x.region_default,
        status: x.status,
      },
    };
  },

  async queryCloudInstances(payload: { provider: string; accountId: number; region?: string; keyword?: string }): Promise<ApiResponse<CloudInstance[]>> {
    const res = await apiService.post<any[]>(`/hosts/cloud/providers/${payload.provider}/instances/query`, {
      account_id: payload.accountId,
      region: payload.region || '',
      keyword: payload.keyword || '',
    });
    return {
      ...res,
      data: (res.data || []).map((x: any) => ({
        instanceId: x.instance_id,
        name: x.name,
        ip: x.ip,
        region: x.region,
        status: x.status,
        os: x.os,
        cpu: Number(x.cpu || 0),
        memoryMB: Number(x.memory_mb || 0),
        diskGB: Number(x.disk_gb || 0),
      })),
    };
  },

  async importCloudInstances(payload: { provider: string; accountId: number; instances: CloudInstance[]; role?: string; labels?: string[] }): Promise<ApiResponse<any>> {
    return apiService.post(`/hosts/cloud/providers/${payload.provider}/instances/import`, {
      account_id: payload.accountId,
      instances: payload.instances.map((x) => ({
        instance_id: x.instanceId,
        name: x.name,
        ip: x.ip,
        region: x.region,
        status: x.status,
        os: x.os,
        cpu: x.cpu,
        memory_mb: x.memoryMB,
        disk_gb: x.diskGB,
      })),
      role: payload.role || '',
      labels: payload.labels || [],
    });
  },

  async kvmPreview(hostId: string, payload: { name: string; cpu: number; memoryMB: number; diskGB: number; networkBridge?: string; template?: string }): Promise<ApiResponse<any>> {
    return apiService.post(`/hosts/virtualization/kvm/hosts/${hostId}/preview`, {
      name: payload.name,
      cpu: payload.cpu,
      memory_mb: payload.memoryMB,
      disk_gb: payload.diskGB,
      network_bridge: payload.networkBridge || 'br0',
      template: payload.template || 'ubuntu-22.04',
    });
  },

  async kvmProvision(hostId: string, payload: { name: string; ip: string; cpu: number; memoryMB: number; diskGB: number; sshUser?: string; password?: string; sshKeyId?: number }): Promise<ApiResponse<any>> {
    return apiService.post(`/hosts/virtualization/kvm/hosts/${hostId}/provision`, {
      name: payload.name,
      ip: payload.ip,
      cpu: payload.cpu,
      memory_mb: payload.memoryMB,
      disk_gb: payload.diskGB,
      ssh_user: payload.sshUser || 'root',
      password: payload.password || '',
      ssh_key_id: payload.sshKeyId,
    });
  },
};
