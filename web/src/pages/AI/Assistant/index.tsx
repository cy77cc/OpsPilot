import React, { useEffect, useMemo, useState } from 'react';
import { Card, Col, Row, Space, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';
import { aiApi, type AIMessage, type AIRun, type AISession } from '../../../api/modules/ai';
import SessionList from '../../../components/AIAssistant/SessionList';
import ChatTimeline from '../../../components/AIAssistant/ChatTimeline';
import Composer from '../../../components/AIAssistant/Composer';
import RunStatus from '../../../components/AIAssistant/RunStatus';
import DiagnosisSummaryCard from '../../../components/AIAssistant/DiagnosisSummaryCard';

const ACTIVE_RUN_STORAGE_KEY = 'ai:lastRunId';

const AssistantPage: React.FC = () => {
  const navigate = useNavigate();
  const [sessions, setSessions] = useState<AISession[]>([]);
  const [activeSession, setActiveSession] = useState<AISession | null>(null);
  const [messages, setMessages] = useState<AIMessage[]>([]);
  const [activeRun, setActiveRun] = useState<AIRun | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    void (async () => {
      const sessionsResp = await aiApi.getSessions();
      const nextSessions = sessionsResp.data || [];
      setSessions(nextSessions);
      if (nextSessions.length > 0) {
        const detailResp = await aiApi.getSession(nextSessions[0].id);
        setActiveSession(detailResp.data);
        setMessages(detailResp.data?.messages || []);
      }

      const storedRunId = sessionStorage.getItem(ACTIVE_RUN_STORAGE_KEY);
      if (storedRunId) {
        const runResp = await aiApi.getRunStatus(storedRunId);
        setActiveRun(runResp.data);
      }
    })();
  }, []);

  const handleSelectSession = async (session: AISession) => {
    const detailResp = await aiApi.getSession(session.id);
    setActiveSession(detailResp.data);
    setMessages(detailResp.data?.messages || []);
  };

  const handleNewSession = async () => {
    const created = await aiApi.createSession({ title: '新对话', scene: 'ai' });
    const next = created.data;
    setSessions((prev) => [next, ...prev]);
    setActiveSession(next);
    setMessages([]);
    setActiveRun(null);
    sessionStorage.removeItem(ACTIVE_RUN_STORAGE_KEY);
  };

  const handleSubmit = async (message: string) => {
    const userMessage: AIMessage = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: message,
      timestamp: new Date().toISOString(),
    };
    const assistantMessage: AIMessage = {
      id: `assistant-${Date.now()}`,
      role: 'assistant',
      content: '',
      timestamp: new Date().toISOString(),
    };

    setMessages((prev) => [...prev, userMessage, assistantMessage]);
    setLoading(true);

    await aiApi.chatStream(
      {
        message,
        session_id: activeSession?.id,
      },
      {
        onInit: ({ session_id, run_id }) => {
          sessionStorage.setItem(ACTIVE_RUN_STORAGE_KEY, run_id);
          setActiveRun({
            run_id,
            status: 'running',
          });
          if (!activeSession) {
            setActiveSession((prev) => prev || ({
              id: session_id,
              title: '新对话',
              messages: [],
              createdAt: new Date().toISOString(),
              updatedAt: new Date().toISOString(),
            }));
          }
        },
        onStatus: (status) => {
          setActiveRun((prev) => prev ? { ...prev, ...status } : null);
        },
        onIntent: (intent) => {
          setActiveRun((prev) => prev ? { ...prev, ...intent } as AIRun : prev);
        },
        onProgress: (progress) => {
          setActiveRun((prev) => prev ? { ...prev, progress_summary: progress.summary } : prev);
        },
        onDelta: ({ contentChunk }) => {
          setMessages((prev) => prev.map((item, index) => (
            index === prev.length - 1 ? { ...item, content: `${item.content}${contentChunk}` } : item
          )));
        },
        onReportReady: ({ report_id, summary }) => {
          setActiveRun((prev) => prev ? {
            ...prev,
            report: { report_id, summary },
          } : prev);
        },
        onDone: (done) => {
          setActiveRun((prev) => prev ? { ...prev, ...done } as AIRun : prev);
          setLoading(false);
        },
        onError: (error) => {
          setMessages((prev) => prev.map((item, index) => (
            index === prev.length - 1 ? { ...item, content: error.message } : item
          )));
          setLoading(false);
        },
      },
    );
  };

  const summaryTitle = useMemo(() => activeSession?.title || 'AI Assistant', [activeSession]);

  return (
    <div style={{ padding: 24 }}>
      <Typography.Title level={2} style={{ marginBottom: 8 }}>AI Assistant</Typography.Title>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 24 }}>
        单一入口的只读问答与诊断工作台。
      </Typography.Paragraph>

      <Row gutter={16} align="stretch">
        <Col xs={24} lg={6}>
          <Card style={{ borderRadius: 20 }}>
            <SessionList
              sessions={sessions}
              activeSessionId={activeSession?.id}
              onSelect={handleSelectSession}
              onNewSession={handleNewSession}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={summaryTitle} style={{ borderRadius: 20, minHeight: 560 }}>
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <ChatTimeline messages={messages} />
              <Composer loading={loading} onSubmit={handleSubmit} />
            </Space>
          </Card>
        </Col>
        <Col xs={24} lg={6}>
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <RunStatus run={activeRun} />
            <DiagnosisSummaryCard
              run={activeRun}
              onOpenReport={(reportId) => navigate(`/ai/diagnosis/${reportId}`)}
            />
          </Space>
        </Col>
      </Row>
    </div>
  );
};

export default AssistantPage;
