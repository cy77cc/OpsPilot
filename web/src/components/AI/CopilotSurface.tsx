import React from 'react';
import { RobotOutlined, CommentOutlined, PlusOutlined } from '@ant-design/icons';
import { Bubble, Conversations, Prompts, Sender, Think, Welcome } from '@ant-design/x';
import type { BubbleListProps, ConversationItemType, PromptsItemType } from '@ant-design/x';
import XMarkdown from '@ant-design/x-markdown';
import { useXChat, useXConversations } from '@ant-design/x-sdk';
import { Button, Drawer, Popover, Space, Tag, Typography } from 'antd';
import { createStyles } from 'antd-style';
import { useLocation } from 'react-router-dom';
import { aiApi } from '../../api/modules/ai';
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
    background: linear-gradient(180deg, ${token.colorBgLayout} 0%, ${token.colorBgContainer} 24%);
  `,
  header: css`
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 16px 12px;
    border-bottom: 1px solid ${token.colorBorderSecondary};
    background: ${token.colorBgContainer};
  `,
  titleWrap: css`
    display: flex;
    flex-direction: column;
    gap: 4px;
  `,
  content: css`
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 16px;
  `,
  senderWrap: css`
    border-top: 1px solid ${token.colorBorderSecondary};
    background: ${token.colorBgContainer};
    padding: 12px 16px 16px;
  `,
  emptyState: css`
    display: flex;
    flex-direction: column;
    gap: 16px;
  `,
  markdown: css`
    line-height: 1.65;

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

export default function CopilotSurface({ open, onClose }: CopilotSurfaceProps) {
  const { styles } = useCopilotStyles();
  const location = useLocation();
  const { scene, context } = React.useMemo(
    () => resolveScene(location.pathname),
    [location.pathname],
  );
  const [inputValue, setInputValue] = React.useState('');
  const [promptItems, setPromptItems] = React.useState<PromptsItemType[]>(toPromptItems(scene));
  const [isBootstrapping, setIsBootstrapping] = React.useState(false);

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
        message: {
          role: (message.role === 'assistant' ? 'assistant' : 'user') as 'assistant' | 'user',
          content: message.content || '',
        },
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
      content: 'Thinking...',
    },
    requestFallback: (_, { error }) => ({
      role: 'assistant',
      content: error.message || 'Request failed',
    }),
  });

  const bubbleRole = React.useMemo<BubbleListProps['role']>(
    () => ({
      assistant: {
        placement: 'start',
        contentRender: (content: string, info) => (
          <div className={styles.markdown}>
            <XMarkdown
              content={content}
              streaming={{
                hasNextChunk: info.status === 'loading' || info.status === 'updating',
                enableAnimation: true,
                animationConfig: {
                  fadeDuration: 180,
                  easing: 'ease-out',
                },
              }}
              components={{
                think: ({ children }: any) => (
                  <Think title="Thinking" loading={false}>
                    {children}
                  </Think>
                ),
                table: ({ children, ...props }: any) => (
                  <div style={{ overflowX: 'auto' }}>
                    <table {...props}>{children}</table>
                  </div>
                ),
              }}
            />
          </div>
        ),
      },
      user: {
        placement: 'end',
      },
    }),
    [styles.markdown],
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
      title={null}
      placement="right"
      size="large"
      open={open}
      onClose={onClose}
      styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', height: '100%' } }}
      destroyOnClose={false}
    >
      <div className={styles.surface}>
        <div className={styles.header}>
          <div className={styles.titleWrap}>
            <Space size={8}>
              <RobotOutlined />
              <Text strong>AI Copilot</Text>
              <Tag color="blue">{scene}</Tag>
            </Space>
            <Text type="secondary">
              Uses the current page context to improve suggestions and tool routing.
            </Text>
          </div>
          <Space size={8}>
            <Button
              type="text"
              icon={<PlusOutlined />}
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
              <Button type="text" icon={<CommentOutlined />} />
            </Popover>
          </Space>
        </div>

        <div className={styles.content}>
          {messages.length === 0 ? (
            <div className={styles.emptyState}>
              <Welcome
                variant="borderless"
                title="Your in-context operations copilot"
                description="Ask about the current page, resource, or issue. Scene context stays hidden from the visible transcript."
              />
              <Prompts
                title="Scene-aware starters"
                items={promptItems}
                onItemClick={(info) => submitMessage(String(info?.data?.description || info?.data?.label || ''))}
              />
            </div>
          ) : (
            <Bubble.List
              autoScroll
              items={messages.map((item) => ({
                key: item.id,
                role: item.message.role,
                content: item.message.content,
                loading: item.status === 'loading' && !item.message.content,
                status: item.status,
              }))}
              role={bubbleRole}
            />
          )}
        </div>

        <div className={styles.senderWrap}>
          <Sender
            value={inputValue}
            onChange={setInputValue}
            onSubmit={(value) => submitMessage(value)}
            loading={isRequesting || isBootstrapping}
            placeholder="Ask about the current host, cluster, service, or task..."
          />
        </div>
      </div>
    </Drawer>
  );
}
