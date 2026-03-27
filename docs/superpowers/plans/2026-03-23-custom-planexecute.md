# Custom PlanExecute Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用自研 `planner + executor + replanner` 替换 Eino prebuilt `planexecute`，同时保持现有审批中断/恢复协议与前端流式体验不回归。

**Architecture:** 新建 `internal/ai/planexecute` 包承载计划与执行内核，仍通过 Eino ADK `Agent/ResumableAgent` 与 `Runner` 集成，不重写 Runner 协议。保留 `checkpointID + ResumeWithParams(Targets[toolCallID])` 的恢复路径，先完成串行执行，再逐步引入优化（收敛判断/回滚检测/步骤优化）。

**Tech Stack:** Go, CloudWeGo Eino ADK, existing `internal/ai/checkpoint/store`, existing AI approval middleware, Go test tooling.

---

## Scope Check

该 spec 覆盖三个强耦合子系统（planner/executor/replanner）和两处集成面（diagnosis/change agent）。它们共享同一恢复协议与事件链路，不适合拆成独立交付计划；本计划按“内核 -> 集成 -> 迁移”分阶段推进，每阶段都可独立验收。

## Execution Prerequisites

- 在独立 worktree 执行（`@using-git-worktrees`）。
- 按 TDD 落地（`@test-driven-development`）：先写失败测试，再最小实现。
- 每个任务完成后跑验证并小步提交（`@verification-before-completion`）。

## File Structure

### New Files

- `internal/ai/planexecute/plan.go`
  - `Plan/Step/Condition`、状态枚举、序列化辅助。
- `internal/ai/planexecute/context.go`
  - `ExecutionContext/ExecutedStep/Checkpoint`、runtime 辅助。
- `internal/ai/planexecute/planner.go`
  - Planner、schema 输出、模板注入入口。
- `internal/ai/planexecute/executor.go`
  - Step 执行循环、依赖检查、超时、动态工具选择。
- `internal/ai/planexecute/replanner.go`
  - Replan 决策、收敛判断入口、回滚/优化钩子。
- `internal/ai/planexecute/agent.go`
  - ADK Agent 适配层（`Run/Resume`）。
- `internal/ai/planexecute/checkpoint_adapter.go`
  - `checkpoint.Store` 适配为 plan runtime checkpoint 接口。
- `internal/ai/planexecute/validator.go`
  - 计划合法性校验（依赖、工具、变量）。
- `internal/ai/planexecute/templates/registry.go`
  - 模板注册与查询。
- `internal/ai/planexecute/templates/diagnosis.go`
  - 诊断模板。
- `internal/ai/planexecute/templates/change.go`
  - 变更与回滚模板。

### Modified Files

- `internal/ai/agents/diagnosis/agent.go`
  - 引入 feature flag 切换到 custom planexecute。
- `internal/ai/agents/change/agent.go`
  - 引入 feature flag 切换到 custom planexecute。
- `internal/service/ai/logic/logic.go`
  - 确认自研 agent 下 `Runner.ResumeWithParams` 行为不变。
- `internal/service/ai/logic/approval_worker.go`
  - 确认异步审批恢复路径不变。
- `internal/ai/runtime/*`（按实际需要）
  - 仅在事件投影字段不兼容时做最小适配。
- `configs/config.yaml` / 对应配置结构体
  - 增加 `feature_flags.custom_planexecute`。
- `web/src/api/modules/ai.ts`
  - 保持 SSE/A2UI 事件类型兼容；必要时新增可选字段但不破坏旧字段。
- `web/src/components/AI/a2uiState.ts`
  - 兼容 plan/replan 新旧 payload，确保 UI 状态机稳定。
- `web/src/components/AI/replyRuntime.ts`
  - 兼容运行态映射与步骤展示逻辑。
- `web/src/components/AI/providers/PlatformChatProvider.ts`
  - 校验流事件消费路径在 custom planexecute 下不回归。
- `web/src/components/AI/ToolReference.tsx`
  - 审批状态展示与交互兼容验证（waiting/submitting/resolved）。

### Test Files

- `internal/ai/planexecute/plan_test.go`
- `internal/ai/planexecute/planner_test.go`
- `internal/ai/planexecute/executor_test.go`
- `internal/ai/planexecute/replanner_test.go`
- `internal/ai/planexecute/agent_test.go`
- `internal/ai/planexecute/checkpoint_adapter_test.go`
- `internal/ai/agents/diagnosis/agent_test.go`（扩展）
- `internal/ai/agents/change/agent_test.go`（扩展）
- `internal/service/ai/logic/*approval*_test.go`（补充回归）
- `web/src/api/modules/ai.streamChunk.test.ts`（扩展）
- `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`（扩展）
- `web/src/components/AI/__tests__/AssistantReply.test.tsx`（扩展）
- `web/src/components/AI/replyRuntime.test.ts`（扩展）
- `web/src/components/AI/historyProjection.test.ts`（扩展）

## Chunk 1: Core Runtime (package `internal/ai/planexecute`)

### Task 1: Core Types + Validation Base

**Files:**
- Create: `internal/ai/planexecute/plan.go`
- Create: `internal/ai/planexecute/context.go`
- Create: `internal/ai/planexecute/validator.go`
- Test: `internal/ai/planexecute/plan_test.go`

- [ ] **Step 1: Write failing tests for type invariants and dependency validation**

```go
func TestPlan_ValidateDependencies_MissingDep(t *testing.T) {
    plan := &Plan{Steps: []*Step{{ID: "s2", DependsOn: []string{"s1"}}}}
    v := NewPlanValidator([]string{"k8s_list_resources"})
    err := v.Validate(plan, []ToolMeta{{Name: "k8s_list_resources"}})
    require.Error(t, err)
    require.Contains(t, err.Error(), "missing dependency")
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/ai/planexecute -run TestPlan_ValidateDependencies_MissingDep -v`
Expected: FAIL with missing implementation errors.

- [ ] **Step 3: Implement minimal types and validator to pass tests**

```go
func (v *PlanValidator) Validate(plan *Plan, available []ToolMeta) error {
    stepIDs := map[string]struct{}{}
    for _, s := range plan.Steps { stepIDs[s.ID] = struct{}{} }
    for _, s := range plan.Steps {
        for _, dep := range s.DependsOn {
            if _, ok := stepIDs[dep]; !ok { return fmt.Errorf("missing dependency: %s", dep) }
        }
    }
    return nil
}
```

- [ ] **Step 4: Run focused tests**

Run: `go test ./internal/ai/planexecute -run TestPlan_ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ai/planexecute/plan.go internal/ai/planexecute/context.go internal/ai/planexecute/validator.go internal/ai/planexecute/plan_test.go
git commit -m "feat(ai): add planexecute core types and validator"
```

### Task 2: Planner (Structured Output + Template Hook)

**Files:**
- Create: `internal/ai/planexecute/planner.go`
- Create: `internal/ai/planexecute/templates/registry.go`
- Create: `internal/ai/planexecute/templates/diagnosis.go`
- Create: `internal/ai/planexecute/templates/change.go`
- Test: `internal/ai/planexecute/planner_test.go`

- [ ] **Step 1: Write failing tests for planner output parsing and template injection**

```go
func TestPlanner_Plan_AppliesTemplate(t *testing.T) {
    // mock model returns plan JSON; assert template steps injected and validated
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/ai/planexecute -run TestPlanner_ -v`
Expected: FAIL with planner/template symbols undefined.

- [ ] **Step 3: Implement minimal planner and schema helper**

```go
func (p *Planner) Plan(ctx context.Context, input []adk.Message) (*Plan, error) {
    msgs, _ := p.genInputFn(ctx, p.buildPlannerInput(input))
    var plan Plan
    if err := generateWithSchema(ctx, p.model, msgs, &plan, planSchema); err != nil { return nil, err }
    p.applyTemplates(&plan)
    return &plan, p.validator.Validate(&plan, nil)
}
```

- [ ] **Step 4: Run planner tests**

Run: `go test ./internal/ai/planexecute -run TestPlanner_ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ai/planexecute/planner.go internal/ai/planexecute/templates/*.go internal/ai/planexecute/planner_test.go
git commit -m "feat(ai): implement planexecute planner with template support"
```

### Task 3: Executor (Dependency, Timeout, Tool Selection, Checkpoint Save)

**Files:**
- Create: `internal/ai/planexecute/executor.go`
- Create: `internal/ai/planexecute/checkpoint_adapter.go`
- Test: `internal/ai/planexecute/executor_test.go`
- Test: `internal/ai/planexecute/checkpoint_adapter_test.go`

- [ ] **Step 1: Write failing tests for dependency gate and timeout status**

```go
func TestExecutor_CheckDependencies_RequiresCompletedDeps(t *testing.T) {
    // dep not executed or failed -> false
}

func TestExecutor_ExecuteStep_TimeoutMarksStepTimeout(t *testing.T) {
    // step ctx deadline exceeded -> StepStatusTimeout
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/ai/planexecute -run 'TestExecutor_|TestPlanCheckpointStore_' -v`
Expected: FAIL.

- [ ] **Step 3: Implement executor core and checkpoint adapter**

```go
func (e *Executor) checkDependencies(execCtx *ExecutionContext, step *Step) bool {
    statusByStepID := map[string]StepStatus{}
    for _, ex := range execCtx.ExecutedSteps { statusByStepID[ex.Step.ID] = ex.Status }
    for _, dep := range step.DependsOn {
        if statusByStepID[dep] != StepStatusCompleted { return false }
    }
    return true
}
```

- [ ] **Step 4: Run executor tests**

Run: `go test ./internal/ai/planexecute -run TestExecutor_ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ai/planexecute/executor.go internal/ai/planexecute/checkpoint_adapter.go internal/ai/planexecute/executor_test.go internal/ai/planexecute/checkpoint_adapter_test.go
git commit -m "feat(ai): implement planexecute executor and checkpoint adapter"
```

## Chunk 2: Replanner + ADK Integration + Rollout

### Task 4: Replanner (Convergence + Optimization/Rollback Hooks)

**Files:**
- Create: `internal/ai/planexecute/replanner.go`
- Test: `internal/ai/planexecute/replanner_test.go`

- [ ] **Step 1: Write failing tests for replan actions (`complete/continue/rollback/wait`)**

```go
func TestReplanner_Run_CompleteOnHighConfidenceConvergence(t *testing.T) {}
func TestReplanner_Run_ReturnRollbackWhenDetectorRequests(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/ai/planexecute -run TestReplanner_ -v`
Expected: FAIL.

- [ ] **Step 3: Implement minimal replanner orchestration**

```go
func (r *Replanner) Run(ctx context.Context, rctx *ReplannerContext) (*ReplanResult, error) {
    conv, _ := r.convergenceChecker.Check(ctx, rctx)
    if conv.Converged && conv.Confidence > 0.8 { return &ReplanResult{Action: ReplanActionComplete}, nil }
    rb, _ := r.rollbackDetector.Analyze(ctx, rctx)
    if rb.NeedRollback { return &ReplanResult{Action: ReplanActionRollback, NewSteps: rb.RollbackSteps}, nil }
    return &ReplanResult{Action: ReplanActionContinue}, nil
}
```

- [ ] **Step 4: Run replanner tests**

Run: `go test ./internal/ai/planexecute -run TestReplanner_ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ai/planexecute/replanner.go internal/ai/planexecute/replanner_test.go
git commit -m "feat(ai): implement planexecute replanner pipeline"
```

### Task 5: ADK Agent Adapter (`Run/Resume`) + Engine Loop

**Files:**
- Create: `internal/ai/planexecute/agent.go`
- Test: `internal/ai/planexecute/agent_test.go`

- [ ] **Step 1: Write failing tests for ADK interface conformance and resume flow**

```go
func TestPlanExecuteAgent_ImplementsResumableAgent(t *testing.T) {
    var _ adk.ResumableAgent = (*PlanExecuteAgent)(nil)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/ai/planexecute -run TestPlanExecuteAgent_ -v`
Expected: FAIL.

- [ ] **Step 3: Implement `Run/Resume` event adapter**

```go
func (a *PlanExecuteAgent) Resume(ctx context.Context, info *adk.ResumeInfo, opts ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
    // restore execution from checkpoint-backed state and continue emitting AgentEvent
    return iter
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/ai/planexecute -run TestPlanExecuteAgent_ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ai/planexecute/agent.go internal/ai/planexecute/agent_test.go
git commit -m "feat(ai): add custom planexecute ADK agent adapter"
```

### Task 6: Backend Contract Compatibility Layer (SSE + Persisted Runtime)

**Files:**
- Modify: `internal/ai/runtime/project.go`
- Modify: `internal/ai/runtime/event_types.go`
- Modify: `internal/ai/runtime/projection_builder.go`
- Test: `internal/ai/runtime/project_test.go`
- Test: `internal/ai/runtime/event_types_test.go`
- Test: `internal/ai/runtime/streamer_test.go`

- [ ] **Step 1: Add failing tests for backward-compatible plan/replan payloads**

```go
func TestPlanEvent_StillEmitsStepsStringArray_WhenInternalStepIsRichStruct(t *testing.T) {}
func TestReplanEvent_StillEmitsLegacyFields_IterationCompletedIsFinal(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/ai/runtime -run 'TestPlanEvent_|TestReplanEvent_' -v`
Expected: FAIL.

- [ ] **Step 3: Implement anti-corruption mapping**

```go
// internal rich step -> external legacy payload
payload.Steps = mapStepsToTitles(richSteps) // []string
// keep iteration/completed/is_final fields stable
```

- [ ] **Step 4: Run runtime tests**

Run: `go test ./internal/ai/runtime -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ai/runtime/project.go internal/ai/runtime/event_types.go internal/ai/runtime/projection_builder.go internal/ai/runtime/project_test.go internal/ai/runtime/event_types_test.go internal/ai/runtime/streamer_test.go
git commit -m "feat(ai): add runtime contract compatibility for custom planexecute"
```

### Task 7: Diagnosis Agent Integration Behind Feature Flag

**Files:**
- Modify: `internal/ai/agents/diagnosis/agent.go`
- Modify: config struct files that load `feature_flags.custom_planexecute`
- Test: `internal/ai/agents/diagnosis/agent_test.go`

- [ ] **Step 1: Add failing tests for flag switch behavior**

```go
func TestNewDiagnosisAgent_UsesCustomPlanExecuteWhenFlagEnabled(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/ai/agents/diagnosis -run TestNewDiagnosisAgent_ -v`
Expected: FAIL.

- [ ] **Step 3: Implement flag-gated constructor path**

```go
if featureFlags.CustomPlanExecute {
    return newCustomDiagnosisAgent(ctx)
}
return newEinoDiagnosisAgent(ctx)
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/ai/agents/diagnosis -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ai/agents/diagnosis/agent.go internal/ai/agents/diagnosis/agent_test.go
# include config files touched by new flag
git commit -m "feat(ai): wire diagnosis agent to custom planexecute via feature flag"
```

### Task 8: Change Agent Integration + Approval Resume Regression

**Files:**
- Modify: `internal/ai/agents/change/agent.go`
- Modify: `internal/service/ai/logic/logic.go` (only if compatibility patch needed)
- Modify: `internal/service/ai/logic/approval_worker.go` (only if compatibility patch needed)
- Test: `internal/ai/agents/change/agent_test.go`
- Test: `internal/service/ai/logic/approval_worker_test.go`

- [ ] **Step 1: Add failing regression tests for resume contract unchanged**

```go
func TestApprovalResume_StillTargetsToolCallID(t *testing.T) {
    // assert Targets map key is toolCallID, not planID
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/service/ai/logic -run TestApprovalResume_ -v`
Expected: FAIL.

- [ ] **Step 3: Integrate custom change agent and keep Runner contract unchanged**

```go
resumeParams := &adk.ResumeParams{Targets: map[string]any{task.ToolCallID: approvalResult}}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/ai/agents/change ./internal/service/ai/logic -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ai/agents/change/agent.go internal/ai/agents/change/agent_test.go internal/service/ai/logic/approval_worker_test.go
# include logic.go/approval_worker.go only if changed
git commit -m "feat(ai): migrate change agent to custom planexecute with approval resume compatibility"
```

### Task 9: Frontend Runtime Compatibility + Approval UX Regression

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/components/AI/a2uiState.ts`
- Modify: `web/src/components/AI/replyRuntime.ts`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/ToolReference.tsx` (only if state mapping changes)
- Test: `web/src/api/modules/ai.streamChunk.test.ts`
- Test: `web/src/components/AI/replyRuntime.test.ts`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Test: `web/src/components/AI/historyProjection.test.ts`

- [ ] **Step 1: Add failing frontend tests for legacy+new payload compatibility**

```ts
it('parses plan/replan events when backend keeps steps:string[] contract', () => {})
it('keeps approval state transitions stable after resume', () => {})
```

- [ ] **Step 2: Run tests to verify failure**

Run: `npm --prefix web run test:run -- ai.streamChunk replyRuntime PlatformChatProvider AssistantReply historyProjection`
Expected: FAIL.

- [ ] **Step 3: Implement frontend compatibility mapping**

```ts
// ai.ts: keep A2UIPlanEvent / A2UIReplanEvent stable
type A2UIPlanEvent = { steps: string[]; iteration: number }
// a2uiState/replyRuntime: treat unknown fields as optional extensions
```

- [ ] **Step 4: Run frontend tests and build**

Run:
- `npm --prefix web run test:run -- ai.streamChunk replyRuntime PlatformChatProvider AssistantReply historyProjection`
- `npm --prefix web run build`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/api/modules/ai.ts web/src/components/AI/a2uiState.ts web/src/components/AI/replyRuntime.ts web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/ToolReference.tsx web/src/api/modules/ai.streamChunk.test.ts web/src/components/AI/replyRuntime.test.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/AssistantReply.test.tsx web/src/components/AI/historyProjection.test.ts
git commit -m "feat(ai-web): keep runtime and approval UX compatible with custom planexecute"
```

### Task 10: End-to-End Verification, Cleanup, and Docs

**Files:**
- Modify: `docs/superpowers/specs/2026-03-23-custom-planexecute-design.md` (status + final decisions)
- Modify: any release notes / ops docs that mention planexecute runtime

- [ ] **Step 1: Run full targeted test suite**

Run:
- `go test ./internal/ai/planexecute/... -v`
- `go test ./internal/ai/runtime -v`
- `go test ./internal/ai/agents/diagnosis ./internal/ai/agents/change -v`
- `go test ./internal/service/ai/logic -run 'Approval|Resume' -v`
- `npm --prefix web run test:run -- ai.streamChunk replyRuntime PlatformChatProvider AssistantReply historyProjection`
- `npm --prefix web run build`

Expected: PASS.

- [ ] **Step 2: Run broader regression suite (if available in repo)**

Run: `go test ./...`
Expected: PASS (or documented known failures unrelated to this change).

- [ ] **Step 3: Remove or isolate prebuilt dependency usage**

Run: `rg -n "adk/prebuilt/planexecute" internal`
Expected: only expected fallback paths remain (or zero if fully migrated).

- [ ] **Step 4: Final commit**

```bash
git add docs/superpowers/specs/2026-03-23-custom-planexecute-design.md docs/superpowers/plans/2026-03-23-custom-planexecute.md
git commit -m "docs(ai): finalize custom planexecute migration and verification plan"
```

## Review Loop Instructions (Per Chunk)

- 对每个 chunk 完成后，执行一次计划文档审查（plan-document-reviewer prompt）。
- 若发现问题，必须在当前 chunk 修复并复审，直到通过。
- 单 chunk 审查重试超过 5 次时，暂停并请求人工决策。

## Done Criteria

- `custom_planexecute=true` 时，Diagnosis/Change 均走自研内核并通过关键回归测试。
- `custom_planexecute=false` 时，仍可走原有路径（安全回滚）。
- 审批恢复协议未变：继续使用 `checkpointID + Targets[toolCallID]`。
- 无新增前端运行态字段破坏；SSE 事件语义保持兼容。
- 前端关键链路（plan/replan 渲染、approval 状态流转、history hydration）在 `vitest + build` 下全部通过。
