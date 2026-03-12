## Why

当前 AI 运维链路把审批点放在 `executor` 内部，导致 mutating 步骤会先进入执行态、再中断等待审批，恢复后还会出现 `summary` 丢失、审批后执行语义不稳定、以及 UI 上“等待确认”和“执行中”交织的体验问题。与此同时，前端聊天体验还有几个配套问题没有收口：历史对话中的 thought chain 展开策略不合理、执行链路细节不够完整、重新生成会重复插入用户消息、以及流程展示噪音过大。审批本质上是执行前门控，不应和专家执行状态机耦合在一起；聊天 UI 也需要围绕这个生命周期一起重构。

## What Changes

- 将 AI 审批门控从 `executor` 内部上移到 `planner` 和 `executor` 之间，形成 `plan -> approval gate -> execute -> summary` 的控制流。
- 将待审批状态从“步骤已进入执行器后暂停”改为“计划已生成但尚未开始执行”，并保留基于 `session_id + plan_id + step_id` 的恢复语义。
- 规定批准后的恢复流必须继续同一 assistant `turn_id`，并在执行完成后继续进入 `summary`，而不是在专家执行结束后直接 `done`。
- 调整 turn/block 与兼容 SSE 事件语义，使审批块表现为执行前门控、批准后退出等待态，并让执行与总结状态明确分层。
- 明确前端审批交互为短生命周期操作块：等待审批时显示 CTA，提交成功后退出等待态并由执行/总结内容接管。
- 调整 thought chain 展示策略：历史恢复消息默认折叠，当前流式对话只展开活跃阶段，并补充 step/tool/result 级别的执行细节。
- 重新定义“重新生成”交互：保留原用户消息，不再通过再次插入同样的用户问题来触发新的回答。
- 收紧链路文案与展示层级，让 thought chain、执行卡片和最终答案各自承担清晰职责，减少重复和内部噪音。

## Capabilities

### New Capabilities
- `ai-pre-execution-approval-gate`: AI 控制平面中的执行前审批门控、恢复进入执行与总结、以及对应状态持久化与事件语义。

### Modified Capabilities
- `aiops-control-plane`: 审批门控从 executor 内部暂停改为 executor 之前的正式控制平面阶段。
- `ai-streaming-events`: 审批、恢复、执行和总结的 SSE/turn-block 生命周期要求需要调整，恢复后必须继续 summary。
- `aiops-card-event-stream`: 审批块和执行块的事件投影顺序需要改为“审批前置、执行后总结”。
- `ai-assistant-drawer`: 审批 UI 必须表现为执行前确认，而不是执行中断点，并在批准后退出等待态。

## Impact

- Backend AI runtime: `internal/ai/orchestrator.go`, `internal/ai/executor`, `internal/ai/runtime`, `internal/ai/projector.go`
- AI transport and API semantics: `/api/v1/ai/chat`, `/api/v1/ai/resume/step`, `/api/v1/ai/resume/step/stream`
- Frontend AI drawer and turn/block reducer: `web/src/components/AI/*`, `web/src/api/modules/ai.ts`
- Frontend conversation restore and regenerate behavior: `web/src/components/AI/hooks/useConversationRestore.ts`, `web/src/components/AI/components/MessageActions.tsx`
- Tests covering approval, resume, execute, and summary continuation across backend and frontend
