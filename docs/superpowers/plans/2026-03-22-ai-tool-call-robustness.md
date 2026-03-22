# AI Tool Call Robustness Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a unified tool-argument normalization layer and a shared recoverable tool-error pipeline so tool failures do not terminate sessions and argument-shape drift is handled consistently across agents.

**Architecture:** Introduce a compose-level tool middleware that normalizes argument keys/types before every tool invocation, then refactor runtime error classification into a shared module consumed by both `Chat` and `ApprovalWorker`. Keep behavior deterministic: only case/style + type coercion fixes, no semantic remapping. Add per-turn retry guard and aligned tests to guarantee consistent `completed_with_tool_errors` convergence.

**Tech Stack:** Go, CloudWeGo Eino ADK/compose tool middleware, GORM DAOs, Go test

---

## File Structure

### New Files
- `internal/ai/tools/middleware/arg_normalizer.go`
  - Unified key canonicalization and type coercion engine for tool args.
- `internal/ai/tools/middleware/arg_normalizer_test.go`
  - Unit tests for field matching priority, enum case-insensitive matching, empty-string handling, and coercion failure metadata.
- `internal/service/ai/logic/tool_error_classifier.go`
  - Shared recoverable invocation-error classifier and synthetic tool-result builder.
- `internal/service/ai/logic/tool_error_classifier_test.go`
  - Unit tests for regex parsing boundaries and classifier behavior.

### Modified Files
- `internal/ai/tools/middleware/approval.go`
  - Add compose middleware adapter helper so normalizer and approval can be composed cleanly.
- `internal/ai/tools/tools.go`
  - Export helper to build default tool middlewares (normalizer + approval for change path).
- `internal/ai/agents/diagnosis/agent.go`
  - Register normalizer middleware for diagnosis executor tools.
- `internal/ai/agents/change/agent.go`
  - Register normalizer before approval middleware.
- `internal/ai/agents/inspection/agent.go`
  - Register normalizer middleware.
- `internal/ai/agents/qa/qa.go`
  - Register normalizer middleware for `history` tool.
- `internal/service/ai/logic/logic.go`
  - Replace duplicated recoverable-tool-error parsing helpers with shared classifier usage.
- `internal/service/ai/logic/approval_worker.go`
  - Reuse shared classifier on stream recv errors; align with Chat behavior.
- `internal/service/ai/logic/logic_test.go`
  - Update/extend integration tests for unified recoverable behavior.
- `internal/service/ai/logic/approval_worker_test.go`
  - Add resume-path recoverable error coverage.

### Optional Documentation Update (if needed after implementation)
- `docs/superpowers/specs/2026-03-22-ai-tool-call-robustness-design.md`
  - Only if implementation deviates in non-trivial ways.

## Chunk 1: Argument Normalization Middleware

### Task 1: Build normalization core (TDD)

**Files:**
- Create: `internal/ai/tools/middleware/arg_normalizer.go`
- Test: `internal/ai/tools/middleware/arg_normalizer_test.go`

- [ ] **Step 1: Write failing tests for key matching priority**

```go
func TestNormalizeArgs_FieldPriority(t *testing.T) {
	input := `{"UserName":"A","user_name":"B"}`
	// exact > case-insensitive > normalized form
	// expect deterministic winner and conflict metadata
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/ai/tools/middleware -run TestNormalizeArgs_FieldPriority -v`
Expected: FAIL with missing normalizer implementation.

- [ ] **Step 3: Implement minimal key canonicalization engine**

```go
type NormalizeResult struct {
	NormalizedJSON string
	NormalizedKeys []string
	Coercions []string
	CoercionFailures []CoercionFailure
}

func NormalizeToolArgs(raw string, schema any) (NormalizeResult, error) {
	// parse raw JSON map with json.Decoder.UseNumber() to avoid float64 precision loss
	// build struct field index by exact / case-insensitive / normalized key
	// resolve deterministic mapping and record conflicts
}
```

- [ ] **Step 4: Run test to verify pass**

Run: `go test ./internal/ai/tools/middleware -run TestNormalizeArgs_FieldPriority -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ai/tools/middleware/arg_normalizer.go internal/ai/tools/middleware/arg_normalizer_test.go
git commit -m "feat(ai): add deterministic tool args key normalization"
```

### Task 2: Add type coercion rules (TDD)

**Files:**
- Modify: `internal/ai/tools/middleware/arg_normalizer.go`
- Modify: `internal/ai/tools/middleware/arg_normalizer_test.go`

- [ ] **Step 1: Write failing tests for type coercion and enum matching**

```go
func TestNormalizeArgs_TypeCoercion(t *testing.T) {
	// "2" -> int(2), "true" -> bool(true), 2 -> "2"
}

func TestNormalizeArgs_EnumCaseInsensitive(t *testing.T) {
	// "OPEN" -> "open" when enum contains open/closed
}

func TestNormalizeArgs_EmptyStringOptionalNumber(t *testing.T) {
	// "" on optional numeric field should not hard-fail
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test -race ./internal/ai/tools/middleware -run 'TestNormalizeArgs_(TypeCoercion|EnumCaseInsensitive|EmptyStringOptionalNumber)' -v`
Expected: FAIL for unimplemented coercion paths.

- [ ] **Step 3: Implement minimal coercion logic**

```go
func coerceValue(dstType reflect.Type, raw any, enumValues []string) (any, bool, *CoercionFailure) {
	// deterministic conversion only
	// enum: case-insensitive unique match
	// empty string for optional scalar => treat as omitted/null equivalent
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/ai/tools/middleware -run TestNormalizeArgs -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ai/tools/middleware/arg_normalizer.go internal/ai/tools/middleware/arg_normalizer_test.go
git commit -m "feat(ai): add tool args type coercion and enum normalization"
```

### Task 3: Wire compose middleware into agents (shadow mode first)

**Files:**
- Modify: `internal/ai/tools/tools.go`
- Modify: `internal/ai/agents/diagnosis/agent.go`
- Modify: `internal/ai/agents/change/agent.go`
- Modify: `internal/ai/agents/inspection/agent.go`
- Modify: `internal/ai/agents/qa/qa.go`
- Test: `internal/ai/tools/middleware/approval_test.go` (extend if helper coverage is needed)

- [ ] **Step 1: Write failing tests for middleware ordering/composition helpers**

```go
func TestBuildToolMiddlewares_NormalizerBeforeApproval(t *testing.T) {
	// ensure normalized args are what approval sees
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/ai/tools/middleware -run TestBuildToolMiddlewares_NormalizerBeforeApproval -v`
Expected: FAIL because helper is not implemented.

- [ ] **Step 3: Implement middleware wiring**

```go
// tools.go
func DefaultReadonlyToolMiddlewares(normalizeCfg middleware.ArgNormalizeConfig) []compose.ToolMiddleware {
	return []compose.ToolMiddleware{middleware.NewArgNormalizationToolMiddleware(normalizeCfg)}
}

func DefaultChangeToolMiddlewares(normalizeCfg middleware.ArgNormalizeConfig, approval compose.ToolMiddleware) []compose.ToolMiddleware {
	return []compose.ToolMiddleware{
		middleware.NewArgNormalizationToolMiddleware(normalizeCfg),
		approval,
	}
}
```

`ArgNormalizeConfig` requires:
- `Enabled bool` (apply normalized args)
- `ShadowMode bool` (collect normalization/coercion signals but return original raw args)

Phase in this plan:
- First wire all agents with `ShadowMode=true, Enabled=false`
- After verification in Chunk 3, switch to `Enabled=true, ShadowMode=false`

- [ ] **Step 4: Update each agent ToolsConfig to use shared helpers**

Run edits in:
- `diagnosis/agent.go`
- `change/agent.go`
- `inspection/agent.go`
- `qa/qa.go`

Expected: all tool-capable agents register normalizer.

- [ ] **Step 5: Run targeted tests**

Run: `go test -race ./internal/ai/agents/... ./internal/ai/tools/... -run 'Test.*(Tools|Middleware|Agent)' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/ai/tools/tools.go internal/ai/agents/diagnosis/agent.go internal/ai/agents/change/agent.go internal/ai/agents/inspection/agent.go internal/ai/agents/qa/qa.go
 git commit -m "feat(ai): register tool args normalization middleware across agents"
```

## Chunk 2: Shared Tool Error Classification + Recovery Unification

### Task 4: Extract shared classifier (TDD)

**Files:**
- Create: `internal/service/ai/logic/tool_error_classifier.go`
- Create: `internal/service/ai/logic/tool_error_classifier_test.go`

- [ ] **Step 1: Write failing classifier tests**

```go
func TestClassifyRecoverableToolInvocationError_StreamPattern(t *testing.T) {}
func TestClassifyRecoverableToolInvocationError_InvokePattern(t *testing.T) {}
func TestClassifyRecoverableToolInvocationError_NoMatch(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test -race ./internal/service/ai/logic -run TestClassifyRecoverableToolInvocationError -v`
Expected: FAIL with missing classifier symbols.

- [ ] **Step 3: Implement classifier + synthetic event builder**

```go
type RecoverableToolInvocationError struct {
	CallID string
	ToolName string
	Message string
}

func ClassifyRecoverableToolInvocationError(err error) (*RecoverableToolInvocationError, bool) {
	// centralized regex parse
}

func BuildRecoverableToolErrorEvent(agentName string, classified *RecoverableToolInvocationError, sourceErr error) (*adk.AgentEvent, bool) {
	// build schema.ToolMessage payload with status=error,error_type=tool_invocation
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/service/ai/logic -run TestClassifyRecoverableToolInvocationError -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/logic/tool_error_classifier.go internal/service/ai/logic/tool_error_classifier_test.go
git commit -m "refactor(ai): extract shared recoverable tool error classifier"
```

### Task 5: Refactor `Chat` and `ApprovalWorker` to one recovery path

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/approval_worker.go`
- Modify: `internal/service/ai/logic/logic_test.go`
- Modify: `internal/service/ai/logic/approval_worker_test.go`

- [ ] **Step 1: Write failing tests for resume-path stream recv recoverable behavior**

```go
func TestApprovalWorker_StreamingToolInvocationRecvError_CompletedWithToolErrors(t *testing.T) {
	// same error text as chat path
	// expect no terminal runtime failure, expect completed_with_tool_errors
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test -race ./internal/service/ai/logic -run TestApprovalWorker_StreamingToolInvocationRecvError_CompletedWithToolErrors -v`
Expected: FAIL with current resume path treating recvErr as fatal.

- [ ] **Step 3: Replace duplicated helpers in `logic.go` with classifier usage**

```go
func recoverableToolErrorEvent(event *adk.AgentEvent) (*adk.AgentEvent, bool) {
	// delegate to shared classifier/builder
}

func recoverableToolErrorFromErr(err error, agentName string) (*adk.AgentEvent, bool) {
	// delegate to shared classifier/builder
}
```

- [ ] **Step 4: Update `approval_worker.go` stream recv branch to reuse same builder**

```go
if recvErr != nil {
	if toolErrEvent, ok := recoverableToolErrorFromErr(recvErr, event.AgentName); ok {
		hasToolErrors = true
		// consume projected events and continue
		break
	}
	// existing fatal path
}
```

- [ ] **Step 5: Run targeted tests**

Run: `go test -race ./internal/service/ai/logic -run 'TestChatKeepsRunAliveOn|TestApprovalWorker_.*Tool.*Error' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/approval_worker.go internal/service/ai/logic/logic_test.go internal/service/ai/logic/approval_worker_test.go
git commit -m "refactor(ai): unify recoverable tool error handling across chat and approval resume"
```

### Task 6: Add per-turn retry circuit breaker (TDD)

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/approval_worker.go`
- Modify: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write failing test for repeated same-call-shape failures**

```go
func TestChat_CircuitBreaksRepeatedSameToolFailure(t *testing.T) {
	// simulate repeated same tool_name+normalized_args failures in one run
	// expect bounded retries and graceful completion_with_tool_errors
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/service/ai/logic -run TestChat_CircuitBreaksRepeatedSameToolFailure -v`
Expected: FAIL due missing breaker state.

- [ ] **Step 3: Implement minimal breaker state**

```go
type toolFailureKey struct {
	ToolName string
	ArgsHash string
}

// in-turn map[toolFailureKey]int with threshold (e.g. 2)
// after threshold, synthesize final tool_result(error) hinting repeated-failure cutoff
```

State constraint:
- The breaker map MUST live in per-run scope in `logic.go` / `approval_worker.go` event loop context.
- Do NOT store breaker state inside tool middleware package globals or shared singletons.

- [ ] **Step 4: Run targeted tests**

Run: `go test -race ./internal/service/ai/logic -run 'TestChat_CircuitBreaksRepeatedSameToolFailure|TestChatKeepsRunAliveOn' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/approval_worker.go internal/service/ai/logic/logic_test.go
git commit -m "feat(ai): add per-turn circuit breaker for repeated tool invocation failures"
```

## Chunk 3: Observability, End-to-End Validation, and Cleanup

### Task 7: Add normalization/error observability and regression tests

**Files:**
- Modify: `internal/ai/tools/middleware/arg_normalizer.go`
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/approval_worker.go`
- Modify: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write failing tests for metadata logging payload shape**

```go
func TestNormalizeArgs_ReportsCoercionFailureDetails(t *testing.T) {
	// expect field/provided/expected captured
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test -race ./internal/ai/tools/middleware -run TestNormalizeArgs_ReportsCoercionFailureDetails -v`
Expected: FAIL if metadata shape missing details.

- [ ] **Step 3: Implement metadata emission hooks**

```go
// emit normalized_keys/coercions/coercion_failures with original provided value
// hook into existing logging mechanism used in ai logic path
```

- [ ] **Step 4: Run package tests**

Run: `go test -race ./internal/ai/tools/middleware ./internal/service/ai/logic -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ai/tools/middleware/arg_normalizer.go internal/service/ai/logic/logic.go internal/service/ai/logic/approval_worker.go internal/service/ai/logic/logic_test.go
git commit -m "chore(ai): add normalization and recoverable tool error observability"
```

### Task 8: Full verification pass and final cleanup

**Files:**
- Modify: `internal/service/ai/logic/logic.go` (remove dead helpers if still present)
- Modify: `internal/service/ai/logic/tool_error_classifier.go` (final naming cleanup)
- Test: `internal/service/ai/logic/logic_test.go`, `internal/service/ai/logic/approval_worker_test.go`, `internal/ai/tools/middleware/arg_normalizer_test.go`

- [ ] **Step 1: Remove dead/redundant code paths identified by refactor**

```go
// remove duplicate regex vars and parser helpers from logic.go
// keep only shared classifier module references
```

- [ ] **Step 2: Run focused regression suites**

Run:
- `go test -race ./internal/service/ai/logic -v`
- `go test -race ./internal/ai/tools/middleware -v`
- `go test -race ./internal/ai/agents/... -v`

Expected: all PASS.

- [ ] **Step 3: Run broader safety suite**

Run: `go test -race ./internal/ai/... ./internal/service/ai/... -v`
Expected: PASS without new flaky failures.

- [ ] **Step 4: Final commit**

```bash
git add internal/ai/tools/middleware internal/ai/agents internal/ai/tools/tools.go internal/service/ai/logic
git commit -m "feat(ai): harden tool invocation with arg normalization and unified error recovery"
```

- [ ] **Step 5: Capture implementation summary in PR/body notes**

Include:
- middleware insertion points
- removed redundant logic helpers
- new tests added
- known follow-ups (if any)

## Plan Review Loop Notes

- This plan is organized in 3 chunks for reviewer passes.
- After each chunk is implemented, run a plan/code review before moving to the next chunk.
- If review finds issues, fix within the same chunk and re-run tests before proceeding.
