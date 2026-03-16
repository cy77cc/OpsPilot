import { describe, expect, it, vi } from 'vitest';
import { aiApi, normalizeVisibleStreamChunk } from './ai';

function buildStream(chunks: string[]) {
  const encoder = new TextEncoder();
  return new ReadableStream<Uint8Array>({
    start(controller) {
      chunks.forEach((chunk) => controller.enqueue(encoder.encode(chunk)));
      controller.close();
    },
  });
}

describe('normalizeVisibleStreamChunk', () => {
  it('preserves markdown whitespace from phase 1 delta payloads', async () => {
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
        contentChunk: '  ## Title\n\n| A | B |\n| - | - |\n',
      }),
    );
  });

  it('passes through plain text', () => {
    expect(normalizeVisibleStreamChunk('hello')).toBe('hello');
  });

  it('unwraps response envelope', () => {
    expect(normalizeVisibleStreamChunk('{"response":"hello"}')).toBe('hello');
  });

  it('hides internal steps envelope', () => {
    expect(normalizeVisibleStreamChunk('{"steps":["a","b"]}')).toBe('');
  });

  it('keeps ordinary json content', () => {
    const json = '{"name":"nginx","replicas":3}';
    expect(normalizeVisibleStreamChunk(json)).toBe(json);
  });
});
