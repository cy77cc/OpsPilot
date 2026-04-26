import React, { useState, useEffect } from 'react';
import { Card, List, Tag, Button, Typography, Modal, message, Spin } from 'antd';
import { FileTextOutlined, EditOutlined, HistoryOutlined } from '@ant-design/icons';
import { hostApi } from '../../../../api/modules/hosts';

const { Text } = Typography;

interface ConfigItem {
  id: string;
  name: string;
  path: string;
  lastModified: string;
  status: 'synced' | 'modified';
}

const ConfigTab: React.FC<{ hostId: string }> = ({ hostId }) => {
  const [loading, setLoading] = useState(false);
  const [configs, setConfigs] = useState<ConfigItem[]>([]);
  const [editingConfig, setEditingConfig] = useState<ConfigItem | null>(null);
  const [fileContent, setFileContent] = useState('');
  const [contentLoading, setContentLoading] = useState(false);

  useEffect(() => {
    // Initial mock configs, can be fetched from a managed-configs API later
    setConfigs([
      { id: '1', name: 'nginx.conf', path: '/etc/nginx/nginx.conf', lastModified: '2024-05-10 10:00:00', status: 'synced' },
      { id: '2', name: 'hosts', path: '/etc/hosts', lastModified: '2024-05-12 14:20:00', status: 'modified' },
      { id: '3', name: 'sysctl.conf', path: '/etc/sysctl.conf', lastModified: '2024-04-20 09:15:00', status: 'synced' },
    ]);
  }, [hostId]);

  const handleEdit = async (item: ConfigItem) => {
    setEditingConfig(item);
    setContentLoading(true);
    try {
      const res = await hostApi.readFile(hostId, item.path);
      setFileContent(res.data.content || '');
    } catch (err) {
      message.error(`读取文件失败: ${item.path}`);
      setFileContent('');
    } finally {
      setContentLoading(false);
    }
  };

  const handleSave = async () => {
    if (!editingConfig) return;
    try {
      await hostApi.writeFile(hostId, editingConfig.path, fileContent);
      message.success('配置已保存并开始分发');
      setEditingConfig(null);
    } catch (err) {
      message.error('保存失败');
    }
  };

  return (
    <Card className="h-full border-none shadow-sm mt-4">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h3 className="text-base font-medium m-0">配置管理</h3>
          <p className="text-gray-400 text-xs mt-1">管理主机上的核心配置文件</p>
        </div>
        <Button type="primary">添加配置托管</Button>
      </div>

      <Spin spinning={loading}>
        <List
          itemLayout="horizontal"
          dataSource={configs}
          renderItem={(item) => (
            <List.Item
              actions={[
                <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(item)}>编辑</Button>,
                <Button type="link" size="small" icon={<HistoryOutlined />}>历史</Button>,
              ]}
              className="hover:bg-gray-50/50 transition-colors px-4 rounded-lg border-b-0 mb-2 bg-gray-50/20"
            >
              <List.Item.Meta
                avatar={<div className="w-10 h-10 rounded-lg bg-blue-50 flex items-center justify-center text-blue-500"><FileTextOutlined style={{ fontSize: 20 }} /></div>}
                title={
                  <div className="flex items-center gap-3">
                    <Text strong>{item.name}</Text>
                    <Tag color={item.status === 'synced' ? 'success' : 'warning'}>
                      {item.status === 'synced' ? '已同步' : '未同步'}
                    </Tag>
                  </div>
                }
                description={
                  <div className="flex flex-col gap-1 mt-1">
                    <Text type="secondary" className="text-xs">{item.path}</Text>
                    <Text type="secondary" style={{ fontSize: 10 }}>最后修改：{item.lastModified}</Text>
                  </div>
                }
              />
            </List.Item>
          )}
        />
      </Spin>

      <Modal
        title={`编辑配置文件: ${editingConfig?.name}`}
        open={!!editingConfig}
        onCancel={() => setEditingConfig(null)}
        onOk={handleSave}
        width={800}
        okText="保存并应用"
        confirmLoading={contentLoading}
      >
        <Spin spinning={contentLoading}>
          <div className="bg-slate-900 text-slate-200 p-0 rounded-lg overflow-hidden">
            <textarea 
              className="w-full bg-transparent p-4 font-mono text-sm min-h-[300px] border-none outline-none resize-none"
              value={fileContent}
              onChange={(e) => setFileContent(e.target.value)}
            />
          </div>
        </Spin>
        <p className="text-xs text-gray-400 mt-2">提示：保存后将自动备份旧配置并尝试重新加载相关服务。</p>
      </Modal>
    </Card>
  );
};

export default ConfigTab;
