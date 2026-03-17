## Context

当前项目已经具备 AI chat 的基础能力，包括 `/api/v1/ai/chat` SSE 接口、session 列表/详情 API，以及前端 `aiApi.chatStream()` 解析逻辑，但入口仍然停留在按钮跳转独立 `/ai` 路由的形态，且当前后端 chat 请求体没有显式携带 `scene`，新会话也被固定写为 `ai` 场景。与此同时，项目已经存在 `AIScenePrompt`、`AISceneConfig`、`getSceneTools()`、`getScenePrompts()` 等场景能力模型，说明系统有意把 AI 体验绑定到不同业务域，但这条链路还没有贯穿到实际聊天交互。

这次 change 同时跨越前端 UI surface、前端 chat adapter、后端 chat contract 和 session persistence，属于明显的跨模块改造。另一个关键背景是团队已经决定 markdown 渲染要充分复用 `@ant-design/x-markdown`，而不是在业务层重复造 markdown 解析与流式格式修正逻辑。

## Goals / Non-Goals

**Goals:**
- 在 `AppLayout` 内提供全局可唤起的 AI Copilot drawer，不打断当前页面工作流。
- 使用 Ant Design X 组件体系与 `useXChat` / `useXConversations` 管理聊天和会话状态，但不机械复制 demo 结构。
- 通过 `PlatformChatProvider` 对接现有 `aiApi.chatStream()` 与 session API，复用现有 SSE 协议和鉴权逻辑。
- 让 assistant 正文统一经过 `@ant-design/x-markdown` 渲染，并支持后续扩展 think/code/table 等能力。
- 为 chat 请求引入 scene 和 scene context，使工具调用与提示词增强建立在当前业务场景之上。
- 保持对现有 richer SSE 事件类型的前向兼容，但以当前后端已稳定提供的 `init/status/intent/delta/done/error` 为第一阶段基础。

**Non-Goals:**
- 不在本次 change 内实现完整的 tool approval、tool result 卡片、thought chain runtime 重构。
- 不以 `antdx.tsx` demo 为逐行实现模板；demo 仅作为交互与组件组合参考。
- 不在本次 change 内新增新的 AI agent/tool 域能力，只聚焦入口 surface、协议对齐和 scene-aware 增强。
- 不在 UI 层把隐式场景提示词伪装成用户可见输入文本。

## Decisions

### 1. 使用 drawer-based copilot 作为统一入口

系统将把 AI 助手集成到 `AppLayout` 内，以 drawer 或等价侧边浮层作为统一入口，由 `AICopilotButton` 控制开合，而不是继续依赖单独 `/ai` 页面跳转。

原因：
- 现有运维场景高度依赖页面上下文，drawer 更适合在不丢失当前工作视角的情况下发起问答。
- 当前按钮已经存在，但目标路由未形成稳定闭环，drawer 能直接与壳层布局结合。

替代方案：
- 保留独立 `/ai` 页面：实现简单，但上下文切换成本高，也不利于场景感知。
- 同时保留页面和 drawer：可行，但会增加状态同步与信息架构复杂度，本次先聚焦一个一等入口。

### 2. 采用自定义 `PlatformChatProvider` 适配现有 SSE API

前端将实现自定义 `PlatformChatProvider`，以 `AbstractChatProvider` / `useXChat` 兼容方式封装 `aiApi.chatStream()`、会话初始化、消息增量拼接和错误回退，而不是直接使用 `DeepSeekChatProvider` 等供应商 provider。

原因：
- 当前后端是自有 `/api/v1/ai/chat` 协议，不是 OpenAI/DeepSeek 原生协议。
- 现有前端 API 层已经负责 token、projectId、SSE 事件拆分和可见 delta 归一化，重复绕过这一层会破坏现有约束。

替代方案：
- 直接复用 `DeepSeekChatProvider` + `XRequest`：只适用于兼容其期望流格式的服务，不适合当前自定义事件协议。
- 完全自建 chat state，不用 `useXChat`：灵活但会放弃现成的对话管理能力和组件生态。

### 3. markdown 渲染统一由 `@ant-design/x-markdown` 承担

assistant 输出内容将统一通过 `@ant-design/x-markdown` 渲染，业务层只负责提供内容和少量业务扩展组件，不再自建 markdown 渲染管线或手工处理 HTML/换行。

原因：
- 团队已明确要求避免“自己造” markdown。
- 当前 SSE delta 已在前端做基础可见内容归一化，适合直接交给标准 markdown 组件处理。
- 统一渲染器有利于后续支持 think 标签、代码块、表格和引用样式。

替代方案：
- 继续使用 `react-markdown` 或自定义 renderers：会引入两套渲染职责，增加维护成本。
- 在消息内容里手工插入 `<br/>` 等 HTML：对流式 markdown 和代码块支持脆弱，不适合作为正式方案。

### 4. scene-aware 能力分为 scene identity 与 scene context 两层

聊天链路将显式区分当前业务场景标识和当前实体上下文。`scene` 表示域，如 `host`/`cluster`/`service`/`k8s`，`context` 表示当前页面实体，如 clusterId、hostId、serviceName、route 等。

原因：
- 单独一个 `scene` 只能粗粒度提升相关性；真正影响工具选择和回答精度的是实体级上下文。
- 分层后既能驱动 session 归类，也能控制 prompt augmentation 和 tool scoping。

替代方案：
- 只传 `scene`：实现简单，但工具选择仍然偏泛化。
- 把整段场景提示词直接拼进用户输入：会污染可见对话与会话标题，不利于历史回放和解释性。

### 5. 场景提示词和工具约束作为隐式输入，不污染用户可见消息

scene prompt、allowed/blocked tools、当前 route/entity context 应在 provider 到后端的请求链路中作为隐式输入注入，用户在聊天记录里看到的仍应是原始输入问题。

原因：
- 保持用户消息语义纯净，便于回放、标题生成和会话比较。
- 场景增强本质上是系统提供的执行上下文，而不是用户亲自输入的文字。

替代方案：
- 将增强提示拼到输入框内容中：实现方便，但对 UX、审计和后续 prompt 维护都不理想。

### 6. 后端 chat 合约需要补齐 scene 传递并按 scene 归档 session

`api/ai/v1` chat request 与 `internal/service/ai/logic` 需要支持接收 `scene` 与上下文载荷，创建/复用 session 时不能再把新会话硬编码为 `ai`。

原因：
- 如果 scene 只停留在前端，所谓 scene-aware 只是 UI 幻觉。
- 会话维度已经有 `scene` 字段，协议层缺失会让前端 session 列表和后端实际持久化脱节。

替代方案：
- 只在前端内存中区分 scene：无法支撑历史会话过滤、场景 prompt 追溯和后端 tool routing。

## Risks / Trade-offs

- [前后端协议扩展影响兼容性] → 通过为 chat request 新增可选字段保持向后兼容，老调用方仍可只传 `message/session_id`。
- [场景注入过强导致模型偏置] → 将场景提示限制为轻量约束与上下文，不把模板式长提示词无条件拼接到每轮对话。
- [Drawer 初始化失败影响主壳层] → 通过边界组件隔离 AI surface 失败，确保 App shell 可独立渲染。
- [Ant Design X demo 与项目真实需求不一致] → 只复用稳定组件与 hook 模式，状态结构、页面布局和事件映射以项目现状为准。
- [未来 richer SSE 事件扩展导致 UI 重构] → provider 和消息模型预留 intent/progress/tool 事件兼容层，但第一阶段只强依赖当前稳定事件集合。

## Migration Plan

1. 在前端壳层中引入 drawer surface 与开关状态，但先保持旧入口按钮兼容。
2. 实现 `PlatformChatProvider` 与 surface 组件，对接现有 chat/session API。
3. 扩展 chat request/logic 以接收并持久化 `scene`，再接入页面 scene resolver 和 scene prompt 获取。
4. 用 `@ant-design/x-markdown` 替换 assistant 正文渲染链。
5. 增加前后端测试覆盖后，移除对独立 `/ai` 路由的隐式依赖。

回滚策略：
- 若 drawer surface 出现问题，可暂时保留按钮但禁用 drawer 渲染，不影响主应用壳层。
- 若 scene-aware 协议扩展导致异常，可退回只发送基础 chat 请求，同时保留前端 surface。

## Open Questions

- scene context 的首版最小字段集是否统一为 `route + resource type + resource id/name`，还是要按页面单独定制？
- scene prompt 应由后端在 chat logic 中统一增强，还是由前端先获取 `getScenePrompts()` 后传递结构化选择结果？
- 当前 `/ai` 页面如果仍有历史用途，是否需要提供 drawer 与页面之间的兼容跳转或深链接能力？
