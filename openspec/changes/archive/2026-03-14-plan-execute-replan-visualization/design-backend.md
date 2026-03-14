# 后端实现方案

## 1. 当前架构分析

### 1.1 现有文件结构

```
internal/ai/
├── orchestrator.go          # 编排核心，处理 SSE 流
├── orchestrator_test.go     # 测试
├── agents/
│   ├── agent.go             # Agent 组装
│   ├── planner/             # 规划器
│   ├── executor/            # 执行器
│   └── replan/              # 重规划器
├── runtime/
│   ├── runtime.go           # 运行时类型定义
│   ├── sse_converter.go     # SSE 事件转换
│   └── context_processor.go # Prompt 模板处理
├── events/
│   └── events.go            # 事件类型常量
└── tools/
    ├── common/              # 公共类型
    └── approval/
        └── gate.go          # 审批门控
```

### 1.2 现有 SSE 事件

```go
// internal/ai/events/events.go
const (
    EventMeta              Name = "meta"
    EventDelta             Name = "delta"
    EventThinkingDelta     Name = "thinking_delta"
    EventToolCall          Name = "tool_call"          // 已定义但未使用
    EventToolResult        Name = "tool_result"        // 已定义但未使用
    EventApprovalRequired  Name = "approval_required"
    EventDone              Name = "done"
    EventError             Name = "error"
)
```

### 1.3 差距分析

| 差距 | 说明 | 解决方案 |
|------|------|---------|
| 缺少阶段事件 | 无法区分 planning/executing/replanning | 新增 `phase_started`/`phase_complete` |
| 计划未发送 | `plan_created` 已定义但从未调用 | 从 Planner 输出解析并发送 |
| 工具事件未使用 | `tool_call`/`tool_result` 已定义但未调用 | 在 Executor 和 Gate 中发送 |
| 阶段检测缺失 | 无法从 ADK 事件推断当前阶段 | 实现 phase_detector |

---

## 2. 新增事件定义

### 2.1 事件常量

```go
// internal/ai/events/events.go

const (
    // === 现有事件 ===
    EventMeta              Name = "meta"
    EventDelta             Name = "delta"
    EventThinkingDelta     Name = "thinking_delta"
    EventToolCall          Name = "tool_call"
    EventToolResult        Name = "tool_result"
    EventApprovalRequired  Name = "approval_required"
    EventDone              Name = "done"
    EventError             Name = "error"

    // === 新增: 阶段生命周期 ===
    EventPhaseStarted      Name = "phase_started"
    EventPhaseComplete     Name = "phase_complete"

    // === 新增: Plan 阶段 ===
    EventPlanGenerated     Name = "plan_generated"

    // === 新增: Execute 阶段 ===
    EventStepStarted       Name = "step_started"
    EventStepComplete      Name = "step_complete"

    // === 新增: Replan 阶段 ===
    EventReplanTriggered   Name = "replan_triggered"
    EventReplanComplete    Name = "replan_complete"
)
```

### 2.2 事件数据结构

```go
// internal/ai/runtime/runtime.go

// PlanStep 规划步骤
type PlanStep struct {
    ID       string `json:"id"`
    Content  string `json:"content"`
    ToolHint string `json:"tool_hint,omitempty"`
}

// PhaseInfo 阶段信息
type PhaseInfo struct {
    Phase  string `json:"phase"`  // planning, executing, replanning
    Title  string `json:"title"`
    Status string `json:"status"` // loading, success, error
}

// StepInfo 步骤信息
type StepInfo struct {
    StepID    string                 `json:"step_id"`
    Title     string                 `json:"title"`
    ToolName  string                 `json:"tool_name,omitempty"`
    Params    map[string]any         `json:"params,omitempty"`
    Status    string                 `json:"status"`
    StartedAt string                 `json:"started_at,omitempty"`
    Summary   string                 `json:"summary,omitempty"`
}

// ToolCallInfo 工具调用信息
type ToolCallInfo struct {
    StepID    string         `json:"step_id"`
    ToolName  string         `json:"tool_name"`
    Arguments map[string]any `json:"arguments"`
}

// ToolResultInfo 工具结果信息
type ToolResultInfo struct {
    StepID    string `json:"step_id"`
    ToolName  string `json:"tool_name"`
    Result    string `json:"result"`
    Status    string `json:"status"` // success, error
}

// ReplanInfo 重规划信息
type ReplanInfo struct {
    Reason         string `json:"reason"`
    CompletedSteps int    `json:"completed_steps"`
}
```

---

## 3. SSEConverter 增强

### 3.1 新增方法

```go
// internal/ai/runtime/sse_converter.go

// OnPhaseStarted 发送阶段开始事件
func (c *SSEConverter) OnPhaseStarted(phase, title string) StreamEvent {
    return StreamEvent{
        Type: EventPhaseStarted,
        Data: PhaseInfo{
            Phase:  phase,
            Title:  title,
            Status: "loading",
        },
    }
}

// OnPhaseComplete 发送阶段完成事件
func (c *SSEConverter) OnPhaseComplete(phase, status string) StreamEvent {
    return StreamEvent{
        Type: EventPhaseComplete,
        Data: PhaseInfo{
            Phase:  phase,
            Status: status,
        },
    }
}

// OnPlanGenerated 发送计划生成事件
func (c *SSEConverter) OnPlanGenerated(planID string, steps []PlanStep) []StreamEvent {
    return []StreamEvent{
        {
            Type: EventPhaseComplete,
            Data: PhaseInfo{Phase: "planning", Status: "success"},
        },
        {
            Type: EventPlanGenerated,
            Data: map[string]any{
                "plan_id": planID,
                "steps":   steps,
                "total":   len(steps),
            },
        },
    }
}

// OnStepStarted 发送步骤开始事件
func (c *SSEConverter) OnStepStarted(info *StepInfo) StreamEvent {
    info.Status = "running"
    info.StartedAt = time.Now().UTC().Format(time.RFC3339)
    return StreamEvent{
        Type: EventStepStarted,
        Data: info,
    }
}

// OnToolCall 发送工具调用事件
func (c *SSEConverter) OnToolCall(info *ToolCallInfo) StreamEvent {
    return StreamEvent{
        Type: EventToolCall,
        Data: info,
    }
}

// OnToolResult 发送工具结果事件
func (c *SSEConverter) OnToolResult(info *ToolResultInfo) StreamEvent {
    return StreamEvent{
        Type: EventToolResult,
        Data: info,
    }
}

// OnStepComplete 发送步骤完成事件
func (c *SSEConverter) OnStepComplete(stepID, status, summary string) StreamEvent {
    return StreamEvent{
        Type: EventStepComplete,
        Data: StepInfo{
            StepID:   stepID,
            Status:   status,
            Summary:  summary,
        },
    }
}

// OnReplanTriggered 发送重规划触发事件
func (c *SSEConverter) OnReplanTriggered(info *ReplanInfo) StreamEvent {
    return StreamEvent{
        Type: EventReplanTriggered,
        Data: info,
    }
}
```

---

## 4. 阶段检测器

### 4.1 实现方案

```go
// internal/ai/runtime/phase_detector.go

package runtime

import (
    "strings"

    "github.com/cloudwego/eino/adk"
)

// PhaseDetector 从 ADK 事件推断当前执行阶段
type PhaseDetector struct {
    currentPhase string
    stepCounter  int
}

// NewPhaseDetector 创建阶段检测器
func NewPhaseDetector() *PhaseDetector {
    return &PhaseDetector{
        currentPhase: "planning", // 初始阶段
    }
}

// Detect 从事件推断当前阶段
// 返回: 阶段名称 (planning/executing/replanning)
func (d *PhaseDetector) Detect(event *adk.AgentEvent) string {
    if event == nil {
        return d.currentPhase
    }

    // 方法1: 从 RunPath 推断
    if path := strings.ToLower(event.RunPath); path != "" {
        switch {
        case strings.Contains(path, "planner") && !strings.Contains(path, "replanner"):
            return "planning"
        case strings.Contains(path, "executor"):
            return "executing"
        case strings.Contains(path, "replanner"):
            return "replanning"
        }
    }

    // 方法2: 从 AgentName 推断
    if name := strings.ToLower(event.AgentName); name != "" {
        switch {
        case name == "planner":
            return "planning"
        case name == "executor":
            return "executing"
        case name == "replanner":
            return "replanning"
        }
    }

    // 方法3: 从消息内容特征推断
    if content := extractTextContent(event); content != "" {
        if strings.Contains(content, `"steps"`) && strings.Contains(content, `"["`) {
            // 可能是计划输出
            if d.currentPhase == "planning" {
                return "planning"
            }
        }
    }

    return d.currentPhase
}

// SetPhase 设置当前阶段
func (d *PhaseDetector) SetPhase(phase string) {
    d.currentPhase = phase
}

// NextStepID 生成下一个步骤ID
func (d *PhaseDetector) NextStepID() string {
    d.stepCounter++
    return fmt.Sprintf("step-%d", d.stepCounter)
}

// 辅助函数
func extractTextContent(event *adk.AgentEvent) string {
    if event.Output == nil || event.Output.MessageOutput == nil {
        return ""
    }

    msg := event.Output.MessageOutput.Message
    if msg == nil {
        return ""
    }

    return msg.Content
}
```

---

## 5. 计划解析器

### 5.1 实现方案

```go
// internal/ai/runtime/plan_parser.go

package runtime

import (
    "encoding/json"
    "strings"

    "github.com/cloudwego/eino/adk"
)

// PlanParser 从 LLM 输出解析结构化计划
type PlanParser struct{}

// Parse 从 Planner 输出解析计划步骤
func (p *PlanParser) Parse(event *adk.AgentEvent) *ParsedPlan {
    if event == nil || event.Output == nil {
        return nil
    }

    content := extractTextContent(event)
    if content == "" {
        return nil
    }

    // 尝试从 JSON 中提取
    if plan := p.parseJSON(content); plan != nil {
        return plan
    }

    // 尝试从文本中提取编号列表
    if plan := p.parseNumberedList(content); plan != nil {
        return plan
    }

    return nil
}

// ParsedPlan 解析后的计划
type ParsedPlan struct {
    PlanID string     `json:"plan_id"`
    Steps  []PlanStep `json:"steps"`
}

// parseJSON 尝试解析 JSON 格式的计划
func (p *PlanParser) parseJSON(content string) *ParsedPlan {
    // 查找 JSON 对象
    start := strings.Index(content, "{")
    if start == -1 {
        return nil
    }

    end := strings.LastIndex(content, "}")
    if end == -1 || end <= start {
        return nil
    }

    jsonStr := content[start : end+1]

    // 尝试解析为步骤对象
    var stepObj struct {
        Steps []string `json:"steps"`
    }
    if err := json.Unmarshal([]byte(jsonStr), &stepObj); err == nil && len(stepObj.Steps) > 0 {
        plan := &ParsedPlan{PlanID: generatePlanID()}
        for i, s := range stepObj.Steps {
            plan.Steps = append(plan.Steps, PlanStep{
                ID:      fmt.Sprintf("step-%d", i+1),
                Content: s,
            })
        }
        return plan
    }

    return nil
}

// parseNumberedList 尝试解析编号列表
func (p *PlanParser) parseNumberedList(content string) *ParsedPlan {
    lines := strings.Split(content, "\n")
    var steps []PlanStep

    for i, line := range lines {
        line = strings.TrimSpace(line)
        // 匹配 "1. xxx" 或 "1) xxx" 或 "- xxx"
        if matched, step := parseStepLine(line); matched {
            steps = append(steps, PlanStep{
                ID:      fmt.Sprintf("step-%d", len(steps)+1),
                Content: step,
            })
        }
    }

    if len(steps) == 0 {
        return nil
    }

    return &ParsedPlan{
        PlanID: generatePlanID(),
        Steps:  steps,
    }
}

func parseStepLine(line string) (bool, string) {
    // 匹配 "1. xxx" 格式
    if idx := strings.Index(line, ". "); idx > 0 && idx < 5 {
        return true, strings.TrimSpace(line[idx+2:])
    }
    // 匹配 "1) xxx" 格式
    if idx := strings.Index(line, ") "); idx > 0 && idx < 5 {
        return true, strings.TrimSpace(line[idx+2:])
    }
    // 匹配 "- xxx" 格式
    if strings.HasPrefix(line, "- ") {
        return true, strings.TrimSpace(line[2:])
    }
    return false, ""
}

func generatePlanID() string {
    return fmt.Sprintf("plan-%d", time.Now().UnixNano())
}
```

---

## 6. Orchestrator 修改

### 6.1 streamExecution 方法增强

```go
// internal/ai/orchestrator.go

func (o *Orchestrator) streamExecution(
    ctx context.Context,
    iter *adk.AsyncIterator[*adk.AgentEvent],
    state *airuntime.ExecutionState,
    emit airuntime.StreamEmitter,
) (*airuntime.ResumeResult, error) {

    detector := runtime.NewPhaseDetector()
    planParser := &runtime.PlanParser{}
    lastPhase := ""
    stepIDMap := make(map[string]string) // toolCallID -> stepID

    for {
        event, ok := iter.Next()
        if !ok {
            break
        }

        // === 1. 错误处理 ===
        if event == nil || event.Err != nil {
            if event != nil && event.Err != nil {
                emit(o.converter.OnError(event.Err))
            }
            continue
        }

        // === 2. 阶段检测 ===
        currentPhase := detector.Detect(event)
        if currentPhase != lastPhase {
            emit(o.converter.OnPhaseStarted(currentPhase, phaseTitle(currentPhase)))
            lastPhase = currentPhase
            state.Phase = currentPhase
        }

        // === 3. 文本增量 ===
        if contents := extractTextChunks(event); len(contents) > 0 {
            for _, c := range contents {
                emit(o.converter.OnTextDelta(c))
            }
        }

        // === 4. 工具调用检测 ===
        if toolCalls := extractToolCalls(event); len(toolCalls) > 0 {
            for _, tc := range toolCalls {
                stepID := detector.NextStepID()
                stepIDMap[tc.ID] = stepID

                emit(o.converter.OnStepStarted(&runtime.StepInfo{
                    StepID:   stepID,
                    Title:    tc.Description,
                    ToolName: tc.Name,
                    Params:   tc.Arguments,
                }))

                emit(o.converter.OnToolCall(&runtime.ToolCallInfo{
                    StepID:    stepID,
                    ToolName:  tc.Name,
                    Arguments: tc.Arguments,
                }))
            }
        }

        // === 5. 工具结果检测 ===
        if toolResult := extractToolResult(event); toolResult != nil {
            stepID := stepIDMap[toolResult.ToolCallID]
            if stepID == "" {
                stepID = detector.NextStepID()
            }

            emit(o.converter.OnToolResult(&runtime.ToolResultInfo{
                StepID:    stepID,
                ToolName:  toolResult.ToolName,
                Result:    toolResult.Content,
                Status:    "success",
            }))

            emit(o.converter.OnStepComplete(stepID, "success", truncateSummary(toolResult.Content)))
        }

        // === 6. 计划解析 (Planning 阶段) ===
        if currentPhase == "planning" {
            if plan := planParser.Parse(event); plan != nil {
                events := o.converter.OnPlanGenerated(plan.PlanID, plan.Steps)
                for _, e := range events {
                    emit(e)
                }
            }
        }

        // === 7. Replan 检测 ===
        if isReplanTriggered(event) {
            emit(o.converter.OnReplanTriggered(&runtime.ReplanInfo{
                Reason:         "根据执行结果调整计划",
                CompletedSteps: len(state.Steps),
            }))
        }

        // === 8. 中断处理 (审批) ===
        if event.Action != nil && event.Action.Interrupted != nil {
            // 现有审批逻辑...
            result := o.handleInterrupt(ctx, event, state, emit)
            if result != nil {
                return result, nil
            }
        }
    }

    // === 9. 完成 ===
    emit(o.converter.OnDone("completed"))
    return nil, nil
}

// 辅助函数
func phaseTitle(phase string) string {
    titles := map[string]string{
        "planning":   "整理执行步骤",
        "executing":  "执行步骤",
        "replanning": "动态调整计划",
    }
    return titles[phase]
}
```

---

## 7. 测试用例

### 7.1 单元测试

```go
// internal/ai/runtime/phase_detector_test.go

func TestPhaseDetector_Detect(t *testing.T) {
    tests := []struct {
        name      string
        event     *adk.AgentEvent
        wantPhase string
    }{
        {
            name: "planner from runpath",
            event: &adk.AgentEvent{
                RunPath: "[{plan_execute_replan} {planner}]",
            },
            wantPhase: "planning",
        },
        {
            name: "executor from agent name",
            event: &adk.AgentEvent{
                AgentName: "Executor",
            },
            wantPhase: "executing",
        },
        {
            name: "replanner detection",
            event: &adk.AgentEvent{
                RunPath: "[{plan_execute_replan} {planner} {execute_replan} {replanner}]",
            },
            wantPhase: "replanning",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            d := NewPhaseDetector()
            got := d.Detect(tt.event)
            assert.Equal(t, tt.wantPhase, got)
        })
    }
}
```

---

## 8. 文件变更汇总

| 文件 | 操作 | 主要变更 |
|------|------|---------|
| `internal/ai/events/events.go` | 修改 | +7 事件常量 |
| `internal/ai/runtime/runtime.go` | 修改 | +5 数据结构 |
| `internal/ai/runtime/sse_converter.go` | 修改 | +7 转换方法 |
| `internal/ai/runtime/phase_detector.go` | 新建 | 阶段检测器 |
| `internal/ai/runtime/plan_parser.go` | 新建 | 计划解析器 |
| `internal/ai/orchestrator.go` | 修改 | streamExecution 增强 |
| `internal/ai/runtime/phase_detector_test.go` | 新建 | 单元测试 |
| `internal/ai/runtime/plan_parser_test.go` | 新建 | 单元测试 |
