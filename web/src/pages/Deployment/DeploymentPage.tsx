import React from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
  Timeline,
  Tooltip,
  message,
} from 'antd';
import { Api } from '../../api';
import type { Cluster } from '../../api/modules/kubernetes';
import type { Host } from '../../api/modules/hosts';
import type { ServiceItem } from '../../api/modules/services';
import type { ClusterBootstrapTask, ClusterCredential, DeployTarget, DeployRelease, DeployReleaseTimelineEvent, EnvironmentBootstrapJob, Inspection } from '../../api/modules/deployment';

const envOptions = [{ value: 'development' }, { value: 'staging' }, { value: 'production' }];
const strategyOptions = [{ value: 'rolling' }, { value: 'blue-green' }, { value: 'canary' }];
const runtimeOptions = [{ value: 'k8s' }, { value: 'compose' }];

type GovernanceForm = {
  service_id: number;
  env: string;
  traffic_policy: string;
  resilience_policy: string;
  access_policy: string;
  slo_policy: string;
};

const parseJSONMap = (raw?: string): Record<string, any> => {
  const content = (raw || '').trim();
  if (!content) return {};
  const parsed = JSON.parse(content);
  if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
    throw new Error('JSON 内容必须是对象');
  }
  return parsed;
};

const parseVariables = (raw?: string): Record<string, string> => {
  const content = (raw || '').trim();
  if (!content) return {};
  const parsed = JSON.parse(content);
  if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
    throw new Error('变量必须是 JSON 对象');
  }
  const out: Record<string, string> = {};
  Object.entries(parsed).forEach(([k, v]) => {
    out[k] = String(v ?? '');
  });
  return out;
};

const releaseStatusColor = (status?: string): string => {
  if (status === 'applied' || status === 'succeeded' || status === 'rollback' || status === 'rolled_back') return 'success';
  if (status === 'failed' || status === 'rejected') return 'error';
  if (status === 'pending_approval') return 'warning';
  return 'processing';
};

const DeploymentPage: React.FC = () => {
  const [loading, setLoading] = React.useState(false);
  const [targets, setTargets] = React.useState<DeployTarget[]>([]);
  const [releases, setReleases] = React.useState<DeployRelease[]>([]);
  const [inspections, setInspections] = React.useState<Inspection[]>([]);
  const [clusters, setClusters] = React.useState<Cluster[]>([]);
  const [hosts, setHosts] = React.useState<Host[]>([]);
  const [services, setServices] = React.useState<ServiceItem[]>([]);
  const [previewManifest, setPreviewManifest] = React.useState('');
  const [previewToken, setPreviewToken] = React.useState('');
  const [runtimeFilter, setRuntimeFilter] = React.useState<'k8s' | 'compose' | undefined>(undefined);
  const [selectedRelease, setSelectedRelease] = React.useState<DeployRelease | null>(null);
  const [previewWarnings, setPreviewWarnings] = React.useState<Array<{ code: string; message: string; level: string }>>([]);
  const [releaseTimeline, setReleaseTimeline] = React.useState<DeployReleaseTimelineEvent[]>([]);
  const [clusterModalOpen, setClusterModalOpen] = React.useState(false);
  const [bootstrapPreview, setBootstrapPreview] = React.useState<{ steps: string[]; expected_endpoint?: string } | null>(null);
  const [bootstrapTasks, setBootstrapTasks] = React.useState<ClusterBootstrapTask[]>([]);
  const [environmentJobs, setEnvironmentJobs] = React.useState<EnvironmentBootstrapJob[]>([]);
  const [credentials, setCredentials] = React.useState<ClusterCredential[]>([]);

  const [targetForm] = Form.useForm();
  const [releaseForm] = Form.useForm();
  const [clusterForm] = Form.useForm();
  const [governanceForm] = Form.useForm<GovernanceForm>();
  const [inspectionForm] = Form.useForm();
  const [bootstrapForm] = Form.useForm();
  const [environmentForm] = Form.useForm();
  const [credentialForm] = Form.useForm();

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const [tRes, rRes, iRes, cRes, hRes, sRes, credRes] = await Promise.all([
        Api.deployment.getTargets(),
        Api.deployment.getReleasesByRuntime({ runtime_type: runtimeFilter }),
        Api.deployment.listInspections(),
        Api.kubernetes.getClusterList({ page: 1, pageSize: 200 }),
        Api.hosts.getHostList({ page: 1, pageSize: 500 }),
        Api.services.getList({ page: 1, pageSize: 500 }),
        Api.deployment.listCredentials(),
      ]);
      setTargets(tRes.data.list || []);
      setReleases(rRes.data.list || []);
      setInspections(iRes.data.list || []);
      setClusters(cRes.data.list || []);
      setHosts(hRes.data.list || []);
      setServices(sRes.data.list || []);
      setCredentials(credRes.data.list || []);
      const taskId = String(localStorage.getItem('clusterBootstrapTaskId') || '').trim();
      if (taskId) {
        try {
          const task = await Api.deployment.getClusterBootstrapTask(taskId);
          setBootstrapTasks([task.data]);
        } catch {
          setBootstrapTasks([]);
        }
      } else {
        setBootstrapTasks([]);
      }
      const envTaskId = String(localStorage.getItem('environmentBootstrapJobId') || '').trim();
      if (envTaskId) {
        try {
          const task = await Api.deployment.getEnvironmentBootstrapJob(envTaskId);
          setEnvironmentJobs([task.data]);
        } catch {
          setEnvironmentJobs([]);
        }
      } else {
        setEnvironmentJobs([]);
      }
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载部署管理失败');
    } finally {
      setLoading(false);
    }
  }, [runtimeFilter]);

  React.useEffect(() => {
    void load();
  }, [load]);

  const createCluster = async () => {
    const v = await clusterForm.validateFields();
    await Api.kubernetes.createCluster({
      name: String(v.name),
      server: String(v.server),
      credential_ref: String(v.credential_ref || ''),
      description: String(v.description || ''),
    });
    message.success('K8s 集群创建成功');
    setClusterModalOpen(false);
    clusterForm.resetFields();
    await load();
  };

  const createTarget = async () => {
    const v = await targetForm.validateFields();
    const payload: any = {
      name: v.name,
      target_type: v.target_type,
      runtime_type: v.target_type,
      cluster_id: v.target_type === 'k8s' ? Number(v.cluster_id || 0) : 0,
      project_id: Number(localStorage.getItem('projectId') || 1),
      team_id: Number(localStorage.getItem('teamId') || 1),
      env: v.env || 'staging',
      nodes: (v.host_ids || []).map((id: number) => ({ host_id: Number(id), role: 'worker', weight: 100 })),
    };
    await Api.deployment.createTarget(payload);
    message.success('部署目标创建成功');
    targetForm.resetFields();
    await load();
  };

  const preview = async () => {
    const v = await releaseForm.validateFields();
    const variables = parseVariables(v.variables_json);
    const resp = await Api.deployment.previewRelease({
      service_id: Number(v.service_id),
      target_id: Number(v.target_id),
      env: v.env || 'staging',
      strategy: v.strategy || 'rolling',
      variables,
    });
    setPreviewManifest(resp.data.resolved_manifest || '');
    setPreviewToken(resp.data.preview_token || '');
    setPreviewWarnings(resp.data.warnings || []);
  };

  const apply = async () => {
    const v = await releaseForm.validateFields();
    if (!previewToken) {
      message.warning('请先执行 Preview，再执行 Apply');
      return;
    }
    const variables = parseVariables(v.variables_json);
    const resp = await Api.deployment.applyRelease({
      service_id: Number(v.service_id),
      target_id: Number(v.target_id),
      env: v.env || 'staging',
      strategy: v.strategy || 'rolling',
      variables,
      preview_token: previewToken,
    });
    if (resp.data.approval_required) {
      message.warning(`release #${resp.data.release_id} 已进入审批，ticket: ${resp.data.approval_ticket || '-'}`);
    } else {
      message.success(`发布已执行，release #${resp.data.release_id}`);
    }
    setPreviewToken('');
    await load();
  };

  const approveRelease = async (releaseId: number) => {
    await Api.deployment.approveRelease(releaseId, {});
    message.success(`release #${releaseId} 已审批并执行`);
    await load();
  };

  const rejectRelease = async (releaseId: number) => {
    await Api.deployment.rejectRelease(releaseId, {});
    message.success(`release #${releaseId} 已拒绝`);
    await load();
  };

  const rollback = async (releaseId: number) => {
    await Api.deployment.rollbackRelease(releaseId);
    message.success(`回滚任务已提交，来源 release #${releaseId}`);
    await load();
  };

  const showReleaseDetail = async (row: DeployRelease) => {
    setSelectedRelease(row);
    try {
      const timelineResp = await Api.deployment.getReleaseTimeline(row.id);
      setReleaseTimeline(timelineResp.data.list || []);
    } catch {
      setReleaseTimeline([]);
    }
  };

  const runInspection = async (stage: 'pre' | 'post' | 'periodic', releaseId?: number) => {
    const v = inspectionForm.getFieldsValue();
    await Api.deployment.runInspection({
      release_id: releaseId || (v.release_id ? Number(v.release_id) : undefined),
      service_id: v.service_id ? Number(v.service_id) : undefined,
      target_id: v.target_id ? Number(v.target_id) : undefined,
      stage,
    });
    message.success('AIOPS 巡检已执行');
    await load();
  };

  const loadGovernance = async () => {
    const v = await governanceForm.validateFields(['service_id', 'env']);
    const resp = await Api.deployment.getGovernance(Number(v.service_id), v.env);
    const row = resp.data || {};
    governanceForm.setFieldsValue({
      traffic_policy: row.traffic_policy_json || '{}',
      resilience_policy: row.resilience_policy_json || '{}',
      access_policy: row.access_policy_json || '{}',
      slo_policy: row.slo_policy_json || '{}',
    });
    message.success('已加载治理策略');
  };

  const saveGovernance = async () => {
    const v = await governanceForm.validateFields();
    await Api.deployment.putGovernance(Number(v.service_id), {
      env: v.env,
      traffic_policy: parseJSONMap(v.traffic_policy),
      resilience_policy: parseJSONMap(v.resilience_policy),
      access_policy: parseJSONMap(v.access_policy),
      slo_policy: parseJSONMap(v.slo_policy),
    });
    message.success('治理策略保存成功');
  };

  const previewBootstrap = async () => {
    const v = await bootstrapForm.validateFields();
    const resp = await Api.deployment.previewClusterBootstrap({
      name: String(v.name),
      control_plane_host_id: Number(v.control_plane_host_id),
      worker_host_ids: (v.worker_host_ids || []).map((x: number) => Number(x)),
      cni: String(v.cni || 'flannel'),
    });
    setBootstrapPreview({ steps: resp.data.steps || [], expected_endpoint: resp.data.expected_endpoint });
    message.success('已生成建群步骤预览');
  };

  const applyBootstrap = async () => {
    const v = await bootstrapForm.validateFields();
    const resp = await Api.deployment.applyClusterBootstrap({
      name: String(v.name),
      control_plane_host_id: Number(v.control_plane_host_id),
      worker_host_ids: (v.worker_host_ids || []).map((x: number) => Number(x)),
      cni: String(v.cni || 'flannel'),
    });
    localStorage.setItem('clusterBootstrapTaskId', resp.data.task_id);
    message.success(`建群任务已启动: ${resp.data.task_id}`);
    await load();
  };

  const applyEnvironmentBootstrap = async () => {
    const v = await environmentForm.validateFields();
    const payload: any = {
      name: String(v.name),
      runtime_type: String(v.runtime_type),
      package_version: String(v.package_version),
      env: String(v.env || 'staging'),
      target_id: Number(v.target_id || 0),
    };
    if (payload.runtime_type === 'k8s') {
      payload.control_plane_host_id = Number(v.control_plane_host_id || 0);
      payload.worker_host_ids = (v.worker_host_ids || []).map((x: number) => Number(x));
    } else {
      payload.node_ids = (v.node_ids || []).map((x: number) => Number(x));
    }
    const resp = await Api.deployment.startEnvironmentBootstrap(payload);
    localStorage.setItem('environmentBootstrapJobId', resp.data.job_id);
    message.success(`环境部署任务已启动: ${resp.data.job_id}`);
    await load();
  };

  const registerPlatformCredential = async () => {
    const clusterId = Number(credentialForm.getFieldValue('platform_cluster_id') || 0);
    if (!clusterId) {
      message.warning('请选择平台集群');
      return;
    }
    await Api.deployment.registerPlatformCredential({
      cluster_id: clusterId,
      name: String(credentialForm.getFieldValue('platform_name') || ''),
      runtime_type: 'k8s',
    });
    message.success('平台凭据注册成功');
    await load();
  };

  const importExternalCredential = async () => {
    const v = await credentialForm.validateFields(['external_name', 'auth_method', 'kubeconfig', 'endpoint', 'ca_cert', 'cert', 'key', 'token']);
    const authMethod = String(v.auth_method || 'kubeconfig');
    const payload: any = {
      name: String(v.external_name),
      runtime_type: 'k8s',
      auth_method: authMethod,
      endpoint: String(v.endpoint || ''),
      kubeconfig: String(v.kubeconfig || ''),
      ca_cert: String(v.ca_cert || ''),
      cert: String(v.cert || ''),
      key: String(v.key || ''),
      token: String(v.token || ''),
    };
    await Api.deployment.importExternalCredential(payload);
    message.success('外部凭据导入成功');
    await load();
  };

  const testCredential = async (id: number) => {
    const resp = await Api.deployment.testCredential(id);
    if (resp.data.connected) {
      message.success(`凭据 #${id} 连通成功 (${resp.data.latency_ms || 0}ms)`);
    } else {
      message.warning(`凭据 #${id} 连通失败: ${resp.data.message}`);
    }
    await load();
  };

  const clusterOptions = clusters.map((c) => ({ value: Number(c.id), label: `${c.name} (#${c.id})` }));
  const hostOptions = hosts.map((h) => ({ value: Number(h.id), label: `${h.name} (${h.ip})` }));
  const serviceOptions = services.map((s) => ({ value: Number(s.id), label: `${s.name} (#${s.id})` }));
  const targetOptions = targets.map((t) => ({ value: t.id, label: `${t.name} [${t.target_type}]` }));

  return (
    <div className="space-y-4">
      <Card title="部署管理（K8s + Compose）" extra={<Button onClick={() => void load()} loading={loading}>刷新</Button>}>
        <Tabs items={[
          {
            key: 'targets',
            label: `Targets (${targets.length})`,
            children: (
              <>
                <Form form={targetForm} layout="vertical">
                  <Row gutter={12}>
                    <Col span={6}><Form.Item name="name" label="目标名称" rules={[{ required: true }]}><Input placeholder="prod-k8s / edge-compose" /></Form.Item></Col>
                    <Col span={4}><Form.Item name="target_type" label="类型" initialValue="k8s" rules={[{ required: true }]}><Select options={[{ value: 'k8s' }, { value: 'compose' }]} /></Form.Item></Col>
                    <Col span={4}><Form.Item name="env" label="环境" initialValue="staging"><Select options={envOptions} /></Form.Item></Col>
                    <Col span={10}>
                      <Form.Item shouldUpdate noStyle>
                        {({ getFieldValue }) => {
                          const targetType = getFieldValue('target_type') || 'k8s';
                          if (targetType === 'k8s') {
                            return (
                              <Form.Item name="cluster_id" label="K8s 集群" rules={[{ required: true, message: '请选择集群' }]}>
                                <Select
                                  options={clusterOptions}
                                  placeholder="选择已有 K8s 集群"
                                  popupRender={(menu) => (
                                    <div>
                                      {menu}
                                      <div style={{ padding: 8, borderTop: '1px solid #f0f0f0' }}>
                                        <Button type="link" onClick={() => setClusterModalOpen(true)}>+ 新建 K8s 集群</Button>
                                      </div>
                                    </div>
                                  )}
                                />
                              </Form.Item>
                            );
                          }
                          return (
                            <Form.Item name="host_ids" label="Compose 主机节点" rules={[{ required: true, message: '请选择至少一台主机' }]}>
                              <Select mode="multiple" options={hostOptions} placeholder="选择主机组（Host 组即 Compose Cluster）" />
                            </Form.Item>
                          );
                        }}
                      </Form.Item>
                    </Col>
                  </Row>
                </Form>
                <Space style={{ marginBottom: 12 }}>
                  <Button type="primary" onClick={() => void createTarget()}>创建部署目标</Button>
                  <Select
                    data-testid="runtime-filter"
                    allowClear
                    style={{ width: 180 }}
                    value={runtimeFilter}
                    options={runtimeOptions}
                    placeholder="筛选运行时"
                    onChange={(v) => setRuntimeFilter((v || undefined) as 'k8s' | 'compose' | undefined)}
                  />
                  <Tooltip title="Compose 目标直接复用主机管理里的主机组，K8s 目标复用集群管理能力。">
                    <Tag color="blue">主机/集群已联动</Tag>
                  </Tooltip>
                </Space>
                <Table
                  rowKey="id"
                  dataSource={targets}
                  pagination={false}
                  columns={[
                    { title: 'ID', dataIndex: 'id', width: 80 },
                    { title: '名称', dataIndex: 'name' },
                    { title: '类型', dataIndex: 'target_type', render: (v: string) => <Tag color={v === 'k8s' ? 'blue' : 'geekblue'}>{v}</Tag> },
                    { title: 'Runtime', dataIndex: 'runtime_type', render: (v: string) => <Tag color={v === 'k8s' ? 'blue' : 'purple'}>{v}</Tag> },
                    {
                      title: '目标资源',
                      render: (_: unknown, r: DeployTarget) => (r.target_type === 'k8s'
                        ? `cluster #${r.cluster_id}`
                        : (r.nodes || []).map((n) => n.name ? `${n.name}(${n.ip})` : `host#${n.host_id}`).join(', ') || '-'),
                    },
                    { title: 'Env', dataIndex: 'env', render: (v: string) => <Tag>{v}</Tag> },
                    { title: '状态', dataIndex: 'status', render: (v: string) => <Tag color={v === 'active' ? 'success' : 'default'}>{v}</Tag> },
                  ]}
                />
              </>
            ),
          },
          {
            key: 'releases',
            label: `Releases (${releases.length})`,
            children: (
              <Row gutter={16}>
                <Col span={10}>
                  <Card size="small" title="发布执行">
                    <Form form={releaseForm} layout="vertical">
                      <Form.Item name="service_id" label="服务" rules={[{ required: true }]}>
                        <Select showSearch options={serviceOptions} placeholder="选择服务" optionFilterProp="label" />
                      </Form.Item>
                      <Form.Item name="target_id" label="部署目标" rules={[{ required: true }]}>
                        <Select showSearch options={targetOptions} placeholder="选择目标" optionFilterProp="label" />
                      </Form.Item>
                      <Form.Item shouldUpdate noStyle>
                        {({ getFieldValue }) => {
                          const targetId = Number(getFieldValue('target_id') || 0);
                          const target = targets.find((t) => t.id === targetId);
                          if (!target) return null;
                          if (target.target_type === 'k8s') {
                            return <Form.Item name="variables_json" label="K8s 变量(JSON)"><Input.TextArea rows={6} placeholder='{"image_tag":"v1.2.3","replicas":"3"}' /></Form.Item>;
                          }
                          return <Form.Item name="variables_json" label="Compose 变量(JSON)"><Input.TextArea rows={6} placeholder='{"COMPOSE_PROJECT_NAME":"svc-a","IMAGE_TAG":"v1.2.3"}' /></Form.Item>;
                        }}
                      </Form.Item>
                      <Row gutter={12}>
                        <Col span={12}><Form.Item name="env" label="环境" initialValue="staging"><Select options={envOptions} /></Form.Item></Col>
                        <Col span={12}><Form.Item name="strategy" label="发布策略" initialValue="rolling"><Select options={strategyOptions} /></Form.Item></Col>
                      </Row>
                      <Space>
                        <Button onClick={() => void preview()}>Preview</Button>
                        <Button type="primary" onClick={() => void apply()}>Apply</Button>
                        <Button onClick={() => void runInspection('pre')}>AIOPS Pre-check</Button>
                      </Space>
                    </Form>
                  </Card>
                </Col>
                <Col span={14}>
                  <Card size="small" title="Preview Manifest">
                    {previewWarnings.length > 0 ? (
                      <Space orientation="vertical" style={{ width: '100%', marginBottom: 8 }}>
                        {previewWarnings.map((w, idx) => <Alert key={`${w.code}-${idx}`} type={w.level === 'error' ? 'error' : 'warning'} showIcon title={w.message} />)}
                      </Space>
                    ) : null}
                    <pre style={{ maxHeight: 280, overflow: 'auto' }}>{previewManifest || '# 暂无预览'}</pre>
                  </Card>
                </Col>
                <Col span={24} style={{ marginTop: 12 }}>
                  <Table
                    rowKey="id"
                    dataSource={releases}
                    pagination={false}
                    columns={[
                      { title: 'Release', dataIndex: 'id', width: 90 },
                      { title: 'Service', dataIndex: 'service_id' },
                      { title: 'Target', dataIndex: 'target_id' },
                      { title: 'Runtime', dataIndex: 'runtime_type', render: (v: string) => <Tag color={v === 'k8s' ? 'blue' : 'purple'}>{v}</Tag> },
                      { title: 'Strategy', dataIndex: 'strategy' },
                      { title: 'Status', dataIndex: 'status', render: (v: string) => <Tag color={releaseStatusColor(v)}>{v}</Tag> },
                      {
                        title: '诊断摘要',
                        render: (_: unknown, row: DeployRelease) => {
                          try {
                            const parsed = JSON.parse(row.diagnostics_json || '[]');
                            const first = Array.isArray(parsed) ? parsed[0] : parsed;
                            return first?.summary || '-';
                          } catch {
                            return '-';
                          }
                        },
                      },
                      { title: '创建时间', dataIndex: 'created_at', render: (v: string) => new Date(v).toLocaleString() },
                      {
                        title: '操作',
                        render: (_: unknown, row: DeployRelease) => (
                          <Space>
                            {row.status === 'pending_approval' ? (
                              <>
                                <Button size="small" type="primary" onClick={() => void approveRelease(row.id)}>Approve</Button>
                                <Button size="small" danger onClick={() => void rejectRelease(row.id)}>Reject</Button>
                              </>
                            ) : (
                              <Button size="small" onClick={() => void rollback(row.id)}>Rollback</Button>
                            )}
                            <Button size="small" onClick={() => void runInspection('post', row.id)}>AIOPS Post-check</Button>
                            <Button size="small" onClick={() => void showReleaseDetail(row)}>详情</Button>
                          </Space>
                        ),
                      },
                    ]}
                  />
                </Col>
              </Row>
            ),
          },
          {
            key: 'governance',
            label: 'Governance',
            children: (
              <Card size="small" title="服务治理策略（策略 + 流量 + SLO）">
                <Form form={governanceForm} layout="vertical" initialValues={{ env: 'staging', traffic_policy: '{}', resilience_policy: '{}', access_policy: '{}', slo_policy: '{}' }}>
                  <Row gutter={12}>
                    <Col span={8}><Form.Item name="service_id" label="服务" rules={[{ required: true }]}><Select showSearch options={serviceOptions} placeholder="选择服务" optionFilterProp="label" /></Form.Item></Col>
                    <Col span={4}><Form.Item name="env" label="环境" rules={[{ required: true }]}><Select options={envOptions} /></Form.Item></Col>
                    <Col span={12}><Space style={{ marginTop: 30 }}><Button onClick={() => void loadGovernance()}>加载策略</Button><Button type="primary" onClick={() => void saveGovernance()}>保存策略</Button></Space></Col>
                  </Row>
                  <Row gutter={12}>
                    <Col span={12}><Form.Item name="traffic_policy" label="流量策略(JSON)"><Input.TextArea rows={7} /></Form.Item></Col>
                    <Col span={12}><Form.Item name="resilience_policy" label="韧性策略(JSON)"><Input.TextArea rows={7} /></Form.Item></Col>
                    <Col span={12}><Form.Item name="access_policy" label="访问策略(JSON)"><Input.TextArea rows={7} /></Form.Item></Col>
                    <Col span={12}><Form.Item name="slo_policy" label="SLO 策略(JSON)"><Input.TextArea rows={7} /></Form.Item></Col>
                  </Row>
                </Form>
              </Card>
            ),
          },
          {
            key: 'aiops',
            label: `AIOPS (${inspections.length})`,
            children: (
              <>
                <Card size="small" title="AIOPS 主动巡检" style={{ marginBottom: 12 }}>
                  <Form form={inspectionForm} layout="inline">
                    <Form.Item name="service_id"><Select allowClear style={{ width: 220 }} showSearch options={serviceOptions} placeholder="服务" optionFilterProp="label" /></Form.Item>
                    <Form.Item name="target_id"><Select allowClear style={{ width: 220 }} showSearch options={targetOptions} placeholder="部署目标" optionFilterProp="label" /></Form.Item>
                    <Form.Item name="release_id"><InputNumber min={1} placeholder="release id" /></Form.Item>
                    <Form.Item name="stage" initialValue="periodic"><Select style={{ width: 160 }} options={[{ value: 'pre' }, { value: 'post' }, { value: 'periodic' }]} /></Form.Item>
                    <Button type="primary" onClick={() => void runInspection((inspectionForm.getFieldValue('stage') || 'periodic') as 'pre' | 'post' | 'periodic')}>运行巡检</Button>
                  </Form>
                </Card>
                <Table
                  rowKey="id"
                  dataSource={inspections}
                  pagination={false}
                  columns={[
                    { title: 'ID', dataIndex: 'id', width: 80 },
                    { title: 'Stage', dataIndex: 'stage', render: (v: string) => <Tag>{v}</Tag> },
                    { title: 'Service/Target', render: (_: unknown, r: Inspection) => `${r.service_id} / ${r.target_id}` },
                    { title: 'Summary', dataIndex: 'summary' },
                    { title: 'Status', dataIndex: 'status', render: (v: string) => <Tag color={v === 'done' ? 'success' : 'processing'}>{v}</Tag> },
                    { title: '时间', dataIndex: 'created_at', render: (v: string) => new Date(v).toLocaleString() },
                  ]}
                />
              </>
            ),
          },
          {
            key: 'bootstrap',
            label: '半自动建群',
            children: (
              <Row gutter={16}>
                <Col span={10}>
                  <Card size="small" title="节点选择与参数">
                    <Form form={bootstrapForm} layout="vertical" initialValues={{ cni: 'flannel' }}>
                      <Form.Item name="name" label="集群名称" rules={[{ required: true }]}><Input placeholder="edge-cluster-a" /></Form.Item>
                      <Form.Item name="control_plane_host_id" label="控制平面节点" rules={[{ required: true }]}>
                        <Select showSearch optionFilterProp="label" options={hostOptions} placeholder="选择一台控制平面主机" />
                      </Form.Item>
                      <Form.Item name="worker_host_ids" label="工作节点">
                        <Select mode="multiple" showSearch optionFilterProp="label" options={hostOptions} placeholder="可选，选择多台工作节点" />
                      </Form.Item>
                      <Form.Item name="cni" label="CNI">
                        <Select options={[{ value: 'flannel' }, { value: 'calico' }]} />
                      </Form.Item>
                      <Space>
                        <Button onClick={() => void previewBootstrap()}>预览步骤</Button>
                        <Button type="primary" onClick={() => void applyBootstrap()}>执行建群</Button>
                      </Space>
                    </Form>
                  </Card>
                </Col>
                <Col span={14}>
                  <Card size="small" title="预览与任务状态">
                    {bootstrapPreview ? (
                      <Alert
                        type="info"
                        showIcon
                        title={`预计 API Endpoint: ${bootstrapPreview.expected_endpoint || '-'}`}
                        description={(bootstrapPreview.steps || []).map((s, idx) => <div key={idx}>{idx + 1}. {s}</div>)}
                        style={{ marginBottom: 12 }}
                      />
                    ) : <Alert type="warning" showIcon title="尚未生成预览步骤" style={{ marginBottom: 12 }} />}
                    <Table
                      rowKey="id"
                      dataSource={bootstrapTasks}
                      pagination={false}
                      columns={[
                        { title: '任务ID', dataIndex: 'id' },
                        { title: '集群名称', dataIndex: 'name' },
                        { title: '状态', dataIndex: 'status', render: (v: string) => <Tag color={v === 'succeeded' ? 'success' : v === 'failed' ? 'error' : 'processing'}>{v}</Tag> },
                        { title: '错误', dataIndex: 'error_message', render: (v: string) => v || '-' },
                        { title: '更新时间', dataIndex: 'updated_at', render: (v: string) => (v ? new Date(v).toLocaleString() : '-') },
                      ]}
                    />
                  </Card>
                </Col>
                <Col span={24} style={{ marginTop: 12 }}>
                  <Card size="small" title="环境部署（SSH + 二进制安装）">
                    <Form form={environmentForm} layout="vertical" initialValues={{ runtime_type: 'k8s', package_version: 'v0.1.0', env: 'staging' }}>
                      <Row gutter={12}>
                        <Col span={6}><Form.Item name="name" label="任务名称" rules={[{ required: true }]}><Input placeholder="env-bootstrap-a" /></Form.Item></Col>
                        <Col span={4}><Form.Item name="runtime_type" label="Runtime" rules={[{ required: true }]}><Select options={runtimeOptions} /></Form.Item></Col>
                        <Col span={4}><Form.Item name="package_version" label="版本" rules={[{ required: true }]}><Input placeholder="v0.1.0" /></Form.Item></Col>
                        <Col span={4}><Form.Item name="env" label="环境"><Select options={envOptions} /></Form.Item></Col>
                        <Col span={6}><Form.Item name="target_id" label="绑定目标"><Select allowClear options={targetOptions} placeholder="可选" /></Form.Item></Col>
                      </Row>
                      <Form.Item shouldUpdate noStyle>
                        {({ getFieldValue }) => {
                          const runtime = getFieldValue('runtime_type') || 'k8s';
                          if (runtime === 'k8s') {
                            return (
                              <Row gutter={12}>
                                <Col span={8}><Form.Item name="control_plane_host_id" label="控制平面节点" rules={[{ required: true }]}><Select options={hostOptions} showSearch optionFilterProp="label" /></Form.Item></Col>
                                <Col span={16}><Form.Item name="worker_host_ids" label="工作节点"><Select mode="multiple" options={hostOptions} showSearch optionFilterProp="label" /></Form.Item></Col>
                              </Row>
                            );
                          }
                          return (
                            <Row gutter={12}>
                              <Col span={24}><Form.Item name="node_ids" label="Compose 节点" rules={[{ required: true }]}><Select mode="multiple" options={hostOptions} showSearch optionFilterProp="label" /></Form.Item></Col>
                            </Row>
                          );
                        }}
                      </Form.Item>
                      <Space>
                        <Button type="primary" onClick={() => void applyEnvironmentBootstrap()}>启动环境部署</Button>
                      </Space>
                    </Form>
                    <Table
                      style={{ marginTop: 12 }}
                      rowKey="id"
                      dataSource={environmentJobs}
                      pagination={false}
                      columns={[
                        { title: '任务ID', dataIndex: 'id' },
                        { title: 'Runtime', dataIndex: 'runtime_type', render: (v: string) => <Tag>{v}</Tag> },
                        { title: '版本', dataIndex: 'package_version' },
                        { title: '状态', dataIndex: 'status', render: (v: string) => <Tag color={v === 'succeeded' ? 'success' : v === 'failed' ? 'error' : 'processing'}>{v}</Tag> },
                        { title: '错误', dataIndex: 'error_message', render: (v: string) => v || '-' },
                        { title: '更新时间', dataIndex: 'updated_at', render: (v: string) => (v ? new Date(v).toLocaleString() : '-') },
                      ]}
                    />
                  </Card>
                </Col>
                <Col span={24} style={{ marginTop: 12 }}>
                  <Card size="small" title="集群凭据（平台注册 / 外部导入）">
                    <Form form={credentialForm} layout="vertical" initialValues={{ auth_method: 'kubeconfig' }}>
                      <Row gutter={12}>
                        <Col span={8}><Form.Item name="platform_cluster_id" label="平台注册集群"><Select allowClear options={clusterOptions} placeholder="选择现有集群" /></Form.Item></Col>
                        <Col span={8}><Form.Item name="platform_name" label="平台注册名称"><Input placeholder="可选，默认集群名" /></Form.Item></Col>
                        <Col span={8}><Space style={{ marginTop: 30 }}><Button onClick={() => void registerPlatformCredential()}>注册平台凭据</Button></Space></Col>
                      </Row>
                      <Row gutter={12}>
                        <Col span={6}><Form.Item name="external_name" label="外部凭据名称" rules={[{ required: true }]}><Input placeholder="external-prod-k8s" /></Form.Item></Col>
                        <Col span={4}><Form.Item name="auth_method" label="认证方式" rules={[{ required: true }]}><Select options={[{ value: 'kubeconfig' }, { value: 'cert' }]} /></Form.Item></Col>
                        <Col span={14}><Form.Item shouldUpdate noStyle>{({ getFieldValue }) => {
                          const auth = getFieldValue('auth_method') || 'kubeconfig';
                          if (auth === 'kubeconfig') {
                            return <Form.Item name="kubeconfig" label="Kubeconfig" rules={[{ required: true }]}><Input.TextArea rows={4} /></Form.Item>;
                          }
                          return (
                            <Row gutter={8}>
                              <Col span={24}><Form.Item name="endpoint" label="Endpoint" rules={[{ required: true }]}><Input placeholder="https://10.0.0.10:6443" /></Form.Item></Col>
                              <Col span={8}><Form.Item name="ca_cert" label="CA Cert" rules={[{ required: true }]}><Input.TextArea rows={3} /></Form.Item></Col>
                              <Col span={8}><Form.Item name="cert" label="Client Cert" rules={[{ required: true }]}><Input.TextArea rows={3} /></Form.Item></Col>
                              <Col span={8}><Form.Item name="key" label="Client Key" rules={[{ required: true }]}><Input.TextArea rows={3} /></Form.Item></Col>
                            </Row>
                          );
                        }}</Form.Item></Col>
                      </Row>
                      <Space><Button type="primary" onClick={() => void importExternalCredential()}>导入外部凭据</Button></Space>
                    </Form>
                    <Table
                      style={{ marginTop: 12 }}
                      rowKey="id"
                      dataSource={credentials}
                      pagination={false}
                      columns={[
                        { title: 'ID', dataIndex: 'id', width: 80 },
                        { title: '名称', dataIndex: 'name' },
                        { title: '来源', dataIndex: 'source', render: (v: string) => <Tag color={v === 'platform_managed' ? 'blue' : 'purple'}>{v}</Tag> },
                        { title: 'Endpoint', dataIndex: 'endpoint', render: (v: string) => v || '-' },
                        { title: '状态', dataIndex: 'status', render: (v: string) => <Tag color={v === 'active' ? 'success' : 'default'}>{v}</Tag> },
                        { title: '最近检测', render: (_: unknown, r: ClusterCredential) => r.last_test_status ? `${r.last_test_status} ${r.last_test_message || ''}` : '-' },
                        { title: '操作', render: (_: unknown, r: ClusterCredential) => <Button size="small" onClick={() => void testCredential(r.id)}>连通性检测</Button> },
                      ]}
                    />
                  </Card>
                </Col>
              </Row>
            ),
          },
        ]} />
      </Card>

      <Modal title="新建 K8s 集群" open={clusterModalOpen} onCancel={() => setClusterModalOpen(false)} onOk={() => void createCluster()} okText="创建">
        <Form form={clusterForm} layout="vertical">
          <Form.Item name="name" label="集群名称" rules={[{ required: true }]}><Input placeholder="prod-cn-hz" /></Form.Item>
          <Form.Item name="server" label="API Server" rules={[{ required: true }]}><Input placeholder="https://10.0.0.10:6443" /></Form.Item>
          <Form.Item name="credential_ref" label="凭据引用"><Input placeholder="env:KUBECONFIG_PROD" /></Form.Item>
          <Form.Item name="description" label="描述"><Input.TextArea rows={3} /></Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`发布详情 #${selectedRelease?.id || ''}`}
        open={!!selectedRelease}
        onCancel={() => {
          setSelectedRelease(null);
          setReleaseTimeline([]);
        }}
        footer={null}
        width={820}
      >
        <pre style={{ maxHeight: 420, overflow: 'auto' }}>
          {selectedRelease ? JSON.stringify({
            runtime_type: selectedRelease.runtime_type,
            status: selectedRelease.status,
            source_release_id: selectedRelease.source_release_id,
            target_revision: selectedRelease.target_revision,
            diagnostics: (() => {
              try {
                return JSON.parse(selectedRelease.diagnostics_json || '[]');
              } catch {
                return selectedRelease.diagnostics_json || [];
              }
            })(),
            verification: (() => {
              try {
                return JSON.parse(selectedRelease.verification_json || '{}');
              } catch {
                return selectedRelease.verification_json || {};
              }
            })(),
          }, null, 2) : ''}
        </pre>
        <Card size="small" title="Release Timeline" style={{ marginTop: 12 }}>
          <Timeline
            items={releaseTimeline.map((item) => ({
              color: item.action.includes('failed') || item.action.includes('rejected') ? 'red' : 'blue',
              children: `${new Date(item.created_at).toLocaleString()} · ${item.action}${item.actor ? ` · actor#${item.actor}` : ''}`,
            }))}
          />
        </Card>
      </Modal>
    </div>
  );
};

export default DeploymentPage;
