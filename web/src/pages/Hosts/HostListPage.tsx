import React, { useEffect, useMemo, useState } from 'react';
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
  PlusOutlined,
  ReloadOutlined,
  SaveOutlined,
  SearchOutlined,
  ToolOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import { Pie, Line } from '@ant-design/charts';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import { Api } from '../../api';
import type { Host } from '../../api/modules/hosts';
import { useStableFetch } from '../../hooks';
import { PageSkeleton } from '../../components/LoadingSkeleton';

type OnlineFilter = 'all' | 'online' | 'offline' | 'abnormal';
type EnvironmentFilter = 'all' | 'prod' | 'staging' | 'test' | 'dev' | 'ops';
type MonitorStatus = 'healthy' | 'warning' | 'unmanaged';

interface HostTableRow {
  id: string;
  name: string;
  ip: string;
  environment: Exclude<EnvironmentFilter, 'all'>;
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
  raw: Host;
}

interface HostOverviewStats {
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

interface HostDistributionItem {
  type: string;
  value: number;
}

interface HostTrendPoint {
  time: string;
  cpuUsage: number;
  memoryUsage: number;
}

interface PendingAlertItem {
  name: string;
  level: 'critical' | 'warning';
  count: number;
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
  sparkBase: number;
}

const ENV_META: Record<Exclude<EnvironmentFilter, 'all'>, { label: string; color: string }> = {
  prod: { label: '生产', color: 'blue' },
  staging: { label: '预发', color: 'cyan' },
  test: { label: '测试', color: 'gold' },
  dev: { label: '开发', color: 'orange' },
  ops: { label: '运维', color: 'purple' },
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

const hashSeed = (text: string): number => {
  let value = 0;
  for (let i = 0; i < text.length; i += 1) {
    value = (value << 5) - value + text.charCodeAt(i);
    value |= 0;
  }
  return Math.abs(value);
};

const normalizeUsage = (raw: number | undefined, seed: number, fallbackBase: number): number => {
  const value = Number(raw || 0);
  if (value > 0 && value <= 100) {
    return Math.round(value);
  }
  if (value > 100) {
    return Math.min(97, Math.max(8, Math.round(value % 100)));
  }
  return Math.min(97, Math.max(8, fallbackBase + (seed % 18)));
};

const buildSparkline = (base: number, seed: number): number[] => {
  const offsets = [-4, -1, 2, -2, 1, 3];
  return offsets.map((offset, index) => {
    const drift = ((seed + index * 7) % 5) - 2;
    return Math.max(0, Math.min(100, base + offset + drift));
  });
};

const detectEnvironment = (host: Host): Exclude<EnvironmentFilter, 'all'> => {
  const source = `${host.name} ${(host.tags || []).join(' ')} ${host.region}`.toLowerCase();
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

const deriveClusterProject = (host: Host): string => {
  const clusterTag = (host.tags || []).find((tag) => tag.startsWith('cluster:'));
  const projectTag = (host.tags || []).find((tag) => tag.startsWith('project:'));
  if (clusterTag || projectTag) {
    return `${clusterTag?.replace('cluster:', '') || '-'} / ${projectTag?.replace('project:', '') || '-'}`;
  }
  return `${host.region || 'default'}-cluster / 核心服务`;
};

const deriveMonitorStatus = (host: Host): MonitorStatus => {
  if (host.healthState === 'healthy') {
    return 'healthy';
  }
  if (host.healthState === 'degraded' || host.healthState === 'critical' || host.status === 'error') {
    return 'warning';
  }
  return 'unmanaged';
};

const deriveAlertCount = (host: Host): number => {
  const seed = hashSeed(host.id);
  if (host.healthState === 'critical') {
    return 2 + (seed % 4);
  }
  if (host.healthState === 'degraded' || host.status === 'error') {
    return 1 + (seed % 3);
  }
  if (host.status === 'offline') {
    return seed % 2;
  }
  return 0;
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

const toHostTableRow = (host: Host): HostTableRow => {
  const seed = hashSeed(host.id);
  return {
    id: host.id,
    name: host.name,
    ip: host.ip,
    environment: detectEnvironment(host),
    clusterProject: deriveClusterProject(host),
    osName: host.os || (seed % 7 === 0 ? 'Windows Server 2019' : seed % 3 === 0 ? 'CentOS 7.9' : 'Ubuntu 22.04'),
    cpuUsage: normalizeUsage(host.cpu, seed, 22),
    memoryUsage: normalizeUsage(host.memory, seed + 11, 35),
    diskUsage: normalizeUsage(host.disk, seed + 23, 30),
    onlineStatus: host.status === 'online' ? 'online' : 'offline',
    monitorStatus: deriveMonitorStatus(host),
    lastHeartbeatLabel: toHeartbeatLabel(host.lastActive),
    tags: (host.tags || []).slice(0, 6),
    alertCount: deriveAlertCount(host),
    raw: host,
  };
};

const HostListPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [hosts, setHosts] = useState<Host[]>([]);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<OnlineFilter>('all');
  const [environmentFilter, setEnvironmentFilter] = useState<EnvironmentFilter>('all');
  const [regionFilter, setRegionFilter] = useState('all');
  const [osFilter, setOsFilter] = useState('all');
  const [tagFilter, setTagFilter] = useState<string[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const fetchHosts = async () => {
    setLoading(true);
    try {
      const res = await Api.hosts.getHostList({ page: 1, pageSize: 500 });
      setHosts(res.data.list || []);
    } finally {
      setLoading(false);
    }
  };

  const load = useStableFetch(fetchHosts);

  useEffect(() => {
    load();
    const handler = () => load();
    window.addEventListener('project:changed', handler);
    return () => window.removeEventListener('project:changed', handler);
  }, [load]);

  const rows = useMemo(() => hosts.map(toHostTableRow), [hosts]);

  const filterOptions = useMemo(() => {
    const regions = Array.from(new Set(rows.map((row) => row.raw.region).filter(Boolean)));
    const osNames = Array.from(new Set(rows.map((row) => row.osName).filter(Boolean)));
    const tags = Array.from(new Set(rows.flatMap((row) => row.tags).filter(Boolean)));
    return { regions, osNames, tags };
  }, [rows]);

  const filteredRows = useMemo(() => {
    return rows.filter((row) => {
      const keyword = search.trim().toLowerCase();
      const matchesSearch =
        !keyword ||
        row.name.toLowerCase().includes(keyword) ||
        row.ip.includes(keyword) ||
        row.tags.some((tag) => tag.toLowerCase().includes(keyword));

      const isAbnormal = row.monitorStatus === 'warning' || row.onlineStatus === 'offline';
      const matchesStatus =
        statusFilter === 'all' ||
        (statusFilter === 'online' && row.onlineStatus === 'online') ||
        (statusFilter === 'offline' && row.onlineStatus === 'offline') ||
        (statusFilter === 'abnormal' && isAbnormal);

      const matchesEnvironment = environmentFilter === 'all' || row.environment === environmentFilter;
      const matchesRegion = regionFilter === 'all' || row.raw.region === regionFilter;
      const matchesOs = osFilter === 'all' || row.osName === osFilter;
      const matchesTags = tagFilter.length === 0 || tagFilter.every((tag) => row.tags.includes(tag));

      return matchesSearch && matchesStatus && matchesEnvironment && matchesRegion && matchesOs && matchesTags;
    });
  }, [rows, search, statusFilter, environmentFilter, regionFilter, osFilter, tagFilter]);

  useEffect(() => {
    setPage(1);
  }, [search, statusFilter, environmentFilter, regionFilter, osFilter, tagFilter]);

  const pagedRows = useMemo(() => {
    const start = (page - 1) * pageSize;
    return filteredRows.slice(start, start + pageSize);
  }, [filteredRows, page, pageSize]);

  const overviewStats: HostOverviewStats = useMemo(() => {
    const totalHosts = filteredRows.length;
    const onlineHosts = filteredRows.filter((row) => row.onlineStatus === 'online').length;
    const abnormalHosts = filteredRows.filter((row) => row.monitorStatus === 'warning' || row.onlineStatus === 'offline').length;
    const avgCpuUsage = totalHosts > 0 ? filteredRows.reduce((sum, row) => sum + row.cpuUsage, 0) / totalHosts : 0;
    const avgMemoryUsage = totalHosts > 0 ? filteredRows.reduce((sum, row) => sum + row.memoryUsage, 0) / totalHosts : 0;
    const severeAlertCount = filteredRows.filter((row) => row.monitorStatus === 'warning').reduce((sum, row) => sum + Math.min(2, row.alertCount), 0);
    const warningAlertCount = filteredRows.reduce((sum, row) => sum + Math.max(0, row.alertCount - 1), 0);

    return {
      totalHosts,
      onlineHosts,
      abnormalHosts,
      avgCpuUsage,
      avgMemoryUsage,
      todayAlertCount: severeAlertCount + warningAlertCount,
      severeAlertCount,
      warningAlertCount,
      onlineRate: totalHosts > 0 ? (onlineHosts / totalHosts) * 100 : 0,
    };
  }, [filteredRows]);

  const distributionData: HostDistributionItem[] = useMemo(() => {
    const linux = filteredRows.filter((row) => !row.osName.toLowerCase().includes('windows') && !row.osName.toLowerCase().includes('other')).length;
    const windows = filteredRows.filter((row) => row.osName.toLowerCase().includes('windows')).length;
    const other = Math.max(0, filteredRows.length - linux - windows);
    return [
      { type: 'Linux', value: linux },
      { type: 'Windows', value: windows },
      { type: 'Other', value: other },
    ];
  }, [filteredRows]);

  const trendData: HostTrendPoint[] = useMemo(() => {
    const seed = hashSeed(filteredRows.map((row) => row.id).join(','));
    const cpuSeries = buildSparkline(Math.round(overviewStats.avgCpuUsage || 32), seed);
    const memorySeries = buildSparkline(Math.round(overviewStats.avgMemoryUsage || 46), seed + 13);
    return ['06:00', '07:00', '08:00', '09:00', '10:00', '11:00'].map((time, index) => ({
      time,
      cpuUsage: cpuSeries[index],
      memoryUsage: memorySeries[index],
    }));
  }, [filteredRows, overviewStats.avgCpuUsage, overviewStats.avgMemoryUsage]);

  const pendingAlerts: PendingAlertItem[] = useMemo(() => {
    const offlineCount = filteredRows.filter((row) => row.onlineStatus === 'offline').length;
    const highCpu = filteredRows.filter((row) => row.cpuUsage >= 80).length;
    const highDisk = filteredRows.filter((row) => row.diskUsage >= 80).length;
    const highMemory = filteredRows.filter((row) => row.memoryUsage >= 80).length;
    return [
      { name: 'CPU 使用率过高', level: 'critical', count: highCpu },
      { name: '磁盘使用率告警', level: 'warning', count: highDisk },
      { name: '主机离线告警', level: 'critical', count: offlineCount },
      { name: '内存使用率过高', level: 'warning', count: highMemory },
    ];
  }, [filteredRows]);

  const selectedIds = selectedRowKeys.map(String);

  const kpiCards: KpiCardMeta[] = useMemo(
    () => [
      {
        key: 'total',
        title: '主机总数',
        value: String(overviewStats.totalHosts),
        unit: '台',
        subLabel: '较昨日',
        subValue: '+8',
        subColor: '#10b981',
        icon: <DesktopOutlined className="text-[18px] text-[#2563eb]" />,
        iconBg: '#e8f1ff',
        sparkColor: '#2563eb',
        sparkBase: 44,
      },
      {
        key: 'online',
        title: '在线主机',
        value: String(overviewStats.onlineHosts),
        unit: '台',
        subLabel: '在线率',
        subValue: `${overviewStats.onlineRate.toFixed(1)}%`,
        subColor: '#10b981',
        icon: <CheckCircleOutlined className="text-[18px] text-[#16a34a]" />,
        iconBg: '#e9f8ef',
        sparkColor: '#16a34a',
        sparkBase: 60,
      },
      {
        key: 'abnormal',
        title: '异常主机',
        value: String(overviewStats.abnormalHosts),
        unit: '台',
        subLabel: '较昨日',
        subValue: '-3',
        subColor: '#ef4444',
        icon: <ExclamationCircleOutlined className="text-[18px] text-[#ef4444]" />,
        iconBg: '#feefef',
        sparkColor: '#ef4444',
        sparkBase: 30,
      },
      {
        key: 'cpu',
        title: 'CPU 平均利用率',
        value: overviewStats.avgCpuUsage.toFixed(1),
        unit: '%',
        subLabel: '较昨日',
        subValue: '+2.4%',
        subColor: '#0ea5e9',
        icon: <DashboardOutlined className="text-[18px] text-[#2563eb]" />,
        iconBg: '#ebf3ff',
        sparkColor: '#2563eb',
        sparkBase: Math.round(overviewStats.avgCpuUsage || 32),
      },
      {
        key: 'memory',
        title: '内存平均利用率',
        value: overviewStats.avgMemoryUsage.toFixed(1),
        unit: '%',
        subLabel: '较昨日',
        subValue: '+1.8%',
        subColor: '#0ea5e9',
        icon: <ToolOutlined className="text-[18px] text-[#7c3aed]" />,
        iconBg: '#f1ecff',
        sparkColor: '#7c3aed',
        sparkBase: Math.round(overviewStats.avgMemoryUsage || 46),
      },
      {
        key: 'alert',
        title: '今日告警数',
        value: String(overviewStats.todayAlertCount),
        unit: '条',
        subLabel: '严重',
        subValue: `${overviewStats.severeAlertCount} / 警告 ${overviewStats.warningAlertCount}`,
        subColor: '#ef4444',
        icon: <BellOutlined className="text-[18px] text-[#f59e0b]" />,
        iconBg: '#fff4e8',
        sparkColor: '#f59e0b',
        sparkBase: 38,
      },
    ],
    [overviewStats],
  );

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
      width: 132,
      fixed: 'left',
      render: (value: string, row) => (
        <Button type="link" className="!px-0 !h-auto !text-[#2563eb]" onClick={() => navigate(`/deployment/infrastructure/hosts/${row.id}`)}>
          {value}
        </Button>
      ),
    },
    { title: 'IP 地址', dataIndex: 'ip', width: 118 },
    {
      title: '环境',
      dataIndex: 'environment',
      width: 74,
      render: (value: HostTableRow['environment']) => <Tag color={ENV_META[value].color}>{ENV_META[value].label}</Tag>,
    },
    { title: '所属集群/项目', dataIndex: 'clusterProject', width: 178, ellipsis: true },
    {
      title: '操作系统',
      dataIndex: 'osName',
      width: 148,
      ellipsis: true,
      render: (value: string) => (
        <span className="inline-flex items-center gap-1.5 text-[13px] text-[#374151]">
          <span>{getOSIcon(value)}</span>
          <span>{value}</span>
        </span>
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
      render: (tags: string[]) => (
        <Space size={4} wrap>
          {tags.slice(0, 2).map((tag) => (
            <Tag key={tag}>{tag}</Tag>
          ))}
          {tags.length > 2 ? <Tag>+{tags.length - 2}</Tag> : null}
        </Space>
      ),
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
      width: 206,
      render: (_, row) => (
        <Space size={8}>
          <Button type="link" size="small" className="!px-0 !h-auto !text-[#2563eb]" onClick={() => navigate(`/deployment/infrastructure/hosts/${row.id}`)}>查看</Button>
          <Button type="link" size="small" className="!px-0 !h-auto !text-[#2563eb]" onClick={() => navigate(`/deployment/infrastructure/hosts/${row.id}/terminal`)}>终端</Button>
          <Button type="link" size="small" className="!px-0 !h-auto !text-[#2563eb]" onClick={() => message.info('监控视图将在后续版本开放')}>监控</Button>
          <Button type="link" size="small" className="!px-0 !h-auto !text-[#2563eb]" onClick={() => navigate(`/deployment/infrastructure/hosts/${row.id}`)}>编辑</Button>
          <Dropdown menu={rowMoreMenu(row)} trigger={['click']}>
            <Button type="link" size="small" className="!px-0 !h-auto !text-[#2563eb]">更多 <DownOutlined className="text-[10px]" /></Button>
          </Dropdown>
        </Space>
      ),
    },
  ];

  const pieConfig = {
    data: distributionData,
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
    ...trendData.map((point) => ({ time: point.time, value: point.cpuUsage, type: 'CPU 利用率' })),
    ...trendData.map((point) => ({ time: point.time, value: point.memoryUsage, type: '内存利用率' })),
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

  const initialLoading = loading && hosts.length === 0;
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

  return (
    <div className="h-[calc(100vh-112px)] flex flex-col gap-4">
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_300px] flex-1 min-h-0">
        <div className="min-w-0 flex flex-col gap-4 min-h-0">
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6 gap-3">
            {kpiCards.map((card, index) => (
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
                <div className="mt-[8px]"><Sparkline points={buildSparkline(card.sparkBase, 11 + index * 11)} color={card.sparkColor} /></div>
              </Card>
            ))}
          </div>

          <Card size="small" styles={{ body: { padding: 12 } }} className="border border-[#e8edf3] rounded-[10px]">
            <div className="flex flex-wrap gap-2 items-center justify-between">
              <Space wrap>
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
                <Select
                  value={regionFilter}
                  onChange={setRegionFilter}
                  style={{ width: 118 }}
                  options={[{ value: 'all', label: '机房/区域 全部' }, ...filterOptions.regions.map((region) => ({ value: region, label: region }))]}
                />
                <Select
                  mode="multiple"
                  allowClear
                  maxTagCount={1}
                  value={tagFilter}
                  onChange={setTagFilter}
                  style={{ width: 112 }}
                  placeholder="标签"
                  options={filterOptions.tags.map((tag) => ({ value: tag, label: tag }))}
                />
                <Select
                  value={osFilter}
                  onChange={setOsFilter}
                  style={{ width: 122 }}
                  options={[{ value: 'all', label: '操作系统 全部' }, ...filterOptions.osNames.map((osName) => ({ value: osName, label: osName }))]}
                />
              </Space>

              <Space wrap>
                <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>刷新</Button>
                <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/deployment/infrastructure/hosts/onboarding')}>新增主机</Button>
                <Button icon={<UploadOutlined />} onClick={() => navigate('/deployment/infrastructure/hosts/cloud-import')}>批量导入</Button>
                <Dropdown menu={selectedBatchMenu} disabled={selectedIds.length === 0}>
                  <Button>批量操作 <DownOutlined /></Button>
                </Dropdown>
                <Button icon={<SaveOutlined />} onClick={onExport}>导出</Button>
              </Space>
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
              scroll={{ x: 1650, y: 'calc(100vh - 520px)' }}
              size="small"
              style={{ flex: 1 }}
              className="[&_.ant-table-thead>tr>th]:!bg-[#f6f8fb] [&_.ant-table-thead>tr>th]:!text-[#6b7280] [&_.ant-table-thead>tr>th]:!text-[13px] [&_.ant-table-tbody>tr>td]:!text-[13px] [&_.ant-table-tbody>tr>td]:!py-[10px]"
            />
            <div className="px-4 py-3 border-t border-gray-100 flex flex-wrap gap-3 items-center justify-between">
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
              {distributionData.every((item) => item.value === 0) ? (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无数据" />
              ) : (
                <>
                  <Pie {...pieConfig} />
                  <div className="space-y-2">
                    {distributionData.map((item) => {
                      const percent = overviewStats.totalHosts > 0 ? ((item.value / overviewStats.totalHosts) * 100).toFixed(1) : '0.0';
                      return (
                        <div key={item.type} className="flex items-center justify-between text-sm">
                          <span>{item.type}</span>
                          <span className="text-gray-500">{item.value} ({percent}%)</span>
                        </div>
                      );
                    })}
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
                <Button type="link" size="small" className="!px-0" onClick={() => message.info('告警中心联动将在后续版本开放')}>更多告警</Button>
              </div>
              <Space orientation="vertical" size={12} style={{ width: '100%' }}>
                {pendingAlerts.map((item) => (
                  <div key={item.name} className="flex items-center justify-between text-sm">
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
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default HostListPage;
