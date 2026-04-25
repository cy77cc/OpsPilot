import React, { useState, useEffect } from 'react';
import { Card } from 'antd';
import { useSearchParams } from 'react-router-dom';
import { CredentialTabs } from './components/CredentialTabs';
import { CredentialStatsCards } from './components/CredentialStatsCards';
import { CredentialToolbar } from './components/CredentialToolbar';
import { CredentialTable } from './components/CredentialTable';
import { CredentialDetailDrawer } from './components/CredentialDetailDrawer';
import { CredentialTypeGuide } from './components/CredentialTypeGuide';
import { PresetAuthTable } from './components/PresetAuthTable';
import { CredentialAuditTable } from './components/CredentialAuditTable';
import { CredentialPermissionTable } from './components/CredentialPermissionTable';
import { hostApi } from '../../../api/modules/hosts';
import type { CredentialItem, CredentialStats } from '../../../api/modules/hosts';

const CredentialsPage: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const activeTab = searchParams.get('tab') || 'key-management';

  const [credentials, setCredentials] = useState<CredentialItem[]>([]);
  const [stats, setStats] = useState<CredentialStats>();
  const [loading, setLoading] = useState(false);
  const [selectedId, setSelectedId] = useState<string | undefined>();

  useEffect(() => {
    // 初始加载统计数据（全局显示）
    hostApi.getCredentialStats().then(res => {
      if (res && res.success) setStats(res.data);
    });
  }, []);

  useEffect(() => {
    if (activeTab === 'key-management') {
      setLoading(true);
      console.log('Fetching credentials...');
      hostApi.getCredentials().then(credsRes => {
        console.log('Credentials Response:', credsRes);

        if (credsRes && credsRes.success) {
          // Handle various data structures
          let list = [];
          if (Array.isArray(credsRes.data)) {
            list = credsRes.data;
          } else if (credsRes.data && Array.isArray(credsRes.data.list)) {
            list = credsRes.data.list;
          } else if (credsRes.data && typeof credsRes.data === 'object') {
            // Fallback for unexpected object structure
            list = Object.values(credsRes.data).find(v => Array.isArray(v)) || [];
          }
          console.log('Resolved Credentials List:', list);
          setCredentials(list);
        }
      }).catch(err => {
        console.error('Failed to fetch credentials:', err);
      }).finally(() => setLoading(false));
    }
  }, [activeTab]);

  const handleTabChange = (key: string) => {
    setSearchParams({ tab: key });
    setSelectedId(undefined);
  };

  return (
    <div className="p-4 h-full flex flex-col gap-4 bg-[#f6f8fb] overflow-hidden">
      {/* 顶部统计卡片 */}
      <div className="flex-none">
        <CredentialStatsCards stats={stats} loading={!stats && loading} />
      </div>

      {/* 主体功能区 */}
      <div className="flex-1 flex flex-col gap-4 min-h-0 overflow-hidden">
        {/* Tab 导航卡片 */}
        <Card 
          size="small" 
          styles={{ body: { padding: '0 16px' } }}
          className="border border-[#e8edf3] rounded-[10px] shadow-none flex-none"
        >
          <CredentialTabs activeKey={activeTab} onChange={handleTabChange} />
        </Card>

        {/* 内容展示区 */}
        <div className="flex-1 min-h-0 overflow-auto pr-1 scrollbar-thin">
          {activeTab === 'key-management' && (
            <div className="flex flex-col gap-4 pb-2">
              <Card 
                size="small" 
                className="border border-[#e8edf3] rounded-[10px] shadow-none"
                styles={{ body: { padding: 16 } }}
              >
                <CredentialToolbar />
                <div className="[&_.ant-table-thead>tr>th]:!bg-[#f6f8fb] [&_.ant-table-thead>tr>th]:!text-[#6b7280] [&_.ant-table-thead>tr>th]:!text-[13px] [&_.ant-table-tbody>tr>td]:!text-[13px]">
                  <CredentialTable 
                    data={credentials} 
                    loading={loading} 
                    onRowClick={(record) => setSelectedId(record.id)} 
                  />
                </div>
              </Card>

              <CredentialTypeGuide />

              <CredentialDetailDrawer 
                credentialId={selectedId} 
                onClose={() => setSelectedId(undefined)} 
              />
            </div>
          )}

          {activeTab === 'preset-auth' && (
            <Card className="border border-[#e8edf3] rounded-[10px] shadow-none">
              <PresetAuthTable />
            </Card>
          )}

          {activeTab === 'usage-audit' && (
            <Card className="border border-[#e8edf3] rounded-[10px] shadow-none">
              <CredentialAuditTable />
            </Card>
          )}

          {activeTab === 'permission' && (
            <Card className="border border-[#e8edf3] rounded-[10px] shadow-none">
              <CredentialPermissionTable />
            </Card>
          )}
        </div>
      </div>
    </div>
  );
};

export default CredentialsPage;