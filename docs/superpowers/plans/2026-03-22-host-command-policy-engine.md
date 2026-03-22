# Host Command Policy Engine Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor host command execution into a unified policy core and converge to two entry points (`host_exec_readonly` / `host_exec_change`) with AST + allowlist + approval boundaries that are auditable, replay-safe, and fail-closed by default.

**Architecture:** Add a policy engine (parse, validate, decide) under `internal/ai/tools/host` and route new tools through it. `readonly` executes only when AST + allowlist checks fully pass; otherwise it triggers approval interruption immediately. `change` defaults to approval. Resume execution uses frozen parameters plus session-role binding checks to prevent replay. Legacy tools remain as compatibility wrappers and are gradually retired.

**Tech Stack:** Go, `mvdan.cc/sh/v3/syntax`, Eino tool middleware/interrupt, existing approval orchestrator, Go test.

---

## Chunk 1: Policy Engine and AST Validation Core

### Task 1: Add Host command policy core types (TDD)

**Files:**
- Create: `internal/ai/tools/host/policy_engine.go`
- Create: `internal/ai/tools/host/policy_types.go`
- Test: `internal/ai/tools/host/policy_engine_test.go`

- [ ] **Step 1: Write failing tests for decision enums and fail-closed baseline**

```go
func TestPolicyEngine_FailClosedOnParserError(t *testing.T) {
    engine := NewHostCommandPolicyEngine(DefaultReadonlyAllowlist())
    got := engine.Evaluate(PolicyInput{ToolName: "host_exec_readonly", CommandRaw: "echo $("})
    require.Equal(t, DecisionRequireApprovalInterrupt, got.DecisionType)
    require.Contains(t, got.ReasonCodes, "parse_error")
}

func TestPolicyEngine_FailClosedOnCommandTooLong(t *testing.T) {
    engine := NewHostCommandPolicyEngine(DefaultReadonlyAllowlist())
    got := engine.Evaluate(PolicyInput{
        ToolName:   "host_exec_readonly",
        CommandRaw: strings.Repeat("a", 4097),
    })
    require.Equal(t, DecisionRequireApprovalInterrupt, got.DecisionType)
    require.Contains(t, got.ReasonCodes, "command_too_long")
}
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/ai/tools/host -run TestPolicyEngine_FailClosedOnParserError -v`
Expected: FAIL (types/engine not implemented yet)

- [ ] **Step 3: Implement minimal decision types and Evaluate skeleton**

```go
type DecisionType string
const (
    DecisionAllowReadonlyExecute DecisionType = "allow_readonly_execute"
    DecisionRequireApprovalInterrupt DecisionType = "require_approval_interrupt"
)
```

Implementation add-on:
- Add a hard command length limit at `Evaluate` entry (`max_length = 4096` bytes).
- If length exceeds limit, do not parse AST; return `require_approval_interrupt` directly.

- [ ] **Step 4: Run tests and confirm pass**

Run: `go test ./internal/ai/tools/host -run TestPolicyEngine_FailClosedOnParserError -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/tools/host/policy_engine.go internal/ai/tools/host/policy_types.go internal/ai/tools/host/policy_engine_test.go
git commit -m "feat(ai/host): add policy engine skeleton with fail-closed decision model"
```

### Task 2: Integrate AST parsing and node traversal (TDD)

**Files:**
- Create: `internal/ai/tools/host/ast_parser.go`
- Modify: `internal/ai/tools/host/policy_engine.go`
- Test: `internal/ai/tools/host/ast_parser_test.go`
- Test: `internal/ai/tools/host/policy_engine_test.go`

- [ ] **Step 1: Write failing tests for parse behavior and command extraction**

```go
func TestParseCommand_CollectsPipelineCommands(t *testing.T) {
    parsed, err := ParseCommand("cat /var/log/syslog | grep error")
    require.NoError(t, err)
    require.ElementsMatch(t, []string{"cat", "grep"}, parsed.BaseCommands)
}
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/ai/tools/host -run="TestParseCommand|TestPolicyEngine" -v`
Expected: FAIL

- [ ] **Step 3: Implement minimal parser (mvdan) + AST summary output**

- [ ] **Step 4: Run tests and confirm pass**

Run: `go test ./internal/ai/tools/host -run="TestParseCommand|TestPolicyEngine" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/tools/host/ast_parser.go internal/ai/tools/host/ast_parser_test.go internal/ai/tools/host/policy_engine.go internal/ai/tools/host/policy_engine_test.go
git commit -m "feat(ai/host): parse shell commands via mvdan AST for policy evaluation"
```

### Task 3: Allowlist + operator rules (awk removed initially, pipeline per-segment validation, chain per-segment validation)

**Files:**
- Create: `internal/ai/tools/host/validator.go`
- Modify: `internal/ai/tools/host/policy_engine.go`
- Modify: `internal/ai/tools/host/policy_types.go`
- Test: `internal/ai/tools/host/validator_test.go`
- Test: `internal/ai/tools/host/policy_engine_test.go`

- [ ] **Step 1: Write failing tests for critical rules**

```go
func TestValidator_RejectsNonAllowlistedCommand(t *testing.T) {}
func TestValidator_RejectsAwkInInitialAllowlist(t *testing.T) {}
func TestValidator_AllowsPipelineWhenEachCommandAllowlisted(t *testing.T) {}
func TestValidator_RejectsRedirectionAndBackground(t *testing.T) {}
func TestValidator_CommandChainRequiresEachSegmentAllowlisted(t *testing.T) {}
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/ai/tools/host -run="TestValidator|TestPolicyEngine" -v`
Expected: FAIL

- [ ] **Step 3: Implement minimal validator**

Implementation points:
- Initial allowlist: `cat, ls, grep, top, free, df, tail` (exclude `awk`)
- Block: redirection, background, command substitution
- Allow: `|`, but each piped sub-command must pass allowlist independently
- `; && ||`: validate each segment; any failing segment => approval

- [ ] **Step 4: Run tests and confirm pass**

Run: `go test ./internal/ai/tools/host -run="TestValidator|TestPolicyEngine" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/tools/host/validator.go internal/ai/tools/host/validator_test.go internal/ai/tools/host/policy_engine.go internal/ai/tools/host/policy_types.go internal/ai/tools/host/policy_engine_test.go
git commit -m "feat(ai/host): enforce allowlist and operator validation on command AST"
```

## Chunk 2: Tool Convergence and Compatibility Migration

### Task 4: Add `host_exec_readonly` / `host_exec_change` and wire policy engine

**Files:**
- Modify: `internal/ai/tools/host/tools.go`
- Modify: `internal/ai/tools/host/policy_engine.go` (interface or constructor injection)
- Test: `internal/ai/tools/host/tools_test.go`

- [ ] **Step 1: Write failing tests for new tool registration and behavior**

```go
func TestNewHostReadonlyTools_ContainsHostExecReadonlyOnly(t *testing.T) {}
func TestHostExecReadonly_InterruptsWhenValidationFails(t *testing.T) {}
func TestHostExecChange_AlwaysRequestsApprovalBeforeExecution(t *testing.T) {}
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/ai/tools/host -run="TestNewHostReadonlyTools|TestHostExecReadonly|TestHostExecChange" -v`
Expected: FAIL

- [ ] **Step 3: Implement new tool entry points + unified facade**

Implementation points:
- `host_exec_readonly`: execute only if policy allows
- `host_exec_change`: default to approval
- Include policy decision summary in output (audit/debug)
- Use dependency injection (DI) for `HostCommandPolicyEngine` in tool construction to avoid hardcoded engine creation and enable mock-based unit tests

- [ ] **Step 4: Run tests and confirm pass**

Run: `go test ./internal/ai/tools/host -run="TestNewHostReadonlyTools|TestHostExecReadonly|TestHostExecChange" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/tools/host/tools.go internal/ai/tools/host/tools_test.go
git commit -m "feat(ai/host): add readonly/change execution tools backed by policy engine"
```

### Task 5: Legacy tool compatibility forwarding and localhost bypass removal

**Files:**
- Modify: `internal/ai/tools/host/tools.go`
- Test: `internal/ai/tools/host/tools_test.go`

- [ ] **Step 1: Write failing tests to ensure legacy paths cannot bypass policy**

```go
func TestLegacyHostExec_UsesPolicyEngine(t *testing.T) {}
func TestLegacyHostExecByTarget_LocalhostCannotBypassPolicy(t *testing.T) {}
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/ai/tools/host -run="TestLegacyHostExec|TestLegacyHostExecByTarget" -v`
Expected: FAIL

- [ ] **Step 3: Implement legacy-to-new facade forwarding**

Implementation points:
- `host_exec` -> forward through readonly/change policy path
- `host_exec_by_target` -> resolve target then still route through policy, no localhost bypass
- `host_ssh_exec_readonly` -> internally reuse new readonly implementation

- [ ] **Step 4: Run tests and confirm pass**

Run: `go test ./internal/ai/tools/host -run="TestLegacyHostExec|TestLegacyHostExecByTarget" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/tools/host/tools.go internal/ai/tools/host/tools_test.go
git commit -m "refactor(ai/host): route legacy host exec tools through unified policy path"
```

## Chunk 3: Approval Bridge, Resume Binding, Replay Protection

### Task 6: Return suspended status on approval interruption + strengthen context binding

**Files:**
- Modify: `internal/ai/tools/middleware/approval.go`
- Modify: `internal/ai/tools/common/approval_orchestrator.go`
- Test: `internal/ai/tools/middleware/approval_test.go`

- [ ] **Step 1: Write failing tests for suspended semantics and binding checks**

```go
func TestApprovalBridge_ReturnsSuspendedPayload(t *testing.T) {}
func TestApprovalResume_RejectsMismatchedSessionOrRole(t *testing.T) {}
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/ai/tools/middleware -run="TestApprovalBridge|TestApprovalResume" -v`
Expected: FAIL

- [ ] **Step 3: Implement approval bridge enhancements**

Implementation points:
- Interruption payload includes `status=suspended` and `approval_id`
- Bind and validate `approval_id + session_id + agent_role`
- On binding mismatch, re-interrupt

- [ ] **Step 4: Run tests and confirm pass**

Run: `go test ./internal/ai/tools/middleware -run="TestApprovalBridge|TestApprovalResume" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/tools/middleware/approval.go internal/ai/tools/common/approval_orchestrator.go internal/ai/tools/middleware/approval_test.go
git commit -m "feat(ai/approval): return suspended interrupts and enforce resume session-role binding"
```

### Task 7: Include `host_exec_change` in default approval coverage and command-class mapping

**Files:**
- Modify: `internal/ai/tools/middleware/approval.go`
- Modify: `internal/ai/tools/common/approval_orchestrator.go`
- Test: `internal/ai/tools/middleware/approval_test.go`

- [ ] **Step 1: Write failing tests for policy coverage**

```go
func TestDefaultNeedsApproval_CoversHostExecChange(t *testing.T) {}
func TestFallbackRequiresApproval_CoversHostExecChange(t *testing.T) {}
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/ai/tools/middleware ./internal/ai/tools/common -run="HostExecChange|NeedsApproval|FallbackRequiresApproval" -v`
Expected: FAIL

- [ ] **Step 3: Implement minimal mapping updates**

- [ ] **Step 4: Run tests and confirm pass**

Run: `go test ./internal/ai/tools/middleware ./internal/ai/tools/common -run="HostExecChange|NeedsApproval|FallbackRequiresApproval" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/tools/middleware/approval.go internal/ai/tools/common/approval_orchestrator.go internal/ai/tools/middleware/approval_test.go
git commit -m "feat(ai/approval): include host_exec_change in default approval policy coverage"
```

## Chunk 4: Toolset Boundaries, Audit Fields, and Full Verification

### Task 8: Update Diagnosis/Change tool assembly boundaries

**Files:**
- Modify: `internal/ai/tools/tools.go`
- Modify: `internal/ai/tools/tools_test.go`

- [ ] **Step 1: Write failing tests for tool boundaries**

```go
func TestNewDiagnosisTools_ExcludesHostExecChange(t *testing.T) {}
func TestNewChangeTools_IncludesHostExecChange(t *testing.T) {}
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/ai/tools -run="TestNewDiagnosisTools|TestNewChangeTools" -v`
Expected: FAIL

- [ ] **Step 3: Implement assembly updates**

Implementation points:
- diagnosis: readonly host execution only
- change: include change host execution plus required readonly tools

- [ ] **Step 4: Run tests and confirm pass**

Run: `go test ./internal/ai/tools -run="TestNewDiagnosisTools|TestNewChangeTools" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/tools/tools.go internal/ai/tools/tools_test.go
git commit -m "refactor(ai/tools): enforce host execution boundaries across diagnosis and change toolsets"
```

### Task 9: Extend audit fields (accountability chain)

**Files:**
- Modify: `internal/ai/tools/common/approval.go`
- Modify: `internal/ai/tools/common/approval_orchestrator.go`
- Modify: `internal/ai/tools/middleware/approval.go`
- Test: `internal/ai/tools/middleware/approval_test.go`

- [ ] **Step 1: Write failing tests for new audit fields**

```go
func TestApprovalAudit_RecordsApproverAndTimestamp(t *testing.T) {}
func TestApprovalAudit_RecordsRejectReason(t *testing.T) {}
func TestApprovalAudit_RecordsParseFailuresAndViolations(t *testing.T) {}
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/ai/tools/middleware ./internal/ai/tools/common -run="TestApprovalAudit" -v`
Expected: FAIL

- [ ] **Step 3: Implement field additions**

Implementation points:
- `approver_id`
- `approval_timestamp`
- `reject_reason`
- Policy-engine intercepted requests (`parse_error` / `policy_violation`) must also be audited with linked `approval_id`

- [ ] **Step 4: Run tests and confirm pass**

Run: `go test ./internal/ai/tools/middleware ./internal/ai/tools/common -run="TestApprovalAudit" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/tools/common/approval.go internal/ai/tools/common/approval_orchestrator.go internal/ai/tools/middleware/approval.go internal/ai/tools/middleware/approval_test.go
git commit -m "feat(ai/audit): record approver identity timestamp and reject reason in approval trail"
```

### Task 10: Full regression and documentation sync

**Files:**
- Modify: `CLAUDE.md`
- Modify: `docs/superpowers/specs/2026-03-22-host-command-policy-engine-design.md` (if implementation differs)

- [ ] **Step 1: Run core test suites**

Run:
- `go test ./internal/ai/tools/host -v`
- `go test ./internal/ai/tools/middleware -v`
- `go test ./internal/ai/tools/common -v`
- `go test ./internal/ai/tools -v`

Expected: all PASS

- [ ] **Step 2: Run targeted regression tests (approval + tool boundaries)**

Run: `go test ./internal/ai/... -run="Approval|HostExec|NewDiagnosisTools|NewChangeTools" -v`
Expected: PASS

- [ ] **Step 3: Update docs and runbook only where needed**

- [ ] **Step 4: Final commit**

```bash
git add CLAUDE.md docs/superpowers/specs/2026-03-22-host-command-policy-engine-design.md
git commit -m "docs: align host command policy implementation notes and tool boundary guidance"
```

- [ ] **Step 5: Delivery report**

Output:
- Change summary
- Security boundary verification results
- Residual risks and next-step recommendations

## Appendix: Implementation Constraints

1. DRY: keep policy decision logic in one place (PolicyEngine).
2. YAGNI: do not add multi-tenant allowlist UI or auto cross-agent transfer in this scope.
3. TDD: each task starts with failing tests, then minimal implementation.
4. Frequent commits: one commit per task for rollback safety.
5. No new bypass paths (including localhost direct shell bypass).

## Plan Review Notes

- Per skill guidance, each chunk should be reviewed by a plan-document-reviewer subagent.
- If explicit subagent review is not available in the current harness, perform equivalent self-review against the same checklist (completeness, clear boundaries, YAGNI, executable test instructions) and record results.
