# 审批模型端到端审计记录

日期：2026-03-22

## 背景

本次审计目标是完整核查 AI 审批链路与工具调用链路，重点验证两个线上现象：

1. 风险命令未触发可处理审批，导致“等待审批但无人可批”。
2. tool 调用失败会否导致整会话失败。

## 结论摘要

当前不是“前端完全没做”，而是前后端链路存在断点，且有一个高优先级后端事件消费缺陷：

1. `host_exec_readonly` 被策略拦截时仅返回 `status=suspended`，未进入统一审批中断模型（无 `tool_approval`、无可审批 `approval_id`）。
2. `approval_requested` outbox 事件被当前 `ApprovalWorker` 直接 claim+done（因 worker 只处理 `approval_decided`），导致审批请求可能被吞。
3. AI 聊天前端只展示 `tool_approval` 状态，不提供审批提交动作；审批动作目前主要在通知中心。
4. tool 调用失败链路已具备降级能力，但错误分类此前对文案格式较敏感，需保持鲁棒匹配。

## 现状模型（后端）

### 1. 审批中间件模型（预期主链路）

当工具调用命中审批：

1. Middleware 评估 `RequiresApproval=true`
2. 触发 `StatefulInterrupt`，产生 `tool_approval` SSE
3. 写入 `AIApprovalTask(status=pending)` 与 outbox `approval_requested`
4. 用户调用 `POST /ai/approvals/:id/submit`
5. 写入 outbox `approval_decided`
6. `ApprovalWorker` 消费 `approval_decided` 并恢复执行

关键代码：

- `internal/ai/tools/middleware/approval.go`
- `internal/ai/tools/common/approval_orchestrator.go`
- `internal/service/ai/logic/logic.go` (`SubmitApproval`)
- `internal/service/ai/logic/approval_worker.go`

### 2. 路由模型

当前是 submit-only：

- 保留：`POST /ai/approvals/:id/submit`
- 未注册：`POST /ai/approvals/:id/resume`

即恢复主要依赖后台 worker，而非前端直连 resume 接口。

## 现状模型（前端）

1. SSE 事件消费支持 `tool_approval`，会更新 runtime。
2. AI 聊天组件展示审批状态，但无“批准/拒绝”按钮。
3. 审批 API (`submitApproval`, `listPendingApprovals`) 已存在，但在 AI 聊天主路径几乎未消费。
4. 通知中心可调用 `confirmApproval`（内部转 `submitApproval`）。

关键代码：

- `web/src/components/AI/providers/PlatformChatProvider.ts`
- `web/src/components/AI/replyRuntime.ts`
- `web/src/components/AI/AssistantReply.tsx`
- `web/src/contexts/NotificationContext.tsx`
- `web/src/api/modules/ai.ts`

## 关键问题清单（按严重度）

### P0-1: `approval_requested` 事件被吞

现象：

- Worker claim 到 outbox 后只处理 `approval_decided`。
- 对其他事件（如 `approval_requested`）直接返回 nil，随后 `MarkDone`。
- 结果：审批请求可能未对外分发即被标记完成。

影响：

- 用户无审批请求可见，流程卡在“等待审批”。

### P0-2: `host_exec_readonly` 未走统一审批模型

现象：

- `host_exec_readonly` 策略引擎命中审批时返回：
  - `approval_required=true`
  - `status=suspended`
- 但不触发中断、不创建审批任务。

影响：

- 模型可描述“等待审批”，但系统无真实审批实体可处理。

### P0-3: AI 聊天无审批操作入口

现象：

- 聊天区能显示 `tool_approval` 活动，但无提交审批交互。

影响：

- 用户必须跳到通知中心（且通知还依赖 `approval_requested` 分发正常）才能推进。

### P1-1: 审批人模型过窄

现状：

- 审批任务绑定 `task.UserID = 请求用户`。
- `SubmitApproval` 强校验同一 user。

影响：

- 无法支持“审批人/审批组”与代审批流程。

### P1-2: 路由与代码认知不一致

现状：

- `ResumeApproval` 逻辑代码仍在，但路由已禁用 resume。

影响：

- 容易在故障排查中误判链路。

## 与现场现象映射

用户观察到：

- 三个预检查命令均显示 `approval_required=true`、`status=suspended`
- 但没有收到可处理审批请求

映射：

1. 命令经 `host_exec_readonly` 被策略阻断 -> 仅 suspended 返回（P0-2）。
2. 即便部分路径写入 `approval_requested`，也可能被 worker 吞（P0-1）。
3. 聊天前端本身无审批提交入口（P0-3）。

## 已完成修复（本轮）

### 修复 A：审批兜底策略增强

当 `host_batch_exec_preview` 命令分类失败（`commandClass == ""`）时，保守要求审批，避免“静默放行 + 工具内再 suspended”的混乱行为。

文件：

- `internal/ai/tools/common/approval_orchestrator.go`
- `internal/ai/tools/common/approval_orchestrator_test.go`

### 修复 B：tool 失败分类鲁棒性增强

扩展可恢复调用失败识别，支持更宽松 `tool_name/call_id` 文案提取，降低错误文案漂移导致 fatal 的概率。

文件：

- `internal/service/ai/logic/tool_error_classifier.go`
- `internal/service/ai/logic/tool_error_classifier_test.go`

### 回归结果

已通过：

```bash
go test ./internal/service/ai/... ./internal/ai/tools/... -count=1
```

## 待办修复建议（优先级）

1. P0：拆分 outbox 消费职责，`approval_requested` 不得由 resume worker 吞掉。
2. P0：将 `host_exec_readonly` 的“需审批”路径统一接入 `StatefulInterrupt + AIApprovalTask`。
3. P0：在 AI 聊天区增加审批操作（批准/拒绝）并直连 `submitApproval`。
4. P1：引入审批人/审批组模型，放宽 `task.UserID` 单人绑定。
5. P1：清理或明确标注 `ResumeApproval` 旧路径，避免维护歧义。

## 关键文件索引

后端：

- `internal/ai/tools/middleware/approval.go`
- `internal/ai/tools/common/approval_orchestrator.go`
- `internal/service/ai/logic/logic.go`
- `internal/service/ai/logic/approval_worker.go`
- `internal/dao/ai/approval_dao.go`
- `internal/dao/ai/approval_outbox_dao.go`
- `internal/ai/tools/host/tools.go`
- `internal/ai/tools/host/policy_engine.go`

前端：

- `web/src/components/AI/providers/PlatformChatProvider.ts`
- `web/src/components/AI/replyRuntime.ts`
- `web/src/components/AI/AssistantReply.tsx`
- `web/src/contexts/NotificationContext.tsx`
- `web/src/api/modules/ai.ts`

