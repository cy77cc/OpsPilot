import React, { useMemo, useState } from 'react';
import { Card, Table, Tag, Button, Space, Select, Row, Col, Badge, Empty, Tooltip, message } from 'antd';
import { AppstoreOutlined, ExclamationCircleOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { Api } from '../../api';
import type { Config, ConfigApp } from '../../api/modules/configs';

const { Option } = Select;

const envs = ['dev', 'test', 'staging', 'prod'] as const;
type ConfigEnv = typeof envs[number];

const envLabels: Record<ConfigEnv, string> = {
  dev: '开发',
  test: '测试',
  staging: '预发布',
  prod: '生产',
};

const envColors: Record<ConfigEnv, string> = {
  dev: 'default',
  test: 'orange',
  staging: 'purple',
  prod: 'red',
};

interface ConfigRow {
  key: string;
  namespace: string;
  configs: Record<ConfigEnv, Config | null>;
}

const ConfigMultiEnvPage: React.FC = () => {
  const [apps, setApps] = useState<ConfigApp[]>([]);
  const [configs, setConfigs] = useState<Config[]>([]);
  const [selectedAppId, setSelectedAppId] = useState<string>('');
  const [viewMode, setViewMode] = useState<'table' | 'detail'>('table');
  const [loading, setLoading] = useState(false);

  React.useEffect(() => {
    Api.configs.getAppList({ page: 1, pageSize: 200 })
      .then((res) => setApps(res.data.list || []))
      .catch((error) => message.error((error as Error).message || '加载应用失败'));
  }, []);

  const handleAppChange = async (appId: string) => {
    setSelectedAppId(appId);
    setLoading(true);
    try {
      const res = await Api.configs.getConfigList({ page: 1, pageSize: 800, appId });
      setConfigs(res.data.list || []);
    } catch (error) {
      message.error((error as Error).message || '加载配置失败');
      setConfigs([]);
    } finally {
      setLoading(false);
    }
  };

  const configRows = useMemo(() => {
    if (!selectedAppId) return [];

    const keySet = new Set(configs.map((c) => c.key));
    const rows: ConfigRow[] = [];

    keySet.forEach((key) => {
      const rowConfigs: Record<ConfigEnv, Config | null> = {
        dev: null,
        test: null,
        staging: null,
        prod: null,
      };

      let namespace = key.split('.')[0] || 'default';
      envs.forEach((env) => {
        const cfg = configs.find((c) => c.key === key && c.env === env);
        if (cfg) {
          rowConfigs[env] = cfg;
          namespace = cfg.key.split('.')[0] || namespace;
        }
      });

      rows.push({ key, namespace, configs: rowConfigs });
    });

    return rows;
  }, [configs, selectedAppId]);

  const getDiffStatus = (row: ConfigRow): 'same' | 'different' => {
    const values = Object.values(row.configs).filter(Boolean).map((c) => (c as Config).value);
    if (values.length <= 1) return 'same';
    return values.every((v) => v === values[0]) ? 'same' : 'different';
  };

  const columns = [
    {
      title: '配置键',
      dataIndex: 'key',
      key: 'key',
      width: 280,
      render: (key: string, record: ConfigRow) => (
        <Space>
          <code className="text-gray-300 bg-gray-800 px-2 py-1 rounded text-sm">{key}</code>
          {getDiffStatus(record) === 'different' && (
            <Tooltip title="各环境配置值不一致">
              <ExclamationCircleOutlined style={{ color: '#faad14' }} />
            </Tooltip>
          )}
        </Space>
      ),
    },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      key: 'namespace',
      render: (ns: string) => <Tag color="cyan">{ns}</Tag>,
    },
    ...envs.map((env) => ({
      title: <Tag color={envColors[env]}>{envLabels[env]}</Tag>,
      key: env,
      width: 220,
      render: (_: unknown, record: ConfigRow) => {
        const config = record.configs[env];
        if (!config) return <Tag icon={<CloseCircleOutlined />}>未配置</Tag>;

        return (
          <div className="text-xs">
            <Space>
              <Badge status="success" />
              <span className="text-gray-400">v{config.version}</span>
            </Space>
            <pre className="mt-1 text-gray-300 truncate max-w-[200px]" style={{ maxHeight: 44, overflow: 'hidden' }}>
              {config.value}
            </pre>
          </div>
        );
      },
    })),
    {
      title: '状态',
      key: 'status',
      width: 100,
      render: (_: unknown, record: ConfigRow) => (
        getDiffStatus(record) === 'same'
          ? <Tag color="success" icon={<CheckCircleOutlined />}>一致</Tag>
          : <Tag color="warning" icon={<ExclamationCircleOutlined />}>差异</Tag>
      ),
    },
  ];

  return (
    <div className="fade-in">
      <Card
        style={{ background: '#16213e', border: '1px solid #2d3748' }}
        title={
          <Space>
            <AppstoreOutlined style={{ color: '#3498db' }} />
            <span className="text-white text-lg">多环境配置看板</span>
          </Space>
        }
        extra={
          <Space>
            <Select
              placeholder="选择应用"
              value={selectedAppId || undefined}
              onChange={handleAppChange}
              style={{ width: 220 }}
            >
              {apps.map((app) => (
                <Option key={app.id} value={app.appId}>{app.name}</Option>
              ))}
            </Select>
            <Button type={viewMode === 'table' ? 'primary' : 'default'} onClick={() => setViewMode('table')}>
              表格视图
            </Button>
            <Button type={viewMode === 'detail' ? 'primary' : 'default'} onClick={() => setViewMode('detail')}>
              详情视图
            </Button>
          </Space>
        }
      >
        {selectedAppId ? (
          viewMode === 'table' ? (
            <Table
              loading={loading}
              dataSource={configRows}
              columns={columns}
              rowKey="key"
              pagination={{ pageSize: 10 }}
              scroll={{ x: 1200 }}
            />
          ) : (
            <div>
              {configRows.map((row) => (
                <Card
                  key={row.key}
                  size="small"
                  style={{
                    marginBottom: 12,
                    background: '#1a1a2e',
                    border: getDiffStatus(row) === 'different' ? '1px solid #faad14' : '1px solid #2d3748',
                  }}
                >
                  <Row gutter={16} align="middle">
                    <Col span={4}>
                      <Space direction="vertical">
                        <code className="text-gray-300 bg-gray-800 px-2 py-1 rounded text-sm">{row.key}</code>
                        <Tag color="cyan">{row.namespace}</Tag>
                      </Space>
                    </Col>
                    {envs.map((env) => (
                      <Col key={env} span={5}>
                        <div className="bg-gray-900 p-2 rounded">
                          <Space>
                            <Tag color={envColors[env]}>{envLabels[env]}</Tag>
                            {row.configs[env] ? <Badge status="success" /> : <Badge status="error" />}
                          </Space>
                          {row.configs[env] && (
                            <pre className="mt-2 text-green-400 text-xs overflow-hidden" style={{ maxHeight: 72 }}>
                              {row.configs[env]!.value}
                            </pre>
                          )}
                        </div>
                      </Col>
                    ))}
                    <Col span={1}>
                      {getDiffStatus(row) === 'different'
                        ? <Tag color="warning">差异</Tag>
                        : <Tag color="success">一致</Tag>}
                    </Col>
                  </Row>
                </Card>
              ))}
            </div>
          )
        ) : (
          <Empty description="请选择应用查看多环境配置" />
        )}
      </Card>

      {selectedAppId && (
        <Card style={{ background: '#16213e', border: '1px solid #2d3748', marginTop: 16 }} title={<span className="text-white">环境配置统计</span>}>
          <Row gutter={16}>
            {envs.map((env) => {
              const count = configRows.filter((r) => r.configs[env]).length;
              const total = configRows.length;
              const diffCount = configRows.filter((r) => getDiffStatus(r) === 'different' && r.configs[env]).length;
              return (
                <Col key={env} span={6}>
                  <div className="text-center p-4 bg-gray-800 rounded-lg">
                    <Tag color={envColors[env]} className="text-lg px-4 py-2 mb-2">{envLabels[env]}</Tag>
                    <div className="text-2xl font-bold text-white">{count}/{total}</div>
                    <div className="text-gray-400 text-sm">已配置 / 总计</div>
                    {diffCount > 0 && <Tag color="warning" className="mt-2">{diffCount} 项有差异</Tag>}
                  </div>
                </Col>
              );
            })}
          </Row>
        </Card>
      )}
    </div>
  );
};

export default ConfigMultiEnvPage;
