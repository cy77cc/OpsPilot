import React from 'react';
import { Card, Progress, Row, Col, Typography, Space } from 'antd';
import { ArrowRightOutlined } from '@ant-design/icons';
import type { Host } from '../../../../api/modules/hosts';

const { Text, Title } = Typography;

interface HostDiskInfoCardProps {
  host: Host | null;
  loading?: boolean;
  onViewDetails?: () => void;
}

const HostDiskInfoCard: React.FC<HostDiskInfoCardProps> = ({ host, loading, onViewDetails }) => {
  return (
    <Card 
      title="磁盘信息" 
      loading={loading} 
      className="h-full"
      extra={
        <a onClick={onViewDetails} className="text-xs text-blue-600 hover:text-blue-800 flex items-center gap-1">
          查看磁盘详情 <ArrowRightOutlined />
        </a>
      }
    >
      <Row gutter={16} align="middle">
        <Col span={10}>
          <Progress
            type="circle"
            percent={40}
            size={100}
            strokeWidth={10}
            format={() => (
              <div className="flex flex-col items-center">
                <Text type="secondary" style={{ fontSize: '12px' }}>已使用</Text>
                <Text strong style={{ fontSize: '18px' }}>40%</Text>
              </div>
            )}
            strokeColor={{
              '0%': '#108ee9',
              '100%': '#87d068',
            }}
          />
        </Col>
        <Col span={14}>
          <div className="mb-4">
            <Text type="secondary" className="text-xs uppercase">总容量</Text>
            <Title level={4} style={{ margin: 0 }}>{host?.disk || 200} GB</Title>
          </div>
          <Space direction="vertical" className="w-full" size="small">
            <div className="flex justify-between items-center text-xs">
              <Space size="small">
                <div className="w-2 h-2 rounded-full bg-blue-500" />
                <Text>已使用</Text>
              </Space>
              <Text strong>80 GB (40%)</Text>
            </div>
            <div className="flex justify-between items-center text-xs">
              <Space size="small">
                <div className="w-2 h-2 rounded-full bg-gray-200" />
                <Text>可用</Text>
              </Space>
              <Text strong>120 GB (60%)</Text>
            </div>
          </Space>
        </Col>
      </Row>
      
      <div className="mt-6">
        <Text type="secondary" className="text-xs mb-2 block">主要挂载点</Text>
        <div className="flex flex-col gap-3">
          <div>
            <div className="flex justify-between items-center text-xs mb-1">
              <Text>/</Text>
              <Text type="secondary">80 GB / 200 GB (40%)</Text>
            </div>
            <Progress percent={40} size="small" showInfo={false} strokeColor="#1890ff" />
          </div>
        </div>
      </div>
    </Card>
  );
};

export default HostDiskInfoCard;
