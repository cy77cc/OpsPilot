import { consumeAIStream, type A2UIStreamHandlers } from './shared';
import type { FormAssistRequest } from '../types/formAssist';
import { buildContextualFetchInit } from '../../../api/requestContext';

export async function formAssistStream(
  params: FormAssistRequest,
  handlers: A2UIStreamHandlers,
  signal?: AbortSignal
): Promise<void> {
  const base = import.meta.env.VITE_API_BASE || '/api/v1';
  const controller = new AbortController();

  const abortFromCaller = () => controller.abort();
  signal?.addEventListener('abort', abortFromCaller, { once: true });

  try {
    const response = await fetch(`${base}/ai/assist/form/stream`, buildContextualFetchInit({
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(params),
      signal: controller.signal,
    }));

    await consumeAIStream(response, handlers);
  } finally {
    signal?.removeEventListener('abort', abortFromCaller);
  }
}

export const assistApi = { formAssistStream };
