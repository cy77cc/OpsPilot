import React, { useState, useCallback, useEffect } from 'react';
import {
  Button,
  Card,
  Steps,
  Form,
  Select,
  Input,
  Space,
  message,
  Alert,
  Descriptions,
  Tag,
} from 'antd';
import {
  ArrowLeftOutlined,
  ArrowRightOutlined,
  CloudUploadOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../api';
import type { ServiceItem } from '../../api/modules/services';
import type { DeployTarget } from '../../api/modules/deployment';

const DeploymentCreatePage: React.FC = () => {
  const navigate = useNavigate();
  const [currentStep, setCurrentStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [services, setServices] = useState<ServiceItem[]>([]);
  const [targets, setTargets] = useState<DeployTarget[]>([]);
  const [previewManifest, setPreviewManifest] = useState('');
  const [previewWarnings, setPreviewWarnings] = useState<Array<{ code: string; message: string; level: string }>>([]);
  const [previewToken, setPreviewToken] = useState('');

  const [form] = Form.useForm();

  // 加载服务和目标列表
  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [servicesRes, targetsRes] = await Promise.all([
        Api.services.getList({ page: 1, pageSize: 500 }),
        Api.deployment.getTargets(),
      ]);
      setServices(servicesRes.data.list || []);
      setTargets(targetsRes.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载数据失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  // 步骤 1: 选择服务和环境
  const renderStep1 = () => {
    const serviceOptions = services.map((s) => ({
      value: s.id,
      label: `${s.name} (#${s.id}) - ${s.env}`,
    }));

    const targetOptions = targets.map((t) => ({
      value: t.id,
      label: `${t.name} [${t.target_type}] - ${t.env}`,
    }));

    return (
      <Card title="选择服务和部署目标">
        <Form form={form} layout="vertical">
          <Form.Item
            name="service_id"
            label="服务"
            rules={[{ required: true, message: '请选择服务' }]}
          >
            <Select
              showSearch
              placeholder="选择要部署的服务"
              options={serviceOptions}
              optionFilterProp="label"
              size="large"
            />
          </Form.Item>

          <Form.Item
            name="target_id"
            label="部署目标"
            rules={[{ required: true, message: '请选择部署目标' }]}
          >
            <Select
              showSearch
              placeholder="选择部署目标"
              options={targetOptions}
              optionFilterProp="label"
              size="large"
            />
          </Form.Item>

          <Form.Item
            name="env"
            label="环境"
            initialValue="staging"
            rules={[{ required: true }]}
          >
            <Select
              size="large"
              options={[
                { value: 'development', label: 'Development' },
                { value: 'staging', label: 'Staging' },
                { value: 'production', label: 'Production' },
              ]}
            />
          </Form.Item>
        </Form>
      </Card>
    );
  };

  // 步骤 2: 选择版本和策略
  const renderStep2 = () => {
    return (
      <Card title="选择部署策略">
        <Form form={form} layout="vertical">
          <Form.Item
            name="strategy"
            label="部署策略"
            initialValue="rolling"
            rules={[{ required: true }]}
          >
            <Select
              size="large"
              options={[
                {
                  value: 'rolling',
                  label: 'Rolling Update - 滚动更新',
                },
                {
                  value: 'blue-green',
                  label: 'Blue-Green - 蓝绿部署',
                },
                {
                  value: 'canary',
                  label: 'Canary - 金丝雀发布',
                },
              ]}
            />
          </Form.Item>

          <Alert
            type="info"
            showIcon
            message="部署策略说明"
            description={
              <div className="space-y-2 text-sm">
                <p>
                  <strong>Rolling Update:</strong> 逐步替换旧版本实例，适合大多数场景
                </p>
                <p>
                  <strong>Blue-Green:</strong> 部署新版本后一次性切换流量，可快速回滚
                </p>
                <p>
                  <strong>Canary:</strong> 先发布到少量实例，验证后再全量发布
                </p>
              </div>
            }
          />
        </Form>
      </Card>
    );
  };

  // 步骤 3: 配置参数
  const renderStep3 = () => {
    const targetId = form.getFieldValue('target_id');
    const target = targets.find((t) => t.id === targetId);

    return (
      <Card title="配置部署参数">
        <Form form={form} layout="vertical">
          <Form.Item
            name="variables_json"
            label={`部署变量 (JSON) - ${target?.target_type === 'k8s' ? 'Kubernetes' : 'Compose'}`}
            extra={
              target?.target_type === 'k8s'
                ? '示例: {"image_tag":"v1.2.3","replicas":"3"}'
                : '示例: {"COMPOSE_PROJECT_NAME":"svc-a","IMAGE_TAG":"v1.2.3"}'
            }
          >
            <Input.TextArea
              rows={8}
              placeholder={
                target?.target_type === 'k8s'
                  ? '{\n  "image_tag": "v1.2.3",\n  "replicas": "3"\n}'
                  : '{\n  "COMPOSE_PROJECT_NAME": "svc-a",\n  "IMAGE_TAG": "v1.2.3"\n}'
              }
            />
          </Form.Item>

          <Button type="primary" onClick={handlePreview} loading={loading}>
            预览部署清单
          </Button>

          {previewWarnings.length > 0 && (
            <div className="mt-4 space-y-2">
              {previewWarnings.map((w, idx) => (
                <Alert
                  key={`${w.code}-${idx}`}
                  type={w.level === 'error' ? 'error' : 'warning'}
                  showIcon
                  message={w.message}
                />
              ))}
            </div>
          )}

          {previewManifest && (
            <Card size="small" title="部署清单预览" className="mt-4">
              <pre className="bg-gray-50 p-4 rounded-lg text-sm overflow-auto max-h-96">
                {previewManifest}
              </pre>
            </Card>
          )}
        </Form>
      </Card>
    );
  };

  // 步骤 4: 确认部署
  const renderStep4 = () => {
    const serviceId = form.getFieldValue('service_id');
    const targetId = form.getFieldValue('target_id');
    const env = form.getFieldValue('env');
    const strategy = form.getFieldValue('strategy');

    const service = services.find((s) => s.id === serviceId);
    const target = targets.find((t) => t.id === targetId);

    return (
      <Card title="确认部署信息">
        <Descriptions bordered column={1}>
          <Descriptions.Item label="服务">
            {service?.name} (#{service?.id})
          </Descriptions.Item>
          <Descriptions.Item label="部署目标">
            {target?.name} [{target?.target_type}]
          </Descriptions.Item>
          <Descriptions.Item label="环境">
            <Tag>{env}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="部署策略">
            <Tag color="blue">{strategy}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="运行时">
            <Tag color="purple">{target?.runtime_type}</Tag>
          </Descriptions.Item>
        </Descriptions>

        {!previewToken && (
          <Alert
            type="warning"
            showIcon
            message="请先在上一步执行预览操作"
            className="mt-4"
          />
        )}

        {previewToken && (
          <Alert
            type="success"
            showIcon
            message="预览已完成，可以执行部署"
            className="mt-4"
          />
        )}
      </Card>
    );
  };

  // 预览部署
  const handlePreview = async () => {
    try {
      const values = await form.validateFields(['service_id', 'target_id', 'env', 'strategy', 'variables_json']);

      let variables: Record<string, string> = {};
      if (values.variables_json) {
        try {
          const parsed = JSON.parse(values.variables_json);
          if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
            Object.entries(parsed).forEach(([k, v]) => {
              variables[k] = String(v ?? '');
            });
          }
        } catch {
          message.error('变量 JSON 格式错误');
          return;
        }
      }

      setLoading(true);
      const resp = await Api.deployment.previewRelease({
        service_id: Number(values.service_id),
        target_id: Number(values.target_id),
        env: values.env,
        strategy: values.strategy,
        variables,
      });

      setPreviewManifest(resp.data.resolved_manifest || '');
      setPreviewToken(resp.data.preview_token || '');
      setPreviewWarnings(resp.data.warnings || []);
      message.success('预览成功');
    } catch (err) {
      message.error(err instanceof Error ? err.message : '预览失败');
    } finally {
      setLoading(false);
    }
  };

  // 执行部署
  const handleDeploy = async () => {
    if (!previewToken) {
      message.warning('请先执行预览操作');
      return;
    }

    try {
      const values = await form.validateFields();

      let variables: Record<string, string> = {};
      if (values.variables_json) {
        try {
          const parsed = JSON.parse(values.variables_json);
          if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
            Object.entries(parsed).forEach(([k, v]) => {
              variables[k] = String(v ?? '');
            });
          }
        } catch {
          message.error('变量 JSON 格式错误');
          return;
        }
      }

      setLoading(true);
      const resp = await Api.deployment.applyRelease({
        service_id: Number(values.service_id),
        target_id: Number(values.target_id),
        env: values.env,
        strategy: values.strategy,
        variables,
        preview_token: previewToken,
      });

      if (resp.data.approval_required) {
        message.warning(
          `部署已提交审批，Release #${resp.data.release_id}，Ticket: ${resp.data.approval_ticket || '-'}`
        );
      } else {
        message.success(`部署已执行，Release #${resp.data.release_id}`);
      }

      navigate('/deployment');
    } catch (err) {
      message.error(err instanceof Error ? err.message : '部署失败');
    } finally {
      setLoading(false);
    }
  };

  // 下一步
  const handleNext = async () => {
    try {
      if (currentStep === 0) {
        await form.validateFields(['service_id', 'target_id', 'env']);
      } else if (currentStep === 1) {
        await form.validateFields(['strategy']);
      }
      setCurrentStep(currentStep + 1);
    } catch {
      // 验证失败，不执行任何操作
    }
  };

  // 上一步
  const handlePrev = () => {
    setCurrentStep(currentStep - 1);
  };

  const steps = [
    { title: '选择服务', description: '选择要部署的服务和目标' },
    { title: '选择策略', description: '选择部署策略' },
    { title: '配置参数', description: '配置部署变量和预览' },
    { title: '确认部署', description: '确认信息并执行部署' },
  ];

  return (
    <div className="space-y-6">
      {/* 页面头部 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">创建部署</h1>
          <p className="text-sm text-gray-500 mt-1">按照步骤完成部署配置</p>
        </div>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/deployment')}>
          返回列表
        </Button>
      </div>

      {/* 步骤指示器 */}
      <Card>
        <Steps current={currentStep} items={steps} />
      </Card>

      {/* 步骤内容 */}
      <div className="min-h-96">
        {currentStep === 0 && renderStep1()}
        {currentStep === 1 && renderStep2()}
        {currentStep === 2 && renderStep3()}
        {currentStep === 3 && renderStep4()}
      </div>

      {/* 操作按钮 */}
      <Card>
        <div className="flex justify-between">
          <Button
            size="large"
            onClick={handlePrev}
            disabled={currentStep === 0}
            icon={<ArrowLeftOutlined />}
          >
            上一步
          </Button>

          <Space>
            <Button size="large" onClick={() => navigate('/deployment')}>
              取消
            </Button>
            {currentStep < steps.length - 1 ? (
              <Button
                type="primary"
                size="large"
                onClick={handleNext}
                icon={<ArrowRightOutlined />}
              >
                下一步
              </Button>
            ) : (
              <Button
                type="primary"
                size="large"
                onClick={handleDeploy}
                loading={loading}
                disabled={!previewToken}
                icon={<CloudUploadOutlined />}
              >
                执行部署
              </Button>
            )}
          </Space>
        </div>
      </Card>
    </div>
  );
};

export default DeploymentCreatePage;
