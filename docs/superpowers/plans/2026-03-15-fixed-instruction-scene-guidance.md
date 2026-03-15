# Fixed Instruction And Scene Guidance Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace per-request dynamic instruction rendering with a fixed agent instruction and move scene-aware runtime guidance into a normalized user-input envelope, while explicitly removing obsolete dynamic-prompt code.

**Architecture:** The agent keeps one fixed system instruction that explains tool domains and scene-biased tool selection rules. The orchestrator becomes responsible for normalizing `RuntimeContext` into a stable text envelope prepended to the raw user request. Cleanup is explicit: any code path or helper that still assembles prompt text from `RuntimeContext` must be removed or reduced to a fixed constant accessor only.

**Tech Stack:** Go 1.26, CloudWeGo Eino ADK, Gin, existing AI runtime/orchestrator stack

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/ai/runtime/instruction.go` | Rewrite | Hold a fixed instruction constant and any tiny fixed accessor only |
| `internal/ai/runtime/instruction_test.go` | Rewrite | Verify fixed instruction behavior only; remove dynamic-context expectations |
| `internal/ai/orchestrator.go` | Modify | Normalize `RuntimeContext`, build runtime-context envelope, prepend it to the raw user request |
| `internal/ai/orchestrator_test.go` | Modify | Verify envelope shaping and runner input behavior; keep approval-flow coverage |
| `internal/ai/agents/agent.go` | Modify | Use the fixed instruction directly and remove per-request instruction session coupling |
| `internal/ai/runtime/context_processor.go` | Clean up or delete | Remove obsolete planexecute-oriented prompt-building and unused session keys if no longer referenced |
| `docs/superpowers/plans/2026-03-15-chat-model-agent-refactor.md` | Modify | Mark the old dynamic-instruction plan obsolete or superseded |
| `docs/superpowers/specs/2026-03-15-chat-model-agent-refactor-design.md` | Modify | Mark the old dynamic-instruction design obsolete or superseded |

---

## Chunk 1: Fixed Instruction And Runtime-Context Envelope

### Task 1: Replace dynamic instruction expectations with fixed instruction behavior

**Files:**
- Modify: `internal/ai/runtime/instruction.go`
- Modify: `internal/ai/runtime/instruction_test.go`
- Modify: `internal/ai/agents/agent.go`

- [ ] **Step 1: Write the failing tests for fixed instruction behavior**

Add or rewrite tests in `internal/ai/runtime/instruction_test.go` to assert:

```go
func TestInstruction_IsStableAcrossRuntimeContexts(t *testing.T) {
	first := BuildInstruction(RuntimeContext{Scene: "deployment:hosts", ProjectID: "1"})
	second := BuildInstruction(RuntimeContext{Scene: "service:list", ProjectID: "9"})
	assert.Equal(t, first, second)
}

func TestInstruction_DescribesSceneBiasWithoutHardRestriction(t *testing.T) {
	result := BuildInstruction(RuntimeContext{})
	assert.Contains(t, result, "scene")
	assert.Contains(t, result, "优先")
	assert.Contains(t, result, "不是硬性限制")
}

func TestInstruction_CoversToolDomainsAndCanonicalSceneRules(t *testing.T) {
	result := BuildInstruction(RuntimeContext{})
	assert.Contains(t, result, "host")
	assert.Contains(t, result, "deployment")
	assert.Contains(t, result, "service")
	assert.Contains(t, result, "kubernetes")
	assert.Contains(t, result, "monitor")
	assert.Contains(t, result, "governance")
	assert.Contains(t, result, "deployment:*")
	assert.Contains(t, result, "service:*")
	assert.Contains(t, result, "host:*")
	assert.Contains(t, result, "k8s:*")
}

func TestInstruction_RequiresReadonlyFirstAndApprovalForMutations(t *testing.T) {
	result := BuildInstruction(RuntimeContext{})
	assert.Contains(t, result, "只读")
	assert.Contains(t, result, "审批")
}
```

- [ ] **Step 2: Run the runtime instruction tests and verify they fail for the right reason**

Run: `go test ./internal/ai/runtime/... -run 'TestInstruction_'`

Expected: FAIL because `BuildInstruction` still renders runtime-specific values or the fixed-prompt assertions are not yet true.

- [ ] **Step 3: Rewrite `instruction.go` to a fixed instruction implementation**

Replace runtime-personalized prompt assembly with:

- one fixed instruction constant
- optional helper `BuildInstruction(RuntimeContext) string` that ignores the input and returns the same constant for every request
- tool-domain guidance text that states:
  - available domains: `host`, `deployment`, `service`, `kubernetes`, `monitor`, `governance`
  - scene biases initial tool selection
  - scene is not a hard boundary
  - readonly-first for investigation
  - mutating tools still require approval
- canonical scene-to-domain preference rules from the spec:
  - `deployment:*` -> `deployment`, `host`, `service`, `kubernetes`
  - `service:*` -> `service`, `deployment`, `kubernetes`
  - `host:*` -> `host`, `deployment`, `monitor`
  - `k8s:*` -> `kubernetes`, `service`, `deployment`

- [ ] **Step 4: Remove per-request instruction session coupling from `agent.go`**

Update `internal/ai/agents/agent.go` so:

- `ChatModelAgentConfig.Instruction` uses the fixed instruction constant directly
- `SessionKeyInstruction` is not required by the normal run path anymore

- [ ] **Step 5: Update tests to verify the fixed behavior passes**

Run: `go test ./internal/ai/runtime/... -run 'TestInstruction_'`

Expected: PASS

- [ ] **Step 6: Commit the fixed-instruction change**

```bash
git add internal/ai/runtime/instruction.go internal/ai/runtime/instruction_test.go internal/ai/agents/agent.go
git commit -m "refactor(ai): switch to fixed agent instruction"
```

### Task 2: Build and use the normalized runtime-context envelope

**Files:**
- Modify: `internal/ai/orchestrator.go`
- Modify: `internal/ai/orchestrator_test.go`

- [ ] **Step 1: Write the failing orchestrator tests for envelope shaping**

Add tests that assert:

```go
func TestBuildRuntimeContextEnvelope_OmitsEmptyBlock(t *testing.T) {
	got := buildRuntimeContextEnvelope(runtime.RuntimeContext{})
	assert.Equal(t, "", got)
}

func TestBuildRuntimeContextEnvelope_UsesCanonicalFieldOrder(t *testing.T) {
	got := buildRuntimeContextEnvelope(runtime.RuntimeContext{
		Scene:       "deployment:hosts",
		ProjectID:   "1",
		CurrentPage: "/deployment/infrastructure/hosts",
		SelectedResources: []runtime.SelectedResource{
			{Name: "node-a", Type: "host"},
		},
	})
	assert.Contains(t, got, "[Runtime Context]\nscene: deployment:hosts\nproject: 1\npage: /deployment/infrastructure/hosts\nselected_resources: node-a(host)")
}

func TestBuildRuntimeContextEnvelope_SummarizesSelectedResourcesAndNormalizesWhitespace(t *testing.T) {
	got := buildRuntimeContextEnvelope(runtime.RuntimeContext{
		Scene: "deployment:hosts",
		SelectedResources: []runtime.SelectedResource{
			{Name: "node-a\nprod", Type: "host"},
		},
	})
	assert.Contains(t, got, "selected_resources: node-a prod(host)")
}

func TestBuildRuntimeContextEnvelope_DoesNotDumpMetadataOrUserContext(t *testing.T) {
	got := buildRuntimeContextEnvelope(runtime.RuntimeContext{
		Scene:       "deployment:hosts",
		UserContext: map[string]any{"uid": 1},
		Metadata:    map[string]any{"scene": "deployment:hosts", "noise": "x"},
	})
	assert.NotContains(t, got, "uid")
	assert.NotContains(t, got, "noise")
}

func TestComposeUserInput_PreservesRawUserRequest(t *testing.T) {
	got := composeUserInput("scene: deployment:hosts", "帮我检查主机状态")
	assert.Contains(t, got, "[User Request]\n帮我检查主机状态")
}

func TestRun_UsesEnvelopeAndPreservesRawUserRequest(t *testing.T) {
	// Use a fake runner seam or equivalent helper to assert the exact input
	// passed into Query(...) contains the runtime block and unchanged raw request.
}
```

- [ ] **Step 2: Run the orchestrator tests and verify they fail**

Run: `go test ./internal/ai -run 'TestBuildRuntimeContextEnvelope|TestComposeUserInput'`

Expected: FAIL because envelope helpers do not exist yet and orchestrator still relies on instruction session values.

- [ ] **Step 3: Implement the minimal envelope helpers in `orchestrator.go`**

Add focused helpers near the bottom of `orchestrator.go`:

- `buildRuntimeContextEnvelope(runtime.RuntimeContext) string`
- `composeUserInput(envelope, raw string) string`
- `normalizeRuntimeLine(string) string`
- `summarizeSelectedResources([]runtime.SelectedResource) string`

Required behavior:

- fixed headers `[Runtime Context]` and `[User Request]`
- field order `scene`, `project`, `page`, `selected_resources`
- omit empty optional fields entirely
- omit the whole runtime block if all fields are empty
- collapse embedded newlines to spaces
- never dump raw `Metadata` or `UserContext`
- summarize `selected_resources` as comma-separated `name(type)` items

- [ ] **Step 4: Change `Run()` to prepend the envelope to the raw user request**

Update `Run()` so it:

- stops computing a runtime-specific instruction for the agent
- builds the effective user input from normalized envelope + raw user request
- passes the combined text to `o.runner.Query(...)`
- adds or reuses a narrow test seam so the exact composed Query input can be asserted in tests

- [ ] **Step 5: Re-run the focused orchestrator tests**

Run: `go test ./internal/ai -run 'TestBuildRuntimeContextEnvelope|TestComposeUserInput|TestRun_UsesEnvelopeAndPreservesRawUserRequest|TestOrchestrator'`

Expected: PASS

- [ ] **Step 6: Commit the envelope implementation**

```bash
git add internal/ai/orchestrator.go internal/ai/orchestrator_test.go
git commit -m "refactor(ai): inject runtime scene guidance into user input"
```

---

## Chunk 2: Remove Obsolete Dynamic-Prompt Code

### Task 3: Remove session-value plumbing that only existed for dynamic instruction rendering

**Files:**
- Modify: `internal/ai/runtime/context_processor.go`

- [ ] **Step 1: Write the failing test or compile target for removing instruction session coupling**

Use a compile-focused verification target for this cleanup:

Run: `go test ./internal/ai/...`

Expected before cleanup: either PASS with obsolete code still referenced, or FAIL once you delete one obsolete reference. The goal of this step is to establish the package set that guards the cleanup.

- [ ] **Step 2: Remove obsolete instruction session key and dead prompt-building helpers**

In `internal/ai/runtime/context_processor.go`:

- delete `SessionKeyInstruction` if it is no longer referenced
- remove `planexecute`-oriented prompt builders if they are unused in the ChatModelAgent path:
  - `BuildPlannerInput`
  - `BuildExecutorInput`
  - `BuildReplannerInput`
  - helper prompt templates tied only to those functions
- if the remaining `ContextProcessor` is not used anywhere after cleanup, delete the file and update references accordingly

- [ ] **Step 3: Re-run AI package tests to verify the cleanup**

Run: `go test ./internal/ai/...`

Expected: PASS

- [ ] **Step 4: Commit the old dynamic-prompt code cleanup**

```bash
git add internal/ai/runtime/context_processor.go internal/ai/runtime/*.go
git commit -m "refactor(ai): remove obsolete dynamic prompt plumbing"
```

### Task 4: Remove or rewrite the old dynamic-instruction tests and assertions

**Files:**
- Modify: `internal/ai/runtime/instruction_test.go`
- Modify: `internal/ai/orchestrator_test.go`

- [ ] **Step 1: Identify tests that still encode the old model**

Delete or rewrite assertions that expect:

- runtime-dependent `BuildInstruction` output
- page/project fallback logic inside the system prompt
- session-driven instruction mutation

- [ ] **Step 2: Add focused regression coverage for the new model**

Keep only tests that assert:

- fixed instruction text is stable
- envelope shape is canonical
- scene bias does not break approval flow or general execution flow

- [ ] **Step 3: Re-run only the touched tests**

Run: `go test ./internal/ai/... -run 'TestInstruction_|TestBuildRuntimeContextEnvelope|TestComposeUserInput|TestOrchestrator'`

Expected: PASS

- [ ] **Step 4: Commit test cleanup**

```bash
git add internal/ai/runtime/instruction_test.go internal/ai/orchestrator_test.go
git commit -m "test(ai): align tests with fixed instruction model"
```

---

## Chunk 3: Supersession Markers And Final Verification

### Task 5: Mark the old dynamic-instruction artifacts obsolete

**Files:**
- Modify: `docs/superpowers/plans/2026-03-15-chat-model-agent-refactor.md`
- Modify: `docs/superpowers/specs/2026-03-15-chat-model-agent-refactor-design.md`

- [ ] **Step 1: Add explicit supersession notes to the old design and plan**

At the top of both files, add a short note that:

- the dynamic `BuildInstruction(ctx)` direction is obsolete
- the new source of truth is `docs/superpowers/specs/2026-03-15-fixed-instruction-scene-guidance-design.md`
- future implementation should follow the fixed-instruction + runtime-context-envelope model

- [ ] **Step 2: Verify the supersession references are discoverable**

Run: `rg -n "superseded|obsolete|fixed-instruction-scene-guidance" docs/superpowers/specs docs/superpowers/plans`

Expected: matches in both the old dynamic-prompt artifacts and the new design doc.

- [ ] **Step 3: Commit the supersession notes**

```bash
git add docs/superpowers/plans/2026-03-15-chat-model-agent-refactor.md docs/superpowers/specs/2026-03-15-chat-model-agent-refactor-design.md
git commit -m "docs(ai): mark dynamic instruction design obsolete"
```

### Task 6: Run final verification for the full refactor

**Files:**
- Run only

- [ ] **Step 1: Run the AI module test suite**

Run: `go test ./internal/ai/...`

Expected: PASS

- [ ] **Step 2: Confirm AI coverage includes scene-biased but cross-domain-capable behavior**

Run: `rg -n "cross-domain|scene bias|scene-biased" internal/ai/*_test.go internal/ai/**/*_test.go`

Expected: at least one targeted test still verifies that scene guidance does not hard-limit cross-domain tool usage.

- [ ] **Step 3: Run the full repository test suite**

Run: `GOCACHE=/tmp/go-build make test`

Expected: PASS

- [ ] **Step 4: Run the frontend production build if `web/dist` is needed for repo-wide tests**

Run: `cd web && npm run build`

Expected: Vite build succeeds and emits `web/dist`

- [ ] **Step 5: Re-run full repository tests if Step 3 depended on build artifacts**

Run: `GOCACHE=/tmp/go-build make test`

Expected: PASS

- [ ] **Step 6: Final commit for verification fallout, if needed**

```bash
git add -A
git commit -m "chore(ai): finalize fixed instruction scene guidance refactor"
```
