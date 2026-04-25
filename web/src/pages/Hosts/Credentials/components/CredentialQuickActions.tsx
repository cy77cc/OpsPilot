import React from 'react';
import { Button, Space } from 'antd';
import type { CredentialDetail } from '../../../../api/modules/hosts';
import { EditOutlined, CopyOutlined, SyncOutlined, DeleteOutlined } from '@ant-design/icons';

export const CredentialQuickActions: React.FC<{ detail: CredentialDetail }> = ({ detail }) => {
  return (
    <div>
      <h3 className="font-semibold mb-2">快捷操作</h3>
      <Space wrap>
        <Button icon={<EditOutlined />}>编辑</Button>
        <Button icon={<CopyOutlined />}>复制配置</Button>
        <Button icon={<SyncOutlined />} danger>轮换密钥</Button>
        <Button icon={<DeleteOutlined />} type="primary" danger>删除</Button>
      </Space>
    </div>
  );
};