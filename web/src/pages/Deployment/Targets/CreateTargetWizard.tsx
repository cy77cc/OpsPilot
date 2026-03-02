import React, { useState, useEffect } from 'react';
import { Steps, Form, Input, Select, Button, Card, Space, message, Radio, Alert } from 'antd';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../../api';



interface TargetFormData {
  name: string;
  environment: string;
  runtime_type: 'k8s' | 'compose';
  credential_id?: number;
  cluster_id?: number;
  namespace?: string;
  host_ids?: number[];
  auto_bootstrap?: boolean;
}

const CreateTargetWizard: React.FC = () => {
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [currentStep, setCurrentStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState<TargetFormData>({
    name: '',
    environment: 'development',
    runtime_type: 'k8s',
    auto_bootstrap: false,
  });
  const [credentials, setCredentials] = useState<any[]>([]);
  const [clusters, setClusters] = useState<any[]>([]);
  const [hosts, setHosts] = useState<any[]>([]);

  useEffect(() => {
    loadResources();
  }, []);

  const loadResources = async () => {
    try {
      const [credRes, clusterRes, hostRes] = await Promise.all([
        Api.deployment.listCredentials(),
        Api.deployment.listClusters(),
        Api.hosts.getHostList({ pageSize: 200 }),
      ]);
      setCredentials(credRes.data.list || []);
      setClusters(clusterRes.data.list || []);
      setHosts(hostRes.data.list || []);
    } catch (err) {
      message.error('加载资源失败');
    }
  };

  const handleNext = async () => {
    try {
      await form.validateFields();
      const values = form.getFieldsValue();
      setFormData({ ...formData, ...values });
      setCurrentStep(currentStep + 1);
    } catch (err) {
      // Validation failed
    }
  };

  const handlePrev = () => {
    setCurrentStep(currentStep - 1);
  };

  const handleSubmit = async () => {
    try {
      setLoading(true);
      const values = form.getFieldsValue();
      const finalData = { ...formData, ...values };

      const res = await Api.deployment.createTarget({
        name: finalData.name,
        target_type: finalData.runtime_type,
        environment: finalData.environment,
        credential_id: finalData.credential_id,
        cluster_id: finalData.cluster_id,
      });

      message.success('部署目标创建成功');

      // If auto_bootstrap is enabled, start bootstrap
      if (finalData.auto_bootstrap && res.data.id) {
        try {
          const bootstrapRes = await Api.deployment.startEnvironmentBootstrap({
            name: `bootstrap-${res.data.id}-${Date.now()}`,
            target_id: res.data.id,
            runtime_type: finalData.runtime_type,
            package_version: 'latest',
          });
          message.info('环境初始化已启动');
          navigate(`/deployment/targets/${res.data.id}/bootstrap/${bootstrapRes.data.job_id}`);
        } catch (err) {
          message.warning('环境初始化启动失败，请稍后手动启动');
          navigate(`/deployment/targets/${res.data.id}`);
        }
      } else {
        navigate(`/deployment/targets/${res.data.id}`);
      }
    } catch (err) {
      message.error(err instanceof Error ? err.message : '创建失败');
    } finally {
      setLoading(false);
    }
  };

  const renderStep0 = () => (
    <Card title="基本信息">
      <Form form={form} layout="vertical">
        <Form.Item
          name="name"
          label="目标名称"
          rules={[{ required: true, message: '请输入目标名称' }]}
          initialValue={formData.name}
        >
          <Input placeholder="例如: production-k8s-cluster" />
        </Form.Item>
        <Form.Item
          name="environment"
          label="环境"
          rules={[{ required: true, message: '请选择环境' }]}
          initialValue={formData.environment}
        >
          <Select
            options={[
              { label: '开发环境', value: 'development' },
              { label: '预发布环境', value: 'staging' },
              { label: '生产环境', value: 'production' },
            ]}
          />
        </Form.Item>
      </Form>
    </Card>
  );

  const renderStep1 = () => (
    <Card title="选择运行时">
      <Form form={form} layout="vertical">
        <Form.Item
          name="runtime_type"
          label="运行时类型"
          rules={[{ required: true, message: '请选择运行时类型' }]}
          initialValue={formData.runtime_type}
        >
          <Radio.Group>
            <Space direction="vertical">
              <Radio value="k8s">
                <div>
                  <div className="font-semibold">Kubernetes</div>
                  <div className="text-xs text-gray-500">适用于容器编排和大规模部署</div>
                </div>
              </Radio>
              <Radio value="compose">
                <div>
                  <div className="font-semibold">Docker Compose</div>
                  <div className="text-xs text-gray-500">适用于单机或小规模部署</div>
                </div>
              </Radio>
            </Space>
          </Radio.Group>
        </Form.Item>
      </Form>
    </Card>
  );

  const renderStep2 = () => {
    const runtimeType = form.getFieldValue('runtime_type') || formData.runtime_type;
    const filteredCredentials = credentials.filter((c) => c.runtime_type === runtimeType);
    const filteredClusters = clusters.filter((c) => c.runtime_type === runtimeType);

    return (
      <Card title="绑定资源">
        <Form form={form} layout="vertical">
          <Form.Item
            name="credential_id"
            label="选择凭证"
            rules={[{ required: true, message: '请选择凭证' }]}
            initialValue={formData.credential_id}
          >
            <Select
              placeholder="选择一个凭证"
              showSearch
              filterOption={(input, option) =>
                (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
              }
              options={filteredCredentials.map((c) => ({
                label: `${c.name} (${c.runtime_type})`,
                value: c.id,
              }))}
            />
          </Form.Item>

          {runtimeType === 'k8s' && (
            <>
              <Form.Item
                name="cluster_id"
                label="选择集群（可选）"
                initialValue={formData.cluster_id}
              >
                <Select
                  placeholder="选择一个集群或留空"
                  allowClear
                  showSearch
                  filterOption={(input, option) =>
                    (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
                  }
                  options={filteredClusters.map((c) => ({
                    label: `${c.name} (${c.endpoint})`,
                    value: c.id,
                  }))}
                />
              </Form.Item>
              <Form.Item
                name="namespace"
                label="命名空间"
                initialValue={formData.namespace || 'default'}
              >
                <Input placeholder="default" />
              </Form.Item>
            </>
          )}

          {runtimeType === 'compose' && (
            <Form.Item
              name="host_ids"
              label="选择主机"
              rules={[{ required: true, message: '请选择至少一台主机' }]}
              initialValue={formData.host_ids}
            >
              <Select
                mode="multiple"
                placeholder="选择一个或多个主机"
                showSearch
                filterOption={(input, option) =>
                  (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
                }
                options={hosts.map((h) => ({
                  label: `${h.name} (${h.ip})`,
                  value: Number(h.id),
                }))}
              />
            </Form.Item>
          )}
        </Form>
      </Card>
    );
  };

  const renderStep3 = () => (
    <Card title="环境初始化">
      <Form form={form} layout="vertical">
        <Alert
          message="环境初始化"
          description="创建部署目标后，可以选择立即初始化环境。初始化过程将安装必要的运行时组件和依赖。"
          type="info"
          showIcon
          className="mb-4"
        />
        <Form.Item
          name="auto_bootstrap"
          label="是否立即初始化环境？"
          initialValue={formData.auto_bootstrap}
        >
          <Radio.Group>
            <Space direction="vertical">
              <Radio value={true}>
                <div>
                  <div className="font-semibold">立即初始化</div>
                  <div className="text-xs text-gray-500">创建后自动启动环境初始化流程</div>
                </div>
              </Radio>
              <Radio value={false}>
                <div>
                  <div className="font-semibold">稍后初始化</div>
                  <div className="text-xs text-gray-500">创建后手动启动初始化</div>
                </div>
              </Radio>
            </Space>
          </Radio.Group>
        </Form.Item>
      </Form>
    </Card>
  );

  const steps = [
    { title: '基本信息', content: renderStep0() },
    { title: '运行时', content: renderStep1() },
    { title: '绑定资源', content: renderStep2() },
    { title: '环境初始化', content: renderStep3() },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-gray-900">创建部署目标</h1>
        <p className="text-sm text-gray-500 mt-1">配置应用部署的目标环境</p>
      </div>

      <Steps current={currentStep} items={steps.map(item => ({ title: item.title }))} />

      <div className="flex justify-between">
        <Button onClick={() => navigate('/deployment/targets')}>取消</Button>
        <Space>
          {currentStep > 0 && <Button onClick={handlePrev}>上一步</Button>}
          {currentStep < steps.length - 1 && (
            <Button type="primary" onClick={handleNext}>
              下一步
            </Button>
          )}
          {currentStep === steps.length - 1 && (
            <Button type="primary" onClick={handleSubmit} loading={loading}>
              创建目标
            </Button>
          )}
        </Space>
      </div>
    </div>
  );
};

export default CreateTargetWizard;
