import React, { useState, useMemo } from 'react';
import { createStyles } from 'antd-style';
import type { AssistantReplyActivity } from './types';
import ToolResultCard from './ToolResultCard';

const useToolReferenceStyles = createStyles(({ token, css }) => ({
  root: css`
    display: inline-flex;
    align-items: center;
    gap: 4px;
    margin-left: 4px;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 12px;
    font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
    cursor: default;
    transition: all 0.2s ease;
  `,
  loading: css`
    color: ${token.colorTextSecondary};
    background: ${token.colorFillQuaternary};
  `,
  success: css`
    color: ${token.colorPrimary};
    background: ${token.colorPrimaryBg};
    cursor: pointer;

    &:hover {
      background: ${token.colorPrimaryBgHover};
    }
  `,
  error: css`
    color: ${token.colorError};
    background: ${token.colorErrorBg};
    cursor: pointer;

    &:hover {
      background: ${token.colorErrorBgHover};
    }
  `,
  icon: css`
    font-size: 10px;
  `,
  spinner: css`
    display: inline-block;
    animation: spin 1s linear infinite;

    @keyframes spin {
      from {
        transform: rotate(0deg);
      }
      to {
        transform: rotate(360deg);
      }
    }
  `,
}));

interface ToolReferenceProps {
  activity: AssistantReplyActivity;
}

export default function ToolReference({ activity }: ToolReferenceProps) {
  const { styles, cx } = useToolReferenceStyles();
  const [cardVisible, setCardVisible] = useState(false);

  const { kind, status, label } = activity;

  // 判断状态
  const isLoading = kind === 'tool_call' && status === 'active';
  const isSuccess = kind === 'tool_result' && status === 'done';
  const isError = kind === 'tool_result' && status === 'error';

  // 获取样式类名
  const stateClass = cx(
    styles.root,
    isLoading && styles.loading,
    isSuccess && styles.success,
    isError && styles.error,
  );

  // 获取图标
  const renderIcon = () => {
    if (isLoading) {
      return <span className={cx(styles.icon, styles.spinner)}>◐</span>;
    }
    if (isError) {
      return <span className={styles.icon}>✗</span>;
    }
    return <span className={styles.icon}>→</span>;
  };

  // 是否可点击
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
        onKeyDown={(e) => {
          if (isClickable && (e.key === 'Enter' || e.key === ' ')) {
            setCardVisible(true);
          }
        }}
      >
        {renderIcon()}
        <span>{label}</span>
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
