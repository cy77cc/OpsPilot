import React, { useEffect, useState } from 'react';
import { Button, Card, Drawer, Form, Input, Modal, Select, Space, Table, Tag, Timeline, message } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { Api } from '../../api';
import type { Config, ConfigApp, ConfigVersion } from '../../api/modules/configs';

const ConfigPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [configs, setConfigs] = useState<Config[]>([]);
  const [apps, setApps] = useState<ConfigApp[]>([]);
  const [createOpen, setCreateOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [histories, setHistories] = useState<ConfigVersion[]>([]);
  const [selected, setSelected] = useState<Config | null>(null);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const [configRes, appRes] = await Promise.all([
        Api.configs.getConfigList({ page: 1, pageSize: 100 }),
        Api.configs.getAppList({ page: 1, pageSize: 100 }),
      ]);
      setConfigs(configRes.data.list || []);
      setApps(appRes.data.list || []);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const createConfig = async () => {
    const v = await form.validateFields();
    await Api.configs.createConfig({
      appId: v.app_id,
      env: v.env,
      key: v.key,
      value: v.value,
      description: v.description,
      status: 'active',
    });
    message.success('配置创建成功');
    setCreateOpen(false);
    form.resetFields();
    load();
  };

  const openHistory = async (item: Config) => {
    setSelected(item);
    const res = await Api.configs.getConfigVersions(String(item.id));
    setHistories(res.data.list || []);
    setHistoryOpen(true);
  };

  return (
    <Card
      title="配置中心"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>新建配置</Button>
        </Space>
      }
    >
      <Table
        rowKey="id"
        loading={loading}
        dataSource={configs}
        columns={[
          { title: 'App', dataIndex: 'appId' },
          { title: '环境', dataIndex: 'env', render: (v: string) => <Tag>{v}</Tag> },
          { title: 'Key', dataIndex: 'key' },
          { title: 'Value', dataIndex: 'value', ellipsis: true },
          { title: '版本', dataIndex: 'version' },
          { title: '状态', dataIndex: 'status', render: (v: string) => <Tag color={v === 'active' ? 'success' : 'default'}>{v}</Tag> },
          { title: '更新时间', dataIndex: 'updatedAt', render: (v: string) => (v ? new Date(v).toLocaleString() : '-') },
          { title: '操作', render: (_: unknown, r: Config) => <Button type="link" onClick={() => openHistory(r)}>历史</Button> },
        ]}
      />

      <Modal title="创建配置" open={createOpen} onCancel={() => setCreateOpen(false)} onOk={createConfig}>
        <Form form={form} layout="vertical">
          <Form.Item name="app_id" label="应用" rules={[{ required: true }]}>
            <Select options={apps.map((a) => ({ value: a.appId, label: `${a.name}(${a.appId})` }))} />
          </Form.Item>
          <Form.Item name="env" label="环境" rules={[{ required: true }]}>
            <Select options={[{ value: 'dev' }, { value: 'test' }, { value: 'prod' }]} />
          </Form.Item>
          <Form.Item name="key" label="Key" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="value" label="Value" rules={[{ required: true }]}><Input.TextArea rows={4} /></Form.Item>
          <Form.Item name="description" label="描述"><Input /></Form.Item>
        </Form>
      </Modal>

      <Drawer title={`配置历史 - ${selected?.key || ''}`} open={historyOpen} onClose={() => setHistoryOpen(false)} width={560}>
        <Timeline
          items={histories.map((h) => ({
            children: (
              <div className="flex justify-between">
                <span>v{h.version} {h.action || ''}</span>
                <span className="text-xs text-gray-500">{h.createdAt ? new Date(h.createdAt).toLocaleString() : ''}</span>
              </div>
            ),
          }))}
        />
      </Drawer>
    </Card>
  );
};

export default ConfigPage;
