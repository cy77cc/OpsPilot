import React from 'react';
import XMarkdown from '@ant-design/x-markdown';
import { createStyles } from 'antd-style';
import type { AssistantReplyRuntime } from './types';

const useAssistantReplyStyles = createStyles(({ token, css }) => ({
  root: css`
    display: flex;
    flex-direction: column;
    gap: 12px;
    width: 100%;
  `,
  phase: css`
    font-size: 12px;
    line-height: 18px;
    color: ${token.colorTextDescription};
    letter-spacing: 0.02em;
  `,
  activities: css`
    display: flex;
    flex-direction: column;
    gap: 6px;
  `,
  activity: css`
    display: flex;
    justify-content: space-between;
    gap: 12px;
    padding-bottom: 6px;
    border-bottom: 1px solid ${token.colorBorderSecondary};
    font-size: 13px;
    line-height: 20px;
    color: ${token.colorTextSecondary};
  `,
  activityDetail: css`
    color: ${token.colorTextDescription};
  `,
  summary: css`
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 10px 12px;
    background: ${token.colorFillQuaternary};
    border-radius: 12px;
  `,
  summaryTitle: css`
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
  `,
  markdown: css`
    width: 100%;
    max-width: 100%;
    line-height: 1.65;
    word-break: break-word;

    pre {
      overflow-x: auto;
      padding: 12px;
      border-radius: 10px;
      background: #111827;
      color: #f9fafb;
    }

    table {
      width: 100%;
      border-collapse: collapse;
    }

    th,
    td {
      border: 1px solid ${token.colorBorderSecondary};
      padding: 8px 10px;
      text-align: left;
    }
  `,
  footer: css`
    font-size: 12px;
    line-height: 18px;
    color: ${token.colorTextDescription};
  `,
}));

interface AssistantReplyProps {
  content: string;
  runtime?: AssistantReplyRuntime;
  status?: string;
}

export function AssistantReply({ content, runtime, status }: AssistantReplyProps) {
  const { styles } = useAssistantReplyStyles();

  return (
    <div className={styles.root}>
      {runtime?.phaseLabel ? <div className={styles.phase}>{runtime.phaseLabel}</div> : null}

      {runtime?.activities?.length ? (
        <div className={styles.activities}>
          {runtime.activities.map((activity) => (
            <div key={activity.id} className={styles.activity}>
              <span>{activity.label}</span>
              {activity.detail ? <span className={styles.activityDetail}>{activity.detail}</span> : null}
            </div>
          ))}
        </div>
      ) : null}

      {runtime?.summary ? (
        <div className={styles.summary}>
          {runtime.summary.title ? <div className={styles.summaryTitle}>{runtime.summary.title}</div> : null}
          {runtime.summary.items?.length ? (
            <div className={styles.summaryGrid}>
              {runtime.summary.items.map((item) => (
                <div key={`${item.label}:${item.value}`} className={styles.summaryItem}>
                  <span className={styles.summaryLabel}>{item.label}</span>
                  <span className={styles.summaryValue}>{item.value}</span>
                </div>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}

      <div className={styles.markdown}>
        <XMarkdown
          content={content}
          streaming={{
            hasNextChunk: status === 'loading' || status === 'updating',
            enableAnimation: true,
            animationConfig: {
              fadeDuration: 180,
              easing: 'ease-out',
            },
          }}
        />
      </div>

      {runtime?.status ? <div className={styles.footer}>{runtime.status.label}</div> : null}
    </div>
  );
}
