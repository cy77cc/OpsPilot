import React, { useState, useCallback, useEffect, useMemo } from 'react';
import {
  Button,
  Card,
  Col,
  Input,
  Row,
  Select,
  Space,
  Statistic,
  Tag,
  message,
  Checkbox,
  Dropdown,
  Empty,
  Badge,
  Segmented,
  Table,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  MoreOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ExclamationCircleOutlined,
  FileTextOutlined,
  AppstoreOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Api } from '../../api';
import type { ServiceItem } from '../../api/modules/services';
import { StaggerList, StaggerItem } from '../../components/Motion';

const VIEW_MODE_KEY = 'service-list-view-mode';
const AUTO_SWITCH_THRESHOLD = 8;

const ServiceListPage: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<ServiceItem[]>([]);
  const [query, setQuery] = useState('');
  const [env, setEnv] = useState<string>('all');
  const [runtime, setRuntime] = useState<string>('all');
  const [serviceKind, setServiceKind] = useState<string>(() => searchParams.get('service_kind') || 'all');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [labelSelector, setLabelSelector] = useState('');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);

  // 视图模式：null 表示让自动逻辑决定
  const [viewMode, setViewMode] = useState<'card' | 'list' | null>(() => {
    const saved = localStorage.getItem(VIEW_MODE_KEY);
    if (saved === 'card' || saved === 'list') return saved;
    return null;
  });

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await Api.services.getList({
        page: 1,
        pageSize: 100,
        env: env === 'all' ? undefined : env,
        runtimeType: runtime === 'all' ? undefined : (runtime as any),
        serviceKind: serviceKind === 'all' ? undefined : (serviceKind as any),
        labelSelector,
        q: query,
      });
      setList(res.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载服务失败');
    } finally {
      setLoading(false);
    }
  }, [env, runtime, serviceKind, labelSelector, query]);

  useEffect(() => {
    const fromURL = searchParams.get('service_kind') || 'all';
    setServiceKind(fromURL);
  }, [searchParams]);

  useEffect(() => {
    void load();
  }, [load]);

  // 根据数量自动决定视图模式
  useEffect(() => {
    if (viewMode === null && list.length > 0) {
      setViewMode(list.length > AUTO_SWITCH_THRESHOLD ? 'list' : 'card');
    }
  }, [list.length, viewMode]);

  // 保存用户选择
  const handleViewModeChange = (mode: string) => {
    const newMode = mode as 'card' | 'list';
    setViewMode(newMode);
    localStorage.setItem(VIEW_MODE_KEY, newMode);
  };

  // 统计数据
  const stats = useMemo(() => {
    const running = list.filter((x) => x.status === 'running').length;
    const deploying = list.filter((x) => x.status === 'deploying' || x.status === 'syncing').length;
    const draft = list.filter((x) => x.status === 'draft').length;
    const error = list.filter((x) => x.status === 'error').length;
    return { running, deploying, draft, error, total: list.length };
  }, [list]);

  // 过滤后的列表
  const filteredList = useMemo(() => {
    if (statusFilter === 'all') return list;
    return list.filter((item) => item.status === statusFilter);
  }, [list, statusFilter]);

  // 批量操作
  const handleBatchAction = (action: string) => {
    if (selectedIds.length === 0) {
      message.warning('请先选择服务');
      return;
    }
    message.info(`批量${action}: ${selectedIds.length} 个服务`);
    // TODO: 实现批量操作逻辑
  };

  // 获取状态图标和颜色
  const getStatusConfig = (status: string) => {
    const configs: Record<string, { icon: React.ReactNode; color: string; text: string }> = {
      running: { icon: <CheckCircleOutlined />, color: 'success', text: '运行中' },
      deploying: { icon: <ClockCircleOutlined />, color: 'processing', text: '部署中' },
      syncing: { icon: <ClockCircleOutlined />, color: 'processing', text: '同步中' },
      error: { icon: <ExclamationCircleOutlined />, color: 'error', text: '错误' },
      draft: { icon: <FileTextOutlined />, color: 'default', text: '草稿' },
    };
    return configs[status] || { icon: null, color: 'default', text: status };
  };

  // 服务卡片组件
  const ServiceCard: React.FC<{ service: ServiceItem }> = ({ service }) => {
    const statusConfig = getStatusConfig(service.status);
    const isSelected = selectedIds.includes(String(service.id));

    return (
      <Card
        hoverable
        className="h-full transition-all duration-200 cursor-pointer"
        style={{
          borderColor: isSelected ? '#6366f1' : undefined,
          boxShadow: isSelected ? '0 0 0 2px rgba(99, 102, 241, 0.1)' : undefined,
        }}
        onClick={() => navigate(`/services/${service.id}`)}
      >
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-start gap-3 flex-1">
            <div onClick={(e) => e.stopPropagation()}>
              <Checkbox
                checked={isSelected}
                onChange={(e) => {
                  if (e.target.checked) {
                    setSelectedIds([...selectedIds, String(service.id)]);
                  } else {
                    setSelectedIds(selectedIds.filter((id) => id !== String(service.id)));
                  }
                }}
              />
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-2">
                <span className="text-lg font-semibold text-gray-900">
                  {service.name}
                </span>
                <Tag color={statusConfig.color} icon={statusConfig.icon}>
                  {statusConfig.text}
                </Tag>
              </div>
              <div className="flex flex-wrap gap-2 text-sm text-gray-600">
                <span>环境: <Tag>{service.env}</Tag></span>
                <span>运行时: <Tag color="blue">{service.runtimeType}</Tag></span>
                <span>分类: <Tag color="gold">{service.serviceKind || 'business'}</Tag></span>
                {service.visibility && <span>可见性: <Tag color="purple">{service.visibility}</Tag></span>}
                {service.owner && <span>负责人: {service.owner}</span>}
              </div>
            </div>
          </div>
          <div onClick={(e) => e.stopPropagation()}>
            <Dropdown
              menu={{
                items: [
                  { key: 'edit', icon: <EditOutlined />, label: '编辑配置' },
                  { type: 'divider' },
                  { key: 'start', icon: <PlayCircleOutlined />, label: '启动服务' },
                  { key: 'stop', icon: <PauseCircleOutlined />, label: '停止服务' },
                  { type: 'divider' },
                  { key: 'delete', icon: <DeleteOutlined />, label: '删除服务', danger: true },
                ],
                onClick: ({ key }) => {
                  if (key === 'edit') navigate(`/services/${service.id}?tab=config`);
                  else message.info(`${key}: ${service.name}`);
                },
              }}
            >
              <Button type="text" icon={<MoreOutlined />} />
            </Dropdown>
          </div>
        </div>

        {service.labels && service.labels.length > 0 && (
          <div className="mt-3 pt-3 border-t border-gray-100">
            <div className="text-xs text-gray-500 mb-2">标签</div>
            <Space size={[4, 4]} wrap>
              {service.labels.slice(0, 4).map((l) => (
                <Tag key={`${l.key}:${l.value}`} className="text-xs">
                  {l.key}={l.value}
                </Tag>
              ))}
              {service.labels.length > 4 && (
                <Tag className="text-xs">+{service.labels.length - 4}</Tag>
              )}
            </Space>
          </div>
        )}
        {service.tags && service.tags.length > 0 && (
          <div className="mt-3 pt-3 border-t border-gray-100">
            <div className="text-xs text-gray-500 mb-2">标签</div>
            <Space size={[4, 4]} wrap>
              {service.tags.map((tag) => (
                <Tag key={tag} color="blue">{tag}</Tag>
              ))}
            </Space>
          </div>
        )}
      </Card>
    );
  };

  // 列表视图列定义
  const columns: ColumnsType<ServiceItem> = [
    {
      title: '服务名',
      dataIndex: 'name',
      key: 'name',
      sorter: (a: ServiceItem, b: ServiceItem) => a.name.localeCompare(b.name, 'zh-CN'),
      render: (text: string, record: ServiceItem) => (
        <a onClick={() => navigate(`/services/${record.id}`)} className="font-medium">
          {text}
        </a>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      sorter: (a: ServiceItem, b: ServiceItem) => a.status.localeCompare(b.status),
      render: (status: string) => {
        const config = getStatusConfig(status);
        return <Tag color={config.color} icon={config.icon}>{config.text}</Tag>;
      },
    },
    {
      title: '环境',
      dataIndex: 'env',
      key: 'env',
      width: 100,
      render: (env: string) => <Tag>{env}</Tag>,
    },
    {
      title: '运行时',
      dataIndex: 'runtimeType',
      key: 'runtimeType',
      width: 100,
      render: (runtime: string) => <Tag color="blue">{runtime}</Tag>,
    },
    {
      title: '分类',
      dataIndex: 'serviceKind',
      key: 'serviceKind',
      width: 110,
      render: (kind: string) => <Tag color="gold">{kind || 'business'}</Tag>,
    },
    {
      title: '可见性',
      dataIndex: 'visibility',
      key: 'visibility',
      width: 120,
      render: (v: string) => <Tag color="purple">{v || 'team'}</Tag>,
    },
    {
      title: '负责人',
      dataIndex: 'owner',
      key: 'owner',
      width: 100,
      render: (owner: string) => owner || '-',
    },
    {
      title: '标签',
      dataIndex: 'labels',
      key: 'labels',
      render: (labels: ServiceItem['labels']) => {
        if (!labels || labels.length === 0) return '-';
        return (
          <Space size={[4, 4]} wrap>
            {labels.slice(0, 3).map((l) => (
              <Tag key={`${l.key}:${l.value}`} className="text-xs">
                {l.key}={l.value}
              </Tag>
            ))}
            {labels.length > 3 && <Tag className="text-xs">+{labels.length - 3}</Tag>}
          </Space>
        );
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_: unknown, record: ServiceItem) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => navigate(`/services/${record.id}?tab=config`)}>
            编辑
          </Button>
          <Button type="link" size="small" icon={<PlayCircleOutlined />} onClick={() => message.info(`启动: ${record.name}`)}>
            启动
          </Button>
          <Button type="link" size="small" icon={<PauseCircleOutlined />} onClick={() => message.info(`停止: ${record.name}`)}>
            停止
          </Button>
          <Button type="link" size="small" danger icon={<DeleteOutlined />} onClick={() => message.info(`删除: ${record.name}`)}>
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* 页面头部 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">服务管理</h1>
          <p className="text-sm text-gray-500 mt-1">管理和监控所有服务实例</p>
        </div>
        <Space>
          <Segmented
            value={viewMode || 'card'}
            onChange={handleViewModeChange}
            options={[
              { value: 'card', icon: <AppstoreOutlined />, label: '卡片' },
              { value: 'list', icon: <UnorderedListOutlined />, label: '列表' },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/services/provision')}>
            创建服务
          </Button>
        </Space>
      </div>

      {/* 统计卡片 */}
      <StaggerList staggerDelay={0.05}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card className="hover:shadow-lg transition-shadow cursor-pointer" onClick={() => setStatusFilter('all')}>
                <Statistic
                  title={<span className="text-gray-600">服务总数</span>}
                  value={stats.total}
                  styles={{ content: { color: '#495057', fontSize: '28px', fontWeight: 600 } }}
                />
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card className="hover:shadow-lg transition-shadow cursor-pointer" onClick={() => setStatusFilter('running')}>
                <Statistic
                  title={<span className="text-gray-600">运行中</span>}
                  value={stats.running}
                  prefix={<CheckCircleOutlined className="text-success" />}
                  styles={{ content: { color: '#10b981', fontSize: '28px', fontWeight: 600 } }}
                />
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card className="hover:shadow-lg transition-shadow cursor-pointer" onClick={() => setStatusFilter('deploying')}>
                <Statistic
                  title={<span className="text-gray-600">部署中</span>}
                  value={stats.deploying}
                  prefix={<ClockCircleOutlined className="text-primary-500" />}
                  styles={{ content: { color: '#6366f1', fontSize: '28px', fontWeight: 600 } }}
                />
              </Card>
            </StaggerItem>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StaggerItem>
              <Card className="hover:shadow-lg transition-shadow cursor-pointer" onClick={() => setStatusFilter('error')}>
                <Statistic
                  title={<span className="text-gray-600">错误</span>}
                  value={stats.error}
                  prefix={<ExclamationCircleOutlined className="text-error" />}
                  styles={{ content: { color: '#ef4444', fontSize: '28px', fontWeight: 600 } }}
                />
              </Card>
            </StaggerItem>
          </Col>
        </Row>
      </StaggerList>

      {/* 筛选和搜索 */}
      <Card>
        <Space orientation="vertical" size="middle" className="w-full">
          <div className="flex flex-wrap gap-3">
            <Input
              placeholder="搜索服务名称或负责人"
              prefix={<SearchOutlined className="text-gray-400" />}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              style={{ width: 240 }}
              allowClear
            />
            <Select
              value={env}
              style={{ width: 140 }}
              options={[
                { value: 'all', label: '全部环境' },
                { value: 'development', label: 'Development' },
                { value: 'staging', label: 'Staging' },
                { value: 'production', label: 'Production' },
              ]}
              onChange={setEnv}
            />
            <Select
              value={runtime}
              style={{ width: 140 }}
              options={[
                { value: 'all', label: '全部运行时' },
                { value: 'k8s', label: 'Kubernetes' },
                { value: 'compose', label: 'Compose' },
                { value: 'helm', label: 'Helm' },
              ]}
              onChange={setRuntime}
            />
            <Select
              value={serviceKind}
              style={{ width: 140 }}
              options={[
                { value: 'all', label: '全部分类' },
                { value: 'middleware', label: '中间件' },
                { value: 'business', label: '业务服务' },
              ]}
              onChange={setServiceKind}
            />
            <Select
              value={statusFilter}
              style={{ width: 120 }}
              options={[
                { value: 'all', label: '全部状态' },
                { value: 'running', label: '运行中' },
                { value: 'deploying', label: '部署中' },
                { value: 'error', label: '错误' },
                { value: 'draft', label: '草稿' },
              ]}
              onChange={setStatusFilter}
            />
            <Input
              placeholder="标签选择器 (app=user)"
              value={labelSelector}
              onChange={(e) => setLabelSelector(e.target.value)}
              style={{ width: 200 }}
              allowClear
            />
          </div>

          {/* 批量操作 */}
          {selectedIds.length > 0 && (
            <div className="flex items-center justify-between p-3 bg-primary-50 rounded-lg border border-primary-200">
              <span className="text-sm text-gray-700">
                已选择 <Badge count={selectedIds.length} showZero className="mx-1" /> 个服务
              </span>
              <Space>
                <Button size="small" onClick={() => setSelectedIds([])}>
                  取消选择
                </Button>
                <Button size="small" icon={<PlayCircleOutlined />} onClick={() => handleBatchAction('启动')}>
                  批量启动
                </Button>
                <Button size="small" icon={<PauseCircleOutlined />} onClick={() => handleBatchAction('停止')}>
                  批量停止
                </Button>
                <Button size="small" danger icon={<DeleteOutlined />} onClick={() => handleBatchAction('删除')}>
                  批量删除
                </Button>
              </Space>
            </div>
          )}
        </Space>
      </Card>

      {/* 服务列表 */}
      {loading ? (
        <Card>
          <div className="text-center py-12">
            <ReloadOutlined spin className="text-4xl text-primary-500 mb-4" />
            <p className="text-gray-500">加载中...</p>
          </div>
        </Card>
      ) : filteredList.length === 0 ? (
        <Card>
          <Empty
            description={
              <span className="text-gray-500">
                {query || env !== 'all' || runtime !== 'all' || statusFilter !== 'all'
                  ? '没有找到匹配的服务'
                  : '还没有创建任何服务'}
              </span>
            }
          >
            {!query && env === 'all' && runtime === 'all' && statusFilter === 'all' && (
              <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/services/provision')}>
                创建第一个服务
              </Button>
            )}
          </Empty>
        </Card>
      ) : viewMode === 'list' ? (
        <Card>
          <Table
            dataSource={filteredList}
            columns={columns}
            rowKey="id"
            pagination={{ pageSize: 10, showSizeChanger: true, showQuickJumper: true, showTotal: (total) => `共 ${total} 条记录` }}
          />
        </Card>
      ) : (
        <StaggerList staggerDelay={0.05}>
          <Row gutter={[16, 16]}>
            {filteredList.map((service) => (
              <Col xs={24} sm={12} lg={8} xl={6} key={service.id}>
                <StaggerItem>
                  <ServiceCard service={service} />
                </StaggerItem>
              </Col>
            ))}
          </Row>
        </StaggerList>
      )}
    </div>
  );
};

export default ServiceListPage;
