import React, { useMemo } from 'react';
import { Drawer, Descriptions, Tag, Typography, Timeline, Divider, Empty, Spin, Space } from 'antd';
import { useCMDBAsset, useCMDBAudits } from '../../../hooks/useCMDB';
import dayjs from 'dayjs';

const { Text, Paragraph } = Typography;

interface AssetDetailDrawerProps {
  ciId: string | number | null;
  visible: boolean;
  onClose: () => void;
}

const AssetDetailDrawer: React.FC<AssetDetailDrawerProps> = ({ ciId, visible, onClose }) => {
  const assetId = ciId ? String(ciId) : undefined;
  
  const { data: assetRes, loading: assetLoading } = useCMDBAsset(assetId);
  const { data: auditRes, loading: auditLoading } = useCMDBAudits(assetId);

  const asset = assetRes?.data;
  const audits = useMemo(() => {
    if (auditRes?.data && Array.isArray(auditRes.data)) {
      return auditRes.data;
    }
    // Return mock data if not implemented or empty as requested
    if (!auditLoading && assetId) {
      return [
        {
          id: 'mock-1',
          action_type: 'UPDATE',
          operator: 'admin',
          created_at: dayjs().subtract(1, 'hour').toISOString(),
          details: { before: { status: 'pending' }, after: { status: 'running' } }
        },
        {
          id: 'mock-2',
          action_type: 'CREATE',
          operator: 'system',
          created_at: dayjs().subtract(1, 'day').toISOString(),
          details: { after: { name: asset?.name || 'New Asset' } }
        }
      ];
    }
    return [];
  }, [auditRes, auditLoading, assetId, asset?.name]);

  const parsedAttrs = useMemo(() => {
    if (!asset?.attrsJson) return {};
    try {
      return JSON.parse(asset.attrsJson);
    } catch (e) {
      console.error('Failed to parse attrsJson:', e);
      return {};
    }
  }, [asset?.attrsJson]);

  const renderJsonValue = (value: any) => {
    if (typeof value === 'object' && value !== null) {
      return (
        <Paragraph ellipsis={{ rows: 2, expandable: true, symbol: 'more' }} style={{ marginBottom: 0 }}>
          <Text code>{JSON.stringify(value)}</Text>
        </Paragraph>
      );
    }
    return String(value);
  };

  return (
    <Drawer
      title="资产详情"
      placement="right"
      onClose={onClose}
      open={visible}
      size="large"
    >
      {assetLoading ? (
        <div style={{ textAlign: 'center', padding: '50px' }}>
          <Spin description="加载中..." />
        </div>
      ) : asset ? (
        <div>
          <Descriptions title="基础信息" bordered column={1} size="small">
            <Descriptions.Item label="名称">{asset.name}</Descriptions.Item>
            <Descriptions.Item label="类型">
              <Tag color="blue">{asset.assetType}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="环境">
              <Tag>{asset.env || 'default'}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="地域">{asset.region || '-'}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={asset.status === 'online' ? 'green' : 'orange'}>{asset.status}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="来源">{asset.source}</Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {asset.createdAt ? dayjs(asset.createdAt).format('YYYY-MM-DD HH:mm:ss') : '-'}
            </Descriptions.Item>
          </Descriptions>

          <Divider orientation={"left" as any} orientationMargin="0">扩展属性</Divider>
          {Object.keys(parsedAttrs).length > 0 ? (
            <Descriptions bordered column={1} size="small">
              {Object.entries(parsedAttrs).map(([key, value]) => (
                <Descriptions.Item key={key} label={key}>
                  {renderJsonValue(value)}
                </Descriptions.Item>
              ))}
            </Descriptions>
          ) : (
            <Empty description="暂无扩展属性" image={Empty.PRESENTED_IMAGE_SIMPLE} />
          )}

          <Divider orientation={"left" as any} orientationMargin="0">审计日志</Divider>
          {auditLoading ? (
            <Spin size="small" />
          ) : audits.length > 0 ? (
            <Timeline
              items={audits.map((item: any) => ({
                color: item.action_type === 'CREATE' ? 'green' : item.action_type === 'DELETE' ? 'red' : 'blue',
                children: (
                  <div key={item.id}>
                    <Space>
                      <Text strong>{item.action_type}</Text>
                      <Text type="secondary">by {item.operator}</Text>
                      <Text type="secondary" style={{ fontSize: '12px' }}>
                        {dayjs(item.created_at).format('YYYY-MM-DD HH:mm:ss')}
                      </Text>
                    </Space>
                    {item.details && (
                      <div style={{ marginTop: '8px', padding: '8px', background: '#f5f5f5', borderRadius: '4px' }}>
                        {item.details.before && (
                          <div>
                            <Text type="secondary" delete>Before: {JSON.stringify(item.details.before)}</Text>
                          </div>
                        )}
                        {item.details.after && (
                          <div>
                            <Text type="success">After: {JSON.stringify(item.details.after)}</Text>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                ),
              }))}
            />
          ) : (
            <Empty description="暂无变更历史" image={Empty.PRESENTED_IMAGE_SIMPLE} />
          )}
        </div>
      ) : (
        <Empty description="未找到资产详情" />
      )}
    </Drawer>
  );
};

export default AssetDetailDrawer;
