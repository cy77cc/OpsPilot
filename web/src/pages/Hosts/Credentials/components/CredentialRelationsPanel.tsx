import React from 'react';
import { List, Typography } from 'antd';
import type { CredentialDetail } from '../../../../api/modules/hosts';

const { Text } = Typography;

export const CredentialRelationsPanel: React.FC<{ detail: CredentialDetail }> = ({ detail }) => {
  return (
    <div>
      <h3 className="font-semibold mb-2">关联信息</h3>
      <List size="small" bordered className="bg-white">
        <List.Item className="flex justify-between">
          <Text>关联主机</Text>
          <div>
            <Text strong className="mr-2">{detail.hostCount} 台</Text>
            <a>查看</a>
          </div>
        </List.Item>
        <List.Item className="flex justify-between">
          <Text>最近使用</Text>
          <Text>{detail.recentUsage || '-'}</Text>
        </List.Item>
      </List>
    </div>
  );
};