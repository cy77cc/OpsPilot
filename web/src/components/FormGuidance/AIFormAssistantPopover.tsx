import React, { useState, useEffect } from 'react';
import { Popover, Input, Button, Alert, Typography, Spin } from 'antd';
import { LoadingOutlined, SendOutlined } from '@ant-design/icons';
import SparklesIcon from '../common/SparklesIcon';

const { TextArea } = Input;
const { Text } = Typography;

interface AIFormAssistantPopoverProps {
  isOpen: boolean;
  isStreaming: boolean;
  prompt: string;
  preview: string;
  error: string | null;
  onCancel: () => void;
  onSubmit: (prompt: string) => void;
  onApply: () => void;
  children: React.ReactNode;
}

const AIFormAssistantPopover: React.FC<AIFormAssistantPopoverProps> = ({
  isOpen,
  isStreaming,
  prompt: initialPrompt,
  preview,
  error,
  onCancel,
  onSubmit,
  onApply,
  children,
}) => {
  const [localPrompt, setLocalPrompt] = useState(initialPrompt);

  // Sync local prompt when initialPrompt changes (e.g. on reset)
  useEffect(() => {
    if (!isOpen) {
      setLocalPrompt('');
    }
  }, [isOpen]);

  const content = (
    <div className="w-[320px] space-y-4 p-1" onClick={(e) => e.stopPropagation()}>
      <div className="space-y-2">
        <div className="flex items-center gap-2 text-xs font-semibold text-indigo-600">
          <SparklesIcon />
          <span>AI 辅助生成</span>
        </div>
        <TextArea
          placeholder="描述你想要的内容，例如：'一个描述云原生架构优势的段落'"
          value={localPrompt}
          onChange={(e) => setLocalPrompt(e.target.value)}
          rows={3}
          disabled={isStreaming}
          autoFocus
        />
        <div className="flex justify-end">
          <Button
            type="primary"
            size="small"
            className="bg-indigo-600"
            icon={isStreaming ? <Spin indicator={<LoadingOutlined style={{ fontSize: 14, color: '#fff' }} spin />} /> : <SendOutlined />}
            onClick={() => onSubmit(localPrompt)}
            disabled={!localPrompt.trim() || isStreaming}
          >
            {isStreaming ? '生成中...' : '生成建议'}
          </Button>
        </div>
      </div>

      {(preview || isStreaming || error) && (
        <div className="space-y-2 border-t pt-3">
          <div className="text-xs font-semibold text-slate-500">建议结果</div>
          <div className="max-h-[200px] min-h-[60px] overflow-y-auto rounded-md bg-slate-50 p-2 text-sm text-slate-700 border border-slate-200">
            {preview ? (
              <div className="whitespace-pre-wrap">{preview}</div>
            ) : isStreaming ? (
              <Text type="secondary" italic>AI 正在构思中...</Text>
            ) : null}
            {error && <Alert message={error} type="error" showIcon className="mt-2" />}
          </div>
          <div className="flex justify-end gap-2">
            <Button size="small" onClick={onCancel}>
              取消
            </Button>
            <Button
              size="small"
              type="primary"
              className="bg-indigo-600"
              onClick={onApply}
              disabled={!preview || isStreaming}
            >
              采纳建议
            </Button>
          </div>
        </div>
      )}
    </div>
  );

  return (
    <Popover
      content={content}
      title={null}
      trigger="click"
      open={isOpen}
      onOpenChange={(visible) => {
        if (!visible && !isStreaming) {
          onCancel();
        }
      }}
      overlayClassName="ai-assist-popover"
    >
      {children}
    </Popover>
  );
};

export default AIFormAssistantPopover;
