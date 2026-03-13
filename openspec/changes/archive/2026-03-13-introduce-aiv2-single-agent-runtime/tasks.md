## 1. AIV2 Runtime Skeleton

- [x] 1.1 Create `internal/aiv2` module structure for runtime, agent builder, approval, stream projection, and tool registry
- [x] 1.2 Implement a single `ChatModelAgent + Runner` runtime entrypoint for chat streaming
- [x] 1.3 Add runtime selection wiring so the AI gateway can route requests to legacy `internal/ai` or `internal/aiv2`
- [x] 1.4 Add configuration or rollout control for enabling `aiv2` without removing the legacy runtime

## 2. Unified Tools and Middleware

- [x] 2.1 Build a unified `aiv2` tool registry that reuses existing host, kubernetes, service, delivery, and observability tools
- [x] 2.2 Remove `expert agent as tool` from the `aiv2` execution path and connect the main agent directly to the unified toolset
- [x] 2.3 Implement `ContextInjectMiddleware` for scene, page, selected resources, and project context injection
- [x] 2.4 Implement `StreamingProjectorMiddleware` to project agent events into the existing SSE contract
- [x] 2.5 Implement `ObservabilityMiddleware` to record runtime, tool, interrupt, and resume telemetry for `aiv2`

## 3. Human-in-the-Loop and Resume

- [x] 3.1 Define tool policy metadata for readonly vs mutating tools and approval requirements
- [x] 3.2 Wrap mutating tools with approval-aware gating that interrupts before mutation
- [x] 3.3 Implement pending tool invocation persistence for checkpoint identity, tool name, tool args, summary, and turn identity
- [x] 3.4 Implement `ResumeWithParams`-backed resume flow for approved and rejected pending actions
- [x] 3.5 Add backend tests covering approval interrupt, approved resume continuation, and rejected terminal completion in `aiv2`

## 4. API, Session Replay, and Frontend Compatibility

- [x] 4.1 Update `api/ai/v1` contracts to support `aiv2` runtime metadata and replay compatibility
- [x] 4.2 Ensure `aiv2` session persistence populates both structured turn replay and legacy-compatible message fields
- [x] 4.3 Preserve `/api/v1/ai/chat`, `/api/v1/ai/resume/step`, and `/api/v1/ai/resume/step/stream` while routing to the selected runtime
- [x] 4.4 Update frontend execution rendering to prefer tool-call chain semantics when events come from `aiv2`
- [x] 4.5 Verify historical replay remains summary-first and markdown-compatible for both legacy and `aiv2` sessions

## 5. Rollout and Verification

- [x] 5.1 Add integration tests comparing legacy and `aiv2` chat flows at the gateway boundary
- [x] 5.2 Add observability assertions or metrics verification so `aiv2` runs are distinguishable from legacy runs
- [x] 5.3 Document migration and rollback behavior for enabling or disabling `aiv2`
- [x] 5.4 Validate the change with `openspec validate --changes "introduce-aiv2-single-agent-runtime" --json`
