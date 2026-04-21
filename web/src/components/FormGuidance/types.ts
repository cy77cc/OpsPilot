import type { FormItemProps } from 'antd';
import type React from 'react';
import type { FormAssistConfig } from '../../features/ai/types/formAssist';

export type FieldGuide = {
  icon?: React.ReactNode;
  whatToEnter: string;
  purpose: string;
  example?: string;
  impact?: string;
  whenRequired?: string;
  formatNotes?: string;
};

export type FocusableChildProps = {
  onFocus?: React.FocusEventHandler<HTMLElement>;
  onBlur?: React.FocusEventHandler<HTMLElement>;
};

export interface GuidedFormItemProps extends Omit<FormItemProps, 'children'> {
  guide?: FieldGuide;
  aiAssist?: FormAssistConfig;
  children: React.ReactElement<FocusableChildProps>;
}
