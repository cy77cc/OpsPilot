import React from 'react';
import { Alert, Descriptions, Modal, Typography } from 'antd';
import type { HostKeyTrustPayload } from '../../api/modules/hosts';

type HostKeyTrustModalProps = {
  open: boolean;
  loading?: boolean;
  mode: 'create' | 'rotate';
  hostKey: HostKeyTrustPayload | null;
  onCancel: () => void;
  onConfirm: () => void;
};

const HostKeyTrustModal: React.FC<HostKeyTrustModalProps> = ({
  open,
  loading,
  mode,
  hostKey,
  onCancel,
  onConfirm,
}) => (
  <Modal
    open={open}
    onCancel={onCancel}
    onOk={onConfirm}
    confirmLoading={loading}
    okText={mode === 'rotate' ? '确认替换' : '确认信任'}
    cancelText="取消"
    title="信任此主机指纹？"
    width={680}
  >
    <Alert
      type={mode === 'rotate' ? 'warning' : 'info'}
      showIcon
      style={{ marginBottom: 12 }}
      message={mode === 'rotate'
        ? '检测到主机指纹变化。确认后将替换已信任指纹并重试操作。'
        : '首次连接主机需要显式信任指纹。确认后将写入信任记录并重试操作。'}
    />
    <Descriptions bordered column={1} size="small">
      <Descriptions.Item label="主机">{hostKey?.host || '-'}:{hostKey?.port || '-'}</Descriptions.Item>
      <Descriptions.Item label="算法">{hostKey?.algorithm || '-'}</Descriptions.Item>
      <Descriptions.Item label="SHA256 指纹">{hostKey?.fingerprintSha256 || '-'}</Descriptions.Item>
      <Descriptions.Item label="公钥">
        <Typography.Text copyable={{ text: hostKey?.publicKey || '' }} style={{ wordBreak: 'break-all' }}>
          {hostKey?.publicKey || '-'}
        </Typography.Text>
      </Descriptions.Item>
      <Descriptions.Item label="known_hosts 路径">{hostKey?.knownHostsPath || '-'}</Descriptions.Item>
    </Descriptions>
  </Modal>
);

export default HostKeyTrustModal;
