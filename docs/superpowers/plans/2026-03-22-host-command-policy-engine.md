# Host Command Policy Engine Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将主机命令执行能力重构为统一策略内核，并收敛为 `host_exec_readonly` / `host_exec_change` 两个入口，确保 AST+白名单+审批边界可审计、可回放防护、默认 fail-closed。

**Architecture:** 在 `internal/ai/tools/host` 新增策略引擎（解析、校验、决策）并由新工具统一调用。`readonly` 只在 AST+allowlist 全通过时执行，否则立即中断审批；`change` 默认审批。审批恢复使用参数快照 + 会话绑定校验，避免重放。旧工具进入兼容转发并逐步下线。

**Tech Stack:** Go, `mvdan.cc/sh/v3/syntax`, Eino tool middleware/interrupt, existing approval orchestrator, Go test。

---

## Chunk 1: 策略引擎与 AST 校验核心

### Task 1: 新增 Host 命令策略核心类型（TDD）

**Files:**
- Create: `internal/ai/tools/host/policy_engine.go`
- Create: `internal/ai/tools/host/policy_types.go`
- Test: `internal/ai/tools/host/policy_engine_test.go`

- [ ] **Step 1: 写失败测试，定义决策枚举与 fail-closed 基线**

```go
func TestPolicyEngine_FailClosedOnParserError(t *testing.T) {
    engine := NewHostCommandPolicyEngine(DefaultReadonlyAllowlist())
    got := engine.Evaluate(PolicyInput{ToolName: "host_exec_readonly", CommandRaw: "echo $("})
    require.Equal(t, DecisionRequireApprovalInterrupt, got.DecisionType)
    require.Contains(t, got.ReasonCodes, "parse_error")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/ai/tools/host -run TestPolicyEngine_FailClosedOnParserError -v`
Expected: FAIL（类型或引擎未实现）

- [ ] **Step 3: 最小实现策略类型与 Evaluate 骨架**

```go
type DecisionType string
const (
    DecisionAllowReadonlyExecute DecisionType = "allow_readonly_execute"
    DecisionRequireApprovalInterrupt DecisionType = "require_approval_interrupt"
)
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/ai/tools/host -run TestPolicyEngine_FailClosedOnParserError -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ai/tools/host/policy_engine.go internal/ai/tools/host/policy_types.go internal/ai/tools/host/policy_engine_test.go
git commit -m "feat(ai/host): add policy engine skeleton with fail-closed decision model"
```

### Task 2: 集成 AST 解析与节点遍历能力（TDD）

**Files:**
- Create: `internal/ai/tools/host/ast_parser.go`
- Modify: `internal/ai/tools/host/policy_engine.go`
- Test: `internal/ai/tools/host/ast_parser_test.go`
- Test: `internal/ai/tools/host/policy_engine_test.go`

- [ ] **Step 1: 写失败测试，验证 parse error 与命令节点提取**

```go
func TestParseCommand_CollectsPipelineCommands(t *testing.T) {
    parsed, err := ParseCommand("cat /var/log/syslog | grep error")
    require.NoError(t, err)
    require.ElementsMatch(t, []string{"cat", "grep"}, parsed.BaseCommands)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/ai/tools/host -run 'TestParseCommand|TestPolicyEngine' -v`
Expected: FAIL

- [ ] **Step 3: 最小实现 Parser（mvdan）+ AST 摘要输出**

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/ai/tools/host -run 'TestParseCommand|TestPolicyEngine' -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ai/tools/host/ast_parser.go internal/ai/tools/host/ast_parser_test.go internal/ai/tools/host/policy_engine.go internal/ai/tools/host/policy_engine_test.go
git commit -m "feat(ai/host): parse shell commands via mvdan AST for policy evaluation"
```

### Task 3: Allowlist + 操作符规则（含 awk 下线、管道逐段校验、链路逐段校验）

**Files:**
- Create: `internal/ai/tools/host/validator.go`
- Modify: `internal/ai/tools/host/policy_engine.go`
- Modify: `internal/ai/tools/host/policy_types.go`
- Test: `internal/ai/tools/host/validator_test.go`
- Test: `internal/ai/tools/host/policy_engine_test.go`

- [ ] **Step 1: 写失败测试覆盖关键规则**

```go
func TestValidator_RejectsNonAllowlistedCommand(t *testing.T) {}
func TestValidator_RejectsAwkInInitialAllowlist(t *testing.T) {}
func TestValidator_AllowsPipelineWhenEachCommandAllowlisted(t *testing.T) {}
func TestValidator_RejectsRedirectionAndBackground(t *testing.T) {}
func TestValidator_CommandChainRequiresEachSegmentAllowlisted(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/ai/tools/host -run 'TestValidator|TestPolicyEngine' -v`
Expected: FAIL

- [ ] **Step 3: 最小实现 Validator**

实现点：
- 初始 allowlist: `cat, ls, grep, top, free, df, tail`（不含 `awk`）
- 禁止：重定向、后台、命令替换
- 允许：`|`，但每个子命令独立过 allowlist
- `; && ||`：逐段校验；任一失败 -> 审批

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/ai/tools/host -run 'TestValidator|TestPolicyEngine' -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ai/tools/host/validator.go internal/ai/tools/host/validator_test.go internal/ai/tools/host/policy_engine.go internal/ai/tools/host/policy_types.go internal/ai/tools/host/policy_engine_test.go
git commit -m "feat(ai/host): enforce allowlist and operator validation on command AST"
```

## Chunk 2: 工具收敛与兼容迁移

### Task 4: 新增 `host_exec_readonly` / `host_exec_change` 并接入策略引擎

**Files:**
- Modify: `internal/ai/tools/host/tools.go`
- Test: `internal/ai/tools/host/tools_test.go`

- [ ] **Step 1: 写失败测试验证新工具注册与行为**

```go
func TestNewHostReadonlyTools_ContainsHostExecReadonlyOnly(t *testing.T) {}
func TestHostExecReadonly_InterruptsWhenValidationFails(t *testing.T) {}
func TestHostExecChange_AlwaysRequestsApprovalBeforeExecution(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/ai/tools/host -run 'TestNewHostReadonlyTools|TestHostExecReadonly|TestHostExecChange' -v`
Expected: FAIL

- [ ] **Step 3: 实现新工具入口 + 统一 Facade**

实现点：
- `host_exec_readonly`：策略允许才执行
- `host_exec_change`：默认审批
- 输出中包含 policy decision 摘要字段（用于审计/排障）

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/ai/tools/host -run 'TestNewHostReadonlyTools|TestHostExecReadonly|TestHostExecChange' -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ai/tools/host/tools.go internal/ai/tools/host/tools_test.go
git commit -m "feat(ai/host): add readonly/change execution tools backed by policy engine"
```

### Task 5: 旧工具兼容转发与本地旁路收口

**Files:**
- Modify: `internal/ai/tools/host/tools.go`
- Test: `internal/ai/tools/host/tools_test.go`

- [ ] **Step 1: 写失败测试验证旧入口不会绕过策略**

```go
func TestLegacyHostExec_UsesPolicyEngine(t *testing.T) {}
func TestLegacyHostExecByTarget_LocalhostCannotBypassPolicy(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/ai/tools/host -run 'TestLegacyHostExec|TestLegacyHostExecByTarget' -v`
Expected: FAIL

- [ ] **Step 3: 实现旧工具到新门面的转发**

实现点：
- `host_exec` -> 转发到 readonly/change 路径（按策略决策）
- `host_exec_by_target` -> 统一 target 解析后仍走策略，不保留 localhost 旁路
- `host_ssh_exec_readonly` -> 内部复用 readonly 新实现

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/ai/tools/host -run 'TestLegacyHostExec|TestLegacyHostExecByTarget' -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ai/tools/host/tools.go internal/ai/tools/host/tools_test.go
git commit -m "refactor(ai/host): route legacy host exec tools through unified policy path"
```

## Chunk 3: 审批桥接、恢复绑定、防重放

### Task 6: 审批中断返回挂起状态并扩展上下文绑定

**Files:**
- Modify: `internal/ai/tools/middleware/approval.go`
- Modify: `internal/ai/tools/common/approval_orchestrator.go`
- Test: `internal/ai/tools/middleware/approval_test.go`

- [ ] **Step 1: 写失败测试覆盖挂起语义与绑定校验**

```go
func TestApprovalBridge_ReturnsSuspendedPayload(t *testing.T) {}
func TestApprovalResume_RejectsMismatchedSessionOrRole(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/ai/tools/middleware -run 'TestApprovalBridge|TestApprovalResume' -v`
Expected: FAIL

- [ ] **Step 3: 实现审批桥接增强**

实现点：
- 中断 payload 明确 `status=suspended` 与 `approval_id`
- 绑定并校验 `approval_id + session_id + agent_role`
- 绑定失败时重新中断

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/ai/tools/middleware -run 'TestApprovalBridge|TestApprovalResume' -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ai/tools/middleware/approval.go internal/ai/tools/common/approval_orchestrator.go internal/ai/tools/middleware/approval_test.go
git commit -m "feat(ai/approval): return suspended interrupts and enforce resume session-role binding"
```

### Task 7: `host_exec_change` 纳入默认审批覆盖与命令类映射

**Files:**
- Modify: `internal/ai/tools/middleware/approval.go`
- Modify: `internal/ai/tools/common/approval_orchestrator.go`
- Test: `internal/ai/tools/middleware/approval_test.go`

- [ ] **Step 1: 写失败测试验证审批覆盖**

```go
func TestDefaultNeedsApproval_CoversHostExecChange(t *testing.T) {}
func TestFallbackRequiresApproval_CoversHostExecChange(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/ai/tools/middleware ./internal/ai/tools/common -run 'HostExecChange|NeedsApproval|FallbackRequiresApproval' -v`
Expected: FAIL

- [ ] **Step 3: 最小实现映射更新**

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/ai/tools/middleware ./internal/ai/tools/common -run 'HostExecChange|NeedsApproval|FallbackRequiresApproval' -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ai/tools/middleware/approval.go internal/ai/tools/common/approval_orchestrator.go internal/ai/tools/middleware/approval_test.go
git commit -m "feat(ai/approval): include host_exec_change in default approval policy coverage"
```

## Chunk 4: 工具集边界、审计字段与全量验证

### Task 8: 更新 Diagnosis/Change 工具装配边界

**Files:**
- Modify: `internal/ai/tools/tools.go`
- Modify: `internal/ai/tools/tools_test.go`

- [ ] **Step 1: 写失败测试验证工具边界**

```go
func TestNewDiagnosisTools_ExcludesHostExecChange(t *testing.T) {}
func TestNewChangeTools_IncludesHostExecChange(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/ai/tools -run 'TestNewDiagnosisTools|TestNewChangeTools' -v`
Expected: FAIL

- [ ] **Step 3: 实现装配调整**

实现点：
- diagnosis: 仅 readonly 主机执行入口
- change: 增加 change 主机执行入口与必要 readonly

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/ai/tools -run 'TestNewDiagnosisTools|TestNewChangeTools' -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ai/tools/tools.go internal/ai/tools/tools_test.go
git commit -m "refactor(ai/tools): enforce host execution boundaries across diagnosis and change toolsets"
```

### Task 9: 审计字段扩展（责任链）

**Files:**
- Modify: `internal/ai/tools/common/approval.go`
- Modify: `internal/ai/tools/common/approval_orchestrator.go`
- Modify: `internal/ai/tools/middleware/approval.go`
- Test: `internal/ai/tools/middleware/approval_test.go`

- [ ] **Step 1: 写失败测试验证新字段写入**

```go
func TestApprovalAudit_RecordsApproverAndTimestamp(t *testing.T) {}
func TestApprovalAudit_RecordsRejectReason(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/ai/tools/middleware ./internal/ai/tools/common -run 'TestApprovalAudit' -v`
Expected: FAIL

- [ ] **Step 3: 实现字段补充**

实现点：
- `approver_id`
- `approval_timestamp`
- `reject_reason`

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/ai/tools/middleware ./internal/ai/tools/common -run 'TestApprovalAudit' -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ai/tools/common/approval.go internal/ai/tools/common/approval_orchestrator.go internal/ai/tools/middleware/approval.go internal/ai/tools/middleware/approval_test.go
git commit -m "feat(ai/audit): record approver identity timestamp and reject reason in approval trail"
```

### Task 10: 全量回归与文档同步

**Files:**
- Modify: `CLAUDE.md`
- Modify: `docs/superpowers/specs/2026-03-22-host-command-policy-engine-design.md` (如实现偏差需回写)

- [ ] **Step 1: 运行核心测试集**

Run:
- `go test ./internal/ai/tools/host -v`
- `go test ./internal/ai/tools/middleware -v`
- `go test ./internal/ai/tools/common -v`
- `go test ./internal/ai/tools -v`

Expected: 全部 PASS

- [ ] **Step 2: 运行目标回归测试（审批 + 工具边界）**

Run: `go test ./internal/ai/... -run 'Approval|HostExec|NewDiagnosisTools|NewChangeTools' -v`
Expected: PASS

- [ ] **Step 3: 更新文档与运行手册（仅必要差异）**

- [ ] **Step 4: 最终提交**

```bash
git add CLAUDE.md docs/superpowers/specs/2026-03-22-host-command-policy-engine-design.md
git commit -m "docs: align host command policy implementation notes and tool boundary guidance"
```

- [ ] **Step 5: 交付说明**

输出：
- 变更摘要
- 安全边界验证结果
- 未覆盖风险与后续建议

## 附录: 实施约束

1. DRY：策略判定逻辑只能在 PolicyEngine 中维护一份。
2. YAGNI：不在本次实现多租户白名单 UI 与自动跨 agent 流转。
3. TDD：每个任务先写失败测试，再最小实现。
4. Frequent commits：每个任务独立提交，便于回滚。
5. 不得新增旁路执行路径（包括 localhost 本地直接 shell）。

## Plan Review Notes

- 按 skill 要求应使用 plan-document-reviewer 子代理做 chunk review。
- 若当前执行环境不允许显式派生子代理，则由主执行者按同一检查项（完整性、边界清晰、YAGNI、测试可执行性）进行等价自审并记录。
