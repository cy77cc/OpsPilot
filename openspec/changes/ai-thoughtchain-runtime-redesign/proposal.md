## Why

The current AI chat runtime mixes legacy `turn/block`, `phase/step`, detached approval flows, and partially reconstructed `thoughtChain` rendering, which has caused broken chain visualization, approval exits, false unavailable states, and fragmented observability. We need to replace this overlap now so the AI module has one clean runtime contract before more fixes continue accumulating on obsolete architecture.

## What Changes

- Redefine `thoughtChain` as the only primary runtime protocol for AI chat execution, replay, and rendering.
- **BREAKING** Remove legacy `turn/block`, `phase/step`, and compatibility-only streaming semantics from the AI chat primary path.
- **BREAKING** Replace detached approval trigger and resume semantics with one unified chain approval decision flow.
- Upgrade the AI assistant frontend to render one native `thoughtChain` timeline with dedicated node cards for `plan`, `step`, `tool`, `approval`, `replan`, and `answer`.
- Preserve markdown fidelity during streaming by removing trimming behavior that mutates user-visible SSE payloads.
- Persist runtime-first replay state, including final-answer markdown, so completed assistant responses survive conversation restore.
- Redesign ThoughtChain node payload semantics so `plan`/`replan` render structured steps, `tool` nodes render beautified raw results, and detailed phase summaries do not leak into the wrong rendering surface.
- Fix new-conversation recommended prompt handling so an in-flight chain never falls into a false "AI assistant unavailable" state before the first server event arrives.
- Add callback-based runtime telemetry for chain and node lifecycle and export it to the existing Prometheus integration.
- Add regression coverage to prevent legacy protocol concepts from re-entering the primary chat flow.

## Capabilities

### New Capabilities
- `ai-thoughtchain-observability`: Defines callback-driven chain/node telemetry and Prometheus export requirements for the new thoughtChain runtime.

### Modified Capabilities
- `ai-thoughtchain-runtime`: Replace transitional chain-native compatibility semantics with a thoughtChain-first runtime contract and node model.
- `ai-streaming-events`: Change AI chat streaming requirements so the primary path emits only thoughtChain lifecycle events and removes legacy phase/block event dependence.
- `ai-pre-execution-approval-gate`: Change approval behavior so approval is modeled as a first-class thoughtChain node with one decision API.
- `ai-chat-session-contract`: Change session replay expectations so live and restored assistant responses are both reconstructed from the same thoughtChain model.
- `prometheus-integration`: Extend Prometheus requirements to ingest thoughtChain runtime metrics emitted by AI lifecycle callbacks.

## Impact

- Backend AI orchestration, SSE emission, approval routing, callback hooks, and replay persistence.
- Frontend AI chat store, assistant message rendering, approval interaction, and recommended prompt submission flow.
- AI chat API contracts for streaming and approval decision handling.
- Monitoring and telemetry integration through existing Prometheus support.
