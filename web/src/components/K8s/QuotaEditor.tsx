import React from 'react';
import { Button, Form, Input, Modal, Space, Table, Tabs, message } from 'antd';
import { Api } from '../../api';
import { GuidedFormItem } from '../FormGuidance';

interface Props {
  clusterId: string;
  actions?: QuotaEditorActions;
}

interface QuotaEditorLoadResult {
  quotas: any[];
  limits: any[];
}

interface SaveQuotaInput {
  clusterId: string;
  quota: {
    namespace: string;
    name: string;
    hard: Record<string, string>;
  };
}

interface SaveLimitInput {
  clusterId: string;
  limitRange: {
    namespace: string;
    name: string;
    default: Record<string, string>;
    default_request: Record<string, string>;
    min: Record<string, string>;
    max: Record<string, string>;
  };
}

interface RemoveQuotaInput {
  clusterId: string;
  name: string;
  namespace: string;
}

export interface QuotaEditorActions {
  load: () => Promise<QuotaEditorLoadResult>;
  saveQuota: (input: SaveQuotaInput) => Promise<void>;
  saveLimit: (input: SaveLimitInput) => Promise<void>;
  removeQuota: (input: RemoveQuotaInput) => Promise<void>;
}

interface QuotaEditorViewProps {
  loading: boolean;
  quotas: any[];
  limits: any[];
  quotaOpen: boolean;
  limitOpen: boolean;
  quotaForm: ReturnType<typeof Form.useForm>[0];
  limitForm: ReturnType<typeof Form.useForm>[0];
  onRefresh: () => void;
  onOpenQuota: () => void;
  onCloseQuota: () => void;
  onSaveQuota: () => void;
  onRemoveQuota: (name: string, namespace: string) => void;
  onOpenLimit: () => void;
  onCloseLimit: () => void;
  onSaveLimit: () => void;
}

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

function useQuotaEditorActions(clusterId: string): QuotaEditorActions {
  const load = React.useCallback(async () => {
    try {
      const [qRes, lRes] = await Promise.all([
        Api.kubernetes.listQuotas(clusterId),
        Api.kubernetes.listLimitRanges(clusterId),
      ]);

      return {
        quotas: qRes.data.list || [],
        limits: lRes.data.list || [],
      };
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载配额失败');
      throw err;
    }
  }, [clusterId]);

  const saveQuota = React.useCallback(async ({ clusterId: targetClusterId, quota }: SaveQuotaInput) => {
    await Api.kubernetes.applyQuota(targetClusterId, quota);
    message.success('Quota 已应用');
  }, []);

  const saveLimit = React.useCallback(async ({ clusterId: targetClusterId, limitRange }: SaveLimitInput) => {
    await Api.kubernetes.createLimitRange(targetClusterId, limitRange);
    message.success('LimitRange 已应用');
  }, []);

  const removeQuota = React.useCallback(async ({ clusterId: targetClusterId, name, namespace }: RemoveQuotaInput) => {
    await Api.kubernetes.deleteQuota(targetClusterId, name, namespace);
    message.success('Quota 已删除');
  }, []);

  return React.useMemo(() => ({
    load,
    saveQuota,
    saveLimit,
    removeQuota,
  }), [load, removeQuota, saveLimit, saveQuota]);
}

const QuotaEditorView: React.FC<QuotaEditorViewProps> = ({
  loading,
  quotas,
  limits,
  quotaOpen,
  limitOpen,
  quotaForm,
  limitForm,
  onRefresh,
  onOpenQuota,
  onCloseQuota,
  onSaveQuota,
  onRemoveQuota,
  onOpenLimit,
  onCloseLimit,
  onSaveLimit,
}) => (
  <>
    <Tabs
      items={[
        {
          key: 'quotas', label: 'ResourceQuotas', children: (
            <div>
              <Space style={{ marginBottom: 12 }}>
                <Button onClick={onRefresh} loading={loading}>刷新</Button>
                <Button type="primary" onClick={onOpenQuota}>新增/更新 Quota</Button>
              </Space>
              <Table
                rowKey={(r) => `${r.namespace}:${r.name}`}
                dataSource={quotas}
                loading={loading}
                columns={[
                  { title: 'Name', dataIndex: 'name' },
                  { title: 'Namespace', dataIndex: 'namespace' },
                  { title: 'Hard', dataIndex: 'hard', render: (h: Record<string, string>) => Object.entries(h || {}).map(([k, v]) => `${k}=${v}`).join(' ; ') || '-' },
                  { title: '操作', render: (_: any, r: any) => <Button size="small" danger onClick={() => onRemoveQuota(r.name, r.namespace)}>删除</Button> },
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
                <Button onClick={onRefresh} loading={loading}>刷新</Button>
                <Button type="primary" onClick={onOpenLimit}>新增/更新 LimitRange</Button>
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
    />
    <Modal title="Quota" open={quotaOpen} onCancel={onCloseQuota} onOk={() => void onSaveQuota()}>
      <Form form={quotaForm} layout="vertical" initialValues={{ namespace: 'default', hard: 'limits.cpu=4\nlimits.memory=8Gi\npods=20' }}>
        <GuidedFormItem label="Namespace" name="namespace" rules={[{ required: true }]}><Input /></GuidedFormItem>
        <GuidedFormItem label="Name" name="name" rules={[{ required: true }]}><Input /></GuidedFormItem>
        <GuidedFormItem label="Hard (每行 key=value)" name="hard" rules={[{ required: true }]}><Input.TextArea rows={6} /></GuidedFormItem>
      </Form>
    </Modal>
    <Modal title="LimitRange" open={limitOpen} onCancel={onCloseLimit} onOk={() => void onSaveLimit()}>
      <Form form={limitForm} layout="vertical" initialValues={{ namespace: 'default', default_values: 'cpu=500m\nmemory=512Mi', default_request: 'cpu=100m\nmemory=128Mi', min_values: 'cpu=50m\nmemory=64Mi', max_values: 'cpu=2\nmemory=2Gi' }}>
        <GuidedFormItem label="Namespace" name="namespace" rules={[{ required: true }]}><Input /></GuidedFormItem>
        <GuidedFormItem label="Name" name="name" rules={[{ required: true }]}><Input /></GuidedFormItem>
        <GuidedFormItem label="Default" name="default_values"><Input.TextArea rows={3} /></GuidedFormItem>
        <GuidedFormItem label="Default Request" name="default_request"><Input.TextArea rows={3} /></GuidedFormItem>
        <GuidedFormItem label="Min" name="min_values"><Input.TextArea rows={3} /></GuidedFormItem>
        <GuidedFormItem label="Max" name="max_values"><Input.TextArea rows={3} /></GuidedFormItem>
      </Form>
    </Modal>
  </>
);

const QuotaEditor: React.FC<Props> = ({ clusterId, actions: actionsProp }) => {
  const defaultActions = useQuotaEditorActions(clusterId);
  const actions = actionsProp || defaultActions;
  const [loading, setLoading] = React.useState(false);
  const [quotas, setQuotas] = React.useState<any[]>([]);
  const [limits, setLimits] = React.useState<any[]>([]);
  const [quotaOpen, setQuotaOpen] = React.useState(false);
  const [limitOpen, setLimitOpen] = React.useState(false);
  const [quotaForm] = Form.useForm();
  const [limitForm] = Form.useForm();

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const data = await actions.load();
      setQuotas(data.quotas);
      setLimits(data.limits);
    } finally {
      setLoading(false);
    }
  }, [actions]);

  React.useEffect(() => { void load(); }, [load]);

  const saveQuota = async () => {
    const v = await quotaForm.validateFields();
    await actions.saveQuota({
      clusterId,
      quota: {
        namespace: v.namespace,
        name: v.name,
        hard: parseKV(v.hard),
      },
    });
    setQuotaOpen(false);
    quotaForm.resetFields();
    await load();
  };

  const saveLimit = async () => {
    const v = await limitForm.validateFields();
    await actions.saveLimit({
      clusterId,
      limitRange: {
        namespace: v.namespace,
        name: v.name,
        default: parseKV(v.default_values),
        default_request: parseKV(v.default_request),
        min: parseKV(v.min_values),
        max: parseKV(v.max_values),
      },
    });
    setLimitOpen(false);
    limitForm.resetFields();
    await load();
  };

  const removeQuota = async (name: string, namespace: string) => {
    await actions.removeQuota({ clusterId, name, namespace });
    await load();
  };

  return (
    <QuotaEditorView
      loading={loading}
      quotas={quotas}
      limits={limits}
      quotaOpen={quotaOpen}
      limitOpen={limitOpen}
      quotaForm={quotaForm}
      limitForm={limitForm}
      onRefresh={() => void load()}
      onOpenQuota={() => setQuotaOpen(true)}
      onCloseQuota={() => setQuotaOpen(false)}
      onSaveQuota={() => void saveQuota()}
      onRemoveQuota={(name, namespace) => void removeQuota(name, namespace)}
      onOpenLimit={() => setLimitOpen(true)}
      onCloseLimit={() => setLimitOpen(false)}
      onSaveLimit={() => void saveLimit()}
    />
  );
};

export default QuotaEditor;
