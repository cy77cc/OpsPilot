export type ScopeState = {
  projectId?: string;
  teamId?: string;
};

const DEFAULT_STORAGE_KEY = 'opspilot.scope';

const normalizeScopeValue = (value?: string): string | undefined => {
  const trimmed = String(value || '').trim();
  return trimmed || undefined;
};

const normalizeScopeState = (state: Partial<ScopeState>): ScopeState => ({
  projectId: normalizeScopeValue(state.projectId),
  teamId: normalizeScopeValue(state.teamId),
});

const readPersistedScope = (storageKey: string): ScopeState => {
  if (typeof window === 'undefined') {
    return {};
  }

  const raw = window.localStorage.getItem(storageKey);
  if (raw) {
    try {
      const parsed = JSON.parse(raw) as Partial<ScopeState>;
      return normalizeScopeState(parsed);
    } catch {
      window.localStorage.removeItem(storageKey);
    }
  }

  // One-way migration for legacy per-key storage.
  return normalizeScopeState({
    projectId: window.localStorage.getItem('projectId') || undefined,
    teamId: window.localStorage.getItem('teamId') || undefined,
  });
};

const writePersistedScope = (storageKey: string, state: ScopeState) => {
  if (typeof window === 'undefined') {
    return;
  }

  if (!state.projectId && !state.teamId) {
    window.localStorage.removeItem(storageKey);
    return;
  }

  window.localStorage.setItem(storageKey, JSON.stringify(state));
};

export type ScopeStore = ReturnType<typeof createScopeStore>;

export function createScopeStore(storageKey = DEFAULT_STORAGE_KEY) {
  let state = readPersistedScope(storageKey);
  const listeners = new Set<() => void>();

  const notify = (previous: ScopeState, next: ScopeState) => {
    if (previous.projectId !== next.projectId && typeof window !== 'undefined') {
      window.dispatchEvent(
        new CustomEvent('project:changed', {
          detail: { projectId: next.projectId },
        })
      );
    }
    listeners.forEach((listener) => listener());
  };

  return {
    getSnapshot: () => state,
    subscribe(listener: () => void) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    setScope(next: Partial<ScopeState>) {
      const previous = state;
      state = normalizeScopeState({ ...state, ...next });
      writePersistedScope(storageKey, state);
      notify(previous, state);
    },
    setProjectId(projectId?: string) {
      this.setScope({ projectId });
    },
    setTeamId(teamId?: string) {
      this.setScope({ teamId });
    },
    clearScope() {
      const previous = state;
      state = {};
      writePersistedScope(storageKey, state);
      notify(previous, state);
    },
  };
}

export const scopeStore = createScopeStore();
