import React from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Divider,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Row,
  Segmented,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd';
import {
  Api,
  type AILLMProvider,
  type AILLMProviderCreatePayload,
  type AILLMProviderUpdatePayload,
  type AILLMProviderImportPayload,
} from '../../api';
import { ApiRequestError } from '../../api/api';
import AccessDeniedPage from '../../components/Auth/AccessDeniedPage';
import {
  EditOutlined,
  DeleteOutlined,
  PlusOutlined,
  ReloadOutlined,
  StarFilled,
  StarOutlined,
  UploadOutlined,
  EyeOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';

const { Text, Paragraph } = Typography;

type ProviderType = 'qwen' | 'ark' | 'ollama' | 'openai' | 'minimax';

const providerChoices: Array<{ label: string; value: ProviderType }> = [
  { label: 'Qwen', value: 'qwen' },
  { label: 'Ark', value: 'ark' },
  { label: 'Ollama', value: 'ollama' },
  { label: 'OpenAI', value: 'openai' },
  { label: 'MiniMax', value: 'minimax' },
];

interface ModelFormValues {
  name: string;
  provider: ProviderType;
  model: string;
  base_url: string;
  api_key?: string;
  temperature: number;
  thinking: boolean;
  is_default: boolean;
  is_enabled: boolean;
  sort_order: number;
}

const initialFormValues: ModelFormValues = {
  name: '',
  provider: 'qwen',
  model: '',
  base_url: '',
  api_key: '',
  temperature: 0.7,
  thinking: false,
  is_default: false,
  is_enabled: true,
  sort_order: 0,
};

const AIModelSettingsPage: React.FC = () => {
  const [loading, setLoading] = React.useState(false);
  const [rows, setRows] = React.useState<AILLMProvider[]>([]);
  const [accessDenied, setAccessDenied] = React.useState(false);

  const [drawerOpen, setDrawerOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<AILLMProvider | null>(null);
  const [saving, setSaving] = React.useState(false);

  const [importOpen, setImportOpen] = React.useState(false);
  const [importing, setImporting] = React.useState(false);
  const [previewing, setPreviewing] = React.useState(false);
  const [importMode, setImportMode] = React.useState<'merge' | 'replace'>('merge');
  const [importJson, setImportJson] = React.useState('');
  const [previewRows, setPreviewRows] = React.useState<AILLMProvider[]>([]);

  const [form] = Form.useForm<ModelFormValues>();

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const resp = await Api.ai.listAdminModels();
      setRows(resp.data?.list || []);
      setAccessDenied(false);
    } catch (err) {
      if (err instanceof ApiRequestError && (err.statusCode === 403 || err.businessCode === 2004)) {
        setAccessDenied(true);
      } else {
        message.error(err instanceof Error ? err.message : '加载模型列表失败');
      }
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    void load();
  }, [load]);

  const summary = React.useMemo(() => {
    const total = rows.length;
    const enabled = rows.filter((item) => item.is_enabled).length;
    const defaults = rows.filter((item) => item.is_default).length;
    const thinking = rows.filter((item) => item.thinking).length;
    return { total, enabled, defaults, thinking };
  }, [rows]);

  const openCreate = () => {
    setEditing(null);
    form.setFieldsValue(initialFormValues);
    setDrawerOpen(true);
  };

  const openEdit = (row: AILLMProvider) => {
    setEditing(row);
    form.setFieldsValue({
      name: row.name,
      provider: (row.provider as ProviderType) || 'qwen',
      model: row.model,
      base_url: row.base_url,
      api_key: '',
      temperature: row.temperature,
      thinking: row.thinking,
      is_default: row.is_default,
      is_enabled: row.is_enabled,
      sort_order: row.sort_order,
    });
    setDrawerOpen(true);
  };

  const closeDrawer = () => {
    setDrawerOpen(false);
    setEditing(null);
    form.resetFields();
  };

  const saveModel = async () => {
    const values = await form.validateFields();
    const basePayload: AILLMProviderUpdatePayload = {
      name: values.name.trim(),
      provider: values.provider,
      model: values.model.trim(),
      base_url: values.base_url.trim(),
      temperature: values.temperature,
      thinking: values.thinking,
      is_default: values.is_default,
      is_enabled: values.is_enabled,
      sort_order: values.sort_order,
      ...(values.api_key ? { api_key: values.api_key.trim() } : {}),
    };

    if (!editing && !basePayload.api_key) {
      message.error('新建模型必须填写 API Key');
      return;
    }

    setSaving(true);
    try {
      if (editing) {
        await Api.ai.updateAdminModel(editing.id, basePayload);
        message.success('模型已更新');
      } else {
        const createPayload: AILLMProviderCreatePayload = {
          name: basePayload.name || '',
          provider: basePayload.provider || '',
          model: basePayload.model || '',
          base_url: basePayload.base_url || '',
          api_key: basePayload.api_key || '',
          temperature: basePayload.temperature,
          thinking: basePayload.thinking,
          is_default: basePayload.is_default,
          is_enabled: basePayload.is_enabled,
          sort_order: basePayload.sort_order,
        };
        await Api.ai.createAdminModel(createPayload);
        message.success('模型已创建');
      }
      closeDrawer();
      await load();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '保存模型失败');
    } finally {
      setSaving(false);
    }
  };

  const setDefault = async (row: AILLMProvider) => {
    try {
      await Api.ai.setAdminDefaultModel(row.id);
      message.success(`已设置默认模型：${row.name}`);
      await load();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '设置默认模型失败');
    }
  };

  const remove = async (row: AILLMProvider) => {
    try {
      await Api.ai.deleteAdminModel(row.id);
      message.success('模型已删除');
      await load();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '删除模型失败');
    }
  };

  const parseImportPayload = (): AILLMProviderImportPayload | null => {
    try {
      const parsed = JSON.parse(importJson || '{}') as { providers?: AILLMProviderCreatePayload[] };
      if (!Array.isArray(parsed.providers) || parsed.providers.length === 0) {
        message.error('导入 JSON 需要包含 providers 数组');
        return null;
      }
      return {
        replace_all: importMode === 'replace',
        providers: parsed.providers,
      };
    } catch {
      message.error('JSON 格式无效，请检查后重试');
      return null;
    }
  };

  const previewImport = async () => {
    const payload = parseImportPayload();
    if (!payload) return;

    setPreviewing(true);
    try {
      const resp = await Api.ai.previewAdminModelImport(payload);
      setPreviewRows(resp.data?.providers || []);
      message.success('导入预览已生成');
    } catch (err) {
      message.error(err instanceof Error ? err.message : '生成导入预览失败');
    } finally {
      setPreviewing(false);
    }
  };

  const submitImport = async () => {
    const payload = parseImportPayload();
    if (!payload) return;

    setImporting(true);
    try {
      const resp = await Api.ai.importAdminModels(payload);
      const created = resp.data?.created ?? 0;
      const updated = resp.data?.updated ?? 0;
      message.success(`导入完成：新增 ${created}，更新 ${updated}`);
      setImportOpen(false);
      setImportJson('');
      setPreviewRows([]);
      await load();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '导入失败');
    } finally {
      setImporting(false);
    }
  };

  if (accessDenied) {
    return <AccessDeniedPage />;
  }

  return (
    <div className="space-y-6">
      <Card
        className="border-0 shadow-sm"
        bodyStyle={{
          background: 'linear-gradient(135deg, rgba(37,99,235,0.08) 0%, rgba(15,23,42,0.04) 100%)',
          borderRadius: 12,
        }}
      >
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <Space align="center" size={10}>
              <div className="h-9 w-9 rounded-lg bg-blue-600 text-white flex items-center justify-center">
                <ThunderboltOutlined />
              </div>
              <div>
                <Text strong style={{ fontSize: 18 }}>AI 模型配置中心</Text>
                <Paragraph type="secondary" className="!mb-0">
                  管理全局默认模型、供应商路由与 OpenClaw JSON 批量导入。
                </Paragraph>
              </div>
            </Space>
          </div>
          <Space wrap>
            <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>刷新</Button>
            <Button icon={<UploadOutlined />} onClick={() => setImportOpen(true)}>JSON 导入</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新增模型</Button>
          </Space>
        </div>
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} md={6}>
          <Card size="small"><Text type="secondary">模型总数</Text><div className="text-2xl font-semibold">{summary.total}</div></Card>
        </Col>
        <Col xs={24} md={6}>
          <Card size="small"><Text type="secondary">已启用</Text><div className="text-2xl font-semibold text-green-600">{summary.enabled}</div></Card>
        </Col>
        <Col xs={24} md={6}>
          <Card size="small"><Text type="secondary">默认模型</Text><div className="text-2xl font-semibold text-blue-600">{summary.defaults}</div></Card>
        </Col>
        <Col xs={24} md={6}>
          <Card size="small"><Text type="secondary">思考模式启用</Text><div className="text-2xl font-semibold text-orange-600">{summary.thinking}</div></Card>
        </Col>
      </Row>

      <Card title="模型清单" extra={<Text type="secondary">后端接口：/api/v1/admin/ai/models</Text>}>
        <Table<AILLMProvider>
          rowKey="id"
          loading={loading}
          dataSource={rows}
          locale={{ emptyText: <Empty description="暂无模型配置" /> }}
          pagination={{ pageSize: 8 }}
          columns={[
            {
              title: '模型名称',
              dataIndex: 'name',
              render: (_: unknown, row) => (
                <Space>
                  <Text strong>{row.name}</Text>
                  {row.is_default ? <Tag color="gold">默认</Tag> : null}
                </Space>
              ),
            },
            {
              title: '供应商 / Model',
              key: 'provider_model',
              render: (_: unknown, row) => (
                <Space direction="vertical" size={0}>
                  <Tag color="blue">{row.provider}</Tag>
                  <Text type="secondary" className="text-xs">{row.model}</Text>
                </Space>
              ),
            },
            {
              title: '连接配置',
              key: 'connection',
              render: (_: unknown, row) => (
                <Space direction="vertical" size={0}>
                  <Text>{row.base_url}</Text>
                  <Text type="secondary" className="text-xs">{row.api_key_masked || '***'}</Text>
                </Space>
              ),
            },
            {
              title: '参数',
              key: 'params',
              render: (_: unknown, row) => (
                <Space size={6} wrap>
                  <Tag>Temp {row.temperature.toFixed(2)}</Tag>
                  {row.thinking ? <Tag color="orange">Thinking</Tag> : <Tag>Fast</Tag>}
                  {row.is_enabled ? <Tag color="success">Enabled</Tag> : <Tag color="default">Disabled</Tag>}
                </Space>
              ),
            },
            {
              title: '操作',
              key: 'actions',
              width: 240,
              render: (_: unknown, row) => (
                <Space size={4}>
                  <Tooltip title="编辑模型">
                    <Button icon={<EditOutlined />} onClick={() => openEdit(row)} />
                  </Tooltip>
                  <Tooltip title={row.is_default ? '当前已是默认模型' : '设为默认'}>
                    <Button
                      type={row.is_default ? 'primary' : 'default'}
                      icon={row.is_default ? <StarFilled /> : <StarOutlined />}
                      onClick={() => void setDefault(row)}
                      disabled={row.is_default}
                    />
                  </Tooltip>
                  <Popconfirm
                    title="确认删除该模型配置？"
                    description={row.is_default ? '默认模型不能删除，请先切换默认模型。' : '删除后不可恢复。'}
                    okText="删除"
                    okButtonProps={{ danger: true }}
                    onConfirm={() => void remove(row)}
                    disabled={row.is_default}
                  >
                    <Button danger icon={<DeleteOutlined />} disabled={row.is_default} />
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>

      <Drawer
        title={editing ? `编辑模型：${editing.name}` : '新增模型配置'}
        width={560}
        open={drawerOpen}
        onClose={closeDrawer}
        destroyOnClose
        extra={<Button type="primary" loading={saving} onClick={() => void saveModel()}>保存</Button>}
      >
        <Form<ModelFormValues> form={form} layout="vertical" initialValues={initialFormValues}>
          <Form.Item name="name" label="显示名称" rules={[{ required: true, message: '请输入模型显示名称' }]}>
            <Input placeholder="例如：Qwen3.5-Plus（生产）" />
          </Form.Item>

          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name="provider" label="供应商" rules={[{ required: true }]}>
                <Select options={providerChoices} placeholder="请选择供应商" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="model" label="模型标识" rules={[{ required: true, message: '请输入模型标识' }]}>
                <Input placeholder="qwen-max / doubao-pro / llama3" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="base_url" label="Base URL" rules={[{ required: true, message: '请输入 Base URL' }]}>
            <Input placeholder="https://dashscope.aliyuncs.com/compatible-mode/v1" />
          </Form.Item>

          <Form.Item
            name="api_key"
            label={editing ? 'API Key（留空则不修改）' : 'API Key'}
            rules={editing ? [] : [{ required: true, message: '请输入 API Key' }]}
          >
            <Input.Password placeholder={editing ? '留空表示保持原值' : '输入 API Key'} />
          </Form.Item>

          <Row gutter={12}>
            <Col span={8}>
              <Form.Item name="temperature" label="Temperature">
                <InputNumber min={0} max={2} step={0.1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="sort_order" label="排序权重">
                <InputNumber style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="thinking" label="思考模式" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>

          <Divider />

          <Space size={24}>
            <Form.Item name="is_enabled" label="启用状态" valuePropName="checked" className="!mb-0">
              <Switch checkedChildren="启用" unCheckedChildren="禁用" />
            </Form.Item>
            <Form.Item name="is_default" label="默认模型" valuePropName="checked" className="!mb-0">
              <Switch checkedChildren="默认" unCheckedChildren="非默认" />
            </Form.Item>
          </Space>
        </Form>
      </Drawer>

      <Modal
        title="OpenClaw JSON 导入"
        open={importOpen}
        width={860}
        onCancel={() => {
          setImportOpen(false);
          setPreviewRows([]);
        }}
        okText="执行导入"
        onOk={() => void submitImport()}
        okButtonProps={{ loading: importing }}
      >
        <Alert
          type="info"
          showIcon
          className="mb-4"
          message="导入说明"
          description="JSON 顶层必须包含 providers 数组。可先点击“生成预览”，确认后再执行导入。"
        />
        <Space className="mb-3">
          <Text>导入模式</Text>
          <Segmented
            value={importMode}
            onChange={(val) => setImportMode(val as 'merge' | 'replace')}
            options={[
              { label: 'Merge（增量）', value: 'merge' },
              { label: 'Replace（覆盖）', value: 'replace' },
            ]}
          />
          <Button icon={<EyeOutlined />} loading={previewing} onClick={() => void previewImport()}>
            生成预览
          </Button>
        </Space>

        <Input.TextArea
          value={importJson}
          onChange={(e) => setImportJson(e.target.value)}
          placeholder='{"providers":[{"name":"Qwen Prod","provider":"qwen","model":"qwen-max","base_url":"https://...","api_key":"sk-***"}]}'
          autoSize={{ minRows: 8, maxRows: 16 }}
        />

        <Divider>预览结果</Divider>

        <Table<AILLMProvider>
          rowKey={(row, idx) => `${row.provider}-${row.model}-${idx}`}
          size="small"
          dataSource={previewRows}
          locale={{ emptyText: '暂无预览数据' }}
          pagination={false}
          columns={[
            { title: '名称', dataIndex: 'name' },
            { title: '供应商', dataIndex: 'provider', render: (v: string) => <Tag color="blue">{v}</Tag> },
            { title: '模型', dataIndex: 'model' },
            { title: 'URL', dataIndex: 'base_url', ellipsis: true },
            { title: '默认', dataIndex: 'is_default', render: (v: boolean) => (v ? <Tag color="gold">是</Tag> : '否') },
          ]}
        />
      </Modal>
    </div>
  );
};

export default AIModelSettingsPage;
