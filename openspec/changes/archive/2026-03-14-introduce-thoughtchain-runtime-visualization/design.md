# Design: ThoughtChain Runtime Visualization

## Context

当前 AI 模块已经具备：

- `plan_execute_replan` 编排与工具执行
- SSE 流输出
- 审批中断与恢复
- turn/message 持久化与历史恢复

但当前实现仍然沿用 block/card 作为过程可视化主模型。用户已确认新的目标不是“把卡片样式换成链样式”，而是：

- 用 `ThoughtChain` 表示一次真实执行链路
- 由后端直接输出链节点语义
- 前端不再从若干旧事件里猜测 plan/tool/replan 所属阶段

## Architecture

### 1. Event Model

后端新增链节点原生事件族，作为主语义源：

- `chain_started`
- `chain_node_open`
- `chain_node_patch`
- `chain_node_close`
- `chain_collapsed`
- `final_answer_started`
- `final_answer_delta`
- `final_answer_done`

兼容事件族可以保留，但降级为兼容输出，不再作为前端主消费对象。

### 2. Node Model

链节点只保留用户叙事所需的五种类型：

- `plan`
- `execute`
- `tool`
- `replan`
- `approval`

节点公共字段：

- `node_id`
- `kind`
- `title`
- `status`
- `summary`
- `details`
- `started_at`
- `finished_at`

审批节点额外字段：

- `approval.request_id`
- `approval.title`
- `approval.risk`
- `approval.details`
- `approval.actions`

设计约束：

- `tool_result` 不单独开新节点，只并入当前 tool 节点详情
- 节点标题必须是用户叙事型文案，而不是技术事件名
- 未来节点不能预先展示

### 3. Runtime Flow

一次正常执行流的目标形态：

1. `chain_started`
2. `chain_node_open(kind=plan, status=loading)`
3. `chain_node_patch(kind=plan, ...)`
4. `chain_node_close(kind=plan, status=done)`
5. `chain_node_open(kind=execute, status=loading)`
6. `chain_node_open(kind=tool, status=loading)` 或先关闭 execute 再打开 tool，具体以单活节点策略实现
7. 工具结果通过 `chain_node_patch` 更新当前 tool 节点详情
8. 如发生纠偏，关闭当前节点并打开 `replan`
9. 链结束后 `chain_collapsed`
10. `final_answer_started`
11. 多次 `final_answer_delta`
12. `final_answer_done`

审批流：

1. 打开 `approval` 节点
2. 节点保持 loading/waiting
3. 内嵌审批区显示
4. 用户批准或拒绝后，通过 patch/close 更新当前节点
5. 若批准，继续后续执行链；若拒绝，链进入终止并走最终答案

### 4. Frontend State Model

前端收口为两块状态：

- `thoughtChainState`
  - `nodes`
  - `isCollapsed`
  - `activeNodeId`
  - `collapsePhase`
- `finalAnswerState`
  - `visible`
  - `streaming`
  - `content`

前端只做三件事：

1. 消费链节点原生事件更新 ThoughtChain
2. 在 `chain_collapsed` 后启动链折叠动画
3. 折叠完成后显示最终答案并消费 `final_answer_delta`

不再让前端从 `phase_started`、`step_started`、`tool_result`、`delta` 手工归纳 ThoughtChain 节点。

### 5. Animation Strategy

动画目标不是炫技，而是保证阶段切换清晰。

过程链：

- 新节点出现：轻微上移 + fade in
- 当前节点完成：loading 状态平滑切换为 done
- 审批节点出现：与普通节点一致，只额外展开内嵌审批区
- 链结束：整体高度收起，标题替换为“思考完成”

最终答案：

- 只在链折叠后出现
- 容器淡入
- 文本按 chunk 微缓冲后做打字机输出

约束：

- 不允许过程链和最终答案同时主展示
- 不允许最终答案先出首段、过程链再折叠

### 6. Persistence and Replay

持久化层需要保留足够信息恢复：

- 链节点顺序
- 节点状态
- 节点详情
- 链是否已折叠
- 最终答案内容和完成状态

恢复原则：

- 未完成会话：恢复过程链并继续流式推进，不提前显示最终答案
- 已完成会话：默认显示“思考完成”的折叠态和最终答案区
- 旧 message 数据只作为兼容读取，不再是新主模型的事实来源

## Data Contract Direction

建议新增事件 payload 形态如下：

```json
{
  "type": "chain_node_open",
  "data": {
    "turn_id": "t_123",
    "node_id": "n_plan_1",
    "kind": "plan",
    "title": "正在整理执行计划",
    "status": "loading",
    "summary": "正在根据当前目标生成执行步骤"
  }
}
```

```json
{
  "type": "final_answer_delta",
  "data": {
    "turn_id": "t_123",
    "chunk": "共检查到 12 台主机，"
  }
}
```

## File Impact

后端重点文件：

- `internal/ai/events/events.go`
- `internal/ai/runtime/runtime.go`
- `internal/ai/runtime/sse_converter.go`
- `internal/ai/orchestrator.go`
- `internal/ai/state/chat_store.go`
- `internal/service/ai/session_recorder.go`

前端重点文件：

- `web/src/api/modules/ai.ts`
- `web/src/components/AI/Copilot.tsx`
- `web/src/components/AI/types.ts`
- `web/src/components/AI/hooks/useConversationRestore.ts`

新增文件方向：

- `web/src/components/AI/thoughtChainRuntime.ts`
- `web/src/components/AI/components/RuntimeThoughtChain.tsx`
- `web/src/components/AI/components/FinalAnswerStream.tsx`

## Migration Strategy

分两步：

1. 后端先同时输出链节点原生事件和兼容事件，前端接入新链模型
2. 新前端稳定后，把 block/card 主渲染路径降为兼容层，不再扩展

## Testing Strategy

测试按根因分析原则分层：

- 后端单元测试：验证事件序列、节点单活约束、审批节点生命周期、最终答案延后开始
- API 流测试：验证 SSE 顺序和 payload 语义，不只校验事件存在
- 前端 reducer 测试：验证节点按 SSE 逐步出现、切换 done、折叠后答案开始
- 交互测试：验证审批节点内嵌审批区、批准后链继续
- 恢复测试：验证未完成/已完成两类历史会话恢复行为

不接受“仅修复报错但不解释原因”的测试策略。任何失败都必须先解释：

- 为什么当前事件序列错了
- 为什么之前实现会这样写
- 为什么新改动能从根上修正这个问题
