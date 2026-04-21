import React, { useEffect, useState } from 'react';
import { Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Radio, Select, Space, Table, Tabs, Tag, message } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../api';
import type { CredentialTemplate, SSHKeyItem } from '../../api/modules/hosts';
import { GuidedFormItem } from '../../components/FormGuidance';

const HostCredentialsPage: React.FC = () => {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState('keys');

  // SSH 密钥相关状态
  const [keysLoading, setKeysLoading] = useState(false);
  const [keys, setKeys] = useState<SSHKeyItem[]>([]);
  const [keyModalOpen, setKeyModalOpen] = useState(false);
  const [verifyOpen, setVerifyOpen] = useState<{ visible: boolean; keyId: string }>({ visible: false, keyId: '' });
  const [keyForm] = Form.useForm();
  const [verifyForm] = Form.useForm();

  // 认证预设相关状态
  const [templatesLoading, setTemplatesLoading] = useState(false);
  const [templates, setTemplates] = useState<CredentialTemplate[]>([]);
  const [templateModalOpen, setTemplateModalOpen] = useState(false);
  const [templateForm] = Form.useForm();

  // 加载 SSH 密钥
  const loadKeys = async () => {
    setKeysLoading(true);
    try {
      const res = await Api.hosts.listSSHKeys();
      setKeys(res.data || []);
    } finally {
      setKeysLoading(false);
    }
  };

  // 加载认证预设
  const loadTemplates = async () => {
    setTemplatesLoading(true);
    try {
      const res = await Api.hosts.listCredentialTemplates();
      setTemplates(res.data || []);
    } finally {
      setTemplatesLoading(false);
    }
  };

  useEffect(() => {
    loadKeys();
    loadTemplates();
  }, []);

  // 创建 SSH 密钥
  const onCreateKey = async () => {
    const values = await keyForm.validateFields();
    try {
      await Api.hosts.createSSHKey({
        name: values.name,
        privateKey: values.privateKey,
        passphrase: values.passphrase,
      });
      message.success('密钥创建成功');
      setKeyModalOpen(false);
      keyForm.resetFields();
      loadKeys();
    } catch (err: any) {
      message.error(err?.message || '创建失败');
    }
  };

  // 验证 SSH 密钥
  const onVerifyKey = async () => {
    const values = await verifyForm.validateFields();
    try {
      const res = await Api.hosts.verifySSHKey(verifyOpen.keyId, {
        ip: values.ip,
        port: values.port,
        username: values.username,
      });
      if (res.data?.reachable) {
        message.success(`验证成功: ${res.data.hostname || ''}`);
      } else {
        message.error(`验证失败: ${res.data?.message || 'unknown error'}`);
      }
    } catch (err: any) {
      message.error(err?.message || '验证失败');
    }
    setVerifyOpen({ visible: false, keyId: '' });
    verifyForm.resetFields();
  };

  
  // 创建认证预设
  const onCreateTemplate = async () => {
    const values = await templateForm.validateFields();
    try {
      await Api.hosts.createCredentialTemplate({
        name: values.name,
        authType: values.authType,
        sshUser: values.sshUser || 'root',
        port: values.port || 22,
        password: values.password,
        sshKeyId: values.sshKeyId ? Number(values.sshKeyId) : undefined,
        description: values.description,
      });
      message.success('预设创建成功');
      setTemplateModalOpen(false);
      templateForm.resetFields();
      loadTemplates();
    } catch (err: any) {
      message.error(err?.message || '创建失败');
    }
  };

  
  const authType = Form.useWatch('authType', templateForm);

  const tabItems = [
    {
      key: 'keys',
      label: 'SSH 密钥',
      children: (
        <Table
          rowKey="id"
          loading={keysLoading}
          dataSource={keys}
          columns={[
            { title: '名称', dataIndex: 'name' },
            { title: '指纹', dataIndex: 'fingerprint', ellipsis: true },
            { title: '算法', dataIndex: 'algorithm', width: 100 },
            {
              title: '加密',
              dataIndex: 'encrypted',
              width: 80,
              render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? 'yes' : 'no'}</Tag>,
            },
            { title: '使用次数', dataIndex: 'usageCount', width: 100 },
            {
              title: '创建时间',
              dataIndex: 'createdAt',
              width: 180,
              render: (v: string) => (v ? new Date(v).toLocaleString() : '-'),
            },
            {
              title: '操作',
              width: 180,
              render: (_: unknown, row: SSHKeyItem) => (
                <Space>
                  <Button type="link" onClick={() => setVerifyOpen({ visible: true, keyId: row.id })}>
                    验证
                  </Button>
                  <Popconfirm
                    title="确定删除此密钥？"
                    okText="确定"
                    cancelText="取消"
                    okButtonProps={{ danger: true }}
                    onConfirm={async () => {
                      try {
                        await Api.hosts.deleteSSHKey(row.id);
                        message.success('删除成功');
                        loadKeys();
                      } catch (err: any) {
                        message.error(err?.message || '删除失败');
                      }
                    }}
                  >
                    <Button type="link" danger>删除</Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      ),
    },
    {
      key: 'templates',
      label: '认证预设',
      children: (
        <Table
          rowKey="id"
          loading={templatesLoading}
          dataSource={templates}
          columns={[
            { title: '名称', dataIndex: 'name' },
            {
              title: '类型',
              dataIndex: 'authType',
              width: 100,
              render: (v: string) => (
                <Tag color={v === 'key' ? 'blue' : 'green'}>{v === 'key' ? '密钥' : '密码'}</Tag>
              ),
            },
            { title: '用户', dataIndex: 'sshUser', width: 100 },
            { title: '端口', dataIndex: 'port', width: 80 },
            { title: '关联密钥', dataIndex: 'sshKeyName', render: (v: string) => v || '-' },
            { title: '描述', dataIndex: 'description', ellipsis: true, render: (v: string) => v || '-' },
            {
              title: '创建时间',
              dataIndex: 'createdAt',
              width: 180,
              render: (v: string) => (v ? new Date(v).toLocaleString() : '-'),
            },
            {
              title: '操作',
              width: 100,
              render: (_: unknown, row: CredentialTemplate) => (
                <Popconfirm
                  title="确定删除此预设？"
                  okText="确定"
                  cancelText="取消"
                  okButtonProps={{ danger: true }}
                  onConfirm={async () => {
                    try {
                      await Api.hosts.deleteCredentialTemplate(row.id);
                      message.success('删除成功');
                      loadTemplates();
                    } catch (err: any) {
                      message.error(err?.message || '删除失败');
                    }
                  }}
                >
                  <Button type="link" danger>删除</Button>
                </Popconfirm>
              ),
            },
          ]}
        />
      ),
    },
  ];

  return (
    <Card
      title="凭证管理"
      extra={
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/deployment/infrastructure/hosts')}>
            返回
          </Button>
          <Button onClick={() => { loadKeys(); loadTemplates(); }} loading={keysLoading || templatesLoading}>
            刷新
          </Button>
          {activeTab === 'keys' ? (
            <Button type="primary" onClick={() => setKeyModalOpen(true)}>
              新增密钥
            </Button>
          ) : (
            <Button type="primary" onClick={() => setTemplateModalOpen(true)}>
              新增预设
            </Button>
          )}
        </Space>
      }
    >
      <Tabs activeKey={activeTab} onChange={setActiveTab} items={tabItems} />

      {/* SSH 密钥创建弹窗 */}
      <Modal
        title="新增 SSH 密钥"
        open={keyModalOpen}
        onCancel={() => setKeyModalOpen(false)}
        onOk={onCreateKey}
        width={600}
      >
        <Form form={keyForm} layout="vertical">
          <GuidedFormItem name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </GuidedFormItem>
          <GuidedFormItem name="privateKey" label="私钥内容（PEM）" rules={[{ required: true }]}>
            <Input.TextArea rows={8} />
          </GuidedFormItem>
          <GuidedFormItem name="passphrase" label="Passphrase（可选）">
            <Input.Password />
          </GuidedFormItem>
        </Form>
      </Modal>

      {/* SSH 密钥验证弹窗 */}
      <Modal
        title="密钥连通性验证"
        open={verifyOpen.visible}
        onCancel={() => setVerifyOpen({ visible: false, keyId: '' })}
        onOk={onVerifyKey}
      >
        <Form form={verifyForm} layout="vertical" initialValues={{ port: 22, username: 'root' }}>
          <GuidedFormItem name="ip" label="目标 IP" rules={[{ required: true }]}>
            <Input />
          </GuidedFormItem>
          <GuidedFormItem name="port" label="端口" rules={[{ required: true }]}>
            <Input type="number" />
          </GuidedFormItem>
          <GuidedFormItem name="username" label="用户名" rules={[{ required: true }]}>
            <Input />
          </GuidedFormItem>
        </Form>
      </Modal>

      {/* 认证预设创建弹窗 */}
      <Modal
        title="新增认证预设"
        open={templateModalOpen}
        onCancel={() => setTemplateModalOpen(false)}
        onOk={onCreateTemplate}
        width={500}
      >
        <Form form={templateForm} layout="vertical" initialValues={{ authType: 'password', sshUser: 'root', port: 22 }}>
          <GuidedFormItem name="name" label="预设名称" rules={[{ required: true, message: '请输入预设名称' }]}>
            <Input placeholder="如: root-22-password" />
          </GuidedFormItem>
          <Form.Item name="authType" label="认证类型" rules={[{ required: true }]}>
            <Radio.Group optionType="button" buttonStyle="solid">
              <Radio.Button value="password">密码认证</Radio.Button>
              <Radio.Button value="key">密钥认证</Radio.Button>
            </Radio.Group>
          </Form.Item>
          <GuidedFormItem name="sshUser" label="SSH 用户" rules={[{ required: true }]}>
            <Input placeholder="默认 root" />
          </GuidedFormItem>
          <GuidedFormItem name="port" label="SSH 端口" rules={[{ required: true }]}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </GuidedFormItem>
          {authType === 'password' ? (
            <GuidedFormItem name="password" label="SSH 密码" rules={[{ required: true, message: '请输入密码' }]}>
              <Input.Password placeholder="SSH 登录密码" />
            </GuidedFormItem>
          ) : (
            <Form.Item name="sshKeyId" label="SSH 密钥" rules={[{ required: true, message: '请选择密钥' }]}>
              <Select
                placeholder="选择已创建的 SSH 密钥"
                options={keys.map((k) => ({ value: k.id, label: `${k.name} (${k.algorithm})` }))}
              />
            </Form.Item>
          )}
          <GuidedFormItem name="description" label="描述">
            <Input.TextArea rows={2} placeholder="可选描述" />
          </GuidedFormItem>
        </Form>
      </Modal>
    </Card>
  );
};

export default HostCredentialsPage;