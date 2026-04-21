import React, { useState, useEffect } from 'react';
import { Popover, Input, Button, Alert, Typography, Spin } from 'antd';
import { LoadingOutlined, SendOutlined, QuestionCircleOutlined } from '@ant-design/icons';
import SparklesIcon from '../common/SparklesIcon';
import type { FieldGuide } from './types';

const { TextArea } = Input;
const { Text } = Typography;

export interface AIFormAssistantPopoverProps {
  guide?: FieldGuide;
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
  guide,
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
    <div className="w-80 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl" onClick={(e) => e.stopPropagation()}>
      {guide && (
        <div className="bg-slate-50 p-4 border-b border-slate-100">
          <div className="flex items-center gap-2 mb-3">
            <div className="bg-indigo-50 text-indigo-500 p-1 rounded-md flex items-center justify-center">
               <QuestionCircleOutlined style={{ fontSize: 14 }} />
            </div>
            <span className="text-[10px] font-bold text-slate-500 uppercase tracking-widest">填写指引</span>
          </div>
          <div className="space-y-3">
            {guide.whatToEnter && (
              <div>
                <div className="text-[10px] font-bold text-slate-400 uppercase mb-1">建议</div>
                <div className="text-sm text-slate-600 leading-relaxed">{guide.whatToEnter}</div>
              </div>
            )}
            {guide.example && (
              <div>
                <div className="text-[10px] font-bold text-slate-400 uppercase mb-1">示例</div>
                <code className="text-xs text-indigo-600 bg-indigo-50 px-1.5 py-0.5 rounded font-mono break-all">
                  {guide.example}
                </code>
              </div>
            )}
          </div>
        </div>
      )}
      
      <div className="p-4 space-y-4">
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
              className="bg-indigo-600 border-none"
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
                className="bg-indigo-600 border-none"
                onClick={onApply}
                disabled={!preview || isStreaming}
              >
                采纳建议
              </Button>
            </div>
          </div>
        )}
      </div>
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
      overlayInnerStyle={{ padding: 0, background: 'none', boxShadow: 'none', border: 'none' }}
    >
      {children}
    </Popover>
  );
};

export default AIFormAssistantPopover;
