import React, { useState } from 'react';
import { createStyles } from 'antd-style';
import type { AssistantReplyActivity } from './types';
import ToolResultCard from './ToolResultCard';

const useToolReferenceStyles = createStyles(({ token, css }) => ({
  root: css`
    display: inline-flex;
    align-items: center;
    vertical-align: baseline;
    white-space: nowrap;
    margin-left: 4px;
    padding: 1px 6px;
    border-radius: 5px;
    font-size: 12px;
    line-height: 18px;
    font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
    cursor: default;
    transition: all 0.2s ease;
    border: 1px solid transparent;
  `,
  loading: css`
    color: rgba(15, 23, 42, 0.42);
    background: ${token.colorFillQuaternary};
    border-color: ${token.colorBorderSecondary};

    .tool-reference-label {
      background-image: linear-gradient(
        90deg,
        rgba(15, 23, 42, 0.38) 0%,
        rgba(37, 99, 235, 0.92) 42%,
        rgba(15, 23, 42, 0.38) 100%
      );
      background-size: 220% 100%;
      background-position: 100% 50%;
      -webkit-background-clip: text;
      background-clip: text;
      color: transparent;
      animation: toolRefSweep 1.2s linear infinite;
    }

    @keyframes toolRefSweep {
      from {
        background-position: 100% 50%;
      }
      to {
        background-position: -100% 50%;
      }
    }
  `,
  success: css`
    color: ${token.colorPrimary};
    background: ${token.colorPrimaryBg};
    border-color: ${token.colorPrimaryBorder};
    cursor: pointer;

    &:hover {
      background: ${token.colorPrimaryBgHover};
    }
  `,
  error: css`
    color: ${token.colorError};
    background: ${token.colorErrorBg};
    border-color: ${token.colorErrorBorder};
    cursor: pointer;

    &:hover {
      background: ${token.colorErrorBgHover};
    }
  `,
  label: css`
    display: inline-block;
  `,
}));

interface ToolReferenceProps { activity: AssistantReplyActivity; }

export default function ToolReference({ activity }: ToolReferenceProps) {
  const { styles, cx } = useToolReferenceStyles();
  const [cardVisible, setCardVisible] = useState(false);

  const { status, label, detail } = activity;
  const normalizedDetail = detail || '';
  const isLoading = status === 'active';
  const isSuccess = status === 'done';
  const isError = status === 'error';
  const interruptedSuffix = normalizedDetail.includes('异常中断')
    ? '（异常中断）'
    : normalizedDetail.includes('未完成')
      ? '（未完成）'
      : '';
  const accessibleLabel = `${label}${interruptedSuffix}`;

  const stateClass = cx(
    styles.root,
    isLoading && styles.loading,
    isSuccess && styles.success,
    isError && styles.error,
  );

  const isClickable = isSuccess || isError;

  const handleClick = () => {
    if (isClickable) {
      setCardVisible(true);
    }
  };

  return (
    <>
      <span
        className={stateClass}
        onClick={handleClick}
        role={isClickable ? 'button' : undefined}
        tabIndex={isClickable ? 0 : undefined}
        aria-label={isClickable ? accessibleLabel : undefined}
        onKeyDown={(e) => {
          if (isClickable && (e.key === 'Enter' || e.key === ' ')) {
            setCardVisible(true);
          }
        }}
      >
        <span className={`tool-reference-label ${styles.label}`}>{label}</span>
        {interruptedSuffix ? <span aria-hidden="true">{interruptedSuffix}</span> : null}
      </span>
      {cardVisible && (
        <ToolResultCard
          activity={activity}
          visible={cardVisible}
          onClose={() => setCardVisible(false)}
        />
      )}
    </>
  );
}
