import React, { useState, useCallback, useEffect } from 'react';
import { Button, Card, Row, Col, Tag, Space, message, Modal, Dropdown, Popconfirm } from 'antd';
import type { MenuProps } from 'antd';
import {
  PlusOutlined, ReloadOutlined, ClusterOutlined, CheckCircleOutlined,
  ExclamationCircleOutlined, MoreOutlined, ImportOutlined,
  DeleteOutlined, EditOutlined, ApiOutlined
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../../api';
import type { Cluster } from '../../../api/modules/cluster';

const ClusterListPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [clusters, setClusters] = useState<Cluster[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await Api.cluster.getClusters();
      setClusters(res.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载集群列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const handleDelete = async (id: number) => {
    try {
      await Api.cluster.deleteCluster(id);
      message.success('集群已删除');
      load();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '删除失败');
    }
  };

  const handleTest = async (id: number) => {
    try {
      const res = await Api.cluster.testCluster(id);
      if (res.data.connected) {
        message.success(`连接成功 (${res.data.latency_ms}ms)`);
      } else {
        message.error(`连接失败: ${res.data.message}`);
      }
    } catch (err) {
      message.error(err instanceof Error ? err.message : '测试连接失败');
    }
  };

  const getStatusColor = (status: string) => {
    const statusMap: Record<string, string> = {
      active: 'success',
      inactive: 'default',
      error: 'error',
      provisioning: 'processing',
    };
    return statusMap[status] || 'default';
  };

  const getSourceColor = (source: string) => {
    return source === 'platform_managed' ? 'blue' : 'purple';
  };

  const getStatusIcon = (status: string) => {
    return status === 'active' ? (
      <CheckCircleOutlined style={{ color: '#52c41a' }} />
    ) : (
      <ExclamationCircleOutlined style={{ color: '#faad14' }} />
    );
  };

  const getClusterMenuItems = (cluster: Cluster): MenuProps['items'] => [
    {
      key: 'detail',
      icon: <ClusterOutlined />,
      label: '查看详情',
      onClick: () => navigate(`/deployment/infrastructure/clusters/${cluster.id}`),
    },
    {
      key: 'test',
      icon: <ApiOutlined />,
      label: '测试连接',
      onClick: () => handleTest(cluster.id),
    },
    { type: 'divider' },
    {
      key: 'delete',
      icon: <DeleteOutlined />,
      danger: true,
      label: (
        <Popconfirm
          title="确定删除此集群？"
          description="删除后将无法恢复，请谨慎操作"
          onConfirm={() => handleDelete(cluster.id)}
          okText="确定"
          cancelText="取消"
        >
          <span>删除集群</span>
        </Popconfirm>
      ),
    },
  ];

  const createMenuItems: MenuProps['items'] = [
    {
      key: 'bootstrap',
      icon: <PlusOutlined />,
      label: '创建集群',
      onClick: () => navigate('/deployment/infrastructure/clusters/bootstrap'),
    },
    {
      key: 'import',
      icon: <ImportOutlined />,
      label: '导入集群',
      onClick: () => navigate('/deployment/infrastructure/clusters/import'),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">集群管理</h1>
          <p className="text-sm text-gray-500 mt-1">管理 Kubernetes 集群，支持自建和导入</p>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
          <Dropdown menu={{ items: createMenuItems }} placement="bottomRight">
            <Button type="primary" icon={<PlusOutlined />}>
              添加集群
            </Button>
          </Dropdown>
        </Space>
      </div>

      <Row gutter={[16, 16]}>
        {clusters.map((cluster) => (
          <Col xs={24} sm={12} lg={8} xl={6} key={cluster.id}>
            <Card
              hoverable
              className="h-full"
              onClick={() => navigate(`/deployment/infrastructure/clusters/${cluster.id}`)}
            >
              <div className="flex flex-col h-full">
                <div className="flex items-center justify-between mb-3">
                  <Space>
                    <ClusterOutlined className="text-2xl text-blue-500" />
                    <span className="text-lg font-semibold truncate max-w-[150px]">{cluster.name}</span>
                  </Space>
                  <Dropdown menu={{ items: getClusterMenuItems(cluster) }} trigger={['click']} placement="bottomRight">
                    <Button type="text" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
                  </Dropdown>
                </div>

                <div className="space-y-2 flex-1">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-500">状态:</span>
                    <Space>
                      {getStatusIcon(cluster.status)}
                      <Tag color={getStatusColor(cluster.status)}>{cluster.status}</Tag>
                    </Space>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-500">来源:</span>
                    <Tag color={getSourceColor(cluster.source)}>
                      {cluster.source === 'platform_managed' ? '平台托管' : '外部导入'}
                    </Tag>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-500">版本:</span>
                    <span className="text-sm">{cluster.k8s_version || cluster.version || '-'}</span>
                  </div>
                  {cluster.node_count !== undefined && (
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-gray-500">节点数:</span>
                      <span className="font-semibold">{cluster.node_count}</span>
                    </div>
                  )}
                </div>

                <div className="pt-3 mt-3 border-t border-gray-200">
                  <div className="text-xs text-gray-500 truncate">
                    {cluster.endpoint || '-'}
                  </div>
                  <div className="text-xs text-gray-400 mt-1">
                    创建于: {new Date(cluster.created_at).toLocaleDateString()}
                  </div>
                </div>
              </div>
            </Card>
          </Col>
        ))}
      </Row>

      {clusters.length === 0 && !loading && (
        <Card className="text-center py-16">
          <ClusterOutlined className="text-6xl text-gray-300 mb-4" />
          <p className="text-gray-500 mb-4">暂无集群</p>
          <Space>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => navigate('/deployment/infrastructure/clusters/bootstrap')}
            >
              创建集群
            </Button>
            <Button
              icon={<ImportOutlined />}
              onClick={() => navigate('/deployment/infrastructure/clusters/import')}
            >
              导入集群
            </Button>
          </Space>
        </Card>
      )}
    </div>
  );
};

export default ClusterListPage;
