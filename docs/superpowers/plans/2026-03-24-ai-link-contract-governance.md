# AI Link Contract Governance Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 6-8 周内完成 AI 链路级契约收口、错误分层、安全幂等和性能治理，确保审批中断/恢复、断线续传和历史回放一致。

**Architecture:** 以 Contract-First 为主线，先修复协议断链（SSE event id、字段对齐、错误语义），再强化审批状态机与前端防重入，最后做会话摘要化和懒加载性能治理，并通过 CI 契约门禁固化。

**Tech Stack:** Go (Gin/GORM), TypeScript (React/Ant Design X), SSE, Vitest, Go test

---

## Chunk 1: Contract And Stream Reliability

### Task 1: 实时 SSE `event_id` 透传与断线续传闭环

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/handler/chat.go`
- Modify: `internal/service/ai/handler/chat_test.go`
- Test: `web/src/api/modules/ai.streamChunk.test.ts`

- [ ] **Step 1: 写后端失败测试（实时流必须包含 SSE id）**

```go
func TestChatHandler_EmitsLiveEventIDs(t *testing.T) {
    // 首次 chat，断言 body 至少包含一个 "id: " 行
}
```

- [ ] **Step 2: 运行后端测试确认失败**

Run: `go test ./internal/service/ai/handler -run EmitsLiveEventIDs -v`
Expected: FAIL，提示未输出 `id:`。

- [ ] **Step 3: 最小实现（写入 event_id + 透传 Last-Event-ID 头）**

```go
// emit 时把 appendRunEvent 返回的 event id 注入 payload，再走 SSEWriter
// reconnect 时优先读取 c.GetHeader("Last-Event-ID")，并与 query/body last_event_id 统一归并后下传 logic
```

- [ ] **Step 4: 新增重连头部测试并确认通过**

Run: `go test ./internal/service/ai/handler -run 'EmitsLiveEventIDs|ReplaysEventsAfterLastEventID|LastEventIDHeader' -v`
Expected: PASS。

- [ ] **Step 5: 更新前端流解析测试并验证**

Run: `npm run test:run -- web/src/api/modules/ai.streamChunk.test.ts`
Expected: PASS，确保事件 id 回调持续可用。

- [ ] **Step 6: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/handler/chat.go internal/service/ai/handler/chat_test.go web/src/api/modules/ai.streamChunk.test.ts
git commit -m "fix: emit stable live sse event ids for ai chat"
```

### Task 2: REST 字段契约对齐（run.report 与 session/message 时间字段）

**Files:**
- Modify: `internal/service/ai/handler/run.go`
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Test: `web/src/api/modules/ai.test.ts`

- [ ] **Step 1: 写前端契约失败测试（兼容 `report.id` / `report.report_id`）**

```ts
it('normalizes run report id from backend payload', () => {
  // 输入 report.id，断言上层可读 report_id
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `npm run test:run -- web/src/api/modules/ai.test.ts`
Expected: FAIL。

- [ ] **Step 3: 实现字段兼容归一化**

```ts
const normalizedReportId = run.report?.report_id ?? run.report?.id;
```

- [ ] **Step 4: 修正会话类型定义，显式支持 `created_at/updated_at` 与历史兼容字段**

```ts
interface AISession { created_at?: string; updated_at?: string; createdAt?: string; updatedAt?: string; }
```

- [ ] **Step 5: 回归测试**

Run: `npm run test:run -- web/src/api/modules/ai.test.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/service/ai/handler/run.go web/src/api/modules/ai.ts web/src/components/AI/CopilotSurface.tsx web/src/api/modules/ai.test.ts
git commit -m "fix: align ai run and session response contract fields"
```

## Chunk 2: Error Taxonomy And Approval Safety

### Task 3: 审批提交错误语义分层 + 后端幂等（not found/forbidden/conflict/server）

**Files:**
- Create: `internal/service/ai/logic/errors.go`
- Modify: `internal/service/ai/logic/approval_write_model.go`
- Modify: `internal/service/ai/handler/approval.go`
- Modify: `internal/service/ai/handler/approval_test.go`
- Modify: `web/src/api/modules/ai.ts`

- [ ] **Step 1: 写后端失败测试（不同场景返回不同业务码）**

```go
func TestSubmitApproval_ErrorTaxonomy(t *testing.T) {
  // 不存在 -> NotFound
  // 非本人 -> Forbidden
  // 已处理 -> Conflict
  // 相同 Idempotency-Key 重试 -> 返回首次结果，不重复执行状态迁移
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/service/ai/handler -run SubmitApproval_ErrorTaxonomy -v`
Expected: FAIL。

- [ ] **Step 3: 定义 typed error + Idempotency-Key 记录结构**

```go
type ApprovalNotFoundError struct{}
type ApprovalForbiddenError struct{}
type ApprovalConflictError struct{}
// approval submit idempotency record: key + approval_id + payload_hash + result_snapshot
```

- [ ] **Step 4: handler 增加 `Idempotency-Key` 读取并按错误类型映射业务码**

```go
idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
switch {
case errors.As(err, &notFound): httpx.NotFound(...)
case errors.As(err, &forbidden): httpx.Fail(...Forbidden...)
case errors.As(err, &conflict): httpx.Fail(...Conflict...)
default: httpx.ServerErr(...)
}
```

- [ ] **Step 5: 前端提交接口补充 `Idempotency-Key` header（UUID）**

```ts
await apiService.post(`/ai/approvals/${id}/submit`, payload, { headers: { 'Idempotency-Key': key } });
```

- [ ] **Step 6: 跑测试确认通过**

Run: `go test ./internal/service/ai/handler -run SubmitApproval -v`
Expected: PASS。

- [ ] **Step 7: Commit**

```bash
git add internal/service/ai/logic/errors.go internal/service/ai/logic/approval_write_model.go internal/service/ai/handler/approval.go internal/service/ai/handler/approval_test.go web/src/api/modules/ai.ts
git commit -m "feat: add idempotent approval submit with explicit error taxonomy"
```

### Task 4: 前端审批防重入与失败可见化

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/contexts/NotificationContext.tsx`
- Modify: `web/src/components/Notification/NotificationItem.tsx`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: 写失败测试（双击审批只发一次）**

```ts
it('submits approval once when user clicks confirm repeatedly', async () => {
  // 连续点击，断言 submitApproval 调用次数为 1
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: FAIL。

- [ ] **Step 3: 实现 per-approval in-flight 锁与失败态**

```ts
if (inflight[approvalId]) return;
setState({ state: 'refresh-needed', message: err.message });
// 每次提交生成 UUID 并传递给 aiApi.submitApproval 的 idempotencyKey
```

- [ ] **Step 4: 给通知面板审批按钮增加 loading/disabled**

```tsx
<Button loading={isSubmitting} disabled={isSubmitting} ... />
```

- [ ] **Step 5: 回归测试**

Run: `npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx web/src/__tests__/Notification/NotificationPanel.test.tsx`
Expected: PASS。

- [ ] **Step 6: Commit**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/contexts/NotificationContext.tsx web/src/components/Notification/NotificationItem.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "fix: enforce single-flight approval actions in ai ui"
```

## Chunk 3: Performance And Data Shape

### Task 5: 会话列表接口去 N+1（摘要化）

**Files:**
- Modify: `internal/dao/ai/chat_dao.go`
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/handler/session.go`
- Modify: `internal/service/ai/handler/session_test.go`
- Modify: `web/src/components/AI/CopilotSurface.tsx`

- [ ] **Step 1: 写后端失败测试（ListSessions 不携带完整 messages）**

```go
func TestListSessions_ReturnsSummaryOnly(t *testing.T) {
  // 断言列表响应无 messages 大字段，或仅最近一条摘要
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/service/ai/handler -run ListSessions_ReturnsSummaryOnly -v`
Expected: FAIL。

- [ ] **Step 3: DAO 增加 summary 查询接口（会话 + 最近消息摘要）**

```go
func (d *AIChatDAO) ListSessionSummaries(...) ([]SessionSummaryRow, error)
```

- [ ] **Step 4: logic/handler 改为返回摘要模型**

```go
{id, title, scene, last_message, updated_at}
```

- [ ] **Step 5: 前端适配摘要数据结构**

Run: `npm run test:run -- web/src/components/AI/historyProjection.test.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/dao/ai/chat_dao.go internal/service/ai/logic/logic.go internal/service/ai/handler/session.go internal/service/ai/handler/session_test.go web/src/components/AI/CopilotSurface.tsx
git commit -m "perf: remove ai sessions n+1 by returning session summaries"
```

### Task 6: 大字段惰性化与分页契约

**Files:**
- Modify: `internal/service/ai/handler/projection.go`
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/components/AI/historyProjection.ts`
- Test: `web/src/components/AI/historyProjection.test.ts`

- [ ] **Step 1: 写失败测试（超大 run projection 分页加载）**

```ts
it('loads historical blocks incrementally via cursor', async () => {
  // 首屏只加载第一页块
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `npm run test:run -- web/src/components/AI/historyProjection.test.ts`
Expected: FAIL。

- [ ] **Step 3: 后端为 projection/content 增加稳定 cursor/limit 参数**

```go
GET /ai/runs/:runId/projection?cursor=&limit=
// cursor = base64("<created_at_unix_nano>_<block_id>")
// 排序: created_at ASC, block_id ASC 作为 deterministic tie-breaker
```

- [ ] **Step 4: 前端懒加载器按稳定 cursor 增量拉取**

```ts
while (hasMore) { await aiApi.getRunProjection(runId, { cursor, limit: 20 }); }
```

- [ ] **Step 5: 回归测试**

Run: `go test ./internal/service/ai/handler -run Projection -v && npm run test:run -- web/src/components/AI/historyProjection.test.ts`
Expected: PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/service/ai/handler/projection.go internal/service/ai/logic/logic.go web/src/api/modules/ai.ts web/src/components/AI/historyProjection.ts web/src/components/AI/historyProjection.test.ts
git commit -m "feat: add cursor pagination for ai projection and content hydration"
```

## Chunk 4: Governance, Compatibility, And Release Gate

### Task 7: unknown event 可观测降级 + 兼容开关

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`

- [ ] **Step 1: 写失败测试（未知事件不驱动主状态但会上报）**

```ts
it('observes unknown stream events without mutating runtime state', async () => {
  // onUnknownEvent 被调用，runtime 无异常迁移
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: FAIL。

- [ ] **Step 3: 在流解析器增加 default 分支与观测回调（含 run/user/tenant 标签）**

```ts
handlers.onUnknownEvent?.({ eventType, payload, eventId, runId, userId, tenantId })
```

- [ ] **Step 4: 回归测试**

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/api/modules/ai.streamChunk.test.ts`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/src/api/modules/ai.ts web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts
git commit -m "feat: observe unknown ai stream events with safe degradation"
```

### Task 8: 契约门禁与发布准入

**Files:**
- Create: `docs/superpowers/contracts/ai-contract-v1.md`
- Create: `docs/superpowers/runbooks/ai-release-gate.md`
- Modify: `.github/workflows/ci.yml` (或项目现有 CI 配置文件)
- Test: `internal/service/ai/handler/routes_contract_test.go`

- [ ] **Step 1: 写合同文档与兼容矩阵**

```md
Event | Version | Backward Compatible | Notes
```

- [ ] **Step 2: 增加 contract diff 校验脚本/命令**

Run: `go test ./internal/service/ai/handler -run RouteContract -v`
Expected: PASS。

- [ ] **Step 3: CI 接入门禁与失败示例**

```yaml
- name: AI Contract Check
  run: make ai-contract-check
```

- [ ] **Step 4: 端到端准入清单写入 runbook**

```md
发布前必须验证: 审批拒绝、审批超时、断线续传、历史回放
```

- [ ] **Step 5: Commit**

```bash
git add docs/superpowers/contracts/ai-contract-v1.md docs/superpowers/runbooks/ai-release-gate.md .github/workflows/ci.yml internal/service/ai/handler/routes_contract_test.go
git commit -m "chore: add ai contract gate and release runbook"
```

---

## 执行顺序建议

1. 先执行 Chunk 1-2（稳定性和安全优先）。
2. 再执行 Chunk 3（性能与容量）。
3. 最后执行 Chunk 4（治理固化与发布门禁）。

## 全量验证命令（每个 Chunk 完成后）

```bash
go test ./internal/service/ai/... ./internal/dao/ai/... -count=1
npm run test:run -- web/src/api/modules/ai*.test.ts web/src/components/AI/**/*.test.tsx web/src/components/AI/**/*.test.ts
```

## 完成标准

1. 关键链路测试全部通过。
2. 合同与实现无偏差（CI 门禁通过）。
3. 审批中断/恢复与断线续传在回归演练中稳定。
4. 会话列表与历史回放性能指标达标。
