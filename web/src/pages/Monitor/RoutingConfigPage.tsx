import React, { useEffect, useRef, useState } from 'react';
import { Button, Card, Form, Input, Modal, Popconfirm, Select, Space, Table, message } from 'antd';
import { Api } from '../../api';
import ScopeSelector, { type ScopeValue } from './components/ScopeSelector';
import { GuidedFormItem } from '../../components/FormGuidance';

type RouteRow = {
  id: string;
  scope: string;
  severity: string;
  channelIds: string[];
  channel_ids_json: string;
  enabled: boolean;
};

type RouteFormValues = {
  scope: string;
  severity: string;
  channelIDs: string;
  enabled: boolean;
};

const readStoredProjectId = (): string | undefined => {
  if (typeof window === 'undefined') return undefined;
  const value = window.localStorage.getItem('projectId');
  return value || undefined;
};

const normalizeProjectId = (value?: string): string | undefined => {
  const trimmed = (value || '').trim();
  return trimmed || undefined;
};

const RoutingConfigPage: React.FC = () => {
  const [createForm] = Form.useForm<RouteFormValues>();
  const [editForm] = Form.useForm<RouteFormValues>();
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [rows, setRows] = useState<RouteRow[]>([]);
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<RouteRow | null>(null);
  const [scope, setScope] = useState<ScopeValue>({ scope: 'global', projectId: readStoredProjectId() });
  const mountedRef = useRef(true);
  const loadSeqRef = useRef(0);
  const normalizedProjectId = normalizeProjectId(scope.projectId);
  const currentProjectId = scope.scope === 'project' ? normalizedProjectId : undefined;

  const parseChannelIDs = (raw: string): string[] => raw.split(',').map((x) => x.trim()).filter(Boolean);

  const normalizeChannelIds = (value: unknown): string[] => {
    if (Array.isArray(value)) return value.map((x) => String(x)).filter(Boolean);
    if (typeof value === 'string') {
      try {
        const parsed = JSON.parse(value);
        if (Array.isArray(parsed)) return parsed.map((x) => String(x)).filter(Boolean);
      } catch {
        return parseChannelIDs(value);
      }
    }
    return [];
  };

  const projectIdForScope = (routeScope: string): string | undefined => (routeScope === 'project' ? normalizedProjectId : undefined);
  const ensureProjectScopeReady = (routeScope: string): boolean => {
    if (routeScope !== 'project') return true;
    if (normalizedProjectId) return true;
    message.error('项目作用域操作需要先选择项目ID');
    return false;
  };

  const load = async (showError = true): Promise<boolean> => {
    const seq = loadSeqRef.current + 1;
    loadSeqRef.current = seq;
    setLoading(true);
    try {
      const res = await Api.monitoring.getSeverityRoutes({ projectId: currentProjectId });
      const list = (res?.data as any)?.list || [];
      if (!mountedRef.current || seq !== loadSeqRef.current) return false;
      setRows(
        list.map((item: any) => {
          const channelIds = normalizeChannelIds(item.channel_ids_json ?? item.channel_ids ?? []);
          return {
            id: String(item.id),
            scope: item.scope || 'global',
            severity: item.severity || '',
            channelIds,
            channel_ids_json: typeof item.channel_ids_json === 'string' ? item.channel_ids_json : JSON.stringify(channelIds),
            enabled: item.enabled !== false,
          };
        }),
      );
      return true;
    } catch {
      if (!mountedRef.current || seq !== loadSeqRef.current) return false;
      if (showError) message.error('路由列表加载失败');
      return false;
    } finally {
      if (mountedRef.current && seq === loadSeqRef.current) {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      loadSeqRef.current += 1;
    };
  }, []);

  useEffect(() => {
    if (scope.scope === 'project' && !currentProjectId) {
      loadSeqRef.current += 1;
      setRows([]);
      setLoading(false);
      return;
    }
    void load();
  }, [scope.scope, currentProjectId]);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    if (normalizedProjectId) {
      window.localStorage.setItem('projectId', normalizedProjectId);
    } else {
      window.localStorage.removeItem('projectId');
    }
  }, [normalizedProjectId]);

  const handleCreate = async () => {
    try {
      const values = await createForm.validateFields();
      if (!ensureProjectScopeReady(values.scope)) {
        return;
      }
      setSubmitting(true);
      try {
        await Api.monitoring.createSeverityRoute({
          projectId: projectIdForScope(values.scope),
          scope: values.scope,
          severity: values.severity,
          channelIds: parseChannelIDs(values.channelIDs),
          enabled: values.enabled,
        });
        message.success('路由创建成功');
        setCreateOpen(false);
        createForm.resetFields();
        const refreshed = await load(false);
        if (!refreshed) message.warning('路由创建成功，但列表刷新失败');
      } catch {
        message.error('路由创建失败');
      } finally {
        setSubmitting(false);
      }
    } catch {
      // form validation errors handled by antd
    }
  };

  const handleOpenEdit = (record: RouteRow) => {
    setEditing(record);
    editForm.setFieldsValue({
      scope: record.scope || 'global',
      severity: record.severity || '',
      channelIDs: record.channelIds.join(','),
      enabled: record.enabled,
    });
  };

  const handleUpdate = async () => {
    if (!editing) return;
    try {
      const values = await editForm.validateFields();
      if (!ensureProjectScopeReady(values.scope)) {
        return;
      }
      setSubmitting(true);
      try {
        await Api.monitoring.updateSeverityRouteByID(editing.id, {
          projectId: projectIdForScope(values.scope),
          scope: values.scope,
          severity: values.severity,
          channelIds: parseChannelIDs(values.channelIDs),
          enabled: values.enabled,
        });
        message.success('路由更新成功');
        setEditing(null);
        const refreshed = await load(false);
        if (!refreshed) message.warning('路由更新成功，但列表刷新失败');
      } catch {
        message.error('路由更新失败');
      } finally {
        setSubmitting(false);
      }
    } catch {
      // form validation errors handled by antd
    }
  };

  const handleDelete = async (record: RouteRow) => {
    if (!ensureProjectScopeReady(record.scope)) {
      return;
    }
    setSubmitting(true);
    try {
      await Api.monitoring.deleteSeverityRoute(record.id, projectIdForScope(record.scope));
      message.success('路由删除成功');
      const refreshed = await load(false);
      if (!refreshed) message.warning('路由删除成功，但列表刷新失败');
    } catch {
      message.error('路由删除失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Card
      title="路由配置"
      size="small"
      extra={(
        <Space size="small">
          <ScopeSelector value={scope} onChange={setScope} />
          <Button type="primary" onClick={() => setCreateOpen(true)} size="small">
            新增路由
          </Button>
        </Space>
      )}
    >
      <Table
        rowKey="id"
        size="small"
        loading={loading}
        dataSource={rows}
        pagination={false}
        columns={[
          { title: '作用域', dataIndex: 'scope' },
          { title: '级别', dataIndex: 'severity' },
          { title: '渠道', dataIndex: 'channel_ids_json', render: (v: string) => v || '[]' },
          { title: '启用', dataIndex: 'enabled', render: (v: boolean) => (v ? '是' : '否') },
          {
            title: '操作',
            key: 'actions',
            render: (_value: unknown, record: RouteRow) => (
              <Space>
                <Button type="link" onClick={() => handleOpenEdit(record)} size="small">
                  编辑
                </Button>
                <Popconfirm title="确定删除此路由？" onConfirm={() => handleDelete(record)}>
                  <Button type="link" danger size="small">
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />
      <Modal
        title="新增路由"
        open={createOpen}
        okText="保存"
        confirmLoading={submitting}
        onOk={() => void handleCreate()}
        onCancel={() => {
          setCreateOpen(false);
          createForm.resetFields();
        }}
      >
        <Form
          form={createForm}
          layout="vertical"
          size="small"
          initialValues={{
            scope: 'global',
            severity: '',
            channelIDs: '',
            enabled: true,
          }}
        >
          <GuidedFormItem label="作用域" name="scope" rules={[{ required: true, message: '请输入作用域' }]}>
            <Input />
          </GuidedFormItem>
          <GuidedFormItem label="级别" name="severity" rules={[{ required: true, message: '请输入级别' }]}>
            <Input placeholder="critical/warning/info" />
          </GuidedFormItem>
          <GuidedFormItem label="渠道ID" name="channelIDs" rules={[{ required: true, message: '请输入渠道ID' }]}>
            <Input placeholder="渠道ID，逗号分隔" />
          </GuidedFormItem>
          <Form.Item label="状态" name="enabled" rules={[{ required: true, message: '请选择状态' }]}>
            <Select
              options={[
                { label: '启用', value: true },
                { label: '禁用', value: false },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title="编辑路由"
        open={!!editing}
        okText="保存"
        confirmLoading={submitting}
        onOk={() => void handleUpdate()}
        onCancel={() => {
          setEditing(null);
          editForm.resetFields();
        }}
      >
        <Form
          form={editForm}
          layout="vertical"
          size="small"
          initialValues={{
            scope: 'global',
            severity: '',
            channelIDs: '',
            enabled: true,
          }}
        >
          <GuidedFormItem label="作用域" name="scope" rules={[{ required: true, message: '请输入作用域' }]}>
            <Input />
          </GuidedFormItem>
          <GuidedFormItem label="级别" name="severity" rules={[{ required: true, message: '请输入级别' }]}>
            <Input placeholder="critical/warning/info" />
          </GuidedFormItem>
          <GuidedFormItem label="渠道ID" name="channelIDs" rules={[{ required: true, message: '请输入渠道ID' }]}>
            <Input placeholder="渠道ID，逗号分隔" />
          </GuidedFormItem>
          <Form.Item label="状态" name="enabled" rules={[{ required: true, message: '请选择状态' }]}>
            <Select
              options={[
                { label: '启用', value: true },
                { label: '禁用', value: false },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default RoutingConfigPage;
