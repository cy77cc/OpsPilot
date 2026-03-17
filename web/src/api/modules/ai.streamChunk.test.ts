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
