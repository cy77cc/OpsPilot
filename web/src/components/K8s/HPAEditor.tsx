import React from 'react';
import { Button, Form, Input, InputNumber, Modal, Space, Table, message } from 'antd';
import { Api } from '../../api';

interface Props {
  clusterId: string;
}

const HPAEditor: React.FC<Props> = ({ clusterId }) => {
  const [loading, setLoading] = React.useState(false);
  const [list, setList] = React.useState<any[]>([]);
  const [open, setOpen] = React.useState(false);
  const [form] = Form.useForm();

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const res = await Api.kubernetes.listHPA(clusterId);
      setList(res.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载 HPA 失败');
    } finally {
      setLoading(false);
    }
  }, [clusterId]);

  React.useEffect(() => { void load(); }, [load]);

  const save = async () => {
    const v = await form.validateFields();
    const found = list.find((x) => x.name === v.name && x.namespace === v.namespace);
    if (found) {
      await Api.kubernetes.updateHPA(clusterId, v.name, v);
      message.success('HPA 已更新');
    } else {
      await Api.kubernetes.createHPA(clusterId, v);
      message.success('HPA 已创建');
    }
    setOpen(false);
    form.resetFields();
    await load();
  };

  const remove = async (name: string, namespace: string) => {
    await Api.kubernetes.deleteHPA(clusterId, name, namespace);
    message.success('HPA 已删除');
    await load();
  };

  return (
    <div>
      <Space style={{ marginBottom: 12 }}>
        <Button onClick={load} loading={loading}>刷新</Button>
        <Button type="primary" onClick={() => setOpen(true)}>新增/更新 HPA</Button>
      </Space>

      <Table
        rowKey={(r) => `${r.namespace}:${r.name}`}
        loading={loading}
        dataSource={list}
        columns={[
          { title: 'Name', dataIndex: 'name' },
          { title: 'Namespace', dataIndex: 'namespace' },
          { title: 'Target', render: (_: any, r: any) => `${r.target_ref_kind}/${r.target_ref_name}` },
          { title: 'Replicas', render: (_: any, r: any) => `${r.min_replicas}-${r.max_replicas}` },
          { title: 'CPU%', dataIndex: 'cpu_utilization' },
          { title: 'MEM%', dataIndex: 'memory_utilization' },
          { title: '操作', render: (_: any, r: any) => <Button danger size="small" onClick={() => void remove(r.name, r.namespace)}>删除</Button> },
        ]}
        pagination={false}
      />

      <Modal title="HPA 策略" open={open} onCancel={() => setOpen(false)} onOk={() => void save()}>
        <Form form={form} layout="vertical" initialValues={{ namespace: 'default', target_ref_kind: 'Deployment', min_replicas: 1, max_replicas: 3, cpu_utilization: 70, memory_utilization: 75 }}>
          <Form.Item label="Namespace" name="namespace" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="Name" name="name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="Target Kind" name="target_ref_kind" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="Target Name" name="target_ref_name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="Min Replicas" name="min_replicas" rules={[{ required: true }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item>
          <Form.Item label="Max Replicas" name="max_replicas" rules={[{ required: true }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item>
          <Form.Item label="CPU Utilization %" name="cpu_utilization"><InputNumber min={1} max={100} style={{ width: '100%' }} /></Form.Item>
          <Form.Item label="Memory Utilization %" name="memory_utilization"><InputNumber min={1} max={100} style={{ width: '100%' }} /></Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default HPAEditor;
