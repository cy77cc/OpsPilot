import React, { useEffect, useRef, useState } from 'react';
import { Button, Card, Drawer, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, message } from 'antd';
import { Api } from '../../api';

type EffectiveRuleRow = {
  id: string;
  name: string;
  metric?: string;
  severity: string;
  threshold?: number;
  scope?: string;
  inherit_key?: string;
};

type RuleFormValues = {
  name: string;
  metric?: string;
  severity: string;
  threshold: number;
};

type RuleChannelBindingRow = {
  channelId: string;
  priority?: number;
  enabled: boolean;
};

type RuleChannelBindingFormValues = {
  channelId: string;
  priority?: number;
  enabled: boolean;
};

const RulesConfigPage: React.FC = () => {
  const [createForm] = Form.useForm<RuleFormValues>();
  const [editForm] = Form.useForm<RuleFormValues>();
  const [bindingForm] = Form.useForm<RuleChannelBindingFormValues>();
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [rows, setRows] = useState<EffectiveRuleRow[]>([]);
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<EffectiveRuleRow | null>(null);
  const [bindingOpen, setBindingOpen] = useState(false);
  const [bindingRule, setBindingRule] = useState<EffectiveRuleRow | null>(null);
  const [bindings, setBindings] = useState<RuleChannelBindingRow[]>([]);
  const [bindingLoading, setBindingLoading] = useState(false);
  const [bindingSubmitting, setBindingSubmitting] = useState(false);
  const [editingBindingChannelId, setEditingBindingChannelId] = useState<string | null>(null);
  const mountedRef = useRef(true);
  const bindingLoadSeqRef = useRef(0);
  const activeBindingRuleIdRef = useRef<string | null>(null);
  const bindingOpenRef = useRef(false);

  const getProjectId = (): string | undefined => {
    if (typeof window === 'undefined') return undefined;
    const value = window.localStorage.getItem('projectId');
    return value || undefined;
  };

  const load = async (showError = true): Promise<boolean> => {
    setLoading(true);
    try {
      const res = await Api.monitoring.getEffectiveRules({ page: 1, pageSize: 50 });
      const list = (res?.data as any)?.list || [];
      if (!mountedRef.current) return false;
      setRows(
        list.map((item: any) => ({
          id: String(item.id),
          name: item.name || '',
          metric: item.metric || '',
          severity: item.severity || '',
          threshold: item.threshold,
          scope: item.scope,
          inherit_key: item.inherit_key,
        })),
      );
      return true;
    } catch {
      if (showError) {
        message.error('规则列表加载失败');
      }
      return false;
    } finally {
      if (mountedRef.current) {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    void load();
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const handleCreate = async () => {
    try {
      const values = await createForm.validateFields();
      setSubmitting(true);
      try {
        await Api.monitoring.createAlertRule({
          name: values.name,
          metric: values.metric || '',
          severity: values.severity,
          threshold: values.threshold,
        });
        message.success('规则创建成功');
        setCreateOpen(false);
        createForm.resetFields();
        const refreshed = await load(false);
        if (!refreshed) {
          message.warning('规则创建成功，但列表刷新失败');
        }
      } catch {
        message.error('规则创建失败');
      } finally {
        setSubmitting(false);
      }
    } catch (error: any) {
      if (!Array.isArray(error?.errorFields)) {
        message.error('规则创建失败');
      }
    }
  };

  const handleOpenEdit = (record: EffectiveRuleRow) => {
    setEditing(record);
    editForm.setFieldsValue({
      name: record.name,
      severity: record.severity || 'warning',
      threshold: record.threshold,
    });
  };

  const handleUpdate = async () => {
    if (!editing) return;
    try {
      const values = await editForm.validateFields();
      setSubmitting(true);
      try {
        await Api.monitoring.updateAlertRule(editing.id, {
          name: values.name,
          severity: values.severity,
          threshold: values.threshold,
        });
        message.success('规则更新成功');
        setEditing(null);
        const refreshed = await load(false);
        if (!refreshed) {
          message.warning('规则更新成功，但列表刷新失败');
        }
      } catch {
        message.error('规则更新失败');
      } finally {
        setSubmitting(false);
      }
    } catch (error: any) {
      if (!Array.isArray(error?.errorFields)) {
        message.error('规则更新失败');
      }
    }
  };

  const handleDelete = async (id: string) => {
    setSubmitting(true);
    try {
      await Api.monitoring.deleteAlertRule(id);
      message.success('规则删除成功');
      const refreshed = await load(false);
      if (!refreshed) {
        message.warning('规则删除成功，但列表刷新失败');
      }
    } catch {
      message.error('规则删除失败');
    } finally {
      setSubmitting(false);
    }
  };

  const loadBindings = async (ruleId: string): Promise<boolean> => {
    const seq = bindingLoadSeqRef.current + 1;
    bindingLoadSeqRef.current = seq;
    setBindingLoading(true);
    try {
      const res = await Api.monitoring.getRuleChannels(ruleId, { projectId: getProjectId() });
      const list = (res?.data as any)?.list || [];
      if (!mountedRef.current || seq !== bindingLoadSeqRef.current || !bindingOpenRef.current || activeBindingRuleIdRef.current !== ruleId) return false;
      setBindings(
        list.map((item: any) => ({
          channelId: String(item.channel_id ?? item.channelId ?? item.id ?? ''),
          priority: item.priority,
          enabled: item.enabled !== false,
        })),
      );
      return true;
    } catch {
      if (seq === bindingLoadSeqRef.current && bindingOpenRef.current && activeBindingRuleIdRef.current === ruleId) {
        message.error('规则渠道绑定加载失败');
      }
      return false;
    } finally {
      if (mountedRef.current && seq === bindingLoadSeqRef.current) {
        setBindingLoading(false);
      }
    }
  };

  const openBindingDrawer = async (record: EffectiveRuleRow) => {
    bindingOpenRef.current = true;
    activeBindingRuleIdRef.current = record.id;
    setBindingRule(record);
    setBindingOpen(true);
    setEditingBindingChannelId(null);
    bindingForm.setFieldsValue({
      channelId: '',
      priority: 1,
      enabled: true,
    });
    await loadBindings(record.id);
  };

  const handleCreateBinding = async () => {
    if (!bindingRule || bindingSubmitting) return;
    try {
      const values = await bindingForm.validateFields();
      setBindingSubmitting(true);
      try {
        await Api.monitoring.createRuleChannelBinding(bindingRule.id, {
          projectId: getProjectId(),
          channelId: values.channelId,
          priority: values.priority,
          enabled: values.enabled,
        });
        message.success('绑定创建成功');
        setEditingBindingChannelId(null);
        bindingForm.setFieldsValue({ channelId: '', priority: 1, enabled: true });
        await loadBindings(bindingRule.id);
      } catch {
        message.error('绑定创建失败');
      } finally {
        setBindingSubmitting(false);
      }
    } catch (error: any) {
      if (!Array.isArray(error?.errorFields)) {
        message.error('绑定创建失败');
      }
    }
  };

  const handlePrepareUpdateBinding = (record: RuleChannelBindingRow) => {
    if (bindingSubmitting) return;
    setEditingBindingChannelId(record.channelId);
    bindingForm.setFieldsValue({
      channelId: record.channelId,
      priority: record.priority,
      enabled: record.enabled,
    });
  };

  const handleUpdateBinding = async () => {
    if (!bindingRule || !editingBindingChannelId || bindingSubmitting) return;
    try {
      const values = await bindingForm.validateFields();
      setBindingSubmitting(true);
      try {
        await Api.monitoring.updateRuleChannelBinding(bindingRule.id, editingBindingChannelId, {
          projectId: getProjectId(),
          priority: values.priority,
          enabled: values.enabled,
        });
        message.success('绑定更新成功');
        setEditingBindingChannelId(null);
        bindingForm.setFieldsValue({ channelId: '', priority: 1, enabled: true });
        await loadBindings(bindingRule.id);
      } catch {
        message.error('绑定更新失败');
      } finally {
        setBindingSubmitting(false);
      }
    } catch (error: any) {
      if (!Array.isArray(error?.errorFields)) {
        message.error('绑定更新失败');
      }
    }
  };

  const handleDeleteBinding = async (channelId: string) => {
    if (!bindingRule || bindingSubmitting) return;
    setBindingSubmitting(true);
    try {
      await Api.monitoring.deleteRuleChannelBinding(bindingRule.id, channelId, getProjectId());
      message.success('绑定删除成功');
      setEditingBindingChannelId(null);
      bindingForm.setFieldsValue({ channelId: '', priority: 1, enabled: true });
      await loadBindings(bindingRule.id);
    } catch {
      message.error('绑定删除失败');
    } finally {
      setBindingSubmitting(false);
    }
  };

  return (
    <Card
      title="规则配置"
      extra={(
        <Button type="primary" onClick={() => setCreateOpen(true)}>
          新增规则
        </Button>
      )}
    >
      <Table
        rowKey="id"
        loading={loading}
        dataSource={rows}
        pagination={false}
        columns={[
          { title: '名称', dataIndex: 'name' },
          { title: '级别', dataIndex: 'severity' },
          { title: '阈值', dataIndex: 'threshold', render: (v: number | undefined) => (v == null ? '-' : v) },
          { title: '作用域', dataIndex: 'scope', render: (v: string | undefined) => v || '-' },
          { title: '继承键', dataIndex: 'inherit_key', render: (v: string | undefined) => v || '-' },
          {
            title: '操作',
            key: 'actions',
            render: (_value: unknown, record: EffectiveRuleRow) => (
              <Space>
                <Button type="link" onClick={() => handleOpenEdit(record)}>
                  编辑
                </Button>
                <Button type="link" onClick={() => void openBindingDrawer(record)}>
                  渠道绑定
                </Button>
                <Popconfirm title="确定删除此规则？" onConfirm={() => handleDelete(record.id)}>
                  <Button type="link" danger>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />
      <Modal
        title="新增规则"
        open={createOpen}
        onOk={() => void handleCreate()}
        onCancel={() => {
          setCreateOpen(false);
          createForm.resetFields();
        }}
        confirmLoading={submitting}
      >
        <Form
          form={createForm}
          layout="vertical"
          initialValues={{
            name: '',
            metric: '',
            severity: 'warning',
            threshold: 0,
          }}
        >
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="指标" name="metric" rules={[{ required: true, message: '请输入指标' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="级别" name="severity" rules={[{ required: true, message: '请选择级别' }]}>
            <Select
              options={[
                { label: 'critical', value: 'critical' },
                { label: 'warning', value: 'warning' },
                { label: 'info', value: 'info' },
              ]}
            />
          </Form.Item>
          <Form.Item label="阈值" name="threshold" rules={[{ required: true, message: '请输入阈值' }]}>
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title="编辑规则"
        open={!!editing}
        onOk={() => void handleUpdate()}
        onCancel={() => {
          setEditing(null);
          editForm.resetFields();
        }}
        confirmLoading={submitting}
      >
        <Form
          form={editForm}
          layout="vertical"
          initialValues={{
            name: '',
            severity: 'warning',
            threshold: 0,
          }}
        >
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="级别" name="severity" rules={[{ required: true, message: '请选择级别' }]}>
            <Select
              options={[
                { label: 'critical', value: 'critical' },
                { label: 'warning', value: 'warning' },
                { label: 'info', value: 'info' },
              ]}
            />
          </Form.Item>
          <Form.Item label="阈值" name="threshold" rules={[{ required: true, message: '请输入阈值' }]}>
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
      <Drawer
        title="规则渠道绑定"
        open={bindingOpen}
        onClose={() => {
          bindingOpenRef.current = false;
          activeBindingRuleIdRef.current = null;
          bindingLoadSeqRef.current += 1;
          setBindingOpen(false);
          setBindingRule(null);
          setEditingBindingChannelId(null);
          setBindings([]);
          bindingForm.resetFields();
        }}
        size="large"
      >
        <Form
          form={bindingForm}
          layout="vertical"
          initialValues={{
            channelId: '',
            priority: 1,
            enabled: true,
          }}
        >
          <Form.Item label="渠道ID" name="channelId" rules={[{ required: true, message: '请输入渠道ID' }]}>
            <Input disabled={!!editingBindingChannelId || bindingSubmitting} />
          </Form.Item>
          <Form.Item label="优先级" name="priority" rules={[{ required: true, message: '请输入优先级' }]}>
            <InputNumber style={{ width: '100%' }} disabled={bindingSubmitting} />
          </Form.Item>
          <Form.Item label="状态" name="enabled" rules={[{ required: true, message: '请选择状态' }]}>
            <Select
              disabled={bindingSubmitting}
              options={[
                { label: '启用', value: true },
                { label: '禁用', value: false },
              ]}
            />
          </Form.Item>
          <Space style={{ marginBottom: 16 }}>
            {editingBindingChannelId ? (
              <>
                <Button type="primary" onClick={() => void handleUpdateBinding()} loading={bindingSubmitting}>
                  更新绑定
                </Button>
                <Button
                  onClick={() => {
                    setEditingBindingChannelId(null);
                    bindingForm.setFieldsValue({ channelId: '', priority: 1, enabled: true });
                  }}
                  disabled={bindingSubmitting}
                >
                  取消编辑
                </Button>
              </>
            ) : (
              <Button type="primary" onClick={() => void handleCreateBinding()} loading={bindingSubmitting}>
                新增绑定
              </Button>
            )}
          </Space>
        </Form>
        <Table
          rowKey="channelId"
          loading={bindingLoading}
          dataSource={bindings}
          pagination={false}
          columns={[
            { title: '渠道ID', dataIndex: 'channelId' },
            { title: '优先级', dataIndex: 'priority', render: (v: number | undefined) => (v == null ? '-' : v) },
            { title: '状态', dataIndex: 'enabled', render: (v: boolean) => (v ? '启用' : '禁用') },
            {
              title: '操作',
              key: 'actions',
              render: (_value: unknown, record: RuleChannelBindingRow) => (
                <Space>
                  <Button type="link" onClick={() => handlePrepareUpdateBinding(record)} disabled={bindingSubmitting}>
                    编辑绑定
                  </Button>
                  <Popconfirm
                    title="确定删除此绑定？"
                    onConfirm={() => handleDeleteBinding(record.channelId)}
                    disabled={bindingSubmitting}
                    okButtonProps={{ loading: bindingSubmitting, disabled: bindingSubmitting }}
                    cancelButtonProps={{ disabled: bindingSubmitting }}
                  >
                    <Button type="link" danger disabled={bindingSubmitting}>
                      删除绑定
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Drawer>
    </Card>
  );
};

export default RulesConfigPage;
