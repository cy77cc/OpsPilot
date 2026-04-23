import React from 'react';
import { Card, Tabs, Table, Tag, Button, Space, Descriptions, Spin, Select, Popconfirm, Typography } from 'antd';
import {
  ReloadOutlined,
  PlusOutlined,
  NodeIndexOutlined,
  AppstoreOutlined,
  CloudServerOutlined,
  SettingOutlined,
  DatabaseOutlined,
  CloudOutlined,
  ToolOutlined,
  InfoCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import type { EventInfo, HPAInfo, LimitRangeInfo, ResourceQuotaInfo } from '../../../../api/modules/cluster';

const { Text } = Typography;

export function ClusterDetailTabs(props: any) {
  const {
    cluster,
    clusterId,
    nodes,
    nodesLoading,
    selectedNamespace,
    setSelectedNamespace,
    namespaces,
    deployments,
    statefulsets,
    daemonsets,
    pods,
    services,
    ingresses,
    configmaps,
    secrets,
    pvcs,
    pvs,
    clusterServices,
    resourceLoading,
    advancedLoading,
    hpas,
    resourceQuotas,
    limitRanges,
    clusterVersion,
    certificates,
    upgradePlan,
    events,
    loadEvents,
    nodeColumns,
    buildWorkloadColumns,
    podColumns,
    serviceColumns,
    ingressColumns,
    configColumns,
    storageColumns,
    clusterServiceColumns,
    handleDeploymentRestart,
    handleDeploymentScale,
    handleDeploymentDelete,
    handleStatefulSetRestart,
    handleStatefulSetScale,
    handleStatefulSetDelete,
    openServiceModal,
    openIngressModal,
    renderFeedback,
    handleRenewCertificates,
    handleClusterUpgrade,
    setAddNodeModalVisible,
  } = props;
  return (
      <Tabs defaultActiveKey="nodes" items={[
        {
          key: 'nodes',
          label: <span><NodeIndexOutlined /> 节点 ({nodes.length})</span>,
          children: (
            <Card title="节点列表" extra={cluster.source === 'platform_managed' && <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddNodeModalVisible(true)}>添加节点</Button>}>
              <Table columns={nodeColumns} dataSource={nodes} rowKey="id" loading={nodesLoading} pagination={false} />
            </Card>
          ),
        },
        {
          key: 'workloads',
          label: <span><AppstoreOutlined /> 工作负载</span>,
          children: (
            <div className="space-y-4">
              <Select style={{ width: 200 }} value={selectedNamespace} onChange={setSelectedNamespace} options={namespaces.map((ns: any) => ({ label: ns.name, value: ns.name }))} loading={resourceLoading} />
              <Spin spinning={resourceLoading}>
                <Card title="Deployments" size="small" className="mb-4">
                  <Table
                    columns={buildWorkloadColumns('deployment', handleDeploymentRestart, handleDeploymentScale, handleDeploymentDelete)}
                    dataSource={deployments}
                    rowKey="name"
                    pagination={false}
                    size="small"
                  />
                </Card>
                <Card title="StatefulSets" size="small" className="mb-4">
                  <Table
                    columns={buildWorkloadColumns('statefulset', handleStatefulSetRestart, handleStatefulSetScale, handleStatefulSetDelete)}
                    dataSource={statefulsets}
                    rowKey="name"
                    pagination={false}
                    size="small"
                  />
                </Card>
                <Card title="DaemonSets" size="small" className="mb-4">
                  <Table columns={[{ title: '名称', dataIndex: 'name' }, { title: 'Desired', dataIndex: 'desired' }, { title: 'Ready', dataIndex: 'ready' }, { title: 'Age', dataIndex: 'age' }]} dataSource={daemonsets} rowKey="name" pagination={false} size="small" />
                </Card>
                <Card title="Pods" size="small">
                  <Table columns={podColumns} dataSource={pods} rowKey="name" pagination={false} size="small" />
                </Card>
              </Spin>
            </div>
          ),
        },
        {
          key: 'services',
          label: <span><CloudServerOutlined /> 服务</span>,
          children: (
            <div className="space-y-4">
              <Select style={{ width: 200 }} value={selectedNamespace} onChange={setSelectedNamespace} options={namespaces.map((ns: any) => ({ label: ns.name, value: ns.name }))} />
              <Spin spinning={resourceLoading}>
                <Card
                  title="Services"
                  size="small"
                  className="mb-4"
                  extra={<Button size="small" type="primary" onClick={() => { openServiceModal('create'); }}>新建 Service</Button>}
                >
                  <Table columns={serviceColumns} dataSource={services} rowKey="name" pagination={false} size="small" />
                </Card>
                <Card
                  title="Ingresses"
                  size="small"
                  extra={<Button size="small" type="primary" onClick={() => { openIngressModal('create'); }}>新建 Ingress</Button>}
                >
                  <Table columns={ingressColumns} dataSource={ingresses} rowKey="name" pagination={false} size="small" />
                </Card>
              </Spin>
            </div>
          ),
        },
        {
          key: 'config',
          label: <span><SettingOutlined /> 配置</span>,
          children: (
            <div className="space-y-4">
              <Select style={{ width: 200 }} value={selectedNamespace} onChange={setSelectedNamespace} options={namespaces.map((ns: any) => ({ label: ns.name, value: ns.name }))} />
              <Spin spinning={resourceLoading}>
                <Card title="ConfigMaps" size="small" className="mb-4">
                  <Table columns={configColumns} dataSource={configmaps} rowKey="name" pagination={false} size="small" />
                </Card>
                <Card title="Secrets" size="small">
                  <Table columns={[...configColumns, { title: '类型', dataIndex: 'type', key: 'type' }]} dataSource={secrets} rowKey="name" pagination={false} size="small" />
                </Card>
              </Spin>
            </div>
          ),
        },
        {
          key: 'storage',
          label: <span><DatabaseOutlined /> 存储</span>,
          children: (
            <div className="space-y-4">
              <Select style={{ width: 200 }} value={selectedNamespace} onChange={setSelectedNamespace} options={namespaces.map((ns: any) => ({ label: ns.name, value: ns.name }))} />
              <Spin spinning={resourceLoading}>
                <Card title="PersistentVolumes" size="small" className="mb-4">
                  <Table columns={[...storageColumns, { title: 'Claim', dataIndex: 'claim_ref', key: 'claim_ref' }]} dataSource={pvs} rowKey="name" pagination={false} size="small" />
                </Card>
                <Card title="PersistentVolumeClaims" size="small">
                  <Table columns={[...storageColumns, { title: 'Volume', dataIndex: 'volume_name', key: 'volume_name' }]} dataSource={pvcs} rowKey="name" pagination={false} size="small" />
                </Card>
              </Spin>
            </div>
          ),
        },
        {
          key: 'deployed-services',
          label: <span><CloudOutlined /> 部署的服务</span>,
          children: (
            <Card title="该集群部署的服务">
              <Table columns={clusterServiceColumns} dataSource={clusterServices} rowKey="id" pagination={false} />
            </Card>
          ),
        },
        {
          key: 'policy',
          label: <span><SettingOutlined /> 策略</span>,
          children: (
            <div className="space-y-4">
              <Select style={{ width: 200 }} value={selectedNamespace} onChange={setSelectedNamespace} options={namespaces.map((ns: any) => ({ label: ns.name, value: ns.name }))} />
              <Spin spinning={advancedLoading}>
                <Card title="HPA (Horizontal Pod Autoscaler)" size="small" className="mb-4">
                  <Table
                    columns={[
                      { title: '名称', dataIndex: 'name', key: 'name' },
                      { title: '引用', dataIndex: 'reference', key: 'reference' },
                      { title: '副本', key: 'replicas', render: (_: any, r: HPAInfo) => `${r.replicas} (${r.min_replicas}-${r.max_replicas})` },
                      { title: 'CPU', key: 'cpu', render: (_: any, r: HPAInfo) => r.current_cpu || '-' },
                      { title: '内存', key: 'mem', render: (_: any, r: HPAInfo) => r.current_mem || '-' },
                      { title: 'Age', dataIndex: 'age', key: 'age' },
                    ]}
                    dataSource={hpas} rowKey="name" pagination={false} size="small"
                  />
                </Card>
                <Card title="ResourceQuota" size="small" className="mb-4">
                  <Table
                    columns={[
                      { title: '名称', dataIndex: 'name', key: 'name' },
                      { title: 'CPU 限制', key: 'cpu', render: (_: any, r: ResourceQuotaInfo) => `${r.used['limits.cpu'] || '-'} / ${r.hard['limits.cpu'] || '-'}` },
                      { title: '内存限制', key: 'mem', render: (_: any, r: ResourceQuotaInfo) => `${r.used['limits.memory'] || '-'} / ${r.hard['limits.memory'] || '-'}` },
                      { title: 'Pods', key: 'pods', render: (_: any, r: ResourceQuotaInfo) => `${r.used['count/pods'] || '0'} / ${r.hard['count/pods'] || '-'}` },
                      { title: 'Age', dataIndex: 'age', key: 'age' },
                    ]}
                    dataSource={resourceQuotas} rowKey="name" pagination={false} size="small"
                  />
                </Card>
                <Card title="LimitRange" size="small">
                  <Table
                    columns={[
                      { title: '名称', dataIndex: 'name', key: 'name' },
                      { title: '类型', dataIndex: 'type', key: 'type' },
                      { title: '默认CPU', key: 'default_cpu', render: (_: any, r: LimitRangeInfo) => r.limits?.[0]?.default?.cpu || '-' },
                      { title: '默认内存', key: 'default_mem', render: (_: any, r: LimitRangeInfo) => r.limits?.[0]?.default?.memory || '-' },
                      { title: 'Age', dataIndex: 'age', key: 'age' },
                    ]}
                    dataSource={limitRanges} rowKey="name" pagination={false} size="small"
                  />
                </Card>
              </Spin>
            </div>
          ),
        },
        {
          key: 'events',
          label: <span><InfoCircleOutlined /> 事件</span>,
          children: (
            <Card title="集群事件" extra={<Button icon={<ReloadOutlined />} onClick={loadEvents}>刷新</Button>}>
              <Table
                columns={[
                  { title: '类型', dataIndex: 'type', key: 'type', width: 80, render: (t: string) => <Tag color={t === 'Normal' ? 'green' : 'red'}>{t}</Tag> },
                  { title: 'Reason', dataIndex: 'reason', key: 'reason', width: 120 },
                  { title: '对象', key: 'object', render: (_: any, r: EventInfo) => `${r.namespace}/${r.name}` },
                  { title: '消息', dataIndex: 'message', key: 'message', ellipsis: true },
                  { title: '来源', dataIndex: 'source', key: 'source', width: 120 },
                  { title: '次数', dataIndex: 'count', key: 'count', width: 60 },
                  { title: 'Age', dataIndex: 'age', key: 'age', width: 80 },
                ]}
                dataSource={events} rowKey={(r, i) => `${r.namespace}-${r.name}-${i}`} pagination={{ pageSize: 20 }} size="small"
              />
            </Card>
          ),
        },
        {
          key: 'maintenance',
          label: <span><ToolOutlined /> 运维</span>,
          children: (
            <div className="space-y-4">
              <Card title="集群版本" size="small" className="mb-4">
                {clusterVersion ? (
                  <Descriptions column={2} size="small">
                    <Descriptions.Item label="Kubernetes">{clusterVersion.kubernetes_version}</Descriptions.Item>
                    <Descriptions.Item label="Platform">{clusterVersion.platform}</Descriptions.Item>
                    <Descriptions.Item label="Go Version">{clusterVersion.go_version}</Descriptions.Item>
                  </Descriptions>
                ) : <Text type="secondary">加载中...</Text>}
              </Card>
              <Card title="证书信息" size="small" className="mb-4" extra={
                cluster?.source === 'platform_managed' && (
                  <Popconfirm
                    title="确定续期所有证书？此操作将重启控制平面组件。"
                    okText="确定"
                    cancelText="取消"
                    onConfirm={() => { void handleRenewCertificates(); }}
                  >
                    <Button size="small" icon={<SyncOutlined />}>续期证书</Button>
                  </Popconfirm>
                )
              }>
                {renderFeedback('cluster:certificates')}
                <Table
                  columns={[
                    { title: '名称', dataIndex: 'name', key: 'name' },
                    { title: 'CA', dataIndex: 'ca', key: 'ca', width: 60, render: (v: boolean) => v ? <Tag color="blue">CA</Tag> : '-' },
                    { title: '过期时间', dataIndex: 'expires_at', key: 'expires_at' },
                    { title: '剩余天数', dataIndex: 'days_left', key: 'days_left', render: (d: number) => <Tag color={d < 30 ? 'red' : d < 90 ? 'orange' : 'green'}>{d} 天</Tag> },
                  ]}
                  dataSource={certificates} rowKey="name" pagination={false} size="small"
                />
              </Card>
              {upgradePlan && cluster?.source === 'platform_managed' && (
                <Card title="升级计划" size="small" extra={
                  upgradePlan.upgradable && (
                    <Popconfirm
                      title="确定升级集群？建议先备份数据。"
                      okText="确定"
                      cancelText="取消"
                      onConfirm={() => { void handleClusterUpgrade(); }}
                    >
                      <Button size="small" type="primary">升级集群</Button>
                    </Popconfirm>
                  )
                }>
                  <Descriptions column={1} size="small">
                    <Descriptions.Item label="当前版本">{upgradePlan.current_version}</Descriptions.Item>
                    <Descriptions.Item label="可升级">{upgradePlan.upgradable ? <Tag color="green">是</Tag> : <Tag color="red">否</Tag>}</Descriptions.Item>
                  </Descriptions>
                  {renderFeedback('cluster:upgrade')}
                  {upgradePlan.warnings?.length > 0 && (
                    <div className="mt-4">
                      <Text type="warning">警告:</Text>
                      <ul className="list-disc pl-6 mt-2">
                        {upgradePlan.warnings.map((w: any, i: number) => <li key={i} className="text-orange-500">{w}</li>)}
                      </ul>
                    </div>
                  )}
                  {upgradePlan.steps?.length > 0 && (
                    <div className="mt-4">
                      <Text>升级步骤:</Text>
                      <ol className="list-decimal pl-6 mt-2">
                        {upgradePlan.steps.map((s: any, i: number) => <li key={i}>{s}</li>)}
                      </ol>
                    </div>
                  )}
                </Card>
              )}
            </div>
          ),
        },
      ]} />
  );
}
