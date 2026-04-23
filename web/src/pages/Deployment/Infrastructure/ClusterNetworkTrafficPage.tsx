import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Alert, Card, Select, Space, Table, Typography, message } from 'antd';
import { useParams } from 'react-router-dom';
import { Api } from '../../../api';
import type { Cluster, NamespaceInfo, ServiceInfo, IngressInfo } from '../../../api/modules/cluster';

const { Title, Text } = Typography;

const ClusterNetworkTrafficPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const clusterId = Number(id);
  const [cluster, setCluster] = useState<Cluster | null>(null);
  const [namespaces, setNamespaces] = useState<NamespaceInfo[]>([]);
  const [namespace, setNamespace] = useState('default');
  const [services, setServices] = useState<ServiceInfo[]>([]);
  const [ingresses, setIngresses] = useState<IngressInfo[]>([]);
  const [loading, setLoading] = useState(false);

  const loadMeta = useCallback(async () => {
    if (!clusterId) {return;}
    try {
      const [clusterRes, nsRes] = await Promise.all([
        Api.cluster.getClusterDetail(clusterId),
        Api.cluster.getNamespaces(clusterId),
      ]);
      setCluster(clusterRes.data);
      const nsList = nsRes.data.list || [];
      setNamespaces(nsList);
      if (nsList.length > 0) {
        setNamespace((prev) => (nsList.some((item) => item.name === prev) ? prev : nsList[0].name));
      }
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载网络元信息失败');
    }
  }, [clusterId]);

  const loadTraffic = useCallback(async (ns: string) => {
    if (!clusterId) {return;}
    setLoading(true);
    try {
      const [svcRes, ingRes] = await Promise.all([
        Api.cluster.getServices(clusterId, ns),
        Api.cluster.getIngresses(clusterId, ns),
      ]);
      setServices(svcRes.data.list || []);
      setIngresses(ingRes.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载网络流量数据失败');
      setServices([]);
      setIngresses([]);
    } finally {
      setLoading(false);
    }
  }, [clusterId]);

  useEffect(() => {
    loadMeta();
  }, [loadMeta]);

  useEffect(() => {
    if (!namespace) {return;}
    loadTraffic(namespace);
  }, [namespace, loadTraffic]);

  const namespaceOptions = useMemo(
    () => namespaces.map((item) => ({ label: item.name, value: item.name })),
    [namespaces],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card>
        <Title level={4} style={{ marginBottom: 4 }}>网络与流量</Title>
        <Text type="secondary">{cluster?.name || `#${clusterId}`}</Text>
      </Card>
      <Alert
        showIcon
        type="info"
        message="Gateway API 优先，Ingress 兼容"
        description="本页优先展示 Gateway API 治理入口，Ingress 保持兼容视图用于渐进迁移。"
      />
      <Card>
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Select
            aria-label="Namespace"
            value={namespace}
            options={namespaceOptions}
            onChange={setNamespace}
            style={{ width: 260 }}
            placeholder="Namespace"
          />
          <Card size="small" title="Gateway API">
            <Text type="secondary">Gateway / HTTPRoute 入口将在 Phase 2 网络治理任务中接入。</Text>
          </Card>
          <Card size="small" title="Services">
            <Table<ServiceInfo>
              size="small"
              rowKey="name"
              loading={loading}
              pagination={false}
              dataSource={services}
              columns={[
                { title: '名称', dataIndex: 'name', key: 'name' },
                { title: '类型', dataIndex: 'type', key: 'type' },
                { title: 'ClusterIP', dataIndex: 'cluster_ip', key: 'cluster_ip' },
                { title: 'Age', dataIndex: 'age', key: 'age' },
              ]}
            />
          </Card>
          <Card size="small" title="Ingresses (兼容)">
            <Table<IngressInfo>
              size="small"
              rowKey="name"
              loading={loading}
              pagination={false}
              dataSource={ingresses}
              columns={[
                { title: '名称', dataIndex: 'name', key: 'name' },
                { title: 'Class', dataIndex: 'ingress_class_name', key: 'ingress_class_name' },
                {
                  title: 'Hosts',
                  key: 'hosts',
                  render: (_, record) => record.hosts?.map((item) => item.host).join(', ') || '-',
                },
                { title: 'Age', dataIndex: 'age', key: 'age' },
              ]}
            />
          </Card>
        </Space>
      </Card>
    </Space>
  );
};

export default ClusterNetworkTrafficPage;
