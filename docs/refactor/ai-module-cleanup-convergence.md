# AI Module Cleanup Convergence

## 目标

在 `plan-execute-replan-visualization` proposal 开始实现前，先把 AI 模块中会与该 proposal 冲突、重复或误导实现方向的旧路径收口。此次收口只定义清理规则，不包含验证要求，也不要求一次性完成所有删除。

核心判断标准只有一条：

- 后续实现只能围绕 `turn -> block` 生命周期继续扩展，不能再以 `thoughtChain/stage` 作为主语义模型。

## 1. 保留路径

### 1.1 前端唯一主路径

后续前端实现只保留下面这条主路径：

- SSE 输入层：[`web/src/api/modules/ai.ts`](/root/project/k8s-manage/web/src/api/modules/ai.ts)
  - 以 `turn_started / block_open / block_delta / block_replace / block_close / turn_state / turn_done` 为主事件面
- turn reducer 与回放模型：[`web/src/components/AI/turnLifecycle.ts`](/root/project/k8s-manage/web/src/components/AI/turnLifecycle.ts)
  - 由 `ChatTurn + TurnBlock` 维护运行中状态和历史回放状态
- block 渲染层：[`web/src/components/AI/messageBlocks.ts`](/root/project/k8s-manage/web/src/components/AI/messageBlocks.ts)、[`web/src/components/AI/components/AssistantMessageBlocks.tsx`](/root/project/k8s-manage/web/src/components/AI/components/AssistantMessageBlocks.tsx)
  - 所有 planning/execution/approval/evidence/final answer 都应落成 block，而不是额外一套 `thoughtChain` UI
- 会话恢复层：[`web/src/components/AI/hooks/useConversationRestore.ts`](/root/project/k8s-manage/web/src/components/AI/hooks/useConversationRestore.ts)
  - 优先消费 `session.turns`
- UI 外层入口：[`web/src/components/AI/AICopilotButton.tsx`](/root/project/k8s-manage/web/src/components/AI/AICopilotButton.tsx)、[`web/src/components/AI/AIAssistantDrawer.tsx`](/root/project/k8s-manage/web/src/components/AI/AIAssistantDrawer.tsx)、[`web/src/components/AI/CopilotSurface.tsx`](/root/project/k8s-manage/web/src/components/AI/CopilotSurface.tsx)、[`web/src/components/AI/Copilot.tsx`](/root/project/k8s-manage/web/src/components/AI/Copilot.tsx)
  - 作为唯一容器入口和可视化 surface，避免再新增平行 chat surface

### 1.2 后端唯一主路径

后续后端实现只保留下面这条主路径：

- 编排入口：[`internal/ai/orchestrator.go`](/root/project/k8s-manage/internal/ai/orchestrator.go)
  - AI core 只负责生成 turn/block 生命周期和少量通用执行事件
- 事件合同：[`internal/ai/events/events.go`](/root/project/k8s-manage/internal/ai/events/events.go)、[`internal/ai/runtime/runtime.go`](/root/project/k8s-manage/internal/ai/runtime/runtime.go)
  - 主合同应围绕 turn/block 生命周期，而不是 `rewrite_result / planner_state / stage_delta / step_update`
- SSE transport shell：[`internal/service/ai/handler.go`](/root/project/k8s-manage/internal/service/ai/handler.go)
  - 仅负责 HTTP/SSE framing、resume 路由和 session 输出，不再拥有另一套旧语义组装
- 持久化与回放：[`internal/ai/state/chat_store.go`](/root/project/k8s-manage/internal/ai/state/chat_store.go)、[`internal/model/ai_chat.go`](/root/project/k8s-manage/internal/model/ai_chat.go)
  - 唯一结构化回放来源是 `ai_chat_turns + ai_chat_blocks`

## 2. 删除/降级路径

### 2.1 必须降级为 legacy-only 的旧事件名

下面这些事件不应再作为后续实现的主驱动事件：

- `rewrite_result`
- `planner_state`
- `plan_created`
- `stage_delta`
- `step_update`
- `clarify_required`
- `replan_started`

规则：

- 如果短期内还不能立刻删掉事件常量，可以保留常量名，但必须从“主语义路径”降级为 `legacy-only compatibility`
- 新增的 plan/execute/replan 可视化语义，不允许继续挂在 `thoughtChain stage` 这条旧事件面上
- `delta / thinking_delta / tool_call / tool_result / approval_required / done / error` 只允许作为 turn/block 内容补充通道存在，不能再反向定义整体流程语义

### 2.2 必须清理的旧状态组装逻辑

下面这些逻辑会把旧模型继续扩散，属于优先清理对象：

- 在后端把 SSE 旧事件组装成 `assistant.ThoughtChain`
- 在存储层把 `thought_chain` 反向投影成 block 列表
- 在前端把 `stage_delta/step_update` 再次还原成 `ThoughtStageItem[]`
- 在 UI 渲染层同时维护 `thoughtChain` 和 `turn.blocks` 两套展示状态

### 2.3 必须清理或降级的旧 UI 入口

下面这些入口不应再承担主渲染职责：

- `ThoughtChain` 作为主过程展示容器
- 基于 `message.thoughtChain` 的内联状态机
- 基于 legacy message 的 assistant 渲染优先级

规则：

- `turn.blocks` 已存在时，禁止继续优先读 `thoughtChain`
- `thoughtChain` 只能作为历史兼容读取的临时 fallback，且 fallback 不能反向影响新实现设计

## 3. 前端清理边界

### 3.1 FE worker 可改/可删文件

- [`web/src/components/AI/hooks/useAIChat.ts`](/root/project/k8s-manage/web/src/components/AI/hooks/useAIChat.ts)
  - 原因：当前仍以 `stage_delta / step_update / approval_required` 组装 `thoughtChain`，这是 proposal 前最核心的误导路径
  - 清理目标：移除或降级 thought-chain reducer，让该 hook 只服务于 turn/block 主路径，或只保留最薄的 legacy fallback

- [`web/src/components/AI/Copilot.tsx`](/root/project/k8s-manage/web/src/components/AI/Copilot.tsx)
  - 原因：当前文件内部仍保留大量 `thoughtChain` 状态更新与渲染分支，直接与 `turn.blocks` 双轨并存
  - 清理目标：删掉以 `ThoughtStageItem[]` 为主的渲染和状态推进逻辑，保留 block 渲染路径

- [`web/src/components/AI/types.ts`](/root/project/k8s-manage/web/src/components/AI/types.ts)
  - 原因：同时暴露 `ChatTurn/TurnBlock` 与 `ThoughtStage*` 两套一等模型，类型层面仍在鼓励双轨实现
  - 清理目标：将 `ThoughtStage*`、旧 SSE 事件类型降级为 compatibility 类型；`ChatTurn/TurnBlock` 成为唯一一等模型

- [`web/src/api/modules/ai.ts`](/root/project/k8s-manage/web/src/api/modules/ai.ts)
  - 原因：当前 SSE parser 同时把 turn/block 和旧 stage 事件视为平级主入口
  - 清理目标：把旧事件 handler 明确标记为 compatibility only，并收缩默认消费面

- [`web/src/components/AI/hooks/useConversationRestore.ts`](/root/project/k8s-manage/web/src/components/AI/hooks/useConversationRestore.ts)
  - 原因：当前虽已优先读 `session.turns`，但仍保留完整 legacy `thoughtChain` 恢复路径
  - 清理目标：保留只读 fallback，但不允许继续扩展 legacy message 恢复能力

- [`web/src/components/AI/thoughtChainMetrics.ts`](/root/project/k8s-manage/web/src/components/AI/thoughtChainMetrics.ts)
  - 原因：指标定义绑定旧事件集合，会持续给后续实现错误约束
  - 建议：直接删除；如果暂时保留，也必须降级为 legacy metrics

- [`web/src/components/AI/thoughtChainMetrics.test.ts`](/root/project/k8s-manage/web/src/components/AI/thoughtChainMetrics.test.ts)
  - 原因：测试在加固旧事件面，不应继续作为未来实现依据
  - 建议：随 `thoughtChainMetrics.ts` 一并删除

### 3.2 FE worker 不应新增的内容

- 不要再新增任何 `ThoughtChain*` 组件或 `ThoughtStage*` reducer
- 不要再新增基于 `stage_delta / step_update` 的新 UI
- 不要再让 `Copilot` 与 `useAIChat` 分别维护两套 assistant 流程状态

## 4. 后端清理边界

### 4.1 BE worker 可改/可删文件

- [`internal/ai/events/events.go`](/root/project/k8s-manage/internal/ai/events/events.go)
  - 原因：事件常量层仍把 plan-stage 旧命名与 turn lifecycle 并列，误导后续编排设计
  - 清理目标：把旧阶段事件降级或移除，只保留 turn/block 主合同及必要通用事件

- [`internal/ai/runtime/runtime.go`](/root/project/k8s-manage/internal/ai/runtime/runtime.go)
  - 原因：当前 runtime 重新导出旧阶段事件，继续把它们定义为一等事件类型
  - 清理目标：收缩 runtime 对外事件面，避免 proposal 在旧合同上继续生长

- [`internal/ai/runtime/sse_converter.go`](/root/project/k8s-manage/internal/ai/runtime/sse_converter.go)
  - 原因：当前 converter 仍主要产出 `stage_delta / step_update`
  - 清理目标：改成输出 turn/block 生命周期；不再把 planning/execution 语义编码成 thought-chain stage patch

- [`internal/ai/orchestrator.go`](/root/project/k8s-manage/internal/ai/orchestrator.go)
  - 原因：主编排器仍通过旧 converter 把运行过程投影到 stage 事件流
  - 清理目标：统一从 orchestrator 直接推进 turn/block 语义，不再为旧 stage 模型兜底

- [`internal/service/ai/session_recorder.go`](/root/project/k8s-manage/internal/service/ai/session_recorder.go)
  - 原因：该文件仍以 `assistant.ThoughtChain` 为中心记录过程，并从旧事件推导最终 message metadata
  - 清理目标：改为记录 turn/block；停止生成 `ThoughtChain`

- [`internal/ai/state/chat_store.go`](/root/project/k8s-manage/internal/ai/state/chat_store.go)
  - 原因：`buildBlocks(record)` 正在把 `record.ThoughtChain` 再投影成 block，属于典型的反向适配旧模型
  - 清理目标：删除基于 `ThoughtChain` 生成 block 的逻辑，让 block 来自真实 block 生命周期而不是 message metadata

- [`internal/service/ai/handler.go`](/root/project/k8s-manage/internal/service/ai/handler.go)
  - 原因：session 输出中仍同时暴露 `messages[].thoughtChain` 和 `turns[]`
  - 清理目标：让 `turns[]` 成为主返回；`thoughtChain` 如果暂留，只能作为 compatibility 字段

- [`internal/service/ai/handler_aiv2_test.go`](/root/project/k8s-manage/internal/service/ai/handler_aiv2_test.go)
  - 原因：当前测试样例仍在加固 `stage_delta / step_update` 流
  - 建议：随着主合同切换同步调整或删除，避免测试继续锁死旧路径

### 4.2 BE worker 不应继续做的事

- 不要再通过 message metadata 生成结构化 turn/block
- 不要再让 gateway/session recorder 发明 AI core 语义
- 不要在 proposal 实现前再新增任何基于 `planner_state`、`stage_delta` 的过渡事件

## 5. 风险提示

下面这些地方即使存在冲突，这次也建议先不要动：

- [`internal/ai/contracts.go`](/root/project/k8s-manage/internal/ai/contracts.go) 中的 rollout/config 开关
  - 原因：这些开关属于运行时切换和回滚控制，不是本次“语义主路径收口”的直接对象

- [`internal/model/ai_chat.go`](/root/project/k8s-manage/internal/model/ai_chat.go) 以及已有 migration 文件
  - 原因：数据表已经同时承载 message 与 turn/block 历史，本次不要做 schema 级清理或历史数据迁移

- `/api/v1/ai/resume/step`、`/api/v1/ai/resume/step/stream`、兼容 alias 路由
  - 原因：这些是外部接口稳定面；本次可以收缩内部语义，但不要先动路由和入口名

- `aiv2` 与 legacy runtime 的运行时选择
  - 原因：这是运行模式问题，不等于过程可视化主路径问题；本次不要把“删 thoughtChain”误做成“删 runtime fallback”

- 文档、OpenSpec archive、历史测试以外的外部消费者假设
  - 原因：本次目标是先给 worker 明确清理边界，不做全仓兼容性证明，也不做批量文档同步

## 执行原则

- 优先删“反向适配旧模型”的代码，再删旧 UI 和旧事件常量
- 凡是 `turn.blocks` 与 `thoughtChain` 同时存在的地方，一律以 `turn.blocks` 为准
- 兼容层如果暂时保留，必须显式标注为 `legacy-only`，不能继续作为 proposal 设计输入
- 本次收口不包含测试和验证要求
