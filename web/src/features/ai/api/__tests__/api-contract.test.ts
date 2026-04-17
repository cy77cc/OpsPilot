import { describe, expect, it } from 'vitest';
import { listUnsupportedMethods } from '../runApi';

describe('ai api contract', () => {
  it('does not expose unsupported methods', () => {
    expect(listUnsupportedMethods()).toEqual([]);
  });
});
