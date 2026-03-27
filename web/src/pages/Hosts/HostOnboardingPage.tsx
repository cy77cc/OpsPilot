import React, { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Divider,
  Form,
  Input,
  InputNumber,
  Modal,
  Radio,
  Select,
  Space,
  Steps,
  Tag,
  message,
} from 'antd';
import { ArrowLeftOutlined, CheckCircleOutlined, CloudUploadOutlined } from '@ant-design/icons';
import { Link, useNavigate } from 'react-router-dom';
import { Api } from '../../api';
import type { HostProbeResult, SSHKeyItem } from '../../api/modules/hosts';
import { useAuth } from '../../components/Auth/AuthContext';

interface StepOneForm {
  name: string;
  ip: string;
  port: number;
  authType: 'password' | 'key';
  username: string;
  password?: string;
  sshKeyId?: number;
}

interface StepThreeForm {
  description?: string;
  labels?: string;
  role?: string;
  clusterId?: number;
  force?: boolean;
}

const HostOnboardingPage: React.FC = () => {
  const navigate = useNavigate();
  const { user } = useAuth();
  const [currentStep, setCurrentStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [probeResult, setProbeResult] = useState<HostProbeResult | null>(null);
  const [stepOneValues, setStepOneValues] = useState<StepOneForm | null>(null);
  const [sshKeys, setSshKeys] = useState<SSHKeyItem[]>([]);
  const [keysLoading, setKeysLoading] = useState(false);
  const [keyModalOpen, setKeyModalOpen] = useState(false);
  const [keyCreating, setKeyCreating] = useState(false);
  const [form] = Form.useForm<StepOneForm & StepThreeForm>();
  const [keyForm] = Form.useForm<{ name: string; privateKey: string; passphrase?: string }>();

  const canForceCreate = user?.username?.toLowerCase() === 'admin';

  const loadSSHKeys = useCallback(async () => {
    setKeysLoading(true);
    try {
      const res = await Api.hosts.listSSHKeys();
      setSshKeys(res.data || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载 SSH 密钥失败');
    } finally {
      setKeysLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadSSHKeys();
  }, [loadSSHKeys]);

  const quickCreateKey = async () => {
    const values = await keyForm.validateFields();
    setKeyCreating(true);
    try {
      const res = await Api.hosts.createSSHKey({
        name: values.name,
        privateKey: values.privateKey,
        passphrase: values.passphrase,
      });
      await loadSSHKeys();
      form.setFieldValue('sshKeyId', Number(res.data.id));
      message.success('密钥创建成功，已自动选中');
      setKeyModalOpen(false);
      keyForm.resetFields();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '创建 SSH 密钥失败');
    } finally {
      setKeyCreating(false);
    }
  };

  const doProbe = async () => {
    const values = await form.validateFields(['name', 'ip', 'port', 'authType', 'username', 'password', 'sshKeyId']);
    setLoading(true);
    try {
      const result = await Api.hosts.probeHost({
        name: values.name,
        ip: values.ip,
        port: values.port,
        authType: values.authType,
        username: values.username,
        password: values.password,
        sshKeyId: values.sshKeyId,
      });
      setStepOneValues(values);
      setProbeResult(result.data);
      setCurrentStep(1);
      if (result.data.reachable) {
        message.success('探测成功，请确认后入库');
      } else {
        message.warning(result.data.message || '探测失败，可修改后重试');
      }
    } finally {
      setLoading(false);
    }
  };

  const confirmCreate = async () => {
    const values = await form.validateFields(['description', 'labels', 'role', 'clusterId', 'force']);
    if (!stepOneValues || !probeResult?.probeToken) {
      message.error('probe_token 不存在，请重新探测');
      return;
    }

    setLoading(true);
    try {
      await Api.hosts.createHost({
        probeToken: probeResult.probeToken,
        name: stepOneValues.name,
        ip: stepOneValues.ip,
        port: stepOneValues.port,
        authType: stepOneValues.authType,
        username: stepOneValues.username,
        password: stepOneValues.password,
        sshKeyId: stepOneValues.sshKeyId,
        description: values.description,
        role: values.role,
        clusterId: values.clusterId,
        tags: (values.labels || '').split(',').map((item) => item.trim()).filter(Boolean),
        force: !!values.force,
      });
      message.success('主机接入成功');
      navigate('/hosts');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fade-in">
      <div className="mb-4">
        <Link to="/hosts" className="flex items-center gap-2 text-gray-400 hover:text-white">
          <ArrowLeftOutlined /> 返回主机列表
        </Link>
      </div>

      <Card style={{ background: '#16213e', border: '1px solid #2d3748' }} className="mb-4">
        <div className="flex items-center gap-4">
          <div className="w-12 h-12 rounded-lg bg-gradient-to-br from-blue-500 to-cyan-500 flex items-center justify-center">
            <CloudUploadOutlined className="text-white text-xl" />
          </div>
          <div>
            <h2 className="text-xl font-bold m-0">主机接入（三步向导）</h2>
            <div className="text-gray-400 text-sm">连接信息 → 探测确认 → 入库确认</div>
          </div>
        </div>
      </Card>

      <Card style={{ background: '#16213e', border: '1px solid #2d3748' }} className="mb-4">
        <Steps
          current={currentStep}
          items={[{ title: '连接信息' }, { title: '探测结果' }, { title: '入库确认' }]}
        />
      </Card>

      <Card style={{ background: '#16213e', border: '1px solid #2d3748' }}>
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            port: 22,
            username: 'root',
            authType: 'password',
            force: false,
          }}
        >
          {currentStep === 0 && (
            <>
              <Alert type="info" showIcon message="填写主机连接信息并执行探测" className="mb-4" />
              <Form.Item name="name" label="主机名称" rules={[{ required: true, message: '请输入主机名称' }]}>
                <Input placeholder="例如: prod-api-01" />
              </Form.Item>
              <Form.Item name="ip" label="主机 IP" rules={[{ required: true, message: '请输入主机IP' }]}>
                <Input placeholder="例如: 10.0.0.21" />
              </Form.Item>
              <Space style={{ width: '100%' }} size={16}>
                <Form.Item name="port" label="SSH 端口" rules={[{ required: true }]} style={{ minWidth: 150 }}>
                  <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                </Form.Item>
                <Form.Item name="username" label="SSH 用户" rules={[{ required: true }]} style={{ minWidth: 200 }}>
                  <Input />
                </Form.Item>
              </Space>
              <Form.Item name="authType" label="认证方式" rules={[{ required: true }]}>
                <Radio.Group>
                  <Radio.Button value="password">密码</Radio.Button>
                  <Radio.Button value="key">SSH 密钥</Radio.Button>
                </Radio.Group>
              </Form.Item>
              <Form.Item noStyle shouldUpdate={(prev, next) => prev.authType !== next.authType}>
                {({ getFieldValue }) =>
                  getFieldValue('authType') === 'password' ? (
                    <Form.Item name="password" label="SSH 密码" rules={[{ required: true, message: '请输入 SSH 密码' }]}>
                      <Input.Password />
                    </Form.Item>
                  ) : (
                    <Form.Item
                      name="sshKeyId"
                      label={(
                        <Space size={8}>
                          <span>SSH 密钥</span>
                          <Button size="small" type="link" onClick={() => setKeyModalOpen(true)} style={{ paddingInline: 0 }}>
                            快速添加密钥
                          </Button>
                        </Space>
                      )}
                      rules={[{ required: true, message: '请选择 SSH 密钥' }]}
                    >
                      <Select
                        placeholder="请选择系统中的 SSH 密钥"
                        loading={keysLoading}
                        showSearch
                        optionFilterProp="label"
                        options={sshKeys.map((key) => ({
                          value: Number(key.id),
                          label: `${key.name} (${key.fingerprint || '-'})`,
                        }))}
                        notFoundContent={keysLoading ? '加载中...' : '暂无密钥，请先快速添加'}
                      />
                    </Form.Item>
                  )
                }
              </Form.Item>
            </>
          )}

          {currentStep === 1 && (
            <>
              <Alert
                type={probeResult?.reachable ? 'success' : 'error'}
                showIcon
                message={probeResult?.reachable ? '主机可达，探测成功' : `探测失败：${probeResult?.message || '未知错误'}`}
                className="mb-4"
              />
              <Descriptions bordered column={2} size="small">
                <Descriptions.Item label="Probe Token" span={2}>{probeResult?.probeToken || '-'}</Descriptions.Item>
                <Descriptions.Item label="连通性">{probeResult?.reachable ? <Tag color="green">reachable</Tag> : <Tag color="red">unreachable</Tag>}</Descriptions.Item>
                <Descriptions.Item label="延迟">{probeResult?.latencyMs} ms</Descriptions.Item>
                <Descriptions.Item label="Hostname">{probeResult?.facts?.hostname || '-'}</Descriptions.Item>
                <Descriptions.Item label="OS">{probeResult?.facts?.os || '-'}</Descriptions.Item>
                <Descriptions.Item label="CPU">{probeResult?.facts?.cpuCores || 0} cores</Descriptions.Item>
                <Descriptions.Item label="Memory">{probeResult?.facts?.memoryMB || 0} MB</Descriptions.Item>
                <Descriptions.Item label="Disk">{probeResult?.facts?.diskGB || 0} GB</Descriptions.Item>
                <Descriptions.Item label="Error Code">{probeResult?.errorCode || '-'}</Descriptions.Item>
              </Descriptions>
              {!!probeResult?.warnings?.length && (
                <Alert type="warning" className="mt-4" message={probeResult.warnings.join('；')} />
              )}
            </>
          )}

          {currentStep === 2 && (
            <>
              <Alert type="info" showIcon message="确认入库参数，提交后完成纳管" className="mb-4" />
              <Form.Item name="description" label="描述">
                <Input placeholder="可选" />
              </Form.Item>
              <Form.Item name="labels" label="标签（逗号分隔）">
                <Input placeholder="prod,api,critical" />
              </Form.Item>
              <Space style={{ width: '100%' }} size={16}>
                <Form.Item name="role" label="角色" style={{ minWidth: 220 }}>
                  <Input placeholder="例如: worker" />
                </Form.Item>
                <Form.Item name="clusterId" label="集群 ID" style={{ minWidth: 180 }}>
                  <InputNumber min={0} style={{ width: '100%' }} />
                </Form.Item>
              </Space>
              {canForceCreate && (
                <Form.Item name="force" label="探测失败时强制创建">
                  <Radio.Group>
                    <Radio value={false}>否</Radio>
                    <Radio value={true}>是（仅 admin）</Radio>
                  </Radio.Group>
                </Form.Item>
              )}
            </>
          )}

          <Divider />
          <div className="flex justify-between">
            <Button
              disabled={currentStep === 0 || loading}
              onClick={() => setCurrentStep((s) => Math.max(0, s - 1))}
            >
              上一步
            </Button>
            <Space>
              <Button onClick={() => navigate('/hosts')}>取消</Button>
              {currentStep === 0 && (
                <Button type="primary" onClick={doProbe} loading={loading}>执行探测</Button>
              )}
              {currentStep === 1 && (
                <Button
                  type="primary"
                  onClick={() => {
                    if (!probeResult?.reachable && !canForceCreate) {
                      message.error('探测失败时仅 admin 可强制入库');
                      return;
                    }
                    setCurrentStep(2);
                  }}
                >
                  下一步
                </Button>
              )}
              {currentStep === 2 && (
                <Button type="primary" icon={<CheckCircleOutlined />} onClick={confirmCreate} loading={loading}>
                  确认入库
                </Button>
              )}
            </Space>
          </div>
        </Form>
      </Card>

      <Modal
        title="快速添加 SSH 密钥"
        open={keyModalOpen}
        onCancel={() => setKeyModalOpen(false)}
        onOk={quickCreateKey}
        okText="创建并使用"
        confirmLoading={keyCreating}
        width={760}
      >
        <Form form={keyForm} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入密钥名称' }]}>
            <Input placeholder="例如: prod-root-key" />
          </Form.Item>
          <Form.Item name="privateKey" label="私钥内容（PEM）" rules={[{ required: true, message: '请输入私钥内容' }]}>
            <Input.TextArea rows={10} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />
          </Form.Item>
          <Form.Item name="passphrase" label="Passphrase（可选）">
            <Input.Password placeholder="若私钥有口令请输入" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default HostOnboardingPage;
