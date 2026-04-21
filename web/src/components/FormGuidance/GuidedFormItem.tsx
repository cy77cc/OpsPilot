import React, { useState } from 'react';
import { Form } from 'antd';
import { QuestionCircleOutlined } from '@ant-design/icons';
import SparklesIcon from '../common/SparklesIcon';
import { useFormAssist } from '../../features/ai/hooks/useFormAssist';
import AIFormAssistantPopover from './AIFormAssistantPopover';
import type { GuidedFormItemProps, FocusableChildProps, FieldGuide } from './types';

const callFocusHandler = (
  handler: React.FocusEventHandler<HTMLElement> | undefined,
  event: React.FocusEvent<HTMLElement>,
) => {
  if (handler) handler(event);
};

const GuideTooltip: React.FC<{ guide: FieldGuide }> = ({ guide }) => (
  <div className="space-y-3 p-1">
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
);

/**
 * AIFieldWrapper handles the layout for AI-assisted fields while ensuring
 * Ant Design Form.Item can still inject value and onChange props.
 */
const AIFieldWrapper: React.FC<{
  children: React.ReactElement;
  aiTrigger: React.ReactNode;
} & any> = ({ children, aiTrigger, ...restProps }) => {
  // Determine if it's a textarea to adjust vertical positioning
  const isTextArea = children.props.rows !== undefined;
  
  return (
    <div className="relative w-full flex items-center">
      <div className="flex-1">
        {React.cloneElement(children, { 
          ...restProps,
          onFocus: (event: React.FocusEvent<HTMLElement>) => {
            if (restProps.onFocus) restProps.onFocus(event);
            if (children.props.onFocus) children.props.onFocus(event);
          },
          onBlur: (event: React.FocusEvent<HTMLElement>) => {
            if (restProps.onBlur) restProps.onBlur(event);
            if (children.props.onBlur) children.props.onBlur(event);
          },
          style: { ...children.props.style, paddingRight: aiTrigger ? '32px' : children.props.style?.paddingRight, ...restProps.style }
        })}
      </div>
      {aiTrigger && (
        <div className={`absolute right-1 z-10 flex items-center ${isTextArea ? 'top-1' : 'top-1/2 -translate-y-1/2'}`}>
          {aiTrigger}
        </div>
      )}
    </div>
  );
};

const GuidedFormItem: React.FC<GuidedFormItemProps> = ({ 
  guide, 
  aiAssist,
  extra, 
  children, 
  ...formItemProps 
}) => {
  const [isFocused, setIsFocused] = useState(false);
  const form = Form.useFormInstance();
  const name = formItemProps.name;
  
  // Watch the current field value for the AI hint logic.
  // Note: Form.useWatch is available in Ant Design 5.x+
  const currentValue = Form.useWatch(name, form) || '';
  
  // Auto-infer AI assistance if not explicitly provided but global feature is on
  const effectiveAiAssist = React.useMemo(() => {
    // If global assist is disabled, don't auto-infer or use provided assist
    const globalEnabled = typeof window !== 'undefined' && localStorage.getItem('ai-form-assist-enabled') !== '0';
    if (!globalEnabled) return undefined;

    if (aiAssist) return aiAssist;

    // Check if child is an AI-supported input type
    const childType = (children as any)?.type?.displayName || (children as any)?.type?.name || '';
    const isSupportedInput = ['Input', 'TextArea', 'Password'].some(name => childType.includes(name));
    
    if (isSupportedInput && name) {
      return {
        scene: 'general',
        fieldMeta: {
          key: String(name),
          label: String(formItemProps.label || name),
          purpose: guide?.purpose || '协助填写表单字段',
          rules: '直接输出结果，不要包含解释',
        }
      };
    }
    return undefined;
  }, [aiAssist, name, formItemProps.label, guide?.purpose, children]);

  const {
    isOpen,
    isStreaming,
    prompt: aiPrompt,
    preview,
    error,
    open,
    cancel,
    submit,
    applySuggestion,
  } = useFormAssist(effectiveAiAssist, currentValue, (val) => {
    if (name) {
      form.setFieldValue(name, val);
    }
  });

  if (!guide && !effectiveAiAssist) {
    return (
      <Form.Item {...formItemProps} extra={extra}>
        {children}
      </Form.Item>
    );
  }

  const child = children as React.ReactElement<FocusableChildProps>;

  const mergedExtra = extra;

  const enhancedChild = React.cloneElement(child, {
    onFocus: (event: React.FocusEvent<HTMLElement>) => {
      setIsFocused(true);
      callFocusHandler(child.props.onFocus, event);
    },
    onBlur: (event: React.FocusEvent<HTMLElement>) => {
      setIsFocused(false);
      callFocusHandler(child.props.onBlur, event);
    },
  });

  const aiTrigger = effectiveAiAssist ? (
    <AIFormAssistantPopover
      isOpen={isOpen}
      isStreaming={isStreaming}
      prompt={aiPrompt}
      preview={preview}
      error={error}
      onCancel={cancel}
      onSubmit={submit}
      onApply={applySuggestion}
    >
      <div 
        className="flex items-center justify-center h-8 w-8 rounded-lg hover:bg-indigo-50 transition-all cursor-pointer text-indigo-500" 
        onClick={(e) => {
          e.stopPropagation();
          open();
        }}
      >
        <SparklesIcon className={isStreaming ? "animate-pulse" : ""} />
      </div>
    </AIFormAssistantPopover>
  ) : null;

  return (
    <Form.Item 
      {...formItemProps} 
      extra={mergedExtra}
      tooltip={guide ? { title: <GuideTooltip guide={guide} />, color: 'white', styles: { root: { maxWidth: '280px' } } } : formItemProps.tooltip}
    >
      <AIFieldWrapper aiTrigger={aiTrigger}>{enhancedChild}</AIFieldWrapper>
    </Form.Item>
  );
};

export default GuidedFormItem;
