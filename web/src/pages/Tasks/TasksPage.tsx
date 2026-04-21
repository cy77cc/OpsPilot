import React, { useEffect, useState } from 'react';
import { Button, Card, Form, Input, Modal, Select, Space, Table, Tag, message } from 'antd';
import { PlayCircleOutlined, ReloadOutlined, StopOutlined } from '@ant-design/icons';
import { Api } from '../../api';
import type { Task } from '../../api/modules/tasks';
import { TableSkeleton } from '../../components/LoadingSkeleton';
import { GuidedFormItem } from '../../components/FormGuidance';

const TasksPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [jobs, setJobs] = useState<Task[]>([]);
  const [createOpen, setCreateOpen] = useState(false);
  const [logOpen, setLogOpen] = useState(false);
  const [logs, setLogs] = useState<any[]>([]);
  const [selectedId, setSelectedId] = useState<string>('');
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const res = await Api.tasks.getTaskList({ page: 1, pageSize: 100 });
      setJobs(res.data.list || []);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const createJob = async () => {
    const v = await form.validateFields();
    await Api.tasks.createTask({
      name: v.name,
      type: v.type,
      schedule: v.cron,
      nextRun: '',
    } as any);
    message.success('任务创建成功');
    setCreateOpen(false);
    form.resetFields();
    load();
  };

  const start = async (id: string) => {
    await Api.tasks.startTask(id);
    message.success('已触发执行');
    load();
  };
  const stop = async (id: string) => {
    await Api.tasks.stopTask(id);
    message.success('已停止');
    load();
  };

  const openLogs = async (id: string) => {
    setSelectedId(id);
    const res = await Api.tasks.getTaskLogs(id, { page: 1, pageSize: 100 });
    setLogs(res.data.list || []);
    setLogOpen(true);
  };

  const isInitialLoading = loading && jobs.length === 0;

  return (
    <Card
      title="任务调度"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading && !isInitialLoading}>刷新</Button>
          <Button type="primary" onClick={() => setCreateOpen(true)}>创建任务</Button>
        </Space>
      }
    >
      {isInitialLoading ? (
        <TableSkeleton toolbar={false} rows={8} columns={5} />
      ) : (
        <Table
          rowKey="id"
          loading={false}
          dataSource={jobs}
          columns={[
          { title: '名称', dataIndex: 'name' },
          { title: '类型', dataIndex: 'type' },
          { title: 'Cron', dataIndex: 'schedule' },
          { title: '状态', dataIndex: 'status', render: (v: string) => <Tag color={v === 'running' ? 'processing' : v === 'success' ? 'success' : 'default'}>{v}</Tag> },
          { title: '创建时间', dataIndex: 'createdAt', render: (v: string) => (v ? new Date(v).toLocaleString() : '-') },
          {
            title: '操作',
            render: (_: unknown, r: Task) => (
              <Space>
                <Button type="link" icon={<PlayCircleOutlined />} onClick={() => start(String(r.id))}>执行</Button>
                <Button type="link" icon={<StopOutlined />} onClick={() => stop(String(r.id))}>停止</Button>
                <Button type="link" onClick={() => openLogs(String(r.id))}>日志</Button>
              </Space>
            ),
          },
          ]}
        />
      )}

      <Modal title="创建任务" open={createOpen} onCancel={() => setCreateOpen(false)} onOk={createJob}>
        <Form form={form} layout="vertical">
          <GuidedFormItem name="name" label="名称" rules={[{ required: true }]}><Input /></GuidedFormItem>
          <Form.Item name="type" label="类型" rules={[{ required: true }]}><Select options={[{ value: 'shell' }, { value: 'script' }]} /></Form.Item>
          <GuidedFormItem name="cron" label="Cron"><Input placeholder="*/5 * * * *" /></GuidedFormItem>
        </Form>
      </Modal>

      <Modal title={`任务日志 #${selectedId}`} open={logOpen} onCancel={() => setLogOpen(false)} footer={null} width={860}>
        <Table
          rowKey="id"
          dataSource={logs}
          columns={[
            { title: '级别', dataIndex: 'level', width: 100 },
            { title: '内容', dataIndex: 'message' },
            { title: '时间', dataIndex: 'created_at', width: 180, render: (v: string) => (v ? new Date(v).toLocaleString() : '-') },
          ]}
          pagination={false}
        />
      </Modal>
    </Card>
  );
};

export default TasksPage;
