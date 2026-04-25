import React from 'react';
import { Card } from 'antd';
import type { CredentialStats } from '../../../../api/modules/hosts';
import { KeyOutlined, CheckCircleOutlined, WarningOutlined, ClockCircleOutlined, SyncOutlined } from '@ant-design/icons';

interface Props {
  stats?: CredentialStats;
  loading?: boolean;
}

export const CredentialStatsCards: React.FC<Props> = ({ stats, loading }) => {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4">
      <Card size="small" loading={loading} className="border border-[#e8edf3] rounded-[10px] shadow-none">
        <div className="text-sm text-gray-500 mb-2">总凭证数</div>
        <div className="flex items-center mb-2">
          <KeyOutlined className="text-2xl mr-2 text-gray-800" />
          <span className="text-3xl font-semibold leading-none text-gray-800">{stats?.total || 0}</span>
        </div>
        <div className="text-xs text-gray-400">较上周 +3</div>
      </Card>
      
      <Card size="small" loading={loading} className="border border-[#e8edf3] rounded-[10px] shadow-none">
        <div className="text-sm text-gray-500 mb-2">可用凭证</div>
        <div className="flex items-center mb-2">
          <CheckCircleOutlined className="text-2xl mr-2 text-gray-800" />
          <span className="text-3xl font-semibold leading-none text-gray-800">{stats?.available || 0}</span>
        </div>
        <div className="text-xs text-gray-400">可用率 {stats?.total ? ((stats.available / stats.total) * 100).toFixed(1) : 0}%</div>
      </Card>

      <Card size="small" loading={loading} className="border border-[#e8edf3] rounded-[10px] shadow-none">
        <div className="text-sm text-gray-500 mb-2">即将过期</div>
        <div className="flex items-center mb-2">
          <ClockCircleOutlined className="text-2xl mr-2 text-gray-800" />
          <span className="text-3xl font-semibold leading-none text-gray-800">{stats?.expiringSoon || 0}</span>
        </div>
        <div className="text-xs text-gray-400">7 天内过期</div>
      </Card>

      <Card size="small" loading={loading} className="border border-[#e8edf3] rounded-[10px] shadow-none">
        <div className="text-sm text-gray-500 mb-2">已过期</div>
        <div className="flex items-center mb-2">
          <WarningOutlined className="text-2xl mr-2 text-gray-800" />
          <span className="text-3xl font-semibold leading-none text-gray-800">{stats?.expired || 0}</span>
        </div>
        <div className="text-xs text-gray-400">已过期凭证</div>
      </Card>

      <Card size="small" loading={loading} className="border border-[#e8edf3] rounded-[10px] shadow-none">
        <div className="text-sm text-gray-500 mb-2">最近更新</div>
        <div className="flex items-center mb-2">
          <SyncOutlined className="text-2xl mr-2 text-gray-800" />
          <span className="text-xl font-semibold leading-none text-gray-800 pt-1">{stats?.recentUpdate || '-'}</span>
        </div>
        <div className="text-xs text-gray-400">由 {stats?.recentUpdateBy || '-'} 更新</div>
      </Card>
    </div>
  );
};
