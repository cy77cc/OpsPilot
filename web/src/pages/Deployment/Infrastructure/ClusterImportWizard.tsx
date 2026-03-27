import React, { useState } from 'react';
import {
  Steps, Form, Input, Button, Card, Space, message, Upload, Alert, Result,
  Radio, Spin, Descriptions, Tag, Divider, Typography
} from 'antd';
import {
  ArrowLeftOutlined, UploadOutlined, CheckCircleOutlined, CloseCircleOutlined,
  LoadingOutlined, FileTextOutlined, SafetyCertificateOutlined, KeyOutlined,
  ApiOutlined, InfoCircleOutlined
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../../api';
import type { Cluster, ClusterImportReq } from '../../../api/modules/cluster';

const { TextArea } = Input;
const { Text } = Typography;

type AuthMethod = 'kubeconfig' | 'certificate' | 'token';

interface ValidationResult {
  valid: boolean;
  message: string;
  endpoint?: string;
  version?: string;
}

interface FormData {
  name: string;
  description?: string;
  auth_method: AuthMethod;
  kubeconfig?: string;
  endpoint?: string;
  ca_cert?: string;
  cert?: string;
  key?: string;
  token?: string;
  skip_tls_verify?: boolean;
}

const authMethodConfig = {
  kubeconfig: {
    icon: <FileTextOutlined />,
    title: 'Kubeconfig 文件',
    description: '最简单的方式，适合个人开发环境',
  },
  certificate: {
    icon: <SafetyCertificateOutlined />,
    title: 'API 地址 + 证书',
    description: '企业级安全，适合生产环境',
  },
  token: {
    icon: <KeyOutlined />,
    title: 'ServiceAccount Token',
    description: '适合 CI/CD 或受限访问场景',
  },
};

const ClusterImportWizard: React.FC = () => {
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [messageApi, messageContextHolder] = message.useMessage();
  const [currentStep, setCurrentStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [validating, setValidating] = useState(false);
  const [authMethod, setAuthMethod] = useState<AuthMethod>('kubeconfig');
  const [validationResult, setValidationResult] = useState<ValidationResult | null>(null);
  const [importedCluster, setImportedCluster] = useState<Cluster | null>(null);
  const watchedName = Form.useWatch('name', form);
  const watchedKubeconfig = Form.useWatch('kubeconfig', form);
  const watchedEndpoint = Form.useWatch('endpoint', form);
  const watchedCaCert = Form.useWatch('ca_cert', form);
  const watchedCert = Form.useWatch('cert', form);
  const watchedKey = Form.useWatch('key', form);
  const watchedToken = Form.useWatch('token', form);

  const handleFileUpload = (file: File, field: string) => {
    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      form.setFieldsValue({ [field]: content });
      messageApi.success('文件已加载');
    };
    reader.onerror = () => {
      messageApi.error('读取文件失败');
    };
    reader.readAsText(file);
    return false;
  };

  const isBlank = (value: unknown): boolean => typeof value !== 'string' || value.trim() === '';

  const buildValidatePayload = (values: Record<string, any>) => {
    const payload: Record<string, string | boolean | undefined> = {};
    payload.name = values.name;
    payload.auth_method = authMethod;

    switch (authMethod) {
      case 'kubeconfig':
        payload.kubeconfig = values.kubeconfig;
        break;
      case 'certificate':
        payload.endpoint = values.endpoint;
        payload.ca_cert = values.ca_cert;
        payload.cert = values.cert;
        payload.key = values.key;
        break;
      case 'token':
        payload.endpoint = values.endpoint;
        payload.ca_cert = values.ca_cert;
        payload.token = values.token;
        payload.skip_tls_verify = values.skip_tls_verify;
        break;
    }

    return payload;
  };

  const handleValidate = async () => {
    try {
      const values = form.getFieldsValue(true);
      const payload = buildValidatePayload(values);

      // 基本验证
      if (isBlank(values.name)) {
        messageApi.error('请输入集群名称');
        return;
      }
      if (authMethod === 'kubeconfig' && isBlank(values.kubeconfig)) {
        messageApi.error('请输入或上传 kubeconfig');
        return;
      }
      if ((authMethod === 'certificate' || authMethod === 'token') && isBlank(values.endpoint)) {
        messageApi.error('请输入 API Server 地址');
        return;
      }

      setValidating(true);
      const res = await Api.cluster.validateImport(payload);
      setValidationResult(res.data);

      if (res.data.valid) {
        messageApi.success('连接验证成功');
      } else {
        messageApi.error(res.data.message);
      }
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : '验证失败';
      messageApi.error(errorMsg);
      setValidationResult({ valid: false, message: errorMsg });
    } finally {
      setValidating(false);
    }
  };

  const handleImport = async () => {
    try {
      setLoading(true);
      const values = form.getFieldsValue(true);
      if (isBlank(values.name)) {
        messageApi.error('请输入集群名称');
        return;
      }

      const payload: ClusterImportReq = {
        name: values.name,
        description: values.description,
        auth_method: authMethod,
      };

      switch (authMethod) {
        case 'kubeconfig':
          if (isBlank(values.kubeconfig)) {
            messageApi.error('请输入或上传 kubeconfig');
            return;
          }
          payload.kubeconfig = values.kubeconfig;
          break;
        case 'certificate':
          if (isBlank(values.endpoint) || isBlank(values.ca_cert) || isBlank(values.cert) || isBlank(values.key)) {
            messageApi.error('请完整填写证书认证所需参数');
            return;
          }
          payload.endpoint = values.endpoint;
          payload.ca_cert = values.ca_cert;
          payload.cert = values.cert;
          payload.key = values.key;
          break;
        case 'token':
          if (isBlank(values.endpoint) || isBlank(values.token)) {
            messageApi.error('请完整填写 Token 认证所需参数');
            return;
          }
          payload.endpoint = values.endpoint;
          payload.ca_cert = values.ca_cert;
          payload.token = values.token;
          payload.skip_tls_verify = values.skip_tls_verify;
          break;
      }

      const res = await Api.cluster.importCluster(payload);
      setImportedCluster(res.data);
      setCurrentStep(5);
      messageApi.success('集群导入成功');
    } catch (err) {
      messageApi.error(err instanceof Error ? err.message : '导入失败');
    } finally {
      setLoading(false);
    }
  };

  // Step 0: 基本信息
  const renderStep0 = () => (
    <Card title="基本信息">
      <Form form={form} layout="vertical">
        <Form.Item
          name="name"
          label="集群名称"
          rules={[{ required: true, message: '请输入集群名称' }]}
        >
          <Input placeholder="例如: production-k8s" />
        </Form.Item>
        <Form.Item name="description" label="描述">
          <TextArea rows={2} placeholder="集群描述（可选）" />
        </Form.Item>
      </Form>
    </Card>
  );

  // Step 1: 认证方式选择
  const renderStep1 = () => (
    <Card title="选择认证方式">
      <Form form={form} layout="vertical">
        <Form.Item name="auth_method" initialValue={authMethod}>
          <Radio.Group
            onChange={(e) => setAuthMethod(e.target.value)}
            className="w-full"
          >
            <Space orientation="vertical" className="w-full" size="middle">
              {(Object.keys(authMethodConfig) as AuthMethod[]).map((method) => {
                const config = authMethodConfig[method];
                return (
                  <Radio
                    key={method}
                    value={method}
                    className="w-full"
                  >
                    <div className="flex items-start gap-3 p-3 border rounded hover:bg-gray-50 cursor-pointer">
                      <span className="text-xl text-blue-500 mt-1">{config.icon}</span>
                      <div>
                        <div className="font-medium">{config.title}</div>
                        <Text type="secondary">{config.description}</Text>
                      </div>
                    </div>
                  </Radio>
                );
              })}
            </Space>
          </Radio.Group>
        </Form.Item>
      </Form>

      <Alert
        type="info"
        className="mt-4"
        title="如何选择？"
        description={
          <ul className="list-disc pl-4 space-y-1">
            <li><strong>Kubeconfig</strong>: 从 ~/.kube/config 复制或上传文件</li>
            <li><strong>证书</strong>: 需要准备 CA 证书、客户端证书和私钥</li>
            <li><strong>Token</strong>: 创建 ServiceAccount 并获取其 token</li>
          </ul>
        }
      />
    </Card>
  );

  // Step 2: 连接配置 (根据认证方式动态渲染)
  const renderStep2 = () => {
    const renderKubeconfigForm = () => (
      <Form form={form} layout="vertical">
        <Form.Item
          name="kubeconfig"
          label="Kubeconfig 内容"
          rules={[{ required: true, message: '请输入或上传 kubeconfig' }]}
        >
          <TextArea
            rows={12}
            placeholder="粘贴 kubeconfig 内容，或点击下方按钮上传文件"
            style={{ fontFamily: 'monospace', fontSize: '12px' }}
          />
        </Form.Item>
        <Upload
          beforeUpload={(file) => handleFileUpload(file, 'kubeconfig')}
          accept=".yaml,.yml,.conf,.config"
          showUploadList={false}
        >
          <Button icon={<UploadOutlined />}>上传 kubeconfig 文件</Button>
        </Upload>
      </Form>
    );

    const renderCertificateForm = () => (
      <Form form={form} layout="vertical">
        <Form.Item
          name="endpoint"
          label="API Server 地址"
          rules={[{ required: true, message: '请输入 API Server 地址' }]}
          extra="例如: https://api.k8s.example.com:6443"
        >
          <Input placeholder="https://api.k8s.example.com:6443" />
        </Form.Item>

        <Divider>证书配置</Divider>

        <Form.Item
          name="ca_cert"
          label="CA 证书"
          rules={[{ required: true, message: '请输入 CA 证书' }]}
          extra="PEM 格式或 Base64 编码"
        >
          <TextArea
            rows={4}
            placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
            style={{ fontFamily: 'monospace', fontSize: '12px' }}
          />
        </Form.Item>
        <Upload
          beforeUpload={(file) => handleFileUpload(file, 'ca_cert')}
          accept=".pem,.crt,.cert"
          showUploadList={false}
        >
          <Button icon={<UploadOutlined />} size="small">上传 CA 证书</Button>
        </Upload>

        <Form.Item
          name="cert"
          label="客户端证书"
          rules={[{ required: true, message: '请输入客户端证书' }]}
          className="mt-4"
        >
          <TextArea
            rows={4}
            placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
            style={{ fontFamily: 'monospace', fontSize: '12px' }}
          />
        </Form.Item>
        <Upload
          beforeUpload={(file) => handleFileUpload(file, 'cert')}
          accept=".pem,.crt,.cert"
          showUploadList={false}
        >
          <Button icon={<UploadOutlined />} size="small">上传客户端证书</Button>
        </Upload>

        <Form.Item
          name="key"
          label="客户端私钥"
          rules={[{ required: true, message: '请输入客户端私钥' }]}
          className="mt-4"
        >
          <TextArea
            rows={4}
            placeholder="-----BEGIN RSA PRIVATE KEY-----&#10;...&#10;-----END RSA PRIVATE KEY-----"
            style={{ fontFamily: 'monospace', fontSize: '12px' }}
          />
        </Form.Item>
        <Upload
          beforeUpload={(file) => handleFileUpload(file, 'key')}
          accept=".pem,.key"
          showUploadList={false}
        >
          <Button icon={<UploadOutlined />} size="small">上传私钥文件</Button>
        </Upload>
      </Form>
    );

    const renderTokenForm = () => (
      <Form form={form} layout="vertical">
        <Form.Item
          name="endpoint"
          label="API Server 地址"
          rules={[{ required: true, message: '请输入 API Server 地址' }]}
          extra="例如: https://api.k8s.example.com:6443"
        >
          <Input placeholder="https://api.k8s.example.com:6443" />
        </Form.Item>

        <Form.Item
          name="ca_cert"
          label="CA 证书（可选）"
          extra="不提供时可以选择跳过 TLS 验证"
        >
          <TextArea
            rows={4}
            placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
            style={{ fontFamily: 'monospace', fontSize: '12px' }}
          />
        </Form.Item>
        <Upload
          beforeUpload={(file) => handleFileUpload(file, 'ca_cert')}
          accept=".pem,.crt,.cert"
          showUploadList={false}
        >
          <Button icon={<UploadOutlined />} size="small">上传 CA 证书</Button>
        </Upload>

        <Form.Item
          name="token"
          label="Bearer Token"
          rules={[{ required: true, message: '请输入 Token' }]}
          className="mt-4"
          extra="可通过 kubectl create token 或查看 Secret 获取"
        >
          <TextArea
            rows={4}
            placeholder="eyJhbGciOiJSUzI1NiIsImtpZCI6Ii..."
            style={{ fontFamily: 'monospace', fontSize: '12px' }}
          />
        </Form.Item>

        <Form.Item name="skip_tls_verify" valuePropName="checked" initialValue={false} className="mt-2">
          <Space>
            <input type="checkbox" className="w-4 h-4" />
            <span>跳过 TLS 证书验证（不推荐）</span>
          </Space>
        </Form.Item>
      </Form>
    );

    const titleMap = {
      kubeconfig: '上传 Kubeconfig',
      certificate: '配置证书认证',
      token: '配置 Token 认证',
    };

    return (
      <Card title={titleMap[authMethod]}>
        {authMethod === 'kubeconfig' && renderKubeconfigForm()}
        {authMethod === 'certificate' && renderCertificateForm()}
        {authMethod === 'token' && renderTokenForm()}
      </Card>
    );
  };

  // Step 3: 连接测试
  const renderStep3 = () => (
    <Card title="连接测试">
      <div className="text-center py-4">
        <Button
          type="primary"
          size="large"
          icon={<ApiOutlined />}
          onClick={handleValidate}
          loading={validating}
        >
          测试连接
        </Button>
      </div>

      {validationResult && (
        <div className="mt-6">
          {validationResult.valid ? (
            <Alert
              type="success"
              title="连接成功"
              description={
                <Descriptions column={1} size="small" className="mt-2">
                  <Descriptions.Item label="API 地址">{validationResult.endpoint}</Descriptions.Item>
                  <Descriptions.Item label="Kubernetes 版本">{validationResult.version}</Descriptions.Item>
                </Descriptions>
              }
              showIcon
              icon={<CheckCircleOutlined />}
            />
          ) : (
            <Alert
              type="error"
              title="连接失败"
              description={
                <div>
                  <p>{validationResult.message}</p>
                  <Divider className="my-2" />
                  <Text type="secondary">
                    常见问题：
                    <ul className="list-disc pl-4 mt-1">
                      <li>API 地址是否正确，是否包含 https://</li>
                      <li>证书是否过期或格式错误</li>
                      <li>网络是否可达（防火墙、安全组）</li>
                      <li>权限是否足够</li>
                    </ul>
                  </Text>
                </div>
              }
              showIcon
              icon={<CloseCircleOutlined />}
            />
          )}
        </div>
      )}

      {!validationResult && (
        <Alert
          type="info"
          className="mt-4"
          title='点击"测试连接"验证配置是否正确'
          showIcon
          icon={<InfoCircleOutlined />}
        />
      )}
    </Card>
  );

  // Step 4: 确认导入
  const renderStep4 = () => {
    const values = form.getFieldsValue();
    return (
      <Card title="确认导入">
        <Descriptions column={2} bordered size="small">
          <Descriptions.Item label="集群名称">{values.name}</Descriptions.Item>
          <Descriptions.Item label="描述">{values.description || '-'}</Descriptions.Item>
          <Descriptions.Item label="认证方式">
            <Tag color="blue">{authMethodConfig[authMethod].title}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="连接状态">
            {validationResult?.valid ? (
              <Tag color="green">已验证</Tag>
            ) : (
              <Tag color="orange">未验证</Tag>
            )}
          </Descriptions.Item>
          {validationResult?.valid && (
            <>
              <Descriptions.Item label="API 地址">{validationResult.endpoint}</Descriptions.Item>
              <Descriptions.Item label="K8s 版本">{validationResult.version}</Descriptions.Item>
            </>
          )}
        </Descriptions>

        <Alert
          type="warning"
          className="mt-4"
          title="注意"
          description="导入后，系统将定期同步集群信息。请确保 API Server 可持续访问。"
          showIcon
        />
      </Card>
    );
  };

  // Step 5: 完成
  const renderStep5 = () => (
    <Result
      status="success"
      title="集群导入成功"
      subTitle={`集群 "${importedCluster?.name}" 已成功导入并开始同步`}
      extra={[
        <Button
          type="primary"
          key="detail"
          onClick={() => navigate(`/deployment/infrastructure/clusters/${importedCluster?.id}`)}
        >
          查看集群
        </Button>,
        <Button key="list" onClick={() => navigate('/deployment/infrastructure/clusters')}>
          返回列表
        </Button>,
        <Button key="importAnother" onClick={() => {
          form.resetFields();
          setValidationResult(null);
          setImportedCluster(null);
          setCurrentStep(0);
        }}>
          继续导入
        </Button>,
      ]}
    />
  );

  const steps = [
    { title: '基本信息', content: renderStep0() },
    { title: '认证方式', content: renderStep1() },
    { title: '连接配置', content: renderStep2() },
    { title: '连接测试', content: renderStep3() },
    { title: '确认导入', content: renderStep4() },
    { title: '完成', content: renderStep5() },
  ];

  const canProceed = () => {
    switch (currentStep) {
      case 0:
        return !!watchedName;
      case 2:
        // 连接配置步骤，检查必填字段
        if (authMethod === 'kubeconfig') {
          return !!watchedKubeconfig;
        }
        if (authMethod === 'certificate') {
          return !!watchedEndpoint && !!watchedCaCert && !!watchedCert && !!watchedKey;
        }
        if (authMethod === 'token') {
          return !!watchedEndpoint && !!watchedToken;
        }
        return false;
      case 3:
        return validationResult?.valid;
      default:
        return true;
    }
  };

  const handleNext = async () => {
    if (currentStep === 0) {
      try {
        await form.validateFields(['name']);
        setCurrentStep(currentStep + 1);
      } catch {
        // Validation failed
      }
    } else if (currentStep === 4) {
      handleImport();
    } else {
      setCurrentStep(currentStep + 1);
    }
  };

  const handlePrev = () => {
    setCurrentStep(currentStep - 1);
  };

  return (
    <div className="space-y-6">
      {messageContextHolder}
      <div className="flex items-center gap-4">
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/deployment/infrastructure/clusters')}>
          返回
        </Button>
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">导入集群</h1>
          <p className="text-sm text-gray-500 mt-1">导入已存在的 Kubernetes 集群进行统一管理</p>
        </div>
      </div>

      {currentStep < 5 && (
        <Steps current={currentStep} items={steps.slice(0, 5).map(s => ({ title: s.title }))} />
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
            {currentStep > 0 && (
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
              <Button
                type="primary"
                onClick={handleNext}
                loading={loading}
                disabled={!validationResult?.valid}
              >
                确认导入
              </Button>
            )}
          </Space>
        </div>
      )}
    </div>
  );
};

export default ClusterImportWizard;
