import React, { useState, useEffect, useMemo } from 'react';
import { Card, Button, Space, Select, Empty, message, Tag, Modal } from 'antd';
import { ReloadOutlined, FullscreenOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../../api';
import type { TopologyService } from '../../../api/modules/deployment';

const DeploymentTopologyPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [services, setServices] = useState<TopologyService[]>([]);
  const [envFilter, setEnvFilter] = useState<string>('all');
  const [selectedNode, setSelectedNode] = useState<TopologyService | null>(null);
  const [detailModalVisible, setDetailModalVisible] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const res = await Api.deployment.getTopology({
        environment: envFilter !== 'all' ? envFilter : undefined,
      });
      setServices(res.data.services || []);
    } catch (err) {
      message.error('加载拓扑数据失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [envFilter]);

  const groupedByEnv = useMemo(() => {
    const groups: Record<string, TopologyService[]> = {};
    services.forEach((s) => {
      if (!groups[s.environment]) {
        groups[s.environment] = [];
      }
      groups[s.environment].push(s);
    });
    return groups;
  }, [services]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'applied':
      case 'ready':
      case 'healthy':
        return '#10b981';
      case 'degraded':
      case 'applying':
        return '#f59e0b';
      case 'failed':
      case 'down':
        return '#ef4444';
      default:
        return '#6c757d';
    }
  };

  const getStatusLabel = (status: string) => {
    const labels: Record<string, string> = {
      applied: '已部署',
      ready: '就绪',
      healthy: '健康',
      degraded: '降级',
      applying: '部署中',
      failed: '失败',
      down: '宕机',
      no_deployments: '未部署',
      unknown: '未知',
    };
    return labels[status] || status;
  };

  const getEnvLabel = (env: string) => {
    const labels: Record<string, string> = {
      production: '生产环境',
      staging: '预发布',
      development: '开发环境',
    };
    return labels[env] || env;
  };

  const getEnvColor = (env: string) => {
    switch (env) {
      case 'production':
        return 'red';
      case 'staging':
        return 'orange';
      case 'development':
        return 'blue';
      default:
        return 'default';
    }
  };

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">部署拓扑</h1>
          <p className="text-sm text-gray-500 mt-1">可视化服务部署状态和依赖关系</p>
        </div>
        <Space>
          <Select
            value={envFilter}
            style={{ width: 140 }}
            options={[
              { value: 'all', label: '全部环境' },
              { value: 'production', label: '生产环境' },
              { value: 'staging', label: '预发布' },
              { value: 'development', label: '开发环境' },
            ]}
            onChange={(v) => {
              setEnvFilter(v);
            }}
          />
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
        </Space>
      </div>

      {/* Topology visualization */}
      {Object.keys(groupedByEnv).length > 0 ? (
        Object.entries(groupedByEnv).map(([env, envServices]) => (
          <Card
            key={env}
            title={
              <Space>
                <Tag color={getEnvColor(env)}>{getEnvLabel(env)}</Tag>
                <span className="text-gray-600">({envServices.length} 个服务)</span>
              </Space>
            }
          >
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
              {envServices.map((service) => (
                <Card
                  key={service.id}
                  className="cursor-pointer hover:shadow-md transition-shadow"
                  onClick={() => {
                    setSelectedNode(service);
                    setDetailModalVisible(true);
                  }}
                >
                  <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <span className="font-semibold text-gray-900">{service.name}</span>
                      <Tag color={getStatusColor(service.status)}>
                        {getStatusLabel(service.status)}
                      </Tag>
                    </div>
                    <div className="text-xs text-gray-500">
                      {service.runtime_type === 'k8s' ? 'Kubernetes' : 'Docker Compose'}
                    </div>
                    {service.last_deployment && (
                      <div className="text-xs text-gray-500">
                        最近部署: {new Date(service.last_deployment).toLocaleString()}
                      </div>
                    )}
                  </div>
                </Card>
              ))}
            </div>
          </Card>
        ))
      ) : (
        <Card>
          <Empty description="暂无部署拓扑数据" />
        </Card>
      )}

      {/* Detail modal */}
      <Modal
        title="服务详情"
        open={detailModalVisible}
        onCancel={() => {
          setDetailModalVisible(false);
          setSelectedNode(null);
        }}
        footer={[
          <Button key="close" onClick={() => setDetailModalVisible(false)}>
            关闭
          </Button>,
          <Button
            key="detail"
            type="primary"
            onClick={() => {
              if (selectedNode) {
                navigate(`/deployment/targets/${selectedNode.target_id}`);
              }
            }}
          >
            查看目标详情
          </Button>,
        ]}
        width={600}
      >
        {selectedNode && (
          <div className="space-y-4">
            <div>
              <div className="text-sm font-semibold mb-1">服务名称:</div>
              <div>{selectedNode.name}</div>
            </div>
            <div>
              <div className="text-sm font-semibold mb-1">环境:</div>
              <Tag color={getEnvColor(selectedNode.environment)}>
                {getEnvLabel(selectedNode.environment)}
              </Tag>
            </div>
            <div>
              <div className="text-sm font-semibold mb-1">状态:</div>
              <Tag color={getStatusColor(selectedNode.status)}>
                {getStatusLabel(selectedNode.status)}
              </Tag>
            </div>
            <div>
              <div className="text-sm font-semibold mb-1">运行时类型:</div>
              <div>{selectedNode.runtime_type === 'k8s' ? 'Kubernetes' : 'Docker Compose'}</div>
            </div>
            {selectedNode.last_deployment && (
              <div>
                <div className="text-sm font-semibold mb-1">最近部署:</div>
                <div>{new Date(selectedNode.last_deployment).toLocaleString()}</div>
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default DeploymentTopologyPage;
