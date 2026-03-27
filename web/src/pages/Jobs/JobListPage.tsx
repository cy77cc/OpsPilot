import React, { useEffect, useMemo, useState } from 'react';
import { Table, Button, Space, Switch, Tag, Modal, notification, Card } from 'antd';
import { PlusOutlined, PlayCircleOutlined, EditOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../api';
import type { Task } from '../../api/modules/tasks';

const JobListPage: React.FC = () => {
  const navigate = useNavigate();
  const [jobs, setJobs] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [confirmLoading, setConfirmLoading] = useState(false);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [jobToDelete, setJobToDelete] = useState<string | null>(null);

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await Api.tasks.getTaskList({ page: 1, pageSize: 200 });
      setJobs(res.data.list || []);
    } catch (error) {
      notification.error({ message: '加载失败', description: (error as Error).message || '获取任务列表失败' });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleStartJob = async (id: string) => {
    await Api.tasks.startTask(id);
    notification.success({ message: '执行成功', description: '已触发任务执行' });
    fetchData();
  };

  const handleToggleEnable = async (id: string, enabled: boolean) => {
    const target = jobs.find((job) => job.id === id);
    if (!target) return;
    await Api.tasks.updateTask(id, { status: enabled ? 'pending' : 'stopped' });
    setJobs((prev) => prev.map((job) => (job.id === id ? { ...job, status: enabled ? 'pending' : 'stopped' } : job)));
  };

  const showDeleteConfirm = (id: string) => {
    setJobToDelete(id);
    setDeleteModalVisible(true);
  };

  const handleDeleteOk = async () => {
    if (!jobToDelete) return;
    setConfirmLoading(true);
    try {
      await Api.tasks.deleteTask(jobToDelete);
      setJobs((prev) => prev.filter((job) => job.id !== jobToDelete));
      notification.success({ message: '删除成功', description: '任务已删除' });
      setDeleteModalVisible(false);
      setJobToDelete(null);
    } finally {
      setConfirmLoading(false);
    }
  };

  const statusMeta = useMemo(() => ({
    running: { color: 'processing', text: '运行中' },
    pending: { color: 'warning', text: '等待中' },
    success: { color: 'success', text: '成功' },
    failed: { color: 'error', text: '失败' },
    stopped: { color: 'default', text: '已停止' },
  } as Record<string, { color: string; text: string }>), []);

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 100 },
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: Task) => (
        <a onClick={() => navigate(`/jobs/${record.id}/history`)}>{text}</a>
      ),
    },
    {
      title: '任务类型',
      dataIndex: 'type',
      key: 'type',
      width: 110,
      render: (type: string) => <Tag color="blue">{type}</Tag>,
    },
    { title: '执行计划', dataIndex: 'schedule', key: 'schedule', width: 150 },
    { title: '描述', dataIndex: 'description', key: 'description', render: (v: string) => v || '-' },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (date: string) => (date ? new Date(date).toLocaleString() : '-'),
    },
    {
      title: '状态',
      key: 'status',
      width: 110,
      render: (_: unknown, record: Task) => {
        const meta = statusMeta[record.status] || { color: 'default', text: record.status };
        return <Tag color={meta.color}>{meta.text}</Tag>;
      },
    },
    {
      title: '启用',
      key: 'enabled',
      width: 90,
      render: (_: unknown, record: Task) => (
        <Switch checked={record.status !== 'stopped'} onChange={(checked) => handleToggleEnable(record.id, checked)} />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 240,
      render: (_: unknown, record: Task) => (
        <Space size="small">
          <Button type="link" icon={<PlayCircleOutlined />} onClick={() => handleStartJob(record.id)} disabled={record.status === 'stopped'}>
            执行
          </Button>
          <Button type="link" icon={<EditOutlined />} onClick={() => navigate(`/jobs/${record.id}/edit`)}>
            编辑
          </Button>
          <Button type="link" danger icon={<DeleteOutlined />} onClick={() => showDeleteConfirm(record.id)}>
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Card
      style={{ background: '#16213e', border: '1px solid #2d3748' }}
      title={<span className="text-white text-lg">任务列表</span>}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={fetchData} loading={loading}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/jobs/create')}>创建任务</Button>
        </Space>
      }
    >
      <Table
        dataSource={jobs}
        columns={columns}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 10, showSizeChanger: true, showQuickJumper: true, showTotal: (total) => `共 ${total} 条记录` }}
      />

      <Modal title="确认删除" open={deleteModalVisible} onOk={handleDeleteOk} onCancel={() => setDeleteModalVisible(false)} confirmLoading={confirmLoading}>
        <p>确定要删除该任务吗？此操作不可恢复。</p>
      </Modal>
    </Card>
  );
};

export default JobListPage;
