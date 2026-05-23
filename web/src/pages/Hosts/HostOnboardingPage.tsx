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
import type { CredentialTemplate, GatewayHostInfo, HostProbeResult, SSHKeyItem } from '../../api/modules/hosts';
import type { HostPluginCatalogItem } from '../../types/host';
import { useAuth } from '../../components/Auth/AuthContext';
import { parseHostKeyTrustError, useHostKeyTrust } from '../../hooks/useHostKeyTrust';
import HostKeyTrustModal from '../../components/Hosts/HostKeyTrustModal';
import { GuidedFormItem } from '../../components/FormGuidance';

interface StepOneForm {
  name: string;
  ip: string;
  port: number;
  credentialTemplateId?: number;
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
  installOpsAgent: 'none' | 'opsagent';
  opsagentVersion?: string;
  jumpHostId?: number;
  gatewayMode?: 'tunnel' | 'proxy' | 'auto';
}

const HostOnboardingPage: React.FC = () => {
  const navigate = useNavigate();
  const { user } = useAuth();
  const [currentStep, setCurrentStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [probeResult, setProbeResult] = useState<HostProbeResult | null>(null);
  const [stepOneValues, setStepOneValues] = useState<StepOneForm | null>(null);
  const [sshKeys, setSshKeys] = useState<SSHKeyItem[]>([]);
  const [credentialTemplates, setCredentialTemplates] = useState<CredentialTemplate[]>([]);
  const [pluginCatalog, setPluginCatalog] = useState<HostPluginCatalogItem[]>([]);
  const [keysLoading, setKeysLoading] = useState(false);
  const [templatesLoading, setTemplatesLoading] = useState(false);
  const [pluginCatalogLoading, setPluginCatalogLoading] = useState(false);
  const [gatewayHosts, setGatewayHosts] = useState<GatewayHostInfo[]>([]);
  const [gatewayHostsLoading, setGatewayHostsLoading] = useState(false);
  const [keyModalOpen, setKeyModalOpen] = useState(false);
  const [keyCreating, setKeyCreating] = useState(false);
  const [form] = Form.useForm<StepOneForm & StepThreeForm>();
  const [keyForm] = Form.useForm<{ name: string; privateKey: string; passphrase?: string }>();
  const retryOperationRef = React.useRef<() => Promise<void>>(async () => {});
  const {
    pendingTrust,
    setPendingTrust,
    confirming,
    runWithTrustRetry,
    confirmTrustAndRetry,
  } = useHostKeyTrust('0');

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

  const loadCredentialTemplates = useCallback(async () => {
    setTemplatesLoading(true);
    try {
      const res = await Api.hosts.listCredentialTemplates();
      setCredentialTemplates(res.data || []);
    } catch {
      // ignore
    } finally {
      setTemplatesLoading(false);
    }
  }, []);

  const loadPluginCatalog = useCallback(async () => {
    setPluginCatalogLoading(true);
    try {
      const res = await Api.hosts.listHostPluginCatalog();
      setPluginCatalog(res.data || []);
    } catch {
      setPluginCatalog([]);
    } finally {
      setPluginCatalogLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadSSHKeys();
    void loadCredentialTemplates();
    void loadPluginCatalog();
  }, [loadCredentialTemplates, loadPluginCatalog, loadSSHKeys]);

  useEffect(() => {
    setGatewayHostsLoading(true);
    Api.hosts.getGatewayHosts()
      .then(setGatewayHosts)
      .catch(() => {})
      .finally(() => setGatewayHostsLoading(false));
  }, []);

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
    const selectedTemplateId = form.getFieldValue('credentialTemplateId') as number | undefined;
    const values = await form.validateFields(
      selectedTemplateId
        ? ['name', 'ip', 'credentialTemplateId']
        : ['name', 'ip', 'port', 'authType', 'username', 'password', 'sshKeyId'],
    );
    setLoading(true);
    try {
      const operation = async () => {
        const result = await Api.hosts.probeHost({
          name: values.name,
          ip: values.ip,
          port: values.port,
          authType: values.authType,
          username: values.username,
          password: values.password,
          sshKeyId: values.sshKeyId,
          credentialTemplateId: values.credentialTemplateId,
        });
        setStepOneValues(values);
        setProbeResult(result.data);
        setCurrentStep(1);
        if (result.data.reachable) {
          message.success('探测成功，请确认后入库');
        } else {
          message.warning(result.data.message || '探测失败，可修改后重试');
        }
      };
      retryOperationRef.current = operation;
      await runWithTrustRetry(operation);
    } catch (err) {
      if (!parseHostKeyTrustError(err)) {
        message.error(err instanceof Error ? err.message : '执行探测失败');
      }
    } finally {
      setLoading(false);
    }
  };

  const confirmCreate = async () => {
    const values = await form.validateFields(['description', 'labels', 'role', 'clusterId', 'force', 'installOpsAgent', 'opsagentVersion', 'jumpHostId', 'gatewayMode']);
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
        credentialTemplateId: stepOneValues.credentialTemplateId,
        description: values.description,
        role: values.role,
        clusterId: values.clusterId,
        tags: (values.labels || '').split(',').map((item) => item.trim()).filter(Boolean),
        force: !!values.force,
        pluginInstalls: values.installOpsAgent === 'opsagent' && values.opsagentVersion
          ? [{
              pluginKey: 'opsagent',
              version: values.opsagentVersion,
            }]
          : [],
        jumpHostId: values.jumpHostId || undefined,
        gatewayMode: values.gatewayMode || undefined,
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
            installOpsAgent: 'none',
          }}
        >
          {currentStep === 0 && (
            <>
              <Alert type="info" showIcon message="填写主机连接信息并执行探测" className="mb-4" />
              <GuidedFormItem name="name" label="主机名称" rules={[{ required: true, message: '请输入主机名称' }]}>
                <Input placeholder="例如: prod-api-01" />
              </GuidedFormItem>
              <GuidedFormItem name="ip" label="主机 IP" rules={[{ required: true, message: '请输入主机IP' }]}>
                <Input placeholder="例如: 10.0.0.21" />
              </GuidedFormItem>
              <GuidedFormItem
                name="credentialTemplateId"
                label={(
                  <Space size={8}>
                    <span>认证预设（可选）</span>
                    <Button size="small" type="link" onClick={() => navigate('/resources/hosts/credentials')} style={{ paddingInline: 0 }}>
                      管理预设
                    </Button>
                  </Space>
                )}
              >
                <Select
                  allowClear
                  showSearch
                  optionFilterProp="label"
                  loading={templatesLoading}
                  placeholder="选择后将优先使用预设认证信息"
                  options={credentialTemplates.map((template) => ({
                    value: Number(template.id),
                    label: `${template.name} · ${template.authType === 'key' ? '密钥' : '密码'} · ${template.sshUser}@${template.port}`,
                  }))}
                  onChange={(value) => {
                    const selected = credentialTemplates.find((item) => Number(item.id) === Number(value));
                    if (!selected) {
                      return;
                    }
                    form.setFieldsValue({
                      authType: selected.authType,
                      username: selected.sshUser || 'root',
                      port: selected.port || 22,
                      sshKeyId: selected.authType === 'key' && selected.sshKeyId ? Number(selected.sshKeyId) : undefined,
                    });
                  }}
                />
              </GuidedFormItem>
              <Space style={{ width: '100%' }} size={16}>
                <GuidedFormItem name="port" label="SSH 端口" rules={[{ required: true }]} style={{ minWidth: 150 }}>
                  <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                </GuidedFormItem>
                <GuidedFormItem name="username" label="SSH 用户" rules={[{ required: true }]} style={{ minWidth: 200 }}>
                  <Input />
                </GuidedFormItem>
              </Space>
              <Form.Item name="authType" label="认证方式" rules={[{ required: true }]}>
                <Radio.Group>
                  <Radio.Button value="password">密码</Radio.Button>
                  <Radio.Button value="key">SSH 密钥</Radio.Button>
                </Radio.Group>
              </Form.Item>
              <Form.Item noStyle shouldUpdate={(prev, next) => prev.authType !== next.authType || prev.credentialTemplateId !== next.credentialTemplateId}>
                {({ getFieldValue }) =>
                  getFieldValue('authType') === 'password' ? (
                    <GuidedFormItem
                      name="password"
                      label="SSH 密码"
                      rules={getFieldValue('credentialTemplateId') ? [] : [{ required: true, message: '请输入 SSH 密码' }]}
                    >
                      <Input.Password disabled={!!getFieldValue('credentialTemplateId')} placeholder={getFieldValue('credentialTemplateId') ? '使用认证预设中的密码' : ''} />
                    </GuidedFormItem>
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
                      rules={getFieldValue('credentialTemplateId') ? [] : [{ required: true, message: '请选择 SSH 密钥' }]}
                    >
                      <Select
                        placeholder="请选择系统中的 SSH 密钥"
                        disabled={!!getFieldValue('credentialTemplateId')}
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
              <GuidedFormItem name="description" label="描述">
                <Input placeholder="可选" />
              </GuidedFormItem>
              <GuidedFormItem name="labels" label="标签（逗号分隔）">
                <Input placeholder="prod,api,critical" />
              </GuidedFormItem>
              <Space style={{ width: '100%' }} size={16}>
                <GuidedFormItem name="role" label="角色" style={{ minWidth: 220 }}>
                  <Input placeholder="例如: worker" />
                </GuidedFormItem>
                <GuidedFormItem name="clusterId" label="集群 ID" style={{ minWidth: 180 }}>
                  <InputNumber min={0} style={{ width: '100%' }} />
                </GuidedFormItem>
              </Space>
              {canForceCreate && (
                <Form.Item name="force" label="探测失败时强制创建">
                  <Radio.Group>
                    <Radio value={false}>否</Radio>
                    <Radio value={true}>是（仅 admin）</Radio>
                  </Radio.Group>
                </Form.Item>
              )}
              <GuidedFormItem name="installOpsAgent" label="主机插件">
                <Radio.Group>
                  <Radio value="none">暂不安装</Radio>
                  <Radio value="opsagent">安装 OpsAgent</Radio>
                </Radio.Group>
              </GuidedFormItem>
              <Form.Item noStyle shouldUpdate={(prev, next) => prev.installOpsAgent !== next.installOpsAgent}>
                {({ getFieldValue }) => getFieldValue('installOpsAgent') === 'opsagent' ? (
                  <GuidedFormItem
                    name="opsagentVersion"
                    label="OpsAgent 版本"
                    rules={[{ required: true, message: '请选择 OpsAgent 版本' }]}
                  >
                    <Select
                      loading={pluginCatalogLoading}
                      placeholder="选择要安装的 OpsAgent 版本"
                      options={pluginCatalog
                        .filter((item) => item.pluginKey === 'opsagent' && item.defaultVersion)
                        .map((item) => ({ label: item.defaultVersion, value: item.defaultVersion }))}
                    />
                  </GuidedFormItem>
                ) : null}
              </Form.Item>
              <GuidedFormItem
                name="jumpHostId"
                label="跳板机"
                guidance="选择一个跳板机来管理无法直连的内网主机。留空表示直连。"
              >
                <Select
                  allowClear
                  placeholder="选择跳板机（可选）"
                  loading={gatewayHostsLoading}
                  options={gatewayHosts.map(g => ({
                    label: `${g.name} (${g.ip})`,
                    value: g.id,
                  }))}
                />
              </GuidedFormItem>
              <Form.Item noStyle shouldUpdate={(prev, next) => prev.jumpHostId !== next.jumpHostId}>
                {({ getFieldValue }) => getFieldValue('jumpHostId') ? (
                  <GuidedFormItem
                    name="gatewayMode"
                    label="连接模式"
                    guidance="隧道模式：目标主机有 Agent，通过跳板机转发 gRPC 流量。代理模式：目标主机无 Agent，跳板机通过 SSH 代为执行。"
                  >
                    <Radio.Group>
                      <Radio value="tunnel">隧道模式</Radio>
                      <Radio value="proxy">代理模式</Radio>
                      <Radio value="auto">自动检测</Radio>
                    </Radio.Group>
                  </GuidedFormItem>
                ) : null}
              </Form.Item>
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
          <GuidedFormItem name="name" label="名称" rules={[{ required: true, message: '请输入密钥名称' }]}>
            <Input placeholder="例如: prod-root-key" />
          </GuidedFormItem>
          <GuidedFormItem name="privateKey" label="私钥内容（PEM）" rules={[{ required: true, message: '请输入私钥内容' }]}>
            <Input.TextArea rows={10} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />
          </GuidedFormItem>
          <GuidedFormItem name="passphrase" label="Passphrase（可选）">
            <Input.Password placeholder="若私钥有口令请输入" />
          </GuidedFormItem>
        </Form>
      </Modal>

      <HostKeyTrustModal
        open={Boolean(pendingTrust)}
        loading={confirming}
        mode={pendingTrust?.errorType === 'ssh_host_key_mismatch' ? 'rotate' : 'create'}
        hostKey={pendingTrust?.hostKey || null}
        onCancel={() => setPendingTrust(null)}
        onConfirm={async () => {
          try {
            await confirmTrustAndRetry(async () => {
              await retryOperationRef.current();
            });
          } catch (err) {
            if (!parseHostKeyTrustError(err)) {
              message.error(err instanceof Error ? err.message : '信任主机指纹失败');
            }
          }
        }}
      />
    </div>
  );
};

export default HostOnboardingPage;
