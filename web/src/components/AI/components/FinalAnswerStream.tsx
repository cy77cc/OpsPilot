import React, { useEffect, useState } from 'react';
import { CodeHighlighter } from '@ant-design/x';
import XMarkdown, { type ComponentProps } from '@ant-design/x-markdown';

interface FinalAnswerStreamProps {
  content: string;
  visible: boolean;
  streaming?: boolean;
  reducedMotion?: boolean;
}

const MarkdownCode: React.FC<ComponentProps> = ({ className, children }) => {
  const lang = className?.match(/language-(\w+)/)?.[1] || '';
  if (typeof children !== 'string') return null;
  return <CodeHighlighter lang={lang}>{children}</CodeHighlighter>;
};

class BlockErrorBoundary extends React.Component<{ fallback: React.ReactNode; children: React.ReactNode }, { hasError: boolean }> {
  state = { hasError: false };

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  componentDidCatch(error: Error) {
    console.error('FinalAnswerStream markdown render failed', error);
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback;
    }
    return this.props.children;
  }
}

export function FinalAnswerStream({
  content,
  visible,
  streaming = false,
  reducedMotion = false,
}: FinalAnswerStreamProps) {
  const [displayContent, setDisplayContent] = useState(reducedMotion ? content : '');

  useEffect(() => {
    if (!visible) {
      setDisplayContent('');
      return;
    }
    if (reducedMotion || !streaming) {
      setDisplayContent(content);
      return;
    }
    if (!content.startsWith(displayContent)) {
      setDisplayContent(content);
      return;
    }
    if (content === displayContent) {
      return;
    }
    const timer = window.setTimeout(() => {
      const remaining = content.slice(displayContent.length);
      const nextChunk = remaining.slice(0, Math.max(remaining.indexOf(' ') + 1, 2));
      setDisplayContent((prev) => prev + nextChunk);
    }, 42);
    return () => window.clearTimeout(timer);
  }, [content, displayContent, reducedMotion, streaming, visible]);

  if (!visible) {
    return null;
  }

  return (
    <div className={`final-answer-stream ${streaming ? 'final-answer-stream--revealing' : 'final-answer-stream--complete'}`}>
      <BlockErrorBoundary fallback={<pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{displayContent}</pre>}>
        <div className="ai-markdown-content">
          <XMarkdown components={{ code: MarkdownCode }}>{displayContent}</XMarkdown>
        </div>
      </BlockErrorBoundary>
      {streaming ? <span aria-label="final-answer-streaming" className="final-answer-stream__caret">|</span> : null}
    </div>
  );
}

export default FinalAnswerStream;
