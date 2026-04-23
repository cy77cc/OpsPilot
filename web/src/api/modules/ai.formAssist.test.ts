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

describe('formAssistStream API', () => {
  it('sends the correct request and parses delta events', async () => {
    const originalFetch = globalThis.fetch;
    const capturedRequests: { url: string; init?: RequestInit }[] = [];
    const frame = (lines: string[]) => `${lines.join('\n')}\n\n`;
    
    const fetchMock = async (url: string | Request | URL, init?: RequestInit) => {
      capturedRequests.push({ url: url.toString(), init });
      return {
        ok: true,
        body: buildStream([
          frame([
            'event: delta',
            'data: {"content":"Suggestion for "}',
          ]),
          frame([
            'event: delta',
            'data: {"content":"the field."}',
          ]),
          frame([
            'event: done',
            'data: {"run_id":"run-123","status":"completed"}',
          ]),
        ]),
      } as Response;
    };
    globalThis.fetch = fetchMock;

    const onDelta = vi.fn();
    const onDone = vi.fn();

    const params = {
      scene: 'test-scene',
      user_prompt: 'Help me fill this',
      field_meta: {
        key: 'field1',
        label: 'Field 1',
        purpose: 'Testing',
      },
      form_context: {
        otherField: 'value',
      },
    };

    try {
      await aiApi.formAssistStream(params, { onDelta, onDone });
    } finally {
      globalThis.fetch = originalFetch;
    }

    expect(capturedRequests).toHaveLength(1);
    expect(capturedRequests[0].url).toContain('/ai/assist/form/stream');
    expect(capturedRequests[0].init?.method).toBe('POST');
    expect(JSON.parse(String(capturedRequests[0].init?.body))).toEqual(params);

    expect(onDelta).toHaveBeenCalledTimes(2);
    expect(onDelta).toHaveBeenNthCalledWith(1, expect.objectContaining({ content: 'Suggestion for ' }));
    expect(onDelta).toHaveBeenNthCalledWith(2, expect.objectContaining({ content: 'the field.' }));
    expect(onDone).toHaveBeenCalledWith(expect.objectContaining({ run_id: 'run-123', status: 'completed' }));
  });

  it('includes headers from localStorage', async () => {
    const originalFetch = globalThis.fetch;
    const capturedRequests: { url: string; init?: RequestInit }[] = [];
    
    const fetchMock = async (url: string | Request | URL, init?: RequestInit) => {
      capturedRequests.push({ url: url.toString(), init });
      return {
        ok: true,
        body: buildStream(['event: done\ndata: {}\n\n']),
      } as Response;
    };
    globalThis.fetch = fetchMock;

    const getItemSpy = vi.spyOn(Storage.prototype, 'getItem').mockImplementation((key) => {
      if (key === 'token') {return 'test-token';}
      if (key === 'projectId') {return 'test-project';}
      return null;
    });

    try {
      await aiApi.formAssistStream({
        scene: 's',
        user_prompt: 'p',
        field_meta: { key: 'k', label: 'l', purpose: 'p' },
        form_context: {},
      }, {});
    } finally {
      globalThis.fetch = originalFetch;
      getItemSpy.mockRestore();
    }

    const headers = capturedRequests[0].init?.headers as Record<string, string>;
    expect(headers['Authorization']).toBe('Bearer test-token');
    expect(headers['X-Project-ID']).toBe('test-project');
  });
});
