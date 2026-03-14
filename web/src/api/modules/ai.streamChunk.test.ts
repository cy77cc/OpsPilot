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
  it('preserves markdown whitespace and blank lines from SSE data lines', async () => {
    const originalFetch = globalThis.fetch;
    const fetchMock = async () => ({
      ok: true,
      body: buildStream([
        'event: final_answer_delta\n',
        'data: {"chunk":"  ## Title\\n\\n| A | B |\\n| - | - |\\n"}\n\n',
      ]),
    }) as Response;
    globalThis.fetch = fetchMock;

    const onFinalAnswerDelta = vi.fn();

    try {
      await aiApi.chatStream(
        { message: 'hi', context: { scene: 'global' } },
        { onFinalAnswerDelta },
      );
    } finally {
      globalThis.fetch = originalFetch;
    }

    expect(onFinalAnswerDelta).toHaveBeenCalledWith(
      expect.objectContaining({
        chunk: '  ## Title\n\n| A | B |\n| - | - |\n',
      }),
    );
  });

  it('passes through plain text', () => {
    expect(normalizeVisibleStreamChunk('你好，平台助手')).toBe('你好，平台助手');
  });

  it('hides internal steps envelope', () => {
    expect(normalizeVisibleStreamChunk('{"steps":["a","b"]}')).toBe('');
  });

  it('unwraps response envelope', () => {
    expect(normalizeVisibleStreamChunk('{"response":"你好！我是平台助手。"}')).toBe('你好！我是平台助手。');
  });

  it('keeps ordinary json content', () => {
    const json = '{"name":"nginx","replicas":3}';
    expect(normalizeVisibleStreamChunk(json)).toBe(json);
  });
});
