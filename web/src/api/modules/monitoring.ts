import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';

const toPositiveIntOrUndefined = (value?: string): number | undefined => {
  if (!value) return undefined;
  const n = Number(value);
  return Number.isInteger(n) && n > 0 ? n : undefined;
};

// 告警数据结构
export interface Alert {
  id: string;
  title: string;
  severity: string;
  source: string;
  status: string;
  createdAt: string;
  latestHealJobId?: string;
  latestHealStatus?: string;
  latestHealUpdatedAt?: string;
  latestHealRunId?: string;
}

// 告警规则数据结构
export interface AlertRule {
  id: string;
  name: string;
  promqlExpr?: string;
  condition?: string;
  metric?: string;
  operator?: string;
  threshold?: number;
  severity: string;
  enabled: boolean;
  channels: string[];
  createdAt: string;
  state?: string;
  windowSec?: number;
  granularitySec?: number;
  dimensionsJson?: string;
}

// 监控指标数据结构
export interface MetricData {
  timestamp: string;
  value: number;
  source?: string;
  labels?: Record<string, any>;
  dimensions?: Record<string, any>;
}

export interface MetricQueryResult {
  window: {
    start: string;
    end: string;
    granularity_sec: number;
  };
  dimensions: Record<string, any>;
  series: MetricData[];
}

// 告警列表请求参数
export interface AlertListParams {
  page?: number;
  pageSize?: number;
  severity?: string;
  status?: string;
  alertId?: string;
}

// 告警规则列表请求参数
export interface AlertRuleListParams {
  page?: number;
  pageSize?: number;
  severity?: string;
  enabled?: boolean;
}

export interface EffectiveRuleListParams {
  projectId?: string;
  page?: number;
  pageSize?: number;
}

// 监控指标请求参数
export interface MetricParams {
  metric: string;
  startTime: string;
  endTime: string;
  granularitySec?: number;
  source?: string;
}

export interface AlertChannel {
  id: string;
  name: string;
  type: string;
  provider: string;
  target: string;
  enabled: boolean;
  projectId?: string;
  configJson?: string;
}

export interface AlertDelivery {
  id: string;
  alertId: string;
  ruleId: string;
  channelId: string;
  channelType: string;
  target: string;
  status: string;
  errorMessage?: string;
  deliveredAt: string;
}

// 监控告警API
export const monitoringApi = {
  // 获取告警列表
  async getAlertList(params?: AlertListParams): Promise<ApiResponse<PaginatedResponse<Alert>>> {
    const response = await apiService.get<Alert[]>('/alerts', {
      params: {
        page: params?.page,
        page_size: params?.pageSize,
        severity: params?.severity,
        status: params?.status,
        alert_id: params?.alertId,
      },
    });
    const raw = Array.isArray(response.data) ? response.data : (response.data as any)?.list || [];
    const total = Array.isArray(response.data) ? response.total || 0 : (response.data as any)?.total || response.total || 0;
    const list = raw.map((item: any) => ({
      id: String(item.id),
      title: item.message || item.title || '',
      severity: item.severity,
      source: item.metric || item.source || '',
      status: item.status,
      createdAt: item.created_at || item.createdAt,
      latestHealJobId: item.latest_heal_job_id || item.latestHealJobId || '',
      latestHealStatus: item.latest_heal_status || item.latestHealStatus || '',
      latestHealUpdatedAt: item.latest_heal_updated_at || item.latestHealUpdatedAt || '',
      latestHealRunId: item.latest_heal_run_id || item.latestHealRunId || '',
    }));
    return {
      ...response,
      data: {
        list,
        total,
      },
    };
  },

  // 获取告警规则列表
  async getAlertRuleList(params?: AlertRuleListParams): Promise<ApiResponse<PaginatedResponse<AlertRule>>> {
    const response = await apiService.get<AlertRule[]>('/alert-rules', {
      params: {
        page: params?.page,
        page_size: params?.pageSize,
      },
    });
    const raw = Array.isArray(response.data) ? response.data : (response.data as any)?.list || [];
    const total = Array.isArray(response.data) ? response.total || 0 : (response.data as any)?.total || response.total || 0;
    const list = raw.map((item: any) => ({
      id: String(item.id),
      name: item.name,
      promqlExpr: item.promql_expr || '',
      condition: `${item.metric} ${item.operator} ${item.threshold}`,
      severity: item.severity,
      enabled: item.enabled,
      channels: item.channels || [],
      createdAt: item.created_at || item.createdAt,
      metric: item.metric,
      operator: item.operator,
      threshold: item.threshold,
      state: item.state,
      windowSec: item.window_sec,
      granularitySec: item.granularity_sec,
      dimensionsJson: item.dimensions_json,
    }));
    return {
      ...response,
      data: {
        list,
        total,
      },
    };
  },
  async getEffectiveRules(params?: EffectiveRuleListParams): Promise<ApiResponse<any>> {
    return apiService.get('/alert-rules/effective', {
      params: {
        project_id: params?.projectId,
        page: params?.page,
        page_size: params?.pageSize,
      },
    });
  },

  // 获取监控指标
  async getMetrics(params: MetricParams): Promise<ApiResponse<MetricQueryResult>> {
    const response = await apiService.get<any>('/metrics', {
      params: {
        metric: params.metric,
        start_time: params.startTime,
        end_time: params.endTime,
        granularity_sec: params.granularitySec,
        source: params.source,
      },
    });
    const rawSeries = (response.data?.series || []) as any[];
    return {
      ...response,
      data: {
        ...(response.data || {}),
        series: rawSeries.map((item: any) => ({
          timestamp: item.timestamp,
          value: Number(item.value || 0),
          source: item.source,
          labels: item.labels || undefined,
          dimensions: item.dimensions || undefined,
        })),
      },
    } as ApiResponse<MetricQueryResult>;
  },
  async syncAlertRules(): Promise<ApiResponse<{ status: string; synced_count: number; synced_at: string }>> {
    return apiService.post('/alerts/rules/sync');
  },
  async createAlertRule(payload: { name: string; metric: string; operator?: string; threshold: number; severity?: string; enabled?: boolean }): Promise<ApiResponse<any>> {
    return apiService.post('/alert-rules', payload);
  },
  async updateAlertRule(id: string, payload: { name?: string; operator?: string; threshold?: number; severity?: string; enabled?: boolean }): Promise<ApiResponse<any>> {
    return apiService.put(`/alert-rules/${encodeURIComponent(id)}`, payload);
  },
  async deleteAlertRule(id: string): Promise<ApiResponse<any>> {
    return apiService.delete(`/alert-rules/${encodeURIComponent(id)}`);
  },
  async enableAlertRule(id: string): Promise<ApiResponse<any>> {
    return apiService.post(`/alert-rules/${encodeURIComponent(id)}/enable`);
  },
  async disableAlertRule(id: string): Promise<ApiResponse<any>> {
    return apiService.post(`/alert-rules/${encodeURIComponent(id)}/disable`);
  },
  async listAlertChannels(): Promise<ApiResponse<PaginatedResponse<AlertChannel>>> {
    const response = await apiService.get<any>('/alert-channels');
    const raw = Array.isArray(response.data) ? response.data : (response.data as any)?.list || [];
    return {
      ...response,
      data: {
        list: raw.map((item: any) => ({
          id: String(item.id),
          name: item.name,
          type: item.type,
          provider: item.provider || '',
          target: item.target || '',
          enabled: !!item.enabled,
          projectId: item.project_id != null ? String(item.project_id) : undefined,
          configJson: item.config_json || '',
        })),
        total: (response.data as any)?.total || response.total || raw.length,
      },
    };
  },
  async createAlertChannel(payload: { name: string; type?: string; provider?: string; target?: string; enabled?: boolean; configJson?: string; projectId?: string }): Promise<ApiResponse<any>> {
    return apiService.post('/alert-channels', {
      name: payload.name,
      type: payload.type,
      provider: payload.provider,
      target: payload.target,
      enabled: payload.enabled,
      config_json: payload.configJson,
      project_id: toPositiveIntOrUndefined(payload.projectId),
    });
  },
  async deleteAlertChannel(id: string): Promise<ApiResponse<any>> {
    return apiService.delete(`/alert-channels/${encodeURIComponent(id)}`);
  },
  async testAlertChannel(payload: { provider: string; target?: string; configJson?: string }): Promise<ApiResponse<any>> {
    return apiService.post('/alert-channels/test', {
      provider: payload.provider,
      target: payload.target,
      config_json: payload.configJson,
    });
  },
  async getRuleChannels(id: string, params?: { projectId?: string }): Promise<ApiResponse<any>> {
    return apiService.get(`/alert-rules/${encodeURIComponent(id)}/channels`, {
      params: {
        project_id: params?.projectId,
      },
    });
  },
  async updateRuleChannels(id: string, channelIds: string[], projectId?: string): Promise<ApiResponse<any>> {
    return apiService.put(`/alert-rules/${encodeURIComponent(id)}/channels`, {
      channel_ids: channelIds.map((x) => Number(x)).filter((x) => Number.isFinite(x) && x > 0),
      project_id: toPositiveIntOrUndefined(projectId),
    });
  },
  async getSeverityRoutes(params?: { projectId?: string }): Promise<ApiResponse<any>> {
    return apiService.get('/alert-routing/severity', {
      params: {
        project_id: params?.projectId,
      },
    });
  },
  async updateSeverityRoutes(payload: any): Promise<ApiResponse<any>> {
    return apiService.put('/alert-routing/severity', payload);
  },
  async createSeverityRoute(payload: { projectId?: string; scope?: string; severity: string; channelIds: string[]; enabled?: boolean }): Promise<ApiResponse<any>> {
    return apiService.post('/alert-routing/severity', {
      project_id: toPositiveIntOrUndefined(payload.projectId),
      scope: payload.scope,
      severity: payload.severity,
      channel_ids: payload.channelIds.map((x) => toPositiveIntOrUndefined(x)).filter((x): x is number => x !== undefined),
      enabled: payload.enabled ?? true,
    });
  },
  async updateSeverityRouteByID(id: string, payload: { projectId?: string; scope?: string; severity: string; channelIds: string[]; enabled?: boolean }): Promise<ApiResponse<any>> {
    return apiService.put(`/alert-routing/severity/${encodeURIComponent(id)}`, {
      project_id: toPositiveIntOrUndefined(payload.projectId),
      scope: payload.scope,
      severity: payload.severity,
      channel_ids: payload.channelIds.map((x) => toPositiveIntOrUndefined(x)).filter((x): x is number => x !== undefined),
      enabled: payload.enabled ?? true,
    });
  },
  async deleteSeverityRoute(id: string, projectId?: string): Promise<ApiResponse<any>> {
    return apiService.delete(`/alert-routing/severity/${encodeURIComponent(id)}`, {
      params: { project_id: toPositiveIntOrUndefined(projectId) },
    });
  },
  async createRuleChannelBinding(ruleId: string, payload: { projectId?: string; channelId: string; priority?: number; enabled?: boolean }): Promise<ApiResponse<any>> {
    return apiService.post(`/alert-rules/${encodeURIComponent(ruleId)}/channels`, {
      project_id: toPositiveIntOrUndefined(payload.projectId),
      channel_id: toPositiveIntOrUndefined(payload.channelId),
      priority: payload.priority,
      enabled: payload.enabled ?? true,
    });
  },
  async updateRuleChannelBinding(ruleId: string, channelId: string, payload: { projectId?: string; priority?: number; enabled?: boolean }): Promise<ApiResponse<any>> {
    return apiService.put(`/alert-rules/${encodeURIComponent(ruleId)}/channels/${encodeURIComponent(channelId)}`, {
      project_id: toPositiveIntOrUndefined(payload.projectId),
      priority: payload.priority,
      enabled: payload.enabled,
    });
  },
  async deleteRuleChannelBinding(ruleId: string, channelId: string, projectId?: string): Promise<ApiResponse<any>> {
    return apiService.delete(`/alert-rules/${encodeURIComponent(ruleId)}/channels/${encodeURIComponent(channelId)}`, {
      params: { project_id: toPositiveIntOrUndefined(projectId) },
    });
  },
  async listAlertDeliveries(params?: { alertId?: string; channelType?: string; status?: string; page?: number; pageSize?: number }): Promise<ApiResponse<PaginatedResponse<AlertDelivery>>> {
    const response = await apiService.get<any>('/alert-deliveries', {
      params: {
        alert_id: params?.alertId,
        channel_type: params?.channelType,
        status: params?.status,
        page: params?.page,
        page_size: params?.pageSize,
      },
    });
    const raw = Array.isArray(response.data) ? response.data : (response.data as any)?.list || [];
    return {
      ...response,
      data: {
        list: raw.map((item: any) => ({
          id: String(item.id),
          alertId: String(item.alert_id || ''),
          ruleId: String(item.rule_id || ''),
          channelId: String(item.channel_id || ''),
          channelType: item.channel_type || '',
          target: item.target || '',
          status: item.status || '',
          errorMessage: item.error_message || '',
          deliveredAt: item.delivered_at || item.created_at || '',
        })),
        total: (response.data as any)?.total || response.total || raw.length,
      },
    };
  },
};
