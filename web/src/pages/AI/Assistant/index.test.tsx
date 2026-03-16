import React from 'react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import AssistantPage from './index';

const aiMocks = vi.hoisted(() => ({
  getSessions: vi.fn(),
  createSession: vi.fn(),
  getSession: vi.fn(),
  getRunStatus: vi.fn(),
  chatStream: vi.fn(),
  navigate: vi.fn(),
}));

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom');
  return {
    ...actual,
    useNavigate: () => aiMocks.navigate,
  };
});

vi.mock('../../../api/modules/ai', () => ({
  aiApi: {
    getSessions: aiMocks.getSessions,
    createSession: aiMocks.createSession,
    getSession: aiMocks.getSession,
    getRunStatus: aiMocks.getRunStatus,
    chatStream: aiMocks.chatStream,
  },
}));

describe('AssistantPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
  });

  it('shows session list and active conversation', async () => {
    aiMocks.getSessions.mockResolvedValue({ data: [{ id: 'sess-1', title: '诊断对话' }] });
    aiMocks.getSession.mockResolvedValue({ data: { id: 'sess-1', title: '诊断对话', messages: [{ id: 'm1', role: 'assistant', content: '历史消息', timestamp: '' }] } });
    aiMocks.getRunStatus.mockResolvedValue({ data: null });

    render(
      <MemoryRouter initialEntries={['/ai']}>
        <Routes>
          <Route path="/ai" element={<AssistantPage />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText('AI Assistant')).toBeInTheDocument();
    expect((await screen.findAllByText('诊断对话')).length).toBeGreaterThan(0);
    expect(await screen.findByText('历史消息')).toBeInTheDocument();
  });

  it('submits message and renders stream updates with diagnosis link', async () => {
    aiMocks.getSessions.mockResolvedValue({ data: [] });
    aiMocks.createSession.mockResolvedValue({ data: { id: 'sess-new', title: '新对话', messages: [], createdAt: '', updatedAt: '' } });
    aiMocks.getRunStatus.mockResolvedValue({ data: null });
    aiMocks.chatStream.mockImplementation(async (_params: any, handlers: any) => {
      handlers.onInit?.({ session_id: 'sess-new', run_id: 'run-1' });
      handlers.onIntent?.({ intent_type: 'diagnosis', assistant_type: 'diagnosis' });
      handlers.onStatus?.({ status: 'running' });
      handlers.onProgress?.({ summary: 'Collecting evidence' });
      handlers.onDelta?.({ contentChunk: '诊断进行中' });
      handlers.onReportReady?.({ report_id: 'report-1', summary: 'Quota exhausted' });
      handlers.onDone?.({ run_id: 'run-1', status: 'completed' });
    });

    render(
      <MemoryRouter initialEntries={['/ai']}>
        <Routes>
          <Route path="/ai" element={<AssistantPage />} />
        </Routes>
      </MemoryRouter>,
    );

    fireEvent.click((await screen.findAllByRole('button', { name: /新建对话/ }))[0]);
    fireEvent.change((await screen.findAllByPlaceholderText('询问平台问题，或请求诊断故障'))[0], { target: { value: 'Diagnose rollout' } });
    fireEvent.click((await screen.findAllByRole('button', { name: /发送消息/ }))[0]);

    expect(await screen.findByText('诊断进行中')).toBeInTheDocument();
    expect(await screen.findByText('Collecting evidence')).toBeInTheDocument();
    expect(await screen.findByText('Quota exhausted')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /查看完整报告/ }));
    expect(aiMocks.navigate).toHaveBeenCalledWith('/ai/diagnosis/report-1');
  });

  it('restores run status after refresh', async () => {
    sessionStorage.setItem('ai:lastRunId', 'run-restore');
    aiMocks.getSessions.mockResolvedValue({ data: [] });
    aiMocks.getRunStatus.mockResolvedValue({ data: { run_id: 'run-restore', status: 'completed', progress_summary: 'Recovered' } });

    render(
      <MemoryRouter initialEntries={['/ai']}>
        <Routes>
          <Route path="/ai" element={<AssistantPage />} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(aiMocks.getRunStatus).toHaveBeenCalledWith('run-restore');
    });
    expect(await screen.findByText('Recovered')).toBeInTheDocument();
  });
});
