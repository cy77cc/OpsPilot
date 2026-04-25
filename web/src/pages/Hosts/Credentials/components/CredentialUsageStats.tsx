import React from 'react';
import type { CredentialDetailViewModel } from '../viewModels';

export const CredentialUsageStats: React.FC<{ detail: CredentialDetailViewModel }> = ({ detail }) => {
  const stats = [
    { label: '使用次数', value: detail.usageCount.toLocaleString() },
    { label: '成功次数', value: detail.successCount.toLocaleString() },
    { label: '失败次数', value: detail.failureCount.toLocaleString() },
    { label: '成功率', value: `${detail.successRate}%` },
  ];

  return (
    <section className="border-b border-[#edf2f7] pb-6">
      <h3 className="mb-1 text-[16px] font-semibold text-[#111827]">使用统计 <span className="font-normal text-[#6b7280]">(近 30 天)</span></h3>
      <div className="mt-4 grid grid-cols-4 gap-0 overflow-hidden rounded-xl border border-[#edf2f7] bg-[#fcfdff]">
        {stats.map((item) => (
          <div
            key={item.label}
            className="border-r border-[#edf2f7] px-4 py-4 last:border-r-0"
          >
            <div className="text-[12px] text-[#6b7280]">{item.label}</div>
            <div className="mt-3 text-[31px] font-semibold leading-none text-[#111827]">{item.value}</div>
          </div>
        ))}
      </div>
    </section>
  );
};
