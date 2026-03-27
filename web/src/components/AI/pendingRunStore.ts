import type { PendingRunMetadata } from './types';

const STORAGE_KEY = 'ai.pending-runs';

type PendingRunMap = Record<string, PendingRunMetadata>;

function readPendingRunMap(): PendingRunMap {
  if (typeof window === 'undefined' || !window.localStorage) {
    return {};
  }

  const raw = window.localStorage.getItem(STORAGE_KEY);
  if (!raw) {
    return {};
  }

  try {
    return JSON.parse(raw) as PendingRunMap;
  } catch {
    return {};
  }
}

function writePendingRunMap(next: PendingRunMap): void {
  if (typeof window === 'undefined' || !window.localStorage) {
    return;
  }

  if (Object.keys(next).length === 0) {
    window.localStorage.removeItem(STORAGE_KEY);
    return;
  }

  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
}

export function listPendingRuns(): PendingRunMetadata[] {
  return Object.values(readPendingRunMap()).sort((left, right) =>
    (right.updatedAt || '').localeCompare(left.updatedAt || ''),
  );
}

export function getPendingRun(runId: string): PendingRunMetadata | null {
  if (!runId) {
    return null;
  }
  return readPendingRunMap()[runId] || null;
}

export function upsertPendingRun(partial: PendingRunMetadata): PendingRunMetadata {
  const current = readPendingRunMap();
  const previous = partial.runId ? current[partial.runId] : undefined;
  const nextRecord: PendingRunMetadata = {
    ...(previous || {}),
    ...partial,
    updatedAt: partial.updatedAt || new Date().toISOString(),
  };
  current[nextRecord.runId] = nextRecord;
  writePendingRunMap(current);
  return nextRecord;
}

export function removePendingRun(runId: string): void {
  if (!runId) {
    return;
  }
  const current = readPendingRunMap();
  if (!current[runId]) {
    return;
  }
  delete current[runId];
  writePendingRunMap(current);
}

export function clearPendingRuns(): void {
  writePendingRunMap({});
}
