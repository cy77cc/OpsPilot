import React from 'react';
import { Card, Typography } from 'antd';

const { Title, Text } = Typography;

export const CredentialTypeGuide: React.FC = () => {
  return (
    <div className="mt-8">
      <Title level={5} className="!mb-4 text-[#1f2937]">凭证类型说明</Title>
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
        <Card size="small" title="SSH 密钥" className="border border-[#e8edf3] rounded-[10px] shadow-none">
          <Text className="text-xs block text-gray-500 mb-2">使用 SSH 公私钥对进行认证，安全性高，推荐使用</Text>
          <Text className="text-xs block mb-1"><b>支持格式：</b> RSA, ED25519</Text>
          <Text className="text-xs block"><b>使用场景：</b> Linux 服务器登录</Text>
        </Card>
        
        <Card size="small" title="用户名密码" className="border border-[#e8edf3] rounded-[10px] shadow-none">
          <Text className="text-xs block text-gray-500 mb-2">使用用户名和密码进行认证</Text>
          <Text className="text-xs block"><b>使用场景：</b> Windows 服务器, 网络设备等</Text>
        </Card>
        
        <Card size="small" title="Token" className="border border-[#e8edf3] rounded-[10px] shadow-none">
          <Text className="text-xs block text-gray-500 mb-2">使用访问令牌进行认证</Text>
          <Text className="text-xs block"><b>使用场景：</b> API 接口, Kubernetes 等</Text>
        </Card>
        
        <Card size="small" title="证书" className="border border-[#e8edf3] rounded-[10px] shadow-none">
          <Text className="text-xs block text-gray-500 mb-2">使用 SSL/TLS 证书进行认证</Text>
          <Text className="text-xs block"><b>使用场景：</b> HTTPS 服务, 数据库等</Text>
        </Card>
      </div>
    </div>
  );
};
