import React, { useState } from 'react';
import XMarkdown from '@ant-design/x-markdown';
import { createStyles } from 'antd-style';
import { Collapse, Button, Skeleton } from 'antd';
import { DownOutlined } from '@ant-design/icons';
import { normalizeMarkdownContent } from './markdownContent';
import type { AssistantReplyActivity, AssistantReplyRuntime, AssistantReplySegment } from './types';
import ToolReference from './ToolReference';

const useAssistantReplyStyles = createStyles(({ token, css }) => ({
  root: css`
    display: flex;
    flex-direction: column;
    gap: 0px;
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

    h1,
    h2,
    h3,
    h4,
    h5,
    h6 {
      margin: 1.1em 0 0.45em;
      color: ${token.colorText};
      font-weight: 700;
      line-height: 1.3;
    }

    h1 {
      font-size: 28px;
    }

    h2 {
      font-size: 22px;
    }

    h3 {
      font-size: 18px;
    }

    h4,
    h5,
    h6 {
      font-size: 16px;
    }

    p {
      margin: 0 0 0.85em;
    }

    ul,
    ol {
      margin: 0 0 0.85em;
      padding-left: 1.4em;
    }

    li {
      margin: 0.2em 0;
    }

    blockquote {
      margin: 0 0 0.85em;
      padding-left: 12px;
      border-left: 3px solid ${token.colorBorder};
      color: ${token.colorTextSecondary};
    }
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

    h1,
    h2,
    h3,
    h4,
    h5,
    h6 {
      margin: 1.1em 0 0.45em;
      color: ${token.colorText};
      font-weight: 700;
      line-height: 1.3;
    }

    h1 {
      font-size: 28px;
    }

    h2 {
      font-size: 22px;
    }

    h3 {
      font-size: 18px;
    }

    h4,
    h5,
    h6 {
      font-size: 16px;
    }

    p {
      margin: 0 0 0.9em;
    }

    ul,
    ol {
      margin: 0 0 0.9em;
      padding-left: 1.5em;
    }

    li {
      margin: 0.25em 0;
    }

    blockquote {
      margin: 0 0 0.9em;
      padding: 0 0 0 12px;
      border-left: 3px solid ${token.colorBorder};
      color: ${token.colorTextSecondary};
    }

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
      margin: 0 0 0.9em;
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
  errorContainer: css`
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    font-size: 13px;
  `,
  inlineToolFlow: css`
    display: inline;
    white-space: normal;
  `,
  inlineText: css`
    display: inline;
    line-height: 1.65;
    white-space: pre-wrap;
  `,
  inlineToolFallback: css`
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
    min-height: 20px;
  `,
}));

interface AssistantReplyProps {
  content: string;
  runtime?: AssistantReplyRuntime;
  status?: string;
  messageId?: string;
  hasRuntime?: boolean;
  onLoadRuntime?: (messageId: string) => Promise<AssistantReplyRuntime | null>;
  onLoadStepContent?: (
    stepId: string,
    stepIndex: number,
  ) => Promise<{
    content: string;
    segments: AssistantReplySegment[];
    activities: AssistantReplyActivity[];
  } | null>;
}

// SimpleMarkdownContent 只渲染 markdown 内容，没有 runtime
function SimpleMarkdownContent({
  content,
  styles,
  isStreaming = false,
}: {
  content: string;
  styles: Record<string, string>;
  isStreaming?: boolean;
}) {
  const normalizedContent = normalizeMarkdownContent(content);
  if (!normalizedContent) return null;
  return (
    <div className={styles.markdown}>
      <XMarkdown content={normalizedContent} streaming={{ hasNextChunk: isStreaming, enableAnimation: true }} />
    </div>
  );
}

function shouldRenderInlineText(text: string): boolean {
  const value = (text || '').trim();
  if (!value) {
    return false;
  }
  if (/\n/.test(value)) {
    return false;
  }
  if (/^\s{0,3}(#{1,6}|\d+\. |- |\* |```|>|---)/.test(value)) {
    return false;
  }
  return true;
}

// StepContentRenderer 按 segments 顺序渲染步骤内容
function StepContentRenderer({
  step,
  activities,
  isStreaming,
  styles,
}: {
  step: { content?: string; segments?: AssistantReplySegment[] };
  activities: AssistantReplyActivity[];
  isStreaming: boolean;
  styles: Record<string, string>;
}) {
  // 如果有 segments，按顺序渲染
  if (step.segments && step.segments.length > 0) {
    const activityMap = new Map<string, AssistantReplyActivity>();
    activities.forEach((a) => activityMap.set(a.id, a));

    const elements: React.ReactNode[] = [];

    step.segments.forEach((segment, index) => {
      if (segment.type === 'text' && segment.text) {
        const inlineText = shouldRenderInlineText(segment.text);
        elements.push(inlineText ? (
          <span key={`text-${index}`} className={styles.inlineText}>
            {normalizeMarkdownContent(segment.text)}
          </span>
        ) : (
          <XMarkdown
            key={`text-${index}`}
            content={normalizeMarkdownContent(segment.text)}
            streaming={{ hasNextChunk: isStreaming, enableAnimation: true }}
          />
        ));
      } else if (segment.type === 'tool_ref' && segment.callId) {
        const activity = activityMap.get(segment.callId);
        if (activity) {
          elements.push(<ToolReference key={`tool-${segment.callId}`} activity={activity} />);
        }
      }
    });

    const hasOnlyTools = elements.length > 0 && elements.every((element) =>
      React.isValidElement(element) && typeof element.key === 'string' && String(element.key).startsWith('tool-'));

    return (
      <div className={hasOnlyTools ? styles.inlineToolFallback : styles.stepMarkdown}>
        <span className={styles.inlineToolFlow}>{elements}</span>
      </div>
    );
  }

  // 向后兼容：使用 content 字段 + activities 追加在末尾
  if (step.content || activities.length > 0) {
    return (
      <div className={styles.stepMarkdown}>
        {step.content && (
          <XMarkdown
            content={normalizeMarkdownContent(step.content)}
            streaming={{ hasNextChunk: isStreaming, enableAnimation: true }}
          />
        )}
      </div>
    );
  }

  return null;
}

// Step 加载状态：完整的状态机
type StepLoadState = 'idle' | 'loading' | 'success' | 'error';

// AssistantReplyContent 渲染完整的 runtime 内容
function AssistantReplyContent({
  content,
  runtime,
  status,
  styles,
  onLoadStepContent,
}: {
  content: string;
  runtime: AssistantReplyRuntime;
  status?: string;
  styles: Record<string, string>;
  onLoadStepContent?: AssistantReplyProps['onLoadStepContent'];
}) {
  // 展开状态
  const [stepExpandStates, setStepExpandStates] = useState<Record<string, boolean>>({});
  // 加载状态：状态机
  const [stepLoadStates, setStepLoadStates] = useState<Record<string, StepLoadState>>({});
  // 内容缓存：异步加载的数据存储于此
  const [stepContentCache, setStepContentCache] = useState<Record<string, {
    content: string;
    segments: AssistantReplySegment[];
    activities: AssistantReplyActivity[];
  } | null>>({});

  const activeStepIndex = runtime?.plan?.activeStepIndex;
  const allSteps = runtime?.plan?.steps || [];
  const hasPlan = runtime?.plan && allSteps.length > 0;

  // 当前执行的步骤
  const activeStep = activeStepIndex !== undefined && activeStepIndex >= 0 && activeStepIndex < allSteps.length
    ? allSteps[activeStepIndex]
    : null;

  // 已完成的步骤（历史对话中所有步骤都是 done）
  const completedSteps = activeStepIndex !== undefined && activeStepIndex >= 0
    ? allSteps.slice(0, activeStepIndex).filter((step) => step.status === 'done')
    : allSteps.filter((step) => step.status === 'done');

  // 是否有 plan 结构
  const isPlanBased = hasPlan && activeStepIndex !== undefined && activeStepIndex >= 0;

  const toolActivities = isPlanBased
    ? runtime?.activities?.filter(
        (activity) => activity.stepIndex === activeStepIndex &&
          activity.kind === 'tool'
      ) || []
    : [];

  const activeStepActivities = isPlanBased
    ? runtime?.activities?.filter(
        (activity) => activity.stepIndex === activeStepIndex &&
          activity.kind !== 'tool'
      ) || []
    : hasPlan
      ? []
      : runtime?.activities || [];

  const standaloneActivities = !hasPlan ? runtime?.activities || [] : [];
  const shouldRenderSummary = Boolean(runtime?.summary?.items?.length);

  const isStreaming = status === 'loading' || status === 'updating';

  // 处理 step 展开，触发懒加载
  const handleStepExpand = async (stepId: string, stepIndex: number) => {
    // 1. 状态检查：防止重复请求（竞态条件处理）
    const currentState = stepLoadStates[stepId] || 'idle';
    if (currentState === 'loading' || currentState === 'success') {
      // 已在加载或已成功，直接展开
      if (currentState === 'success') {
        setStepExpandStates(prev => ({ ...prev, [stepId]: true }));
      }
      return;
    }

    // 2. 检查是否已有缓存
    if (stepContentCache[stepId]) {
      setStepExpandStates(prev => ({ ...prev, [stepId]: true }));
      return;
    }

    if (!onLoadStepContent) {
      return;
    }

    // 3. 设置 loading 状态
    setStepLoadStates(prev => ({ ...prev, [stepId]: 'loading' }));

    try {
      const result = await onLoadStepContent(stepId, stepIndex);
      if (result) {
        // 4. 成功：存储内容并标记成功
        setStepContentCache(prev => ({ ...prev, [stepId]: result }));
        setStepLoadStates(prev => ({ ...prev, [stepId]: 'success' }));
        setStepExpandStates(prev => ({ ...prev, [stepId]: true }));
      } else {
        // 5. 返回 null：标记错误
        setStepLoadStates(prev => ({ ...prev, [stepId]: 'error' }));
      }
    } catch {
      // 6. 异常：标记错误
      setStepLoadStates(prev => ({ ...prev, [stepId]: 'error' }));
    }
  };

  // 重试函数
  const handleRetry = (stepId: string, stepIndex: number) => {
    // 重置状态后重新触发加载
    setStepLoadStates(prev => ({ ...prev, [stepId]: 'idle' }));
    handleStepExpand(stepId, stepIndex);
  };

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
                <StepContentRenderer
                  step={step}
                  activities={runtime?.activities || []}
                  isStreaming={false}
                  styles={styles}
                />
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
            <StepContentRenderer
              step={activeStep}
              activities={toolActivities}
              isStreaming={isStreaming}
              styles={styles}
            />
            {activeStepActivities.map((activity) => (
              <div key={activity.id} className={styles.activity}>
                <span>{activity.label}</span>
                {activity.detail ? <span className={styles.activityDetail}>{activity.detail}</span> : null}
              </div>
            ))}
          </div>
        </div>
      ): null}

      {standaloneActivities.length > 0 ? (
        <div className={styles.activities}>
          {standaloneActivities.map((activity) => (
            activity.kind === 'tool' ? (
              <ToolReference key={activity.id} activity={activity} />
            ) : (
              <div key={activity.id} className={styles.activity}>
                <span>{activity.label}</span>
                {activity.detail ? <span className={styles.activityDetail}>{activity.detail}</span> : null}
              </div>
            )
          ))}
        </div>
      ) : null}

      {shouldRenderSummary ? (
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
            content={normalizeMarkdownContent(content)}
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
  const isStreaming = status === 'loading' || status === 'updating';

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
        <SimpleMarkdownContent content={content} styles={styles} isStreaming={isStreaming} />
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
        <SimpleMarkdownContent content={content} styles={styles} isStreaming={isStreaming} />
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
        <SimpleMarkdownContent content={content} styles={styles} isStreaming={isStreaming} />
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
        <SimpleMarkdownContent content={content} styles={styles} isStreaming={isStreaming} />
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
      <SimpleMarkdownContent content={content} styles={styles} isStreaming={isStreaming} />
    </div>
  );
}
