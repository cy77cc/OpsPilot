import React, { useState } from 'react';
import { Form, Tooltip } from 'antd';
import SparklesIcon from '../common/SparklesIcon';
import FieldGuideCard from './FieldGuideCard';
import { useFormAssist } from '../../features/ai/hooks/useFormAssist';
import AIFormAssistantPopover from './AIFormAssistantPopover';
import type { GuidedFormItemProps, FocusableChildProps } from './types';

const callFocusHandler = (
  handler: React.FocusEventHandler<HTMLElement> | undefined,
  event: React.FocusEvent<HTMLElement>,
) => {
  if (handler) handler(event);
};

/**
 * AIFieldWrapper handles the layout for AI-assisted fields while ensuring
 * Ant Design Form.Item can still inject value and onChange props.
 */
const AIFieldWrapper: React.FC<{
  children: React.ReactElement;
  aiTrigger: React.ReactNode;
  id?: string;
  value?: any;
  onChange?: (val: any) => void;
}> = ({ children, aiTrigger, id, value, onChange }) => {
  // Determine if it's a textarea to adjust vertical positioning
  const isTextArea = children.props.rows !== undefined;
  
  return (
    <div className="relative w-full flex items-center">
      <div className="flex-1">
        {React.cloneElement(children, { 
          id, 
          value, 
          onChange,
          style: { ...children.props.style, paddingRight: '32px' }
        })}
      </div>
      <div className={`absolute right-1 z-10 flex items-center ${isTextArea ? 'top-1' : 'top-1/2 -translate-y-1/2'}`}>
        {aiTrigger}
      </div>
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
    if (aiAssist) return aiAssist;
    
    // If global assist is disabled, don't auto-infer
    const globalEnabled = typeof window !== 'undefined' && localStorage.getItem('ai-form-assist-enabled') !== '0';
    if (!globalEnabled) return undefined;

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
    isEnabled,
    isOpen,
    isStreaming,
    prompt: aiPrompt,
    preview,
    error,
    showHint,
    open,
    cancel,
    submit,
    applySuggestion,
    dismissHint,
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

  const mergedExtra =
    isFocused && guide ? (
      <div className="space-y-2">
        <FieldGuideCard guide={guide} />
        {extra != null ? <div>{extra}</div> : null}
      </div>
    ) : (
      extra
    );

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

  const aiTrigger = (effectiveAiAssist && isEnabled) ? (
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
      <Tooltip
        title="✨ 需要 AI 帮助？"
        open={showHint}
        onOpenChange={(visible) => {
          if (!visible) dismissHint();
        }}
        placement="topRight"
        color="#6366f1" // indigo-500
      >
        <div 
          className="flex items-center justify-center h-8 w-8 rounded-full hover:bg-indigo-50 transition-colors"
          onClick={(e) => {
            e.stopPropagation();
            open();
          }}
        >
          <SparklesIcon
            className={`cursor-pointer ${
              isStreaming ? 'animate-pulse' : ''
            }`}
          />
        </div>
      </Tooltip>
    </AIFormAssistantPopover>
  ) : null;

  return (
    <Form.Item {...formItemProps} extra={mergedExtra}>
      {aiTrigger ? (
        <AIFieldWrapper aiTrigger={aiTrigger} id={formItemProps.id}>{enhancedChild}</AIFieldWrapper>
      ) : (
        enhancedChild
      )}
    </Form.Item>
  );
};

export default GuidedFormItem;
