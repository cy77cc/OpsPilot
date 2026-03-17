## Why

当前 AI Copilot 已经能流式展示最终回答，但对于 `planner -> executor -> replanner` 这类复杂 agent 流程，用户仍然只能看到零散状态文案或最终结果，无法理解执行链路。现有后端事件日志已经提供了 `run_path`、`tool_call` 和 `replanner.response` 等足够的结构化信号，现在需要把这些信号转成前端可读的思考过程展示，而不是继续把所有中间输出混在普通 markdown 回复里。

## What Changes

- 为 AI Copilot 引入基于 `run_path` 的 ThoughtChain 过程展示，用于表现复杂 agent 的 planning/execution/replanning 流程。
- 将 `executor` 下的 `tool_call` 归入思考过程中的子节点，作为用户可见的执行动作链。
- 将 `planner` 与 `replanner` 的解释性输出解析为思考过程外的用户可读流式正文，而不是 ThoughtChain 子项。
- 当 `replanner` 产生 `response` 时，将之前的 planning/replanning 过程摘要折叠，仅继续向主消息区流式展示最终 response。
- 以最终 `replanner` 的完成信号作为该思维链的结束条件，并让整个思考过程卡片从 loading 收束到 completed。
- 保持普通 `QAAgent` 的轻量展示方式，不强制对所有消息渲染 ThoughtChain。

## Capabilities

### New Capabilities
- `ai-copilot-thought-chain`: 定义 AI Copilot 如何将 `run_path`、`tool_call` 和 `replanner.response` 映射为可折叠的思考过程与最终回答。

### Modified Capabilities
- `ai-copilot-drawer`: 调整抽屉内 assistant 消息的流式展示语义，使复杂 agent 的思考过程、过渡性摘要和最终回答分层呈现。

## Impact

- Frontend: `web/src/components/AI`, `web/src/api/modules/ai.ts`, `web/src/components/AI/providers/PlatformChatProvider.ts`, Copilot 消息渲染与状态归并逻辑。
- Backend: `internal/service/ai/logic` 需要把现有 agent 运行事件稳定转换为前端可消费的流式状态与 thought-chain 数据。
- Data/API: AI chat SSE 需要补充或归一运行路径、工具调用、planner/replanner 输出和结束信号的消费约定。
- UX: 复杂 agent 显示为默认折叠的 `Deep thinking` 卡片，普通 QA 消息继续保持正文优先的轻量体验。
