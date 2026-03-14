# Proposal: introduce-thoughtchain-runtime-visualization

## Summary

将当前 AI 助手的 `plan/status/tool/approval` 卡片式过程展示，重构为基于 `@ant-design/x` `ThoughtChain` 的单链式叙事过程展示，并将后端 SSE 协议收口为链节点原生事件。

该变更要求：

- 过程展示只保留一条用户叙事型链路
- 链节点按 SSE 实时推进，而不是预填未来阶段
- `approval_required` 作为普通链节点处理
- 过程链完成后自动折叠为“思考完成”
- 只有在链折叠后，最终答案区才开始流式打字机输出

## Why

当前 `plan-execute-replan-visualization` 虽然已经实现了 plan/step/tool/replan 的结构化事件与 UI 展示，但用户体验和架构语义仍然偏离目标：

- 前端主路径仍以 block/card 为中心，`ThoughtChain` 只是辅助容器，而不是唯一主展示模型
- 后端 SSE 仍以 `phase_started`、`tool_result`、`delta` 等通用事件为主，前端需要自行推断链路语义
- plan、tool、replan 在视觉上是并列卡片，不能自然表达一次 `planner -> executor -> tool_call -> replan -> executor` 的连续执行过程
- 最终答案与中间过程没有严格分离，planning JSON、replan 自述、tool 结果容易污染最终 assistant 内容
- 当前最终答案不是严格的链完成后再呈现的打字机流式输出

继续在现有 block/card 主路径上叠补丁，只会让协议和 UI 再次分叉。这次变更需要明确纠偏：由后端直接发“链节点原生协议”，前端只负责渲染 ThoughtChain 和最终答案。

## Goals

- 建立以 `ThoughtChain` 为唯一主展示路径的 AI 过程体验
- 用链节点原生 SSE 协议替代前端基于旧事件拼装过程链的做法
- 将 `plan`、`execute`、`tool`、`replan`、`approval` 统一为用户叙事型链节点
- 保证过程链只显示“已经发生和正在发生”的节点，不预渲染未来节点
- 保证每个节点的 `loading -> done` 生命周期由真实 SSE 驱动
- 保证过程链结束后先折叠，再显示最终答案
- 保证最终答案以 append-only 的流式打字机效果呈现
- 保证会话持久化和历史恢复能重建相同链路与最终答案关系

## Non-Goals

- 不在本次变更中继续扩展现有 block/card 主路径
- 不把 `tool_result` 单独提升为链上新节点
- 不把技术事件名直接暴露给用户作为节点标题
- 不要求保留当前 `phase/status/tool` 卡片式 UI 作为等价主展示

## Scope

### In Scope

- 后端新增链节点原生 SSE 协议
- orchestrator/runtime 侧改造为链节点语义源
- 前端基于 `ThoughtChain` 的单链渲染与折叠动画
- 最终答案延后展示与打字机流式输出
- `approval_required` 链节点内嵌审批区
- 会话持久化与恢复对新链模型的支持
- 兼容期内的旧事件降级支持

### Out of Scope

- 新增多种可切换的过程展示模式
- 面向最终用户暴露 debug/raw JSON 作为默认 UI
- 重做整个 AI 抽屉的视觉风格或品牌皮肤

## Success Criteria

- 用户在执行中只看到 ThoughtChain 过程链，不看到最终答案区
- 新节点仅在对应 SSE 到达时出现，当前节点为 loading，切换后前节点变 done
- `approval_required` 在链中表现为普通节点，并可内嵌审批交互
- 过程链结束后自动折叠成“思考完成”
- 折叠完成后最终答案区开始流式打字机展示
- 最终答案不混入 plan JSON、tool 参数或 replan 中间文本
- 恢复历史会话时，过程链与最终答案关系保持一致

## Risks

- 后端协议切换会影响现有前端和测试契约，需要清晰的兼容边界
- 过程链折叠与最终答案启动之间的动画编排如果处理不好，会出现闪动或重复内容
- 历史会话恢复如果仍混用旧 message/block 数据，容易再次出现展示错位

## Proposed Change Name Rationale

使用 `introduce-thoughtchain-runtime-visualization` 作为 change 名称，是为了强调本次改动不是简单 UI 替换，而是运行时事件语义、持久化投影和最终呈现方式的一次收口。
