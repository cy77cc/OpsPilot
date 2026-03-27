import React, { useEffect, useState } from 'react';
import { Button, Card, Form, Input, Modal, Select, Space, Table, Tag, message } from 'antd';
import { Api } from '../../api';
import type { CMDBAsset } from '../../api/modules/cmdb';

const CMDBPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [assets, setAssets] = useState<CMDBAsset[]>([]);
  const [open, setOpen] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [relationCount, setRelationCount] = useState(0);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const [res, rel] = await Promise.all([Api.cmdb.listAssets(), Api.cmdb.listRelations()]);
      setAssets(res.data || []);
      setRelationCount((rel.data || []).length);
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

  return (
    <Card
      title="CMDB 资产台账"
      extra={
        <Space>
          <Tag color="blue">关系数: {relationCount}</Tag>
          <Button loading={syncing} onClick={syncNow}>同步资产</Button>
          <Button type="primary" onClick={() => setOpen(true)}>新增资产</Button>
        </Space>
      }
    >
      <Table
        rowKey="id"
        loading={loading}
        dataSource={assets}
        columns={[
          { title: 'ID', dataIndex: 'id', width: 80 },
          { title: '名称', dataIndex: 'name' },
          { title: '类型', dataIndex: 'assetType' },
          { title: '来源', dataIndex: 'source' },
          { title: '状态', dataIndex: 'status' },
          { title: 'Owner', dataIndex: 'owner' },
        ]}
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
