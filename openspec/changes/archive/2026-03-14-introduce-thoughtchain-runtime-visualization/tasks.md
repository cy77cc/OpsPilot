# Tasks: introduce-thoughtchain-runtime-visualization

## 1. Backend Native Chain Protocol

- [x] 1.1 在 `internal/ai/events/events.go` 新增链节点原生事件常量：
  - `chain_started`
  - `chain_node_open`
  - `chain_node_patch`
  - `chain_node_close`
  - `chain_collapsed`
  - `final_answer_started`
  - `final_answer_delta`
  - `final_answer_done`
- [x] 1.2 在 `internal/ai/runtime/runtime.go` 新增链节点数据结构和最终答案流数据结构
- [x] 1.3 重构 `internal/ai/runtime/sse_converter.go`，由运行时直接投影用户叙事型链节点事件
- [x] 1.4 重构 `internal/ai/orchestrator.go`，确保 `plan -> execute -> tool -> replan -> approval` 生命周期输出为链节点原生协议，而不是前端推断
- [x] 1.5 约束 `tool_result` 只更新当前 tool 节点详情，不生成新链节点
- [x] 1.6 保留兼容事件输出，但将其降级为 compatibility 层

## 2. Final Answer Separation

- [x] 2.1 后端保证最终答案只通过 `final_answer_*` 事件输出，不再混入过程链事件
- [x] 2.2 过滤 planner JSON、tool 参数、replan 中间态，避免污染最终答案内容
- [x] 2.3 确保 `chain_collapsed` 先于 `final_answer_started`
- [x] 2.4 为最终答案增加 append-only chunk 输出，支持前端打字机渲染

## 3. Frontend ThoughtChain Runtime

- [x] 3.1 在 `web/src/api/modules/ai.ts` 新增链节点原生 SSE 事件类型与 handler
- [x] 3.2 新建 `web/src/components/AI/thoughtChainRuntime.ts`，作为前端链节点状态机
- [x] 3.3 新建 `web/src/components/AI/components/RuntimeThoughtChain.tsx`，以 `@ant-design/x` `ThoughtChain` 为唯一主过程展示组件
- [x] 3.4 新建 `web/src/components/AI/components/FinalAnswerStream.tsx`，负责最终答案的流式打字机展示
- [x] 3.5 改造 `web/src/components/AI/Copilot.tsx`，移除 block/card 作为主路径的过程渲染地位
- [x] 3.6 将 `approval_required` 映射为普通链节点，并在节点详情中内嵌审批区

## 4. Motion and Transition

- [x] 4.1 设计并实现节点出现、loading->done、链折叠、最终答案淡入的过渡动画
- [x] 4.2 保证执行中只显示过程链，过程链折叠后才显示最终答案
- [x] 4.3 处理 reduced-motion 场景下的降级行为

## 5. Persistence and Replay

- [x] 5.1 调整持久化模型，确保链节点顺序、状态、最终答案分离存储
- [x] 5.2 更新 `internal/service/ai/session_recorder.go` 和 `internal/ai/state/chat_store.go` 支持新模型
- [x] 5.3 更新 `web/src/components/AI/hooks/useConversationRestore.ts`，恢复已完成和未完成会话时保持相同体验
- [x] 5.4 明确旧 message/block 数据的兼容读取策略，不再作为新主模型的事实来源

## 6. Verification

- [x] 6.1 后端测试覆盖正常流、审批流、replan 流、异常流的链节点事件顺序
- [x] 6.2 前端测试覆盖节点逐步出现、状态切换、折叠后答案启动、审批节点展示
- [x] 6.3 API 测试覆盖 `chain_collapsed` 与 `final_answer_started` 的顺序保证
- [x] 6.4 恢复测试覆盖未完成会话与已完成会话
- [x] 6.5 文档更新：记录新原生协议、动画约束和兼容策略
