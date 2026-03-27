import React, { useState } from 'react';
import { Card, Table, Tag, Button, Space, Input, Select, Row, Col, Descriptions, Progress, Statistic, Timeline, Drawer } from 'antd';
import {
  SearchOutlined,
  ReloadOutlined,
  PoweroffOutlined,
  SyncOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import { hosts as mockHosts } from '../../data/mockData';
import type { Host } from '../../types';

const { Option } = Select;

const HostsPage: React.FC = () => {
  const [selectedHost, setSelectedHost] = useState<Host | null>(null);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [regionFilter, setRegionFilter] = useState<string>('all');

  const statusColors: Record<string, string> = {
    online: 'success',
    offline: 'default',
    warning: 'warning',
    maintenance: 'processing',
  };

  const filteredHosts = mockHosts.filter(host => {
    const matchSearch = host.name.toLowerCase().includes(searchText.toLowerCase()) || 
                       host.ip.includes(searchText);
    const matchStatus = statusFilter === 'all' || host.status === statusFilter;
    const matchRegion = regionFilter === 'all' || host.region === regionFilter;
    return matchSearch && matchStatus && matchRegion;
  });

  const columns = [
    { 
      title: '主机名称', 
      dataIndex: 'name', 
      key: 'name', 
      render: (text: string, record: Host) => (
        <a onClick={() => { setSelectedHost(record); setDrawerVisible(true); }} className="text-blue-400 hover:text-blue-300">
          {text}
        </a>
      ) 
    },
    { title: 'IP地址', dataIndex: 'ip', key: 'ip', render: (ip: string) => <code className="text-gray-300">{ip}</code> },
    { title: '状态', dataIndex: 'status', key: 'status', render: (status: string) => <Tag color={statusColors[status]}>{status.toUpperCase()}</Tag> },
    { title: '机房', dataIndex: 'region', key: 'region' },
    { 
      title: 'CPU', 
      dataIndex: 'cpu', 
      key: 'cpu', 
      render: (val: number) => (
        <div className="w-24">
          <Progress percent={val} size="small" strokeColor={val > 80 ? '#ff4757' : val > 60 ? '#ffc107' : '#00d9a5'} />
        </div>
      )
    },
    { 
      title: '内存', 
      dataIndex: 'memory', 
      key: 'memory', 
      render: (val: number) => (
        <div className="w-24">
          <Progress percent={val} size="small" strokeColor={val > 85 ? '#ff4757' : val > 70 ? '#ffc107' : '#00d9a5'} />
        </div>
      )
    },
    { 
      title: '磁盘', 
      dataIndex: 'disk', 
      key: 'disk', 
      render: (val: number) => (
        <div className="w-24">
          <Progress percent={val} size="small" strokeColor={val > 90 ? '#ff4757' : val > 70 ? '#ffc107' : '#3498db'} />
        </div>
      )
    },
    { title: '标签', dataIndex: 'tags', key: 'tags', render: (tags: string[]) => (
      <Space>
        {tags.slice(0, 2).map(tag => <Tag key={tag} color="blue">{tag}</Tag>)}
        {tags.length > 2 && <Tag>+{tags.length - 2}</Tag>}
      </Space>
    )},
    { title: '最后活跃', dataIndex: 'lastActive', key: 'lastActive', render: (time: string) => <span className="text-gray-500 text-xs">{time}</span> },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Host) => (
        <Space>
          <Button type="link" icon={<EyeOutlined />} onClick={() => { setSelectedHost(record); setDrawerVisible(true); }}>详情</Button>
          <Button type="link" icon={<SyncOutlined />}>重启</Button>
          <Button type="link" danger icon={<PoweroffOutlined />}>关机</Button>
        </Space>
      ),
    },
  ];

  const regions = [...new Set(mockHosts.map(h => h.region))];
  const onlineCount = mockHosts.filter(h => h.status === 'online').length;
  const warningCount = mockHosts.filter(h => h.status === 'warning').length;

  return (
    <div className="fade-in">
      <Row gutter={[16, 16]} className="mb-4">
        <Col xs={24} sm={8}>
          <Card style={{ background: '#16213e', border: '1px solid #2d3748' }}>
            <Statistic
              title={<span className="text-gray-400">在线主机</span>}
              value={onlineCount}
              valueStyle={{ color: '#00d9a5' }}
              suffix={`/ ${mockHosts.length}`}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card style={{ background: '#16213e', border: '1px solid #2d3748' }}>
            <Statistic
              title={<span className="text-gray-400">警告主机</span>}
              value={warningCount}
              valueStyle={{ color: '#ffc107' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card style={{ background: '#16213e', border: '1px solid #2d3748' }}>
            <Statistic
              title={<span className="text-gray-400">离线主机</span>}
              value={mockHosts.filter(h => h.status === 'offline').length}
              valueStyle={{ color: '#ff4757' }}
            />
          </Card>
        </Col>
      </Row>

      <Card 
        style={{ background: '#16213e', border: '1px solid #2d3748' }}
        title={<span className="text-white text-lg">主机列表</span>}
        extra={
          <Space>
            <Input
              placeholder="搜索主机名称或IP"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={e => setSearchText(e.target.value)}
              style={{ width: 200 }}
            />
            <Select value={statusFilter} onChange={setStatusFilter} style={{ width: 120 }}>
              <Option value="all">全部状态</Option>
              <Option value="online">在线</Option>
              <Option value="warning">警告</Option>
              <Option value="offline">离线</Option>
              <Option value="maintenance">维护中</Option>
            </Select>
            <Select value={regionFilter} onChange={setRegionFilter} style={{ width: 150 }}>
              <Option value="all">全部机房</Option>
              {regions.map(r => <Option key={r} value={r}>{r}</Option>)}
            </Select>
            <Button icon={<ReloadOutlined />}>刷新</Button>
          </Space>
        }
      >
        <Table
          dataSource={filteredHosts}
          columns={columns}
          rowKey="id"
          pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total) => `共 ${total} 台主机` }}
        />
      </Card>

      <Drawer
        title={selectedHost?.name || '主机详情'}
        placement="right"
        width={600}
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
      >
        {selectedHost && (
          <>
            <Descriptions title="基本信息" column={2} bordered size="small">
              <Descriptions.Item label="主机名称">{selectedHost.name}</Descriptions.Item>
              <Descriptions.Item label="IP地址">{selectedHost.ip}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={statusColors[selectedHost.status]}>{selectedHost.status.toUpperCase()}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="机房">{selectedHost.region}</Descriptions.Item>
              <Descriptions.Item label="创建时间">{selectedHost.createdAt}</Descriptions.Item>
              <Descriptions.Item label="最后活跃">{selectedHost.lastActive}</Descriptions.Item>
              <Descriptions.Item label="标签" span={2}>
                {selectedHost.tags.map(tag => <Tag key={tag} color="blue">{tag}</Tag>)}
              </Descriptions.Item>
            </Descriptions>

            <Card size="small" title="资源使用" className="mt-4" style={{ background: '#0f3460', border: '1px solid #2d3748' }}>
              <Row gutter={16}>
                <Col span={8}>
                  <Statistic 
                    title="CPU" 
                    value={selectedHost.cpu} 
                    suffix="%" 
                    valueStyle={{ color: selectedHost.cpu > 80 ? '#ff4757' : '#00d9a5' }}
                  />
                  <Progress percent={selectedHost.cpu} showInfo={false} strokeColor={selectedHost.cpu > 80 ? '#ff4757' : '#00d9a5'} />
                </Col>
                <Col span={8}>
                  <Statistic 
                    title="内存" 
                    value={selectedHost.memory} 
                    suffix="%" 
                    valueStyle={{ color: selectedHost.memory > 85 ? '#ff4757' : '#00d9a5' }}
                  />
                  <Progress percent={selectedHost.memory} showInfo={false} strokeColor={selectedHost.memory > 85 ? '#ff4757' : '#00d9a5'} />
                </Col>
                <Col span={8}>
                  <Statistic 
                    title="磁盘" 
                    value={selectedHost.disk} 
                    suffix="%" 
                    valueStyle={{ color: selectedHost.disk > 90 ? '#ff4757' : '#3498db' }}
                  />
                  <Progress percent={selectedHost.disk} showInfo={false} strokeColor={selectedHost.disk > 90 ? '#ff4757' : '#3498db'} />
                </Col>
              </Row>
            </Card>

            <Card size="small" title="网络流量" className="mt-4" style={{ background: '#0f3460', border: '1px solid #2d3748' }}>
              <Statistic 
                title="网络使用" 
                value={selectedHost.network} 
                suffix="Mbps"
                valueStyle={{ color: '#3498db' }}
              />
              <Progress percent={selectedHost.network} showInfo={false} strokeColor="#3498db" />
            </Card>

            <Card size="small" title="操作日志" className="mt-4" style={{ background: '#0f3460', border: '1px solid #2d3748' }}>
              <Timeline
                items={[
                  { color: 'green', children: '2024-02-20 10:30:00 系统健康检查通过' },
                  { color: 'green', children: '2024-02-20 09:00:00 定时任务执行成功' },
                  { color: 'yellow', children: '2024-02-20 08:15:00 CPU使用率告警 (85%)' },
                  { color: 'green', children: '2024-02-20 00:00:00 系统更新完成' },
                ]}
              />
            </Card>
          </>
        )}
      </Drawer>
    </div>
  );
};

export default HostsPage;
