import React from 'react';
import { Button, Form, Input, InputNumber, Modal, Space, Table, Tag, message } from 'antd';
import { Api } from '../../api';

interface Props {
  clusterId: string;
}

const NamespacePolicyPanel: React.FC<Props> = ({ clusterId }) => {
  const [loading, setLoading] = React.useState(false);
  const [namespaces, setNamespaces] = React.useState<any[]>([]);
  const [bindings, setBindings] = React.useState<any[]>([]);
  const [nsOpen, setNsOpen] = React.useState(false);
  const [bindOpen, setBindOpen] = React.useState(false);
  const [nsForm] = Form.useForm();
  const [bindForm] = Form.useForm();

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const teamId = localStorage.getItem('teamId') || '';
      const [nsRes, bindRes] = await Promise.all([
        Api.kubernetes.getClusterNamespaces(clusterId),
        Api.kubernetes.getNamespaceBindings(clusterId, teamId || undefined),
      ]);
      setNamespaces(nsRes.data.list || []);
      setBindings(bindRes.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载命名空间失败');
    } finally {
      setLoading(false);
    }
  }, [clusterId]);

  React.useEffect(() => { void load(); }, [load]);

  const createNamespace = async () => {
    const v = await nsForm.validateFields();
    await Api.kubernetes.createNamespace(clusterId, { name: v.name, env: v.env });
    message.success('命名空间已创建');
    setNsOpen(false);
    nsForm.resetFields();
    await load();
  };

  const saveBindings = async () => {
    const v = await bindForm.validateFields();
    await Api.kubernetes.putNamespaceBindings(clusterId, String(v.team_id),
      (v.namespaces || []).map((x: string) => ({ namespace: x.trim() })).filter((x: any) => x.namespace),
    );
    message.success('绑定已更新');
    setBindOpen(false);
    bindForm.resetFields();
    await load();
  };

  const removeNamespace = async (name: string) => {
    await Api.kubernetes.deleteNamespace(clusterId, name);
    message.success('命名空间删除请求已提交');
    await load();
  };

  return (
    <div>
      <Space style={{ marginBottom: 12 }}>
        <Button onClick={load} loading={loading}>刷新</Button>
        <Button type="primary" onClick={() => setNsOpen(true)}>新建 Namespace</Button>
        <Button onClick={() => setBindOpen(true)}>管理 Team 绑定</Button>
      </Space>

      <Table
        size="small"
        rowKey="name"
        loading={loading}
        dataSource={namespaces}
        columns={[
          { title: 'Namespace', dataIndex: 'name' },
          { title: '状态', dataIndex: 'status' },
          { title: '标签', dataIndex: 'labels', render: (labels: Record<string, string>) => Object.entries(labels || {}).slice(0, 3).map(([k, v]) => <Tag key={k}>{k}={v}</Tag>) },
          { title: '操作', render: (_: any, r: any) => <Button danger type="link" onClick={() => removeNamespace(r.name)}>删除</Button> },
        ]}
        pagination={false}
      />

      <Table
        style={{ marginTop: 16 }}
        size="small"
        rowKey={(r) => `${r.team_id}-${r.namespace}`}
        dataSource={bindings}
        columns={[
          { title: 'TeamID', dataIndex: 'team_id' },
          { title: 'Namespace', dataIndex: 'namespace' },
          { title: '环境', dataIndex: 'env', render: (v: string) => v || '-' },
          { title: '只读', dataIndex: 'readonly', render: (v: boolean) => <Tag color={v ? 'orange' : 'green'}>{v ? 'readonly' : 'rw'}</Tag> },
        ]}
        pagination={false}
      />

      <Modal title="新建 Namespace" open={nsOpen} onCancel={() => setNsOpen(false)} onOk={() => void createNamespace()}>
        <Form form={nsForm} layout="vertical">
          <Form.Item label="名称" name="name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="环境" name="env"><Input placeholder="development/staging/production" /></Form.Item>
        </Form>
      </Modal>

      <Modal title="更新 Team Namespace 绑定" open={bindOpen} onCancel={() => setBindOpen(false)} onOk={() => void saveBindings()}>
        <Form form={bindForm} layout="vertical" initialValues={{ team_id: Number(localStorage.getItem('teamId') || 1) }}>
          <Form.Item label="Team ID" name="team_id" rules={[{ required: true }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item>
          <Form.Item label="Namespaces" name="namespaces" rules={[{ required: true, message: '至少一个 namespace' }]}>
            <Input placeholder="逗号分隔: default,dev,staging" onBlur={(e) => bindForm.setFieldValue('namespaces', String(e.target.value || '').split(',').map((x) => x.trim()).filter(Boolean))} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default NamespacePolicyPanel;
