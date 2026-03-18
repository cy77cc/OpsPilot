import React, { useState } from 'react';
import XMarkdown from '@ant-design/x-markdown';
import { createStyles } from 'antd-style';
import { Collapse } from 'antd';
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
}));

interface AssistantReplyProps {
  content: string;
  runtime?: AssistantReplyRuntime;
  status?: string;
}

function getVisibleActivities(runtime?: AssistantReplyRuntime): AssistantReplyActivity[] {
  if (!runtime?.activities?.length) {
    return [];
  }
  return runtime.activities.filter((activity) => activity.stepIndex === undefined && activity.kind !== 'plan' && activity.kind !== 'replan');
}

export function AssistantReply({ content, runtime, status }: AssistantReplyProps) {
  const { styles } = useAssistantReplyStyles();
  const visibleActivities = getVisibleActivities(runtime);
  const activeStepIndex = runtime?.plan?.activeStepIndex;
  const allSteps = runtime?.plan?.steps || [];

  // 当前执行的步骤
  const activeStep = activeStepIndex !== undefined && activeStepIndex < allSteps.length
    ? allSteps[activeStepIndex]
    : null;

  // 已完成的步骤
  // - 当有当前步骤时：当前步骤之前的已完成步骤
  // - 当计划完成时：所有已完成的步骤
  const completedSteps = activeStepIndex !== undefined
    ? allSteps.slice(0, activeStepIndex).filter((step) => step.status === 'done')
    : allSteps.filter((step) => step.status === 'done');

  // 当前步骤的 activities
  const activeStepActivities = activeStepIndex !== undefined
    ? runtime?.activities?.filter((activity) => activity.stepIndex === activeStepIndex) || []
    : [];

  return (
    <div className={styles.root}>

      {visibleActivities.length ? (
        <div className={styles.activities}>
          {visibleActivities.map((activity) => (
            <div key={activity.id} className={styles.activity}>
              <span>{activity.label}</span>
              {activity.detail ? <span className={styles.activityDetail}>{activity.detail}</span> : null}
            </div>
          ))}
        </div>
      ) : null}

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
                    hasNextChunk: status === 'loading' || status === 'updating',
                    enableAnimation: true,
                    animationConfig: {
                      fadeDuration: 180,
                      easing: 'ease-out',
                    },
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
              hasNextChunk: status === 'loading' || status === 'updating',
              enableAnimation: true,
              animationConfig: {
                fadeDuration: 180,
                easing: 'ease-out',
              },
            }}
          />
        </div>
      ) : null}

      {runtime?.status ? <div className={styles.footer}>{runtime.status.label}</div> : null}
    </div>
  );
}
