import React, { useEffect, useState } from 'react';
import { Button, Card, Form, Input, Select, Space, Table, message } from 'antd';
import { Api } from '../../api';
import type { Host } from '../../api/modules/hosts';

const HostVirtualizationPage: React.FC = () => {
  const [hosts, setHosts] = useState<Host[]>([]);
  const [preview, setPreview] = useState<any>(null);
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    Api.hosts.getHostList({ page: 1, pageSize: 500 }).then((res) => {
      setHosts((res.data?.list || []).filter((x) => x.status === 'online'));
    });
  }, []);

  const doPreview = async () => {
    const values = await form.validateFields();
    setLoading(true);
    try {
      const res = await Api.hosts.kvmPreview(values.hostId, {
        name: values.name,
        cpu: Number(values.cpu),
        memoryMB: Number(values.memoryMB),
        diskGB: Number(values.diskGB),
      });
      setPreview(res.data);
      message.success('预检查完成');
    } finally {
      setLoading(false);
    }
  };

  const doProvision = async () => {
    const values = await form.validateFields();
    setLoading(true);
    try {
      const res = await Api.hosts.kvmProvision(values.hostId, {
        name: values.name,
        ip: values.ip,
        cpu: Number(values.cpu),
        memoryMB: Number(values.memoryMB),
        diskGB: Number(values.diskGB),
        sshUser: values.sshUser,
        password: values.password,
      });
      message.success(`虚拟化创建成功, task=${res.data?.task?.id || '-'}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card title="KVM 虚拟化创建（从宿主机虚拟出新机器）">
      <Form
        form={form}
        layout="vertical"
        initialValues={{ cpu: 2, memoryMB: 4096, diskGB: 50, sshUser: 'root' }}
      >
        <Form.Item name="hostId" label="宿主机" rules={[{ required: true }]}>
          <Select
            options={hosts.map((x) => ({ value: x.id, label: `${x.name} (${x.ip})` }))}
            placeholder="选择在线宿主机"
          />
        </Form.Item>
        <Form.Item name="name" label="新机器名称" rules={[{ required: true }]}><Input /></Form.Item>
        <Form.Item name="ip" label="新机器IP" rules={[{ required: true }]}><Input /></Form.Item>
        <Space style={{ display: 'flex' }} size={16}>
          <Form.Item name="cpu" label="CPU" rules={[{ required: true }]}><Input type="number" /></Form.Item>
          <Form.Item name="memoryMB" label="内存MB" rules={[{ required: true }]}><Input type="number" /></Form.Item>
          <Form.Item name="diskGB" label="磁盘GB" rules={[{ required: true }]}><Input type="number" /></Form.Item>
        </Space>
        <Space style={{ display: 'flex' }} size={16}>
          <Form.Item name="sshUser" label="SSH用户"><Input /></Form.Item>
          <Form.Item name="password" label="SSH密码"><Input.Password /></Form.Item>
        </Space>
        <Space>
          <Button onClick={doPreview} loading={loading}>预检查</Button>
          <Button type="primary" onClick={doProvision} loading={loading}>创建并纳管</Button>
        </Space>
      </Form>

      {preview && (
        <Table
          style={{ marginTop: 16 }}
          pagination={false}
          dataSource={[preview]}
          rowKey={() => 'preview'}
          columns={[
            { title: '宿主机ID', dataIndex: 'host_id' },
            { title: 'Hypervisor', dataIndex: 'hypervisor' },
            { title: 'Ready', dataIndex: 'ready', render: (v: boolean) => (v ? 'yes' : 'no') },
            { title: 'Message', dataIndex: 'message' },
          ]}
        />
      )}
    </Card>
  );
};

export default HostVirtualizationPage;
