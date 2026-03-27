# Host Exec Unified Approval Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unify host execution to a single `host_exec` tool and enforce one approval interrupt/resume model across all tools, with fail-closed risk decisions and safe migration.

**Architecture:** Keep approval logic centralized in `internal/ai/tools/middleware/approval.go` and remove approval-like behavior from tool outputs. Consolidate host execution entry points in `internal/ai/tools/host/tools.go` while preserving phased migration safety for in-flight runs. Align risk registry, agent toolsets, and runtime tests so all tool families follow identical middleware semantics.

**Tech Stack:** Go, Eino ADK/tool middleware, Gin service layer, GORM-backed policy data, Go test

---

## File Structure

- Modify: `internal/ai/tools/host/tools.go`
  - Consolidate host execution contract around `host_exec` (`host_id`, `command`, `script`) and remove direct legacy registrations from model-facing toolsets.
- Modify: `internal/ai/tools/host/tools_test.go`
  - Cover host_exec input contract and removal of legacy tool exposure.
- Modify: `internal/ai/tools/middleware/approval.go`
  - Enforce unified approval decision flow and remove legacy host-tool-name special casing.
- Modify: `internal/ai/tools/middleware/approval_test.go`
  - Verify unified middleware behavior for `host_exec` and non-host high-risk tools.
- Modify: `internal/ai/tools/common/risk_registry.go`
  - Keep only active tool names and enforce fail-closed fallback behavior.
- Modify: `internal/ai/tools/common/risk_registry_test.go`
  - Update registry coverage assertions to the post-consolidation tool surface.
- Modify: `internal/ai/tools/tools.go`
  - Ensure diagnosis/change/inspection/qa toolsets consume the new host execution surface.
- Modify: `internal/ai/tools/tools_test.go`
  - Replace `host_exec_change` expectations with `host_exec` expectations.
- Modify: `internal/ai/agents/prompt/prompt.go`
  - Update planner/executor guidance to use `host_exec` only.
- Modify: `internal/service/ai/logic/logic_test.go`
  - Keep replay compatibility for historical legacy events while validating new event payloads use `host_exec`.
- Modify: `docs/superpowers/specs/2026-03-23-host-exec-unified-approval-design.md`
  - Keep spec in sync if implementation detail deltas appear during coding.

## Chunk 1: Host Tool Surface Consolidation

### Task 1: Lock Host Exec Contract with Failing Tests

**Files:**
- Test: `internal/ai/tools/host/tools_test.go`

- [ ] **Step 1: Write failing tests for `host_exec` input contract**

```go
func TestHostExec_RejectsWhenCommandAndScriptBothProvided(t *testing.T) {}
func TestHostExec_RejectsWhenCommandAndScriptBothEmpty(t *testing.T) {}
func TestNewHostTools_DoesNotExposeLegacyExecToolNames(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ai/tools/host -run "TestHostExec_RejectsWhenCommandAndScriptBothProvided|TestHostExec_RejectsWhenCommandAndScriptBothEmpty|TestNewHostTools_DoesNotExposeLegacyExecToolNames" -count=1`
Expected: FAIL with missing behavior and/or unexpected legacy tool names.

- [ ] **Step 3: Commit failing tests**

```bash
git add internal/ai/tools/host/tools_test.go
git commit -m "test(host): add host_exec contract and legacy exposure tests"
```

### Task 2: Implement Single `host_exec` Entry and Remove Legacy Exposure

**Files:**
- Modify: `internal/ai/tools/host/tools.go`
- Test: `internal/ai/tools/host/tools_test.go`

- [ ] **Step 1: Implement `host_exec` schema and validation rules**

```go
type HostExecInput struct {
    HostID  int    `json:"host_id"`
    Command string `json:"command,omitempty"`
    Script  string `json:"script,omitempty"`
}

// Validation contract
// - host_id invalid => error
// - command && script both empty => error
// - command && script both non-empty => error
// NOTE: strings are value fields; omitted and explicit empty both decode to "".
// Validation MUST use value checks (input.Command == "" && input.Script == "").
```

- [ ] **Step 2: Keep execution path unified through policy-aware facade**

```go
// Route both command/script via unified policy-aware execution entry.
// Script path must still go through middleware risk decision before execute.
return runPolicyAwareExecByHostID(ctx, svcCtx, "host_exec", hostID, command, script)
```

- [ ] **Step 3: Remove legacy host exec tool registrations from model-facing sets**

```go
func NewHostTools(ctx context.Context) []tool.InvokableTool {
    // no legacy wrappers in returned list
}
```

- [ ] **Step 4: Run targeted host tool tests**

Run: `go test ./internal/ai/tools/host -count=1`
Expected: PASS.

- [ ] **Step 5: Commit implementation**

```bash
git add internal/ai/tools/host/tools.go internal/ai/tools/host/tools_test.go
git commit -m "feat(host-tools): unify to single host_exec entry"
```

## Chunk 2: Unified Approval Middleware and Risk Registry

### Task 3: Add Failing Middleware Tests for Unified `host_exec` Semantics

**Files:**
- Test: `internal/ai/tools/middleware/approval_test.go`

- [ ] **Step 1: Add failing tests for unified behavior**

```go
func TestCommandClassForTool_HostExec_CommandPath(t *testing.T) {}
func TestCommandClassForTool_HostExec_ScriptPath(t *testing.T) {}
func TestDefaultNeedsApproval_CoversHostExecOnly(t *testing.T) {}
```

- [ ] **Step 2: Verify failure before implementation**

Run: `go test ./internal/ai/tools/middleware -run "TestCommandClassForTool_HostExec_CommandPath|TestCommandClassForTool_HostExec_ScriptPath|TestDefaultNeedsApproval_CoversHostExecOnly" -count=1`
Expected: FAIL (old legacy-name behavior still present).

- [ ] **Step 3: Commit failing tests**

```bash
git add internal/ai/tools/middleware/approval_test.go
git commit -m "test(approval): add unified host_exec middleware behavior tests"
```

### Task 4: Refactor Approval Middleware to Single Host Exec Handling

**Files:**
- Modify: `internal/ai/tools/middleware/approval.go`
- Test: `internal/ai/tools/middleware/approval_test.go`

- [ ] **Step 1: Remove legacy host exec name branches from command-class and precheck logic**

```go
switch toolName {
case "host_exec":
    return hostExecCommandClass(args)
default:
    return defaultCommandClass(toolName)
}
```

- [ ] **Step 2: Ensure middleware stays single approval gateway**

```go
// No pseudo suspended result returned from tool layer.
// requires approval => always StatefulInterrupt.
```

- [ ] **Step 3: Enforce fail-closed on uncertain classification**

```go
if class == "" {
    class = "unknown"
}
```

- [ ] **Step 4: Run middleware test suite**

Run: `go test ./internal/ai/tools/middleware -count=1`
Expected: PASS.

- [ ] **Step 5: Commit middleware refactor**

```bash
git add internal/ai/tools/middleware/approval.go internal/ai/tools/middleware/approval_test.go
git commit -m "refactor(approval): unify host_exec gating and remove legacy branches"
```

### Task 5: Update Risk Registry to Post-Consolidation Tool Set

**Files:**
- Modify: `internal/ai/tools/common/risk_registry.go`
- Test: `internal/ai/tools/common/risk_registry_test.go`

- [ ] **Step 1: Write failing registry coverage assertions for consolidated host tools**

```go
// Expected host tools include host_exec, host_batch*, host_batch_status_update
// Expected host legacy names excluded.
```

- [ ] **Step 2: Run registry tests and confirm failure**

Run: `go test ./internal/ai/tools/common -run "TestRiskRegistry_ConditionalReadonlyHostExecWrappers|TestRiskRegistry_CoversKnownHighRiskTools" -count=1`
Expected: FAIL with old expected names.

- [ ] **Step 3: Update registry defaults and tests**

```go
"host_exec": {NeedsApproval: true, ConditionalByCommandClass: true},
// remove host_exec_readonly, host_exec_change, host_exec_by_target, host_ssh_exec_readonly
```

- [ ] **Step 4: Re-run registry tests**

Run: `go test ./internal/ai/tools/common -count=1`
Expected: PASS.

- [ ] **Step 5: Commit registry changes**

```bash
git add internal/ai/tools/common/risk_registry.go internal/ai/tools/common/risk_registry_test.go
git commit -m "refactor(risk): align host risk registry with single host_exec surface"
```

### Task 5.5: Migrate DB Tool Policy Records to `host_exec`

**Files:**
- Create: `storage/migration/20260323_host_exec_policy_unification.sql` (or project-standard migration location)
- Modify: `storage/migration/dev_auto.go` (if required by migration registry)
- Test: `storage/migration/ai_approval_migration_test.go` (or nearest migration test file)

- [ ] **Step 1: Add failing migration test for legacy host policy names**

```go
func TestMigrateLegacyHostPolicyToolNamesToHostExec(t *testing.T) {}
```

- [ ] **Step 2: Run migration test to verify failure**

Run: `go test ./storage/migration -run TestMigrateLegacyHostPolicyToolNamesToHostExec -count=1`
Expected: FAIL before SQL migration is applied.

- [ ] **Step 3: Implement SQL migration**

```sql
-- Normalize legacy policy records
UPDATE ai_tool_risk_policies
SET tool_name = 'host_exec'
WHERE tool_name IN ('host_exec_readonly','host_exec_change','host_exec_by_target','host_ssh_exec_readonly');
```

- [ ] **Step 4: Re-run migration tests**

Run: `go test ./storage/migration -count=1`
Expected: PASS with normalized tool names.

- [ ] **Step 5: Commit DB migration**

```bash
git add storage/migration/20260323_host_exec_policy_unification.sql storage/migration/dev_auto.go storage/migration/ai_approval_migration_test.go
git commit -m "feat(migration): unify legacy host policy tool names to host_exec"
```

## Chunk 3: Toolset, Prompt, and Runtime Compatibility

### Task 6: Update Toolset Composition and Agent Expectations

**Files:**
- Modify: `internal/ai/tools/tools.go`
- Modify: `internal/ai/tools/tools_test.go`

- [ ] **Step 1: Add/adjust failing tests for diagnosis/change host tool expectations**

```go
func TestNewDiagnosisTools_IncludesHostExec(t *testing.T) {}
func TestNewChangeTools_IncludesHostExec(t *testing.T) {}
func TestNewDiagnosisTools_ExcludesLegacyHostExecNames(t *testing.T) {}

// Use set/map-based assertions on tool names; do not assert by index order.
// Example:
// names := collectToolNames(...)
// require.Contains(t, names, "host_exec")
// require.NotContains(t, names, "host_exec_readonly")
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/ai/tools -run "TestNewDiagnosisTools_IncludesHostExec|TestNewChangeTools_IncludesHostExec|TestNewDiagnosisTools_ExcludesLegacyHostExecNames" -count=1`
Expected: FAIL until toolset composition is updated.

- [ ] **Step 3: Update toolset builders to depend on consolidated host tool names only**

```go
// Diagnosis/Change paths should reference host_exec and exclude removed legacy names.
```

- [ ] **Step 4: Run toolset tests**

Run: `go test ./internal/ai/tools -count=1`
Expected: PASS.

- [ ] **Step 5: Commit toolset updates**

```bash
git add internal/ai/tools/tools.go internal/ai/tools/tools_test.go
git commit -m "refactor(tools): align diagnosis/change toolsets with host_exec unification"
```

### Task 7: Update Prompt Contracts to Prefer `host_exec`

**Files:**
- Modify: `internal/ai/agents/prompt/prompt.go`
- Test: `internal/ai/agents/prompt/prompt_test.go`

- [ ] **Step 1: Add failing prompt assertions for single host exec naming**

```go
func TestChangeExecutorPrompt_UsesHostExecOnly(t *testing.T) {}
```

- [ ] **Step 2: Run prompt tests to verify failure**

Run: `go test ./internal/ai/agents/prompt -run TestChangeExecutorPrompt_UsesHostExecOnly -count=1`
Expected: FAIL if legacy names still appear.

- [ ] **Step 3: Update prompt text and test fixtures**

```go
// Replace legacy host tool references with host_exec.
```

- [ ] **Step 4: Re-run prompt tests**

Run: `go test ./internal/ai/agents/prompt -count=1`
Expected: PASS.

- [ ] **Step 5: Commit prompt changes**

```bash
git add internal/ai/agents/prompt/prompt.go internal/ai/agents/prompt/prompt_test.go
git commit -m "chore(prompt): standardize host operation references to host_exec"
```

### Task 8: Keep Runtime Replay Compatible During Phased Migration

**Files:**
- Modify: `internal/service/ai/logic/logic_test.go`
- Modify: `internal/service/ai/logic/logic.go` (only if behavior adjustment needed)

- [ ] **Step 1: Add failing replay test covering legacy in-flight event compatibility**

```go
func TestReplayWaitingApproval_LegacyHostExecNamesRemainReadable(t *testing.T) {}
```

- [ ] **Step 2: Run compatibility test and confirm failure if needed**

Run: `go test ./internal/service/ai/logic -run TestReplayWaitingApproval_LegacyHostExecNamesRemainReadable -count=1`
Expected: FAIL if replay/mapping is broken.

- [ ] **Step 3: Implement minimal compatibility mapping for phased rollout**

```go
// Keep old event parsing readable for replay while new emissions use host_exec.
```

- [ ] **Step 4: Run AI logic tests**

Run: `go test ./internal/service/ai/logic -count=1`
Expected: PASS.

- [ ] **Step 5: Commit compatibility fix**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "fix(ai-logic): preserve legacy approval replay compatibility during host_exec migration"
```

## Chunk 4: End-to-End Verification and Documentation Sync

### Task 9: Full Verification Pass

**Files:**
- Test: existing suites only

- [ ] **Step 1: Run focused suites in order**

Run: `go test ./internal/ai/tools/host ./internal/ai/tools/middleware ./internal/ai/tools/common ./internal/ai/tools ./internal/ai/agents/prompt ./internal/service/ai/logic -count=1`
Expected: PASS.

- [ ] **Step 2: Run broader AI domain regression**

Run: `go test ./internal/ai/... ./internal/service/ai/... -count=1`
Expected: PASS.

- [ ] **Step 3: Commit verification notes if any test baseline updates were required**

```bash
git add internal/ai/tools/host/tools_test.go internal/ai/tools/middleware/approval_test.go internal/ai/tools/common/risk_registry_test.go internal/ai/tools/tools_test.go internal/ai/agents/prompt/prompt_test.go internal/service/ai/logic/logic_test.go
git commit -m "test(ai): refresh baselines for host_exec unified approval"
```

### Task 10: Documentation and Rollout Notes

**Files:**
- Modify: `docs/superpowers/specs/2026-03-23-host-exec-unified-approval-design.md` (if implementation deltas discovered)
- Create: `docs/superpowers/plans/2026-03-23-host-exec-unified-approval-rollout.md` (optional operational checklist)

- [ ] **Step 1: Diff implementation vs spec and capture any intentional deviations**

```markdown
- Decision:
- Reason:
- Impact:
```

- [ ] **Step 2: Add phased rollout checklist (Phase 1/2/3 gates) for operators**

```markdown
- Legacy invocation metric < threshold for 7d
- DB policy migration complete
- Removal gate approved
```

- [ ] **Step 3: Commit doc sync**

```bash
git add docs/superpowers/specs/2026-03-23-host-exec-unified-approval-design.md docs/superpowers/plans/2026-03-23-host-exec-unified-approval-rollout.md
git commit -m "docs(ai): add rollout gates for host_exec unified approval migration"
```
