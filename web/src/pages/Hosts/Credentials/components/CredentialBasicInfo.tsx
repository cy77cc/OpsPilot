import React from 'react';
import { Button, Tag } from 'antd';
import { CopyOutlined, PlusOutlined } from '@ant-design/icons';
import type { CredentialDetailViewModel } from '../viewModels';

export const CredentialBasicInfo: React.FC<{ detail: CredentialDetailViewModel }> = ({ detail }) => {
  const items = [
    {
      label: '凭证名称',
      value: detail.name,
      extra: <Button type="text" size="small" icon={<CopyOutlined />} className="!h-auto !p-0 !text-[#94a3b8]" />,
    },
    { label: '凭证类型', value: detail.typeLabel },
    { label: '认证方式', value: detail.authMethodLabel },
    { label: '描述', value: detail.description || '-' },
    {
      label: '标签',
      value: (
        <div className="flex flex-wrap gap-2">
          {detail.tags?.map((tag) => (
            <Tag key={tag} className="!rounded-md !border-[#d8e8ff] !bg-[#f7fbff] !px-2 !py-0 !text-[#4f83ff]">
              {tag}
            </Tag>
          ))}
          <Button type="text" size="small" icon={<PlusOutlined />} className="!h-auto !p-0 !text-[#94a3b8]" />
        </div>
      ),
    },
    { label: '创建时间', value: detail.createdAt },
    { label: '创建人', value: detail.createdBy },
    { label: '更新时间', value: detail.updatedAt },
    { label: '更新人', value: detail.updatedBy },
  ];

  return (
    <section className="border-b border-[#edf2f7] pb-6">
      <h3 className="mb-4 text-[16px] font-semibold text-[#111827]">基本信息</h3>
      <div className="space-y-3">
        {items.map((item) => (
          <div key={item.label} className="flex items-start gap-4 text-[13px]">
            <div className="w-[74px] shrink-0 text-[#6b7280]">{item.label}</div>
            <div className="flex min-w-0 flex-1 items-center gap-2 text-[#111827]">
              <div className="break-all">{item.value}</div>
              {item.extra || null}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
};
