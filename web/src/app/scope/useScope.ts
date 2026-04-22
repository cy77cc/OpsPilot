import { useSyncExternalStore } from 'react';
import { scopeStore } from './scopeStore';

export function useScope() {
  const scope = useSyncExternalStore(scopeStore.subscribe, scopeStore.getSnapshot, scopeStore.getSnapshot);

  return {
    ...scope,
    scope,
    setScope: scopeStore.setScope.bind(scopeStore),
    setProjectId: scopeStore.setProjectId.bind(scopeStore),
    setTeamId: scopeStore.setTeamId.bind(scopeStore),
    clearScope: scopeStore.clearScope.bind(scopeStore),
  };
}
