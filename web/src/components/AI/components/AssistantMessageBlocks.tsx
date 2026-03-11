import React, { useEffect, useState } from 'react';
import { theme } from 'antd';
import { Think, CodeHighlighter } from '@ant-design/x';
import XMarkdown, { type ComponentProps } from '@ant-design/x-markdown';
import { RecommendationCard } from './RecommendationCard';
import type { AssistantMessageBlock, RawEvidenceBlock, RecommendationsBlock } from '../messageBlocks';

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

const ThinkingMessageBlock: React.FC<{ content: string; isStreaming?: boolean }> = ({ content, isStreaming }) => {
  const [expanded, setExpanded] = useState(Boolean(isStreaming));

  useEffect(() => {
    setExpanded(Boolean(isStreaming));
  }, [isStreaming]);

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

export function AssistantMessageBlocks({
  blocks,
  onRecommendationSelect,
}: {
  blocks: AssistantMessageBlock[];
  onRecommendationSelect?: (prompt: string) => void;
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
            return <MarkdownBlock key={block.id} content={block.content} />;
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
