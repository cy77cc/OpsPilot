# AI 模块架构重构：迁移至 ChatModelAgent

## 概述

将现有 plan-execute 架构重构为 `adk.NewChatModelAgent`，简化执行流程，统一审批机制。

## 背景与问题

### 当前架构
```
Planner → Executor → Replanner (plan-execute 循环)
```

### 存在问题
1. **复杂度高**：plan parsing、phase detection、replanning 逻辑复杂
2. **维护困难**：三个 Agent 协作，状态管理复杂
3. **效果不佳**：plan-execute 架构在实际场景中表现不理想

## 目标架构

```
ChatModelAgent (单 Agent 直接管理工具)
    └── Tools: [Gate(tool1), Gate(tool2), ...]
                └── Gate 包装：统一审批拦截
```

### 核心变更

| 变更项 | 描述 |
|--------|------|
| Agent 类型 | `adk.NewChatModelAgent` 替代 plan-execute |
| 工具管理 | Agent 直接管理工具，自动处理 tool calling loop |
| 审批机制 | Gate 包装工具，统一拦截变更操作 |
| System Prompt | 静态模板 + 动态变量注入 |

## 详细设计

### 1. Agent 构建

**文件**: `internal/ai/agents/agent.go`

```go
func NewAgent(ctx context.Context, deps Deps) (adk.ResumableAgent, error) {
    model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
        Timeout:  60 * time.Second,
        Thinking: false,
        Temp:     0.2,
    })
    if err != nil {
        return nil, err
    }

    registry := aitools.NewRegistry(deps.PlatformDeps)
    tools := registry.ADKTools(deps.ApprovalDecisionMaker)

    return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:          "OpsPilotAgent",
        Instruction:   "", // 由 Orchestrator 动态注入
        Model:         model,
        ToolsConfig:   adk.ToolsConfig{Tools: tools},
        MaxIterations: 20,
    })
}
```

### 2. 工具注册

**文件**: `internal/ai/tools/registry.go` (新增方法)

```go
// ADKTools 将注册的工具转换为 ADK tool 列表，
// 变更类工具通过 Gate 包装实现审批拦截。
func (r *Registry) ADKTools(decisionMaker *airuntime.ApprovalDecisionMaker) []tool.BaseTool {
    var tools []tool.BaseTool
    for name, spec := range r.tools {
        adkTool := adaptToADKTool(spec)
        // 变更类工具需要审批
        if spec.Mode == ModeMutating || spec.Risk == RiskHigh {
            adkTool = approval.NewGate(adkTool, spec.Capability, decisionMaker, nil)
        }
        tools = append(tools, adkTool)
    }
    return tools
}
```

### 3. 工具适配

**新文件**: `internal/ai/tools/adapt.go`

```go
// adaptToADKTool 将 ToolSpec 转换为 ADK InvokableTool
func adaptToADKTool(spec ToolSpec) tool.InvokableTool {
    return tool.NewTool(
        spec.Name,
        spec.Description,
        func(ctx context.Context, args string) (string, error) {
            params := parseArgs(args)
            result, err := spec.Execute(ctx, deps, params)
            if err != nil {
                return "", err
            }
            return json.Marshal(result.Result)
        },
    )
}
```

### 4. System Prompt 模板

**新文件**: `internal/ai/runtime/instruction.go`

```go
const InstructionTemplate = `你是 OpsPilot 智能运维助手，负责协助用户管理 Kubernetes 集群、主机、服务等基础设施资源。

## 核心能力
- 集群管理：查询集群状态、节点信息、资源使用情况
- 主机运维：批量执行命令、查看日志、监控状态
- 服务管理：部署、扩缩容、重启、查看状态
- 故障排查：分析日志、诊断问题、提供建议

## 工作原则
1. 优先使用只读工具收集信息，确认后再执行变更操作
2. 变更操作需要用户确认后才可执行
3. 操作前说明目的和预期影响
4. 遇到错误时分析原因并提供解决建议

## 当前上下文
- 场景: {scene_name}
- 项目: {project_name}
- 选中资源: {selected_resources}

请根据用户需求，合理使用工具完成任务。`

func BuildInstruction(ctx RuntimeContext) string {
    result := InstructionTemplate
    result = strings.ReplaceAll(result, "{scene_name}", ctx.SceneName)
    result = strings.ReplaceAll(result, "{project_name}", ctx.ProjectName)
    result = strings.ReplaceAll(result, "{selected_resources}", formatResources(ctx.SelectedResources))
    return result
}
```

### 5. Orchestrator 简化

**文件**: `internal/ai/orchestrator.go`

移除：
- `planParser`、`phaseDetector` 字段
- `streamExecution` 中的 plan parsing、phase detection 逻辑
- 所有与 `PlanEvent`、`StepEvent`、`ReplanEvent` 相关的处理

保留：
- `runner`、`checkpoints`、`executions` 字段
- `Run`、`Resume`、`ResumeStream` 方法
- SSE 事件转发逻辑
- 审批中断处理

简化后的 `streamExecution`：
```go
func (o *Orchestrator) streamExecution(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent], state *ExecutionState, emit StreamEmitter) error {
    for {
        event, ok := iter.Next()
        if !ok {
            break
        }
        if event.Err != nil {
            return event.Err
        }
        // 直接转发 ADK 事件
        emit(convertEvent(event))
        // 处理中断
        if event.Action != nil && event.Action.Interrupted != nil {
            return handleInterrupt(event)
        }
    }
    return nil
}
```

## 清理清单

### 删除文件

| 文件路径 | 原因 |
|----------|------|
| `internal/ai/runtime/plan_parser.go` | 不再需要计划解析 |
| `internal/ai/runtime/phase_detector.go` | 不再需要阶段检测 |

### 删除类型定义 (`runtime/runtime.go`)

| 类型 | 原因 |
|------|------|
| `PhaseName` | 不再需要阶段概念 |
| `PhaseEvent` | 不再需要阶段事件 |
| `PlanEvent` | 不再需要计划事件 |
| `PlanStep` | 不再需要计划步骤 |
| `StepEvent` | 不再需要步骤事件 |
| `ReplanEvent` | 不再需要重规划事件 |
| `ChainNodeKind` 相关常量 | 简化为 tool 事件 |
| `ChainNodeInfo` | 简化事件结构 |

### 简化 Orchestrator

移除以下逻辑：
- `planningText`、`planningStarted`、`planningCompleted` 状态变量
- `planNodeID`、`executeNodeID` 节点 ID 管理
- `openChainNode`、`patchChainNode`、`replaceChainNode`、`closeActiveNode` 函数
- `stepsToChainDetails`、`planStepsToStructured`、`toolResultChainPayload` 辅助函数
- `claimStepForTool`、`sortedStepIDs` 步骤管理函数

### 保留文件

| 文件路径 | 处理方式 |
|----------|----------|
| `internal/ai/agents/agent.go` | 重写，使用 ChatModelAgent |
| `internal/ai/orchestrator.go` | 大幅简化 |
| `internal/ai/runtime/runtime.go` | 删除计划相关类型，保留状态类型 |
| `internal/ai/tools/registry.go` | 新增 `ADKTools()` 方法 |
| `internal/ai/tools/approval/gate.go` | 保留，小调整适配新架构 |
| `internal/ai/tools/approval/summary.go` | 保留 |

## 执行计划

### Phase 1: 工具适配层
1. 创建 `internal/ai/tools/adapt.go`，实现 `adaptToADKTool`
2. 在 `Registry` 添加 `ADKTools()` 方法
3. 编写单元测试验证工具转换

### Phase 2: System Prompt
1. 创建 `internal/ai/runtime/instruction.go`
2. 实现 `BuildInstruction()` 函数
3. 编写单元测试验证模板渲染

### Phase 3: Agent 重构
1. 重写 `internal/ai/agents/agent.go`
2. 更新 `NewAgent()` 使用 ChatModelAgent
3. 集成工具和 instruction

### Phase 4: Orchestrator 简化
1. 移除 plan parsing、phase detection 相关代码
2. 简化 `streamExecution()` 事件处理
3. 保留审批中断恢复逻辑

### Phase 5: 清理旧代码
1. 删除 `runtime/plan_parser.go`
2. 删除 `runtime/phase_detector.go`
3. 清理 `runtime/runtime.go` 中废弃类型
4. 清理 `orchestrator.go` 中废弃函数

### Phase 6: 测试与验证
1. 更新现有测试
2. 手动验证审批流程
3. 验证 SSE 事件流

## 风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| ADK API 变化 | 查阅最新文档，使用稳定 API |
| 审批中断恢复失败 | 复用现有 Gate 逻辑，充分测试 |
| 前端事件格式变化 | 保持 SSE 事件格式兼容 |
