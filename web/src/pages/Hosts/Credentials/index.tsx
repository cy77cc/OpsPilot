import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Card } from 'antd';
import { useSearchParams } from 'react-router-dom';
import { hostApi } from '../../../api/modules/hosts';
import type { CredentialDetail, CredentialItem, CredentialStats } from '../../../api/modules/hosts';
import { CredentialAuditTable } from './components/CredentialAuditTable';
import { CredentialDetailDrawer } from './components/CredentialDetailDrawer';
import { CredentialPermissionTable } from './components/CredentialPermissionTable';
import { CredentialStatsCards } from './components/CredentialStatsCards';
import { CredentialTable } from './components/CredentialTable';
import { CredentialTabs } from './components/CredentialTabs';
import { CredentialToolbar } from './components/CredentialToolbar';
import { CredentialTypeGuide } from './components/CredentialTypeGuide';
import { CreateCredentialModal } from './components/CreateCredentialModal';
import { PresetAuthTable } from './components/PresetAuthTable';
import {
  toCredentialDetailViewModel,
  toCredentialRowViewModel,
  type CredentialDetailViewModel,
} from './viewModels';

const CredentialsPage: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const activeTab = searchParams.get('tab') || 'key-management';

  const [credentials, setCredentials] = useState<CredentialItem[]>([]);
  const [stats, setStats] = useState<CredentialStats>();
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [selectedId, setSelectedId] = useState<string>();
  const [detail, setDetail] = useState<CredentialDetailViewModel | null>(null);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isDetailOpen, setIsDetailOpen] = useState(true);

  const [searchKeyword, setSearchKeyword] = useState('');
  const [filterType, setFilterType] = useState<string | undefined>();
  const [filterStatus, setFilterStatus] = useState<string | undefined>();

  const fetchStats = useCallback(() => {
    hostApi.getCredentialStats().then((res) => {
      if (res?.success) {
        setStats(res.data);
      }
    });
  }, []);

  const fetchCredentials = useCallback(() => {
    setLoading(true);
    hostApi
      .getCredentials()
      .then((credsRes) => {
        if (credsRes?.success) {
          const list = Array.isArray(credsRes.data?.list)
            ? credsRes.data.list
            : Array.isArray(credsRes.data)
              ? credsRes.data
              : [];
          setCredentials(list);
        }
      })
      .finally(() => setLoading(false));
  }, []);

  const fetchDetail = useCallback((id: string) => {
    setDetailLoading(true);
    hostApi
      .getCredentialDetail(id)
      .then((res) => {
        if (res?.success) {
          setDetail(toCredentialDetailViewModel(res.data as CredentialDetail));
        }
      })
      .finally(() => setDetailLoading(false));
  }, []);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  useEffect(() => {
    if (activeTab === 'key-management') {
      fetchCredentials();
    }
  }, [activeTab, fetchCredentials]);

  const filteredCredentials = useMemo(() => {
    const normalized = credentials.map(toCredentialRowViewModel);
    return normalized.filter((item) => {
      const keyword = searchKeyword.trim().toLowerCase();
      const matchKeyword =
        !keyword ||
        item.name.toLowerCase().includes(keyword) ||
        item.description?.toLowerCase().includes(keyword) ||
        item.tags.some((tag) => tag.toLowerCase().includes(keyword));
      const matchType = !filterType || item.type === filterType;
      const matchStatus = !filterStatus || item.status === filterStatus;
      return matchKeyword && matchType && matchStatus;
    });
  }, [credentials, searchKeyword, filterStatus, filterType]);

  useEffect(() => {
    if (activeTab !== 'key-management') {
      return;
    }
    if (filteredCredentials.length === 0) {
      setSelectedId(undefined);
      setDetail(null);
      return;
    }
    const stillExists = selectedId && filteredCredentials.some((item) => item.id === selectedId);
    const nextId = stillExists ? selectedId : filteredCredentials[0]?.id;
    if (nextId && nextId !== selectedId) {
      setSelectedId(nextId);
      setIsDetailOpen(true);
    }
  }, [activeTab, filteredCredentials, selectedId]);

  useEffect(() => {
    if (!selectedId || activeTab !== 'key-management') {
      return;
    }
    fetchDetail(selectedId);
  }, [activeTab, fetchDetail, selectedId]);

  const handleRefresh = () => {
    fetchStats();
    if (activeTab === 'key-management') {
      fetchCredentials();
      if (selectedId) {
        fetchDetail(selectedId);
      }
    }
  };

  const handleTabChange = (key: string) => {
    setSearchParams({ tab: key });
    if (key !== 'key-management') {
      setDetail(null);
      setSelectedId(undefined);
    }
  };

  return (
    <div className="h-[calc(100vh-112px)] flex flex-col gap-4 overflow-auto bg-[#f7f9fc]">
      <div className="flex w-full flex-col gap-3">

        <Card
          className="rounded-2xl border border-[#e6edf5] bg-white shadow-[0_8px_24px_rgba(15,23,42,0.04)]"
          styles={{ body: { padding: '0 20px' } }}
        >
          <CredentialTabs activeKey={activeTab} onChange={handleTabChange} />
        </Card>

        <CredentialStatsCards stats={stats} loading={!stats && loading} />

        {activeTab === 'key-management' ? (
          <>
            <section className="overflow-hidden rounded-2xl border border-[#e6edf5] bg-white shadow-[0_12px_28px_rgba(15,23,42,0.04)]">
              <div className="grid min-h-[720px] grid-cols-1 xl:grid-cols-[minmax(0,1fr)_392px]">
                <div className="min-w-0 px-3 py-3 xl:border-r xl:border-[#edf2f7]">
                  <CredentialToolbar
                    onRefresh={handleRefresh}
                    onCreate={() => setIsCreateModalOpen(true)}
                    onSearch={setSearchKeyword}
                    onTypeChange={setFilterType}
                    onStatusChange={setFilterStatus}
                  />
                  <CredentialTable
                    data={filteredCredentials}
                    loading={loading}
                    selectedId={selectedId}
                    onRowClick={(record) => {
                      setSelectedId(record.id);
                      setIsDetailOpen(true);
                    }}
                    onRefresh={handleRefresh}
                  />
                </div>

                <CredentialDetailDrawer
                  detail={detail}
                  loading={detailLoading}
                  open={isDetailOpen}
                  onClose={() => setIsDetailOpen(false)}
                  onRefresh={handleRefresh}
                />
              </div>
            </section>

            <CredentialTypeGuide />

            <CreateCredentialModal
              open={isCreateModalOpen}
              onCancel={() => setIsCreateModalOpen(false)}
              onSuccess={() => {
                setIsCreateModalOpen(false);
                handleRefresh();
              }}
            />
          </>
        ) : null}

        {activeTab === 'preset-auth' ? (
          <Card className="rounded-2xl border border-[#e6edf5] shadow-[0_8px_24px_rgba(15,23,42,0.04)]" styles={{ body: { padding: 20 } }}>
            <PresetAuthTable />
          </Card>
        ) : null}

        {activeTab === 'usage-audit' ? (
          <Card className="rounded-2xl border border-[#e6edf5] shadow-[0_8px_24px_rgba(15,23,42,0.04)]" styles={{ body: { padding: 20 } }}>
            <CredentialAuditTable />
          </Card>
        ) : null}

        {activeTab === 'permission' ? (
          <Card className="rounded-2xl border border-[#e6edf5] shadow-[0_8px_24px_rgba(15,23,42,0.04)]" styles={{ body: { padding: 20 } }}>
            <CredentialPermissionTable />
          </Card>
        ) : null}
      </div>
    </div>
  );
};

export default CredentialsPage;
