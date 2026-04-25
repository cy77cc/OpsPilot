import React, { useEffect, useState, useCallback, useMemo, useRef, useLayoutEffect } from 'react';
import { 
  Button, 
  Card, 
  Col, 
  Dropdown, 
  Input, 
  Progress, 
  Row, 
  Select, 
  Space, 
  Table, 
  Tag, 
  Tooltip,
  message,
  Popconfirm,
  Empty,
  Typography,
  Pagination,
  Badge
} from 'antd';
import { 
  PlusOutlined, 
  ReloadOutlined, 
  DownloadOutlined, 
  MoreOutlined,
  SearchOutlined,
  FilterOutlined,
  ArrowRightOutlined,
  ApiOutlined,
  EditOutlined,
  DownOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  DesktopOutlined,
  DashboardOutlined,
  ToolOutlined,
  BellOutlined,
  ImportOutlined
} from '@ant-design/icons';
import { 
  Layers, 
  Activity, 
  AlertTriangle, 
  HardDrive, 
  Box, 
  Bell,
  ExternalLink,
  Monitor,
  Trash2,
  RefreshCw,
  Info
} from 'lucide-react';
import { Pie, Line } from '@ant-design/charts';
import { useNavigate } from 'react-router-dom';
import { clusterApi, type Cluster } from '../../../api/modules/cluster';
import { dashboardApi, type OverviewResponseV2 } from '../../../api/modules/dashboard';
import { useStableFetch } from '../../../hooks';
import dayjs from 'dayjs';

const { Option } = Select;

// Reusable Sparkline component from HostListPage
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
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} aria-hidden="true" style={{ marginTop: 8 }}>
      <path d={path} fill="none" stroke={color} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
};

const ClusterListPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [dashboardData, setDashboardData] = useState<OverviewResponseV2 | null>(null);
  
  const [searchText, setSearchText] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [envFilter, setEnvFilter] = useState('all');
  const [typeFilter, setTypeFilter] = useState('all');
  
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const tableAreaRef = useRef<HTMLDivElement | null>(null);
  const [tableScrollY, setTableScrollY] = useState(320);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [clusterRes, dashboardRes] = await Promise.all([
        clusterApi.getClusters(),
        dashboardApi.getOverviewV2('6h')
      ]);
      setClusters(clusterRes.data.list || []);
      setDashboardData(dashboardRes.data);
    } catch (err) {
      console.error(err);
      message.error('加载数据失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const load = useStableFetch(fetchData);

  useEffect(() => {
    load();
  }, [load]);

  // Dynamic table height adjustment like HostListPage
  const updateTableScrollY = useCallback(() => {
    const tableArea = tableAreaRef.current;
    if (!tableArea) return;
    const nextHeight = Math.max(180, tableArea.clientHeight - 40); // Subtract header height
    setTableScrollY(nextHeight);
  }, []);

  useLayoutEffect(() => {
    updateTableScrollY();
    window.addEventListener('resize', updateTableScrollY);
    return () => window.removeEventListener('resize', updateTableScrollY);
  }, [updateTableScrollY]);

  const filteredClusters = useMemo(() => {
    return clusters.filter(c => {
      const matchesSearch = c.name.toLowerCase().includes(searchText.toLowerCase()) || 
                            (c.endpoint || '').toLowerCase().includes(searchText.toLowerCase());
      const matchesStatus = statusFilter === 'all' || c.status === statusFilter;
      const matchesEnv = envFilter === 'all' || c.management_mode === envFilter;
      const matchesType = typeFilter === 'all' || c.type === typeFilter;
      return matchesSearch && matchesStatus && matchesEnv && matchesType;
    });
  }, [clusters, searchText, statusFilter, envFilter, typeFilter]);

  const pagedRows = useMemo(() => {
    const start = (page - 1) * pageSize;
    return filteredClusters.slice(start, start + pageSize);
  }, [filteredClusters, page, pageSize]);

  const handleDelete = async (id: number) => {
    try {
      await clusterApi.deleteCluster(id);
      message.success('集群已删除');
      load();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '删除失败');
    }
  };

  const handleTest = async (id: number) => {
    try {
      const res = await clusterApi.testCluster(id);
      if (res.data.connected) {
        message.success(`连接成功 (${res.data.latency_ms}ms)`);
      } else {
        message.error(`连接失败: ${res.data.message}`);
      }
    } catch (err) {
      message.error(err instanceof Error ? err.message : '测试连接失败');
    }
  };

  const trendData = useMemo(() => {
    if (!dashboardData) return [];
    const cpuSeries = dashboardData.resources.cpuUsage[0]?.data || [];
    return cpuSeries.map((point, index) => ({
      time: dayjs(point.timestamp).format('HH:mm'),
      value: point.value,
      type: 'CPU 利用率'
    })).concat((dashboardData.resources.memoryUsage[0]?.data || []).map((point) => ({
      time: dayjs(point.timestamp).format('HH:mm'),
      value: point.value,
      type: '内存利用率'
    })));
  }, [dashboardData]);

  const kpiCards = useMemo(() => [
    {
      title: '集群总数',
      value: String(dashboardData?.health.clusters.total || clusters.length),
      unit: '个',
      subLabel: '较昨日',
      subValue: '+1',
      subColor: '#10b981',
      icon: <Layers size={18} color="#2563eb" />,
      iconBg: '#e8f1ff',
      sparkColor: '#2563eb',
      sparkPoints: [30, 32, 31, 35, 34, 36]
    },
    {
      title: '健康集群',
      value: String(dashboardData?.health.clusters.healthy || clusters.filter(c => c.status === 'active').length),
      unit: '个',
      subLabel: '健康率',
      subValue: '100%',
      subColor: '#10b981',
      icon: <CheckCircleOutlined className="text-[18px] text-[#16a34a]" />,
      iconBg: '#e9f8ef',
      sparkColor: '#16a34a',
      sparkPoints: [98, 100, 100, 100, 100, 100]
    },
    {
      title: '异常集群',
      value: String(dashboardData?.health.clusters.unhealthy || 0),
      unit: '个',
      subLabel: '需要关注',
      subValue: '0',
      subColor: '#ef4444',
      icon: <ExclamationCircleOutlined className="text-[18px] text-[#ef4444]" />,
      iconBg: '#feefef',
      sparkColor: '#ef4444',
      sparkPoints: [2, 1, 0, 0, 0, 0]
    },
    {
      title: '节点总数',
      value: String(dashboardData?.health.hosts.total || clusters.reduce((acc, c) => acc + (c.node_count || 0), 0)),
      unit: '台',
      subLabel: '在线',
      subValue: '4',
      subColor: '#10b981',
      icon: <HardDrive size={18} color="#722ed1" />,
      iconBg: '#f1ecff',
      sparkColor: '#722ed1',
      sparkPoints: [4, 4, 4, 4, 4, 4]
    },
    {
      title: 'Pod 总数',
      value: '8742',
      unit: '个',
      subLabel: '运行中',
      subValue: '8742',
      subColor: '#0ea5e9',
      icon: <Box size={18} color="#13c2c2" />,
      iconBg: '#e6fffb',
      sparkColor: '#13c2c2',
      sparkPoints: [8500, 8600, 8650, 8700, 8720, 8742]
    },
    {
      title: '今日告警数',
      value: String(dashboardData?.alerts.firing || 0),
      unit: '条',
      subLabel: '待处理',
      subValue: '0',
      subColor: '#f59e0b',
      icon: <BellOutlined className="text-[18px] text-[#f59e0b]" />,
      iconBg: '#fff4e8',
      sparkColor: '#f59e0b',
      sparkPoints: [5, 3, 4, 2, 1, 0]
    }
  ], [dashboardData, clusters]);

  const columns = [
    {
      title: '集群名',
      dataIndex: 'name',
      key: 'name',
      width: 180,
      fixed: 'left' as const,
      render: (text: string, record: Cluster) => (
        <Button 
          type="link" 
          className="!p-0 !h-auto !text-[#2563eb] font-medium text-[13px]"
          onClick={() => navigate(`/resources/clusters/${record.id}`)}
        >
          {text}
          <div className="text-[#9ca3af] text-[11px] font-normal mt-0.5">ID: {record.id}</div>
        </Button>
      ),
    },
    {
      title: 'API 地址',
      dataIndex: 'endpoint',
      key: 'endpoint',
      width: 200,
      render: (text: string) => <div className="text-[#6b7280] text-[12px] truncate w-[180px]">{text || '-'}</div>,
    },
    {
      title: '环境',
      dataIndex: 'management_mode',
      key: 'env',
      width: 90,
      render: (val: string) => (
        <Tag color={val === 'managed' ? 'blue' : 'gold'} className="border-none rounded-[4px] px-2 py-0 text-[12px]">
          {val === 'managed' ? '生产' : '测试'}
        </Tag>
      ),
    },
    {
      title: '区域',
      key: 'region',
      width: 80,
      render: () => <span className="text-[#4b5563] text-[12px]">华北</span>,
    },
    {
      title: '类型/版本',
      key: 'type_version',
      width: 130,
      render: (record: Cluster) => (
        <div className="flex flex-col">
          <Tag color="cyan" className="m-0 text-[11px] border-none px-2 py-0 w-fit rounded-[4px]">{record.type || 'K8s'}</Tag>
          <div className="text-[#9ca3af] text-[11px] mt-1">{record.k8s_version || record.version || '-'}</div>
        </div>
      ),
    },
    {
      title: '节点/NS',
      key: 'resources',
      width: 100,
      render: (record: Cluster) => (
        <div className="flex flex-col text-[12px]">
          <div className="text-[#374151] font-medium">节点: {record.node_count || 0}</div>
          <div className="text-[#9ca3af] text-[11px] mt-0.5">NS: 12</div>
        </div>
      ),
    },
    {
      title: '资源使用',
      key: 'usage',
      width: 180,
      render: (record: Cluster) => {
        const clusterStats = dashboardData?.resources.clusters.find(c => c.clusterId === record.id);
        const cpuPercent = clusterStats?.cpu.usagePercent || 0;
        const memPercent = clusterStats?.memory.usagePercent || 0;
        return (
          <div className="flex flex-col gap-2 py-1">
            <div className="flex items-center gap-2">
              <span className="text-[11px] text-[#9ca3af] w-8">CPU</span>
              <Progress percent={cpuPercent} showInfo={false} strokeColor={cpuPercent >= 80 ? '#ef4444' : '#16a34a'} trailColor="#e5e7eb" size={{ height: 4 }} className="flex-1 !m-0" />
              <span className="text-[11px] text-[#4b5563] w-8 text-right">{cpuPercent}%</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-[11px] text-[#9ca3af] w-8">MEM</span>
              <Progress percent={memPercent} showInfo={false} strokeColor="#2563eb" trailColor="#e5e7eb" size={{ height: 4 }} className="flex-1 !m-0" />
              <span className="text-[11px] text-[#4b5563] w-8 text-right">{memPercent}%</span>
            </div>
          </div>
        );
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const isActive = status === 'active' || status === 'connected' || status === 'ready';
        return (
          <span className="inline-flex items-center rounded-[6px] px-2 py-[1px] text-xs font-medium" 
                style={{ backgroundColor: isActive ? '#e8f8f0' : '#f5f6f8', color: isActive ? '#16a34a' : '#6b7280' }}>
            <span className={`w-1.5 h-1.5 rounded-full mr-1.5 ${isActive ? 'bg-[#16a34a]' : 'bg-[#6b7280]'}`} />
            {status || 'unknown'}
          </span>
        );
      },
    },
    {
      title: '操作',
      key: 'action',
      fixed: 'right' as const,
      width: 160,
      render: (_: any, record: Cluster) => (
        <Space size={12}>
          <Tooltip title="详情"><ExternalLink size={16} className="text-[#6b7280] cursor-pointer hover:text-[#2563eb]" onClick={() => navigate(`/resources/clusters/${record.id}`)} /></Tooltip>
          <Tooltip title="节点"><HardDrive size={16} className="text-[#6b7280] cursor-pointer hover:text-[#2563eb]" onClick={() => navigate(`/resources/clusters/${record.id}/nodes`)} /></Tooltip>
          <Tooltip title="监控"><Monitor size={16} className="text-[#6b7280] cursor-pointer hover:text-[#2563eb]" /></Tooltip>
          <Dropdown
            trigger={['click']}
            menu={{
              items: [
                { key: 'test', label: '测试连接', icon: <ApiOutlined />, onClick: () => handleTest(record.id) },
                { key: 'edit', label: '编辑集群', icon: <EditOutlined /> },
                { key: 'sync', label: '同步配置', icon: <RefreshCw size={14} /> },
                { type: 'divider' },
                { 
                  key: 'delete', 
                  label: (
                    <Popconfirm title="确定下线此集群？" onConfirm={() => handleDelete(record.id)}>
                      <span className="text-red-500">下线集群</span>
                    </Popconfirm>
                  ), 
                  icon: <Trash2 size={14} className="text-red-500" />,
                  danger: true 
                },
              ],
            }}
          >
            <MoreOutlined className="text-[#6b7280] cursor-pointer hover:text-[#2563eb] text-lg" />
          </Dropdown>
        </Space>
      ),
    },
  ];

  const distribution = useMemo(() => {
    return [
      { name: 'ACK', value: 4, percent: 16.7 },
      { name: 'EKS', value: 2, percent: 8.3 },
      { name: 'K8s', value: 12, percent: 50.0 },
      { name: 'k3s', value: 6, percent: 25.0 },
    ];
  }, []);

  const pieConfig = {
    data: distribution.map(item => ({ type: item.name, value: item.value })),
    angleField: 'value',
    colorField: 'type',
    innerRadius: 0.7,
    legend: false,
    color: ['#FAAD14', '#722ED1', '#2563eb', '#10b981'],
    label: false,
    height: 140,
    autoFit: true,
    tooltip: { title: 'type' },
    interactions: [{ type: 'element-active' }],
  };

  const lineConfig = {
    data: trendData,
    xField: 'time',
    yField: 'value',
    seriesField: 'type',
    color: ['#16a34a', '#2563eb'],
    point: { size: 2 },
    smooth: true,
    height: 155,
    autoFit: true,
    yAxis: { title: false, grid: { line: { style: { stroke: '#f3f4f6' } } } },
    xAxis: { grid: false },
  };

  return (
    <div className="h-[calc(100vh-112px)] flex flex-col gap-4">
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_300px] flex-1 min-h-0">
        <div className="min-w-0 flex flex-col gap-4 min-h-0">
          {/* KPI Cards Row */}
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6 gap-3">
            {kpiCards.map((card, idx) => (
              <Card key={idx} size="small" styles={{ body: { padding: '14px 16px 12px' } }} className="border border-[#e8edf3] rounded-[10px]">
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
                <Sparkline points={card.sparkPoints} color={card.sparkColor} />
              </Card>
            ))}
          </div>

          {/* Toolbar Card */}
          <Card size="small" styles={{ body: { padding: 12 } }} className="border border-[#e8edf3] rounded-[10px]">
            <div className="overflow-x-auto">
              <div className="inline-flex min-w-max items-center gap-2 whitespace-nowrap">
                <Input 
                  placeholder="搜索集群名/API/标签" 
                  prefix={<SearchOutlined className="text-[#9ca3af]" />} 
                  className="w-[220px]"
                  value={searchText}
                  onChange={e => setSearchText(e.target.value)}
                />
                <Select value={statusFilter} onChange={setStatusFilter} style={{ width: 105 }}>
                  <Option value="all">状态 全部</Option>
                  <Option value="active">在线</Option>
                  <Option value="error">异常</Option>
                </Select>
                <Select value={envFilter} onChange={setEnvFilter} style={{ width: 110 }}>
                  <Option value="all">环境 全部</Option>
                  <Option value="managed">生产</Option>
                  <Option value="unmanaged">测试</Option>
                </Select>
                <Select value={typeFilter} onChange={setTypeFilter} style={{ width: 105 }}>
                  <Option value="all">类型 全部</Option>
                  <Option value="k8s">K8s</Option>
                  <Option value="k3s">k3s</Option>
                </Select>
                
                <span className="mx-1 h-6 w-px bg-[#e5e7eb]" />

                <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>刷新</Button>
                <Dropdown menu={{ 
                  items: [
                    { key: 'bootstrap', label: '创建新集群', icon: <PlusOutlined />, onClick: () => navigate('/resources/clusters/bootstrap') },
                    { key: 'import', label: '导入 Kubeconfig', icon: <ImportOutlined />, onClick: () => navigate('/resources/clusters/import') },
                  ]
                }}>
                  <Button type="primary" icon={<PlusOutlined />}>新增集群 <DownOutlined className="text-[10px]" /></Button>
                </Dropdown>
                <Button icon={<DownloadOutlined />}>导出</Button>
                <Dropdown disabled={selectedRowKeys.length === 0} menu={{
                  items: [
                    { key: 'sync', label: '批量同步配置', icon: <RefreshCw size={14} /> },
                    { type: 'divider' },
                    { key: 'delete', label: '批量下线', icon: <Trash2 size={14} />, danger: true },
                  ],
                }}>
                  <Button>批量操作 <DownOutlined className="text-[10px]" /></Button>
                </Dropdown>
              </div>
            </div>
          </Card>

          {/* Table Card */}
          <Card 
            size="small" 
            styles={{ body: { padding: 0, height: '100%', display: 'flex', flexDirection: 'column' } }} 
            className="border border-[#e8edf3] rounded-[10px] overflow-hidden flex-1 min-h-0"
          >
            <div ref={tableAreaRef} className="flex-1 min-h-0">
              <Table
                rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }}
                columns={columns}
                dataSource={pagedRows}
                rowKey="id"
                loading={loading}
                pagination={false}
                scroll={{ x: 1400, y: tableScrollY }}
                size="small"
                className="[&_.ant-table-thead>tr>th]:!bg-[#f6f8fb] [&_.ant-table-thead>tr>th]:!text-[#6b7280] [&_.ant-table-thead>tr>th]:!text-[13px] [&_.ant-table-tbody>tr>td]:!text-[13px] [&_.ant-table-tbody>tr>td]:!py-[10px]"
              />
            </div>
            <div className="px-4 py-3 border-t border-gray-100 flex flex-wrap gap-3 items-center justify-between">
              <Typography.Text type="secondary">共 {filteredClusters.length} 条</Typography.Text>
              <Pagination
                current={page}
                pageSize={pageSize}
                total={filteredClusters.length}
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

        {/* Sidebar */}
        <div className="space-y-4 h-full overflow-auto pr-1">
          <Card size="small" styles={{ body: { padding: 0 } }} className="border border-[#e8edf3] rounded-[10px] overflow-hidden">
            <div className="px-4 py-3 border-b border-[#edf0f5] flex items-center justify-between">
              <span className="text-[15px] font-semibold text-[#1f2937]">集群分布</span>
              <Select size="small" defaultValue="type" options={[{ value: 'type', label: '集群类型分布' }]} style={{ width: 118 }} />
            </div>
            <div className="px-4 py-3">
              <Pie {...pieConfig} />
              <div className="space-y-2 mt-2">
                {distribution.map((item) => (
                  <div key={item.name} className="flex items-center justify-between text-sm">
                    <div className="flex items-center gap-2">
                      <span className="w-2 h-2 rounded-full" style={{ backgroundColor: pieConfig.color[distribution.indexOf(item)] }} />
                      <span>{item.name}</span>
                    </div>
                    <span className="text-gray-500">{item.value} ({item.percent.toFixed(1)}%)</span>
                  </div>
                ))}
              </div>
            </div>
          </Card>

          <Card size="small" styles={{ body: { padding: 0 } }} className="border border-[#e8edf3] rounded-[10px] overflow-hidden">
            <div className="px-4 py-3">
              <div className="flex items-center justify-between mb-2">
                <div className="text-[15px] font-semibold text-[#1f2937]">资源使用趋势 <span className="text-[#9ca3af] text-xs font-normal">(全部集群)</span></div>
                <Select size="small" defaultValue="6h" options={[{ value: '6h', label: '近 6 小时' }]} style={{ width: 92 }} />
              </div>
              <Line {...lineConfig} />
            </div>
          </Card>

          <Card size="small" styles={{ body: { padding: 0 } }} className="border border-[#e8edf3] rounded-[10px] overflow-hidden">
            <div className="px-4 py-3">
              <div className="flex items-center justify-between mb-2">
                <div className="text-[15px] font-semibold text-[#1f2937]">待处理告警</div>
                <Button type="link" size="small" className="!p-0" onClick={() => navigate('/observability/monitor/alerts')}>更多告警</Button>
              </div>
              {(!dashboardData?.alerts.recent || dashboardData.alerts.recent.length === 0) ? (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无告警" />
              ) : (
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  {(dashboardData.alerts.recent || []).slice(0, 5).map((item, idx) => (
                    <div key={idx} className="flex items-center justify-between text-sm">
                      <div className="flex items-center gap-2 truncate flex-1">
                        <Badge color={item.severity === 'critical' ? '#ef4444' : '#f59e0b'} />
                        <span className="truncate">{item.title}</span>
                      </div>
                      <span className="text-[#9ca3af] text-[11px] ml-2">{dayjs(item.createdAt).fromNow()}</span>
                    </div>
                  ))}
                </Space>
              )}
            </div>
          </Card>

          {/* Assistance Card remains but matches Host style */}
          <Card className="border-none bg-[#1677FF] text-white rounded-[10px]" styles={{ body: { padding: '16px' } }}>
            <div className="flex items-center gap-3">
              <div className="p-2 bg-white/20 rounded-lg"><Info size={20} /></div>
              <div className="flex-1 min-w-0">
                <div className="text-[14px] font-bold">运维助手</div>
                <div className="text-[11px] opacity-80 truncate">集群运行状况良好</div>
              </div>
              <Button ghost size="small" className="flex-shrink-0 border-white text-white text-[11px] h-7 px-2">去处理</Button>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default ClusterListPage;
