import React, { useEffect, useRef, useState } from 'react';
import { Button, Card, Drawer, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, message } from 'antd';
import { SyncOutlined } from '@ant-design/icons';
import { Api } from '../../api';
import ScopeSelector, { type ScopeValue } from './components/ScopeSelector';
import { GuidedFormItem } from '../../components/FormGuidance';
import { monitorFieldGuides } from '../../constants/fieldGuides';
import { PageSkeleton } from '../../components/LoadingSkeleton';

type EffectiveRuleRow = {
  id: string;
  name: string;
  metric?: string;
  promqlExpr?: string;
  severity: string;
  threshold?: number;
  durationSec?: number;
  labelsJson?: string;
  annotationsJson?: string;
  scope?: string;
  inherit_key?: string;
};

type RuleFormValues = {
  name: string;
  metric?: string;
  promqlExpr?: string;
  severity: string;
  threshold?: number;
  durationSec?: number;
  labelsJson?: string;
  annotationsJson?: string;
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

const readStoredProjectId = (): string | undefined => {
  if (typeof window === 'undefined') return undefined;
  const value = window.localStorage.getItem('projectId');
  return value || undefined;
};

const normalizeProjectId = (value?: string): string | undefined => {
  const trimmed = (value || '').trim();
  return trimmed || undefined;
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
  const [scope, setScope] = useState<ScopeValue>({ scope: 'global', projectId: readStoredProjectId() });
  const mountedRef = useRef(true);
  const loadSeqRef = useRef(0);
  const bindingLoadSeqRef = useRef(0);
  const activeBindingRuleIdRef = useRef<string | null>(null);
  const bindingOpenRef = useRef(false);
  const currentProjectId = scope.scope === 'project' ? normalizeProjectId(scope.projectId) : undefined;

  const ensureProjectScopeReady = (): boolean => {
    if (scope.scope !== 'project') return true;
    if (currentProjectId) return true;
    message.error('项目作用域操作需要先选择项目ID');
    return false;
  };

  const load = async (showError = true): Promise<boolean> => {
    const seq = loadSeqRef.current + 1;
    loadSeqRef.current = seq;
    setLoading(true);
    try {
      const res = await Api.monitoring.getEffectiveRules({ projectId: currentProjectId, page: 1, pageSize: 50 });
      const list = (res?.data as any)?.list || [];
      if (!mountedRef.current || seq !== loadSeqRef.current) return false;
      setRows(
        list.map((item: any) => ({
          id: String(item.id),
          name: item.name || '',
          metric: item.metric || '',
          promqlExpr: item.promql_expr || '',
          severity: item.severity || '',
          threshold: item.threshold,
          durationSec: item.duration_sec,
          labelsJson: item.labels_json,
          annotationsJson: item.annotations_json,
          scope: item.scope,
          inherit_key: item.inherit_key,
        })),
      );
      return true;
    } catch {
      if (!mountedRef.current || seq !== loadSeqRef.current) return false;
      if (showError) {
        message.error('规则列表加载失败');
      }
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
    const projectId = normalizeProjectId(scope.projectId);
    if (projectId) {
      window.localStorage.setItem('projectId', projectId);
    } else {
      window.localStorage.removeItem('projectId');
    }
  }, [scope.projectId]);

  const handleCreate = async () => {
    try {
      const values = await createForm.validateFields();
      setSubmitting(true);
      try {
        await Api.monitoring.createAlertRule({
          name: values.name,
          metric: values.metric || '',
          promql_expr: values.promqlExpr || '',
          severity: values.severity,
          threshold: values.threshold,
          duration_sec: values.durationSec,
          labels_json: values.labelsJson,
          annotations_json: values.annotationsJson,
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
      metric: record.metric,
      promqlExpr: record.promqlExpr,
      severity: record.severity || 'warning',
      threshold: record.threshold,
      durationSec: record.durationSec,
      labelsJson: record.labelsJson,
      annotationsJson: record.annotationsJson,
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
          metric: values.metric || '',
          promql_expr: values.promqlExpr || '',
          severity: values.severity,
          threshold: values.threshold,
          duration_sec: values.durationSec,
          labels_json: values.labelsJson,
          annotations_json: values.annotationsJson,
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
    if (scope.scope === 'project' && !currentProjectId) {
      setBindings([]);
      return false;
    }
    const seq = bindingLoadSeqRef.current + 1;
    bindingLoadSeqRef.current = seq;
    setBindingLoading(true);
    try {
      const res = await Api.monitoring.getRuleChannels(ruleId, { projectId: currentProjectId });
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
    if (!ensureProjectScopeReady()) return;
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
    if (!ensureProjectScopeReady()) return;
    try {
      const values = await bindingForm.validateFields();
      setBindingSubmitting(true);
      try {
        await Api.monitoring.createRuleChannelBinding(bindingRule.id, {
          projectId: currentProjectId,
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
    if (!ensureProjectScopeReady()) return;
    try {
      const values = await bindingForm.validateFields();
      setBindingSubmitting(true);
      try {
        await Api.monitoring.updateRuleChannelBinding(bindingRule.id, editingBindingChannelId, {
          projectId: currentProjectId,
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
    if (!ensureProjectScopeReady()) return;
    setBindingSubmitting(true);
    try {
      await Api.monitoring.deleteRuleChannelBinding(bindingRule.id, channelId, currentProjectId);
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

  const handleSyncRules = async () => {
    try {
      await Api.monitoring.syncAlertRules();
      message.success('规则同步成功');
      void load();
    } catch (error: any) {
      message.error(error?.message || '规则同步失败');
    }
  };

  if (loading && rows.length === 0) {
    return <PageSkeleton />;
  }

  return (
    <Card
      title="告警规则配置"
     
      extra={(
        <Space>
          <ScopeSelector value={scope} onChange={setScope} />
          <Button icon={<SyncOutlined />} onClick={handleSyncRules}>
            同步规则
          </Button>
          <Button type="primary" onClick={() => setCreateOpen(true)}>
            新增规则
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
              promqlExpr: '',
              severity: 'warning',
              threshold: 0,
              durationSec: 300,
              labelsJson: '',
              annotationsJson: '',
            }}
          >
            <GuidedFormItem label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
              <Input />
            </GuidedFormItem>
            <GuidedFormItem label="指标名称 (非必填)" name="metric">
              <Input placeholder="如果不使用自定义 PromQL，可填此项生成默认查询" />
            </GuidedFormItem>
            <GuidedFormItem label="PromQL 表达式" name="promqlExpr" guide={monitorFieldGuides.promqlExpr}>
              <Input.TextArea rows={2} placeholder='例如: job:request_latency_seconds:mean5m{job="myjob"} > 0.5' />
            </GuidedFormItem>
            <Form.Item label="级别" name="severity" rules={[{ required: true, message: '请选择级别' }]}>
              <Select
                options={[
                  { label: 'critical', value: 'critical' },
                  { label: 'warning', value: 'warning' },
                  { label: 'info', value: 'info' },
                ]}
              />
            </Form.Item>
            <GuidedFormItem label="阈值 (仅默认查询时使用)" name="threshold">
              <InputNumber style={{ width: '100%' }} />
            </GuidedFormItem>
            <GuidedFormItem label="持续时间 (For)" name="durationSec" rules={[{ required: true, message: '请输入持续时间 (秒)' }]} guide={monitorFieldGuides.durationSec}>
              <InputNumber style={{ width: '100%' }} min={0} addonAfter="秒" />
            </GuidedFormItem>
            <GuidedFormItem label="附加标签 (Labels JSON)" name="labelsJson" guide={monitorFieldGuides.labelsJson}>
              <Input.TextArea rows={2} placeholder='例如: {"team": "frontend"}' />
            </GuidedFormItem>
            <GuidedFormItem label="详情注解 (Annotations JSON)" name="annotationsJson" guide={monitorFieldGuides.annotationsJson}>
              <Input.TextArea rows={2} placeholder='例如: {"summary": "服务响应延迟"}' />
            </GuidedFormItem>
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
              metric: '',
              promqlExpr: '',
              severity: 'warning',
              threshold: 0,
              durationSec: 300,
              labelsJson: '',
              annotationsJson: '',
            }}
          >
            <GuidedFormItem label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
              <Input />
            </GuidedFormItem>
            <GuidedFormItem label="指标名称 (非必填)" name="metric">
              <Input placeholder="如果不使用自定义 PromQL，可填此项生成默认查询" />
            </GuidedFormItem>
            <GuidedFormItem label="PromQL 表达式" name="promqlExpr" guide={monitorFieldGuides.promqlExpr}>
              <Input.TextArea rows={2} placeholder='例如: job:request_latency_seconds:mean5m{job="myjob"} > 0.5' />
            </GuidedFormItem>
            <Form.Item label="级别" name="severity" rules={[{ required: true, message: '请选择级别' }]}>
              <Select
                options={[
                  { label: 'critical', value: 'critical' },
                  { label: 'warning', value: 'warning' },
                  { label: 'info', value: 'info' },
                ]}
              />
            </Form.Item>
            <GuidedFormItem label="阈值 (仅默认查询时使用)" name="threshold">
              <InputNumber style={{ width: '100%' }} />
            </GuidedFormItem>
            <GuidedFormItem label="持续时间 (For)" name="durationSec" rules={[{ required: true, message: '请输入持续时间 (秒)' }]} guide={monitorFieldGuides.durationSec}>
              <InputNumber style={{ width: '100%' }} min={0} addonAfter="秒" />
            </GuidedFormItem>
            <GuidedFormItem label="附加标签 (Labels JSON)" name="labelsJson" guide={monitorFieldGuides.labelsJson}>
              <Input.TextArea rows={2} placeholder='例如: {"team": "frontend"}' />
            </GuidedFormItem>
            <GuidedFormItem label="详情注解 (Annotations JSON)" name="annotationsJson" guide={monitorFieldGuides.annotationsJson}>
              <Input.TextArea rows={2} placeholder='例如: {"summary": "服务响应延迟"}' />
            </GuidedFormItem>
          </Form>
        </Modal>        <Drawer
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
            <GuidedFormItem label="渠道ID" name="channelId" rules={[{ required: true, message: '请输入渠道ID' }]}>
              <Input disabled={!!editingBindingChannelId || bindingSubmitting} />
            </GuidedFormItem>
            <GuidedFormItem label="优先级" name="priority" rules={[{ required: true, message: '请输入优先级' }]}>
              <InputNumber style={{ width: '100%' }} disabled={bindingSubmitting} />
            </GuidedFormItem>
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
