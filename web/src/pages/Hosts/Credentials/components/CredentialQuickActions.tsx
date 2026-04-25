import React from 'react';
import { Button, Modal, Space, message } from 'antd';
import {
  CopyOutlined,
  DeleteOutlined,
  EditOutlined,
  ExclamationCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { hostApi } from '../../../../api/modules/hosts';
import type { CredentialDetailViewModel } from '../viewModels';

interface Props {
  detail: CredentialDetailViewModel;
  onRefresh: () => void;
  onClose: () => void;
}

export const CredentialQuickActions: React.FC<Props> = ({ detail, onRefresh, onClose }) => {
  const handleComingSoon = () => message.info('该操作将在后续联调中接入');

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
          const res = detail.id.startsWith('key-')
            ? await hostApi.deleteSSHKey(realId)
            : await hostApi.deleteCredentialTemplate(realId);
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
    <section>
      <h3 className="mb-4 text-[16px] font-semibold text-[#111827]">快捷操作</h3>
      <Space wrap size={12}>
        <Button icon={<EditOutlined />} onClick={handleComingSoon} className="!rounded-lg !border-[#d8e1ee]">
          编辑
        </Button>
        <Button icon={<CopyOutlined />} onClick={handleComingSoon} className="!rounded-lg !border-[#d8e1ee]">
          复制
        </Button>
        <Button icon={<SyncOutlined />} onClick={handleComingSoon} className="!rounded-lg !border-[#d8e1ee]">
          轮换密钥
        </Button>
        <Button icon={<DeleteOutlined />} danger onClick={handleDelete} className="!rounded-lg">
          删除
        </Button>
      </Space>
    </section>
  );
};
