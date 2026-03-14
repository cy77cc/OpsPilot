# Plan-Execute-Replan Implementation Remap

## 目标

`plan-execute-replan-visualization` 这组 OpenSpec artifacts 的产品目标仍然成立，但实现落点已经被 AI 模块 cleanup 重排。当前代码库的主语义路径是：

- 前端：`Copilot.tsx` 驱动会话状态，`turnLifecycle.ts` 维护 `ChatTurn + TurnBlock`，`messageBlocks.ts`/`AssistantMessageBlocks.tsx` 负责渲染
- 后端：`internal/ai/orchestrator.go` 驱动执行，`internal/ai/runtime/*` 提供运行时状态与 SSE 转换，`events/events.go` 中 turn/block 事件是主合同，旧 stage 事件只保留兼容语义

因此，本说明只做一件事：把 proposal/design/tasks 里仍指向 `useAIChat/thoughtChain` 的实现说明，重映射到 cleanup 后仍然有效的主路径。

## 1. artifacts 与 cleanup 后代码结构的偏差

### 1.1 proposal.md 的偏差

- 架构图仍把前端主入口写成 `useAIChat -> ThinkingProcessPanel -> StageTimeline`
- 文件变更清单仍把 `web/src/components/AI/hooks/useAIChat.ts` 视为前端状态中心
- 后端方案仍把 `phase_started / phase_complete / plan_generated / step_started` 视为新的主事件面

重映射结论：

- proposal 的“展示 plan-execute-replan 过程”目标可以保留
- proposal 的“前端通过 thoughtChain stage 面板承载”这部分已经过时
- proposal 的“后端新增平行阶段事件流”不能再直接照做，必须优先落到 turn/block 主合同

### 1.2 design-frontend.md 的偏差

- `web/src/components/AI/hooks/useAIChat.ts` 已不是当前实现主路径
- `ThoughtStageItem`、`ThinkingProcessPanel`、`StageTimeline`、`PlanStepsList`、`ToolExecutionTimeline` 被当作一等模型与一等容器
- 设计默认通过 `stage_delta / step_update / approval_required` 推动 UI

重映射结论：

- 前端一等模型应改为 `ChatTurn`、`TurnBlock`、`AssistantMessageBlock`
- 规划、执行、审批、证据、结论都应作为 block 渲染，不应再新增一套 thoughtChain 容器
- 如果仍需保留“阶段感”，应通过 plan/status/tool/approval block 的组合表达，而不是恢复 `ThoughtStageItem[]`

### 1.3 design-backend.md 的偏差

- 文档仍假设需要新建 `phase_detector.go`、`plan_parser.go`
- 文档仍假设 `runtime.go` 需要扩出一批面向 stage 的结构体
- 文档仍把旧 `tool_call/tool_result` 升级为带 step 语义的独立阶段事件链

重映射结论：

- 后端真实主入口是 `orchestrator.go`，不是“phase detector + plan parser + stage event”并列组合
- 运行时主合同应围绕 `meta / turn_started / turn_state / delta / approval_required / done / error` 与 block 生命周期扩展
- `phase_detector.go`、`plan_parser.go` 只应在 orchestrator 无法直接产出 block payload 时，作为内部辅助模块引入；不再是设计上的必选落点

### 1.4 tasks.md 的偏差

- Phase 2 仍要求在 `useAIChat.ts` 中新增 reducer 和 handler
- Phase 3 仍要求新建一批 `ThoughtChain*` 风格组件
- Phase 4 把 `Copilot.tsx` 只当作“集成新组件”的壳，而不是当前状态与渲染主入口
- 后端任务默认围绕“新增 stage 事件”展开，而非在现有 orchestrator/runtime 主路径中落块

重映射结论：

- 任务可继续，但文件级目标必须改写
- task 的执行单位应从 “stage reducer / timeline panel” 改成 “turn reducer / block schema / block render”

## 2. 新的前端实现落点

前端必须基于 `Copilot.tsx` 和 turn-block 主路径继续实现。

### 2.1 唯一主路径

- `web/src/components/AI/Copilot.tsx`
  - 当前真实入口
  - 负责 SSE handler 注册、消息缓冲、assistant turn 生命周期推进
  - 即使文件里还残留 `thoughtChain` 兼容逻辑，新的 visualization 也应优先接入这里，而不是复活 `useAIChat.ts`
- `web/src/components/AI/turnLifecycle.ts`
  - 负责 `turn_started / turn_state / block_open / block_delta / block_replace / block_close / turn_done` 的 reducer 语义
  - Plan/Execute/Replan 的“过程可视化”应优先表达为 block 的打开、更新、关闭
- `web/src/components/AI/types.ts`
  - 一等模型是 `ChatTurn`、`TurnBlock`
  - 新需求应扩展 `TurnBlock.data` 的 payload 约定，而不是继续扩 `ThoughtStageItem`
- `web/src/components/AI/messageBlocks.ts`
  - 负责把 `TurnBlock` 规整成渲染块
  - 如果要展示 plan steps、tool timeline、approval waiting、evidence，应先定义 block 类型与 payload 形状
- `web/src/components/AI/components/AssistantMessageBlocks.tsx`
  - 真实渲染面
  - 新 UI 落点应是增强现有 `plan/tool/approval/status/evidence` block 的展示，而不是新增并行 `ThinkingProcessPanel`

### 2.2 前端实现原则

- `Copilot.tsx` 是状态接线板，不再把 `useAIChat.ts` 当作必须恢复的中间层
- 先定义“一个 turn 内需要哪些 block”，再决定是否需要补充新的 block type
- “阶段感”来自 block 顺序和 block 状态，不来自另一份 thoughtChain reducer
- `thoughtChain` 只作为 legacy fallback 读取，不再作为新实现输入

### 2.3 推荐 block 映射

- planning：`status` + `plan`
- executing：`tool` + `status` + `evidence`
- approval waiting：`approval`
- replanning：优先复用 `status` 或 `plan` 的 replace/update；只有在现有 block 无法表达时，再新增专用 block
- final answer：现有 `text` block

## 3. 新的后端实现落点

后端必须基于 orchestrator/runtime 当前主路径继续实现。

### 3.1 唯一主路径

- `internal/ai/orchestrator.go`
  - 当前对外唯一编排入口
  - 负责 `Run / Resume / ResumeStream`
  - 应作为 plan-execute-replan 可视化语义的唯一收口点
- `internal/ai/runtime/sse_converter.go`
  - 负责把 orchestrator 内部状态投射为 SSE
  - 新增语义优先做成 turn/block 生命周期事件或已有主事件 payload 扩展
- `internal/ai/runtime/runtime.go`
  - 负责运行时公共类型和执行状态
  - 可扩的是 `ExecutionState` / `StepState` / block payload 需要的公共字段，不应回到 stage-first 类型设计
- `internal/ai/events/events.go`
  - turn/block 事件是主合同
  - `rewrite_result / planner_state / plan_created / stage_delta / step_update / replan_started` 只能作为兼容层，不应再作为新主路径

### 3.2 后端实现原则

- plan/execute/replan 的可视化语义，应由 orchestrator 直接根据真实执行进度发出
- 如果需要“步骤”概念，优先落在 `ExecutionState.Steps` 和 block payload 中
- 如果需要“重规划”提示，优先作为 turn/block 更新而不是再开一条 stage 事件总线
- 任何新事件设计都应先问：能否用 `turn_started / turn_state / block_* / approval_required / done / error` 表达

### 3.3 可接受的辅助模块边界

- `phase_detector.go`
  - 只有当 ADK 事件无法直接映射为 block 时，才作为 orchestrator 内部推断器引入
  - 不再作为对外语义中心
- `plan_parser.go`
  - 只有当 planner 输出必须解析成结构化步骤时才需要
  - 产物应服务于 block payload 或 `ExecutionState.Steps`，不是服务于 `ThoughtStageItem`

## 4. 建议的实现顺序

### 4.1 可以立即开始的 task slice

最小可启动 slice：

1. 定义前后端共同认可的 block 级语义
   - 明确 planning、executing、approval、replanning 各自落成哪类 block
   - 明确每类 block 的最小 payload
2. 在后端 orchestrator/runtime 上补齐这些 block 的发射点
   - 不先追求“完美阶段图”
   - 先让 plan、tool、approval 能稳定出现在一个 turn 里
3. 在 `Copilot.tsx` + `turnLifecycle.ts` 接上这些 block
   - 先保证 reducer 正确推进
4. 在 `AssistantMessageBlocks.tsx` 完成最小展示
   - 优先 plan/tool/approval/status
   - replan 先复用 status 或 plan replace

这个 slice 可以直接开工，因为它不依赖删除 legacy thoughtChain，也不依赖先发明新的阶段事件体系。

### 4.2 后续顺序

1. 后端先把 block 事件稳定输出
2. 前端接 turn/block reducer
3. 前端增强 block 呈现细节
4. 最后再决定哪些 legacy stage 兼容逻辑可以继续缩减

### 4.3 不建议的顺序

- 先新建 `ThinkingProcessPanel`
- 先恢复 `useAIChat.ts` 为主 reducer
- 先设计一整套 `phase_started / step_started / replan_triggered` 平行事件面

这些顺序都会把实现重新拉回 cleanup 已经否定的旧路径。

## 5. tasks.md 的继续项与语义重解释

### 5.1 可以继续推进的项

- Phase 1 的“后端事件增强”
  - 语义上继续
  - 但实现落点改为 orchestrator/runtime 的 turn-block 主路径
- Phase 4 的“Copilot 集成”
  - 语义上继续
  - 但 `Copilot.tsx` 不是简单集成壳，而是前端主状态入口
- 审批展示、工具执行展示、最终结论展示
  - 都可以继续
  - 只是要改成 block 语义
- 文档更新
  - 可以立即继续

### 5.2 需要语义上重解释的项

- `web/src/components/AI/hooks/useAIChat.ts` 相关任务
  - 重解释为：在 `Copilot.tsx`、`turnLifecycle.ts`、`ai.ts` 中完成 turn/block 状态推进
- `ThoughtStageItem`、`PlanStep`、`ToolExecution` 扩展任务
  - 重解释为：扩展 `TurnBlock` payload 和必要的 runtime step state
- `ThinkingProcessPanel.tsx` / `StageTimeline.tsx` / `PlanStepsList.tsx` / `ToolExecutionTimeline.tsx`
  - 重解释为：增强 `messageBlocks.ts` 和 `AssistantMessageBlocks.tsx` 对现有 block 的渲染
  - 只有在 block 渲染无法承载时，才考虑新增局部 block 组件，而不是新增总容器
- `phase_started / phase_complete / plan_generated / step_started / step_complete / replan_triggered`
  - 重解释为：优先用 turn/block 生命周期表达
  - 只有确实无法表达时，才补兼容事件，不应反客为主
- `phase_detector.go` / `plan_parser.go`
  - 重解释为：可选辅助模块，不是阶段一必须交付物

### 5.3 建议按语义重写 tasks.md 时采用的口径

- “新增 thoughtChain stage reducer” 改为 “在 turn/block reducer 上承载过程可视化”
- “新增过程面板组件” 改为 “增强 block 渲染与 block payload”
- “新增阶段事件” 改为 “在现有 turn/block SSE 主合同上补足 plan/execute/replan 语义”

## 结论

`plan-execute-replan-visualization` 不需要推翻目标，但必须放弃 artifacts 中的 `useAIChat/thoughtChain` 实现假设。新的实现映射应统一到：

- 前端：`Copilot.tsx` -> `turnLifecycle.ts` -> `messageBlocks.ts` -> `AssistantMessageBlocks.tsx`
- 后端：`orchestrator.go` -> `runtime/sse_converter.go` -> `events/runtime` 主合同

最先可执行的 task slice 不是“新建 thoughtChain 面板”，而是“定义 block 语义并在 orchestrator/Copilot 主路径打通一个可视化 turn”。
