import React, { useEffect, useState } from 'react';
import { Button, Card, Input, Modal, Space, Table, Tag, message } from 'antd';
import { PlayCircleOutlined, ReloadOutlined, ToolOutlined } from '@ant-design/icons';
import { Api } from '../../api';
import type { ToolExecution, ToolItem } from '../../api/modules/tools';

const { Search } = Input;

const ToolsPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [tools, setTools] = useState<ToolItem[]>([]);
  const [keyword, setKeyword] = useState('');
  const [executionOpen, setExecutionOpen] = useState(false);
  const [executions, setExecutions] = useState<ToolExecution[]>([]);
  const [selectedTool, setSelectedTool] = useState<ToolItem | null>(null);

  const load = async () => {
    setLoading(true);
    try {
      const res = await Api.tools.getToolList({ page: 1, pageSize: 100 });
      setTools(res.data.list || []);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const execute = async (tool: ToolItem) => {
    await Api.tools.executeTool(tool.id);
    message.success(`已触发执行: ${tool.name}`);
    const list = await Api.tools.listExecutions({ toolId: tool.id, page: 1, pageSize: 20 });
    setSelectedTool(tool);
    setExecutions(list.data.list || []);
    setExecutionOpen(true);
  };

  const filtered = tools.filter((t) => t.name.toLowerCase().includes(keyword.toLowerCase()));

  return (
    <Card
      title={<span><ToolOutlined className="mr-2" />工具集成</span>}
      extra={
        <Space>
          <Search placeholder="搜索工具" value={keyword} onChange={(e) => setKeyword(e.target.value)} style={{ width: 220 }} />
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>刷新</Button>
        </Space>
      }
    >
      <Table
        rowKey="id"
        loading={loading}
        dataSource={filtered}
        columns={[
          { title: '名称', dataIndex: 'name' },
          { title: '类型', dataIndex: 'type', render: (v: string) => <Tag>{v}</Tag> },
          { title: '路径', dataIndex: 'path', ellipsis: true },
          { title: '状态', dataIndex: 'enabled', render: (v: boolean) => <Tag color={v ? 'success' : 'default'}>{v ? '启用' : '禁用'}</Tag> },
          {
            title: '操作',
            render: (_: unknown, row: ToolItem) => (
              <Button type="link" icon={<PlayCircleOutlined />} onClick={() => execute(row)}>
                执行
              </Button>
            ),
          },
        ]}
      />

      <Modal
        title={`执行记录 - ${selectedTool?.name || ''}`}
        open={executionOpen}
        onCancel={() => setExecutionOpen(false)}
        footer={null}
        width={900}
      >
        <Table
          rowKey="id"
          dataSource={executions}
          columns={[
            { title: '状态', dataIndex: 'status', render: (v: string) => <Tag color={v === 'success' ? 'success' : v === 'failed' ? 'error' : 'processing'}>{v}</Tag> },
            { title: '输出', dataIndex: 'output', ellipsis: true },
            { title: '错误', dataIndex: 'error', ellipsis: true },
            { title: '开始', dataIndex: 'startTime', render: (v?: string) => (v ? new Date(v).toLocaleString() : '-') },
            { title: '结束', dataIndex: 'endTime', render: (v?: string) => (v ? new Date(v).toLocaleString() : '-') },
          ]}
          pagination={false}
        />
      </Modal>
    </Card>
  );
};

export default ToolsPage;

