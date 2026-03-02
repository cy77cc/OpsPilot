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
  Radio,
  Divider,
} from 'antd';
import {
  ArrowLeftOutlined,
  RocketOutlined,
  WarningOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../api';
import type { ServiceItem } from '../../api/modules/services';
import type { DeployTarget } from '../../api/modules/deployment';



const EnhancedDeploymentCreatePage: React.FC = () => {
  const navigate = useNavigate();
  const [currentStep, setCurrentStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [services, setServices] = useState<ServiceItem[]>([]);
  const [targets, setTargets] = useState<DeployTarget[]>([]);
  const [previewData, setPreviewData] = useState<any>(null);
  const [previewToken, setPreviewToken] = useState('');
  const [searchService, setSearchService] = useState('');
  const [searchTarget, setSearchTarget] = useState('');

  const [form] = Form.useForm();

  // Load services and targets
  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [servicesRes, targetsRes] = await Promise.all([
        Api.services.getList({ page: 1, pageSize: 500 }),
        Api.deployment.listTargets(),
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

  // Step 1: Service and target selection
  const renderStep1 = () => {
    const filteredServices = services.filter((s) =>
      s.name.toLowerCase().includes(searchService.toLowerCase())
    );
    const filteredTargets = targets.filter((t) =>
      t.name.toLowerCase().includes(searchTarget.toLowerCase()) ||
      t.environment.toLowerCase().includes(searchTarget.toLowerCase())
    );

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
              placeholder="搜索并选择服务"
              size="large"
              onSearch={setSearchService}
              filterOption={false}
              options={filteredServices.map((s) => ({
                value: s.id,
                label: `${s.name} - ${s.runtimeType}`,
              }))}
            />
          </Form.Item>

          <Form.Item
            name="target_id"
            label="部署目标"
            rules={[{ required: true, message: '请选择部署目标' }]}
          >
            <Select
              showSearch
              placeholder="搜索并选择部署目标"
              size="large"
              onSearch={setSearchTarget}
              filterOption={false}
              options={filteredTargets.map((t) => ({
                value: t.id,
                label: `${t.name} [${t.environment}] - ${t.runtime_type}`,
              }))}
            />
          </Form.Item>
        </Form>
      </Card>
    );
  };

  // Step 2: Variable configuration
  const renderStep2 = () => {
    return (
      <Card title="配置变量">
        <Alert
          message="模板变量检测"
          description="系统将自动检测服务模板中的变量，您可以在下方配置这些变量的值。"
          type="info"
          showIcon
          className="mb-4"
        />
        <Form form={form} layout="vertical">
          <Form.Item
            name="variables"
            label="部署变量 (JSON)"
            extra="示例: {&quot;image_tag&quot;:&quot;v1.2.3&quot;,&quot;replicas&quot;:&quot;3&quot;}"
          >
            <Input.TextArea
              rows={10}
              placeholder={'{\n  "image_tag": "v1.2.3",\n  "replicas": "3"\n}'}
            />
          </Form.Item>
        </Form>
      </Card>
    );
  };

  // Step 3: Manifest preview
  const renderStep3 = () => {
    const handlePreview = async () => {
      try {
        setLoading(true);
        const values = form.getFieldsValue();
        let variables = {};
        if (values.variables) {
          try {
            variables = JSON.parse(values.variables);
          } catch {
            message.error('变量 JSON 格式错误');
            return;
          }
        }

        const res = await Api.deployment.previewRelease({
          service_id: values.service_id,
          target_id: values.target_id,
          variables,
        });

        setPreviewData(res.data);
        setPreviewToken(res.data.preview_token || '');
      } catch (err) {
        message.error(err instanceof Error ? err.message : '预览失败');
      } finally {
        setLoading(false);
      }
    };

    return (
      <Card title="清单预览">
        {!previewData ? (
          <div className="text-center py-8">
            <Button type="primary" size="large" onClick={handlePreview} loading={loading}>
              生成预览
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            {/* Checks and warnings */}
            {previewData.checks && previewData.checks.length > 0 && (
              <Alert
                message="检查结果"
                description={
                  <div className="space-y-1">
                    {previewData.checks.map((check: any, idx: number) => (
                      <div key={idx} className="text-sm">
                        <Tag color={check.level === 'error' ? 'error' : 'warning'}>
                          {check.level}
                        </Tag>
                        {check.message}
                      </div>
                    ))}
                  </div>
                }
                type={previewData.checks.some((c: any) => c.level === 'error') ? 'error' : 'warning'}
                showIcon
              />
            )}

            {previewData.warnings && previewData.warnings.length > 0 && (
              <Alert
                message="警告"
                description={
                  <div className="space-y-1">
                    {previewData.warnings.map((warn: any, idx: number) => (
                      <div key={idx} className="text-sm">
                        {warn.message}
                      </div>
                    ))}
                  </div>
                }
                type="warning"
                showIcon
              />
            )}

            {/* Manifest */}
            <div>
              <div className="text-sm font-semibold mb-2">生成的清单:</div>
              <pre className="bg-gray-900 text-gray-100 p-4 rounded overflow-auto max-h-96 text-xs">
                {previewData.resolved_yaml || previewData.manifest || '无清单内容'}
              </pre>
            </div>

            <Button onClick={handlePreview} loading={loading}>
              重新生成预览
            </Button>
          </div>
        )}
      </Card>
    );
  };

  // Step 4: Deployment strategy
  const renderStep4 = () => {
    return (
      <Card title="部署策略">
        <Form form={form} layout="vertical">
          <Form.Item
            name="strategy"
            label="选择部署策略"
            initialValue="rolling"
            rules={[{ required: true }]}
          >
            <Radio.Group>
              <Space direction="vertical" className="w-full">
                <Radio value="rolling">
                  <div>
                    <div className="font-semibold">Rolling Update - 滚动更新</div>
                    <div className="text-xs text-gray-500">逐步替换旧版本实例，适合大多数场景</div>
                  </div>
                </Radio>
                <Radio value="blue-green">
                  <div>
                    <div className="font-semibold">Blue-Green - 蓝绿部署</div>
                    <div className="text-xs text-gray-500">部署新版本后一次性切换流量，可快速回滚</div>
                  </div>
                </Radio>
                <Radio value="canary">
                  <div>
                    <div className="font-semibold">Canary - 金丝雀发布</div>
                    <div className="text-xs text-gray-500">先发布到少量实例，验证后再全量发布</div>
                  </div>
                </Radio>
              </Space>
            </Radio.Group>
          </Form.Item>
        </Form>
      </Card>
    );
  };

  // Step 5: Confirmation
  const renderStep5 = () => {
    const values = form.getFieldsValue();
    const service = services.find((s) => s.id === values.service_id);
    const target = targets.find((t) => t.id === values.target_id);
    const isProduction = target?.environment === 'production';

    return (
      <Card title="确认发布">
        <Descriptions column={1} bordered>
          <Descriptions.Item label="服务">{service?.name}</Descriptions.Item>
          <Descriptions.Item label="部署目标">
            {target?.name}
            <Tag color={isProduction ? 'red' : 'blue'} className="ml-2">
              {target?.environment}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="运行时">
            <Tag color={target?.runtime_type === 'k8s' ? 'blue' : 'green'}>
              {target?.runtime_type === 'k8s' ? 'Kubernetes' : 'Docker Compose'}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="部署策略">
            {values.strategy === 'rolling' && 'Rolling Update - 滚动更新'}
            {values.strategy === 'blue-green' && 'Blue-Green - 蓝绿部署'}
            {values.strategy === 'canary' && 'Canary - 金丝雀发布'}
          </Descriptions.Item>
        </Descriptions>

        {isProduction && (
          <Alert
            message="生产环境部署警告"
            description="您正在向生产环境部署，此操作需要审批。发布将进入待审批状态，需要相关人员批准后才会执行。"
            type="warning"
            showIcon
            icon={<WarningOutlined />}
            className="mt-4"
          />
        )}
      </Card>
    );
  };

  const handleNext = async () => {
    try {
      await form.validateFields();
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
      let variables = {};
      if (values.variables) {
        try {
          variables = JSON.parse(values.variables);
        } catch {
          message.error('变量 JSON 格式错误');
          return;
        }
      }

      const res = await Api.deployment.applyRelease({
        service_id: values.service_id,
        target_id: values.target_id,
        variables,
        strategy: values.strategy || 'rolling',
        preview_token: previewToken,
      });

      message.success('发布已创建');
      navigate(`/deployment/${res.data.release_id}`);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '创建发布失败');
    } finally {
      setLoading(false);
    }
  };

  const steps = [
    { title: '选择服务', content: renderStep1() },
    { title: '配置变量', content: renderStep2() },
    { title: '清单预览', content: renderStep3() },
    { title: '部署策略', content: renderStep4() },
    { title: '确认发布', content: renderStep5() },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/deployment')}>
            返回
          </Button>
          <div>
            <h1 className="text-2xl font-semibold text-gray-900">创建发布</h1>
            <p className="text-sm text-gray-500 mt-1">配置并部署服务到目标环境</p>
          </div>
        </div>
      </div>

      <Steps current={currentStep} items={steps.map(item => ({ title: item.title }))} />

      <div className="mt-6">{steps[currentStep].content}</div>

      <div className="flex justify-between">
        <Button onClick={() => navigate('/deployment')}>取消</Button>
        <Space>
          {currentStep > 0 && <Button onClick={handlePrev}>上一步</Button>}
          {currentStep < steps.length - 1 && (
            <Button type="primary" onClick={handleNext}>
              下一步
            </Button>
          )}
          {currentStep === steps.length - 1 && (
            <Button type="primary" icon={<RocketOutlined />} onClick={handleSubmit} loading={loading}>
              创建发布
            </Button>
          )}
        </Space>
      </div>
    </div>
  );
};

export default EnhancedDeploymentCreatePage;
