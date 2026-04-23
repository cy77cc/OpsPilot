import React from 'react';
import { Button, Form, Input, InputNumber, Modal, Space, Table, Tag } from 'antd';
import type { FormInstance } from 'antd';
import { useScope } from '../../app/scope/useScope';
import { GuidedFormItem } from '../FormGuidance';
import { useNamespacePolicyActions } from './hooks/useNamespacePolicyActions';
import type { NamespacePolicyActions } from './hooks/useNamespacePolicyActions';

interface Props {
  clusterId: string;
  actions?: NamespacePolicyActions;
}

interface NamespacePolicyViewProps {
  loading: boolean;
  namespaces: any[];
  bindings: any[];
  nsOpen: boolean;
  bindOpen: boolean;
  nsForm: FormInstance;
  bindForm: FormInstance;
  onRefresh: () => void;
  onOpenNamespace: () => void;
  onOpenBindings: () => void;
  onCloseNamespace: () => void;
  onCloseBindings: () => void;
  onCreateNamespace: () => void;
  onSaveBindings: () => void;
  onRemoveNamespace: (name: string) => void;
}

const NamespacePolicyView: React.FC<NamespacePolicyViewProps> = ({
  loading,
  namespaces,
  bindings,
  nsOpen,
  bindOpen,
  nsForm,
  bindForm,
  onRefresh,
  onOpenNamespace,
  onOpenBindings,
  onCloseNamespace,
  onCloseBindings,
  onCreateNamespace,
  onSaveBindings,
  onRemoveNamespace,
}) => (
  <div>
    <Space style={{ marginBottom: 12 }}>
      <Button onClick={onRefresh} loading={loading}>刷新</Button>
      <Button type="primary" onClick={onOpenNamespace}>新建 Namespace</Button>
      <Button onClick={onOpenBindings}>管理 Team 绑定</Button>
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
        { title: '操作', render: (_: any, r: any) => <Button danger type="link" onClick={() => onRemoveNamespace(r.name)}>删除</Button> },
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

    <Modal title="新建 Namespace" open={nsOpen} onCancel={onCloseNamespace} onOk={() => void onCreateNamespace()}>
      <Form form={nsForm} layout="vertical">
        <GuidedFormItem label="名称" name="name" rules={[{ required: true }]}><Input /></GuidedFormItem>
        <GuidedFormItem label="环境" name="env"><Input placeholder="development/staging/production" /></GuidedFormItem>
      </Form>
    </Modal>

    <Modal title="更新 Team Namespace 绑定" open={bindOpen} onCancel={onCloseBindings} onOk={() => void onSaveBindings()}>
      <Form form={bindForm} layout="vertical">
        <GuidedFormItem label="Team ID" name="team_id" rules={[{ required: true }]}><InputNumber min={1} style={{ width: '100%' }} /></GuidedFormItem>
        <GuidedFormItem label="Namespaces" name="namespaces" rules={[{ required: true, message: '至少一个 namespace' }]}>
          <Input placeholder="逗号分隔: default,dev,staging" onBlur={(e) => bindForm.setFieldValue('namespaces', String(e.target.value || '').split(',').map((x) => x.trim()).filter(Boolean))} />
        </GuidedFormItem>
      </Form>
    </Modal>
  </div>
);

const NamespacePolicyPanel: React.FC<Props> = ({ clusterId, actions: actionsProp }) => {
  const { teamId, setTeamId } = useScope();
  const defaultActions = useNamespacePolicyActions({ clusterId, teamId, setTeamId });
  const actions = actionsProp || defaultActions;
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
      const data = await actions.load();
      setNamespaces(data.namespaces);
      setBindings(data.bindings);
    } finally {
      setLoading(false);
    }
  }, [actions]);

  React.useEffect(() => { void load(); }, [load]);
  React.useEffect(() => {
    bindForm.setFieldValue('team_id', Number(teamId || 1));
  }, [bindForm, teamId]);

  const createNamespace = React.useCallback(async () => {
    const v = await nsForm.validateFields();
    await actions.createNamespace({ clusterId, name: v.name, env: v.env });
    setNsOpen(false);
    nsForm.resetFields();
    await load();
  }, [actions, clusterId, load, nsForm]);

  const saveBindings = React.useCallback(async () => {
    const v = await bindForm.validateFields();
    await actions.saveBindings({
      clusterId,
      teamId: String(v.team_id),
      namespaces: Array.isArray(v.namespaces) ? v.namespaces : [],
    });
    setBindOpen(false);
    bindForm.resetFields();
    await load();
  }, [actions, bindForm, clusterId, load]);

  const removeNamespace = React.useCallback(async (name: string) => {
    await actions.removeNamespace({ clusterId, name });
    await load();
  }, [actions, clusterId, load]);

  return (
    <NamespacePolicyView
      loading={loading}
      namespaces={namespaces}
      bindings={bindings}
      nsOpen={nsOpen}
      bindOpen={bindOpen}
      nsForm={nsForm}
      bindForm={bindForm}
      onRefresh={() => void load()}
      onOpenNamespace={() => setNsOpen(true)}
      onOpenBindings={() => setBindOpen(true)}
      onCloseNamespace={() => setNsOpen(false)}
      onCloseBindings={() => setBindOpen(false)}
      onCreateNamespace={() => void createNamespace()}
      onSaveBindings={() => void saveBindings()}
      onRemoveNamespace={(name) => void removeNamespace(name)}
    />
  );
};

export default NamespacePolicyPanel;
