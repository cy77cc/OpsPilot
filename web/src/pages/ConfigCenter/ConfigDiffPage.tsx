import React, { useMemo, useState } from 'react';
import { Card, Table, Tag, Button, Space, Select, Modal, message, Popconfirm, Descriptions, Empty } from 'antd';
import { HistoryOutlined, RollbackOutlined, DiffOutlined } from '@ant-design/icons';
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

const ConfigDiffPage: React.FC = () => {
  const [apps, setApps] = useState<ConfigApp[]>([]);
  const [configs, setConfigs] = useState<Config[]>([]);
  const [selectedAppId, setSelectedAppId] = useState<string>('');
  const [selectedConfigId, setSelectedConfigId] = useState<string>('');

  const [historyOpen, setHistoryOpen] = useState(false);
  const [diffOpen, setDiffOpen] = useState(false);
  const [history, setHistory] = useState<ConfigVersion[]>([]);
  const [selectedVersion, setSelectedVersion] = useState<ConfigVersion | null>(null);
  const [loading, setLoading] = useState(false);

  const loadApps = async () => {
    const appRes = await Api.configs.getAppList({ page: 1, pageSize: 200 });
    setApps(appRes.data.list || []);
  };

  const loadConfigs = async (appId: string) => {
    const configRes = await Api.configs.getConfigList({ page: 1, pageSize: 500, appId });
    setConfigs(configRes.data.list || []);
  };

  React.useEffect(() => {
    loadApps().catch((error) => message.error((error as Error).message));
  }, []);

  const selectedConfig = useMemo(
    () => configs.find((item) => item.id === selectedConfigId) || null,
    [configs, selectedConfigId]
  );

  const handleAppChange = async (appId: string) => {
    setSelectedAppId(appId);
    setSelectedConfigId('');
    setHistory([]);
    setSelectedVersion(null);
    await loadConfigs(appId);
  };

  const openHistory = async () => {
    if (!selectedConfig) return;
    setLoading(true);
    try {
      const res = await Api.configs.getConfigVersions(selectedConfig.id, { page: 1, pageSize: 100 });
      setHistory(res.data.list || []);
      setHistoryOpen(true);
    } catch (error) {
      message.error((error as Error).message || '加载历史失败');
    } finally {
      setLoading(false);
    }
  };

  const handleRollback = async (version: ConfigVersion) => {
    if (!selectedConfig) return;
    await Api.configs.updateConfig(selectedConfig.id, {
      appId: selectedConfig.appId,
      key: selectedConfig.key,
      env: selectedConfig.env,
      status: selectedConfig.status,
      description: `rollback from v${version.version}`,
      value: version.value,
    });
    message.success(`已回滚到 v${version.version}`);
    const refreshed = await Api.configs.getConfigDetail(selectedConfig.id);
    setConfigs((prev) => prev.map((item) => (item.id === selectedConfig.id ? refreshed.data : item)));
    setHistoryOpen(false);
  };

  const columns = [
    {
      title: '版本',
      key: 'version',
      render: (_: unknown, record: ConfigVersion) => <Tag color="blue">v{record.version}</Tag>,
    },
    {
      title: '动作',
      dataIndex: 'action',
      key: 'action',
      render: (action: string) => <Tag>{action}</Tag>,
    },
    {
      title: '环境',
      dataIndex: 'env',
      key: 'env',
      render: (env: string) => <Tag color={envColors[env] || 'default'}>{env.toUpperCase()}</Tag>,
    },
    {
      title: '操作人',
      dataIndex: 'operator',
      key: 'operator',
    },
    {
      title: '时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (time: string) => (time ? new Date(time).toLocaleString() : '-'),
    },
    {
      title: '操作',
      key: 'actionColumn',
      render: (_: unknown, record: ConfigVersion) => (
        <Space>
          <Button type="link" icon={<DiffOutlined />} size="small" onClick={() => { setSelectedVersion(record); setDiffOpen(true); }}>
            对比
          </Button>
          <Popconfirm title="确定回滚到此版本？" onConfirm={() => handleRollback(record)} okText="确定" cancelText="取消">
            <Button type="link" icon={<RollbackOutlined />} size="small">回滚</Button>
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
            <DiffOutlined style={{ color: '#3498db' }} />
            <span className="text-white text-lg">Diff 对比与回滚</span>
          </Space>
        }
        extra={
          <Space>
            <Select placeholder="选择应用" value={selectedAppId || undefined} onChange={handleAppChange} style={{ width: 180 }}>
              {apps.map((app) => (
                <Option key={app.id} value={app.appId}>{app.name}</Option>
              ))}
            </Select>
            <Select
              placeholder="选择配置"
              value={selectedConfigId || undefined}
              onChange={setSelectedConfigId}
              style={{ width: 260 }}
              disabled={!selectedAppId}
              showSearch
              optionFilterProp="children"
            >
              {configs.map((cfg) => (
                <Option key={cfg.id} value={cfg.id}>{cfg.key} ({cfg.env})</Option>
              ))}
            </Select>
            <Button icon={<HistoryOutlined />} onClick={openHistory} disabled={!selectedConfig} loading={loading}>
              版本历史
            </Button>
          </Space>
        }
      >
        {selectedConfig ? (
          <Descriptions bordered size="small" style={{ background: '#1a1a2e' }}>
            <Descriptions.Item label="配置键">{selectedConfig.key}</Descriptions.Item>
            <Descriptions.Item label="环境">
              <Tag color={envColors[selectedConfig.env] || 'default'}>{selectedConfig.env.toUpperCase()}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="版本">
              <Tag color="blue">v{selectedConfig.version}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={selectedConfig.status === 'active' ? 'success' : 'default'}>{selectedConfig.status}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="更新时间" span={2}>
              {selectedConfig.updatedAt ? new Date(selectedConfig.updatedAt).toLocaleString() : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="当前值" span={3}>
              <pre className="bg-gray-900 p-3 rounded text-green-400 text-sm overflow-x-auto max-h-56">{selectedConfig.value}</pre>
            </Descriptions.Item>
          </Descriptions>
        ) : (
          <Empty description="请选择应用和配置项" />
        )}
      </Card>

      <Modal title={`变更历史 - ${selectedConfig?.key || ''}`} open={historyOpen} onCancel={() => setHistoryOpen(false)} footer={null} width={980}>
        <Table dataSource={history.slice().sort((a, b) => b.version - a.version)} columns={columns} rowKey="id" pagination={{ pageSize: 8 }} />
      </Modal>

      <Modal title="配置对比" open={diffOpen} onCancel={() => setDiffOpen(false)} width={1200} footer={null}>
        {selectedConfig && selectedVersion ? (
          <div className="grid grid-cols-2 gap-4">
            <div>
              <h4 className="text-white mb-2">当前版本 v{selectedConfig.version}</h4>
              <div style={{ border: '1px solid #3d4f6f', borderRadius: 6, overflow: 'hidden' }}>
                <Editor
                  height="360px"
                  language="plaintext"
                  value={selectedConfig.value}
                  theme="vs-dark"
                  options={{ readOnly: true, minimap: { enabled: false }, fontSize: 13 }}
                />
              </div>
            </div>
            <div>
              <h4 className="text-white mb-2">历史版本 v{selectedVersion.version}</h4>
              <div style={{ border: '1px solid #3d4f6f', borderRadius: 6, overflow: 'hidden' }}>
                <Editor
                  height="360px"
                  language="plaintext"
                  value={selectedVersion.value}
                  theme="vs-dark"
                  options={{ readOnly: true, minimap: { enabled: false }, fontSize: 13 }}
                />
              </div>
            </div>
          </div>
        ) : null}
      </Modal>
    </div>
  );
};

export default ConfigDiffPage;
