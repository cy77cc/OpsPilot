import React, { useState } from 'react';
import { Button, Input, Modal, Typography, message } from 'antd';
import { EyeOutlined, EyeInvisibleOutlined } from '@ant-design/icons';
import type { CredentialDetail } from '../../../../api/modules/hosts';

const { Text } = Typography;

export const CredentialSecretPanel: React.FC<{ detail: CredentialDetail }> = ({ detail }) => {
  const [visible, setVisible] = useState(false);
  const [confirmModal, setConfirmModal] = useState(false);

  const handleReveal = () => {
    if (!visible) {
      setConfirmModal(true);
    } else {
      setVisible(false);
    }
  };

  const confirmReveal = () => {
    setVisible(true);
    setConfirmModal(false);
    message.warning('查看凭证动作已被审计记录');
  };

  return (
    <div>
      <h3 className="font-semibold mb-2">密钥内容</h3>
      <div className="bg-gray-50 p-4 rounded border border-gray-200">
        {!visible ? (
          <div className="flex flex-col items-center justify-center py-4">
            <Text type="secondary" className="mb-2">密钥已加密存储，仅在需要时查看</Text>
            <Button icon={<EyeOutlined />} onClick={handleReveal}>查看密钥</Button>
          </div>
        ) : (
          <div>
            <Input.TextArea value={detail.secret || '******'} readOnly autoSize={{ minRows: 3, maxRows: 10 }} className="font-mono text-xs mb-2" />
            <Button icon={<EyeInvisibleOutlined />} onClick={handleReveal}>隐藏密钥</Button>
          </div>
        )}
      </div>

      <Modal
        title="安全确认"
        open={confirmModal}
        onOk={confirmReveal}
        onCancel={() => setConfirmModal(false)}
        okText="确认查看"
      >
        <p>您正在尝试查看敏感凭证信息，此操作将被记录到系统审计日志中。是否继续？</p>
      </Modal>
    </div>
  );
};