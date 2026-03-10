# Design: Agentic Multi-Expert Architecture

## 1. 设计目标

本方案基于 Eino `v0.8.0` 的 ADK 新能力，重构 AI 执行链路为：

```text
Gateway / API
    -> AI Orchestrator Host
        -> Rewrite
        -> Planner
        -> Executor Runtime
            -> Expert Agents as Tools
        -> Summarizer
```

目标不是单纯“拆成更多 Agent”，而是形成一条同时满足以下条件的链路：

- 入口可以理解口语化输入
- 中间可以生成稳定计划
- 执行阶段保持运行时确定性
- 前端可以感知 AI 正在思考和工作
- 用户最终可以得到清晰 `summary`

## 2. 关键设计原则

### 2.1 半结构化双通道

所有关键阶段的输出采用半结构化协议：

- 一部分是机器可消费的结构化字段
- 一部分是用于解释和消歧的自然语言 `narrative`

约束：

- 执行语义以结构化字段为准
- `narrative` 用于解释、展示、下游协作和消歧
- 关键执行字段不得只存在于自然语言中

### 2.2 模型自治与运行时确定性分离

模型负责：

- 输入规整
- 资源理解
- 计划生成
- 专家内工具选择
- 最终总结

代码负责：

- DAG 调度
- step 状态机
- 审批与恢复
- 超时与重试
- 事件顺序与状态持久化

### 2.3 前端消费高层语义，而不是底层日志

前端以 `ThoughtChain + 正文回答 + summary` 为核心体验：

- `ThoughtChain` 展示阶段过程
- 正文回答展示最终用户可读输出，并支持流式 `delta`
- `summary` 展示最终结构化结论

前端不直接消费底层工具日志作为主体验。

## 3. Eino 0.8.0 的采用方式

本方案明确采用以下 `v0.8.0` 能力：

### 3.1 Agent Tool

使用 `adk.NewAgentTool(...)` 将 Expert Agent 暴露给 Executor Runtime 调用。

适用对象：

- HostOpsExpert
- K8sExpert
- ServiceExpert
- DeliveryExpert
- ObservabilityExpert

### 3.2 Transfer / SubAgent

使用 `Transfer / SetSubAgents / DeterministicTransfer` 表达 agent-friendly 的阶段切换。

推荐原则：

- `Rewrite -> Planner` 适合使用 transfer
- `Summarizer` 适合作为执行结束后的总结阶段
- `Executor Runtime` 的进入与退出优先由 Orchestrator Host 控制

说明：

- Transfer 适合表达阶段 handoff
- 不用于替代 Executor Runtime 的确定性调度职责
- 不应通过“Planner 直接 transfer 到 Summarizer”绕过执行层

### 3.3 Interrupt / Resume

使用 ADK 的 interrupt / resume 语义承接审批和恢复流程，但对平台对外接口收敛为 plan-step 级恢复模型。

补充约束：

- Resume 必须具备幂等语义
- 同一 `plan_id + step_id + approval_decision` 不得重复触发写操作
- 对 mutating step 必须提供去重键、等价 request_id 或同等级别的重复保护

### 3.4 Middleware

使用 ADK middleware 支撑：

- 输入规整
- 结果压缩
- 事件包装
- 总结增强

## 4. 系统分层

## 4.1 Gateway / Route

职责：

- HTTP / SSE transport
- 请求映射
- auth
- session shell

不负责：

- AI 编排
- 规划
- 审批语义
- step 状态机

## 4.2 AI Orchestrator Host

`internal/ai` 提供唯一稳定入口。

职责：

- 接收 `RunRequest / ResumeRequest`
- 初始化 `RuntimeContext`
- 串联 Rewrite / Planner / Executor Runtime / Summarizer
- 统一 trace / session / execution lifecycle
- 输出面向前端的高层事件

## 4.3 Rewrite

职责：

- 将用户口语化输入规整为稳定任务表达
- 提取资源 hint
- 判断任务模式
- 输出供 Planner 消费的半结构化结果

不负责：

- resolve 最终资源 ID
- 权限检查
- 生成最终 ExecutionPlan
- 调用 mutating tool

## 4.4 Planner

职责：

- 资源解析
- 权限预检查
- 澄清歧义
- 输出结构化 `ExecutionPlan`

## 4.5 Executor Runtime

职责：

- 执行 `ExecutionPlan`
- 按 step 依赖调度
- 调用 Expert Agent Tools
- 处理审批、恢复、超时、重试
- 维护 `ExecutionState`

## 4.6 Experts

每个专家只持有本领域工具。

公共原则：

- 不暴露 Planner support tools
- 不直接承担全局调度职责
- 通过 Agent Tool 接入

## 4.7 Summarizer

职责：

- 汇总步骤结果与证据
- 形成用户可读 `summary`
- 判断是否需要补充调查

## 5. 运行时主流程

```text
User Input
   |
   v
Rewrite
   |
   v
Planner
   |---- clarify / reject / direct_reply
   |
   v
ExecutionPlan
   |
   v
Executor Runtime
   |
   +--> Expert Agent Tool
   +--> Expert Agent Tool
   +--> Expert Agent Tool
   |
   v
Summarizer
   |
   +--> summary
   +--> need_more_investigation?
```

流程分支：

1. `Rewrite` 完成后进入 `Planner`
2. `Planner` 可能直接：
   - `clarify`
   - `reject`
   - `direct_reply`
   - `plan`
3. `plan` 时进入 `Executor Runtime`
4. `Executor Runtime` 执行 steps，必要时进入 `waiting_approval`
5. 所有可执行步骤完成后进入 `Summarizer`
6. `Summarizer` 输出：
   - 最终 `summary`
   - 或提示需要补充调查

## 6. 输入输出契约

## 6.1 RunRequest

```ts
type RunRequest = {
  session_id?: string;
  message: string;
  runtime_context?: RuntimeContext;
};
```

## 6.2 RuntimeContext

```ts
type RuntimeContext = {
  scene?: string;
  route?: string;
  project_id?: string;
  current_page?: string;
  selected_resources?: Array<{
    type: string;
    id?: string;
    name?: string;
  }>;
  user_context?: Record<string, unknown>;
};
```

约束：

- 由 Gateway 生成标准结构
- 不直接透传原始前端 payload

## 6.3 RewriteOutput

```ts
type RewriteOutput = {
  normalized_goal: string;
  operation_mode: "query" | "investigate" | "mutate";
  resource_hints?: {
    service_name?: string;
    cluster_name?: string;
    host_name?: string;
    namespace?: string;
  };
  domain_hints?: Array<"service" | "k8s" | "hostops" | "delivery" | "observability">;
  ambiguity_flags?: string[];
  narrative: string;
};
```

说明：

- `normalized_goal` 供 Planner 稳定消费
- `narrative` 解释字段组合的真实语义，避免歧义

示例：

```json
{
  "normalized_goal": "排查 payment-api 响应变慢，并核对近期发布是否相关",
  "operation_mode": "investigate",
  "resource_hints": {
    "service_name": "payment-api"
  },
  "domain_hints": ["service", "observability", "delivery"],
  "ambiguity_flags": [],
  "narrative": "用户希望确认 payment-api 当前是否出现响应变慢，并结合监控与近期发布判断根因。本轮任务仅调查，不执行变更。"
}
```

## 6.4 PlannerDecision

```ts
type PlannerDecision =
  | { type: "clarify"; message: string; candidates?: Array<Record<string, unknown>>; narrative: string }
  | { type: "reject"; reason: string; narrative: string }
  | { type: "direct_reply"; message: string; narrative: string }
  | { type: "plan"; plan: ExecutionPlan; narrative: string };
```

## 6.5 ExecutionPlan

```ts
type ExecutionPlan = {
  plan_id: string;
  goal: string;
  resolved: ResolvedResources;
  narrative: string;
  steps: PlanStep[];
};

type ResolvedResources = {
  service_id?: number;
  service_name?: string;
  cluster_id?: number;
  cluster_name?: string;
  host_ids?: number[];
  namespace?: string;
};

type PlanStep = {
  step_id: string;
  title: string;
  expert: "hostops" | "k8s" | "service" | "delivery" | "observability";
  intent: string;
  task: string;
  input?: Record<string, unknown>;
  depends_on?: string[];
  mode: "readonly" | "mutating";
  risk: "low" | "medium" | "high";
  narrative?: string;
};
```

约束：

- `step_id` 稳定
- `input` 面向机器
- `task / narrative` 面向解释和消歧

## 6.6 StepResult

```ts
type StepResult = {
  step_id: string;
  status: "completed" | "failed" | "blocked" | "waiting_approval" | "cancelled";
  summary?: string;
  evidence?: Array<Record<string, unknown>>;
  error?: {
    code?: string;
    message: string;
  };
};
```

## 6.7 SummaryOutput

```ts
type SummaryOutput = {
  summary: string;
  conclusion?: string;
  next_actions?: string[];
  need_more_investigation: boolean;
  narrative: string;
};
```

说明：

- `summary` 用于结构化结论视图
- 最终用户可读正文仍由 `delta` 流输出承载

## 7. Executor Runtime 设计

## 7.1 Step 状态机

初版状态机：

```text
pending
ready
running
waiting_approval
completed
failed
blocked
cancelled
```

状态流转：

```text
pending -> ready -> running -> completed
running -> waiting_approval
waiting_approval -> ready
running -> failed
failed(upstream) -> blocked(downstream)
running -> cancelled
```

## 7.2 状态语义

- `pending`: 尚未满足依赖
- `ready`: 已满足依赖，可执行
- `running`: 正在执行
- `waiting_approval`: 命中审批，等待用户确认
- `completed`: 成功完成
- `failed`: 执行失败
- `blocked`: 因上游失败而不可继续
- `cancelled`: 用户取消或超时终止

## 7.3 调度原则

- 无依赖 steps 可并行
- 有依赖 steps 串行等待
- Executor 不直接调用底层平台工具
- Executor 只调用 Expert Agent Tool

## 7.4 恢复原则

审批恢复按 `plan_id + step_id` 进行，不再以模糊 interrupt target 作为前端主模型。

```ts
type ResumeRequest = {
  session_id: string;
  plan_id: string;
  step_id: string;
  approved: boolean;
  reason?: string;
};
```

约束：

- 仅恢复对应 step
- 不重跑已完成 step
- 被上游失败阻断的 step 不自动恢复
- 同一恢复请求重复到达时不得重复执行副作用

## 8. Expert Agent 接入方式

每个 Expert 作为单独 Agent，并通过 `NewAgentTool(...)` 接入。

```text
Executor Runtime
   -> ServiceExpertTool
   -> ObservabilityExpertTool
   -> DeliveryExpertTool
```

Expert 设计原则：

- 内部自行选择本领域工具
- 不暴露 Planner support tools
- 不承担全局调度
- 输出统一 `StepResult` 风格摘要

## 9. 前端 ThoughtChain 方案

前端新增：

```tsx
import { ThoughtChain } from "@ant-design/x";
```

## 9.1 页面结构

```text
AI Panel
├── ThoughtChain
│   ├── rewrite
│   ├── plan
│   ├── execute
│   ├── user_action
│   └── summary
├── Assistant Answer
└── Summary View
```

说明：

- `ThoughtChain` 展示阶段过程
- `Assistant Answer` 展示最终用户可读正文，并支持流式增量
- `Summary View` 展示最终结构化结论

## 9.2 阶段设计

| Key | 标题 | 展示内容 |
|-----|------|----------|
| `rewrite` | 理解你的问题 | Rewrite 后的任务表达、资源线索、模式识别 |
| `plan` | 整理排查计划 | 已解析资源、专家选择、步骤摘要 |
| `execute` | 调用专家执行 | 专家进度、关键发现、阶段结果 |
| `user_action` | 等待你处理 | 澄清或审批动作 |
| `summary` | 生成结论 | 总结摘要 |

## 9.3 ThoughtChain 数据模型

```ts
type ThoughtStageKey = "rewrite" | "plan" | "execute" | "user_action" | "summary";

type ThoughtStageStatus = "loading" | "success" | "error" | "abort";

type ThoughtStageItem = {
  key: ThoughtStageKey;
  title: string;
  description?: React.ReactNode;
  content?: React.ReactNode;
  footer?: React.ReactNode;
  status?: ThoughtStageStatus;
  collapsible?: boolean;
  blink?: boolean;
};
```

## 9.4 展示原则

- 展示用户能理解的进展
- 不直接倾倒 tool JSON
- `execute` 优先展示专家摘要，不直接展示全部工具级噪音
- `summary` 最终仍以独立正文输出
- `clarify` 与 `approval` 可以共用一个阶段容器，但 MUST 在标题、说明、操作和恢复语义上明确区分

## 10. SSE / 事件语义

前端消费高层阶段事件，同时保留最终正文流式通道。

## 10.1 建议事件类型

```text
meta
rewrite_result
planner_state
plan_created
step_update
approval_required
clarify_required
replan_started
delta
summary
done
error
```

## 10.2 EventMeta

```ts
type EventMeta = {
  session_id: string;
  trace_id: string;
  plan_id?: string;
  iteration?: number;
  timestamp: string;
};
```

## 10.3 rewrite_result

```ts
type RewriteResultEvent = EventMeta & {
  rewrite: RewriteOutput;
  user_visible_summary: string;
};
```

## 10.4 plan_created

```ts
type PlanCreatedEvent = EventMeta & {
  plan: ExecutionPlan;
  user_visible_summary: string;
};
```

## 10.5 step_update

对前端统一使用 `step_update`，而不是同时暴露 `step_start / step_result / expert_progress`。

```ts
type StepUpdateEvent = EventMeta & {
  step_id: string;
  expert: string;
  status: "ready" | "running" | "waiting_approval" | "completed" | "failed" | "blocked";
  title?: string;
  user_visible_summary?: string;
};
```

## 10.6 approval_required / clarify_required

```ts
type ApprovalRequiredEvent = EventMeta & {
  plan_id: string;
  step_id: string;
  title: string;
  risk: "medium" | "high";
  action_summary: string;
  preview?: Record<string, unknown>;
};
```

```ts
type ClarifyRequiredEvent = EventMeta & {
  title: string;
  message: string;
  candidates?: Array<Record<string, unknown>>;
  kind: "clarify";
};
```

约束：

- `approval_required` 表示“信息足够，但执行前需授权”
- `clarify_required` 表示“信息不足，需用户补充”
- 二者可以映射到同一 `user_action` 阶段，但前端标题、说明、CTA、恢复语义 MUST 不同

## 10.7 delta

最终正文回答仍支持流式输出。

```ts
type DeltaEvent = EventMeta & {
  content_chunk: string;
};
```

约束：

- `delta` 用于最终用户可读正文流
- `summary` 用于结构化结论
- `summary` MUST NOT 取代最终正文输出能力

## 10.8 summary

```ts
type SummaryEvent = EventMeta & {
  output: SummaryOutput;
};
```

## 10.9 replan_started

当系统准备进入新一轮规划时，应发出显式高层事件。

```ts
type ReplanStartedEvent = EventMeta & {
  reason?: string;
  previous_plan_id?: string;
};
```

说明：

- 用于让前端感知“当前进入了新一轮规划”
- 不应由前端通过覆盖旧 ThoughtChain 状态来猜测是否发生了重规划

## 11. 目录建议

```text
internal/ai/
├── gateway_contract.go
├── orchestrator.go
├── config.go
├── events/
├── runtime/
│   ├── execution_state.go
│   ├── step_state.go
│   └── resume.go
├── rewrite/
├── planner/
├── executor/
├── experts/
└── summarizer/
```

## 12. 风险与决策

### 12.1 为什么不用“所有层都是 Agent”

因为 `Executor` 承担的是运行时职责，强行做成高度自治 Agent 会削弱确定性。

### 12.2 为什么采用半结构化协议

因为纯结构化容易缺语义，纯自然语言又不稳定。半结构化最适合跨 Rewrite / Planner / Expert / Frontend 的长链路协作。

### 12.3 为什么前端要用 ThoughtChain

因为复杂任务里，用户需要知道 AI 正在哪一阶段工作，才能接受更长的等待。

### 12.4 为什么要增加模型约束

因为这条链路里存在多个模型参与阶段：

- Rewrite
- Planner
- Experts
- Summarizer

如果缺少明确约束，最容易出现的偏移包括：

- Rewrite 擅自补出并不存在的资源解析结果
- Planner 在资源未唯一确定时继续生成执行计划
- Expert 越过领域边界调用不该使用的工具
- Summarizer 将推断误写为事实

因此需要在 prompt、结构化输出校验和运行时规则三层同时约束。

### 12.5 为什么要增加灰度与回滚约束

因为新方案同时改动：

- 后端入口边界
- 多阶段模型执行
- Resume 语义
- 前端 ThoughtChain 协议

如果缺少 rollout / rollback 机制，任何单点问题都可能影响整条链路。

因此新链路应支持：

- feature flag
- 灰度启用
- 兼容旧链路
- 明确回滚条件

## 13. 模型执行约束

### 13.1 Rewrite Guardrails

- Rewrite 只做语言规整和 hint 抽取
- Rewrite MUST NOT 伪造资源 ID、权限结论、执行结论
- Rewrite 在不确定时保留 `ambiguity_flags`
- Rewrite 不负责消解高风险歧义
- 应定义“弱 hint 可继续 / 强歧义必须 clarify”的规则

### 13.2 Planner Guardrails

- Planner 在 unresolved / ambiguous 资源场景 MUST 输出 `clarify`
- Planner MUST 用结构化字段表达 `mode/risk/depends_on`
- Planner MUST NOT 通过 `narrative` 隐式塞入执行要求
- Planner MUST NOT 持有 mutating tool

### 13.3 Expert Guardrails

- Expert MUST 只调用本领域工具
- Expert MUST NOT 代替 Planner 做全局资源解析
- Expert MUST 输出观察结果和结论摘要
- Expert SHOULD 区分“工具直接观测结果”和“基于结果的判断”

### 13.4 Summarizer Guardrails

- Summarizer MUST 基于 `StepResult / Evidence` 生成结论
- Summarizer MUST 标记不确定性
- 证据不足时 MUST 输出 `need_more_investigation=true`
- Summarizer MUST NOT 把推断写成已证实事实

## 14. 落地风险补充

### 14.1 ExecutionState 持久化

当前方案已把 `ExecutionState` 作为关键前提，因此实现前必须明确：

- 持久化介质
- 落盘时机
- 重启恢复边界
- 与事件流的对齐关系

### 14.2 Resume 幂等

恢复请求一旦缺乏幂等保护，会直接放大 mutating step 的重复执行风险。

### 14.3 灰度与回滚

新链路必须先灰度，再切主，不宜一次性替换全部旧路径。

## 15. 需要继续细化的点

后续设计可继续细化：

- Rewrite 的 prompt 和评估标准
- Planner 的 resolve 评分模型
- Expert 输入输出包装
- ExecutionState 的持久化存储
- ThoughtChain 的具体 UI 交互细节
- rollout / rollback 的执行策略
