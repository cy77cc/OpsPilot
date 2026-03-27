import React, { useEffect, useState } from 'react';
import { Button, Card, Form, Input, Modal, Space, Table, Tag, message } from 'antd';
import { Api } from '../../api';
import type { SSHKeyItem } from '../../api/modules/hosts';

const HostKeysPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [keys, setKeys] = useState<SSHKeyItem[]>([]);
  const [open, setOpen] = useState(false);
  const [verifyOpen, setVerifyOpen] = useState<{ visible: boolean; keyId: string }>({ visible: false, keyId: '' });
  const [form] = Form.useForm();
  const [verifyForm] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const res = await Api.hosts.listSSHKeys();
      setKeys(res.data || []);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const onCreate = async () => {
    const values = await form.validateFields();
    await Api.hosts.createSSHKey({ name: values.name, privateKey: values.privateKey, passphrase: values.passphrase });
    message.success('密钥创建成功');
    setOpen(false);
    form.resetFields();
    load();
  };

  const onVerify = async () => {
    const values = await verifyForm.validateFields();
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
    setVerifyOpen({ visible: false, keyId: '' });
    verifyForm.resetFields();
  };

  return (
    <Card
      title="SSH 密钥管理"
      extra={<Space><Button onClick={load} loading={loading}>刷新</Button><Button type="primary" onClick={() => setOpen(true)}>新增密钥</Button></Space>}
    >
      <Table
        rowKey="id"
        loading={loading}
        dataSource={keys}
        columns={[
          { title: '名称', dataIndex: 'name' },
          { title: '指纹', dataIndex: 'fingerprint' },
          { title: '算法', dataIndex: 'algorithm' },
          { title: '加密', dataIndex: 'encrypted', render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? 'yes' : 'no'}</Tag> },
          { title: '使用次数', dataIndex: 'usageCount' },
          { title: '创建时间', dataIndex: 'createdAt', render: (v: string) => (v ? new Date(v).toLocaleString() : '-') },
          {
            title: '操作',
            render: (_: unknown, row: SSHKeyItem) => (
              <Space>
                <Button type="link" onClick={() => setVerifyOpen({ visible: true, keyId: row.id })}>连通性验证</Button>
                <Button
                  type="link"
                  danger
                  onClick={() => {
                    Modal.confirm({
                      title: `确认删除密钥 ${row.name}`,
                      onOk: async () => {
                        await Api.hosts.deleteSSHKey(row.id);
                        message.success('删除成功');
                        load();
                      },
                    });
                  }}
                >删除</Button>
              </Space>
            ),
          },
        ]}
      />

      <Modal title="新增 SSH 密钥" open={open} onCancel={() => setOpen(false)} onOk={onCreate} width={760}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="privateKey" label="私钥内容（PEM）" rules={[{ required: true }]}><Input.TextArea rows={10} /></Form.Item>
          <Form.Item name="passphrase" label="Passphrase（可选）"><Input.Password /></Form.Item>
        </Form>
      </Modal>

      <Modal title="密钥连通性验证" open={verifyOpen.visible} onCancel={() => setVerifyOpen({ visible: false, keyId: '' })} onOk={onVerify}>
        <Form form={verifyForm} layout="vertical" initialValues={{ port: 22, username: 'root' }}>
          <Form.Item name="ip" label="目标 IP" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="port" label="端口" rules={[{ required: true }]}><Input type="number" /></Form.Item>
          <Form.Item name="username" label="用户名" rules={[{ required: true }]}><Input /></Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default HostKeysPage;
