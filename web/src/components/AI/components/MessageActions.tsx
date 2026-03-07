import React from 'react';
import { Button, Tooltip, message, Divider } from 'antd';
import {
  CopyOutlined,
  LikeOutlined,
  DislikeOutlined,
  ReloadOutlined,
} from '@ant-design/icons';

interface MessageActionsProps {
  /** 消息内容，用于复制 */
  content: string;
  /** 消息 ID，用于重新生成 */
  messageId: string;
  /** 是否正在加载 */
  isLoading?: boolean;
  /** 复制回调 */
  onCopy?: () => void;
  /** 重新生成回调 */
  onRegenerate?: () => void;
  /** 点赞回调 */
  onLike?: () => void;
  /** 点踩回调 */
  onDislike?: () => void;
}

/**
 * 消息操作组件
 * 提供复制、点赞、点踩、重新生成等操作
 */
export function MessageActions({
  content,
  messageId,
  isLoading = false,
  onCopy,
  onRegenerate,
  onLike,
  onDislike,
}: MessageActionsProps) {
  // 复制到剪贴板
  const handleCopy = async () => {
    if (!content) {
      message.warning('没有可复制的内容');
      return;
    }

    try {
      await navigator.clipboard.writeText(content);
      message.success('已复制到剪贴板');
      onCopy?.();
    } catch (err) {
      // 降级方案：使用 document.execCommand
      try {
        const textArea = document.createElement('textarea');
        textArea.value = content;
        textArea.style.position = 'fixed';
        textArea.style.left = '-9999px';
        document.body.appendChild(textArea);
        textArea.select();
        document.execCommand('copy');
        document.body.removeChild(textArea);
        message.success('已复制到剪贴板');
        onCopy?.();
      } catch {
        message.error('复制失败，请检查浏览器权限');
      }
    }
  };

  // 重新生成
  const handleRegenerate = () => {
    if (isLoading) return;
    onRegenerate?.();
  };

  return (
    <div style={{ display: 'flex', marginTop: 4, alignItems: 'center' }}>
      <Tooltip title="复制">
        <Button
          type="text"
          size="small"
          icon={<CopyOutlined />}
          onClick={handleCopy}
          disabled={!content}
        />
      </Tooltip>
      <Tooltip title="有帮助">
        <Button
          type="text"
          size="small"
          icon={<LikeOutlined />}
          onClick={onLike}
        />
      </Tooltip>
      <Tooltip title="无帮助">
        <Button
          type="text"
          size="small"
          icon={<DislikeOutlined />}
          onClick={onDislike}
        />
      </Tooltip>
      <Divider type="vertical" style={{ height: 16, margin: '0 4px' }} />
      <Tooltip title={isLoading ? '重新生成中...' : '重新生成'}>
        <Button
          type="text"
          size="small"
          icon={<ReloadOutlined spin={isLoading} />}
          onClick={handleRegenerate}
          disabled={isLoading}
          loading={isLoading}
        />
      </Tooltip>
    </div>
  );
}
