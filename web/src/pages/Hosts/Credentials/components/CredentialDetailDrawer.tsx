import React, { useEffect, useState } from 'react';
import { Drawer, Spin, Divider } from 'antd';
import { hostApi } from '../../../../api/modules/hosts';
import type { CredentialDetail } from '../../../../api/modules/hosts';
import { CredentialBasicInfo } from './CredentialBasicInfo';
import { CredentialSecretPanel } from './CredentialSecretPanel';
import { CredentialRelationsPanel } from './CredentialRelationsPanel';
import { CredentialUsageStats } from './CredentialUsageStats';
import { CredentialQuickActions } from './CredentialQuickActions';

interface Props {
  credentialId?: string;
  onClose: () => void;
  onRefresh: () => void;
}

export const CredentialDetailDrawer: React.FC<Props> = ({ credentialId, onClose, onRefresh }) => {
  const [detail, setDetail] = useState<CredentialDetail | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (credentialId) {
      setLoading(true);
      hostApi.getCredentialDetail(credentialId).then(res => {
        if (res && res.success) setDetail(res.data);
      }).catch(err => {
        console.error('Failed to fetch detail:', err);
      }).finally(() => setLoading(false));
    } else {
      setDetail(null);
    }
  }, [credentialId]);

  return (
    <Drawer
      title="凭证详情"
      placement="right"
      size="large"
      onClose={onClose}
      open={!!credentialId}
      destroyOnClose
    >
      <Spin spinning={loading}>
        {detail && (
          <div className="flex flex-col gap-4">
            <CredentialBasicInfo detail={detail} />
            <Divider className="my-0" />
            <CredentialSecretPanel detail={detail} />
            <Divider className="my-0" />
            <CredentialRelationsPanel detail={detail} />
            <Divider className="my-0" />
            <CredentialUsageStats detail={detail} />
            <Divider className="my-0" />
            <CredentialQuickActions detail={detail} onRefresh={onRefresh} onClose={onClose} />
          </div>
        )}
      </Spin>
    </Drawer>
  );
};
