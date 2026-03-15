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

// V2 types
export interface WorkloadHealth {
  total: number;
  healthy: number;
}

export interface WorkloadStats {
  deployments: WorkloadHealth;
  statefulsets: WorkloadHealth;
  daemonsets: WorkloadHealth;
  services: number;
  ingresses: number;
}

export interface ResourceMetric {
  allocatable: number;
  requested: number;
  usage: number;
  usagePercent: number;
}

export interface PodStats {
  total: number;
  running: number;
  pending: number;
  failed: number;
}

export interface ClusterResource {
  clusterId: number;
  clusterName: string;
  cpu: ResourceMetric;
  memory: ResourceMetric;
  pods: PodStats;
}

export interface DeploymentStats {
  running: number;
  pendingApproval: number;
  todayTotal: number;
  todaySuccess: number;
  todayFailed: number;
}

export interface CICDStats {
  running: number;
  queued: number;
  todayTotal: number;
  success: number;
  failed: number;
}

export interface IssuePodStats {
  total: number;
  byType: Record<string, number>;
}

export interface HealthOverview {
  hosts: HealthStats;
  clusters: HealthStats;
  applications: HealthStats;
  workloads: WorkloadStats;
}

export interface ResourcesOverview {
  hosts: MetricSeries[];
  clusters: ClusterResource[];
}

export interface OperationsOverview {
  deployments: DeploymentStats;
  cicd: CICDStats;
  issuePods: IssuePodStats;
}

export interface OverviewResponseV2 {
  health: HealthOverview;
  resources: ResourcesOverview;
  operations: OperationsOverview;
  alerts: {
    firing: number;
    recent: AlertItem[];
  };
  events: EventItem[];
  ai: AIActivity;
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

// V2 normalizers
const normalizeWorkloadHealth = (data: any): WorkloadHealth => ({
  total: Number(data?.total || 0),
  healthy: Number(data?.healthy || 0),
});

const normalizeWorkloadStats = (data: any): WorkloadStats => ({
  deployments: normalizeWorkloadHealth(data?.deployments || {}),
  statefulsets: normalizeWorkloadHealth(data?.statefulsets || {}),
  daemonsets: normalizeWorkloadHealth(data?.daemonsets || {}),
  services: Number(data?.services || 0),
  ingresses: Number(data?.ingresses || 0),
});

const normalizeResourceMetric = (data: any): ResourceMetric => ({
  allocatable: Number(data?.allocatable || 0),
  requested: Number(data?.requested || 0),
  usage: Number(data?.usage || 0),
  usagePercent: Number(data?.usagePercent || 0),
});

const normalizeClusterResource = (data: any): ClusterResource => ({
  clusterId: Number(data?.clusterId || 0),
  clusterName: String(data?.clusterName || ''),
  cpu: normalizeResourceMetric(data?.cpu || {}),
  memory: normalizeResourceMetric(data?.memory || {}),
  pods: {
    total: Number(data?.pods?.total || 0),
    running: Number(data?.pods?.running || 0),
    pending: Number(data?.pods?.pending || 0),
    failed: Number(data?.pods?.failed || 0),
  },
});

const normalizeDeploymentStats = (data: any): DeploymentStats => ({
  running: Number(data?.running || 0),
  pendingApproval: Number(data?.pendingApproval || 0),
  todayTotal: Number(data?.todayTotal || 0),
  todaySuccess: Number(data?.todaySuccess || 0),
  todayFailed: Number(data?.todayFailed || 0),
});

const normalizeCICDStats = (data: any): CICDStats => ({
  running: Number(data?.running || 0),
  queued: Number(data?.queued || 0),
  todayTotal: Number(data?.todayTotal || 0),
  success: Number(data?.success || 0),
  failed: Number(data?.failed || 0),
});

const normalizeIssuePodStats = (data: any): IssuePodStats => ({
  total: Number(data?.total || 0),
  byType: data?.byType || {},
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

  async getOverviewV2(timeRange: TimeRange = '1h'): Promise<ApiResponse<OverviewResponseV2>> {
    const response = await apiService.get<any>('/dashboard/overview/v2', {
      params: { time_range: timeRange },
    });

    const raw = response.data || {};
    return {
      ...response,
      data: {
        health: {
          hosts: normalizeHealthStats(raw?.health?.hosts),
          clusters: normalizeHealthStats(raw?.health?.clusters),
          applications: normalizeHealthStats(raw?.health?.applications),
          workloads: normalizeWorkloadStats(raw?.health?.workloads),
        },
        resources: {
          hosts: Array.isArray(raw?.resources?.hosts)
            ? raw.resources.hosts.map(normalizeMetricSeries)
            : [],
          clusters: Array.isArray(raw?.resources?.clusters)
            ? raw.resources.clusters.map(normalizeClusterResource)
            : [],
        },
        operations: {
          deployments: normalizeDeploymentStats(raw?.operations?.deployments),
          cicd: normalizeCICDStats(raw?.operations?.cicd),
          issuePods: normalizeIssuePodStats(raw?.operations?.issuePods),
        },
        alerts: {
          firing: Number(raw?.alerts?.firing || 0),
          recent: Array.isArray(raw?.alerts?.recent)
            ? raw.alerts.recent.map(normalizeAlertItem)
            : [],
        },
        events: Array.isArray(raw?.events)
          ? raw.events.map(normalizeEventItem)
          : [],
        ai: normalizeAIActivity(raw?.ai),
      },
    };
  },
};
