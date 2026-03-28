import React, { useEffect, useMemo, useRef, useState } from 'react';
import { createStyles } from 'antd-style';
import { Button } from 'antd';
import { aiApi, isApprovalConflictError } from '../../api/modules/ai';
import type { AssistantReplyActivity, AssistantReplyApprovalState } from './types';
import ToolResultCard from './ToolResultCard';

const APPROVAL_REFRESH_NEEDED_MESSAGE = '审批状态可能已变更，需刷新后查看结果';

const useToolReferenceStyles = createStyles(({ token, css }) => ({
  wrapper: css`
    display: inline-flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
    vertical-align: top;
    max-width: min(100%, 560px);
  `,
  root: css`
    display: inline-flex;
    align-items: center;
    gap: 6px;
    vertical-align: baseline;
    white-space: nowrap;
    padding: 1px 6px;
    border-radius: 5px;
    font-size: 12px;
    line-height: 18px;
    font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
    cursor: default;
    transition: all 0.2s ease;
    border: 1px solid transparent;
    background: transparent;
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
  approval: css`
    color: ${token.colorPrimary};
    background: ${token.colorPrimaryBg};
    border-color: ${token.colorPrimaryBorder};
    cursor: pointer;
    padding: 4px 8px;

    &:hover {
      background: ${token.colorPrimaryBgHover};
    }
  `,
  approvalDanger: css`
    color: ${token.colorError};
    background: ${token.colorErrorBg};
    border-color: ${token.colorErrorBorder};

    &:hover {
      background: ${token.colorErrorBgHover};
    }
  `,
  approvalSuccess: css`
    color: ${token.colorSuccess};
    background: ${token.colorSuccessBg};
    border-color: ${token.colorSuccessBorder};

    &:hover {
      background: ${token.colorSuccessBgHover};
    }
  `,
  label: css`
    display: inline-block;
  `,
  state: css`
    font-size: 11px;
    line-height: 16px;
    opacity: 0.9;
  `,
  approvalActions: css`
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  `,
}));

type LocalApprovalUiState =
  | 'waiting-approval'
  | 'submitting'
  | 'approved'
  | 'rejected'
  | 'refresh-needed'
  | 'expired';

interface ToolReferenceProps {
  activity: AssistantReplyActivity;
}

function createIdempotencyKey(approvalId: string, approved: boolean): string {
  const decision = approved ? 'approve' : 'reject';
  if (typeof globalThis !== 'undefined' && globalThis.crypto && typeof globalThis.crypto.randomUUID === 'function') {
    return `${approvalId}:${decision}:${globalThis.crypto.randomUUID()}`;
  }
  return `${approvalId}:${decision}:${Date.now()}:${Math.random().toString(16).slice(2)}`;
}

function normalizeApprovalState(activity: AssistantReplyActivity): LocalApprovalUiState {
  switch (activity.approvalState) {
    case 'submitting':
      return 'submitting';
    case 'approved':
    case 'approved_resuming':
    case 'approved_retrying':
    case 'approved_failed_terminal':
    case 'approved_done':
      return 'approved';
    case 'rejected':
      return 'rejected';
    case 'expired':
      return 'expired';
    case 'refresh-needed':
      return 'refresh-needed';
    case 'waiting-approval':
    default:
      return 'waiting-approval';
  }
}

function getApprovalStateLabel(state: LocalApprovalUiState): string {
  switch (state) {
    case 'submitting':
      return '提交中';
    case 'approved':
      return '已批准';
    case 'rejected':
      return '已拒绝';
    case 'expired':
      return '审批已过期';
    case 'refresh-needed':
      return '需刷新';
    case 'waiting-approval':
    default:
      return '等待审批';
  }
}

function mapServerStatusToApprovalState(status?: string): LocalApprovalUiState {
  switch ((status || '').toLowerCase()) {
    case 'approved':
      return 'approved';
    case 'rejected':
      return 'rejected';
    case 'expired':
      return 'expired';
    case 'pending':
    default:
      return 'waiting-approval';
  }
}

function extractErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  if (typeof error === 'object' && error && 'message' in error) {
    return String((error as { message?: unknown }).message || '未知错误');
  }
  return '未知错误';
}

function shouldDefaultApprovalOpen(state: LocalApprovalUiState): boolean {
  return state === 'waiting-approval' || state === 'submitting' || state === 'refresh-needed';
}

function buildApprovalActivity(
  activity: AssistantReplyActivity,
  approvalState: LocalApprovalUiState,
  approvalMessage?: string,
): AssistantReplyActivity {
  const mappedState: AssistantReplyApprovalState = approvalState;
  return {
    ...activity,
    approvalState: mappedState,
    approvalMessage,
  };
}

export default function ToolReference({ activity }: ToolReferenceProps) {
  const { styles, cx } = useToolReferenceStyles();
  const [cardVisible, setCardVisible] = useState(false);
  const [approvalState, setApprovalState] = useState<LocalApprovalUiState>(() => normalizeApprovalState(activity));
  const [approvalMessage, setApprovalMessage] = useState<string | undefined>(activity.approvalMessage);
  const [approvalPanelOpen, setApprovalPanelOpen] = useState<boolean>(() => shouldDefaultApprovalOpen(normalizeApprovalState(activity)));
  const submitLockRef = useRef(false);

  useEffect(() => {
    if (submitLockRef.current) {
      return;
    }
    const nextState = normalizeApprovalState(activity);
    setApprovalState(nextState);
    setApprovalMessage(activity.approvalMessage);
    if (shouldDefaultApprovalOpen(nextState)) {
      setApprovalPanelOpen(true);
    }
  }, [activity.approvalId, activity.approvalMessage, activity.approvalState, activity.status]);

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
  const isApproval = activity.kind === 'tool_approval';

  const approvalChipClass = useMemo(() => {
    if (approvalState === 'approved') {
      return cx(styles.root, styles.approval, styles.approvalSuccess);
    }
    if (approvalState === 'rejected' || approvalState === 'refresh-needed' || approvalState === 'expired') {
      return cx(styles.root, styles.approval, styles.approvalDanger);
    }
    return cx(styles.root, styles.approval);
  }, [approvalState, cx, styles.approval, styles.approvalDanger, styles.approvalSuccess, styles.root]);

  const stateClass = cx(
    styles.root,
    isLoading && styles.loading,
    isSuccess && styles.success,
    isError && styles.error,
  );

  const isClickable = isSuccess || isError;

  const handleNonApprovalClick = () => {
    if (isClickable) {
      setCardVisible(true);
    }
  };

  const broadcastApprovalUpdate = (approvalId: string, nextState: LocalApprovalUiState) => {
    if (nextState !== 'approved' && nextState !== 'rejected' && nextState !== 'expired') {
      return;
    }
    window.dispatchEvent(new CustomEvent('ai-approval-updated', {
      detail: {
        token: approvalId,
        status: nextState,
      },
    }));
  };

  const handleApprovalSubmit = async (approved: boolean) => {
    const approvalId = activity.approvalId;
    if (!approvalId || submitLockRef.current || approvalState !== 'waiting-approval') {
      return;
    }

    submitLockRef.current = true;
    setApprovalState('submitting');
    setApprovalMessage(undefined);
    setApprovalPanelOpen(true);

    try {
      const response = await aiApi.submitApproval(
        approvalId,
        { approved },
        { idempotencyKey: createIdempotencyKey(approvalId, approved) },
      );
      const nextState = mapServerStatusToApprovalState(response.data?.status || (approved ? 'approved' : 'rejected'));
      setApprovalState(nextState);
      setApprovalMessage(response.data?.message);
      broadcastApprovalUpdate(approvalId, nextState);
    } catch (error) {
      if (isApprovalConflictError(error)) {
        try {
          const refreshed = await aiApi.getApproval(approvalId);
          const nextState = mapServerStatusToApprovalState(refreshed.data?.status);
          setApprovalState(nextState);
          setApprovalMessage(undefined);
          broadcastApprovalUpdate(approvalId, nextState);
        } catch {
          setApprovalState('refresh-needed');
          setApprovalMessage(APPROVAL_REFRESH_NEEDED_MESSAGE);
        }
      } else {
        setApprovalState('refresh-needed');
        setApprovalMessage(`提交失败：${extractErrorMessage(error)}`);
      }
    } finally {
      submitLockRef.current = false;
    }
  };

  if (!isApproval) {
    return (
      <>
        <span
          className={stateClass}
          onClick={handleNonApprovalClick}
          role={isClickable ? 'button' : undefined}
          tabIndex={isClickable ? 0 : undefined}
          aria-label={isClickable ? accessibleLabel : undefined}
          onKeyDown={(event) => {
            if (isClickable && (event.key === 'Enter' || event.key === ' ')) {
              setCardVisible(true);
            }
          }}
        >
          <span className={`tool-reference-label ${styles.label}`}>{label}</span>
          {interruptedSuffix ? <span aria-hidden="true">{interruptedSuffix}</span> : null}
        </span>
        {cardVisible ? (
          <ToolResultCard
            activity={activity}
            visible={cardVisible}
            onClose={() => setCardVisible(false)}
          />
        ) : null}
      </>
    );
  }

  const displayActivity = buildApprovalActivity(activity, approvalState, approvalMessage);
  const allowApprovalActions = approvalState === 'waiting-approval' || approvalState === 'submitting';

  return (
    <div className={styles.wrapper}>
      <button
        type="button"
        className={approvalChipClass}
        onClick={() => setApprovalPanelOpen((current) => !current)}
      >
        <span className={`tool-reference-label ${styles.label}`}>{label}</span>
        <span className={styles.state}>{getApprovalStateLabel(approvalState)}</span>
      </button>

      {approvalPanelOpen ? (
        <ToolResultCard activity={displayActivity} inline>
          {allowApprovalActions ? (
            <div className={styles.approvalActions}>
              <Button
                type="primary"
                onClick={() => void handleApprovalSubmit(true)}
                disabled={approvalState === 'submitting'}
                loading={approvalState === 'submitting'}
              >
                批准
              </Button>
              <Button
                onClick={() => void handleApprovalSubmit(false)}
                disabled={approvalState === 'submitting'}
              >
                拒绝
              </Button>
            </div>
          ) : null}
        </ToolResultCard>
      ) : null}
    </div>
  );
}
