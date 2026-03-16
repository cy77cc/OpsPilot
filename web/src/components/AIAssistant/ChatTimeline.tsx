import React from 'react';
import { List, Typography } from 'antd';
import type { AIMessage } from '../../api/modules/ai';

interface ChatTimelineProps {
  messages: AIMessage[];
}

const ChatTimeline: React.FC<ChatTimelineProps> = ({ messages }) => {
  return (
    <List
      dataSource={messages}
      locale={{ emptyText: '发送第一条消息开始对话' }}
      renderItem={(message) => (
        <List.Item style={{ justifyContent: message.role === 'user' ? 'flex-end' : 'flex-start', border: 'none' }}>
          <div
            style={{
              maxWidth: '80%',
              padding: '12px 14px',
              borderRadius: 16,
              background: message.role === 'user' ? '#1677ff' : '#f5f5f5',
              color: message.role === 'user' ? '#fff' : 'inherit',
              whiteSpace: 'pre-wrap',
            }}
          >
            <Typography.Text style={{ color: 'inherit' }}>{message.content || '...'}</Typography.Text>
          </div>
        </List.Item>
      )}
    />
  );
};

export default ChatTimeline;
