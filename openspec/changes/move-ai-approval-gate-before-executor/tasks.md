## 1. Approval Gate Runtime

- [x] 1.1 在 `internal/ai/orchestrator.go` 引入 planner 之后、executor 之前的 pre-execution approval gate 判断与状态投影
- [x] 1.2 将 mutating/high-risk step 的等待审批状态改为 execution 之前建立，而不是在 executor step 进入 running 后建立
- [x] 1.3 调整 execution state/runtime phase，使 gate 状态与 actual execute 状态可区分但仍兼容现有 resume identity

## 2. Resume And Summary Continuation

- [x] 2.1 重构 `Resume` / `ResumeStream`，使批准后的恢复流进入 `execute -> summary` 而不是在 executor 完成后直接 `done`
- [x] 2.2 保证恢复流继续使用原 `turn_id`，并在拒绝时终止为 cancelled/rejected 状态
- [x] 2.3 清理 executor 内部的审批触发职责，保留执行与证据生产边界

## 3. Event Projection And API Compatibility

- [x] 3.1 调整 projector 与 SSE 兼容事件顺序，使 `approval_required` 出现在任何 gated `tool_call/tool_result` 之前
- [x] 3.2 更新 turn/block 投影，确保 approval block 表达 execution gate 而不是执行中断点
- [x] 3.3 保持 `/api/v1/ai/resume/step` 与 `/api/v1/ai/resume/step/stream` 兼容，并让 resumed stream 继续 summary

## 4. Frontend Approval Experience

- [x] 4.1 调整 AI 抽屉审批块语义为“执行前确认”，等待态时显示 CTA，批准或取消后立即退出等待态
- [x] 4.2 用轻量状态提示替代批准后的长期审批卡，并让执行块与总结块在同一 turn 中接续显示
- [x] 4.3 确保审批失败时只显示轻量失败反馈和重试入口，不恢复成可无限重复点击的初始表单
- [x] 4.4 调整 thought chain 展示策略：当前流式对话默认展开活跃阶段，历史恢复消息默认折叠
- [x] 4.5 补全 execute thought chain 结构化细节，合并展示 step_update、tool_call、tool_result，去掉重复刷屏
- [x] 4.6 重构“重新生成”交互，保留原用户消息时序，不再追加重复 user message
- [x] 4.7 落地四层信息架构：thought chain、approval gate、execution cards、final answer 的固定顺序与职责边界
- [x] 4.8 将审批块样式收敛为与聊天界面协调的门控气泡，并在成功后切换为轻量回执
- [x] 4.9 收紧链路显示内容，明确 thought chain、执行卡片和最终回答的职责边界

## 5. Validation

- [x] 5.1 为 orchestrator / resume stream 增加回归测试，覆盖审批前置、批准后 execute+summary、拒绝终止
- [x] 5.2 为前端 AI 抽屉增加交互测试，覆盖审批块短生命周期、历史折叠策略、结构化执行细节和 regenerate 行为
- [x] 5.3 运行 `openspec validate --changes "move-ai-approval-gate-before-executor" --json` 并修复校验问题
