import React, { useMemo, useState } from 'react';
import { createStyles } from 'antd-style';
import { Button, Collapse, Modal } from 'antd';
import type { AssistantReplyActivity } from './types';

const useToolResultCardStyles = createStyles(({ token, css }) => ({
  panel: css`
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 12px;
    border-radius: 12px;
    border: 1px solid ${token.colorBorderSecondary};
    background: ${token.colorBgContainer};
  `,
  header: css`
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
  `,
  titleBlock: css`
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  `,
  title: css`
    font-size: 13px;
    font-weight: 600;
    color: ${token.colorText};
    word-break: break-word;
  `,
  detail: css`
    font-size: 12px;
    line-height: 18px;
    color: ${token.colorTextDescription};
  `,
  statusBadge: css`
    display: inline-flex;
    align-items: center;
    padding: 2px 8px;
    border-radius: 999px;
    font-size: 12px;
    line-height: 18px;
    white-space: nowrap;
    border: 1px solid transparent;
  `,
  statusPending: css`
    color: ${token.colorPrimary};
    background: ${token.colorPrimaryBg};
    border-color: ${token.colorPrimaryBorder};
  `,
  statusSuccess: css`
    color: ${token.colorSuccess};
    background: ${token.colorSuccessBg};
    border-color: ${token.colorSuccessBorder};
  `,
  statusWarning: css`
    color: ${token.colorWarning};
    background: ${token.colorWarningBg};
    border-color: ${token.colorWarningBorder};
  `,
  statusDanger: css`
    color: ${token.colorError};
    background: ${token.colorErrorBg};
    border-color: ${token.colorErrorBorder};
  `,
  summarySection: css`
    display: flex;
    flex-direction: column;
    gap: 8px;
  `,
  sectionTitle: css`
    font-size: 12px;
    line-height: 18px;
    color: ${token.colorTextDescription};
    text-transform: uppercase;
    letter-spacing: 0.04em;
  `,
  summaryGrid: css`
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
    gap: 8px 12px;
  `,
  summaryItem: css`
    display: flex;
    flex-direction: column;
    gap: 2px;
  `,
  summaryLabel: css`
    font-size: 12px;
    line-height: 18px;
    color: ${token.colorTextDescription};
  `,
  summaryValue: css`
    font-size: 14px;
    line-height: 22px;
    color: ${token.colorText};
    font-weight: 600;
    word-break: break-word;
  `,
  fallback: css`
    font-size: 13px;
    line-height: 20px;
    color: ${token.colorTextDescription};
  `,
  rawToggle: css`
    align-self: flex-start;
    padding: 0;
    height: auto;
  `,
  rawContent: css`
    margin: 0;
    padding: 10px 12px;
    border-radius: 10px;
    background: ${token.colorFillTertiary};
    color: ${token.colorTextSecondary};
    font-size: 12px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-word;
    overflow: auto;
  `,
  actionRow: css`
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  `,
  modalContent: css`
    max-height: 60vh;
    overflow: auto;
  `,
  argsCollapse: css`
    margin-bottom: 8px;

    .ant-collapse-header {
      font-size: 12px !important;
      padding: 4px 8px !important;
    }

    .ant-collapse-content-box {
      padding: 8px !important;
    }
  `,
  monoContent: css`
    font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
    font-size: 12px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-all;
    color: ${token.colorText};
  `,
  argsContent: css`
    font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
    font-size: 11px;
    line-height: 1.4;
    white-space: pre-wrap;
    word-break: break-all;
    color: ${token.colorTextSecondary};
  `,
  truncated: css`
    color: ${token.colorTextTertiary};
    font-style: italic;
    margin-top: 8px;
    font-size: 11px;
  `,
}));

interface ToolResultCardProps {
  activity: AssistantReplyActivity;
  visible?: boolean;
  onClose?: () => void;
  inline?: boolean;
  children?: React.ReactNode;
}

const MAX_CONTENT_SIZE = 10000;
const LINE_THRESHOLD = 20;
const NO_STRUCTURED_PREVIEW_TEXT = 'No structured preview available';

function formatContent(content: string, maxSize = MAX_CONTENT_SIZE): { formatted: string; truncated: boolean } {
  if (!content) {
    return { formatted: '', truncated: false };
  }

  let formatted = content;
  let truncated = false;

  if (formatted.length > maxSize) {
    formatted = formatted.slice(0, maxSize);
    truncated = true;
  }

  try {
    const parsed = JSON.parse(formatted);
    formatted = JSON.stringify(parsed, null, 2);
  } catch {
    // Keep non-JSON payloads untouched.
  }

  return { formatted, truncated };
}

function getDisplayMode(content: string): 'popover' | 'modal' {
  if (!content) {
    return 'popover';
  }

  try {
    const parsed = JSON.parse(content);
    const formatted = JSON.stringify(parsed, null, 2);
    return formatted.split('\n').length > LINE_THRESHOLD ? 'modal' : 'popover';
  } catch {
    return content.split('\n').length > LINE_THRESHOLD ? 'modal' : 'popover';
  }
}

function formatApprovalPreview(preview?: Record<string, unknown>): string {
  if (!preview) {
    return '';
  }
  try {
    return JSON.stringify(preview, null, 2);
  } catch {
    return String(preview);
  }
}

function getApprovalStatusMeta(activity: AssistantReplyActivity): {
  label: string;
  className: 'pending' | 'success' | 'warning' | 'danger';
} {
  switch (activity.approvalState) {
    case 'submitting':
      return { label: '提交中', className: 'warning' };
    case 'approved':
    case 'approved_resuming':
    case 'approved_retrying':
    case 'approved_failed_terminal':
    case 'approved_done':
      return { label: '已批准', className: 'success' };
    case 'rejected':
      return { label: '已拒绝', className: 'danger' };
    case 'expired':
      return { label: '审批已过期', className: 'warning' };
    case 'refresh-needed':
      return { label: '需刷新', className: 'danger' };
    case 'waiting-approval':
    default:
      return { label: '等待审批', className: 'pending' };
  }
}

function renderStatusClass(
  styles: Record<string, string>,
  className: 'pending' | 'success' | 'warning' | 'danger',
): string {
  if (className === 'success') {
    return `${styles.statusBadge} ${styles.statusSuccess}`;
  }
  if (className === 'warning') {
    return `${styles.statusBadge} ${styles.statusWarning}`;
  }
  if (className === 'danger') {
    return `${styles.statusBadge} ${styles.statusDanger}`;
  }
  return `${styles.statusBadge} ${styles.statusPending}`;
}

function ToolResultModalContent({
  activity,
  styles,
}: {
  activity: AssistantReplyActivity;
  styles: Record<string, string>;
}) {
  const { label, status, arguments: args, rawContent } = activity;
  const { formatted, truncated } = useMemo(() => formatContent(rawContent || ''), [rawContent]);
  const formattedArgs = useMemo(() => {
    if (!args) {
      return '';
    }
    try {
      return JSON.stringify(args, null, 2);
    } catch {
      return String(args);
    }
  }, [args]);

  const statusLabel = status === 'error' ? '失败' : '成功';
  const statusClass = status === 'error'
    ? `${styles.statusBadge} ${styles.statusDanger}`
    : `${styles.statusBadge} ${styles.statusSuccess}`;

  return (
    <>
      <div className={styles.header}>
        <div className={styles.titleBlock}>
          <span className={styles.title}>{label}</span>
        </div>
        <span className={statusClass}>{statusLabel}</span>
      </div>
      <div className={styles.modalContent}>
        {args && Object.keys(args).length > 0 ? (
          <Collapse
            className={styles.argsCollapse}
            ghost
            items={[
              {
                key: 'args',
                label: '调用参数',
                children: <pre className={styles.argsContent}>{formattedArgs}</pre>,
              },
            ]}
          />
        ) : null}
        <pre className={styles.monoContent}>{formatted}</pre>
        {truncated ? <div className={styles.truncated}>... 内容已截断，完整内容过大</div> : null}
      </div>
    </>
  );
}

export default function ToolResultCard({
  activity,
  visible = true,
  onClose,
  inline = false,
  children,
}: ToolResultCardProps) {
  const { styles } = useToolResultCardStyles();
  const [showRawPayload, setShowRawPayload] = useState(false);

  if (activity.kind === 'tool_approval' || inline) {
    const { label, detail, approvalPreview, approvalPreviewSummary, approvalMessage } = activity;
    const statusMeta = getApprovalStatusMeta(activity);
    const rawPayload = formatApprovalPreview(approvalPreview);

    return (
      <div className={styles.panel} data-testid="approval-tool-card">
        <div className={styles.header}>
          <div className={styles.titleBlock}>
            <span className={styles.title}>{label}</span>
            {detail ? <span className={styles.detail}>{detail}</span> : null}
            {approvalMessage ? <span className={styles.detail}>{approvalMessage}</span> : null}
          </div>
          <span className={renderStatusClass(styles, statusMeta.className)}>{statusMeta.label}</span>
        </div>

        <div className={styles.summarySection}>
          <div className={styles.sectionTitle}>Operation To Approve</div>
          {approvalPreviewSummary && approvalPreviewSummary.length > 0 ? (
            <div className={styles.summaryGrid}>
              {approvalPreviewSummary.map((row) => (
                <div key={`${row.key}:${row.value}`} className={styles.summaryItem}>
                  <span className={styles.summaryLabel}>{row.label}</span>
                  <span className={styles.summaryValue}>{row.value}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className={styles.fallback}>{NO_STRUCTURED_PREVIEW_TEXT}</div>
          )}
        </div>

        {approvalPreview ? (
          <div className={styles.summarySection}>
            <Button
              type="link"
              className={styles.rawToggle}
              onClick={() => setShowRawPayload((current) => !current)}
            >
              {showRawPayload ? '隐藏原始审批载荷' : '查看原始审批载荷'}
            </Button>
            {showRawPayload ? <pre className={styles.rawContent}>{rawPayload}</pre> : null}
          </div>
        ) : null}

        {children ? <div className={styles.actionRow}>{children}</div> : null}
      </div>
    );
  }

  if (!visible) {
    return null;
  }

  const displayMode = getDisplayMode(activity.rawContent || '');
  const modalWidth = displayMode === 'modal' ? 600 : 400;

  return (
    <Modal
      open={visible}
      onCancel={onClose}
      footer={null}
      title={null}
      width={modalWidth}
      closable
      maskClosable
      centered
      styles={{
        body: { padding: '12px 16px' },
      }}
    >
      <ToolResultModalContent activity={activity} styles={styles} />
    </Modal>
  );
}
