import React, { useMemo } from 'react';
import { createStyles } from 'antd-style';
import { Modal, Collapse, Typography } from 'antd';
import type { AssistantReplyActivity } from './types';

const useToolResultCardStyles = createStyles(({ token, css }) => ({
  popover: css`
    max-width: 360px;
    max-height: 300px;
    overflow: auto;
  `,
  popoverHeader: css`
    display: flex;
    align-items: center;
    gap: 8px;
    padding-bottom: 8px;
    border-bottom: 1px solid ${token.colorBorderSecondary};
    margin-bottom: 8px;
  `,
  popoverTitle: css`
    font-size: 13px;
    font-weight: 500;
    color: ${token.colorText};
  `,
  popoverStatus: css`
    font-size: 11px;
    padding: 2px 6px;
    border-radius: 4px;
  `,
  popoverStatusSuccess: css`
    color: ${token.colorSuccess};
    background: ${token.colorSuccessBg};
  `,
  popoverStatusError: css`
    color: ${token.colorError};
    background: ${token.colorErrorBg};
  `,
  content: css`
    font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
    font-size: 12px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-all;
    color: ${token.colorText};
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
  argsContent: css`
    font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
    font-size: 11px;
    line-height: 1.4;
    white-space: pre-wrap;
    word-break: break-all;
    color: ${token.colorTextSecondary};
  `,
  modalContent: css`
    max-height: 60vh;
    overflow: auto;
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
  visible: boolean;
  onClose: () => void;
}

const MAX_CONTENT_SIZE = 10000;
const LINE_THRESHOLD = 20;

/**
 * 判断内容显示模式
 */
function getDisplayMode(content: string): 'popover' | 'modal' {
  if (!content) return 'popover';

  try {
    const parsed = JSON.parse(content);
    const formatted = JSON.stringify(parsed, null, 2);
    const lineCount = formatted.split('\n').length;
    return lineCount > LINE_THRESHOLD ? 'modal' : 'popover';
  } catch {
    // 非 JSON 内容
    const lineCount = content.split('\n').length;
    return lineCount > LINE_THRESHOLD ? 'modal' : 'popover';
  }
}

/**
 * 格式化内容
 */
function formatContent(content: string, maxSize = MAX_CONTENT_SIZE): { formatted: string; truncated: boolean } {
  if (!content) {
    return { formatted: '', truncated: false };
  }

  let formatted: string;
  let truncated = false;

  // 截断超长内容
  if (content.length > maxSize) {
    formatted = content.slice(0, maxSize);
    truncated = true;
  } else {
    formatted = content;
  }

  // 尝试格式化 JSON
  try {
    const parsed = JSON.parse(formatted);
    formatted = JSON.stringify(parsed, null, 2);
  } catch {
    // 非 JSON，保持原样
  }

  return { formatted, truncated };
}

export default function ToolResultCard({ activity, visible, onClose }: ToolResultCardProps) {
  const { styles, cx } = useToolResultCardStyles();

  const { label, status, arguments: args, rawContent } = activity;

  // 格式化内容
  const { formatted, truncated } = useMemo(() => formatContent(rawContent || ''), [rawContent]);

  // 判断显示模式
  const displayMode = useMemo(() => getDisplayMode(rawContent || ''), [rawContent]);

  // 格式化参数
  const formattedArgs = useMemo(() => {
    if (!args) return '';
    try {
      return JSON.stringify(args, null, 2);
    } catch {
      return String(args);
    }
  }, [args]);

  // 渲染内容
  const renderContent = () => (
    <>
      {args && Object.keys(args).length > 0 && (
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
      )}
      <pre className={styles.content}>{formatted}</pre>
      {truncated && (
        <div className={styles.truncated}>
          ... 内容已截断，完整内容过大
        </div>
      )}
    </>
  );

  // 渲染状态标签
  const renderStatusTag = () => {
    if (status === 'error') {
      return <span className={cx(styles.popoverStatus, styles.popoverStatusError)}>失败</span>;
    }
    return <span className={cx(styles.popoverStatus, styles.popoverStatusSuccess)}>成功</span>;
  };

  if (displayMode === 'popover') {
    return (
      <Modal
        open={visible}
        onCancel={onClose}
        footer={null}
        title={null}
        width={400}
        closable
        maskClosable
        centered
        styles={{
          body: { padding: '12px 16px' },
        }}
      >
        <div className={styles.popoverHeader}>
          <span className={styles.popoverTitle}>{label}</span>
          {renderStatusTag()}
        </div>
        <div className={styles.popover}>
          {renderContent()}
        </div>
      </Modal>
    );
  }

  // Modal 模式
  return (
    <Modal
      open={visible}
      onCancel={onClose}
      footer={null}
      title={
        <div className={styles.popoverHeader} style={{ borderBottom: 'none', marginBottom: 0, paddingBottom: 0 }}>
          <span className={styles.popoverTitle}>{label}</span>
          {renderStatusTag()}
        </div>
      }
      width={600}
      closable
      maskClosable
      centered
    >
      <div className={styles.modalContent}>
        {renderContent()}
      </div>
    </Modal>
  );
}
