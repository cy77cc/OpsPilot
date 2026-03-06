import React from 'react';
import { Button, Empty } from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import type { Conversation } from '../types';

interface ConversationsPanelProps {
  conversations: Conversation[];
  currentId?: string;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  onCreate: () => void;
}

/**
 * 会话列表面板
 */
export function ConversationsPanel({
  conversations,
  currentId,
  onSelect,
  onDelete,
  onCreate,
}: ConversationsPanelProps) {
  if (conversations.length === 0) {
    return (
      <div className="ai-conversations-panel">
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="暂无会话"
          style={{ padding: '12px 0' }}
        >
          <Button type="primary" icon={<PlusOutlined />} onClick={onCreate}>
            新建会话
          </Button>
        </Empty>
      </div>
    );
  }

  return (
    <div className="ai-conversations-panel">
      <div className="ai-conversations-list">
        {conversations.map((conv) => (
          <div
            key={conv.id}
            className={`ai-conversation-item ${conv.id === currentId ? 'active' : ''}`}
            onClick={() => onSelect(conv.id)}
          >
            <span className="conversation-title">{conv.title}</span>
            <span className="conversation-time">
              {formatTime(conv.updatedAt || conv.createdAt)}
            </span>
            <Button
              type="text"
              size="small"
              icon={<DeleteOutlined />}
              className="conversation-delete"
              onClick={(e) => {
                e.stopPropagation();
                onDelete(conv.id);
              }}
            />
          </div>
        ))}
      </div>
    </div>
  );
}

/**
 * 格式化时间
 */
function formatTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  const days = Math.floor(diff / 86400000);

  if (minutes < 1) return '刚刚';
  if (minutes < 60) return `${minutes}分钟前`;
  if (hours < 24) return `${hours}小时前`;
  if (days < 7) return `${days}天前`;

  return date.toLocaleDateString('zh-CN', {
    month: 'numeric',
    day: 'numeric',
  });
}
