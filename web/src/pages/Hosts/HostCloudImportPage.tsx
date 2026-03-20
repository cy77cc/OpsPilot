import React, { useEffect, useState } from 'react';
import { Button, Card, Form, Input, Select, Space, Table, Tag, message, Popconfirm } from 'antd';
import { DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import { Api } from '../../api';
import type { CloudAccount, CloudInstance, CloudProviderInfo } from '../../api/modules/hosts';

// 云厂商选项
const providerOptions = [
  { value: 'volcengine', label: '火山云' },
  { value: 'alicloud', label: '阿里云' },
  { value: 'tencent', label: '腾讯云' },
];

// 火山云可用区选项
const volcengineZoneOptions = [
  { value: '', label: '全部可用区' },
  { value: 'cn-beijing-a', label: '华北2（北京）- 可用区A' },
  { value: 'cn-beijing-b', label: '华北2（北京）- 可用区B' },
  { value: 'cn-shanghai-a', label: '华东2（上海）- 可用区A' },
  { value: 'cn-shanghai-b', label: '华东2（上海）- 可用区B' },
  { value: 'cn-guangzhou-a', label: '华南1（广州）- 可用区A' },
];

const HostCloudImportPage: React.FC = () => {
  const [accounts, setAccounts] = useState<CloudAccount[]>([]);
  const [providers, setProviders] = useState<CloudProviderInfo[]>([]);
  const [instances, setInstances] = useState<CloudInstance[]>([]);
  const [selected, setSelected] = useState<React.Key[]>([]);
  const [loading, setLoading] = useState(false);
  const [querying, setQuerying] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [accountForm] = Form.useForm();
  const [queryForm] = Form.useForm();

  // 加载云账号列表
  const loadAccounts = async () => {
    setLoading(true);
    try {
      const res = await Api.hosts.listCloudAccounts();
      setAccounts(res.data || []);
    } catch (err: any) {
      console.error('加载云账号失败:', err);
      // 如果是认证错误，不显示错误消息（由全局拦截器处理）
      if (err?.code !== 2003) {
        message.error(err?.message || '加载云账号失败');
      }
    } finally {
      setLoading(false);
    }
  };

  // 加载云厂商列表
  const loadProviders = async () => {
    try {
      const res = await Api.hosts.listCloudProviders();
      setProviders(res.data || []);
    } catch {
      setProviders(providerOptions.map((x) => ({ name: x.value, displayName: x.label })));
    }
  };

  useEffect(() => {
    loadAccounts();
    loadProviders();
  }, []);

  // 创建云账号
  const createAccount = async () => {
    const values = await accountForm.validateFields();
    try {
      await Api.hosts.createCloudAccount(values);
      message.success('云账号创建成功');
      accountForm.resetFields();
      loadAccounts();
    } catch (err: any) {
      message.error(err?.message || '创建失败');
    }
  };

  // 删除云账号
  const deleteAccount = async (accountId: string) => {
    setDeleting(accountId);
    try {
      await Api.hosts.deleteCloudAccount(accountId);
      message.success('删除成功');
      loadAccounts();
    } catch (err: any) {
      message.error(err?.message || '删除失败');
    } finally {
      setDeleting(null);
    }
  };

  // 选择账号后自动填充 region
  const handleAccountChange = (accountId: string) => {
    const acc = accounts.find((a) => a.id === accountId);
    if (acc) {
      queryForm.setFieldsValue({
        provider: acc.provider,
        region: acc.regionDefault || '',
        zone: '',
      });
    }
  };

  // 查询实例
  const queryInstances = async () => {
    const values = await queryForm.validateFields();
    setQuerying(true);
    try {
      const res = await Api.hosts.queryCloudInstances({
        provider: values.provider,
        accountId: Number(values.accountId),
        region: values.region || undefined,
        zone: values.zone || undefined,
        keyword: values.keyword || undefined,
      });
      setInstances(res.data || []);
      setSelected([]);
      if ((res.data || []).length === 0) {
        message.info('未查询到实例，请检查地域/可用区是否正确');
      }
    } catch (err: any) {
      message.error(err?.message || '查询失败');
    } finally {
      setQuerying(false);
    }
  };

  // 导入选中实例
  const importSelected = async () => {
    const values = await queryForm.validateFields();
    const picked = instances.filter((x) => selected.includes(x.instanceId));
    if (picked.length === 0) {
      message.warning('请选择要导入的实例');
      return;
    }
    try {
      const res = await Api.hosts.importCloudInstances({
        provider: values.provider,
        accountId: Number(values.accountId),
        instances: picked,
        role: values.role || '',
        labels: values.labels ? String(values.labels).split(',').map((x) => x.trim()).filter(Boolean) : [],
      });
      message.success(`导入成功，任务ID: ${res.data?.task?.id || '-'}`);
      setSelected([]);
    } catch (err: any) {
      message.error(err?.message || '导入失败');
    }
  };

  // 获取云厂商显示名称
  const getProviderLabel = (name: string) => {
    const found = providers.find((p) => p.name === name);
    if (found) return found.displayName;
    const staticOption = providerOptions.find((o) => o.value === name);
    return staticOption?.label || name;
  };

  // 账号下拉选项（按云厂商分组）
  const accountOptions = providerOptions.map((p) => ({
    label: p.label,
    options: accounts
      .filter((a) => a.provider === p.value)
      .map((a) => ({
        label: `${a.accountName}${a.regionDefault ? ` (${a.regionDefault})` : ''}`,
        value: a.id,
      })),
  })).filter((g) => g.options.length > 0);

  // 获取当前选择的 provider
  const currentProvider = Form.useWatch('provider', queryForm);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {/* 云账号管理 */}
      <Card
        title="云账号管理"
        extra={<Button icon={<ReloadOutlined />} onClick={loadAccounts} loading={loading}>刷新</Button>}
      >
        <Form form={accountForm} layout="inline" initialValues={{ provider: 'volcengine' }}>
          <Form.Item name="provider" rules={[{ required: true }]}>
            <Select style={{ width: 120 }} options={providerOptions} />
          </Form.Item>
          <Form.Item name="accountName" rules={[{ required: true }]}>
            <Input placeholder="账号名称" style={{ width: 140 }} />
          </Form.Item>
          <Form.Item name="accessKeyId" rules={[{ required: true }]}>
            <Input placeholder="AccessKey ID" style={{ width: 180 }} />
          </Form.Item>
          <Form.Item name="accessKeySecret" rules={[{ required: true }]}>
            <Input.Password placeholder="AccessKey Secret" style={{ width: 180 }} />
          </Form.Item>
          <Form.Item name="regionDefault">
            <Input placeholder="默认地域（如 cn-beijing）" style={{ width: 180 }} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" onClick={createAccount}>添加账号</Button>
          </Form.Item>
        </Form>

        {/* 已有账号列表 */}
        {accounts.length > 0 && (
          <Table
            size="small"
            style={{ marginTop: 16 }}
            dataSource={accounts}
            rowKey="id"
            pagination={false}
            columns={[
              {
                title: '云厂商',
                dataIndex: 'provider',
                width: 100,
                render: (v) => <Tag color={v === 'volcengine' ? 'orange' : 'blue'}>{getProviderLabel(v)}</Tag>,
              },
              { title: '账号名称', dataIndex: 'accountName', width: 150 },
              { title: 'AccessKey ID', dataIndex: 'accessKeyId', width: 200, ellipsis: true },
              { title: '默认地域', dataIndex: 'regionDefault', width: 120 },
              {
                title: '状态',
                dataIndex: 'status',
                width: 80,
                render: (v) => <Tag color={v === 'active' ? 'green' : 'default'}>{v || 'active'}</Tag>,
              },
              {
                title: '操作',
                width: 80,
                render: (_, record) => (
                  <Popconfirm
                    title="确定删除该云账号？"
                    description="删除后无法恢复"
                    onConfirm={() => deleteAccount(record.id)}
                  >
                    <Button
                      type="link"
                      danger
                      size="small"
                      icon={<DeleteOutlined />}
                      loading={deleting === record.id}
                    />
                  </Popconfirm>
                ),
              },
            ]}
          />
        )}
      </Card>

      {/* 实例查询与导入 */}
      <Card
        title="实例查询与导入"
        extra={
          <Space>
            <span style={{ color: '#999', fontSize: 12 }}>
              已选 {selected.length} 个实例
            </span>
            <Button type="primary" onClick={importSelected} disabled={selected.length === 0}>
              导入选中实例
            </Button>
          </Space>
        }
      >
        <Form form={queryForm} layout="inline">
          <Form.Item name="accountId" rules={[{ required: true, message: '请选择账号' }]}>
            <Select
              style={{ width: 240 }}
              placeholder="选择云账号"
              options={accountOptions}
              onChange={handleAccountChange}
              showSearch
              optionFilterProp="label"
            />
          </Form.Item>
          <Form.Item name="provider" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="region">
            <Input placeholder="地域（如 cn-beijing）" style={{ width: 150 }} />
          </Form.Item>
          {currentProvider === 'volcengine' && (
            <Form.Item name="zone">
              <Select
                style={{ width: 200 }}
                placeholder="可用区（可选）"
                options={volcengineZoneOptions}
                allowClear
              />
            </Form.Item>
          )}
          <Form.Item name="keyword">
            <Input placeholder="关键词过滤" style={{ width: 120 }} />
          </Form.Item>
          <Form.Item name="role">
            <Input placeholder="导入角色" style={{ width: 100 }} />
          </Form.Item>
          <Form.Item name="labels">
            <Input placeholder="标签（逗号分隔）" style={{ width: 130 }} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" onClick={queryInstances} loading={querying}>查询实例</Button>
          </Form.Item>
        </Form>

        {/* 实例列表 */}
        <Table
          rowKey="instanceId"
          loading={querying}
          rowSelection={{
            selectedRowKeys: selected,
            onChange: setSelected,
            selections: [Table.SELECTION_ALL, Table.SELECTION_INVERT, Table.SELECTION_NONE],
          }}
          dataSource={instances}
          style={{ marginTop: 16 }}
          pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total) => `共 ${total} 条` }}
          columns={[
            { title: '实例ID', dataIndex: 'instanceId', width: 150, ellipsis: true },
            { title: '名称', dataIndex: 'name', width: 150, ellipsis: true },
            { title: 'IP', dataIndex: 'ip', width: 130 },
            { title: '地域', dataIndex: 'region', width: 100 },
            {
              title: '状态',
              dataIndex: 'status',
              width: 80,
              render: (v) => (
                <Tag color={v === 'running' ? 'green' : v === 'stopped' ? 'default' : 'orange'}>
                  {v}
                </Tag>
              ),
            },
            { title: '系统', dataIndex: 'os', width: 140, ellipsis: true },
            { title: 'CPU', dataIndex: 'cpu', width: 60, align: 'right' },
            { title: '内存', dataIndex: 'memoryMB', width: 80, align: 'right', render: (v) => `${v} MB` },
            { title: '磁盘', dataIndex: 'diskGB', width: 70, align: 'right', render: (v) => `${v} GB` },
          ]}
          locale={{ emptyText: accounts.length === 0 ? '请先添加云账号' : '暂无实例，请选择账号后查询' }}
        />
      </Card>
    </Space>
  );
};

export default HostCloudImportPage;
