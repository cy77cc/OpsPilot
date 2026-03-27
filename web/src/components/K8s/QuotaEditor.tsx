import React from 'react';
import { Button, Form, Input, Modal, Space, Table, Tabs, message } from 'antd';
import { Api } from '../../api';

interface Props {
  clusterId: string;
}

const QuotaEditor: React.FC<Props> = ({ clusterId }) => {
  const [loading, setLoading] = React.useState(false);
  const [quotas, setQuotas] = React.useState<any[]>([]);
  const [limits, setLimits] = React.useState<any[]>([]);
  const [quotaOpen, setQuotaOpen] = React.useState(false);
  const [limitOpen, setLimitOpen] = React.useState(false);
  const [quotaForm] = Form.useForm();
  const [limitForm] = Form.useForm();

  const parseKV = (raw: string): Record<string, string> => {
    const out: Record<string, string> = {};
    String(raw || '').split('\n').forEach((line) => {
      const trimmed = line.trim();
      if (!trimmed) return;
      const [k, ...rest] = trimmed.split('=');
      if (!k) return;
      out[k.trim()] = rest.join('=').trim();
    });
    return out;
  };

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const [qRes, lRes] = await Promise.all([
        Api.kubernetes.listQuotas(clusterId),
        Api.kubernetes.listLimitRanges(clusterId),
      ]);
      setQuotas(qRes.data.list || []);
      setLimits(lRes.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载配额失败');
    } finally {
      setLoading(false);
    }
  }, [clusterId]);

  React.useEffect(() => { void load(); }, [load]);

  const saveQuota = async () => {
    const v = await quotaForm.validateFields();
    await Api.kubernetes.applyQuota(clusterId, { namespace: v.namespace, name: v.name, hard: parseKV(v.hard) });
    message.success('Quota 已应用');
    setQuotaOpen(false);
    quotaForm.resetFields();
    await load();
  };

  const saveLimit = async () => {
    const v = await limitForm.validateFields();
    await Api.kubernetes.createLimitRange(clusterId, {
      namespace: v.namespace,
      name: v.name,
      default: parseKV(v.default_values),
      default_request: parseKV(v.default_request),
      min: parseKV(v.min_values),
      max: parseKV(v.max_values),
    });
    message.success('LimitRange 已应用');
    setLimitOpen(false);
    limitForm.resetFields();
    await load();
  };

  const removeQuota = async (name: string, namespace: string) => {
    await Api.kubernetes.deleteQuota(clusterId, name, namespace);
    message.success('Quota 已删除');
    await load();
  };

  return (
    <Tabs
      items={[
        {
          key: 'quotas', label: 'ResourceQuotas', children: (
            <div>
              <Space style={{ marginBottom: 12 }}>
                <Button onClick={load} loading={loading}>刷新</Button>
                <Button type="primary" onClick={() => setQuotaOpen(true)}>新增/更新 Quota</Button>
              </Space>
              <Table
                rowKey={(r) => `${r.namespace}:${r.name}`}
                dataSource={quotas}
                loading={loading}
                columns={[
                  { title: 'Name', dataIndex: 'name' },
                  { title: 'Namespace', dataIndex: 'namespace' },
                  { title: 'Hard', dataIndex: 'hard', render: (h: Record<string, string>) => Object.entries(h || {}).map(([k, v]) => `${k}=${v}`).join(' ; ') || '-' },
                  { title: '操作', render: (_: any, r: any) => <Button size="small" danger onClick={() => void removeQuota(r.name, r.namespace)}>删除</Button> },
                ]}
                pagination={false}
              />
            </div>
          ),
        },
        {
          key: 'limits', label: 'LimitRanges', children: (
            <div>
              <Space style={{ marginBottom: 12 }}>
                <Button onClick={load} loading={loading}>刷新</Button>
                <Button type="primary" onClick={() => setLimitOpen(true)}>新增/更新 LimitRange</Button>
              </Space>
              <Table
                rowKey={(r) => `${r.namespace}:${r.name}`}
                dataSource={limits}
                loading={loading}
                columns={[
                  { title: 'Name', dataIndex: 'name' },
                  { title: 'Namespace', dataIndex: 'namespace' },
                  { title: 'Limits', dataIndex: 'limits', render: (v: any) => <pre style={{ margin: 0, maxHeight: 120, overflow: 'auto' }}>{JSON.stringify(v, null, 2)}</pre> },
                ]}
                pagination={false}
              />
            </div>
          ),
        },
      ]}
    >
      <Modal title="Quota" open={quotaOpen} onCancel={() => setQuotaOpen(false)} onOk={() => void saveQuota()}>
        <Form form={quotaForm} layout="vertical" initialValues={{ namespace: 'default', hard: 'limits.cpu=4\nlimits.memory=8Gi\npods=20' }}>
          <Form.Item label="Namespace" name="namespace" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="Name" name="name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="Hard (每行 key=value)" name="hard" rules={[{ required: true }]}><Input.TextArea rows={6} /></Form.Item>
        </Form>
      </Modal>
      <Modal title="LimitRange" open={limitOpen} onCancel={() => setLimitOpen(false)} onOk={() => void saveLimit()}>
        <Form form={limitForm} layout="vertical" initialValues={{ namespace: 'default', default_values: 'cpu=500m\nmemory=512Mi', default_request: 'cpu=100m\nmemory=128Mi', min_values: 'cpu=50m\nmemory=64Mi', max_values: 'cpu=2\nmemory=2Gi' }}>
          <Form.Item label="Namespace" name="namespace" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="Name" name="name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="Default" name="default_values"><Input.TextArea rows={3} /></Form.Item>
          <Form.Item label="Default Request" name="default_request"><Input.TextArea rows={3} /></Form.Item>
          <Form.Item label="Min" name="min_values"><Input.TextArea rows={3} /></Form.Item>
          <Form.Item label="Max" name="max_values"><Input.TextArea rows={3} /></Form.Item>
        </Form>
      </Modal>
    </Tabs>
  );
};

export default QuotaEditor;
