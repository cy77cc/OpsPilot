import { scopeStore } from '../app/scope/scopeStore';

const toHeaderObject = (headers?: HeadersInit): Record<string, string> => {
  if (!headers) {
    return {};
  }
  if (headers instanceof Headers) {
    return Object.fromEntries(headers.entries());
  }
  if (Array.isArray(headers)) {
    return Object.fromEntries(headers);
  }
  return { ...headers };
};

export function getRequestContextHeaders(): Record<string, string> {
  const { projectId, teamId } = scopeStore.getSnapshot();

  return {
    ...(projectId ? { 'X-Project-ID': projectId } : {}),
    ...(teamId ? { 'X-Team-ID': teamId } : {}),
  };
}

export function buildContextualFetchInit(init: RequestInit = {}): RequestInit {
  return {
    ...init,
    credentials: 'include',
    headers: {
      ...toHeaderObject(init.headers),
      ...getRequestContextHeaders(),
    },
  };
}
