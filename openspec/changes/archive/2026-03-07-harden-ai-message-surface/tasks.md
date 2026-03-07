## 1. AI surface boundary hardening

- [x] 1.1 Trace the current AI entry path from `AppLayout` through `AICopilotButton`, `AIAssistantDrawer`, and `Copilot` and document the runtime failure boundary to preserve.
- [x] 1.2 Introduce an AI-surface loading/error boundary so drawer initialization failures degrade locally instead of affecting the main application shell.
- [x] 1.3 Define and wire a user-visible fallback state for AI-surface initialization failures.

## 2. Message normalization and block rendering

- [x] 2.1 Define the initial frontend AI message block model for markdown, code, thinking, recommendations, and fallback/error blocks.
- [x] 2.2 Refactor the current `Copilot` message rendering path so raw assistant content is normalized into blocks before rich rendering.
- [x] 2.3 Split block rendering responsibilities into independent renderers that can fail and degrade locally.
- [x] 2.4 Add safe fallback behavior for failed or unsupported rich block types without breaking the rest of the message list.

## 3. Build stability and verification

- [x] 3.1 Rework AI-surface-sensitive build partitioning so rich rendering dependencies are isolated by stable feature boundaries instead of fragile vendor taxonomy chunks.
- [x] 3.2 Add frontend runtime smoke verification for production build load, shell availability, and fatal page/module initialization errors.
- [x] 3.3 Add verification coverage that proves AI-surface failures remain local to the panel and do not blank the main application shell.
