import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  Dropdown,
  Empty,
  Input,
  message,
  Pagination,
  Popover,
  Progress,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { MenuProps, TableProps } from 'antd';
import {
  BellOutlined,
  CheckCircleOutlined,
  DashboardOutlined,
  DesktopOutlined,
  DownOutlined,
  ExclamationCircleOutlined,
  FilterOutlined,
  PlusOutlined,
  ReloadOutlined,
  SaveOutlined,
  SearchOutlined,
  ToolOutlined,
} from '@ant-design/icons';
import { Pie, Line } from '@ant-design/charts';
import AlibabaCloudColor from '@lobehub/icons/es/AlibabaCloud/components/Color';
import TencentCloudColor from '@lobehub/icons/es/TencentCloud/components/Color';
import VolcengineColor from '@lobehub/icons/es/Volcengine/components/Color';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import { Api } from '../../api';
import type {
  Host,
  HostDistributionData,
  HostListParams,
  HostOverviewStats,
  HostPendingAlert,
  HostTrendDataPoint,
} from '../../api/modules/hosts';
import { useStableFetch } from '../../hooks';
import { PageSkeleton } from '../../components/LoadingSkeleton';

type OnlineFilter = 'all' | 'online' | 'offline' | 'abnormal';
type EnvironmentFilter = 'all' | 'prod' | 'staging' | 'test' | 'dev' | 'ops';
type SourceFilter = 'all' | 'manual_ssh' | 'cloud_import' | 'kvm_provision';
type MonitorStatus = 'healthy' | 'warning' | 'unmanaged';

interface HostTableRow {
  id: string;
  name: string;
  ip: string;
  environment: Exclude<EnvironmentFilter, 'all'>;
  clusterName: string;
  projectName: string;
  clusterProject: string;
  osName: string;
  cpuUsage: number;
  memoryUsage: number;
  diskUsage: number;
  onlineStatus: 'online' | 'offline';
  monitorStatus: MonitorStatus;
  lastHeartbeatLabel: string;
  tags: string[];
  alertCount: number;
  source: Exclude<SourceFilter, 'all'>;
  raw: Host;
}

interface KpiCardMeta {
  key: string;
  title: string;
  value: string;
  unit: string;
  subLabel: string;
  subValue: string;
  subColor: string;
  icon: React.ReactNode;
  iconBg: string;
  sparkColor: string;
  sparkPoints: number[];
}

const EMPTY_OVERVIEW: HostOverviewStats = {
  totalHosts: 0,
  onlineHosts: 0,
  abnormalHosts: 0,
  avgCpuUsage: 0,
  avgMemoryUsage: 0,
  todayAlertCount: 0,
  severeAlertCount: 0,
  warningAlertCount: 0,
  onlineRate: 0,
};

const ENV_META: Record<Exclude<EnvironmentFilter, 'all'>, { label: string; color: string }> = {
  prod: { label: '生产', color: 'blue' },
  staging: { label: '预发', color: 'cyan' },
  test: { label: '测试', color: 'gold' },
  dev: { label: '开发', color: 'orange' },
  ops: { label: '运维', color: 'purple' },
};

const SOURCE_META: Record<Exclude<SourceFilter, 'all'>, { label: string; color: string }> = {
  manual_ssh: { label: 'SSH 接入', color: 'blue' },
  cloud_import: { label: '云厂商导入', color: 'cyan' },
  kvm_provision: { label: '虚拟化创建', color: 'purple' },
};

const CLOUD_PROVIDER_META: Record<string, string> = {
  alicloud: '阿里云',
  tencent: '腾讯云',
  volcengine: '火山云',
  ucloud: 'UCloud',
};

const Sparkline: React.FC<{ points: number[]; color: string }> = ({ points, color }) => {
  const width = 84;
  const height = 22;
  const min = Math.min(...points);
  const max = Math.max(...points);
  const range = max - min || 1;
  const path = points
    .map((point, index) => {
      const x = (index / (points.length - 1)) * width;
      const y = height - ((point - min) / range) * (height - 2) - 1;
      return `${index === 0 ? 'M' : 'L'}${x.toFixed(1)} ${y.toFixed(1)}`;
    })
    .join(' ');

  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} aria-hidden="true">
      <path d={path} fill="none" stroke={color} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
};

const UsageCell: React.FC<{ value: number; color: string }> = ({ value, color }) => (
  <div className="w-[74px]">
    <div className="text-[13px] leading-4 text-[#4b5563] mb-1">{value}%</div>
    <Progress
      percent={value}
      showInfo={false}
      strokeColor={color}
      railColor="#e5e7eb"
      size={{ height: 4 }}
      className="!m-0"
    />
  </div>
);

const StatusPill: React.FC<{ text: string; bg: string; color: string }> = ({ text, bg, color }) => (
  <span className="inline-flex items-center rounded-[6px] px-2 py-[1px] text-xs font-medium" style={{ backgroundColor: bg, color }}>
    {text}
  </span>
);

const parseTags = (host: Host): string[] => {
  if (Array.isArray(host.tags)) {
    return host.tags.map((tag) => String(tag).trim()).filter(Boolean);
  }
  return [];
};

const detectEnvironment = (host: Host): Exclude<EnvironmentFilter, 'all'> => {
  if (host.environment && host.environment in ENV_META) {
    return host.environment;
  }
  const source = `${host.name} ${parseTags(host).join(' ')} ${host.region}`.toLowerCase();
  if (source.includes('prod')) {
    return 'prod';
  }
  if (source.includes('stg') || source.includes('stage') || source.includes('pre')) {
    return 'staging';
  }
  if (source.includes('test')) {
    return 'test';
  }
  if (source.includes('dev')) {
    return 'dev';
  }
  if (source.includes('ops')) {
    return 'ops';
  }
  return 'prod';
};

const deriveClusterProject = (host: Host): { clusterName: string; projectName: string } => {
  const tags = parseTags(host);
  const clusterTag = tags.find((tag) => tag.startsWith('cluster:'));
  const projectTag = tags.find((tag) => tag.startsWith('project:'));
  return {
    clusterName: clusterTag?.replace('cluster:', '') || `${host.region || 'default'}-cluster`,
    projectName: projectTag?.replace('project:', '') || '核心服务',
  };
};

const deriveOnlineStatus = (status: string): 'online' | 'offline' => {
  const normalized = String(status || '').toLowerCase();
  return normalized === 'online' || normalized === 'active' ? 'online' : 'offline';
};

const deriveMonitorStatus = (host: Host): MonitorStatus => {
  if (host.monitorStatus === 'healthy' || host.monitorStatus === 'warning' || host.monitorStatus === 'unmanaged') {
    return host.monitorStatus;
  }
  if (host.healthState === 'healthy') {
    return 'healthy';
  }
  if (host.healthState === 'degraded' || host.healthState === 'critical') {
    return 'warning';
  }
  return 'unmanaged';
};

const deriveSource = (host: Host): Exclude<SourceFilter, 'all'> => {
  const source = String(host.source || '').trim().toLowerCase();
  if (source === 'cloud_import' || source === 'kvm_provision') {
    return source;
  }
  return 'manual_ssh';
};

const formatCloudProvider = (provider?: string): string => {
  const normalized = String(provider || '').trim().toLowerCase();
  if (!normalized) {
    return '';
  }
  return CLOUD_PROVIDER_META[normalized] || normalized;
};

const renderCloudProviderLogo = (provider?: string): React.ReactNode => {
  const normalized = String(provider || '').trim().toLowerCase();
  if (!normalized) {
    return null;
  }
  if (normalized === 'alicloud') {
    return <AlibabaCloudColor size={12} />;
  }
  if (normalized === 'tencent') {
    return <TencentCloudColor size={12} />;
  }
  if (normalized === 'volcengine') {
    return <VolcengineColor size={12} />;
  }
  if (normalized === 'ucloud') {
    return (
      <span
        className="inline-flex h-3.5 min-w-3.5 items-center justify-center rounded-sm bg-[#16a34a] px-1 text-[9px] leading-none text-white"
        aria-hidden="true"
      >
        U
      </span>
    );
  }
  return (
    <span
      className="inline-flex h-3.5 min-w-3.5 items-center justify-center rounded-sm bg-[#6b7280] px-1 text-[9px] leading-none text-white"
      aria-hidden="true"
    >
      C
    </span>
  );
};

const toHeartbeatLabel = (isoTime?: string): string => {
  if (!isoTime) {
    return '--';
  }
  const minutes = Math.max(1, dayjs().diff(dayjs(isoTime), 'minute'));
  if (minutes < 60) {
    return `${minutes} 分钟前`;
  }
  if (minutes < 60 * 24) {
    return `${Math.floor(minutes / 60)} 小时前`;
  }
  return `${Math.floor(minutes / (60 * 24))} 天前`;
};

const normalizePercent = (value: number | undefined): number => {
  const n = Number(value || 0);
  if (n <= 0) {
    return 0;
  }
  if (n > 100) {
    return Math.min(100, Math.round(n));
  }
  return Math.round(n);
};

const buildSparkPoints = (trend: HostTrendDataPoint[], key: 'cpuUsage' | 'memoryUsage', fallback: number): number[] => {
  if (trend.length > 0) {
    return trend.map((point) => Math.max(0, Math.min(100, Number(point[key] || 0))));
  }
  return [fallback - 2, fallback + 1, fallback - 1, fallback + 2, fallback, fallback + 1].map((v) => Math.max(0, Math.min(100, v)));
};

const getOSIcon = (osName: string): string => {
  const normalized = osName.toLowerCase();
  if (normalized.includes('ubuntu')) {
    return '🟠';
  }
  if (normalized.includes('centos') || normalized.includes('rocky')) {
    return '🟣';
  }
  if (normalized.includes('windows')) {
    return '🟦';
  }
  return '⚪';
};

const HostListPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [hosts, setHosts] = useState<Host[]>([]);
  const [overview, setOverview] = useState<HostOverviewStats>(EMPTY_OVERVIEW);
  const [distribution, setDistribution] = useState<HostDistributionData[]>([]);
  const [trend, setTrend] = useState<HostTrendDataPoint[]>([]);
  const [pendingAlerts, setPendingAlerts] = useState<HostPendingAlert[]>([]);

  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<OnlineFilter>('all');
  const [environmentFilter, setEnvironmentFilter] = useState<EnvironmentFilter>('all');
  const [regionFilter, setRegionFilter] = useState('all');
  const [osFilter, setOsFilter] = useState('all');
  const [tagFilter, setTagFilter] = useState<string[]>([]);
  const [sourceFilter, setSourceFilter] = useState<SourceFilter>('all');

  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const tableAreaRef = useRef<HTMLDivElement | null>(null);
  const paginationRef = useRef<HTMLDivElement | null>(null);
  const [tableScrollY, setTableScrollY] = useState(320);

  const queryParams: HostListParams = useMemo(() => ({
    keyword: search.trim() || undefined,
    status: statusFilter === 'all' ? undefined : statusFilter,
    environment: environmentFilter === 'all' ? undefined : environmentFilter,
    region: regionFilter === 'all' ? undefined : regionFilter,
    tags: tagFilter.length > 0 ? tagFilter : undefined,
    os: osFilter === 'all' ? undefined : osFilter,
  }), [search, statusFilter, environmentFilter, regionFilter, tagFilter, osFilter]);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [listRes, overviewRes, distributionRes, trendRes, pendingRes] = await Promise.all([
        Api.hosts.getHostList(queryParams),
        Api.hosts.getHostOverview(queryParams),
        Api.hosts.getHostDistribution(queryParams),
        Api.hosts.getHostUsageTrend({ ...queryParams, hours: 6 }),
        Api.hosts.getHostPendingAlerts(),
      ]);
      setHosts(listRes.data.list || []);
      setOverview(overviewRes.data || EMPTY_OVERVIEW);
      setDistribution(distributionRes.data || []);
      setTrend(trendRes.data || []);
      setPendingAlerts(pendingRes.data || []);
    } finally {
      setLoading(false);
    }
  };

  const load = useStableFetch(fetchData);

  useEffect(() => {
    load();
    const handler = () => load();
    window.addEventListener('project:changed', handler);
    return () => window.removeEventListener('project:changed', handler);
  }, [load]);

  useEffect(() => {
    setPage(1);
  }, [queryParams, sourceFilter]);

  const updateTableScrollY = useCallback(() => {
    const tableArea = tableAreaRef.current;
    if (!tableArea) {
      return;
    }
    const nextHeight = Math.max(180, tableArea.clientHeight);
    setTableScrollY(nextHeight);
  }, []);

  useLayoutEffect(() => {
    updateTableScrollY();
    const tableArea = tableAreaRef.current;
    if (!tableArea || typeof ResizeObserver === 'undefined') {
      return;
    }
    const observer = new ResizeObserver(() => {
      updateTableScrollY();
    });
    observer.observe(tableArea);
    if (paginationRef.current) {
      observer.observe(paginationRef.current);
    }
    window.addEventListener('resize', updateTableScrollY);
    return () => {
      observer.disconnect();
      window.removeEventListener('resize', updateTableScrollY);
    };
  }, [updateTableScrollY]);

  const rows = useMemo<HostTableRow[]>(() => {
    return hosts.map((host) => {
      const tags = parseTags(host);
      const monitorStatus = deriveMonitorStatus(host);
      const alertCount = Number(host.alertCount || 0);
      const clusterInfo = deriveClusterProject(host);
      return {
        id: host.id,
        name: host.name,
        ip: host.ip,
        environment: detectEnvironment(host),
        clusterName: clusterInfo.clusterName,
        projectName: clusterInfo.projectName,
        clusterProject: `${clusterInfo.clusterName} / ${clusterInfo.projectName}`,
        osName: host.os || 'Unknown',
        cpuUsage: normalizePercent(host.cpuUsagePct),
        memoryUsage: normalizePercent(host.memoryUsagePct),
        diskUsage: normalizePercent(host.diskUsagePct),
        onlineStatus: deriveOnlineStatus(host.status),
        monitorStatus,
        lastHeartbeatLabel: toHeartbeatLabel(host.lastHeartbeatAt || host.lastActive),
        tags,
        alertCount,
        source: deriveSource(host),
        raw: host,
      };
    });
  }, [hosts]);

  const filteredRows = useMemo(() => {
    if (sourceFilter === 'all') {
      return rows;
    }
    return rows.filter((row) => row.source === sourceFilter);
  }, [rows, sourceFilter]);

  useEffect(() => {
    updateTableScrollY();
  }, [updateTableScrollY, filteredRows.length, page, pageSize, selectedRowKeys.length]);

  const filterOptions = useMemo(() => {
    const regions = Array.from(new Set(filteredRows.map((row) => row.raw.region).filter(Boolean)));
    const osNames = Array.from(new Set(filteredRows.map((row) => row.osName).filter(Boolean)));
    const tags = Array.from(new Set(filteredRows.flatMap((row) => row.tags).filter(Boolean)));
    return { regions, osNames, tags };
  }, [filteredRows]);

  const pagedRows = useMemo(() => {
    const start = (page - 1) * pageSize;
    return filteredRows.slice(start, start + pageSize);
  }, [filteredRows, page, pageSize]);

  const selectedIds = selectedRowKeys.map(String);
  const activeAdvancedFilterCount = useMemo(() => {
    let count = 0;
    if (regionFilter !== 'all') {
      count += 1;
    }
    if (tagFilter.length > 0) {
      count += 1;
    }
    if (osFilter !== 'all') {
      count += 1;
    }
    if (sourceFilter !== 'all') {
      count += 1;
    }
    return count;
  }, [regionFilter, tagFilter, osFilter, sourceFilter]);

  const kpiCards: KpiCardMeta[] = useMemo(() => {
    const cpuPoints = buildSparkPoints(trend, 'cpuUsage', overview.avgCpuUsage || 30);
    const memoryPoints = buildSparkPoints(trend, 'memoryUsage', overview.avgMemoryUsage || 40);
    return [
      {
        key: 'total',
        title: '主机总数',
        value: String(overview.totalHosts),
        unit: '台',
        subLabel: '较昨日',
        subValue: `${Math.max(0, Math.round(overview.totalHosts * 0.02)) >= 1 ? `+${Math.max(0, Math.round(overview.totalHosts * 0.02))}` : '+0'}`,
        subColor: '#10b981',
        icon: <DesktopOutlined className="text-[18px] text-[#2563eb]" />,
        iconBg: '#e8f1ff',
        sparkColor: '#2563eb',
        sparkPoints: cpuPoints,
      },
      {
        key: 'online',
        title: '在线主机',
        value: String(overview.onlineHosts),
        unit: '台',
        subLabel: '在线率',
        subValue: `${overview.onlineRate.toFixed(1)}%`,
        subColor: '#10b981',
        icon: <CheckCircleOutlined className="text-[18px] text-[#16a34a]" />,
        iconBg: '#e9f8ef',
        sparkColor: '#16a34a',
        sparkPoints: memoryPoints,
      },
      {
        key: 'abnormal',
        title: '异常主机',
        value: String(overview.abnormalHosts),
        unit: '台',
        subLabel: '较昨日',
        subValue: `${overview.abnormalHosts > 0 ? `-${Math.min(overview.abnormalHosts, 3)}` : '-0'}`,
        subColor: '#ef4444',
        icon: <ExclamationCircleOutlined className="text-[18px] text-[#ef4444]" />,
        iconBg: '#feefef',
        sparkColor: '#ef4444',
        sparkPoints: cpuPoints,
      },
      {
        key: 'cpu',
        title: 'CPU 平均利用率',
        value: overview.avgCpuUsage.toFixed(1),
        unit: '%',
        subLabel: '较昨日',
        subValue: '+0.0%',
        subColor: '#0ea5e9',
        icon: <DashboardOutlined className="text-[18px] text-[#2563eb]" />,
        iconBg: '#ebf3ff',
        sparkColor: '#2563eb',
        sparkPoints: cpuPoints,
      },
      {
        key: 'memory',
        title: '内存平均利用率',
        value: overview.avgMemoryUsage.toFixed(1),
        unit: '%',
        subLabel: '较昨日',
        subValue: '+0.0%',
        subColor: '#0ea5e9',
        icon: <ToolOutlined className="text-[18px] text-[#7c3aed]" />,
        iconBg: '#f1ecff',
        sparkColor: '#7c3aed',
        sparkPoints: memoryPoints,
      },
      {
        key: 'alert',
        title: '今日告警数',
        value: String(overview.todayAlertCount),
        unit: '条',
        subLabel: '严重',
        subValue: `${overview.severeAlertCount} / 警告 ${overview.warningAlertCount}`,
        subColor: '#ef4444',
        icon: <BellOutlined className="text-[18px] text-[#f59e0b]" />,
        iconBg: '#fff4e8',
        sparkColor: '#f59e0b',
        sparkPoints: cpuPoints,
      },
    ];
  }, [trend, overview]);

  const onBatchAction = async (action: 'maintenance' | 'online') => {
    if (selectedIds.length === 0) {
      message.warning('请先选择主机');
      return;
    }
    await Api.hosts.batchUpdate({ hostIds: selectedIds, action });
    message.success('批量操作已执行');
    setSelectedRowKeys([]);
    load();
  };

  const onExport = () => {
    if (filteredRows.length === 0) {
      message.info('没有可导出的数据');
      return;
    }
    const header = '主机名,IP,环境,集群项目,操作系统,CPU(%),内存(%),磁盘(%),在线状态,监控状态,最近心跳,告警数\n';
    const body = filteredRows
      .map((row) => [
        row.name,
        row.ip,
        ENV_META[row.environment].label,
        row.clusterProject,
        row.osName,
        row.cpuUsage,
        row.memoryUsage,
        row.diskUsage,
        row.onlineStatus === 'online' ? '在线' : '离线',
        row.monitorStatus === 'healthy' ? '正常' : row.monitorStatus === 'warning' ? '告警' : '未纳管',
        row.lastHeartbeatLabel,
        row.alertCount,
      ].join(','))
      .join('\n');

    const blob = new Blob([`\uFEFF${header}${body}`], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `hosts-${dayjs().format('YYYYMMDD-HHmmss')}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const onRefreshClick: React.MouseEventHandler<HTMLElement> = (event) => {
    event.preventDefault();
    load();
  };

  const rowMoreMenu = (row: HostTableRow): MenuProps => ({
    items: [
      { key: 'alerts', label: '告警记录' },
      { key: 'script', label: '执行脚本' },
      { key: 'labels', label: '标签管理' },
      { key: 'migrate', label: '迁移项目' },
      { key: 'offline', label: '下线主机' },
      { key: 'delete', danger: true, label: '删除' },
    ],
    onClick: async ({ key }) => {
      if (key === 'alerts') {
        navigate(`/monitor?status=firing&keyword=${encodeURIComponent(row.name)}`);
        return;
      }
      if (key === 'offline') {
        await Api.hosts.hostAction(row.id, 'offline');
        message.success('主机已下线');
        load();
        return;
      }
      if (key === 'delete') {
        await Api.hosts.deleteHost(row.id);
        message.success('主机已删除');
        load();
        return;
      }
      message.info('该功能将在后续版本开放');
    },
  });

  const columns: TableProps<HostTableRow>['columns'] = [
    {
      title: '主机名',
      dataIndex: 'name',
      width: 140,
      fixed: 'left',
      ellipsis: false,
      onCell: () => ({
        style: {
          whiteSpace: 'normal',
          wordBreak: 'break-all',
        },
      }),
      render: (value: string, row) => (
        <Button
          type="link"
          className="!px-0 !h-auto !text-[#2563eb] !whitespace-normal !break-all !text-left"
          onClick={() => navigate(`/deployment/infrastructure/hosts/${row.id}`)}
        >
          {value}
        </Button>
      ),
    },
    { title: 'IP 地址', dataIndex: 'ip', width: 125 },
    {
      title: '接入来源',
      dataIndex: 'source',
      width: 100,
      render: (value: Exclude<SourceFilter, 'all'>) => <Tag color={SOURCE_META[value].color}>{SOURCE_META[value].label}</Tag>,
    },
    {
      title: '环境',
      dataIndex: 'environment',
      width: 60,
      render: (value: HostTableRow['environment']) => <Tag color={ENV_META[value].color}>{ENV_META[value].label}</Tag>,
    },
    {
      title: '所属集群/项目',
      dataIndex: 'clusterProject',
      width: 130,
      ellipsis: false,
      onCell: () => ({
        style: {
          whiteSpace: 'normal',
          wordBreak: 'break-word',
          lineHeight: 1.35,
          overflow: 'hidden',
        },
      }),
      render: (_: string, row) => (
        <div className="text-[13px] text-[#374151] overflow-hidden">
          <div className="truncate overflow-hidden">{row.clusterName}</div>
          <div className="truncate text-[#9ca3af] overflow-hidden">{row.projectName}</div>
        </div>
      ),
    },
    {
      title: '操作系统',
      dataIndex: 'osName',
      width: 148,
      ellipsis: false,
      onCell: () => ({
        style: {
          whiteSpace: 'normal',
          wordBreak: 'break-word',
          lineHeight: 1.35,
        },
      }),
      render: (value: string) => (
        <div className="inline-flex items-start gap-1.5 text-[13px] text-[#374151]">
          <span className="mt-[1px]">{getOSIcon(value)}</span>
          <span className="whitespace-normal break-words">{value}</span>
        </div>
      ),
    },
    {
      title: 'CPU',
      dataIndex: 'cpuUsage',
      width: 88,
      render: (value: number) => <UsageCell value={value} color={value >= 80 ? '#ef4444' : '#16a34a'} />,
    },
    {
      title: '内存',
      dataIndex: 'memoryUsage',
      width: 88,
      render: (value: number) => <UsageCell value={value} color="#2563eb" />,
    },
    {
      title: '磁盘',
      dataIndex: 'diskUsage',
      width: 88,
      render: (value: number) => <UsageCell value={value} color={value >= 80 ? '#ef4444' : '#f59e0b'} />,
    },
    {
      title: '在线状态',
      dataIndex: 'onlineStatus',
      width: 88,
      render: (value: 'online' | 'offline') =>
        value === 'online' ? <StatusPill text="在线" bg="#e8f8f0" color="#16a34a" /> : <StatusPill text="离线" bg="#f5f6f8" color="#6b7280" />,
    },
    {
      title: '监控状态',
      dataIndex: 'monitorStatus',
      width: 88,
      render: (value: MonitorStatus) => {
        if (value === 'healthy') {
          return <StatusPill text="正常" bg="#e8f8f0" color="#16a34a" />;
        }
        if (value === 'warning') {
          return <StatusPill text="告警" bg="#fff3e6" color="#d97706" />;
        }
        return <StatusPill text="未纳管" bg="#f5f6f8" color="#6b7280" />;
      },
    },
    { title: '最近心跳', dataIndex: 'lastHeartbeatLabel', width: 92 },
    {
      title: '标签',
      dataIndex: 'tags',
      width: 150,
      render: (tags: string[], row) => {
        const cloudProvider = row.source === 'cloud_import' ? formatCloudProvider(row.raw.provider) : '';
        const visibleTagLimit = cloudProvider ? 1 : 2;
        const visibleTags = tags.slice(0, visibleTagLimit);
        const hiddenCount = Math.max(0, tags.length - visibleTagLimit);
        const showEmpty = !cloudProvider && tags.length === 0;

        return (
          <Space size={4} wrap>
            {cloudProvider ? (
              <Tag color="geekblue">
                <span className="inline-flex items-center gap-1 whitespace-nowrap">
                  <span className="inline-flex items-center leading-none">{renderCloudProviderLogo(row.raw.provider)}</span>
                  <span>{cloudProvider}</span>
                </span>
              </Tag>
            ) : null}
            {visibleTags.map((tag) => (
              <Tag key={tag}>{tag}</Tag>
            ))}
            {hiddenCount > 0 ? <Tag>+{hiddenCount}</Tag> : null}
            {showEmpty ? <span className="text-[#9ca3af]">-</span> : null}
          </Space>
        );
      },
    },
    {
      title: '告警数',
      dataIndex: 'alertCount',
      width: 72,
      align: 'center',
      render: (value: number) => <span style={{ color: value > 0 ? '#ef4444' : '#16a34a', fontWeight: 600 }}>{value}</span>,
    },
    {
      title: '操作',
      key: 'actions',
      fixed: 'right',
      width: 220,
      render: (_, row) => (
        <Space size={8}>
          <Button type="link" size="small" className="!px-0 !h-auto !text-[#2563eb]" onClick={() => navigate(`/deployment/infrastructure/hosts/${row.id}`)}>查看</Button>
          <Button type="link" size="small" className="!px-0 !h-auto !text-[#2563eb]" onClick={() => navigate(`/deployment/infrastructure/hosts/${row.id}/terminal`)}>终端</Button>
          <Button
            type="link"
            size="small"
            className="!px-0 !h-auto !text-[#2563eb]"
            onClick={() => navigate(`/monitor?status=firing&keyword=${encodeURIComponent(row.name || row.ip)}`)}
          >
            监控
          </Button>
          <Dropdown menu={rowMoreMenu(row)} trigger={['click']}>
            <Button type="link" size="small" className="!px-0 !h-auto !text-[#2563eb]">更多 <DownOutlined className="text-[10px]" /></Button>
          </Dropdown>
        </Space>
      ),
    },
  ];

  const pieConfig = {
    data: distribution.map((item) => ({ type: item.name, value: item.value })),
    angleField: 'value',
    colorField: 'type',
    innerRadius: 0.7,
    legend: false,
    color: ['#2563eb', '#10b981', '#94a3b8'],
    label: false,
    height: 150,
    autoFit: true,
    tooltip: { title: 'type' },
    interactions: [{ type: 'element-active' }],
  };

  const trendLineData = [
    ...trend.map((point) => ({ time: point.time, value: point.cpuUsage, type: 'CPU 利用率' })),
    ...trend.map((point) => ({ time: point.time, value: point.memoryUsage, type: '内存利用率' })),
  ];

  const lineConfig = {
    data: trendLineData,
    xField: 'time',
    yField: 'value',
    seriesField: 'type',
    color: ['#2563eb', '#10b981'],
    point: { size: 2 },
    smooth: true,
    height: 155,
    autoFit: true,
    yAxis: { title: false, grid: { line: { style: { stroke: '#f3f4f6' } } } },
    xAxis: { grid: false },
  };

  const initialLoading = loading && rows.length === 0;
  if (initialLoading) {
    return <PageSkeleton />;
  }

  const selectedBatchMenu: MenuProps = {
    items: [
      { key: 'maintenance', label: '批量设为维护' },
      { key: 'online', label: '批量上线' },
    ],
    onClick: ({ key }) => onBatchAction(key as 'maintenance' | 'online'),
  };

  const createHostMenu: MenuProps = {
    items: [
      { key: 'ssh', label: 'SSH 接入' },
      { key: 'cloud', label: '云厂商导入' },
      { key: 'virtualization', label: '虚拟化创建' },
    ],
    onClick: ({ key }) => {
      if (key === 'ssh') {
        navigate('/deployment/infrastructure/hosts/onboarding');
        return;
      }
      if (key === 'cloud') {
        navigate('/deployment/infrastructure/hosts/cloud-import');
        return;
      }
      if (key === 'virtualization') {
        navigate('/deployment/infrastructure/hosts/virtualization');
      }
    },
  };

  const toolbarMoreMenu: MenuProps = {
    items: [
      {
        key: 'export',
        label: (
          <span className="inline-flex items-center gap-2">
            <SaveOutlined />
            导出
          </span>
        ),
      },
    ],
    onClick: ({ key }) => {
      if (key === 'export') {
        onExport();
      }
    },
  };

  return (
    <div className="h-[calc(100vh-112px)] flex flex-col gap-4">
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_300px] flex-1 min-h-0">
        <div className="min-w-0 flex flex-col gap-4 min-h-0">
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6 gap-3">
            {kpiCards.map((card) => (
              <Card key={card.key} size="small" styles={{ body: { padding: '14px 16px 12px' } }} className="border border-[#e8edf3] rounded-[10px]">
                <div className="flex justify-between items-start">
                  <div>
                    <div className="text-[#6b7280] text-[13px]">{card.title}</div>
                    <div className="mt-[6px] flex items-end gap-1">
                      <span className="text-[40px] leading-none font-semibold text-[#111827] tracking-[-0.02em]">{card.value}</span>
                      <span className="text-[18px] leading-6 text-[#374151] mb-[3px]">{card.unit}</span>
                    </div>
                    <div className="text-[13px] text-[#6b7280] mt-[8px]">
                      {card.subLabel} <span style={{ color: card.subColor }}>{card.subValue}</span>
                    </div>
                  </div>
                  <span className="w-10 h-10 rounded-xl inline-flex items-center justify-center" style={{ backgroundColor: card.iconBg }}>
                    {card.icon}
                  </span>
                </div>
                <div className="mt-[8px]"><Sparkline points={card.sparkPoints} color={card.sparkColor} /></div>
              </Card>
            ))}
          </div>

          <Card size="small" styles={{ body: { padding: 12 } }} className="border border-[#e8edf3] rounded-[10px]">
            <div className="overflow-x-auto">
              <div className="inline-flex min-w-max items-center gap-2 whitespace-nowrap">
                <Input
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder="搜索主机名、IP 或标签"
                  style={{ width: 220 }}
                  suffix={<SearchOutlined className="text-[#9ca3af]" />}
                />
                <Select
                  value={statusFilter}
                  onChange={setStatusFilter}
                  style={{ width: 105 }}
                  options={[
                    { value: 'all', label: '状态 全部' },
                    { value: 'online', label: '在线' },
                    { value: 'offline', label: '离线' },
                    { value: 'abnormal', label: '异常' },
                  ]}
                />
                <Select
                  value={environmentFilter}
                  onChange={setEnvironmentFilter}
                  style={{ width: 110 }}
                  options={[{ value: 'all', label: '环境 全部' }, ...Object.entries(ENV_META).map(([value, meta]) => ({ value, label: meta.label }))]}
                />
                <Popover
                  trigger="click"
                  placement="bottomLeft"
                  content={(
                    <div className="w-[320px]">
                      <Space direction="vertical" size={10} style={{ width: '100%' }}>
                        <Select
                          value={regionFilter}
                          onChange={setRegionFilter}
                          style={{ width: '100%' }}
                          options={[{ value: 'all', label: '机房/区域 全部' }, ...filterOptions.regions.map((region) => ({ value: region, label: region }))]}
                        />
                        <Select
                          mode="multiple"
                          allowClear
                          maxTagCount={2}
                          value={tagFilter}
                          onChange={setTagFilter}
                          style={{ width: '100%' }}
                          placeholder="标签"
                          options={filterOptions.tags.map((tag) => ({ value: tag, label: tag }))}
                        />
                        <Select
                          value={osFilter}
                          onChange={setOsFilter}
                          style={{ width: '100%' }}
                          options={[{ value: 'all', label: '操作系统 全部' }, ...filterOptions.osNames.map((osName) => ({ value: osName, label: osName }))]}
                        />
                        <Select
                          value={sourceFilter}
                          onChange={setSourceFilter}
                          style={{ width: '100%' }}
                          options={[
                            { value: 'all', label: '接入来源 全部' },
                            { value: 'manual_ssh', label: 'SSH 接入' },
                            { value: 'cloud_import', label: '云厂商导入' },
                            { value: 'kvm_provision', label: '虚拟化创建' },
                          ]}
                        />
                        <Button
                          onClick={() => {
                            setRegionFilter('all');
                            setTagFilter([]);
                            setOsFilter('all');
                            setSourceFilter('all');
                          }}
                        >
                          重置更多筛选
                        </Button>
                      </Space>
                    </div>
                  )}
                >
                  <Button htmlType="button" icon={<FilterOutlined />}>
                    更多筛选{activeAdvancedFilterCount > 0 ? ` (${activeAdvancedFilterCount})` : ''}
                  </Button>
                </Popover>
                <span className="mx-1 h-6 w-px bg-[#e5e7eb]" />

                <Button htmlType="button" icon={<ReloadOutlined />} onClick={onRefreshClick} loading={loading}>刷新</Button>
                <Dropdown menu={createHostMenu}>
                  <Button htmlType="button" type="primary" icon={<PlusOutlined />}>新增主机 <DownOutlined /></Button>
                </Dropdown>
                <Button htmlType="button" icon={<ToolOutlined />} onClick={() => navigate('/deployment/infrastructure/hosts/credentials')}>密钥凭证管理</Button>
                <Dropdown menu={selectedBatchMenu} disabled={selectedIds.length === 0}>
                  <Button htmlType="button">批量操作 <DownOutlined /></Button>
                </Dropdown>
                <Dropdown menu={toolbarMoreMenu}>
                  <Button htmlType="button">更多 <DownOutlined /></Button>
                </Dropdown>
              </div>
            </div>

            {selectedIds.length > 0 ? (
              <Alert className="mt-3" type="info" showIcon message={`已选择 ${selectedIds.length} 台主机，可执行批量操作。`} />
            ) : null}
          </Card>

          <Card
            size="small"
            styles={{ body: { padding: 0, height: '100%', display: 'flex', flexDirection: 'column' } }}
            className="border border-[#e8edf3] rounded-[10px] overflow-hidden flex-1 min-h-0"
          >
            <div ref={tableAreaRef} className="flex-1 min-h-0">
              <Table<HostTableRow>
                rowKey="id"
                loading={loading}
                columns={columns}
                dataSource={pagedRows}
                locale={{
                  emptyText: (
                    <Empty
                      description={filteredRows.length === 0 ? '暂无符合筛选条件的主机' : '暂无数据'}
                      image={Empty.PRESENTED_IMAGE_SIMPLE}
                    />
                  ),
                }}
                rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }}
                pagination={false}
                scroll={{ x: 1650, y: tableScrollY }}
                size="small"
                style={{ height: '100%' }}
                className="[&_.ant-table-thead>tr>th]:!bg-[#f6f8fb] [&_.ant-table-thead>tr>th]:!text-[#6b7280] [&_.ant-table-thead>tr>th]:!text-[13px] [&_.ant-table-thead>tr>th]:!overflow-hidden [&_.ant-table-tbody>tr>td]:!text-[13px] [&_.ant-table-tbody>tr>td]:!py-[10px] [&_.ant-table-tbody>tr>td]:!overflow-hidden [&_.ant-table-body]:!scrollbar-thin [&_.ant-table-content]:!scrollbar-thin [&_.ant-table-body::-webkit-scrollbar]:!w-1.5 [&_.ant-table-body::-webkit-scrollbar]:!h-1.5 [&_.ant-table-content::-webkit-scrollbar]:!w-1.5 [&_.ant-table-content::-webkit-scrollbar]:!h-1.5"
              />
            </div>
            <div ref={paginationRef} className="px-4 py-3 border-t border-gray-100 flex flex-wrap gap-3 items-center justify-between">
              <Typography.Text type="secondary">共 {filteredRows.length} 条</Typography.Text>
              <Pagination
                current={page}
                pageSize={pageSize}
                total={filteredRows.length}
                showSizeChanger
                showQuickJumper
                pageSizeOptions={[10, 20, 50, 100]}
                onChange={(nextPage, nextPageSize) => {
                  setPage(nextPage);
                  setPageSize(nextPageSize);
                }}
              />
            </div>
          </Card>
        </div>

        <div className="space-y-4 h-full overflow-auto pr-1">
          <Card size="small" styles={{ body: { padding: 0 } }} className="border border-[#e8edf3] rounded-[10px] overflow-hidden">
            <div className="px-4 py-3 border-b border-[#edf0f5] flex items-center justify-between">
              <span className="text-[15px] font-semibold text-[#1f2937]">主机分布</span>
              <Select size="small" defaultValue="os" options={[{ value: 'os', label: '操作系统分布' }]} style={{ width: 118 }} />
            </div>
            <div className="px-4 py-3">
              {distribution.length === 0 ? (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无数据" />
              ) : (
                <>
                  <Pie {...pieConfig} />
                  <div className="space-y-2">
                    {distribution.map((item) => (
                      <div key={item.name} className="flex items-center justify-between text-sm">
                        <span>{item.name}</span>
                        <span className="text-gray-500">{item.value} ({item.percent.toFixed(1)}%)</span>
                      </div>
                    ))}
                  </div>
                </>
              )}
            </div>
          </Card>

          <Card size="small" styles={{ body: { padding: 0 } }} className="border border-[#e8edf3] rounded-[10px] overflow-hidden">
            <div className="px-4 py-3">
              <div className="flex items-center justify-between mb-2">
                <div className="text-[15px] font-semibold text-[#1f2937]">资源使用趋势 <span className="text-[#9ca3af] text-xs font-normal">(全部主机)</span></div>
                <Select size="small" defaultValue="6h" options={[{ value: '6h', label: '近 6 小时' }]} style={{ width: 92 }} />
              </div>
              <Line {...lineConfig} />
            </div>
          </Card>

          <Card size="small" styles={{ body: { padding: 0 } }} className="border border-[#e8edf3] rounded-[10px] overflow-hidden">
            <div className="px-4 py-3">
              <div className="flex items-center justify-between mb-2">
                <div className="text-[15px] font-semibold text-[#1f2937]">待处理告警</div>
                <Button type="link" size="small" className="!px-0" onClick={() => navigate('/monitor?status=firing&source=host')}>更多告警</Button>
              </div>
              {pendingAlerts.length === 0 ? (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无告警" />
              ) : (
                <Space orientation="vertical" size={12} style={{ width: '100%' }}>
                  {pendingAlerts.map((item) => (
                    <div key={`${item.level}-${item.name}`} className="flex items-center justify-between text-sm">
                      <div className="flex items-center gap-2">
                        <Badge color={item.level === 'critical' ? '#ef4444' : '#f59e0b'} />
                        <span>{item.name}</span>
                      </div>
                      <span className={item.level === 'critical' ? 'text-red-500 font-semibold' : 'text-amber-500 font-semibold'}>
                        {item.level === 'critical' ? '严重' : '警告'} {item.count}
                      </span>
                    </div>
                  ))}
                </Space>
              )}
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default HostListPage;
