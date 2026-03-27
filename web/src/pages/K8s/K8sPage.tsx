import React, { useEffect, useState } from 'react';
import { Button, Card, Drawer, Form, Input, InputNumber, Modal, Space, Table, Tabs, Tag, message } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { Api } from '../../api';
import type { Cluster, Node } from '../../api/modules/kubernetes';
import ClusterOverview from '../../components/K8s/ClusterOverview';
import NamespacePolicyPanel from '../../components/K8s/NamespacePolicyPanel';
import RolloutPanel from '../../components/K8s/RolloutPanel';
import HPAEditor from '../../components/K8s/HPAEditor';
import QuotaEditor from '../../components/K8s/QuotaEditor';

const K8sPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [deployments, setDeployments] = useState<any[]>([]);
  const [pods, setPods] = useState<any[]>([]);
  const [services, setServices] = useState<any[]>([]);
  const [ingresses, setIngresses] = useState<any[]>([]);
  const [events, setEvents] = useState<any[]>([]);
  const [dataSourceHint, setDataSourceHint] = useState<string>('');
  const [selectedCluster, setSelectedCluster] = useState<Cluster | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [deployOpen, setDeployOpen] = useState(false);
  const [deployPreview, setDeployPreview] = useState<any>(null);
  const [topologyOpen, setTopologyOpen] = useState(false);
  const [topology, setTopology] = useState<any>(null);
  const [form] = Form.useForm();
  const [createForm] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const res = await Api.kubernetes.getClusterList({ page: 1, pageSize: 50 });
      setClusters(res.data.list || []);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const openDetail = async (cluster: Cluster) => {
    setSelectedCluster(cluster);
    const [nodeRes, deploymentRes, podRes, svcRes, ingRes] = await Promise.all([
      Api.kubernetes.getClusterNodes(String(cluster.id), { page: 1, pageSize: 100 }),
      Api.kubernetes.getClusterDeployments(String(cluster.id)),
      Api.kubernetes.getClusterPods(String(cluster.id)),
      Api.kubernetes.getClusterServices(String(cluster.id)),
      Api.kubernetes.getClusterIngresses(String(cluster.id)),
    ]);
    setNodes(nodeRes.data.list || []);
    setDeployments(deploymentRes.data || []);
    setPods(podRes.data || []);
    setServices(svcRes.data || []);
    setIngresses(ingRes.data || []);
    setDataSourceHint([nodeRes.dataSource, podRes.dataSource, svcRes.dataSource].filter(Boolean).join('/'));
    const evRes = await Api.kubernetes.getClusterEvents(String(cluster.id));
    setEvents(evRes.data || []);
    setDrawerOpen(true);
  };

  const previewDeploy = async () => {
    if (!selectedCluster) return;
    const values = await form.validateFields();
    const preview = await Api.kubernetes.previewDeploy(String(selectedCluster.id), values);
    setDeployPreview(preview.data);
  };

  const applyDeploy = async () => {
    if (!selectedCluster) return;
    const values = await form.validateFields();
    await Api.kubernetes.applyDeploy(String(selectedCluster.id), values);
    message.success('部署已应用');
    setDeployOpen(false);
    setDeployPreview(null);
    openDetail(selectedCluster);
  };

  const connectTest = async (cluster: Cluster) => {
    const res = await Api.kubernetes.testClusterConnect(String(cluster.id));
    const result = res.data as any;
    if (result?.connected) {
      message.success(`连接测试通过（${result.latency_ms}ms）`);
    } else {
      message.warning(`连接失败：${result?.message || 'unknown error'}`);
    }
    load();
  };

  const createCluster = async () => {
    const values = await createForm.validateFields();
    await Api.kubernetes.createCluster(values);
    message.success('集群已创建');
    setCreateOpen(false);
    createForm.resetFields();
    load();
  };

  const openTopology = async (cluster: Cluster) => {
    const res = await Api.topology.getClusterServices(String(cluster.id));
    setTopology(res.data);
    setTopologyOpen(true);
  };

  return (
    <Card
      title="Kubernetes 集群"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} loading={loading} onClick={load}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>添加集群</Button>
        </Space>
      }
    >
      <Table
        rowKey="id"
        loading={loading}
        dataSource={clusters}
        columns={[
          { title: '名称', dataIndex: 'name' },
          { title: '版本', dataIndex: 'version' },
          { title: '状态', dataIndex: 'status', render: (v: string) => <Tag color={v === 'connected' ? 'success' : 'default'}>{v}</Tag> },
          { title: '创建时间', dataIndex: 'createdAt', render: (v: string) => (v ? new Date(v).toLocaleString() : '-') },
          { title: '操作', render: (_: unknown, r: Cluster) => <Space><Button type="link" onClick={() => openDetail(r)}>详情</Button><Button type="link" onClick={() => connectTest(r)}>连接测试</Button><Button type="link" onClick={() => openTopology(r)}>拓扑</Button><Button type="link" onClick={() => { setSelectedCluster(r); setDeployOpen(true); }}>部署向导</Button></Space> },
        ]}
      />

      <Drawer title={`集群详情 - ${selectedCluster?.name || ''}`} open={drawerOpen} onClose={() => setDrawerOpen(false)} width={980}>
        {dataSourceHint ? <Tag color={dataSourceHint.includes('live') ? 'success' : 'warning'}>data_source: {dataSourceHint}</Tag> : null}
        <Tabs
          items={[
            { key: 'overview', label: 'Overview', children: <ClusterOverview nodes={nodes} deployments={deployments} pods={pods} services={services} ingresses={ingresses} dataSourceHint={dataSourceHint} /> },
            { key: 'namespaces-policy', label: 'Namespaces', children: selectedCluster ? <NamespacePolicyPanel clusterId={String(selectedCluster.id)} /> : null },
            { key: 'rollouts', label: 'Rollouts', children: selectedCluster ? <RolloutPanel clusterId={String(selectedCluster.id)} /> : null },
            { key: 'hpa', label: 'HPA', children: selectedCluster ? <HPAEditor clusterId={String(selectedCluster.id)} /> : null },
            { key: 'quota', label: 'Quotas', children: selectedCluster ? <QuotaEditor clusterId={String(selectedCluster.id)} /> : null },
            { key: 'nodes', label: 'Nodes', children: <Table rowKey="id" dataSource={nodes} columns={[{ title: '名称', dataIndex: 'name' }, { title: 'IP', dataIndex: 'ip' }, { title: '状态', dataIndex: 'status' }, { title: '角色', dataIndex: 'role' }, { title: 'Pods', dataIndex: 'pods' }]} pagination={false} /> },
            { key: 'namespaces', label: 'Namespaces', children: <Table rowKey="id" dataSource={[...new Set(deployments.map((d) => d.namespace))].map((n, idx) => ({ id: idx, name: n }))} columns={[{ title: '命名空间', dataIndex: 'name' }]} pagination={false} /> },
            { key: 'workloads', label: 'Workloads', children: <Table rowKey="id" dataSource={deployments} columns={[{ title: '命名空间', dataIndex: 'namespace' }, { title: '名称', dataIndex: 'name' }, { title: '镜像', dataIndex: 'image' }, { title: '副本', dataIndex: 'replicas' }, { title: '状态', dataIndex: 'status' }]} pagination={false} /> },
            { key: 'network', label: 'Network', children: <><Table rowKey="id" dataSource={services} columns={[{ title: '名称', dataIndex: 'name' }, { title: '命名空间', dataIndex: 'namespace' }, { title: '类型', dataIndex: 'type' }, { title: 'ClusterIP', dataIndex: 'cluster_ip' }, { title: '端口', render: (_: any, r: any) => (r.ports || []).map((p: any) => `${p.port}:${p.targetPort}`).join(', ') }]} pagination={false} /><div className="h-4" /><Table rowKey="id" dataSource={ingresses} columns={[{ title: 'Ingress', dataIndex: 'name' }, { title: 'Host', dataIndex: 'host' }, { title: 'Path', dataIndex: 'path' }, { title: 'Service', dataIndex: 'service' }, { title: 'TLS', dataIndex: 'tls', render: (v: boolean) => <Tag color={v ? 'success' : 'default'}>{String(v)}</Tag> }]} pagination={false} /></> },
            { key: 'events', label: 'Events', children: <Table rowKey={(_r: any, idx?: number) => String(idx)} dataSource={events} columns={[{ title: 'Type', render: (_: any, r: any) => r?.type || r?.reason || '-' }, { title: 'Message', render: (_: any, r: any) => r?.message || r?.note || JSON.stringify(r) }]} pagination={false} /> },
          ]}
        />
      </Drawer>

      <Drawer title="Cluster Topology" open={topologyOpen} onClose={() => setTopologyOpen(false)} width={820}>
        <pre style={{ maxHeight: 620, overflow: 'auto' }}>{JSON.stringify(topology, null, 2)}</pre>
      </Drawer>

      <Modal title="添加集群" open={createOpen} onCancel={() => setCreateOpen(false)} onOk={createCluster} okText="创建">
        <Form form={createForm} layout="vertical">
          <Form.Item name="name" label="集群名称" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="server" label="API Server" rules={[{ required: true }]}><Input placeholder="https://127.0.0.1:6443" /></Form.Item>
          <Form.Item name="kubeconfig" label="Kubeconfig"><Input.TextArea rows={6} /></Form.Item>
          <Form.Item name="credential_ref" label="Credential Ref"><Input placeholder="env:KUBECONFIG_PROD or vault:cluster/prod" /></Form.Item>
          <Form.Item name="description" label="描述"><Input.TextArea rows={2} /></Form.Item>
        </Form>
      </Modal>

      <Modal title={`部署向导 - ${selectedCluster?.name || ''}`} open={deployOpen} onCancel={() => { setDeployOpen(false); setDeployPreview(null); }} onOk={applyDeploy} okText="确认应用">
        <Form form={form} layout="vertical" initialValues={{ namespace: 'default', replicas: 1 }}>
          <Form.Item name="namespace" label="命名空间" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="name" label="应用名称" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="image" label="镜像" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="replicas" label="副本"><InputNumber min={1} /></Form.Item>
        </Form>
        <Button onClick={previewDeploy}>预览 Diff</Button>
        {deployPreview ? <pre style={{ marginTop: 12, maxHeight: 180, overflow: 'auto' }}>{JSON.stringify(deployPreview, null, 2)}</pre> : null}
      </Modal>
    </Card>
  );
};

export default K8sPage;
