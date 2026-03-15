import React from 'react';
import { Card, Progress, Select, Empty, Typography } from 'antd';
import {
  DashboardOutlined,
  DatabaseOutlined,
  CloudOutlined,
} from '@ant-design/icons';
import type { ClusterResource } from '../../api/modules/dashboard';

const { Text } = Typography;

interface Props {
  data: ClusterResource[];
  loading?: boolean;
}

const ClusterResourceCard: React.FC<Props> = ({ data, loading }) => {
  const [selectedCluster, setSelectedCluster] = React.useState<number | null>(
    data.length > 0 ? data[0].clusterId : null
  );

  // Update selected cluster when data changes
  React.useEffect(() => {
    if (data.length > 0 && !data.find(c => c.clusterId === selectedCluster)) {
      setSelectedCluster(data[0].clusterId);
    }
  }, [data, selectedCluster]);

  const cluster = data.find((c) => c.clusterId === selectedCluster);

  if (data.length === 0 && !loading) {
    return (
      <Card title="集群资源" size="small">
        <Empty description="暂无集群数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      </Card>
    );
  }

  return (
    <Card
      title="集群资源"
      size="small"
      extra={
        data.length > 1 && (
          <Select
            value={selectedCluster}
            onChange={setSelectedCluster}
            style={{ width: 120 }}
            size="small"
            options={data.map((c) => ({
              label: c.clusterName,
              value: c.clusterId,
            }))}
          />
        )
      }
      loading={loading}
    >
      {cluster && (
        <div className="space-y-4">
          {/* CPU 使用率 */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <DashboardOutlined className="text-blue-500" />
                <Text strong>CPU</Text>
              </div>
              <Text type="secondary" className="text-xs">
                {cluster.cpu.usagePercent.toFixed(1)}%
              </Text>
            </div>
            <Progress
              percent={cluster.cpu.usagePercent}
              size="small"
              format={() => `${cluster.cpu.usage.toFixed(1)} / ${cluster.cpu.allocatable.toFixed(1)} 核`}
            />
            <div className="flex justify-between mt-1">
              <Text type="secondary" className="text-xs">
                已请求: {cluster.cpu.requested.toFixed(1)} 核
              </Text>
            </div>
          </div>

          {/* 内存使用率 */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <DatabaseOutlined className="text-green-500" />
                <Text strong>内存</Text>
              </div>
              <Text type="secondary" className="text-xs">
                {cluster.memory.usagePercent.toFixed(1)}%
              </Text>
            </div>
            <Progress
              percent={cluster.memory.usagePercent}
              size="small"
              strokeColor="#52c41a"
              format={() => `${(cluster.memory.usage / 1024).toFixed(1)} / ${(cluster.memory.allocatable / 1024).toFixed(1)} GB`}
            />
            <div className="flex justify-between mt-1">
              <Text type="secondary" className="text-xs">
                已请求: {(cluster.memory.requested / 1024).toFixed(1)} GB
              </Text>
            </div>
          </div>

          {/* Pod 统计 */}
          <div className="pt-2 border-t border-gray-100">
            <div className="flex items-center gap-2 mb-2">
              <CloudOutlined className="text-purple-500" />
              <Text strong>Pod</Text>
            </div>
            <div className="grid grid-cols-4 gap-2 text-center">
              <div>
                <Text strong className="text-lg">{cluster.pods.total}</Text>
                <br />
                <Text type="secondary" className="text-xs">总数</Text>
              </div>
              <div>
                <Text strong className="text-lg text-green-500">{cluster.pods.running}</Text>
                <br />
                <Text type="secondary" className="text-xs">运行中</Text>
              </div>
              <div>
                <Text strong className="text-lg text-yellow-500">{cluster.pods.pending}</Text>
                <br />
                <Text type="secondary" className="text-xs">等待中</Text>
              </div>
              <div>
                <Text strong className="text-lg text-red-500">{cluster.pods.failed}</Text>
                <br />
                <Text type="secondary" className="text-xs">失败</Text>
              </div>
            </div>
          </div>
        </div>
      )}
    </Card>
  );
};

export default ClusterResourceCard;
