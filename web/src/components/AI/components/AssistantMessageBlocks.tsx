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

  // 动态标题：思考中时显示渐变色动画效果
  const title = isStreaming ? (
    <span className="ai-thinking-title-animated">正在思考</span>
  ) : '已思考';

  return (
    <BlockErrorBoundary fallback={<pre style={{ whiteSpace: 'pre-wrap', margin: '0 0 12px' }}>{content}</pre>}>
      <div style={{ marginBottom: 12 }}>
        <Think
          loading={isStreaming}
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

const StatusMessageBlock: React.FC<{ block: StatusBlock }> = ({ block }) => (
  <BlockErrorBoundary fallback={<pre>{block.content}</pre>}>
    <div style={{ marginBottom: 12, padding: '10px 12px', borderRadius: 12, background: 'rgba(0,0,0,0.03)' }}>
      {block.title && <div style={{ fontSize: 12, opacity: 0.7, marginBottom: 4 }}>{block.title}</div>}
      <div style={{ fontSize: 13, lineHeight: 1.5 }}>{block.content}</div>
    </div>
  </BlockErrorBoundary>
);

const PlanMessageBlock: React.FC<{ block: PlanBlock }> = ({ block }) => (
  <BlockErrorBoundary fallback={<pre>{block.content || JSON.stringify(block.payload, null, 2)}</pre>}>
    <details style={{ marginBottom: 12 }}>
      <summary>{block.title || '执行计划'}</summary>
      <div style={{ marginTop: 8 }}>
        {block.content ? <MarkdownBlock content={block.content} /> : null}
        {block.payload ? (
          <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
            {JSON.stringify(block.payload, null, 2)}
          </pre>
        ) : null}
      </div>
    </details>
  </BlockErrorBoundary>
);

const ToolMessageBlock: React.FC<{ block: ToolExecutionBlock }> = ({ block }) => {
  const payload = block.payload || {};
  const resultPayload = payload.result;
  const tool: ToolExecution = {
    id: block.id,
    name: String(payload.tool_name || payload.tool || block.title || 'tool'),
    status: block.status === 'error' ? 'error' : block.status === 'success' ? 'success' : 'running',
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

const ApprovalMessageBlock: React.FC<{ block: ApprovalBlock; onApprovalDecision?: (payload: Record<string, unknown>, approved: boolean) => void }> = ({ block, onApprovalDecision }) => {
  const payload = block.payload || {};
  const confirmation: ConfirmationRequest = {
    id: block.id,
    title: String(payload.title || block.title || '等待确认'),
    description: String(payload.user_visible_summary || payload.summary || '此操作需要你的确认后继续执行'),
    risk: (payload.risk || 'high') as 'low' | 'medium' | 'high',
    details: payload,
    onConfirm: () => onApprovalDecision?.(payload, true),
    onCancel: () => onApprovalDecision?.(payload, false),
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
  onApprovalDecision?: (payload: Record<string, unknown>, approved: boolean) => void;
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
