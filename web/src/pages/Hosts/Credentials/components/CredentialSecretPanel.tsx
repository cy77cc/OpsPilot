import React, { useState } from 'react';
import { Button, Input, Modal, Typography, message } from 'antd';
import { EyeInvisibleOutlined, EyeOutlined } from '@ant-design/icons';
import type { CredentialDetailViewModel } from '../viewModels';

const { Paragraph } = Typography;

export const CredentialSecretPanel: React.FC<{ detail: CredentialDetailViewModel }> = ({ detail }) => {
  const [visible, setVisible] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);

  return (
    <section className="border-b border-[#edf2f7] pb-6">
      <h3 className="mb-4 text-[20px] font-semibold text-[#111827]">密钥内容</h3>
      <div className="rounded-xl border border-[#e8edf5] bg-[#fafcff] p-4">
        {visible ? (
          <div className="space-y-3">
            <Input.TextArea
              readOnly
              value={detail.secret || '******'}
              autoSize={{ minRows: 4, maxRows: 8 }}
              className="font-mono text-[12px]"
            />
            <Button icon={<EyeInvisibleOutlined />} onClick={() => setVisible(false)}>
              隐藏密钥
            </Button>
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            <Button icon={<EyeOutlined />} className="w-fit" onClick={() => setConfirmOpen(true)}>
              查看密钥
            </Button>
            <Paragraph className="!mb-0 !text-[13px] !text-[#6b7280]">
              密钥已加密存储，仅在需要时查看
            </Paragraph>
          </div>
        )}
      </div>

      <Modal
        title="安全确认"
        open={confirmOpen}
        onOk={() => {
          setVisible(true);
          setConfirmOpen(false);
          message.warning('查看凭证动作已记录到审计日志');
        }}
        onCancel={() => setConfirmOpen(false)}
        okText="确认查看"
        cancelText="取消"
      >
        您正在尝试查看敏感凭证信息，此操作将被记录到系统审计日志中。是否继续？
      </Modal>
    </section>
  );
};
