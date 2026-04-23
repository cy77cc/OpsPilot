import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Card, Select, Space, Table, Tabs, Typography, message } from 'antd';
import { useParams } from 'react-router-dom';
import { Api } from '../../../api';
import type {
  Cluster,
  NamespaceInfo,
  ConfigMapInfo,
  SecretInfo,
  PVCInfo,
  PVInfo,
} from '../../../api/modules/cluster';

const { Title, Text } = Typography;

const ClusterConfigStoragePage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const clusterId = Number(id);
  const [cluster, setCluster] = useState<Cluster | null>(null);
  const [namespaces, setNamespaces] = useState<NamespaceInfo[]>([]);
  const [namespace, setNamespace] = useState('default');
  const [configmaps, setConfigmaps] = useState<ConfigMapInfo[]>([]);
  const [secrets, setSecrets] = useState<SecretInfo[]>([]);
  const [pvcs, setPvcs] = useState<PVCInfo[]>([]);
  const [pvs, setPvs] = useState<PVInfo[]>([]);
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
      message.error(err instanceof Error ? err.message : '加载配置与存储元信息失败');
    }
  }, [clusterId]);

  const loadData = useCallback(async (ns: string) => {
    if (!clusterId) {return;}
    setLoading(true);
    try {
      const [cmRes, secretRes, pvcRes, pvRes] = await Promise.all([
        Api.cluster.getConfigMaps(clusterId, ns),
        Api.cluster.getSecrets(clusterId, ns),
        Api.cluster.getPVCs(clusterId, ns),
        Api.cluster.getPVs(clusterId),
      ]);
      setConfigmaps(cmRes.data.list || []);
      setSecrets(secretRes.data.list || []);
      setPvcs(pvcRes.data.list || []);
      setPvs(pvRes.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载配置与存储数据失败');
      setConfigmaps([]);
      setSecrets([]);
      setPvcs([]);
      setPvs([]);
    } finally {
      setLoading(false);
    }
  }, [clusterId]);

  useEffect(() => {
    loadMeta();
  }, [loadMeta]);

  useEffect(() => {
    if (!namespace) {return;}
    loadData(namespace);
  }, [namespace, loadData]);

  const namespaceOptions = useMemo(
    () => namespaces.map((item) => ({ label: item.name, value: item.name })),
    [namespaces],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card>
        <Title level={4} style={{ marginBottom: 4 }}>配置与存储</Title>
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
                key: 'configmaps',
                label: 'ConfigMaps',
                children: (
                  <Table<ConfigMapInfo>
                    size="small"
                    rowKey="name"
                    loading={loading}
                    pagination={false}
                    dataSource={configmaps}
                    columns={[
                      { title: '名称', dataIndex: 'name', key: 'name' },
                      { title: 'Keys', key: 'keys', render: (_, record) => record.data_keys?.length || 0 },
                      { title: 'Age', dataIndex: 'age', key: 'age' },
                    ]}
                  />
                ),
              },
              {
                key: 'secrets',
                label: 'Secrets',
                children: (
                  <Table<SecretInfo>
                    size="small"
                    rowKey="name"
                    loading={loading}
                    pagination={false}
                    dataSource={secrets}
                    columns={[
                      { title: '名称', dataIndex: 'name', key: 'name' },
                      { title: '类型', dataIndex: 'type', key: 'type' },
                      { title: 'Keys', key: 'keys', render: (_, record) => record.data_keys?.length || 0 },
                      { title: 'Age', dataIndex: 'age', key: 'age' },
                    ]}
                  />
                ),
              },
              {
                key: 'pvcs',
                label: 'PVCs',
                children: (
                  <Table<PVCInfo>
                    size="small"
                    rowKey="name"
                    loading={loading}
                    pagination={false}
                    dataSource={pvcs}
                    columns={[
                      { title: '名称', dataIndex: 'name', key: 'name' },
                      { title: '状态', dataIndex: 'status', key: 'status' },
                      { title: '容量', dataIndex: 'capacity', key: 'capacity' },
                      { title: 'StorageClass', dataIndex: 'storage_class', key: 'storage_class' },
                      { title: 'Age', dataIndex: 'age', key: 'age' },
                    ]}
                  />
                ),
              },
              {
                key: 'pvs',
                label: 'PVs',
                children: (
                  <Table<PVInfo>
                    size="small"
                    rowKey="name"
                    loading={loading}
                    pagination={false}
                    dataSource={pvs}
                    columns={[
                      { title: '名称', dataIndex: 'name', key: 'name' },
                      { title: '状态', dataIndex: 'status', key: 'status' },
                      { title: '容量', dataIndex: 'capacity', key: 'capacity' },
                      { title: 'StorageClass', dataIndex: 'storage_class', key: 'storage_class' },
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

export default ClusterConfigStoragePage;
