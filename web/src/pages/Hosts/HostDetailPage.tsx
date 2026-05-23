import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Breadcrumb,
  Button,
  Card,
  Col,
  Descriptions,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Radio,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tag,
  message,
} from 'antd';
import { ArrowLeftOutlined, EditOutlined, ReloadOutlined, DeleteOutlined } from '@ant-design/icons';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { Api } from '../../api';
import type { Host, HostAuditItem, HostHealthSnapshot, HostMetricPoint, SSHKeyItem } from '../../api/modules/hosts';
import { useStableFetch } from '../../hooks';
import { parseHostKeyTrustError, useHostKeyTrust } from '../../hooks/useHostKeyTrust';
import { DetailSkeleton } from '../../components/LoadingSkeleton';
import HostKeyTrustModal from '../../components/Hosts/HostKeyTrustModal';
import { GuidedFormItem } from '../../components/FormGuidance';

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
  const [loading, setLoading] = useState(false);
  const [host, setHost] = useState<Host | null>(null);
  const [metrics, setMetrics] = useState<HostMetricPoint[]>([]);
  const [audits, setAudits] = useState<HostAuditItem[]>([]);

  const [editOpen, setEditOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [sshKeys, setSSHKeys] = useState<SSHKeyItem[]>([]);
  const [keysLoading, setKeysLoading] = useState(false);
  const [keyModalOpen, setKeyModalOpen] = useState(false);
  const [keyCreating, setKeyCreating] = useState(false);

  const [editForm] = Form.useForm<HostEditFormValues>();
  const [keyForm] = Form.useForm<{ name: string; privateKey: string; passphrase?: string }>();
  const retryOperationRef = React.useRef<() => Promise<void>>(async () => {});
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
    if (!id) {return;}
    setLoading(true);
    try {
      const [hostRes, metricRes, auditRes] = await Promise.all([
        Api.hosts.getHostDetail(id),
        Api.hosts.getHostMetrics(id),
        Api.hosts.getHostAudits(id),
      ]);
      setHost(hostRes.data);
      setMetrics(metricRes.data || []);
      setAudits(auditRes.data || []);
    } finally {
      setLoading(false);
    }
  }, [id]);

  // Use stable fetch to prevent duplicate requests (e.g., from React StrictMode)
  const load = useStableFetch(fetchData);

  useEffect(() => {
    load();
  }, [id, load]);

  useEffect(() => {
    if (!editOpen) {return;}
    void loadSSHKeys();
  }, [editOpen, loadSSHKeys]);

  const latest = useMemo(() => {
    if (metrics.length === 0) {return { cpu: 0, memory: 0, disk: 0, network: 0 };}
    return metrics[0];
  }, [metrics]);

  const runAction = (action: string, confirm = false) => {
    const exec = async () => {
      await Api.hosts.hostAction(id, action);
      message.success('操作已提交');
      await load();
    };
    if (!confirm) {
      void exec();
      return;
    }
    Modal.confirm({
      title: `确认执行 ${action}`,
      onOk: exec,
    });
  };

  const runHealthCheck = async () => {
    if (!id) {return;}
    const operation = async () => {
      const res = await Api.hosts.runHealthCheck(id, true);
      const data: Partial<HostHealthSnapshot> = res.data || {};
      Modal.info({
        title: '健康检查结果',
        width: 680,
        content: (
          <Descriptions bordered size="small" column={1}>
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
    if (!host) {return;}
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
      message.success('密钥创建成功，已自动选中');
      setKeyModalOpen(false);
      keyForm.resetFields();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '创建 SSH 密钥失败');
    } finally {
      setKeyCreating(false);
    }
  };

  const saveEdit = async () => {
    if (!id) {return;}
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

        const check = await Api.hosts.sshCheck(id);
        message.success(check.data?.reachable ? '保存成功，SSH 连通性正常' : `保存成功，SSH 检查失败：${check.data?.message || '未知错误'}`);
        setEditOpen(false);
        await load();
      };
      retryOperationRef.current = operation;
      await runWithTrustRetry(operation);
    } catch (err) {
      if (!parseHostKeyTrustError(err)) {
        message.error(err instanceof Error ? err.message : '保存主机失败');
      }
    } finally {
      setSaving(false);
    }
  };

  const isInitialLoading = loading && !host;

  if (isInitialLoading) {
    return <DetailSkeleton summaryCards={4} sections={4} />;
  }

  return (
    <div>
      <Breadcrumb
        className="mb-4"
        items={[
          { title: <Link to="/hosts">主机管理</Link> },
          { title: host?.name || id },
        ]}
      />

      <Card
        title={<Space><Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/resources/hosts')}>返回</Button><span>{host?.name || '主机详情'}</span></Space>}
        extra={(
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading && !isInitialLoading}>刷新</Button>
            <Button icon={<EditOutlined />} onClick={openEditModal}>编辑主机</Button>
            <Button onClick={() => navigate(`/resources/hosts/${id}/terminal`)}>终端</Button>
            <Button onClick={() => void runHealthCheck()}>健康检查</Button>
            <Button onClick={() => runAction('restart', true)}>重启</Button>
            <Button danger onClick={() => runAction('shutdown', true)}>关机</Button>
            <Popconfirm
              title="确定删除此主机？"
              okText="确定"
              cancelText="取消"
              okButtonProps={{ danger: true }}
              onConfirm={async () => {
                await Api.hosts.deleteHost(id);
                message.success('主机已删除');
                navigate('/resources/hosts');
              }}
            >
              <Button danger icon={<DeleteOutlined />}>删除</Button>
            </Popconfirm>
          </Space>
        )}
      >
        <Alert
          type={host?.healthState === 'critical' ? 'error' : host?.healthState === 'degraded' ? 'warning' : 'info'}
          showIcon
          title={`健康状态: ${host?.healthState || 'unknown'}`}
          description={host?.maintenanceReason ? `维护信息: ${host.maintenanceReason}` : undefined}
          style={{ marginBottom: 16 }}
        />
        <Row gutter={[16, 16]}>
          <Col span={6}><Card><Statistic title="CPU" value={latest.cpu} suffix="%" /></Card></Col>
          <Col span={6}><Card><Statistic title="内存" value={latest.memory} suffix="%" /></Card></Col>
          <Col span={6}><Card><Statistic title="磁盘" value={latest.disk} suffix="%" /></Card></Col>
          <Col span={6}><Card><Statistic title="网络" value={latest.network} suffix="Mbps" /></Card></Col>
        </Row>

        <div className="mt-4" />
        <Descriptions bordered size="small" column={2}>
          <Descriptions.Item label="名称">{host?.name}</Descriptions.Item>
          <Descriptions.Item label="状态"><Tag color={host?.status === 'online' ? 'success' : host?.status === 'maintenance' ? 'warning' : 'default'}>{host?.status || '-'}</Tag></Descriptions.Item>
          <Descriptions.Item label="健康"><Tag color={host?.healthState === 'healthy' ? 'success' : host?.healthState === 'degraded' ? 'warning' : host?.healthState === 'critical' ? 'error' : 'default'}>{host?.healthState || 'unknown'}</Tag></Descriptions.Item>
          <Descriptions.Item label="维护原因">{host?.maintenanceReason || '-'}</Descriptions.Item>
          <Descriptions.Item label="IP">{host?.ip}</Descriptions.Item>
          <Descriptions.Item label="系统">{host?.os || '-'}</Descriptions.Item>
          <Descriptions.Item label="SSH">{host?.username || 'root'}:{host?.port || 22}</Descriptions.Item>
          <Descriptions.Item label="区域">{host?.region || '-'}</Descriptions.Item>
          <Descriptions.Item label="维护开始">{host?.maintenanceStartedAt ? new Date(host.maintenanceStartedAt).toLocaleString() : '-'}</Descriptions.Item>
          <Descriptions.Item label="维护截止">{host?.maintenanceUntil ? new Date(host.maintenanceUntil).toLocaleString() : '-'}</Descriptions.Item>
          <Descriptions.Item label="CPU 核数">{host?.cpu}</Descriptions.Item>
          <Descriptions.Item label="内存 MB">{host?.memory}</Descriptions.Item>
          <Descriptions.Item label="磁盘 GB">{host?.disk}</Descriptions.Item>
          <Descriptions.Item label="更新时间">{host?.lastActive ? new Date(host.lastActive).toLocaleString() : '-'}</Descriptions.Item>
          <Descriptions.Item label="标签" span={2}>{(host?.tags || []).join(', ') || '-'}</Descriptions.Item>
          <Descriptions.Item label="描述" span={2}>{host?.description || '-'}</Descriptions.Item>
          {host?.jumpHostId && (
            <>
              <Descriptions.Item label="跳板机">{host.jumpHostName || host.jumpHostId}</Descriptions.Item>
              <Descriptions.Item label="连接模式">
                <Tag color="blue">
                  {host.gatewayMode === 'tunnel' ? '隧道模式' : host.gatewayMode === 'proxy' ? '代理模式' : '自动检测'}
                </Tag>
              </Descriptions.Item>
            </>
          )}
        </Descriptions>

        <div className="mt-4" />
        <Card title="指标序列（最近60条）" size="small">
          <Table
            rowKey="id"
            pagination={{ pageSize: 10 }}
            dataSource={metrics}
            columns={[
              { title: '时间', dataIndex: 'createdAt', render: (v: string) => (v ? new Date(v).toLocaleString() : '-') },
              { title: 'CPU', dataIndex: 'cpu', render: (v: number) => `${v}%` },
              { title: '内存', dataIndex: 'memory', render: (v: number) => `${v}%` },
              { title: '磁盘', dataIndex: 'disk', render: (v: number) => `${v}%` },
              { title: '网络', dataIndex: 'network', render: (v: number) => `${v} Mbps` },
            ]}
          />
        </Card>

        <div className="mt-4" />
        <Card title="操作审计" size="small">
          <Table
            rowKey="id"
            pagination={{ pageSize: 10 }}
            dataSource={audits}
            columns={[
              { title: '时间', dataIndex: 'createdAt', render: (v: string) => (v ? new Date(v).toLocaleString() : '-') },
              { title: '动作', dataIndex: 'action' },
              { title: '操作人', dataIndex: 'operator' },
              { title: '详情', dataIndex: 'detail' },
            ]}
          />
        </Card>
      </Card>

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
              message.error(err instanceof Error ? err.message : '信任主机指纹失败');
            }
          }
        }}
      />
    </div>
  );
};

export default HostDetailPage;
