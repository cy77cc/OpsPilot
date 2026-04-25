import React from 'react';
import { Button, Spin } from 'antd';
import { CloseOutlined } from '@ant-design/icons';
import type { CredentialDetailViewModel } from '../viewModels';
import { CredentialBasicInfo } from './CredentialBasicInfo';
import { CredentialQuickActions } from './CredentialQuickActions';
import { CredentialRelationsPanel } from './CredentialRelationsPanel';
import { CredentialSecretPanel } from './CredentialSecretPanel';
import { CredentialUsageStats } from './CredentialUsageStats';

interface Props {
  detail: CredentialDetailViewModel | null;
  loading: boolean;
  open: boolean;
  onClose: () => void;
  onRefresh: () => void;
}

export const CredentialDetailDrawer: React.FC<Props> = ({
  detail,
  loading,
  open,
  onClose,
  onRefresh,
}) => {
  if (!open) {
    return null;
  }

  return (
    <aside className="min-h-[720px] bg-white">
      <div className="flex items-center justify-between border-b border-[#edf2f7] px-5 py-4">
        <h2 className="text-[18px] font-semibold text-[#111827]">凭证详情</h2>
        <Button
          type="text"
          icon={<CloseOutlined />}
          onClick={onClose}
          className="!text-[#94a3b8]"
        />
      </div>
      <Spin spinning={loading}>
        {detail ? (
          <div className="space-y-6 px-5 py-4">
            <CredentialBasicInfo detail={detail} />
            <CredentialSecretPanel detail={detail} />
            <CredentialRelationsPanel detail={detail} />
            <CredentialUsageStats detail={detail} />
            <CredentialQuickActions detail={detail} onRefresh={onRefresh} onClose={onClose} />
          </div>
        ) : (
          <div className="px-5 py-10 text-center text-[14px] text-[#6b7280]">请选择左侧凭证查看详情</div>
        )}
      </Spin>
    </aside>
  );
};
