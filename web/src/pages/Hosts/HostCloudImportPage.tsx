import React, { useEffect, useState } from 'react';
import { Button, Card, Checkbox, Form, Input, Modal, Select, Space, Table, Tag, message, Popconfirm, Tooltip } from 'antd';
import { DeleteOutlined, QuestionCircleOutlined, ReloadOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../api';
import type { CloudAccount, CloudInstance, CloudProviderInfo } from '../../api/modules/hosts';

// 云厂商选项
const providerOptions = [
  { value: 'volcengine', label: '火山云' },
  { value: 'alicloud', label: '阿里云' },
  { value: 'ucloud', label: 'UCLOUD' },
  { value: 'tencent', label: '腾讯云' },
];

// 产品类型选项（根据云厂商动态变化）
const productTypeOptions: Record<string, { value: string; label: string }[]> = {
  ucloud: [
    { value: 'uhost', label: '云服务器' },
    { value: 'ulighthost', label: '轻量应用云主机' },
  ],
  alicloud: [
    { value: 'ecs', label: '云服务器 ECS' },
  ],
  volcengine: [
    { value: 'ecs', label: '云服务器' },
  ],
  tencent: [
    { value: 'cvm', label: '云服务器' },
  ],
};

// 地域和可用区类型
interface RegionInfo {
  regionId: string;
  localName: string;
}

interface ZoneInfo {
  zoneId: string;
  localName: string;
}

const HostCloudImportPage: React.FC = () => {
  const navigate = useNavigate();
  const [accounts, setAccounts] = useState<CloudAccount[]>([]);
  const [providers, setProviders] = useState<CloudProviderInfo[]>([]);
  const [instances, setInstances] = useState<CloudInstance[]>([]);
  const [selected, setSelected] = useState<React.Key[]>([]);
  const [loading, setLoading] = useState(false);
  const [querying, setQuerying] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [regions, setRegions] = useState<RegionInfo[]>([]);
  const [zones, setZones] = useState<ZoneInfo[]>([]);
  const [loadingRegions, setLoadingRegions] = useState(false);
  const [loadingZones, setLoadingZones] = useState(false);
  const [importing, setImporting] = useState(false);
  const [confirmModalVisible, setConfirmModalVisible] = useState(false);
  const [resultModalVisible, setResultModalVisible] = useState(false);
  const [importedHosts, setImportedHosts] = useState<any[]>([]);
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
      await Api.hosts.createCloudAccount({
        provider: values.provider,
        productType: values.productType,
        accountName: values.accountName,
        accessKeyId: values.accessKeyId,
        accessKeySecret: values.accessKeySecret,
        regionDefault: values.regionDefault,
        // UCloud 额外配置
        projectId: values.projectId,
        isIntl: values.isIntl,
      });
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

  // 选择账号后加载地域列表
  const handleAccountChange = async (accountId: string) => {
    const acc = accounts.find((a) => a.id === accountId);
    if (acc) {
      queryForm.setFieldsValue({
        provider: acc.provider,
        region: undefined,
        zone: undefined,
      });
      setRegions([]);
      setZones([]);

      // 加载地域列表
      setLoadingRegions(true);
      try {
        const res = await Api.hosts.listCloudRegions(acc.provider, accountId);
        const list = Array.isArray(res.data) ? res.data : (res.data as any)?.list || [];
        setRegions(list);
      } catch (err: any) {
        console.error('加载地域失败:', err);
      } finally {
        setLoadingRegions(false);
      }
    }
  };

  // 选择地域后加载可用区列表
  const handleRegionChange = async (region: string) => {
    const provider = queryForm.getFieldValue('provider');
    const accountId = queryForm.getFieldValue('accountId');
    queryForm.setFieldsValue({ zone: undefined });
    setZones([]);

    if (!region || !accountId) return;

    setLoadingZones(true);
    try {
      const res = await Api.hosts.listCloudZones(provider, accountId, region);
      const list = Array.isArray(res.data) ? res.data : (res.data as any)?.list || [];
      setZones(list);
    } catch (err: any) {
      console.error('加载可用区失败:', err);
    } finally {
      setLoadingZones(false);
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

  // 点击导入按钮，显示确认弹窗
  const handleImportClick = () => {
    if (selected.length === 0) {
      message.warning('请选择要导入的实例');
      return;
    }
    setConfirmModalVisible(true);
  };

  // 确认导入
  const confirmImport = async () => {
    const values = await queryForm.validateFields();
    const picked = instances.filter((x) => selected.includes(x.instanceId));

    setConfirmModalVisible(false);
    setImporting(true);

    try {
      const res = await Api.hosts.importCloudInstances({
        provider: values.provider,
        accountId: Number(values.accountId),
        instances: picked,
        role: values.role || '',
        labels: values.labels ? String(values.labels).split(',').map((x) => x.trim()).filter(Boolean) : [],
      });

      const result = res.data || {};
      const importedCount = result.nodes?.length || 0;
      const skippedCount = result.skipped?.length || 0;

      if (importedCount > 0) {
        message.success(`成功导入 ${importedCount} 台主机${skippedCount > 0 ? `，跳过 ${skippedCount} 台已存在` : ''}`);
      } else if (skippedCount > 0) {
        message.warning(`全部 ${skippedCount} 台实例已存在，未导入新主机`);
      }

      setSelected([]);
      setImportedHosts(result.nodes || []);
      setResultModalVisible(true);
    } catch (err: any) {
      message.error(err?.message || '导入失败');
    } finally {
      setImporting(false);
    }
  };

  // 获取选中的实例列表
  const getSelectedInstances = () => {
    return instances.filter((x) => selected.includes(x.instanceId));
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
          <Form.Item shouldUpdate>
            {({ getFieldValue }) => {
              const provider = getFieldValue('provider');
              const options = productTypeOptions[provider] || [];
              if (options.length <= 1) return null;
              return (
                <Form.Item name="productType" rules={[{ required: true, message: '请选择产品类型' }]}>
                  <Select style={{ width: 140 }} placeholder="产品类型" options={options} />
                </Form.Item>
              );
            }}
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
          <Form.Item shouldUpdate>
            {({ getFieldValue }) => {
              const provider = getFieldValue('provider');
              if (provider !== 'ucloud') return null;
              return (
                <>
                  <Form.Item name="projectId">
                    <Input
                      placeholder="项目 ID（子账户必填）"
                      style={{ width: 140 }}
                    />
                  </Form.Item>
                  <Form.Item name="isIntl" valuePropName="checked">
                    <Checkbox>
                      <Tooltip title="使用国际版 API 端点（api.intl.ucloud.cn）。国内站账号通常无需勾选，国际站账号需勾选。">
                        国际版 <QuestionCircleOutlined style={{ color: '#999' }} />
                      </Tooltip>
                    </Checkbox>
                  </Form.Item>
                </>
              );
            }}
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
              {
                title: '产品类型',
                dataIndex: 'productType',
                width: 120,
                render: (v, record) => {
                  const label = productTypeOptions[record.provider]?.find(o => o.value === v)?.label || v || '云服务器';
                  return <Tag>{label}</Tag>;
                },
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
            <Button type="primary" onClick={handleImportClick} disabled={selected.length === 0} loading={importing}>
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
          <Form.Item name="region" rules={[{ required: true, message: '请选择地域' }]}>
            <Select
              style={{ width: 180 }}
              placeholder="选择地域"
              loading={loadingRegions}
              options={regions.map((r) => ({ value: r.regionId, label: r.localName || r.regionId }))}
              onChange={handleRegionChange}
              showSearch
              optionFilterProp="label"
            />
          </Form.Item>
          <Form.Item name="zone">
            <Select
              style={{ width: 200 }}
              placeholder="可用区（可选）"
              loading={loadingZones}
              options={zones.map((z) => ({ value: z.zoneId, label: z.localName || z.zoneId }))}
              allowClear
            />
          </Form.Item>
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
          loading={querying || importing}
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

      {/* 确认导入弹窗 */}
      <Modal
        title="确认导入"
        open={confirmModalVisible}
        onCancel={() => setConfirmModalVisible(false)}
        onOk={confirmImport}
        okText="确认导入"
        cancelText="取消"
        width={600}
      >
        <p>即将导入 <strong>{selected.length}</strong> 台实例：</p>
        <div style={{ maxHeight: 300, overflow: 'auto', marginTop: 16 }}>
          <Table
            size="small"
            dataSource={getSelectedInstances().map((x) => ({
              key: x.instanceId,
              name: x.name,
              ip: x.ip,
              region: x.region,
              status: x.status,
            }))}
            columns={[
              { title: '名称', dataIndex: 'name', ellipsis: true },
              { title: 'IP', dataIndex: 'ip', width: 130 },
              { title: '地域', dataIndex: 'region', width: 100 },
              { title: '状态', dataIndex: 'status', width: 80, render: (v) => (
                <Tag color={v === 'running' ? 'green' : 'default'}>{v}</Tag>
              )},
            ]}
            pagination={false}
          />
        </div>
      </Modal>

      {/* 导入结果弹窗 */}
      <Modal
        title="导入成功"
        open={resultModalVisible}
        onCancel={() => setResultModalVisible(false)}
        footer={[
          <Button key="continue" onClick={() => setResultModalVisible(false)}>
            继续导入
          </Button>,
          <Button key="detail" type="primary" onClick={() => {
            setResultModalVisible(false);
            navigate('/hosts');
          }}>
            查看主机列表
          </Button>,
        ]}
        width={600}
      >
        <p>成功导入 <strong>{importedHosts.length}</strong> 台主机</p>
        {importedHosts.length > 0 && (
          <div style={{ maxHeight: 300, overflow: 'auto', marginTop: 16 }}>
            <Table
              size="small"
              dataSource={importedHosts.map((h: any) => ({
                key: h.id,
                name: h.name,
                ip: h.ip,
                status: h.status,
              }))}
              columns={[
                { title: '主机名', dataIndex: 'name', ellipsis: true },
                { title: 'IP', dataIndex: 'ip', width: 130 },
                { title: '状态', dataIndex: 'status', width: 80, render: (v) => <Tag color="green">{v}</Tag> },
              ]}
              pagination={false}
            />
          </div>
        )}
      </Modal>
    </Space>
  );
};

export default HostCloudImportPage;
