# Proposal: 修复AI助手流式输出和工具动画

## 问题背景

在Hybrid MOE Agent架构重构后，出现了两个严重的用户体验问题：

### 问题1: 流式输出丢失

**现象**: 前端需要等待很长时间才能一次性收到完整结果，没有逐字输出效果。

**根因分析**:

```go
// platform_agent.go: Stream() 方法
func (p *PlatformAgent) Stream(ctx context.Context, messages []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
    // ...
    result, err := p.orchestrator.Execute(ctx, req)  // 阻塞调用，等待完整执行
    if err != nil {
        return nil, err
    }
    return schema.StreamReaderFromArray([]*schema.Message{  // 假流式，实际已全部完成
        schema.AssistantMessage(result.Response, nil),
    }), nil
}
```

问题链:
1. `Stream()` 调用 `orchestrator.Execute()` - 阻塞等待
2. `orchestrator.Execute()` 调用 `executor.ExecuteStep()`
3. `executor.ExecuteStep()` 调用 `exp.Agent.Generate()` - 非流式
4. 完成后将结果包装成假流式返回

### 问题2: 工具调用无动画

**现象**: AI开始调用工具后，前端没有动画效果，用户不知道正在执行什么操作。

**根因分析**:

```go
// executor.go: ExecuteStep() 方法
func (e *ExpertExecutor) ExecuteStep(ctx context.Context, step *ExecutionStep, ...) (*ExpertResult, error) {
    // ...
    resp, err := exp.Agent.Generate(ctx, []*schema.Message{...})  // context丢失了tool event emitter
    // ...
}
```

问题链:
1. `chat_handler.go` 中通过 `tools.WithToolEventEmitter(ctx, emitter)` 设置事件发射器
2. 但 `PlatformAgent.Stream()` 没有传递这个context到orchestrator
3. Orchestrator/Executor 没有转发context到expert agent
4. 工具执行时不触发 `tool_call`/`tool_result` 事件
5. 前端收不到事件，无法显示动画

## 目标

### 主要目标

1. **恢复流式输出** - 单专家场景使用真正的streaming，用户可以看到逐字输出
2. **恢复工具动画** - 工具调用时前端能收到 `tool_call`/`tool_result` 事件

### 非目标

- 不改变多专家协作的核心逻辑
- 不改变现有的路由决策机制
- 不引入新的依赖

## 范围

### 修改的文件

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `internal/ai/platform_agent.go` | 修改 | 修复Stream()方法 |
| `internal/ai/experts/orchestrator.go` | 修改 | 添加StreamExecute方法 |
| `internal/ai/experts/executor.go` | 修改 | 支持流式执行和context传递 |
| `internal/ai/experts/types.go` | 修改 | 添加流式相关类型 |

### 接口变更

```go
// Orchestrator 新增方法
type Orchestrator interface {
    Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error)
    StreamExecute(ctx context.Context, req *ExecuteRequest) (*schema.StreamReader[*schema.Message], error)  // 新增
}

// ExpertExecutor 新增方法
type ExpertExecutor interface {
    ExecuteStep(ctx context.Context, step *ExecutionStep, priorResults []ExpertResult, baseMessage string) (*ExpertResult, error)
    StreamStep(ctx context.Context, step *ExecutionStep, priorResults []ExpertResult, baseMessage string) (*schema.StreamReader[*schema.Message], error)  // 新增
}
```

## 解决方案

### 方案1: 单专家直接流式

当路由决策为 `StrategySingle` 时，直接使用专家的 `Stream()` 方法：

```go
func (p *PlatformAgent) Stream(ctx context.Context, messages []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
    // ...
    if req.Decision.Strategy == experts.StrategySingle {
        if exp, ok := p.registry.GetExpert(req.Decision.PrimaryExpert); ok && exp.Agent != nil {
            return exp.Agent.Stream(ctx, messages)  // 真正的流式
        }
    }
    // 多专家场景后续处理...
}
```

### 方案2: 多专家顺序流式

对于多专家协作场景，实现顺序流式输出：

1. 先流式输出主专家的响应
2. 然后流式输出辅助专家的补充内容
3. 最后流式输出聚合结果

### 方案3: Context传递修复

确保tool event emitter context正确传递：

```go
// executor.go
func (e *ExpertExecutor) StreamStep(ctx context.Context, step *ExecutionStep, ...) (*schema.StreamReader[*schema.Message], error) {
    // ctx已经包含了tool event emitter
    // 直接传递给expert agent
    return exp.Agent.Stream(ctx, messages)
}
```

## 风险评估

| 风险 | 等级 | 缓解措施 |
|------|------|----------|
| 流式输出中断 | 中 | 添加重试机制，优雅降级到非流式 |
| Context传递错误 | 低 | 单元测试覆盖 |
| 多专家流式复杂 | 中 | 先实现单专家，多专家可暂时降级 |

## 验收标准

1. 单专家场景能看到逐字流式输出
2. 工具调用时前端显示动画效果
3. `tool_call`/`tool_result` 事件正常触发
4. 多专家场景至少能返回结果（即使不是流式）
5. 现有功能无回归
