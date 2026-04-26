import React from 'react';
import { Card, Row, Col, Progress, Statistic } from 'antd';
import type { Host } from '../../../../api/modules/hosts';

interface HostResourceSummaryCardProps {
  host: Host | null;
  loading?: boolean;
}

const HostResourceSummaryCard: React.FC<HostResourceSummaryCardProps> = ({ host, loading }) => {
  return (
    <Card title="资源使用率" loading={loading} className="h-full">
      <Row gutter={[16, 24]}>
        {/* Row 1: Ring Charts */}
        <Col span={8} className="flex flex-col items-center">
          <Progress 
            type="circle" 
            percent={host?.cpuUsagePct || 23} 
            size={80} 
            strokeColor={host?.cpuUsagePct && host.cpuUsagePct > 80 ? '#ff4d4f' : '#1890ff'}
          />
          <div className="mt-2 text-center">
            <div className="text-gray-400 text-xs">CPU 使用率</div>
            <div className="text-sm font-medium">0.92 / {host?.cpu || 4} Core</div>
          </div>
        </Col>
        <Col span={8} className="flex flex-col items-center">
          <Progress 
            type="circle" 
            percent={host?.memoryUsagePct || 48} 
            size={80}
            strokeColor={host?.memoryUsagePct && host.memoryUsagePct > 80 ? '#ff4d4f' : '#52c41a'}
          />
          <div className="mt-2 text-center">
            <div className="text-gray-400 text-xs">内存使用率</div>
            <div className="text-sm font-medium">1.92 / {host?.memory ? (host.memory / 1024).toFixed(1) : 4} GB</div>
          </div>
        </Col>
        <Col span={8} className="flex flex-col items-center">
          <Progress 
            type="circle" 
            percent={host?.diskUsagePct || 35} 
            size={80}
            strokeColor={host?.diskUsagePct && host.diskUsagePct > 80 ? '#ff4d4f' : '#faad14'}
          />
          <div className="mt-2 text-center">
            <div className="text-gray-400 text-xs">磁盘使用率</div>
            <div className="text-sm font-medium">80 / {host?.disk || 200} GB</div>
          </div>
        </Col>

        {/* Row 2: Secondary Metrics */}
        <Col span={8}>
          <Statistic 
            title={<span className="text-xs text-gray-400">负载 (1/5/15min)</span>} 
            value="0.32 / 0.28 / 0.25"
            valueStyle={{ fontSize: '16px', fontWeight: 500 }}
          />
        </Col>
        <Col span={8}>
          <Statistic 
            title={<span className="text-xs text-gray-400">运行进程数</span>} 
            value={128}
            valueStyle={{ fontSize: '16px', fontWeight: 500 }}
          />
        </Col>
        <Col span={8}>
          <Statistic 
            title={<span className="text-xs text-gray-400">当前连接数</span>} 
            value={36}
            valueStyle={{ fontSize: '16px', fontWeight: 500 }}
          />
        </Col>
      </Row>
    </Card>
  );
};

export default HostResourceSummaryCard;
