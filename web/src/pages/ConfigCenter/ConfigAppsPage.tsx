import React, { useEffect, useMemo, useState } from 'react';
import { Card, Table, Tag, Button, Space, Input, Modal, Form, message } from 'antd';
import { SearchOutlined, PlusOutlined, AppstoreOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../api';
import type { Config, ConfigApp } from '../../api/modules/configs';

const ConfigAppsPage: React.FC = () => {
  const navigate = useNavigate();
  const [apps, setApps] = useState<ConfigApp[]>([]);
  const [configs, setConfigs] = useState<Config[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const [appRes, configRes] = await Promise.all([
        Api.configs.getAppList({ page: 1, pageSize: 200 }),
        Api.configs.getConfigList({ page: 1, pageSize: 500 }),
      ]);
      setApps(appRes.data.list || []);
      setConfigs(configRes.data.list || []);
    } catch (error) {
      message.error((error as Error).message || '加载应用失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const filteredApps = useMemo(
    () => apps.filter((app) =>
      app.name.toLowerCase().includes(searchText.toLowerCase()) ||
      (app.description || '').toLowerCase().includes(searchText.toLowerCase()) ||
      app.appId.toLowerCase().includes(searchText.toLowerCase())
    ),
    [apps, searchText]
  );

  const getConfigCount = (appId: string) => configs.filter((item) => item.appId === appId).length;

  const getNamespaces = (appId: string) => {
    const set = new Set<string>();
    configs
      .filter((item) => item.appId === appId)
      .forEach((item) => set.add(item.key.split('.')[0] || 'default'));
    return Array.from(set).slice(0, 3);
  };

  const handleSubmit = async () => {
    const values = await form.validateFields();
    await Api.configs.createApp({
      app_id: values.appId,
      name: values.name,
      description: values.description,
    });
    message.success('应用创建成功');
    setIsModalVisible(false);
    form.resetFields();
    await load();
  };

  const columns = [
    {
      title: '应用名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: ConfigApp) => (
        <Space>
          <AppstoreOutlined style={{ color: '#3498db', fontSize: 18 }} />
          <a onClick={() => navigate(`/configcenter/list?appId=${record.appId}`)}>{text}</a>
          <Tag>{record.appId}</Tag>
        </Space>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      render: (text: string) => <span>{text || '-'}</span>,
    },
    {
      title: '命名空间',
      key: 'namespaces',
      render: (_: unknown, record: ConfigApp) => {
        const namespaces = getNamespaces(record.appId);
        if (namespaces.length === 0) {
          return <Tag>暂无</Tag>;
        }
        return (
          <Space wrap>
            {namespaces.map((ns) => (
              <Tag key={ns} color="blue">{ns}</Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: '配置数量',
      key: 'configCount',
      render: (_: unknown, record: ConfigApp) => <Tag color="green">{getConfigCount(record.appId)}</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (time: string) => (time ? new Date(time).toLocaleString() : '-'),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: ConfigApp) => (
        <Button type="link" onClick={() => navigate(`/configcenter/list?appId=${record.appId}`)}>
          进入配置
        </Button>
      ),
    },
  ];

  return (
    <div className="fade-in">
      <Card
        style={{ background: '#16213e', border: '1px solid #2d3748' }}
        title={<span className="text-white text-lg">应用管理</span>}
        extra={
          <Space>
            <Input
              placeholder="搜索应用名称/标识/描述"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 240 }}
              allowClear
            />
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setIsModalVisible(true)}>
              新建应用
            </Button>
          </Space>
        }
      >
        <Table
          loading={loading}
          dataSource={filteredApps}
          columns={columns}
          rowKey="id"
          pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total) => `共 ${total} 个应用` }}
        />
      </Card>

      <Modal
        title="新建应用"
        open={isModalVisible}
        onOk={handleSubmit}
        onCancel={() => setIsModalVisible(false)}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item label="应用标识 (app_id)" name="appId" rules={[{ required: true, message: '请输入应用标识' }]}>
            <Input placeholder="例如: user-service" />
          </Form.Item>
          <Form.Item label="应用名称" name="name" rules={[{ required: true, message: '请输入应用名称' }]}>
            <Input placeholder="例如: User Service" />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} placeholder="请输入应用描述" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ConfigAppsPage;
