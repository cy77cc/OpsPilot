import { describe, expect, it } from 'vitest';
import { normalizeVisibleStreamChunk } from './ai';

describe('normalizeVisibleStreamChunk', () => {
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
