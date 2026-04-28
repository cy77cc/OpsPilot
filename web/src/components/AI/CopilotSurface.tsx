import React from 'react';
import {
  RobotOutlined,
  CommentOutlined,
  PlusOutlined,
  PaperClipOutlined,
  CloseOutlined,
  VerticalAlignBottomOutlined,
} from '@ant-design/icons';
import { Bubble, Conversations, Prompts, Sender, Welcome } from '@ant-design/x';
import type { BubbleListProps, ConversationItemType, PromptsItemType } from '@ant-design/x';
import { useXChat, useXConversations } from '@ant-design/x-sdk';
import { App, Button, Drawer, Popover, Space, Tag, Typography } from 'antd';
import { aiApi } from '../../api/modules/ai';
import { AssistantReply } from './AssistantReply';
import { MessageActionBar } from './MessageActionBar';
import {
  hydrateAssistantHistoryFromProjection,
  resetHistoryProjectionCache,
} from './historyProjection';
import { PlatformChatProvider } from './providers';
import type { AssistantReplyRuntime, ChatRequest, ConversationSummary, XChatMessage } from './types';
import type { AIMessage } from '../../features/ai/api/shared';
import type { AISession } from '../../features/ai/api/shared';

interface XChatItemExtraInfo {
  messageId?: string;
  runtime?: AssistantReplyRuntime;
  message?: AIMessage;
}

interface XChatItem {
  key: string;
  label: string;
  extraInfo?: XChatItemExtraInfo;
}
import { useCopilotSessionReducer } from './hooks/useCopilotSessionReducer';
import { useCopilotStream } from './hooks/useCopilotStream';
import { useSceneResolver } from './hooks/useSceneResolver';
import {
  buildHistoricalPendingRuntime,
  mapHistoryMessageStatus,
  NEW_SESSION_KEY,
  SCENE_FALLBACK_PROMPTS,
} from './copilotSurface.constants';
import { useCopilotStyles } from './copilotSurface.styles';

const { Text } = Typography;

type BubbleStatus = 'success' | 'error' | 'abort' | 'loading' | 'local' | 'updating';

function toBubbleStatus(status?: string): BubbleStatus | undefined {
  switch (status) {
    case 'success':
    case 'error':
    case 'abort':
    case 'loading':
    case 'local':
    case 'updating':
      return status;
    default:
      return undefined;
  }
}

function toConversationItems(items: ConversationSummary[]): ConversationItemType[] {
  return items.map((item) => ({
    key: item.key,
    label: item.label,
  }));
}

function getConversationLabelFromSummary(item: { title?: string; last_message?: { content?: string } }): string {
  const title = typeof item?.title === 'string' ? item.title.trim() : '';
  if (title) {
    return title;
  }

  const lastMessageContent = typeof item?.last_message?.content === 'string'
    ? item.last_message.content.trim()
    : '';
  if (lastMessageContent) {
    return lastMessageContent.slice(0, 32);
  }

  return 'New chat';
}

function toPromptItems(scene: string, prompts?: Array<{ id: number; prompt_text: string }>): PromptsItemType[] {
  if (prompts && prompts.length > 0) {
    return prompts.map((prompt) => ({
      key: String(prompt.id),
      label: prompt.prompt_text,
      description: prompt.prompt_text,
    }));
  }
  return SCENE_FALLBACK_PROMPTS[scene] || SCENE_FALLBACK_PROMPTS.ai;
}

interface CopilotSurfaceProps {
  open: boolean;
  onClose: () => void;
}

export function buildAssistantErrorContent(
  previousContent: string | undefined,
  errorMessage: string,
) {
  const content = (previousContent || '').trim();
  const error = (errorMessage || 'Request failed').trim();

  if (!content) {
    return error;
  }

  return `${content}\n\n---\n\nError: ${error}`;
}

export async function copyAssistantReplyToClipboard(
  finalMarkdownBody: string,
  runtime?: XChatMessage['runtime'],
): Promise<void> {
  const parts: string[] = [];

  // 1. Add Plan Steps (completed steps only or all if done)
  if (runtime?.plan?.steps) {
    runtime.plan.steps.forEach((step, index) => {
      if (step.status === 'done' || runtime?.status?.kind === 'completed') {
        parts.push(`## 步骤 ${index + 1}: ${step.title}`);
        if (step.content) {
          parts.push(step.content);
        }
        if (step.segments) {
          step.segments.forEach(seg => {
            if (seg.type === 'text' && seg.text) {
              parts.push(seg.text);
            }
          });
        }
        parts.push('\n---\n');
      }
    });
  }

  // 2. Add Summary items
  if (runtime?.summary?.items && runtime.summary.items.length > 0) {
    parts.push(`### ${runtime.summary.title || 'Summary'}`);
    runtime.summary.items.forEach(item => {
      parts.push(`- **${item.label}**: ${item.value}`);
    });
    parts.push('\n');
  }

  // 3. Add Final Body
  if (finalMarkdownBody) {
    parts.push(finalMarkdownBody);
  }

  const copyContent = parts.join('\n\n').trim();
  if (!copyContent || !navigator?.clipboard?.writeText) {
    return;
  }
  await navigator.clipboard.writeText(copyContent);
}

export default function CopilotSurface({ open, onClose }: CopilotSurfaceProps) {
  const { styles } = useCopilotStyles();
  const { message: messageApi } = App.useApp();
  const { scene, context } = useSceneResolver();
  const {
    state: {
      attachedFiles,
      drawerWidth,
      inputValue,
      isBootstrapping,
      promptItems,
    },
    addAttachedFiles,
    clearAttachedFiles,
    removeAttachedFile,
    setDrawerWidth,
    setInputValue,
    setIsBootstrapping,
    setPromptItems,
  } = useCopilotSessionReducer(toPromptItems(scene));
  const resizeStateRef = React.useRef<{ startX: number; startWidth: number } | null>(null);

  const handleResizeMouseDown = React.useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    resizeStateRef.current = { startX: e.clientX, startWidth: drawerWidth };
    document.body.style.userSelect = 'none';
    document.body.style.cursor = 'col-resize';

    const handleMouseMove = (ev: MouseEvent) => {
      if (!resizeStateRef.current) {return;}
      const delta = resizeStateRef.current.startX - ev.clientX;
      const newWidth = Math.max(320, Math.min(window.innerWidth * 0.9, resizeStateRef.current.startWidth + delta));
      setDrawerWidth(newWidth);
    };

    const handleMouseUp = () => {
      resizeStateRef.current = null;
      document.body.style.userSelect = '';
      document.body.style.cursor = '';
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  }, [drawerWidth]);
  const fileInputRef = React.useRef<HTMLInputElement>(null);

  const {
    conversations,
    activeConversationKey,
    setActiveConversationKey,
    addConversation,
    setConversation,
    setConversations,
    getConversation,
  } = useXConversations({
    defaultConversations: [{ key: NEW_SESSION_KEY, label: 'New chat' }],
    defaultActiveConversationKey: NEW_SESSION_KEY,
  });

  const provider = React.useMemo(
    () =>
      new PlatformChatProvider({
        scene,
        getSceneContext: () => context,
        getSessionId: () =>
          activeConversationKey === NEW_SESSION_KEY ? undefined : activeConversationKey,
      }),
    [activeConversationKey, context, scene],
  );

  const defaultMessages = React.useCallback(
    async ({ conversationKey }: { conversationKey?: string }) => {
      if (!conversationKey || conversationKey === NEW_SESSION_KEY) {
        return [];
      }
      const response = await aiApi.getSession(conversationKey);
      const session = response?.data;
      const messages = Array.isArray(session?.messages) ? session.messages : [];
      return Promise.all(messages.map(async (message) => {
        const hydrated = await hydrateAssistantHistoryFromProjection(message);
        return {
          message: {
            ...hydrated,
            runtime: buildHistoricalPendingRuntime(hydrated.runtime, message),
          },
          status: mapHistoryMessageStatus(message.status),
        };
      }));
    },
    [],
  );

  const {
    messages,
    onRequest,
    isRequesting,
    queueRequest,
  } = useXChat<XChatMessage, XChatMessage, ChatRequest, { content: string }>({
    provider,
    conversationKey: activeConversationKey,
    defaultMessages,
    requestPlaceholder: {
      role: 'assistant',
      content: '[准备中]',
    },
    requestFallback: (_, { error, messageInfo }) => ({
      role: 'assistant',
      content: buildAssistantErrorContent(
        messageInfo?.message?.content,
        error.message || 'Request failed',
      ),
    }),
  });
  const {
    buildStepContentLoader,
    contentRef,
    handleScrollToBottom,
    markPendingSend,
    renderedMessages,
    scrollToBottom,
    showScrollBottomBtn,
  } = useCopilotStream({
    messages,
    open,
    activeConversationKey,
  });

  const bubbleRole = React.useMemo<BubbleListProps['role']>(
    () => ({
      assistant: (item) => ({
        placement: 'start',
        variant: 'borderless',
        footer: (_content: string, info) => (
          <MessageActionBar
            status={info.status}
            messageId={(item as unknown as XChatItem).extraInfo?.messageId}
            message={(item as unknown as XChatItem).extraInfo?.message}
            runtime={(item as unknown as XChatItem).extraInfo?.runtime}
            renderedMessages={renderedMessages}
            currentKey={String(item.key)}
            onRequest={onRequest}
            onSuccess={(msg) => messageApi.success(msg)}
            onError={(msg) => messageApi.error(msg)}
            copyAssistantReplyToClipboard={copyAssistantReplyToClipboard}
          />
        ),
        styles: {
          root: {
            paddingInline: 0,
            maxWidth: '100%',
          },
          content: {
            padding: 0,
            border: 'none',
            borderRadius: 0,
            background: 'transparent',
            boxShadow: 'none',
          },
          body: {
            padding: 0,
          },
        },
        contentRender: (content: string, info) => (
          <div data-message-anchor={(item as unknown as XChatItem).extraInfo?.messageId}>
            <AssistantReply
              content={content}
              runtime={(info as { extraInfo?: XChatItemExtraInfo }).extraInfo?.runtime}
              status={info.status}
              onLoadStepContent={buildStepContentLoader((info as { extraInfo?: XChatItemExtraInfo }).extraInfo?.runtime)}
            />
          </div>
        ),
      }),
      user: {
        placement: 'end',
        styles: {
          content: {
            borderRadius: 14,
            border: 'none',
            boxShadow: 'none',
          },
        },
      },
    }),
    [buildStepContentLoader, messageApi, renderedMessages, onRequest],
  );

  React.useEffect(() => {
    let cancelled = false;
    const loadSceneData = async () => {
      const sessionsResult = await aiApi.getSessions(scene);

      if (cancelled) {
        return;
      }

      try {
        const sessionItems: ConversationSummary[] = ((sessionsResult?.data as AISession[]) || []).map((item) => ({
          key: item.id,
          label: getConversationLabelFromSummary(item),
          scene: (item as AISession & { scene?: string }).scene || scene,
          updatedAt: item.updatedAt ?? item.updated_at,
        }));

        const items = sessionItems.length > 0
          ? toConversationItems(sessionItems)
          : [{ key: NEW_SESSION_KEY, label: 'New chat' }];
        setConversations(items);
        setActiveConversationKey(items[0].key);
        setPromptItems(toPromptItems(scene));
      } catch {
        if (!cancelled) {
          setConversations([{ key: NEW_SESSION_KEY, label: 'New chat' }]);
          setActiveConversationKey(NEW_SESSION_KEY);
          setPromptItems(toPromptItems(scene));
        }
      }
    };

    loadSceneData();

    return () => {
      cancelled = true;
    };
  }, [scene, setActiveConversationKey, setConversations]);

  React.useEffect(() => {
    if (!open) {
      return;
    }
    resetHistoryProjectionCache();
  }, [activeConversationKey, open]);

  const ensureSession = React.useCallback(
    async (firstMessage: string) => {
      if (activeConversationKey !== NEW_SESSION_KEY) {
        return activeConversationKey;
      }

      setIsBootstrapping(true);
      try {
        const response = await aiApi.createSession({
          title: firstMessage.slice(0, 32) || 'New chat',
          scene,
        });
        const session = response.data;
        addConversation(
          {
            key: session.id,
            label: session.title || 'New chat',
          },
          'prepend',
        );
        setActiveConversationKey(session.id);
        return session.id;
      } finally {
        setIsBootstrapping(false);
      }
    },
    [activeConversationKey, addConversation, scene, setActiveConversationKey],
  );

  const handleFileChange = React.useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || []);
    addAttachedFiles(files);
    e.target.value = '';
  }, [addAttachedFiles]);

  const removeFile = React.useCallback((index: number) => {
    removeAttachedFile(index);
  }, [removeAttachedFile]);

  const submitMessage = React.useCallback(
    async (rawMessage?: string) => {
      const message = (rawMessage ?? inputValue).trim();
      if (!message) {
        return;
      }
      const MAX_MESSAGE_LENGTH = 32768;
      if (message.length > MAX_MESSAGE_LENGTH) {
        return;
      }

      markPendingSend();

      const targetKey = await ensureSession(message);
      if (targetKey !== activeConversationKey) {
        queueRequest(targetKey, {
          message,
          sessionId: targetKey,
          scene,
          context,
        });
      } else {
        onRequest({
          message,
          sessionId: targetKey,
          scene,
          context,
        });
      }

      const currentConversation = getConversation(targetKey);
      if (currentConversation && currentConversation.label === 'New chat') {
        setConversation(targetKey, {
          ...currentConversation,
          label: message.slice(0, 24),
        });
      }
      setInputValue('');
      clearAttachedFiles();

      requestAnimationFrame(() => {
        scrollToBottom('auto', true);
      });
    },
    [
      activeConversationKey,
      context,
      ensureSession,
      getConversation,
      inputValue,
      clearAttachedFiles,
      onRequest,
      queueRequest,
      scene,
      scrollToBottom,
      setConversation,
      setInputValue,
      markPendingSend,
    ],
  );

  const isGenerating = isRequesting;

  return (
    <Drawer
      title={(
        <div className={styles.header}>
          <div className={styles.titleWrap}>
            <Space size={8}>
              <RobotOutlined />
              <Text strong className={styles.titleText}>AI 助手</Text>
              <Tag color="blue">{scene}</Tag>
            </Space>
          </div>
        </div>
      )}
      extra={(
        <Space size={10}>
          <Button
            className={styles.headerActionBtn}
            type="text"
            icon={<PlusOutlined />}
            aria-label="新建会话"
            onClick={() => setActiveConversationKey(NEW_SESSION_KEY)}
          />
          <Popover
            trigger="click"
            placement="bottomRight"
            content={(
              <div style={{ width: 280, maxHeight: 360, overflow: 'auto' }}>
                <Conversations
                  items={conversations}
                  activeKey={activeConversationKey}
                  onActiveChange={setActiveConversationKey}
                />
              </div>
            )}
          >
            <Button
              className={styles.headerActionBtn}
              type="text"
              icon={<CommentOutlined />}
              aria-label="查看历史会话"
            />
          </Popover>
        </Space>
      )}
      placement="right"
      size={drawerWidth}
      open={open}
      onClose={onClose}
      styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', height: '100%', position: 'relative' } }}
      destroyOnHidden={false}
    >
      <div className={styles.resizeHandle} onMouseDown={handleResizeMouseDown} />
      <div className={styles.surface}>
        <div ref={contentRef} className={styles.content} data-testid="copilot-scroll-container">
          {messages.length === 0 ? (
            <div className={styles.emptyState}>
              <Welcome
                variant="borderless"
                title="你好，我是您的智能运维助手!"
                description="我会结合你所在页面的上下文，给出更贴近业务的分析与建议。"
              />
              <Prompts
                title="快捷提问"
                items={promptItems}
                onItemClick={(info) => submitMessage(String(info?.data?.description || info?.data?.label || ''))}
              />
            </div>
          ) : (
            <div className={styles.chatCard}>
              <Bubble.List
                items={renderedMessages.map((item) => ({
                  key: item.renderKey,
                  role: item.message.role,
                  content: item.message.role === 'user'
                    ? <div data-message-anchor={item.renderKey}>{item.message.content}</div>
                    : item.message.content,
                  loading: item.status === 'loading' && !item.message.content,
                  status: toBubbleStatus(item.status),
                  extraInfo: {
                    messageId: item.renderKey,
                    runtime: item.message.runtime,
                    message: item.message,
                  },
                }))}
                role={bubbleRole}
              />
            </div>
          )}
        </div>

        {messages.length > 0 && showScrollBottomBtn && (
          <Button
            className={`${styles.scrollBottomBtn}${isGenerating ? ` ${styles.scrollBottomBtnLoading}` : ''}`}
            type="default"
            shape="circle"
            icon={<VerticalAlignBottomOutlined />}
            onClick={handleScrollToBottom}
            aria-label="快速回到底部"
            title={isGenerating ? '正在生成，点击快速回到底部' : '快速回到底部'}
          />
        )}

        <div className={styles.senderWrap}>
          {attachedFiles.length > 0 && (
            <div className={styles.fileList}>
              {attachedFiles.map((file, index) => (
                <div key={`${file.name}-${file.size}-${file.lastModified}`} className={styles.fileItem}>
                  <PaperClipOutlined style={{ fontSize: 12, flexShrink: 0 }} />
                  <span className={styles.fileName}>{file.name}</span>
                  <Button
                    type="text"
                    size="small"
                    icon={<CloseOutlined style={{ fontSize: 10 }} />}
                    onClick={() => removeFile(index)}
                    style={{ width: 18, height: 18, minWidth: 18, padding: 0, flexShrink: 0 }}
                  />
                </div>
              ))}
            </div>
          )}
          <input
            ref={fileInputRef}
            type="file"
            multiple
            style={{ display: 'none' }}
            onChange={handleFileChange}
          />
          <div className={styles.senderRow}>

            <div className={styles.senderFlex}>
              <Sender
                value={inputValue}
                onChange={setInputValue}
                prefix={<Button
                  className={styles.attachBtn}
                  type="text"
                  icon={<PaperClipOutlined style={{ fontSize: 18 }} />}
                  onClick={() => fileInputRef.current?.click()}
                  title="添加附件"
                />}
                onSubmit={(value) => submitMessage(value)}
                loading={isRequesting || isBootstrapping}
                placeholder="提问或输入 / 使用技能"
                allowSpeech
              />
            </div>
          </div>
        </div>
      </div>
    </Drawer>
  );
}
