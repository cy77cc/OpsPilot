import { describe, expect, it, vi } from 'vitest';
import { aiApi } from './ai';

function buildStream(chunks: string[]) {
  const encoder = new TextEncoder();
  return new ReadableStream<Uint8Array>({
    start(controller) {
      chunks.forEach((chunk) => controller.enqueue(encoder.encode(chunk)));
      controller.close();
    },
  });
}

describe('a2ui delta stream parsing', () => {
  it('parses run_state events with resumable statuses', async () => {
    const originalFetch = globalThis.fetch;
    const frame = (lines: string[]) => `${lines.join('\n')}\n\n`;
    const fetchMock = async () => ({
      ok: true,
      body: buildStream([
        frame([
          'id: evt-run-state-1',
          'event: run_state',
          'data: {"run_id":"run-1","status":"waiting_approval","agent":"executor"}',
        ]),
        frame([
          'id: evt-run-state-2',
          'event: run_state',
          'data: {"run_id":"run-1","status":"resume_failed_retryable","agent":"executor"}',
        ]),
      ]),
    }) as Response;
    globalThis.fetch = fetchMock;

    const onEventId = vi.fn();
    const onRunState = vi.fn();

    try {
      await aiApi.chatStream(
        { message: 'hi' },
        { onEventId, onRunState },
      );
    } finally {
      globalThis.fetch = originalFetch;
    }

    expect(onEventId).toHaveBeenNthCalledWith(1, 'evt-run-state-1');
    expect(onEventId).toHaveBeenNthCalledWith(2, 'evt-run-state-2');
    expect(onRunState).toHaveBeenNthCalledWith(1, expect.objectContaining({
      run_id: 'run-1',
      status: 'waiting_approval',
      agent: 'executor',
    }));
    expect(onRunState).toHaveBeenNthCalledWith(2, expect.objectContaining({
      run_id: 'run-1',
      status: 'resume_failed_retryable',
      agent: 'executor',
    }));
  });

  it('parses SSE event ids and approval resume events', async () => {
    const originalFetch = globalThis.fetch;
    const capturedBodies: Array<Record<string, unknown>> = [];
    const frame = (lines: string[]) => `${lines.join('\n')}\n\n`;
    const fetchMock = async (_input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.body) {
        capturedBodies.push(JSON.parse(String(init.body)));
      }
      return {
        ok: true,
        body: buildStream([
          frame([
            'id: evt-1',
            'event: ai.run.resuming',
            'data: {"run_id":"run-1","session_id":"session-1","approval_id":"approval-1"}',
          ]),
          frame([
            'id: evt-2',
            'event: ai.run.resumed',
            'data: {"run_id":"run-1","session_id":"session-1","approval_id":"approval-1"}',
          ]),
          frame([
            'id: evt-3',
            'event: ai.run.resume_failed',
            'data: {"run_id":"run-1","session_id":"session-1","retryable":true,"message":"retry later"}',
          ]),
          frame([
            'id: evt-4',
            'event: ai.run.completed',
            'data: {"run_id":"run-1","session_id":"session-1"}',
          ]),
          frame([
            'id: evt-5',
            'event: ai.approval.expired',
            'data: {"run_id":"run-1","session_id":"session-1","approval_id":"approval-1","expires_at":"2026-03-24T09:00:00Z"}',
          ]),
        ]),
      } as Response;
    };
    globalThis.fetch = fetchMock;

    const onEventId = vi.fn();
    const onRunResuming = vi.fn();
    const onRunResumed = vi.fn();
    const onRunResumeFailed = vi.fn();
    const onRunCompleted = vi.fn();
    const onApprovalExpired = vi.fn();

    try {
      await aiApi.chatStream(
        { message: 'hi', lastEventId: 'evt-previous' },
        {
          onEventId,
          onRunResuming,
          onRunResumed,
          onRunResumeFailed,
          onRunCompleted,
          onApprovalExpired,
        },
      );
    } finally {
      globalThis.fetch = originalFetch;
    }

    expect(capturedBodies).toHaveLength(1);
    expect(capturedBodies[0]).toEqual(expect.objectContaining({
      message: 'hi',
      lastEventId: 'evt-previous',
      last_event_id: 'evt-previous',
    }));
    expect(onEventId).toHaveBeenNthCalledWith(1, 'evt-1');
    expect(onEventId).toHaveBeenNthCalledWith(2, 'evt-2');
    expect(onEventId).toHaveBeenNthCalledWith(3, 'evt-3');
    expect(onEventId).toHaveBeenNthCalledWith(4, 'evt-4');
    expect(onEventId).toHaveBeenNthCalledWith(5, 'evt-5');
    expect(onRunResuming).toHaveBeenCalledWith(
      expect.objectContaining({ run_id: 'run-1', session_id: 'session-1', approval_id: 'approval-1' }),
    );
    expect(onRunResumed).toHaveBeenCalledWith(
      expect.objectContaining({ run_id: 'run-1', session_id: 'session-1', approval_id: 'approval-1' }),
    );
    expect(onRunResumeFailed).toHaveBeenCalledWith(
      expect.objectContaining({ retryable: true, message: 'retry later' }),
    );
    expect(onRunCompleted).toHaveBeenCalledWith(
      expect.objectContaining({ run_id: 'run-1', session_id: 'session-1' }),
    );
    expect(onApprovalExpired).toHaveBeenCalledWith(
      expect.objectContaining({ approval_id: 'approval-1', expires_at: '2026-03-24T09:00:00Z' }),
    );
  });

  it('parses ops_plan_updated runtime snapshots', async () => {
    const originalFetch = globalThis.fetch;
    const frame = (lines: string[]) => `${lines.join('\n')}\n\n`;
    const fetchMock = async () => ({
      ok: true,
      body: buildStream([
        frame([
          'id: evt-7',
          'event: ops_plan_updated',
          'data: {"run_id":"run-1","session_id":"session-1","runtime":{"phase":"planning","phaseLabel":"Task Board","todos":[{"id":"todo-1","content":"检查节点","status":"in_progress"}]}}',
        ]),
      ]),
    }) as Response;
    globalThis.fetch = fetchMock;

    const onEventId = vi.fn();
    const onOpsPlanUpdated = vi.fn();

    try {
      await aiApi.chatStream(
        { message: 'hi', lastEventId: 'evt-previous' },
        {
          onEventId,
          onOpsPlanUpdated,
        },
      );
    } finally {
      globalThis.fetch = originalFetch;
    }

    expect(onEventId).toHaveBeenCalledWith('evt-7');
    expect(onOpsPlanUpdated).toHaveBeenCalledWith(
      expect.objectContaining({
        run_id: 'run-1',
        session_id: 'session-1',
        runtime: expect.objectContaining({
          phase: 'planning',
          phaseLabel: 'Task Board',
          todos: [
            expect.objectContaining({
              id: 'todo-1',
              content: '检查节点',
              status: 'in_progress',
            }),
          ],
        }),
      }),
    );
  });

  it('marks cursor expired stream errors as recoverable', async () => {
    const originalFetch = globalThis.fetch;
    const frame = (lines: string[]) => `${lines.join('\n')}\n\n`;
    const fetchMock = async () => ({
      ok: true,
      body: buildStream([
        frame([
          'id: evt-6',
          'event: error',
          'data: {"code":"AI_STREAM_CURSOR_EXPIRED","message":"last_event_id is too old; refresh the stream snapshot"}',
        ]),
      ]),
    }) as Response;
    globalThis.fetch = fetchMock;

    const onError = vi.fn();

    try {
      await aiApi.chatStream(
        { message: 'hi', lastEventId: 'evt-previous' },
        { onError },
      );
    } finally {
      globalThis.fetch = originalFetch;
    }

    expect(onError).toHaveBeenCalledWith(
      expect.objectContaining({
        code: 'AI_STREAM_CURSOR_EXPIRED',
        recoverable: true,
        message: 'last_event_id is too old; refresh the stream snapshot',
      }),
    );
  });

  it('preserves markdown whitespace from delta payloads', async () => {
    const originalFetch = globalThis.fetch;
    const fetchMock = async () => ({
      ok: true,
      body: buildStream([
        'event: delta\ndata: {"content":"  ## Title\\n\\n| A | B |\\n| - | - |\\n"}\n\n',
      ]),
    }) as Response;
    globalThis.fetch = fetchMock;

    const onDelta = vi.fn();

    try {
      await aiApi.chatStream(
        { message: 'hi' },
        { onDelta },
      );
    } finally {
      globalThis.fetch = originalFetch;
    }

    expect(onDelta).toHaveBeenCalledWith(
      expect.objectContaining({
        content: '  ## Title\n\n| A | B |\n| - | - |\n',
      }),
    );
  });

  it('passes structured delta content through without envelope normalization', async () => {
    const originalFetch = globalThis.fetch;
    const fetchMock = async () => ({
      ok: true,
      body: buildStream([
        'event: delta\ndata: {"content":"{\\"steps\\":[\\"a\\",\\"b\\"]}"}\n\n',
      ]),
    }) as Response;
    globalThis.fetch = fetchMock;

    const onDelta = vi.fn();

    try {
      await aiApi.chatStream(
        { message: 'hi' },
        { onDelta },
      );
    } finally {
      globalThis.fetch = originalFetch;
    }

    expect(onDelta).toHaveBeenCalledWith(
      expect.objectContaining({
        content: '{"steps":["a","b"]}',
      }),
    );
  });
});
