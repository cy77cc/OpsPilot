import React, { useState, useEffect } from 'react';
import { Card, Button, Space, Descriptions, Tag, Table, Empty, message, Tabs } from 'antd';
import {
  ArrowLeftOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  SyncOutlined,
  ExclamationCircleOutlined,
  CloudServerOutlined,
  HistoryOutlined,
} from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { Api } from '../../../api';
import type { DeployTarget } from '../../../api/modules/deployment';

const DeploymentTargetDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [loading, setLoading] = useState(false);
  const [target, setTarget] = useState<DeployTarget | null>(null);
  const [deployHistory, setDeployHistory] = useState<any[]>([]);

  const load = async () => {
    if (!id) return;
    setLoading(true);
    try {
      const res = await Api.deployment.getTargetDetail(Number(id));
      setTarget(res.data);

      // Load deployment history
      const historyRes = await Api.deployment.getReleases({ target_id: Number(id) });
      setDeployHistory(historyRes.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [id]);

  const getReadinessConfig = (status: string) => {
    const configs: Record<string, { icon: React.ReactNode; color: string; text: string }> = {
      ready: { icon: <CheckCircleOutlined />, color: 'success', text: '就绪' },
      not_ready: { icon: <CloseCircleOutlined />, color: 'error', text: '未就绪' },
      bootstrapping: { icon: <SyncOutlined spin />, color: 'processing', text: '初始化中' },
    };
    return configs[status] || { icon: <ExclamationCircleOutlined />, color: 'default', text: status };
  };

  const historyColumns = [
    {
      title: '发布 ID',
      dataIndex: 'id',
      key: 'id',
      render: (id: number) => <a onClick={() => navigate(`/deployment/${id}`)}>{id}</a>,
    },
    {
      title: '服务',
      dataIndex: 'service_name',
      key: 'service_name',
    },
    {
      title: '状态',
      dataIndex: 'state',
      key: 'state',
      render: (state: string) => {
        const colors: Record<string, string> = {
          pending_approval: 'orange',
          approved: 'blue',
          applying: 'processing',
          applied: 'success',
          failed: 'error',
          rejected: 'default',
        };
        return <Tag color={colors[state] || 'default'}>{state}</Tag>;
      },
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time: string) => new Date(time).toLocaleString(),
    },
  ];

  if (!target) {
    return (
      <Card>
        <div className="text-center py-12">
          <ReloadOutlined spin className="text-4xl text-primary-500 mb-4" />
          <p className="text-gray-500">加载中...</p>
        </div>
      </Card>
    );
  }

  const readinessConfig = getReadinessConfig(target.readiness_status);

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/deployment/targets')}>
            返回
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <CloudServerOutlined className="text-2xl text-blue-500" />
              <h1 className="text-2xl font-semibold text-gray-900">{target.name}</h1>
              <Tag color={readinessConfig.color} icon={readinessConfig.icon}>
                {readinessConfig.text}
              </Tag>
            </div>
            <p className="text-sm text-gray-500 mt-1">部署目标详情</p>
          </div>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
        </Space>
      </div>

      {/* Target info */}
      <Card title="基本信息">
        <Descriptions column={2} bordered>
          <Descriptions.Item label="目标名称">{target.name}</Descriptions.Item>
          <Descriptions.Item label="环境">
            <Tag color={target.environment === 'production' ? 'red' : target.environment === 'staging' ? 'orange' : 'blue'}>
              {target.environment}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="运行时类型">
            <Tag color={target.runtime_type === 'k8s' ? 'blue' : 'green'}>
              {target.runtime_type === 'k8s' ? 'Kubernetes' : 'Docker Compose'}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="就绪状态">
            <Tag color={readinessConfig.color} icon={readinessConfig.icon}>
              {readinessConfig.text}
            </Tag>
          </Descriptions.Item>
          {target.cluster_name && (
            <Descriptions.Item label="集群名称">{target.cluster_name}</Descriptions.Item>
          )}
          {target.namespace && (
            <Descriptions.Item label="命名空间">{target.namespace}</Descriptions.Item>
          )}
          {target.credential_id && (
            <Descriptions.Item label="凭证 ID">{target.credential_id}</Descriptions.Item>
          )}
          <Descriptions.Item label="创建时间">
            {new Date(target.created_at).toLocaleString()}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* Tabs for nodes and deployment history */}
      <Card>
        <Tabs
          items={[
            {
              key: 'nodes',
              label: (
                <span>
                  <CloudServerOutlined /> 节点信息
                </span>
              ),
              children: target.nodes && target.nodes.length > 0 ? (
                <Table
                  dataSource={target.nodes}
                  rowKey="name"
                  pagination={false}
                  columns={[
                    { title: '节点名称', dataIndex: 'name', key: 'name' },
                    { title: 'IP 地址', dataIndex: 'ip', key: 'ip' },
                    {
                      title: '状态',
                      dataIndex: 'status',
                      key: 'status',
                      render: (status: string) => (
                        <Tag color={status === 'ready' ? 'success' : 'default'}>{status}</Tag>
                      ),
                    },
                    { title: 'CPU', dataIndex: 'cpu', key: 'cpu' },
                    { title: '内存', dataIndex: 'memory', key: 'memory' },
                  ]}
                />
              ) : (
                <Empty description="暂无节点信息" />
              ),
            },
            {
              key: 'history',
              label: (
                <span>
                  <HistoryOutlined /> 部署历史
                </span>
              ),
              children: deployHistory.length > 0 ? (
                <Table
                  dataSource={deployHistory}
                  rowKey="id"
                  columns={historyColumns}
                  pagination={{ pageSize: 10 }}
                />
              ) : (
                <Empty description="暂无部署历史" />
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
};

export default DeploymentTargetDetailPage;
