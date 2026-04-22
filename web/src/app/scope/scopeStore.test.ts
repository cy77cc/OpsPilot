import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createScopeStore } from './scopeStore';

describe('createScopeStore', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('hydrates persisted project/team scope and notifies subscribers on update', () => {
    localStorage.setItem('opspilot.scope', JSON.stringify({ projectId: '42', teamId: '7' }));

    const store = createScopeStore('opspilot.scope');
    const listener = vi.fn();
    const unsubscribe = store.subscribe(listener);

    expect(store.getSnapshot()).toEqual({ projectId: '42', teamId: '7' });

    store.setScope({ projectId: '84' });

    expect(store.getSnapshot()).toEqual({ projectId: '84', teamId: '7' });
    expect(JSON.parse(localStorage.getItem('opspilot.scope') || '{}')).toEqual({ projectId: '84', teamId: '7' });
    expect(listener).toHaveBeenCalledTimes(1);

    unsubscribe();
  });
});
