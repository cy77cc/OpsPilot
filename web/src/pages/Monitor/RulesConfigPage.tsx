import React, { useEffect, useRef, useState } from 'react';
import { Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, message } from 'antd';
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

const RulesConfigPage: React.FC = () => {
  const [createForm] = Form.useForm<RuleFormValues>();
  const [editForm] = Form.useForm<RuleFormValues>();
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [rows, setRows] = useState<EffectiveRuleRow[]>([]);
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<EffectiveRuleRow | null>(null);
  const mountedRef = useRef(true);

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
    </Card>
  );
};

export default RulesConfigPage;
