import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';
import { buildContextualFetchInit } from '../requestContext';

export interface Host {
  id: string;
  name: string;
  ip: string;
  status: string;
  healthState?: 'healthy' | 'degraded' | 'critical' | 'unknown';
  monitorStatus?: 'healthy' | 'warning' | 'unmanaged';
  cpu: number;
  memory: number;
  disk: number;
  cpuUsagePct?: number;
  memoryUsagePct?: number;
  diskUsagePct?: number;
  network: number;
  tags: string[];
  environment?: 'prod' | 'staging' | 'test' | 'dev' | 'ops';
  alertCount?: number;
  region: string;
  createdAt: string;
  lastActive: string;
  lastHeartbeatAt?: string;
  os?: string;
  osVersion?: string;
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
  keyword?: string;
  status?: string;
  environment?: string;
  region?: string;
  tags?: string[];
  os?: string;
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
  credentialTemplateId?: number;
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

export interface HostOverviewStats {
  totalHosts: number;
  onlineHosts: number;
  abnormalHosts: number;
  avgCpuUsage: number;
  avgMemoryUsage: number;
  todayAlertCount: number;
  severeAlertCount: number;
  warningAlertCount: number;
  onlineRate: number;
}

export interface HostDistributionData {
  name: string;
  value: number;
  percent: number;
}

export interface HostTrendDataPoint {
  time: string;
  cpuUsage: number;
  memoryUsage: number;
}

export interface HostPendingAlert {
  name: string;
  level: 'critical' | 'warning';
  count: number;
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
  authType?: 'password' | 'key';
  username?: string;
  password?: string;
  sshKeyId?: number;
  credentialTemplateId?: number;
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
  hostKey?: HostKeyTrustPayload;
  expiresAt: string;
}

export interface HostKeyTrustPayload {
  host: string;
  port: number;
  algorithm: string;
  fingerprintSha256: string;
  publicKey: string;
  knownHostsPath?: string;
  trustedFingerprints?: string[];
}

export interface HostKeyTrustErrorData {
  errorType: 'ssh_host_key_unknown' | 'ssh_host_key_mismatch' | 'ssh_host_key_revoked';
  hostKey: HostKeyTrustPayload;
  probeToken?: string;
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
  productType: string;
  accountName: string;
  accessKeyId: string;
  regionDefault: string;
  status: string;
}

export interface CloudProviderInfo {
  name: string;
  displayName: string;
  productType?: string;
  productTypeName?: string;
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

export interface CredentialTemplate {
  id: string;
  name: string;
  authType: 'password' | 'key';
  sshUser: string;
  port: number;
  sshKeyId?: string;
  sshKeyName?: string;
  description?: string;
  createdBy: number;
  createdAt: string;
  updatedAt: string;
}

export interface CreateCredentialTemplateParams {
  name: string;
  authType: 'password' | 'key';
  sshUser?: string;
  port?: number;
  password?: string;
  sshKeyId?: number;
  description?: string;
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

const toHostKeyTrustPayload = (raw: any): HostKeyTrustPayload => ({
  host: String(raw?.host || '').trim(),
  port: Number(raw?.port || 0),
  algorithm: String(raw?.algorithm || '').trim(),
  fingerprintSha256: String(raw?.fingerprint_sha256 || raw?.fingerprintSha256 || '').trim(),
  publicKey: String(raw?.public_key || raw?.publicKey || '').trim(),
  knownHostsPath: raw?.known_hosts_path || raw?.knownHostsPath || undefined,
  trustedFingerprints: Array.isArray(raw?.trusted_fingerprints)
    ? raw.trusted_fingerprints.map((x: unknown) => String(x).trim()).filter(Boolean)
    : Array.isArray(raw?.trustedFingerprints)
      ? raw.trustedFingerprints.map((x: unknown) => String(x).trim()).filter(Boolean)
      : undefined,
});

const inferHostKeyErrorType = (rawType: unknown, message: unknown): HostKeyTrustErrorData['errorType'] => {
  const normalized = String(rawType || '').trim();
  if (normalized === 'ssh_host_key_unknown' || normalized === 'ssh_host_key_mismatch' || normalized === 'ssh_host_key_revoked') {
    return normalized;
  }
  const lowered = String(message || '').toLowerCase();
  if (lowered.includes('revoked')) {
    return 'ssh_host_key_revoked';
  }
  if (lowered.includes('mismatch')) {
    return 'ssh_host_key_mismatch';
  }
  return 'ssh_host_key_unknown';
};

const throwHostKeyTrustRequired = (raw: any, fallbackType?: HostKeyTrustErrorData['errorType']) => {
  const hostKeyRaw = raw?.host_key || raw?.hostKey;
  if (!hostKeyRaw) {
    return;
  }
  const error: any = new Error(raw?.message || raw?.error_message || 'ssh host key verification failed');
  error.businessCode = 2000;
  error.details = {
    error_type: inferHostKeyErrorType(raw?.error_type || raw?.errorType || fallbackType, raw?.message || raw?.error_message),
    host_key: hostKeyRaw,
    probe_token: raw?.probe_token || raw?.probeToken,
  };
  throw error;
};

export interface CredentialItem {
  id: string;
  name: string;
  description?: string;
  type: 'ssh_key' | 'password' | 'token' | 'certificate';
  authMethod: string;
  hostCount: number;
  tags: string[];
  status: 'available' | 'expiring_soon' | 'expired' | 'disabled';
  expireAt?: string;
  updatedAt: string;
  updatedBy: string;
}

export interface CredentialDetail extends CredentialItem {
  secret?: string;
  createdAt: string;
  createdBy: string;
  usageCount: number;
  successCount: number;
  failureCount: number;
  successRate: number;
  recentUsage?: string;
}

export interface CredentialUsageRecord {
  id: string;
  time: string;
  credentialName: string;
  operator: string;
  target: string;
  method: string;
  result: 'success' | 'failure';
  sourceIp: string;
  remark?: string;
}

export interface CredentialPermissionItem {
  id: string;
  credentialName: string;
  targetUserOrRole: string;
  permissions: string[];
  scope: string;
  effectiveTime: string;
  expireTime: string;
  status: 'active' | 'expired';
}

export interface CredentialStats {
  total: number;
  available: number;
  expiringSoon: number;
  expired: number;
  recentUpdate: string;
  recentUpdateBy: string;
}

export const hostApi = {
  async getHostList(params?: HostListParams): Promise<ApiResponse<PaginatedResponse<Host>>> {
    const response = await apiService.get<Host[]>('/hosts', {
      params: {
        page: params?.page,
        page_size: params?.pageSize,
        keyword: params?.keyword,
        status: params?.status,
        environment: params?.environment,
        region: params?.region,
        tags: params?.tags?.join(','),
        os: params?.os,
      },
    });
    const rawData = response.data as any;
    const items = Array.isArray(rawData) ? rawData : (rawData?.list || []);
    const total = Array.isArray(rawData) ? (response.total || items.length) : Number(rawData?.total || response.total || items.length);
    const list = items.map((item: any) => ({
      id: String(item.id),
      name: item.name,
      ip: item.ip,
      status: item.status,
      healthState: item.health_state || 'unknown',
      monitorStatus: item.monitor_status || undefined,
      cpu: item.cpu_cores ?? item.cpu ?? 0,
      memory: item.memory_mb ?? item.memory ?? 0,
      disk: item.disk_gb ?? item.disk ?? 0,
      cpuUsagePct: Number(item.cpu_usage_pct || 0),
      memoryUsagePct: Number(item.memory_usage_pct || 0),
      diskUsagePct: Number(item.disk_usage_pct || 0),
      network: item.network ?? 0,
      tags: item.tags ?? parseLabels(item.labels),
      environment: item.environment || undefined,
      alertCount: Number(item.alert_count || 0),
      region: item.region ?? '',
      source: item.source,
      provider: item.provider,
      providerInstanceId: item.provider_instance_id,
      parentHostId: item.parent_host_id ? String(item.parent_host_id) : undefined,
      maintenanceReason: item.maintenance_reason || '',
      maintenanceBy: Number(item.maintenance_by || 0),
      maintenanceStartedAt: item.maintenance_started_at || undefined,
      maintenanceUntil: item.maintenance_until || undefined,
      os: item.os || item.os_name || item.osName || '',
      osVersion: item.os_version || item.osVersion || '',
      createdAt: item.created_at ?? item.createdAt,
      lastActive: item.updated_at ?? item.lastActive,
      lastHeartbeatAt: item.last_heartbeat_at || item.lastHeartbeatAt || undefined,
    }));
    return {
      ...response,
      data: {
        list,
        total,
      },
    };
  },

  async getHostOverview(params?: HostListParams): Promise<ApiResponse<HostOverviewStats>> {
    const response = await apiService.get<any>('/hosts/overview', {
      params: {
        keyword: params?.keyword,
        status: params?.status,
        environment: params?.environment,
        region: params?.region,
        tags: params?.tags?.join(','),
        os: params?.os,
      },
    });
    const d = response.data || {};
    return {
      ...response,
      data: {
        totalHosts: Number(d.total_hosts || 0),
        onlineHosts: Number(d.online_hosts || 0),
        abnormalHosts: Number(d.abnormal_hosts || 0),
        avgCpuUsage: Number(d.avg_cpu_usage || 0),
        avgMemoryUsage: Number(d.avg_memory_usage || 0),
        todayAlertCount: Number(d.today_alert_count || 0),
        severeAlertCount: Number(d.severe_alert_count || 0),
        warningAlertCount: Number(d.warning_alert_count || 0),
        onlineRate: Number(d.online_rate || 0),
      },
    };
  },

  async getHostDistribution(params?: HostListParams): Promise<ApiResponse<HostDistributionData[]>> {
    const response = await apiService.get<any>('/hosts/distribution', {
      params: {
        keyword: params?.keyword,
        status: params?.status,
        environment: params?.environment,
        region: params?.region,
        tags: params?.tags?.join(','),
        os: params?.os,
      },
    });
    const rows = Array.isArray(response.data) ? response.data : (response.data?.list || []);
    return {
      ...response,
      data: rows.map((x: any) => ({
        name: String(x.name || ''),
        value: Number(x.value || 0),
        percent: Number(x.percent || 0),
      })),
    };
  },

  async getHostUsageTrend(params?: HostListParams & { hours?: number }): Promise<ApiResponse<HostTrendDataPoint[]>> {
    const response = await apiService.get<any>('/hosts/usage-trend', {
      params: {
        keyword: params?.keyword,
        status: params?.status,
        environment: params?.environment,
        region: params?.region,
        tags: params?.tags?.join(','),
        os: params?.os,
        hours: params?.hours,
      },
    });
    const rows = Array.isArray(response.data) ? response.data : (response.data?.list || []);
    return {
      ...response,
      data: rows.map((x: any) => ({
        time: String(x.time || ''),
        cpuUsage: Number(x.cpu_usage || 0),
        memoryUsage: Number(x.memory_usage || 0),
      })),
    };
  },

  async getHostPendingAlerts(): Promise<ApiResponse<HostPendingAlert[]>> {
    const response = await apiService.get<any>('/hosts/pending-alerts');
    const rows = Array.isArray(response.data) ? response.data : (response.data?.list || []);
    return {
      ...response,
      data: rows.map((x: any) => ({
        name: String(x.name || ''),
        level: String(x.level || 'warning') === 'critical' ? 'critical' : 'warning',
        count: Number(x.count || 0),
      })),
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
        os: item.os || item.os_name || item.osName || '',
        osVersion: item.os_version || item.osVersion || '',
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
      credential_template_id: data.credentialTemplateId,
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
      credential_template_id: data.credentialTemplateId,
    });
    const d = res.data || {};
    throwHostKeyTrustRequired(d);
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
        hostKey: d.host_key ? toHostKeyTrustPayload(d.host_key) : undefined,
        expiresAt: d.expires_at,
      },
    };
  },

  async updateCredentials(id: string, data: { authType: 'password' | 'key'; username: string; password?: string; sshKeyId?: number; port?: number }): Promise<ApiResponse<any>> {
    const res = await apiService.put<any>(`/hosts/${id}/credentials`, {
      auth_type: data.authType,
      username: data.username,
      password: data.password,
      ssh_key_id: data.sshKeyId,
      port: data.port || 22,
    });
    throwHostKeyTrustRequired(res.data);
    return res;
  },

  async trustHostKey(id: string, payload: HostKeyTrustPayload & { replaceExisting?: boolean; probeToken?: string }): Promise<ApiResponse<any>> {
    return apiService.post(`/hosts/${id}/trust-host-key`, {
      host: payload.host,
      port: payload.port,
      algorithm: payload.algorithm,
      fingerprint_sha256: payload.fingerprintSha256,
      public_key: payload.publicKey,
      probe_token: payload.probeToken,
      replace_existing: !!payload.replaceExisting,
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
    throwHostKeyTrustRequired(d);
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
    const res = await apiService.post<any>(`/hosts/${id}/ssh/check`);
    throwHostKeyTrustRequired(res.data);
    return res;
  },

  async sshExec(id: string, command: string): Promise<ApiResponse<SSHExecResult>> {
    return apiService.post(`/hosts/${id}/ssh/exec`, { command });
  },

  async createTerminalSession(id: string): Promise<ApiResponse<HostTerminalSession>> {
    const res = await apiService.post<any>(`/hosts/${id}/terminal/sessions`);
    throwHostKeyTrustRequired(res.data);
    return res;
  },

  async getTerminalSession(id: string, sessionId: string): Promise<ApiResponse<any>> {
    return apiService.get(`/hosts/${id}/terminal/sessions/${sessionId}`);
  },

  async closeTerminalSession(id: string, sessionId: string): Promise<ApiResponse<any>> {
    return apiService.delete(`/hosts/${id}/terminal/sessions/${sessionId}`);
  },

  async listFiles(id: string, dirPath: string): Promise<ApiResponse<{ path: string; list: HostFileItem[]; total: number }>> {
    const res = await apiService.get<any>(`/hosts/${id}/files`, { params: { path: dirPath } });
    throwHostKeyTrustRequired(res.data);
    return res;
  },

  async readFile(id: string, filePath: string): Promise<ApiResponse<{ path: string; content: string }>> {
    const res = await apiService.get<any>(`/hosts/${id}/files/content`, { params: { path: filePath } });
    throwHostKeyTrustRequired(res.data);
    return res;
  },

  async writeFile(id: string, filePath: string, content: string): Promise<ApiResponse<{ path: string; size: number }>> {
    const res = await apiService.put<any>(`/hosts/${id}/files/content`, { path: filePath, content });
    throwHostKeyTrustRequired(res.data);
    return res;
  },

  async uploadFile(id: string, dirPath: string, file: File): Promise<ApiResponse<{ path: string }>> {
    const form = new FormData();
    form.append('file', file);
    const res = await apiService.post<any>(`/hosts/${id}/files/upload`, form, {
      params: { path: dirPath },
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    throwHostKeyTrustRequired(res.data);
    return res;
  },

  async downloadFile(id: string, filePath: string): Promise<Blob> {
    const base = import.meta.env.VITE_API_BASE || '/api/v1';
    const resp = await fetch(
      `${base}/hosts/${id}/files/download?path=${encodeURIComponent(filePath)}`,
      buildContextualFetchInit()
    );
    const contentType = String(resp.headers.get('content-type') || '').toLowerCase();
    if (contentType.includes('application/json')) {
      const payload = await resp.json().catch(() => null);
      if (payload?.data) {
        throwHostKeyTrustRequired(payload.data);
      }
      throw new Error(payload?.msg || payload?.message || `下载失败: ${resp.status}`);
    }
    if (!resp.ok) {
      throw new Error(`下载失败: ${resp.status}`);
    }
    return await resp.blob();
  },

  async mkdir(id: string, dirPath: string): Promise<ApiResponse<{ path: string }>> {
    const res = await apiService.post<any>(`/hosts/${id}/files/mkdir`, { path: dirPath });
    throwHostKeyTrustRequired(res.data);
    return res;
  },

  async renamePath(id: string, oldPath: string, newPath: string): Promise<ApiResponse<{ old_path: string; new_path: string }>> {
    const res = await apiService.post<any>(`/hosts/${id}/files/rename`, { old_path: oldPath, new_path: newPath });
    throwHostKeyTrustRequired(res.data);
    return res;
  },

  async deletePath(id: string, targetPath: string): Promise<ApiResponse<{ path: string }>> {
    const res = await apiService.delete<any>(`/hosts/${id}/files`, { params: { path: targetPath } });
    throwHostKeyTrustRequired(res.data);
    return res;
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

  async getCredentials(params?: any): Promise<ApiResponse<{ list: CredentialItem[]; total: number }>> {
    return apiService.get('/credentials', { params });
  },

  async getCredentialStats(): Promise<ApiResponse<CredentialStats>> {
    return apiService.get('/credentials/stats');
  },

  async getCredentialDetail(id: string): Promise<ApiResponse<CredentialDetail>> {
    return apiService.get(`/credentials/${id}`);
  },

  async getCredentialUsageRecords(params?: any): Promise<ApiResponse<{ list: CredentialUsageRecord[]; total: number }>> {
    return apiService.get('/credentials/usage-records', { params });
  },

  async getCredentialUsageStats(id: string): Promise<ApiResponse<any>> {
    return apiService.get(`/credentials/${id}/usage-stats`);
  },

  async getCredentialPermissions(params?: any): Promise<ApiResponse<{ list: CredentialPermissionItem[]; total: number }>> {
    return apiService.get('/credentials/permissions', { params });
  },

  async listSSHKeys(): Promise<ApiResponse<SSHKeyItem[]>> {
    const res = await apiService.get<any>('/credentials/ssh_keys');
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
    const res = await apiService.get<any>('/hosts/cloud/accounts', { params: { provider } });
    const rawList = Array.isArray(res.data) ? res.data : (res.data?.list || []);
    return {
      ...res,
      data: rawList.map((x: any) => ({
        id: String(x.id),
        provider: x.provider,
        productType: x.product_type || 'uhost',
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
        productType: x.product_type,
        productTypeName: x.product_type_name,
      })),
    };
  },

  async createCloudAccount(payload: any): Promise<ApiResponse<CloudAccount>> {
    const res = await apiService.post<any>('/hosts/cloud/accounts', {
      provider: payload.provider,
      product_type: payload.productType || '',
      account_name: payload.accountName,
      access_key_id: payload.accessKeyId,
      access_key_secret: payload.accessKeySecret,
      region_default: payload.regionDefault || '',
      project_id: payload.projectId || '',
      is_intl: payload.isIntl || false,
    });
    const x = res.data || {};
    return {
      ...res,
      data: {
        id: String(x.id),
        provider: x.provider,
        productType: x.product_type || 'uhost',
        accountName: x.account_name,
        accessKeyId: x.access_key_id,
        regionDefault: x.region_default,
        status: x.status,
      },
    };
  },

  async queryCloudInstances(payload: any): Promise<ApiResponse<CloudInstance[]>> {
    const res = await apiService.post<any>(`/hosts/cloud/providers/${payload.provider}/instances/query`, payload);
    const rawList = Array.isArray(res.data) ? res.data : (res.data?.list || []);
    return {
      ...res,
      data: rawList.map((x: any) => ({
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

  async deleteCloudAccount(accountId: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/hosts/cloud/accounts/${accountId}`);
  },

  async listCloudRegions(provider: string, accountId: string): Promise<ApiResponse<any[]>> {
    const res = await apiService.get<any>(`/hosts/cloud/providers/${provider}/regions`, { params: { account_id: accountId } });
    const rawList = Array.isArray(res.data) ? res.data : (res.data?.list || []);
    return { ...res, data: rawList };
  },

  async listCloudZones(provider: string, accountId: string, region: string): Promise<ApiResponse<any[]>> {
    const res = await apiService.get<any>(`/hosts/cloud/providers/${provider}/zones`, { params: { account_id: accountId, region } });
    const rawList = Array.isArray(res.data) ? res.data : (res.data?.list || []);
    return { ...res, data: rawList };
  },

  async importCloudInstances(payload: any): Promise<ApiResponse<any>> {
    return apiService.post(`/hosts/cloud/providers/${payload.provider}/instances/import`, payload);
  },

  async listCredentialTemplates(): Promise<ApiResponse<CredentialTemplate[]>> {
    const res = await apiService.get<any>('/credentials/templates');
    const rawList = Array.isArray(res.data) ? res.data : (res.data?.list || []);
    return {
      ...res,
      data: rawList.map((x: any) => ({
        id: String(x.id),
        name: x.name,
        authType: x.auth_type,
        sshUser: x.ssh_user,
        port: Number(x.port || 22),
        sshKeyId: x.ssh_key_id ? String(x.ssh_key_id) : undefined,
        sshKeyName: x.ssh_key_name || undefined,
        description: x.description || '',
        createdBy: Number(x.created_by || 0),
        createdAt: x.created_at,
        updatedAt: x.updated_at,
      })),
    };
  },

  async createCredentialTemplate(payload: any): Promise<ApiResponse<CredentialTemplate>> {
    const res = await apiService.post<any>('/credentials/templates', {
      name: payload.name,
      auth_type: payload.authType,
      ssh_user: payload.sshUser || 'root',
      port: payload.port || 22,
      password: payload.password || '',
      ssh_key_id: payload.sshKeyId,
      description: payload.description || '',
    });
    const x = res.data || {};
    return {
      ...res,
      data: {
        id: String(x.id),
        name: x.name,
        authType: x.auth_type,
        sshUser: x.ssh_user,
        port: Number(x.port || 22),
        sshKeyId: x.ssh_key_id ? String(x.ssh_key_id) : undefined,
        description: x.description || '',
        createdBy: Number(x.created_by || 0),
        createdAt: x.created_at,
        updatedAt: x.updated_at,
      },
    };
  },

  async deleteCredentialTemplate(id: string): Promise<ApiResponse<void>> {
    return apiService.delete(`/credentials/templates/${id}`);
  },

  async kvmPreview(hostId: string, payload: any): Promise<ApiResponse<any>> {
    return apiService.post(`/hosts/virtualization/kvm/hosts/${hostId}/preview`, payload);
  },

  async kvmProvision(hostId: string, payload: any): Promise<ApiResponse<any>> {
    return apiService.post(`/hosts/virtualization/kvm/hosts/${hostId}/provision`, payload);
  },

  async getHostProcesses(id: string): Promise<ApiResponse<any[]>> {
    return apiService.get(`/hosts/${id}/processes`);
  },

  async killProcess(id: string, pid: number): Promise<ApiResponse<void>> {
    return apiService.post(`/hosts/${id}/processes/${pid}/kill`);
  },

  async getHostServices(id: string): Promise<ApiResponse<any[]>> {
    return apiService.get(`/hosts/${id}/services`);
  },

  async serviceAction(id: string, name: string, action: string): Promise<ApiResponse<void>> {
    return apiService.post(`/hosts/${id}/services/${name}/actions`, { action });
  },

  async getHostDisks(id: string): Promise<ApiResponse<any[]>> {
    return apiService.get(`/hosts/${id}/disks`);
  },

  async getHostNetworkInterfaces(id: string): Promise<ApiResponse<any[]>> {
    return apiService.get(`/hosts/${id}/network-interfaces`);
  },

  async getHostPackages(id: string): Promise<ApiResponse<any[]>> {
    return apiService.get(`/hosts/${id}/packages`);
  },

  async getHostAlarms(id: string): Promise<ApiResponse<any[]>> {
    return apiService.get(`/hosts/${id}/alarms`);
  },
};
