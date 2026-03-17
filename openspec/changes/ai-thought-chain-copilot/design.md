## Context

当前 Copilot 前端主要围绕 `init / intent / status / delta / done / error` 这组基础 SSE 事件工作，assistant 正文直接流式进入 markdown 区。与此同时，后端在 [event_log.go](/root/project/k8s-manage/internal/service/ai/logic/event_log.go) 记录了更完整的 agent 事件日志，其中复杂 agent 已经具备：

- `run_path`：表明当前事件所属的 agent 调用层级
- `tool_calls`：表明 executor 触发的具体工具动作
- `planner` / `replanner` 文本输出：表明阶段性计划或结论
- `replanner.response` 与结束动作：表明最终收束内容和循环结束时机

本次 change 不追求像历史方案那样重建一套后端“原生 ThoughtChain 事件协议”，而是优先利用现有运行语义，为 Copilot 提供一套稳定的前端过程展示模型。目标是让复杂 agent 在 UI 上呈现为：

- 一个使用 Ant Design X `Think` 组件承载、默认折叠的 `Deep thinking` 卡片
- `Think` 内部使用 Ant Design X `ThoughtChain` 组件，仅展示 executor 的工具动作链
- 卡片外的正文，展示 planner/replanner 解析后的用户可读总结
- 当 `replanner.response` 到达后，正文切换为最终回答，之前的过程摘要自动折叠

## Goals / Non-Goals

**Goals:**
- 让复杂 agent 的过程展示建立在 `run_path` 归并和 `tool_call` 子节点基础上，而不是仅靠最终 markdown 文本。
- 明确要求使用 Ant Design X 官方 `Think` 和 `ThoughtChain` 组件承载思考过程，而不是再实现一套自定义容器。
- 让 `planner`/`replanner` 输出以用户可读的流式摘要形式展示在思考过程外，避免和工具链条混杂。
- 在 `replanner.response` 出现时，将过程性摘要收束，让正文进入最终回答模式。
- 用最终 `replanner` 完成信号作为 ThoughtChain 结束条件，形成一致的 loading/completed 生命周期。
- 让 `QAAgent` 等简单场景继续保持轻量展示，不被复杂过程 UI 干扰。

**Non-Goals:**
- 不在本次 change 内重构整个 AI 运行时协议为全新的原生 chain 事件族。
- 不要求把 planner、replanner、tool result 全部逐条可视化成独立 ThoughtChain 节点。
- 不在本次 change 内重做 Copilot 抽屉整体视觉语言或会话结构。
- 不在本次 change 内解决所有历史会话重放兼容问题，只要求新流式链路具备稳定展示语义。

## Decisions

### 1. `run_path` 作为 ThoughtChain 分层的主依据

前端 ThoughtChain 的归属关系将以 `run_path` 为准，而不是靠消息文本猜测阶段。对于复杂 agent：

- `OpsPilotAgent -> DiagnosisAgent -> planner` 视为 planning 阶段
- `... -> executor` 视为执行阶段容器
- `... -> replanner` 视为重规划阶段
- `executor` 下的 `tool_call` 事件作为该 executor 节点的下一层 ThoughtChain item

原因：
- `run_path` 已经表达了真实的 agent 调用层级，稳定性高于文本关键词。
- 这样可以避免为每种 agent 架构单独写前端 heuristics。

替代方案：
- 仅根据 `agent_name` 和正文内容猜阶段：实现简单，但多轮 `executor -> replanner -> executor` 时容易归属错误。
- 后端先转换为全新链协议：更理想，但改造面更大，不适合作为当前提案的最小可行路径。

### 2. ThoughtChain 只展示 executor 的动作链

UI 中必须直接使用 Ant Design X `Think` 作为思考过程容器，内部直接使用 Ant Design X `ThoughtChain` 渲染执行链，而不是新建平行的自定义卡片/时间线组件。`Think/ThoughtChain` 将聚焦执行动作，而不是承载所有中间语义。具体规则：

- `executor` 事件提供执行阶段容器
- `tool_call` 作为 ThoughtChain item
- 对应的工具结果、状态和摘要并入同一 tool item
- planner/replanner 的自然语言输出不进入 ThoughtChain，而在卡片外展示

原因：
- 用户更容易把 ThoughtChain 理解为“做了什么动作”，而不是“模型说了什么话”。
- planner/replanner 文本更适合作为对用户的解释，不适合塞进动作链里。
- 直接采用 `Think`/`ThoughtChain` 可以复用 Ant Design X 的交互语义、默认折叠行为和 AI 组件体系，而不是让 Copilot 再维护一套近似但不兼容的过程 UI。

替代方案：
- 把 planner/replanner 也都放进 ThoughtChain：会让链条里混入长文本，破坏动作感。
- 把所有 delta 都暴露为节点：会造成严重碎片化和视觉抖动。
- 自定义实现“思考卡片 + 时间线”：自由度更高，但会偏离官方 AI 组件语义，增加后续维护和样式一致性成本。

### 3. 正文分为过程摘要和最终回答两段状态

assistant 正文需要从单一字符串升级为两个逻辑区域：

- `transientSummary`：承载 planner/replanner 的过程性流式说明
- `finalResponse`：承载 `replanner.response` 触发后的最终回答

当 `replanner.response` 首次出现时：

- `transientSummary` 默认收起
- `finalResponse` 成为主消息区继续流式更新
- ThoughtChain 仍保留，但默认折叠在 `Deep thinking` 内

原因：
- 过程说明和最终结论是两类不同信息，混在一个 markdown 缓冲区里会不断重排。
- `replanner.response` 是现有日志里最可靠的“最终话术切换点”。

替代方案：
- 始终把 planner/replanner 输出直接 append 到正文：最终消息会充满中间过程噪音。
- 收到 `response` 后直接替换整个正文：虽然简洁，但会丢失过程摘要的可回看能力。

### 4. 使用 `replanner` 的完成信号作为链结束条件

对于 plan-execute-replan 结构的 agent，ThoughtChain 的结束不以任意 assistant 文本停止为准，而以最终 `replanner` 的完成信号为准。UI 生命周期是：

- 初始为 `Deep thinking (loading)`
- 出现 planning/execution/replanning 信号时持续更新
- 检测到 `replanner.response` 后进入 finalizing
- 检测到最终 `replanner done` 后整体切为 completed

原因：
- 这比单纯依赖顶层 `done` 更接近 agent 结构本身的收束点。
- 可以避免在 `response` 已出现但循环尚未结束时过早结束卡片状态。

替代方案：
- 仅使用顶层 `done`：简单，但难以区分复杂 agent 内部是否已真正收束。
- 以最后一个 `executor` 完成为结束：会漏掉最终 `replanner.response` 的收尾阶段。

### 5. 普通 QA 保持降级路径

如果对话只出现简单的 `transfer_to_agent -> QAAgent -> final answer` 流程，则不渲染复杂 ThoughtChain，只展示轻量状态和正文输出。

原因：
- 复杂过程 UI 只在复杂 agent 上有价值。
- 这样可以减少普通问答的视觉噪音和实现复杂度。

替代方案：
- 所有 agent 一律展示 ThoughtChain：对简单问答没有收益，反而增加认知负担。

## Risks / Trade-offs

- [前端需要消费更多运行语义] -> 通过定义一层归并 reducer，把原始 SSE/log 事件先映射成稳定的 `thinking state`，避免渲染层直接处理底层细节。
- [`run_path` 结构后续变化可能影响归并] -> 在归并层集中维护路径识别规则，并为未知路径保留降级到普通正文的回退策略。
- [`replanner.response` 缺失时无法切换最终回答模式] -> 约定回退到最后一轮可见 planner/replanner 文本，避免消息空白。
- [工具调用过多导致 ThoughtChain 过长] -> 默认折叠整个 `Think` 卡片，并对连续工具项做 executor 轮次归并和摘要化展示。
- [前后端事件消费约定不清晰] -> 在 API 层补充类型和测试，明确哪些事件驱动 ThoughtChain，哪些事件驱动正文摘要。

## Migration Plan

1. 在后端或 API 适配层补充可稳定消费的运行事件字段，确保 `run_path`、工具调用和 replanner 输出能到达前端。
2. 在前端引入 `thinking state` 归并模型，先完成事件到状态的转换，再接入官方 `Think + ThoughtChain + Markdown` 组合渲染。
3. 对 `CopilotSurface` 做消息层升级，使 assistant 消息能同时承载过程摘要、最终回答和 ThoughtChain 状态。
4. 为简单 QA 保留原有轻量路径，确保非复杂 agent 不受影响。

回滚策略：
- 如果 ThoughtChain 展示不稳定，可以退回只显示正文流式输出，同时保留事件归并层代码，不影响基础 chat 能力。
- 如果 `run_path` 归并在某些 agent 上表现异常，可以仅对 `DiagnosisAgent` 等已知 plan-execute-replan agent 启用此展示。

## Open Questions

- 首版是让后端直接发新的 SSE 字段，还是先由前端从现有日志/事件中归并推导 ThoughtChain 状态？
- `tool_result` 首版是否展示完整结果，还是只展示摘要与成功/失败状态？
- 历史会话回放是否需要同步补齐 ThoughtChain 数据，还是先只保证新会话的实时展示？
