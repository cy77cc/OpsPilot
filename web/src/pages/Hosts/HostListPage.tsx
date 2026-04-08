import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Dropdown,
  Input,
  Modal,
  Select,
  Space,
  Tag,
  message,
  Row,
  Col,
  Statistic,
  Progress,
  Badge,
  Checkbox,
  Empty,
  Alert,
  Descriptions,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  DesktopOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  CloseCircleOutlined,
  ToolOutlined,
  CodeOutlined,
  MoreOutlined,
  PlayCircleOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import { Api } from '../../api';
import type { Host, HostHealthSnapshot } from '../../api/modules/hosts';
import { useNavigate } from 'react-router-dom';
import { useStableFetch } from '../../hooks';
import { StaggerList, StaggerItem } from '../../components/Motion';
import { PageSkeleton } from '../../components/LoadingSkeleton';

// 云提供商 Logo 组件
const ProviderLogo: React.FC<{ provider: string; size?: number }> = ({ provider, size = 14 }) => {
  const logos: Record<string, React.ReactNode> = {
    volcengine: (
      <svg viewBox="0 0 24 24" width={size} height={size} style={{ display: 'inline-block', verticalAlign: 'middle' }}>
        <path fill="#FF4D4F" d="M19.8772 1.4685L24 2.5326v18.9426l-4.1228 1.0563V1.4685zm-13.3481 9.428l4.115 1.0641v8.9786l-4.115 1.0642v-11.107zM0 2.572l4.115 1.0642v16.7354L0 21.428V2.572zm17.4553 5.6205v11.107l-4.1228-1.0642V9.2568l4.1228-1.0642z"/>
      </svg>
    ),
    alicloud: (
      <svg viewBox="0 0 24 24" width={size} height={size} style={{ display: 'inline-block', verticalAlign: 'middle' }}>
        <path fill="#FF6A00" d="M3.996 4.517h5.291L8.01 6.324 4.153 7.506a1.668 1.668 0 0 0-1.165 1.601v5.786a1.668 1.668 0 0 0 1.165 1.6l3.857 1.183 1.277 1.807H3.996A3.996 3.996 0 0 1 0 15.487V8.513a3.996 3.996 0 0 1 3.996-3.996m16.008 0h-5.291l1.277 1.807 3.857 1.182c.715.227 1.17.889 1.165 1.601v5.786a1.668 1.668 0 0 1-1.165 1.6l-3.857 1.183-1.277 1.807h5.291A3.996 3.996 0 0 0 24 15.487V8.513a3.996 3.996 0 0 0-3.996-3.996m-4.007 8.345H8.002v-1.804h7.995Z"/>
      </svg>
    ),
    tencent: (
      <svg viewBox="0 0 24 24" width={size} height={size} style={{ display: 'inline-block', verticalAlign: 'middle' }}>
        <path fill="#00A4FF" d="M21.395 15.035a40 40 0 0 0-.803-2.264l-1.079-2.695c.001-.032.014-.562.014-.836C19.526 4.632 17.351 0 12 0S4.474 4.632 4.474 9.241c0 .274.013.804.014.836l-1.08 2.695a39 39 0 0 0-.802 2.264c-1.021 3.283-.69 4.643-.438 4.673.54.065 2.103-2.472 2.103-2.472 0 1.469.756 3.387 2.394 4.771-.612.188-1.363.479-1.845.835-.434.32-.379.646-.301.778.343.578 5.883.369 7.482.189 1.6.18 7.14.389 7.483-.189.078-.132.132-.458-.301-.778-.483-.356-1.233-.646-1.846-.836 1.637-1.384 2.393-3.302 2.393-4.771 0 0 1.563 2.537 2.103 2.472.251-.03.581-1.39-.438-4.673"/>
      </svg>
    ),
    aws: (
      <svg viewBox="0 0 24 24" width={size} height={size} style={{ display: 'inline-block', verticalAlign: 'middle' }}>
        <path fill="#FF9900" d="M6.763 10.036c0 .296.032.535.088.71.064.176.144.368.256.576.04.063.056.127.056.183 0 .08-.048.16-.152.24l-.503.335a.383.383 0 0 1-.208.072c-.08 0-.16-.04-.239-.112a2.47 2.47 0 0 1-.287-.375 6.18 6.18 0 0 1-.248-.471c-.622.734-1.405 1.101-2.347 1.101-.67 0-1.205-.191-1.596-.574-.391-.384-.59-.894-.59-1.533 0-.678.239-1.23.726-1.644.487-.415 1.133-.623 1.955-.623.272 0 .551.024.846.064.296.04.6.104.918.176v-.583c0-.607-.127-1.03-.375-1.277-.255-.248-.686-.367-1.3-.367-.28 0-.568.031-.863.103-.295.072-.583.16-.862.272a2.287 2.287 0 0 1-.28.104.488.488 0 0 1-.127.023c-.112 0-.168-.08-.168-.247v-.391c0-.128.016-.224.056-.28a.597.597 0 0 1 .224-.167c.279-.144.614-.264 1.005-.36a4.84 4.84 0 0 1 1.246-.151c.95 0 1.644.216 2.091.647.439.43.662 1.085.662 1.963v2.586zm-3.24 1.214c.263 0 .534-.048.822-.144.287-.096.543-.271.758-.51.128-.152.224-.32.272-.512.047-.191.08-.423.08-.694v-.335a6.66 6.66 0 0 0-.735-.136 6.02 6.02 0 0 0-.75-.048c-.535 0-.926.104-1.19.32-.263.215-.39.518-.39.917 0 .375.095.655.295.846.191.2.47.296.838.296zm6.41.862c-.144 0-.24-.024-.304-.08-.064-.048-.12-.16-.168-.311L7.586 5.55a1.398 1.398 0 0 1-.072-.32c0-.128.064-.2.191-.2h.783c.151 0 .255.025.31.08.065.048.113.16.16.312l1.342 5.284 1.245-5.284c.04-.16.088-.264.151-.312a.549.549 0 0 1 .32-.08h.638c.152 0 .256.025.32.08.063.048.12.16.151.312l1.261 5.348 1.381-5.348c.048-.16.104-.264.16-.312a.52.52 0 0 1 .311-.08h.743c.127 0 .2.065.2.2 0 .04-.009.08-.017.128a1.137 1.137 0 0 1-.056.2l-1.923 6.17c-.048.16-.104.263-.168.311a.51.51 0 0 1-.303.08h-.687c-.151 0-.255-.024-.32-.08-.063-.056-.119-.16-.15-.32l-1.238-5.148-1.23 5.14c-.04.16-.087.264-.15.32-.065.056-.177.08-.32.08zm10.256.215c-.415 0-.83-.048-1.229-.143-.399-.096-.71-.2-.918-.32-.128-.071-.215-.151-.247-.223a.563.563 0 0 1-.048-.224v-.407c0-.167.064-.247.183-.247.048 0 .096.008.144.024.048.016.12.048.2.08.271.12.566.215.878.279.319.064.63.096.95.096.502 0 .894-.088 1.165-.264a.86.86 0 0 0 .415-.758.777.777 0 0 0-.215-.559c-.144-.151-.416-.287-.807-.415l-1.157-.36c-.583-.183-1.014-.454-1.277-.813a1.902 1.902 0 0 1-.4-1.158c0-.335.073-.63.216-.886.144-.255.335-.479.575-.654.24-.184.51-.32.83-.415.32-.096.655-.136 1.006-.136.175 0 .359.008.535.032.183.024.35.056.518.088.16.04.312.08.455.127.144.048.256.096.336.144a.69.69 0 0 1 .24.2.43.43 0 0 1 .071.263v.375c0 .168-.064.256-.184.256a.83.83 0 0 1-.303-.096 3.652 3.652 0 0 0-1.532-.311c-.455 0-.815.071-1.062.223-.248.152-.375.383-.375.71 0 .224.08.416.24.567.159.152.454.304.877.44l1.134.358c.574.184.99.44 1.237.767.247.327.367.702.367 1.117 0 .343-.072.655-.207.926-.144.272-.336.511-.583.703-.248.2-.543.343-.886.447-.36.111-.734.167-1.142.167zM21.698 16.207c-2.626 1.94-6.442 2.969-9.722 2.969-4.598 0-8.74-1.7-11.87-4.526-.247-.223-.024-.527.272-.351 3.384 1.963 7.559 3.153 11.877 3.153 2.914 0 6.114-.607 9.06-1.852.439-.2.814.287.383.607zM22.792 14.961c-.336-.43-2.22-.207-3.074-.103-.255.032-.295-.192-.063-.36 1.5-1.053 3.967-.75 4.254-.399.287.36-.08 2.826-1.485 4.007-.215.184-.423.088-.327-.151.32-.79 1.03-2.57.695-2.994z"/>
      </svg>
    ),
    ucloud: (
      <svg viewBox="0 0 24 24" width={size} height={size} style={{ display: 'inline-block', verticalAlign: 'middle' }}>
        <text x="4" y="17" fontSize="14" fontWeight="bold" fill="#5C6BC0">U</text>
      </svg>
    ),
  };

  return logos[provider] || null;
};

const HostListPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [hosts, setHosts] = useState<Host[]>([]);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [availabilityFilter, setAvailabilityFilter] = useState<'all' | 'available' | 'assigned'>('all');
  const [providerFilter, setProviderFilter] = useState('all');
  const [selected, setSelected] = useState<string[]>([]);
  const [group, setGroup] = useState('');
  const [hostAssignments, setHostAssignments] = useState<Record<string, { clusters: string[]; targets: string[] }>>({});

  // 云提供商配置
  const providerConfig: Record<string, { name: string }> = {
    volcengine: { name: '火山引擎' },
    alicloud: { name: '阿里云' },
    tencent: { name: '腾讯云' },
    aws: { name: 'AWS' },
    ucloud: { name: 'UCloud' },
  };

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await Api.hosts.getHostList({
        page: 1,
        pageSize: 200,
        status: statusFilter === 'all' ? undefined : statusFilter,
        region: group || undefined,
      });
      const hostList = res.data.list || [];
      setHosts(hostList);

      // Load assignment information for each host
      const assignments: Record<string, { clusters: string[]; targets: string[] }> = {};
      for (const host of hostList) {
        assignments[host.id] = { clusters: [], targets: [] };
        // Note: In a real implementation, this would be a batch API call
        // For now, we'll check if the host has cluster_id or target assignments
        // This is a placeholder - the backend should provide this data
      }
      setHostAssignments(assignments);
    } finally {
      setLoading(false);
    }
  };

  // Use stable fetch to prevent duplicate requests (e.g., from React StrictMode)
  const load = useStableFetch(fetchData);

  useEffect(() => {
    load();
    const handler = () => load();
    window.addEventListener('project:changed', handler);
    return () => window.removeEventListener('project:changed', handler);
  }, [statusFilter, group, load]);

  // 统计数据
  const stats = useMemo(() => {
    const online = hosts.filter((h) => h.status === 'online').length;
    const offline = hosts.filter((h) => h.status === 'offline').length;
    const maintenance = hosts.filter((h) => h.status === 'maintenance').length;
    const error = hosts.filter((h) => h.status === 'error').length;
    const healthRate = hosts.length > 0 ? Math.round((online / hosts.length) * 100) : 0;
    return { online, offline, maintenance, error, total: hosts.length, healthRate };
  }, [hosts]);

  const filtered = useMemo(
    () =>
      hosts.filter((h) => {
        const hitSearch =
          h.name.toLowerCase().includes(search.toLowerCase()) ||
          h.ip.includes(search) ||
          (h.region || '').toLowerCase().includes(search.toLowerCase());
        const hitStatus = statusFilter === 'all' || h.status === statusFilter;
        const hitProvider = providerFilter === 'all' || h.provider === providerFilter;

        // Availability filter
        const assignments = hostAssignments[h.id];
        const isAssigned = assignments && (assignments.clusters.length > 0 || assignments.targets.length > 0);
        const hitAvailability =
          availabilityFilter === 'all' ||
          (availabilityFilter === 'assigned' && isAssigned) ||
          (availabilityFilter === 'available' && !isAssigned);

        return hitSearch && hitStatus && hitProvider && hitAvailability;
      }),
    [hosts, search, statusFilter, providerFilter, availabilityFilter, hostAssignments]
  );

  const batchAction = async (action: string) => {
    if (selected.length === 0) {
      message.warning('请选择主机');
      return;
    }
    await Api.hosts.batchUpdate({
      hostIds: selected,
      action,
    });
    message.success('批量操作已执行');
    setSelected([]);
    load();
  };

  const quickAction = async (id: string, action: string) => {
    await Api.hosts.hostAction(id, action);
    message.success('操作成功');
    load();
  };

  const runHealthCheck = async (id: string) => {
    const res = await Api.hosts.runHealthCheck(id, true);
    const data: Partial<HostHealthSnapshot> = res.data || {};
    Modal.info({
      title: '健康检查结果',
      width: 680,
      content: (
        <Descriptions bordered size="small" column={1}>
          <Descriptions.Item label="健康状态">{data.state || 'unknown'}</Descriptions.Item>
          <Descriptions.Item label="连通性">{data.connectivityStatus || '-'}</Descriptions.Item>
          <Descriptions.Item label="资源">{data.resourceStatus || '-'}</Descriptions.Item>
          <Descriptions.Item label="系统">{data.systemStatus || '-'}</Descriptions.Item>
          <Descriptions.Item label="延迟">{data.latencyMs || 0} ms</Descriptions.Item>
          <Descriptions.Item label="错误">{data.errorMessage || '-'}</Descriptions.Item>
        </Descriptions>
      ),
    });
  };

  const batchExec = async () => {
    if (selected.length === 0) {
      message.warning('请选择主机');
      return;
    }
    let command = 'hostname';
    Modal.confirm({
      title: '批量命令执行（二次确认）',
      width: 720,
      content: (
        <Space direction="vertical" style={{ width: '100%' }}>
          <Alert type="warning" showIcon message="高风险操作" description={`即将在 ${selected.length} 台主机执行命令，请确认影响范围。`} />
          <Input defaultValue={command} onChange={(e) => { command = e.target.value; }} placeholder="请输入命令" />
        </Space>
      ),
      onOk: async () => {
        if (!command.trim()) throw new Error('命令不能为空');
        const res = await Api.hosts.batchExec(selected, command.trim());
        message.success(`批量执行完成: ${Object.keys(res.data || {}).length} 台`);
      },
    });
  };

  // 获取状态配置
  const getStatusConfig = (status: string) => {
    const configs: Record<string, { icon: React.ReactNode; color: string; text: string }> = {
      online: { icon: <CheckCircleOutlined />, color: 'success', text: '在线' },
      offline: { icon: <CloseCircleOutlined />, color: 'default', text: '离线' },
      maintenance: { icon: <ToolOutlined />, color: 'warning', text: '维护中' },
      error: { icon: <ExclamationCircleOutlined />, color: 'error', text: '错误' },
    };
    return configs[status] || { icon: null, color: 'default', text: status };
  };

  // 主机卡片组件
  const HostCard: React.FC<{ host: Host }> = ({ host }) => {
    const statusConfig = getStatusConfig(host.status);
    const isSelected = selected.includes(host.id);
    const assignments = hostAssignments[host.id] || { clusters: [], targets: [] };
    const isAssigned = assignments.clusters.length > 0 || assignments.targets.length > 0;
    const provider = host.provider ? providerConfig[host.provider] : null;

    return (
      <Card
        hoverable
        className="transition-all duration-200 flex flex-col"
        styles={{ body: { padding: '16px', flex: 1, display: 'flex', flexDirection: 'column' } }}
        style={{
          height: 280,
          borderColor: isSelected ? '#6366f1' : undefined,
          boxShadow: isSelected ? '0 0 0 2px rgba(99, 102, 241, 0.1)' : undefined,
        }}
      >
        {/* 头部：名称和操作 */}
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-2 flex-1 min-w-0">
            <Checkbox
              checked={isSelected}
              onChange={(e) => {
                if (e.target.checked) {
                  setSelected([...selected, host.id]);
                } else {
                  setSelected(selected.filter((id) => id !== host.id));
                }
              }}
            />
            <Tooltip title={host.name}>
              <a
                onClick={() => navigate(`/deployment/infrastructure/hosts/${host.id}`)}
                className="text-base font-semibold text-gray-900 hover:text-primary-600 truncate max-w-[160px] block"
              >
                {host.name}
              </a>
            </Tooltip>
          </div>
          <Dropdown
            menu={{
              items: [
                { key: 'check', icon: <CheckCircleOutlined />, label: '健康检查' },
                { key: 'restart', icon: <PlayCircleOutlined />, label: '重启' },
                { key: 'ssh', icon: <CodeOutlined />, label: 'SSH 执行' },
                { key: 'terminal', icon: <CodeOutlined />, label: '打开终端' },
                { type: 'divider' },
                { key: 'maintenance', icon: <ToolOutlined />, label: '设为维护' },
                { key: 'delete', icon: <DeleteOutlined />, label: '删除主机', danger: true },
              ],
              onClick: async ({ key }) => {
                if (key === 'check') {
                  await runHealthCheck(host.id);
                } else if (key === 'restart') {
                  await quickAction(host.id, key);
                } else if (key === 'delete') {
                  Modal.confirm({
                    title: '确认删除主机',
                    content: `确定要删除主机 "${host.name}" (${host.ip}) 吗？此操作不可恢复。`,
                    okText: '确认删除',
                    okButtonProps: { danger: true },
                    onOk: async () => {
                      await Api.hosts.deleteHost(host.id);
                      message.success('主机已删除');
                      await load();
                    },
                  });
                } else if (key === 'ssh') {
                  let command = 'uptime';
                  Modal.confirm({
                    title: 'SSH 命令执行（二次确认）',
                    width: 720,
                    content: (
                      <Space direction="vertical" style={{ width: '100%' }}>
                        <Alert type="warning" showIcon message="请确认目标主机与命令风险" description={`目标: ${host.name}(${host.ip})`} />
                        <Input defaultValue={command} onChange={(e) => { command = e.target.value; }} placeholder="请输入命令" />
                      </Space>
                    ),
                    onOk: async () => {
                      const res = await Api.hosts.sshExec(host.id, command.trim());
                      Modal.info({
                        title: '执行结果',
                        content: <pre>{res.data.stdout || res.data.stderr || ''}</pre>,
                        width: 720,
                      });
                    },
                  });
                } else if (key === 'terminal') {
                  navigate(`/deployment/infrastructure/hosts/${host.id}/terminal`);
                } else if (key === 'maintenance') {
                  let reason = 'scheduled-maintenance';
                  Modal.confirm({
                    title: '设为维护',
                    content: (
                      <Space direction="vertical" style={{ width: '100%' }}>
                        <Input defaultValue={reason} onChange={(e) => { reason = e.target.value; }} placeholder="维护原因" />
                      </Space>
                    ),
                    onOk: async () => {
                      await Api.hosts.hostAction(host.id, 'maintenance', { reason: reason.trim() });
                      message.success('已设置维护');
                      await load();
                    },
                  });
                }
              },
            }}
          >
            <Button type="text" size="small" icon={<MoreOutlined />} />
          </Dropdown>
        </div>

        {/* 状态标签行 */}
        <div className="flex items-center gap-1.5 mb-2 flex-wrap">
          <Tag color={statusConfig.color} icon={statusConfig.icon} style={{ marginRight: 0 }}>
            {statusConfig.text}
          </Tag>
          <Tag color={host.healthState === 'healthy' ? 'green' : host.healthState === 'degraded' ? 'orange' : host.healthState === 'critical' ? 'red' : 'default'} style={{ marginRight: 0 }}>
            {host.healthState || 'unknown'}
          </Tag>
          {provider && (
            <Tag style={{ marginRight: 0, display: 'inline-flex', alignItems: 'center', gap: 4 }}>
              <ProviderLogo provider={host.provider} size={14} />
              <span>{provider.name}</span>
            </Tag>
          )}
          {isAssigned && <Tag color="blue" style={{ marginRight: 0 }}>已分配</Tag>}
        </div>

        {/* 信息行 */}
        <div className="text-sm text-gray-500 mb-3 truncate">
          <span className="mr-3">IP: {host.ip}</span>
          {host.region && <span>区域: {host.region}</span>}
        </div>

        {/* 资源使用情况 */}
        <div className="space-y-2 mt-auto">
          <div>
            <div className="flex justify-between text-sm mb-1">
              <span className="text-gray-600">CPU</span>
              <span className="text-gray-800 font-medium">{host.cpu || 0}%</span>
            </div>
            <Progress
              percent={Math.min(100, host.cpu || 0)}
              strokeColor={(host.cpu || 0) > 80 ? '#ef4444' : (host.cpu || 0) > 60 ? '#f59e0b' : '#10b981'}
              showInfo={false}
              size="small"
            />
          </div>
          <div>
            <div className="flex justify-between text-sm mb-1">
              <span className="text-gray-600">内存</span>
              <span className="text-gray-800 font-medium">{host.memory || 0} MB</span>
            </div>
            <Progress
              percent={Math.min(100, ((host.memory || 0) / 16384) * 100)}
              strokeColor="#6366f1"
              showInfo={false}
              size="small"
            />
          </div>
          <div>
            <div className="flex justify-between text-sm mb-1">
              <span className="text-gray-600">磁盘</span>
              <span className="text-gray-800 font-medium">{host.disk || 0} GB</span>
            </div>
            <Progress
              percent={Math.min(100, ((host.disk || 0) / 500) * 100)}
              strokeColor="#8b5cf6"
              showInfo={false}
              size="small"
            />
          </div>
        </div>
      </Card>
    );
  };

  const isInitialLoading = loading && hosts.length === 0;

  if (isInitialLoading) {
    return <PageSkeleton />;
  }

  return (
    <div className="space-y-6">
      {/* 页面头部 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">主机管理</h1>
          <p className="text-sm text-gray-500 mt-1">管理和监控所有主机资源</p>
        </div>
        <Space>
          <Button onClick={() => navigate('/deployment/infrastructure/hosts/credentials')}>
            凭证管理
          </Button>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading && !isInitialLoading}>
            刷新
          </Button>
          <Dropdown
            menu={{
              items: [
                { key: 'onboarding', label: 'SSH 接入（密码/密钥）' },
                { key: 'cloud', label: '云平台导入（阿里云/腾讯云）' },
                { key: 'virt', label: 'KVM 虚拟化创建' },
              ],
              onClick: ({ key }) => {
                if (key === 'onboarding') navigate('/hosts/onboarding');
                if (key === 'cloud') navigate('/hosts/cloud-import');
                if (key === 'virt') navigate('/hosts/virtualization');
              },
            }}
          >
            <Button type="primary" icon={<PlusOutlined />}>
              新增主机
            </Button>
          </Dropdown>
        </Space>
      </div>

      {/* 统计卡片 */}
      <StaggerList staggerDelay={0.05}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card
                className="hover:shadow-lg transition-shadow cursor-pointer"
                onClick={() => setStatusFilter('all')}
              >
                <Statistic
                  title={<span className="text-gray-600">主机总数</span>}
                  value={stats.total}
                  prefix={<DesktopOutlined className="text-primary-500" />}
                  valueStyle={{ color: '#495057', fontSize: '28px', fontWeight: 600 }}
                />
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card
                className="hover:shadow-lg transition-shadow cursor-pointer"
                onClick={() => setStatusFilter('online')}
              >
                <Statistic
                  title={<span className="text-gray-600">在线主机</span>}
                  value={stats.online}
                  prefix={<CheckCircleOutlined className="text-success" />}
                  valueStyle={{ color: '#10b981', fontSize: '28px', fontWeight: 600 }}
                />
                <Progress
                  percent={stats.healthRate}
                  strokeColor="#10b981"
                  showInfo={false}
                  className="mt-2"
                />
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card
                className="hover:shadow-lg transition-shadow cursor-pointer"
                onClick={() => setStatusFilter('maintenance')}
              >
                <Statistic
                  title={<span className="text-gray-600">维护中</span>}
                  value={stats.maintenance}
                  prefix={<ToolOutlined className="text-warning" />}
                  valueStyle={{ color: '#f59e0b', fontSize: '28px', fontWeight: 600 }}
                />
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card
                className="hover:shadow-lg transition-shadow cursor-pointer"
                onClick={() => setStatusFilter('error')}
              >
                <Statistic
                  title={<span className="text-gray-600">错误</span>}
                  value={stats.error}
                  prefix={<ExclamationCircleOutlined className="text-error" />}
                  valueStyle={{ color: '#ef4444', fontSize: '28px', fontWeight: 600 }}
                />
              </Card>
            </StaggerItem>
          </Col>
        </Row>
      </StaggerList>

      {/* 筛选和搜索 */}
      <Card>
        <Space direction="vertical" size="middle" className="w-full">
          <div className="flex flex-wrap gap-3">
            <Input
              placeholder="搜索主机名称、IP 或区域"
              prefix={<SearchOutlined className="text-gray-400" />}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              style={{ width: 280 }}
              allowClear
            />
            <Select
              value={statusFilter}
              style={{ width: 140 }}
              options={[
                { value: 'all', label: '全部状态' },
                { value: 'online', label: '在线' },
                { value: 'offline', label: '离线' },
                { value: 'maintenance', label: '维护中' },
                { value: 'error', label: '错误' },
              ]}
              onChange={setStatusFilter}
            />
            <Select
              value={providerFilter}
              style={{ width: 140 }}
              options={[
                { value: 'all', label: '全部厂商' },
                { value: 'volcengine', label: '火山引擎' },
                { value: 'alicloud', label: '阿里云' },
                { value: 'tencent', label: '腾讯云' },
                { value: 'ucloud', label: 'UCloud' },
                { value: 'aws', label: 'AWS' },
              ]}
              onChange={setProviderFilter}
            />
            <Select
              value={availabilityFilter}
              style={{ width: 140 }}
              options={[
                { value: 'all', label: '全部主机' },
                { value: 'available', label: '可用' },
                { value: 'assigned', label: '已分配' },
              ]}
              onChange={setAvailabilityFilter}
            />
            <Input
              placeholder="区域筛选"
              value={group}
              onChange={(e) => setGroup(e.target.value)}
              style={{ width: 140 }}
              allowClear
            />
          </div>

          {/* 批量操作 */}
          {selected.length > 0 && (
            <div className="flex items-center justify-between p-3 bg-primary-50 rounded-lg border border-primary-200">
              <span className="text-sm text-gray-700">
                已选择 <Badge count={selected.length} showZero className="mx-1" /> 台主机
              </span>
              <Space>
                <Button size="small" onClick={() => setSelected([])}>
                  取消选择
                </Button>
                <Button size="small" onClick={() => batchAction('maintenance')}>
                  批量维护
                </Button>
                <Button size="small" onClick={() => batchAction('online')}>
                  批量上线
                </Button>
                <Button size="small" icon={<CodeOutlined />} onClick={batchExec}>
                  批量 SSH 执行
                </Button>
              </Space>
            </div>
          )}
        </Space>
      </Card>

      {/* 主机列表 - 卡片视图 */}
      {loading ? (
        <Card>
          <div className="text-center py-12">
            <ReloadOutlined spin className="text-4xl text-primary-500 mb-4" />
            <p className="text-gray-500">加载中...</p>
          </div>
        </Card>
      ) : filtered.length === 0 ? (
        <Card>
          <Empty
            description={
              <span className="text-gray-500">
                {search || statusFilter !== 'all' || providerFilter !== 'all' || group
                  ? '没有找到匹配的主机'
                  : '还没有添加任何主机'}
              </span>
            }
          >
            {!search && statusFilter === 'all' && providerFilter === 'all' && !group && (
              <Dropdown
                menu={{
                  items: [
                    { key: 'onboarding', label: 'SSH 接入' },
                    { key: 'cloud', label: '云平台导入' },
                    { key: 'virt', label: 'KVM 虚拟化' },
                  ],
                  onClick: ({ key }) => {
                    if (key === 'onboarding') navigate('/hosts/onboarding');
                    if (key === 'cloud') navigate('/hosts/cloud-import');
                    if (key === 'virt') navigate('/hosts/virtualization');
                  },
                }}
              >
                <Button type="primary" icon={<PlusOutlined />}>
                  添加第一台主机
                </Button>
              </Dropdown>
            )}
          </Empty>
        </Card>
      ) : (
        <StaggerList staggerDelay={0.05}>
          <Row gutter={[16, 16]}>
            {filtered.map((host) => (
              <Col xs={24} sm={12} md={8} lg={6} xl={4} key={host.id}>
                <StaggerItem>
                  <HostCard host={host} />
                </StaggerItem>
              </Col>
            ))}
          </Row>
        </StaggerList>
      )}
    </div>
  );
};

export default HostListPage;
