import React from 'react';
import { Card, Col, Row, Statistic, List, Tag, Empty, Typography } from 'antd';
import {
  RobotOutlined,
  ThunderboltOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  MessageOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { AIActivity } from '../../api/modules/dashboard';

interface AIActivityCardProps {
  data: AIActivity;
  loading?: boolean;
}

const sceneColorMap: Record<string, string> = {
  host: 'blue',
  cluster: 'purple',
  service: 'green',
  k8s: 'cyan',
  default: 'default',
};

const sceneLabelMap: Record<string, string> = {
  host: '主机',
  cluster: '集群',
  service: '服务',
  k8s: 'K8s',
};

const formatRelativeTime = (time: string): string => {
  const at = dayjs(time);
  const now = dayjs();
  const diffMinutes = now.diff(at, 'minute');
  if (diffMinutes < 60) {
    return `${Math.max(1, diffMinutes)} 分钟前`;
  }
  const diffHours = now.diff(at, 'hour');
  if (diffHours < 24 && now.isSame(at, 'day')) {
    return `${diffHours} 小时前`;
  }
  return at.format('MM-DD HH:mm');
};

const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
};

const formatTokens = (count: number): string => {
  if (count < 1000) return `${count}`;
  if (count < 1000000) return `${(count / 1000).toFixed(1)}K`;
  return `${(count / 1000000).toFixed(1)}M`;
};

const AIActivityCard: React.FC<AIActivityCardProps> = ({ data, loading }) => {
  const { stats, sessions, byScene } = data;

  return (
    <Card
      title={
        <span>
          <RobotOutlined className="mr-2" />
          AI 助手活动
        </span>
      }
      styles={{ body: { padding: '16px 24px' } }}
      loading={loading}
    >
      {/* 统计指标 */}
      <Row gutter={16}>
        <Col span={6}>
          <Statistic
            title="会话数"
            value={stats.sessionCount}
            prefix={<MessageOutlined className="text-blue-500" />}
            valueStyle={{ fontSize: 20 }}
          />
        </Col>
        <Col span={6}>
          <Statistic
            title="Token 消耗"
            value={formatTokens(stats.tokenCount)}
            prefix={<ThunderboltOutlined className="text-orange-500" />}
            valueStyle={{ fontSize: 20 }}
          />
        </Col>
        <Col span={6}>
          <Statistic
            title="平均响应"
            value={formatDuration(stats.avgDurationMs)}
            prefix={<ClockCircleOutlined className="text-purple-500" />}
            valueStyle={{ fontSize: 20 }}
          />
        </Col>
        <Col span={6}>
          <Statistic
            title="成功率"
            value={stats.successRate}
            suffix="%"
            prefix={<CheckCircleOutlined className="text-green-500" />}
            valueStyle={{ fontSize: 20, color: stats.successRate >= 95 ? '#22c55e' : stats.successRate >= 80 ? '#f59e0b' : '#ef4444' }}
          />
        </Col>
      </Row>

      {/* 按场景分布 */}
      {Object.keys(byScene).length > 0 && (
        <div className="mt-4 mb-4">
          <Typography.Text type="secondary" className="text-xs">
            场景分布:
          </Typography.Text>
          <div className="mt-2 flex flex-wrap gap-2">
            {Object.entries(byScene).map(([scene, count]) => (
              <Tag key={scene} color={sceneColorMap[scene] || sceneColorMap.default}>
                {sceneLabelMap[scene] || scene}: {count}
              </Tag>
            ))}
          </div>
        </div>
      )}

      {/* 最近会话 */}
      <div className="mt-4">
        <Typography.Text type="secondary" className="text-xs">
          最近对话:
        </Typography.Text>
        <List
          className="mt-2"
          dataSource={sessions}
          locale={{ emptyText: <Empty description="暂无对话记录" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
          renderItem={(item) => (
            <List.Item className="!py-2 !px-0">
              <div className="flex w-full items-center justify-between">
                <div className="flex items-center gap-2">
                  <Tag color={sceneColorMap[item.scene] || sceneColorMap.default}>
                    {sceneLabelMap[item.scene] || item.scene}
                  </Tag>
                  <Typography.Text className="text-sm" ellipsis style={{ maxWidth: 200 }}>
                    {item.title}
                  </Typography.Text>
                </div>
                <Typography.Text type="secondary" className="text-xs">
                  {formatRelativeTime(item.createdAt)}
                </Typography.Text>
              </div>
            </List.Item>
          )}
        />
      </div>
    </Card>
  );
};

export default AIActivityCard;
