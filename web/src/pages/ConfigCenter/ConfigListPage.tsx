import React, { useEffect, useMemo, useState } from 'react';
import { Card, Table, Tag, Button, Space, Input, Select, Modal, Form, Drawer, message, Popconfirm, Empty, Timeline } from 'antd';
import { SearchOutlined, PlusOutlined, EditOutlined, DeleteOutlined, HistoryOutlined, FileTextOutlined } from '@ant-design/icons';
import { useSearchParams } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import { Api } from '../../api';
import type { Config, ConfigApp, ConfigVersion } from '../../api/modules/configs';

const { Option } = Select;

const envColors: Record<string, string> = {
  dev: 'default',
  test: 'orange',
  staging: 'purple',
  prod: 'red',
};

const ConfigListPage: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialAppId = searchParams.get('appId') || '';

  const [loading, setLoading] = useState(false);
  const [apps, setApps] = useState<ConfigApp[]>([]);
  const [configs, setConfigs] = useState<Config[]>([]);
  const [selectedAppId, setSelectedAppId] = useState<string>(initialAppId);
  const [envFilter, setEnvFilter] = useState<string>('all');
  const [searchText, setSearchText] = useState('');

  const [isEditorVisible, setIsEditorVisible] = useState(false);
  const [selectedConfig, setSelectedConfig] = useState<Config | null>(null);
  const [editorValue, setEditorValue] = useState('');
  const [historyOpen, setHistoryOpen] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [history, setHistory] = useState<ConfigVersion[]>([]);

  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const [appRes, configRes] = await Promise.all([
        Api.configs.getAppList({ page: 1, pageSize: 200 }),
        Api.configs.getConfigList({ page: 1, pageSize: 500, appId: selectedAppId || undefined }),
      ]);
      setApps(appRes.data.list || []);
      setConfigs(configRes.data.list || []);
    } catch (error) {
      message.error((error as Error).message || '加载配置失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [selectedAppId]);

  const filteredConfigs = useMemo(() => {
    return configs.filter((config) => {
      if (selectedAppId && config.appId !== selectedAppId) return false;
      if (envFilter !== 'all' && config.env !== envFilter) return false;
      if (!searchText) return true;
      const search = searchText.toLowerCase();
      return config.key.toLowerCase().includes(search) || (config.description || '').toLowerCase().includes(search);
    });
  }, [configs, selectedAppId, envFilter, searchText]);

  const namespaces = useMemo(() => {
    const values = new Set<string>();
    filteredConfigs.forEach((item) => values.add(item.key.split('.')[0] || 'default'));
    return Array.from(values);
  }, [filteredConfigs]);

  const handleAppChange = (appId: string) => {
    setSelectedAppId(appId);
    setSearchParams(appId ? { appId } : {});
  };

  const handleAdd = () => {
    if (!selectedAppId) {
      message.warning('请先选择应用');
      return;
    }
    setSelectedConfig(null);
    form.resetFields();
    form.setFieldsValue({ env: 'dev', status: 'active' });
    setEditorValue('');
    setIsEditorVisible(true);
  };

  const handleEdit = (config: Config) => {
    setSelectedConfig(config);
    form.setFieldsValue({
      key: config.key,
      env: config.env,
      status: config.status,
      description: config.description,
    });
    setEditorValue(config.value);
    setIsEditorVisible(true);
  };

  const handleSubmit = async () => {
    const values = await form.validateFields();
    if (!editorValue.trim()) {
      message.warning('配置值不能为空');
      return;
    }

    const payload = {
      appId: selectedAppId,
      key: values.key,
      env: values.env,
      status: values.status,
      description: values.description,
      value: editorValue,
    };

    if (selectedConfig) {
      await Api.configs.updateConfig(selectedConfig.id, payload);
      message.success('配置更新成功');
    } else {
      await Api.configs.createConfig(payload);
      message.success('配置创建成功');
    }

    setIsEditorVisible(false);
    await load();
  };

  const handleDelete = async (id: string) => {
    await Api.configs.deleteConfig(id);
    message.success('配置删除成功');
    await load();
  };

  const openHistory = async (config: Config) => {
    setSelectedConfig(config);
    setHistoryOpen(true);
    setHistoryLoading(true);
    try {
      const res = await Api.configs.getConfigVersions(config.id, { page: 1, pageSize: 100 });
      setHistory(res.data.list || []);
    } catch (error) {
      message.error((error as Error).message || '加载历史失败');
    } finally {
      setHistoryLoading(false);
    }
  };

  const columns = [
    {
      title: '配置键',
      dataIndex: 'key',
      key: 'key',
      width: 300,
      render: (key: string) => <code className="text-gray-300 bg-gray-800 px-2 py-1 rounded text-sm">{key}</code>,
    },
    {
      title: '命名空间',
      key: 'namespace',
      render: (_: unknown, record: Config) => <Tag color="cyan">{record.key.split('.')[0] || 'default'}</Tag>,
    },
    {
      title: '环境',
      dataIndex: 'env',
      key: 'env',
      render: (env: string) => <Tag color={envColors[env] || 'default'}>{env.toUpperCase()}</Tag>,
    },
    {
      title: '版本',
      dataIndex: 'version',
      key: 'version',
      render: (v: number) => <Tag color="blue">v{v}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (v: string) => <Tag color={v === 'active' ? 'success' : 'default'}>{v}</Tag>,
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      render: (time: string) => (time ? new Date(time).toLocaleString() : '-'),
    },
    {
      title: '操作',
      key: 'action',
      width: 220,
      render: (_: unknown, record: Config) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} size="small" onClick={() => handleEdit(record)}>编辑</Button>
          <Button type="link" icon={<HistoryOutlined />} size="small" onClick={() => openHistory(record)}>历史</Button>
          <Popconfirm title="确定删除此配置？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
            <Button type="link" danger icon={<DeleteOutlined />} size="small">删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="fade-in">
      <Card
        style={{ background: '#16213e', border: '1px solid #2d3748' }}
        title={
          <Space>
            <FileTextOutlined style={{ color: '#3498db' }} />
            <span className="text-white text-lg">配置列表</span>
          </Space>
        }
        extra={
          <Space wrap>
            <Select placeholder="选择应用" value={selectedAppId || undefined} onChange={handleAppChange} style={{ width: 180 }} allowClear>
              {apps.map((app) => (
                <Option key={app.id} value={app.appId}>{app.name}</Option>
              ))}
            </Select>
            <Select value={envFilter} onChange={setEnvFilter} style={{ width: 120 }}>
              <Option value="all">全部环境</Option>
              <Option value="dev">开发</Option>
              <Option value="test">测试</Option>
              <Option value="staging">预发布</Option>
              <Option value="prod">生产</Option>
            </Select>
            <Input
              placeholder="搜索配置键/描述"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 180 }}
              allowClear
            />
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>新建配置</Button>
          </Space>
        }
      >
        {selectedAppId ? (
          <Table
            loading={loading}
            dataSource={filteredConfigs}
            columns={columns}
            rowKey="id"
            pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total) => `共 ${total} 项配置` }}
          />
        ) : (
          <Empty description="请先选择一个应用" />
        )}
      </Card>

      <Modal title={selectedConfig ? '编辑配置' : '新建配置'} open={isEditorVisible} onOk={handleSubmit} onCancel={() => setIsEditorVisible(false)} width={900} okText="保存">
        <Form form={form} layout="vertical">
          <Space style={{ width: '100%' }} styles={{ item: { flex: 1 } }}>
            <Form.Item label="配置键" name="key" rules={[{ required: true, message: '请输入配置键' }]}>
              <Input placeholder="例如: redis.pool.size" disabled={!!selectedConfig} />
            </Form.Item>
            <Form.Item label="环境" name="env" rules={[{ required: true }]}>
              <Select>
                <Option value="dev">开发</Option>
                <Option value="test">测试</Option>
                <Option value="staging">预发布</Option>
                <Option value="prod">生产</Option>
              </Select>
            </Form.Item>
            <Form.Item label="状态" name="status" rules={[{ required: true }]}>
              <Select>
                <Option value="active">active</Option>
                <Option value="inactive">inactive</Option>
              </Select>
            </Form.Item>
          </Space>
          <Form.Item label="描述" name="description">
            <Input placeholder="用于说明配置用途" />
          </Form.Item>
          <Form.Item label="配置值" required>
            <div style={{ border: '1px solid #3d4f6f', borderRadius: 6, overflow: 'hidden' }}>
              <Editor
                height="300px"
                language="plaintext"
                value={editorValue}
                onChange={(value) => setEditorValue(value || '')}
                theme="vs-dark"
                options={{
                  minimap: { enabled: false },
                  fontSize: 13,
                  lineNumbers: 'on',
                  scrollBeyondLastLine: false,
                  automaticLayout: true,
                }}
              />
            </div>
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={`配置历史 - ${selectedConfig?.key || ''}`}
        placement="right"
        width={620}
        open={historyOpen}
        onClose={() => setHistoryOpen(false)}
      >
        {historyLoading ? (
          <div>加载中...</div>
        ) : (
          <Timeline
            items={history
              .slice()
              .sort((a, b) => b.version - a.version)
              .map((item) => ({
                color: item.action === 'delete' ? 'red' : item.action === 'create' ? 'green' : 'blue',
                children: (
                  <div>
                    <Space>
                      <Tag color="blue">v{item.version}</Tag>
                      <Tag>{item.action}</Tag>
                      <span>{new Date(item.createdAt).toLocaleString()}</span>
                      <span>by {item.operator}</span>
                    </Space>
                    <pre className="bg-gray-900 p-3 rounded text-green-400 text-sm overflow-x-auto mt-2">{item.value}</pre>
                  </div>
                ),
              }))}
          />
        )}
        {!historyLoading && history.length === 0 && <Empty description="暂无历史记录" />}
        {!historyLoading && namespaces.length > 0 && (
          <div className="mt-4 text-gray-400 text-sm">命名空间候选: {namespaces.join(', ')}</div>
        )}
      </Drawer>
    </div>
  );
};

export default ConfigListPage;
