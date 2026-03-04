# Design: 流式输出修复方案

## 问题根源

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         当前架构的问题链                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   chat_handler.go                                                          │
│   ┌───────────────────────────────────────────────────────────────────┐    │
│   │ ctx = tools.WithToolEventEmitter(ctx, emitter)  // 设置事件发射器  │    │
│   │ stream, err := h.svcCtx.AI.Stream(streamCtx, inputMessages)       │    │
│   │ // 期望: 逐字输出 + tool_call/tool_result 事件                     │    │
│   └───────────────────────────────────────────────────────────────────┘    │
│                                   │                                         │
│                                   ▼                                         │
│   platform_agent.go                                                         │
│   ┌───────────────────────────────────────────────────────────────────┐    │
│   │ func (p *PlatformAgent) Stream(ctx, messages) {                   │    │
│   │     // 问题1: context被忽略，没有传给orchestrator                   │    │
│   │     result, err := p.orchestrator.Execute(ctx, req)  // 阻塞!     │    │
│   │     return StreamReaderFromArray(result)  // 假流式                │    │
│   │ }                                                                 │    │
│   └───────────────────────────────────────────────────────────────────┘    │
│                                   │                                         │
│                                   ▼                                         │
│   orchestrator.go                                                           │
│   ┌───────────────────────────────────────────────────────────────────┐    │
│   │ func (o *Orchestrator) Execute(ctx, req) {                        │    │
│   │     results, err := o.executePlan(ctx, plan, req)                 │    │
│   │     // 没有流式支持                                                │    │
│   │ }                                                                 │    │
│   └───────────────────────────────────────────────────────────────────┘    │
│                                   │                                         │
│                                   ▼                                         │
│   executor.go                                                               │
│   ┌───────────────────────────────────────────────────────────────────┐    │
│   │ func (e *ExpertExecutor) ExecuteStep(ctx, step, ...) {            │    │
│   │     // 问题2: 使用Generate而非Stream                               │    │
│   │     resp, err := exp.Agent.Generate(ctx, messages)  // 非流式!    │    │
│   │     // 问题3: context传递到expert，但Generate不走tool emitter     │    │
│   │ }                                                                 │    │
│   └───────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│   结果:                                                                     │
│   ❌ 没有流式输出 - 用户等待很久                                             │
│   ❌ 没有tool_call/tool_result事件 - 没有动画                               │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 修复方案

### 方案: 单专家直接流式 + Context传递

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         修复后的架构                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   chat_handler.go                                                          │
│   ┌───────────────────────────────────────────────────────────────────┐    │
│   │ ctx = tools.WithToolEventEmitter(ctx, emitter)                    │    │
│   │ stream, err := h.svcCtx.AI.Stream(streamCtx, inputMessages)       │    │
│   └───────────────────────────────────────────────────────────────────┘    │
│                                   │                                         │
│                                   ▼                                         │
│   platform_agent.go (修复后)                                                │
│   ┌───────────────────────────────────────────────────────────────────┐    │
│   │ func (p *PlatformAgent) Stream(ctx, messages) {                   │    │
│   │     req := p.buildExecuteRequest(ctx, messages)                   │    │
│   │                                                                   │    │
│   │     // 单专家场景: 直接流式                                        │    │
│   │     if req.Decision.Strategy == StrategySingle {                  │    │
│   │         exp, _ := p.registry.GetExpert(req.Decision.PrimaryExpert)│    │
│   │         return exp.Agent.Stream(ctx, messages)  // 真流式! ✓     │    │
│   │     }                                                             │    │
│   │                                                                   │    │
│   │     // 多专家场景: 顺序流式                                        │    │
│   │     return p.orchestrator.StreamExecute(ctx, req)                 │    │
│   │ }                                                                 │    │
│   └───────────────────────────────────────────────────────────────────┘    │
│                                   │                                         │
│                                   ▼                                         │
│   orchestrator.go (新增StreamExecute)                                       │
│   ┌───────────────────────────────────────────────────────────────────┐    │
│   │ func (o *Orchestrator) StreamExecute(ctx, req) {                  │    │
│   │     // 方案: 使用channel合并多个专家的流式输出                       │    │
│   │     return o.streamPlan(ctx, plan, req)                           │    │
│   │ }                                                                 │    │
│   └───────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│   结果:                                                                     │
│   ✓ 流式输出 - 用户看到逐字输出                                              │
│   ✓ tool_call/tool_result事件 - 动画正常                                    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 详细设计

### 1. PlatformAgent.Stream() 修复

```go
func (p *PlatformAgent) Stream(ctx context.Context, messages []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
    if p == nil {
        return nil, fmt.Errorf("agent not initialized")
    }

    req := p.buildExecuteRequest(ctx, messages)

    // Case 1: 没有路由决策，使用默认agent
    if req == nil || req.Decision == nil {
        return p.Runnable.Stream(ctx, messages)
    }

    // Case 2: 单专家 - 直接流式 (关键修复)
    if req.Decision.Strategy == experts.StrategySingle {
        exp, ok := p.registry.GetExpert(req.Decision.PrimaryExpert)
        if ok && exp != nil && exp.Agent != nil {
            // 直接使用专家的Stream方法，context包含tool event emitter
            return exp.Agent.Stream(ctx, messages)
        }
    }

    // Case 3: 多专家 - 流式执行
    return p.orchestrator.StreamExecute(ctx, req)
}
```

### 2. Orchestrator.StreamExecute() 新增

```go
// StreamExecute 流式执行多专家协作
func (o *Orchestrator) StreamExecute(ctx context.Context, req *ExecuteRequest) (*schema.StreamReader[*schema.Message], error) {
    if req == nil || req.Decision == nil {
        return nil, fmt.Errorf("route decision is required")
    }

    plan := o.buildPlan(req.Decision)

    // 创建输出channel
    ch := make(chan *schema.Message, 64)

    go func() {
        defer close(ch)

        results := make([]ExpertResult, 0, len(plan.Steps))

        for i, step := range plan.Steps {
            // 流式执行每个步骤
            stream, err := o.executor.StreamStep(ctx, &step, results, req.Message)
            if err != nil {
                // 错误处理
                ch <- schema.AssistantMessage(fmt.Sprintf("专家 %s 执行失败: %v", step.ExpertName, err), nil)
                return
            }

            // 收集流式输出
            var content strings.Builder
            for {
                msg, recvErr := stream.Recv()
                if errors.Is(recvErr, io.EOF) {
                    break
                }
                if recvErr != nil {
                    break
                }
                if msg != nil {
                    ch <- msg  // 转发给前端
                    content.WriteString(msg.Content)
                }
            }

            results = append(results, ExpertResult{
                ExpertName: step.ExpertName,
                Output:     content.String(),
            })
        }

        // 聚合最终结果 (可选)
        if len(results) > 1 {
            finalResp, _ := o.aggregateResults(ctx, results, req)
            ch <- schema.AssistantMessage("\n\n---\n**综合分析:**\n"+finalResp, nil)
        }
    }()

    return schema.StreamReaderFromChannel(ch), nil
}
```

### 3. ExpertExecutor.StreamStep() 新增

```go
// StreamStep 流式执行单个专家步骤
func (e *ExpertExecutor) StreamStep(ctx context.Context, step *ExecutionStep, priorResults []ExpertResult, baseMessage string) (*schema.StreamReader[*schema.Message], error) {
    if step == nil {
        return nil, fmt.Errorf("execution step is nil")
    }

    exp, ok := e.registry.GetExpert(step.ExpertName)
    if !ok || exp == nil {
        return nil, fmt.Errorf("expert not found: %s", step.ExpertName)
    }

    if exp.Agent == nil {
        // 降级: 返回静态消息
        return schema.StreamReaderFromArray([]*schema.Message{
            schema.AssistantMessage("专家模型未初始化", nil),
        }), nil
    }

    msg := e.buildExpertMessage(step, priorResults, baseMessage)

    // 关键: 使用Stream而非Generate，context包含tool event emitter
    return exp.Agent.Stream(ctx, []*schema.Message{
        schema.UserMessage(msg),
    })
}
```

### 4. 类型定义补充

```go
// types.go 补充

// ExecuteRequest 添加History字段（如果还没有）
type ExecuteRequest struct {
    Message        string
    Decision       *RouteDecision
    RuntimeContext map[string]any
    History        []*schema.Message  // 确保存在
}

// ExecuteResult 添加流式支持
type ExecuteResult struct {
    Response string
    Traces   []ExpertTrace
    Metadata map[string]any
    Stream   *schema.StreamReader[*schema.Message]  // 可选: 流式结果
}
```

## 数据流对比

### 修复前

```
User Message
    │
    ▼
PlatformAgent.Stream()
    │
    ├─► orchestrator.Execute() ──► blocking wait
    │         │
    │         └─► executor.ExecuteStep() ──► exp.Agent.Generate() (非流式)
    │                                          │
    │                                          └─► Tool execution (无事件)
    │
    └─► StreamReaderFromArray([final_result])  ──► 假流式输出
```

### 修复后

```
User Message
    │
    ▼
PlatformAgent.Stream(ctx with emitter)
    │
    ├─► StrategySingle?
    │       │
    │       └─► exp.Agent.Stream(ctx) ──► 真流式输出 ✓
    │                 │
    │                 └─► Tool execution (触发tool_call/tool_result事件) ✓
    │
    └─► StrategySequential/Parallel?
            │
            └─► orchestrator.StreamExecute(ctx)
                      │
                      └─► executor.StreamStep(ctx) ──► exp.Agent.Stream(ctx)
                                                            │
                                                            └─► 顺序流式输出 ✓
```

## 前端事件流

修复后，前端将正常收到以下SSE事件：

```
event: meta
data: {"sessionId":"sess-xxx",...}

event: tool_call           ← 工具调用开始，触发动画
data: {"tool":"host_list_inventory","call_id":"xxx",...}

event: tool_result         ← 工具调用完成，动画结束
data: {"tool":"host_list_inventory","call_id":"xxx",...}

event: thinking_delta      ← 思考过程流式输出
data: {"contentChunk":"让我查询一下..."}

event: delta               ← 内容流式输出
data: {"contentChunk":"找到3台主机..."}

event: done
data: {"session":{...},"stream_state":"ok"}
```

## 错误处理

### 专家不可用

```go
func (p *PlatformAgent) Stream(ctx context.Context, messages []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
    // ...
    exp, ok := p.registry.GetExpert(req.Decision.PrimaryExpert)
    if !ok || exp == nil || exp.Agent == nil {
        // 降级: 使用默认agent
        return p.Runnable.Stream(ctx, messages)
    }
    // ...
}
```

### 流式中断

```go
func (o *Orchestrator) StreamExecute(ctx context.Context, req *ExecuteRequest) (*schema.StreamReader[*schema.Message], error) {
    // ...
    go func() {
        defer close(ch)
        defer func() {
            if r := recover(); r != nil {
                ch <- schema.AssistantMessage(fmt.Sprintf("执行异常: %v", r), nil)
            }
        }()
        // ...
    }()
    // ...
}
```

## 性能考虑

1. **单专家场景**: 与修复前性能一致，但用户体验提升（流式输出）
2. **多专家场景**: 新增goroutine管理，但开销可控
3. **Channel缓冲**: 使用64缓冲，避免阻塞

## 测试策略

1. **单元测试**:
   - 测试单专家流式输出
   - 测试context正确传递
   - 测试tool event触发

2. **集成测试**:
   - 测试完整对话流程
   - 测试前端事件接收

3. **回归测试**:
   - 确保多专家场景仍能工作
   - 确保Generate()方法不受影响
