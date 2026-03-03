import React, { useState, useEffect, useCallback } from 'react';
import { Steps, Form, Input, Select, Button, Card, Space, message, Spin, Result, Alert, Descriptions, Progress, Tag } from 'antd';
import { ArrowLeftOutlined, CheckCircleOutlined, CloseCircleOutlined, LoadingOutlined, SyncOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../../api';
import type { BootstrapProfile, BootstrapTask, BootstrapVersionItem, BootstrapValidationIssue } from '../../../api/modules/cluster';
import type { Host } from '../../../api/modules/hosts';

const { TextArea } = Input;

interface BootstrapFormData {
  name: string;
  profile_id?: number;
  control_plane_host_id?: number;
  worker_host_ids?: number[];
  k8s_version?: string;
  version_channel?: string;
  cni?: string;
  pod_cidr?: string;
  service_cidr?: string;
  repo_mode?: 'online' | 'mirror';
  repo_url?: string;
  image_repository?: string;
  endpoint_mode?: 'nodeIP' | 'vip' | 'lbDNS';
  control_plane_endpoint?: string;
  vip_provider?: 'kube-vip' | 'keepalived';
  etcd_mode?: 'stacked' | 'external';
  external_etcd?: {
    endpoints?: string[];
    ca_cert?: string;
    cert?: string;
    key?: string;
  };
}

interface HostOption {
  id: number;
  name: string;
  ip: string;
}

const ClusterBootstrapWizard: React.FC = () => {
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [currentStep, setCurrentStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [formData, setFormData] = useState<BootstrapFormData>({
    name: '',
    cni: 'calico',
    k8s_version: '1.28.0',
    version_channel: 'stable-1',
    pod_cidr: '10.244.0.0/16',
    service_cidr: '10.96.0.0/12',
    repo_mode: 'online',
    endpoint_mode: 'nodeIP',
    etcd_mode: 'stacked',
  });
  const [versionOptions, setVersionOptions] = useState<BootstrapVersionItem[]>([
    { version: '1.28.0', channel: 'local-supported', status: 'supported' },
  ]);
  const [profiles, setProfiles] = useState<BootstrapProfile[]>([]);
  const [hosts, setHosts] = useState<HostOption[]>([]);
  const [previewData, setPreviewData] = useState<{
    steps: string[];
    expected_endpoint: string;
    warnings?: string[];
    validation_issues?: BootstrapValidationIssue[];
    diagnostics?: Record<string, unknown>;
  } | null>(null);
  const [taskId, setTaskId] = useState<string | null>(null);
  const [taskStatus, setTaskStatus] = useState<BootstrapTask | null>(null);
  const [clusterId, setClusterId] = useState<number | null>(null);
  const watchedName = Form.useWatch('name', form);
  const watchedControlPlaneHostId = Form.useWatch('control_plane_host_id', form);
  const watchedK8sVersion = Form.useWatch('k8s_version', form);
  const watchedCni = Form.useWatch('cni', form);
  const watchedPodCidr = Form.useWatch('pod_cidr', form);
  const watchedServiceCidr = Form.useWatch('service_cidr', form);

  useEffect(() => {
    loadHosts();
    loadBootstrapVersions();
    loadBootstrapProfiles();
  }, []);

  // Poll task status when taskId is set
  useEffect(() => {
    if (!taskId) return;

    const pollInterval = setInterval(async () => {
      try {
        const res = await Api.cluster.getBootstrapTask(taskId);
        setTaskStatus(res.data);

        if (res.data.cluster_id) {
          setClusterId(res.data.cluster_id);
        }

        if (res.data.status !== 'running' && res.data.status !== 'queued') {
          clearInterval(pollInterval);
        }
      } catch (err) {
        console.error('Failed to poll task status:', err);
      }
    }, 2000);

    return () => clearInterval(pollInterval);
  }, [taskId]);

  const loadHosts = async () => {
    try {
      const res = await Api.hosts.getHostList();
      // Convert host id from string to number for API compatibility
      const hostOptions: HostOption[] = (res.data.list || []).map((h: Host) => ({
        id: Number(h.id),
        name: h.name,
        ip: h.ip,
      }));
      setHosts(hostOptions);
    } catch (err) {
      message.error('加载主机列表失败');
    }
  };

  const loadBootstrapVersions = async () => {
    try {
      const res = await Api.cluster.getBootstrapVersions();
      const list = res.data.list || [];
      if (list.length > 0) {
        setVersionOptions(list);
        const supported = list.find((v) => v.status === 'supported');
        if (supported) {
          form.setFieldsValue({ k8s_version: supported.version, version_channel: res.data.default_channel || 'stable-1' });
          setFormData((prev) => ({ ...prev, k8s_version: supported.version, version_channel: res.data.default_channel || 'stable-1' }));
        }
      }
    } catch {
      // keep local fallback
    }
  };

  const loadBootstrapProfiles = async () => {
    try {
      const res = await Api.cluster.getBootstrapProfiles();
      setProfiles(res.data.list || []);
    } catch {
      setProfiles([]);
    }
  };

  const handleProfileChange = (profileId?: number) => {
    if (!profileId) {
      return;
    }
    const profile = profiles.find((p) => p.id === profileId);
    if (!profile) {
      return;
    }
    form.setFieldsValue({
      profile_id: profile.id,
      version_channel: profile.version_channel,
      k8s_version: profile.k8s_version || form.getFieldValue('k8s_version'),
      repo_mode: profile.repo_mode,
      repo_url: profile.repo_url,
      image_repository: profile.image_repository,
      endpoint_mode: profile.endpoint_mode,
      control_plane_endpoint: profile.control_plane_endpoint,
      vip_provider: profile.vip_provider,
      etcd_mode: profile.etcd_mode,
      external_etcd_endpoints: profile.external_etcd?.endpoints?.join(','),
      external_etcd_ca_cert: profile.external_etcd?.ca_cert,
      external_etcd_cert: profile.external_etcd?.cert,
      external_etcd_key: profile.external_etcd?.key,
    });
  };

  const handleNext = async () => {
    try {
      await form.validateFields();
      const values = form.getFieldsValue();
      setFormData({ ...formData, ...values });

      if (currentStep === 3) {
        // Preview step - load preview data
        await loadPreview();
      }

      setCurrentStep(currentStep + 1);
    } catch (err) {
      // Validation failed
    }
  };

  const handlePrev = () => {
    setCurrentStep(currentStep - 1);
  };

  const loadPreview = async () => {
    setPreviewLoading(true);
    try {
      const values = form.getFieldsValue();
      const finalData = { ...formData, ...values };
      const externalEtcd = {
        endpoints: (values.external_etcd_endpoints || '').split(',').map((x: string) => x.trim()).filter(Boolean),
        ca_cert: values.external_etcd_ca_cert,
        cert: values.external_etcd_cert,
        key: values.external_etcd_key,
      };

      const res = await Api.cluster.previewBootstrap({
        name: finalData.name,
        profile_id: finalData.profile_id,
        control_plane_host_id: finalData.control_plane_host_id!,
        worker_host_ids: finalData.worker_host_ids || [],
        k8s_version: finalData.k8s_version,
        version_channel: finalData.version_channel,
        cni: finalData.cni,
        pod_cidr: finalData.pod_cidr,
        service_cidr: finalData.service_cidr,
        repo_mode: finalData.repo_mode,
        repo_url: finalData.repo_url,
        image_repository: finalData.image_repository,
        endpoint_mode: finalData.endpoint_mode,
        control_plane_endpoint: finalData.control_plane_endpoint,
        vip_provider: finalData.vip_provider,
        etcd_mode: finalData.etcd_mode,
        external_etcd: externalEtcd,
      });

      setPreviewData({
        steps: res.data.steps,
        expected_endpoint: res.data.expected_endpoint,
        warnings: res.data.warnings,
        validation_issues: res.data.validation_issues,
        diagnostics: res.data.diagnostics,
      });
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载预览失败');
    } finally {
      setPreviewLoading(false);
    }
  };

  const handleSubmit = async () => {
    try {
      setLoading(true);
      const values = form.getFieldsValue();
      const externalEtcd = {
        endpoints: (values.external_etcd_endpoints || '').split(',').map((x: string) => x.trim()).filter(Boolean),
        ca_cert: values.external_etcd_ca_cert,
        cert: values.external_etcd_cert,
        key: values.external_etcd_key,
      };
      const finalData = { ...formData, ...values };
      finalData.external_etcd = externalEtcd;

      const res = await Api.cluster.applyBootstrap({
        name: finalData.name,
        profile_id: finalData.profile_id,
        control_plane_host_id: finalData.control_plane_host_id!,
        worker_host_ids: finalData.worker_host_ids || [],
        k8s_version: finalData.k8s_version,
        version_channel: finalData.version_channel,
        cni: finalData.cni,
        pod_cidr: finalData.pod_cidr,
        service_cidr: finalData.service_cidr,
        repo_mode: finalData.repo_mode,
        repo_url: finalData.repo_url,
        image_repository: finalData.image_repository,
        endpoint_mode: finalData.endpoint_mode,
        control_plane_endpoint: finalData.control_plane_endpoint,
        vip_provider: finalData.vip_provider,
        etcd_mode: finalData.etcd_mode,
        external_etcd: externalEtcd,
      });

      setTaskId(res.data.task_id);
      setCurrentStep(5); // Move to execution progress step
      message.success('集群创建任务已提交');
    } catch (err) {
      message.error(err instanceof Error ? err.message : '创建集群失败');
    } finally {
      setLoading(false);
    }
  };

  const getStepStatus = (status: string) => {
    switch (status) {
      case 'succeeded':
        return { icon: <CheckCircleOutlined />, status: 'finish', color: '#52c41a' };
      case 'failed':
        return { icon: <CloseCircleOutlined />, status: 'error', color: '#ff4d4f' };
      case 'running':
        return { icon: <LoadingOutlined />, status: 'process', color: '#1890ff' };
      default:
        return { icon: <SyncOutlined />, status: 'wait', color: '#d9d9d9' };
    }
  };

  const renderStep0 = () => (
    <Card title="基本信息">
      <Form form={form} layout="vertical">
        <Form.Item
          name="name"
          label="集群名称"
          rules={[{ required: true, message: '请输入集群名称' }]}
          initialValue={formData.name}
        >
          <Input placeholder="例如: production-k8s-cluster" />
        </Form.Item>
        <Form.Item name="description" label="描述">
          <TextArea rows={2} placeholder="集群描述（可选）" />
        </Form.Item>
      </Form>
    </Card>
  );

  const renderStep1 = () => (
    <Card title="选择 Control Plane 节点">
      <Form form={form} layout="vertical">
        <Form.Item
          name="control_plane_host_id"
          label="Control Plane 主机"
          rules={[{ required: true, message: '请选择 Control Plane 主机' }]}
          initialValue={formData.control_plane_host_id}
        >
          <Select
            placeholder="选择一台主机作为 Control Plane"
            showSearch
            filterOption={(input, option) =>
              (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
            }
            options={hosts.map((h) => ({
              label: `${h.name} (${h.ip})`,
              value: h.id,
            }))}
          />
        </Form.Item>
        <Alert
          type="info"
          message="提示"
          description="Control Plane 节点将运行 Kubernetes 控制平面组件（API Server, Controller Manager, Scheduler, etcd）。建议选择资源充足的主机（至少 2核4G）。"
          showIcon
        />
      </Form>
    </Card>
  );

  const renderStep2 = () => (
    <Card title="选择 Worker 节点">
      <Form form={form} layout="vertical">
        <Form.Item
          name="worker_host_ids"
          label="Worker 主机"
          initialValue={formData.worker_host_ids}
        >
          <Select
            mode="multiple"
            placeholder="选择一个或多个主机作为 Worker 节点（可选）"
            showSearch
            filterOption={(input, option) =>
              (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
            }
            options={hosts.map((h) => ({
              label: `${h.name} (${h.ip})`,
              value: h.id,
            }))}
          />
        </Form.Item>
        <Alert
          type="info"
          message="提示"
          description="Worker 节点将运行应用工作负载。可以稍后添加更多 Worker 节点。"
          showIcon
        />
      </Form>
    </Card>
  );

  const renderStep3 = () => (
    <Card title="网络配置">
      <Form form={form} layout="vertical">
        <Form.Item name="profile_id" label="Bootstrap Profile（可选）" initialValue={formData.profile_id}>
          <Select
            allowClear
            placeholder="选择已有 profile 自动填充高级参数"
            options={profiles.map((p) => ({ label: `${p.name} (${p.repo_mode}/${p.endpoint_mode}/${p.etcd_mode})`, value: p.id }))}
            onChange={handleProfileChange}
          />
        </Form.Item>
        <Form.Item
          name="k8s_version"
          label="Kubernetes 版本"
          rules={[{ required: true }]}
          initialValue={formData.k8s_version}
        >
          <Select
            options={versionOptions.map((v) => ({
              label: `${v.version}${v.status === 'supported' ? ' (支持)' : ' (阻止)'}`,
              value: v.version,
              disabled: v.status !== 'supported',
            }))}
          />
        </Form.Item>
        <Form.Item
          name="version_channel"
          label="版本通道"
          initialValue={formData.version_channel}
        >
          <Select
            options={[
              { label: 'stable-1 (默认)', value: 'stable-1' },
              { label: 'stable', value: 'stable' },
            ]}
          />
        </Form.Item>
        <Form.Item
          name="cni"
          label="CNI 网络插件"
          rules={[{ required: true }]}
          initialValue={formData.cni}
        >
          <Select
            options={[
              { label: 'Calico (推荐生产环境)', value: 'calico' },
              { label: 'Flannel (简单易用)', value: 'flannel' },
              { label: 'Cilium (高性能，支持 eBPF)', value: 'cilium' },
            ]}
          />
        </Form.Item>
        <Form.Item
          name="pod_cidr"
          label="Pod CIDR"
          rules={[{ required: true }]}
          initialValue={formData.pod_cidr}
        >
          <Input placeholder="10.244.0.0/16" />
        </Form.Item>
        <Form.Item
          name="service_cidr"
          label="Service CIDR"
          rules={[{ required: true }]}
          initialValue={formData.service_cidr}
        >
          <Input placeholder="10.96.0.0/12" />
        </Form.Item>
        <Alert
          className="mb-4"
          type="info"
          message="高级配置"
          description="可按需配置弱离线镜像/仓库、VIP 入口和 etcd 模式。"
          showIcon
        />
        <Form.Item name="repo_mode" label="安装源模式" initialValue={formData.repo_mode}>
          <Select
            options={[
              { label: '在线 (online)', value: 'online' },
              { label: '内网镜像 (mirror)', value: 'mirror' },
            ]}
          />
        </Form.Item>
        <Form.Item name="repo_url" label="内网包仓地址（mirror 时建议）" initialValue={formData.repo_url}>
          <Input placeholder="例如: https://apt-mirror.local/kubernetes" />
        </Form.Item>
        <Form.Item name="image_repository" label="镜像仓库" initialValue={formData.image_repository}>
          <Input placeholder="例如: registry.aliyuncs.com/google_containers 或 registry.local/k8s" />
        </Form.Item>
        <Form.Item name="endpoint_mode" label="控制平面入口模式" initialValue={formData.endpoint_mode}>
          <Select
            options={[
              { label: 'nodeIP', value: 'nodeIP' },
              { label: 'vip', value: 'vip' },
              { label: 'lbDNS', value: 'lbDNS' },
            ]}
          />
        </Form.Item>
        <Form.Item name="control_plane_endpoint" label="Control Plane Endpoint" initialValue={formData.control_plane_endpoint}>
          <Input placeholder="例如: 10.0.0.10:6443 或 k8s-api.example.com:6443" />
        </Form.Item>
        <Form.Item name="vip_provider" label="VIP Provider" initialValue={formData.vip_provider}>
          <Select
            options={[
              { label: 'kube-vip', value: 'kube-vip' },
              { label: 'keepalived', value: 'keepalived' },
            ]}
          />
        </Form.Item>
        <Form.Item name="etcd_mode" label="etcd 模式" initialValue={formData.etcd_mode}>
          <Select
            options={[
              { label: 'stacked', value: 'stacked' },
              { label: 'external', value: 'external' },
            ]}
          />
        </Form.Item>
        <Form.Item name="external_etcd_endpoints" label="external etcd endpoints（逗号分隔）">
          <Input placeholder="https://10.0.0.21:2379,https://10.0.0.22:2379" />
        </Form.Item>
        <Form.Item name="external_etcd_ca_cert" label="external etcd CA cert (PEM)">
          <TextArea rows={2} />
        </Form.Item>
        <Form.Item name="external_etcd_cert" label="external etcd client cert (PEM)">
          <TextArea rows={2} />
        </Form.Item>
        <Form.Item name="external_etcd_key" label="external etcd client key (PEM)">
          <TextArea rows={2} />
        </Form.Item>
        <Alert
          type="info"
          message="网络配置说明"
          description={
            <div>
              <p><strong>Pod CIDR:</strong> Pod 网络地址范围，不能与主机网络重叠</p>
              <p><strong>Service CIDR:</strong> Service ClusterIP 地址范围</p>
              <p><strong>CNI:</strong> Calico 功能丰富支持网络策略；Flannel 简单易用；Cilium 高性能</p>
            </div>
          }
        />
      </Form>
    </Card>
  );

  const renderStep4 = () => (
    <Card title="确认配置" loading={previewLoading}>
      <Descriptions column={2} bordered size="small">
        <Descriptions.Item label="集群名称">{formData.name}</Descriptions.Item>
        <Descriptions.Item label="K8s 版本">{formData.k8s_version}</Descriptions.Item>
        <Descriptions.Item label="版本通道">{formData.version_channel || '-'}</Descriptions.Item>
        <Descriptions.Item label="Repo 模式">{formData.repo_mode || '-'}</Descriptions.Item>
        <Descriptions.Item label="镜像仓库">{formData.image_repository || '-'}</Descriptions.Item>
        <Descriptions.Item label="入口模式">{formData.endpoint_mode || '-'}</Descriptions.Item>
        <Descriptions.Item label="控制平面入口">{formData.control_plane_endpoint || '-'}</Descriptions.Item>
        <Descriptions.Item label="VIP Provider">{formData.vip_provider || '-'}</Descriptions.Item>
        <Descriptions.Item label="etcd 模式">{formData.etcd_mode || '-'}</Descriptions.Item>
        <Descriptions.Item label="CNI 插件">
          <Tag color="blue">{formData.cni}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="Pod CIDR">{formData.pod_cidr}</Descriptions.Item>
        <Descriptions.Item label="Service CIDR">{formData.service_cidr}</Descriptions.Item>
        <Descriptions.Item label="API 地址">{previewData?.expected_endpoint || '-'}</Descriptions.Item>
      </Descriptions>

      {previewData?.steps && (
        <div className="mt-4">
          <h4 className="font-semibold mb-2">安装步骤预览:</h4>
          <ol className="list-decimal pl-6 space-y-1">
            {previewData.steps.map((step, index) => (
              <li key={index} className="text-gray-700">{step}</li>
            ))}
          </ol>
        </div>
      )}
      {(previewData?.warnings || []).length > 0 && (
        <Alert
          className="mt-4"
          type="warning"
          showIcon
          message="预检告警"
          description={
            <ul className="list-disc pl-4">
              {(previewData?.warnings || []).map((w) => (
                <li key={w}>{w}</li>
              ))}
            </ul>
          }
        />
      )}
      {(previewData?.validation_issues || []).length > 0 && (
        <Alert
          className="mt-4"
          type="error"
          showIcon
          message="参数校验问题"
          description={
            <ul className="list-disc pl-4">
              {(previewData?.validation_issues || []).map((i) => (
                <li key={`${i.field}-${i.code}`}>
                  [{i.domain || 'general'}] {i.field}: {i.message} {i.remediation ? `（建议: ${i.remediation}）` : ''}
                </li>
              ))}
            </ul>
          }
        />
      )}

      <Alert
        className="mt-4"
        type="warning"
        message="注意事项"
        description="创建过程需要 5-15 分钟，期间请勿关闭页面。脚本将在选定主机上执行 kubeadm 安装。"
        showIcon
      />
    </Card>
  );

  const renderStep5 = () => (
    <Card title="执行进度">
      {taskStatus ? (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="font-semibold">任务 ID: {taskId}</span>
            <Tag color={
              taskStatus.status === 'succeeded' ? 'green' :
              taskStatus.status === 'failed' ? 'red' :
              taskStatus.status === 'running' ? 'blue' : 'default'
            }>
              {taskStatus.status}
            </Tag>
          </div>

          <Progress
            percent={
              taskStatus.status === 'succeeded' ? 100 :
              taskStatus.steps ? Math.round((taskStatus.steps.filter(s => s.status === 'succeeded').length / taskStatus.steps.length) * 100) : 0
            }
            status={
              taskStatus.status === 'failed' ? 'exception' :
              taskStatus.status === 'succeeded' ? 'success' : 'active'
            }
          />

          <div className="space-y-2">
            {taskStatus.steps?.map((step, index) => {
              const stepInfo = getStepStatus(step.status);
              return (
                <div key={index} className="flex items-center gap-3 p-2 bg-gray-50 rounded">
                  <span style={{ color: stepInfo.color }}>{stepInfo.icon}</span>
                  <span className="font-medium">{step.name}</span>
                  <Tag color={step.status === 'succeeded' ? 'green' : step.status === 'failed' ? 'red' : 'blue'}>
                    {step.status}
                  </Tag>
                  {step.message && (
                    <span className="text-gray-500 text-sm">{step.message}</span>
                  )}
                </div>
              );
            })}
          </div>

          {(taskStatus.resolved_config_json || taskStatus.diagnostics_json) && (
            <Card size="small" title="有效参数与诊断摘要">
              {taskStatus.resolved_config_json && (
                <pre className="text-xs overflow-auto bg-gray-50 p-2 rounded">{taskStatus.resolved_config_json}</pre>
              )}
              {taskStatus.diagnostics_json && (
                <pre className="text-xs overflow-auto bg-gray-50 p-2 rounded mt-2">{taskStatus.diagnostics_json}</pre>
              )}
            </Card>
          )}

          {taskStatus.error_message && (
            <Alert type="error" message="错误信息" description={taskStatus.error_message} />
          )}

          {taskStatus.status === 'succeeded' && clusterId && (
            <Result
              status="success"
              title="集群创建成功"
              subTitle={`集群 "${formData.name}" 已成功创建`}
              extra={[
                <Button type="primary" key="detail" onClick={() => navigate(`/deployment/infrastructure/clusters/${clusterId}`)}>
                  查看集群
                </Button>,
                <Button key="list" onClick={() => navigate('/deployment/infrastructure/clusters')}>
                  返回列表
                </Button>,
              ]}
            />
          )}

          {taskStatus.status === 'failed' && (
            <Result
              status="error"
              title="集群创建失败"
              subTitle={taskStatus.error_message || '请查看步骤详情了解失败原因'}
              extra={[
                <Button type="primary" key="retry" onClick={() => { setTaskId(null); setTaskStatus(null); setCurrentStep(4); }}>
                  重试
                </Button>,
                <Button key="list" onClick={() => navigate('/deployment/infrastructure/clusters')}>
                  返回列表
                </Button>,
              ]}
            />
          )}
        </div>
      ) : (
        <div className="text-center py-8">
          <Spin size="large" />
          <p className="mt-4 text-gray-500">正在初始化...</p>
        </div>
      )}
    </Card>
  );

  const steps = [
    { title: '基本信息', content: renderStep0() },
    { title: 'Control Plane', content: renderStep1() },
    { title: 'Worker 节点', content: renderStep2() },
    { title: '网络配置', content: renderStep3() },
    { title: '确认配置', content: renderStep4() },
    { title: '执行进度', content: renderStep5() },
  ];

  const canProceed = () => {
    const isFilled = (value: unknown) => {
      if (typeof value === 'string') {
        return value.trim().length > 0;
      }
      return value !== undefined && value !== null;
    };

    switch (currentStep) {
      case 0:
        return isFilled(watchedName);
      case 1:
        return isFilled(watchedControlPlaneHostId);
      case 3:
        return isFilled(watchedK8sVersion) &&
               isFilled(watchedCni) &&
               isFilled(watchedPodCidr) &&
               isFilled(watchedServiceCidr);
      default:
        return true;
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/deployment/infrastructure/clusters')}>
          返回
        </Button>
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">创建 Kubernetes 集群</h1>
          <p className="text-sm text-gray-500 mt-1">通过自动化 Bootstrap 创建新集群</p>
        </div>
      </div>

      {currentStep < 5 && (
        <Steps current={currentStep} items={steps.slice(0, 5).map(item => ({ title: item.title }))} />
      )}

      <div className="min-h-[400px]">
        {steps[currentStep].content}
      </div>

      {currentStep < 5 && (
        <div className="flex justify-between">
          <Button onClick={() => navigate('/deployment/infrastructure/clusters')}>
            取消
          </Button>
          <Space>
            {currentStep > 0 && currentStep < 5 && (
              <Button onClick={handlePrev}>
                上一步
              </Button>
            )}
            {currentStep < 4 && (
              <Button type="primary" onClick={handleNext} disabled={!canProceed()}>
                下一步
              </Button>
            )}
            {currentStep === 4 && (
              <Button type="primary" onClick={handleSubmit} loading={loading}>
                开始创建
              </Button>
            )}
          </Space>
        </div>
      )}
    </div>
  );
};

export default ClusterBootstrapWizard;
