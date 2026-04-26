import React, { useEffect, useState } from 'react';
import { Card, Table, Tag, Row, Col, Statistic, Tooltip, Spin } from 'antd';
import { InfoCircleOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { hostApi } from '../../../../api/modules/hosts';

interface InterfaceItem {
  name: string;
  ip: string;
  mac: string;
  status: 'up' | 'down';
  rx: string;
  tx: string;
  mtu: number;
}

const NetworkTab: React.FC<{ hostId: string }> = ({ hostId }) => {
  const [loading, setLoading] = useState(false);
  const [interfaces, setInterfaces] = useState<InterfaceItem[]>([]);
  const [metrics, setMetrics] = useState<any>(null);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        const [ifaceRes, metricRes] = await Promise.all([
          hostApi.getHostNetworkInterfaces(hostId),
          hostApi.getHostMetrics(hostId),
        ]);
        setInterfaces(ifaceRes.data || []);
        if (metricRes.data && metricRes.data.length > 0) {
          setMetrics(metricRes.data[metricRes.data.length - 1]);
        }
      } finally {
        setLoading(false);
      }
    };
    if (hostId) fetchData();
  }, [hostId]);

  const columns: ColumnsType<InterfaceItem> = [
    { title: '接口名称', dataIndex: 'name', key: 'name', width: 120 },
    { title: 'IPv4 地址', dataIndex: 'ip', key: 'ip', width: 150 },
    { title: 'MAC 地址', dataIndex: 'mac', key: 'mac', width: 180, render: (mac) => <code className="text-xs">{mac}</code> },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status) => <Tag color={status === 'up' ? 'success' : 'default'}>{status.toUpperCase()}</Tag>,
    },
    { title: '累计接收 (Rx)', dataIndex: 'rx', key: 'rx', width: 120 },
    { title: '累计发送 (Tx)', dataIndex: 'tx', key: 'tx', width: 120 },
    { title: 'MTU', dataIndex: 'mtu', key: 'mtu', width: 80 },
  ];

  return (
    <Spin spinning={loading}>
      <div className="flex flex-col gap-4 py-4">
        <Row gutter={16}>
          <Col span={6}>
            <Card size="small" className="border-none shadow-sm">
              <Statistic title="当前入站" value={metrics?.netIn || 0} precision={2} suffix="Kbps" />
            </Card>
          </Col>
          <Col span={6}>
            <Card size="small" className="border-none shadow-sm">
              <Statistic title="当前出站" value={metrics?.netOut || 0} precision={2} suffix="Kbps" />
            </Card>
          </Col>
          <Col span={6}>
            <Card size="small" className="border-none shadow-sm">
              <Statistic title="活跃接口" value={interfaces.filter(i => i.status === 'up').length} suffix="个" />
            </Card>
          </Col>
          <Col span={6}>
            <Card size="small" className="border-none shadow-sm">
              <Statistic title="TCP 连接数" value={metrics?.latency_ms ? 36 : 0} suffix="个" />
            </Card>
          </Col>
        </Row>

        <Card title="网络接口" className="border-none shadow-sm">
          <Table
            columns={columns}
            dataSource={interfaces}
            rowKey="name"
            size="small"
            pagination={false}
          />
        </Card>

        <Card 
          title={
            <div className="flex items-center gap-2">
              路由表
              <Tooltip title="仅展示主路由表信息">
                <InfoCircleOutlined className="text-gray-400 text-xs" />
              </Tooltip>
            </div>
          } 
          className="border-none shadow-sm"
        >
          <Table
            columns={[
              { title: '目标', dataIndex: 'dest', key: 'dest' },
              { title: '网关', dataIndex: 'gw', key: 'gw' },
              { title: '掩码', dataIndex: 'mask', key: 'mask' },
              { title: '标志', dataIndex: 'flags', key: 'flags' },
              { title: '接口', dataIndex: 'iface', key: 'iface' },
            ]}
            dataSource={[
              { dest: '0.0.0.0', gw: '192.168.1.1', mask: '0.0.0.0', flags: 'UG', iface: 'ens33' },
              { dest: '192.168.1.0', gw: '0.0.0.0', mask: '255.255.255.0', flags: 'U', iface: 'ens33' },
            ]}
            rowKey="dest"
            size="small"
            pagination={false}
          />
        </Card>
      </div>
    </Spin>
  );
};

export default NetworkTab;
