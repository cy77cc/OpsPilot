import React, { useState, useCallback, useEffect } from 'react';
import {
  Card, Tabs, Table, Tag, Button, Space, Descriptions, Spin, message,
  Modal, Form, Input, Popconfirm, Drawer, Badge, Tooltip, Typography,
  Select
} from 'antd';
import {
  ArrowLeftOutlined, ReloadOutlined, ClusterOutlined,
  DeleteOutlined, EditOutlined, ApiOutlined,
  PlusOutlined, SyncOutlined, NodeIndexOutlined, InfoCircleOutlined,
  AppstoreOutlined, CloudServerOutlined, SettingOutlined,
  DatabaseOutlined, CloudOutlined, ToolOutlined
} from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { Api } from '../../../api';
import type {
  Cluster, ClusterNode, NamespaceInfo, DeploymentInfo,
  StatefulSetInfo, DaemonSetInfo, PodInfo, ServiceInfo,
  ConfigMapInfo, SecretInfo, PVCInfo, PVInfo, ClusterServiceInfo,
  EventInfo, HPAInfo, ResourceQuotaInfo, LimitRangeInfo,
  ClusterVersionInfo, CertificateInfo, ClusterUpgradePlan
} from '../../../api/modules/cluster';

const { Text, Title } = Typography;

const ClusterDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const clusterId = Number(id);

  // Basic state
  const [loading, setLoading] = useState(false);
  const [cluster, setCluster] = useState<Cluster | null>(null);
  const [nodes, setNodes] = useState<ClusterNode[]>([]);
  const [nodesLoading, setNodesLoading] = useState(false);

  // Resource state
  const [namespaces, setNamespaces] = useState<NamespaceInfo[]>([]);
  const [selectedNamespace, setSelectedNamespace] = useState<string>('default');
  const [deployments, setDeployments] = useState<DeploymentInfo[]>([]);
  const [statefulsets, setStatefulsets] = useState<StatefulSetInfo[]>([]);
  const [daemonsets, setDaemonsets] = useState<DaemonSetInfo[]>([]);
  const [pods, setPods] = useState<PodInfo[]>([]);
  const [services, setServices] = useState<ServiceInfo[]>([]);
  const [configmaps, setConfigmaps] = useState<ConfigMapInfo[]>([]);
  const [secrets, setSecrets] = useState<SecretInfo[]>([]);
  const [pvcs, setPvcs] = useState<PVCInfo[]>([]);
  const [pvs, setPvs] = useState<PVInfo[]>([]);
  const [clusterServices, setClusterServices] = useState<ClusterServiceInfo[]>([]);
  const [resourceLoading, setResourceLoading] = useState(false);

  // Advanced operations state
  const [events, setEvents] = useState<EventInfo[]>([]);
  const [hpas, setHPAs] = useState<HPAInfo[]>([]);
  const [resourceQuotas, setResourceQuotas] = useState<ResourceQuotaInfo[]>([]);
  const [limitRanges, setLimitRanges] = useState<LimitRangeInfo[]>([]);
  const [clusterVersion, setClusterVersion] = useState<ClusterVersionInfo | null>(null);
  const [certificates, setCertificates] = useState<CertificateInfo[]>([]);
  const [upgradePlan, setUpgradePlan] = useState<ClusterUpgradePlan | null>(null);
  const [advancedLoading, setAdvancedLoading] = useState(false);

  // Modals
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [addNodeModalVisible, setAddNodeModalVisible] = useState(false);
  const [nodeDrawerVisible, setNodeDrawerVisible] = useState(false);
  const [selectedNode, setSelectedNode] = useState<ClusterNode | null>(null);
  const [editForm] = Form.useForm();
  const [addNodeForm] = Form.useForm();

  const loadCluster = useCallback(async () => {
    if (!clusterId) return;
    setLoading(true);
    try {
      const res = await Api.cluster.getClusterDetail(clusterId);
      setCluster(res.data);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载集群信息失败');
    } finally {
      setLoading(false);
    }
  }, [clusterId]);

  const loadNodes = useCallback(async () => {
    if (!clusterId) return;
    setNodesLoading(true);
    try {
      const res = await Api.cluster.getClusterNodes(clusterId);
      setNodes(res.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载节点列表失败');
    } finally {
      setNodesLoading(false);
    }
  }, [clusterId]);

  const loadNamespaces = useCallback(async () => {
    if (!clusterId) return;
    try {
      const res = await Api.cluster.getNamespaces(clusterId);
      setNamespaces(res.data.list || []);
    } catch (err) {
      console.error('Failed to load namespaces:', err);
    }
  }, [clusterId]);

  const loadResources = useCallback(async (namespace: string) => {
    if (!clusterId) return;
    setResourceLoading(true);
    try {
      const [depRes, stsRes, dsRes, podRes, svcRes, cmRes, secRes, pvcRes] = await Promise.all([
        Api.cluster.getDeployments(clusterId, namespace),
        Api.cluster.getStatefulSets(clusterId, namespace),
        Api.cluster.getDaemonSets(clusterId, namespace),
        Api.cluster.getPods(clusterId, namespace),
        Api.cluster.getServices(clusterId, namespace),
        Api.cluster.getConfigMaps(clusterId, namespace),
        Api.cluster.getSecrets(clusterId, namespace),
        Api.cluster.getPVCs(clusterId, namespace),
      ]);
      setDeployments(depRes.data.list || []);
      setStatefulsets(stsRes.data.list || []);
      setDaemonsets(dsRes.data.list || []);
      setPods(podRes.data.list || []);
      setServices(svcRes.data.list || []);
      setConfigmaps(cmRes.data.list || []);
      setSecrets(secRes.data.list || []);
      setPvcs(pvcRes.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载资源失败');
    } finally {
      setResourceLoading(false);
    }
  }, [clusterId]);

  const loadPVs = useCallback(async () => {
    if (!clusterId) return;
    try {
      const res = await Api.cluster.getPVs(clusterId);
      setPvs(res.data.list || []);
    } catch (err) {
      console.error('Failed to load PVs:', err);
    }
  }, [clusterId]);

  const loadClusterServices = useCallback(async () => {
    if (!clusterId) return;
    try {
      const res = await Api.cluster.getClusterServices(clusterId);
      setClusterServices(res.data.list || []);
    } catch (err) {
      console.error('Failed to load cluster services:', err);
    }
  }, [clusterId]);

  const loadEvents = useCallback(async () => {
    if (!clusterId) return;
    try {
      const res = await Api.cluster.getEvents(clusterId);
      setEvents(res.data.list || []);
    } catch (err) {
      console.error('Failed to load events:', err);
    }
  }, [clusterId]);

  const loadAdvancedResources = useCallback(async (namespace: string) => {
    if (!clusterId) return;
    setAdvancedLoading(true);
    try {
      const [hpaRes, quotaRes, limitRes] = await Promise.all([
        Api.cluster.getHPAs(clusterId, namespace),
        Api.cluster.getResourceQuotas(clusterId, namespace),
        Api.cluster.getLimitRanges(clusterId, namespace),
      ]);
      setHPAs(hpaRes.data.list || []);
      setResourceQuotas(quotaRes.data.list || []);
      setLimitRanges(limitRes.data.list || []);
    } catch (err) {
      console.error('Failed to load advanced resources:', err);
    } finally {
      setAdvancedLoading(false);
    }
  }, [clusterId]);

  const loadClusterInfo = useCallback(async () => {
    if (!clusterId) return;
    try {
      const [versionRes, certRes, planRes] = await Promise.all([
        Api.cluster.getClusterVersion(clusterId),
        Api.cluster.getCertificates(clusterId),
        Api.cluster.getUpgradePlan(clusterId),
      ]);
      setClusterVersion(versionRes.data);
      setCertificates(certRes.data.list || []);
      setUpgradePlan(planRes.data);
    } catch (err) {
      console.error('Failed to load cluster info:', err);
    }
  }, [clusterId]);

  useEffect(() => {
    loadCluster();
    loadNodes();
    loadNamespaces();
    loadPVs();
    loadClusterServices();
    loadEvents();
    loadClusterInfo();
  }, [loadCluster, loadNodes, loadNamespaces, loadPVs, loadClusterServices, loadEvents, loadClusterInfo]);

  useEffect(() => {
    if (selectedNamespace) {
      loadResources(selectedNamespace);
      loadAdvancedResources(selectedNamespace);
    }
  }, [selectedNamespace, loadResources, loadAdvancedResources]);

  const handleTestConnection = async () => {
    if (!clusterId) return;
    try {
      const res = await Api.cluster.testCluster(clusterId);
      if (res.data.connected) {
        message.success(`连接成功 (${res.data.latency_ms}ms)，K8s 版本: ${res.data.version}`);
      } else {
        message.error(`连接失败: ${res.data.message}`);
      }
    } catch (err) {
      message.error(err instanceof Error ? err.message : '测试连接失败');
    }
  };

  const handleSyncNodes = async () => {
    if (!clusterId) return;
    try {
      const res = await Api.cluster.syncClusterNodes(clusterId);
      setNodes(res.data.list || []);
      message.success('节点信息已同步');
    } catch (err) {
      message.error(err instanceof Error ? err.message : '同步失败');
    }
  };

  const handleEdit = async (values: { name: string; description: string }) => {
    if (!clusterId) return;
    try {
      await Api.cluster.updateCluster(clusterId, values);
      message.success('更新成功');
      setEditModalVisible(false);
      loadCluster();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '更新失败');
    }
  };

  const handleDelete = async () => {
    if (!clusterId) return;
    try {
      await Api.cluster.deleteCluster(clusterId);
      message.success('集群已删除');
      navigate('/deployment/infrastructure/clusters');
    } catch (err) {
      message.error(err instanceof Error ? err.message : '删除失败');
    }
  };

  const handleAddNodes = async (values: { hostIds: string; role: string }) => {
    if (!clusterId) return;
    const hostIds = values.hostIds.split(',').map(s => Number(s.trim())).filter(n => !isNaN(n));
    if (hostIds.length === 0) {
      message.error('请输入有效的 Host ID');
      return;
    }
    try {
      const res = await Api.cluster.addClusterNodes(clusterId, { host_ids: hostIds, role: values.role });
      message.success(res.data.message);
      setAddNodeModalVisible(false);
      addNodeForm.resetFields();
      loadNodes();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '添加节点失败');
    }
  };

  const handleRemoveNode = async (nodeName: string) => {
    if (!clusterId) return;
    try {
      await Api.cluster.removeClusterNode(clusterId, nodeName);
      message.success('节点已移除');
      loadNodes();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '移除节点失败');
    }
  };

  const getStatusColor = (status: string) => {
    const statusMap: Record<string, string> = {
      active: 'success', inactive: 'default', error: 'error', provisioning: 'processing',
    };
    return statusMap[status] || 'default';
  };

  const getNodeStatusBadge = (status: string) => {
    if (status === 'ready') return <Badge status="success" text="Ready" />;
    if (status === 'notready') return <Badge status="error" text="NotReady" />;
    return <Badge status="warning" text="Unknown" />;
  };

  // Table columns
  const nodeColumns = [
    { title: '名称', dataIndex: 'name', key: 'name', render: (name: string, record: ClusterNode) => <a onClick={() => { setSelectedNode(record); setNodeDrawerVisible(true); }}>{name}</a> },
    { title: 'IP', dataIndex: 'ip', key: 'ip' },
    { title: '角色', dataIndex: 'role', key: 'role', render: (role: string) => <Tag color={role === 'control-plane' ? 'blue' : 'green'}>{role}</Tag> },
    { title: '状态', dataIndex: 'status', key: 'status', render: (status: string) => getNodeStatusBadge(status) },
    { title: 'Kubelet', dataIndex: 'kubelet_version', key: 'kubelet_version' },
    { title: '容器运行时', dataIndex: 'container_runtime', key: 'container_runtime', render: (r: string) => r?.split('/')[0] || '-' },
    { title: 'CPU/内存', key: 'resources', render: (_: any, r: ClusterNode) => <span>{r.allocatable_cpu || '-'} / {r.allocatable_mem || '-'}</span> },
    {
      title: '操作', key: 'actions', width: 100,
      render: (_: any, record: ClusterNode) => (
        <Space>
          <Tooltip title="查看详情"><Button type="link" size="small" icon={<InfoCircleOutlined />} onClick={() => { setSelectedNode(record); setNodeDrawerVisible(true); }} /></Tooltip>
          {cluster?.source === 'platform_managed' && record.role !== 'control-plane' && (
            <Popconfirm title="确定移除此节点？" onConfirm={() => handleRemoveNode(record.name)} okText="确定" cancelText="取消">
              <Button type="link" size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  const workloadColumns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: 'Ready', key: 'ready', render: (_: any, r: DeploymentInfo) => `${r.ready}/${r.replicas}` },
    { title: 'Age', dataIndex: 'age', key: 'age' },
  ];

  const podColumns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '状态', dataIndex: 'status', key: 'status', render: (s: string) => <Tag color={s === 'Running' ? 'green' : 'blue'}>{s}</Tag> },
    { title: 'Ready', dataIndex: 'ready', key: 'ready' },
    { title: '节点', dataIndex: 'node_name', key: 'node_name' },
    { title: 'IP', dataIndex: 'pod_ip', key: 'pod_ip' },
    { title: 'Age', dataIndex: 'age', key: 'age' },
  ];

  const serviceColumns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '类型', dataIndex: 'type', key: 'type' },
    { title: 'ClusterIP', dataIndex: 'cluster_ip', key: 'cluster_ip' },
    { title: '端口', key: 'ports', render: (_: any, r: ServiceInfo) => r.ports?.map(p => `${p.port}:${p.target_port}`).join(', ') || '-' },
    { title: 'Age', dataIndex: 'age', key: 'age' },
  ];

  const configColumns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: 'Data Keys', key: 'keys', render: (_: any, r: ConfigMapInfo | SecretInfo) => r.data_keys?.length || 0 },
    { title: 'Age', dataIndex: 'age', key: 'age' },
  ];

  const storageColumns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '状态', dataIndex: 'status', key: 'status' },
    { title: '容量', dataIndex: 'capacity', key: 'capacity' },
    { title: '访问模式', dataIndex: 'access_modes', key: 'access_modes' },
    { title: 'StorageClass', dataIndex: 'storage_class', key: 'storage_class' },
    { title: 'Age', dataIndex: 'age', key: 'age' },
  ];

  const clusterServiceColumns = [
    { title: '服务名称', dataIndex: 'name', key: 'name' },
    { title: '项目', dataIndex: 'project_name', key: 'project_name' },
    { title: '环境', dataIndex: 'env', key: 'env', render: (e: string) => <Tag color="blue">{e}</Tag> },
    { title: '状态', dataIndex: 'status', key: 'status' },
    { title: '最后部署', dataIndex: 'last_deploy_at', key: 'last_deploy_at' },
  ];

  if (loading) return <div className="flex items-center justify-center h-64"><Spin size="large" /></div>;
  if (!cluster) return <div className="text-center py-16"><ClusterOutlined className="text-6xl text-gray-300 mb-4" /><p className="text-gray-500">集群不存在</p><Button onClick={() => navigate('/deployment/infrastructure/clusters')}>返回列表</Button></div>;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/deployment/infrastructure/clusters')}>返回</Button>
          <div className="flex items-center gap-3">
            <ClusterOutlined className="text-2xl text-blue-500" />
            <div>
              <Title level={4} className="m-0">{cluster.name}</Title>
              <Space className="mt-1">
                <Tag color={getStatusColor(cluster.status)}>{cluster.status}</Tag>
                <Tag color={cluster.source === 'platform_managed' ? 'blue' : 'purple'}>{cluster.source === 'platform_managed' ? '平台托管' : '外部导入'}</Tag>
              </Space>
            </div>
          </div>
        </div>
        <Space>
          <Button icon={<SyncOutlined />} onClick={handleSyncNodes} loading={nodesLoading}>同步节点</Button>
          <Button icon={<ApiOutlined />} onClick={handleTestConnection}>测试连接</Button>
          <Button icon={<EditOutlined />} onClick={() => { editForm.setFieldsValue({ name: cluster.name, description: cluster.description }); setEditModalVisible(true); }}>编辑</Button>
          <Popconfirm title="确定删除此集群？" onConfirm={handleDelete} okText="确定" cancelText="取消">
            <Button danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      </div>

      {/* Overview */}
      <Card title="基本信息">
        <Descriptions column={3}>
          <Descriptions.Item label="集群名称">{cluster.name}</Descriptions.Item>
          <Descriptions.Item label="K8s 版本">{cluster.k8s_version || cluster.version || '-'}</Descriptions.Item>
          <Descriptions.Item label="节点数量">{cluster.node_count}</Descriptions.Item>
          <Descriptions.Item label="API 地址">{cluster.endpoint || '-'}</Descriptions.Item>
          <Descriptions.Item label="Pod CIDR">{cluster.pod_cidr || '-'}</Descriptions.Item>
          <Descriptions.Item label="Service CIDR">{cluster.service_cidr || '-'}</Descriptions.Item>
          <Descriptions.Item label="描述" span={3}>{cluster.description || '-'}</Descriptions.Item>
        </Descriptions>
      </Card>

      {/* Tabs */}
      <Tabs defaultActiveKey="nodes" items={[
        {
          key: 'nodes',
          label: <span><NodeIndexOutlined /> 节点 ({nodes.length})</span>,
          children: (
            <Card title="节点列表" extra={cluster.source === 'platform_managed' && <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddNodeModalVisible(true)}>添加节点</Button>}>
              <Table columns={nodeColumns} dataSource={nodes} rowKey="id" loading={nodesLoading} pagination={false} />
            </Card>
          ),
        },
        {
          key: 'workloads',
          label: <span><AppstoreOutlined /> 工作负载</span>,
          children: (
            <div className="space-y-4">
              <Select style={{ width: 200 }} value={selectedNamespace} onChange={setSelectedNamespace} options={namespaces.map(ns => ({ label: ns.name, value: ns.name }))} loading={resourceLoading} />
              <Spin spinning={resourceLoading}>
                <Card title="Deployments" size="small" className="mb-4">
                  <Table columns={workloadColumns} dataSource={deployments} rowKey="name" pagination={false} size="small" />
                </Card>
                <Card title="StatefulSets" size="small" className="mb-4">
                  <Table columns={workloadColumns} dataSource={statefulsets} rowKey="name" pagination={false} size="small" />
                </Card>
                <Card title="DaemonSets" size="small" className="mb-4">
                  <Table columns={[{ title: '名称', dataIndex: 'name' }, { title: 'Desired', dataIndex: 'desired' }, { title: 'Ready', dataIndex: 'ready' }, { title: 'Age', dataIndex: 'age' }]} dataSource={daemonsets} rowKey="name" pagination={false} size="small" />
                </Card>
                <Card title="Pods" size="small">
                  <Table columns={podColumns} dataSource={pods} rowKey="name" pagination={false} size="small" />
                </Card>
              </Spin>
            </div>
          ),
        },
        {
          key: 'services',
          label: <span><CloudServerOutlined /> 服务</span>,
          children: (
            <div className="space-y-4">
              <Select style={{ width: 200 }} value={selectedNamespace} onChange={setSelectedNamespace} options={namespaces.map(ns => ({ label: ns.name, value: ns.name }))} />
              <Spin spinning={resourceLoading}>
                <Card title="Services" size="small" className="mb-4">
                  <Table columns={serviceColumns} dataSource={services} rowKey="name" pagination={false} size="small" />
                </Card>
                <Card title="Ingresses" size="small">
                  <Table columns={[{ title: '名称', dataIndex: 'name' }, { title: 'Hosts', key: 'hosts', render: (_: any, r: any) => r.hosts?.map((h: any) => h.host).join(', ') || '-' }, { title: 'Age', dataIndex: 'age' }]} dataSource={[]} rowKey="name" pagination={false} size="small" />
                </Card>
              </Spin>
            </div>
          ),
        },
        {
          key: 'config',
          label: <span><SettingOutlined /> 配置</span>,
          children: (
            <div className="space-y-4">
              <Select style={{ width: 200 }} value={selectedNamespace} onChange={setSelectedNamespace} options={namespaces.map(ns => ({ label: ns.name, value: ns.name }))} />
              <Spin spinning={resourceLoading}>
                <Card title="ConfigMaps" size="small" className="mb-4">
                  <Table columns={configColumns} dataSource={configmaps} rowKey="name" pagination={false} size="small" />
                </Card>
                <Card title="Secrets" size="small">
                  <Table columns={[...configColumns, { title: '类型', dataIndex: 'type', key: 'type' }]} dataSource={secrets} rowKey="name" pagination={false} size="small" />
                </Card>
              </Spin>
            </div>
          ),
        },
        {
          key: 'storage',
          label: <span><DatabaseOutlined /> 存储</span>,
          children: (
            <div className="space-y-4">
              <Select style={{ width: 200 }} value={selectedNamespace} onChange={setSelectedNamespace} options={namespaces.map(ns => ({ label: ns.name, value: ns.name }))} />
              <Spin spinning={resourceLoading}>
                <Card title="PersistentVolumes" size="small" className="mb-4">
                  <Table columns={[...storageColumns, { title: 'Claim', dataIndex: 'claim_ref', key: 'claim_ref' }]} dataSource={pvs} rowKey="name" pagination={false} size="small" />
                </Card>
                <Card title="PersistentVolumeClaims" size="small">
                  <Table columns={[...storageColumns, { title: 'Volume', dataIndex: 'volume_name', key: 'volume_name' }]} dataSource={pvcs} rowKey="name" pagination={false} size="small" />
                </Card>
              </Spin>
            </div>
          ),
        },
        {
          key: 'deployed-services',
          label: <span><CloudOutlined /> 部署的服务</span>,
          children: (
            <Card title="该集群部署的服务">
              <Table columns={clusterServiceColumns} dataSource={clusterServices} rowKey="id" pagination={false} />
            </Card>
          ),
        },
        {
          key: 'policy',
          label: <span><SettingOutlined /> 策略</span>,
          children: (
            <div className="space-y-4">
              <Select style={{ width: 200 }} value={selectedNamespace} onChange={setSelectedNamespace} options={namespaces.map(ns => ({ label: ns.name, value: ns.name }))} />
              <Spin spinning={advancedLoading}>
                <Card title="HPA (Horizontal Pod Autoscaler)" size="small" className="mb-4">
                  <Table
                    columns={[
                      { title: '名称', dataIndex: 'name', key: 'name' },
                      { title: '引用', dataIndex: 'reference', key: 'reference' },
                      { title: '副本', key: 'replicas', render: (_: any, r: HPAInfo) => `${r.replicas} (${r.min_replicas}-${r.max_replicas})` },
                      { title: 'CPU', key: 'cpu', render: (_: any, r: HPAInfo) => r.current_cpu || '-' },
                      { title: '内存', key: 'mem', render: (_: any, r: HPAInfo) => r.current_mem || '-' },
                      { title: 'Age', dataIndex: 'age', key: 'age' },
                    ]}
                    dataSource={hpas} rowKey="name" pagination={false} size="small"
                  />
                </Card>
                <Card title="ResourceQuota" size="small" className="mb-4">
                  <Table
                    columns={[
                      { title: '名称', dataIndex: 'name', key: 'name' },
                      { title: 'CPU 限制', key: 'cpu', render: (_: any, r: ResourceQuotaInfo) => `${r.used['limits.cpu'] || '-'} / ${r.hard['limits.cpu'] || '-'}` },
                      { title: '内存限制', key: 'mem', render: (_: any, r: ResourceQuotaInfo) => `${r.used['limits.memory'] || '-'} / ${r.hard['limits.memory'] || '-'}` },
                      { title: 'Pods', key: 'pods', render: (_: any, r: ResourceQuotaInfo) => `${r.used['count/pods'] || '0'} / ${r.hard['count/pods'] || '-'}` },
                      { title: 'Age', dataIndex: 'age', key: 'age' },
                    ]}
                    dataSource={resourceQuotas} rowKey="name" pagination={false} size="small"
                  />
                </Card>
                <Card title="LimitRange" size="small">
                  <Table
                    columns={[
                      { title: '名称', dataIndex: 'name', key: 'name' },
                      { title: '类型', dataIndex: 'type', key: 'type' },
                      { title: '默认CPU', key: 'default_cpu', render: (_: any, r: LimitRangeInfo) => r.limits?.[0]?.default?.cpu || '-' },
                      { title: '默认内存', key: 'default_mem', render: (_: any, r: LimitRangeInfo) => r.limits?.[0]?.default?.memory || '-' },
                      { title: 'Age', dataIndex: 'age', key: 'age' },
                    ]}
                    dataSource={limitRanges} rowKey="name" pagination={false} size="small"
                  />
                </Card>
              </Spin>
            </div>
          ),
        },
        {
          key: 'events',
          label: <span><InfoCircleOutlined /> 事件</span>,
          children: (
            <Card title="集群事件" extra={<Button icon={<ReloadOutlined />} onClick={loadEvents}>刷新</Button>}>
              <Table
                columns={[
                  { title: '类型', dataIndex: 'type', key: 'type', width: 80, render: (t: string) => <Tag color={t === 'Normal' ? 'green' : 'red'}>{t}</Tag> },
                  { title: 'Reason', dataIndex: 'reason', key: 'reason', width: 120 },
                  { title: '对象', key: 'object', render: (_: any, r: EventInfo) => `${r.namespace}/${r.name}` },
                  { title: '消息', dataIndex: 'message', key: 'message', ellipsis: true },
                  { title: '来源', dataIndex: 'source', key: 'source', width: 120 },
                  { title: '次数', dataIndex: 'count', key: 'count', width: 60 },
                  { title: 'Age', dataIndex: 'age', key: 'age', width: 80 },
                ]}
                dataSource={events} rowKey={(r, i) => `${r.namespace}-${r.name}-${i}`} pagination={{ pageSize: 20 }} size="small"
              />
            </Card>
          ),
        },
        {
          key: 'maintenance',
          label: <span><ToolOutlined /> 运维</span>,
          children: (
            <div className="space-y-4">
              <Card title="集群版本" size="small" className="mb-4">
                {clusterVersion ? (
                  <Descriptions column={2} size="small">
                    <Descriptions.Item label="Kubernetes">{clusterVersion.kubernetes_version}</Descriptions.Item>
                    <Descriptions.Item label="Platform">{clusterVersion.platform}</Descriptions.Item>
                    <Descriptions.Item label="Go Version">{clusterVersion.go_version}</Descriptions.Item>
                  </Descriptions>
                ) : <Text type="secondary">加载中...</Text>}
              </Card>
              <Card title="证书信息" size="small" className="mb-4" extra={
                cluster?.source === 'platform_managed' && (
                  <Popconfirm
                    title="续期证书"
                    description="确定要续期所有证书吗？此操作将重启控制平面组件。"
                    onConfirm={async () => {
                      try {
                        const res = await Api.cluster.renewCertificates(clusterId);
                        message.success(res.data.message);
                        loadClusterInfo();
                      } catch (err) {
                        message.error(err instanceof Error ? err.message : '证书续期失败');
                      }
                    }}
                    okText="确定"
                    cancelText="取消"
                  >
                    <Button size="small" icon={<SyncOutlined />}>续期证书</Button>
                  </Popconfirm>
                )
              }>
                <Table
                  columns={[
                    { title: '名称', dataIndex: 'name', key: 'name' },
                    { title: 'CA', dataIndex: 'ca', key: 'ca', width: 60, render: (v: boolean) => v ? <Tag color="blue">CA</Tag> : '-' },
                    { title: '过期时间', dataIndex: 'expires_at', key: 'expires_at' },
                    { title: '剩余天数', dataIndex: 'days_left', key: 'days_left', render: (d: number) => <Tag color={d < 30 ? 'red' : d < 90 ? 'orange' : 'green'}>{d} 天</Tag> },
                  ]}
                  dataSource={certificates} rowKey="name" pagination={false} size="small"
                />
              </Card>
              {upgradePlan && cluster?.source === 'platform_managed' && (
                <Card title="升级计划" size="small" extra={
                  upgradePlan.upgradable && (
                    <Popconfirm
                      title="升级集群"
                      description="确定要升级集群吗？建议先备份数据。"
                      onConfirm={async () => {
                        try {
                          // Extract version number from current version (e.g., v1.28.0 -> 1.29.0)
                          const currentParts = upgradePlan.current_version.replace('v', '').split('.');
                          const nextMinor = parseInt(currentParts[1]) + 1;
                          const targetVersion = `${currentParts[0]}.${nextMinor}.0`;
                          const res = await Api.cluster.upgradeCluster(clusterId, targetVersion);
                          message.success(res.data.message);
                        } catch (err) {
                          message.error(err instanceof Error ? err.message : '升级预览失败');
                        }
                      }}
                      okText="确定"
                      cancelText="取消"
                    >
                      <Button size="small" type="primary">升级集群</Button>
                    </Popconfirm>
                  )
                }>
                  <Descriptions column={1} size="small">
                    <Descriptions.Item label="当前版本">{upgradePlan.current_version}</Descriptions.Item>
                    <Descriptions.Item label="可升级">{upgradePlan.upgradable ? <Tag color="green">是</Tag> : <Tag color="red">否</Tag>}</Descriptions.Item>
                  </Descriptions>
                  {upgradePlan.warnings?.length > 0 && (
                    <div className="mt-4">
                      <Text type="warning">警告:</Text>
                      <ul className="list-disc pl-6 mt-2">
                        {upgradePlan.warnings.map((w, i) => <li key={i} className="text-orange-500">{w}</li>)}
                      </ul>
                    </div>
                  )}
                  {upgradePlan.steps?.length > 0 && (
                    <div className="mt-4">
                      <Text>升级步骤:</Text>
                      <ol className="list-decimal pl-6 mt-2">
                        {upgradePlan.steps.map((s, i) => <li key={i}>{s}</li>)}
                      </ol>
                    </div>
                  )}
                </Card>
              )}
            </div>
          ),
        },
      ]} />

      {/* Modals */}
      <Modal title="编辑集群" open={editModalVisible} onCancel={() => setEditModalVisible(false)} footer={null}>
        <Form form={editForm} layout="vertical" onFinish={handleEdit}>
          <Form.Item name="name" label="集群名称" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="description" label="描述"><Input.TextArea rows={3} /></Form.Item>
          <div className="flex justify-end gap-2">
            <Button onClick={() => setEditModalVisible(false)}>取消</Button>
            <Button type="primary" htmlType="submit">保存</Button>
          </div>
        </Form>
      </Modal>

      <Modal title="添加节点" open={addNodeModalVisible} onCancel={() => { setAddNodeModalVisible(false); addNodeForm.resetFields(); }} footer={null}>
        <Form form={addNodeForm} layout="vertical" onFinish={handleAddNodes}>
          <Form.Item name="hostIds" label="主机 ID" rules={[{ required: true }]} extra="多个 ID 用逗号分隔"><Input placeholder="例如: 1,2,3" /></Form.Item>
          <Form.Item name="role" label="角色" initialValue="worker">
            <Select options={[{ label: 'Worker', value: 'worker' }, { label: 'Control Plane', value: 'control-plane' }]} />
          </Form.Item>
          <div className="flex justify-end gap-2">
            <Button onClick={() => setAddNodeModalVisible(false)}>取消</Button>
            <Button type="primary" htmlType="submit">添加</Button>
          </div>
        </Form>
      </Modal>

      <Drawer title={`节点详情: ${selectedNode?.name}`} placement="right" width={600} onClose={() => setNodeDrawerVisible(false)} open={nodeDrawerVisible}>
        {selectedNode && (
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="名称" span={2}>{selectedNode.name}</Descriptions.Item>
            <Descriptions.Item label="IP">{selectedNode.ip}</Descriptions.Item>
            <Descriptions.Item label="状态">{getNodeStatusBadge(selectedNode.status)}</Descriptions.Item>
            <Descriptions.Item label="角色"><Tag color={selectedNode.role === 'control-plane' ? 'blue' : 'green'}>{selectedNode.role}</Tag></Descriptions.Item>
            <Descriptions.Item label="Kubelet">{selectedNode.kubelet_version}</Descriptions.Item>
            <Descriptions.Item label="容器运行时">{selectedNode.container_runtime}</Descriptions.Item>
            <Descriptions.Item label="操作系统">{selectedNode.os_image}</Descriptions.Item>
            <Descriptions.Item label="内核版本">{selectedNode.kernel_version}</Descriptions.Item>
            <Descriptions.Item label="CPU">{selectedNode.allocatable_cpu}</Descriptions.Item>
            <Descriptions.Item label="内存">{selectedNode.allocatable_mem}</Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </div>
  );
};

export default ClusterDetailPage;
