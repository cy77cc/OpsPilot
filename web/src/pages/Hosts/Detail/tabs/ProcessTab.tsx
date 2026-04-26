import React, { useState, useEffect, useCallback } from 'react';
import { Card, Table, Tag, Input, Space, Button, Modal, message } from 'antd';
import { SearchOutlined, ReloadOutlined, StopOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { hostApi } from '../../../../api/modules/hosts';

interface ProcessItem {
  pid: number;
  user: string;
  cpu: number;
  memory: number;
  vsz: number;
  rss: number;
  state: string;
  start: string;
  time: string;
  command: string;
}

const ProcessTab: React.FC<{ hostId: string }> = ({ hostId }) => {
  const [searchText, setSearchText] = useState('');
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<ProcessItem[]>([]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await hostApi.getHostProcesses(hostId);
      setData(res.data || []);
    } finally {
      setLoading(false);
    }
  }, [hostId]);

  useEffect(() => {
    if (hostId) fetchData();
  }, [hostId, fetchData]);

  const handleKill = (pid: number) => {
    Modal.confirm({
      title: '确认终止进程',
      content: `确定要终止 PID 为 ${pid} 的进程吗？这可能会导致应用异常。`,
      okText: '确定',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await hostApi.killProcess(hostId, pid);
          message.success(`已发送终止信号到进程 ${pid}`);
          fetchData();
        } catch (err) {
          message.error('操作失败');
        }
      },
    });
  };

  const columns: ColumnsType<ProcessItem> = [
    { title: 'PID', dataIndex: 'pid', key: 'pid', width: 80, sorter: (a, b) => a.pid - b.pid },
    { title: '用户', dataIndex: 'user', key: 'user', width: 100 },
    { title: 'CPU (%)', dataIndex: 'cpu', key: 'cpu', width: 100, sorter: (a, b) => a.cpu - b.cpu, render: (val) => <span className={val > 10 ? 'text-red-500 font-medium' : ''}>{val}%</span> },
    { title: '内存 (%)', dataIndex: 'memory', key: 'memory', width: 100, sorter: (a, b) => a.memory - b.memory, render: (val) => <span>{val}%</span> },
    { title: '状态', dataIndex: 'state', key: 'state', width: 80, render: (state) => <Tag color={state === 'R' ? 'green' : 'default'}>{state}</Tag> },
    { title: '启动时间', dataIndex: 'start', key: 'start', width: 100 },
    { title: '运行命令', dataIndex: 'command', key: 'command', ellipsis: true },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_, record) => (
        <Button 
          type="link" 
          danger 
          size="small" 
          icon={<StopOutlined />} 
          onClick={() => handleKill(record.pid)}
        >
          终止
        </Button>
      ),
    },
  ];

  return (
    <Card className="h-full border-none shadow-sm mt-4">
      <div className="flex justify-between items-center mb-4">
        <Space>
          <Input
            placeholder="搜索进程名、PID或用户..."
            prefix={<SearchOutlined className="text-gray-400" />}
            value={searchText}
            onChange={e => setSearchText(e.target.value)}
            style={{ width: 300 }}
            allowClear
          />
          <Button icon={<ReloadOutlined />} onClick={fetchData} loading={loading}>
            刷新
          </Button>
        </Space>
        <div className="text-gray-400 text-xs">
          当前共运行 {data.length} 个进程
        </div>
      </div>

      <Table
        columns={columns}
        dataSource={data.filter(p => 
          p.command.toLowerCase().includes(searchText.toLowerCase()) || 
          p.pid.toString().includes(searchText) ||
          p.user.toLowerCase().includes(searchText.toLowerCase())
        )}
        rowKey="pid"
        size="small"
        pagination={{ pageSize: 15, showSizeChanger: true }}
        loading={loading}
      />
    </Card>
  );
};

export default ProcessTab;
