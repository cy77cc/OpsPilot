import React, { useEffect, useRef, useState, useCallback } from 'react';
import { Button, Card, Form, Input, Modal, Popconfirm, Select, Space, Table, message } from 'antd';
import { Api } from '../../api';
import { useScope } from '../../app/scope/useScope';
import { GuidedFormItem } from '../../components/FormGuidance';
import ScopeSelector, { type ScopeValue } from './components/ScopeSelector';
import { channelFieldGuides } from './channelFieldGuides';
import { useRegisterMonitorRefresh } from './MonitorRefreshContext';

type ChannelTestForm = {
  provider: string;
  target?: string;
  configJson?: string;
};

type ChannelFormValues = {
  channelName: string;
  channelProvider: string;
  channelTarget?: string;
  channelConfigJson?: string;
};

type ChannelRow = {
  id: string;
  name: string;
  provider?: string;
  target?: string;
  configJson?: string;
};

const normalizeProjectId = (value?: string): string | undefined => {
  const trimmed = (value || '').trim();
  return trimmed || undefined;
};

const ChannelsConfigPage: React.FC = () => {
  const [form] = Form.useForm<ChannelTestForm>();
  const [createForm] = Form.useForm<ChannelFormValues>();
  const [editForm] = Form.useForm<ChannelFormValues>();
  const [submitting, setSubmitting] = useState(false);
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<ChannelRow[]>([]);
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<ChannelRow | null>(null);
  const { projectId: storedProjectId, setProjectId } = useScope();
  const [scope, setScope] = useState<ScopeValue>({ scope: 'global', projectId: storedProjectId });
  const mountedRef = useRef(true);
  const loadSeqRef = useRef(0);
  const currentProjectId = scope.scope === 'project' ? normalizeProjectId(scope.projectId) : undefined;

  const ensureProjectScopeReady = (): boolean => {
    if (scope.scope !== 'project') {return true;}
    if (currentProjectId) {return true;}
    message.error('项目作用域操作需要先选择项目ID');
    return false;
  };

  const loadChannels = useCallback(async (showError = true): Promise<boolean> => {
    const seq = loadSeqRef.current + 1;
    loadSeqRef.current = seq;
    setLoading(true);
    try {
      const res = await Api.monitoring.listAlertChannels({ projectId: currentProjectId });
      const list = (res?.data as any)?.list || [];
      if (!mountedRef.current || seq !== loadSeqRef.current) {return false;}
      setRows(
        list.map((item: any) => ({
          id: String(item.id),
          name: item.name || '',
          provider: item.provider || '',
          target: item.target || '',
          configJson: item.configJson || item.config_json || '',
        })),
      );
      return true;
    } catch {
      if (!mountedRef.current || seq !== loadSeqRef.current) {return false;}
      if (showError) {message.error('渠道列表加载失败');}
      return false;
    } finally {
      if (mountedRef.current && seq === loadSeqRef.current) {
        setLoading(false);
      }
    }
  }, [currentProjectId]);

  useRegisterMonitorRefresh(loadChannels, loading);

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
    void loadChannels();
  }, [scope.scope, currentProjectId]);

  useEffect(() => {
    setScope((previous) => (
      previous.projectId === storedProjectId
        ? previous
        : { ...previous, projectId: storedProjectId }
    ));
  }, [storedProjectId]);

  useEffect(() => {
    const projectId = normalizeProjectId(scope.projectId);
    if (projectId !== storedProjectId) {
      setProjectId(projectId);
    }
  }, [scope.projectId, storedProjectId, setProjectId]);

  const handleTestSend = async () => {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      await Api.monitoring.testAlertChannel(values);
      message.success('测试发送成功');
    } catch {
      message.error('测试发送失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleCreate = async () => {
    if (!ensureProjectScopeReady()) {return;}
    try {
      const values = await createForm.validateFields();
      setSubmitting(true);
      try {
        await Api.monitoring.createAlertChannel({
          name: values.channelName,
          provider: values.channelProvider,
          target: values.channelTarget,
          configJson: values.channelConfigJson,
          projectId: currentProjectId,
        });
        message.success('渠道创建成功');
        setCreateOpen(false);
        createForm.resetFields();
        const refreshed = await loadChannels(false);
        if (!refreshed) {message.warning('渠道创建成功，但列表刷新失败');}
      } catch {
        message.error('渠道创建失败');
      } finally {
        setSubmitting(false);
      }
    } catch {
      // form validation errors handled by antd
    }
  };

  const handleOpenEdit = (record: ChannelRow) => {
    setEditing(record);
    editForm.setFieldsValue({
      channelName: record.name,
      channelProvider: record.provider || 'webhook',
      channelTarget: record.target || '',
      channelConfigJson: record.configJson || '{}',
    });
  };

  const handleUpdate = async () => {
    if (!editing) {return;}
    if (!ensureProjectScopeReady()) {return;}
    try {
      const values = await editForm.validateFields();
      setSubmitting(true);
      try {
        await Api.monitoring.updateAlertChannel(editing.id, {
          name: values.channelName,
          provider: values.channelProvider,
          target: values.channelTarget,
          configJson: values.channelConfigJson,
          projectId: currentProjectId,
        });
        message.success('渠道更新成功');
        setEditing(null);
        const refreshed = await loadChannels(false);
        if (!refreshed) {message.warning('渠道更新成功，但列表刷新失败');}
      } catch {
        message.error('渠道更新失败');
      } finally {
        setSubmitting(false);
      }
    } catch {
      // form validation errors handled by antd
    }
  };

  const handleDelete = async (id: string) => {
    setSubmitting(true);
    try {
      await Api.monitoring.deleteAlertChannel(id);
      message.success('渠道删除成功');
      const refreshed = await loadChannels(false);
      if (!refreshed) {message.warning('渠道删除成功，但列表刷新失败');}
    } catch (err: any) {
      const code = String(err?.code ?? err?.status ?? err?.response?.status ?? '');
      const blockers = err?.data?.blockers || err?.response?.data?.blockers || [];
      if (code === '409' && Array.isArray(blockers) && blockers.length > 0) {
        Modal.error({
          title: '删除失败：存在依赖',
          content: (
            <div>
              {blockers.map((b: any, idx: number) => (
                <div key={`${b.type || 'blocker'}-${idx}`}>{`${b.type || 'unknown'}: ${b.count ?? 0}`}</div>
              ))}
            </div>
          ),
        });
        return;
      }
      message.error('渠道删除失败');
      } finally {
      setSubmitting(false);
      }
      };

      return (
      <Space orientation="vertical" style={{ width: '100%' }}>      <Card
        title="通知渠道配置"
       
        extra={(
          <Space>
            <ScopeSelector value={scope} onChange={setScope} />
            <Button type="primary" onClick={() => setCreateOpen(true)}>
              新增渠道
            </Button>
          </Space>
        )}
      >
        <Table
          rowKey="id"
         
          loading={loading}
          dataSource={rows}
          pagination={false}
          columns={[
            { title: '名称', dataIndex: 'name' },
            { title: 'Provider', dataIndex: 'provider', render: (v: string | undefined) => v || '-' },
            { title: '目标地址', dataIndex: 'target', render: (v: string | undefined) => v || '-' },
            {
              title: '操作',
              key: 'actions',
              render: (_v: unknown, record: ChannelRow) => (
                <Space>
                  <Button type="link" onClick={() => handleOpenEdit(record)}>
                    编辑
                  </Button>
                  <Popconfirm title="确定删除此渠道？" onConfirm={() => handleDelete(record.id)}>
                    <Button type="link" danger>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>
      <Card title="渠道测试发送">
        <Form
          form={form}
          layout="vertical"
         
          initialValues={{
            provider: 'webhook',
            target: '',
            configJson: '{}',
          }}
        >
          <GuidedFormItem
            label="Provider"
            name="provider"
            guide={channelFieldGuides.provider}
            rules={[{ required: true, message: '请选择 provider' }]}
          >
            <Select
              options={[
                { label: 'Webhook', value: 'webhook' },
                { label: 'Log', value: 'log' },
                { label: 'Email', value: 'email' },
              ]}
            />
          </GuidedFormItem>
          <GuidedFormItem label="目标地址" name="target" guide={channelFieldGuides.target}>
            <Input placeholder="https://example.com/hook" />
          </GuidedFormItem>
          <GuidedFormItem
            label="配置 JSON"
            name="configJson"
            guide={channelFieldGuides.configJson}
            aiAssist={{
              scene: 'monitoring',
              fieldMeta: {
                key: 'config_json',
                label: '配置 JSON',
                purpose: 'Generate valid channel configuration JSON',
                rules: 'Return valid JSON only. No markdown fences. No explanation.',
              },
              getFormContext: () => ({
                provider: form.getFieldValue('provider'),
                target: form.getFieldValue('target'),
              }),
            }}
          >
            <Input.TextArea rows={4} />
          </GuidedFormItem>
          <Space>
            <Button type="primary" onClick={handleTestSend} loading={submitting}>
              测试发送
            </Button>
          </Space>
        </Form>
      </Card>
      <Modal
        title="新增渠道"
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
         
          initialValues={{ channelName: '', channelProvider: 'webhook', channelTarget: '', channelConfigJson: '{}' }}
        >
          <GuidedFormItem label="名称" name="channelName" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="渠道名称" />
          </GuidedFormItem>
          <GuidedFormItem
            label="Provider"
            name="channelProvider"
            guide={channelFieldGuides.provider}
            rules={[{ required: true, message: '请输入 provider' }]}
          >
            <Input />
          </GuidedFormItem>
          <GuidedFormItem label="目标地址" name="channelTarget" guide={channelFieldGuides.target}>
            <Input />
          </GuidedFormItem>
          <GuidedFormItem
            label="配置 JSON"
            name="channelConfigJson"
            guide={channelFieldGuides.configJson}
            aiAssist={{
              scene: 'monitoring',
              fieldMeta: {
                key: 'config_json',
                label: '配置 JSON',
                purpose: 'Generate valid channel configuration JSON',
                rules: 'Return valid JSON only. No markdown fences. No explanation.',
              },
              getFormContext: () => ({
                provider: createForm.getFieldValue('channelProvider'),
                target: createForm.getFieldValue('channelTarget'),
              }),
            }}
          >
            <Input.TextArea rows={4} />
          </GuidedFormItem>
        </Form>
      </Modal>
      <Modal
        title="编辑渠道"
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
         
          initialValues={{ channelName: '', channelProvider: 'webhook', channelTarget: '', channelConfigJson: '{}' }}
        >
          <GuidedFormItem label="名称" name="channelName" rules={[{ required: true, message: '请输入名称' }]}>
            <Input />
          </GuidedFormItem>
          <GuidedFormItem
            label="Provider"
            name="channelProvider"
            guide={channelFieldGuides.provider}
            rules={[{ required: true, message: '请输入 provider' }]}
          >
            <Input />
          </GuidedFormItem>
          <GuidedFormItem label="目标地址" name="channelTarget" guide={channelFieldGuides.target}>
            <Input />
          </GuidedFormItem>
          <GuidedFormItem
            label="配置 JSON"
            name="channelConfigJson"
            guide={channelFieldGuides.configJson}
            aiAssist={{
              scene: 'monitoring',
              fieldMeta: {
                key: 'config_json',
                label: '配置 JSON',
                purpose: 'Generate valid channel configuration JSON',
                rules: 'Return valid JSON only. No markdown fences. No explanation.',
              },
              getFormContext: () => ({
                provider: editForm.getFieldValue('channelProvider'),
                target: editForm.getFieldValue('channelTarget'),
              }),
            }}
          >
            <Input.TextArea rows={4} />
          </GuidedFormItem>
        </Form>
      </Modal>
    </Space>
  );
};

export default ChannelsConfigPage;
