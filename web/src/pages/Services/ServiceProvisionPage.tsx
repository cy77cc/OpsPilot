import React from 'react';
import { Alert, Button, Card, Col, Form, Input, InputNumber, Row, Select, Space, Tabs, Tag, Typography, message } from 'antd';
import { ArrowLeftOutlined, SaveOutlined, SwapOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import { Api } from '../../api';
import type { LabelKV, StandardServiceConfig, TemplateVar } from '../../api/modules/services';
import type { Project } from '../../api/modules/projects';

const { Text } = Typography;

const ideChrome = {
  border: '1px solid #2a2f3a',
  borderRadius: 10,
  background: '#0f1117',
  color: '#d4d4d4',
} as const;

const ServiceProvisionPage: React.FC = () => {
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [loading, setLoading] = React.useState(false);
  const [previewing, setPreviewing] = React.useState(false);
  const [activeTarget, setActiveTarget] = React.useState<'k8s' | 'compose' | 'helm'>('k8s');
  const [previewByTarget, setPreviewByTarget] = React.useState<Record<string, string>>({});
  const [diagnosticsByTarget, setDiagnosticsByTarget] = React.useState<Record<string, Array<{ level: string; code: string; message: string }>>>({});
  const [detectedVars, setDetectedVars] = React.useState<TemplateVar[]>([]);
  const [unresolvedVars, setUnresolvedVars] = React.useState<string[]>([]);
  const [varValues, setVarValues] = React.useState<Record<string, string>>({});

  // 项目选择相关
  const [projects, setProjects] = React.useState<Project[]>([]);
  const [canSwitchProject, setCanSwitchProject] = React.useState(false);
  const [currentProjectId, setCurrentProjectId] = React.useState<string>(localStorage.getItem('projectId') || '');

  const mode = Form.useWatch('config_mode', form) || 'standard';
  const valuesSnapshot = Form.useWatch([], form);

  // 加载项目列表和权限
  React.useEffect(() => {
    const loadProjects = async () => {
      try {
        const res = await Api.projects.list();
        const projectList = res.data.list || [];
        setProjects(projectList);
        if (!currentProjectId && projectList.length > 0) {
          setCurrentProjectId(projectList[0].id);
        }
      } catch {
        setProjects([]);
      }
    };

    const checkPermission = async () => {
      try {
        const res = await Api.rbac.checkPermission('project', 'switch');
        setCanSwitchProject(res.data.hasPermission);
      } catch {
        setCanSwitchProject(false);
      }
    };

    loadProjects();
    checkPermission();
  }, [currentProjectId]);

  const currentProjectName = projects.find(p => p.id === currentProjectId)?.name || '当前项目';

  const toLabels = (raw: string[]): LabelKV[] =>
    (raw || [])
      .map((x) => x.trim())
      .filter(Boolean)
      .map((pair) => {
        const [key, ...rest] = pair.split('=');
        return { key: key.trim(), value: rest.join('=').trim() };
      })
      .filter((x) => x.key);

  const buildStandardConfig = (values: any): StandardServiceConfig => ({
    image: values.image,
    replicas: values.replicas || 1,
    ports: [{
      name: 'http',
      protocol: 'TCP',
      container_port: values.container_port || 8080,
      service_port: values.service_port || 80,
    }],
    envs: (values.envs || []).map((x: string) => {
      const [k, ...r] = x.split('=');
      return { key: k.trim(), value: r.join('=').trim() };
    }).filter((x: any) => x.key),
    resources: {
      cpu: values.cpu || '500m',
      memory: values.memory || '512Mi',
    },
  });

  const refreshPreview = React.useCallback(async () => {
    const values = form.getFieldsValue(true);
    if (!values.name) return;
    try {
      setPreviewing(true);
      const targets: Array<'k8s' | 'compose'> = ['k8s', 'compose'];
      const nextPreview: Record<string, string> = {};
      const nextDiag: Record<string, Array<{ level: string; code: string; message: string }>> = {};
      for (const t of targets) {
        const res = await Api.services.preview({
          mode: values.config_mode,
          target: t,
          service_name: values.name,
          service_type: values.service_type,
          standard_config: values.config_mode === 'standard' ? buildStandardConfig(values) : undefined,
          custom_yaml: values.config_mode === 'custom' ? values.custom_yaml : undefined,
          variables: varValues,
        });
        nextPreview[t] = res.data.resolved_yaml || res.data.rendered_yaml || '';
        nextDiag[t] = res.data.diagnostics || [];
        setDetectedVars(res.data.detected_vars || []);
        setUnresolvedVars(res.data.unresolved_vars || []);
      }
      setPreviewByTarget(nextPreview);
      setDiagnosticsByTarget(nextDiag);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '预览失败');
    } finally {
      setPreviewing(false);
    }
  }, [form, varValues]);

  React.useEffect(() => {
    const timer = setTimeout(() => { void refreshPreview(); }, 300);
    return () => clearTimeout(timer);
  }, [valuesSnapshot, refreshPreview]);

  const transformToCustom = async () => {
    const values = await form.validateFields(['name', 'service_type', 'render_target', 'image']);
    const all = form.getFieldsValue(true);
    const res = await Api.services.transform({
      standard_config: buildStandardConfig(all),
      target: values.render_target,
      service_name: values.name,
      service_type: values.service_type,
    });
    form.setFieldsValue({ config_mode: 'custom', custom_yaml: res.data.custom_yaml });
    setDetectedVars(res.data.detected_vars || []);
    message.success('已转换为自定义 YAML');
  };

  const onFinish = async (values: any) => {
    setLoading(true);
    try {
      const created = await Api.services.create({
        project_id: Number(currentProjectId || localStorage.getItem('projectId') || 0) || undefined,
        name: values.name,
        env: values.env,
        owner: values.owner,
        runtime_type: values.runtime_type,
        config_mode: values.config_mode,
        service_kind: values.service_kind,
        visibility: values.visibility,
        icon: values.icon,
        tags: values.tags || [],
        service_type: values.service_type,
        render_target: values.render_target,
        labels: toLabels(values.labels || []),
        standard_config: values.config_mode === 'standard' ? buildStandardConfig(values) : undefined,
        custom_yaml: values.config_mode === 'custom' ? values.custom_yaml : undefined,
        source_template_version: 'v1',
        status: 'draft',
      });
      if (Object.keys(varValues).length > 0 && values.env) {
        await Api.services.upsertVariableValues(String(created.data.id), {
          env: values.env,
          values: varValues,
          secret_keys: [],
        });
      }
      message.success('服务创建成功');
      navigate(`/services/${created.data.id}`);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '创建失败');
    } finally {
      setLoading(false);
    }
  };

  const activePreview = previewByTarget[activeTarget] || '# 暂无输出';

  return (
    <div className="space-y-4">
      {/* 页面顶部导航 */}
      <div className="flex items-center gap-4">
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/services')}>
          返回
        </Button>
        <h1 className="text-xl font-semibold text-gray-900">服务工作室</h1>
      </div>

      <Card style={{ background: '#0b0f16', border: '1px solid #1f2937' }}>
        <Form
          form={form}
          layout="vertical"
          onFinish={onFinish}
          initialValues={{
            env: 'staging',
            owner: 'system',
            runtime_type: 'k8s',
            config_mode: 'standard',
            service_kind: 'business',
            visibility: 'team',
            service_type: 'stateless',
            render_target: 'k8s',
            replicas: 1,
            service_port: 80,
            container_port: 8080,
            cpu: '500m',
            memory: '512Mi',
          }}
        >
          <Row gutter={12}>
            <Col span={12}>
              <Card size="small" style={ideChrome} title={<Text style={{ color: '#cfd8e3' }}>编辑器</Text>}>
                <Row gutter={10}>
                  <Col span={12}>
                    <Form.Item label="项目" required>
                      {canSwitchProject ? (
                        <Select
                          value={currentProjectId}
                          options={projects.map(p => ({ value: p.id, label: p.name }))}
                          onChange={(val) => {
                            setCurrentProjectId(val);
                            localStorage.setItem('projectId', val);
                          }}
                          placeholder="选择项目"
                        />
                      ) : (
                        <Input value={currentProjectName} disabled />
                      )}
                    </Form.Item>
                  </Col>
                  <Col span={12}>
                    <Form.Item label="环境" name="env">
                      <Select options={[
                        { value: 'development', label: '开发环境' },
                        { value: 'staging', label: '测试环境' },
                        { value: 'production', label: '生产环境' },
                      ]} />
                    </Form.Item>
                  </Col>
                </Row>
                <Row gutter={10}>
                  <Col span={12}><Form.Item label="服务名" name="name" rules={[{ required: true }]}><Input placeholder="user-service" /></Form.Item></Col>
                  <Col span={12}><Form.Item label="负责人" name="owner" rules={[{ required: true }]}><Input /></Form.Item></Col>
                </Row>
                <Row gutter={10}>
                  <Col span={8}><Form.Item label="运行时" name="runtime_type"><Select options={[{ value: 'k8s' }, { value: 'compose' }, { value: 'helm' }]} /></Form.Item></Col>
                  <Col span={8}><Form.Item label="配置模式" name="config_mode"><Select options={[{ value: 'standard', label: '通用配置' }, { value: 'custom', label: '自定义 YAML' }]} /></Form.Item></Col>
                  <Col span={8}><Form.Item label="类型" name="service_type"><Select options={[{ value: 'stateless' }, { value: 'stateful' }]} /></Form.Item></Col>
                </Row>
                <Row gutter={10}>
                  <Col span={6}><Form.Item label="服务分类" name="service_kind"><Select options={[{ value: 'middleware', label: '中间件' }, { value: 'business', label: '业务服务' }]} /></Form.Item></Col>
                  <Col span={6}><Form.Item label="可见性" name="visibility"><Select options={[{ value: 'private' }, { value: 'team' }, { value: 'team-granted' }, { value: 'public' }]} /></Form.Item></Col>
                  <Col span={6}><Form.Item label="图标" name="icon"><Input placeholder="图标URL/名称" /></Form.Item></Col>
                  <Col span={6}><Form.Item label="服务标签" name="tags"><Select mode="tags" placeholder="mysql,database" /></Form.Item></Col>
                </Row>
                <Row gutter={10}>
                  <Col span={24}><Form.Item label="标签(key=value)" name="labels"><Select mode="tags" placeholder="app=user,tier=backend" /></Form.Item></Col>
                </Row>

                {mode === 'standard' ? (
                  <>
                    <Row gutter={10}>
                      <Col span={24}><Form.Item label="镜像" name="image" rules={[{ required: true }]}><Input placeholder="ghcr.io/org/app:v1" /></Form.Item></Col>
                    </Row>
                    <Row gutter={10}>
                      <Col span={8}><Form.Item label="副本" name="replicas"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                      <Col span={8}><Form.Item label="服务端口" name="service_port"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                      <Col span={8}><Form.Item label="容器端口" name="container_port"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                    </Row>
                    <Row gutter={10}>
                      <Col span={12}><Form.Item label="CPU" name="cpu"><Input placeholder="500m" /></Form.Item></Col>
                      <Col span={12}><Form.Item label="内存" name="memory"><Input placeholder="512Mi" /></Form.Item></Col>
                    </Row>
                    <Form.Item label="环境变量(KEY=VALUE)" name="envs"><Select mode="tags" /></Form.Item>
                    <Button icon={<SwapOutlined />} onClick={transformToCustom}>转换为自定义 YAML</Button>
                  </>
                ) : (
                  <Form.Item label="自定义 YAML" name="custom_yaml" rules={[{ required: true, message: '请输入 YAML' }]}>
                    <Editor
                      height="280px"
                      defaultLanguage="yaml"
                      theme="vs-dark"
                      options={{
                        minimap: { enabled: true },
                        fontSize: 13,
                        smoothScrolling: true,
                        stickyScroll: { enabled: true },
                      }}
                    />
                  </Form.Item>
                )}

                <Card size="small" style={{ marginTop: 10, background: '#111827', border: '1px solid #243041' }} title={<Text style={{ color: '#cfd8e3' }}>模板变量</Text>}>
                  {detectedVars.length === 0 ? <Text type="secondary">未检测到模板变量（{'{{var}}'}）</Text> : null}
                  <Space orientation="vertical" style={{ width: '100%' }}>
                    {detectedVars.map((item) => (
                      <Input
                        key={item.name}
                        addonBefore={<span>{item.name}{item.required ? <Tag color="red" style={{ marginLeft: 8 }}>必填</Tag> : null}</span>}
                        placeholder={item.default || '变量值'}
                        value={varValues[item.name] || ''}
                        onChange={(e) => setVarValues((prev) => ({ ...prev, [item.name]: e.target.value }))}
                      />
                    ))}
                  </Space>
                  {unresolvedVars.length > 0 ? <Alert style={{ marginTop: 8 }} type="warning" showIcon message={`未解析变量: ${unresolvedVars.join(', ')}`} /> : null}
                </Card>
                <Space style={{ marginTop: 12 }}>
                  <Button icon={<SaveOutlined />} htmlType="submit" loading={loading}>创建服务</Button>
                  <Button onClick={() => void refreshPreview()} loading={previewing}>刷新预览</Button>
                </Space>
              </Card>
            </Col>

            <Col span={12}>
              <Card size="small" style={ideChrome} title={<Text style={{ color: '#cfd8e3' }}>预览</Text>}>
                <Tabs
                  activeKey={activeTarget}
                  onChange={(k) => setActiveTarget(k as 'k8s' | 'compose' | 'helm')}
                  items={[
                    { key: 'k8s', label: 'K8s 配置' },
                    { key: 'compose', label: 'Compose 配置' },
                    { key: 'helm', label: 'Helm' },
                  ]}
                />
                <Editor
                  height="520px"
                  defaultLanguage="yaml"
                  value={activeTarget === 'helm' ? (previewByTarget.k8s || '# Helm 首期复用 K8s 渲染预览') : activePreview}
                  theme="vs-dark"
                  options={{
                    readOnly: true,
                    minimap: { enabled: true },
                    lineNumbers: 'on',
                    scrollBeyondLastLine: false,
                  }}
                />
                <div style={{ background: '#111827', borderTop: '1px solid #2b3442', marginTop: 6, padding: '6px 10px', borderRadius: 6 }}>
                  <Space>
                    <Tag color="processing">target: {activeTarget}</Tag>
                    <Tag color={previewing ? 'warning' : 'success'}>{previewing ? '渲染中' : '就绪'}</Tag>
                    <Tag color={unresolvedVars.length > 0 ? 'error' : 'default'}>未解析: {unresolvedVars.length}</Tag>
                  </Space>
                </div>
                {(diagnosticsByTarget[activeTarget] || []).length > 0 ? (
                  <Card size="small" style={{ marginTop: 10, background: '#111827', border: '1px solid #243041' }} title={<Text style={{ color: '#cfd8e3' }}>诊断信息</Text>}>
                    <Space orientation="vertical" size={4}>
                      {(diagnosticsByTarget[activeTarget] || []).map((d, idx) => (
                        <Text key={`${d.code}-${idx}`} type={d.level === 'error' ? 'danger' : 'secondary'}>
                          [{d.level}] {d.code}: {d.message}
                        </Text>
                      ))}
                    </Space>
                  </Card>
                ) : null}
              </Card>
            </Col>
          </Row>
        </Form>
      </Card>
    </div>
  );
};

export default ServiceProvisionPage;
