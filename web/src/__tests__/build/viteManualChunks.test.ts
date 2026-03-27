// @vitest-environment node

import { describe, expect, it } from 'vitest';
import viteConfig from '../../../vite.config';

describe('vite manualChunks', () => {
  it('keeps ant-design x code highlighter dependencies in the same chunk', () => {
    const output = viteConfig.build?.rollupOptions?.output;

    if (!output || Array.isArray(output) || typeof output.manualChunks !== 'function') {
      throw new Error('manualChunks is not configured');
    }

    const meta = {
      getModuleInfo: () => null,
      getModuleIds: function* () {},
    };

    const codeHighlighterChunk = output.manualChunks(
      '/root/project/k8s-manage/web/node_modules/@ant-design/x/es/code-highlighter/CodeHighlighter.js',
      meta,
    );
    const syntaxHighlighterChunk = output.manualChunks(
      '/root/project/k8s-manage/web/node_modules/react-syntax-highlighter/dist/cjs/styles/prism/index.js',
      meta,
    );

    expect(codeHighlighterChunk).toBe(syntaxHighlighterChunk);
  });
});
