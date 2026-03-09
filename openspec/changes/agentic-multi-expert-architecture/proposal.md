# Proposal: Agentic Multi-Expert Architecture

## 概述

基于 Eino ADK 的 `Transfer` + `AgentAsTool` 特性，重构 AI 模块为四层 Agentic 架构：Planner → Executor → Expert → Summarizer，实现多领域专家协作的智能运维助手。

## 动机

### 当前问题

1. **单体 Agent 负担过重**: 现有 `react.Agent` 扁平注册 50+ 工具，所有领域工具混在一起，导致：
   - 推理效率低，模型需要从大量工具中选择
   - 领域边界模糊，难以针对性优化 prompt
   - 工具冲突风险增加

2. **资源解析分散**: 用户说"检查 payment-api"，需要专家自己去查询 service_id，造成：
   - 多个专家重复查询
   - 用户自然语言理解不一致
   - 缺乏统一的权限预检查

3. **缺乏规划能力**: 当前 Agent 直接执行工具调用，没有显式的计划制定和调整机制

4. **结果汇总不完整**: 执行完多个工具后，缺乏统一的汇总和判断机制

### 目标状态

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    四层 Agentic Multi-Expert 架构                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Planner Agent ──► Executor Agent ──► Domain Experts ──► Summarizer Agent │
│                                                                             │
│   - Planner: 意图理解 + 资源解析 + 权限检查 + 计划制定                        │
│   - Executor: 计划执行 + 依赖调度 + 审批恢复 + 结果收集                        │
│   - Experts: 领域任务执行 (HostOps/K8s/Service/Delivery/Observability)       │
│   - Summarizer: 结果汇总 + 完整性判断                                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                            Planner Agent                             │  │
│   │                                                                      │  │
│   │   输入: 用户请求 + 运行时上下文                                       │  │
│   │                                                                      │  │
│   │   Tools:                                                             │  │
│   │   ├── resolve_service      # 服务名称 → service_id                  │  │
│   │   ├── resolve_cluster      # 集群名称 → cluster_id                  │  │
│   │   ├── resolve_host         # 主机名称 → host_id                     │  │
│   │   ├── check_permission     # 权限预检查                             │  │
│   │   └── get_user_context     # 获取用户上下文                         │  │
│   │                                                                      │  │
│   │   输出:                                                              │  │
│   │   - 需要澄清 → 输出结构化 PlannerDecision(type=clarify)               │  │
│   │   - 可以执行 → 输出结构化 PlannerDecision(type=plan)                  │  │
│   │                                                                      │  │
│   └────────────────────────────────┬────────────────────────────────────┘  │
│                                    │                                        │
│                                    │ 结构化计划                             │
│                                    ▼                                        │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                           Executor Agent                             │  │
│   │                                                                      │  │
│   │   职责:                                                              │  │
│   │   - 执行结构化计划与 DAG 调度                                         │  │
│   │   - 调度专家: 无依赖并行，有依赖串行                                  │  │
│   │   - 错误重试 (max_retry=3)                                           │  │
│   │   - 审批等待与 Resume 恢复                                            │  │
│   │                                                                      │  │
│   │   无平台 Tools，通过 AgentAsTool 调用专家                             │  │
│   │                                                                      │  │
│   └────────────────────────────────┬────────────────────────────────────┘  │
│                                    │                                        │
│            ┌───────────────────────┼───────────────────────┐               │
│            │                       │                       │               │
│            ▼             ▼              ▼              ▼             ▼     │
│   ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌────────────┐ ┌────┐│
│   │HostOpsExpert │ │  K8sExpert   │ │ServiceExpert │ │DeliveryExp │ │Obs ││
│   │ os_*/host_*  │ │    k8s_*     │ │service_*/    │ │ cicd_*/    │ │mon ││
│   │              │ │              │ │deployment_*/ │ │ job_*      │ │top ││
│   │              │ │              │ │credential_*/ │ │            │ │aud ││
│   │              │ │              │ │config_*      │ │            │ │    ││
│   └──────────────┘ └──────────────┘ └──────────────┘ └────────────┘ └────┘│
│                                    │                                        │
│                                    ▼                                        │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                          Summarizer Agent                            │  │
│   │                                                                      │  │
│   │   职责:                                                              │  │
│   │   - 汇总各专家结果                                                   │  │
│   │   - 判断完整性，是否需要补充调查                                      │  │
│   │   - 输出最终回答 (流式)                                              │  │
│   │                                                                      │  │
│   │   需要补充 → 回到 Planner (受 max_iterations 限制)                    │  │
│   │                                                                      │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 各层职责

#### 1. Planner Agent

**核心职责**:
- 理解用户意图
- 解析资源 ID（用户说"payment-api" → 查到 service_id=123）
- 权限预检查
- 制定执行计划

**工具集**:

| 工具 | 说明 |
|------|------|
| `resolve_service` | 根据服务名称/关键词解析服务ID，支持模糊匹配 |
| `resolve_cluster` | 根据集群名称/环境解析集群ID |
| `resolve_host` | 根据主机名/IP解析主机ID |
| `check_permission` | 检查当前用户对指定资源的操作权限 |
| `get_user_context` | 获取用户当前上下文信息（当前页面、选中的资源等） |

**输出模式**:

- 面向系统：输出结构化 `PlannerDecision`
- 面向用户：保留流式自然语言 `delta`

Planner 不再依赖自然语言正文驱动执行，真正进入 Executor 的必须是结构化 `ExecutionPlan`。

**需要澄清时的直接回复**:

```
用户: "检查 payment 服务"
Planner: "找到多个匹配的服务，请选择：
          1. payment-api (生产环境)
          2. payment-gateway (生产环境)
          3. payment-worker (测试环境)
          请回复序号或完整名称。"
```

#### 2. Executor Agent

**核心职责**:
- 解析计划步骤，提取专家调用
- 分析依赖关系 (depends_on)
- 调度执行：无依赖并行，有依赖串行
- 收集结果，错误重试

**无平台工具**，通过 AgentAsTool 调用专家

**调度策略**:

```
Step 1: ServiceExpert (无依赖)          → 执行
Step 2: K8sExpert (依赖 Step 1)         → 等待 Step 1 完成后执行
Step 3: ObservabilityExpert (依赖 Step 1) → 等待 Step 1 完成后执行

执行顺序:
Step 1 → [Step 2, Step 3 并行]
```

补充约束：

- Executor 使用稳定 `step_id` 管理依赖、状态、事件和恢复
- 进入审批的步骤持久化为 `waiting_approval`
- `Resume(...)` 只恢复被阻断的单个 step，不重跑已完成步骤

#### 3. Domain Expert Agents

**HostOpsExpert** - 主机运维专家:
- CPU/内存/磁盘监控
- 进程管理
- 系统日志查询
- 网络状态检查
- 只读命令与批量变更预览
- 批量执行与状态变更（需 review/approval）

**K8sExpert** - Kubernetes 专家:
- Pod/Deployment/Service 查询
- 容器日志获取
- 事件分析
- 资源状态检查

**ServiceExpert** - 服务管理专家:
- 服务状态查询
- 服务部署预览与执行
- 服务目录查询
- 部署目标、凭证与配置差异分析

**DeliveryExpert** - 交付专家:
- CI/CD 流水线查询和触发
- Job 执行状态和运行
- 发布链路问题分析

**ObservabilityExpert** - 观测专家:
- 指标查询和分析
- 告警规则和状态
- 服务拓扑分析
- 审计日志分析

#### 4. Summarizer Agent

**核心职责**:
- 汇总各专家结果
- 判断完整性，是否需要补充调查
- 输出最终回答（流式）

**判断逻辑**:

```go
type SummarizerDecision struct {
    NeedMoreInvestigation bool     // 是否需要补充调查
    NextSteps             string   // 如果需要，下一步调查什么
    CurrentConclusion     string   // 当前结论
    Confidence            float64  // 置信度
}
```

如果 `NeedMoreInvestigation=true` 且没超过最大迭代次数，回到 Planner 重新规划。

补充约束：

- Summarizer 只消费 `Planner/Executor/Expert` 的结构化结果
- Summarizer 不直接回查平台工具
- Summarizer 必须明确缺失事实和重规划提示，而不只给一个布尔值

### 执行控制

| 参数 | 值 | 说明 |
|------|-----|------|
| `max_retry` | 3 | 单个步骤最大重试次数 |
| `max_iterations` | 5 | Planner → Summarizer 最大循环次数 |
| `expert_timeout` | 60s | 单个专家超时 |
| `total_timeout` | 300s | 总超时 |

### SSE 事件流

```
event: meta
data: {"session_id":"sess-xxx","trace_id":"trace-xxx","iteration":1}

event: delta
data: {"contentChunk":"我来帮你排查 payment-api 响应慢的问题..."}  # Planner 流式输出

event: planner_state
data: {"state":"planning","session_id":"sess-xxx","trace_id":"trace-xxx","iteration":1}

event: plan_created
data: {"plan_id":"plan-xxx","steps":[...], "resolved":{...}}   # 计划创建

event: step_start
data: {"step_id":"step-1", "expert":"ServiceExpert"}

event: expert_progress
data: {"step_id":"step-1","expert":"ServiceExpert", "status":"running"}

event: expert_progress
data: {"step_id":"step-1","expert":"ServiceExpert", "status":"done"}

event: step_result
data: {"step_id":"step-1","ok":true,"summary":"..."}

event: step_start
data: {"step_id":"step-2", "expert":"K8sExpert"}

event: step_start
data: {"step_id":"step-3", "expert":"ObservabilityExpert"}    # 并行

event: approval_required
data: {"step_id":"step-4","risk":"high","scope":"production"} # 如命中审批

event: delta
data: {"contentChunk":"## 排查结果\n"}                           # Summarizer 流式输出

event: done
data: {"stream_state":"ok"}
```

## 目录结构

```
internal/ai/
├── planner/
│   ├── planner.go           # Planner Agent 实现
│   ├── prompt.go            # 系统提示词 (专家目录)
│   ├── tools.go             # Planner 工具装配
│   ├── resolve.go           # resolve_* / inventory 复用
│   ├── context.go           # get_user_context
│   └── permission.go        # check_permission
├── executor/
│   ├── executor.go          # Executor Agent 实现
│   └── scheduler.go         # 调度器 (并行/串行执行)
├── experts/
│   ├── registry.go          # 专家注册表
│   ├── shared/
│   │   └── types.go
│   ├── hostops/
│   │   ├── expert.go
│   │   └── prompt.go
│   ├── k8s/
│   │   ├── expert.go
│   │   └── prompt.go
│   ├── service/
│   │   ├── expert.go
│   │   └── prompt.go
│   ├── delivery/
│   │   ├── expert.go
│   │   └── prompt.go
│   └── observability/
│       ├── expert.go
│       └── prompt.go
├── summarizer/
│   ├── summarizer.go        # Summarizer Agent 实现
│   └── prompt.go
├── orchestrator.go          # 顶层编排入口
└── config.go                # 配置 (max_retry, max_iterations, timeout, rollout)
```

## 关键设计决策

| 决策点 | 选择 | 理由 |
|--------|------|------|
| 计划格式 | 结构化 `PlannerDecision/ExecutionPlan` + 流式自然语言 | 系统执行稳定，用户仍可见过程 |
| 执行模式 | 流式输出 | 提高用户体验，减少等待感 |
| 资源解析 | Planner 负责 | 统一入口，避免专家重复查询 |
| 权限检查 | Planner 预检查 | 提前拒绝无权限操作，减少无效执行 |
| 专家依赖 | Planner 标注，Executor 用 Go 代码调度 | 保持 DAG 执行确定性 |
| 错误处理 | max_retry=3，友好报错 | 平衡可靠性和用户体验 |
| 用户澄清 | Planner 直接回复 | 简化流程，提高响应速度 |
| 循环控制 | max_iterations=5 | 防止无限循环，控制成本 |
| 高风险变更 | Executor/tool middleware 审批收口 | 避免审批逻辑散落在 prompt |

## 迁移策略

本次迁移采用“先双轨、后切主、最后删除”的方式，而不是直接在旧链路上原地重写。

### Phase 1: 基础框架 (Week 1)

1. 创建目录结构
2. 实现 Planner Agent 基础框架
3. 实现 resolve_* 工具
4. 实现计划解析器

### Phase 2: 专家迁移 (Week 2)

1. 迁移 Host 相关工具到 HostOpsExpert
2. 迁移 K8s 相关工具到 K8sExpert
3. 迁移 Service 相关工具到 ServiceExpert
4. 迁移 Delivery 相关工具到 DeliveryExpert
5. 迁移 Observability 相关工具到 ObservabilityExpert

### Phase 3: Executor & Summarizer (Week 3)

1. 实现 Executor Agent
2. 实现调度器 (并行/串行)
3. 实现 Summarizer Agent
4. 实现循环控制

### Phase 4: 集成测试 (Week 4)

1. 端到端测试
2. SSE 事件流验证
3. 错误处理测试
4. 性能测试

### Phase 5: 切换与清理

1. 将新 orchestrator 切为默认入口
2. 将旧 `internal/ai/agent.go` 降级为兼容层
3. 删除旧单体 `react.Agent + 全量工具池` 主链路
4. 清理旧 4 expert 命名和过时文档

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| LLM 调用次数增加 | 成本上升、延迟增加 | 并行执行、缓存资源解析结果 |
| 层级间信息丢失 | 执行结果不完整 | 结构化计划格式、上下文注入 |
| 专家调用失败 | 任务中断 | 重试机制、优雅降级 |
| 无限循环 | 成本失控 | max_iterations 限制 |
| 上下文膨胀 | 质量下降、token 超限 | `StepDigest/IterationDigest` 压缩、固定裁剪顺序 |
| 高风险工具误触发 | 造成错误变更 | `ToolMeta` 风险分级、review/approval、preview/apply 分离 |

## 成功指标

1. **功能完整性**: 支持所有现有工具操作
2. **响应时间**: P95 < 10s (简单查询), P95 < 60s (复杂排查)
3. **准确率**: 资源解析准确率 > 95%
4. **用户满意度**: 支持流式输出，用户感知延迟降低 50%

## 相关文档

- `openspec/specs/ai-assistant-adk-architecture/spec.md` - ADK 架构规范
- `openspec/specs/ai-control-plane-baseline/spec.md` - AI 控制平面基线
