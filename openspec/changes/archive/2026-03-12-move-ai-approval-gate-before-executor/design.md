## Context

当前 AI 控制平面的主链路是 `rewrite -> planner -> executor -> summarizer`，但 mutating/high-risk 步骤的审批门控放在 `executor` 内部的 step 调度过程中。结果是步骤先进入执行器状态，再被切换到 `waiting_approval`，前端同时接收到“执行中”和“等待确认”的混合信号；批准后 `ResumeStream` 只续跑 `executor.Resume()`，没有重新进入 `summary`，因此用户会看到专家执行结束后直接 `done`。

这条链路还放大了两个问题：
- 审批在语义上是执行前门控，却被实现成执行中断点，导致 UI 和状态模型不自然。
- `executor` 同时承担“审批决策门控”和“真实专家执行”两种职责，使 `resume`、turn/block 投影、以及失败处理都更难稳定。
- thought chain 现在只在实时对话里部分有用，历史恢复场景缺少折叠策略；执行细节虽然被采集，但前端没有把 step/tool/result 组织成稳定、易读的链路视图。
- “重新生成” 仍然通过重新发送同一条用户消息来触发，会污染会话时序，让历史看起来像用户重复提问。

## Goals / Non-Goals

**Goals:**
- 把 mutating/high-risk 步骤的审批门控前移到 `planner` 和 `executor` 之间。
- 让批准后的恢复流进入 `execute -> summary`，并继续同一 assistant `turn_id`。
- 统一前后端状态语义：审批是执行前 gate，不再是 executor 内部中断。
- 让前端审批块表现为短生命周期前置确认交互，批准后退出等待态并由执行/总结内容接管。
- 让 thought chain 在实时对话和历史恢复中都有一致且低噪音的折叠/展开策略。
- 让执行链路展示包含 step、tool call、tool result 等必要细节，但不退化成刷屏日志。
- 让“重新生成”在原用户消息上下文内生成新的 assistant 回答，而不是追加重复用户消息。

**Non-Goals:**
- 不重写 planner 本身的领域计划结构。
- 不引入新的独立审批中心或额外审批数据库模型。
- 不放弃现有 `/api/v1/ai/resume/step` 与 `/api/v1/ai/resume/step/stream` 路由兼容性。

## Decisions

### Decision: 审批门控上移为 orchestrator 的显式阶段

审批判定不再由 `executor.advanceScheduler()` 在 step 进入 `running` 后触发，而由 orchestrator 在拿到 `planner.DecisionPlan` 后扫描步骤风险策略并生成待审批 gate。

Rationale:
- 审批本质是“执行前授权”，而不是“执行开始后的暂停”。
- orchestrator 已经拥有 plan、turn、stream 和 summary 的完整生命周期视角，更适合做 gate。
- 这可以让 `executor` 专注于执行与证据生产。

Alternatives considered:
- 保留 executor 内审批，只补 `ResumeStream` summary。被拒绝，因为只能修当前 bug，不能修正错误的生命周期语义。
- 把审批判断下沉到 planner。被拒绝，因为 planner 应输出计划，不应直接管理运行时 gate 状态。

### Decision: 执行状态分成 pre-execution gate 与 actual execution 两段

执行前审批使用新的控制平面状态表达“plan 已准备，但 execution 尚未开始”。恢复批准后才初始化真正的 executor 运行。

Rationale:
- 这可以避免 step 先显示 `running` 再切 `waiting_approval`。
- 前端可以稳定展示“等待确认”而不是“执行中 + 等待确认”混态。

Alternatives considered:
- 复用现有 `waiting_approval` 但继续由 executor 生成。被拒绝，因为仍然隐含 step 已进入执行器。

### Decision: 批准后的 ResumeStream 必须继续 summary

`ResumeStream` 在批准后不再只调用 `executor.Resume()` 后直接 `done`，而是必须沿用主链路的 execute 完成逻辑，在 executor 成功完成后进入 `summarizeExecution()`。

Rationale:
- 用户感知的一次 assistant turn 不能在 resume 后只给执行结果、不产出总结。
- 这让聊天流与首次执行流在结果层保持一致。

Alternatives considered:
- 前端自己拼“执行结果即最终答案”。被拒绝，因为总结属于 AI core 行为，不该由前端推断。

### Decision: 前端审批块采用短生命周期门控交互

审批块只在等待用户决策时显示 CTA。用户确认或拒绝后，审批块立即退出等待态，替换为轻量状态提示，然后由执行块和总结块继续占据消息主内容。

Rationale:
- 审批块是前置门控，不应该在执行继续后还像一张永久表单卡一样占位。
- 这和新的后端 gate 语义一致。

Alternatives considered:
- 保留审批卡并置灰长期展示。被拒绝，因为会与随后的执行/总结块争夺视觉主位。

### Decision: thought chain 采用“当前活跃展开、历史默认折叠”的展示策略

thought chain 不再一刀切处理。当前流式 turn 默认展开最新活跃阶段；已完成消息和历史恢复消息默认折叠，仅在用户主动查看时展开。

Rationale:
- 当前对话中的思维链主要承担实时过程反馈，展开有价值。
- 历史对话以回看结论为主，默认全部展开会制造噪音。

Alternatives considered:
- 所有消息默认展开。被拒绝，因为历史对话密度过高。
- 所有消息默认折叠。被拒绝，因为当前流式阶段会失去过程反馈。

### Decision: 执行链路展示要结构化补全 step/tool/result，而不是堆叠文本

前端必须把 `step_update`、`tool_call`、`tool_result` 映射成结构化 thought chain 细节，而不是仅把摘要文本拼接成多行字符串。

Rationale:
- 用户要的是“当前在做什么、调用了什么、结果是什么”，不是无层次日志。
- 结构化 detail 更适合做去重、状态更新和历史回放。

Alternatives considered:
- 仅保留工具卡片，不补 thought chain 细节。被拒绝，因为 thought chain 是当前对话最直接的过程反馈区。

### Decision: AI 抽屉采用四层信息架构

同一 assistant turn 的主要信息按固定顺序展示为：`thought chain -> approval gate -> execution cards -> final answer`。这四层分别承担过程反馈、授权门控、动作状态和最终结论，不允许大段信息在多个区域重复堆叠。

Rationale:
- 运维类对话需要同时展示“系统在做什么”“用户要不要授权”“实际做了什么”“最终结论是什么”，但每层职责必须清楚。
- 固定层级可以让审批、执行、总结在视觉和语义上都更稳定。

Alternatives considered:
- 将 thought chain、工具卡和最终回答混在同一 Markdown 正文中。被拒绝，因为会把过程、动作和结论搅在一起。

### Decision: approval gate 采用短生命周期气泡而不是长期卡片

审批块在等待决策时显示为浅色门控气泡，点击确认或取消后立即退出等待态，只保留轻量回执，再由执行区和总结区接管主内容。

Rationale:
- 审批是流程节点，不是持久表单。
- 长期保留大卡片会和执行/总结争夺视觉焦点。

Alternatives considered:
- 批准后保留原审批卡并仅置灰。被拒绝，因为会让用户误以为仍可交互或仍处于待处理状态。

### Decision: regenerate 改成原用户消息下的 assistant 重答

“重新生成” 不再 append 一条新的用户消息。实现上可以是原地替换当前 assistant 回答，或在同一用户问题下维护 assistant 版本，但 UI 默认只显示最新回答。

Rationale:
- 用户并没有重新提问，只是要求同一问题重新作答。
- 这可以避免会话历史被伪造出重复 user turn。

Alternatives considered:
- 保持当前“删除 assistant 后重新调用 submit”。被拒绝，因为它会重新插入 user message，扭曲会话顺序。

### Decision: final answer 只承担结论与建议

最终回答区只承载结论、关键事实和下一步建议；审批状态解释、工具细节和重复阶段摘要不得再次在最终回答中大段重述。

Rationale:
- 用户最终要的是结论，不是重新阅读整个执行日志。
- 这可以避免 final answer 变成对上方卡片的重复拷贝。

Alternatives considered:
- 在 final answer 中再次完整复述审批、执行和工具日志。被拒绝，因为会显著增加噪音。

## Risks / Trade-offs

- [Risk] 将审批从 executor 挪出后，现有 `ExecutionState` 和 resume 逻辑可能出现兼容性断层。 → Mitigation: 保留 `session_id + plan_id + step_id` 作为恢复身份，并在 rollout 期间兼容旧状态读取。
- [Risk] orchestrator 会承担更多 gate 状态管理，若边界处理不清会重新变胖。 → Mitigation: 把 gate 判断和 gate state 投影收敛为单独 helper，不让 handler 或前端承担语义。
- [Risk] 前端审批块“立即消失”若实现不当，会让用户误以为点击未生效。 → Mitigation: 在消失前展示短暂 submitting 状态，并替换为轻量确认提示。
- [Risk] 旧 stream consumer 仍然依赖 `approval_required` 与 `done` 的组合语义。 → Mitigation: 保留兼容 SSE 事件，但修正其生命周期顺序和 summary continuation。
- [Risk] thought chain 细节过多会重新回到日志刷屏。 → Mitigation: 限制默认展示层级，只显示阶段摘要和结构化关键 detail，重复事件做幂等更新。
- [Risk] regenerate 如果直接覆盖原 assistant 内容，可能失去旧答案上下文。 → Mitigation: 默认只显示最新版本，但内部保留可选的版本元数据或最小重答标记，后续再决定是否显式 expose。
- [Risk] 审批气泡过于强调警告色会破坏整体聊天界面协调性。 → Mitigation: 使用低饱和浅色门控气泡、风险 badge 和轻量回执，而不是大面积警报底色。

## Migration Plan

1. 在 orchestrator 中引入 pre-execution approval gate，并让新请求先走 gate 再进 executor。
2. 调整 `Resume` / `ResumeStream` 从 gate 恢复到 execute，再进入 summary。
3. 更新 projector 与 SSE 兼容事件，使审批块、执行块、summary 块顺序与新生命周期一致。
4. 更新前端审批气泡、thought chain 展示策略、四层信息架构、执行 detail 投影和 regenerate 交互语义。
5. 用回归测试覆盖首次审批、批准恢复、拒绝终止、resume 后 summary 连续性、历史对话折叠策略和 regenerate 行为。

## Open Questions

- 旧的 `ExecutionStatusWaitingApproval` 是否保留命名，还是在实现中增加更明确的 pre-execution gate phase；当前建议保留外部兼容值，内部 phase 改为更具体的 gate 名称。
- 如果一个 plan 含多个需要审批的 mutating steps，本次 change 默认先 gate 当前可执行 step，而不一次性要求用户批准整张 plan。
