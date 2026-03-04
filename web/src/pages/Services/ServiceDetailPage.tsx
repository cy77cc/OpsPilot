import React, { useState, useCallback, useEffect, useMemo } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Form,
  Input,
  Row,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
  message,
  Statistic,
  Progress,
  Badge,
  Empty,
  Modal,
} from 'antd';
import {
  ArrowLeftOutlined,
  CloudUploadOutlined,
  EditOutlined,
  ReloadOutlined,
  SaveOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ExclamationCircleOutlined,
  FileTextOutlined,
  DesktopOutlined,
  DatabaseOutlined,
  ApiOutlined,
  BarChartOutlined,
  SettingOutlined,
  FileSearchOutlined,
  CloseOutlined,
} from '@ant-design/icons';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { Api } from '../../api';
import type {
  LabelKV,
  ServiceCreateParams,
  ServiceEvent,
  ServiceItem,
  ServiceReleaseRecord,
  ServiceRevision,
  StandardServiceConfig,
  TemplateVar,
  VariableValueSet,
} from '../../api/modules/services';

type ServiceEditFormValues = {
  name: string;
  env: string;
  owner: string;
  service_kind: string;
  service_type: 'stateless' | 'stateful';
  runtime_type: 'k8s' | 'compose' | 'helm';
  config_mode: 'standard' | 'custom';
  render_target: 'k8s' | 'compose' | 'helm';
  status: string;
  labels_text?: string;
  standard_config_text?: string;
  custom_yaml?: string;
};

const parseLabelText = (text: string): LabelKV[] => {
  const rows = String(text || '').split('\n').map((x) => x.trim()).filter(Boolean);
  const list: LabelKV[] = [];
  rows.forEach((row) => {
    const idx = row.indexOf('=');
    if (idx <= 0) {
      return;
    }
    const key = row.slice(0, idx).trim();
    const value = row.slice(idx + 1).trim();
    if (key) {
      list.push({ key, value });
    }
  });
  return list;
};

// 获取状态配置
const getStatusConfig = (status: string) => {
  const configs: Record<string, { icon: React.ReactNode; color: string; text: string }> = {
    running: { icon: <CheckCircleOutlined />, color: 'success', text: '运行中' },
    deploying: { icon: <ClockCircleOutlined />, color: 'processing', text: '部署中' },
    syncing: { icon: <ClockCircleOutlined />, color: 'processing', text: '同步中' },
    error: { icon: <ExclamationCircleOutlined />, color: 'error', text: '错误' },
    draft: { icon: <FileTextOutlined />, color: 'default', text: '草稿' },
    stopped: { icon: <ExclamationCircleOutlined />, color: 'default', text: '已停止' },
  };
  return configs[status] || { icon: null, color: 'default', text: status };
};

const ServiceDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const [loading, setLoading] = useState(false);
  const [service, setService] = useState<ServiceItem | null>(null);
  const [events, setEvents] = useState<ServiceEvent[]>([]);
  const [revisions, setRevisions] = useState<ServiceRevision[]>([]);
  const [releases, setReleases] = useState<ServiceReleaseRecord[]>([]);
  const [varSchema, setVarSchema] = useState<TemplateVar[]>([]);
  const [varSet, setVarSet] = useState<VariableValueSet | null>(null);
  const [previewYAML, setPreviewYAML] = useState('');
  const [previewWarnings, setPreviewWarnings] = useState<Array<{ level: string; code: string; message: string }>>([]);
  const [deploying, setDeploying] = useState(false);
  const [activeTab, setActiveTab] = useState(() => searchParams.get('tab') || 'overview');
  const [varForm] = Form.useForm();
  const [editForm] = Form.useForm<ServiceEditFormValues>();
  const [editing, setEditing] = useState(false);
  const [editSaving, setEditSaving] = useState(false);

  const env = Form.useWatch('env', varForm) || 'staging';

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const [detail, eventRes, revRes, relRes, schemaRes] = await Promise.all([
        Api.services.getDetail(id),
        Api.services.getEvents(id),
        Api.services.listRevisions(id),
        Api.services.listReleases(id),
        Api.services.getVariableSchema(id),
      ]);
      setService(detail.data);
      setEvents(eventRes.data.list || []);
      setRevisions(revRes.data.list || []);
      setReleases(relRes.data.list || []);
      setVarSchema(schemaRes.data.vars || []);
      varForm.setFieldValue('env', detail.data.env || 'staging');
      // 初始化编辑表单
      editForm.setFieldsValue({
        name: detail.data.name,
        env: detail.data.env,
        owner: detail.data.owner,
        service_kind: detail.data.serviceKind,
        service_type: detail.data.serviceType || 'stateless',
        runtime_type: detail.data.runtimeType,
        config_mode: detail.data.configMode,
        render_target: detail.data.renderTarget || (detail.data.runtimeType === 'helm' ? 'k8s' : detail.data.runtimeType),
        status: detail.data.status,
        labels_text: (detail.data.labels || []).map((x) => `${x.key}=${x.value}`).join('\n'),
        standard_config_text: detail.data.standardConfig ? JSON.stringify(detail.data.standardConfig, null, 2) : '{\n  "image": "",\n  "replicas": 1,\n  "ports": [],\n  "envs": []\n}',
        custom_yaml: detail.data.customYaml || '',
      });
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载服务详情失败');
    } finally {
      setLoading(false);
    }
  }, [id, varForm, editForm]);

  const loadVarSet = useCallback(async () => {
    if (!id) return;
    try {
      const resp = await Api.services.getVariableValues(id, env);
      setVarSet(resp.data);
      const values: Record<string, any> = { env };
      Object.entries(resp.data.values || {}).forEach(([k, v]) => { values[`var_${k}`] = v; });
      varForm.setFieldsValue(values);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载变量集失败');
    }
  }, [id, env, varForm]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!service || !id) return;
    void loadVarSet();
  }, [service, id, loadVarSet]);

  // 10秒自动刷新
  useEffect(() => {
    const interval = setInterval(() => {
      void load();
    }, 10000);
    return () => clearInterval(interval);
  }, [load]);

  // 统计数据
  const stats = useMemo(() => {
    const successReleases = releases.filter((r) => r.status === 'succeeded').length;
    const failedReleases = releases.filter((r) => r.status === 'failed').length;
    const successRate = releases.length > 0 ? Math.round((successReleases / releases.length) * 100) : 0;
    return { successReleases, failedReleases, successRate, totalReleases: releases.length };
  }, [releases]);

  const statusConfig = service ? getStatusConfig(service.status) : null;

  const saveVarValues = async () => {
    if (!id) return;
    const values = await varForm.validateFields();
    const envVal = values.env;
    const vars: Record<string, string> = {};
    Object.keys(values).forEach((k) => {
      if (k.startsWith('var_') && String(values[k] || '').trim() !== '') {
        vars[k.replace(/^var_/, '')] = String(values[k]);
      }
    });
    const resp = await Api.services.upsertVariableValues(id, {
      env: envVal,
      values: vars,
      secret_keys: [],
    });
    setVarSet(resp.data);
    message.success('变量集已保存');
  };

  const deploy = async () => {
    if (!id) return;
    setDeploying(true);
    try {
      const values = varForm.getFieldsValue(true);
      const vars: Record<string, string> = {};
      Object.keys(values).forEach((k) => {
        if (k.startsWith('var_') && String(values[k] || '').trim() !== '') {
          vars[k.replace(/^var_/, '')] = String(values[k]);
        }
      });
      // TODO: 弹窗选择集群和命名空间
      const resp = await Api.services.deploy(id, {
        env: values.env || service?.env,
        variables: vars,
      });
      const releaseId = resp.data.unified_release_id || resp.data.release_record_id;
      message.success(`部署已触发，release #${releaseId}`);
      await load();
    } catch (err) {
      const errorText = err instanceof Error ? err.message : '部署失败';
      if (errorText.includes('deploy target not configured')) {
        message.error('部署目标未配置：请先在部署目标中绑定项目/团队/环境对应目标，或为该服务设置默认部署目标');
      } else {
        message.error(errorText);
      }
    } finally {
      setDeploying(false);
    }
  };

  const createRevision = async () => {
    if (!id || !service) return;
    await Api.services.createRevision(id, {
      config_mode: service.configMode,
      render_target: (service.runtimeType === 'helm' ? 'k8s' : service.runtimeType) as 'k8s' | 'compose',
      standard_config: service.standardConfig,
      custom_yaml: service.customYaml,
      variable_schema: varSchema,
    });
    message.success('新 revision 已创建');
    await load();
  };

  const startEditing = () => {
    if (!service) return;
    editForm.setFieldsValue({
      name: service.name,
      env: service.env,
      owner: service.owner,
      service_kind: service.serviceKind,
      service_type: service.serviceType || 'stateless',
      runtime_type: service.runtimeType,
      config_mode: service.configMode,
      render_target: service.renderTarget || (service.runtimeType === 'helm' ? 'k8s' : service.runtimeType),
      status: service.status,
      labels_text: (service.labels || []).map((x) => `${x.key}=${x.value}`).join('\n'),
      standard_config_text: service.standardConfig ? JSON.stringify(service.standardConfig, null, 2) : '{\n  "image": "",\n  "replicas": 1,\n  "ports": [],\n  "envs": []\n}',
      custom_yaml: service.customYaml || '',
    });
    setEditing(true);
  };

  const cancelEditing = () => {
    setEditing(false);
    // 恢复原始值
    if (service) {
      editForm.setFieldsValue({
        name: service.name,
        env: service.env,
        owner: service.owner,
        service_kind: service.serviceKind,
        service_type: service.serviceType || 'stateless',
        runtime_type: service.runtimeType,
        config_mode: service.configMode,
        render_target: service.renderTarget || (service.runtimeType === 'helm' ? 'k8s' : service.runtimeType),
        status: service.status,
        labels_text: (service.labels || []).map((x) => `${x.key}=${x.value}`).join('\n'),
        standard_config_text: service.standardConfig ? JSON.stringify(service.standardConfig, null, 2) : '{\n  "image": "",\n  "replicas": 1,\n  "ports": [],\n  "envs": []\n}',
        custom_yaml: service.customYaml || '',
      });
    }
  };

  const saveServiceEdit = async () => {
    if (!id || !service) return;
    const values = await editForm.validateFields();

    let standardConfig: StandardServiceConfig | undefined;
    if (values.config_mode === 'standard') {
      try {
        standardConfig = JSON.parse(values.standard_config_text || '{}') as StandardServiceConfig;
      } catch {
        message.error('标准配置 JSON 格式错误');
        return;
      }
    }

    const payload: Partial<ServiceCreateParams> = {
      name: values.name,
      env: values.env,
      owner: values.owner,
      service_kind: values.service_kind,
      service_type: values.service_type,
      runtime_type: values.runtime_type,
      config_mode: values.config_mode,
      render_target: (values.render_target === 'helm' ? 'k8s' : values.render_target) as 'k8s' | 'compose',
      status: values.status,
      labels: parseLabelText(values.labels_text || ''),
      standard_config: values.config_mode === 'standard' ? standardConfig : undefined,
      custom_yaml: values.config_mode === 'custom' ? (values.custom_yaml || '') : '',
    };

    setEditSaving(true);
    try {
      await Api.services.update(id, payload);
      message.success('服务配置已更新');
      setEditing(false);
      await load();
      Modal.confirm({
        title: '是否基于最新配置创建 Revision？',
        okText: '创建 Revision',
        cancelText: '稍后再说',
        onOk: () => createRevision(),
      });
    } catch (err) {
      message.error(err instanceof Error ? err.message : '保存服务配置失败');
    } finally {
      setEditSaving(false);
    }
  };

  // 渲染配置 Tab 内容
  const renderConfigTab = () => (
    <Row gutter={16}>
      <Col xs={24} lg={12}>
        <div className="space-y-4">
          {/* 服务配置卡片 */}
          <Card
            title="服务配置"
            size="small"
            extra={
              editing ? (
                <Space>
                  <Button size="small" icon={<CloseOutlined />} onClick={cancelEditing} disabled={editSaving}>
                    取消
                  </Button>
                  <Button size="small" type="primary" icon={<SaveOutlined />} onClick={saveServiceEdit} loading={editSaving}>
                    保存
                  </Button>
                </Space>
              ) : (
                <Button size="small" icon={<EditOutlined />} onClick={startEditing}>
                  编辑
                </Button>
              )
            }
          >
            <Form form={editForm} layout="vertical">
              <Row gutter={12}>
                <Col span={12}>
                  <Form.Item name="name" label="服务名" rules={[{ required: true, message: '请输入服务名' }]}>
                    <Input disabled={!editing} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="env" label="环境" rules={[{ required: true }]}>
                    <Select disabled={!editing} options={[{ value: 'development' }, { value: 'staging' }, { value: 'production' }]} />
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={12}>
                <Col span={12}>
                  <Form.Item name="owner" label="负责人" rules={[{ required: true, message: '请输入负责人' }]}>
                    <Input disabled={!editing} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="status" label="状态" rules={[{ required: true }]}>
                    <Select disabled={!editing} options={[{ value: 'draft' }, { value: 'running' }, { value: 'stopped' }, { value: 'error' }]} />
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={12}>
                <Col span={8}>
                  <Form.Item name="service_kind" label="服务分类" rules={[{ required: true }]}>
                    <Input disabled={!editing} placeholder="web/worker/job" />
                  </Form.Item>
                </Col>
                <Col span={8}>
                  <Form.Item name="service_type" label="服务类型" rules={[{ required: true }]}>
                    <Select disabled={!editing} options={[{ value: 'stateless' }, { value: 'stateful' }]} />
                  </Form.Item>
                </Col>
                <Col span={8}>
                  <Form.Item name="runtime_type" label="运行时" rules={[{ required: true }]}>
                    <Select disabled={!editing} options={[{ value: 'k8s' }, { value: 'compose' }, { value: 'helm' }]} />
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={12}>
                <Col span={12}>
                  <Form.Item name="config_mode" label="配置模式" rules={[{ required: true }]}>
                    <Select disabled={!editing} options={[{ value: 'standard' }, { value: 'custom' }]} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="render_target" label="渲染目标" rules={[{ required: true }]}>
                    <Select disabled={!editing} options={[{ value: 'k8s' }, { value: 'compose' }, { value: 'helm' }]} />
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item name="labels_text" label="标签（每行 key=value）">
                <Input.TextArea rows={3} disabled={!editing} placeholder={'app=api\nteam=platform'} />
              </Form.Item>

              <Form.Item noStyle shouldUpdate={(prev, next) => prev.config_mode !== next.config_mode}>
                {({ getFieldValue }) => (
                  getFieldValue('config_mode') === 'standard' ? (
                    <Form.Item name="standard_config_text" label="标准配置（JSON）" rules={[{ required: true, message: '请输入标准配置 JSON' }]}>
                      <Input.TextArea rows={10} disabled={!editing} />
                    </Form.Item>
                  ) : (
                    <Form.Item name="custom_yaml" label="自定义 YAML" rules={[{ required: true, message: '请输入 YAML 配置' }]}>
                      <Input.TextArea rows={10} disabled={!editing} />
                    </Form.Item>
                  )
                )}
              </Form.Item>
            </Form>
          </Card>

          {/* 环境变量集卡片 */}
          <Card title="环境变量集" size="small">
            <Form form={varForm} layout="vertical">
              <Form.Item name="env" label="环境">
                <Select
                  options={[
                    { value: 'development', label: 'Development' },
                    { value: 'staging', label: 'Staging' },
                    { value: 'production', label: 'Production' },
                  ]}
                />
              </Form.Item>
              {varSchema.map((v) => (
                <Form.Item
                  key={v.name}
                  name={`var_${v.name}`}
                  label={`${v.name}${v.required ? ' *' : ''}`}
                >
                  <Input placeholder={v.default || ''} />
                </Form.Item>
              ))}
              <Button onClick={saveVarValues}>保存变量集</Button>
            </Form>
            {varSet?.updated_at && (
              <Alert
                type="info"
                style={{ marginTop: 8 }}
                message={`最近更新: ${new Date(varSet.updated_at).toLocaleString()}`}
              />
            )}
          </Card>
        </div>
      </Col>
      <Col xs={24} lg={12}>
        {/* 渲染输出预览 */}
        <Card title="渲染输出预览" size="small">
          {previewWarnings.length > 0 && (
            <Space direction="vertical" style={{ width: '100%', marginBottom: 8 }}>
              {previewWarnings.map((w, idx) => (
                <Alert
                  key={`${w.code}-${idx}`}
                  type={w.level === 'error' ? 'error' : 'warning'}
                  showIcon
                  message={w.message}
                />
              ))}
            </Space>
          )}
          <pre
            className="bg-gray-50 p-4 rounded-lg text-sm overflow-auto"
            style={{ maxHeight: 600 }}
          >
            {previewYAML || service?.renderedYaml || service?.customYaml || '# 暂无输出'}
          </pre>
        </Card>
      </Col>
    </Row>
  );

  return (
    <div className="space-y-6">
      {/* 页面头部 */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/services')}>
            返回
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-semibold text-gray-900">{service?.name || '服务详情'}</h1>
              {statusConfig && (
                <Tag color={statusConfig.color} icon={statusConfig.icon} className="text-sm">
                  {statusConfig.text}
                </Tag>
              )}
            </div>
            <p className="text-sm text-gray-500 mt-1">
              {service?.env && <span>环境: {service.env}</span>}
              {service?.owner && <span className="ml-4">负责人: {service.owner}</span>}
            </p>
          </div>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>
            刷新
          </Button>
          <Button icon={<SaveOutlined />} onClick={createRevision}>
            创建 Revision
          </Button>
          <Button icon={<CloudUploadOutlined />} onClick={() => navigate(`/services/${id}/deploy`)}>
            部署向导
          </Button>
          <Button icon={<SettingOutlined />} onClick={() => navigate(`/services/${id}/visibility`)}>
            可见性设置
          </Button>
          <Button type="primary" loading={deploying} onClick={deploy}>快速部署</Button>
        </Space>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card className="hover:shadow-lg transition-shadow">
            <Statistic
              title={<span className="text-gray-600">总发布次数</span>}
              value={stats.totalReleases}
              prefix={<ApiOutlined className="text-primary-500" />}
              valueStyle={{ color: '#495057', fontSize: '24px', fontWeight: 600 }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="hover:shadow-lg transition-shadow">
            <Statistic
              title={<span className="text-gray-600">成功率</span>}
              value={stats.successRate}
              suffix="%"
              prefix={<CheckCircleOutlined className="text-success" />}
              valueStyle={{ color: '#10b981', fontSize: '24px', fontWeight: 600 }}
            />
            <Progress
              percent={stats.successRate}
              strokeColor="#10b981"
              showInfo={false}
              className="mt-2"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="hover:shadow-lg transition-shadow">
            <Statistic
              title={<span className="text-gray-600">版本数</span>}
              value={revisions.length}
              prefix={<DatabaseOutlined className="text-primary-500" />}
              valueStyle={{ color: '#495057', fontSize: '24px', fontWeight: 600 }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="hover:shadow-lg transition-shadow">
            <Statistic
              title={<span className="text-gray-600">事件数</span>}
              value={events.length}
              prefix={<FileSearchOutlined className="text-primary-500" />}
              valueStyle={{ color: '#495057', fontSize: '24px', fontWeight: 600 }}
            />
          </Card>
        </Col>
      </Row>

      {/* Tab 内容 */}
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'overview',
              label: (
                <span>
                  <DesktopOutlined className="mr-2" />
                  概览
                </span>
              ),
              children: (
                <div className="space-y-6">
                  {/* 基本信息 */}
                  <div>
                    <h3 className="text-base font-semibold text-gray-900 mb-4">基本信息</h3>
                    {service && (
                      <Descriptions bordered column={2}>
                        <Descriptions.Item label="服务名称">{service.name}</Descriptions.Item>
                        <Descriptions.Item label="环境">
                          <Tag>{service.env}</Tag>
                        </Descriptions.Item>
                        <Descriptions.Item label="运行时">
                          <Tag color="blue">{service.runtimeType}</Tag>
                        </Descriptions.Item>
                        <Descriptions.Item label="配置模式">
                          <Tag>{service.configMode}</Tag>
                        </Descriptions.Item>
                        <Descriptions.Item label="状态">
                          {statusConfig && (
                            <Tag color={statusConfig.color} icon={statusConfig.icon}>
                              {statusConfig.text}
                            </Tag>
                          )}
                        </Descriptions.Item>
                        <Descriptions.Item label="负责人">{service.owner || '-'}</Descriptions.Item>
                        <Descriptions.Item label="项目 ID">{service.projectId || '-'}</Descriptions.Item>
                        <Descriptions.Item label="团队 ID">{service.teamId || '-'}</Descriptions.Item>
                        <Descriptions.Item label="服务分类">{service.serviceKind || '-'}</Descriptions.Item>
                        <Descriptions.Item label="可见性">{service.visibility || 'team'}</Descriptions.Item>
                        <Descriptions.Item label="服务类型">{service.serviceType || '-'}</Descriptions.Item>
                        <Descriptions.Item label="部署次数">{service.deployCount || 0}</Descriptions.Item>
                        <Descriptions.Item label="模板引擎">{service.templateEngineVersion || 'v1'}</Descriptions.Item>
                        <Descriptions.Item label="最新 Revision">{service.lastRevisionId || '-'}</Descriptions.Item>
                        <Descriptions.Item label="标签" span={2}>
                          {service.labels && service.labels.length > 0 ? (
                            <Space size={[4, 4]} wrap>
                              {service.labels.map((l) => (
                                <Tag key={`${l.key}:${l.value}`}>
                                  {l.key}={l.value}
                                </Tag>
                              ))}
                            </Space>
                          ) : (
                            '-'
                          )}
                        </Descriptions.Item>
                        <Descriptions.Item label="服务标签" span={2}>
                          {service.tags && service.tags.length > 0 ? service.tags.map((t) => <Tag key={t}>{t}</Tag>) : '-'}
                        </Descriptions.Item>
                      </Descriptions>
                    )}
                  </div>

                  {/* 最近发布 */}
                  <div>
                    <h3 className="text-base font-semibold text-gray-900 mb-4">最近发布</h3>
                    {releases.length > 0 ? (
                      <Table
                        rowKey="id"
                        dataSource={releases.slice(0, 5)}
                        pagination={false}
                        size="small"
                        columns={[
                          { title: 'ID', dataIndex: 'id', width: 80 },
                          {
                            title: '环境',
                            dataIndex: 'env',
                            width: 100,
                            render: (v: string) => <Tag>{v}</Tag>,
                          },
                          {
                            title: '目标',
                            dataIndex: 'deploy_target',
                            width: 100,
                            render: (v: string) => <Tag color="blue">{v}</Tag>,
                          },
                          {
                            title: '集群/命名空间',
                            width: 180,
                            render: (_: any, r: ServiceReleaseRecord) => `${r.cluster_id} / ${r.namespace}`,
                          },
                          {
                            title: '状态',
                            dataIndex: 'status',
                            width: 100,
                            render: (v: string) => (
                              <Tag
                                color={
                                  v === 'succeeded' ? 'success' : v === 'failed' ? 'error' : 'processing'
                                }
                              >
                                {v}
                              </Tag>
                            ),
                          },
                          {
                            title: '时间',
                            dataIndex: 'created_at',
                            render: (v: string) => new Date(v).toLocaleString(),
                          },
                        ]}
                      />
                    ) : (
                      <Empty description="暂无发布记录" image={Empty.PRESENTED_IMAGE_SIMPLE} />
                    )}
                  </div>
                </div>
              ),
            },
            {
              key: 'config',
              label: (
                <span>
                  <SettingOutlined className="mr-2" />
                  配置
                </span>
              ),
              children: renderConfigTab(),
            },
            {
              key: 'logs',
              label: (
                <span>
                  <FileTextOutlined className="mr-2" />
                  日志
                </span>
              ),
              children: (
                <Empty
                  description="日志查看功能开发中"
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                />
              ),
            },
            {
              key: 'monitoring',
              label: (
                <span>
                  <BarChartOutlined className="mr-2" />
                  监控
                </span>
              ),
              children: (
                <Empty
                  description="监控图表功能开发中"
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                />
              ),
            },
            {
              key: 'revisions',
              label: (
                <span>
                  <DatabaseOutlined className="mr-2" />
                  版本 <Badge count={revisions.length} showZero className="ml-1" />
                </span>
              ),
              children: revisions.length > 0 ? (
                <Table
                  rowKey="id"
                  dataSource={revisions}
                  pagination={{ pageSize: 10 }}
                  columns={[
                    { title: 'Revision #', dataIndex: 'revision_no', width: 100 },
                    {
                      title: '配置模式',
                      dataIndex: 'config_mode',
                      width: 120,
                      render: (v: string) => <Tag>{v}</Tag>,
                    },
                    {
                      title: '渲染目标',
                      dataIndex: 'render_target',
                      width: 120,
                      render: (v: string) => <Tag color="blue">{v}</Tag>,
                    },
                    { title: '创建人', dataIndex: 'created_by', width: 120 },
                    {
                      title: '创建时间',
                      dataIndex: 'created_at',
                      render: (v: string) => new Date(v).toLocaleString(),
                    },
                  ]}
                />
              ) : (
                <Empty description="暂无版本记录" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              ),
            },
            {
              key: 'releases',
              label: (
                <span>
                  <CloudUploadOutlined className="mr-2" />
                  发布 <Badge count={releases.length} showZero className="ml-1" />
                </span>
              ),
              children: releases.length > 0 ? (
                <Table
                  rowKey="id"
                  dataSource={releases}
                  pagination={{ pageSize: 10 }}
                  columns={[
                    { title: 'Release #', dataIndex: 'id', width: 100 },
                    {
                      title: '环境',
                      dataIndex: 'env',
                      width: 100,
                      render: (v: string) => <Tag>{v}</Tag>,
                    },
                    {
                      title: '部署目标',
                      dataIndex: 'deploy_target',
                      width: 120,
                      render: (v: string) => <Tag color="blue">{v}</Tag>,
                    },
                    {
                      title: '集群/命名空间',
                      width: 180,
                      render: (_: any, r: ServiceReleaseRecord) => `${r.cluster_id} / ${r.namespace}`,
                    },
                    {
                      title: '状态',
                      dataIndex: 'status',
                      width: 120,
                      render: (v: string) => (
                        <Tag color={v === 'succeeded' ? 'success' : v === 'failed' ? 'error' : 'processing'}>
                          {v}
                        </Tag>
                      ),
                    },
                    {
                      title: '创建时间',
                      dataIndex: 'created_at',
                      render: (v: string) => new Date(v).toLocaleString(),
                    },
                  ]}
                />
              ) : (
                <Empty description="暂无发布记录" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              ),
            },
            {
              key: 'events',
              label: (
                <span>
                  <FileSearchOutlined className="mr-2" />
                  事件 <Badge count={events.length} showZero className="ml-1" />
                </span>
              ),
              children: events.length > 0 ? (
                <Table
                  rowKey="id"
                  dataSource={events}
                  pagination={{ pageSize: 10 }}
                  columns={[
                    {
                      title: '时间',
                      dataIndex: 'createdAt',
                      width: 180,
                      render: (v: string) => (v ? new Date(v).toLocaleString() : '-'),
                    },
                    {
                      title: '类型',
                      dataIndex: 'type',
                      width: 120,
                      render: (v: string) => <Tag>{v}</Tag>,
                    },
                    {
                      title: '级别',
                      dataIndex: 'level',
                      width: 100,
                      render: (v: string) => (
                        <Tag color={v === 'warning' ? 'warning' : v === 'error' ? 'error' : 'blue'}>
                          {v}
                        </Tag>
                      ),
                    },
                    { title: '内容', dataIndex: 'message' },
                  ]}
                />
              ) : (
                <Empty description="暂无事件记录" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
};

export default ServiceDetailPage;
