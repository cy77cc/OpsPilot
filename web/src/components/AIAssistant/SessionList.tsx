import React from 'react';
import { Button, List, Typography } from 'antd';
import type { AISession } from '../../api/modules/ai';

interface SessionListProps {
  sessions: AISession[];
  activeSessionId?: string;
  onSelect: (session: AISession) => void;
  onNewSession: () => void;
}

const SessionList: React.FC<SessionListProps> = ({ sessions, activeSessionId, onSelect, onNewSession }) => {
  return (
    <div>
      <Button type="primary" block onClick={onNewSession} style={{ marginBottom: 12 }}>
        新建对话
      </Button>
      <List
        dataSource={sessions}
        locale={{ emptyText: '暂无会话' }}
        renderItem={(session) => (
          <List.Item
            style={{
              cursor: 'pointer',
              borderRadius: 12,
              padding: '10px 12px',
              marginBottom: 8,
              background: session.id === activeSessionId ? '#f0f7ff' : '#fff',
              border: session.id === activeSessionId ? '1px solid #91caff' : '1px solid #f0f0f0',
            }}
            onClick={() => onSelect(session)}
          >
            <div>
              <Typography.Text strong>{session.title || '未命名对话'}</Typography.Text>
            </div>
          </List.Item>
        )}
      />
    </div>
  );
};

export default SessionList;
