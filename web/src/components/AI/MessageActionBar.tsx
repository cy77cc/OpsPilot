import React from 'react';
import {
  CopyOutlined,
  LikeOutlined,
  DislikeOutlined,
  ReloadOutlined,
  LoadingOutlined,
} from '@ant-design/icons';
import { Button, Space, Tooltip, Typography } from 'antd';
import { aiApi } from '../../api/modules/ai';
import type { AssistantReplyRuntime, ChatRequest, XChatMessage } from './types';
import type { AIMessage } from '../../features/ai/api/shared';

const { Text } = Typography;

interface MessageActionBarProps {
  status?: string;
  messageId?: string;
  message?: AIMessage;
  runtime?: AssistantReplyRuntime;
  renderedMessages: Array<{ renderKey: string; message: XChatMessage }>;
  currentKey: string;
  onRequest: (requestParams: Partial<ChatRequest>) => void;
  onSuccess: (msg: string) => void;
  onError: (msg: string) => void;
  copyAssistantReplyToClipboard: (
    content: string,
    runtime?: AssistantReplyRuntime,
  ) => Promise<void>;
}

export function MessageActionBar({
  status,
  messageId,
  message,
  runtime,
  renderedMessages,
  currentKey,
  onRequest,
  onSuccess,
  onError,
  copyAssistantReplyToClipboard,
}: MessageActionBarProps) {
  const isStreaming = status === 'loading' || status === 'updating';

  if (isStreaming) {
    return (
      <Space size={6} style={{ color: 'rgba(17, 24, 39, 0.62)' }}>
        <LoadingOutlined spin />
        <Text type="secondary">正在生成...</Text>
      </Space>
    );
  }

  return (
    <div style={{ display: 'flex', gap: 2 }}>
      <Tooltip title="复制">
        <Button
          type="text"
          size="small"
          icon={<CopyOutlined />}
          onClick={async () => {
            try {
              await copyAssistantReplyToClipboard(
                message?.content || '',
                runtime,
              );
              onSuccess('内容已复制');
            } catch {
              onError('复制失败');
            }
          }}
        />
      </Tooltip>
      <Tooltip title="赞同">
        <Button
          type="text"
          size="small"
          icon={<LikeOutlined />}
          onClick={async () => {
            if (!messageId) return;
            try {
              await aiApi.submitMessageFeedback(messageId, 'like');
              onSuccess('感谢您的反馈！');
            } catch {
              onError('提交反馈失败');
            }
          }}
        />
      </Tooltip>
      <Tooltip title="踩">
        <Button
          type="text"
          size="small"
          icon={<DislikeOutlined />}
          onClick={async () => {
            if (!messageId) return;
            try {
              await aiApi.submitMessageFeedback(messageId, 'dislike');
              onSuccess('感谢您的反馈！');
            } catch {
              onError('提交反馈失败');
            }
          }}
        />
      </Tooltip>
      <Tooltip title="重新生成">
        <Button
          type="text"
          size="small"
          icon={<ReloadOutlined />}
          onClick={() => {
            const currentIndex = renderedMessages.findIndex(
              (m) => m.renderKey === currentKey,
            );
            if (currentIndex === -1) return;

            let lastUserPrompt = '';
            for (let i = currentIndex - 1; i >= 0; i -= 1) {
              if (renderedMessages[i].message.role === 'user') {
                lastUserPrompt = renderedMessages[i].message.content || '';
                break;
              }
            }

            if (lastUserPrompt) {
              onRequest({ message: lastUserPrompt });
            }
          }}
        />
      </Tooltip>
    </div>
  );
}
