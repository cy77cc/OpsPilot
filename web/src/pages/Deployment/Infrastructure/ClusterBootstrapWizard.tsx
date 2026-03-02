import React, { useState } from 'react';
import { Steps, Form, Input, Select, Button, Card, Space, message } from 'antd';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../../api';



interface BootstrapFormData {
  name: string;
  control_plane_host_id?: number;
  worker_host_ids?: number[];
  cni?: string;
}

const ClusterBootstrapWizard: React.FC = () => {
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [currentStep, setCurrentStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState<BootstrapFormData>({
    name: '',
    cni: 'calico',
  });
  const [hosts, setHosts] = useState<Array<{ id: string; name: string; ip: string }>>([]);

  React.useEffect(() => {
    loadHosts();
  }, []);

  const loadHosts = async () => {
    try {
      const res = await Api.hosts.getHostList();
      setHosts(res.data.list || []);
    } catch (err) {
      message.error('加载主机列表失败');
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

      const res = await Api.deployment.applyClusterBootstrap({
        name: finalData.name,
        control_plane_host_id: finalData.control_plane_host_id!,
        worker_host_ids: finalData.worker_host_ids,
        cni: finalData.cni,
      });

      message.success('集群创建任务已提交');
      navigate(`/deployment/infrastructure/clusters/bootstrap/${res.data.task_id}`);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '创建集群失败');
    } finally {
      setLoading(false);
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
        <div className="p-4 bg-blue-50 rounded-lg">
          <p className="text-sm text-blue-800">
            <strong>提示:</strong> Control Plane 节点将运行 Kubernetes 控制平面组件（API Server, Controller Manager, Scheduler, etcd）。
            建议选择资源充足的主机。
          </p>
        </div>
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
        <div className="p-4 bg-blue-50 rounded-lg">
          <p className="text-sm text-blue-800">
            <strong>提示:</strong> Worker 节点将运行应用工作负载。可以稍后添加更多 Worker 节点。
          </p>
        </div>
      </Form>
    </Card>
  );

  const renderStep3 = () => (
    <Card title="配置 CNI 网络插件">
      <Form form={form} layout="vertical">
        <Form.Item
          name="cni"
          label="CNI 插件"
          rules={[{ required: true, message: '请选择 CNI 插件' }]}
          initialValue={formData.cni}
        >
          <Select
            options={[
              { label: 'Calico (推荐)', value: 'calico' },
              { label: 'Flannel', value: 'flannel' },
              { label: 'Weave Net', value: 'weave' },
              { label: 'Cilium', value: 'cilium' },
            ]}
          />
        </Form.Item>
        <div className="p-4 bg-blue-50 rounded-lg">
          <p className="text-sm text-blue-800">
            <strong>Calico:</strong> 功能丰富，支持网络策略，适合生产环境。<br />
            <strong>Flannel:</strong> 简单易用，适合开发和测试环境。
          </p>
        </div>
      </Form>
    </Card>
  );

  const steps = [
    { title: '基本信息', content: renderStep0() },
    { title: 'Control Plane', content: renderStep1() },
    { title: 'Worker 节点', content: renderStep2() },
    { title: 'CNI 配置', content: renderStep3() },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-gray-900">创建 Kubernetes 集群</h1>
        <p className="text-sm text-gray-500 mt-1">通过自动化 Bootstrap 创建新集群</p>
      </div>

      <Steps current={currentStep} items={steps.map(item => ({ title: item.title }))} />

      <div className="flex justify-between">
        <Button onClick={() => navigate('/deployment/infrastructure/clusters')}>
          取消
        </Button>
        <Space>
          {currentStep > 0 && (
            <Button onClick={handlePrev}>
              上一步
            </Button>
          )}
          {currentStep < steps.length - 1 && (
            <Button type="primary" onClick={handleNext}>
              下一步
            </Button>
          )}
          {currentStep === steps.length - 1 && (
            <Button type="primary" onClick={handleSubmit} loading={loading}>
              创建集群
            </Button>
          )}
        </Space>
      </div>
    </div>
  );
};

export default ClusterBootstrapWizard;
