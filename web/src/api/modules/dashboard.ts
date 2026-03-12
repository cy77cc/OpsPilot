import apiService from '../api';
import type { ApiResponse } from '../api';

export type TimeRange = '1h' | '6h' | '24h';

export interface HealthStats {
  total: number;
  healthy: number;
  degraded?: number;
  unhealthy?: number;
  offline?: number;
}

export interface AlertItem {
  id: string;
  title: string;
  severity: string;
  source: string;
  createdAt: string;
}

export interface EventItem {
  id: string;
  type: string;
  message: string;
  createdAt: string;
}

export interface MetricPoint {
  timestamp: string;
  value: number;
}

export interface MetricSeries {
  hostId: number;
  hostName: string;
  data: MetricPoint[];
}

export interface AIStatsSummary {
  sessionCount: number;
  tokenCount: number;
  avgDurationMs: number;
  successRate: number;
  previousChange?: string;
}

export interface AISessionItem {
  id: string;
  scene: string;
  title: string;
  status: string;
  createdAt: string;
}

export interface AIActivity {
  stats: AIStatsSummary;
  sessions: AISessionItem[];
  byScene: Record<string, number>;
}

export interface OverviewResponse {
  hosts: HealthStats;
  clusters: HealthStats;
  services: HealthStats;
  alerts: {
    firing: number;
    recent: AlertItem[];
  };
  events: EventItem[];
  metrics: {
    cpu_usage: MetricSeries[];
    memory_usage: MetricSeries[];
  };
  ai: AIActivity;
}

const normalizeHealthStats = (data: any): HealthStats => ({
  total: Number(data?.total || 0),
  healthy: Number(data?.healthy || 0),
  degraded: Number(data?.degraded || 0),
  unhealthy: Number(data?.unhealthy || 0),
  offline: Number(data?.offline || 0),
});

const normalizeAlertItem = (item: any): AlertItem => ({
  id: String(item?.id || ''),
  title: String(item?.title || ''),
  severity: String(item?.severity || 'info'),
  source: String(item?.source || ''),
  createdAt: String(item?.createdAt || item?.created_at || ''),
});

const normalizeEventItem = (item: any): EventItem => ({
  id: String(item?.id || ''),
  type: String(item?.type || 'event'),
  message: String(item?.message || ''),
  createdAt: String(item?.createdAt || item?.created_at || ''),
});

const normalizeMetricPoint = (item: any): MetricPoint => ({
  timestamp: String(item?.timestamp || ''),
  value: Number(item?.value || 0),
});

const normalizeMetricSeries = (item: any): MetricSeries => ({
  hostId: Number(item?.hostId || 0),
  hostName: String(item?.hostName || ''),
  data: Array.isArray(item?.data) ? item.data.map(normalizeMetricPoint) : [],
});

const normalizeAIStats = (data: any): AIStatsSummary => ({
  sessionCount: Number(data?.sessionCount || 0),
  tokenCount: Number(data?.tokenCount || 0),
  avgDurationMs: Number(data?.avgDurationMs || 0),
  successRate: Number(data?.successRate || 0),
  previousChange: String(data?.previousChange || ''),
});

const normalizeAISession = (item: any): AISessionItem => ({
  id: String(item?.id || ''),
  scene: String(item?.scene || ''),
  title: String(item?.title || ''),
  status: String(item?.status || 'success'),
  createdAt: String(item?.createdAt || item?.created_at || ''),
});

const normalizeAIActivity = (data: any): AIActivity => ({
  stats: normalizeAIStats(data?.stats || {}),
  sessions: Array.isArray(data?.sessions) ? data.sessions.map(normalizeAISession) : [],
  byScene: data?.byScene || {},
});

export const dashboardApi = {
  async getOverview(timeRange: TimeRange = '1h'): Promise<ApiResponse<OverviewResponse>> {
    const response = await apiService.get<any>('/dashboard/overview', {
      params: { time_range: timeRange },
    });

    const raw = response.data || {};
    return {
      ...response,
      data: {
        hosts: normalizeHealthStats(raw.hosts),
        clusters: normalizeHealthStats(raw.clusters),
        services: normalizeHealthStats(raw.services),
        alerts: {
          firing: Number(raw?.alerts?.firing || 0),
          recent: Array.isArray(raw?.alerts?.recent) ? raw.alerts.recent.map(normalizeAlertItem) : [],
        },
        events: Array.isArray(raw?.events) ? raw.events.map(normalizeEventItem) : [],
        metrics: {
          cpu_usage: Array.isArray(raw?.metrics?.cpu_usage) ? raw.metrics.cpu_usage.map(normalizeMetricSeries) : [],
          memory_usage: Array.isArray(raw?.metrics?.memory_usage) ? raw.metrics.memory_usage.map(normalizeMetricSeries) : [],
        },
        ai: normalizeAIActivity(raw?.ai),
      },
    };
  },
};
