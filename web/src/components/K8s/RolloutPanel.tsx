import React from 'react';
import { Button, Form, Input, InputNumber, Modal, Select, Space, Table, Tag, message } from 'antd';
import { Api } from '../../api';

interface Props {
  clusterId: string;
}

const RolloutPanel: React.FC<Props> = ({ clusterId }) => {
  const [loading, setLoading] = React.useState(false);
  const [rollouts, setRollouts] = React.useState<any[]>([]);
  const [modalOpen, setModalOpen] = React.useState(false);
  const [preview, setPreview] = React.useState('');
  const [form] = Form.useForm();

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const res = await Api.kubernetes.listRollouts(clusterId);
      setRollouts(res.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载 Rollouts 失败');
    } finally {
      setLoading(false);
    }
  }, [clusterId]);

  React.useEffect(() => { void load(); }, [load]);

  const doPreview = async () => {
    const values = await form.validateFields();
    const res = await Api.kubernetes.previewRollout(clusterId, values);
    setPreview(res.data.manifest || '');
  };

  const doApply = async () => {
    const values = await form.validateFields();
    await Api.kubernetes.applyRollout(clusterId, values);
    message.success('Rollout 已应用');
    setModalOpen(false);
    setPreview('');
    form.resetFields();
    await load();
  };

  const act = async (name: string, namespace: string, action: 'promote' | 'abort' | 'rollback') => {
    if (action === 'promote') {
      await Api.kubernetes.promoteRollout(clusterId, name, { namespace });
    } else if (action === 'abort') {
      await Api.kubernetes.abortRollout(clusterId, name, { namespace });
    } else {
      await Api.kubernetes.rollbackRollout(clusterId, name, { namespace });
    }
    message.success(`操作已提交: ${action}`);
    await load();
  };

  return (
    <div>
      <Space style={{ marginBottom: 12 }}>
        <Button onClick={load} loading={loading}>刷新</Button>
        <Button type="primary" onClick={() => setModalOpen(true)}>新建/更新 Rollout</Button>
      </Space>
      <Table
        rowKey={(r) => `${r.namespace}:${r.name}`}
        loading={loading}
        dataSource={rollouts}
        columns={[
          { title: 'Name', dataIndex: 'name' },
          { title: 'Namespace', dataIndex: 'namespace' },
          { title: 'Strategy', dataIndex: 'strategy', render: (v: string) => <Tag color="blue">{v}</Tag> },
          { title: 'Phase', dataIndex: 'phase' },
          { title: 'Replicas', render: (_: any, r: any) => `${r.ready_replicas || 0}/${r.replicas || 0}` },
          {
            title: '操作',
            render: (_: any, r: any) => (
              <Space>
                <Button size="small" onClick={() => void act(r.name, r.namespace, 'promote')}>Promote</Button>
                <Button size="small" onClick={() => void act(r.name, r.namespace, 'abort')}>Abort</Button>
                <Button size="small" danger onClick={() => void act(r.name, r.namespace, 'rollback')}>Rollback</Button>
              </Space>
            ),
          },
        ]}
        pagination={false}
      />

      <Modal
        title="Rollout Wizard"
        open={modalOpen}
        onCancel={() => { setModalOpen(false); setPreview(''); }}
        onOk={() => void doApply()}
        width={760}
      >
        <Form form={form} layout="vertical" initialValues={{ strategy: 'rolling', namespace: 'default', replicas: 1 }}>
          <Form.Item label="Namespace" name="namespace" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="Name" name="name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="Image" name="image" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="Replicas" name="replicas"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item>
          <Form.Item label="Strategy" name="strategy"><Select options={[{ value: 'rolling' }, { value: 'blue-green' }, { value: 'canary' }]} /></Form.Item>
          <Space>
            <Button onClick={() => void doPreview()}>Preview</Button>
          </Space>
          <pre style={{ marginTop: 12, maxHeight: 260, overflow: 'auto' }}>{preview || '# 先点击 Preview'}</pre>
        </Form>
      </Modal>
    </div>
  );
};

export default RolloutPanel;
