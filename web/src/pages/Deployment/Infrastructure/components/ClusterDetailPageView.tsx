import React, { useState, useCallback } from 'react';
import {
  Card, Tabs, Table, Tag, Button, Space, Descriptions, Spin, message,
  Modal, Form, Input, Popconfirm, Badge, Tooltip, Typography,
  Select, Dropdown, InputNumber
} from 'antd';
import {
  ArrowLeftOutlined, ReloadOutlined, ClusterOutlined,
  DeleteOutlined, EditOutlined, ApiOutlined,
  PlusOutlined, SyncOutlined, NodeIndexOutlined, InfoCircleOutlined,
  AppstoreOutlined, CloudServerOutlined, SettingOutlined,
  DatabaseOutlined, CloudOutlined, ToolOutlined,
  MoreOutlined, SafetyOutlined, AuditOutlined
} from '@ant-design/icons';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { Api } from '../../../../api';
import { DetailSkeleton } from '../../../../components/LoadingSkeleton';
import { GuidedFormItem } from '../../../../components/FormGuidance';
import ClusterOperationsPanel from './ClusterOperationsPanel';
import ClusterOverviewPanel from './ClusterOverviewPanel';
import ClusterDetailOverlays from './ClusterDetailOverlays';
import { ClusterDetailTabs } from './ClusterDetailTabs';
import { useClusterDetail } from '../hooks/useClusterDetail';
import { useClusterResources } from '../hooks/useClusterResources';
import { useClusterDetailPageOperations } from '../hooks/useClusterDetailPageOperations';
import type {
  ClusterNode, DeploymentInfo,
  StatefulSetInfo, PodInfo, ServiceInfo, IngressInfo,
  ConfigMapInfo, SecretInfo, HPAInfo, ResourceQuotaInfo, LimitRangeInfo,
  ClusterOperationApproval, ClusterOperationResponse, ClusterOperationState,
  ClusterServiceMutationPayload, ClusterIngressMutationPayload, EventInfo
} from '../../../../api/modules/cluster';

const { Text, Title } = Typography;
const HIGH_RISK_RUNBOOK_PATH = '/docs/runbooks/cluster-high-risk-operations.md';

type HighRiskFailureGuidance = {
  summary: string;
  href: string;
};

const getHighRiskFailureGuidance = (actionKey: string): HighRiskFailureGuidance | undefined => {
  if (actionKey.includes('drain')) {
    return {
      summary: '核对未驱逐 Pod、PDB 与 DaemonSet 阻塞，必要时先补齐疏散窗口或人工迁移后再重试。',
      href: HIGH_RISK_RUNBOOK_PATH,
    };
  }
  if (actionKey.includes('remove')) {
    return {
      summary: '确认节点已完成 drain、已从流量与自动伸缩池摘除，再核对 kubelet 与云主机状态后重试移除。',
      href: HIGH_RISK_RUNBOOK_PATH,
    };
  }
  if (actionKey.includes('renew')) {
    return {
      summary: '逐项核对 apiserver、controller-manager、scheduler 证书与静态 Pod 重启情况，再决定回滚或重签。',
      href: HIGH_RISK_RUNBOOK_PATH,
    };
  }
  if (actionKey.includes('upgrade')) {
    return {
      summary: '先冻结变更并确认 etcd 与控制平面备份可恢复，再按失败阶段逐项修复后重新生成升级计划。',
      href: HIGH_RISK_RUNBOOK_PATH,
    };
  }
  return undefined;
};

const buildFailureFeedbackMessage = (messageText: string, guidance?: HighRiskFailureGuidance) => {
  if (!guidance) {
    return messageText;
  }
  return (
    <span>
      {messageText}
      {' '}处置建议：{guidance.summary}{' '}
      <a href={guidance.href} target="_blank" rel="noopener noreferrer">
        查看运行手册
      </a>
    </span>
  );
};

const ClusterDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const clusterId = Number(id);

  const [infoExpanded, setInfoExpanded] = useState(false);
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

  if (isInitialLoading) return <DetailSkeleton summaryCards={3} sections={4} />;
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
          <Button icon={<AuditOutlined />} onClick={() => navigate(`/deployment/infrastructure/clusters/${clusterId}/operations`)}>
            操作中心
          </Button>
          <Button icon={<SafetyOutlined />} onClick={() => navigate(buildPolicyReleaseTraceLink())}>
            策略发布轨迹
          </Button>
          <Button icon={<SyncOutlined />} onClick={handleSyncNodes} loading={nodesLoading}>同步节点</Button>
          <Button icon={<ApiOutlined />} onClick={handleTestConnection}>测试连接</Button>
          <Button icon={<EditOutlined />} onClick={() => { editForm.setFieldsValue({ name: cluster.name, description: cluster.description }); setEditModalVisible(true); }}>编辑</Button>
          <Popconfirm title="确定删除此集群？" onConfirm={handleDelete} okText="确定" cancelText="取消">
            <Button danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      </div>

      <ClusterOverviewPanel cluster={cluster} statusColor={getStatusColor(cluster.status)}>
        <ClusterOperationsPanel
          operationCenterHref={`/deployment/infrastructure/clusters/${clusterId}/operations`}
          securityHref={`/deployment/infrastructure/clusters/${clusterId}/security`}
          policyHref={`/deployment/infrastructure/clusters/${clusterId}/policies`}
          nodesLoading={nodesLoading}
          onOpenOperationCenter={handleOpenOperationCenter}
          onSyncNodes={handleSyncNodes}
        />
      </ClusterOverviewPanel>

      <div>
        <Button onClick={() => setInfoExpanded((v) => !v)}>
          {infoExpanded ? '收起基础信息' : '展开基础信息'}
        </Button>
      </div>
      {infoExpanded ? (
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
      ) : null}

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
      />

      {/* Modals */}
      <Modal title="编辑集群" open={editModalVisible} onCancel={() => setEditModalVisible(false)} footer={null}>
        <Form form={editForm} layout="vertical" onFinish={handleEdit}>
          <GuidedFormItem name="name" label="集群名称" rules={[{ required: true }]}><Input /></GuidedFormItem>
          <GuidedFormItem name="description" label="描述"><Input.TextArea rows={3} /></GuidedFormItem>
          <div className="flex justify-end gap-2">
            <Button onClick={() => setEditModalVisible(false)}>取消</Button>
            <Button type="primary" htmlType="submit">保存</Button>
          </div>
        </Form>
      </Modal>

      <Modal title="添加节点" open={addNodeModalVisible} onCancel={() => { setAddNodeModalVisible(false); addNodeForm.resetFields(); }} footer={null}>
        <Form form={addNodeForm} layout="vertical" onFinish={handleAddNodes}>
          <GuidedFormItem name="hostIds" label="主机 ID" rules={[{ required: true }]} extra="多个 ID 用逗号分隔"><Input placeholder="例如: 1,2,3" /></GuidedFormItem>
          <Form.Item name="role" label="角色" initialValue="worker">
            <Select options={[{ label: 'Worker', value: 'worker' }, { label: 'Control Plane', value: 'control-plane' }]} />
          </Form.Item>
          <div className="flex justify-end gap-2">
            <Button onClick={() => setAddNodeModalVisible(false)}>取消</Button>
            <Button type="primary" htmlType="submit">添加</Button>
          </div>
        </Form>
      </Modal>

      <Modal
        title="调整副本数"
        open={scaleModalVisible}
        onCancel={() => {
          setScaleModalVisible(false);
          setPendingScaleOperation(null);
          scaleForm.resetFields();
        }}
        onOk={submitScaleOperation}
        okText="提交扩缩容"
        cancelText="取消"
        confirmLoading={Boolean(pendingScaleOperation && nodeMutationLoadingKey === pendingScaleOperation.loadingKey)}
        destroyOnHidden
      >
        <Form form={scaleForm} layout="vertical">
          <GuidedFormItem
            name="replicas"
            label="replicas"
            rules={[{ required: true, message: '请输入副本数' }]}
            initialValue={pendingScaleOperation?.currentReplicas}
          >
            <InputNumber min={0} className="w-full" aria-label="replicas" />
          </GuidedFormItem>
        </Form>
      </Modal>

      <ClusterDetailOverlays
        serviceModalVisible={serviceModalVisible}
        pendingServiceModal={pendingServiceModal}
        submitServiceModal={submitServiceModal}
        onCloseServiceModal={() => {
          setServiceModalVisible(false);
          setPendingServiceModal(null);
          serviceForm.resetFields();
        }}
        serviceForm={serviceForm}
        ingressModalVisible={ingressModalVisible}
        pendingIngressModal={pendingIngressModal}
        submitIngressModal={submitIngressModal}
        onCloseIngressModal={() => {
          setIngressModalVisible(false);
          setPendingIngressModal(null);
          ingressForm.resetFields();
        }}
        ingressForm={ingressForm}
        approvalModalVisible={approvalModalVisible}
        pendingApprovalOperation={pendingApprovalOperation}
        submitApprovalToken={submitApprovalToken}
        onCloseApprovalModal={() => {
          setApprovalModalVisible(false);
          setPendingApprovalOperation(null);
          approvalForm.resetFields();
        }}
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
    </div>
  );
};

export default ClusterDetailPage;
