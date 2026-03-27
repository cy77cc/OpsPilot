import React, { useState, useCallback, useEffect } from 'react';
import { Button, Table, Tag, Space, message, Modal, Select } from 'antd';
import { PlusOutlined, ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined, SyncOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../../api';
import type { ClusterCredential } from '../../../api/modules/deployment';
import RegisterPlatformCredentialModal from './RegisterPlatformCredentialModal';
import ImportExternalCredentialModal from './ImportExternalCredentialModal';

const CredentialListPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [credentials, setCredentials] = useState<ClusterCredential[]>([]);
  const [runtimeFilter, setRuntimeFilter] = useState<'all' | 'k8s' | 'compose'>('all');
  const [registerModalVisible, setRegisterModalVisible] = useState(false);
  const [importModalVisible, setImportModalVisible] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = runtimeFilter === 'all' ? {} : { runtime_type: runtimeFilter };
      const res = await Api.deployment.listCredentials(params);
      setCredentials(res.data.list || []);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载凭证列表失败');
    } finally {
      setLoading(false);
    }
  }, [runtimeFilter]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleTest = async (id: number) => {
    try {
      const res = await Api.deployment.testCredential(id);
      if (res.data.connected) {
        message.success(`连接成功 (延迟: ${res.data.latency_ms}ms)`);
      } else {
        message.error(`连接失败: ${res.data.message}`);
      }
      await load();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '测试连接失败');
    }
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '运行时',
      dataIndex: 'runtime_type',
      key: 'runtime_type',
      render: (type: string) => (
        <Tag color={type === 'k8s' ? 'blue' : 'green'}>{type.toUpperCase()}</Tag>
      ),
    },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      render: (source: string) => (
        <Tag color={source === 'platform_managed' ? 'purple' : 'orange'}>
          {source === 'platform_managed' ? '平台托管' : '外部导入'}
        </Tag>
      ),
    },
    {
      title: 'Endpoint',
      dataIndex: 'endpoint',
      key: 'endpoint',
      ellipsis: true,
    },
    {
      title: '认证方式',
      dataIndex: 'auth_method',
      key: 'auth_method',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'active' ? 'success' : 'default'}>{status}</Tag>
      ),
    },
    {
      title: '最近测试',
      key: 'last_test',
      render: (_: any, record: ClusterCredential) => {
        if (!record.last_test_at) return '-';
        return (
          <Space direction="vertical" size="small">
            <Space>
              {record.last_test_status === 'ok' ? (
                <CheckCircleOutlined style={{ color: '#52c41a' }} />
              ) : (
                <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
              )}
              <span>{new Date(record.last_test_at).toLocaleString()}</span>
            </Space>
            {record.last_test_message && (
              <span className="text-xs text-gray-500">{record.last_test_message}</span>
            )}
          </Space>
        );
      },
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: ClusterCredential) => (
        <Space>
          <Button
            size="small"
            icon={<SyncOutlined />}
            onClick={() => handleTest(record.id)}
          >
            测试连接
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">集群凭证管理</h1>
          <p className="text-sm text-gray-500 mt-1">管理 Kubernetes 集群访问凭证</p>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setRegisterModalVisible(true)}
          >
            注册平台凭证
          </Button>
          <Button
            icon={<PlusOutlined />}
            onClick={() => setImportModalVisible(true)}
          >
            导入外部凭证
          </Button>
        </Space>
      </div>

      <div className="flex items-center gap-4">
        <span className="text-sm text-gray-600">运行时筛选:</span>
        <Select
          value={runtimeFilter}
          onChange={setRuntimeFilter}
          style={{ width: 120 }}
          options={[
            { label: '全部', value: 'all' },
            { label: 'K8s', value: 'k8s' },
            { label: 'Compose', value: 'compose' },
          ]}
        />
      </div>

      <Table
        columns={columns}
        dataSource={credentials}
        rowKey="id"
        loading={loading}
        pagination={{
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条`,
        }}
      />

      <RegisterPlatformCredentialModal
        visible={registerModalVisible}
        onCancel={() => setRegisterModalVisible(false)}
        onSuccess={() => {
          setRegisterModalVisible(false);
          void load();
        }}
      />

      <ImportExternalCredentialModal
        visible={importModalVisible}
        onCancel={() => setImportModalVisible(false)}
        onSuccess={() => {
          setImportModalVisible(false);
          void load();
        }}
      />
    </div>
  );
};

export default CredentialListPage;
