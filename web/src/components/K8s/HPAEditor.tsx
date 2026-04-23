import React from 'react';
import { Button, Form, Input, InputNumber, Modal, Space, Table, message } from 'antd';
import { Api } from '../../api';
import { GuidedFormItem } from '../FormGuidance';

interface Props {
  clusterId: string;
  actions?: HPAEditorActions;
}

interface HPAEditorLoadResult {
  list: any[];
}

interface SaveHPAInput {
  clusterId: string;
  existing: any[];
  hpa: any;
}

interface RemoveHPAInput {
  clusterId: string;
  name: string;
  namespace: string;
}

export interface HPAEditorActions {
  load: () => Promise<HPAEditorLoadResult>;
  save: (input: SaveHPAInput) => Promise<void>;
  remove: (input: RemoveHPAInput) => Promise<void>;
}

interface HPAEditorViewProps {
  loading: boolean;
  list: any[];
  open: boolean;
  form: ReturnType<typeof Form.useForm>[0];
  onRefresh: () => void;
  onOpen: () => void;
  onClose: () => void;
  onSave: () => void;
  onRemove: (name: string, namespace: string) => void;
}

const HPAEditorView: React.FC<HPAEditorViewProps> = ({
  loading,
  list,
  open,
  form,
  onRefresh,
  onOpen,
  onClose,
  onSave,
  onRemove,
}) => (
  <div>
    <Space style={{ marginBottom: 12 }}>
      <Button onClick={onRefresh} loading={loading}>刷新</Button>
      <Button type="primary" onClick={onOpen}>新增/更新 HPA</Button>
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
        { title: '操作', render: (_: any, r: any) => <Button danger size="small" onClick={() => onRemove(r.name, r.namespace)}>删除</Button> },
      ]}
      pagination={false}
    />

    <Modal title="HPA 策略" open={open} onCancel={onClose} onOk={() => void onSave()}>
      <Form form={form} layout="vertical" initialValues={{ namespace: 'default', target_ref_kind: 'Deployment', min_replicas: 1, max_replicas: 3, cpu_utilization: 70, memory_utilization: 75 }}>
        <GuidedFormItem label="Namespace" name="namespace" rules={[{ required: true }]}><Input /></GuidedFormItem>
        <GuidedFormItem label="Name" name="name" rules={[{ required: true }]}><Input /></GuidedFormItem>
        <GuidedFormItem label="Target Kind" name="target_ref_kind" rules={[{ required: true }]}><Input /></GuidedFormItem>
        <GuidedFormItem label="Target Name" name="target_ref_name" rules={[{ required: true }]}><Input /></GuidedFormItem>
        <GuidedFormItem label="Min Replicas" name="min_replicas" rules={[{ required: true }]}><InputNumber min={1} style={{ width: '100%' }} /></GuidedFormItem>
        <GuidedFormItem label="Max Replicas" name="max_replicas" rules={[{ required: true }]}><InputNumber min={1} style={{ width: '100%' }} /></GuidedFormItem>
        <GuidedFormItem label="CPU Utilization %" name="cpu_utilization"><InputNumber min={1} max={100} style={{ width: '100%' }} /></GuidedFormItem>
        <GuidedFormItem label="Memory Utilization %" name="memory_utilization"><InputNumber min={1} max={100} style={{ width: '100%' }} /></GuidedFormItem>
      </Form>
    </Modal>
  </div>
);

function useHPAEditorActions(clusterId: string): HPAEditorActions {
  const load = React.useCallback(async () => {
    try {
      const res = await Api.kubernetes.listHPA(clusterId);
      return { list: res.data.list || [] };
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载 HPA 失败');
      throw err;
    }
  }, [clusterId]);

  const save = React.useCallback(async ({ clusterId: targetClusterId, existing, hpa }: SaveHPAInput) => {
    const found = existing.find((item) => item.name === hpa.name && item.namespace === hpa.namespace);
    if (found) {
      await Api.kubernetes.updateHPA(targetClusterId, hpa.name, hpa);
      message.success('HPA 已更新');
      return;
    }

    await Api.kubernetes.createHPA(targetClusterId, hpa);
    message.success('HPA 已创建');
  }, []);

  const remove = React.useCallback(async ({ clusterId: targetClusterId, name, namespace }: RemoveHPAInput) => {
    await Api.kubernetes.deleteHPA(targetClusterId, name, namespace);
    message.success('HPA 已删除');
  }, []);

  return React.useMemo(() => ({
    load,
    save,
    remove,
  }), [load, remove, save]);
}

const HPAEditor: React.FC<Props> = ({ clusterId, actions: actionsProp }) => {
  const defaultActions = useHPAEditorActions(clusterId);
  const actions = actionsProp || defaultActions;
  const [loading, setLoading] = React.useState(false);
  const [list, setList] = React.useState<any[]>([]);
  const [open, setOpen] = React.useState(false);
  const [form] = Form.useForm();

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const data = await actions.load();
      setList(data.list);
    } finally {
      setLoading(false);
    }
  }, [actions]);

  React.useEffect(() => { void load(); }, [load]);

  const save = async () => {
    const v = await form.validateFields();
    await actions.save({ clusterId, existing: list, hpa: v });
    setOpen(false);
    form.resetFields();
    await load();
  };

  const remove = async (name: string, namespace: string) => {
    await actions.remove({ clusterId, name, namespace });
    await load();
  };

  return (
    <HPAEditorView
      loading={loading}
      list={list}
      open={open}
      form={form}
      onRefresh={() => void load()}
      onOpen={() => setOpen(true)}
      onClose={() => setOpen(false)}
      onSave={() => void save()}
      onRemove={(name, namespace) => void remove(name, namespace)}
    />
  );
};

export default HPAEditor;
