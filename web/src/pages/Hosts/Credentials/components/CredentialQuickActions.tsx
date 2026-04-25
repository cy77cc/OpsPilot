import React from 'react';
import { Button, Space, Modal, message } from 'antd';
import type { CredentialDetail } from '../../../../api/modules/hosts';
import { EditOutlined, CopyOutlined, SyncOutlined, DeleteOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { hostApi } from '../../../../api/modules/hosts';

interface Props {
  detail: CredentialDetail;
  onRefresh: () => void;
  onClose: () => void;
}

export const CredentialQuickActions: React.FC<Props> = ({ detail, onRefresh, onClose }) => {
  const handleDelete = () => {
    const realId = detail.id.replace(/^(key|tpl)-/, '');
    Modal.confirm({
      title: '确认删除凭证',
      icon: <ExclamationCircleOutlined />,
      content: `确定要删除凭证 "${detail.name}" 吗？此操作不可撤销。`,
      okText: '确认删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          let res;
          if (detail.id.startsWith('key-')) {
            res = await hostApi.deleteSSHKey(realId);
          } else {
            res = await hostApi.deleteCredentialTemplate(realId);
          }
          if (res.success) {
            message.success('凭证已删除');
            onClose();
            onRefresh();
          }
        } catch (err: any) {
          message.error(err.message || '删除失败');
        }
      },
    });
  };

  return (
    <div>
      <h3 className="font-semibold mb-2 text-sm text-gray-700">快捷操作</h3>
      <Space wrap>
        <Button icon={<EditOutlined />} size="small">编辑</Button>
        <Button icon={<CopyOutlined />} size="small">复制配置</Button>
        <Button icon={<SyncOutlined />} size="small" danger>轮换密钥</Button>
        <Button icon={<DeleteOutlined />} size="small" type="primary" danger onClick={handleDelete}>删除</Button>
      </Space>
    </div>
  );
};
