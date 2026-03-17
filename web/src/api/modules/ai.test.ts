import { afterEach, describe, expect, it, vi } from 'vitest';
import { aiApi } from './ai';
import apiService from '../api';

function buildStream(chunks: string[]) {
  const encoder = new TextEncoder();
  return new ReadableStream<Uint8Array>({
    start(controller) {
      chunks.forEach((chunk) => controller.enqueue(encoder.encode(chunk)));
      controller.close();
    },
  });
}

describe('aiApi.chatStream', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it('consumes a2ui stream events', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: meta\ndata: {"session_id":"sess-1","run_id":"run-1","turn":1}\n\n',
        'event: agent_handoff\ndata: {"from":"OpsPilotAgent","to":"DiagnosisAgent","intent":"diagnosis"}\n\n',
        'event: plan\ndata: {"steps":["inspect pods"],"iteration":0}\n\n',
        'event: delta\ndata: {"content":"Collecting evidence"}\n\n',
        'event: done\ndata: {"run_id":"run-1","status":"completed","iterations":1}\n\n',
      ]),
    } as Response);

    const onMeta = vi.fn();
    const onAgentHandoff = vi.fn();
    const onPlan = vi.fn();
    const onDelta = vi.fn();
    const onDone = vi.fn();

    await aiApi.chatStream(
      { message: 'Diagnose rollout', session_id: 'sess-1' },
      { onMeta, onAgentHandoff, onPlan, onDelta, onDone },
    );

    expect(fetchMock).toHaveBeenCalled();
    expect(onMeta).toHaveBeenCalledWith(expect.objectContaining({ session_id: 'sess-1', run_id: 'run-1', turn: 1 }));
    expect(onAgentHandoff).toHaveBeenCalledWith(expect.objectContaining({ intent: 'diagnosis', to: 'DiagnosisAgent' }));
    expect(onPlan).toHaveBeenCalledWith(expect.objectContaining({ iteration: 0, steps: ['inspect pods'] }));
    expect(onDelta).toHaveBeenCalledWith(expect.objectContaining({ content: 'Collecting evidence' }));
    expect(onDone).toHaveBeenCalledWith(expect.objectContaining({ run_id: 'run-1', status: 'completed', iterations: 1 }));
  });

  it('maps delta events to visible answer content', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: delta\ndata: {"content":"hello from backend"}\n\n',
        'event: done\ndata: {"run_id":"run-1","status":"completed","iterations":0}\n\n',
      ]),
    } as Response);

    const onDelta = vi.fn();

    await aiApi.chatStream(
      { message: 'hi' },
      { onDelta },
    );

    expect(onDelta).toHaveBeenCalledWith(expect.objectContaining({ content: 'hello from backend' }));
  });

  it('preserves error payloads', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: error\ndata: {"message":"stream failed","code":"stream_failed"}\n\n',
      ]),
    } as Response);

    const onError = vi.fn();

    await aiApi.chatStream(
      { message: 'hi' },
      { onError },
    );

    expect(onError).toHaveBeenCalledWith(expect.objectContaining({
      message: 'stream failed',
      code: 'stream_failed',
    }));
  });
});

describe('aiApi phase 1 endpoints', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('provides session CRUD methods', async () => {
    const getMock = vi.spyOn(apiService, 'get').mockResolvedValue({ success: true, data: [] });
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({ success: true, data: { id: 'sess-1' } });
    const deleteMock = vi.spyOn(apiService, 'delete').mockResolvedValue({ success: true, data: undefined });

    await aiApi.getSessions();
    await aiApi.createSession({ title: 'New Session', scene: 'ai' });
    await aiApi.getSession('sess-1');
    await aiApi.deleteSession('sess-1');

    expect(getMock).toHaveBeenNthCalledWith(1, '/ai/sessions', undefined);
    expect(postMock).toHaveBeenCalledWith('/ai/sessions', { title: 'New Session', scene: 'ai' });
    expect(getMock).toHaveBeenNthCalledWith(2, '/ai/sessions/sess-1');
    expect(deleteMock).toHaveBeenCalledWith('/ai/sessions/sess-1');
  });

  it('fetches run status and diagnosis report', async () => {
    const getMock = vi.spyOn(apiService, 'get').mockResolvedValue({ success: true, data: {} });

    await aiApi.getRunStatus('run-1');
    await aiApi.getDiagnosisReport('report-1');

    expect(getMock).toHaveBeenNthCalledWith(1, '/ai/runs/run-1');
    expect(getMock).toHaveBeenNthCalledWith(2, '/ai/diagnosis/report-1');
  });
});
