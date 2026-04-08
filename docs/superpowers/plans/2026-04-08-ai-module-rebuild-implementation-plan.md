# AI Module Rebuild Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the new `Copilot + Runtime` AI architecture from the approved design set, replacing control-flow glue with canonical runtime/event/policy boundaries.

**Architecture:** Build from core domain and runtime first, then tool/policy pipeline, then event/projection, then gateway/UI integration, then evaluation harness and stabilization. Keep canonical truth chain strict: `runtime facts -> canonical events -> derived projections -> API/UI`.

**Tech Stack:** Go backend (`internal/service/ai`, `internal/ai`, `api/ai/v1`), React frontend (`web/src/components/AI`, `web/src/api/modules/ai.ts`), MySQL/GORM DAOs, existing AI scripts under `script/ai`.

---

## Implementation Guardrails

- No backward-compatibility-first workarounds; prefer clean replacement.
- No new dependency without explicit approval.
- Any new framework/library usage must be preceded by current-doc verification (`ctx7` first).
- Keep `replan` reserved for new run attempts; same-run changes use `refinement/reassessment`.
- Canonical cursor for continuity is persisted `event_id`.

## Phase 0: Baseline and Safety Net

### Task 0.1: Lock contract test baseline

**Files:**
- Modify: `internal/service/ai/handler/routes_contract_test.go`
- Modify: `web/src/api/modules/ai.contract.test.ts`

- [ ] Add assertions for canonical cursor semantics (`last_event_id` / `ack_event_id`) and approval response field aliases.
- [ ] Add assertions that transport-only events are non-authoritative.
- [ ] Run:
  - `go test ./internal/service/ai/handler -run Contract -v`
  - `cd web && npm test -- ai.contract.test.ts`

### Task 0.2: Freeze existing regressions for approval/resume

**Files:**
- Modify: `internal/service/ai/logic/approval_worker_test.go`
- Modify: `internal/service/ai/logic/run_tailer_test.go`

- [ ] Add tests that confirm:
  - resume failure is terminal for current run
  - retryable recovery path remains non-terminal worker-only
  - unknown/expired cursor fails explicitly
- [ ] Run:
  - `go test ./internal/service/ai/logic -run "Approval|Tailer|Resume" -v`

## Phase 1: Core Runtime Boundary Refactor

### Task 1.1: Introduce explicit run-attempt semantics

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/chat/service.go`
- Modify: `internal/service/ai/approval/service.go`

- [ ] Extract attempt identity decision points from monolithic chat flow.
- [ ] Enforce rule: resume keeps same run; retry/replan creates new run.
- [ ] Keep API behavior stable where contract still expects same endpoints.
- [ ] Run:
  - `go test ./internal/service/ai/... -run "Chat|Run|Approval" -v`

### Task 1.2: Align runtime state transitions with spec

**Files:**
- Modify: `internal/service/ai/logic/iterator_processor.go`
- Modify: `internal/service/ai/logic/approval_orchestrator.go`
- Modify: `internal/service/ai/logic/run_resume_projection.go`

- [ ] Ensure transitions follow: `created -> routing -> planning/executing -> waiting_approval -> resuming -> terminal`.
- [ ] Remove ambiguous same-run `replan` naming in code paths/events.
- [ ] Emit explicit failure categories for invalid checkpoint/resume.
- [ ] Run:
  - `go test ./internal/service/ai/logic -run "Iterator|Approval|Resume|Projection" -v`

## Phase 2: Tool Contract and Policy Pipeline

### Task 2.1: Normalize tool contract metadata path

**Files:**
- Modify: `internal/ai/agents/*/tools.go`
- Modify: `internal/ai/agents/orchestrator/tools.go`
- Modify: `internal/service/ai/logic/tool_error_classifier.go`

- [ ] Standardize tool metadata fields required by policy and replay.
- [ ] Add clear `use_when`/`dont_use_when` and evidence requirements.
- [ ] Remove prompt-only tool gating logic where deterministic checks exist.
- [ ] Run:
  - `go test ./internal/ai/... -run "Tool|Policy|Host" -v`

### Task 2.2: Enforce full policy decision model

**Files:**
- Modify: `internal/ai/common/middleware/approval.go`
- Modify: `internal/service/ai/logic/approval_orchestrator.go`
- Modify: `internal/service/ai/logic/errors.go`

- [ ] Enforce and persist all outcomes: `allow`, `deny`, `dry_run_only`, `require_approval`.
- [ ] Ensure validation/schema failures map deterministically to policy denial reason.
- [ ] Keep alternate path decision in runtime layer, not policy ownership.
- [ ] Run:
  - `go test ./internal/ai/common/middleware -v`
  - `go test ./internal/service/ai/logic -run "Approval|Policy|Error" -v`

## Phase 3: Event and Projection Canonicalization

### Task 3.1: Canonical event envelope hardening

**Files:**
- Modify: `internal/ai/runtime/event_types.go`
- Modify: `internal/ai/runtime/events.go`
- Modify: `internal/service/ai/logic/approval_event_contract.go`

- [ ] Ensure canonical events carry consistent identity/order/causality fields.
- [ ] Enforce `turn_refined` terminology and remove `turn_replanned` drift.
- [ ] Ensure causality precedence (`caused_by_event_id` vs `parent_event_id`) is explicit in payload builders.
- [ ] Run:
  - `go test ./internal/ai/runtime -v`
  - `go test ./internal/service/ai/logic -run "Event|Contract" -v`

### Task 3.2: Replay + SSE cursor convergence

**Files:**
- Modify: `internal/service/ai/logic/run_tailer.go`
- Modify: `internal/service/ai/chat/sse_writer.go`
- Modify: `internal/service/ai/handler/sse_writer_test.go`

- [ ] Use canonical persisted `event_id` as attach/reconnect authority.
- [ ] Keep summary-only transport frames from advancing canonical cursor.
- [ ] Ensure unknown/expired cursor returns explicit error.
- [ ] Run:
  - `go test ./internal/service/ai/handler -run SSE -v`
  - `go test ./internal/service/ai/logic -run Tailer -v`

## Phase 4: Gateway Contract Alignment

### Task 4.1: Approval field naming + alias compatibility

**Files:**
- Modify: `api/ai/v1/ai.go`
- Modify: `internal/service/ai/approval/handler.go`
- Modify: `web/src/api/modules/ai.ts`

- [ ] Make canonical external names consistent (`tool_call_id`, `arguments_json`, `preview_json`) with alias mapping from legacy fields.
- [ ] Keep transport compatibility while documenting deprecation behavior.
- [ ] Run:
  - `go test ./internal/service/ai/handler -run Approval -v`
  - `cd web && npm test -- ai.approval.test.ts`

### Task 4.2: Projection response carries continuity marker

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `api/ai/v1/ai.go`
- Modify: `web/src/api/modules/ai.ts`

- [ ] Add `ack_event_id` (canonical continuity marker) in projection response.
- [ ] Keep `next_cursor` for block paging but prohibit treating it as event cursor.
- [ ] Run:
  - `go test ./internal/service/ai/logic -run Projection -v`
  - `cd web && npm test -- ai.streamChunk.test.ts`

## Phase 5: Copilot UI Read-Model Integration

### Task 5.1: Separate transcript vs run-projection rendering

**Files:**
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/historyProjection.ts`

- [ ] Assistant operational blocks must come from projection; user turns from session transcript layer.
- [ ] Add explicit `status` block rendering and first-class `waiting_approval` view.
- [ ] Ensure resume-failed retry starts new run attempt, never mutates failed run.
- [ ] Run:
  - `cd web && npm test -- components/AI`

### Task 5.2: Approval card completeness and hydration

**Files:**
- Modify: `web/src/components/AI/ToolReference.tsx`
- Modify: `web/src/pages/Deployment/ApprovalCenterPage.tsx`
- Modify: `web/src/api/modules/ai.ts`

- [ ] Approval card requires reason, expected impact, and resume/checkpoint context.
- [ ] Treat `tool_approval` SSE payload as partial; hydrate canonical snapshot via approval detail API.
- [ ] Run:
  - `cd web && npm test -- ai.approval.test.ts`

## Phase 6: Evaluation Harness Implementation

### Task 6.1: Serialize evaluation case schema

**Files:**
- Create: `internal/ai/eval/cases/schema.go`
- Create: `internal/ai/eval/cases/loader.go`
- Create: `internal/ai/eval/cases/schema_test.go`

- [ ] Implement schema with explicit fields for:
  - canonical facts vs derived views
  - route/tool/outcome expectations
  - `expected_runtime_failure_category`
  - `expected_judge_failure_category`
- [ ] Run:
  - `go test ./internal/ai/eval/cases -v`

### Task 6.2: Implement judge pipeline + aggregation

**Files:**
- Create: `internal/ai/eval/judges/route_judge.go`
- Create: `internal/ai/eval/judges/tool_judge.go`
- Create: `internal/ai/eval/judges/outcome_judge.go`
- Create: `internal/ai/eval/judges/transcript_judge.go`
- Create: `internal/ai/eval/runner/aggregate.go`

- [ ] Enforce run order: route -> tool -> outcome -> transcript.
- [ ] Default gating: route/tool/outcome blocking; transcript advisory unless case opts in.
- [ ] Persist per-judge verdict and aggregated verdict.
- [ ] Run:
  - `go test ./internal/ai/eval/... -v`

## Phase 7: Stabilization and Rollout

### Task 7.1: Cross-module regression pass

**Files:**
- Modify: `script/ai/check_contract_parity.sh`
- Modify: `script/ai/validate_multi_domain_rollout.sh`

- [ ] Add checks for cursor semantics, approval snapshot completeness, and `replan` semantics.
- [ ] Keep scripts as validation wrappers, not truth generators.
- [ ] Run:
  - `bash script/ai/check_contract_parity.sh`
  - `bash script/ai/validate_multi_domain_rollout.sh`

### Task 7.2: Remove obsolete control-flow glue

**Files:**
- Modify/Delete: `internal/service/ai/logic/*` (only stale glue after replacements are proven)
- Modify/Delete: legacy projection duplication in frontend AI components

- [ ] Remove dead paths only after tests and contracts pass.
- [ ] Verify no hidden dependency on deleted glue remains.
- [ ] Run:
  - `go test ./internal/service/ai/... -v`
  - `cd web && npm test -- ai`

## Verification Gates (must pass before merge)

- `go test ./internal/service/ai/...`
- `go test ./internal/ai/...`
- `go test ./internal/ai/eval/...` (new harness packages)
- `cd web && npm test -- ai`
- `bash script/ai/check_contract_parity.sh`
- `bash script/ai/validate_multi_domain_rollout.sh`

## Delivery Notes

- Keep commits small and phase-aligned.
- Each phase commit should include:
  - changed files
  - what ambiguity was removed
  - what tests were run
  - what remains not tested

