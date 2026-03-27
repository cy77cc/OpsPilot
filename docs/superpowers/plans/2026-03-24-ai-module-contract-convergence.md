# AI Module Contract Convergence Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 收敛 AI 模块前后端契约，消除前端调用后端未实现接口、补齐审批/续流关键链路，并以自动化测试和 CI 门禁防止回归。

**Architecture:** 以后端 `RegisterAIHandlers` 注册路由和 `api/ai/v1` 为唯一契约源，前端 `aiApi` 改为受白名单驱动。流式协议沿用 `id/event/data`，前端统一处理 `AI_STREAM_CURSOR_EXPIRED` 与审批恢复状态。通过“契约一致性测试 + SSE 协议测试 + e2e 关键路径”形成闭环。

**Tech Stack:** Go (Gin/GORM), TypeScript (Vite/Vitest/React), SSE, Playwright e2e, ripgrep, npm, go test

---

## Scope Check

本 spec 涉及的子系统（路由契约、前端 API 收口、SSE/续流、审批链路、CI 校验）属于同一条 AI 会话链路，不拆分为独立计划。执行顺序按“先止血，再收口，再加固”进行。

## File Structure (Responsibilities)

### Backend contract and handlers
- Modify: `internal/service/ai/routes.go`
- Modify: `internal/service/ai/handler/session_test.go`
- Create: `internal/service/ai/handler/routes_contract_test.go`
- Purpose: 固化后端可用 AI 路由白名单，并通过测试快照保证注册契约稳定。

### Frontend API contract surface
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/api/modules/ai.test.ts`
- Create: `web/src/api/modules/ai.contract.test.ts`
- Purpose: 让前端 API 仅暴露后端存在接口；对未实现接口显式报错而非静默 404。

### Prompt/data source integration in UI
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
- Purpose: 修复 `getScenePrompts` 断链（改用已实现源或降级策略），不再触发不存在路由。

### Stream resume and runtime error handling
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`
- Purpose: 打通 `lastEventId` 透传，统一处理 `AI_STREAM_CURSOR_EXPIRED`。

### Approval chain stabilization
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/approval_worker.go`
- Modify: `internal/service/ai/logic/tool_error_classifier.go`
- Modify: `internal/service/ai/logic/logic_test.go`
- Modify: `internal/service/ai/logic/tool_error_classifier_test.go`
- Purpose: 保证审批中断（含流式 Recv error 路径）进入 `waiting_approval` 而非提前结束。

### Contract consistency gate and docs
- Create: `script/ai/check_contract_parity.sh`
- Modify: `.github/workflows/*` (选择已有前后端测试 workflow 注入校验)
- Modify: `docs/superpowers/specs/2026-03-24-ai-module-contract-convergence-design.md` (如实现偏差需回写)
- Purpose: 在 CI 阶段自动比对“前端 API 映射 vs 后端注册路由”。

## Implementation Rules

- 严格使用 @test-driven-development：先写失败测试，再最小实现，再回归。
- 严格使用 @verification-before-completion：每个任务结束必须执行对应验证命令。
- 严格 DRY/YAGNI：不引入 spec 外新接口，不做无关重构。
- 每个任务完成后单独 commit，提交信息语义化且可回滚。

## Chunk 1: Backend Contract Freeze

### Task 1: 固化 AI 路由白名单测试

**Files:**
- Create: `internal/service/ai/handler/routes_contract_test.go`
- Modify: `internal/service/ai/handler/session_test.go`
- Test: `internal/service/ai/handler/routes_contract_test.go`

- [ ] **Step 1: 写失败测试，定义后端白名单路由集合**
```go
func TestRegisterAIHandlers_RouteContract(t *testing.T) {
  // assert exact supported routes only
}
```

- [ ] **Step 2: 运行测试确认失败**
Run: `go test ./internal/service/ai/handler -run RouteContract -v`
Expected: FAIL（路由断言未满足或测试文件未实现）

- [ ] **Step 3: 最小实现/调整路由注册与断言**
```go
expected := []string{
  "POST /api/v1/ai/chat",
  "GET /api/v1/ai/sessions",
  // ...
}
```

- [ ] **Step 4: 重新运行测试确认通过**
Run: `go test ./internal/service/ai/handler -run RouteContract -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/service/ai/handler/routes_contract_test.go internal/service/ai/handler/session_test.go
git commit -m "test: freeze ai backend route contract"
```

### Task 2: 审批中断链路（后端）防提前结束回归

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/approval_worker.go`
- Modify: `internal/service/ai/logic/tool_error_classifier.go`
- Modify: `internal/service/ai/logic/logic_test.go`
- Modify: `internal/service/ai/logic/tool_error_classifier_test.go`
- Test: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: 写失败测试覆盖“Recv() interrupt error -> waiting_approval”**
```go
func TestChatPausesWaitingApprovalOnStreamingInterruptRecvError(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**
Run: `go test ./internal/service/ai/logic -run StreamingInterrupt -v`
Expected: FAIL（run 状态为 completed/failed 而非 waiting_approval）

- [ ] **Step 3: 最小实现，支持从 error 恢复 interrupt event**
```go
func recoverableInterruptEventFromErr(err error, agentName string) (*adk.AgentEvent, bool)
```

- [ ] **Step 4: 在 `logic.go`/`approval_worker.go` 的 stream recv error 分支接入 interrupt 恢复**
```go
if interruptEvent, ok := recoverableInterruptEventFromErr(err, event.AgentName); ok {
  // project and keep waiting approval
}
```

- [ ] **Step 5: 运行逻辑层回归测试**
Run: `go test ./internal/service/ai/logic -v`
Expected: PASS

- [ ] **Step 6: 运行 handler + logic 组合回归**
Run: `go test ./internal/service/ai/handler ./internal/service/ai/logic`
Expected: `ok` for both packages

- [ ] **Step 7: Commit**
```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/approval_worker.go internal/service/ai/logic/tool_error_classifier.go internal/service/ai/logic/logic_test.go internal/service/ai/logic/tool_error_classifier_test.go
git commit -m "fix: keep approval runs waiting on streaming interrupt"
```

## Chunk 2: Frontend API Contract Convergence

### Task 3: 前端 API 收口（白名单 + 显式未实现错误）

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/api/modules/ai.test.ts`
- Create: `web/src/api/modules/ai.contract.test.ts`
- Test: `web/src/api/modules/ai.contract.test.ts`

- [ ] **Step 1: 写失败测试，断言允许的 API 集合与后端一致**
```ts
it('exposes only backend-supported ai endpoints', () => {
  expect(contract).toEqual([...])
})
```

- [ ] **Step 2: 运行测试确认失败**
Run: `npm run test:run -- src/api/modules/ai.contract.test.ts`
Expected: FAIL（存在漂移接口）

- [ ] **Step 3: 最小实现，重构 `aiApi` 导出为“支持接口 + disabled 接口”**
```ts
function notImplementedByBackend(endpoint: string): never {
  throw new Error(`NotImplementedByBackend: ${endpoint}`)
}
```

- [ ] **Step 4: 运行 API 模块测试**
Run: `npm run test:run -- src/api/modules/ai.test.ts src/api/modules/ai.contract.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add web/src/api/modules/ai.ts web/src/api/modules/ai.test.ts web/src/api/modules/ai.contract.test.ts
git commit -m "refactor: converge ai frontend api to backend contract"
```

### Task 4: 修复 `scene prompts` 断链

**Files:**
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
- Test: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`

- [ ] **Step 1: 写失败测试，断言不再依赖未实现的 `getScenePrompts` 路由**
```ts
it('loads prompt items without calling unavailable scene prompts endpoint', async () => {})
```

- [ ] **Step 2: 运行测试确认失败**
Run: `npm run test:run -- src/components/AI/__tests__/CopilotSurface.test.tsx`
Expected: FAIL（仍调用旧接口）

- [ ] **Step 3: 最小实现，切换到已实现数据源/静态降级策略**
```ts
const promptsResp = { data: { prompts: [] } } // when backend route unavailable
```

- [ ] **Step 4: 运行组件测试**
Run: `npm run test:run -- src/components/AI/__tests__/CopilotSurface.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add web/src/components/AI/CopilotSurface.tsx web/src/components/AI/__tests__/CopilotSurface.test.tsx
git commit -m "fix: remove ai scene prompts hard dependency"
```

## Chunk 3: Stream Resume and Cursor Error Handling

### Task 5: 打通 `lastEventId` 透传

**Files:**
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`

- [ ] **Step 1: 写失败测试，断言 `PlatformChatProvider` 将 `lastEventId` 透传至 `aiApi.chatStream`**
```ts
expect(aiApi.chatStream).toHaveBeenCalledWith(expect.objectContaining({ lastEventId: 'evt-1' }), expect.anything(), expect.anything())
```

- [ ] **Step 2: 运行测试确认失败**
Run: `npm run test:run -- src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: FAIL（参数未透传）

- [ ] **Step 3: 最小实现，补齐 request 参数透传**
```ts
lastEventId: params.lastEventId,
```

- [ ] **Step 4: 运行测试确认通过**
Run: `npm run test:run -- src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/types.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts
git commit -m "feat: wire ai stream cursor resume in provider"
```

### Task 6: 统一处理 `AI_STREAM_CURSOR_EXPIRED`

**Files:**
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Test: `web/src/api/modules/ai.streamChunk.test.ts`

- [ ] **Step 1: 写失败测试，模拟 `event:error` 且 `code=AI_STREAM_CURSOR_EXPIRED` 的恢复行为**
```ts
it('handles AI_STREAM_CURSOR_EXPIRED with deterministic recovery state', async () => {})
```

- [ ] **Step 2: 运行测试确认失败**
Run: `npm run test:run -- src/api/modules/ai.streamChunk.test.ts src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: FAIL

- [ ] **Step 3: 最小实现，封装 cursor expired 分支为可恢复状态更新**
```ts
if (payload.code === 'AI_STREAM_CURSOR_EXPIRED') { /* mark recoverable + refresh-needed */ }
```

- [ ] **Step 4: 运行测试确认通过**
Run: `npm run test:run -- src/api/modules/ai.streamChunk.test.ts src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add web/src/api/modules/ai.streamChunk.test.ts web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts
git commit -m "fix: normalize cursor expired recovery in ai stream"
```

## Chunk 4: CI Gate and Final Verification

### Task 7: 增加契约一致性脚本与 CI 门禁

**Files:**
- Create: `script/ai/check_contract_parity.sh`
- Modify: `.github/workflows/*`（选择当前运行 go+web 测试的工作流）
- Test: `script/ai/check_contract_parity.sh`

- [ ] **Step 1: 写失败脚本测试输入（fixture）**
```bash
# expected mismatch exits 1
```

- [ ] **Step 2: 运行脚本确认失败路径存在**
Run: `bash script/ai/check_contract_parity.sh`
Expected: exit code 1 when mismatch exists

- [ ] **Step 3: 最小实现脚本（提取后端路由 + 前端 API endpoint 映射并比较）**
```bash
backend_routes=$(...)
frontend_routes=$(...)
```

- [ ] **Step 4: 在 CI workflow 注入脚本执行**
Run: `bash script/ai/check_contract_parity.sh`
Expected: `Contract parity check passed`

- [ ] **Step 5: Commit**
```bash
git add script/ai/check_contract_parity.sh .github/workflows
git commit -m "ci: enforce ai contract parity check"
```

### Task 8: 全量验证与文档回写

**Files:**
- Modify: `docs/superpowers/specs/2026-03-24-ai-module-contract-convergence-design.md`（仅当实现偏差）
- Modify: `docs/superpowers/plans/2026-03-24-ai-module-contract-convergence.md`（勾选完成后可选）

- [ ] **Step 1: 运行后端测试套件**
Run: `go test ./internal/service/ai/handler ./internal/service/ai/logic`
Expected: all `ok`

- [ ] **Step 2: 运行前端 AI 关键测试套件**
Run: `npm run test:run -- src/api/modules/ai.test.ts src/api/modules/ai.streamChunk.test.ts src/components/AI/__tests__/PlatformChatProvider.test.ts src/components/AI/__tests__/AssistantReply.test.tsx src/components/AI/__tests__/CopilotSurface.test.tsx`
Expected: all PASS

- [ ] **Step 3: 运行契约一致性脚本**
Run: `bash script/ai/check_contract_parity.sh`
Expected: PASS

- [ ] **Step 4: 回写 spec 偏差（若有）并记录风险**
```md
## Implementation Notes
- ...
```

- [ ] **Step 5: 最终 Commit**
```bash
git add docs/superpowers/specs/2026-03-24-ai-module-contract-convergence-design.md docs/superpowers/plans/2026-03-24-ai-module-contract-convergence.md
git commit -m "docs: finalize ai contract convergence implementation notes"
```

## Plan Review Loop (Operational Guidance)

- 本计划按 Chunk 执行，每个 Chunk 完成后做一次计划/实现一致性 review。
- 当前环境如无法使用 plan-document-reviewer 子代理，则执行本地替代审阅：
  1. 对照 spec 验证本 Chunk 任务完成度
  2. 对照测试命令验证可重复性
  3. 对照文件责任检查是否越界改动
- 若出现连续 5 次审阅不通过，暂停并请求人工裁决。

## Execution Order

1. Chunk 1（后端契约冻结 + 审批链路稳定）
2. Chunk 2（前端 API 收口 + 场景提示断链修复）
3. Chunk 3（续流与游标错误处理）
4. Chunk 4（CI 门禁 + 最终验证）

## Done Criteria

- 前端不再调用后端未实现 AI 接口。
- 审批触发后 run 稳定进入 `waiting_approval`，不会提前结束。
- `lastEventId` 透传与 `AI_STREAM_CURSOR_EXPIRED` 恢复链路验证通过。
- CI 契约一致性检查可阻断漂移回归。
