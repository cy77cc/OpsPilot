# Guided Form Item Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign `GuidedFormItem` to use a manual exclamation icon trigger that opens a Popover, replacing the focus-based automatic expansion.

**Architecture:** 
- Refactor `GuidedFormItem` to move the guide trigger from input focus to an icon in the label.
- Update `FieldGuideCard` to be visually compatible with a Popover and adopt the project's indigo/slate theme.
- Use Ant Design `Popover` for the guidance display.

**Tech Stack:** React, Ant Design, TailwindCSS, `antd-style`.

---

### Task 1: Refactor FieldGuideCard Styling

**Files:**
- Modify: `web/src/components/FormGuidance/FieldGuideCard.tsx`

- [ ] **Step 1: Update FieldGuideCard to use Indigo/Slate theme**

```tsx
import React from 'react';
import type { FieldGuide } from './types';

type FieldGuideRowProps = {
  label: string;
  value: string;
};

const FieldGuideRow: React.FC<FieldGuideRowProps> = ({ label, value }) => (
  <div className="space-y-1">
    <div className="text-[11px] font-bold uppercase tracking-wider text-indigo-600">{label}</div>
    <div className="text-sm leading-relaxed text-slate-600">{value}</div>
  </div>
);

type FieldGuideCardProps = {
  guide: FieldGuide;
};

const FieldGuideCard: React.FC<FieldGuideCardProps> = ({ guide }) => {
  const rows = [
    { label: '这里填什么', value: guide.whatToEnter },
    { label: '这个值是干嘛的', value: guide.purpose },
    { label: '推荐示例', value: guide.example },
    { label: '填错会怎样', value: guide.impact },
    { label: '什么时候必填', value: guide.whenRequired },
    { label: '格式要求', value: guide.formatNotes },
  ].filter((row): row is { label: string; value: string } => Boolean(row.value && row.value.trim()));

  return (
    <div
      data-testid="field-guide-card"
      className="space-y-4 py-1"
    >
      <div className="space-y-4">
        {rows.map((row) => (
          <FieldGuideRow key={row.label} label={row.label} value={row.value} />
        ))}
      </div>
    </div>
  );
};

export default FieldGuideCard;
```

- [ ] **Step 2: Commit styling changes**

```bash
git add web/src/components/FormGuidance/FieldGuideCard.tsx
git commit -m "style(web): modernize FieldGuideCard for popover usage"
```

### Task 2: Update GuidedFormItem to Manual Trigger

**Files:**
- Modify: `web/src/components/FormGuidance/GuidedFormItem.tsx`

- [ ] **Step 1: Update GuidedFormItem implementation**

```tsx
import React, { useState } from 'react';
import { Form, Tooltip, Popover } from 'antd';
import { InfoCircleOutlined } from '@ant-design/icons';
import SparklesIcon from '../common/SparklesIcon';
import FieldGuideCard from './FieldGuideCard';
import { useFormAssist } from '../../features/ai/hooks/useFormAssist';
import AIFormAssistantPopover from './AIFormAssistantPopover';
import type { GuidedFormItemProps, FocusableChildProps } from './types';

// ... AIFieldWrapper remains same ...

const GuidedFormItem: React.FC<GuidedFormItemProps> = ({ 
  guide, 
  aiAssist,
  extra, 
  children, 
  ...formItemProps 
}) => {
  const form = Form.useFormInstance();
  const name = formItemProps.name;
  const currentValue = Form.useWatch(name, form) || '';

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
  } = useFormAssist(aiAssist, currentValue, (val) => {
    if (name) {
      form.setFieldValue(name, val);
    }
  });

  const guideIcon = guide ? (
    <Popover
      content={<div className="w-72"><FieldGuideCard guide={guide} /></div>}
      title={<div className="flex items-center gap-2 border-b pb-2 mb-2"><span className="text-base">💡</span><span className="font-semibold text-slate-800">填写指南</span></div>}
      trigger="click"
      placement="topRight"
      overlayClassName="field-guide-popover"
    >
      <InfoCircleOutlined 
        className="ml-1.5 text-indigo-500 hover:text-indigo-600 cursor-pointer text-sm transition-colors" 
        onClick={(e) => e.stopPropagation()}
      />
    </Popover>
  ) : null;

  const enhancedLabel = formItemProps.label ? (
    <span className="flex items-center">
      {formItemProps.label}
      {guideIcon}
    </span>
  ) : null;

  // Render Logic
  // ... (similar to existing but without isFocused logic)
  return (
    <Form.Item 
      {...formItemProps} 
      label={enhancedLabel}
      extra={extra}
    >
      {/* AI trigger logic remains same but clean up focus handlers from child */}
      {/* ... implementation details ... */}
    </Form.Item>
  );
};
```

- [ ] **Step 2: Verify existing tests pass and update if necessary**

Run: `npm test web/src/components/FormGuidance/GuidedFormItem.test.tsx`

- [ ] **Step 3: Commit changes**

```bash
git add web/src/components/FormGuidance/GuidedFormItem.tsx
git commit -m "feat(web): change GuidedFormItem to manual exclamation trigger"
```

### Task 3: Final Verification

- [ ] **Step 1: Check UI consistency across multiple pages**
- [ ] **Step 2: Verify AI Assist still works correctly**
