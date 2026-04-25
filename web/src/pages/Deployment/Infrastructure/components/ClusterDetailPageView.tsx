import React, { useState, useCallback, useMemo, useRef, useLayoutEffect, useEffect } from 'react';
import {
  Card, Tabs, Table, Tag, Button, Space, Descriptions, Spin, message,
  Modal, Form, Input, Popconfirm, Badge, Tooltip, Typography,
  Select, Dropdown, InputNumber, Row, Col, Progress, Empty, Pagination, Breadcrumb
} from 'antd';
import {
  ArrowLeftOutlined, ReloadOutlined, ClusterOutlined,
  DeleteOutlined, EditOutlined, ApiOutlined,
  PlusOutlined, SyncOutlined, NodeIndexOutlined, InfoCircleOutlined,
  AppstoreOutlined, CloudServerOutlined, SettingOutlined,
  DatabaseOutlined, CloudOutlined, ToolOutlined,
  MoreOutlined, SafetyOutlined, AuditOutlined, DownOutlined,
  CheckCircleOutlined, ExclamationCircleOutlined, MonitorOutlined,
  CopyOutlined, RightOutlined, BellOutlined,
  GlobalOutlined, RocketOutlined, DashboardOutlined
} from '@ant-design/icons';
import { 
  Layers, Activity, AlertTriangle, HardDrive, Box, Bell, 
  Info, ExternalLink, RefreshCw, Trash2, Settings, ShieldCheck,
  History, Layout, PieChart as PieIcon, LineChart as LineIcon,
  Terminal, Monitor
} from 'lucide-react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import { Pie, Line } from '@ant-design/charts';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { Api } from '../../../../api';
import { DetailSkeleton } from '../../../../components/LoadingSkeleton';
import ClusterDetailOverlays from './ClusterDetailOverlays';
import { ClusterDetailTabs } from './ClusterDetailTabs';
import { useClusterDetail } from '../hooks/useClusterDetail';
import { useClusterResources } from '../hooks/useClusterResources';
import { useClusterDetailPageOperations } from '../hooks/useClusterDetailPageOperations';

dayjs.extend(relativeTime);
const { Text, Title } = Typography;

// Sparkline component from HostListPage
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

const KPICard: React.FC<{ title: string; value: string; unit: string; subLabel: string; subValue: string; subColor: string; icon: React.ReactNode; iconBg: string; sparkPoints: number[]; sparkColor: string }> = ({ 
  title, value, unit, subLabel, subValue, subColor, icon, iconBg, sparkPoints, sparkColor 
}) => (
  <Card size="small" styles={{ body: { padding: '14px 16px 12px' } }} className="border border-[#e8edf3] rounded-[10px]">
    <div className="flex justify-between items-start">
      <div>
        <div className="text-[#6b7280] text-[13px]">{title}</div>
        <div className="mt-[6px] flex items-end gap-1">
          <span className="text-[32px] leading-none font-semibold text-[#111827] tracking-[-0.02em]">{value}</span>
          <span className="text-[16px] leading-6 text-[#374151] mb-[2px]">{unit}</span>
        </div>
        <div className="text-[12px] text-[#6b7280] mt-[8px]">
          {subLabel} <span style={{ color: subColor }}>{subValue}</span>
        </div>
      </div>
      <span className="w-10 h-10 rounded-xl inline-flex items-center justify-center" style={{ backgroundColor: iconBg }}>
        {icon}
      </span>
    </div>
    <Sparkline points={sparkPoints} color={sparkColor} />
  </Card>
);

const ClusterDetailPageView: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const clusterId = Number(id);

  const {
    isInitialLoading,
    cluster,
    nodes,
    nodesLoading,
    events,
    clusterVersion,
    certificates,
    upgradePlan,
    loadCluster,
    loadNodes,
    loadEvents,
    loadClusterInfo,
    syncNodes,
  } = useClusterDetail(clusterId);
  
  const {
    namespaces,
    selectedNamespace,
    setSelectedNamespace,
    deployments,
    statefulsets,
    daemonsets,
    pods,
    services,
    ingresses,
    configmaps,
    secrets,
    pvcs,
    pvs,
    clusterServices,
    resourceLoading,
    hpas,
    resourceQuotas,
    limitRanges,
    advancedLoading,
    refreshSelectedNamespaceResources,
  } = useClusterResources(clusterId);

  const {
    editModalVisible,
    setEditModalVisible,
    addNodeModalVisible,
    setAddNodeModalVisible,
    nodeDrawerVisible,
    setNodeDrawerVisible,
    selectedNode,
    approvalModalVisible,
    setApprovalModalVisible,
    pendingApprovalOperation,
    setPendingApprovalOperation,
    scaleModalVisible,
    setScaleModalVisible,
    pendingScaleOperation,
    setPendingScaleOperation,
    nodeMutationLoadingKey,
    nodeMetadataLoadingKey,
    serviceModalVisible,
    setServiceModalVisible,
    ingressModalVisible,
    setIngressModalVisible,
    pendingServiceModal,
    setPendingServiceModal,
    pendingIngressModal,
    setPendingIngressModal,
    editForm,
    addNodeForm,
    approvalForm,
    scaleForm,
    serviceForm,
    ingressForm,
    nodeLabelForm,
    nodeTaintForm,
    buildPolicyReleaseTraceLink,
    getStatusColor,
    getNodeStatusBadge,
    handleSyncNodes,
    handleTestConnection,
    handleOpenOperationCenter,
    handleEdit,
    handleDelete,
    handleAddNodes,
    submitScaleOperation,
    submitServiceModal,
    submitIngressModal,
    submitApprovalToken,
    handleNodeMetadataOperation,
    nodeColumns,
    buildWorkloadColumns,
    podColumns,
    serviceColumns,
    ingressColumns,
    configColumns,
    storageColumns,
    clusterServiceColumns,
    openServiceModal,
    openIngressModal,
    handleDeploymentRestart,
    handleDeploymentScale,
    handleDeploymentDelete,
    handleStatefulSetRestart,
    handleStatefulSetScale,
    handleStatefulSetDelete,
    handleRenewCertificates,
    handleClusterUpgrade,
    renderFeedback,
  } = useClusterDetailPageOperations({
    clusterId,
    cluster,
    selectedNamespace,
    refreshSelectedNamespaceResources,
    loadClusterInfo,
    loadNodes,
    loadCluster,
    syncNodes,
    upgradePlan,
    navigate,
  });

  const [activeTab, setActiveTab] = useState('overview');

  if (isInitialLoading) return <DetailSkeleton summaryCards={3} sections={4} />;
  if (!cluster) {
    return (
      <div className="h-[calc(100vh-112px)] flex flex-col items-center justify-center bg-white rounded-lg">
        <ClusterOutlined className="text-6xl text-gray-200 mb-4" />
        <p className="text-gray-400 text-lg">该集群不存在或已被移除</p>
        <Button type="primary" className="mt-4" onClick={() => navigate('/deployment/infrastructure/clusters')}>返回集群列表</Button>
      </div>
    );
  }

  const kpis = [
    { title: '节点总数', value: String(cluster.node_count || 0), unit: '台', subLabel: '较昨日', subValue: '+2', subColor: '#10b981', icon: <HardDrive size={18} color="#2563eb" />, iconBg: '#e8f1ff', sparkPoints: [54, 54, 55, 55, 56, 56], sparkColor: '#2563eb' },
    { title: '命名空间', value: String(namespaces.length || 0), unit: '个', subLabel: '较昨日', subValue: '+3', subColor: '#10b981', icon: <Layers size={18} color="#7c3aed" />, iconBg: '#f1ecff', sparkPoints: [80, 82, 83, 83, 85, 86], sparkColor: '#7c3aed' },
    { title: 'Pod 总数', value: '1,324', unit: '个', subLabel: '较昨日', subValue: '+48', subColor: '#10b981', icon: <Box size={18} color="#13c2c2" />, iconBg: '#e6fffb', sparkPoints: [1200, 1250, 1280, 1300, 1310, 1324], sparkColor: '#13c2c2' },
    { title: 'CPU 使用率', value: '43', unit: '%', subLabel: '较昨日', subValue: '+3%', subColor: '#f59e0b', icon: <DashboardOutlined className="text-[18px] text-[#2563eb]" />, iconBg: '#ebf3ff', sparkPoints: [40, 42, 41, 44, 43, 43], sparkColor: '#2563eb' },
    { title: '内存使用率', value: '62', unit: '%', subLabel: '较昨日', subValue: '+4%', subColor: '#f59e0b', icon: <ToolOutlined className="text-[18px] text-[#7c3aed]" />, iconBg: '#f1ecff', sparkPoints: [58, 60, 61, 63, 62, 62], sparkColor: '#7c3aed' },
    { title: '今日告警', value: '2', unit: '条', subLabel: '严重 1', subValue: '/ 警告 1', subColor: '#ef4444', icon: <BellOutlined className="text-[18px] text-[#f59e0b]" />, iconBg: '#fff4e8', sparkPoints: [1, 3, 2, 4, 1, 2], sparkColor: '#f59e0b' },
  ];

  return (
    <div className="flex flex-col gap-4">
      {/* 1. Breadcrumb */}
      <Breadcrumb items={[{ title: '资源管理' }, { title: '集群管理' }, { title: cluster.name }]} className="text-[12px] text-[#6b7280]" />

      {/* 2. Page Title & Action Bar */}
      <div className="flex justify-between items-start">
        <div>
          {/* Identity Bar */}
          <div className="flex items-center gap-3 bg-white px-4 py-2 rounded-lg border border-[#e8edf3]">
            <div className="w-8 h-8 rounded-md bg-blue-50 flex items-center justify-center">
              <ClusterOutlined className="text-blue-500 text-lg" />
            </div>
            <span className="font-bold text-[16px] text-[#111827]">{cluster.name}</span>
            <Tag color="green" className="m-0 border-none px-2 py-0 rounded-[4px] bg-[#e8f8f0] text-[#16a34a]">生产</Tag>
            <Tag color="blue" className="m-0 border-none px-2 py-0 rounded-[4px] bg-[#eef2ff] text-[#4f46e5]">华北-北京</Tag>
            <Tag color="cyan" className="m-0 border-none px-2 py-0 rounded-[4px] bg-[#e6fffb] text-[#0891b2]">{cluster.type || 'K8s'}</Tag>
            <span className="text-[#6b7280] text-[13px]">{cluster.k8s_version || cluster.version || 'v1.29.2'}</span>
            <span className="h-4 w-px bg-gray-200 mx-1" />
            <div className="flex items-center gap-1.5">
              <div className="w-1.5 h-1.5 rounded-full bg-[#16a34a]" />
              <span className="text-[13px] text-[#16a34a] font-medium">正常</span>
            </div>
          </div>
        </div>
        <Space size={8}>
          <Button icon={<AuditOutlined />} onClick={() => navigate(`/deployment/infrastructure/clusters/${clusterId}/operations`)}>操作中心</Button>
          <Button icon={<SafetyOutlined />} onClick={() => navigate(buildPolicyReleaseTraceLink())}>策略发布轨迹</Button>
          <Button icon={<SyncOutlined />} onClick={handleSyncNodes} loading={nodesLoading}>同步节点</Button>
          <Button icon={<ApiOutlined />} onClick={handleTestConnection}>测试连接</Button>
          <Dropdown
            menu={{
              items: [
                { key: 'edit', label: '编辑集群', icon: <EditOutlined />, onClick: () => { editForm.setFieldsValue({ name: cluster.name, description: cluster.description }); setEditModalVisible(true); } },
                { type: 'divider' },
                { key: 'delete', label: '删除集群', danger: true, icon: <DeleteOutlined />, onClick: handleDelete },
              ]
            }}
          >
            <Button icon={<MoreOutlined />} />
          </Dropdown>
        </Space>
      </div>

      {/* 3. KPI Indicator Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 xl:grid-cols-6 gap-3 mt-1">
        {kpis.map((kpi, idx) => <KPICard key={idx} {...kpi} />)}
      </div>

      {/* 4. Middle Summary Area (3 Columns) */}
      <Row gutter={[16, 16]}>
        {/* Basic Info */}
        <Col xs={24} lg={12} xl={10}>
          <Card 
            title={<span className="text-[14px] font-bold"><InfoCircleOutlined className="mr-2 text-blue-500" />基础信息</span>}
            size="small"
            className="h-full border-[#e8edf3] rounded-[10px]"
            extra={<Button type="text" size="small" icon={<EditOutlined className="text-gray-400" />} onClick={() => { editForm.setFieldsValue({ name: cluster.name, description: cluster.description }); setEditModalVisible(true); }}/>}
          >
            <div className="grid grid-cols-2 gap-x-4 gap-y-4 py-2">
              {[
                { label: '集群名称', value: cluster.name },
                { label: 'API 地址', value: cluster.endpoint || '-', copyable: true },
                { label: '集群类型', value: cluster.type || 'K8s' },
                { label: 'K8s 版本', value: cluster.k8s_version || cluster.version || 'v1.29.2' },
                { label: '集群 ID', value: `cls-${cluster.id.toString(16).padEnd(6, '0')}` },
                { label: '所属项目', value: '支付核心平台' },
                { label: '创建时间', value: '2024-03-18 10:20' },
                { label: '最近同步', value: '1 分钟前' },
                { label: '认证方式', value: 'kubeconfig' },
                { label: '网络插件', value: 'Cilium' },
                { label: '容器运行时', value: 'containerd' },
                { label: '备注', value: cluster.description || '-' },
              ].map((item, idx) => (
                <div key={idx} className="flex flex-col gap-0.5">
                  <span className="text-[#9ca3af] text-[11px]">{item.label}</span>
                  <div className="flex items-center gap-1">
                    <span className="text-[#374151] text-[13px] font-medium truncate max-w-[120px]" title={item.value}>{item.value}</span>
                    {item.copyable && <CopyOutlined className="text-[10px] text-blue-500 cursor-pointer" onClick={() => { navigator.clipboard.writeText(item.value); message.success('已复制'); }} />}
                  </div>
                </div>
              ))}
            </div>
          </Card>
        </Col>

        {/* Cluster Health */}
        <Col xs={24} lg={12} xl={9}>
          <Card 
            title={<span className="text-[14px] font-bold"><Activity className="mr-2 text-green-500 inline-block" size={16} />集群健康</span>}
            size="small"
            className="h-full border-[#e8edf3] rounded-[10px]"
          >
            <div className="flex items-center gap-6 py-4">
              <div className="relative flex-shrink-0">
                <Progress 
                  type="circle" 
                  percent={96} 
                  strokeColor="#10b981" 
                  strokeWidth={8} 
                  width={100}
                  format={(p) => (
                    <div className="flex flex-col">
                      <span className="text-2xl font-bold text-[#111827]">{p}</span>
                      <span className="text-[11px] text-[#6b7280]">健康分</span>
                    </div>
                  )}
                />
              </div>
              <div className="flex-1 space-y-3">
                {[
                  { name: 'API Server', status: '正常' },
                  { name: 'Scheduler', status: '正常' },
                  { name: 'Controller', status: '正常' },
                  { name: 'etcd', status: '正常' },
                  { name: 'CoreDNS', status: '正常' },
                  { name: 'Ingress', status: '正常' },
                ].map((comp, idx) => (
                  <div key={idx} className="flex items-center justify-between text-[12px]">
                    <span className="text-[#6b7280]">{comp.name}</span>
                    <div className="flex items-center gap-1.5">
                      <div className="w-1.5 h-1.5 rounded-full bg-[#16a34a]" />
                      <span className="text-[#16a34a] font-medium">{comp.status}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </Card>
        </Col>

        {/* Quick Actions */}
        <Col xs={24} lg={24} xl={5}>
          <Card 
            title={<span className="text-[14px] font-bold"><RocketOutlined className="mr-2 text-orange-500" />快捷操作</span>}
            size="small"
            className="h-full border-[#e8edf3] rounded-[10px]"
          >
            <div className="flex flex-col gap-2 py-1">
              <Button block className="h-9 text-left px-3 flex items-center justify-between group border-[#e5e7eb] hover:border-blue-500 transition-all rounded-lg" onClick={() => setActiveTab('nodes')}>
                <Space><HardDrive size={16} className="text-blue-500" /> <span className="text-[13px] font-medium text-[#374151]">查看节点</span></Space>
                <RightOutlined className="text-[10px] text-gray-300 group-hover:text-blue-500 transition-colors" />
              </Button>
              <Button block className="h-9 text-left px-3 flex items-center justify-between group border-[#e5e7eb] hover:border-blue-500 transition-all rounded-lg" onClick={() => setActiveTab('workloads')}>
                <Space><Box size={16} className="text-cyan-500" /> <span className="text-[13px] font-medium text-[#374151]">查看工作负载</span></Space>
                <RightOutlined className="text-[10px] text-gray-300 group-hover:text-blue-500 transition-colors" />
              </Button>
              <Button block className="h-9 text-left px-3 flex items-center justify-between group border-[#e5e7eb] hover:border-blue-500 transition-all rounded-lg" onClick={() => setActiveTab('monitor')}>
                <Space><Monitor size={16} className="text-indigo-500" /> <span className="text-[13px] font-medium text-[#374151]">查看监控</span></Space>
                <RightOutlined className="text-[10px] text-gray-300 group-hover:text-blue-500 transition-colors" />
              </Button>
              <Button block className="h-9 text-left px-3 flex items-center justify-between group border-[#e5e7eb] hover:border-orange-500 transition-all rounded-lg">
                <Space><Terminal size={18} className="text-orange-500" /> <span className="text-[13px] font-medium text-[#374151]">打开终端</span></Space>
                <RightOutlined className="text-[10px] text-gray-300 group-hover:text-orange-500 transition-colors" />
              </Button>
              <Button block className="h-9 text-left px-3 flex items-center justify-between group border-[#e5e7eb] hover:border-green-500 transition-all rounded-lg" onClick={handleSyncNodes} loading={nodesLoading}>
                <Space><RefreshCw size={16} className="text-green-500" /> <span className="text-[13px] font-medium text-[#374151]">同步配置</span></Space>
                <RightOutlined className="text-[10px] text-gray-300 group-hover:text-green-500 transition-colors" />
              </Button>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 5. Function Detail Tabs */}
      <div className="bg-white rounded-xl border border-[#e8edf3]">
        <div className="px-6 border-b border-[#f3f4f6]">
          <Tabs 
            activeKey={activeTab} 
            onChange={setActiveTab}
            className="[&_.ant-tabs-tab]:py-4 [&_.ant-tabs-tab]:m-0 [&_.ant-tabs-tab+.ant-tabs-tab]:ml-8 [&_.ant-tabs-tab-btn]:text-[14px] [&_.ant-tabs-tab-btn]:font-medium [&_.ant-tabs-tab-active_.ant-tabs-tab-btn]:text-blue-600"
            items={[
              { key: 'overview', label: '概览' },
              { key: 'nodes', label: '节点' },
              { key: 'namespaces', label: '命名空间' },
              { key: 'workloads', label: '工作负载' },
              { key: 'monitor', label: '监控' },
              { key: 'events', label: '事件' },
              { key: 'alerts', label: '告警' },
              { key: 'audit', label: '操作记录' },
            ]}
          />
        </div>
        
        <div className="p-6">
          <ClusterDetailTabs
            cluster={cluster}
            clusterId={clusterId}
            nodes={nodes}
            nodesLoading={nodesLoading}
            selectedNamespace={selectedNamespace}
            setSelectedNamespace={setSelectedNamespace}
            namespaces={namespaces}
            deployments={deployments}
            statefulsets={statefulsets}
            daemonsets={daemonsets}
            pods={pods}
            services={services}
            ingresses={ingresses}
            configmaps={configmaps}
            secrets={secrets}
            pvcs={pvcs}
            pvs={pvs}
            clusterServices={clusterServices}
            resourceLoading={resourceLoading}
            advancedLoading={advancedLoading}
            hpas={hpas}
            resourceQuotas={resourceQuotas}
            limitRanges={limitRanges}
            clusterVersion={clusterVersion}
            certificates={certificates}
            upgradePlan={upgradePlan}
            events={events}
            loadEvents={loadEvents}
            nodeColumns={nodeColumns}
            buildWorkloadColumns={buildWorkloadColumns}
            podColumns={podColumns}
            serviceColumns={serviceColumns}
            ingressColumns={ingressColumns}
            configColumns={configColumns}
            storageColumns={storageColumns}
            clusterServiceColumns={clusterServiceColumns}
            handleDeploymentRestart={handleDeploymentRestart}
            handleDeploymentScale={handleDeploymentScale}
            handleDeploymentDelete={handleDeploymentDelete}
            handleStatefulSetRestart={handleStatefulSetRestart}
            handleStatefulSetScale={handleStatefulSetScale}
            handleStatefulSetDelete={handleStatefulSetDelete}
            openServiceModal={openServiceModal}
            openIngressModal={openIngressModal}
            renderFeedback={renderFeedback}
            handleRenewCertificates={handleRenewCertificates}
            handleClusterUpgrade={handleClusterUpgrade}
            setAddNodeModalVisible={setAddNodeModalVisible}
            activeKey={activeTab}
            onChange={setActiveTab}
          />
        </div>
      </div>

      <ClusterDetailOverlays
        serviceModalVisible={serviceModalVisible}
        pendingServiceModal={pendingServiceModal}
        submitServiceModal={submitServiceModal}
        onCloseServiceModal={() => { setServiceModalVisible(false); setPendingServiceModal(null); serviceForm.resetFields(); }}
        serviceForm={serviceForm}
        ingressModalVisible={ingressModalVisible}
        pendingIngressModal={pendingIngressModal}
        submitIngressModal={submitIngressModal}
        onCloseIngressModal={() => { setIngressModalVisible(false); setPendingIngressModal(null); ingressForm.resetFields(); }}
        ingressForm={ingressForm}
        approvalModalVisible={approvalModalVisible}
        pendingApprovalOperation={pendingApprovalOperation}
        submitApprovalToken={submitApprovalToken}
        onCloseApprovalModal={() => { setApprovalModalVisible(false); setPendingApprovalOperation(null); approvalForm.resetFields(); }}
        approvalForm={approvalForm}
        nodeMutationLoadingKey={nodeMutationLoadingKey}
        nodeDrawerVisible={nodeDrawerVisible}
        onCloseNodeDrawer={() => setNodeDrawerVisible(false)}
        selectedNode={selectedNode}
        getNodeStatusBadge={getNodeStatusBadge}
        nodeLabelForm={nodeLabelForm}
        nodeTaintForm={nodeTaintForm}
        nodeMetadataLoadingKey={nodeMetadataLoadingKey}
        handleNodeMetadataOperation={handleNodeMetadataOperation}
      />

      {/* Edit Modal */}
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

      {/* Add Node Modal */}
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

      {/* Scale Modal */}
      <Modal
        title="调整副本数"
        open={scaleModalVisible}
        onCancel={() => { setScaleModalVisible(false); setPendingScaleOperation(null); scaleForm.resetFields(); }}
        onOk={submitScaleOperation}
        okText="提交扩缩容"
        cancelText="取消"
        confirmLoading={Boolean(pendingScaleOperation && nodeMutationLoadingKey === pendingScaleOperation.loadingKey)}
      >
        <Form form={scaleForm} layout="vertical">
          <Form.Item name="replicas" label="副本数" rules={[{ required: true, message: '请输入副本数' }]} initialValue={pendingScaleOperation?.currentReplicas}>
            <InputNumber min={0} className="w-full" />
          </Form.Item>
        </Form>
      </Modal>

      <style>{`
        .scrollbar-thin::-webkit-scrollbar { width: 6px; height: 6px; }
        .scrollbar-thin::-webkit-scrollbar-track { background: transparent; }
        .scrollbar-thin::-webkit-scrollbar-thumb { background: #e5e7eb; border-radius: 10px; }
        .scrollbar-thin::-webkit-scrollbar-thumb:hover { background: #d1d5db; }
      `}</style>
    </div>
  );
};

export default ClusterDetailPageView;
