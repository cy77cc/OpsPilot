# DeepAgents Unified Migration Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 以单一计划完成 DeepAgents 后端迁移与前端交互适配，在保持现有 SSE/HITL 兼容的前提下引入 `write_ops_todos` + Task Board + 审批断线兜底。

**Architecture:** 先打通后端 DeepAgents 主入口（`internal/ai/agent` + `internal/ai/todo` + `logic.NewAILogic` 接入），再接入前端 `ai.ts -> PlatformChatProvider -> replyRuntime -> AssistantReply` 的结构化 todos 渲染。协议层保持现有事件不破坏，新增 `ops_plan_updated` 作为可选增强，使用 full snapshot 覆盖策略。

**Tech Stack:** Go 1.26, CloudWeGo Eino ADK/DeepAgents, Gin/GORM, TypeScript, React, Ant Design, Vitest

---

## Scope Check

原两份计划分别覆盖后端迁移与前端设计，属于同一交付链路（后端事件/状态驱动前端渲染）。本计划将其合并为一个可执行序列，避免并行计划的契约漂移。

## File Structure (Responsibilities)

### Backend: DeepAgents runtime and todo
- Create: `internal/ai/agent/main.go`
- Create: `internal/ai/agent/prompt.go`
- Create: `internal/ai/agent/qa.go`
- Create: `internal/ai/agent/k8s.go`
- Create: `internal/ai/agent/host.go`
- Create: `internal/ai/agent/monitor.go`
- Create: `internal/ai/agent/change.go`
- Create: `internal/ai/agent/inspection.go`
- Create: `internal/ai/todo/types.go`
- Create: `internal/ai/todo/tool.go`
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/config/config.go`
- Purpose: 提供可运行的 DeepAgents 主入口、定制 TODO 工具、feature flag 控制与逻辑层接入。

### Backend: contracts and tests
- Create: `internal/ai/agent/deepagents_poc_test.go`
- Create: `internal/ai/agent/subagents_test.go`
- Create: `internal/ai/agent/main_test.go`
- Create: `internal/ai/todo/types_test.go`
- Create: `internal/ai/todo/tool_test.go`
- Modify: `internal/ai/runtime/event_types.go`
- Modify: `internal/ai/runtime/event_types_test.go`
- Modify: `internal/ai/events/events.go`
- Modify: `internal/service/ai/logic/logic_test.go`
- Modify: `internal/service/ai/logic/approval_worker_test.go`
- Purpose: 固化 nested interrupt/resume、ops todo snapshot、事件协议和审批恢复链路。

### Frontend: protocol, runtime, rendering
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/api/modules/ai.test.ts`
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/components/AI/replyRuntime.ts`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/ToolReference.tsx`
- Modify: `web/src/components/AI/replyRuntime.test.ts`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Purpose: 解析 `ops_plan_updated`，维护 runtime.todos 快照，渲染 Task Board，处理审批 submitting 超时与重内容性能。

## Implementation Rules

- 必须使用 @test-driven-development：每个任务先失败测试，再最小实现。
- 必须使用 @verification-before-completion：每任务结束执行验证命令。
- DRY/YAGNI：不引入第二套消息模型，不重写 `useXChat/Bubble.List`。
- 每个任务单独 commit，保证回滚粒度。

## Chunk 1: Backend Gate And Foundation

### Task 1: Nested interrupt/resume POC gate

**Files:**
- Create: `internal/ai/agent/deepagents_poc_test.go`
- Test: `internal/ai/agent/deepagents_poc_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestDeepAgentsPOC_InterruptResumeAcrossSubAgent(t *testing.T) {
  // assert interrupt -> waiting_approval -> resume -> done
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/ai/agent -run TestDeepAgentsPOC_InterruptResumeAcrossSubAgent -v`
Expected: FAIL

- [ ] **Step 3: Add minimal test scaffolding/fakes**

```go
// minimal deep agent test scaffold to simulate nested approval interrupt
```

- [ ] **Step 4: Run test to verify pass**

Run: `go test ./internal/ai/agent -run TestDeepAgentsPOC_InterruptResumeAcrossSubAgent -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/agent/deepagents_poc_test.go
git commit -m "test(ai): add deepagents nested interrupt resume gate"
```

### Task 2: Add `feature_flags.ai_deepagents`

**Files:**
- Modify: `internal/config/config.go`
- Create/Modify: `internal/config/config_test.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestFeatureFlags_AIDeepAgents_DefaultFalse(t *testing.T) {}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/config -run AIDeepAgents -v`
Expected: FAIL

- [ ] **Step 3: Add minimal config field/getter**

```go
AIDeepAgents *bool `mapstructure:"ai_deepagents"`
```

- [ ] **Step 4: Run test to verify pass**

Run: `go test ./internal/config -run AIDeepAgents -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add ai_deepagents feature flag"
```

### Task 3: Create `OpsTODO` model and `write_ops_todos` middleware

**Files:**
- Create: `internal/ai/todo/types.go`
- Create: `internal/ai/todo/tool.go`
- Create: `internal/ai/todo/types_test.go`
- Create: `internal/ai/todo/tool_test.go`
- Test: `internal/ai/todo/tool_test.go`

- [ ] **Step 1: Write failing tests for JSON contract and snapshot session write**

```go
func TestOpsTODO_JSONContract(t *testing.T) {}
func TestWriteOpsTodosMiddleware_StoresSnapshotInSession(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/ai/todo -v`
Expected: FAIL

- [ ] **Step 3: Implement minimal types + middleware**

```go
const SessionKeyOpsTodos = "opspilot_session_key_ops_todos"
func NewWriteOpsTodosMiddleware() (adk.AgentMiddleware, error)
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/ai/todo -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/todo/types.go internal/ai/todo/tool.go internal/ai/todo/types_test.go internal/ai/todo/tool_test.go
git commit -m "feat(ai): add ops todo model and write_ops_todos middleware"
```

## Chunk 2: Backend DeepAgents Integration

### Task 4: Implement Sub-Agent constructors

**Files:**
- Create: `internal/ai/agent/qa.go`
- Create: `internal/ai/agent/k8s.go`
- Create: `internal/ai/agent/host.go`
- Create: `internal/ai/agent/monitor.go`
- Create: `internal/ai/agent/change.go`
- Create: `internal/ai/agent/inspection.go`
- Create: `internal/ai/agent/subagents_test.go`
- Test: `internal/ai/agent/subagents_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestNewK8sAgent_Config(t *testing.T) {}
func TestNewChangeAgent_ContainsApprovalMiddleware(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/ai/agent -run "SubAgent|ChangeAgent" -v`
Expected: FAIL

- [ ] **Step 3: Implement minimal constructors**

```go
func NewK8sAgent(ctx context.Context) (adk.Agent, error)
func NewChangeAgent(ctx context.Context) (adk.Agent, error)
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/ai/agent -run "SubAgent|ChangeAgent" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/agent/qa.go internal/ai/agent/k8s.go internal/ai/agent/host.go internal/ai/agent/monitor.go internal/ai/agent/change.go internal/ai/agent/inspection.go internal/ai/agent/subagents_test.go
git commit -m "feat(ai): add deepagents subagent constructors"
```

### Task 5: Implement main `NewOpsPilotAgent`

**Files:**
- Create: `internal/ai/agent/main.go`
- Create: `internal/ai/agent/prompt.go`
- Create: `internal/ai/agent/main_test.go`
- Test: `internal/ai/agent/main_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestNewOpsPilotAgent_UsesCustomWriteOpsTodos(t *testing.T) {}
func TestNewOpsPilotAgent_RegistersSubAgents(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/ai/agent -run NewOpsPilotAgent -v`
Expected: FAIL

- [ ] **Step 3: Implement minimal main agent wiring**

```go
func NewOpsPilotAgent(ctx context.Context) (adk.ResumableAgent, error)
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/ai/agent -run NewOpsPilotAgent -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/agent/main.go internal/ai/agent/prompt.go internal/ai/agent/main_test.go
git commit -m "feat(ai): add opspilot deepagents main entry"
```

### Task 6: Wire DeepAgents into `logic.NewAILogic`

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`
- Test: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestNewAILogic_InitializesDeepAgentsRouterWhenFlagEnabled(t *testing.T) {}
func TestNewAILogic_DeepAgentsInitFailureDegradesGracefully(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/service/ai/logic -run NewAILogic -v`
Expected: FAIL

- [ ] **Step 3: Implement minimal logic initialization**

```go
if config.IsAIDeepAgentsEnabled() {
  aiRouter, err := agent.NewOpsPilotAgent(runtimectx.WithServices(context.Background(), svcCtx))
  // graceful fallback when err != nil
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/service/ai/logic -run NewAILogic -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "feat(ai): wire deepagents router in ai logic"
```

### Task 7: Add optional `ops_plan_updated` backend event contract

**Files:**
- Modify: `internal/ai/runtime/event_types.go`
- Modify: `internal/ai/runtime/event_types_test.go`
- Modify: `internal/ai/events/events.go`
- Test: `internal/ai/runtime/event_types_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestMarshalUnmarshal_OpsPlanUpdatedPayload(t *testing.T) {}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/ai/runtime -run OpsPlanUpdated -v`
Expected: FAIL

- [ ] **Step 3: Implement minimal event constant and payload type**

```go
const EventTypeOpsPlanUpdated EventType = "ops_plan_updated"
```

- [ ] **Step 4: Run runtime tests**

Run: `go test ./internal/ai/runtime -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/runtime/event_types.go internal/ai/runtime/event_types_test.go internal/ai/events/events.go
git commit -m "feat(ai): add ops_plan_updated runtime event"
```

## Chunk 3: Frontend Protocol And Runtime Integration

### Task 8: Add `ops_plan_updated` stream handler in API module

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`
- Modify: `web/src/api/modules/ai.test.ts`
- Test: `web/src/api/modules/ai.streamChunk.test.ts`

- [ ] **Step 1: Write failing test**

```ts
it('dispatches ops_plan_updated event', () => {})
```

- [ ] **Step 2: Run test to verify failure**

Run: `npm run test:run -- src/api/modules/ai.streamChunk.test.ts -t "ops_plan_updated"`
Expected: FAIL

- [ ] **Step 3: Implement minimal handler contract**

```ts
onOpsPlanUpdated?: (payload: A2UIOpsPlanUpdatedEvent) => void;
```

- [ ] **Step 4: Run tests to verify pass**

Run: `npm run test:run -- src/api/modules/ai.streamChunk.test.ts src/api/modules/ai.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/api/modules/ai.ts web/src/api/modules/ai.streamChunk.test.ts web/src/api/modules/ai.test.ts
git commit -m "feat(ai-web): support ops_plan_updated stream event"
```

### Task 9: Add runtime todo model and provider snapshot update

**Files:**
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/components/AI/replyRuntime.ts`
- Modify: `web/src/components/AI/replyRuntime.test.ts`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Test: `web/src/components/AI/replyRuntime.test.ts`

- [ ] **Step 1: Write failing tests**

```ts
it('replaces todos by full snapshot', () => {})
it('provider writes ops_plan_updated snapshot into runtime', () => {})
```

- [ ] **Step 2: Run tests to verify failure**

Run: `npm run test:run -- src/components/AI/replyRuntime.test.ts src/components/AI/__tests__/PlatformChatProvider.test.ts -t "snapshot|ops_plan_updated"`
Expected: FAIL

- [ ] **Step 3: Implement minimal runtime + provider logic**

```ts
export function applyOpsPlanUpdated(runtime, payload) {
  return { ...runtime, todos: [...(payload.todos || [])] };
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `npm run test:run -- src/components/AI/replyRuntime.test.ts src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/types.ts web/src/components/AI/replyRuntime.ts web/src/components/AI/replyRuntime.test.ts web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts
git commit -m "feat(ai-web): add runtime todo snapshot and provider integration"
```

## Chunk 4: Frontend Rendering, Approval Fallback, Perf

### Task 10: Render Task Board in `AssistantReply`

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: Write failing test**

```tsx
it('renders collapsible task board from runtime.todos', () => {})
```

- [ ] **Step 2: Run test to verify failure**

Run: `npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx -t "task board"`
Expected: FAIL

- [ ] **Step 3: Implement minimal Task Board UI**

```tsx
{runtime?.todos?.length ? <Collapse /* Task Board */ /> : null}
```

- [ ] **Step 4: Run test to verify pass**

Run: `npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx -t "task board"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "feat(ai-web): render task board in assistant reply"
```

### Task 11: Approval `submitting` timeout fallback + refresh entry

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/ToolReference.tsx`
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: Write failing test**

```ts
it('moves submitting approval to refresh-needed after 15s', () => {})
```

- [ ] **Step 2: Run test to verify failure**

Run: `npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx -t "15s|refresh-needed"`
Expected: FAIL

- [ ] **Step 3: Implement minimal timeout fallback**

```tsx
const APPROVAL_SUBMITTING_TIMEOUT_MS = 15000;
```

- [ ] **Step 4: Run tests to verify pass**

Run: `npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx src/api/modules/ai.approval.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/ToolReference.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "fix(ai-web): add approval submitting timeout fallback"
```

### Task 12: Agent label debounce + heavy content viewport mounting

**Files:**
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`

- [ ] **Step 1: Write failing tests**

```ts
it('debounces rapid agent handoff status updates', () => {})
it('externalizes oversized content and mounts by viewport', () => {})
```

- [ ] **Step 2: Run tests to verify failure**

Run: `npm run test:run -- src/components/AI/__tests__/PlatformChatProvider.test.ts src/components/AI/__tests__/AssistantReply.test.tsx -t "debounce|viewport|查看原文"`
Expected: FAIL

- [ ] **Step 3: Implement minimal debounce + threshold guards**

```ts
const AGENT_STATUS_DEBOUNCE_MS = 400;
const LARGE_CONTENT_BYTES_THRESHOLD = 64 * 1024;
const LARGE_CONTENT_LINE_THRESHOLD = 200;
```

- [ ] **Step 4: Run tests to verify pass**

Run: `npm run test:run -- src/components/AI/__tests__/PlatformChatProvider.test.ts src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/AssistantReply.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "perf(ai-web): debounce agent status and optimize heavy content rendering"
```

## Chunk 5: End-to-End Verification And Handoff

### Task 13: Backend approval compatibility regression

**Files:**
- Modify: `internal/service/ai/logic/approval_worker_test.go`
- Modify: `internal/service/ai/logic/logic_test.go`
- Test: `internal/service/ai/logic/approval_worker_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestChat_DeepAgentsApprovalReplayKeepsWaitingApproval(t *testing.T) {}
func TestResume_DeepAgentsApprovalFlowCompletes(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/service/ai/logic -run "ApprovalReplay|ApprovalFlow" -v`
Expected: FAIL

- [ ] **Step 3: Implement minimal compatibility fixes**

```go
// keep waiting_approval run_state semantics in replay/resume path
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/service/ai/logic -run "Approval" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/logic/approval_worker_test.go internal/service/ai/logic/logic_test.go
git commit -m "test(ai): verify deepagents approval replay and resume compatibility"
```

### Task 14: Final verification suite

**Files:**
- Verify only

- [ ] **Step 1: Run backend suites**

Run: `go test ./internal/ai/agent ./internal/ai/todo ./internal/ai/runtime ./internal/service/ai/logic ./internal/service/ai/handler -v`
Expected: PASS

- [ ] **Step 2: Run frontend suites**

Run: `npm run test:run -- src/api/modules/ai.streamChunk.test.ts src/api/modules/ai.test.ts src/components/AI/replyRuntime.test.ts src/components/AI/__tests__/PlatformChatProvider.test.ts src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: PASS

- [ ] **Step 3: Run lint/checks**

Run: `npm run lint && go test ./...`
Expected: PASS（或仅存在既有非本变更失败）

- [ ] **Step 4: Final commit if needed**

```bash
git add internal/ai/agent internal/ai/todo internal/ai/runtime/event_types.go internal/ai/runtime/event_types_test.go internal/ai/events/events.go internal/config/config.go internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go internal/service/ai/logic/approval_worker_test.go web/src/api/modules/ai.ts web/src/api/modules/ai.test.ts web/src/api/modules/ai.streamChunk.test.ts web/src/components/AI/types.ts web/src/components/AI/replyRuntime.ts web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/AssistantReply.tsx web/src/components/AI/ToolReference.tsx web/src/components/AI/replyRuntime.test.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "feat(ai): complete deepagents unified backend and frontend migration"
```
