# DeepAgents 架构迁移设计文档

## 概述

### 背景

当前 AI 模块采用 Plan-Execute-Replan 架构，存在以下问题：

1. **规划过于刚性** - 每次请求都强制先规划，简单任务开销大
2. **上下文污染** - Planner/Executor/Replanner 共享上下文，复杂任务容易混乱
3. **代码复杂度高** - 三组件维护成本大

### 目标

迁移至 CloudWeGo Eino DeepAgents 架构，实现：

- 按需规划，LLM 自主决定是否需要 write_todos
- Sub-Agent 上下文隔离，按领域划分
- 保留现有 HITL 审批机制

### 参考资料

- [DeepAgents 官方文档](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/deepagents/)
- 参考实现：`/root/learn/eino-examples/adk/multiagent/deep/`

### DeepAgents API 确认

基于 `github.com/cloudwego/eino/adk/prebuilt/deep` 源码验证：

```go
// Config 定义 DeepAgent 配置（已验证）
type Config struct {
    Name          string                      // Agent 名称
    Description   string                      // Agent 描述
    ChatModel     model.ToolCallingChatModel  // LLM 模型
    Instruction   string                      // System Prompt
    SubAgents     []adk.Agent                 // 子 Agent 列表
    ToolsConfig   adk.ToolsConfig             // 工具配置
    MaxIteration  int                         // 最大迭代次数

    // 关键配置项（已验证存在）
    WithoutWriteTodos       bool              // 禁用内置 write_todos
    WithoutGeneralSubAgent  bool              // 禁用通用子 Agent
    Middlewares             []adk.AgentMiddleware // 中间件

    // Session 存储
    OutputKey string                         // 输出存储到 Session 的 Key
}

// New 创建 DeepAgent 实例（已验证签名）
func New(ctx context.Context, cfg *Config) (adk.ResumableAgent, error)
```

**TaskTool 自动注册机制**：

DeepAgents 内部自动为 `SubAgents` 创建 `task` 工具，LLM 通过调用 `task` 工具委派给子 Agent：

```go
// DeepAgents 内部实现（简化）
func newTaskTool(subAgents []adk.Agent) tool.InvokableTool {
    return &taskTool{
        subAgents: map[string]tool.InvokableTool{
            "QAAgent":   adk.NewAgentTool(ctx, qaAgent),
            "K8sAgent":  adk.NewAgentTool(ctx, k8sAgent),
            // ...
        },
    }
}

// task 工具参数
type taskToolArgument struct {
    SubagentType string `json:"subagent_type"` // 子 Agent 名称
    Description  string `json:"description"`   // 任务描述
}
```

---

## 架构设计

### 整体架构

```
用户请求
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│                    OpsPilotAgent (Main)                      │
│  - deep.New() + 定制 write_ops_todos                         │
│  - 自主决定规划时机                                           │
│  - 通过 TaskTool 委派 Sub-Agent                               │
└─────────────────────────────────────────────────────────────┘
    │
    ├─── 简单问答 ──────────────► QAAgent (RAG 检索)
    │
    ├─── K8s 诊断/查询 ────────► K8sAgent (只读工具)
    │
    ├─── 主机操作 ─────────────► HostAgent (只读工具)
    │
    ├─── 监控查询 ─────────────► MonitorAgent (只读工具)
    │
    └─── 变更操作 ─────────────► ChangeAgent (写工具 + HITL)
                                    │
                                    ▼
                              ApprovalMiddleware
                              (复用现有实现)
```

### 与原架构对比

| 维度 | 原 Plan-Execute | 新 DeepAgents |
|------|----------------|---------------|
| **规划方式** | 独立 Planner Agent 先生成完整计划 | write_todos 工具，按需规划 |
| **执行模式** | Executor → Replanner 循环 | ReAct 循环，主 Agent 自由调用工具 |
| **上下文** | Planner/Executor/Replanner 共享 Session | Sub-Agent 上下文隔离 |
| **灵活性** | 每次任务必须先规划 | 可跳过规划，直接执行简单任务 |
| **任务委派** | 通过 Router 路由到不同 Agent | TaskTool 委派给隔离的子 Agent |
| **HITL** | ApprovalMiddleware 原生支持 | 复用 ApprovalMiddleware |

---

## 详细设计

### 0. 嵌套中断恢复可行性验证（POC Gate）

在进入正式实现前，先验证 DeepAgents 嵌套层级中的 HITL 穿透性：

1. Main Agent 通过 `task` 委派到 `ChangeAgent`
2. `ChangeAgent` 调用高风险工具触发 `StatefulInterrupt`
3. 中断可冒泡到最外层 Runner，并正确输出 `tool_approval` SSE
4. 前端审批后，`ResumeWithParams` 调用 Main Agent
5. 恢复参数可透传回原挂起的 `ChangeAgent` 上下文并继续执行

**POC 通过标准**：

- 能稳定复现 `interrupt -> approve -> resume -> done` 全链路
- 恢复后不丢失原始 tool call 上下文（call_id / approval_id / arguments）
- Run 状态从 `waiting_approval` 正确回到执行态并最终收敛

若 POC 失败（框架不支持跨层 Resume 透传），需在 Main Agent 增加一层恢复状态机（或 Session 映射层）后再推进 Phase 1/2。

### 1. OpsTODO 结构

基于运维场景定制的任务项结构：

```go
// Package todo 提供运维场景定制的 TODO 结构和工具。
package todo

// OpsTODO 运维场景任务项。
//
// 包含三层字段：
//   - 基础层：任务描述、状态
//   - 基础设施层：集群、命名空间、资源类型
//   - 风险控制层：风险等级、审批需求
//   - 执行上下文层：预估耗时、依赖关系
type OpsTODO struct {
    // ========== 基础字段 ==========
    Content    string `json:"content"`              // 任务描述
    ActiveForm string `json:"activeForm"`           // 进行中状态描述
    Status     string `json:"status"`               // pending | in_progress | completed

    // ========== 基础设施层 ==========
    Cluster       string `json:"cluster,omitempty"`       // 目标集群名称
    Namespace     string `json:"namespace,omitempty"`     // 目标命名空间
    ResourceType  string `json:"resourceType,omitempty"`  // Pod/Deployment/Service/Node/ConfigMap

    // ========== 风险控制层 ==========
    RiskLevel       string `json:"riskLevel,omitempty"`       // low | medium | high | critical
    RequiresApproval bool  `json:"requiresApproval,omitempty"` // 是否需要人工审批

    // ========== 执行上下文层 ==========
    EstimatedDuration string   `json:"estimatedDuration,omitempty"` // 预估耗时，如 "2m", "1h"
    DependsOn         []string `json:"dependsOn,omitempty"`        // 前置 Todo ID 列表
}
```

**字段注入策略**：

| 字段 | 来源 | 说明 |
|------|------|------|
| `cluster` | 前端路由状态自动注入 | 当前选中的集群 |
| `namespace` | 前端路由状态自动注入 | 当前选中的命名空间 |
| `resourceType` | LLM 根据任务内容推断 | 操作的资源类型 |
| `riskLevel` | LLM 根据操作类型判断 | low/medium/high/critical |
| `requiresApproval` | LLM 根据工具类型判断 | 写操作默认为 true |
| `dependsOn` | LLM 根据任务依赖关系生成 | 复杂变更流程 |

### 2. Sub-Agent 划分

#### 2.1 Agent 清单

| Agent | 类型 | 工具集 | 需要审批 | 温度 | 最大迭代 |
|-------|------|--------|---------|------|---------|
| QAAgent | 只读 | rag_search, faq_query | 否 | 0.3 | 10 |
| K8sAgent | 只读 | k8s_query, k8s_logs, k8s_events, k8s_list_resources | 否 | 0.0 | 20 |
| HostAgent | 只读 | host_exec, os_get_cpu_mem, host_list_inventory | 否 | 0.0 | 20 |
| MonitorAgent | 只读 | monitor_metric, monitor_alert, monitor_alert_rule_list | 否 | 0.1 | 15 |
| ChangeAgent | 写操作 | k8s_scale_deployment, k8s_restart_deployment, k8s_delete_pod, host_exec | 是 | 0.0 | 30 |
| InspectionAgent | 定时 | inspection_tools, report_generate | 否 | 0.1 | 50 |

#### 2.2 Agent 详细设计

**QAAgent**

```go
// QAAgent 知识问答子 Agent。
//
// 专注于知识库检索和 FAQ 查询，不执行任何集群操作。
// 通过 RAG 检索内部知识库，回答运维相关问题。
func NewQAAgent(ctx context.Context) (adk.Agent, error) {
    return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:          "QAAgent",
        Description:   "Knowledge assistant for OpsPilot documentation and FAQ. Use for general questions about platform usage, best practices, and troubleshooting guides.",
        Instruction:   qaAgentInstruction,
        Model:         model,
        ToolsConfig:   qaToolsConfig,
        MaxIterations: 10,
    })
}
```

**K8sAgent**

```go
// K8sAgent Kubernetes 只读操作子 Agent。
//
// 提供集群资源查询、日志获取、事件查看等只读操作。
// 严禁执行任何写操作，所有工具均为只读。
func NewK8sAgent(ctx context.Context) (adk.Agent, error) {
    return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:          "K8sAgent",
        Description:   "Kubernetes read-only operations agent. Query resources, fetch logs, view events. No write operations allowed.",
        Instruction:   k8sAgentInstruction,
        Model:         model,
        ToolsConfig:   k8sReadonlyToolsConfig,
        MaxIterations: 20,
    })
}
```

**HostAgent**

```go
// HostAgent 主机操作子 Agent（只读）。
//
// 提供主机状态查询、只读命令执行、资源监控等操作。
// 写操作需要通过 ChangeAgent 执行。
func NewHostAgent(ctx context.Context) (adk.Agent, error) {
    return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:          "HostAgent",
        Description:   "Host operations agent for read-only commands. Query host status, run readonly commands, check system resources. Write operations require ChangeAgent.",
        Instruction:   hostAgentInstruction,
        Model:         model,
        ToolsConfig:   hostReadonlyToolsConfig,
        MaxIterations: 20,
    })
}
```

**MonitorAgent**

```go
// MonitorAgent 监控查询子 Agent。
//
// 提供 Prometheus 指标查询、告警查看、链路追踪等监控能力。
// 所有操作均为只读。
func NewMonitorAgent(ctx context.Context) (adk.Agent, error) {
    return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:          "MonitorAgent",
        Description:   "Monitoring and observability agent. Query Prometheus metrics, view alerts, trace requests. Read-only operations.",
        Instruction:   monitorAgentInstruction,
        Model:         model,
        ToolsConfig:   monitorToolsConfig,
        MaxIterations: 15,
    })
}
```

**ChangeAgent**

```go
// ChangeAgent 变更操作子 Agent。
//
// 专门处理所有写操作，内置 ApprovalMiddleware 实现 HITL 审批。
// 高风险操作（重启、扩缩容、删除）在执行前触发审批流程。
//
// HITL 工作流:
//  1. 调用高风险工具（如 k8s_restart_deployment）
//  2. ApprovalMiddleware 拦截，触发 StatefulInterrupt
//  3. Runner 通过 SSE 发送 tool_approval 事件给前端
//  4. 用户审批确认或拒绝
//  5. ResumeWithParams 携带审批结果恢复执行
func NewChangeAgent(ctx context.Context) (adk.ResumableAgent, error) {
    return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:          "ChangeAgent",
        Description:   "Change operations agent with human-in-the-loop approval. Scale deployments, restart pods, execute host commands. All write operations require approval.",
        Instruction:   changeAgentInstruction,
        Model:         model,
        ToolsConfig:   changeToolsConfig,
        MaxIterations: 30,
        Middlewares: []adk.ChatModelAgentMiddleware{
            ApprovalMiddleware(approvalMiddlewareConfig(ctx)),
        },
    })
}
```

**InspectionAgent**

```go
// InspectionAgent 定时巡检子 Agent。
//
// 由调度系统调用，执行定期巡检任务并生成报告。
// 不参与用户聊天路由。
func NewInspectionAgent(ctx context.Context) (adk.Agent, error) {
    return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:          "InspectionAgent",
        Description:   "Scheduled inspection agent for cluster health checks. Generate reports, detect anomalies. Not for interactive chat.",
        Instruction:   inspectionAgentInstruction,
        Model:         model,
        ToolsConfig:   inspectionToolsConfig,
        MaxIterations: 50,
    })
}
```

### 3. Main Agent 入口

```go
// Package agent 提供 OpsPilot Agent 实现。
package agent

// mustAgent 辅助函数，Agent 创建失败时 panic。
//
// 用于初始化阶段，确保 Agent 必须创建成功。
func mustAgent(agent adk.Agent, err error) adk.Agent {
    if err != nil {
        panic(fmt.Sprintf("failed to create agent: %v", err))
    }
    return agent
}

// NewOpsPilotAgent 创建主入口 Agent。
//
// 使用 DeepAgents 架构，内置 write_ops_todos 工具支持按需规划。
// LLM 自主决定是否需要规划，简单任务可直接执行。
func NewOpsPilotAgent(ctx context.Context) (adk.ResumableAgent, error) {
    // 获取主模型
    mainChatModel, err := chatmodel.GetDefaultChatModel(ctx, nil, chatmodel.ChatModelConfig{
        Timeout:  120 * time.Second,
        Thinking: false,
        Temp:     0.0,
    })
    if err != nil {
        return nil, fmt.Errorf("get main chat model: %w", err)
    }

    // 初始化子 Agent
    subAgents := []adk.Agent{
        mustAgent(NewQAAgent(ctx)),
        mustAgent(NewK8sAgent(ctx)),
        mustAgent(NewHostAgent(ctx)),
        mustAgent(NewMonitorAgent(ctx)),
        mustAgent(NewChangeAgent(ctx)),
    }

    // 构建主 Agent 工具集（平台发现工具）
    mainToolsConfig := adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: tools.NewPlatformTools(ctx),
        },
    }

    // 创建定制 write_ops_todos 中间件
    writeOpsTodosMiddleware, err := todo.NewWriteOpsTodosMiddleware()
    if err != nil {
        return nil, fmt.Errorf("create write_ops_todos middleware: %w", err)
    }

    // 创建 DeepAgent
    return deep.New(ctx, &deep.Config{
        Name:          "OpsPilotAgent",
        Description:   "OpsPilot infrastructure operations assistant with intelligent task planning and approval-gated execution",
        Instruction:   opsPilotInstruction,
        ChatModel:     mainChatModel,
        SubAgents:     subAgents,
        ToolsConfig:   mainToolsConfig,
        MaxIteration:  50,
        // 禁用内置 write_todos，使用定制版本
        WithoutWriteTodos: true,
        // 注入定制中间件
        Middlewares: []adk.AgentMiddleware{writeOpsTodosMiddleware},
    })
}
```

**TaskTool 委派机制说明**：

1. DeepAgents 自动为 `SubAgents` 创建 `task` 工具
2. LLM 调用 `task` 工具时传入 `subagent_type` 和 `description`
3. 内部路由到对应 Sub-Agent 执行，上下文隔离
4. Sub-Agent 执行完成后返回结果给 Main Agent

### 4. ToolsConfig 构建示例

```go
// Package tools 提供工具集构建函数。
package tools

// NewQATools 构建 QAAgent 工具集。
func NewQATools(ctx context.Context) []tool.BaseTool {
    return []tool.BaseTool{
        NewRAGSearchTool(ctx),
        NewFAQQueryTool(ctx),
    }
}

// NewK8sReadonlyTools 构建 K8sAgent 只读工具集。
func NewK8sReadonlyTools(ctx context.Context) []tool.BaseTool {
    return []tool.BaseTool{
        kubernetes.NewQueryTool(ctx),
        kubernetes.NewLogsTool(ctx),
        kubernetes.NewEventsTool(ctx),
        kubernetes.NewListResourcesTool(ctx),
    }
}

// NewChangeTools 构建 ChangeAgent 写操作工具集。
func NewChangeTools(ctx context.Context) []tool.BaseTool {
    return []tool.BaseTool{
        kubernetes.K8sScaleDeployment(ctx),
        kubernetes.K8sRestartDeployment(ctx),
        kubernetes.K8sDeletePod(ctx),
        host.HostExec(ctx),
    }
}

// 工具集配置示例
qaToolsConfig := adk.ToolsConfig{
    ToolsNodeConfig: compose.ToolsNodeConfig{
        Tools: NewQATools(ctx),
    },
}
```

### 5. Session 管理机制

```go
// Package todo 提供 Session Key 常量。
package todo

// Session Key 常量定义。
const (
    // SessionKeyOpsTodos 存储 OpsTODO 列表的 Session Key
    SessionKeyOpsTodos = "opspilot_session_key_ops_todos"
)

// Session 生命周期：
// 1. Runner 创建时初始化 Session（通过 CheckPointStore）
// 2. Agent 执行过程中通过 adk.AddSessionValue/GetSessionValue 读写
// 3. 执行完成或中断时，Session 持久化到 CheckPointStore
// 4. Resume 时从 CheckPointStore 恢复 Session

// 跨 Sub-Agent Session 隔离：
// - DeepAgents 的 Sub-Agent 执行时上下文隔离
// - Main Agent 的 Session 数据不自动传递给 Sub-Agent
// - 如需共享，通过 TaskTool 参数传递

// 基础设施上下文（Infra Context）约束：
// - 在 Session 维护当前 cluster_id / namespace / host scope
// - K8s/Host 相关工具执行前通过中间件校验上下文
// - 若工具参数缺失且 Session 有默认值：自动补全
// - 若参数与 Session 冲突：优先要求模型显式确认或返回可读错误
```

### 6. 定制 write_ops_todos 工具

```go
// Package todo 提供运维场景定制的 TODO 工具。
package todo

const writeOpsTodosDescription = `
Record and track operational tasks with infrastructure context.
Use this tool when planning multi-step operations involving Kubernetes resources, host commands, or infrastructure changes.

The tool captures:
- Task description and status
- Target cluster, namespace, and resource type
- Risk level and approval requirements
- Dependencies between tasks

Use proactively for complex operations. Skip for simple queries.
`

// NewWriteOpsTodosMiddleware 创建 write_ops_todos 中间件（导出函数，供主入口调用）。
func NewWriteOpsTodosMiddleware() (adk.AgentMiddleware, error) {
    t, err := utils.InferTool("write_ops_todos", writeOpsTodosDescription,
        func(ctx context.Context, input writeOpsTodosArguments) (string, error) {
            // 存储到 Session
            adk.AddSessionValue(ctx, SessionKeyOpsTodos, input.Todos)

            // 格式化输出
            var summary strings.Builder
            summary.WriteString("Task plan updated:\n")
            for i, todo := range input.Todos {
                status := "○"
                if todo.Status == "in_progress" {
                    status = "◐"
                } else if todo.Status == "completed" {
                    status = "●"
                }
                summary.WriteString(fmt.Sprintf("%d. %s %s", i+1, status, todo.Content))
                if todo.Cluster != "" {
                    summary.WriteString(fmt.Sprintf(" [%s/%s]", todo.Cluster, todo.Namespace))
                }
                if todo.RiskLevel != "" {
                    summary.WriteString(fmt.Sprintf(" (%s)", todo.RiskLevel))
                }
                summary.WriteString("\n")
            }
            return summary.String(), nil
        })

    return adk.AgentMiddleware{
        AdditionalInstruction: opsTodosPrompt,
        AdditionalTools:       []tool.BaseTool{t},
    }, nil
}
```

**任务记忆约束（防偏离）**：

- `write_ops_todos` 不仅写入 Session，还应在 Main Agent 每轮推理前把未完成任务（`status=pending|in_progress`）注入附加指令。
- 若执行流偏离未完成任务（例如无关工具调用），应优先拉回到任务清单或向用户说明偏离原因。
- 建议增加 Hook：当同一子任务连续失败超过 3 次，停止盲目重试并请求用户介入。

### 5. System Prompt 设计

**Main Agent Instruction**

```markdown
You are OpsPilot, an intelligent infrastructure operations assistant.

## Capabilities

You can help users with:
- Kubernetes cluster management (query, diagnose, scale, restart)
- Host operations (execute commands, check status)
- Monitoring and observability (metrics, alerts, traces)
- Knowledge queries (documentation, FAQ)

## Task Planning

Use write_ops_todos when:
- Operation involves multiple steps
- Changes affect multiple resources
- Task has dependencies
- Operation requires approval

Skip planning for simple queries.

## Sub-Agent Delegation

Use task tool to delegate to specialized agents:
- QAAgent: Knowledge and documentation questions
- K8sAgent: Kubernetes read-only operations
- HostAgent: Host status and readonly commands
- MonitorAgent: Metrics, alerts, traces
- ChangeAgent: Write operations (requires approval)

## Safety

- Always verify cluster and namespace before operations
- High-risk operations require human approval
- Never execute unapproved destructive operations
```

---

## 架构映射表

| 现有组件 | 新架构 | 迁移说明 |
|---------|--------|---------|
| RouterAgent | OpsPilotAgent | Main Agent 直接处理路由，不再需要独立 Router |
| DiagnosisAgent | K8sAgent + MonitorAgent | 诊断功能拆分到 K8sAgent（资源查询）和 MonitorAgent（指标查询） |
| ChangeAgent | ChangeAgent（重写） | 使用 ChatModelAgent + ApprovalMiddleware，移除 Planner/Executor/Replanner |
| QAAgent | QAAgent（重写） | 使用 ChatModelAgent，移除 PlanExecute 包装 |
| InspectionAgent | InspectionAgent（重写） | 使用 ChatModelAgent，保留调度入口 |
| Planner | write_ops_todos 工具 | 作为 Main Agent 的工具，LLM 自主调用 |
| Executor | Main Agent ReAct 循环 | DeepAgents 内置 ReAct 循环处理执行 |
| Replanner | Main Agent ReAct 循环 | 动态调整通过工具调用实现 |

---

## 目录结构

```
internal/ai/
├── agent/                    # Agent 实现
│   ├── main.go              # OpsPilotAgent 主入口
│   ├── qa.go                # QAAgent
│   ├── k8s.go               # K8sAgent
│   ├── host.go              # HostAgent
│   ├── monitor.go           # MonitorAgent
│   ├── change.go            # ChangeAgent (含 HITL)
│   ├── inspection.go        # InspectionAgent
│   └── prompt.go            # 所有 Agent 的 Instruction
│
├── todo/                     # 定制 TODO
│   ├── types.go             # OpsTODO 结构定义
│   └── tool.go              # write_ops_todos 工具实现
│
├── chatmodel/                # LLM 配置（保留）
│
├── tools/                    # 工具实现（保留）
│   ├── kubernetes/          # K8s 工具
│   ├── host/                # 主机工具
│   ├── monitor/             # 监控工具
│   ├── platform/            # 平台工具
│   └── middleware/          # ApprovalMiddleware（保留）
│
├── runtime/                  # SSE/Projection（保留）
│
└── checkpoint/               # CheckPointStore（保留）

# 删除的目录
# internal/ai/agents/planner/     (移除)
# internal/ai/agents/executor/    (移除)
# internal/ai/agents/replanner/   (移除)
# internal/ai/agents/planexecuteutil/ (移除)
# internal/ai/agents/diagnosis/   (合并到 K8sAgent)
# internal/ai/agents/change/      (重写为 agent/change.go)
# internal/ai/agents/qa/          (重写为 agent/qa.go)
# internal/ai/agents/inspection/  (重写为 agent/inspection.go)
```

---

## 迁移计划

### Phase 1: 基础架构搭建

1. 创建新分支 `feature/deepagents`
2. 创建 `internal/ai/agent/` 目录
3. 实现 `todo/types.go` 和 `todo/tool.go`
4. 实现 `agent/main.go` 主入口

### Phase 2: Sub-Agent 实现

1. 实现 QAAgent、K8sAgent、HostAgent、MonitorAgent
2. 实现 ChangeAgent（复用 ApprovalMiddleware）
3. 实现 InspectionAgent

### Phase 3: 集成测试

1. 更新 HTTP Handler 适配新架构
2. 端到端测试各场景（问答、K8s 只读、Host 只读、监控、写操作审批、审批恢复）
3. 性能对比测试（简单问答 P95 延迟、复杂任务成功率、token 消耗）

### Phase 4: 清理与合并

1. 删除旧的 planner/executor/replanner 代码
2. 更新文档
3. 合并到 main 分支

---

## 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| LLM 规划能力不足 | 复杂任务可能跳过规划 | Prompt 中明确规划场景 |
| 迁移期间功能缺失 | 影响线上服务 | 新分支开发，充分测试后合并 |
| ApprovalMiddleware 适配 | 写操作审批可能失效 | 复用现有实现，增加单元测试 |
| 性能变化 | Token 消耗可能增加 | 监控对比，优化 Prompt |
| CheckPoint 数据格式变化 | 现有会话无法恢复 | 设计兼容层或清除旧数据 |
| SSE 事件格式变化 | 前端解析失败 | 保持事件格式一致，增加适配测试 |
| 子 Agent 中断无法冒泡 | 审批流失效，写操作卡死 | 将嵌套 interrupt/resume POC 设为前置 Gate，未通过不进入开发阶段 |
| 上下文丢失或参数漂移 | 误查/误操作错误集群或命名空间 | Session 维护 infra context + 工具调用前校验与自动补全 |
| ReAct 循环重试失控 | token 失控、响应延迟恶化 | 失败重试上限（同子任务最多 3 次）+ 熔断后请求人工介入 |

### API 兼容性风险

| 组件 | 兼容性 | 处理策略 |
|------|--------|---------|
| CheckPointStore | 兼容 | 复用现有实现，数据格式不变 |
| SSE 事件类型 | 兼容 | 保持 `meta/delta/tool_call/tool_approval/tool_result/run_state/done/error` 格式 |
| ResumeWithParams | 兼容 | 复用现有审批恢复逻辑 |
| 前端 useXPlatformChat | 兼容 | 无需修改，事件格式一致 |

### 回滚策略

1. **代码回滚**：
   - 保留 `main` 分支不变，在 `feature/deepagents` 分支开发
   - 测试不通过时，直接删除 feature 分支
   - 合并后发现问题，`git revert` 回滚

2. **数据回滚**：
   - CheckPointStore 数据格式不变，无需数据迁移
   - 如需清理测试数据，提供清理脚本

3. **快速切换**：
   - 新增 feature flag `feature_flags.ai_deepagents` 控制新架构开关（需先在 `internal/config/config.go` 的 `FeatureFlags` 中补充字段）
   - 默认关闭，验证后开启
   - 建议灰度顺序：开发环境 -> 预发环境 -> 生产小流量 -> 全量

---

## 前端适配说明

新架构对前端的影响：

| 组件 | 变化 | 操作 |
|------|------|------|
| useXPlatformChat | 基本无变化 | 保持现有实现；仅在移除 `plan/replan` 事件时需要同步调整 |
| SSE 事件处理 | 条件无变化 | 前提是继续兼容 `plan/replan` 或保持现有降级逻辑 |
| 审批流程 | 无变化 | ApprovalMiddleware 复用 |
| 场景路由映射 | 可能调整 | 根据测试结果优化 |

**前端通常无需修改**，除非：
- 需要展示 `write_ops_todos` 的任务列表（新功能）
- 需要优化 Sub-Agent 切换的 UI 反馈
- 后端在迁移中去除或重定义 `plan/replan` 事件

**推荐增强（平滑过渡）**：

- 在 `write_ops_todos` 更新时，额外发送结构化事件（建议：`ops_plan_updated`），载荷为 Todos 数组。
- 前端可直接渲染结构化任务步骤，避免依赖自然语言 delta 解析，降低迁移期 UI 回归风险。

---

## 验收标准

1. **功能验收**
   - [ ] POC Gate：嵌套 `StatefulInterrupt` 冒泡与 `ResumeWithParams` 透传验证通过
   - [ ] 问答场景：`QAAgent` 能返回知识库答案，且不触发写工具
   - [ ] K8s 只读：`K8sAgent` 可完成资源查询/日志/事件，且不触发审批
   - [ ] 主机只读：`HostAgent` 可执行 `host_exec` 只读命令并返回结构化结果
   - [ ] 监控查询：`MonitorAgent` 可查询告警与指标（`monitor_alert` / `monitor_metric`）
   - [ ] 写操作审批：`ChangeAgent` 调用高风险工具时产生 `tool_approval`，拒绝后不中断系统稳定性
   - [ ] 审批恢复：审批通过后可 `ResumeWithParams` 恢复并完成 run
   - [ ] 上下文约束：缺省 `cluster/namespace` 可自动补全，冲突参数会被拒绝或要求确认
   - [ ] 定时巡检：`InspectionAgent` 可被调度入口触发并产出巡检结果

2. **性能验收**
   - [ ] 简单任务 P95 响应时间 ≤ 原架构（基线与样本量需记录）
   - [ ] 复杂任务成功率 ≥ 原架构
   - [ ] 单次复杂任务 token 消耗涨幅在可接受阈值内（建议先以 20% 为报警线）
   - [ ] 死循环防护：同一子任务失败 >3 次时可终止重试并返回人工介入提示

3. **代码质量**
   - [ ] 测试覆盖率 ≥ 40%
   - [ ] 代码通过 lint 检查
   - [ ] 删除无用代码（含旧 planner/executor/replanner 残留引用）
   - [ ] 文档与实现对齐（工具名、事件名、feature flag 名称一致）
