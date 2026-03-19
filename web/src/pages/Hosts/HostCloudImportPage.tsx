import React, { useEffect, useState } from 'react';
import { Button, Card, Form, Input, Select, Space, Table, Tag, message } from 'antd';
import { Api } from '../../api';
import type { CloudAccount, CloudInstance, CloudProviderInfo } from '../../api/modules/hosts';

// 云厂商选项（静态定义，可替换为动态获取）
const providerOptions = [
  { value: 'volcengine', label: '火山云' },
  { value: 'alicloud', label: '阿里云' },
  { value: 'tencent', label: '腾讯云' },
];

const HostCloudImportPage: React.FC = () => {
  const [accounts, setAccounts] = useState<CloudAccount[]>([]);
  const [providers, setProviders] = useState<CloudProviderInfo[]>([]);
  const [instances, setInstances] = useState<CloudInstance[]>([]);
  const [selected, setSelected] = useState<React.Key[]>([]);
  const [loading, setLoading] = useState(false);
  const [accountForm] = Form.useForm();
  const [queryForm] = Form.useForm();

  const loadAccounts = async () => {
    const res = await Api.hosts.listCloudAccounts();
    setAccounts(res.data || []);
  };

  const loadProviders = async () => {
    try {
      const res = await Api.hosts.listCloudProviders();
      setProviders(res.data || []);
    } catch {
      // 如果动态获取失败，使用静态定义
      setProviders(providerOptions.map((x) => ({ name: x.value, displayName: x.label })));
    }
  };

  useEffect(() => {
    loadAccounts();
    loadProviders();
  }, []);

  const createAccount = async () => {
    const values = await accountForm.validateFields();
    await Api.hosts.createCloudAccount(values);
    message.success('云账号创建成功');
    accountForm.resetFields();
    loadAccounts();
  };

  const queryInstances = async () => {
    const values = await queryForm.validateFields();
    setLoading(true);
    try {
      const res = await Api.hosts.queryCloudInstances(values);
      setInstances(res.data || []);
    } finally {
      setLoading(false);
    }
  };

  const importSelected = async () => {
    const values = await queryForm.validateFields();
    const picked = instances.filter((x) => selected.includes(x.instanceId));
    if (picked.length === 0) {
      message.warning('请选择实例');
      return;
    }
    const res = await Api.hosts.importCloudInstances({
      provider: values.provider,
      accountId: Number(values.accountId),
      instances: picked,
      role: values.role || '',
      labels: values.labels ? String(values.labels).split(',').map((x) => x.trim()).filter(Boolean) : [],
    });
    message.success(`导入完成，任务ID: ${res.data?.task?.id || '-'}`);
    setSelected([]);
  };

  // 获取云厂商显示名称
  const getProviderLabel = (name: string) => {
    const found = providers.find((p) => p.name === name);
    if (found) return found.displayName;
    const staticOption = providerOptions.find((o) => o.value === name);
    return staticOption?.label || name;
  };

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card title="云账号管理">
        <Form form={accountForm} layout="inline" initialValues={{ provider: 'volcengine' }}>
          <Form.Item name="provider" rules={[{ required: true }]}>
            <Select style={{ width: 120 }} options={providerOptions} />
          </Form.Item>
          <Form.Item name="accountName" rules={[{ required: true }]}><Input placeholder="账号名称" /></Form.Item>
          <Form.Item name="accessKeyId" rules={[{ required: true }]}><Input placeholder="AccessKeyId" /></Form.Item>
          <Form.Item name="accessKeySecret" rules={[{ required: true }]}><Input.Password placeholder="AccessKeySecret" /></Form.Item>
          <Form.Item name="regionDefault"><Input placeholder="默认地域" /></Form.Item>
          <Form.Item><Button type="primary" onClick={createAccount}>新增账号</Button></Form.Item>
        </Form>
        <div style={{ marginTop: 12 }}>
          {accounts.map((acc) => (
            <Tag key={acc.id}>{getProviderLabel(acc.provider)}:{acc.accountName} ({acc.regionDefault || '-'})</Tag>
          ))}
        </div>
      </Card>

      <Card title="实例查询与导入" extra={<Button type="primary" onClick={importSelected}>导入选中实例</Button>}>
        <Form form={queryForm} layout="inline" initialValues={{ provider: 'volcengine' }}>
          <Form.Item name="provider" rules={[{ required: true }]}>
            <Select style={{ width: 120 }} options={providerOptions} />
          </Form.Item>
          <Form.Item name="accountId" rules={[{ required: true }]}><Input placeholder="账号ID" /></Form.Item>
          <Form.Item name="region"><Input placeholder="地域(可选)" /></Form.Item>
          <Form.Item name="keyword"><Input placeholder="关键词" /></Form.Item>
          <Form.Item name="role"><Input placeholder="导入角色(可选)" /></Form.Item>
          <Form.Item name="labels"><Input placeholder="标签,逗号分隔" /></Form.Item>
          <Form.Item><Button onClick={queryInstances} loading={loading}>查询实例</Button></Form.Item>
        </Form>

        <Table
          rowKey="instanceId"
          loading={loading}
          rowSelection={{ selectedRowKeys: selected, onChange: setSelected }}
          dataSource={instances}
          style={{ marginTop: 16 }}
          columns={[
            { title: '实例ID', dataIndex: 'instanceId' },
            { title: '名称', dataIndex: 'name' },
            { title: 'IP', dataIndex: 'ip' },
            { title: '地域', dataIndex: 'region' },
            { title: '状态', dataIndex: 'status' },
            { title: '系统', dataIndex: 'os' },
            { title: 'CPU', dataIndex: 'cpu' },
            { title: '内存MB', dataIndex: 'memoryMB' },
            { title: '磁盘GB', dataIndex: 'diskGB' },
          ]}
        />
      </Card>
    </Space>
  );
};

export default HostCloudImportPage;
