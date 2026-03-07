## Why

The AI assistant surface is evolving from simple markdown chat into a rich message surface with code blocks, thinking blocks, recommendations, and future tool/result content. The current implementation concentrates transport, rendering, and feature-specific UI inside the Copilot entry path, which increases the risk of white-screen failures and makes new rich message types expensive to add safely.

## What Changes

- Introduce a hardened AI message surface model that separates AI panel loading, message normalization, and block rendering concerns.
- Add failure-isolation requirements so AI surface initialization failures do not break the main application shell.
- Add block-level degradation requirements so rich message rendering failures fall back to safe textual output instead of breaking the whole panel.
- Define an extensible AI message block model for markdown, code, thinking, recommendations, and future rich message types.
- Add verification requirements for production-like runtime smoke coverage that can catch white-screen regressions caused by frontend chunking or runtime initialization order.
- Adjust frontend build/runtime expectations so AI-rich rendering dependencies are isolated by feature boundaries rather than fragile vendor taxonomy chunking.

## Capabilities

### New Capabilities
- `ai-message-surface`: Defines the normalized AI message block model, renderer isolation, degradation behavior, and future-rich-content extension contract.

### Modified Capabilities
- `ai-assistant-drawer`: Tighten requirements so AI surface failures degrade locally and never blank the main application shell.
- `testing-baseline`: Add frontend runtime smoke expectations that detect white-screen regressions and fatal browser initialization errors before release.

## Impact

- Affected frontend areas: `web/src/components/AI/*`, `web/src/components/Layout/AppLayout.tsx`, message rendering paths, lazy-loading boundaries, and runtime error isolation points.
- Affected build/runtime concerns: Vite chunking strategy, AI-surface dependency loading, and production-smoke verification workflow.
- Affected product behavior: AI panel fallback states, per-message/per-block degradation, and future AI rich-content extensibility.
