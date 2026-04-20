import React from 'react';
import { Form } from 'antd';
import type { FormItemProps } from 'antd';
import FieldGuideCard from './FieldGuideCard';
import type { FieldGuide } from './types';

type FocusableChildProps = {
  onFocus?: React.FocusEventHandler<HTMLElement>;
  onBlur?: React.FocusEventHandler<HTMLElement>;
};

export interface GuidedFormItemProps extends Omit<FormItemProps, 'children'> {
  guide?: FieldGuide;
  children: React.ReactElement<FocusableChildProps>;
}

const callFocusHandler = (
  handler: React.FocusEventHandler<HTMLElement> | undefined,
  event: React.FocusEvent<HTMLElement>,
) => {
  if (handler) handler(event);
};

const GuidedFormItem: React.FC<GuidedFormItemProps> = ({ guide, extra, children, ...formItemProps }) => {
  if (!guide) {
    return (
      <Form.Item {...formItemProps} extra={extra}>
        {children}
      </Form.Item>
    );
  }

  const [isFocused, setIsFocused] = React.useState(false);
  const child = children as React.ReactElement<FocusableChildProps>;

  const mergedExtra =
    isFocused ? (
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

  return (
    <Form.Item {...formItemProps} extra={mergedExtra}>
      {enhancedChild}
    </Form.Item>
  );
};

export default GuidedFormItem;
