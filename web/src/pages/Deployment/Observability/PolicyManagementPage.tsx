import React, { useState, useEffect } from 'react';
import { Card, Button, Space, Table, Modal, Form, Input, Select, message, Tabs, Switch, Tag } from 'antd';
import { PlusOutlined, ReloadOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { Api } from '../../../api';
import type { Policy } from '../../../api/modules/deployment';

const PolicyManagementPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingPolicy, setEditingPolicy] = useState<Policy | null>(null);
  const [form] = Form.useForm();
  const [typeFilter, setTypeFilter] = useState<string>('');

  const load = async () => {
    setLoading(true);
    try {
      const res = await Api.deployment.getPolicies({
        type: typeFilter || undefined,
      });
      setPolicies(res.data.list || []);
    } catch (err) {
      message.error('加载策略失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [typeFilter]);

  const handleCreate = () => {
    setEditingPolicy(null);
    form.resetFields();
    form.setFieldsValue({ enabled: true });
    setModalVisible(true);
  };

  const handleEdit = (policy: Policy) => {
    setEditingPolicy(policy);
    form.setFieldsValue({
      name: policy.name,
      type: policy.type,
      target_id: policy.target_id,
      config: JSON.stringify(policy.config, null, 2),
      enabled: policy.enabled,
    });
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除此策略吗？',
      okText: '删除',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: async () => {
        try {
          await Api.deployment.deletePolicy(id);
          message.success('策略已删除');
          load();
        } catch (err) {
          message.error('删除失败');
        }
      },
    });
  };

  const handleToggleEnabled = async (policy: Policy) => {
    try {
      await Api.deployment.updatePolicy(policy.id, { enabled: !policy.enabled });
      message.success(`策略已${policy.enabled ? '禁用' : '启用'}`);
      load();
    } catch (err) {
      message.error('操作失败');
    }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      let config = {};
      if (values.config) {
        try {
          config = JSON.parse(values.config);
        } catch {
          message.error('配置 JSON 格式错误');
          return;
        }
      }

      if (editingPolicy) {
        await Api.deployment.updatePolicy(editingPolicy.id, {
          name: values.name,
          type: values.type,
          config,
          enabled: values.enabled,
        });
        message.success('策略已更新');
      } else {
        await Api.deployment.createPolicy({
          name: values.name,
          type: values.type,
          target_id: values.target_id,
          config,
          enabled: values.enabled,
        });
        message.success('策略已创建');
      }

      setModalVisible(false);
      load();
    } catch (err) {
      message.error(editingPolicy ? '更新失败' : '创建失败');
    }
  };

  const getTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
      traffic: '流量策略',
      resilience: '弹性策略',
      access: '访问策略',
      slo: 'SLO 策略',
    };
    return labels[type] || type;
  };

  const getTypeColor = (type: string) => {
    const colors: Record<string, string> = {
      traffic: 'blue',
      resilience: 'green',
      access: 'orange',
      slo: 'purple',
    };
    return colors[type] || 'default';
  };

  const columns: ColumnsType<Policy> = [
    {
      title: '策略名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '策略类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => <Tag color={getTypeColor(type)}>{getTypeLabel(type)}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean, record: Policy) => (
        <Switch checked={enabled} onChange={() => handleToggleEnabled(record)} />
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time: string) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: Policy) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDelete(record.id)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  const groupedPolicies = policies.reduce((acc, policy) => {
    const key = policy.type;
    if (!acc[key]) {
      acc[key] = [];
    }
    acc[key].push(policy);
    return acc;
  }, {} as Record<string, Policy[]>);

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">策略管理</h1>
          <p className="text-sm text-gray-500 mt-1">配置服务的流量、弹性、访问和 SLO 策略</p>
        </div>
        <Space>
          <Select
            value={typeFilter}
            style={{ width: 140 }}
            allowClear
            placeholder="策略类型"
            options={[
              { value: '', label: '全部类型' },
              { value: 'traffic', label: '流量策略' },
              { value: 'resilience', label: '弹性策略' },
              { value: 'access', label: '访问策略' },
              { value: 'slo', label: 'SLO 策略' },
            ]}
            onChange={setTypeFilter}
          />
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            创建策略
          </Button>
        </Space>
      </div>

      {/* Policies grouped by type */}
      <Card>
        {Object.keys(groupedPolicies).length > 0 ? (
          <Tabs
            items={Object.entries(groupedPolicies).map(([type, typePolicies]) => ({
              key: type,
              label: (
                <span>
                  <Tag color={getTypeColor(type)}>{getTypeLabel(type)}</Tag>
                  <span className="text-gray-500">({typePolicies.length})</span>
                </span>
              ),
              children: (
                <Table
                  dataSource={typePolicies}
                  columns={columns}
                  rowKey="id"
                  pagination={false}
                />
              ),
            }))}
          />
        ) : (
          <div className="text-center text-gray-500 py-8">暂无策略数据</div>
        )}
      </Card>

      {/* Create/Edit modal */}
      <Modal
        title={editingPolicy ? '编辑策略' : '创建策略'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => {
          setModalVisible(false);
          setEditingPolicy(null);
        }}
        width={720}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="策略名称"
            rules={[{ required: true, message: '请输入策略名称' }]}
          >
            <Input placeholder="例如: Rate Limiting" />
          </Form.Item>
          <Form.Item
            name="type"
            label="策略类型"
            rules={[{ required: true, message: '请选择策略类型' }]}
          >
            <Select
              placeholder="选择策略类型"
              options={[
                { value: 'traffic', label: '流量策略' },
                { value: 'resilience', label: '弹性策略' },
                { value: 'access', label: '访问策略' },
                { value: 'slo', label: 'SLO 策略' },
              ]}
            />
          </Form.Item>
          <Form.Item name="target_id" label="关联部署目标">
            <Input type="number" placeholder="部署目标 ID（可选）" />
          </Form.Item>
          <Form.Item name="config" label="配置 (JSON)">
            <Input.TextArea
              rows={10}
              placeholder={'{\n  "rate": 1000,\n  "per": "minute"\n}'}
            />
          </Form.Item>
          <Form.Item name="enabled" label="启用状态" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default PolicyManagementPage;
