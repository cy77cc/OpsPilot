import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Form,
  Input,
  InputNumber,
  Modal,
  Radio,
  Row,
  Col,
  Select,
  Space,
  Card,
  Button,
  message,
  Alert,
  Descriptions,
} from 'antd';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import { Api } from '../../../api';
import type { Host, HostMetricPoint, SSHKeyItem, HostHealthSnapshot } from '../../../api/modules/hosts';
import { useStableFetch } from '../../../hooks';
import { parseHostKeyTrustError, useHostKeyTrust } from '../../../hooks/useHostKeyTrust';
import { DetailSkeleton } from '../../../components/LoadingSkeleton';
import HostKeyTrustModal from '../../../components/Hosts/HostKeyTrustModal';
import { GuidedFormItem } from '../../../components/FormGuidance';

import HostDetailHeader from './components/HostDetailHeader';
import HostDetailTabs from './components/HostDetailTabs';
import OverviewTab from './tabs/OverviewTab';
import MonitorTab from './tabs/MonitorTab';
import ProcessTab from './tabs/ProcessTab';
import ServiceTab from './tabs/ServiceTab';
import DiskTab from './tabs/DiskTab';
import NetworkTab from './tabs/NetworkTab';
import PackageTab from './tabs/PackageTab';
import ConfigTab from './tabs/ConfigTab';
import AlarmTab from './tabs/AlarmTab';
import OperationLogTab from './tabs/OperationLogTab';
import PlaceholderTab from './tabs/PlaceholderTab';

type HostEditFormValues = {
  name: string;
  status: string;
  region?: string;
  description?: string;
  tags?: string;
  authType: 'password' | 'key';
  username: string;
  port: number;
  password?: string;
  sshKeyId?: number;
};

const HostDetailPage: React.FC = () => {
  const { id = '' } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const activeTab = searchParams.get('tab') || 'overview';

  const [loading, setLoading] = useState(false);
  const [host, setHost] = useState<Host | null>(null);
  const [metrics, setMetrics] = useState<HostMetricPoint[]>([]);

  const [editOpen, setEditOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [sshKeys, setSSHKeys] = useState<SSHKeyItem[]>([]);
  const [keysLoading, setKeysLoading] = useState(false);
  const [keyModalOpen, setKeyModalOpen] = useState(false);
  const [keyCreating, setKeyCreating] = useState(false);

  const [editForm] = Form.useForm<HostEditFormValues>();
  const [keyForm] = Form.useForm<{ name: string; privateKey: string; passphrase?: string }>();
  const retryOperationRef = React.useRef<() => Promise<void>>(async () => { });

  const {
    pendingTrust,
    setPendingTrust,
    confirming,
    runWithTrustRetry,
    confirmTrustAndRetry,
  } = useHostKeyTrust(id);

  const loadSSHKeys = useCallback(async () => {
    setKeysLoading(true);
    try {
      const res = await Api.hosts.listSSHKeys();
      setSSHKeys(res.data || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载 SSH 密钥失败');
    } finally {
      setKeysLoading(false);
    }
  }, []);

  const fetchData = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const [hostRes, metricRes] = await Promise.all([
        Api.hosts.getHostDetail(id),
        Api.hosts.getHostMetrics(id),
      ]);
      setHost(hostRes.data);
      setMetrics(metricRes.data || []);
    } finally {
      setLoading(false);
    }
  }, [id]);

  const load = useStableFetch(fetchData);

  useEffect(() => {
    load();
  }, [id, load]);

  useEffect(() => {
    if (!editOpen) return;
    void loadSSHKeys();
  }, [editOpen, loadSSHKeys]);

  const handleAction = (action: string) => {
    if (action === 'terminal') {
      navigate(`/resources/hosts/${id}/terminal`);
      return;
    }

    const exec = async () => {
      await Api.hosts.hostAction(id, action);
      message.success('操作已提交');
      await load();
    };

    if (['restart', 'shutdown'].includes(action)) {
      Modal.confirm({
        title: `确认执行 ${action === 'restart' ? '重启' : '关机'}`,
        content: `确定要对主机 ${host?.name} 执行此操作吗？`,
        onOk: exec,
      });
    } else {
      void exec();
    }
  };

  const runHealthCheck = async () => {
    if (!id) return;
    const operation = async () => {
      const res = await Api.hosts.runHealthCheck(id, true);
      const data: Partial<HostHealthSnapshot> = res.data || {};
      Modal.info({
        title: '健康检查结果',
        width: 680,
        content: (
          <Descriptions bordered size="small" column={1} className="mt-4">
            <Descriptions.Item label="健康状态">{data.state || 'unknown'}</Descriptions.Item>
            <Descriptions.Item label="连通性">{data.connectivityStatus || '-'}</Descriptions.Item>
            <Descriptions.Item label="资源">{data.resourceStatus || '-'}</Descriptions.Item>
            <Descriptions.Item label="系统">{data.systemStatus || '-'}</Descriptions.Item>
            <Descriptions.Item label="延迟">{data.latencyMs || 0} ms</Descriptions.Item>
            <Descriptions.Item label="错误">{data.errorMessage || '-'}</Descriptions.Item>
          </Descriptions>
        ),
      });
      await load();
    };
    retryOperationRef.current = operation;
    try {
      await runWithTrustRetry(operation);
    } catch (err) {
      if (!parseHostKeyTrustError(err)) {
        message.error(err instanceof Error ? err.message : '健康检查失败');
      }
    }
  };

  const openEditModal = () => {
    if (!host) return;
    editForm.setFieldsValue({
      name: host.name,
      status: host.status || 'offline',
      region: host.region || '',
      description: host.description || '',
      tags: (host.tags || []).join(', '),
      authType: host.sshKeyId ? 'key' : 'password',
      username: host.username || 'root',
      port: host.port || 22,
      sshKeyId: host.sshKeyId,
      password: '',
    });
    setEditOpen(true);
  };

  const saveEdit = async () => {
    if (!id) return;
    const values = await editForm.validateFields();
    const tags = (values.tags || '').split(',').map((x) => x.trim()).filter(Boolean);

    setSaving(true);
    try {
      const operation = async () => {
        await Api.hosts.updateHost(id, {
          name: values.name,
          status: values.status,
          region: values.region || '',
          description: values.description || '',
          tags,
        });

        await Api.hosts.updateCredentials(id, {
          authType: values.authType,
          username: values.username,
          port: values.port || 22,
          password: values.authType === 'password' ? (values.password || '') : undefined,
          sshKeyId: values.authType === 'key' ? values.sshKeyId : undefined,
        });

        message.success('保存成功');
        setEditOpen(false);
        await load();
      };
      retryOperationRef.current = operation;
      await runWithTrustRetry(operation);
    } catch (err) {
      if (!parseHostKeyTrustError(err)) {
        message.error(err instanceof Error ? err.message : '保存失败');
      }
    } finally {
      setSaving(false);
    }
  };

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
      editForm.setFieldValue('sshKeyId', Number(res.data.id));
      editForm.setFieldValue('authType', 'key');
      message.success('密钥创建成功');
      setKeyModalOpen(false);
      keyForm.resetFields();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '创建失败');
    } finally {
      setKeyCreating(false);
    }
  };

  const tabContent: Record<string, React.ReactNode> = {
    overview: (
      <OverviewTab
        host={host}
        loading={loading}
        onAction={handleAction}
        onTabChange={(key) => setSearchParams({ tab: key })}
      />
    ),
    monitor: <MonitorTab hostId={id} />,
    process: <ProcessTab hostId={id} />,
    service: <ServiceTab hostId={id} />,
    disk: <DiskTab hostId={id} />,
    network: <NetworkTab hostId={id} />,
    packages: <PackageTab hostId={id} />,
    config: <ConfigTab hostId={id} />,
    alarm: <AlarmTab hostId={id} />,
    logs: <OperationLogTab hostId={id} />,
  };

  if (loading && !host) {
    return <DetailSkeleton summaryCards={4} sections={4} />;
  }

  return (
    <div className="bg-gray-50 min-h-full -m-6">
      {/* header + tab 固定区域 */}
      <div className="sticky top-0 z-20 border-b border-gray-200 bg-white shadow-sm">
        <div className="px-3 py-1">
          <HostDetailHeader
            host={host}
            onEdit={openEditModal}
            onTerminal={() => navigate(`/resources/hosts/${id}/terminal`)}
            onHealthCheck={runHealthCheck}
            onAction={handleAction}
          />
        </div>
        <div className="border-t border-slate-100 bg-gray-50/95 px-6 backdrop-blur">
          <HostDetailTabs
            activeTab={activeTab}
            onChange={(key) => setSearchParams({ tab: key })}
            tabContent={tabContent}
            navOnly
          />
        </div>
      </div>

      <div className="px-4">
        {(host?.healthState === 'critical' || host?.healthState === 'degraded' || host?.maintenanceReason) && (
          <Alert
            type={host?.healthState === 'critical' ? 'error' : host?.healthState === 'degraded' ? 'warning' : 'info'}
            showIcon
            title={`健康状态: ${host?.healthState || 'unknown'}`}
            description={host?.maintenanceReason ? `维护信息: ${host.maintenanceReason}` : undefined}
            className="mb-4"
          />
        )}

        {tabContent[activeTab] || <div className="p-8 text-center text-gray-400">正在开发中...</div>}
      </div>

      <Modal
        title="编辑主机"
        open={editOpen}
        onCancel={() => setEditOpen(false)}
        onOk={() => void saveEdit()}
        okText="保存"
        confirmLoading={saving}
        width={760}
      >
        <Form form={editForm} layout="vertical">
          <Row gutter={12}>
            <Col span={12}>
              <GuidedFormItem name="name" label="主机名称" rules={[{ required: true, message: '请输入主机名称' }]}>
                <Input />
              </GuidedFormItem>
            </Col>
            <Col span={12}>
              <Form.Item name="status" label="状态" rules={[{ required: true }]}>
                <Select options={[{ value: 'online', label: 'online' }, { value: 'offline', label: 'offline' }, { value: 'maintenance', label: 'maintenance' }]} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={12}>
            <Col span={12}>
              <GuidedFormItem name="region" label="区域">
                <Input placeholder="可选" />
              </GuidedFormItem>
            </Col>
            <Col span={12}>
              <GuidedFormItem name="tags" label="标签（逗号分隔）">
                <Input placeholder="prod,web,critical" />
              </GuidedFormItem>
            </Col>
          </Row>

          <GuidedFormItem name="description" label="描述">
            <Input.TextArea rows={2} />
          </GuidedFormItem>

          <Card size="small" title="SSH 凭据" style={{ marginBottom: 8 }}>
            <Row gutter={12}>
              <Col span={8}>
                <Form.Item name="authType" label="认证方式" rules={[{ required: true }]}>
                  <Radio.Group optionType="button" buttonStyle="solid">
                    <Radio value="password">密码</Radio>
                    <Radio value="key">SSH 密钥</Radio>
                  </Radio.Group>
                </Form.Item>
              </Col>
              <Col span={8}>
                <GuidedFormItem name="username" label="SSH 用户" rules={[{ required: true }]}>
                  <Input />
                </GuidedFormItem>
              </Col>
              <Col span={8}>
                <GuidedFormItem name="port" label="SSH 端口" rules={[{ required: true }]}>
                  <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                </GuidedFormItem>
              </Col>
            </Row>

            <Form.Item noStyle shouldUpdate={(prev, next) => prev.authType !== next.authType}>
              {({ getFieldValue }) => (
                getFieldValue('authType') === 'password' ? (
                  <GuidedFormItem name="password" label="SSH 密码" rules={[{ required: true, message: '请输入 SSH 密码' }]}>
                    <Input.Password placeholder="请输入新密码" />
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
              )}
            </Form.Item>
          </Card>
        </Form>
      </Modal>

      <Modal
        title="快速添加 SSH 密钥"
        open={keyModalOpen}
        onCancel={() => setKeyModalOpen(false)}
        onOk={() => void quickCreateKey()}
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
              message.error(err instanceof Error ? err.message : '信任失败');
            }
          }
        }}
      />
    </div>
  );
};

export default HostDetailPage;
