import { beforeEach, describe, expect, it } from 'vitest';
import { scopeStore } from '../app/scope/scopeStore';
import { buildContextualFetchInit, getRequestContextHeaders } from './requestContext';

describe('requestContext', () => {
  beforeEach(() => {
    localStorage.clear();
    scopeStore.clearScope();
  });

  it('builds request headers from scopeStore and not from ad hoc localStorage keys', () => {
    scopeStore.setScope({ projectId: '42', teamId: '7' });

    expect(getRequestContextHeaders()).toEqual({
      'X-Project-ID': '42',
      'X-Team-ID': '7',
    });
  });

  it('adds credentials include and merges caller headers with scope headers', () => {
    scopeStore.setScope({ projectId: '42' });

    expect(
      buildContextualFetchInit({
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      })
    ).toEqual({
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        'X-Project-ID': '42',
      },
    });
  });
});
