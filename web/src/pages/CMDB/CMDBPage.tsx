import React, { useEffect, useState } from 'react';
import { Button, Card, Form, Input, Modal, Select, Space, Table, Tag, message, Layout, theme, Drawer, Empty, Segmented } from 'antd';
import { TableOutlined, ClusterOutlined } from '@ant-design/icons';
import { Api } from '../../api';
import type { CMDBAsset, CMDBTopologyData } from '../../api/modules/cmdb';
import { TableSkeleton } from '../../components/LoadingSkeleton';
import AssetTree from './components/AssetTree';
import TopologyGraph from './components/TopologyGraph';
import AssetDetailDrawer from './components/AssetDetailDrawer';

const { Sider, Content } = Layout;

const CMDBPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [assets, setAssets] = useState<CMDBAsset[]>([]);
  const [topologyData, setTopologyData] = useState<CMDBTopologyData>({ nodes: [], edges: [] });
  const [viewMode, setViewMode] = useState<'table' | 'topology'>('table');
  const [open, setOpen] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [relationCount, setRelationCount] = useState(0);
  const [form] = Form.useForm();
  
  const [selectedCIID, setSelectedCIID] = useState<string | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken();

  const load = async () => {
    setLoading(true);
    try {
      const [res, rel, topo] = await Promise.all([
        Api.cmdb.listAssets(), 
        Api.cmdb.listRelations(),
        Api.cmdb.getTopology()
      ]);
      setAssets(res.data || []);
      setRelationCount((rel.data || []).length);
      // Ensure topo.data has the right format for CMDBTopologyData
      setTopologyData({
        nodes: (topo.data?.nodes || []).map((n: any) => ({
          id: String(n.id),
          label: n.name,
          type: n.ci_type || n.asset_type || 'default',
          status: n.status,
        })),
        edges: (topo.data?.edges || []).map((e: any) => ({
          id: String(e.id),
          source: String(e.from_ci_id || e.from_asset_id),
          target: String(e.to_ci_id || e.to_asset_id),
          relation_type: e.relation_type,
        })),
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const create = async () => {
    const values = await form.validateFields();
    await Api.cmdb.createAsset({
      assetType: values.assetType,
      name: values.name,
      owner: values.owner,
      source: values.source,
    });
    message.success('资产创建成功');
    setOpen(false);
    form.resetFields();
    load();
  };

  const syncNow = async () => {
    setSyncing(true);
    try {
      await Api.cmdb.triggerSync();
      message.success('已触发 CMDB 同步');
      await load();
    } finally {
      setSyncing(false);
    }
  };

  const handleSelectCI = (ciId: string | null) => {
    setSelectedCIID(ciId);
    if (ciId) {
      setDrawerOpen(true);
    }
  };

  const handleNodeDoubleClick = async (id: string) => {
    message.info(`正在获取节点 ${id} 的子拓扑...`);
    try {
        const res = await Api.cmdb.getTopologySubgraph({ rootId: parseInt(id), depth: 2 });
        if (res.data) {
            // merge or replace topology data
            setTopologyData(res.data);
        }
    } catch (e) {
        message.error('获取拓扑数据失败');
    }
  };

  const isInitialLoading = loading && assets.length === 0;

  return (
    <Card
      title="CMDB 资产管理"
      styles={{ body: { padding: 0 } }}
      extra={
        <Space>
          <Segmented
            options={[
              { label: '列表', value: 'table', icon: <TableOutlined /> },
              { label: '拓扑', value: 'topology', icon: <ClusterOutlined /> },
            ]}
            value={viewMode}
            onChange={(value) => setViewMode(value as 'table' | 'topology')}
          />
          <Tag color="blue">关系数: {relationCount}</Tag>
          <Button loading={syncing} onClick={syncNow}>同步资产</Button>
          <Button type="primary" onClick={() => setOpen(true)}>新增资产</Button>
        </Space>
      }
    >
      <Layout style={{ background: colorBgContainer, minHeight: 'calc(100vh - 200px)' }}>
        <Sider
          width={280}
          style={{
            background: colorBgContainer,
            borderRight: '1px solid #f0f0f0',
            overflow: 'auto',
          }}
        >
          <div style={{ padding: '16px 0' }}>
            <AssetTree onSelect={handleSelectCI} />
          </div>
        </Sider>
        <Content style={{ padding: '16px', minHeight: 280, background: '#fff', position: 'relative' }}>
          {isInitialLoading ? (
            <TableSkeleton toolbar={false} rows={8} columns={6} />
          ) : viewMode === 'table' ? (
            <Table
              rowKey="id"
              loading={loading}
              dataSource={assets}
              size="small"
              columns={[
                { title: 'ID', dataIndex: 'id', width: 80 },
                { title: '名称', dataIndex: 'name' },
                { title: '类型', dataIndex: 'assetType' },
                { title: '来源', dataIndex: 'source' },
                { title: '状态', dataIndex: 'status' },
                { title: 'Owner', dataIndex: 'owner' },
              ]}
              onRow={(record) => ({
                onClick: () => handleSelectCI(record.id),
                style: { cursor: 'pointer' },
              })}
              rowClassName={(record) => record.id === selectedCIID ? 'ant-table-row-selected' : ''}
            />
          ) : (
            <div style={{ height: 'calc(100vh - 300px)' }}>
              <TopologyGraph 
                data={topologyData} 
                selectedCIID={selectedCIID}
                onNodeSelect={handleSelectCI}
                onNodeDoubleClick={handleNodeDoubleClick}
              />
            </div>
          )}
        </Content>
      </Layout>

      <AssetDetailDrawer 
        ciId={selectedCIID} 
        visible={drawerOpen} 
        onClose={() => setDrawerOpen(false)} 
      />

      <Modal title="新增资产" open={open} onCancel={() => setOpen(false)} onOk={create}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="资产名称" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="assetType" label="资产类型" rules={[{ required: true }]}><Select options={[{ value: 'host' }, { value: 'service' }, { value: 'cluster' }, { value: 'custom' }]} /></Form.Item>
          <Form.Item name="source" label="来源" initialValue="manual"><Select options={[{ value: 'manual' }, { value: 'system' }]} /></Form.Item>
          <Form.Item name="owner" label="负责人"><Input /></Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default CMDBPage;
