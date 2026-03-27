import React, { useState, useEffect } from 'react';
import { Card, Table, Button, Space, Input, Select, DatePicker, Tag, Modal, message } from 'antd';
import {
  ReloadOutlined,
  SearchOutlined,
  DownloadOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { Api } from '../../../api';
import type { AuditLog } from '../../../api/modules/deployment';

const { RangePicker } = DatePicker;

const AuditLogsPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [actionFilter, setActionFilter] = useState<string>('');
  const [resourceTypeFilter, setResourceTypeFilter] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState('');
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs | null, dayjs.Dayjs | null]>([null, null]);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);

  const load = async () => {
    setLoading(true);
    try {
      const res = await Api.deployment.getAuditLogs({
        page,
        page_size: pageSize,
        action_type: actionFilter || undefined,
        resource_type: resourceTypeFilter || undefined,
      });
      setLogs(res.data.list || []);
      setTotal(res.data.total || 0);
    } catch (err) {
      message.error('加载审计日志失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [page, pageSize, actionFilter, resourceTypeFilter]);

  const handleExport = (format: 'csv' | 'json') => {
    if (format === 'json') {
      const dataStr = JSON.stringify(logs, null, 2);
      const dataBlob = new Blob([dataStr], { type: 'application/json' });
      const url = URL.createObjectURL(dataBlob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `audit-logs-${Date.now()}.json`;
      link.click();
    } else {
      const headers = ['ID', 'Action', 'Actor', 'Resource Type', 'Resource ID', 'Created At'];
      const csvContent = [
        headers.join(','),
        ...logs.map((log) =>
          [
            log.id,
            log.action_type,
            log.actor_name,
            log.resource_type,
            log.resource_id,
            log.created_at,
          ].join(',')
        ),
      ].join('\n');
      const dataBlob = new Blob([csvContent], { type: 'text/csv' });
      const url = URL.createObjectURL(dataBlob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `audit-logs-${Date.now()}.csv`;
      link.click();
    }
    message.success(`已导出 ${format.toUpperCase()} 格式`);
  };

  const getFilteredLogs = () => {
    let filtered = logs;

    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        (log) =>
          log.action_type.toLowerCase().includes(query) ||
          log.actor_name.toLowerCase().includes(query) ||
          JSON.stringify(log.detail).toLowerCase().includes(query)
      );
    }

    // Date range filter
    if (dateRange[0] && dateRange[1]) {
      filtered = filtered.filter((log) => {
        const logDate = dayjs(log.created_at);
        return logDate.isAfter(dateRange[0]) && logDate.isBefore(dateRange[1]);
      });
    }

    return filtered;
  };

  const getActionTagColor = (actionType: string) => {
    if (actionType.includes('create') || actionType.includes('apply')) return 'blue';
    if (actionType.includes('update')) return 'orange';
    if (actionType.includes('delete')) return 'red';
    if (actionType.includes('approve')) return 'green';
    if (actionType.includes('reject')) return 'default';
    return 'default';
  };

  const getActionLabel = (actionType: string) => {
    const labels: Record<string, string> = {
      release_apply: '发布应用',
      release_approve: '审批通过',
      release_reject: '审批拒绝',
      release_rollback: '回滚',
      target_create: '创建目标',
      target_update: '更新目标',
      target_delete: '删除目标',
      cluster_bootstrap: '集群引导',
      credential_create: '创建凭证',
      credential_test: '测试凭证',
    };
    return labels[actionType] || actionType;
  };

  const columns: ColumnsType<AuditLog> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: '操作',
      dataIndex: 'action_type',
      key: 'action_type',
      render: (actionType: string) => (
        <Tag color={getActionTagColor(actionType)}>{getActionLabel(actionType)}</Tag>
      ),
    },
    {
      title: '操作人',
      dataIndex: 'actor_name',
      key: 'actor_name',
    },
    {
      title: '资源类型',
      dataIndex: 'resource_type',
      key: 'resource_type',
      render: (type: string) => {
        const labels: Record<string, string> = {
          release: '发布',
          target: '目标',
          cluster: '集群',
          credential: '凭证',
        };
        return labels[type] || type;
      },
    },
    {
      title: '资源 ID',
      dataIndex: 'resource_id',
      key: 'resource_id',
      render: (id: number, record: AuditLog) => {
        if (record.resource_type === 'release') {
          return <a onClick={() => navigate(`/deployment/${id}`)}>#{id}</a>;
        }
        return `#${id}`;
      },
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time: string) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'actions',
      width: 100,
      render: (_: any, record: AuditLog) => (
        <Button
          type="link"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => {
            setSelectedLog(record);
            setDetailModalVisible(true);
          }}
        >
          详情
        </Button>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">审计日志</h1>
          <p className="text-sm text-gray-500 mt-1">查看所有系统操作记录</p>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
          <Button icon={<DownloadOutlined />} onClick={() => handleExport('csv')}>
            导出 CSV
          </Button>
          <Button icon={<DownloadOutlined />} onClick={() => handleExport('json')}>
            导出 JSON
          </Button>
        </Space>
      </div>

      {/* Filters */}
      <Card>
        <Space direction="vertical" size="middle" className="w-full">
          <div className="flex flex-wrap gap-3">
            <Input
              placeholder="搜索操作、操作人或详情"
              prefix={<SearchOutlined className="text-gray-400" />}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              style={{ width: 280 }}
              allowClear
            />
            <Select
              value={actionFilter}
              style={{ width: 160 }}
              placeholder="操作类型"
              allowClear
              options={[
                { value: '', label: '全部操作' },
                { value: 'release_apply', label: '发布应用' },
                { value: 'release_approve', label: '审批通过' },
                { value: 'release_reject', label: '审批拒绝' },
                { value: 'target_create', label: '创建目标' },
                { value: 'target_update', label: '更新目标' },
                { value: 'target_delete', label: '删除目标' },
              ]}
              onChange={setActionFilter}
            />
            <Select
              value={resourceTypeFilter}
              style={{ width: 160 }}
              placeholder="资源类型"
              allowClear
              options={[
                { value: '', label: '全部资源' },
                { value: 'release', label: '发布' },
                { value: 'target', label: '目标' },
                { value: 'cluster', label: '集群' },
                { value: 'credential', label: '凭证' },
              ]}
              onChange={setResourceTypeFilter}
            />
            <RangePicker
              value={dateRange}
              onChange={(dates) => setDateRange(dates as [dayjs.Dayjs | null, dayjs.Dayjs | null])}
              style={{ width: 280 }}
            />
          </div>
        </Space>
      </Card>

      {/* Audit log table */}
      <Card>
        <Table
          dataSource={getFilteredLogs()}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Card>

      {/* Detail modal */}
      <Modal
        title="审计日志详情"
        open={detailModalVisible}
        onCancel={() => {
          setDetailModalVisible(false);
          setSelectedLog(null);
        }}
        footer={[
          <Button key="close" onClick={() => setDetailModalVisible(false)}>
            关闭
          </Button>,
        ]}
        width={720}
      >
        {selectedLog && (
          <div className="space-y-4">
            <div>
              <div className="text-sm font-semibold mb-1">操作:</div>
              <Tag color={getActionTagColor(selectedLog.action_type)}>
                {getActionLabel(selectedLog.action_type)}
              </Tag>
            </div>
            <div>
              <div className="text-sm font-semibold mb-1">操作人:</div>
              <div>{selectedLog.actor_name}</div>
            </div>
            <div>
              <div className="text-sm font-semibold mb-1">资源:</div>
              <div>
                {selectedLog.resource_type} #{selectedLog.resource_id}
              </div>
            </div>
            <div>
              <div className="text-sm font-semibold mb-1">时间:</div>
              <div>{new Date(selectedLog.created_at).toLocaleString()}</div>
            </div>
            <div>
              <div className="text-sm font-semibold mb-2">详细信息:</div>
              <pre className="bg-gray-900 text-gray-100 p-4 rounded overflow-auto max-h-96 text-xs">
                {JSON.stringify(selectedLog.detail, null, 2)}
              </pre>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default AuditLogsPage;
