import React, { useCallback, useEffect, useState } from 'react';
import { Button, Card, Modal, Select, Space, Table, Tag, message, Popconfirm, Typography } from 'antd';
import { Api } from '../../../../api';
import type { Host } from '../../../../api/modules/hosts';
import type { HostPluginCatalogItem, HostPluginInstance, HostPluginTask } from '../../../../types/host';

interface PluginTabProps {
  host: Host | null;
}

const PluginTab: React.FC<PluginTabProps> = ({ host }) => {
  const rows = host?.pluginInstances || [];
  const hasOpsAgent = rows.some((r) => r.pluginKey === 'opsagent' && r.installStatus !== 'uninstalled');

  const [installModalOpen, setInstallModalOpen] = useState(false);
  const [catalog, setCatalog] = useState<HostPluginCatalogItem[]>([]);
  const [selectedVersion, setSelectedVersion] = useState<string>('');
  const [installing, setInstalling] = useState(false);
  const [taskPolling, setTaskPolling] = useState<HostPluginTask | null>(null);

  const loadCatalog = useCallback(async () => {
    try {
      const res = await Api.hosts.listHostPluginCatalog();
      setCatalog(res.data || []);
    } catch {
      // ignore
    }
  }, []);

  useEffect(() => {
    if (installModalOpen) {
      void loadCatalog();
    }
  }, [installModalOpen, loadCatalog]);

  const handleInstall = async () => {
    if (!host?.id || !selectedVersion) return;
    setInstalling(true);
    try {
      const res = await Api.hosts.installPlugin(host.id, 'opsagent', selectedVersion);
      message.success('安装任务已创建');
      setInstallModalOpen(false);
      setSelectedVersion('');
      // Start polling task status
      pollTaskStatus(String(res.data.task_id));
    } catch (err) {
      message.error(err instanceof Error ? err.message : '安装失败');
    } finally {
      setInstalling(false);
    }
  };

  const handleUninstall = async (instanceId: string) => {
    if (!host?.id) return;
    try {
      const res = await Api.hosts.uninstallPlugin(host.id, instanceId);
      message.success('卸载任务已创建');
      pollTaskStatus(String(res.data.task_id));
    } catch (err) {
      message.error(err instanceof Error ? err.message : '卸载失败');
    }
  };

  const pollTaskStatus = async (taskId: string) => {
    const maxAttempts = 60;
    for (let i = 0; i < maxAttempts; i++) {
      await new Promise((r) => setTimeout(r, 2000));
      try {
        const res = await Api.hosts.getPluginTask(taskId);
        setTaskPolling(res.data);
        if (res.data.status === 'succeeded' || res.data.status === 'failed') {
          if (res.data.status === 'succeeded') {
            message.success('操作完成');
          } else {
            message.error(`操作失败: ${res.data.errorMessage}`);
          }
          // Refresh host detail
          window.location.reload();
          return;
        }
      } catch {
        // ignore polling errors
      }
    }
  };

  const opsVersions = catalog
    .filter((item) => item.pluginKey === 'opsagent' && item.defaultVersion)
    .map((item) => ({ label: item.defaultVersion, value: item.defaultVersion }));

  return (
    <Card
      title="插件实例"
      extra={
        !hasOpsAgent ? (
          <Button type="primary" size="small" onClick={() => setInstallModalOpen(true)}>
            安装 Agent
          </Button>
        ) : undefined
      }
    >
      <Table
        rowKey={(row) => `${row.pluginKey}-${row.installedVersion}-${row.lastSeenAt || ''}`}
        dataSource={rows}
        pagination={false}
        columns={[
          { title: '插件', dataIndex: 'pluginKey' },
          { title: '版本', dataIndex: 'installedVersion', render: (value: string) => value || '-' },
          { title: '安装状态', dataIndex: 'installStatus', render: (value: string) => <Tag>{value || '-'}</Tag> },
          { title: '运行状态', dataIndex: 'runtimeStatus', render: (value: string) => <Tag>{value || '-'}</Tag> },
          { title: '健康状态', dataIndex: 'healthStatus', render: (value: string) => <Tag>{value || '-'}</Tag> },
          { title: '最近心跳', dataIndex: 'lastSeenAt', render: (value: string) => value ? new Date(value).toLocaleString() : '-' },
          {
            title: '操作',
            key: 'actions',
            render: (_: unknown, record: HostPluginInstance) => {
              if (record.installStatus === 'uninstalled') return null;
              return (
                <Popconfirm
                  title="确定卸载此 Agent？"
                  description="将停止服务并删除所有相关文件"
                  onConfirm={() => handleUninstall(record.pluginKey)}
                  okText="卸载"
                  cancelText="取消"
                  okButtonProps={{ danger: true }}
                >
                  <Button type="link" danger size="small">
                    卸载
                  </Button>
                </Popconfirm>
              );
            },
          },
        ]}
        locale={{ emptyText: '暂无插件实例' }}
      />

      {taskPolling && (
        <div style={{ marginTop: 16 }}>
          <Typography.Text type="secondary">
            任务 #{taskPolling.id}: {taskPolling.status}
            {taskPolling.errorMessage && ` - ${taskPolling.errorMessage}`}
          </Typography.Text>
        </div>
      )}

      <Modal
        title="安装 OpsAgent"
        open={installModalOpen}
        onCancel={() => setInstallModalOpen(false)}
        onOk={handleInstall}
        confirmLoading={installing}
        okText="安装"
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Typography.Text>选择要安装的 OpsAgent 版本：</Typography.Text>
          <Select
            style={{ width: '100%' }}
            placeholder="选择版本"
            options={opsVersions}
            value={selectedVersion || undefined}
            onChange={setSelectedVersion}
          />
        </Space>
      </Modal>
    </Card>
  );
};

export default PluginTab;
