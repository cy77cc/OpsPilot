import React from 'react';
import {
  RobotOutlined,
  CommentOutlined,
  PlusOutlined,
  PaperClipOutlined,
  CloseOutlined,
  VerticalAlignBottomOutlined,
  ReloadOutlined,
  CopyOutlined,
  LikeOutlined,
  DislikeOutlined,
} from '@ant-design/icons';
import { Bubble, Conversations, Prompts, Sender, Welcome } from '@ant-design/x';
import type { BubbleListProps, ConversationItemType, PromptsItemType } from '@ant-design/x';
import { useXChat, useXConversations } from '@ant-design/x-sdk';
import { Button, Drawer, Popover, Space, Tag, Typography } from 'antd';
import { createStyles } from 'antd-style';
import { useLocation } from 'react-router-dom';
import { aiApi } from '../../api/modules/ai';
import { AssistantReply } from './AssistantReply';
import { hydrateAssistantHistoryMessage } from './historyRuntime';
import { PlatformChatProvider } from './providers';
import type { ChatRequest, ConversationSummary, SceneContext, XChatMessage } from './types';

const { Text } = Typography;

const NEW_SESSION_KEY = '__new__';

const SCENE_FALLBACK_PROMPTS: Record<string, PromptsItemType[]> = {
  host: [
    { key: 'host-health', label: '诊断主机健康', description: '帮我检查当前主机的健康状态和风险点' },
    { key: 'host-services', label: '检查服务状态', description: '分析这台主机上的关键服务状态' },
  ],
  cluster: [
    { key: 'cluster-health', label: '诊断集群健康', description: '帮我分析当前集群的健康状态和关键异常' },
    { key: 'cluster-capacity', label: '评估集群容量', description: '评估当前集群的资源容量与潜在瓶颈' },
  ],
  service: [
    { key: 'service-release', label: '发布影响分析', description: '分析当前服务最近发布的潜在影响' },
    { key: 'service-deps', label: '服务依赖梳理', description: '梳理这个服务的依赖与潜在故障点' },
  ],
  k8s: [
    { key: 'k8s-workload', label: '工作负载巡检', description: '检查当前 Kubernetes 工作负载的异常情况' },
    { key: 'k8s-events', label: '事件总结', description: '总结当前 Kubernetes 事件流里的异常信号' },
  ],
  ai: [
    { key: 'ai-general', label: '开始提问', description: '描述你当前遇到的问题或你想完成的操作' },
  ],
};

const useCopilotStyles = createStyles(({ token, css }) => ({
  surface: css`
    display: flex;
    flex-direction: column;
    height: 100%;
  `,
  header: css`
    display: flex;
    align-items: center;
  `,
  titleWrap: css`
    display: flex;
    flex-direction: column;
    gap: 4px;
  `,
  titleText: css`
    font-size: 18px;
    line-height: 26px;
    font-weight: 600;
    color: #111827;
  `,
  subtitleText: css`
    font-size: 13px;
    line-height: 20px;
    color: #6b7280;
  `,
  content: css`
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 16px;
    background: transparent;
  `,
  contentToolbar: css`
    display: flex;
    justify-content: flex-end;
    margin-bottom: 12px;
  `,
  headerActionBtn: css`
    width: 36px;
    height: 36px;
    font-size: 18px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  `,
  chatCard: css`
    background: transparent;
    border: none;
    border-radius: 0;
    padding: 0;
  `,
  senderWrap: css`
    padding: 12px 16px 16px;
  `,
  senderRow: css`
    display: flex;
    align-items: flex-end;
    gap: 6px;
  `,
  attachBtn: css`
    flex-shrink: 0;
    width: 36px;
    height: 36px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  `,
  senderFlex: css`
    flex: 1;
    min-width: 0;
  `,
  fileList: css`
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
    margin-bottom: 8px;
  `,
  fileItem: css`
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 3px 6px 3px 8px;
    border-radius: 6px;
    background: ${token.colorFillSecondary};
    border: 1px solid ${token.colorBorderSecondary};
    font-size: 12px;
    color: ${token.colorText};
    max-width: 220px;
  `,
  fileName: css`
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
  `,
  emptyState: css`
    display: flex;
    flex-direction: column;
    gap: 16px;
  `,
  resizeHandle: css`
    position: absolute;
    top: 0;
    left: 0;
    width: 5px;
    height: 100%;
    cursor: col-resize;
    z-index: 100;
    background: transparent;
    border-left: 2px solid transparent;
    transition: border-color 0.15s;

    &:hover {
      border-left-color: ${token.colorPrimary};
    }
  `,
  scrollBottomBtn: css`
    position: absolute;
    right: 24px;
    bottom: 92px;
    z-index: 120;
    box-shadow: 0 8px 20px rgba(17, 24, 39, 0.14);
  `,
}));

function resolveScene(pathname: string): { scene: string; context: SceneContext } {
  const normalized = pathname || '/';
  const segments = normalized.split('/').filter(Boolean);

  if (normalized.startsWith('/deployment/infrastructure/hosts') || normalized.startsWith('/hosts')) {
    return {
      scene: 'host',
      context: {
        route: normalized,
        resourceType: 'host',
        resourceId: segments[segments.length - 1],
      },
    };
  }

  if (normalized.startsWith('/k8s-legacy')) {
    return {
      scene: 'k8s',
      context: {
        route: normalized,
        resourceType: 'k8s',
      },
    };
  }

  if (normalized.startsWith('/deployment/infrastructure/clusters') || normalized.startsWith('/k8s')) {
    return {
      scene: 'cluster',
      context: {
        route: normalized,
        resourceType: 'cluster',
        resourceId: segments[segments.length - 1],
      },
    };
  }

  if (normalized.startsWith('/services')) {
    return {
      scene: 'service',
      context: {
        route: normalized,
        resourceType: 'service',
        resourceId: segments[1],
      },
    };
  }

  return {
    scene: 'ai',
    context: {
      route: normalized,
      resourceType: 'page',
    },
  };
}

function toConversationItems(items: ConversationSummary[]): ConversationItemType[] {
  return items.map((item) => ({
    key: item.key,
    label: item.label,
  }));
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

export default function CopilotSurface({ open, onClose }: CopilotSurfaceProps) {
  const { styles } = useCopilotStyles();
  const contentRef = React.useRef<HTMLDivElement>(null);
  const shouldScrollToBottomRef = React.useRef(true);
  const [showScrollBottomBtn, setShowScrollBottomBtn] = React.useState(false);
  const location = useLocation();
  const { scene, context } = React.useMemo(
    () => resolveScene(location.pathname),
    [location.pathname],
  );
  const [drawerWidth, setDrawerWidth] = React.useState(736);
  const resizeStateRef = React.useRef<{ startX: number; startWidth: number } | null>(null);

  const handleResizeMouseDown = React.useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    resizeStateRef.current = { startX: e.clientX, startWidth: drawerWidth };
    document.body.style.userSelect = 'none';
    document.body.style.cursor = 'col-resize';

    const handleMouseMove = (ev: MouseEvent) => {
      if (!resizeStateRef.current) return;
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

  const [inputValue, setInputValue] = React.useState('');
  const [promptItems, setPromptItems] = React.useState<PromptsItemType[]>(toPromptItems(scene));
  const [isBootstrapping, setIsBootstrapping] = React.useState(false);
  const [attachedFiles, setAttachedFiles] = React.useState<File[]>([]);
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
      return messages.map((message) => ({
        message: hydrateAssistantHistoryMessage(message, session?.turns || []),
        status: (message.status === 'done' ? 'success' : 'loading') as 'success' | 'loading',
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

  React.useEffect(() => {
    shouldScrollToBottomRef.current = true;
  }, [activeConversationKey, open]);

  React.useEffect(() => {
    if (!open || messages.length === 0 || !shouldScrollToBottomRef.current) {
      return;
    }

    const frame = requestAnimationFrame(() => {
      const el = contentRef.current;
      if (!el) {
        return;
      }

      el.scrollTo({ top: el.scrollHeight, behavior: 'auto' });
      shouldScrollToBottomRef.current = false;
    });

    return () => cancelAnimationFrame(frame);
  }, [messages.length, open]);

  React.useEffect(() => {
    const el = contentRef.current;
    if (!el || !open) {
      return;
    }

    const updateBtnVisible = () => {
      const distanceToBottom = el.scrollHeight - el.scrollTop - el.clientHeight;
      setShowScrollBottomBtn(distanceToBottom > 120);
    };

    updateBtnVisible();
    el.addEventListener('scroll', updateBtnVisible, { passive: true });

    return () => {
      el.removeEventListener('scroll', updateBtnVisible);
    };
  }, [messages.length, open]);

  const handleScrollToBottom = React.useCallback(() => {
    const el = contentRef.current;
    if (!el) {
      return;
    }
    el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' });
  }, []);

  const bubbleRole = React.useMemo<BubbleListProps['role']>(
    () => ({
      assistant: {
        placement: 'start',
        variant: 'borderless',
        footer: (
          <div style={{ display: 'flex' }}>
            <Button type="text" size="small" icon={<CopyOutlined />} />
            <Button type="text" size="small" icon={<LikeOutlined />} />
            <Button type="text" size="small" icon={<DislikeOutlined />} />
            <Button type="text" size="small" icon={<ReloadOutlined />} />
          </div>
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
          <AssistantReply
            content={content}
            runtime={(info as any).extraInfo?.runtime}
            status={info.status}
          />
        ),
      },
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
    [],
  );

  React.useEffect(() => {
    let cancelled = false;
    const loadSceneData = async () => {
      const [sessionsResult, promptsResult] = await Promise.allSettled([
        aiApi.getSessions(scene),
        aiApi.getScenePrompts(scene),
      ]);

      if (cancelled) {
        return;
      }

      try {
        const sessionsResp = sessionsResult.status === 'fulfilled'
          ? sessionsResult.value
          : { data: [] };
        const promptsResp = promptsResult.status === 'fulfilled'
          ? promptsResult.value
          : { data: { prompts: [] } };

        const sessionItems: ConversationSummary[] = ((sessionsResp as any)?.data || []).map((item: any) => ({
          key: item.id,
          label: item.title || 'New chat',
          scene: item.scene || scene,
          updatedAt: item.updatedAt || item.updated_at,
        }));

        const items = sessionItems.length > 0
          ? toConversationItems(sessionItems)
          : [{ key: NEW_SESSION_KEY, label: 'New chat' }];
        setConversations(items);
        setActiveConversationKey(items[0].key);
        setPromptItems(toPromptItems(scene, (promptsResp as any)?.data?.prompts));
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
    setAttachedFiles((prev) => [...prev, ...files]);
    e.target.value = '';
  }, []);

  const removeFile = React.useCallback((index: number) => {
    setAttachedFiles((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const submitMessage = React.useCallback(
    async (rawMessage?: string) => {
      const message = (rawMessage ?? inputValue).trim();
      if (!message) {
        return;
      }

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
      setAttachedFiles([]);
    },
    [
      activeConversationKey,
      context,
      ensureSession,
      getConversation,
      inputValue,
      onRequest,
      queueRequest,
      scene,
      setConversation,
    ],
  );

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
        <div ref={contentRef} className={styles.content}>
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
                autoScroll
                items={messages.map((item) => ({
                  key: item.id,
                  role: item.message.role,
                  content: item.message.content,
                  loading: item.status === 'loading' && !item.message.content,
                  status: item.status,
                  extraInfo: {
                    runtime: item.message.runtime,
                  },
                }))}
                role={bubbleRole}
              />
            </div>
          )}
        </div>

        {messages.length > 0 && showScrollBottomBtn && (
          <Button
            className={styles.scrollBottomBtn}
            type="primary"
            shape="circle"
            icon={<VerticalAlignBottomOutlined />}
            onClick={handleScrollToBottom}
            aria-label="快速回到底部"
            title="快速回到底部"
          />
        )}

        <div className={styles.senderWrap}>
          {attachedFiles.length > 0 && (
            <div className={styles.fileList}>
              {attachedFiles.map((file, index) => (
                <div key={index} className={styles.fileItem}>
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
