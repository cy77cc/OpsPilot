import React, { useState, useCallback } from 'react';
import {
  Tag, Button, Space, message,
  Modal, Form, Popconfirm, Badge, Tooltip, Typography,
  Dropdown,
} from 'antd';
import {
  DeleteOutlined,
  InfoCircleOutlined,
  MoreOutlined,
  SafetyOutlined,
  AuditOutlined,
} from '@ant-design/icons';
import { Link } from 'react-router-dom';
import { Api } from '../../../../api';
import type {
  ClusterNode,
  DeploymentInfo,
  StatefulSetInfo,
  PodInfo,
  ServiceInfo,
  IngressInfo,
  ConfigMapInfo,
  SecretInfo,
  ClusterOperationApproval,
  ClusterOperationResponse,
  ClusterOperationState,
  ClusterServiceMutationPayload,
  ClusterIngressMutationPayload,
} from '../../../../api/modules/cluster';

const { Text } = Typography;
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

export function useClusterDetailPageOperations(params: {
  clusterId: number;
  cluster: any;
  selectedNamespace: string;
  refreshSelectedNamespaceResources: () => Promise<void>;
  loadClusterInfo: () => Promise<void>;
  loadNodes: () => Promise<void>;
  loadCluster: () => Promise<void>;
  syncNodes: () => Promise<void>;
  upgradePlan: any;
  navigate: (path: string) => void;
}) {
  const {
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
  } = params;
  // Modals
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [addNodeModalVisible, setAddNodeModalVisible] = useState(false);
  const [nodeDrawerVisible, setNodeDrawerVisible] = useState(false);
  const [selectedNode, setSelectedNode] = useState<ClusterNode | null>(null);
  const [approvalModalVisible, setApprovalModalVisible] = useState(false);
  const [pendingApprovalOperation, setPendingApprovalOperation] = useState<null | {
    title: string;
    actionKey: string;
    loadingKey: string;
    feedbackKey: string;
    approval?: ClusterOperationApproval;
    retry: (approvalToken?: string) => Promise<ClusterOperationResponse<any>>;
    refresh?: () => Promise<void>;
  }>(null);
  const [scaleModalVisible, setScaleModalVisible] = useState(false);
  const [pendingScaleOperation, setPendingScaleOperation] = useState<null | {
    title: string;
    actionKey: string;
    loadingKey: string;
    feedbackKey: string;
    currentReplicas: number;
    execute: (replicas: number, approvalToken?: string) => Promise<ClusterOperationResponse<any>>;
  }>(null);
  const [nodeMutationLoadingKey, setNodeMutationLoadingKey] = useState<string>('');
  const [nodeMetadataLoadingKey, setNodeMetadataLoadingKey] = useState<string>('');
  const [serviceModalVisible, setServiceModalVisible] = useState(false);
  const [ingressModalVisible, setIngressModalVisible] = useState(false);
  const [pendingServiceModal, setPendingServiceModal] = useState<null | {
    mode: 'create' | 'edit';
    record?: ServiceInfo;
  }>(null);
  const [pendingIngressModal, setPendingIngressModal] = useState<null | {
    mode: 'create' | 'edit';
    record?: IngressInfo;
  }>(null);
  const [operationFeedback, setOperationFeedback] = useState<Record<string, {
    action: string;
    state: ClusterOperationState;
    message: string;
    audit_id?: string | number;
    guidance?: HighRiskFailureGuidance;
  }>>({});
  const [editForm] = Form.useForm();
  const [addNodeForm] = Form.useForm();
  const [approvalForm] = Form.useForm();
  const [scaleForm] = Form.useForm();
  const [serviceForm] = Form.useForm();
  const [ingressForm] = Form.useForm();
  const [nodeLabelForm] = Form.useForm();
  const [nodeTaintForm] = Form.useForm();

  const buildOperationLink = useCallback((auditId?: string | number) => {
    if (!auditId) {return '';}
    return `/deployment/infrastructure/clusters/${clusterId}/operations?audit_id=${encodeURIComponent(String(auditId))}`;
  }, [clusterId]);

  const buildPolicyReleaseTraceLink = useCallback((releaseId?: string | number, auditId?: string | number) => {
    const params = new URLSearchParams();
    params.set('resource', 'policy_release');
    if (releaseId) {
      params.set('release_id', String(releaseId));
    }
    if (auditId) {
      params.set('audit_id', String(auditId));
    }
    return `/deployment/infrastructure/clusters/${clusterId}/operations?${params.toString()}`;
  }, [clusterId]);

  const recordOperationFeedback = useCallback((resourceKey: string, feedback: {
    action: string;
    state: ClusterOperationState;
    message: string;
    audit_id?: string | number;
    guidance?: HighRiskFailureGuidance;
  }) => {
    setOperationFeedback((prev) => ({
      ...prev,
      [resourceKey]: feedback,
    }));
  }, []);

  const openApprovalModal = useCallback((
    operation: {
      title: string;
      actionKey: string;
      loadingKey: string;
      feedbackKey: string;
      approval?: ClusterOperationApproval;
      retry: (approvalToken?: string) => Promise<ClusterOperationResponse<any>>;
      refresh?: () => Promise<void>;
    },
  ) => {
    setPendingApprovalOperation(operation);
    approvalForm.resetFields();
    approvalForm.setFieldsValue({ approval_token: '' });
    setApprovalModalVisible(true);
  }, [approvalForm]);

  const executeClusterOperation = useCallback(async <T,>(
    actionKey: string,
    loadingKey: string,
    feedbackKey: string,
    actionLabel: string,
    runner: (approvalToken?: string) => Promise<ClusterOperationResponse<T>>,
    refresh?: () => Promise<void>,
  ) => {
    setNodeMutationLoadingKey(loadingKey);
    try {
      const result = await runner(undefined);
      const failureGuidance = result.state === 'failed'
        ? getHighRiskFailureGuidance(actionKey)
        : undefined;
      recordOperationFeedback(feedbackKey, {
        action: actionLabel,
        state: result.state,
        message: result.message,
        audit_id: result.audit_id,
        guidance: failureGuidance,
      });

      if (result.state === 'approval_required') {
        openApprovalModal({
          title: actionLabel,
          actionKey,
          loadingKey,
          feedbackKey,
          approval: result.approval,
          retry: runner,
          refresh,
        });
        message.warning(result.approval?.ticket
          ? `${actionLabel} 已进入审批，ticket: ${result.approval.ticket}`
          : `${actionLabel} 已进入审批`);
        return result;
      }

      if (result.state === 'rejected') {
        message.error(result.message || `${actionLabel} 已拒绝`);
      } else if (result.state === 'failed') {
        const failureMessage = result.message || `${actionLabel} 失败`;
        message.error(buildFailureFeedbackMessage(failureMessage, failureGuidance));
      } else {
        message.success(result.message || `${actionLabel} 成功`);
      }

      if (result.audit_id) {
        message.info(
          <span>
            审计记录已生成，前往{' '}
            <a href={buildOperationLink(result.audit_id)}>操作中心</a>
          </span>,
        );
      }

      if (refresh) {
        await refresh();
      } else if (actionKey.includes('upgrade') || actionKey.includes('renew')) {
        await loadClusterInfo();
      } else {
        await loadNodes();
      }
      return result;
    } catch (err) {
      const failureMessage = err instanceof Error ? err.message : `${actionLabel} 失败`;
      const failureGuidance = getHighRiskFailureGuidance(actionKey);
      recordOperationFeedback(feedbackKey, {
        action: actionLabel,
        state: 'failed',
        message: failureMessage,
        guidance: failureGuidance,
      });
      message.error(buildFailureFeedbackMessage(failureMessage, failureGuidance));
      throw err;
    } finally {
      setNodeMutationLoadingKey('');
    }
  }, [buildOperationLink, loadClusterInfo, loadNodes, openApprovalModal, recordOperationFeedback]);

  const submitApprovalToken = useCallback(async () => {
    if (!pendingApprovalOperation) {return;}
    try {
      const values = await approvalForm.validateFields();
      const token = String(values.approval_token || '').trim();
      if (!token) {
        message.warning('请输入 approval_token');
        return;
      }
      setNodeMutationLoadingKey(pendingApprovalOperation.loadingKey);
      const result = await pendingApprovalOperation.retry(token);
      const failureGuidance = result.state === 'failed'
        ? getHighRiskFailureGuidance(pendingApprovalOperation.actionKey)
        : undefined;
      recordOperationFeedback(pendingApprovalOperation.feedbackKey, {
        action: pendingApprovalOperation.title,
        state: result.state,
        message: result.message,
        audit_id: result.audit_id,
        guidance: failureGuidance,
      });
      setApprovalModalVisible(false);
      setPendingApprovalOperation(null);
      approvalForm.resetFields();
      if (result.state === 'completed') {
        message.success(result.message || `${pendingApprovalOperation.title} 已完成`);
        if (result.audit_id) {
          message.info(
            <span>
              审计记录已生成，前往{' '}
              <a href={buildOperationLink(result.audit_id)}>操作中心</a>
            </span>,
          );
        }
        if (pendingApprovalOperation.refresh) {
          await pendingApprovalOperation.refresh();
        } else if (pendingApprovalOperation.actionKey.includes('upgrade') || pendingApprovalOperation.actionKey.includes('renew')) {
          await loadClusterInfo();
        } else {
          await loadNodes();
        }
        return;
      }
      if (result.state === 'approval_required') {
        openApprovalModal({
          title: pendingApprovalOperation.title,
          actionKey: pendingApprovalOperation.actionKey,
          loadingKey: pendingApprovalOperation.loadingKey,
          feedbackKey: pendingApprovalOperation.feedbackKey,
          approval: result.approval,
          retry: pendingApprovalOperation.retry,
          refresh: pendingApprovalOperation.refresh,
        });
        message.warning(result.approval?.ticket
          ? `${pendingApprovalOperation.title} 仍需审批，ticket: ${result.approval.ticket}`
          : `${pendingApprovalOperation.title} 仍需审批`);
        return;
      }
      if (result.state === 'rejected') {
        message.error(result.message || `${pendingApprovalOperation.title} 已拒绝`);
      } else if (result.state === 'failed') {
        const failureMessage = result.message || `${pendingApprovalOperation.title} 失败`;
        message.error(buildFailureFeedbackMessage(failureMessage, failureGuidance));
      }
      if (pendingApprovalOperation.refresh) {
        await pendingApprovalOperation.refresh();
      } else if (pendingApprovalOperation.actionKey.includes('upgrade') || pendingApprovalOperation.actionKey.includes('renew')) {
        await loadClusterInfo();
      } else {
        await loadNodes();
      }
    } catch (err) {
      if (err instanceof Error && pendingApprovalOperation) {
        const failureGuidance = getHighRiskFailureGuidance(pendingApprovalOperation.actionKey);
        recordOperationFeedback(pendingApprovalOperation.feedbackKey, {
          action: pendingApprovalOperation.title,
          state: 'failed',
          message: err.message,
          guidance: failureGuidance,
        });
        message.error(buildFailureFeedbackMessage(err.message, failureGuidance));
      } else {
        message.error(err instanceof Error ? err.message : '提交审批失败');
      }
    } finally {
      setNodeMutationLoadingKey('');
    }
  }, [approvalForm, buildOperationLink, loadClusterInfo, loadNodes, openApprovalModal, pendingApprovalOperation, recordOperationFeedback]);

  const performNodeOperation = useCallback(async (
    actionKey: string,
    node: ClusterNode,
    runner: (approvalToken?: string) => Promise<ClusterOperationResponse<any>>,
  ) => {
    const actionLabels: Record<string, string> = {
      cordon: '节点隔离',
      uncordon: '节点恢复',
      drain: '节点排空',
      remove: '节点移除',
    };
    const actionLabel = actionLabels[actionKey] || actionKey;
    await executeClusterOperation(actionKey, `${node.name}:${actionKey}`, node.name, actionLabel, runner);
  }, [executeClusterOperation]);

  const handleNodeMetadataOperation = useCallback(async (
    kind: 'label' | 'taint',
    mode: 'upsert' | 'remove',
    node: ClusterNode,
    values: { key: string; value?: string; effect?: string; approvalToken?: string },
  ) => {
    const resourceKey = `${node.name}:${kind}:${values.key}`;
    setNodeMetadataLoadingKey(resourceKey);
    try {
      const runner = async (approvalToken?: string) => {
        if (kind === 'label' && mode === 'upsert') {
          return Api.cluster.upsertNodeLabel(clusterId, node.name, {
            key: values.key,
            value: values.value,
            approval_token: approvalToken,
          }).then((resp) => resp.data);
        }
        if (kind === 'label' && mode === 'remove') {
          return Api.cluster.removeNodeLabel(clusterId, node.name, {
            key: values.key,
            value: values.value,
            approval_token: approvalToken,
          }).then((resp) => resp.data);
        }
        if (kind === 'taint' && mode === 'upsert') {
          return Api.cluster.upsertNodeTaint(clusterId, node.name, {
            key: values.key,
            value: values.value,
            effect: values.effect,
            approval_token: approvalToken,
          }).then((resp) => resp.data);
        }
        return Api.cluster.removeNodeTaint(clusterId, node.name, {
          key: values.key,
          value: values.value,
          effect: values.effect,
          approval_token: approvalToken,
        }).then((resp) => resp.data);
      };

      const result = await executeClusterOperation(
        `${kind}:${mode}`,
        resourceKey,
        node.name,
        `${kind === 'label' ? '标签' : '污点'}${mode === 'upsert' ? '更新' : '删除'}`,
        runner,
      );
      if (result.state !== 'approval_required') {
        nodeLabelForm.resetFields();
        nodeTaintForm.resetFields();
      }
    } catch {
      // handled by executeClusterOperation
    } finally {
      setNodeMetadataLoadingKey('');
    }
  }, [clusterId, executeClusterOperation, nodeLabelForm, nodeTaintForm]);

  const renderFeedback = useCallback((feedbackKey: string) => {
    const feedback = operationFeedback[feedbackKey];
    if (!feedback) {
      return null;
    }

    return (
      <Space direction="vertical" size={2}>
        <Space size={6} wrap>
          <Tag color={
            feedback.state === 'completed'
              ? 'green'
              : feedback.state === 'approval_required'
                ? 'orange'
                : 'red'
          }>
            {feedback.action}
          </Tag>
          {feedback.audit_id && (
            <Link to={buildOperationLink(feedback.audit_id)}>
              审计
            </Link>
          )}
        </Space>
        <Text type={feedback.state === 'completed' ? 'secondary' : 'danger'}>
          {feedback.message}
        </Text>
        {feedback.guidance && (
          <Text type="secondary">
            处置建议：{feedback.guidance.summary}{' '}
            <a href={feedback.guidance.href} target="_blank" rel="noopener noreferrer">
              查看运行手册
            </a>
          </Text>
        )}
      </Space>
    );
  }, [buildOperationLink, operationFeedback]);

  const executeWorkloadOperation = useCallback(async (
    actionKey: string,
    feedbackKey: string,
    actionLabel: string,
    runner: (approvalToken?: string) => Promise<ClusterOperationResponse<any>>,
  ) => {
    await executeClusterOperation(
      actionKey,
      `${feedbackKey}:${actionKey}`,
      feedbackKey,
      actionLabel,
      runner,
      refreshSelectedNamespaceResources,
    );
  }, [executeClusterOperation, refreshSelectedNamespaceResources]);

  const openScaleOperation = useCallback((
    title: string,
    actionKey: string,
    feedbackKey: string,
    currentReplicas: number,
    execute: (replicas: number, approvalToken?: string) => Promise<ClusterOperationResponse<any>>,
  ) => {
    setPendingScaleOperation({
      title,
      actionKey,
      loadingKey: `${feedbackKey}:${actionKey}`,
      feedbackKey,
      currentReplicas,
      execute,
    });
    scaleForm.setFieldsValue({ replicas: currentReplicas });
    setScaleModalVisible(true);
  }, [scaleForm]);

  const submitScaleOperation = useCallback(async () => {
    if (!pendingScaleOperation) {return;}
    try {
      const values = await scaleForm.validateFields();
      const replicas = Number(values.replicas);
      await executeClusterOperation(
        pendingScaleOperation.actionKey,
        pendingScaleOperation.loadingKey,
        pendingScaleOperation.feedbackKey,
        pendingScaleOperation.title,
        (approvalToken) => pendingScaleOperation.execute(replicas, approvalToken),
        refreshSelectedNamespaceResources,
      );
      setScaleModalVisible(false);
      setPendingScaleOperation(null);
      scaleForm.resetFields();
    } catch {
      // executeClusterOperation already handles user-visible feedback
    }
  }, [executeClusterOperation, pendingScaleOperation, refreshSelectedNamespaceResources, scaleForm]);

  const buildServicePayload = useCallback((values: {
    name: string;
    type: string;
    selector_text: string;
    port: number;
    target_port: string;
    protocol?: string;
    node_port?: number | null;
  }, approvalToken?: string): ClusterServiceMutationPayload => {
    const selector = String(values.selector_text || '')
      .split(',')
      .map((entry) => entry.trim())
      .filter(Boolean)
      .reduce<Record<string, string>>((acc, entry) => {
        const [key, ...rest] = entry.split('=');
        const trimmedKey = key?.trim();
        const trimmedValue = rest.join('=').trim();
        if (trimmedKey && trimmedValue) {
          acc[trimmedKey] = trimmedValue;
        }
        return acc;
      }, {});
    const nodePort = typeof values.node_port === 'number' && !Number.isNaN(values.node_port) ? values.node_port : undefined;
    return {
      name: String(values.name || '').trim(),
      type: values.type,
      selector,
      ports: [{
        name: 'http',
        port: Number(values.port),
        target_port: String(values.target_port || '').trim(),
        protocol: values.protocol || 'TCP',
        node_port: nodePort,
      }],
      approval_token: approvalToken,
    };
  }, []);

  const buildIngressPayload = useCallback((values: {
    name: string;
    ingress_class_name?: string;
    host: string;
    path: string;
    path_type?: string;
    service_name: string;
    service_port: number;
    tls_secret_name?: string;
  }, approvalToken?: string): ClusterIngressMutationPayload => {
    const host = String(values.host || '').trim();
    const tlsSecretName = String(values.tls_secret_name || '').trim();
    return {
      name: String(values.name || '').trim(),
      ingress_class_name: String(values.ingress_class_name || '').trim() || undefined,
      rules: [{
        host,
        paths: [{
          path: String(values.path || '').trim(),
          path_type: values.path_type || 'Prefix',
          service_name: String(values.service_name || '').trim(),
          service_port: Number(values.service_port),
        }],
      }],
      tls: tlsSecretName ? [{ secret_name: tlsSecretName, hosts: [host] }] : undefined,
      approval_token: approvalToken,
    };
  }, []);

  const openServiceModal = useCallback((mode: 'create' | 'edit', record?: ServiceInfo) => {
    setPendingServiceModal({ mode, record });
    serviceForm.setFieldsValue({
      name: record?.name ?? '',
      type: record?.type ?? 'ClusterIP',
      selector_text: record?.selector ? Object.entries(record.selector).map(([key, value]) => `${key}=${value}`).join(',') : '',
      port: record?.ports?.[0]?.port ?? 80,
      target_port: record?.ports?.[0]?.target_port ?? '80',
      protocol: record?.ports?.[0]?.protocol ?? 'TCP',
      node_port: undefined,
    });
    setServiceModalVisible(true);
  }, [serviceForm]);

  const openIngressModal = useCallback((mode: 'create' | 'edit', record?: IngressInfo) => {
    setPendingIngressModal({ mode, record });
    ingressForm.setFieldsValue({
      name: record?.name ?? '',
      ingress_class_name: '',
      host: record?.hosts?.[0]?.host ?? '',
      path: record?.hosts?.[0]?.paths?.[0] ?? '/',
      path_type: 'Prefix',
      service_name: '',
      service_port: 80,
      tls_secret_name: '',
    });
    setIngressModalVisible(true);
  }, [ingressForm]);

  const submitServiceModal = useCallback(async () => {
    if (!clusterId) {return;}
    try {
      const values = await serviceForm.validateFields();
      const payload = buildServicePayload(values);
      const mode = pendingServiceModal?.mode ?? 'create';
      const serviceName = mode === 'edit' ? pendingServiceModal?.record?.name : payload.name;
      if (!serviceName) {
        message.error('Service 名称不能为空');
        return;
      }
      const result = await executeClusterOperation(
        mode === 'edit' ? 'service.update' : 'service.create',
        `service:${serviceName}:${mode}`,
        `service:${serviceName}`,
        mode === 'edit' ? 'Service 更新' : 'Service 创建',
        (approvalToken) => {
          const requestPayload = buildServicePayload(values, approvalToken);
          if (mode === 'edit') {
            return Api.cluster.updateService(clusterId, selectedNamespace, serviceName, requestPayload).then((resp) => resp.data);
          }
          return Api.cluster.createService(clusterId, selectedNamespace, requestPayload).then((resp) => resp.data);
        },
        refreshSelectedNamespaceResources,
      );
      if (result.state === 'completed' || result.state === 'approval_required') {
        setServiceModalVisible(false);
        setPendingServiceModal(null);
        serviceForm.resetFields();
      }
    } catch (err) {
      if (err instanceof Error) {
        message.error(err.message);
      }
    }
  }, [buildServicePayload, clusterId, executeClusterOperation, pendingServiceModal, refreshSelectedNamespaceResources, selectedNamespace, serviceForm]);

  const submitIngressModal = useCallback(async () => {
    if (!clusterId) {return;}
    try {
      const values = await ingressForm.validateFields();
      const payload = buildIngressPayload(values);
      const mode = pendingIngressModal?.mode ?? 'create';
      const ingressName = mode === 'edit' ? pendingIngressModal?.record?.name : payload.name;
      if (!ingressName) {
        message.error('Ingress 名称不能为空');
        return;
      }
      const result = await executeClusterOperation(
        mode === 'edit' ? 'ingress.update' : 'ingress.create',
        `ingress:${ingressName}:${mode}`,
        `ingress:${ingressName}`,
        mode === 'edit' ? 'Ingress 更新' : 'Ingress 创建',
        (approvalToken) => {
          const requestPayload = buildIngressPayload(values, approvalToken);
          if (mode === 'edit') {
            return Api.cluster.updateIngress(clusterId, selectedNamespace, ingressName, requestPayload).then((resp) => resp.data);
          }
          return Api.cluster.createIngress(clusterId, selectedNamespace, requestPayload).then((resp) => resp.data);
        },
        refreshSelectedNamespaceResources,
      );
      if (result.state === 'completed' || result.state === 'approval_required') {
        setIngressModalVisible(false);
        setPendingIngressModal(null);
        ingressForm.resetFields();
      }
    } catch (err) {
      if (err instanceof Error) {
        message.error(err.message);
      }
    }
  }, [buildIngressPayload, clusterId, executeClusterOperation, ingressForm, pendingIngressModal, refreshSelectedNamespaceResources, selectedNamespace]);

  const handleServiceDelete = useCallback((service: ServiceInfo) => {
    if (!clusterId) {return;}
    return executeClusterOperation(
      'service.delete',
      `service:${service.name}:delete`,
      `service:${service.name}`,
      'Service 删除',
      (approvalToken) => Api.cluster.deleteService(clusterId, selectedNamespace, service.name, { approval_token: approvalToken }).then((resp) => resp.data),
      refreshSelectedNamespaceResources,
    );
  }, [clusterId, executeClusterOperation, refreshSelectedNamespaceResources, selectedNamespace]);

  const handleIngressDelete = useCallback((ingress: IngressInfo) => {
    if (!clusterId) {return;}
    return executeClusterOperation(
      'ingress.delete',
      `ingress:${ingress.name}:delete`,
      `ingress:${ingress.name}`,
      'Ingress 删除',
      (approvalToken) => Api.cluster.deleteIngress(clusterId, selectedNamespace, ingress.name, { approval_token: approvalToken }).then((resp) => resp.data),
      refreshSelectedNamespaceResources,
    );
  }, [clusterId, executeClusterOperation, refreshSelectedNamespaceResources, selectedNamespace]);

  const handleTestConnection = async () => {
    if (!clusterId) {return;}
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

  const handleSyncNodes = syncNodes;
  const handleOpenOperationCenter = useCallback(() => {
    navigate(`/deployment/infrastructure/clusters/${clusterId}/operations`);
  }, [clusterId, navigate]);

  const handleEdit = async (values: { name: string; description: string }) => {
    if (!clusterId) {return;}
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
    if (!clusterId) {return;}
    try {
      await Api.cluster.deleteCluster(clusterId);
      message.success('集群已删除');
      navigate('/deployment/infrastructure/clusters');
    } catch (err) {
      message.error(err instanceof Error ? err.message : '删除失败');
    }
  };

  const handleAddNodes = async (values: { hostIds: string; role: string }) => {
    if (!clusterId) {return;}
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

  const handleNodeCordon = useCallback((node: ClusterNode) => {
    if (!clusterId) {return Promise.resolve();}
    return performNodeOperation('cordon', node, (approvalToken) => (
      Api.cluster.cordonNode(clusterId, node.name, { approval_token: approvalToken }).then((resp) => resp.data)
    ));
  }, [clusterId, performNodeOperation]);

  const handleNodeUncordon = useCallback((node: ClusterNode) => {
    if (!clusterId) {return Promise.resolve();}
    return performNodeOperation('uncordon', node, (approvalToken) => (
      Api.cluster.uncordonNode(clusterId, node.name, { approval_token: approvalToken }).then((resp) => resp.data)
    ));
  }, [clusterId, performNodeOperation]);

  const handleNodeDrain = useCallback((node: ClusterNode) => {
    if (!clusterId) {return Promise.resolve();}
    return performNodeOperation('drain', node, (approvalToken) => (
      Api.cluster.drainNode(clusterId, node.name, {
        approval_token: approvalToken,
        ignore_daemonsets: true,
        delete_emptydir_data: false,
        force: false,
        grace_period_seconds: 30,
        timeout_seconds: 300,
      }).then((resp) => resp.data)
    ));
  }, [clusterId, performNodeOperation]);

  const handleNodeRemove = useCallback((node: ClusterNode) => {
    if (!clusterId) {return Promise.resolve();}
    return performNodeOperation('remove', node, (approvalToken) => (
      Api.cluster.removeClusterNode(clusterId, node.name, { approval_token: approvalToken }).then((resp) => resp.data)
    ));
  }, [clusterId, performNodeOperation]);

  const handleRenewCertificates = useCallback(() => {
    if (!clusterId) {return;}
    return executeClusterOperation(
      'renew-certificates',
      'cluster:certificates',
      'cluster:certificates',
      '证书续期',
      (approvalToken) => Api.cluster.renewCertificates(clusterId, { approval_token: approvalToken }).then((resp) => resp.data),
    );
  }, [clusterId, executeClusterOperation]);

  const handleClusterUpgrade = useCallback(() => {
    if (!clusterId || !upgradePlan) {return;}
    const currentParts = upgradePlan.current_version.replace('v', '').split('.');
    const nextMinor = parseInt(currentParts[1], 10) + 1;
    const targetVersion = `${currentParts[0]}.${nextMinor}.0`;

    return executeClusterOperation(
      'upgrade',
      `cluster:upgrade:${targetVersion}`,
      'cluster:upgrade',
      '集群升级',
      (approvalToken) => Api.cluster.upgradeCluster(clusterId, {
        target_version: targetVersion,
        approval_token: approvalToken,
      }).then((resp) => resp.data),
    );
  }, [clusterId, executeClusterOperation, upgradePlan]);

  const handleDeploymentRestart = useCallback((deployment: DeploymentInfo) => {
    if (!clusterId) {return Promise.resolve();}
    return executeWorkloadOperation(
      'deployment.restart',
      `deployment:${deployment.name}`,
      'Deployment 重启',
      (approvalToken) => Api.cluster.restartDeployment(clusterId, selectedNamespace, deployment.name, { approval_token: approvalToken }).then((resp) => resp.data),
    );
  }, [clusterId, executeWorkloadOperation, selectedNamespace]);

  const handleDeploymentScale = useCallback((deployment: DeploymentInfo) => {
    if (!clusterId) {return;}
    openScaleOperation(
      'Deployment 扩缩容',
      'deployment.scale',
      `deployment:${deployment.name}`,
      deployment.replicas,
      (replicas, approvalToken) => Api.cluster.scaleDeployment(clusterId, selectedNamespace, deployment.name, {
        replicas,
        approval_token: approvalToken,
      }).then((resp) => resp.data),
    );
  }, [clusterId, openScaleOperation, selectedNamespace]);

  const handleDeploymentDelete = useCallback((deployment: DeploymentInfo) => {
    if (!clusterId) {return;}
    return executeWorkloadOperation(
      'deployment.delete',
      `deployment:${deployment.name}`,
      'Deployment 删除',
      (approvalToken) => Api.cluster.deleteDeployment(clusterId, selectedNamespace, deployment.name, { approval_token: approvalToken }).then((resp) => resp.data),
    );
  }, [clusterId, executeWorkloadOperation, selectedNamespace]);

  const handleStatefulSetRestart = useCallback((statefulset: StatefulSetInfo) => {
    if (!clusterId) {return Promise.resolve();}
    return executeWorkloadOperation(
      'statefulset.restart',
      `statefulset:${statefulset.name}`,
      'StatefulSet 重启',
      (approvalToken) => Api.cluster.restartStatefulSet(clusterId, selectedNamespace, statefulset.name, { approval_token: approvalToken }).then((resp) => resp.data),
    );
  }, [clusterId, executeWorkloadOperation, selectedNamespace]);

  const handleStatefulSetScale = useCallback((statefulset: StatefulSetInfo) => {
    if (!clusterId) {return;}
    openScaleOperation(
      'StatefulSet 扩缩容',
      'statefulset.scale',
      `statefulset:${statefulset.name}`,
      statefulset.replicas,
      (replicas, approvalToken) => Api.cluster.scaleStatefulSet(clusterId, selectedNamespace, statefulset.name, {
        replicas,
        approval_token: approvalToken,
      }).then((resp) => resp.data),
    );
  }, [clusterId, openScaleOperation, selectedNamespace]);

  const handleStatefulSetDelete = useCallback((statefulset: StatefulSetInfo) => {
    if (!clusterId) {return;}
    return executeWorkloadOperation(
      'statefulset.delete',
      `statefulset:${statefulset.name}`,
      'StatefulSet 删除',
      (approvalToken) => Api.cluster.deleteStatefulSet(clusterId, selectedNamespace, statefulset.name, { approval_token: approvalToken }).then((resp) => resp.data),
    );
  }, [clusterId, executeWorkloadOperation, selectedNamespace]);

  const handlePodDelete = useCallback((pod: PodInfo) => {
    if (!clusterId) {return;}
    return executeWorkloadOperation(
      'pod.delete',
      `pod:${pod.name}`,
      'Pod 删除',
      (approvalToken) => Api.cluster.deletePod(clusterId, selectedNamespace, pod.name, { approval_token: approvalToken }).then((resp) => resp.data),
    );
  }, [clusterId, executeWorkloadOperation, selectedNamespace]);

  const getStatusColor = (status: string) => {
    const statusMap: Record<string, string> = {
      active: 'success', inactive: 'default', error: 'error', provisioning: 'processing',
    };
    return statusMap[status] || 'default';
  };

  const getNodeStatusBadge = (status: string) => {
    if (status === 'ready') {return <Badge status="success" text="Ready" />;}
    if (status === 'notready') {return <Badge status="error" text="NotReady" />;}
    return <Badge status="warning" text="Unknown" />;
  };

  const runNodeMenuAction = useCallback((record: ClusterNode, key: string) => {
    if (key === 'cordon') {return void handleNodeCordon(record);}
    if (key === 'uncordon') {return void handleNodeUncordon(record);}
    if (key === 'drain') {return void handleNodeDrain(record);}
    if (key === 'remove') {
      Modal.confirm({
        title: '移除节点',
        content: `确定要移除节点 ${record.name} 吗？此操作可能影响工作负载调度。`,
        okText: '确定',
        cancelText: '取消',
        okButtonProps: { danger: true },
        onOk: () => handleNodeRemove(record),
      });
    }
  }, [handleNodeCordon, handleNodeUncordon, handleNodeDrain, handleNodeRemove]);

  const nodeOperationMenuItems = (record: ClusterNode) => ([
    {
      key: 'cordon',
      label: 'Cordon',
      icon: <SafetyOutlined />,
    },
    {
      key: 'uncordon',
      label: 'Uncordon',
    },
    {
      key: 'drain',
      label: 'Drain',
      icon: <AuditOutlined />,
    },
    {
      key: 'remove',
      label: 'Remove',
      danger: true,
      icon: <DeleteOutlined />,
    },
  ]);

  // Table columns
  const nodeColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: ClusterNode) => (
        <Space direction="vertical" size={0}>
          <Button type="link" className="p-0 h-auto" onClick={() => { setSelectedNode(record); setNodeDrawerVisible(true); }}>{name}</Button>
          {renderFeedback(record.name)}
        </Space>
      ),
    },
    { title: 'IP', dataIndex: 'ip', key: 'ip' },
    { title: '角色', dataIndex: 'role', key: 'role', render: (role: string) => <Tag color={role === 'control-plane' ? 'blue' : 'green'}>{role}</Tag> },
    { title: '状态', dataIndex: 'status', key: 'status', render: (status: string) => getNodeStatusBadge(status) },
    { title: 'Kubelet', dataIndex: 'kubelet_version', key: 'kubelet_version' },
    { title: '容器运行时', dataIndex: 'container_runtime', key: 'container_runtime', render: (r: string) => r?.split('/')[0] || '-' },
    { title: 'CPU/内存', key: 'resources', render: (_: any, r: ClusterNode) => <span>{r.allocatable_cpu || '-'} / {r.allocatable_mem || '-'}</span> },
    {
      title: '操作', key: 'actions', width: 180,
      render: (_: any, record: ClusterNode) => (
        <Space direction="vertical" size={4}>
          <Space>
            <Tooltip title="查看详情">
              <Button
                type="link"
                size="small"
                icon={<InfoCircleOutlined />}
                onClick={() => { setSelectedNode(record); setNodeDrawerVisible(true); }}
              />
            </Tooltip>
            {cluster?.source === 'platform_managed' && record.role !== 'control-plane' && (
              <Dropdown
                trigger={['click']}
                menu={{
                  items: nodeOperationMenuItems(record),
                  onClick: ({ key }) => { runNodeMenuAction(record, key); },
                }}
              >
                <Button size="small" icon={<MoreOutlined />}>操作</Button>
              </Dropdown>
            )}
          </Space>
          {operationFeedback[record.name] && (
            <Tag color={
              operationFeedback[record.name].state === 'completed'
                ? 'green'
                : operationFeedback[record.name].state === 'approval_required'
                  ? 'orange'
                  : 'red'
            }>
              {operationFeedback[record.name].state}
            </Tag>
          )}
        </Space>
      ),
    },
  ];

  const buildWorkloadColumns = <T extends DeploymentInfo | StatefulSetInfo>(
    kind: 'deployment' | 'statefulset',
    onRestart: (record: T) => Promise<void> | void,
    onScale: (record: T) => void,
    onDelete: (record: T) => void,
  ) => [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => (
        <Space direction="vertical" size={0}>
          <Text>{name}</Text>
          {renderFeedback(`${kind}:${name}`)}
        </Space>
      ),
    },
    { title: 'Ready', key: 'ready', render: (_: any, r: T) => `${r.ready}/${r.replicas}` },
    { title: 'Age', dataIndex: 'age', key: 'age' },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: T) => (
        <Space size={4} wrap>
          <Button
            size="small"
            aria-label={`重启 ${kind === 'deployment' ? 'Deployment' : 'StatefulSet'} ${record.name}`}
            loading={nodeMutationLoadingKey === `${kind}:${record.name}:${kind}.restart`}
            onClick={() => { void onRestart(record); }}
          >
            重启
          </Button>
          <Button
            size="small"
            aria-label={`扩缩容 ${kind === 'deployment' ? 'Deployment' : 'StatefulSet'} ${record.name}`}
            loading={nodeMutationLoadingKey === `${kind}:${record.name}:${kind}.scale`}
            onClick={() => { onScale(record); }}
          >
            扩缩容
          </Button>
          <Popconfirm
            title={`确定删除此 ${kind === 'deployment' ? 'Deployment' : 'StatefulSet'}？`}
            okText="确定"
            cancelText="取消"
            okButtonProps={{ danger: true }}
            onConfirm={() => { void onDelete(record); }}
          >
            <Button
              size="small"
              danger
              aria-label={`删除 ${kind === 'deployment' ? 'Deployment' : 'StatefulSet'} ${record.name}`}
              loading={nodeMutationLoadingKey === `${kind}:${record.name}:${kind}.delete`}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const podColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => (
        <Space direction="vertical" size={0}>
          <Text>{name}</Text>
          {renderFeedback(`pod:${name}`)}
        </Space>
      ),
    },
    { title: '状态', dataIndex: 'status', key: 'status', render: (s: string) => <Tag color={s === 'Running' ? 'green' : 'blue'}>{s}</Tag> },
    { title: 'Ready', dataIndex: 'ready', key: 'ready' },
    { title: '节点', dataIndex: 'node_name', key: 'node_name' },
    { title: 'IP', dataIndex: 'pod_ip', key: 'pod_ip' },
    { title: 'Age', dataIndex: 'age', key: 'age' },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: PodInfo) => (
        <Popconfirm
          title="确定删除此 Pod？"
          okText="确定"
          cancelText="取消"
          okButtonProps={{ danger: true }}
          onConfirm={() => { void handlePodDelete(record); }}
        >
          <Button
            size="small"
            danger
            aria-label={`删除 Pod ${record.name}`}
            loading={nodeMutationLoadingKey === `pod:${record.name}:pod.delete`}
          >
            删除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  const serviceColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => (
        <Space direction="vertical" size={0}>
          <Text>{name}</Text>
          {renderFeedback(`service:${name}`)}
        </Space>
      ),
    },
    { title: '类型', dataIndex: 'type', key: 'type' },
    { title: 'ClusterIP', dataIndex: 'cluster_ip', key: 'cluster_ip' },
    { title: '端口', key: 'ports', render: (_: any, r: ServiceInfo) => r.ports?.map(p => `${p.port}:${p.target_port}`).join(', ') || '-' },
    { title: 'Age', dataIndex: 'age', key: 'age' },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: ServiceInfo) => (
        <Space size={4} wrap>
          <Button size="small" aria-label={`编辑 Service ${record.name}`} onClick={() => { openServiceModal('edit', record); }}>
            编辑
          </Button>
          <Popconfirm
            title="确定删除此 Service？"
            okText="确定"
            cancelText="取消"
            okButtonProps={{ danger: true }}
            onConfirm={() => { void handleServiceDelete(record); }}
          >
            <Button size="small" danger aria-label={`删除 Service ${record.name}`}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const ingressColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => (
        <Space direction="vertical" size={0}>
          <Text>{name}</Text>
          {renderFeedback(`ingress:${name}`)}
        </Space>
      ),
    },
    {
      title: 'Hosts',
      key: 'hosts',
      render: (_: any, record: IngressInfo) => record.hosts?.map((host) => host.host).join(', ') || '-',
    },
    {
      title: '路径',
      key: 'paths',
      render: (_: any, record: IngressInfo) => record.hosts?.flatMap((host) => host.paths || []).join(', ') || '-',
    },
    { title: 'Age', dataIndex: 'age', key: 'age' },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: IngressInfo) => (
        <Space size={4} wrap>
          <Button size="small" aria-label={`编辑 Ingress ${record.name}`} onClick={() => { openIngressModal('edit', record); }}>
            编辑
          </Button>
          <Popconfirm
            title="确定删除此 Ingress？"
            okText="确定"
            cancelText="取消"
            okButtonProps={{ danger: true }}
            onConfirm={() => { void handleIngressDelete(record); }}
          >
            <Button size="small" danger aria-label={`删除 Ingress ${record.name}`}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
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

  return {
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
    setSelectedNode,
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
    handlePodDelete,
  };
}
