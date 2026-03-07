// @vitest-environment node

import { afterAll, describe, expect, it } from 'vitest';
import { build, type InlineConfig, type Rollup } from 'vite';
import { rmSync } from 'node:fs';
import { resolve } from 'node:path';
import viteConfig from '../../../vite.config';

const outDir = resolve(process.cwd(), 'dist-test-ai-surface');

describe('AI surface production build', () => {
  afterAll(() => {
    rmSync(outDir, { recursive: true, force: true });
  });

  it('keeps the AI surface in a separate non-entry chunk', async () => {
    const result = await build({
      ...viteConfig,
      logLevel: 'silent',
      build: {
        ...viteConfig.build,
        outDir,
        write: false,
        minify: false,
      },
    } as InlineConfig);

    const outputs = Array.isArray(result) ? result : [result];
    const chunks = outputs
      .flatMap((output) => ('output' in output ? output.output : []))
      .filter((asset): asset is Rollup.OutputChunk => asset.type === 'chunk');

    const aiChunk = chunks.find((chunk) =>
      Object.keys(chunk.modules).some((id) =>
        id.includes('/src/components/AI/CopilotSurface.tsx') || id.includes('/src/components/AI/Copilot.tsx'),
      ),
    );

    expect(aiChunk).toBeDefined();
    expect(aiChunk?.isEntry).toBe(false);

    const appEntryChunk = chunks.find((chunk) => chunk.isEntry);
    expect(appEntryChunk).toBeDefined();
    expect(
      Object.keys(appEntryChunk?.modules ?? {}).some((id) => id.includes('/src/components/AI/Copilot.tsx')),
    ).toBe(false);
  }, 120000);
});
