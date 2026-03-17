## Why

当前 AI 助手入口仍然偏向独立页面导航，无法在用户保持当前业务上下文的情况下快速发起对话，也没有把现有的场景能力、快捷提示词和场景工具约束真正纳入聊天链路。现在需要把 AI 助手升级为一个全局可唤起的 drawer surface，并让它基于页面场景主动提升回答与工具调用的相关性。

## What Changes

- 新增全局 AI Copilot drawer，作为 `AppLayout` 内的统一聊天入口，替代悬空或割裂的独立页面入口体验。
- 新增 `PlatformChatProvider`，把现有 `/api/v1/ai/chat` SSE 接口适配到 Ant Design X 的 `useXChat` 交互模型。
- 统一使用 `@ant-design/x-markdown` 渲染 assistant 正文，避免重复实现 markdown 渲染、流式内容格式化和扩展标签支持。
- 新增会话列表、会话切换、快捷提示词、发送区和基础状态反馈，使 UI 与现有 session API 和 SSE API 对齐。
- 新增 scene-aware chat 上下文注入能力，让当前页面场景、业务实体上下文和场景提示词成为模型与工具选择的隐式输入。
- 扩展 AI chat 协议与会话处理，使聊天请求和会话记录能够携带场景信息，而不是固定落到通用 `ai` 场景。

## Capabilities

### New Capabilities
- `ai-copilot-drawer`: 定义全局抽屉式 AI 助手界面的行为、会话体验和 markdown 呈现要求。
- `scene-aware-ai-chat`: 定义 AI 聊天如何感知当前页面场景、注入场景提示与上下文，并据此约束会话与工具调用。

### Modified Capabilities
- None.

## Impact

- Frontend: `web/src/components/AI`, `web/src/components/Layout`, `web/src/api/modules/ai.ts`, AI surface integration and tests.
- Backend: `api/ai/v1`, `internal/service/ai`, AI session/chat logic, scene-aware request handling, and related tests.
- Data/API: AI chat request payload and session persistence behavior need to carry scene/context metadata compatibly.
- Dependencies: `@ant-design/x`, `@ant-design/x-sdk`, and `@ant-design/x-markdown` become the primary chat surface foundation.
