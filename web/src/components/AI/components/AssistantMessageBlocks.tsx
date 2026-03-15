import React, { useEffect, useState } from 'react';
import { theme } from 'antd';
import { Think, CodeHighlighter } from '@ant-design/x';
import XMarkdown, { type ComponentProps } from '@ant-design/x-markdown';
import { RecommendationCard } from './RecommendationCard';
import { ToolCard } from './ToolCard';
import { ConfirmationPanel } from './ConfirmationPanel';
import type {
  ApprovalBlock,
  AssistantMessageBlock,
  ErrorBlock,
  EvidenceBlock,
  PlanBlock,
  RawEvidenceBlock,
  RecommendationsBlock,
  StatusBlock,
  ToolExecutionBlock,
} from '../messageBlocks';
import type { ConfirmationRequest, ToolExecution } from '../types';

class BlockErrorBoundary extends React.Component<{
  fallback: React.ReactNode;
  children: React.ReactNode;
}, { hasError: boolean }> {
  state = { hasError: false };

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  componentDidCatch(error: Error) {
    console.error('AI message block failed to render', error);
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback;
    }

    return this.props.children;
  }
}

const RawCodeBlock: React.FC<{ lang?: string; content: string }> = ({ lang, content }) => (
  <CodeHighlighter lang={lang}>{content}</CodeHighlighter>
);

const SafeCodeBlock: React.FC<{ lang?: string; content: string }> = ({ lang, content }) => (
  <BlockErrorBoundary fallback={<pre><code>{content}</code></pre>}>
    <RawCodeBlock lang={lang} content={content} />
  </BlockErrorBoundary>
);

const MarkdownCode: React.FC<ComponentProps> = ({ className, children }) => {
  const lang = className?.match(/language-(\w+)/)?.[1] || '';

  if (typeof children !== 'string') return null;

  return <SafeCodeBlock lang={lang} content={children} />;
};

const RawMarkdownBlock: React.FC<{ content: string }> = ({ content }) => (
  <div className="ai-markdown-content">
    <XMarkdown components={{ code: MarkdownCode }}>{content}</XMarkdown>
  </div>
);

const MarkdownBlock: React.FC<{ content: string }> = ({ content }) => (
  <BlockErrorBoundary fallback={<pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{content}</pre>}>
    <RawMarkdownBlock content={content} />
  </BlockErrorBoundary>
);

const StreamingMarkdownBlock: React.FC<{ content: string; streaming?: boolean }> = ({ content, streaming }) => {
  const [visible, setVisible] = useState(content);

  useEffect(() => {
    if (!streaming) {
      setVisible(content);
      return;
    }
    if (content.startsWith(visible)) {
      const remainder = content.slice(visible.length);
      if (!remainder) {
        return;
      }
      const timer = window.setTimeout(() => {
        setVisible((prev) => prev + remainder);
      }, 24);
      return () => window.clearTimeout(timer);
    }
    setVisible(content);
  }, [content, streaming, visible]);

  return <MarkdownBlock content={visible} />;
};

const ThinkingMessageBlock: React.FC<{ content: string; isStreaming?: boolean }> = ({ content, isStreaming }) => {
  const [expanded, setExpanded] = useState(false);
  const [displayStreaming, setDisplayStreaming] = useState(Boolean(isStreaming));

  useEffect(() => {
    if (isStreaming) {
      setDisplayStreaming(true);
      return undefined;
    }
    const timer = window.setTimeout(() => {
      setDisplayStreaming(false);
    }, 600);
    return () => window.clearTimeout(timer);
  }, [isStreaming, content]);

  const title = displayStreaming ? (
    <span className="ai-thinking-title-animated">正在思考</span>
  ) : '已思考';

  return (
    <BlockErrorBoundary fallback={<pre style={{ whiteSpace: 'pre-wrap', margin: '0 0 12px' }}>{content}</pre>}>
      <div style={{ marginBottom: 12 }}>
        <Think
          loading={displayStreaming}
          title={title}
          expanded={expanded}
          onExpand={setExpanded}
        >
          {content}
        </Think>
      </div>
    </BlockErrorBoundary>
  );
};

const RecommendationMessageBlock: React.FC<{
  block: RecommendationsBlock;
  onRecommendationSelect?: (prompt: string) => void;
}> = ({ block, onRecommendationSelect }) => (
  <BlockErrorBoundary
    fallback={(
      <div style={{ marginTop: 12 }}>
        {block.recommendations.map((item) => (
          <div key={item.id}>{item.title}</div>
        ))}
      </div>
    )}
  >
    {onRecommendationSelect ? (
      <RecommendationCard recommendations={block.recommendations} onSelect={onRecommendationSelect} />
    ) : null}
  </BlockErrorBoundary>
);

const RawEvidenceMessageBlock: React.FC<{ block: RawEvidenceBlock }> = ({ block }) => (
  <BlockErrorBoundary
    fallback={(
      <pre style={{ whiteSpace: 'pre-wrap', margin: '12px 0 0' }}>
        {block.items.join('\n')}
      </pre>
    )}
  >
    <div style={{ marginTop: 12 }}>
      <div style={{ fontSize: 12, opacity: 0.75, marginBottom: 6 }}>原始执行证据</div>
      <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
        {block.items.map((item) => `- ${item}`).join('\n')}
      </pre>
    </div>
  </BlockErrorBoundary>
);

const StatusMessageBlock: React.FC<{ block: StatusBlock }> = ({ block }) => {
  const { token } = theme.useToken();
  const statusTone = getBlockStatusTone(block.status, token);
  const metaItems = buildStatusMeta(block.payload);

  return (
    <BlockErrorBoundary fallback={<pre>{block.content}</pre>}>
      <div
        style={{
          marginBottom: 12,
          padding: '10px 12px',
          borderRadius: 12,
          background: statusTone.bg,
          border: `1px solid ${statusTone.border}`,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4, flexWrap: 'wrap' }}>
          {block.title ? <div style={{ fontSize: 12, fontWeight: 600, color: statusTone.text }}>{block.title}</div> : null}
          {block.status ? (
            <span style={{
              fontSize: 11,
              padding: '2px 8px',
              borderRadius: 999,
              background: token.colorBgContainer,
              color: statusTone.text,
              border: `1px solid ${statusTone.border}`,
            }}
            >
              {formatBlockStatus(block.status)}
            </span>
          ) : null}
        </div>
        <div style={{ fontSize: 13, lineHeight: 1.6 }}>{block.content}</div>
        {metaItems.length > 0 ? (
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginTop: 8 }}>
            {metaItems.map((item) => (
              <span
                key={`${item.label}:${item.value}`}
                style={{
                  fontSize: 11,
                  color: token.colorTextSecondary,
                  padding: '2px 8px',
                  borderRadius: 999,
                  background: token.colorBgContainer,
                  border: `1px solid ${token.colorBorderSecondary}`,
                }}
              >
                {item.label}: {item.value}
              </span>
            ))}
          </div>
        ) : null}
      </div>
    </BlockErrorBoundary>
  );
};

const PlanMessageBlock: React.FC<{ block: PlanBlock }> = ({ block }) => {
  const { token } = theme.useToken();

  return (
    <BlockErrorBoundary fallback={<pre>{block.content || JSON.stringify(block.payload, null, 2)}</pre>}>
      <details style={{ marginBottom: 12 }}>
        <summary>{block.title || '执行计划'}</summary>
        <div style={{ marginTop: 8 }}>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: block.content ? 8 : 0 }}>
            {typeof block.total === 'number' ? (
              <span style={metaChipStyle(token)}>共 {block.total} 步</span>
            ) : null}
            {block.steps && block.steps.length > 0 ? (
              <span style={metaChipStyle(token)}>已接收 {block.steps.length} 个计划步骤</span>
            ) : null}
          </div>
          {block.content ? <MarkdownBlock content={block.content} /> : null}
          {block.steps && block.steps.length > 0 ? (
            <ol style={{ margin: '8px 0 0', paddingLeft: 20 }}>
              {block.steps.map((step, index) => (
                <li key={step.id || `${block.id}:${index}`} style={{ marginBottom: 10 }}>
                  <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
                    <div style={{ fontWeight: 500 }}>
                      {step.content || step.title || `步骤 ${index + 1}`}
                    </div>
                    {step.status ? <span style={metaChipStyle(token)}>{formatBlockStatus(step.status)}</span> : null}
                    {step.id ? <span style={metaChipStyle(token)}>ID {step.id}</span> : null}
                  </div>
                  {step.tool_hint ? (
                    <div style={{ fontSize: 12, color: token.colorTextSecondary, marginTop: 4 }}>
                      工具提示: <code>{step.tool_hint}</code>
                    </div>
                  ) : null}
                  {step.summary ? (
                    <div style={{ fontSize: 12, color: token.colorTextSecondary, marginTop: 4 }}>
                      {step.summary}
                    </div>
                  ) : null}
                </li>
              ))}
            </ol>
          ) : null}
          {block.payload && !block.content && (!block.steps || block.steps.length === 0) ? (
            <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
              {JSON.stringify(block.payload, null, 2)}
            </pre>
          ) : null}
        </div>
      </details>
    </BlockErrorBoundary>
  );
};

const ToolMessageBlock: React.FC<{ block: ToolExecutionBlock }> = ({ block }) => {
  const payload = block.payload || {};
  const resultPayload = payload.result;
  const tool: ToolExecution = {
    id: block.id,
    name: String(payload.tool_name || payload.tool || block.title || 'tool'),
    status: block.status === 'error' ? 'error' : block.status === 'success' ? 'success' : 'running',
    summary: typeof payload.user_visible_summary === 'string'
      ? String(payload.user_visible_summary)
      : typeof payload.summary === 'string'
        ? String(payload.summary)
        : typeof payload.title === 'string'
          ? String(payload.title)
          : undefined,
    target: typeof payload.target === 'string'
      ? String(payload.target)
      : typeof payload.step_id === 'string'
        ? `step_id=${String(payload.step_id)}`
        : typeof payload.plan_id === 'string'
          ? `plan_id=${String(payload.plan_id)}`
        : typeof payload.host_id === 'string' || typeof payload.host_id === 'number'
          ? `host_id=${String(payload.host_id)}`
        : undefined,
    params: (payload.params as Record<string, unknown>) || undefined,
    result: typeof resultPayload === 'object' && resultPayload ? {
      ok: (resultPayload as Record<string, unknown>).ok !== false,
      data: (resultPayload as Record<string, unknown>).data,
      error: typeof (resultPayload as Record<string, unknown>).error === 'string' ? String((resultPayload as Record<string, unknown>).error) : undefined,
      latency_ms: typeof (resultPayload as Record<string, unknown>).latency_ms === 'number' ? Number((resultPayload as Record<string, unknown>).latency_ms) : undefined,
    } : undefined,
    error: typeof payload.error === 'string' ? String(payload.error) : undefined,
  };
  return <ToolCard tool={tool} />;
};

const ApprovalMessageBlock: React.FC<{ block: ApprovalBlock; onApprovalDecision?: (payload: Record<string, unknown>, approved: boolean, editedArgs?: string, reason?: string) => void }> = ({ block, onApprovalDecision }) => {
  const payload = block.payload || {};
  const confirmation: ConfirmationRequest = {
    id: block.id,
    title: String(payload.title || block.title || '等待确认'),
    description: String(payload.summary || payload.title || '此操作需要你的确认后继续执行'),
    risk: (payload.risk || 'high') as 'low' | 'medium' | 'high',
    status: (payload.status || 'waiting_user') as 'waiting_user' | 'submitting' | 'failed',
    errorMessage: typeof payload.error_message === 'string' ? payload.error_message : undefined,
    details: payload,
    // Canonical fields for resume identity
    toolName: payload.tool_name ? String(payload.tool_name) : undefined,
    toolDisplayName: payload.tool_display_name ? String(payload.tool_display_name) : undefined,
    planId: payload.plan_id ? String(payload.plan_id) : undefined,
    stepId: payload.step_id ? String(payload.step_id) : undefined,
    checkpointId: payload.checkpoint_id ? String(payload.checkpoint_id) : undefined,
    target: payload.target ? String(payload.target) : undefined,
    argumentsJson: payload.arguments_json ? String(payload.arguments_json) : undefined,
    editable: true,
    onConfirm: (editedArgs?: string) => onApprovalDecision?.(payload, true, editedArgs),
    onCancel: (reason?: string) => onApprovalDecision?.(payload, false, undefined, reason),
  };
  return <ConfirmationPanel confirmation={confirmation} />;
};

const EvidenceMessageBlock: React.FC<{ block: EvidenceBlock }> = ({ block }) => (
  <BlockErrorBoundary fallback={<pre>{block.items.join('\n')}</pre>}>
    <div style={{ marginTop: 12 }}>
      <div style={{ fontSize: 12, opacity: 0.75, marginBottom: 6 }}>{block.title || '执行证据'}</div>
      <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
        {block.items.map((item) => `- ${item}`).join('\n')}
      </pre>
    </div>
  </BlockErrorBoundary>
);

const ErrorMessageBlock: React.FC<{ block: ErrorBlock }> = ({ block }) => (
  <BlockErrorBoundary fallback={<pre>{block.content}</pre>}>
    <div style={{ marginBottom: 12, padding: '10px 12px', borderRadius: 12, background: 'rgba(255,77,79,0.08)', color: '#cf1322' }}>
      <div style={{ fontWeight: 600, marginBottom: 4 }}>{block.title || '执行错误'}</div>
      <div style={{ whiteSpace: 'pre-wrap' }}>{block.content}</div>
    </div>
  </BlockErrorBoundary>
);

export function AssistantMessageBlocks({
  blocks,
  onRecommendationSelect,
  onApprovalDecision,
}: {
  blocks: AssistantMessageBlock[];
  onRecommendationSelect?: (prompt: string) => void;
  onApprovalDecision?: (payload: Record<string, unknown>, approved: boolean, editedArgs?: string, reason?: string) => void;
}) {
  const { token } = theme.useToken();

  return (
    <>
      {blocks.map((block) => {
        switch (block.type) {
          case 'thinking':
            return (
              <ThinkingMessageBlock
                key={block.id}
                content={block.content}
                isStreaming={block.isStreaming}
              />
            );
          case 'markdown':
            return <StreamingMarkdownBlock key={block.id} content={block.content} streaming={block.streaming} />;
          case 'status':
            return <StatusMessageBlock key={block.id} block={block} />;
          case 'plan':
            return <PlanMessageBlock key={block.id} block={block} />;
          case 'tool':
            return <ToolMessageBlock key={block.id} block={block} />;
          case 'approval':
            return <ApprovalMessageBlock key={block.id} block={block} onApprovalDecision={onApprovalDecision} />;
          case 'evidence':
            return <EvidenceMessageBlock key={block.id} block={block} />;
          case 'error':
            return <ErrorMessageBlock key={block.id} block={block} />;
          case 'recommendations':
            return (
              <RecommendationMessageBlock
                key={block.id}
                block={block}
                onRecommendationSelect={onRecommendationSelect}
              />
            );
          case 'raw_evidence':
            return <RawEvidenceMessageBlock key={block.id} block={block} />;
          case 'fallback':
          default:
            return (
              <pre
                key={block.id}
                style={{
                  whiteSpace: 'pre-wrap',
                  margin: 0,
                  color: token.colorText,
                }}
              >
                {('content' in block && block.content) || ''}
              </pre>
            );
        }
      })}
    </>
  );
}

export default AssistantMessageBlocks;

function getBlockStatusTone(status: string | undefined, token: any) {
  const normalized = String(status || '').toLowerCase();
  if (normalized.includes('error') || normalized.includes('failed')) {
    return {
      bg: token.colorErrorBg,
      border: token.colorErrorBorder,
      text: token.colorError,
    };
  }
  if (normalized.includes('success') || normalized.includes('completed')) {
    return {
      bg: token.colorSuccessBg,
      border: token.colorSuccessBorder,
      text: token.colorSuccess,
    };
  }
  return {
    bg: token.colorPrimaryBg,
    border: token.colorPrimaryBorder,
    text: token.colorPrimary,
  };
}

function buildStatusMeta(payload: Record<string, unknown> | undefined): Array<{ label: string; value: string }> {
  if (!payload) {
    return [];
  }
  const items: Array<{ label: string; value: string }> = [];
  if (typeof payload.phase === 'string') {
    items.push({ label: '阶段', value: payload.phase });
  }
  if (typeof payload.plan_id === 'string') {
    items.push({ label: '计划', value: payload.plan_id });
  }
  if (typeof payload.previous_plan_id === 'string') {
    items.push({ label: '原计划', value: payload.previous_plan_id });
  }
  if (typeof payload.completed_steps === 'number') {
    items.push({ label: '已完成', value: `${payload.completed_steps} 步` });
  }
  return items.slice(0, 4);
}

function formatBlockStatus(status: string): string {
  switch (status) {
    case 'running':
    case 'loading':
    case 'streaming':
      return '进行中';
    case 'success':
    case 'completed':
      return '已完成';
    case 'error':
    case 'failed':
      return '失败';
    case 'waiting_approval':
      return '待确认';
    default:
      return status;
  }
}

function metaChipStyle(token: any): React.CSSProperties {
  return {
    fontSize: 11,
    color: token.colorTextSecondary,
    padding: '2px 8px',
    borderRadius: 999,
    background: token.colorBgContainer,
    border: `1px solid ${token.colorBorderSecondary}`,
  };
}
