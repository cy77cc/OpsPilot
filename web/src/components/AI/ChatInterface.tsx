import React, { useEffect, useRef, useState } from 'react';
import { Input, Button, Avatar, Space, Typography, Card, Tag, Collapse, Alert, Tooltip } from 'antd';
import { SendOutlined, MessageOutlined, ToolOutlined, BulbOutlined, WarningOutlined, ArrowDownOutlined, PlusOutlined, HistoryOutlined, PushpinOutlined, DeleteOutlined, DownloadOutlined, CopyOutlined, FileMarkdownOutlined, CodeOutlined, EditOutlined } from '@ant-design/icons';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { Api } from '../../api';
import type { AIMessage, AISession, EmbeddedRecommendation, ToolTrace } from '../../api';

const { Text, Paragraph } = Typography;

interface ChatInterfaceProps {
  sessionId?: string;
  scene?: string;
  onSessionCreate?: (session: AISession) => void;
  onSessionUpdate?: (session: AISession) => void;
  className?: string;
}

type StreamState = 'idle' | 'running' | 'timeout' | 'done' | 'error';
type MessagePhase = 'awaiting_first_token' | 'streaming' | 'done' | 'error';
type LocalMessage = AIMessage & { turnId?: string; phase?: MessagePhase };

const renderMarkdown = (content: string) => (
  <div className="ai-markdown-content">
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        table({ children }) {
          return (
            <div className="ai-markdown-table-wrap">
              <table className="ai-markdown-table">{children}</table>
            </div>
          );
        },
        thead({ children }) {
          return <thead className="ai-markdown-thead">{children}</thead>;
        },
        tbody({ children }) {
          return <tbody className="ai-markdown-tbody">{children}</tbody>;
        },
        tr({ children }) {
          return <tr className="ai-markdown-tr">{children}</tr>;
        },
        th({ children }) {
          return <th className="ai-markdown-th">{children}</th>;
        },
        td({ children }) {
          return <td className="ai-markdown-td">{children}</td>;
        },
        code({ className, children, ...props }) {
          const match = /language-(\w+)/.exec(className || '');
          const codeText = String(children).replace(/\n$/, '');
          if (match) {
            return (
              <SyntaxHighlighter
                style={oneLight}
                language={match[1]}
                PreTag="div"
                customStyle={{ margin: '8px 0', borderRadius: 6 }}
              >
                {codeText}
              </SyntaxHighlighter>
            );
          }
          return (
            <code className={className} style={{ background: '#f5f6f7', padding: '2px 4px', borderRadius: 4 }} {...props}>
              {children}
            </code>
          );
        },
      }}
    >
      {content}
    </ReactMarkdown>
  </div>
);

const ChatInterface: React.FC<ChatInterfaceProps> = ({
  sessionId,
  scene = 'global',
  onSessionCreate,
  onSessionUpdate,
  className,
}) => {
  const [messages, setMessages] = useState<LocalMessage[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [loading, setLoading] = useState(false);
  const [streamState, setStreamState] = useState<StreamState>('idle');
  const [streamError, setStreamError] = useState('');
  const [streamNotice, setStreamNotice] = useState('');
  const [currentSession, setCurrentSession] = useState<AISession | null>(null);
  const [lastPrompt, setLastPrompt] = useState('');
  const [traceRawVisible, setTraceRawVisible] = useState<Record<string, boolean>>({});
  const [followupLoadingId, setFollowupLoadingId] = useState('');
  const [activeAssistantMessageId, setActiveAssistantMessageId] = useState('');
  const [recRevealMap, setRecRevealMap] = useState<Record<string, number>>({});
  const [showScrollBottom, setShowScrollBottom] = useState(false);
  const [sessionList, setSessionList] = useState<AISession[]>([]);
  const [sessionLoading, setSessionLoading] = useState(false);
  const [sessionKeyword, setSessionKeyword] = useState('');
  const [pinnedSessionIds, setPinnedSessionIds] = useState<string[]>([]);
  const [activeAnchorId, setActiveAnchorId] = useState('');
  const [pendingNewSessionId, setPendingNewSessionId] = useState('');

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const shouldAutoScrollRef = useRef(true);
  const revealTimerRef = useRef<Record<string, number[]>>({});

  const clearRecommendationReveal = (messageId: string) => {
    const timers = revealTimerRef.current[messageId] || [];
    timers.forEach((timer) => window.clearTimeout(timer));
    delete revealTimerRef.current[messageId];
  };

  const startRecommendationReveal = (messageId: string, total: number) => {
    clearRecommendationReveal(messageId);
    if (total <= 0) {
      setRecRevealMap((prev) => ({ ...prev, [messageId]: 0 }));
      return;
    }
    setRecRevealMap((prev) => ({ ...prev, [messageId]: 0 }));
    const timers: number[] = [];
    for (let i = 0; i < total; i++) {
      const timer = window.setTimeout(() => {
        setRecRevealMap((prev) => ({ ...prev, [messageId]: i + 1 }));
      }, 120 * (i + 1));
      timers.push(timer);
    }
    revealTimerRef.current[messageId] = timers;
  };

  useEffect(() => {
    return () => {
      Object.keys(revealTimerRef.current).forEach((key) => clearRecommendationReveal(key));
    };
  }, []);

  // 记录已加载的 scene，避免重复加载
  const loadedSceneRef = useRef<string | null>(null);

  const loadSession = async (forceReload = false) => {
    // 如果 scene 没变且不是强制刷新，跳过
    if (!forceReload && loadedSceneRef.current === scene && !sessionId) {
      return;
    }
    loadedSceneRef.current = scene;

    try {
      if (sessionId) {
        const response = await Api.ai.getSessionDetail(sessionId);
        setMessages(response.data.messages as LocalMessage[]);
        setCurrentSession(response.data);
        return;
      }
      const res = await Api.ai.getCurrentSession(scene);
      if (res.data) {
        setMessages((res.data.messages || []) as LocalMessage[]);
        setCurrentSession(res.data);
      } else {
        setMessages([]);
        setCurrentSession(null);
      }
    } catch (error) {
      console.error('加载会话失败:', error);
    }
  };

  const loadSessions = async () => {
    if (sessionId) return;
    setSessionLoading(true);
    try {
      const res = await Api.ai.getSessions(scene);
      const list = (res.data || []).slice().sort((a, b) => {
        return new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime();
      });
      setSessionList(list);
    } catch (error) {
      console.error('加载会话列表失败:', error);
    } finally {
      setSessionLoading(false);
    }
  };

  useEffect(() => {
    loadSession(true);
    void loadSessions();
  }, [sessionId, scene]);

  useEffect(() => {
    const key = `ai:pinned:sessions:${scene}`;
    try {
      const raw = localStorage.getItem(key);
      const parsed = raw ? JSON.parse(raw) : [];
      if (Array.isArray(parsed)) {
        setPinnedSessionIds(parsed.filter((id) => typeof id === 'string'));
      }
    } catch {
      setPinnedSessionIds([]);
    }
  }, [scene]);

  useEffect(() => {
    if (shouldAutoScrollRef.current) {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages, streamState]);

  const handleScroll = () => {
    const el = scrollContainerRef.current;
    if (!el) return;
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80;
    shouldAutoScrollRef.current = nearBottom;
    setShowScrollBottom(!nearBottom);
    const anchors = messages
      .filter((msg) => msg.role === 'user')
      .map((msg) => ({ id: msg.id, top: (document.getElementById(`msg-${msg.id}`)?.offsetTop || 0) }));
    const current = anchors
      .filter((item) => item.top <= el.scrollTop + 90)
      .sort((a, b) => b.top - a.top)[0];
    if (current?.id) {
      setActiveAnchorId(current.id);
    }
  };

  const scrollToBottom = () => {
    shouldAutoScrollRef.current = true;
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    setShowScrollBottom(false);
  };

  const patchAssistantMessage = (assistantID: string, turnID: string | undefined, patch: (item: LocalMessage) => LocalMessage) => {
    setMessages((prev) => {
      let matched = false;
      const next = prev.map((item) => {
        if (item.role !== 'assistant') {
          return item;
        }
        const byTurn = turnID && item.turnId === turnID;
        const byId = item.id === assistantID;
        if (!byTurn && !byId) {
          return item;
        }
        matched = true;
        return patch(item);
      });
      return matched ? next : prev;
    });
  };

  const markStreaming = (assistantID: string, turnID?: string) => {
    patchAssistantMessage(assistantID, turnID, (item) => {
      if (item.phase === 'awaiting_first_token' || !item.phase) {
        return { ...item, phase: 'streaming', turnId: turnID || item.turnId };
      }
      return { ...item, turnId: turnID || item.turnId };
    });
  };

  const attachTraceToAssistant = (assistantID: string, turnID: string | undefined, trace: ToolTrace) => {
    patchAssistantMessage(assistantID, turnID, (item) => ({
      ...item,
      turnId: turnID || item.turnId,
      phase: item.phase === 'awaiting_first_token' ? 'streaming' : item.phase,
      traces: [...(item.traces || []), trace],
    }));
  };

  const sendMessage = async (messageText: string) => {
    if (!messageText.trim() || loading) return;

    const previousSessionID = currentSession?.id;
    const userMessage: LocalMessage = {
      id: `msg-${Date.now()}`,
      role: 'user',
      content: messageText,
      timestamp: new Date().toISOString(),
    };
    const assistantMessageID = `msg-${Date.now()}-assistant`;
    const assistantPlaceholder: LocalMessage = {
      id: assistantMessageID,
      role: 'assistant',
      content: '',
      thinking: '',
      traces: [],
      timestamp: new Date().toISOString(),
      phase: 'awaiting_first_token',
    };

    setMessages((prev) => [...prev, userMessage, assistantPlaceholder]);
    setInputValue('');
    setLoading(true);
    setStreamState('running');
    setStreamError('');
    setStreamNotice('');
    setLastPrompt(messageText);
    setActiveAssistantMessageId(assistantMessageID);

    let latestSession: AISession | undefined;
    let activeTurnID = '';

    const requestSessionID = pendingNewSessionId || currentSession?.id;

    try {
      await Api.ai.chatStream(
        {
          sessionId: requestSessionID,
          message: messageText,
          context: { scene },
        },
        {
          onMeta: (meta) => {
            activeTurnID = meta.turn_id || activeTurnID;
            if (requestSessionID && meta.sessionId === requestSessionID) {
              setPendingNewSessionId('');
            }
            setCurrentSession((prev) => {
              if (prev?.id === meta.sessionId) {
                return prev;
              }
              return {
                id: meta.sessionId,
                title: 'AI Session',
                messages: [],
                createdAt: meta.createdAt,
                updatedAt: meta.createdAt,
              };
            });
            if (activeTurnID) {
              patchAssistantMessage(assistantMessageID, activeTurnID, (item) => ({ ...item, turnId: activeTurnID }));
            }
          },
          onDelta: (delta) => {
            const turnID = delta.turn_id || activeTurnID;
            markStreaming(assistantMessageID, turnID);
            patchAssistantMessage(assistantMessageID, turnID, (item) => ({
              ...item,
              turnId: turnID || item.turnId,
              content: `${item.content}${delta.contentChunk}`,
            }));
          },
          onThinkingDelta: (delta) => {
            const turnID = delta.turn_id || activeTurnID;
            markStreaming(assistantMessageID, turnID);
            patchAssistantMessage(assistantMessageID, turnID, (item) => ({
              ...item,
              turnId: turnID || item.turnId,
              thinking: `${item.thinking || ''}${delta.contentChunk}`,
            }));
          },
          onToolCall: (payload) => {
            const turnID = payload.turn_id || activeTurnID;
            attachTraceToAssistant(assistantMessageID, turnID, {
              id: `trace-${Date.now()}-${Math.random().toString(36).slice(2)}`,
              type: 'tool_call',
              payload: payload as Record<string, any>,
              timestamp: new Date().toISOString(),
            });
          },
          onToolResult: (payload) => {
            const turnID = payload.turn_id || activeTurnID;
            attachTraceToAssistant(assistantMessageID, turnID, {
              id: `trace-${Date.now()}-${Math.random().toString(36).slice(2)}`,
              type: 'tool_result',
              payload: payload as Record<string, any>,
              timestamp: new Date().toISOString(),
            });
          },
          onApprovalRequired: (payload) => {
            const turnID = payload.turn_id || activeTurnID;
            attachTraceToAssistant(assistantMessageID, turnID, {
              id: `trace-${Date.now()}-${Math.random().toString(36).slice(2)}`,
              type: 'approval_required',
              payload: payload as Record<string, any>,
              timestamp: new Date().toISOString(),
            });
          },
          onToolIntentUnresolved: (payload) => {
            const turnID = payload.turn_id || activeTurnID;
            attachTraceToAssistant(assistantMessageID, turnID, {
              id: `trace-${Date.now()}-${Math.random().toString(36).slice(2)}`,
              type: 'tool_missing',
              payload: payload as Record<string, any>,
              timestamp: new Date().toISOString(),
            });
          },
          onDone: (done) => {
            latestSession = done.session;
            setCurrentSession(done.session);
            void loadSessions();
            const turnID = done.turn_id || activeTurnID;

            if (done.turn_recommendations && done.turn_recommendations.length > 0) {
              const topRecommendations = done.turn_recommendations.slice(0, 3);
              patchAssistantMessage(assistantMessageID, turnID, (item) => ({
                ...item,
                turnId: turnID || item.turnId,
                phase: 'done',
                recommendations: topRecommendations,
              }));
              startRecommendationReveal(assistantMessageID, topRecommendations.length);
            } else {
              patchAssistantMessage(assistantMessageID, turnID, (item) => ({ ...item, turnId: turnID || item.turnId, phase: 'done' }));
            }

            if (done.stream_state === 'partial') {
              setStreamState('timeout');
              setStreamError('工具结果不完整，可重试本轮对话。');
              const missing = done.tool_summary?.missing || [];
              const missingCallIDs = done.tool_summary?.missing_call_ids || [];
              if (missing.length > 0) {
                attachTraceToAssistant(assistantMessageID, turnID, {
                  id: `trace-${Date.now()}-${Math.random().toString(36).slice(2)}`,
                  type: 'tool_missing',
                  payload: {
                    tool: 'runtime',
                    missing,
                    missing_call_ids: missingCallIDs,
                    summary: done.tool_summary,
                  },
                  timestamp: new Date().toISOString(),
                });
              }
              return;
            }
            if (done.stream_state === 'failed') {
              patchAssistantMessage(assistantMessageID, turnID, (item) => ({ ...item, phase: 'error' }));
              setStreamState('error');
              return;
            }
            setStreamState('done');
          },
          onError: (err) => {
            const turnID = err.turn_id || activeTurnID;
            patchAssistantMessage(assistantMessageID, turnID, (item) => ({ ...item, phase: 'error' }));
            if (err.code === 'tool_timeout_soft') {
              setStreamNotice(err.message || '工具执行较慢，正在继续等待结果…');
              setStreamState('running');
              return;
            }
            if (err.code === 'tool_result_missing') {
              setStreamState('timeout');
              setStreamError(err.message || '工具结果不完整，可重试本轮对话。');
              return;
            }
            setStreamState(err.code === 'tool_timeout_hard' || err.message?.includes('超时') ? 'timeout' : 'error');
            setStreamError(err.message || 'AI服务暂时不可用');
          },
        },
      );

      if (latestSession !== undefined) {
        if (latestSession.id !== previousSessionID && onSessionCreate) {
          onSessionCreate(latestSession);
        } else if (onSessionUpdate) {
          onSessionUpdate(latestSession);
        }
      }
    } catch (error) {
      console.error('发送消息失败:', error);
      setStreamState('error');
      setStreamError('AI服务暂时不可用，请稍后再试。');
      patchAssistantMessage(assistantMessageID, undefined, (item) => ({
        ...item,
        phase: 'error',
        content: item.content || '抱歉，AI服务暂时不可用，请稍后再试。',
      }));
    } finally {
      setLoading(false);
      setActiveAssistantMessageId('');
    }
  };

  const handleSend = async () => {
    await sendMessage(inputValue.trim());
  };

  const handleRetry = async () => {
    if (!lastPrompt) return;
    await sendMessage(lastPrompt);
  };

  const handleOpenSession = async (id: string) => {
    try {
      const response = await Api.ai.getSessionDetail(id);
      setMessages((response.data.messages || []) as LocalMessage[]);
      setCurrentSession(response.data);
      setPendingNewSessionId('');
      setStreamState('idle');
      setStreamError('');
      setStreamNotice('');
    } catch (error) {
      console.error('加载会话详情失败:', error);
    }
  };

  const handleNewSession = () => {
    setPendingNewSessionId(`sess-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`);
    setCurrentSession(null);
    setMessages([]);
    setInputValue('');
    setStreamState('idle');
    setStreamError('');
    setStreamNotice('');
  };

  const persistPinnedSessions = (ids: string[]) => {
    setPinnedSessionIds(ids);
    localStorage.setItem(`ai:pinned:sessions:${scene}`, JSON.stringify(ids));
  };

  const togglePinSession = (id: string) => {
    if (pinnedSessionIds.includes(id)) {
      persistPinnedSessions(pinnedSessionIds.filter((x) => x !== id));
      return;
    }
    persistPinnedSessions([id, ...pinnedSessionIds]);
  };

  const deleteSession = async (id: string) => {
    try {
      await Api.ai.deleteSession(id);
      if (currentSession?.id === id) {
        handleNewSession();
      }
      persistPinnedSessions(pinnedSessionIds.filter((x) => x !== id));
      await loadSessions();
    } catch (error) {
      console.error('删除会话失败:', error);
    }
  };

  const renameSession = async (item: AISession) => {
    const nextTitle = window.prompt('请输入会话名称（1-64字）', item.title || 'AI Session');
    if (nextTitle === null) return;
    const trimmed = nextTitle.trim();
    if (!trimmed) return;
    try {
      const res = await Api.ai.updateSessionTitle(item.id, trimmed);
      const updated = res.data;
      setSessionList((prev) => prev.map((session) => (session.id === updated.id ? { ...session, ...updated } : session)));
      if (currentSession?.id === updated.id) {
        setCurrentSession((prev) => (prev ? { ...prev, title: updated.title, updatedAt: updated.updatedAt } : prev));
      }
    } catch (error) {
      console.error('重命名会话失败:', error);
    }
  };

  const visibleSessions = sessionList.filter((item) => {
    const key = sessionKeyword.trim().toLowerCase();
    if (!key) return true;
    return `${item.title} ${item.id}`.toLowerCase().includes(key);
  });

  const sortSessions = (items: AISession[]) => {
    return items.slice().sort((a, b) => {
      const pinA = pinnedSessionIds.includes(a.id) ? 1 : 0;
      const pinB = pinnedSessionIds.includes(b.id) ? 1 : 0;
      if (pinA !== pinB) return pinB - pinA;
      return new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime();
    });
  };

  const now = new Date();
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const startOfYesterday = startOfToday - 24 * 60 * 60 * 1000;
  const groupedSessions = {
    today: [] as AISession[],
    yesterday: [] as AISession[],
    earlier: [] as AISession[],
  };
  sortSessions(visibleSessions).forEach((item) => {
    const ts = new Date(item.updatedAt).getTime();
    if (ts >= startOfToday) {
      groupedSessions.today.push(item);
    } else if (ts >= startOfYesterday) {
      groupedSessions.yesterday.push(item);
    } else {
      groupedSessions.earlier.push(item);
    }
  });

  const messageAnchors = messages
    .filter((msg) => msg.role === 'user')
    .map((msg, idx) => ({
      id: msg.id,
      label: `#${idx + 1} ${(msg.content || '').slice(0, 28) || '用户输入'}${(msg.content || '').length > 28 ? '...' : ''}`,
    }));

  useEffect(() => {
    if (!messageAnchors.length) {
      setActiveAnchorId('');
      return;
    }
    if (!activeAnchorId) {
      setActiveAnchorId(messageAnchors[0].id);
    }
  }, [messageAnchors, activeAnchorId]);

  const jumpToMessage = (id: string) => {
    const target = document.getElementById(`msg-${id}`);
    target?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    setActiveAnchorId(id);
  };

  const buildSessionMarkdown = () => {
    const title = currentSession?.title || 'AI Session';
    const lines = [`# ${title}`, '', `- Scene: ${scene}`, `- Session: ${currentSession?.id || '-'}`, `- ExportedAt: ${new Date().toISOString()}`, ''];
    messages.forEach((msg) => {
      const role = msg.role === 'assistant' ? 'Assistant' : msg.role === 'user' ? 'User' : 'System';
      lines.push(`## ${role} @ ${new Date(msg.timestamp).toLocaleString()}`);
      lines.push('');
      lines.push(msg.content || '');
      lines.push('');
      if (msg.thinking) {
        lines.push('<details><summary>Thinking</summary>');
        lines.push('');
        lines.push(msg.thinking);
        lines.push('');
        lines.push('</details>');
        lines.push('');
      }
    });
    return lines.join('\n');
  };

  const downloadText = (filename: string, content: string, type = 'text/plain;charset=utf-8') => {
    const blob = new Blob([content], { type });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  };

  const exportMarkdown = () => {
    const sid = currentSession?.id || `session-${Date.now()}`;
    downloadText(`${sid}.md`, buildSessionMarkdown(), 'text/markdown;charset=utf-8');
  };

  const exportJSON = () => {
    const sid = currentSession?.id || `session-${Date.now()}`;
    const payload = {
      session: currentSession,
      scene,
      exportedAt: new Date().toISOString(),
      messages,
    };
    downloadText(`${sid}.json`, JSON.stringify(payload, null, 2), 'application/json;charset=utf-8');
  };

  const copyReplaySummary = async () => {
    const latestAssistant = [...messages].reverse().find((m) => m.role === 'assistant');
    const latestUser = [...messages].reverse().find((m) => m.role === 'user');
    const summary = [
      `会话: ${currentSession?.title || 'AI Session'} (${currentSession?.id || '-'})`,
      `场景: ${scene}`,
      `时间: ${new Date().toLocaleString()}`,
      `最新用户输入: ${(latestUser?.content || '-').slice(0, 180)}`,
      `最新助手回复: ${(latestAssistant?.content || '-').slice(0, 260)}`,
      `消息总数: ${messages.length}`,
    ].join('\n');
    try {
      await navigator.clipboard.writeText(summary);
    } catch {
      const ta = document.createElement('textarea');
      ta.value = summary;
      document.body.appendChild(ta);
      ta.select();
      document.execCommand('copy');
      document.body.removeChild(ta);
    }
  };

  const handleRecommendationFollowup = async (rec: EmbeddedRecommendation) => {
    const prompt = (rec.followup_prompt || rec.content || '').trim();
    if (!prompt || loading) return;
    setFollowupLoadingId(rec.id);
    try {
      await sendMessage(prompt);
    } finally {
      setFollowupLoadingId('');
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      void handleSend();
    }
  };

  return (
    <Card
      title={(
        <Space>
          <MessageOutlined />
          <Text strong>AI 助手</Text>
          {streamState === 'running' ? <Tag color="processing">Streaming</Tag> : null}
        </Space>
      )}
      className={`ai-chat-interface ${className || ''}`}
      style={{ height: '100%' }}
      styles={{
        body: {
          height: '100%',
          minHeight: 0,
          display: 'flex',
          flexDirection: 'column',
          padding: 0,
        },
      }}
    >
      <div className="ai-chat-shell">
        {!sessionId ? (
          <aside className="ai-chat-session-sidebar">
            <div className="ai-chat-session-header">
              <Space>
                <HistoryOutlined />
                <Text strong>会话目录</Text>
              </Space>
              <Tooltip title="新会话">
                <Button size="small" type="text" icon={<PlusOutlined />} onClick={handleNewSession} />
              </Tooltip>
            </div>
            <Input
              size="small"
              placeholder="搜索会话"
              value={sessionKeyword}
              onChange={(e) => setSessionKeyword(e.target.value)}
            />
            <div className="ai-chat-session-list-wrap">
              {sessionLoading ? <Text type="secondary">加载中...</Text> : null}
              {!sessionLoading && visibleSessions.length === 0 ? <Text type="secondary">暂无会话</Text> : null}
              {[
                { key: 'today', title: '今天', list: groupedSessions.today },
                { key: 'yesterday', title: '昨天', list: groupedSessions.yesterday },
                { key: 'earlier', title: '更早', list: groupedSessions.earlier },
              ].map((group) => (
                group.list.length > 0 ? (
                  <div key={group.key} className="ai-chat-session-group">
                    <div className="ai-chat-session-group-title">{group.title}</div>
                    {group.list.map((item) => {
                      const active = currentSession?.id === item.id;
                      const pinned = pinnedSessionIds.includes(item.id);
                      return (
                        <div key={item.id} className={`ai-chat-session-item ${active ? 'is-active' : ''}`}>
                          <button type="button" className="ai-chat-session-btn" onClick={() => void handleOpenSession(item.id)}>
                            <span className="ai-chat-session-title">{item.title || 'AI Session'}</span>
                            <span className="ai-chat-session-time">{new Date(item.updatedAt).toLocaleString()}</span>
                          </button>
                          <div className="ai-chat-session-actions">
                            <Button
                              size="small"
                              type="text"
                              icon={<PushpinOutlined />}
                              className={pinned ? 'is-pinned' : ''}
                              onClick={(e) => {
                                e.stopPropagation();
                                togglePinSession(item.id);
                              }}
                            />
                            <Button
                              size="small"
                              type="text"
                              icon={<EditOutlined />}
                              onClick={(e) => {
                                e.stopPropagation();
                                void renameSession(item);
                              }}
                            />
                            <Button
                              size="small"
                              type="text"
                              danger
                              icon={<DeleteOutlined />}
                              onClick={(e) => {
                                e.stopPropagation();
                                void deleteSession(item.id);
                              }}
                            />
                          </div>
                        </div>
                      );
                    })}
                  </div>
                ) : null
              ))}
            </div>
          </aside>
        ) : null}

        <div className="ai-chat-main-pane">
          {messages.length > 0 ? (
            <div className="ai-chat-ops-bar">
              <Space size={6}>
                <Tooltip title="导出 Markdown">
                  <Button size="small" icon={<FileMarkdownOutlined />} onClick={exportMarkdown}>
                    Markdown
                  </Button>
                </Tooltip>
                <Tooltip title="导出 JSON">
                  <Button size="small" icon={<CodeOutlined />} onClick={exportJSON}>
                    JSON
                  </Button>
                </Tooltip>
                <Tooltip title="复制回放摘要">
                  <Button size="small" type="primary" icon={<CopyOutlined />} onClick={() => void copyReplaySummary()}>
                    复制摘要
                  </Button>
                </Tooltip>
              </Space>
              <Tag icon={<DownloadOutlined />} color="blue">
                可导出
              </Tag>
            </div>
          ) : null}
          {messageAnchors.length > 1 ? (
            <div className="ai-chat-anchor-nav">
              <div className="ai-chat-anchor-title">轮次导航</div>
              <div className="ai-chat-anchor-list">
                {messageAnchors.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    className={`ai-chat-anchor-btn ${activeAnchorId === item.id ? 'is-active' : ''}`}
                    onClick={() => jumpToMessage(item.id)}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
            </div>
          ) : null}
          <div ref={scrollContainerRef} onScroll={handleScroll} className="ai-chat-message-scroll">
            <div className="ai-chat-stream-list">
          {messages.map((message) => {
            const isAssistant = message.role === 'assistant';
            const isUser = message.role === 'user';
            const visibleRecommendationCount = Math.min(message.recommendations?.length || 0, recRevealMap[message.id] || 0);
            const visibleRecommendations = (message.recommendations || []).slice(0, visibleRecommendationCount);
            const showAwaiting = isAssistant && message.phase === 'awaiting_first_token' && !message.content && !(message.thinking || '').trim() && !(message.traces || []).length;
            const showCursor = isAssistant && message.id === activeAssistantMessageId && message.phase === 'streaming';

            return (
              <div key={message.id} className={`ai-chat-message-row ${isUser ? 'ai-chat-message-row-user' : 'ai-chat-message-row-assistant'}`}>
                <span id={`msg-${message.id}`} className="ai-chat-anchor-target" />
                {isAssistant ? (
                  <>
                    <Avatar icon={<MessageOutlined />} className="ai-chat-avatar ai-chat-avatar-assistant" />
                    <div className="ai-chat-message-main">
                      <div className="ai-chat-message-meta">
                        <Text className="ai-chat-message-author">AI 助手</Text>
                        <Text type="secondary" className="ai-chat-message-time">{new Date(message.timestamp).toLocaleTimeString()}</Text>
                      </div>
                      <div className="ai-chat-assistant-content">
                        {showAwaiting ? (
                          <div className="ai-first-token-loading" aria-label="等待模型首包">
                            <span className="ai-first-token-dot" />
                            <span className="ai-first-token-dot" />
                            <span className="ai-first-token-dot" />
                          </div>
                        ) : null}

                        {message.thinking ? (
                          <div className="ai-thinking-block">
                            <Collapse
                              size="small"
                              ghost
                              items={[{
                                key: `thinking-${message.id}`,
                                label: (
                                  <Space className="ai-thinking-header">
                                    <BulbOutlined />
                                    <Text className="ai-thinking-title">查看思考过程</Text>
                                    <Tag color="cyan">{message.thinking.length} chars</Tag>
                                  </Space>
                                ),
                                children: (
                                  <div className="ai-thinking-content">
                                    {renderMarkdown(message.thinking)}
                                  </div>
                                ),
                              }]}
                            />
                          </div>
                        ) : null}

                        {message.traces && message.traces.length > 0 ? (
                          <div className="ai-trace-block">
                            <Collapse
                              size="small"
                              ghost
                              items={[{
                                key: `trace-${message.id}`,
                                label: (
                                  <Space className="ai-trace-header">
                                    <ToolOutlined />
                                    <Text className="ai-trace-title">工具调用轨迹</Text>
                                    <Tag color="gold">{message.traces.length}</Tag>
                                  </Space>
                                ),
                                children: (
                                  <Space direction="vertical" style={{ width: '100%' }} size={8}>
                                    {message.traces.map((trace) => {
                                      const isRawShown = !!traceRawVisible[trace.id];
                                      const traceStatus = trace.type === 'tool_result'
                                        ? ((trace.payload?.result?.ok || trace.payload?.payload?.result?.ok) ? 'success' : 'error')
                                        : (trace.type === 'approval_required' || trace.type === 'tool_missing' ? 'warning' : 'processing');
                                      return (
                                        <Card key={trace.id} size="small" className="ai-trace-card">
                                          <Space className="ai-trace-card-head">
                                            <Space>
                                              <Tag color={traceStatus === 'success' ? 'green' : traceStatus === 'error' ? 'red' : traceStatus === 'warning' ? 'orange' : 'blue'}>
                                                {trace.type}
                                              </Tag>
                                              <Text strong>{trace.payload?.tool || trace.payload?.payload?.tool || 'unknown-tool'}</Text>
                                            </Space>
                                            <Button size="small" type="link" onClick={() => setTraceRawVisible((prev) => ({ ...prev, [trace.id]: !isRawShown }))}>
                                              {isRawShown ? '隐藏原始JSON' : '显示原始JSON'}
                                            </Button>
                                          </Space>
                                          <Text type="secondary" className="ai-trace-time">{new Date(trace.timestamp).toLocaleTimeString()}</Text>
                                          {isRawShown ? (
                                            <pre className="ai-trace-json">
                                              {JSON.stringify(trace.payload, null, 2)}
                                            </pre>
                                          ) : null}
                                        </Card>
                                      );
                                    })}
                                  </Space>
                                ),
                              }]}
                            />
                          </div>
                        ) : null}

                        {(message.content || '').trim() ? (
                          <div className="ai-assistant-markdown-wrap">
                            {renderMarkdown(message.content)}
                            {showCursor ? <span className="ai-typewriter-cursor" /> : null}
                          </div>
                        ) : null}

                        {visibleRecommendations.length > 0 ? (
                          <div className="ai-recommendation-chips-wrap">
                            <Text strong className="ai-inline-recommendations-title">下一步建议</Text>
                            <div className="ai-recommendation-chip-list">
                              {visibleRecommendations.map((rec, idx) => (
                                <Button
                                  key={rec.id}
                                  className="ai-recommendation-chip"
                                  loading={followupLoadingId === rec.id}
                                  onClick={() => void handleRecommendationFollowup(rec)}
                                  title={rec.content}
                                  style={{ animationDelay: `${idx * 80}ms` }}
                                >
                                  {rec.title || rec.content}
                                </Button>
                              ))}
                            </div>
                          </div>
                        ) : null}
                      </div>
                    </div>
                  </>
                ) : null}

                {isUser ? (
                  <div className="ai-chat-user-bubble-wrap">
                    <div className="ai-chat-user-bubble">
                      <Paragraph className="ai-chat-user-paragraph">{message.content}</Paragraph>
                    </div>
                  </div>
                ) : null}
              </div>
            );
            })}
            </div>

            {streamNotice ? (
              <div className="ai-chat-stream-notice">
                <Alert type="info" showIcon message={streamNotice} />
              </div>
            ) : null}

            {showScrollBottom ? (
              <Button
                className="ai-scroll-bottom-btn"
                type="primary"
                shape="round"
                size="small"
                icon={<ArrowDownOutlined />}
                onClick={scrollToBottom}
              >
                回到底部
              </Button>
            ) : null}

            <div ref={messagesEndRef} />
          </div>

          {(streamState === 'timeout' || streamState === 'error') ? (
            <div style={{ padding: '8px 16px 0' }}>
              <Alert
                type={streamState === 'timeout' ? 'warning' : 'error'}
                showIcon
                icon={<WarningOutlined />}
                message={streamState === 'timeout' ? '工具执行可能超时' : '流式对话发生错误'}
                description={streamError || '你可以重试本轮对话。'}
                action={<Button size="small" onClick={() => void handleRetry()}>重试本轮</Button>}
              />
            </div>
          ) : null}

          <div className="ai-chat-input-wrap">
            <Input.TextArea
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="请输入您的问题..."
              autoSize={{ minRows: 2, maxRows: 6 }}
              disabled={loading}
            />
            <div className="ai-chat-send-row">
              <Button
                type="primary"
                icon={<SendOutlined />}
                onClick={() => void handleSend()}
                loading={loading}
                disabled={!inputValue.trim() || loading}
              >
                发送
              </Button>
            </div>
          </div>
        </div>
      </div>
    </Card>
  );
};

export default ChatInterface;
