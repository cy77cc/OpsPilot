import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Space, Table, Tag, Typography, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { Link, useParams } from 'react-router-dom';
import { ReloadOutlined } from '@ant-design/icons';
import { Api } from '../../../api';
import type { Cluster, ClusterNode } from '../../../api/modules/cluster';

const { Text, Title } = Typography;

const ClusterNodesPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const clusterId = Number(id);
  const [cluster, setCluster] = useState<Cluster | null>(null);
  const [nodes, setNodes] = useState<ClusterNode[]>([]);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    if (!clusterId) return;
    setLoading(true);
    try {
      const [clusterRes, nodeRes] = await Promise.all([
        Api.cluster.getClusterDetail(clusterId),
        Api.cluster.getClusterNodes(clusterId),
      ]);
      setCluster(clusterRes.data);
      setNodes(nodeRes.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载节点与容量失败');
      setNodes([]);
    } finally {
      setLoading(false);
    }
  }, [clusterId]);

  useEffect(() => {
    load();
  }, [load]);

  const columns: ColumnsType<ClusterNode> = useMemo(() => [
    { title: '节点', dataIndex: 'name', key: 'name' },
    { title: 'IP', dataIndex: 'ip', key: 'ip' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (value: string) => <Tag color={value === 'ready' ? 'green' : 'orange'}>{value}</Tag>,
    },
    { title: '角色', dataIndex: 'role', key: 'role' },
    { title: 'CPU', dataIndex: 'allocatable_cpu', key: 'allocatable_cpu', render: (v?: string) => v || '-' },
    { title: '内存', dataIndex: 'allocatable_mem', key: 'allocatable_mem', render: (v?: string) => v || '-' },
    { title: 'Kubelet', dataIndex: 'kubelet_version', key: 'kubelet_version', render: (v?: string) => v || '-' },
  ], []);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card>
        <Space style={{ width: '100%', justifyContent: 'space-between' }} align="start">
          <div>
            <Title level={4} style={{ marginBottom: 4 }}>节点与容量</Title>
            <Text type="secondary">{cluster?.name || `#${clusterId}`}</Text>
          </div>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>刷新</Button>
            <Link to={`/deployment/infrastructure/clusters/${clusterId}/operations`}>进入操作中心</Link>
          </Space>
        </Space>
      </Card>
      <Card>
        <Table<ClusterNode>
          size="small"
          rowKey="name"
          columns={columns}
          dataSource={nodes}
          loading={loading}
          pagination={false}
        />
      </Card>
    </Space>
  );
};

export default ClusterNodesPage;
