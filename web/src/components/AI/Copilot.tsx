/**
 * Copilot 组件（简化版）
 * 基本聊天组件，使用 @ant-design/x 实现
 */
import React, { useState, useRef, useCallback, useMemo } from 'react';
import {
  CloseOutlined,
  CommentOutlined,
  GlobalOutlined,
  EnvironmentOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import { Bubble, Sender, Welcome, Prompts } from '@ant-design/x';
import { XMarkdown } from '@ant-design/x-markdown';
import type { BubbleListRef } from '@ant-design/x/es/bubble';
import { Button, Select, Space, Tooltip, theme, Skeleton } from 'antd';
import { aiApi } from '../../api/modules/ai';
import { getSceneLabel } from './constants/sceneMapping';
import type { ChatMessage, MessageRole } from './types';
import type { SceneOption } from './hooks/useAutoScene';
import { useScenePrompts } from './hooks/useScenePrompts';

const { useToken } = theme;

interface CopilotProps {
  open?: boolean;
  onClose?: () => void;
  scene: string;
  selectValue?: string;
  onSceneChange?: (scene: string) => void;
  availableScenes?: SceneOption[];
  isAuto?: boolean;
}

/**
 * Copilot 主组件
 */
export const Copilot: React.FC<CopilotProps> = ({
  open = true,
  onClose,
  scene,
  selectValue,
  onSceneChange,
  availableScenes = [{ key: 'global', label: '全局助手' }],
  isAuto = true,
}) => {
  const { token } = useToken();
  const listRef = useRef<BubbleListRef>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  // 状态
  const [inputValue, setInputValue] = useState('');
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string | undefined>();

  // 场景提示词
  const { prompts: scenePrompts } = useScenePrompts({ scene, enabled: open });

  // 发送消息
  const handleSubmit = useCallback(async (val: string) => {
    if (!val.trim() || isLoading) return;
    const trimmed = val.trim();

    // 添加用户消息
    const userMessage: ChatMessage = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: trimmed,
      createdAt: new Date().toISOString(),
    };

    // 创建助手消息占位
    const assistantId = `assistant-${Date.now()}`;
    const assistantMessage: ChatMessage = {
      id: assistantId,
      role: 'assistant',
      content: '',
      createdAt: new Date().toISOString(),
    };

    setMessages(prev => [...prev, userMessage, assistantMessage]);
    setIsLoading(true);

    let assistantContent = '';
    abortControllerRef.current = new AbortController();

    try {
      await aiApi.chatStream(
        { sessionId, message: trimmed, context: { scene } },
        {
          onMeta: (data) => {
            if (data.sessionId) setSessionId(data.sessionId);
          },
          onDelta: (data) => {
            assistantContent += data.contentChunk || '';
            setMessages(prev => prev.map(m =>
              m.id === assistantId
                ? { ...m, content: assistantContent }
                : m
            ));
          },
          onDone: () => {
            setIsLoading(false);
          },
          onError: (data) => {
            setMessages(prev => prev.map(m =>
              m.id === assistantId
                ? { ...m, content: data.message || '请求失败，请稍后重试' }
                : m
            ));
            setIsLoading(false);
          },
        },
        abortControllerRef.current.signal,
      );
    } catch (error) {
      if ((error as Error).name !== 'AbortError') {
        setMessages(prev => prev.map(m =>
          m.id === assistantId
            ? { ...m, content: '请求失败，请稍后重试' }
            : m
        ));
      }
      setIsLoading(false);
    }
  }, [isLoading, scene, sessionId]);

  // 中止请求
  const handleAbort = useCallback(() => {
    abortControllerRef.current?.abort();
    setIsLoading(false);
  }, []);

  // 新建会话
  const handleNewConversation = useCallback(() => {
    setMessages([]);
    setSessionId(undefined);
  }, []);

  // 场景选择器
  const sceneSelector = useMemo(() => {
    const sceneLabel = getSceneLabel(scene);
    const displayValue = selectValue || scene;

    if (!onSceneChange || availableScenes.length <= 1) {
      return (
        <span style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 4,
          padding: '2px 8px',
          background: token.colorPrimaryBg,
          borderRadius: token.borderRadiusSM,
          fontSize: 12,
          color: token.colorPrimary,
        }}>
          {scene === 'global' ? <GlobalOutlined /> : <EnvironmentOutlined />}
          {isAuto ? `自动: ${sceneLabel}` : sceneLabel}
        </span>
      );
    }

    return (
      <Select
        value={displayValue}
        onChange={onSceneChange}
        size="small"
        style={{ width: 160 }}
      >
        {availableScenes.map(s => (
          <Select.Option key={s.key} value={s.key}>
            <Space>
              {s.key === 'global' ? <GlobalOutlined /> : <EnvironmentOutlined />}
              {s.label}
            </Space>
          </Select.Option>
        ))}
      </Select>
    );
  }, [scene, selectValue, isAuto, onSceneChange, availableScenes, token]);

  // 角色配置
  const role = useMemo(() => ({
    assistant: { placement: 'start' as const },
    user: { placement: 'end' as const },
  }), []);

  if (!open) return null;

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      height: '100%',
      background: token.colorBgContainer,
      color: token.colorText,
    }}>
      {/* 头部 */}
      <div style={{
        height: 52,
        boxSizing: 'border-box',
        borderBottom: `1px solid ${token.colorBorder}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '0 12px 0 16px',
        flexShrink: 0,
      }}>
        <div style={{
          fontWeight: 600,
          fontSize: 15,
          display: 'flex',
          alignItems: 'center',
          gap: 8,
        }}>
          {sceneSelector}
          <span>AI Copilot</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <Tooltip title="新建对话">
            <Button
              type="text"
              icon={<PlusOutlined />}
              onClick={handleNewConversation}
            />
          </Tooltip>
          <Tooltip title="会话列表">
            <Button type="text" icon={<CommentOutlined />} />
          </Tooltip>
          {onClose && (
            <Tooltip title="关闭">
              <Button
                type="text"
                icon={<CloseOutlined />}
                onClick={onClose}
              />
            </Tooltip>
          )}
        </div>
      </div>

      {/* 消息列表 */}
      <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column', minHeight: 0 }}>
        {messages.length > 0 ? (
          <Bubble.List
            ref={listRef}
            style={{ paddingInline: 16, height: '100%' }}
            items={messages.map(m => ({
              key: m.id,
              content: m.role === 'assistant'
                ? <div className="ai-markdown-content"><XMarkdown>{m.content}</XMarkdown></div>
                : m.content,
              role: m.role,
              loading: m.role === 'assistant' && isLoading && !m.content,
            }))}
            role={role}
          />
        ) : (
          <>
            <Welcome
              variant="borderless"
              title="👋 你好，我是 AI Copilot"
              description="我可以帮助你进行部署管理、服务治理、监控运维等操作"
              style={{
                margin: 16,
                padding: 16,
                borderRadius: 8,
                background: token.colorBgTextHover,
              }}
            />
            <Prompts
              vertical
              title="我可以帮你："
              items={scenePrompts.length > 0 ? scenePrompts : [{ key: 'default', description: '有什么可以帮助你的？' }]}
              onItemClick={(info) => handleSubmit(info?.data?.description as string)}
              style={{ margin: '0 16px 16px' }}
            />
          </>
        )}
      </div>

      {/* 输入框 */}
      <div style={{
        padding: '12px 16px',
        borderTop: `1px solid ${token.colorBorder}`,
        background: token.colorBgContainer,
        flexShrink: 0,
      }}>
        <Sender
          loading={isLoading}
          value={inputValue}
          onChange={setInputValue}
          onSubmit={() => {
            handleSubmit(inputValue);
            setInputValue('');
          }}
          onCancel={handleAbort}
          placeholder="输入消息..."
          allowSpeech
        />
      </div>
    </div>
  );
};

export default Copilot;
