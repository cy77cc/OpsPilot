## Context

当前 `internal/ai` 运行时基于多阶段控制平面：`rewrite -> planner -> executor(expert agent) -> summarizer`。这条链路在治理、专家隔离和阶段化可视化上有优势，但其代价也很明显：

- 一次请求通常需要多次模型调用，导致端到端延迟高。
- `executor -> expert tool -> expert agent -> tools` 引入了额外的模型跳转，不利于学习单 agent tool-calling 的核心机制。
- human-in-the-loop 目前更多依赖业务控制流显式插入审批阶段，而不是直接利用 Eino ADK 的 `Interrupt / ResumeWithParams / CheckPointStore`。
- 前端事件、会话存储和审批 UI 已经相对成熟，后端运行时是当前最适合重新设计的部分。

项目已经使用 `github.com/cloudwego/eino v0.8.0`，具备 `ChatModelAgent`、`Runner`、`ChatModelAgentMiddleware`、`Interrupt / ResumeWithParams` 等能力，因此可以引入与官方范式更一致的 `aiv2` 单 agent runtime。

## Goals / Non-Goals

**Goals:**
- 新增 `internal/aiv2`，实现单 `ChatModelAgent + Runner` 的 AI runtime。
- 复用现有平台工具能力，但移除 `expert agent as tool` 这一层模型调用。
- 基于 Eino HITL 构建审批中断与恢复，批准对象是“具体 pending tool call”，而不是重新规划整条链路。
- 使用 `ChatModelAgentMiddleware` 统一处理上下文注入、审批策略、流式事件投影和可观测性。
- 保持现有前端 SSE/turn-block 协议尽量兼容，让聊天 UI 可以在不重写的前提下接入 `aiv2`。
- 保留 `internal/ai` 作为兼容运行时，通过配置切换或桥接方式逐步引入 `aiv2`。

**Non-Goals:**
- 不在首版移除现有 `internal/ai` 控制平面。
- 不在首版重构所有前端抽屉组件或历史会话模型。
- 不在首版引入多 agent / supervisor / expert-as-tool 的新层次。
- 不强制所有旧会话迁移为 `aiv2` 专属存储格式；首版以兼容回放为主。

## Decisions

### 1. 新增并行 `internal/aiv2` 运行时，而不是原地重写 `internal/ai`

**Decision:** 以 `internal/aiv2` 作为并行新模块实现单 agent runtime，旧 `internal/ai` 继续保留。

**Why:**
- `internal/ai` 的文件命名、状态机和运行时语义已经深度绑定多阶段控制平面。
- 原地改造会造成“文件名还是 planner/executor，行为却已经不是 planner/executor”的语义混乱。
- 并行新 runtime 更适合学习、灰度和回退。

**Alternatives considered:**
- 原地重构 `internal/ai`：改动面小，但概念污染严重，不利于长期维护。
- 完全替换旧 runtime：风险高，不利于对比与回滚。

### 2. `aiv2` 使用单 `ChatModelAgent + Runner` 作为唯一模型执行宿主

**Decision:** `aiv2` 只保留一个主 `ChatModelAgent`，通过统一工具注册表完成查询、变更和最终回答，不再保留 rewrite/planner/executor/summarizer 分段模型。

**Why:**
- 单 agent 路径更适合学习 ReAct tool-calling 核心机制。
- 可以直接减少多次模型调用带来的延迟。
- 与 Eino v0.8 HITL / middleware 的设计更一致。

**Alternatives considered:**
- 保留 rewrite 或 summarizer 两段式：延迟依然较高，且会继续引入多模型语义。
- 使用多个子 agent：适合复杂控制平面，不适合学习优先和低延迟优先的目标。

### 3. 复用现有 tools，但不复用 `ExpertAgent` 包装层

**Decision:** `aiv2` 复用现有 host/k8s/service/delivery/observability tools 与依赖注入方式，但不再通过 `executor -> expert tool -> expert agent` 间接调用。

**Why:**
- tools 是当前最有价值、最稳定的能力资产。
- `ExpertAgent` 是额外模型调用开销的主要来源之一。
- 单 agent runtime 需要的是“统一工具集”，而不是“专家再代理工具”。

**Alternatives considered:**
- 完全复制并重写工具：成本高，容易产生行为漂移。
- 保留 expert agent：会抵消单 agent runtime 的核心收益。

### 4. human-in-the-loop 基于 Eino 原生 `Interrupt / ResumeWithParams`

**Decision:** mutating tool 的审批能力基于 `Runner` 的 checkpoint/resume 机制实现，pending action 的粒度是“具体 tool invocation”，不是“模糊的后续行为”。

**Why:**
- 这与 Eino ADK HITL 官方能力模型一致。
- 批准后可以恢复同一次运行，而不是重新发起一轮新请求。
- 批准的是确定的动作，更稳定、可审计，也避免重新让模型自由决定。

**Alternatives considered:**
- 在业务流程中显式插审批阶段：更像当前控制平面，不符合单 agent 目标。
- 让 agent 自己决定何时停下来审批：不稳定，也不利于审计。

### 5. `ChatModelAgentMiddleware` 作为 `aiv2` 的横切能力骨架

**Decision:** `aiv2` 首版即引入 middleware 链，至少包含：
- `ContextInjectMiddleware`
- `ApprovalPolicyMiddleware`
- `StreamingProjectorMiddleware`
- `ObservabilityMiddleware`

**Why:**
- 避免新的 runtime 再次膨胀成手写 orchestrator。
- 审批、事件投影、上下文注入和可观测性本质上是横切逻辑。
- middleware 更符合 Eino 0.8 的推荐扩展方式。

**Alternatives considered:**
- 在 runtime 主流程里硬编码所有逻辑：短期快，长期会重新长成新的复杂 orchestrator。

### 6. handler 和前端协议保持兼容，runtime 通过模式切换接入

**Decision:** 保持 `/api/v1/ai/chat` 与 `/api/v1/ai/resume/step/stream` 作为统一入口，handler 内部根据配置或请求模式将流量路由到 `internal/ai` 或 `internal/aiv2`。

**Why:**
- 前端不需要为 `aiv2` 重写整套调用层。
- 可以保留现有 SSE/turn-block 协议，并逐步优化 `aiv2` 的事件内容。
- 灰度与回滚路径明确。

**Alternatives considered:**
- 新开 `/api/v2/ai/*`：边界清晰，但前端和权限链都要重复接入，首版迁移成本高。

### 7. 首版存储策略采用“双层状态”：Eino checkpoint + 现有会话/执行兼容存储

**Decision:** `aiv2` 首版允许同时使用：
- Eino `CheckPointStore` 处理 agent interrupt/resume
- 现有会话存储 / 兼容执行态存储维持前端历史回放、审批查询和运营可见性

**Why:**
- 直接完全切到纯 checkpoint store 会让前端历史会话、审批卡、turn replay 一起重做。
- 首版的重点是跑通单 agent runtime，不是彻底推翻存储模型。

**Alternatives considered:**
- 纯 Eino checkpoint 驱动全部状态：最“原生”，但迁移跨度过大。

## Risks / Trade-offs

- **[风险] 单 agent 暴露统一工具集后，模型的工具选择空间更大，误用工具概率上升。**  
  → 通过工具 policy metadata、approval gate、scene/context middleware 和严格 tool 描述降低误用概率。

- **[风险] 首版双 runtime 并存会增加维护成本。**  
  → 通过配置开关、共享 handler 契约和共享工具注册层减少分叉；将 `aiv2` 定位为默认学习/实验模式，旧 runtime 保持兼容。

- **[风险] checkpoint 与现有 execution/session state 双写可能产生状态漂移。**  
  → 先限定 `aiv2` 的关键状态边界：checkpoint 是运行事实源，兼容 execution/session store 只负责 UI 和查询；避免双向回写。

- **[风险] 前端虽然可复用，但执行过程语义会从“专家执行”切到“工具调用链”，存在展示差异。**  
  → 保持现有 SSE 事件分类兼容，并在 `aiv2` 中明确输出 tool-centric payload，让前端逐步收敛。

- **[风险] 不再显式分 rewrite/planner/summarizer 后，某些结构化计划和解释性输出可能弱化。**  
  → 通过更强的主 agent prompt、middleware 注入上下文、必要时引入可选“structured plan block”而非强制多阶段模型来补足。

## Migration Plan

1. 在 `internal/aiv2` 中实现最小单 agent readonly runtime，先验证查询型任务、流式输出和前端兼容。
2. 引入统一工具注册层，将现有 tools 按域复用到 `aiv2`。
3. 为 mutating tools 增加 approval-aware wrapper，并接入 `Interrupt / ResumeWithParams`。
4. 在 `internal/service/ai` handler 中增加 runtime 分流，支持配置切换到 `aiv2`。
5. 让前端继续复用现有 SSE 协议，并将执行态展示逐步收敛为 tool-call centric。
6. 观察稳定性和延迟收益后，再决定是否让 `aiv2` 成为默认 runtime，并逐步收缩旧 `internal/ai` 的职责。

**Rollback:**  
通过配置或 runtime selector 直接切回现有 `internal/ai` 路径；由于 handler 契约和前端协议保持兼容，回滚不要求前端配套变更。

## Open Questions

- `aiv2` 首版的 runtime 选择是仅用配置开关，还是允许按请求维度切换？
- Eino `CheckPointStore` 与现有 Redis execution state 是否共享底层存储，还是逻辑隔离更安全？
- 首版是否需要保留某种轻量 structured plan 输出，还是完全依赖单 agent 的工具链与最终回答？
- 工具 policy metadata 是直接附着在工具注册层，还是先以外部规则表维护更稳妥？
