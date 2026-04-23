import React from 'react';
import {
  isProjectionHydrationPending,
  loadStepContent,
} from '../historyProjection';
import type { XChatMessage } from '../types';

const SCROLL_BUTTON_RECOVERY_THRESHOLD = 24;
const SCROLL_BUTTON_VISIBILITY_THRESHOLD = 120;

type FollowState = 'following' | 'detached';

interface CopilotMessageItem {
  id?: string | number;
  status?: string;
  message: XChatMessage;
}

interface CopilotRenderedMessage extends CopilotMessageItem {
  renderKey: string;
}

interface UseCopilotStreamOptions {
  messages: CopilotMessageItem[];
  open: boolean;
  activeConversationKey?: string;
}

export function useCopilotStream({
  messages,
  open,
  activeConversationKey,
}: UseCopilotStreamOptions) {
  const contentRef = React.useRef<HTMLDivElement>(null);
  const retainedAssistantBodiesRef = React.useRef(new Map<string, string>());
  const followStateRef = React.useRef<FollowState>('following');
  const programmaticScrollRef = React.useRef(false);
  const pendingInitialScrollRef = React.useRef(false);
  const pendingSendScrollRef = React.useRef(false);
  const [showScrollBottomBtn, setShowScrollBottomBtn] = React.useState(false);

  const renderedMessages = React.useMemo<CopilotRenderedMessage[]>(() => {
    const retainedAssistantBodies = retainedAssistantBodiesRef.current;
    const nextKeys = new Set<string>();
    const nextMessages = messages.map((item, index) => {
      const renderKey = String(item.id || `${item.message.role}-${index}`);
      nextKeys.add(renderKey);

      if (item.message.role !== 'assistant') {
        return {
          ...item,
          renderKey,
        };
      }

      const currentContent = item.message.content || '';
      const retainedContent = retainedAssistantBodies.get(renderKey);
      const displayContent = isProjectionHydrationPending(item.message) && retainedContent
        ? retainedContent
        : currentContent;

      if (currentContent.trim()) {
        retainedAssistantBodies.set(renderKey, displayContent);
      }

      return {
        ...item,
        message: {
          ...item.message,
          content: displayContent,
        },
        renderKey,
      };
    });

    Array.from(retainedAssistantBodies.keys()).forEach((key) => {
      if (!nextKeys.has(key)) {
        retainedAssistantBodies.delete(key);
      }
    });

    return nextMessages;
  }, [messages]);

  const buildStepContentLoader = React.useCallback(
    (runtime?: XChatMessage['runtime']) => async (_stepId: string, stepIndex: number) => {
      if (!runtime?._executorBlocks) {
        return null;
      }

      const executorBlocks = runtime._executorBlocks;
      if (stepIndex < 0 || stepIndex >= executorBlocks.length) {
        return null;
      }

      const block = executorBlocks[stepIndex];
      if (!block) {
        return null;
      }

      return loadStepContent(block, stepIndex);
    },
    [],
  );

  const withProgrammaticScroll = React.useCallback((callback: () => void) => {
    programmaticScrollRef.current = true;
    callback();
    requestAnimationFrame(() => {
      programmaticScrollRef.current = false;
    });
  }, []);

  const scrollToBottom = React.useCallback((behavior: ScrollBehavior = 'auto', force = false) => {
    const element = contentRef.current;
    if (!element || (!force && followStateRef.current !== 'following')) {
      return;
    }

    withProgrammaticScroll(() => {
      element.scrollTo({ top: element.scrollHeight, behavior });
    });
  }, [withProgrammaticScroll]);

  const markPendingSend = React.useCallback(() => {
    followStateRef.current = 'following';
    pendingSendScrollRef.current = true;
  }, []);

  React.useLayoutEffect(() => {
    followStateRef.current = 'following';
    pendingInitialScrollRef.current = true;
  }, [activeConversationKey, open]);

  React.useLayoutEffect(() => {
    if (!open) {
      return;
    }
    if (!pendingInitialScrollRef.current && !pendingSendScrollRef.current) {
      return;
    }
    if (pendingInitialScrollRef.current && renderedMessages.length === 0) {
      return;
    }

    const frameId = requestAnimationFrame(() => {
      if (pendingInitialScrollRef.current || pendingSendScrollRef.current) {
        scrollToBottom('auto', true);
      }
      pendingInitialScrollRef.current = false;
      pendingSendScrollRef.current = false;
    });

    return () => cancelAnimationFrame(frameId);
  }, [activeConversationKey, open, renderedMessages.length, scrollToBottom]);

  React.useEffect(() => {
    if (!open) {
      return;
    }

    const element = contentRef.current;
    if (!element) {
      return;
    }

    const resizeObserver = new ResizeObserver(() => {
      if (followStateRef.current === 'following') {
        scrollToBottom('auto');
      }
    });

    resizeObserver.observe(element);
    return () => {
      resizeObserver.disconnect();
    };
  }, [open, scrollToBottom]);

  React.useEffect(() => {
    const element = contentRef.current;
    if (!element || !open) {
      return;
    }

    const updateButtonVisibility = () => {
      const distanceToBottom = element.scrollHeight - element.scrollTop - element.clientHeight;
      if (!programmaticScrollRef.current) {
        followStateRef.current = distanceToBottom <= SCROLL_BUTTON_RECOVERY_THRESHOLD
          ? 'following'
          : 'detached';
      }
      setShowScrollBottomBtn(distanceToBottom > SCROLL_BUTTON_VISIBILITY_THRESHOLD);
    };

    updateButtonVisibility();
    element.addEventListener('scroll', updateButtonVisibility, { passive: true });

    return () => {
      element.removeEventListener('scroll', updateButtonVisibility);
    };
  }, [messages.length, open]);

  const handleScrollToBottom = React.useCallback(() => {
    followStateRef.current = 'following';
    scrollToBottom('smooth', true);
  }, [scrollToBottom]);

  return {
    buildStepContentLoader,
    contentRef,
    handleScrollToBottom,
    markPendingSend,
    renderedMessages,
    scrollToBottom,
    showScrollBottomBtn,
  };
}
