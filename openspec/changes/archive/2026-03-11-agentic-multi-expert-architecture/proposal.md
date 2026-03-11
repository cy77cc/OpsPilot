# Proposal: Agentic Multi-Expert Architecture

## 概述

基于 Eino `v0.8.0` 的 ADK 新能力，重构 AI 编排入口为：

```text
Gateway / API
    -> AI Orchestrator Host
        -> Rewrite
        -> Planner
        -> Executor Runtime
            -> Expert Agents as Tools
        -> Summarizer
```

本次变更的目标不是简单把单体 Agent 拆成更多 Agent，而是建立一条完整、可演进的 AI 执行链路：

- 在入口增加 `Rewrite` 阶段，将用户口语化输入改写为稳定的任务表达
- 用 `Planner` 负责资源解析、权限预检查、结构化计划
- 用后端 `Executor Runtime` 负责确定性的 DAG 调度、审批、恢复和状态机
- 用 `Expert Agents as Tools` 承接领域执行能力
- 用 `Summarizer` 输出最终结论
- 用前端 `ThoughtChain` 展示阶段性进展，让用户感知 AI 正在工作

## 本次提案要解决的问题

### 1. 当前 AI 编排边界不清晰

现有实现中，AI 入口、流式输出、工具调用、审批恢复和会话语义没有稳定收敛到一个明确的后端宿主边界。

结果是：

- Gateway 层和 AI core 的职责容易漂移
- 后续接入 `Rewrite / Planner / Executor / Summarizer` 时缺少稳定入口
- 前后端都难以围绕统一契约演进

### 2. 用户输入过于口语化，直接进入规划不稳定

用户经常会输入类似：

- “看看 payment-api 最近是不是有点慢”
- “顺便查一下是不是刚发版”
- “帮我处理一下这个服务”

如果直接把这类输入送给 Planner，会让 Planner 同时承担：

- 语言清洗
- 意图规整
- 资源识别
- 计划生成

这会降低计划稳定性，也会放大 prompt 复杂度。

### 3. 原提案对前端对接不够

原提案更多停留在后端架构层，虽然列出了 SSE 事件，但没有真正定义：

- 前端需要消费的对象模型
- 哪些内容要展示给用户
- 哪些内容只属于后端运行时细节

这会导致前端拿到很多事件名，却没有清晰的 UI 语义。

### 4. 用户缺少“AI 正在工作”的感知

如果界面只展示最终答案，用户在复杂排查场景中会感知为：

- 卡住了
- 没响应
- 不知道 AI 到底做了什么

尤其复杂调查、跨专家协作、审批等待场景下，必须把阶段性过程以产品化方式呈现出来。

## 核心设计调整

相较于原提案，本次提案做以下关键调整。

### 调整 1: 明确采用 Eino `v0.8.0` 新特性

本次方案不再泛泛地描述 “基于 Eino ADK”，而是明确利用 `v0.8.0` 中已具备的能力：

- `adk.NewAgentTool(...)`
- `Transfer / SetSubAgents / DeterministicTransfer`
- `interrupt / resume`
- `ADK middleware`
- `prebuilt planexecute / supervisor` 的设计思路

采用原则：

- `Rewrite / Planner / Summarizer` 适合使用 Agent 与 Transfer 语义
- `Experts` 适合通过 `AgentAsTool` 接入
- `Executor` 仍由后端运行时代码负责，不把调度确定性交回模型

### 调整 2: 在入口新增 Rewrite 阶段

新增显式 `Rewrite` 阶段，用于把用户口语化输入改写成稳定的任务表达。

职责：

- 规整用户意图
- 抽取资源线索
- 初步判断任务模式
- 生成更适合 Planner 消费的输入

示意：

```text
用户输入:
  “帮我看看 payment-api 最近是不是有点慢，顺便查下是不是刚发版”

Rewrite 输出:
  - goal: 排查 payment-api 响应变慢，并核对近期发布是否相关
  - resource_hint: service_name=payment-api
  - operation_mode: investigate
  - candidate_domains: service, observability, delivery
```

`Rewrite` 不负责：

- 最终资源解析
- 权限检查
- 执行计划生成
- 调用 mutating 工具

### 调整 3: Executor 定义为 Runtime，而不是纯 Agent

原提案中将 `Executor` 命名为 Agent，但本质职责包括：

- DAG 调度
- 并行/串行控制
- 重试
- 超时
- 审批等待
- Resume 恢复
- step 状态管理

这些职责属于确定性运行时，而不是模型自治职责。

因此本次提案将其重新定义为：

```text
Executor Runtime
```

它可以调用 Expert Agent Tools，但自身不应退化为一个再次持有高度自治的“大编排 Agent”。

### 调整 4: 前端引入 ThoughtChain 作为阶段可视化主载体

前端新增：

```tsx
import { ThoughtChain } from "@ant-design/x";
```

用 `ThoughtChain` 展示用户可感知的 AI 工作阶段，而不是直接暴露底层工具日志。

推荐的一级阶段：

1. 理解你的问题
2. 整理排查计划
3. 调用专家执行
4. 等待确认或补充
5. 生成结论

其中：

- `ThoughtChain` 用于展示过程
- 正文回答用于输出最终给用户的可读答案
- `summary` 用于输出最终结构化结论

目标是让用户感知：

- AI 已开始工作
- 当前处于哪个阶段
- 为什么正在等待
- 最终结论是什么

### 调整 5: 后端入口重新定义，不再沿用旧 handler 假设

原提案默认存在一个稳定的 handler/orchestrator seam，但当前旧 handler 已删除。

因此本次提案明确要求：

- `internal/service/ai` 仅负责 route 和 transport shell
- `internal/ai` 负责 AI orchestration host
- 所有 AI 运行时语义在 `internal/ai` 内收敛

这比继续围绕旧 handler 设计更符合当前后端规划。

## 新的后端架构边界

### 1. Gateway / Route 层

职责：

- 请求映射
- 鉴权
- session shell
- SSE transport framing

不负责：

- 规划
- 调度
- 审批语义
- 前端事件语义定义

### 2. AI Orchestrator Host

位于 `internal/ai`，作为唯一稳定编排入口。

职责：

- 接收 `RunRequest / ResumeRequest`
- 串联 Rewrite / Planner / Executor Runtime / Summarizer
- 负责统一 trace、session、execution state 生命周期
- 负责输出统一事件语义

### 3. Rewrite

职责：

- 口语输入规整
- 意图归一化
- 任务模式识别
- 资源 hint 提取

### 4. Planner

职责：

- 资源解析
- 权限预检查
- 用户澄清
- 输出结构化 `ExecutionPlan`

### 5. Executor Runtime

职责：

- 基于 `ExecutionPlan` 执行 step
- 管理依赖关系和状态机
- 调用 Expert Agent Tools
- 处理审批、恢复、重试和超时

### 6. Expert Agents as Tools

领域专家包括：

- HostOpsExpert
- K8sExpert
- ServiceExpert
- DeliveryExpert
- ObservabilityExpert

各专家只持有本领域工具，不暴露 Planner support tools。

### 7. Summarizer

职责：

- 汇总执行证据
- 形成对用户可读的结论
- 判断是否需要补充调查
- 输出 `summary`

## 新的前端对接方案

### 1. 前端不再直接围绕零散工具事件设计

前端对接应围绕以下 UI 对象，而不是仅围绕 SSE 事件名：

- `ThoughtChain`
- `PlanView`
- `StepView`
- `ApprovalView`
- `SummaryView`

### 2. ThoughtChain 的展示原则

`ThoughtChain` 只展示用户应该看到的阶段内容，不展示纯后端调试信息。

展示原则：

- 展示 AI 在做什么
- 展示当前进度和等待原因
- 展示关键阶段结果摘要
- 不直接倾倒 tool JSON 和底层调度噪音

### 3. summary 的职责

`summary` 仍然是最终对用户输出的正文内容，不与 `ThoughtChain` 混为一谈。

关系如下：

```text
ThoughtChain = 过程感知
summary      = 最终结论
```

### 4. 前端阶段建议

| 阶段 Key | 用户可见标题 | 说明 |
|---------|-------------|------|
| `rewrite` | 理解你的问题 | 展示归一化任务表达 |
| `plan` | 整理排查计划 | 展示资源解析和计划摘要 |
| `execute` | 调用专家执行 | 展示专家阶段进展和关键发现 |
| `user_action` | 等待你处理 | 展示澄清或审批动作 |
| `summary` | 生成结论 | 展示最终汇总结论摘要 |

### 5. 建议的前端事件语义

为了让前端能稳定消费，本次提案要求后端按阶段输出高层语义事件。

建议事件包括：

- `meta`
- `rewrite_result`
- `planner_state`
- `plan_created`
- `step_update`
- `approval_required`
- `clarify_required`
- `delta`
- `summary`
- `done`
- `error`

其中：

- `step_update` 面向前端，优先表达状态变化
- 后端内部如需 `step_start / step_result / expert_progress`，可以内部使用，但对前端输出应尽量收敛
- `delta` 用于最终用户可读正文的流式输出
- `summary` 用于最终结构化结论，不替代正文流式能力

## 原提案中删除或收敛的内容

本次提案显式删去或收敛以下不合理部分。

### 1. 删除“所有层都按 Agent 对待”的隐含假设

不再把 `Executor` 继续包装成一个高度自治的 Agent 概念。

改为：

- `Rewrite / Planner / Summarizer` 是 agent-friendly 的阶段
- `Experts` 是 agent-as-tool
- `Executor` 是 runtime

### 2. 删除对旧 handler 结构的依赖假设

原提案默认旧 handler 仍是稳定基础，这与当前实际情况不符。

本次改为：

- 按新的后端规划重新定义入口
- 不再围绕旧 handler 组织运行时

### 3. 收敛前端事件设计

原提案列出大量 SSE 事件，但没有前端对象模型支撑。

本次改为：

- 先定义前端 UI 模型
- 再定义少量高语义事件
- 不把底层调度细节直接裸露给前端

### 4. 删除“只靠 summary 解释执行过程”的默认体验

本次明确要求：

- 必须有 `ThoughtChain` 展示阶段过程
- `summary` 只负责最终答案

## 建议目录结构

```text
internal/ai/
├── gateway_contract.go
├── orchestrator.go
├── config.go
├── events/
├── runtime/
├── rewrite/
├── planner/
├── executor/
├── experts/
└── summarizer/
```

说明：

- `gateway_contract.go` 负责 `RunRequest / ResumeRequest / StreamEvent`
- `events/` 负责前后端事件 schema
- `runtime/` 负责 `ExecutionState / StepState / Resume`

## 迁移策略

本次变更按依赖关系推进，而不是沿用原提案的“大而全周计划”。

### Phase A: Core Boundary

- 重定义 AI route 与 AI core 的边界
- 定义 `RunRequest / ResumeRequest / StreamEvent`
- 在 `internal/ai` 建立稳定 orchestrator host
- 建立新链路 rollout 开关与回滚路径

### Phase B: Rewrite + Planner

- 引入 Rewrite 阶段
- 打通 Rewrite 到 Planner 的输入契约
- 统一资源解析、澄清、权限预检查

### Phase C: Executor Runtime + Experts

- 实现 step 状态机
- 管理审批、恢复、超时、重试
- 以 Agent Tool 方式挂载 Experts

### Phase D: Frontend ThoughtChain Contract

- 收敛 SSE 事件 schema
- 设计 `ThoughtChain` 数据模型
- 打通阶段可视化与 summary 输出

### Phase E: Summarization + Replan

- 补齐 Summarizer 输出契约
- 处理 need-more-investigation / replan

## 成功标准

### 后端

- AI core 成为唯一稳定编排边界
- `Rewrite -> Planner -> Executor Runtime -> Experts -> Summarizer` 链路成立
- 审批与恢复基于统一运行时状态
- Eino `v0.8.0` 新特性被明确用于合适边界

### 前端

- AI 面板使用 `ThoughtChain` 展示阶段过程
- 用户能看到每个阶段的可理解内容
- `summary` 独立输出最终结论
- 复杂任务中，用户能够感知 AI 正在持续工作

### 体验

- 用户能理解 AI 正在做什么
- 复杂排查中的等待成本更可接受
- 前后端围绕统一契约协作，而不是各自猜测运行时语义
- 新链路可以在灰度失败时安全回退
