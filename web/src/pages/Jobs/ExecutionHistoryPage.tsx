import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Table, Button, Tag, Card, Divider, Statistic, Progress, Descriptions, Alert, message, Space } from 'antd';
import { ReloadOutlined, PlayCircleOutlined } from '@ant-design/icons';
import { useParams } from 'react-router-dom';
import { Api } from '../../api';
import type { TaskExecution, TaskLog } from '../../api/modules/tasks';

const ExecutionHistoryPage: React.FC = () => {
  const { jobId } = useParams<{ jobId: string }>();
  const [executions, setExecutions] = useState<TaskExecution[]>([]);
  const [logs, setLogs] = useState<TaskLog[]>([]);
  const [currentExecution, setCurrentExecution] = useState<TaskExecution | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const terminalRef = useRef<HTMLDivElement>(null);

  const loadData = async () => {
    if (!jobId) return;
    setIsLoading(true);
    try {
      const [execRes, logRes] = await Promise.all([
        Api.tasks.getTaskExecutions(jobId, { page: 1, pageSize: 100 }),
        Api.tasks.getTaskLogs(jobId, { page: 1, pageSize: 500 }),
      ]);
      const execList = execRes.data.list || [];
      setExecutions(execList);
      setLogs(logRes.data.list || []);
      setCurrentExecution(execList[0] || null);
    } catch (error) {
      message.error((error as Error).message || '加载执行历史失败');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [jobId]);

  useEffect(() => {
    if (terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight;
    }
  }, [logs, currentExecution]);

  const handleRerun = async () => {
    if (!jobId) return;
    await Api.tasks.startTask(jobId);
    message.success('已触发重新执行');
    await loadData();
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'running': return 'blue';
      case 'success': return 'green';
      case 'failed': return 'red';
      case 'stopped': return 'default';
      case 'pending': return 'orange';
      default: return 'default';
    }
  };

  const stats = useMemo(() => {
    const total = executions.length;
    const success = executions.filter((e) => e.status === 'success').length;
    const failed = executions.filter((e) => e.status === 'failed').length;
    const durations = executions
      .filter((e) => e.startTime && e.endTime)
      .map((e) => Math.max(0, (new Date(e.endTime!).getTime() - new Date(e.startTime!).getTime()) / 1000));
    const avgDuration = durations.length ? Math.round(durations.reduce((a, b) => a + b, 0) / durations.length) : 0;
    return { total, success, failed, avgDuration };
  }, [executions]);

  const progress = currentExecution?.status === 'success' ? 100 : currentExecution?.status === 'running' ? 65 : currentExecution?.status === 'failed' ? 100 : 0;

  const columns = [
    { title: '执行ID', dataIndex: 'id', key: 'id', width: 120 },
    { title: '开始时间', dataIndex: 'startTime', key: 'startTime', render: (time: string) => (time ? new Date(time).toLocaleString() : '-') },
    { title: '结束时间', dataIndex: 'endTime', key: 'endTime', render: (time: string) => (time ? new Date(time).toLocaleString() : '-') },
    {
      title: '执行时长',
      key: 'duration',
      render: (_: unknown, record: TaskExecution) => {
        if (record.startTime && record.endTime) {
          const start = new Date(record.startTime).getTime();
          const end = new Date(record.endTime).getTime();
          return `${Math.round((end - start) / 1000)} 秒`;
        }
        return '-';
      },
    },
    { title: '状态', key: 'status', render: (_: unknown, record: TaskExecution) => <Tag color={getStatusColor(record.status)}>{record.status}</Tag> },
    { title: '退出码', dataIndex: 'exitCode', key: 'exitCode', render: (code: number) => (code === undefined ? '-' : code) },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: TaskExecution) => (
        <Button size="small" type="link" onClick={() => setCurrentExecution(record)}>
          查看详情
        </Button>
      ),
    },
  ];

  const currentLogs = logs.filter((log) => !currentExecution || !log.executionId || log.executionId === currentExecution.id);

  return (
    <Card
      style={{ background: '#16213e', border: '1px solid #2d3748' }}
      title={<span className="text-white text-lg">执行历史</span>}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={loadData} loading={isLoading}>刷新</Button>
          <Button type="primary" icon={<PlayCircleOutlined />} onClick={handleRerun} loading={isLoading}>重新执行</Button>
        </Space>
      }
    >
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <Card size="small"><Statistic title="总执行次数" value={stats.total} valueStyle={{ color: '#3f8600' }} /></Card>
        <Card size="small"><Statistic title="成功执行" value={stats.success} valueStyle={{ color: '#3f8600' }} /></Card>
        <Card size="small"><Statistic title="失败次数" value={stats.failed} valueStyle={{ color: '#cf1322' }} /></Card>
        <Card size="small"><Statistic title="平均时长" value={stats.avgDuration} suffix="秒" valueStyle={{ color: '#1890ff' }} /></Card>
      </div>

      {currentExecution && (
        <Card size="small" className="mb-4" title="当前执行">
          <Descriptions column={4} bordered>
            <Descriptions.Item label="状态"><Tag color={getStatusColor(currentExecution.status)}>{currentExecution.status}</Tag></Descriptions.Item>
            <Descriptions.Item label="开始时间">{currentExecution.startTime ? new Date(currentExecution.startTime).toLocaleString() : '-'}</Descriptions.Item>
            <Descriptions.Item label="执行ID">{currentExecution.id}</Descriptions.Item>
            <Descriptions.Item label="主机">{currentExecution.hostIp || '-'}</Descriptions.Item>
          </Descriptions>
          <div className="mt-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm font-medium">执行进度</span>
              <span className="text-xs text-gray-500">{progress}%</span>
            </div>
            <Progress percent={progress} status={currentExecution.status === 'failed' ? 'exception' : 'active'} />
          </div>
          {currentExecution.output && (
            <div className="mt-4">
              <div className="text-gray-400 mb-2">执行输出</div>
              <pre className="bg-gray-900 p-3 rounded text-green-400 text-sm overflow-x-auto max-h-56">{currentExecution.output}</pre>
            </div>
          )}
        </Card>
      )}

      <div className="bg-black text-green-400 font-mono p-4 rounded-lg mb-6 h-80 overflow-auto" ref={terminalRef}>
        {currentLogs.length === 0 ? (
          <div className="text-gray-500">等待日志输出 ...</div>
        ) : (
          currentLogs.map((log) => (
            <div key={log.id}>
              [{new Date(log.timestamp).toLocaleTimeString()}] [{log.level.toUpperCase()}] {log.message}
            </div>
          ))
        )}
      </div>

      {!isLoading && !currentExecution && (
        <Alert message="提示" description="没有执行记录，请手动执行或等待调度触发" type="info" showIcon />
      )}

      <Divider />

      <h3 className="text-lg font-semibold mb-4 text-white">执行历史列表</h3>
      <Table dataSource={executions} columns={columns} rowKey="id" loading={isLoading} pagination={{ pageSize: 10 }} />
    </Card>
  );
};

export default ExecutionHistoryPage;
