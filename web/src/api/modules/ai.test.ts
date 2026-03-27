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

  it('consumes ops_plan_updated stream events', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      body: buildStream([
        'event: ops_plan_updated\ndata: {"run_id":"run-1","session_id":"sess-1","runtime":{"todos":[{"id":"todo-1","content":"检查节点","status":"pending"}]}}\n\n',
      ]),
    } as Response);

    const onOpsPlanUpdated = vi.fn();

    await aiApi.chatStream(
      { message: 'hi' },
      { onOpsPlanUpdated },
    );

    expect(onOpsPlanUpdated).toHaveBeenCalledWith(expect.objectContaining({
      run_id: 'run-1',
      session_id: 'sess-1',
      runtime: expect.objectContaining({
        todos: [
          expect.objectContaining({
            id: 'todo-1',
            content: '检查节点',
            status: 'pending',
          }),
        ],
      }),
    }));
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
    await aiApi.getRunProjection('run-1');
    await aiApi.getRunContent('content-1');
    await aiApi.getDiagnosisReport('report-1');

    expect(getMock).toHaveBeenNthCalledWith(1, '/ai/runs/run-1');
    expect(getMock).toHaveBeenNthCalledWith(2, '/ai/runs/run-1/projection', undefined);
    expect(getMock).toHaveBeenNthCalledWith(3, '/ai/run-contents/content-1');
    expect(getMock).toHaveBeenNthCalledWith(4, '/ai/diagnosis/report-1');
  });

  it('normalizes run report ids from legacy and canonical payloads', async () => {
    const getMock = vi.spyOn(apiService, 'get')
      .mockResolvedValueOnce({
        success: true,
        data: {
          run_id: 'run-1',
          status: 'completed',
          report: {
            id: 'report-legacy',
            summary: 'legacy report',
          },
        },
      })
      .mockResolvedValueOnce({
        success: true,
        data: {
          run_id: 'run-1',
          status: 'completed',
          report: {
            report_id: 'report-canonical',
            summary: 'canonical report',
          },
        },
      });

    const legacyResponse = await aiApi.getRunStatus('run-1');
    const canonicalResponse = await aiApi.getRunStatus('run-1');

    expect(getMock).toHaveBeenNthCalledWith(1, '/ai/runs/run-1');
    expect(getMock).toHaveBeenNthCalledWith(2, '/ai/runs/run-1');
    expect(legacyResponse.data.report).toEqual(expect.objectContaining({
      id: 'report-legacy',
      report_id: 'report-legacy',
    }));
    expect(canonicalResponse.data.report).toEqual(expect.objectContaining({
      id: 'report-canonical',
      report_id: 'report-canonical',
    }));
  });

  it('normalizes session timestamps from snake_case and camelCase payloads', async () => {
    const getMock = vi.spyOn(apiService, 'get')
      .mockResolvedValueOnce({
        success: true,
        data: [
          {
            id: 'sess-1',
            title: 'Legacy session',
            scene: 'ai',
            messages: [],
            created_at: '2026-03-24T10:00:00Z',
            updated_at: '2026-03-24T11:00:00Z',
          },
        ],
      })
      .mockResolvedValueOnce({
        success: true,
        data: {
          id: 'sess-2',
          title: 'Camel session',
          scene: 'ai',
          messages: [],
          createdAt: '2026-03-24T12:00:00Z',
          updatedAt: '2026-03-24T13:00:00Z',
        },
      });

    const listResponse = await aiApi.getSessions();
    const detailResponse = await aiApi.getSession('sess-2');

    expect(getMock).toHaveBeenNthCalledWith(1, '/ai/sessions', undefined);
    expect(getMock).toHaveBeenNthCalledWith(2, '/ai/sessions/sess-2');
    expect(listResponse.data[0]).toEqual(expect.objectContaining({
      createdAt: '2026-03-24T10:00:00Z',
      updatedAt: '2026-03-24T11:00:00Z',
      created_at: '2026-03-24T10:00:00Z',
      updated_at: '2026-03-24T11:00:00Z',
    }));
    expect(detailResponse.data).toEqual(expect.objectContaining({
      createdAt: '2026-03-24T12:00:00Z',
      updatedAt: '2026-03-24T13:00:00Z',
      created_at: '2026-03-24T12:00:00Z',
      updated_at: '2026-03-24T13:00:00Z',
    }));
  });

  it('preserves resumable credentials on historical assistant messages and runs', async () => {
    const getMock = vi.spyOn(apiService, 'get')
      .mockResolvedValueOnce({
        success: true,
        data: {
          id: 'sess-1',
          title: 'Resumable session',
          scene: 'ai',
          messages: [
            {
              id: 'msg-1',
              role: 'assistant',
              content: '',
              status: 'waiting_approval',
              run_id: 'run-1',
              client_request_id: 'req-1',
              latest_event_id: 'evt-9',
              approval_id: 'approval-1',
              resumable: true,
            },
          ],
        },
      })
      .mockResolvedValueOnce({
        success: true,
        data: {
          run_id: 'run-1',
          status: 'waiting_approval',
          client_request_id: 'req-1',
          latest_event_id: 'evt-9',
          approval_id: 'approval-1',
          resumable: true,
        },
      });

    const sessionResponse = await aiApi.getSession('sess-1');
    const runResponse = await aiApi.getRunStatus('run-1');

    expect(getMock).toHaveBeenNthCalledWith(1, '/ai/sessions/sess-1');
    expect(getMock).toHaveBeenNthCalledWith(2, '/ai/runs/run-1');
    expect(sessionResponse.data.messages[0]).toEqual(expect.objectContaining({
      run_id: 'run-1',
      client_request_id: 'req-1',
      latest_event_id: 'evt-9',
      approval_id: 'approval-1',
      resumable: true,
    }));
    expect(runResponse.data).toEqual(expect.objectContaining({
      run_id: 'run-1',
      client_request_id: 'req-1',
      latest_event_id: 'evt-9',
      approval_id: 'approval-1',
      resumable: true,
    }));
  });
});

describe('aiApi admin model endpoints', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('calls admin model CRUD and import endpoints', async () => {
    const getMock = vi.spyOn(apiService, 'get').mockResolvedValue({ success: true, data: {} });
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({ success: true, data: {} });
    const putMock = vi.spyOn(apiService, 'put').mockResolvedValue({ success: true, data: {} });
    const deleteMock = vi.spyOn(apiService, 'delete').mockResolvedValue({ success: true, data: undefined });

    const payload = {
      name: 'Qwen Prod',
      provider: 'qwen',
      model: 'qwen-max',
      base_url: 'https://example.com',
      api_key: 'sk-test',
      temperature: 0.7,
      thinking: true,
      is_default: false,
      is_enabled: true,
      sort_order: 10,
    };

    await aiApi.listAdminModels();
    await aiApi.getAdminModel(1);
    await aiApi.createAdminModel(payload);
    await aiApi.updateAdminModel(1, { name: 'Qwen Prod v2', api_key: 'sk-rotate' });
    await aiApi.setAdminDefaultModel(1);
    await aiApi.deleteAdminModel(2);
    await aiApi.previewAdminModelImport({ replace_all: false, providers: [payload] });
    await aiApi.importAdminModels({ replace_all: true, providers: [payload] });

    expect(getMock).toHaveBeenNthCalledWith(1, '/admin/ai/models');
    expect(getMock).toHaveBeenNthCalledWith(2, '/admin/ai/models/1');
    expect(postMock).toHaveBeenNthCalledWith(1, '/admin/ai/models', payload);
    expect(putMock).toHaveBeenNthCalledWith(1, '/admin/ai/models/1', { name: 'Qwen Prod v2', api_key: 'sk-rotate' });
    expect(putMock).toHaveBeenNthCalledWith(2, '/admin/ai/models/1/default');
    expect(deleteMock).toHaveBeenCalledWith('/admin/ai/models/2');
    expect(postMock).toHaveBeenNthCalledWith(2, '/admin/ai/models/import/preview', { replace_all: false, providers: [payload] });
    expect(postMock).toHaveBeenNthCalledWith(3, '/admin/ai/models/import', { replace_all: true, providers: [payload] });
  });
});
