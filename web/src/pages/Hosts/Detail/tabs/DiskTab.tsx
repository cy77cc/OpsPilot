import React, { useEffect, useState } from 'react';
import { Card, Table, Progress, Row, Col, Statistic, Tag, Spin } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { hostApi } from '../../../../api/modules/hosts';

interface PartitionItem {
  filesystem: string;
  type: string;
  size: string;
  used: string;
  available: string;
  usagePct: number;
  mounted: string;
}

const DiskTab: React.FC<{ hostId: string }> = ({ hostId }) => {
  const [loading, setLoading] = useState(false);
  const [partitions, setPartitions] = useState<PartitionItem[]>([]);
  const [metrics, setMetrics] = useState<any>(null);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        const [diskRes, metricRes] = await Promise.all([
          hostApi.getHostDisks(hostId),
          hostApi.getHostMetrics(hostId),
        ]);
        setPartitions(diskRes.data || []);
        if (metricRes.data && metricRes.data.length > 0) {
          setMetrics(metricRes.data[metricRes.data.length - 1]);
        }
      } finally {
        setLoading(false);
      }
    };
    if (hostId) fetchData();
  }, [hostId]);

  const columns: ColumnsType<PartitionItem> = [
    { title: '文件系统', dataIndex: 'filesystem', key: 'filesystem', width: 150 },
    { title: '类型', dataIndex: 'type', key: 'type', width: 80, render: (type) => <Tag>{type}</Tag> },
    { title: '容量', dataIndex: 'size', key: 'size', width: 100 },
    { title: '已用', dataIndex: 'used', key: 'used', width: 100 },
    { title: '可用', dataIndex: 'available', key: 'available', width: 100 },
    {
      title: '使用率',
      dataIndex: 'usagePct',
      key: 'usagePct',
      width: 250,
      render: (pct) => (
        <div className="flex items-center gap-2">
          <Progress 
            percent={pct} 
            size="small" 
            status={pct > 90 ? 'exception' : pct > 70 ? 'active' : 'normal'}
            strokeColor={pct > 90 ? '#ff4d4f' : pct > 70 ? '#faad14' : '#1890ff'}
          />
        </div>
      ),
    },
    { title: '挂载点', dataIndex: 'mounted', key: 'mounted' },
  ];

  return (
    <Spin spinning={loading}>
      <div className="flex flex-col gap-4 py-4">
        <Row gutter={16}>
          <Col span={6}>
            <Card size="small" className="border-none shadow-sm">
              <Statistic title="总磁盘数" value={partitions.length} suffix="个" />
            </Card>
          </Col>
          <Col span={6}>
            <Card size="small" className="border-none shadow-sm">
              <Statistic title="已用分区" value={partitions.filter(p => p.usagePct > 80).length} suffix="个" valueStyle={{ color: '#faad14' }} />
            </Card>
          </Col>
          <Col span={6}>
            <Card size="small" className="border-none shadow-sm">
              <Statistic title="主要分区使用" value={partitions.length > 0 ? partitions[0].usagePct : 0} suffix="%" />
            </Card>
          </Col>
          <Col span={6}>
            <Card size="small" className="border-none shadow-sm">
              <Statistic title="健康状态" value="良好" valueStyle={{ color: '#52c41a' }} />
            </Card>
          </Col>
        </Row>

        <Card title="磁盘分区详情" className="border-none shadow-sm">
          <Table
            columns={columns}
            dataSource={partitions}
            rowKey="mounted"
            size="small"
            pagination={false}
          />
        </Card>

        <Card title="磁盘 I/O 状态" className="border-none shadow-sm">
          <Row gutter={16}>
            <Col span={8}>
              <Statistic title="读取速度" value={metrics?.diskIo || 0} precision={2} suffix="MB/s" />
            </Col>
            <Col span={8}>
              <Statistic title="写入速度" value={0} precision={2} suffix="MB/s" />
            </Col>
            <Col span={8}>
              <Statistic title="延迟" value={metrics?.latency_ms || 0} precision={2} suffix="ms" />
            </Col>
          </Row>
        </Card>
      </div>
    </Spin>
  );
};

export default DiskTab;
