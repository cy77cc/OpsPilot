import React, { useState } from 'react';
import { Button } from 'antd';
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
  approvalActions: css`
    display: inline-flex;
    align-items: center;
    gap: 6px;
    margin-left: 8px;
    vertical-align: middle;
  `,
  approvalMessage: css`
    display: inline-block;
    margin-left: 8px;
    font-size: 12px;
    line-height: 18px;
    color: ${token.colorTextDescription};
  `,
  approvalSuccess: css`
    color: ${token.colorSuccess};
  `,
  approvalError: css`
    color: ${token.colorError};
  `,
  label: css`
    display: inline-block;
  `,
}));

interface ToolReferenceProps {
  activity: AssistantReplyActivity;
  onApprovalDecision?: (activity: AssistantReplyActivity, approved: boolean) => Promise<void> | void;
}

export default function ToolReference({ activity, onApprovalDecision }: ToolReferenceProps) {
  const { styles, cx } = useToolReferenceStyles();
  const [cardVisible, setCardVisible] = useState(false);

  const { status, label, detail, approvalState, approvalMessage } = activity;
  const normalizedDetail = detail || '';
  const isLoading = status === 'active';
  const isSuccess = status === 'done';
  const isError = status === 'error';
  const isApproval = activity.kind === 'tool_approval';
  const isSubmittingApproval = approvalState === 'submitting';
  const isWaitingApproval = isApproval && (approvalState === 'waiting-approval' || approvalState === undefined || approvalState === null);
  const isRefreshNeeded = approvalState === 'refresh-needed';
  const isApprovalResolved = approvalState === 'approved' || approvalState === 'rejected' || approvalState === 'approved_done' || approvalState === 'approved_failed_terminal' || approvalState === 'expired';
  const interruptedSuffix = normalizedDetail.includes('异常中断')
    ? '（异常中断）'
    : normalizedDetail.includes('未完成')
      ? '（未完成）'
      : '';
  const approvalSuffix = approvalState === 'approved'
    ? '（已批准）'
    : approvalState === 'rejected'
      ? '（已拒绝）'
      : approvalState === 'approved_resuming'
        ? '（结果确认中）'
        : approvalState === 'approved_retrying'
          ? '（重试中）'
          : approvalState === 'approved_done'
            ? '（已完成）'
            : approvalState === 'approved_failed_terminal'
              ? '（恢复失败）'
              : approvalState === 'expired'
                ? '（已过期）'
                : isSubmittingApproval
                  ? '（提交中）'
                  : isWaitingApproval
                    ? '（待审批）'
                    : isRefreshNeeded
                      ? '（需刷新）'
                      : '';
  const accessibleLabel = `${label}${interruptedSuffix}`;
  const approvalMessageClass = approvalState === 'approved' || approvalState === 'approved_done'
    ? styles.approvalSuccess
    : approvalState === 'rejected' || approvalState === 'approved_failed_terminal' || approvalState === 'expired'
      ? styles.approvalError
      : undefined;

  const stateClass = cx(
    styles.root,
    isLoading && styles.loading,
    isSuccess && styles.success,
    isError && styles.error,
  );

  const isClickable = isApproval ? (isApprovalResolved || isError) : (isSuccess || isError);

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
        {approvalSuffix ? <span aria-hidden="true">{approvalSuffix}</span> : null}
      </span>
      {isApproval && isWaitingApproval && (
        <span className={styles.approvalActions}>
          <Button
            size="small"
            type="primary"
            loading={isSubmittingApproval}
            disabled={isSubmittingApproval}
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void onApprovalDecision?.(activity, true);
            }}
          >
            批准
          </Button>
          <Button
            size="small"
            danger
            disabled={isSubmittingApproval}
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void onApprovalDecision?.(activity, false);
            }}
          >
            拒绝
          </Button>
        </span>
      )}
      {isApproval && approvalMessage ? (
        <span className={cx(styles.approvalMessage, approvalMessageClass)}>{approvalMessage}</span>
      ) : null}
      {isApproval && isApprovalResolved ? (
        <span
          className={cx(
            styles.approvalMessage,
            approvalState === 'approved' || approvalState === 'approved_done' ? styles.approvalSuccess : styles.approvalError,
          )}
        >
          {approvalState === 'approved' || approvalState === 'approved_done' ? '已批准' : approvalState === 'expired' ? '已过期' : '已拒绝'}
        </span>
      ) : null}
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
