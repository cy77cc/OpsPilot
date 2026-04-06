import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Card, Select, Space, Table, Tabs, Typography, message } from 'antd';
import { useParams } from 'react-router-dom';
import { Api } from '../../../api';
import type {
  Cluster,
  NamespaceInfo,
  DeploymentInfo,
  StatefulSetInfo,
  PodInfo,
} from '../../../api/modules/cluster';

const { Title, Text } = Typography;

const ClusterWorkloadsPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const clusterId = Number(id);
  const [cluster, setCluster] = useState<Cluster | null>(null);
  const [namespaces, setNamespaces] = useState<NamespaceInfo[]>([]);
  const [namespace, setNamespace] = useState('default');
  const [deployments, setDeployments] = useState<DeploymentInfo[]>([]);
  const [statefulsets, setStatefulsets] = useState<StatefulSetInfo[]>([]);
  const [pods, setPods] = useState<PodInfo[]>([]);
  const [loading, setLoading] = useState(false);

  const loadClusterAndNamespaces = useCallback(async () => {
    if (!clusterId) return;
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
      message.error(err instanceof Error ? err.message : '加载工作负载元信息失败');
    }
  }, [clusterId]);

  const loadWorkloads = useCallback(async (ns: string) => {
    if (!clusterId) return;
    setLoading(true);
    try {
      const [depRes, stsRes, podRes] = await Promise.all([
        Api.cluster.getDeployments(clusterId, ns),
        Api.cluster.getStatefulSets(clusterId, ns),
        Api.cluster.getPods(clusterId, ns),
      ]);
      setDeployments(depRes.data.list || []);
      setStatefulsets(stsRes.data.list || []);
      setPods(podRes.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载工作负载失败');
      setDeployments([]);
      setStatefulsets([]);
      setPods([]);
    } finally {
      setLoading(false);
    }
  }, [clusterId]);

  useEffect(() => {
    loadClusterAndNamespaces();
  }, [loadClusterAndNamespaces]);

  useEffect(() => {
    if (!namespace) return;
    loadWorkloads(namespace);
  }, [namespace, loadWorkloads]);

  const namespaceOptions = useMemo(
    () => namespaces.map((item) => ({ label: item.name, value: item.name })),
    [namespaces],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card>
        <Title level={4} style={{ marginBottom: 4 }}>工作负载</Title>
        <Text type="secondary">{cluster?.name || `#${clusterId}`}</Text>
      </Card>
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
          <Tabs
            items={[
              {
                key: 'deployments',
                label: 'Deployments',
                children: (
                  <Table<DeploymentInfo>
                    size="small"
                    rowKey="name"
                    loading={loading}
                    pagination={false}
                    dataSource={deployments}
                    columns={[
                      { title: '名称', dataIndex: 'name', key: 'name' },
                      { title: '副本', dataIndex: 'replicas', key: 'replicas' },
                      { title: 'Ready', dataIndex: 'ready', key: 'ready' },
                      { title: 'Age', dataIndex: 'age', key: 'age' },
                    ]}
                  />
                ),
              },
              {
                key: 'statefulsets',
                label: 'StatefulSets',
                children: (
                  <Table<StatefulSetInfo>
                    size="small"
                    rowKey="name"
                    loading={loading}
                    pagination={false}
                    dataSource={statefulsets}
                    columns={[
                      { title: '名称', dataIndex: 'name', key: 'name' },
                      { title: '副本', dataIndex: 'replicas', key: 'replicas' },
                      { title: 'Ready', dataIndex: 'ready', key: 'ready' },
                      { title: 'Age', dataIndex: 'age', key: 'age' },
                    ]}
                  />
                ),
              },
              {
                key: 'pods',
                label: 'Pods',
                children: (
                  <Table<PodInfo>
                    size="small"
                    rowKey="name"
                    loading={loading}
                    pagination={false}
                    dataSource={pods}
                    columns={[
                      { title: '名称', dataIndex: 'name', key: 'name' },
                      { title: '状态', dataIndex: 'status', key: 'status' },
                      { title: 'Node', dataIndex: 'node', key: 'node' },
                      { title: 'Age', dataIndex: 'age', key: 'age' },
                    ]}
                  />
                ),
              },
            ]}
          />
        </Space>
      </Card>
    </Space>
  );
};

export default ClusterWorkloadsPage;
