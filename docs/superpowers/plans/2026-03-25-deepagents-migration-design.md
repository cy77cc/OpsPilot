# DeepAgents 架构迁移 Implementation Plan

> Superseded by `docs/superpowers/plans/2026-03-25-deepagents-unified-implementation.md`.

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 AI 后端从“未初始化 AIRouter + 旧规划语义残留”迁移到可运行的 DeepAgents 主入口，保留 HITL 审批和现有 SSE 兼容契约。

**Architecture:** 新增 `internal/ai/agent`（主入口与 Sub-Agent）+ `internal/ai/todo`（定制 `write_ops_todos`）作为核心；`internal/service/ai/logic.NewAILogic` 通过 feature flag 选择是否启用 DeepAgents；事件层保持现有 `meta/agent_handoff/delta/tool_approval/tool_result/run_state/done/error`，新增 `ops_plan_updated` 为可选增强事件。

**Tech Stack:** Go 1.26, CloudWeGo Eino ADK/DeepAgents, GORM, Gin, SSE, go test

---

## Scope Check

该 spec 涉及同一条运行时链路（agent 架构、todo 工具、审批恢复、SSE 投影、feature flag），属于一个可独立交付子系统，不再拆分。

## File Structure (Responsibilities)

### DeepAgents 主干（新增）
- Create: `internal/ai/agent/main.go`
- Create: `internal/ai/agent/prompt.go`
- Create: `internal/ai/agent/qa.go`
- Create: `internal/ai/agent/k8s.go`
- Create: `internal/ai/agent/host.go`
- Create: `internal/ai/agent/monitor.go`
- Create: `internal/ai/agent/change.go`
- Create: `internal/ai/agent/inspection.go`
- Create: `internal/ai/agent/main_test.go`
- Purpose: 定义可运行的 `OpsPilotAgent` 与 Sub-Agent 组装，替代当前 `AIRouter=nil` 状态。

### 定制 TODO 工具（新增）
- Create: `internal/ai/todo/types.go`
- Create: `internal/ai/todo/tool.go`
- Create: `internal/ai/todo/tool_test.go`
- Purpose: 定义 `OpsTODO` 与 `write_ops_todos` middleware，支持 full snapshot 存储。

### 逻辑层接入
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`
- Purpose: 在 `NewAILogic` 中初始化 DeepAgents，并保证失败时可控降级。

### 运行时事件与前端契约兼容
- Modify: `internal/ai/runtime/event_types.go`
- Modify: `internal/ai/runtime/event_types_test.go`
- Modify: `internal/ai/events/events.go`
- Modify: `internal/service/ai/logic/logic_test.go`
- Purpose: 新增 `ops_plan_updated`（可选）且保持旧事件不破坏。

### Feature flag 与配置
- Modify: `internal/config/config.go`
- Modify: `config/config.yaml` (若仓库有默认配置文件)
- Modify: `internal/service/ai/logic/logic_test.go`
- Purpose: 增加 `feature_flags.ai_deepagents` 控制迁移开关。

## Implementation Rules

- 使用 @test-driven-development：每个能力先写失败测试再实现。
- 使用 @verification-before-completion：每个任务结束必须跑最小回归。
- 保持 DRY/YAGNI：仅实现本次迁移所需最小 Agent 与工具集。
- 每个任务后立即 commit，避免大提交难回滚。

## Chunk 1: POC Gate（嵌套审批恢复）

### Task 1: 建立嵌套 interrupt/resume POC 测试

**Files:**
- Create: `internal/ai/agent/deepagents_poc_test.go`
- Test: `internal/ai/agent/deepagents_poc_test.go`

- [ ] **Step 1: 写失败测试，模拟 Main -> ChangeAgent -> tool_approval -> resume**

```go
func TestDeepAgentsPOC_InterruptResumeAcrossSubAgent(t *testing.T) {
    // 断言链路: waiting_approval -> resume -> completed
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/ai/agent -run TestDeepAgentsPOC_InterruptResumeAcrossSubAgent -v`
Expected: FAIL（包/构造尚不存在）

- [ ] **Step 3: 最小实现 POC scaffolding（仅测试所需 fake tool + resumable flow）**

```go
// create minimal deep.New config with one change subagent and approval middleware
```

- [ ] **Step 4: 重新运行 POC 测试**

Run: `go test ./internal/ai/agent -run TestDeepAgentsPOC_InterruptResumeAcrossSubAgent -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/agent/deepagents_poc_test.go
git commit -m "test(ai): add deepagents nested interrupt resume poc gate"
```

## Chunk 2: 定制 TODO 与事件协议

### Task 2: 实现 `OpsTODO` 类型与 session key

**Files:**
- Create: `internal/ai/todo/types.go`
- Create: `internal/ai/todo/types_test.go`
- Test: `internal/ai/todo/types_test.go`

- [ ] **Step 1: 写失败测试，断言 JSON 标签与状态枚举约束**

```go
func TestOpsTODO_JSONContract(t *testing.T) {
    // assert status/risk fields and json tags
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/ai/todo -run TestOpsTODO_JSONContract -v`
Expected: FAIL

- [ ] **Step 3: 最小实现 `OpsTODO` 与 `SessionKeyOpsTodos`**

```go
const SessionKeyOpsTodos = "opspilot_session_key_ops_todos"
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/ai/todo -run TestOpsTODO_JSONContract -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/todo/types.go internal/ai/todo/types_test.go
git commit -m "feat(ai): add ops todo model and session key"
```

### Task 3: 实现 `write_ops_todos` middleware

**Files:**
- Create: `internal/ai/todo/tool.go`
- Create: `internal/ai/todo/tool_test.go`
- Test: `internal/ai/todo/tool_test.go`

- [ ] **Step 1: 写失败测试，断言调用后会覆盖写入 Session todos 快照**

```go
func TestWriteOpsTodosMiddleware_StoresSnapshotInSession(t *testing.T) {}
```

- [ ] **Step 2: 写失败测试，断言 summary 文本包含状态符号与基础设施上下文**

```go
func TestWriteOpsTodosMiddleware_ReturnsFormattedSummary(t *testing.T) {}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/ai/todo -run WriteOpsTodosMiddleware -v`
Expected: FAIL

- [ ] **Step 4: 最小实现 middleware（含 `AdditionalInstruction` + `AdditionalTools`）**

```go
func NewWriteOpsTodosMiddleware() (adk.AgentMiddleware, error)
```

- [ ] **Step 5: 运行 todo 包回归**

Run: `go test ./internal/ai/todo -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ai/todo/tool.go internal/ai/todo/tool_test.go
git commit -m "feat(ai): implement write_ops_todos middleware"
```

### Task 4: 新增 `ops_plan_updated` 事件类型（可选增强）

**Files:**
- Modify: `internal/ai/runtime/event_types.go`
- Modify: `internal/ai/runtime/event_types_test.go`
- Modify: `internal/ai/events/events.go`
- Test: `internal/ai/runtime/event_types_test.go`

- [ ] **Step 1: 写失败测试，断言 `ops_plan_updated` payload 可序列化反序列化**

```go
func TestMarshalUnmarshal_OpsPlanUpdatedPayload(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/ai/runtime -run OpsPlanUpdated -v`
Expected: FAIL

- [ ] **Step 3: 最小实现事件常量与 payload 结构**

```go
const EventTypeOpsPlanUpdated EventType = "ops_plan_updated"
```

- [ ] **Step 4: 运行 runtime 回归**

Run: `go test ./internal/ai/runtime -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/runtime/event_types.go internal/ai/runtime/event_types_test.go internal/ai/events/events.go
git commit -m "feat(ai): add optional ops_plan_updated runtime event type"
```

## Chunk 3: DeepAgents 主入口与 Sub-Agent

### Task 5: 实现 Sub-Agent 构造器（QA/K8s/Host/Monitor/Change/Inspection）

**Files:**
- Create: `internal/ai/agent/qa.go`
- Create: `internal/ai/agent/k8s.go`
- Create: `internal/ai/agent/host.go`
- Create: `internal/ai/agent/monitor.go`
- Create: `internal/ai/agent/change.go`
- Create: `internal/ai/agent/inspection.go`
- Create: `internal/ai/agent/subagents_test.go`
- Test: `internal/ai/agent/subagents_test.go`

- [ ] **Step 1: 写失败测试，断言每个 Agent 名称、描述、MaxIterations 与工具集非空**

```go
func TestNewK8sAgent_Config(t *testing.T) {}
func TestNewChangeAgent_ContainsApprovalMiddleware(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/ai/agent -run "SubAgent|ChangeAgent" -v`
Expected: FAIL

- [ ] **Step 3: 最小实现各 Agent 构造器**

```go
func NewK8sAgent(ctx context.Context) (adk.Agent, error)
func NewChangeAgent(ctx context.Context) (adk.Agent, error)
```

- [ ] **Step 4: 运行子 Agent 回归**

Run: `go test ./internal/ai/agent -run "SubAgent|ChangeAgent" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/agent/qa.go internal/ai/agent/k8s.go internal/ai/agent/host.go internal/ai/agent/monitor.go internal/ai/agent/change.go internal/ai/agent/inspection.go internal/ai/agent/subagents_test.go

git commit -m "feat(ai): add deepagents subagent constructors"
```

### Task 6: 实现 `NewOpsPilotAgent` 主入口

**Files:**
- Create: `internal/ai/agent/main.go`
- Create: `internal/ai/agent/prompt.go`
- Create: `internal/ai/agent/main_test.go`
- Test: `internal/ai/agent/main_test.go`

- [ ] **Step 1: 写失败测试，断言主 Agent 配置启用 `WithoutWriteTodos=true` 与 todo middleware 注入**

```go
func TestNewOpsPilotAgent_UsesCustomWriteOpsTodos(t *testing.T) {}
```

- [ ] **Step 2: 写失败测试，断言 SubAgents 列表包含 ChangeAgent 且支持 task 委派**

```go
func TestNewOpsPilotAgent_RegistersSubAgents(t *testing.T) {}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/ai/agent -run NewOpsPilotAgent -v`
Expected: FAIL

- [ ] **Step 4: 最小实现主入口构造**

```go
func NewOpsPilotAgent(ctx context.Context) (adk.ResumableAgent, error)
```

- [ ] **Step 5: 运行 agent 包全量回归**

Run: `go test ./internal/ai/agent -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ai/agent/main.go internal/ai/agent/prompt.go internal/ai/agent/main_test.go

git commit -m "feat(ai): add opspilot deepagents main entry"
```

## Chunk 4: 逻辑层接入与开关控制

### Task 7: 增加 `feature_flags.ai_deepagents`

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go` (若存在；否则创建)
- Test: `internal/config/config_test.go`

- [ ] **Step 1: 写失败测试，断言新字段可从配置读取并默认 false**

```go
func TestFeatureFlags_AIDeepAgents_DefaultFalse(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/config -run AIDeepAgents -v`
Expected: FAIL

- [ ] **Step 3: 最小实现 FeatureFlags 字段与 getter**

```go
AIDeepAgents *bool `mapstructure:"ai_deepagents"`
```

- [ ] **Step 4: 运行配置测试**

Run: `go test ./internal/config -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add ai_deepagents feature flag"
```

### Task 8: `NewAILogic` 接入 DeepAgents，并保留安全降级

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`
- Test: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: 写失败测试，断言开关开启时 `AIRouter` 被初始化**

```go
func TestNewAILogic_InitializesDeepAgentsRouterWhenFlagEnabled(t *testing.T) {}
```

- [ ] **Step 2: 写失败测试，断言构造失败时不会 panic 且会降级到可用错误路径**

```go
func TestNewAILogic_DeepAgentsInitFailureDegradesGracefully(t *testing.T) {}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/service/ai/logic -run NewAILogic -v`
Expected: FAIL

- [ ] **Step 4: 最小实现 `logic.NewAILogic` 初始化路径**

```go
if config.IsAIDeepAgentsEnabled() {
    aiRouter, err := agent.NewOpsPilotAgent(runtimectx.WithServices(context.Background(), svcCtx))
    // graceful fallback when err != nil
}
```

- [ ] **Step 5: 运行 logic 包回归**

Run: `go test ./internal/service/ai/logic -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "feat(ai): wire deepagents router into ai logic initialization"
```

### Task 9: 端到端审批恢复链路回归（含 waiting_approval）

**Files:**
- Modify: `internal/service/ai/logic/approval_worker_test.go`
- Modify: `internal/service/ai/logic/logic_test.go`
- Test: `internal/service/ai/logic/approval_worker_test.go`

- [ ] **Step 1: 写失败测试，断言审批事件回放后 run 状态保持 `waiting_approval`**

```go
func TestChat_DeepAgentsApprovalReplayKeepsWaitingApproval(t *testing.T) {}
```

- [ ] **Step 2: 写失败测试，断言 ResumeWithParams 后状态回到执行并收敛 done**

```go
func TestResume_DeepAgentsApprovalFlowCompletes(t *testing.T) {}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/service/ai/logic -run "ApprovalReplay|ApprovalFlow" -v`
Expected: FAIL

- [ ] **Step 4: 最小实现必要适配（若需在投影层补 run_state）**

```go
// preserve waiting_approval semantics for nested deepagents interrupt path
```

- [ ] **Step 5: 运行审批链路回归**

Run: `go test ./internal/service/ai/logic -run "Approval" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/service/ai/logic/approval_worker_test.go internal/service/ai/logic/logic_test.go
git commit -m "test(ai): verify deepagents approval interrupt and resume chain"
```

## Final Verification

- [ ] **Step 1: 运行后端 AI 迁移核心测试**

Run: `go test ./internal/ai/agent ./internal/ai/todo ./internal/ai/runtime ./internal/service/ai/logic -v`
Expected: PASS

- [ ] **Step 2: 运行 AI 服务层回归**

Run: `go test ./internal/service/ai/handler ./internal/service/ai/logic`
Expected: PASS

- [ ] **Step 3: 运行仓库基础检查（若项目已有 lint 命令则执行）**

Run: `go test ./...`
Expected: PASS（或仅出现已知非本变更问题）

- [ ] **Step 4: 最终 Commit（若前序任务未拆 commit，则在此补齐）**

```bash
git add internal/ai/agent internal/ai/todo internal/ai/runtime/event_types.go internal/ai/runtime/event_types_test.go internal/ai/events/events.go internal/config/config.go internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go internal/service/ai/logic/approval_worker_test.go
git commit -m "feat(ai): migrate runtime to deepagents architecture with approval compatibility"
```
