## 1. Chat contract and scene-awareness

- [x] 1.1 Extend the AI chat request contract and backend handler/logic to accept optional `scene` and structured scene context fields.
- [x] 1.2 Update AI session creation and retrieval flows so new conversations persist under the resolved scene instead of always defaulting to `ai`.
- [x] 1.3 Define how scene prompts, scene config, and tool constraints are loaded and injected as hidden augmentation for chat requests.
- [x] 1.4 Add backend tests covering scene-aware chat request handling, session persistence, and scene-filtered session retrieval.

## 2. Frontend provider and drawer surface

- [x] 2.1 Implement a `PlatformChatProvider` that adapts `aiApi.chatStream()` and existing session APIs to Ant Design X chat hooks.
- [x] 2.2 Build the copilot drawer surface in `web/src/components/AI` with conversation list, sender, quick prompts, session switching, and streaming message states.
- [x] 2.3 Integrate the drawer open/close state into `AppLayout` and update `AICopilotButton` so the primary entry opens the in-shell copilot surface.
- [x] 2.4 Add frontend tests for provider behavior, drawer interaction, and shell-safe failure isolation.

## 3. Markdown and scene-aware UX

- [x] 3.1 Render assistant message bodies through `@ant-design/x-markdown` and remove any ad hoc markdown-to-HTML formatting in the main assistant response path.
- [x] 3.2 Add supported markdown customizations for AI-specific rendering needs such as think blocks, code blocks, and tables without replacing the primary renderer.
- [x] 3.3 Implement scene resolution in the AI surface so the active route/page contributes scene identity, entity context, and scene-relevant quick prompts.
- [x] 3.4 Verify the copilot experience against the current SSE event set and ensure unsupported richer events degrade gracefully without blocking the first release.
