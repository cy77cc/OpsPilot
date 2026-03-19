import React, { useState } from 'react';
import XMarkdown from '@ant-design/x-markdown';
import { createStyles } from 'antd-style';
import { Collapse, Button, Skeleton } from 'antd';
import { DownOutlined } from '@ant-design/icons';
import type { AssistantReplyActivity, AssistantReplyRuntime } from './types';

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
  planSteps: css`
    display: flex;
    flex-direction: column;
    gap: 10px;
  `,
  activeStep: css`
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px;
    background: ${token.colorFillQuaternary};
    border-radius: 8px;
    border: 1px solid ${token.colorBorderSecondary};
  `,
  activeStepHeader: css`
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    line-height: 22px;
    color: ${token.colorText};
    font-weight: 500;
  `,
  activeStepBody: css`
    display: flex;
    flex-direction: column;
    gap: 6px;
  `,
  stepMarkdown: css`
    width: 100%;
    max-width: 100%;
    line-height: 1.65;
    word-break: break-word;
  `,
  completedStepsCollapse: css`
    .ant-collapse-header {
      font-size: 12px;
      color: ${token.colorTextSecondary};
      padding: 4px 0 !important;
    }
    .ant-collapse-content-box {
      padding: 8px 0 !important;
    }
  `,
  completedStepItem: css`
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 8px 0;
    border-bottom: 1px solid ${token.colorBorderSecondary};
    &:last-child {
      border-bottom: none;
    }
  `,
  completedStepTitle: css`
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    color: ${token.colorTextSecondary};
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
  expandButton: css`
    padding: 0;
    height: auto;
    font-size: 13px;
    color: ${token.colorPrimary};
    margin-top: 4px;

    &:hover {
      background: transparent;
    }
  `,
  loadingContainer: css`
    padding: 12px;
    background: ${token.colorFillQuaternary};
    border-radius: 8px;
  `,
}));

interface AssistantReplyProps {
  content: string;
  runtime?: AssistantReplyRuntime;
  status?: string;
  messageId?: string;
  hasRuntime?: boolean;
  onLoadRuntime?: (messageId: string) => Promise<AssistantReplyRuntime | null>;
}

// SimpleMarkdownContent 只渲染 markdown 内容，没有 runtime
function SimpleMarkdownContent({ content, styles }: { content: string; styles: Record<string, string> }) {
  if (!content) return null;
  return (
    <div className={styles.markdown}>
      <XMarkdown content={content} />
    </div>
  );
}

// AssistantReplyContent 渲染完整的 runtime 内容
function AssistantReplyContent({
  content,
  runtime,
  status,
  styles,
}: {
  content: string;
  runtime: AssistantReplyRuntime;
  status?: string;
  styles: Record<string, string>;
}) {
  const activeStepIndex = runtime?.plan?.activeStepIndex;
  const allSteps = runtime?.plan?.steps || [];

  // 当前执行的步骤
  const activeStep = activeStepIndex !== undefined && activeStepIndex >= 0 && activeStepIndex < allSteps.length
    ? allSteps[activeStepIndex]
    : null;

  // 已完成的步骤（历史对话中所有步骤都是 done）
  const completedSteps = activeStepIndex !== undefined && activeStepIndex >= 0
    ? allSteps.slice(0, activeStepIndex).filter((step) => step.status === 'done')
    : allSteps.filter((step) => step.status === 'done');

  // 当前步骤的 activities
  const activeStepActivities = activeStepIndex !== undefined && activeStepIndex >= 0
    ? runtime?.activities?.filter((activity) => activity.stepIndex === activeStepIndex) || []
    : [];

  const isStreaming = status === 'loading' || status === 'updating';

  return (
    <>
      {runtime?.phaseLabel ? <div className={styles.phase}>{runtime.phaseLabel}</div> : null}

      {/* 已完成的步骤（折叠，可展开） */}
      {completedSteps.length > 0 ? (
        <Collapse
          className={styles.completedStepsCollapse}
          ghost
          items={[{
            key: 'completed',
            label: `已完成 ${completedSteps.length} 个步骤`,
            children: completedSteps.map((step) => (
              <div key={step.id} className={styles.completedStepItem}>
                <div className={styles.completedStepTitle}>
                  <span>✓</span>
                  <span>{step.title}</span>
                </div>
                {step.content ? (
                  <div className={styles.stepMarkdown}>
                    <XMarkdown content={step.content} />
                  </div>
                ) : null}
              </div>
            )),
          }]}
        />
      ) : null}

      {/* 当前执行的步骤（展开） */}
      {activeStep ? (
        <div className={styles.activeStep}>
          <div className={styles.activeStepHeader}>
            <span>◐</span>
            <span>{activeStep.title}</span>
          </div>
          <div className={styles.activeStepBody}>
            {activeStep.content ? (
              <div className={styles.stepMarkdown}>
                <XMarkdown
                  content={activeStep.content}
                  streaming={{
                    hasNextChunk: isStreaming,
                    enableAnimation: true,
                  }}
                />
              </div>
            ) : null}
            {activeStepActivities.map((activity) => (
              <div key={activity.id} className={styles.activity}>
                <span>{activity.label}</span>
                {activity.detail ? <span className={styles.activityDetail}>{activity.detail}</span> : null}
              </div>
            ))}
          </div>
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

      {content ? (
        <div className={styles.markdown}>
          <XMarkdown
            content={content}
            streaming={{
              hasNextChunk: isStreaming,
              enableAnimation: true,
            }}
          />
        </div>
      ) : null}

      {runtime?.status ? <div className={styles.footer}>{runtime.status.label}</div> : null}
    </>
  );
}

export function AssistantReply({
  content,
  runtime,
  status,
  messageId,
  hasRuntime,
  onLoadRuntime,
}: AssistantReplyProps) {
  const { styles } = useAssistantReplyStyles();
  const [localRuntime, setLocalRuntime] = useState<AssistantReplyRuntime | null>(null);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState(false);

  // 显示的 runtime：优先使用已加载的，其次使用传入的
  const displayRuntime = localRuntime || runtime;

  // 有 runtime 时直接显示完整内容
  if (displayRuntime) {
    return (
      <div className={styles.root}>
        <AssistantReplyContent
          content={content}
          runtime={displayRuntime}
          status={status}
          styles={styles}
        />
      </div>
    );
  }

  // 无 runtime 且没有懒加载能力，只显示 markdown
  if (!messageId || !hasRuntime || !onLoadRuntime) {
    return (
      <div className={styles.root}>
        <SimpleMarkdownContent content={content} styles={styles} />
      </div>
    );
  }

  // 懒加载场景
  const handleExpand = async () => {
    if (loading) return;
    setLoading(true);
    setExpanded(true);
    const loaded = await onLoadRuntime(messageId);
    if (loaded) {
      setLocalRuntime(loaded);
    }
    setLoading(false);
  };

  // 加载中
  if (loading) {
    return (
      <div className={styles.root}>
        <SimpleMarkdownContent content={content} styles={styles} />
        <div className={styles.loadingContainer}>
          <Skeleton active paragraph={{ rows: 3 }} />
        </div>
      </div>
    );
  }

  // 已展开但加载失败
  if (expanded && !localRuntime) {
    return (
      <div className={styles.root}>
        <SimpleMarkdownContent content={content} styles={styles} />
        <Button type="link" className={styles.expandButton} onClick={handleExpand}>
          重试加载详情
        </Button>
      </div>
    );
  }

  // 未展开，显示展开按钮
  if (!expanded) {
    return (
      <div className={styles.root}>
        <SimpleMarkdownContent content={content} styles={styles} />
        <Button
          type="link"
          className={styles.expandButton}
          onClick={handleExpand}
          icon={<DownOutlined />}
        >
          展开详情
        </Button>
      </div>
    );
  }

  // 已展开且有 localRuntime（正常情况会走第一个分支）
  return (
    <div className={styles.root}>
      <SimpleMarkdownContent content={content} styles={styles} />
    </div>
  );
}
