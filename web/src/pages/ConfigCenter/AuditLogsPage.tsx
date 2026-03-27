import React, { useMemo, useState } from 'react';
import { Card, Table, Tag, Button, Space, Select, DatePicker, Input, Row, Col, Timeline, Drawer, message, Empty } from 'antd';
import { HistoryOutlined, SearchOutlined, EyeOutlined, EditOutlined, DeleteOutlined, SyncOutlined, RollbackOutlined, PlusOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { Api } from '../../api';
import type { Config, ConfigApp, ConfigVersion } from '../../api/modules/configs';

const { Option } = Select;
const { RangePicker } = DatePicker;

type AuditAction = 'create' | 'update' | 'delete' | 'release' | 'rollback';

interface AuditLogItem {
  id: string;
  appId: string;
  appName: string;
  key: string;
  env: string;
  action: AuditAction;
  operator: string;
  timestamp: string;
  details: string;
  value: string;
}

const actionIcons: Record<AuditAction, React.ReactNode> = {
  create: <PlusOutlined style={{ color: '#52c41a' }} />,
  update: <EditOutlined style={{ color: '#1890ff' }} />,
  delete: <DeleteOutlined style={{ color: '#ff4d4f' }} />,
  release: <SyncOutlined style={{ color: '#722ed1' }} />,
  rollback: <RollbackOutlined style={{ color: '#faad14' }} />,
};

const actionColors: Record<AuditAction, string> = {
  create: 'green',
  update: 'blue',
  delete: 'red',
  release: 'purple',
  rollback: 'orange',
};

const actionLabels: Record<AuditAction, string> = {
  create: '创建',
  update: '更新',
  delete: '删除',
  release: '发布',
  rollback: '回滚',
};

const normalizeAction = (action: string): AuditAction => {
  if (action === 'create' || action === 'update' || action === 'delete' || action === 'release' || action === 'rollback') {
    return action;
  }
  return 'update';
};

const AuditLogsPage: React.FC = () => {
  const [apps, setApps] = useState<ConfigApp[]>([]);
  const [logs, setLogs] = useState<AuditLogItem[]>([]);
  const [loading, setLoading] = useState(false);

  const [selectedAppId, setSelectedAppId] = useState<string>('all');
  const [actionFilter, setActionFilter] = useState<string>('all');
  const [dateRange, setDateRange] = useState<[Dayjs | null, Dayjs | null] | null>(null);
  const [searchText, setSearchText] = useState('');

  const [isDetailVisible, setIsDetailVisible] = useState(false);
  const [selectedLog, setSelectedLog] = useState<AuditLogItem | null>(null);

  const load = async (appId?: string) => {
    setLoading(true);
    try {
      const appRes = await Api.configs.getAppList({ page: 1, pageSize: 200 });
      const appList = appRes.data.list || [];
      setApps(appList);

      const configRes = await Api.configs.getConfigList({ page: 1, pageSize: 500, appId });
      const configList = configRes.data.list || [];

      const historyResponses = await Promise.all(
        configList.map((cfg: Config) => Api.configs.getConfigVersions(cfg.id, { page: 1, pageSize: 50 }))
      );

      const nextLogs: AuditLogItem[] = [];
      historyResponses.forEach((res, index) => {
        const cfg = configList[index];
        const app = appList.find((a) => a.appId === cfg.appId);
        (res.data.list || []).forEach((item: ConfigVersion) => {
          nextLogs.push({
            id: `${cfg.id}-${item.id}`,
            appId: cfg.appId,
            appName: app?.name || cfg.appId,
            key: cfg.key,
            env: cfg.env,
            action: normalizeAction(item.action),
            operator: item.operator || 'system',
            timestamp: item.createdAt,
            details: item.description || `${item.action} 配置版本 v${item.version}`,
            value: item.value || '',
          });
        });
      });

      setLogs(nextLogs);
    } catch (error) {
      message.error((error as Error).message || '加载审计日志失败');
      setLogs([]);
    } finally {
      setLoading(false);
    }
  };

  React.useEffect(() => {
    load().catch((error) => message.error((error as Error).message));
  }, []);

  React.useEffect(() => {
    if (selectedAppId === 'all') {
      load().catch((error) => message.error((error as Error).message));
    } else {
      load(selectedAppId).catch((error) => message.error((error as Error).message));
    }
  }, [selectedAppId]);

  const filteredLogs = useMemo(() => {
    return logs
      .filter((log) => {
        if (selectedAppId !== 'all' && log.appId !== selectedAppId) return false;
        if (actionFilter !== 'all' && log.action !== actionFilter) return false;
        if (dateRange && dateRange[0] && dateRange[1]) {
          const logTime = dayjs(log.timestamp);
          if (logTime.isBefore(dateRange[0]) || logTime.isAfter(dateRange[1])) return false;
        }
        if (searchText) {
          const search = searchText.toLowerCase();
          return (
            log.key.toLowerCase().includes(search) ||
            log.details.toLowerCase().includes(search) ||
            log.operator.toLowerCase().includes(search)
          );
        }
        return true;
      })
      .sort((a, b) => dayjs(b.timestamp).valueOf() - dayjs(a.timestamp).valueOf());
  }, [logs, selectedAppId, actionFilter, dateRange, searchText]);

  const stats = useMemo(
    () => ({
      total: filteredLogs.length,
      create: filteredLogs.filter((l) => l.action === 'create').length,
      update: filteredLogs.filter((l) => l.action === 'update').length,
      release: filteredLogs.filter((l) => l.action === 'release').length,
      rollback: filteredLogs.filter((l) => l.action === 'rollback').length,
    }),
    [filteredLogs]
  );

  const columns = [
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 180,
      render: (time: string) => (time ? new Date(time).toLocaleString() : '-'),
    },
    {
      title: '操作类型',
      dataIndex: 'action',
      key: 'action',
      width: 110,
      render: (action: AuditAction) => <Tag color={actionColors[action]} icon={actionIcons[action]}>{actionLabels[action]}</Tag>,
    },
    {
      title: '应用',
      dataIndex: 'appName',
      key: 'appName',
      render: (name: string) => <Tag color="blue">{name}</Tag>,
    },
    {
      title: '配置键',
      dataIndex: 'key',
      key: 'key',
      render: (key: string) => <code className="text-gray-300 bg-gray-800 px-2 py-1 rounded text-sm">{key}</code>,
    },
    {
      title: '环境',
      dataIndex: 'env',
      key: 'env',
      render: (env: string) => <Tag color="cyan">{env}</Tag>,
    },
    {
      title: '操作人',
      dataIndex: 'operator',
      key: 'operator',
    },
    {
      title: '详情',
      dataIndex: 'details',
      key: 'details',
      render: (details: string) => <span className="truncate max-w-[260px]" title={details}>{details}</span>,
    },
    {
      title: '操作',
      key: 'view',
      render: (_: unknown, record: AuditLogItem) => (
        <Button type="link" icon={<EyeOutlined />} size="small" onClick={() => { setSelectedLog(record); setIsDetailVisible(true); }}>
          详情
        </Button>
      ),
    },
  ];

  const timelineItems = filteredLogs.slice(0, 20).map((log) => ({
    color: actionColors[log.action],
    children: (
      <div className="flex justify-between items-start">
        <div>
          <Space>
            {actionIcons[log.action]}
            <span className="text-white">{actionLabels[log.action]}</span>
            <code className="text-gray-300 bg-gray-800 px-1 rounded text-xs">{log.key}</code>
          </Space>
          <div className="text-gray-400 text-sm mt-1">{log.details}</div>
        </div>
        <div className="text-right">
          <div className="text-gray-500 text-xs">{new Date(log.timestamp).toLocaleString()}</div>
          <div className="text-gray-400 text-xs">{log.operator}</div>
        </div>
      </div>
    ),
  }));

  return (
    <div className="fade-in">
      <Row gutter={16} className="mb-4">
        <Col span={4}><Card size="small" style={{ background: '#1a1a2e' }}><div className="text-center"><div className="text-2xl font-bold text-white">{stats.total}</div><div className="text-gray-400">总记录</div></div></Card></Col>
        <Col span={4}><Card size="small" style={{ background: '#1a1a2e' }}><div className="text-center"><div className="text-2xl font-bold text-green-400">{stats.create}</div><div className="text-gray-400">创建</div></div></Card></Col>
        <Col span={4}><Card size="small" style={{ background: '#1a1a2e' }}><div className="text-center"><div className="text-2xl font-bold text-blue-400">{stats.update}</div><div className="text-gray-400">更新</div></div></Card></Col>
        <Col span={4}><Card size="small" style={{ background: '#1a1a2e' }}><div className="text-center"><div className="text-2xl font-bold text-purple-400">{stats.release}</div><div className="text-gray-400">发布</div></div></Card></Col>
        <Col span={4}><Card size="small" style={{ background: '#1a1a2e' }}><div className="text-center"><div className="text-2xl font-bold text-orange-400">{stats.rollback}</div><div className="text-gray-400">回滚</div></div></Card></Col>
      </Row>

      <Card
        style={{ background: '#16213e', border: '1px solid #2d3748' }}
        title={<Space><HistoryOutlined style={{ color: '#3498db' }} /><span className="text-white text-lg">审计日志</span></Space>}
        extra={
          <Space>
            <Input placeholder="搜索配置键、详情或操作人" prefix={<SearchOutlined />} value={searchText} onChange={(e) => setSearchText(e.target.value)} style={{ width: 240 }} allowClear />
            <Select value={selectedAppId} onChange={setSelectedAppId} style={{ width: 180 }}>
              <Option value="all">全部应用</Option>
              {apps.map((app) => <Option key={app.id} value={app.appId}>{app.name}</Option>)}
            </Select>
            <Select value={actionFilter} onChange={setActionFilter} style={{ width: 110 }}>
              <Option value="all">全部操作</Option>
              <Option value="create">创建</Option>
              <Option value="update">更新</Option>
              <Option value="delete">删除</Option>
              <Option value="release">发布</Option>
              <Option value="rollback">回滚</Option>
            </Select>
            <RangePicker value={dateRange} onChange={(dates) => setDateRange(dates as [Dayjs | null, Dayjs | null] | null)} />
          </Space>
        }
      >
        {filteredLogs.length === 0 ? (
          <Empty description={loading ? '加载中...' : '暂无审计记录'} />
        ) : (
          <Table
            loading={loading}
            dataSource={filteredLogs}
            columns={columns}
            rowKey="id"
            pagination={{ pageSize: 15, showSizeChanger: true, showTotal: (total) => `共 ${total} 条记录` }}
            size="small"
          />
        )}
      </Card>

      <Card style={{ background: '#16213e', border: '1px solid #2d3748', marginTop: 16 }} title={<span className="text-white">时间线视图</span>}>
        {timelineItems.length === 0 ? <Empty description="暂无时间线数据" /> : <Timeline items={timelineItems} />}
      </Card>

      <Drawer title="变更详情" placement="right" width={700} open={isDetailVisible} onClose={() => setIsDetailVisible(false)}>
        {selectedLog && (
          <div>
            <div className="mb-4">
              <Space direction="vertical">
                <Space>
                  <Tag color={actionColors[selectedLog.action]} icon={actionIcons[selectedLog.action]}>{actionLabels[selectedLog.action]}</Tag>
                  <span className="text-lg text-white">{selectedLog.key}</span>
                </Space>
                <span className="text-gray-400">{selectedLog.details}</span>
              </Space>
            </div>

            <div className="grid grid-cols-2 gap-4 mb-4">
              <div><div className="text-gray-500 text-sm">应用</div><div className="text-white">{selectedLog.appName}</div></div>
              <div><div className="text-gray-500 text-sm">环境</div><div className="text-white">{selectedLog.env}</div></div>
              <div><div className="text-gray-500 text-sm">操作人</div><div className="text-white">{selectedLog.operator}</div></div>
              <div><div className="text-gray-500 text-sm">时间</div><div className="text-white">{new Date(selectedLog.timestamp).toLocaleString()}</div></div>
            </div>

            <div>
              <div className="text-gray-500 text-sm mb-2">配置值快照</div>
              <pre className="bg-gray-900 p-3 rounded text-green-400 text-sm overflow-x-auto max-h-96">{selectedLog.value || '-'}</pre>
            </div>
          </div>
        )}
      </Drawer>
    </div>
  );
};

export default AuditLogsPage;
