# AI Tool Failure Resilience Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make ordinary AI tool failures recoverable so the assistant can continue the current run and session, while only fatal runtime failures terminate the run.

**Architecture:** The implementation shifts command and tool execution from allow-or-block semantics to execute-and-report semantics. Tools should return structured success or error payloads, runtime should classify errors before deciding whether to stop the run, and the streaming/persistence/frontend layers must represent runs that complete with tool errors instead of treating every error as fatal.

**Tech Stack:** Go, Gin, GORM, CloudWeGo Eino ADK, TypeScript, React, existing AI SSE runtime and reply state adapters

---

**Related spec:** `docs/superpowers/specs/2026-03-19-ai-tool-failure-resilience-design.md`

## File Map

### Backend runtime and persistence

- Modify: `internal/service/ai/logic/logic.go`
  Responsibility: classify runtime vs tool errors during chat execution, persist the correct run status, keep session continuity.
- Modify: `internal/dao/ai/run_dao.go`
  Responsibility: support new terminal run statuses such as `completed_with_tool_errors` and `failed_runtime`.
- Modify: `internal/model/ai.go`
  Responsibility: confirm allowed run status vocabulary and any comments used by API/persistence consumers.
- Modify: `internal/ai/runtime/projector.go`
  Responsibility: convert recoverable failures into projected events instead of treating them as fatal.
- Modify: `internal/ai/runtime/normalize.go`
  Responsibility: normalize tool failures into a stable shape the projector can interpret.
- Modify: `internal/ai/runtime/events.go`
  Responsibility: document and expose event semantics for recoverable tool errors and degraded runs.
- Modify: `internal/ai/runtime/project.go`
  Responsibility: store runtime activity state that can express tool errors without marking the whole run fatal.

### Backend tools and prompts

- Modify: `internal/ai/tools/host/tools.go`
  Responsibility: remove the readonly command whitelist behavior for single-host readonly tools, return structured error payloads instead of hard-failing ordinary command attempts, keep only minimal platform-protection guards.
- Modify: `internal/ai/tools/tools.go`
  Responsibility: keep tool registration consistent if any shared wrappers or common helpers are introduced.
- Modify: `internal/ai/tools/middleware/approval.go`
  Responsibility: keep approval rejection semantics aligned with “recoverable tool outcome, not fatal run error”.
- Modify: `internal/ai/tools/common/approval.go`
  Responsibility: document approval-result semantics if the structured outcome shape is expanded.
- Modify: `internal/ai/agents/prompt/prompt.go`
  Responsibility: strengthen planner/executor prompts so models own risk judgment, prefer conservative actions, and recover after tool errors.

### Frontend streaming and rendering

- Modify: `web/src/api/modules/ai.ts`
  Responsibility: keep API event typings aligned with the new run states and tool result status shape.
- Modify: `web/src/components/AI/replyRuntime.ts`
  Responsibility: map recoverable tool failures and degraded completion into assistant reply runtime.
- Modify: `web/src/components/AI/a2uiState.ts`
  Responsibility: represent `tool_result(error)` and non-fatal degraded runs distinctly from fatal run errors.

### Tests

- Modify: `internal/service/ai/logic/logic_test.go`
  Responsibility: cover recoverable tool failures, fatal runtime failures, and session continuity.
- Modify: `internal/service/ai/handler/chat_test.go`
  Responsibility: verify SSE behavior does not terminate on recoverable tool failures.
- Modify: `internal/ai/runtime/projector_test.go`
  Responsibility: verify projector emits `tool_result(error)` and keeps the run alive.
- Modify: `internal/ai/runtime/normalize_test.go`
  Responsibility: verify error normalization rules.
- Modify: `internal/ai/tools/tools_test.go`
  Responsibility: verify tool descriptions and shared behavior if helper contracts change.
- Modify: `internal/ai/agents/prompt/prompt_test.go`
  Responsibility: verify prompt invariants if prompt tests snapshot important rules.
- Modify: `web/src/components/AI/replyRuntime.test.ts`
  Responsibility: verify reply runtime can display tool errors and degraded completion.
- Modify: `web/src/components/AI/__tests__/a2uiState.test.ts`
  Responsibility: verify state reducer distinguishes tool errors from fatal errors.

## Chunk 1: Runtime Status Vocabulary and Failing Tests

### Task 1: Define the new run and stream semantics in tests first

**Files:**
- Modify: `internal/service/ai/logic/logic_test.go`
- Modify: `internal/service/ai/handler/chat_test.go`
- Modify: `internal/ai/runtime/projector_test.go`
- Modify: `internal/ai/runtime/normalize_test.go`

- [ ] **Step 1: Add a failing logic test for recoverable tool failure**

Write a test in `internal/service/ai/logic/logic_test.go` that simulates a tool failure event and asserts:
- the final run status is `completed_with_tool_errors`
- the assistant message is persisted
- the method returns `nil`
- the session remains usable

- [ ] **Step 2: Run the failing logic test**

Run: `go test ./internal/service/ai/logic -run TestChatKeepsRunAliveOnRecoverableToolFailure -v`
Expected: FAIL because current logic marks the run failed or exits early.

- [ ] **Step 3: Add a failing logic test for fatal runtime failure**

Write a companion test in `internal/service/ai/logic/logic_test.go` that simulates a true runtime failure and asserts:
- the final run status is `failed_runtime`
- the assistant message status is `error`
- the session still allows future turns

- [ ] **Step 4: Run the fatal failure test**

Run: `go test ./internal/service/ai/logic -run TestChatMarksFatalRuntimeFailure -v`
Expected: FAIL because the current code only knows `failed`.

- [ ] **Step 5: Add a failing projector test**

In `internal/ai/runtime/projector_test.go`, add a test that feeds a recoverable tool error through the projector and expects:
- one `tool_result` event with an error status
- no terminal `error` event

- [ ] **Step 6: Run the projector test**

Run: `go test ./internal/ai/runtime -run TestProjectorEmitsToolResultErrorWithoutFatalError -v`
Expected: FAIL because the projector currently routes failures through fatal error handling.

- [ ] **Step 7: Add a failing normalize test**

In `internal/ai/runtime/normalize_test.go`, add a test that verifies recoverable tool error normalization yields a stable shape the projector can consume.

- [ ] **Step 8: Run the normalize test**

Run: `go test ./internal/ai/runtime -run TestNormalizeRecoverableToolError -v`
Expected: FAIL because there is no recoverable/fatal classification layer yet.

- [ ] **Step 9: Add a failing handler SSE test**

In `internal/service/ai/handler/chat_test.go`, add a test that verifies the SSE stream can include `tool_result(error)` and later `done` in the same run.

- [ ] **Step 10: Run the handler test**

Run: `go test ./internal/service/ai/handler -run TestChatStreamsRecoverableToolErrorAndDone -v`
Expected: FAIL because current behavior treats the error path as terminal.

- [ ] **Step 11: Commit the test-only red chunk**

```bash
git add internal/service/ai/logic/logic_test.go internal/service/ai/handler/chat_test.go internal/ai/runtime/projector_test.go internal/ai/runtime/normalize_test.go
git commit -m "test(ai): define recoverable tool failure behavior"
```

## Chunk 2: Runtime Classification and Persistence

### Task 2: Implement recoverable-vs-fatal error classification

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/dao/ai/run_dao.go`
- Modify: `internal/model/ai.go`
- Modify: `internal/ai/runtime/normalize.go`
- Modify: `internal/ai/runtime/projector.go`
- Modify: `internal/ai/runtime/events.go`
- Modify: `internal/ai/runtime/project.go`

- [ ] **Step 1: Introduce explicit run status vocabulary**

Update `internal/dao/ai/run_dao.go` and any related comments in `internal/model/ai.go` so the codebase recognizes:
- `completed`
- `completed_with_tool_errors`
- `failed_runtime`

- [ ] **Step 2: Update terminal status handling**

Adjust `isTerminalRunStatus` in `internal/dao/ai/run_dao.go` so the new statuses write `finished_at` correctly.

- [ ] **Step 3: Add minimal classification helpers**

In `internal/service/ai/logic/logic.go` or a small adjacent helper, add code that classifies event errors into:
- recoverable tool errors
- fatal runtime errors

Keep the first implementation narrow and explicit. Do not over-generalize.

- [ ] **Step 4: Change the main chat loop to stop treating all `event.Err` as fatal**

Refactor the event loop in `internal/service/ai/logic/logic.go` so recoverable tool errors:
- emit a structured tool error projection
- set an in-memory `hasToolErrors` flag
- do not early-return from `Chat`

- [ ] **Step 5: Preserve fatal runtime behavior**

Keep fatal runtime errors terminating the current run, but map them to `failed_runtime` instead of the generic `failed`.

- [ ] **Step 6: Teach normalization/projector about recoverable tool errors**

Update `internal/ai/runtime/normalize.go` and `internal/ai/runtime/projector.go` to support a normalized recoverable error path that emits `tool_result` with an error status instead of an `error` event.

- [ ] **Step 7: Record degraded runtime state**

Update `internal/ai/runtime/project.go` and `internal/ai/runtime/events.go` so persisted runtime/activity state can represent:
- a tool activity ending in error
- a run that finishes in degraded mode

- [ ] **Step 8: Persist the right final run status**

In `internal/service/ai/logic/logic.go`, set final run status to:
- `completed` when no tool errors occurred
- `completed_with_tool_errors` when the reply finished but one or more tool calls failed
- `failed_runtime` on fatal runtime failure

- [ ] **Step 9: Run backend unit tests for this chunk**

Run:
- `go test ./internal/ai/runtime -v`
- `go test ./internal/service/ai/logic -v`
- `go test ./internal/service/ai/handler -v`

Expected: PASS for the newly added tests and any existing runtime/chat tests.

- [ ] **Step 10: Commit the runtime classification chunk**

```bash
git add internal/service/ai/logic/logic.go internal/dao/ai/run_dao.go internal/model/ai.go internal/ai/runtime/normalize.go internal/ai/runtime/projector.go internal/ai/runtime/events.go internal/ai/runtime/project.go internal/service/ai/logic/logic_test.go internal/service/ai/handler/chat_test.go internal/ai/runtime/projector_test.go internal/ai/runtime/normalize_test.go
git commit -m "feat(ai): keep runs alive on recoverable tool failures"
```

## Chunk 3: Tool Contract and Prompt Policy

### Task 3: Remove brittle command gating and return structured tool errors

**Files:**
- Modify: `internal/ai/tools/host/tools.go`
- Modify: `internal/ai/tools/tools.go`
- Modify: `internal/ai/tools/middleware/approval.go`
- Modify: `internal/ai/tools/common/approval.go`
- Modify: `internal/ai/tools/tools_test.go`
- Modify: `internal/ai/agents/prompt/prompt.go`
- Modify: `internal/ai/agents/prompt/prompt_test.go`

- [ ] **Step 1: Add a failing host tool test for command whitelist removal**

Create or extend tests around `internal/ai/tools/host/tools.go` so a non-whitelisted but ordinary command no longer returns `command not allowed`.

- [ ] **Step 2: Run the host tool test**

Run: `go test ./internal/ai/tools -run TestHostExecDoesNotRejectUnknownCommandByWhitelist -v`
Expected: FAIL because the whitelist still blocks the command.

- [ ] **Step 3: Replace readonly whitelist checks in single-host command tools**

Refactor `internal/ai/tools/host/tools.go` so `host_ssh_exec_readonly`, `host_exec`, and `host_exec_by_target` no longer reject ordinary commands through `isReadonlyHostCommand`.

Keep only narrow hard blocks for commands that clearly threaten the platform host or runtime itself. Do not attempt comprehensive command semantics.

- [ ] **Step 4: Return structured error payloads instead of ordinary Go errors where possible**

Update command execution tools to prefer returning objects containing:
- `ok`
- `error_type`
- `retryable`
- `summary`
- `stdout`
- `stderr`
- `exit_code`

When a true infrastructure invariant is broken, Go error is still acceptable.

- [ ] **Step 5: Align approval rejection with recoverable tool outcomes**

Update `internal/ai/tools/middleware/approval.go` and `internal/ai/tools/common/approval.go` so approval rejection remains a recoverable tool outcome that downstream runtime code can project cleanly.

- [ ] **Step 6: Strengthen model prompts**

In `internal/ai/agents/prompt/prompt.go`, update change and diagnosis executor/planner prompts to explicitly require:
- conservative tool and command choice
- recovery after tool failure
- explanation of failed steps and next actions
- avoiding blind retries of the same failing command

- [ ] **Step 7: Update prompt/tool tests**

Modify `internal/ai/tools/tools_test.go` and `internal/ai/agents/prompt/prompt_test.go` to reflect the new contract and prompt invariants.

- [ ] **Step 8: Run backend tests for tools and prompts**

Run:
- `go test ./internal/ai/tools/... -v`
- `go test ./internal/ai/agents/prompt -v`

Expected: PASS, including the new host tool and prompt coverage.

- [ ] **Step 9: Commit the tool/prompt chunk**

```bash
git add internal/ai/tools/host/tools.go internal/ai/tools/tools.go internal/ai/tools/middleware/approval.go internal/ai/tools/common/approval.go internal/ai/tools/tools_test.go internal/ai/agents/prompt/prompt.go internal/ai/agents/prompt/prompt_test.go
git commit -m "feat(ai): make tool failures recoverable data"
```

## Chunk 4: Frontend State, Replay, and Verification

### Task 4: Surface tool errors without treating the run as dead

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/components/AI/replyRuntime.ts`
- Modify: `web/src/components/AI/a2uiState.ts`
- Modify: `web/src/components/AI/replyRuntime.test.ts`
- Modify: `web/src/components/AI/__tests__/a2uiState.test.ts`

- [ ] **Step 1: Add a failing frontend reducer test**

In `web/src/components/AI/__tests__/a2uiState.test.ts`, add a test that feeds:
- `tool_call`
- `tool_result(error)`
- `done`

and expects the final UI state to show a completed run with a failed tool step, not a fatal run failure.

- [ ] **Step 2: Run the reducer test**

Run: `pnpm --dir web test -- a2uiState`
Expected: FAIL because current state logic likely conflates tool error and terminal error.

- [ ] **Step 3: Add a failing reply runtime test**

In `web/src/components/AI/replyRuntime.test.ts`, add a test that verifies assistant reply runtime can render:
- a failed tool activity
- a degraded completion label
- continued assistant content after the failure

- [ ] **Step 4: Run the reply runtime test**

Run: `pnpm --dir web test -- replyRuntime`
Expected: FAIL because the current mapping does not model degraded completion cleanly.

- [ ] **Step 5: Update API typings**

In `web/src/api/modules/ai.ts`, add any missing typings for:
- `tool_result.status`
- new run status values
- degraded run state if needed

- [ ] **Step 6: Update reducer and runtime mapping**

Refactor `web/src/components/AI/a2uiState.ts` and `web/src/components/AI/replyRuntime.ts` so:
- tool errors remain visible as activity errors
- fatal runtime errors still show terminal failure
- completed runs with tool errors render a degraded but completed state

- [ ] **Step 7: Run frontend tests**

Run:
- `pnpm --dir web test -- a2uiState`
- `pnpm --dir web test -- replyRuntime`

Expected: PASS.

- [ ] **Step 8: Run focused cross-layer verification**

Run:
- `go test ./internal/ai/runtime ./internal/ai/tools/... ./internal/service/ai/logic ./internal/service/ai/handler ./internal/ai/agents/prompt -v`
- `pnpm --dir web test -- a2uiState replyRuntime`

Expected:
- Go tests PASS
- frontend tests PASS

- [ ] **Step 9: Perform a manual stream sanity check**

Start the app in the usual local dev setup and verify one scenario manually:
- ask the assistant to execute a command/tool call that will fail in an ordinary way
- confirm the UI shows the failed tool activity
- confirm assistant text continues
- confirm the session accepts another follow-up message

Document the observed scenario and outcome in the PR or commit notes.

- [ ] **Step 10: Commit the frontend and verification chunk**

```bash
git add web/src/api/modules/ai.ts web/src/components/AI/replyRuntime.ts web/src/components/AI/a2uiState.ts web/src/components/AI/replyRuntime.test.ts web/src/components/AI/__tests__/a2uiState.test.ts
git commit -m "feat(web): show degraded ai runs with tool failures"
```

## Final Verification Checklist

- [ ] Recoverable tool failures no longer terminate the current run
- [ ] Fatal runtime failures still terminate the current run
- [ ] Approval rejection behaves as a recoverable tool outcome
- [ ] Single-host command tools no longer depend on a readonly command whitelist
- [ ] Prompt text instructs the model to recover after tool errors
- [ ] Frontend distinguishes degraded completion from fatal run failure
- [ ] Session remains usable after both recoverable and fatal run outcomes

## Notes for the Implementer

- Keep the first pass narrow. Do not invent a giant universal error framework.
- Prefer helper functions over large inline condition chains in `logic.go`.
- Do not silently change unrelated tool semantics while touching host command behavior.
- If a change requires a DB migration for status vocabulary, stop and decide whether the existing schema already tolerates the new string values before adding migration churn.
- If you need to split large files while implementing, do it only when the extraction clarifies a boundary already described in the spec.
