import React, { useMemo } from 'react';
import { 
  Card, Tabs, Table, Tag, Button, Space, Descriptions, Spin, 
  Select, Popconfirm, Typography, Row, Col, Progress, Empty, Badge 
} from 'antd';
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
  ArrowRightOutlined,
  HistoryOutlined,
  AlertOutlined,
  MonitorOutlined
} from '@ant-design/icons';
import { Pie, Line } from '@ant-design/charts';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import type { EventInfo, HPAInfo, LimitRangeInfo, ResourceQuotaInfo, ClusterNode } from '../../../../api/modules/cluster';

const { Text, Title } = Typography;

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
    activeKey,
  } = props;

  const navigate = useNavigate();

  // Mock data for overview tab charts
  const trendData = useMemo(() => [
    { time: '06:00', value: 35, type: 'CPU 使用率' },
    { time: '07:00', value: 38, type: 'CPU 使用率' },
    { time: '08:00', value: 42, type: 'CPU 使用率' },
    { time: '09:00', value: 45, type: 'CPU 使用率' },
    { time: '10:00', value: 43, type: 'CPU 使用率' },
    { time: '11:00', value: 46, type: 'CPU 使用率' },
    { time: '12:00', value: 43, type: 'CPU 使用率' },
    { time: '06:00', value: 58, type: '内存使用率' },
    { time: '07:00', value: 60, type: '内存使用率' },
    { time: '08:00', value: 62, type: '内存使用率' },
    { time: '09:00', value: 65, type: '内存使用率' },
    { time: '10:00', value: 63, type: '内存使用率' },
    { time: '11:00', value: 64, type: '内存使用率' },
    { time: '12:00', value: 62, type: '内存使用率' },
  ], []);

  const nsDistributionData = useMemo(() => [
    { type: 'kube-system', value: 28 },
    { type: 'ingress-nginx', value: 16 },
    { type: 'monitoring', value: 14 },
    { type: 'payment', value: 18 },
    { type: 'default', value: 10 },
  ], []);

  const workloadStats = [
    { type: 'Deployment', count: deployments.length || 128, delta: '+2', color: '#2563eb' },
    { type: 'StatefulSet', count: statefulsets.length || 12, delta: '0', color: '#7c3aed' },
    { type: 'DaemonSet', count: daemonsets.length || 8, delta: '0', color: '#0891b2' },
    { type: 'Job', count: 36, delta: '+5', color: '#d97706' },
    { type: 'CronJob', count: 9, delta: '0', color: '#4b5563' },
    { type: 'Service', count: services.length || 214, delta: '+12', color: '#16a34a' },
    { type: 'Ingress', count: ingresses.length || 18, delta: '+1', color: '#db2777' },
  ];

  const renderOverview = () => (
    <div className="space-y-4">
      {/* Row 1: Trend, Distribution, Alerts */}
      <Row gutter={16}>
        <Col span={10}>
          <Card 
            title={<span className="text-[14px] font-bold text-[#1f2937]">资源使用趋势</span>} 
            size="small" 
            className="h-full border-[#e8edf3] rounded-[10px] overflow-hidden"
            extra={<Select size="small" defaultValue="6h" options={[{ value: '6h', label: '近 6h' }]} style={{ width: 80 }} />}
          >
            <div className="h-[220px]">
              <Line 
                data={trendData}
                xField="time"
                yField="value"
                seriesField="type"
                color={['#2563eb', '#10b981']}
                point={{ size: 2 }}
                smooth={true}
                autoFit={true}
              />
            </div>
          </Card>
        </Col>
        <Col span={7}>
          <Card 
            title={<span className="text-[14px] font-bold text-[#1f2937]">命名空间资源分布</span>} 
            size="small" 
            className="h-full border-[#e8edf3] rounded-[10px] overflow-hidden"
          >
            <div className="h-[220px]">
              <Pie 
                data={nsDistributionData}
                angleField="value"
                colorField="type"
                innerRadius={0.65}
                legend={{ position: 'bottom', layout: 'horizontal' }}
                label={false}
                autoFit={true}
              />
            </div>
          </Card>
        </Col>
        <Col span={7}>
          <Card 
            title={<span className="text-[14px] font-bold text-[#1f2937]">最近告警</span>} 
            size="small" 
            className="h-full border-[#e8edf3] rounded-[10px] overflow-hidden"
            extra={<Button type="link" size="small" onClick={() => props.onChange?.('alerts')}>更多 <ArrowRightOutlined className="text-[10px]" /></Button>}
          >
            <div className="space-y-4 py-1">
              {[
                { title: 'Pod 重启次数异常', level: 'critical', target: 'payment/checkout-service-xxx', time: '12:18:32' },
                { title: '节点磁盘使用率偏高', level: 'warning', target: 'worker-02', time: '11:42:07' },
                { title: 'etcd 延迟升高', level: 'warning', target: 'master-01', time: '10:33:48' },
                { title: 'Pod 内存使用率过高', level: 'warning', target: 'payment/order-service-xxx', time: '09:58:11' },
              ].map((alert, idx) => (
                <div key={idx} className="flex justify-between items-start text-[12px] group cursor-pointer">
                  <div className="flex gap-2 min-w-0">
                    <Badge color={alert.level === 'critical' ? '#ef4444' : '#f59e0b'} className="mt-1.5" />
                    <div className="min-w-0">
                      <div className="font-medium text-[#374151] truncate group-hover:text-blue-500 transition-colors">{alert.title}</div>
                      <div className="text-[#9ca3af] text-[11px] mt-0.5 truncate">{alert.target}</div>
                    </div>
                  </div>
                  <span className="text-[#9ca3af] text-[11px] flex-shrink-0 ml-2">{alert.time}</span>
                </div>
              ))}
            </div>
          </Card>
        </Col>
      </Row>

      {/* Row 2: Nodes and Events */}
      <Row gutter={16}>
        <Col span={14}>
          <Card 
            title={<span className="text-[14px] font-bold text-[#1f2937]">节点概览</span>} 
            size="small" 
            className="h-full border-[#e8edf3] rounded-[10px] overflow-hidden"
            extra={<Button type="link" size="small" onClick={() => props.onChange?.('nodes')}>查看全部 <ArrowRightOutlined className="text-[10px]" /></Button>}
          >
            <Table 
              size="small"
              pagination={false}
              columns={[
                { title: '节点名', dataIndex: 'name', key: 'name', render: (text) => <span className="font-medium text-[#374151]">{text}</span> },
                { title: '角色', dataIndex: 'role', key: 'role', render: (role) => <Tag className="m-0 border-none bg-gray-100 text-gray-600 rounded-[4px]">{role || 'worker'}</Tag> },
                { title: 'CPU', key: 'cpu', render: (_, r: any) => <div className="w-16"><Progress percent={r.cpu_usage || 28} size="small" showInfo={false} strokeWidth={4} strokeColor="#2563eb" /></div> },
                { title: '内存', key: 'mem', render: (_, r: any) => <div className="w-16"><Progress percent={r.memory_usage || 46} size="small" showInfo={false} strokeWidth={4} strokeColor="#10b981" /></div> },
                { title: 'Pod', key: 'pods', render: (_, r: any) => <span className="text-[11px] text-[#6b7280]">{r.pod_count || '98'}/110</span> },
                { title: '状态', dataIndex: 'status', key: 'status', render: (s) => <Badge status={s === 'Ready' || s === 'active' ? 'success' : 'error'} text={s || 'Ready'} className="text-[12px]" /> },
              ]}
              dataSource={nodes.slice(0, 5)}
              rowKey="id"
              className="[&_.ant-table-thead>tr>th]:!bg-[#f9fafb] [&_.ant-table-thead>tr>th]:!py-2 [&_.ant-table-tbody>tr>td]:!py-2"
            />
          </Card>
        </Col>
        <Col span={10}>
          <Card 
            title={<span className="text-[14px] font-bold text-[#1f2937]">最近事件</span>} 
            size="small" 
            className="h-full border-[#e8edf3] rounded-[10px] overflow-hidden"
            extra={<Button type="link" size="small" onClick={() => props.onChange?.('events')}>更多 <ArrowRightOutlined className="text-[10px]" /></Button>}
          >
            <div className="flex flex-col gap-3 py-1">
              {events.slice(0, 5).map((event: any, idx: number) => (
                <div key={idx} className="flex gap-3 text-[12px] items-start">
                  <span className="text-[#9ca3af] font-mono whitespace-nowrap">{event.age || '12:20:15'}</span>
                  <div className="flex-1 truncate">
                    <Tag className={`m-0 border-none px-1 py-0 text-[10px] leading-tight ${event.type === 'Normal' ? 'bg-blue-50 text-blue-500' : 'bg-red-50 text-red-500'}`}>{event.type}</Tag>
                    <span className="ml-2 text-[#4b5563]" title={event.message}>{event.message}</span>
                  </div>
                </div>
              ))}
              {events.length === 0 && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无近期事件" className="py-4" />}
            </div>
          </Card>
        </Col>
      </Row>

      {/* Row 3: Workload Summary Cards */}
      <div className="grid grid-cols-7 gap-3">
        {workloadStats.map((stat, idx) => (
          <Card key={idx} size="small" styles={{ body: { padding: '12px' } }} className="border border-[#e8edf3] rounded-[10px] hover:border-blue-500 transition-all cursor-pointer group">
            <div className="text-[#6b7280] text-[11px] mb-1">{stat.type}</div>
            <div className="flex items-end justify-between">
              <span className="text-[20px] font-bold text-[#111827] leading-none">{stat.count}</span>
              <span className={`text-[10px] ${stat.delta.startsWith('+') ? 'text-green-500' : 'text-gray-400'}`}>{stat.delta}</span>
            </div>
            <div className="mt-2 h-1 bg-[#f3f4f6] rounded-full overflow-hidden">
              <div className="h-full rounded-full transition-all group-hover:opacity-80" style={{ width: '60%', backgroundColor: stat.color }} />
            </div>
          </Card>
        ))}
      </div>
    </div>
  );

  return (
    <div className="cluster-detail-content flex flex-col">
      {activeKey === 'overview' ? renderOverview() : (
        <Tabs activeKey={activeKey} onChange={props.onChange} className="hidden-tabs-nav" items={[
          {
            key: 'nodes',
            label: '节点',
            children: (
              <Card title="节点列表" extra={cluster.source === 'platform_managed' && <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddNodeModalVisible(true)}>添加节点</Button>} className="border-[#e8edf3] rounded-[10px]">
                <Table columns={nodeColumns} dataSource={nodes} rowKey="id" loading={nodesLoading} pagination={false} size="small" />
              </Card>
            ),
          },
          {
            key: 'workloads',
            label: '工作负载',
            children: (
              <div className="space-y-4">
                <div className="flex items-center gap-3 bg-gray-50 p-3 rounded-lg border border-[#e8edf3]">
                  <span className="text-[13px] text-[#6b7280]">命名空间:</span>
                  <Select style={{ width: 220 }} value={selectedNamespace} onChange={setSelectedNamespace} options={namespaces.map((ns: any) => ({ label: ns.name, value: ns.name }))} loading={resourceLoading} />
                </div>
                <Spin spinning={resourceLoading}>
                  <Space direction="vertical" size={16} className="w-full">
                    <Card title="Deployments" size="small" className="border-[#e8edf3] rounded-[10px]">
                      <Table
                        columns={buildWorkloadColumns('deployment', handleDeploymentRestart, handleDeploymentScale, handleDeploymentDelete)}
                        dataSource={deployments}
                        rowKey="name"
                        pagination={false}
                        size="small"
                      />
                    </Card>
                    <Card title="StatefulSets" size="small" className="border-[#e8edf3] rounded-[10px]">
                      <Table
                        columns={buildWorkloadColumns('statefulset', handleStatefulSetRestart, handleStatefulSetScale, handleStatefulSetDelete)}
                        dataSource={statefulsets}
                        rowKey="name"
                        pagination={false}
                        size="small"
                      />
                    </Card>
                  </Space>
                </Spin>
              </div>
            ),
          },
          {
            key: 'namespaces',
            label: '命名空间',
            children: (
              <Card title="命名空间" className="border-[#e8edf3] rounded-[10px]">
                <Table 
                  dataSource={namespaces} 
                  rowKey="name" 
                  size="small"
                  columns={[
                    { title: '名称', dataIndex: 'name', key: 'name', render: (t) => <span className="font-medium text-blue-600">{t}</span> },
                    { title: '状态', dataIndex: 'status', key: 'status', render: (s) => <Tag color={s === 'Active' ? 'green' : 'red'}>{s}</Tag> },
                    { title: '创建时间', dataIndex: 'creation_timestamp', key: 'creation_timestamp' },
                  ]} 
                />
              </Card>
            )
          },
          {
            key: 'monitor',
            label: '监控',
            children: (
              <div className="flex flex-col items-center justify-center py-20 bg-gray-50 rounded-xl border border-dashed border-gray-300">
                <MonitorOutlined className="text-4xl text-gray-300 mb-4" />
                <Title level={5} className="!text-gray-400">监控视图集成中</Title>
                <Text type="secondary">Prometheus / Grafana 面板将在此处展示</Text>
              </div>
            )
          },
          {
            key: 'events',
            label: '事件',
            children: (
              <Card title="集群事件" extra={<Button icon={<ReloadOutlined />} onClick={loadEvents}>刷新</Button>} className="border-[#e8edf3] rounded-[10px]">
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
            key: 'alerts',
            label: '告警',
            children: (
              <div className="flex flex-col items-center justify-center py-20 bg-gray-50 rounded-xl border border-dashed border-gray-300">
                <AlertOutlined className="text-4xl text-gray-300 mb-4" />
                <Title level={5} className="!text-gray-400">告警管理集成中</Title>
                <Text type="secondary">当前集群的实时告警列表</Text>
              </div>
            )
          },
          {
            key: 'audit',
            label: '操作记录',
            children: (
              <Card title="操作记录" className="border-[#e8edf3] rounded-[10px]">
                <Empty description="暂无操作记录" />
              </Card>
            )
          },
        ]} />
      )}
      <style>{`
        .hidden-tabs-nav .ant-tabs-nav { display: none; }
        .ant-table-thead > tr > th { background: #f9fafb !important; }
      `}</style>
    </div>
  );
}
