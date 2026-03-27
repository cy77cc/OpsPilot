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
import { StaggerList, StaggerItem } from '../../components/Motion';

const HostListPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [hosts, setHosts] = useState<Host[]>([]);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [availabilityFilter, setAvailabilityFilter] = useState<'all' | 'available' | 'assigned'>('all');
  const [selected, setSelected] = useState<string[]>([]);
  const [group, setGroup] = useState('');
  const [hostAssignments, setHostAssignments] = useState<Record<string, { clusters: string[]; targets: string[] }>>({});

  const load = async () => {
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

  useEffect(() => {
    load();
    const handler = () => load();
    window.addEventListener('project:changed', handler as EventListener);
    return () => window.removeEventListener('project:changed', handler as EventListener);
  }, [statusFilter, group]);

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

        // Availability filter
        const assignments = hostAssignments[h.id];
        const isAssigned = assignments && (assignments.clusters.length > 0 || assignments.targets.length > 0);
        const hitAvailability =
          availabilityFilter === 'all' ||
          (availabilityFilter === 'assigned' && isAssigned) ||
          (availabilityFilter === 'available' && !isAssigned);

        return hitSearch && hitStatus && hitAvailability;
      }),
    [hosts, search, statusFilter, availabilityFilter, hostAssignments]
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

    return (
      <Card
        hoverable
        className="h-full transition-all duration-200"
        style={{
          borderColor: isSelected ? '#6366f1' : undefined,
          boxShadow: isSelected ? '0 0 0 2px rgba(99, 102, 241, 0.1)' : undefined,
        }}
      >
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-start gap-3 flex-1">
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
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-2">
                <a
                  onClick={() => navigate(`/deployment/infrastructure/hosts/${host.id}`)}
                  className="text-lg font-semibold text-gray-900 hover:text-primary-600"
                >
                  {host.name}
                </a>
                <Tag color={statusConfig.color} icon={statusConfig.icon}>
                  {statusConfig.text}
                </Tag>
                <Tag color={host.healthState === 'healthy' ? 'green' : host.healthState === 'degraded' ? 'orange' : host.healthState === 'critical' ? 'red' : 'default'}>
                  健康: {host.healthState || 'unknown'}
                </Tag>
                {isAssigned && (
                  <Tag color="blue">已分配</Tag>
                )}
              </div>
              <div className="flex flex-wrap gap-2 text-sm text-gray-600">
                <span>IP: {host.ip}</span>
                {host.region && <span>区域: {host.region}</span>}
              </div>

              {/* Assignment information */}
              {isAssigned && (
                <div className="mt-2 pt-2 border-t border-gray-200">
                  <div className="text-xs text-gray-500 space-y-1">
                    {assignments.clusters.length > 0 && (
                      <div>集群: {assignments.clusters.join(', ')}</div>
                    )}
                    {assignments.targets.length > 0 && (
                      <div>部署目标: {assignments.targets.join(', ')}</div>
                    )}
                  </div>
                </div>
              )}

              {/* Resource capacity */}
              <div className="mt-2 pt-2 border-t border-gray-200">
                <div className="text-xs text-gray-500 space-y-1">
                  <div>容量: {host.cpu || 0} 核 CPU / {host.memory || 0} MB 内存 / {host.disk || 0} GB 磁盘</div>
                  {host.maintenanceReason ? <div>维护原因: {host.maintenanceReason}</div> : null}
                  {host.maintenanceUntil ? <div>维护截止: {new Date(host.maintenanceUntil).toLocaleString()}</div> : null}
                </div>
              </div>
            </div>
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
            <Button type="text" icon={<MoreOutlined />} />
          </Dropdown>
        </div>

        {/* 资源使用情况 */}
        <div className="space-y-3">
          <div>
            <div className="flex justify-between text-sm mb-1">
              <span className="text-gray-600">CPU</span>
              <span className="text-gray-900 font-medium">{host.cpu || 0}%</span>
            </div>
            <Progress
              percent={Math.min(100, host.cpu || 0)}
              strokeColor={
                (host.cpu || 0) > 80 ? '#ef4444' : (host.cpu || 0) > 60 ? '#f59e0b' : '#10b981'
              }
              showInfo={false}
              size="small"
            />
          </div>

          <div>
            <div className="flex justify-between text-sm mb-1">
              <span className="text-gray-600">内存</span>
              <span className="text-gray-900 font-medium">{host.memory || 0} MB</span>
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
              <span className="text-gray-900 font-medium">{host.disk || 0} GB</span>
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

  return (
    <div className="space-y-6">
      {/* 页面头部 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">主机管理</h1>
          <p className="text-sm text-gray-500 mt-1">管理和监控所有主机资源</p>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
          <Dropdown
            menu={{
              items: [
                { key: 'onboarding', label: 'SSH 接入（密码/密钥）' },
                { key: 'cloud', label: '云平台导入（阿里云/腾讯云）' },
                { key: 'virt', label: 'KVM 虚拟化创建' },
                { key: 'keys', label: 'SSH 密钥管理' },
              ],
              onClick: ({ key }) => {
                if (key === 'onboarding') navigate('/hosts/onboarding');
                if (key === 'cloud') navigate('/hosts/cloud-import');
                if (key === 'virt') navigate('/hosts/virtualization');
                if (key === 'keys') navigate('/hosts/keys');
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
                {search || statusFilter !== 'all' || group
                  ? '没有找到匹配的主机'
                  : '还没有添加任何主机'}
              </span>
            }
          >
            {!search && statusFilter === 'all' && !group && (
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
              <Col xs={24} sm={12} lg={8} xl={6} key={host.id}>
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
