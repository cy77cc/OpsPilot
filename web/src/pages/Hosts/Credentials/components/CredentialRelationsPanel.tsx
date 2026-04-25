import React from 'react';
import type { CredentialDetailViewModel } from '../viewModels';

export const CredentialRelationsPanel: React.FC<{ detail: CredentialDetailViewModel }> = ({ detail }) => {
  const items = [
    { label: '关联主机', value: `${detail.hostCount} 台`, action: '查看' },
    { label: '关联集群', value: `${detail.relationClusterCount} 个`, action: '查看' },
    { label: '最近使用', value: detail.recentUsage || '-' },
  ];

  return (
    <section className="border-b border-[#edf2f7] pb-6">
      <h3 className="mb-4 text-[20px] font-semibold text-[#111827]">关联信息</h3>
      <div className="space-y-3 text-[14px]">
        {items.map((item) => (
          <div key={item.label} className="flex items-center justify-between gap-4">
            <div className="text-[#6b7280]">{item.label}</div>
            <div className="flex items-center gap-3 text-[#111827]">
              <span>{item.value}</span>
              {item.action ? (
                <button type="button" className="text-[#2f6bff]">
                  {item.action}
                </button>
              ) : null}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
};
