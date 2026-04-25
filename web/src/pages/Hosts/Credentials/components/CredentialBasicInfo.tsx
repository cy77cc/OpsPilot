import React from 'react';
import { Descriptions, Tag } from 'antd';
import type { CredentialDetail } from '../../../../api/modules/hosts';

export const CredentialBasicInfo: React.FC<{ detail: CredentialDetail }> = ({ detail }) => {
  return (
    <div>
      <h3 className="font-semibold mb-2">基本信息</h3>
      <Descriptions column={2} size="small">
        <Descriptions.Item label="凭证名称">{detail.name}</Descriptions.Item>
        <Descriptions.Item label="类型">{detail.type}</Descriptions.Item>
        <Descriptions.Item label="认证方式">{detail.authMethod}</Descriptions.Item>
        <Descriptions.Item label="创建时间">{detail.createdAt}</Descriptions.Item>
        <Descriptions.Item label="创建人">{detail.createdBy}</Descriptions.Item>
        <Descriptions.Item label="更新时间">{detail.updatedAt}</Descriptions.Item>
        <Descriptions.Item label="更新人">{detail.updatedBy}</Descriptions.Item>
        <Descriptions.Item label="标签" span={2}>
          {detail.tags.map(tag => <Tag key={tag}>{tag}</Tag>)}
        </Descriptions.Item>
        <Descriptions.Item label="描述" span={2}>{detail.description}</Descriptions.Item>
      </Descriptions>
    </div>
  );
};