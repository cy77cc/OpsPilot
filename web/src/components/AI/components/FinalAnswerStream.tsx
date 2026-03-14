import React, { useEffect, useState } from 'react';

interface FinalAnswerStreamProps {
  content: string;
  visible: boolean;
  streaming?: boolean;
  reducedMotion?: boolean;
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
      <div className="ai-markdown-content">{displayContent}</div>
      {streaming ? <span aria-label="final-answer-streaming" className="final-answer-stream__caret">|</span> : null}
    </div>
  );
}

export default FinalAnswerStream;
