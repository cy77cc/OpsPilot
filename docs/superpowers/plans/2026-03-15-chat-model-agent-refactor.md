# ChatModelAgent 架构重构实施计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将现有 plan-execute 架构重构为 `adk.NewChatModelAgent`，简化执行流程，统一审批机制。

**Architecture:** 单一 ChatModelAgent 直接管理工具，通过 Gate 包装实现统一审批拦截，移除复杂的 plan parsing 和 phase detection 逻辑。

**Tech Stack:** Go 1.26, CloudWeGo Eino ADK, Gin, GORM

---

## File Structure

| 文件 | 操作 | 职责 |
|------|------|------|
| `internal/ai/runtime/instruction.go` | 新建 | System Prompt 模板和动态渲染 |
| `internal/ai/tools/adapt.go` | 新建 | 工具到 ADK tool 的适配，含 Gate 包装 |
| `internal/ai/agents/agent.go` | 重写 | ChatModelAgent 构建 |
| `internal/ai/orchestrator.go` | 大幅简化 | 移除 plan 逻辑，简化事件处理 |
| `internal/ai/runtime/runtime.go` | 清理 | 删除废弃类型定义 |
| `internal/ai/runtime/plan_parser.go` | 删除 | 不再需要 |
| `internal/ai/runtime/phase_detector.go` | 删除 | 不再需要 |

---

## Chunk 1: 工具适配层

### Task 1: 创建工具适配器

**Files:**
- Create: `internal/ai/tools/adapt.go`
- Test: `internal/ai/tools/adapt_test.go`

- [ ] **Step 1: 创建 adapt.go 文件骨架**

```go
// Package tools 提供 AI 工具的注册、适配和执行能力。
//
// 本文件实现 ToolSpec 到 ADK InvokableTool 的适配，
// 并通过 Gate 包装实现变更类工具的统一审批拦截。
package tools

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/tool"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	approvaltools "github.com/cy77cc/OpsPilot/internal/ai/tools/approval"
)

// ADKToolAdapter 将 ToolSpec 转换为 ADK InvokableTool。
// 同时处理只读工具和变更工具的 Gate 包装。
type ADKToolAdapter struct {
	registry      *Registry
	decisionMaker *airuntime.ApprovalDecisionMaker
}

// NewADKToolAdapter 创建工具适配器。
func NewADKToolAdapter(registry *Registry, decisionMaker *airuntime.ApprovalDecisionMaker) *ADKToolAdapter {
	return &ADKToolAdapter{
		registry:      registry,
		decisionMaker: decisionMaker,
	}
}

// AdaptAll 将注册表中的所有工具转换为 ADK tool 列表。
// 变更类工具自动通过 Gate 包装实现审批拦截。
func (a *ADKToolAdapter) AdaptAll() []tool.BaseTool {
	if a.registry == nil {
		return nil
	}

	caps := a.registry.List()
	tools := make([]tool.BaseTool, 0, len(caps))

	for _, cap := range caps {
		spec, ok := a.registry.Get(cap.Name)
		if !ok {
			continue
		}

		adkTool := a.adaptTool(spec)
		tools = append(tools, adkTool)
	}

	return tools
}

// adaptTool 转换单个工具，根据模式决定是否包装 Gate。
func (a *ADKToolAdapter) adaptTool(spec ToolSpec) tool.BaseTool {
	// TODO: 实现工具转换
	return nil
}
```

- [ ] **Step 2: 实现 adaptTool 方法**

```go
// adaptTool 转换单个工具，根据模式决定是否包装 Gate。
func (a *ADKToolAdapter) adaptTool(spec ToolSpec) tool.BaseTool {
	// 创建基础工具
	baseTool := &invokableToolWrapper{
		name:        spec.Name,
		description: spec.Description,
		input:       spec.Input,
		execute:     spec.Execute,
		preview:     spec.Preview,
	}

	// 判断是否需要审批包装
	needApproval := spec.Mode == ModeMutating || spec.Risk == RiskHigh

	if !needApproval {
		return baseTool
	}

	// 包装 Gate 实现审批拦截
	approvalSpec := airuntime.ApprovalToolSpec{
		Name:        spec.Name,
		DisplayName: spec.DisplayName,
		Description: spec.Description,
		Mode:        string(spec.Mode),
		Risk:        string(spec.Risk),
		Category:    spec.Category,
	}

	return approvaltools.NewGate(baseTool, approvalSpec, a.decisionMaker, nil)
}

// invokableToolWrapper 实现 tool.InvokableTool 接口。
type invokableToolWrapper struct {
	name        string
	description string
	input       any
	execute     func(context.Context, common.PlatformDeps, map[string]any) (*Execution, error)
	preview     func(context.Context, common.PlatformDeps, map[string]any) (any, error)
}

func (w *invokableToolWrapper) Info(ctx context.Context) (*schema.ToolInfo, error) {
	paramsOneOf, err := schema.NewParamsOneOfByGoStruct(w.input)
	if err != nil {
		paramsOneOf = nil
	}

	return &schema.ToolInfo{
		Name:  w.name,
		Desc:  w.description,
		ParamsOneOf: paramsOneOf,
	}, nil
}

func (w *invokableToolWrapper) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	params := make(map[string]any)
	if strings.TrimSpace(argumentsInJSON) != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			return "", fmt.Errorf("parse arguments: %w", err)
		}
	}

	// 从 context 获取 PlatformDeps
	deps := common.PlatformDepsFromContext(ctx)
	if deps == nil {
		return "", fmt.Errorf("platform deps not found in context")
	}

	result, err := w.execute(ctx, deps, params)
	if err != nil {
		return "", err
	}

	output, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}

	return string(output), nil
}
```

- [ ] **Step 3: 添加必要的 import**

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	approvaltools "github.com/cy77cc/OpsPilot/internal/ai/tools/approval"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)
```

- [ ] **Step 4: 在 common 包添加 PlatformDepsFromContext**

**File:** `internal/ai/tools/common/common.go`

```go
// PlatformDepsFromContext 从 context 中提取 PlatformDeps。
// 如果不存在则返回 nil。
func PlatformDepsFromContext(ctx context.Context) *PlatformDeps {
	if ctx == nil {
		return nil
	}
	if v := ctx.Value(platformDepsKey{}); v != nil {
		if deps, ok := v.(*PlatformDeps); ok {
			return deps
		}
	}
	return nil
}

// ContextWithPlatformDeps 将 PlatformDeps 注入到 context。
func ContextWithPlatformDeps(ctx context.Context, deps *PlatformDeps) context.Context {
	return context.WithValue(ctx, platformDepsKey{}, deps)
}

type platformDepsKey struct{}
```

- [ ] **Step 5: 创建测试文件**

**File:** `internal/ai/tools/adapt_test.go`

```go
package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestADKToolAdapter_AdaptAll(t *testing.T) {
	registry := NewRegistry(common.PlatformDeps{})
	adapter := NewADKToolAdapter(registry, nil)

	tools := adapter.AdaptAll()
	assert.NotEmpty(t, tools, "should have tools")

	// 验证工具信息
	for _, tool := range tools {
		info, err := tool.Info(context.Background())
		require.NoError(t, err)
		assert.NotEmpty(t, info.Name)
		assert.NotEmpty(t, info.Desc)
	}
}

func TestADKToolAdapter_MutatingToolNeedsGate(t *testing.T) {
	registry := NewRegistry(common.PlatformDeps{})
	adapter := NewADKToolAdapter(registry, nil)

	spec, ok := registry.Get("service_deploy_apply")
	require.True(t, ok, "service_deploy_apply should exist")

	adapted := adapter.adaptTool(spec)
	assert.NotNil(t, adapted)

	// 验证变更工具被 Gate 包装
	info, err := adapted.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "service_deploy_apply", info.Name)
}
```

- [ ] **Step 6: 运行测试验证**

```bash
go test ./internal/ai/tools/... -v -run "TestADKToolAdapter"
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/ai/tools/adapt.go internal/ai/tools/adapt_test.go internal/ai/tools/common/common.go
git commit -m "feat(ai): add ADK tool adapter with Gate wrapper for approval"
```

---

## Chunk 2: System Prompt 模板

### Task 2: 创建 Instruction 生成器

**Files:**
- Create: `internal/ai/runtime/instruction.go`
- Test: `internal/ai/runtime/instruction_test.go`

- [ ] **Step 1: 创建 instruction.go**

```go
// Package runtime 定义 AI 运行时的核心类型和组件。
//
// 本文件提供 System Prompt 模板和动态渲染能力，
// 支持根据 RuntimeContext 注入场景、项目、选中资源等上下文信息。
package runtime

import (
	"fmt"
	"strings"
)

// InstructionTemplate 是 OpsPilot 智能运维助手的系统提示词模板。
// 占位符 {xxx} 会在运行时被 RuntimeContext 中的对应值替换。
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

// BuildInstruction 根据 RuntimeContext 渲染系统提示词。
// 空值字段会被替换为默认文本，确保模板完整有效。
func BuildInstruction(ctx RuntimeContext) string {
	result := InstructionTemplate

	// 替换场景名称
	sceneName := strings.TrimSpace(ctx.SceneName)
	if sceneName == "" {
		sceneName = "通用"
	}
	result = strings.ReplaceAll(result, "{scene_name}", sceneName)

	// 替换项目名称
	projectName := strings.TrimSpace(ctx.ProjectName)
	if projectName == "" {
		projectName = "未指定"
	}
	result = strings.ReplaceAll(result, "{project_name}", projectName)

	// 替换选中资源
	selectedResources := formatSelectedResources(ctx.SelectedResources)
	result = strings.ReplaceAll(result, "{selected_resources}", selectedResources)

	return result
}

// formatSelectedResources 格式化选中资源列表为可读文本。
func formatSelectedResources(resources []SelectedResource) string {
	if len(resources) == 0 {
		return "无"
	}

	var sb strings.Builder
	for i, r := range resources {
		if i > 0 {
			sb.WriteString(", ")
		}
		name := strings.TrimSpace(r.Name)
		if name == "" {
			name = r.ID
		}
		if r.Type != "" {
			sb.WriteString(fmt.Sprintf("%s(%s)", name, r.Type))
		} else {
			sb.WriteString(name)
		}
	}
	return sb.String()
}
```

- [ ] **Step 2: 创建测试文件**

**File:** `internal/ai/runtime/instruction_test.go`

```go
package runtime

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildInstruction_Basic(t *testing.T) {
	ctx := RuntimeContext{
		SceneName:   "主机管理",
		ProjectName: "生产环境",
	}

	result := BuildInstruction(ctx)

	assert.Contains(t, result, "场景: 主机管理")
	assert.Contains(t, result, "项目: 生产环境")
	assert.Contains(t, result, "选中资源: 无")
	assert.Contains(t, result, "你是 OpsPilot 智能运维助手")
}

func TestBuildInstruction_WithSelectedResources(t *testing.T) {
	ctx := RuntimeContext{
		SceneName:   "Kubernetes",
		ProjectName: "测试集群",
		SelectedResources: []SelectedResource{
			{Type: "pod", Name: "nginx-123", Namespace: "default"},
			{Type: "service", Name: "my-service", Namespace: "default"},
		},
	}

	result := BuildInstruction(ctx)

	assert.Contains(t, result, "场景: Kubernetes")
	assert.Contains(t, result, "项目: 测试集群")
	assert.Contains(t, result, "nginx-123(pod)")
	assert.Contains(t, result, "my-service(service)")
}

func TestBuildInstruction_EmptyContext(t *testing.T) {
	ctx := RuntimeContext{}

	result := BuildInstruction(ctx)

	assert.Contains(t, result, "场景: 通用")
	assert.Contains(t, result, "项目: 未指定")
	assert.Contains(t, result, "选中资源: 无")
}

func TestBuildInstruction_CompleteTemplate(t *testing.T) {
	ctx := RuntimeContext{
		SceneName:   "主机管理",
		ProjectName: "生产环境",
	}

	result := BuildInstruction(ctx)

	// 验证没有未替换的占位符
	assert.NotContains(t, result, "{")
	assert.NotContains(t, result, "}")

	// 验证核心内容完整
	assert.True(t, strings.Contains(result, "核心能力"))
	assert.True(t, strings.Contains(result, "工作原则"))
	assert.True(t, strings.Contains(result, "当前上下文"))
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./internal/ai/runtime/... -v -run "TestBuildInstruction"
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/ai/runtime/instruction.go internal/ai/runtime/instruction_test.go
git commit -m "feat(ai): add system prompt template with dynamic context injection"
```

---

## Chunk 3: Agent 重构

### Task 3: 重写 Agent 构建逻辑

**Files:**
- Modify: `internal/ai/agents/agent.go`

- [ ] **Step 1: 重写 agent.go**

```go
// Package agents 构建 OpsPilot Agent。
//
// 本文件使用 adk.NewChatModelAgent 创建单一 Agent，
// 直接管理工具并自动处理 tool calling loop。
// 变更类工具通过 Gate 包装实现统一审批拦截。
package agents

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	aitools "github.com/cy77cc/OpsPilot/internal/ai/tools"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

// Deps 是 Agent 所需的外部依赖。
type Deps struct {
	PlatformDeps     common.PlatformDeps         // 平台服务依赖（数据库、外部 API 等）
	DecisionMaker    *airuntime.ApprovalDecisionMaker // 审批决策器
}

// NewAgent 构建并返回 ChatModelAgent。
// Agent 直接管理工具，变更类工具通过 Gate 包装实现审批拦截。
func NewAgent(ctx context.Context, deps Deps) (adk.ResumableAgent, error) {
	// 创建 ChatModel
	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: false,
		Temp:     0.2,
	})
	if err != nil {
		return nil, err
	}

	// 创建工具注册表
	registry := aitools.NewRegistry(deps.PlatformDeps)

	// 适配工具为 ADK 格式，变更工具自动包装 Gate
	adapter := aitools.NewADKToolAdapter(registry, deps.DecisionMaker)
	tools := adapter.AdaptAll()

	// 创建 ChatModelAgent
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "OpsPilotAgent",
		Instruction:   "", // 由 Orchestrator 动态注入
		Model:         model,
		ToolsConfig:   adk.ToolsConfig{Tools: tools},
		MaxIterations: 20,
	})
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./internal/ai/...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/ai/agents/agent.go
git commit -m "refactor(ai): rewrite agent using ChatModelAgent with tool management"
```

---

## Chunk 4: Orchestrator 简化

### Task 4: 简化 Orchestrator 事件处理

**Files:**
- Modify: `internal/ai/orchestrator.go`

- [ ] **Step 1: 简化 NewOrchestrator**

移除 `contextProcessor`、`sceneResolver` 字段，简化初始化逻辑。

```go
// NewOrchestrator 创建并初始化 Orchestrator。
func NewOrchestrator(_ any, executionStore *airuntime.ExecutionStore, deps common.PlatformDeps) *Orchestrator {
	ctx := context.Background()
	checkpointStore := airuntime.NewCheckpointStore(nil, "")

	// 创建审批决策器
	sceneResolver := airuntime.NewSceneConfigResolver(nil)
	approvals := airuntime.NewApprovalDecisionMaker(airuntime.ApprovalDecisionMakerOptions{
		ResolveScene: sceneResolver.Resolve,
	})

	summaries := approvaltools.NewSummaryRenderer()

	if executionStore == nil {
		executionStore = airuntime.NewExecutionStore(nil, "")
	}

	// 创建 Agent
	agent, err := agents.NewAgent(ctx, agents.Deps{
		PlatformDeps:  deps,
		DecisionMaker: approvals,
	})
	if err != nil {
		return &Orchestrator{
			executions: executionStore,
			checkpoints: checkpointStore,
			converter:   airuntime.NewSSEConverter(),
			approvals:   approvals,
			summaries:   summaries,
		}
	}

	return &Orchestrator{
		runner: adk.NewRunner(ctx, adk.RunnerConfig{
			Agent:           agent,
			CheckPointStore: checkpointStore,
			EnableStreaming: true,
		}),
		checkpoints: checkpointStore,
		executions:  executionStore,
		converter:   airuntime.NewSSEConverter(),
		approvals:   approvals,
		summaries:   summaries,
	}
}
```

- [ ] **Step 2: 简化 Orchestrator 结构体**

移除 `contextProcessor`、`sceneResolver` 字段：

```go
// Orchestrator 封装 ADK Runner，提供流式执行与审批恢复能力。
type Orchestrator struct {
	runner      *adk.Runner                       // ADK 运行器；nil 表示模型不可用
	checkpoints *airuntime.CheckpointStore        // 保存 Agent 断点，支持审批后续跑
	executions  *airuntime.ExecutionStore         // 保存每次执行的状态快照
	converter   *airuntime.SSEConverter           // 将 Agent 事件转换为标准 SSE StreamEvent
	approvals   *airuntime.ApprovalDecisionMaker  // 判断工具调用是否需要人工审批
	summaries   *approvaltools.SummaryRenderer    // 生成审批请求的人类可读摘要
}
```

- [ ] **Step 3: 简化 Run 方法**

移除 scene resolution 逻辑，直接使用 RuntimeContext：

```go
func (o *Orchestrator) Run(ctx context.Context, req airuntime.RunRequest, emit airuntime.StreamEmitter) error {
	startedAt := time.Now().UTC()
	if o == nil || o.runner == nil {
		return fmt.Errorf("orchestrator runner is nil")
	}
	if strings.TrimSpace(req.Message) == "" {
		return fmt.Errorf("message is empty")
	}
	if emit == nil {
		emit = func(airuntime.StreamEvent) bool { return true }
	}

	sessionID := firstNonEmpty(req.SessionID, uuid.NewString())
	planID := uuid.NewString()
	turnID := uuid.NewString()
	checkpointID := uuid.NewString()

	// 构建动态 instruction
	instruction := airuntime.BuildInstruction(req.RuntimeContext)

	// 构建 session values
	adkValues := map[string]any{
		airuntime.SessionKeyRuntimeContext: req.RuntimeContext,
		airuntime.SessionKeySessionID:      sessionID,
		airuntime.SessionKeyPlanID:         planID,
		airuntime.SessionKeyTurnID:         turnID,
	}

	// 初始化执行状态
	state := airuntime.ExecutionState{
		TraceID:        uuid.NewString(),
		SessionID:      sessionID,
		PlanID:         planID,
		TurnID:         turnID,
		Message:        req.Message,
		Scene:          req.RuntimeContext.Scene,
		Status:         airuntime.ExecutionStatusRunning,
		Phase:          "running",
		RuntimeContext: req.RuntimeContext,
		CheckpointID:   checkpointID,
		Steps:          map[string]airuntime.StepState{},
	}
	_ = o.executions.Save(ctx, state)

	// 发送 meta 事件
	emit(airuntime.StreamEvent{Type: airuntime.EventMeta, Data: map[string]any{
		"session_id": sessionID,
		"plan_id":    planID,
		"turn_id":    turnID,
		"trace_id":   state.TraceID,
	}})

	// 调用 Runner
	iter := o.runner.Query(ctx, strings.TrimSpace(req.Message),
		adk.WithCheckPointID(checkpointID),
		adk.WithSessionValues(adkValues),
		adk.WithInstruction(instruction),
	)

	_, err := o.streamExecution(ctx, iter, &state, emit)
	aiobs.ObserveAgentExecution(aiobs.ExecutionRecord{
		Operation: "run",
		Scene:     req.RuntimeContext.Scene,
		Status:    statusFromExecutionState(state.Status, err),
		Duration:  time.Since(startedAt),
	})
	return err
}
```

- [ ] **Step 4: 大幅简化 streamExecution 方法**

移除所有 plan parsing、phase detection、chain node 管理逻辑：

```go
func (o *Orchestrator) streamExecution(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent], state *airuntime.ExecutionState, emit airuntime.StreamEmitter) (*airuntime.ResumeResult, error) {
	if iter == nil {
		return nil, fmt.Errorf("event iterator is nil")
	}
	if emit == nil {
		emit = func(airuntime.StreamEvent) bool { return true }
	}

	var lastText string
	chainStarted := false
	chainStartedAt := time.Time{}

	emitChainStarted := func() {
		if chainStarted {
			return
		}
		emit(o.converter.OnChainStarted(state.TurnID))
		chainStarted = true
		chainStartedAt = time.Now().UTC()
	}

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			emit(o.converter.OnError(state.Phase, event.Err))
			return nil, event.Err
		}

		// 处理文本输出
		contents, err := eventTextContents(event)
		if err != nil {
			return nil, fmt.Errorf("failed to parse agent message chunks: %w", err)
		}

		for _, content := range contents {
			if strings.TrimSpace(content) == "" {
				continue
			}
			emitChainStarted()
			emit(airuntime.StreamEvent{
				Type: airuntime.EventDelta,
				Data: map[string]any{"content": content},
			})
			lastText = mergeTextProgress(lastText, content)
		}

		// 处理工具调用事件
		if isToolOutputEvent(event) {
			emitChainStarted()
			toolName := strings.TrimSpace(event.Output.MessageOutput.ToolName)
			toolResult := strings.TrimSpace(strings.Join(contents, ""))

			emit(airuntime.StreamEvent{
				Type: airuntime.EventToolResult,
				Data: map[string]any{
					"tool_name": toolName,
					"result":    toolResult,
				},
			})
		}

		// 处理审批中断
		if event.Action != nil && event.Action.Interrupted != nil {
			return o.handleInterrupt(ctx, event, state, emit)
		}
	}

	// 执行完成
	state.Status = airuntime.ExecutionStatusCompleted
	_ = o.executions.Save(ctx, *state)

	if chainStarted {
		emit(o.converter.OnChainCompleted(state.TurnID, string(state.Status)))
		if !chainStartedAt.IsZero() {
			aiobs.ObserveThoughtChain(aiobs.ThoughtChainRecord{
				Scene:    state.Scene,
				Status:   string(state.Status),
				Duration: time.Since(chainStartedAt),
			})
		}
	}

	emit(o.converter.OnDone(string(state.Status)))

	return &airuntime.ResumeResult{
		Resumed:   true,
		SessionID: state.SessionID,
		PlanID:    state.PlanID,
		TurnID:    state.TurnID,
		Status:    string(state.Status),
		Message:   lastNonEmpty(lastText, "执行完成。"),
	}, nil
}

// handleInterrupt 处理审批中断事件。
func (o *Orchestrator) handleInterrupt(ctx context.Context, event *adk.AgentEvent, state *airuntime.ExecutionState, emit airuntime.StreamEmitter) (*airuntime.ResumeResult, error) {
	info := interruptApprovalInfo(event)
	stepID := uuid.NewString()

	pending := &airuntime.PendingApproval{
		ID:              uuid.NewString(),
		PlanID:          state.PlanID,
		StepID:          stepID,
		CheckpointID:    state.CheckpointID,
		Target:          stepID,
		Status:          "pending",
		Title:           firstNonEmpty(info.ToolDisplayName, info.ToolName, "待确认步骤"),
		Mode:            firstNonEmpty(info.Mode, "mutating"),
		Risk:            firstNonEmpty(info.RiskLevel, "medium"),
		Summary:         firstNonEmpty(info.Summary, "执行到敏感步骤，需要确认后继续。"),
		ToolName:        info.ToolName,
		ToolDisplayName: info.ToolDisplayName,
		Params:          info.Params,
		ArgumentsInJSON: info.ArgumentsInJSON,
		CreatedAt:       time.Now().UTC(),
		ExpiresAt:       time.Now().UTC().Add(24 * time.Hour),
	}

	state.Status = airuntime.ExecutionStatusWaitingApproval
	state.PendingApproval = pending
	_ = o.checkpoints.BindIdentity(ctx, state.SessionID, state.PlanID, stepID, state.CheckpointID, stepID)
	_ = o.executions.Save(ctx, *state)

	// 发送审批事件
	emit(airuntime.StreamEvent{
		Type: airuntime.EventChainPaused,
		Data: map[string]any{
			"turn_id": state.TurnID,
			"reason":  "waiting_approval",
			"approval": map[string]any{
				"id":                pending.ID,
				"plan_id":           pending.PlanID,
				"step_id":           pending.StepID,
				"title":             pending.Title,
				"tool_name":         pending.ToolName,
				"tool_display_name": pending.ToolDisplayName,
				"risk":              pending.Risk,
				"summary":           pending.Summary,
				"arguments_json":    pending.ArgumentsInJSON,
			},
		},
	})
	emit(o.converter.OnDone(string(state.Status)))

	return &airuntime.ResumeResult{
		Interrupted: true,
		SessionID:   state.SessionID,
		PlanID:      state.PlanID,
		StepID:      stepID,
		TurnID:      state.TurnID,
		Status:      string(state.Status),
		Message:     "执行已中断，等待审批。",
	}, nil
}
```

- [ ] **Step 5: 删除废弃的辅助函数**

删除以下函数：
- `stepsToChainDetails`
- `planStepsToStructured`
- `toolResultChainPayload`
- `rowsFromPayload`
- `inferStructuredResource`
- `stepEventFromState`
- `claimStepForTool`
- `sortedStepIDs`

- [ ] **Step 6: 验证编译**

```bash
go build ./internal/ai/...
```

Expected: no errors

- [ ] **Step 7: Commit**

```bash
git add internal/ai/orchestrator.go
git commit -m "refactor(ai): simplify orchestrator by removing plan parsing and phase detection"
```

---

## Chunk 5: 清理旧代码

### Task 5: 删除废弃文件

**Files:**
- Delete: `internal/ai/runtime/plan_parser.go`
- Delete: `internal/ai/runtime/phase_detector.go`
- Modify: `internal/ai/runtime/runtime.go`

- [ ] **Step 1: 删除 plan_parser.go**

```bash
git rm internal/ai/runtime/plan_parser.go
```

- [ ] **Step 2: 删除 phase_detector.go**

```bash
git rm internal/ai/runtime/phase_detector.go
```

- [ ] **Step 3: 清理 runtime.go 中的废弃类型**

从 `runtime.go` 中删除以下类型定义：
- `PhaseName` 及其常量
- `PhaseEvent`
- `PlanEvent`
- `PlanStep`
- `StepEvent`
- `ReplanEvent`
- `ChainNodeKind` 及其常量
- `ChainNodeInfo`
- `FinalAnswerEvent`

保留以下类型：
- `EventType` 及其常量（SSE 事件类型）
- `StreamEvent`
- `StreamEmitter`
- `Runtime` 接口
- `RunRequest`
- `RuntimeContext`
- `SelectedResource`
- `ResumeRequest`
- `ResumeResult`
- `SceneConfig` 及相关类型
- `ExecutionStatus` 及其常量
- `StepStatus` 及其常量
- `StepState`
- `PendingApproval`
- `ExecutionState`
- `ExecutionStore`
- `CheckpointStore`

- [ ] **Step 4: 验证编译**

```bash
go build ./internal/ai/...
```

Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor(ai): remove deprecated plan parsing and phase detection code"
```

---

## Chunk 6: 测试与验证

### Task 6: 更新测试

**Files:**
- Modify: `internal/ai/orchestrator_test.go`
- Run: Integration tests

- [ ] **Step 1: 更新 orchestrator_test.go**

移除与 plan parsing、phase detection 相关的测试用例，保留：
- Agent 创建测试
- 审批中断恢复测试
- SSE 事件流测试

- [ ] **Step 2: 运行所有 AI 模块测试**

```bash
go test ./internal/ai/... -v
```

Expected: All tests pass

- [ ] **Step 3: 运行完整测试套件**

```bash
make test
```

Expected: All tests pass

- [ ] **Step 4: 手动验证审批流程**

启动服务，测试：
1. 发起对话，触发变更工具调用
2. 验证审批中断事件
3. 验证审批通过后恢复执行
4. 验证审批拒绝后停止执行

- [ ] **Step 5: 最终 Commit**

```bash
git add -A
git commit -m "test(ai): update tests for ChatModelAgent architecture"
```

---

## Summary

| Phase | 文件 | 变更 |
|-------|------|------|
| Chunk 1 | `tools/adapt.go` | 新建工具适配器 |
| Chunk 2 | `runtime/instruction.go` | 新建 System Prompt 模板 |
| Chunk 3 | `agents/agent.go` | 重写为 ChatModelAgent |
| Chunk 4 | `orchestrator.go` | 大幅简化事件处理 |
| Chunk 5 | `runtime/*.go` | 删除废弃代码 |
| Chunk 6 | `*_test.go` | 更新测试验证 |
