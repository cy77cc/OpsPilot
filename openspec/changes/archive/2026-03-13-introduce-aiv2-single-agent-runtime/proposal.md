## Why

当前 AI 模块基于 `rewrite -> planner -> executor(expert agent) -> summarizer` 的多阶段控制平面实现，链路长、模型调用层级多，导致响应慢、调试复杂，也不适合作为学习 Eino ADK 和 HITL 的主路径。项目已经使用 `eino v0.8.0`，具备 `ChatModelAgent`、`Runner`、`Interrupt/ResumeWithParams`、`ChatModelAgentMiddleware` 等能力，适合引入一套更贴近官方范式的单 agent runtime。

## What Changes

- 新增 `aiv2` 单 agent runtime，使用单个 `ChatModelAgent + Runner` 驱动工具调用、流式输出和最终回答。
- 新增统一工具注册层，复用现有 host/k8s/service/delivery/observability tools，但移除 `expert agent as tool` 这一层模型调用。
- 新增基于 Eino HITL 的审批中断机制，使用 `Interrupt/ResumeWithParams` 处理 mutating tool 的 human-in-the-loop。
- 新增 `ChatModelAgentMiddleware` 链，统一处理上下文注入、审批策略、流式事件投影和可观测性。
- 保留现有 `internal/ai` 运行时作为兼容路径，通过 handler/runtime 模式开关接入 `aiv2`，不在首版直接移除旧架构。
- 保持现有前端 SSE/turn-block 交互契约可复用，优先兼容 `thinking_delta`、`tool_call`、`tool_result`、`approval_required`、`delta`、`done`。

## Capabilities

### New Capabilities
- `ai-v2-single-agent-runtime`: 基于单 `ChatModelAgent` 的新 AI 运行时，覆盖工具调用、流式输出、审批中断与恢复。
- `ai-v2-agent-middleware`: 基于 `ChatModelAgentMiddleware` 的统一横切能力链，覆盖上下文注入、审批策略、事件投影和可观测性。

### Modified Capabilities
- `ai-assistant-command-bridge`: AI handler 和 resume 接口需要支持旧 runtime 与 `aiv2` runtime 并行切换。
- `ai-streaming-events`: 流式事件契约需要兼容单 agent runtime 的 tool-call / approval / final-answer 生命周期。
- `ai-assistant-drawer`: 前端抽屉需要继续消费兼容事件，但执行过程展示从“专家执行”收敛为“工具调用链”。
- `ai-chat-session-contract`: 会话存储与回放需要支持 `aiv2` 单 agent 的历史 turn/replay 语义。

## Impact

- 后端新增 `internal/aiv2` 模块，可能包含 runtime、agent builder、approval、tool registry、stream projector、checkpoint bridge 等子模块。
- 现有 `internal/service/ai` handler 需要增加 runtime 分流，支持 `ai`/`aiv2` 双运行时。
- 现有 tools 需要被统一注册到单 agent runtime，并为 mutating tools 补齐审批 policy metadata。
- Redis execution state / Eino checkpoint store / AI session storage 需要协调，用于 approval interrupt/resume 和历史回放。
- 前端主要复用现有聊天 UI 和 SSE 渲染逻辑，但执行态文案和工具链展示会调整。
