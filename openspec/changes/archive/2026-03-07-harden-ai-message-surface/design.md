## Context

The current AI assistant entry path is mounted from `web/src/components/Layout/AppLayout.tsx` through `web/src/components/AI/AICopilotButton.tsx`, `web/src/components/AI/AIAssistantDrawer.tsx`, and finally `web/src/components/AI/Copilot.tsx`. `Copilot.tsx` currently combines panel orchestration, SSE transport parsing, message state, markdown rendering, code highlighting, thinking presentation, recommendations, and message actions in one dependency-heavy path.

This works while the AI surface remains simple, but recent rich-rendering changes showed that adding a new rendering dependency can destabilize production runtime initialization. The product direction also requires more message types in the near term, including code, thinking, recommendations, tool results, and future structured content. The design therefore needs to solve both immediate runtime hardening and longer-term extensibility.

## Goals / Non-Goals

**Goals:**
- Keep the main application shell usable even if the AI panel fails to initialize.
- Ensure a single rich-render block failure degrades to safe fallback output instead of breaking the full AI panel.
- Introduce a normalized AI message block model that supports current and near-term message types.
- Move runtime/build decisions toward feature-boundary isolation rather than fragile vendor chunk taxonomy.
- Add production-like frontend smoke verification for white-screen regressions and fatal runtime initialization errors.

**Non-Goals:**
- Rebuild the backend AI protocol in one release.
- Replace all markdown rendering in the frontend.
- Implement every future rich block type immediately.
- Eliminate all manual chunking across the application; only AI-surface-sensitive boundaries are in scope.

## Decisions

### 1. Split the AI surface into three concerns: entry boundary, message normalization, and block rendering

The AI assistant should evolve from a single rich component into:
- an AI surface boundary responsible for lazy loading and local failure fallback,
- a message normalization layer that converts raw message content into typed blocks,
- a block renderer layer that renders markdown, code, thinking, recommendations, and future blocks independently.

This separates transport/orchestration concerns from rich-content concerns and makes future content types additive rather than invasive.

Alternatives considered:
- Keep extending `Copilot.tsx` with more conditional renderers: fastest short-term, but it keeps the current failure propagation path and coupling.
- Fully switch to a backend-defined document AST immediately: strongest long-term contract, but too large a migration for the immediate hardening goal.

### 2. Use a front-end block schema first, with optional future backend alignment

The first step should normalize raw AI messages into a front-end-owned block model rather than immediately changing the backend response contract. This allows current SSE/string payloads to remain compatible while the frontend establishes a stable render contract.

The initial block model should cover:
- markdown
- code
- thinking
- recommendations
- tool-result / future structured blocks
- fallback / error blocks

Alternatives considered:
- Stay purely string-based: simplest, but future rich content becomes an accumulation of parser hacks and special cases.
- Make the backend authoritative for block structure immediately: desirable later, but it would unnecessarily couple runtime hardening to protocol migration.

### 3. Require two isolation boundaries: panel-level and block-level

The AI panel must fail locally without taking down the main shell, and any individual block renderer must fail locally without breaking the full AI message list. This means:
- `AppLayout`-level shell stability is a hard requirement,
- AI panel initialization gets a local fallback state,
- each block renderer provides safe degradation behavior.

Alternatives considered:
- Only add a top-level error boundary: protects the shell, but a single bad block would still erase the whole panel.
- Only add block fallbacks: improves message resilience, but does not protect the application shell from initialization failures.

### 4. Shift build strategy from vendor taxonomy to feature boundary stability

The AI surface should no longer depend on manual chunk groupings that split tightly coupled runtime dependencies by library family. Instead, build partitioning should prefer:
- lazy loading the AI surface as a feature,
- isolating truly independent heavy modules,
- avoiding manual chunk partitions across tightly coupled React/Ant Design/AI render stacks.

Alternatives considered:
- Keep refining vendor chunk taxonomy: tempting for bundle organization, but brittle because runtime dependencies do not respect taxonomy boundaries.
- Remove all manual chunking globally: safe but unnecessarily broad for the targeted problem.

### 5. Add production-like frontend runtime smoke verification

The verification baseline should include a browser/runtime smoke step that proves:
- the production build loads,
- the application shell renders,
- no fatal page/module initialization error occurs,
- AI-surface failures do not blank the shell.

Alternatives considered:
- Rely on build success plus unit tests: insufficient, because the current failure mode appears after bundling and browser initialization.
- Add only full end-to-end journey tests: valuable, but slower and less targeted than a dedicated runtime smoke gate.

## Risks / Trade-offs

- [Front-end block schema may drift from backend message semantics] → Mitigation: treat the block schema as an internal normalization contract first, and explicitly document future backend alignment as a later step.
- [Introducing boundaries may increase component count and indirection] → Mitigation: accept small structural complexity in exchange for clear failure containment and additive extensibility.
- [Lazy-loading the AI surface may slightly delay panel open time] → Mitigation: prefer predictable shell stability over eager loading; optimize panel loading separately if needed.
- [Feature-boundary chunking may produce larger chunks than fine-grained vendor splitting] → Mitigation: prioritize runtime safety first, then optimize with measured boundaries rather than taxonomy-based splits.
- [Smoke verification can become flaky if over-scoped] → Mitigation: keep the smoke check narrow: shell renders, no fatal runtime error, AI failure degrades locally.

## Migration Plan

1. Introduce the AI surface boundary and local fallback behavior around the current drawer/copilot entry path.
2. Refactor the current Copilot message rendering path into normalization and block-render responsibilities without changing the external user workflow.
3. Add the initial block schema and migrate existing markdown/code/thinking/recommendations rendering onto it.
4. Narrow AI-related chunking to stable feature boundaries and remove brittle vendor taxonomy assumptions from the AI path.
5. Add runtime smoke verification against production build output.

Rollback strategy:
- Revert to the previous AI entry path if a boundary refactor introduces regressions.
- Keep backend payload compatibility unchanged during the first phase so rollback remains front-end only.

## Open Questions

- Should tool execution results become first-class structured blocks in the first iteration, or remain transitional UI attached to assistant messages?
- Should the AI surface be loaded only on demand, or should some lightweight shell preload remain for responsiveness?
- At what point should the backend own the block schema instead of the frontend normalization layer?
