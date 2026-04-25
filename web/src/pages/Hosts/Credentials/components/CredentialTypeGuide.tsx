import React from 'react';
import { KeyOutlined, SafetyCertificateOutlined, SafetyOutlined, UnlockOutlined } from '@ant-design/icons';

const items = [
  {
    title: 'SSH 密钥',
    icon: KeyOutlined,
    accent: '#f59e0b',
    soft: '#fff7e6',
    description: '使用 SSH 公私钥对进行认证，安全性高，推荐使用',
    lines: ['支持格式：RSA、ED25519', '使用场景：Linux 服务器登录'],
  },
  {
    title: '用户名密码',
    icon: UnlockOutlined,
    accent: '#2f6bff',
    soft: '#edf3ff',
    description: '使用用户名和密码进行认证',
    lines: ['使用场景：Windows 服务器、网络设备等'],
  },
  {
    title: 'Token',
    icon: SafetyOutlined,
    accent: '#7a5af8',
    soft: '#f3efff',
    description: '使用访问令牌进行认证',
    lines: ['使用场景：API 接口、Kubernetes 等'],
  },
  {
    title: '证书',
    icon: SafetyCertificateOutlined,
    accent: '#22a06b',
    soft: '#edf9f1',
    description: '使用 SSL/TLS 证书进行认证',
    lines: ['使用场景：HTTPS 服务、数据库等'],
  },
];

export const CredentialTypeGuide: React.FC = () => {
  return (
    <section className="rounded-2xl border border-[#e6edf5] bg-white px-5 py-5 shadow-[0_12px_28px_rgba(15,23,42,0.04)]">
      <h2 className="mb-4 text-[20px] font-semibold text-[#111827]">凭证类型说明</h2>
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-4 md:grid-cols-2">
        {items.map((item) => {
          const Icon = item.icon;
          return (
            <div key={item.title} className="rounded-2xl border border-[#edf2f7] bg-[#fdfefe] p-4">
              <div
                className="mb-4 flex h-10 w-10 items-center justify-center rounded-full text-[18px]"
                style={{ backgroundColor: item.soft, color: item.accent }}
              >
                <Icon />
              </div>
              <div className="text-[20px] font-semibold text-[#111827]">{item.title}</div>
              <div className="mt-3 text-[13px] leading-6 text-[#6b7280]">{item.description}</div>
              <div className="mt-4 space-y-2 text-[13px] text-[#4b5565]">
                {item.lines.map((line) => (
                  <div key={line}>{line}</div>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
};
