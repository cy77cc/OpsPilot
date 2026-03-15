# Agent Frontend Display Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 重构 AI 审批中断链路并收敛旧前端展示实现，使审批恢复、参数编辑、Planner 展示和 ThoughtChain 运行时渲染基于同一套真实协议工作。

**Architecture:** 先统一后端审批运行时协议，从 `ApprovalInterruptInfo` 到 `PendingApproval`、SSE approval payload、`ResumeRequest` 保持字段闭环，再调整前端以 runtime approval node 作为唯一审批交互入口，逐步降级旧 `message.confirmation` / legacy thought chain 分支。Planner 与 ThoughtChain 展示只做基于现有 runtime 事件的收敛，不新增第二套展示协议。

**Tech Stack:** Go 1.26, CloudWeGo Eino ADK, React 19, TypeScript, Ant Design 6, @ant-design/x, Vitest

---

## File Map

### Runtime approval protocol

- Modify: `internal/ai/runtime/approval.go`
  - 扩展 `ApprovalInterruptInfo`，让中断事件携带原始参数 JSON 与恢复所需元信息。
- Modify: `internal/ai/runtime/runtime.go`
  - 扩展 `ResumeRequest` 与 `PendingApproval`，定义统一的审批恢复载荷。
- Modify: `internal/ai/tools/approval/gate.go`
  - 保存原始参数和编辑后参数，恢复执行时优先使用编辑结果。
- Modify: `internal/ai/orchestrator.go`
  - 从 ADK interrupt 构建完整 `PendingApproval`，并把完整 approval payload 发给前端。
- Modify: `internal/ai/orchestrator_test.go`
  - 覆盖审批暂停、恢复、编辑参数透传、SSE payload 完整性。

### Approval HTTP bridge

- Modify: `internal/service/ai/tooling_handlers.go`
  - 让审批决策接口接受 `edited_arguments` 并透传到 runtime resume。
- Modify: `internal/model/ai_approval.go`
  - 仅在确认确有持久化需求时扩字段；默认复用现有 `ParamsJSON`，避免无意义 schema 漂移。
- Test: `internal/service/ai/...`
  - 如已有 handler/unit test 入口，补充审批接口透传测试；若无现成测试文件，则新增最小覆盖文件。

### Frontend approval runtime path

- Modify: `web/src/components/AI/types.ts`
  - 扩展 runtime approval node / `ConfirmationRequest` 协议，支持参数编辑与恢复身份。
- Modify: `web/src/components/AI/thoughtChainRuntime.ts`
  - 从 SSE approval payload 恢复完整 approval state，避免前端自行拼接身份字段。
- Modify: `web/src/components/AI/components/ConfirmationPanel.tsx`
  - 从“确认框”升级为“审批编辑器”，支持 JSON 编辑、校验、提交失败重试。
- Modify: `web/src/components/AI/components/RuntimeThoughtChain.tsx`
  - 以 runtime node approval payload 为准渲染审批，不再依赖缩水 `details` 猜字段。
- Modify: `web/src/components/AI/Copilot.tsx`
  - 审批提交统一走 runtime approval payload，减少 `message.confirmation` 兼容分支。
- Modify: `web/src/api/modules/ai.ts`
  - 扩展审批决策接口签名，支持 `edited_arguments`。
- Test: `web/src/components/AI/thoughtChainRuntime.test.ts`
  - 验证 runtime approval payload 到前端 state 的映射。
- Test: `web/src/components/AI/Copilot.test.tsx`
  - 验证审批确认、编辑参数、失败重试路径。
- Test: `web/src/api/modules/ai.test.ts`
  - 验证审批决策请求体包含 `edited_arguments`。

### Planner and runtime display cleanup

- Modify: `web/src/components/AI/components/RuntimeThoughtChain.tsx`
  - 收敛 Planner JSON 显示策略，使用 runtime structured data 展示，不直接泄露原始 JSON 片段。
- Modify: `web/src/components/AI/Copilot.tsx`
  - 明确 runtime nodes 为主路径，legacy `thoughtChain` 仅用于历史回放兼容。
- Test: `web/src/components/AI/Copilot.test.tsx`
  - 补充 runtime approval / planner rendering 不回退到旧展示的测试。

---

## Conflicts To Resolve Explicitly

### Must clean up old implementations

- 当前前端把 `approval:<stepId>` 当成审批节点 ID 同时又把它当作审批决策目标，这只是 UI node key，不是稳定恢复身份。实现时必须改为“后端下发完整 approval payload，前端只透传，不再自行拼 target”。
- 当前 Orchestrator 发给前端的 approval payload 丢掉了大部分运行时字段，只保留 `plan_id`、`step_id`、`tool_name`。实现时必须清理这套缩水 payload。
- 当前 `message.confirmation` 与 `message.runtime.nodes[].approval` 同时存在。实现时 runtime node 是主路径，`message.confirmation` 仅保留兼容读路径，停止新增逻辑分支。
- 当前 legacy `thoughtChain` 仍参与部分审批/阶段状态更新。实现时审批只绑定 runtime nodes，legacy thought chain 不再承担 live runtime 职责。
- 当前计划里假设 Planner 美化和追加式 runtime 更新是“新功能”，但仓库已有基础实现。实现时应基于现有 runtime reducer 收敛，不再引入第二套 parser/state machine。

### Non-goals

- 不新增复杂 schema-driven 表单编辑器；本次只支持 JSON 文本编辑与语法校验。
- 不重构整个 chat replay 存储格式；仅调整 live runtime 展示主路径与必要的 replay 投影兼容。
- 不引入新的审批中心业务流；本次只修复 AI runtime approval 的真实链路。

---

## Chunk 1: Rebuild Approval Runtime Protocol

### Task 1: Extend runtime structs around the real approval identity

**Files:**
- Modify: `internal/ai/runtime/approval.go`
- Modify: `internal/ai/runtime/runtime.go`
- Test: `internal/ai/orchestrator_test.go`

- [ ] **Step 1: Write the failing test for approval payload completeness**

在 [internal/ai/orchestrator_test.go](/root/project/k8s-manage/internal/ai/orchestrator_test.go) 新增一个测试，断言审批中断事件被转成的 `PendingApproval` / SSE approval payload 至少包含：

```go
func TestOrchestratorApprovalInterruptIncludesArgumentsAndResumeIdentity(t *testing.T) {
	// assert pending approval keeps plan_id, step_id, checkpoint_id, tool_name, arguments_json
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/ai/... -run TestOrchestratorApprovalInterruptIncludesArgumentsAndResumeIdentity -v`
Expected: FAIL，因为当前 payload 没有 `arguments_json` / 完整恢复身份字段。

- [ ] **Step 3: Extend `ApprovalInterruptInfo` with runtime-complete fields**

在 [approval.go](/root/project/k8s-manage/internal/ai/runtime/approval.go) 中扩展：

```go
type ApprovalInterruptInfo struct {
	PlanID          string         `json:"plan_id,omitempty"`
	StepID          string         `json:"step_id,omitempty"`
	CheckpointID    string         `json:"checkpoint_id,omitempty"`
	Target          string         `json:"target,omitempty"`
	ToolName        string         `json:"tool_name,omitempty"`
	ToolDisplayName string         `json:"tool_display_name,omitempty"`
	Mode            string         `json:"mode,omitempty"`
	RiskLevel       string         `json:"risk_level,omitempty"`
	Summary         string         `json:"summary,omitempty"`
	Params          map[string]any `json:"params,omitempty"`
	ArgumentsInJSON string         `json:"arguments_json,omitempty"`
	Environment     string         `json:"environment,omitempty"`
	Namespace       string         `json:"namespace,omitempty"`
}
```

- [ ] **Step 4: Extend runtime resume and pending approval structs**

在 [runtime.go](/root/project/k8s-manage/internal/ai/runtime/runtime.go) 中扩展：

```go
type ResumeRequest struct {
	SessionID       string `json:"session_id,omitempty"`
	PlanID          string `json:"plan_id,omitempty"`
	StepID          string `json:"step_id,omitempty"`
	Target          string `json:"target,omitempty"`
	CheckpointID    string `json:"checkpoint_id,omitempty"`
	Approved        bool   `json:"approved"`
	Reason          string `json:"reason,omitempty"`
	EditedArguments string `json:"edited_arguments,omitempty"`
}

type PendingApproval struct {
	ID              string         `json:"id,omitempty"`
	PlanID          string         `json:"plan_id,omitempty"`
	StepID          string         `json:"step_id,omitempty"`
	CheckpointID    string         `json:"checkpoint_id,omitempty"`
	Target          string         `json:"target,omitempty"`
	Status          string         `json:"status,omitempty"`
	Title           string         `json:"title,omitempty"`
	Mode            string         `json:"mode,omitempty"`
	Risk            string         `json:"risk,omitempty"`
	Summary         string         `json:"summary,omitempty"`
	ApprovalKey     string         `json:"approval_key,omitempty"`
	ToolName        string         `json:"tool_name,omitempty"`
	ToolDisplayName string         `json:"tool_display_name,omitempty"`
	Params          map[string]any `json:"params,omitempty"`
	ArgumentsInJSON string         `json:"arguments_json,omitempty"`
	CreatedAt       time.Time      `json:"created_at,omitempty"`
	ExpiresAt       time.Time      `json:"expires_at,omitempty"`
}
```

- [ ] **Step 5: Run focused runtime tests**

Run: `go test ./internal/ai/... -run 'TestOrchestratorApprovalInterruptIncludesArgumentsAndResumeIdentity|TestOrchestratorApprovalPauseAndResumeEmitLifecycleCallbacks' -v`
Expected: PASS

- [ ] **Step 6: Commit the struct protocol baseline**

```bash
git add internal/ai/runtime/approval.go internal/ai/runtime/runtime.go internal/ai/orchestrator_test.go
git commit -m "refactor(ai): define complete approval runtime protocol"
```

### Task 2: Preserve edited arguments through gate resume

**Files:**
- Modify: `internal/ai/tools/approval/gate.go`
- Test: `internal/ai/orchestrator_test.go`

- [ ] **Step 1: Write the failing test for edited arguments resume**

新增测试覆盖“审批通过后使用编辑后的 JSON 参数调用工具，而不是原始参数”：

```go
func TestApprovalGateUsesEditedArgumentsOnResume(t *testing.T) {
	// assert edited_arguments overrides arguments_json when approved
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/ai/... -run TestApprovalGateUsesEditedArgumentsOnResume -v`
Expected: FAIL，因为当前 `executeResumed` 只会使用 `ArgumentsInJSON`。

- [ ] **Step 3: Extend interrupt state and resume handling**

在 [gate.go](/root/project/k8s-manage/internal/ai/tools/approval/gate.go) 中实现：

```go
type interruptState struct {
	ArgumentsInJSON string `json:"arguments_json"`
	EditedArguments string `json:"edited_arguments,omitempty"`
	Approved        bool   `json:"approved,omitempty"`
	Reason          string `json:"reason,omitempty"`
}
```

并在 `handleResume` / `executeResumed` 中保存、优先使用 `EditedArguments`。

- [ ] **Step 4: Run focused runtime tests**

Run: `go test ./internal/ai/... -run 'TestApprovalGateUsesEditedArgumentsOnResume|TestOrchestratorApprovalPauseAndResumeEmitLifecycleCallbacks' -v`
Expected: PASS

- [ ] **Step 5: Commit gate resume behavior**

```bash
git add internal/ai/tools/approval/gate.go internal/ai/orchestrator_test.go
git commit -m "feat(ai): support edited approval arguments on resume"
```

### Task 3: Make orchestrator emit one canonical approval payload

**Files:**
- Modify: `internal/ai/orchestrator.go`
- Test: `internal/ai/orchestrator_test.go`

- [ ] **Step 1: Write the failing test for canonical approval SSE payload**

新增测试，断言 `chain_node_open` 的 `approval` payload 至少包含：

```go
{
	"id": "...",
	"request_id": "...",
	"plan_id": "plan-2",
	"step_id": "step-approval",
	"checkpoint_id": "checkpoint-2",
	"tool_name": "host_delete",
	"tool_display_name": "删除主机",
	"risk": "high",
	"summary": "删除主机前需要审批",
	"arguments_json": "{\"id\":7}",
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/ai/... -run TestOrchestratorApprovalSSEPayloadIsCanonical -v`
Expected: FAIL，因为当前 `approval.details` 只带少量字段。

- [ ] **Step 3: Update interrupt extraction and pending approval mapping**

在 [orchestrator.go](/root/project/k8s-manage/internal/ai/orchestrator.go) 中同步修改：

- `interruptApprovalInfo`
- `pendingApprovalFromInterrupt`
- approval `ChainNodeInfo` 的 `Approval` payload 构造

要求：

- `map[string]any` 兼容路径也要读取 `checkpoint_id`、`target`、`arguments_json`
- SSE approval payload 直接平铺 canonical 字段，不再只塞进 `details`
- `details` 可以保留，但只能作为展示补充，不能再承担恢复身份职责

- [ ] **Step 4: Update resume params to carry `edited_arguments`**

在 `resume` 方法里构造 ADK `ResumeParams` 时加入：

```go
if strings.TrimSpace(req.EditedArguments) != "" {
	resumeData["edited_arguments"] = strings.TrimSpace(req.EditedArguments)
}
```

- [ ] **Step 5: Run runtime tests**

Run: `go test ./internal/ai/... -v`
Expected: PASS

- [ ] **Step 6: Commit orchestrator protocol cleanup**

```bash
git add internal/ai/orchestrator.go internal/ai/orchestrator_test.go
git commit -m "refactor(ai): emit canonical approval runtime payload"
```

---

## Chunk 2: Bridge Approval HTTP Decisions To The Runtime Protocol

### Task 4: Extend approval decision API to forward edited arguments

**Files:**
- Modify: `internal/service/ai/tooling_handlers.go`
- Modify: `internal/service/ai/handler.go`
- Test: `internal/service/ai/tooling_handlers_test.go`

- [ ] **Step 1: Create or extend a handler test for approval decision payload**

如果测试文件不存在则新建 [tooling_handlers_test.go](/root/project/k8s-manage/internal/service/ai/tooling_handlers_test.go)，编写测试：

```go
func TestDecideChainApprovalForwardsEditedArgumentsToResumeRequest(t *testing.T) {
	// assert HTTP request edited_arguments reaches coreai.ResumeRequest
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/ai/... -run TestDecideChainApprovalForwardsEditedArgumentsToResumeRequest -v`
Expected: FAIL，因为当前请求体和 `approvalResumeRequest` 都不支持 `edited_arguments`。

- [ ] **Step 3: Extend the request struct and resume adapter**

在 [tooling_handlers.go](/root/project/k8s-manage/internal/service/ai/tooling_handlers.go) 中：

- 扩展 `chainApprovalDecisionRequest`
- 扩展 `approvalResumeRequest`
- 在流式和非流式审批通过路径都传 `EditedArguments`

最小目标：

```go
type chainApprovalDecisionRequest struct {
	Approved        bool   `json:"approved"`
	Reason          string `json:"reason,omitempty"`
	EditedArguments string `json:"edited_arguments,omitempty"`
}
```

- [ ] **Step 4: Decide whether database persistence needs change**

检查 [ai_approval.go](/root/project/k8s-manage/internal/model/ai_approval.go) 与审批中心读取逻辑：

- 若仅需在恢复执行时透传编辑参数，则不改表结构
- 若产品要求审计编辑后的参数，再单独追加 migration 与字段

本次默认：**不新增 migration**，避免扩大范围。

- [ ] **Step 5: Run handler and runtime tests**

Run: `go test ./internal/service/ai/... ./internal/ai/... -v`
Expected: PASS

- [ ] **Step 6: Commit the HTTP bridge update**

```bash
git add internal/service/ai/tooling_handlers.go internal/service/ai/handler.go internal/service/ai/tooling_handlers_test.go
git commit -m "feat(ai): forward edited approval arguments through HTTP bridge"
```

---

## Chunk 3: Make Frontend Approval Use Runtime Payload As The Single Source Of Truth

### Task 5: Extend frontend approval types to match runtime protocol

**Files:**
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/components/AI/thoughtChainRuntime.ts`
- Test: `web/src/components/AI/thoughtChainRuntime.test.ts`

- [ ] **Step 1: Write the failing runtime mapper test**

在 [thoughtChainRuntime.test.ts](/root/project/k8s-manage/web/src/components/AI/thoughtChainRuntime.test.ts) 新增：

```ts
it('maps canonical approval payload into runtime approval state', () => {
  // expect argumentsJson, checkpointId, planId, stepId, toolName to be preserved
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && npm run test -- thoughtChainRuntime.test.ts`
Expected: FAIL，因为当前 `approvalFromPayload` 只映射标题、描述、风险和 details。

- [ ] **Step 3: Extend `ConfirmationRequest` and runtime approval types**

在 [types.ts](/root/project/k8s-manage/web/src/components/AI/types.ts) 中为 runtime approval 增加必要字段：

```ts
export interface ConfirmationRequest {
  id: string;
  title: string;
  description: string;
  risk: RiskLevel;
  status?: 'waiting_user' | 'submitting' | 'failed';
  errorMessage?: string;
  details?: Record<string, unknown>;
  toolName?: string;
  toolDisplayName?: string;
  planId?: string;
  stepId?: string;
  checkpointId?: string;
  target?: string;
  argumentsJson?: string;
  editable?: boolean;
  onConfirm: (editedArgs?: string) => void;
  onCancel: (reason?: string) => void;
}
```

- [ ] **Step 4: Update runtime mapper**

在 [thoughtChainRuntime.ts](/root/project/k8s-manage/web/src/components/AI/thoughtChainRuntime.ts) 中：

- `approvalFromPayload` 直接读取 canonical approval payload
- 不再依赖 `details` 猜测 `plan_id` / `step_id`
- 保留 `details` 仅用于展示附加信息

- [ ] **Step 5: Run focused frontend tests**

Run: `cd web && npm run test -- thoughtChainRuntime.test.ts`
Expected: PASS

- [ ] **Step 6: Commit type alignment**

```bash
git add web/src/components/AI/types.ts web/src/components/AI/thoughtChainRuntime.ts web/src/components/AI/thoughtChainRuntime.test.ts
git commit -m "refactor(web): align approval runtime types with backend payload"
```

### Task 6: Upgrade `ConfirmationPanel` from confirm box to approval editor

**Files:**
- Modify: `web/src/components/AI/components/ConfirmationPanel.tsx`
- Test: `web/src/components/AI/Copilot.test.tsx`

- [ ] **Step 1: Write the failing UI test for edited approval submission**

在 [Copilot.test.tsx](/root/project/k8s-manage/web/src/components/AI/Copilot.test.tsx) 中新增用例：

```ts
it('submits edited approval arguments from runtime approval panel', async () => {
  // render approval node, edit JSON, click confirm, expect edited_arguments in API call
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && npm run test -- Copilot.test.tsx`
Expected: FAIL，因为当前面板没有 JSON 编辑器，也不会把编辑后的参数传给 `onConfirm`。

- [ ] **Step 3: Implement minimal approval editor behavior**

在 [ConfirmationPanel.tsx](/root/project/k8s-manage/web/src/components/AI/components/ConfirmationPanel.tsx) 中：

- 使用本地 state 保存编辑中的 JSON
- 只做 JSON 语法校验
- `onConfirm` 传 `editedArgs?: string`
- 保持现有 submitting / failed / retry 视觉状态

限制：

- 不新增 schema 表单
- 不做格式化 diff
- 不做业务字段级校验

- [ ] **Step 4: Run focused UI test**

Run: `cd web && npm run test -- Copilot.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit approval editor UI**

```bash
git add web/src/components/AI/components/ConfirmationPanel.tsx web/src/components/AI/Copilot.test.tsx
git commit -m "feat(web): allow editing approval arguments before resume"
```

### Task 7: Remove frontend dependence on synthetic approval node identity

**Files:**
- Modify: `web/src/components/AI/Copilot.tsx`
- Modify: `web/src/api/modules/ai.ts`
- Test: `web/src/api/modules/ai.test.ts`
- Test: `web/src/components/AI/Copilot.test.tsx`

- [ ] **Step 1: Write the failing API test for `edited_arguments`**

在 [ai.test.ts](/root/project/k8s-manage/web/src/api/modules/ai.test.ts) 中新增：

```ts
it('posts edited_arguments when approving a chain node', async () => {
  await aiApi.decideChainApproval('plan-1', 'approval:step-1', true, 'looks safe', '{"id":7,"force":true}');
  // expect request body contains edited_arguments
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && npm run test -- ai.test.ts`
Expected: FAIL，因为当前接口签名不接受 `edited_arguments`。

- [ ] **Step 3: Extend API module signatures**

在 [ai.ts](/root/project/k8s-manage/web/src/api/modules/ai.ts) 中扩展：

```ts
decideChainApproval(
  chainId: string,
  nodeId: string,
  approved: boolean,
  reason?: string,
  editedArguments?: string,
)
```

流式接口同样扩展。

- [ ] **Step 4: Refactor `Copilot.handleApprovalDecision`**

在 [Copilot.tsx](/root/project/k8s-manage/web/src/components/AI/Copilot.tsx) 中收敛逻辑：

- `handleApprovalDecision` 接受 `approvalPayload` 与 `editedArgs`
- `chainId`、`stepId`、`checkpointId`、`target` 均直接取 runtime approval payload
- 不再依赖 `payload.id || payload.step_id || assistantId` 猜主身份
- `approval:<stepId>` 仅保留为 UI node id，不再承担恢复语义

- [ ] **Step 5: Degrade legacy `message.confirmation` usage**

调整 `Copilot` 中审批相关分支：

- live runtime 审批状态以 `message.runtime.nodes[].approval` 为准
- `message.confirmation` 仅保留兼容渲染或过渡状态
- 不再给新逻辑追加新的 `message.confirmation` 专有字段

- [ ] **Step 6: Run frontend test suite for AI surface**

Run: `cd web && npm run test -- ai.test.ts Copilot.test.tsx thoughtChainRuntime.test.ts`
Expected: PASS

- [ ] **Step 7: Commit frontend approval path cleanup**

```bash
git add web/src/api/modules/ai.ts web/src/api/modules/ai.test.ts web/src/components/AI/Copilot.tsx web/src/components/AI/Copilot.test.tsx web/src/components/AI/thoughtChainRuntime.test.ts
git commit -m "refactor(web): make runtime approval payload the single source of truth"
```

---

## Chunk 4: Finish Planner And ThoughtChain Display Cleanup On Top Of Runtime Nodes

### Task 8: Consolidate planner display around runtime structured content

**Files:**
- Modify: `web/src/components/AI/components/RuntimeThoughtChain.tsx`
- Modify: `web/src/components/AI/Copilot.tsx`
- Test: `web/src/components/AI/Copilot.test.tsx`

- [ ] **Step 1: Write the failing test for planner JSON filtering**

新增测试，断言 Planner 原始 JSON 不直接显示，但结构化步骤仍可见：

```ts
it('renders planner steps without leaking raw planner json fragments', () => {
  // expect human-readable steps; expect raw {"steps":...} fragments hidden
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd web && npm run test -- Copilot.test.tsx`
Expected: FAIL，如果当前仍存在 raw fragment 外漏或结构化数据未正确显示。

- [ ] **Step 3: Keep runtime node rendering additive and runtime-first**

在 [RuntimeThoughtChain.tsx](/root/project/k8s-manage/web/src/components/AI/components/RuntimeThoughtChain.tsx) 中：

- 保留现有追加式 runtime node 渲染
- 收敛 JSON fragment 过滤规则
- 优先展示 `structured.steps` / `details`，其次才展示文本 body

在 [Copilot.tsx](/root/project/k8s-manage/web/src/components/AI/Copilot.tsx) 中：

- live streaming 路径只使用 runtime nodes
- legacy `thoughtChain` 仅用于 restore / compatibility fallback

- [ ] **Step 4: Run relevant frontend tests**

Run: `cd web && npm run test -- Copilot.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit display cleanup**

```bash
git add web/src/components/AI/components/RuntimeThoughtChain.tsx web/src/components/AI/Copilot.tsx web/src/components/AI/Copilot.test.tsx
git commit -m "refactor(web): consolidate planner and runtime chain display"
```

---

## Final Verification

- [ ] **Step 1: Run backend tests**

Run: `go test ./internal/ai/... ./internal/service/ai/... -v`
Expected: PASS

- [ ] **Step 2: Run frontend targeted tests**

Run: `cd web && npm run test -- ai.test.ts Copilot.test.tsx thoughtChainRuntime.test.ts`
Expected: PASS

- [ ] **Step 3: Run frontend typecheck**

Run: `cd web && npm run typecheck`
Expected: PASS

- [ ] **Step 4: Run focused production build checks if needed**

Run: `cd web && npm run build`
Expected: PASS

- [ ] **Step 5: Review old-implementation cleanup conditions**

人工检查以下条件是否成立：

- runtime approval payload 已成为 live approval 的唯一身份来源
- `approval:<stepId>` 只作为 UI node key，不再作为恢复协议主键
- `message.confirmation` 不再承担 live runtime 主状态
- legacy `thoughtChain` 不再参与新的审批交互逻辑
- approval edited arguments 可从前端传到 runtime gate 并覆盖原始参数

- [ ] **Step 6: Final commit**

```bash
git add internal/ai/runtime/approval.go internal/ai/runtime/runtime.go internal/ai/tools/approval/gate.go internal/ai/orchestrator.go internal/ai/orchestrator_test.go internal/service/ai/tooling_handlers.go internal/service/ai/tooling_handlers_test.go web/src/api/modules/ai.ts web/src/api/modules/ai.test.ts web/src/components/AI/types.ts web/src/components/AI/thoughtChainRuntime.ts web/src/components/AI/thoughtChainRuntime.test.ts web/src/components/AI/components/ConfirmationPanel.tsx web/src/components/AI/components/RuntimeThoughtChain.tsx web/src/components/AI/Copilot.tsx web/src/components/AI/Copilot.test.tsx
git commit -m "refactor(ai): rebuild approval runtime display and resume flow"
```

## Notes For The Implementer

- 先做审批协议，再做前端编辑器。顺序反过来会把错误状态固化到 UI 层。
- 对已有测试要优先做“最小修正”，不要因为引入新协议而重写整套 runtime event model。
- 如果在实现中发现审批数据库记录必须持久化编辑后的参数，先停下来单独补 migration 计划，不要把它悄悄塞进本次任务。
- 若某个旧分支确认无人使用，可以在实现任务中删除；若只是不确定，则先降级为兼容路径并加 TODO，不要一次删太多。
