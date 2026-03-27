# Custom Plan-Execute-Replan Implementation

**Date:** 2026-03-23
**Status:** Draft
**Author:** Claude

## Overview

Replace the Eino `planexecute` prebuilt components with a custom implementation that provides full control over Planner, Executor, and Replanner behavior. This enables richer plan structures, dynamic tool selection, step-level timeouts, convergence detection, rollback awareness, and reusable step templates.

## Background

### Current Implementation

The existing implementation uses CloudWeGo Eino's prebuilt `planexecute` package:

```go
planner, _ := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
    ToolCallingChatModel: model,
    GenInputFn:           genPlannerInputFn,
})

executor, _ := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
    Model:         model,
    ToolsConfig:   adk.ToolsConfig{...},
    GenInputFn:    genExecutorInputFn,
})

replanner, _ := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
    ChatModel: model,
})
```

### Limitations

1. **Planner**: Steps are just strings — no metadata, dependencies, conditions, or templates
2. **Executor**: Fixed tool set per agent — no dynamic selection, no step-level timeouts
3. **Replanner**: Supports basic replan/finish decisions, but lacks convergence scoring, rollback awareness, and step optimization hooks

### Goals

- Rich step metadata: ID, priority, risk level, timeout, required tools, dependencies
- Conditional/branching plans: steps with conditions (if X then Y else Z)
- Step templates: reusable patterns for common operations
- Step-level timeout control with configurable defaults
- Dynamic tool selection based on step requirements
- Checkpoint at step boundaries for reliable resume
- Convergence detection to stop replanning when goal is achieved
- Step merging/splitting for plan optimization
- Rollback awareness for safer change operations
- Context-aware replanning with external signals (alerts, user feedback)

## Design

### Package Structure

```
internal/ai/planexecute/
├── plan.go              # Plan, Step, StepTemplate types
├── context.go           # ExecutionContext, ExecutionHistory, Checkpoint
├── planner.go           # Planner implementation
├── executor.go          # Executor implementation
├── replanner.go         # Replanner implementation
├── agent.go             # BuildPlanExecuteAgent() - composes all three
├── convergence.go       # ConvergenceChecker
├── optimizer.go         # StepOptimizer (merge/split)
├── rollback.go          # RollbackDetector
├── templates/
│   ├── registry.go      # Template registry
│   ├── diagnosis.go     # Diagnosis step templates
│   └── change.go        # Change operation templates
└── validator.go         # Plan validation logic
```

### Core Types

#### Plan and Step

```go
// Plan represents an executable plan with rich metadata
type Plan struct {
    ID           string            // Unique plan ID (for plan tracking)
    Goal         string            // User's original objective
    Steps        []*Step           // Ordered steps with metadata
    Variables    map[string]any    // Plan-level variables (cluster_id, namespace, etc.)
    CreatedAt    time.Time
    Status       PlanStatus        // pending, executing, completed, failed, rolled_back
}

// Step represents a single execution unit
type Step struct {
    ID           string          // Unique step ID
    Title        string          // Human-readable title
    Description  string          // Detailed description
    Status       StepStatus      // pending, active, completed, failed, skipped, timeout

    // Rich metadata
    Priority     int             // Execution priority (higher = more important)
    RiskLevel    RiskLevel       // low, medium, high, critical
    Timeout      time.Duration   // Step-specific timeout (0 = use default)
    RequiredTools []string       // Tools needed for this step

    // Dependencies & Conditions
    DependsOn    []string        // Step IDs that must complete first
    Condition    *Condition      // Optional: conditional execution

    // Execution state
    Result       string          // Execution result (populated after execution)
    Error        error           // Execution error (if any)
    StartedAt    *time.Time
    CompletedAt  *time.Time
}

// Condition enables branching logic
type Condition struct {
    Type      ConditionType  // if_result, if_variable, if_tool_result
    Target    string         // Step ID or variable name to check
    Operator  string         // equals, contains, matches, exists
    Value     any            // Value to compare against
    ThenSteps []string       // Steps to add if condition is true
    ElseSteps []string       // Steps to add if condition is false
}

// RiskLevel for operations
type RiskLevel string
const (
    RiskLow      RiskLevel = "low"
    RiskMedium   RiskLevel = "medium"
    RiskHigh     RiskLevel = "high"
    RiskCritical RiskLevel = "critical"
)

// PlanStatus tracks overall plan state
type PlanStatus string
const (
    PlanStatusPending     PlanStatus = "pending"
    PlanStatusExecuting   PlanStatus = "executing"
    PlanStatusCompleted   PlanStatus = "completed"
    PlanStatusFailed      PlanStatus = "failed"
    PlanStatusRolledBack  PlanStatus = "rolled_back"
)

// StepStatus tracks individual step state
type StepStatus string
const (
    StepStatusPending   StepStatus = "pending"
    StepStatusActive    StepStatus = "active"
    StepStatusCompleted StepStatus = "completed"
    StepStatusFailed    StepStatus = "failed"
    StepStatusSkipped   StepStatus = "skipped"
    StepStatusTimeout   StepStatus = "timeout"
)
```

#### ExecutionContext

```go
// ExecutionContext carries execution state across the pipeline
type ExecutionContext struct {
    Plan          *Plan
    CurrentStep   int
    LoopIteration int
    ExecutedSteps []*ExecutedStep
    Variables     map[string]any    // Runtime variables
    Checkpoints   []*Checkpoint     // For resume capability
    Interrupted   bool              // HITL interrupt flag
    InterruptData map[string]any    // Approval data, etc.
}

// ExecutedStep records a completed step
type ExecutedStep struct {
    Step       *Step
    Status     StepStatus
    Result     string
    Error      error
    Duration   time.Duration
    ToolCalls  []ToolCallRecord
}

// ToolCallRecord tracks tool invocations within a step
type ToolCallRecord struct {
    CallID    string
    ToolName  string
    Arguments map[string]any
    Result    string
    Error     error
}

// Checkpoint persists execution state for resume
type Checkpoint struct {
    Plan          *Plan            // Full plan state (required for resume)
    CheckpointID  string           // Runner checkpoint ID (same ID used by adk.Runner)
    StepIndex     int              // Current step index
    LoopIteration int              // Current execute-replan loop iteration
    ExecutedSteps []*ExecutedStep  // Steps that have been executed
    Variables     map[string]any   // Runtime variables
    Timestamp     time.Time
    Status        CheckpointStatus
}

type CheckpointStatus string
const (
    CheckpointActive     CheckpointStatus = "active"
    CheckpointCompleted  CheckpointStatus = "completed"
    CheckpointInterrupted CheckpointStatus = "interrupted"
)
```

#### Result Types

```go
// StepResult contains the outcome of step execution
type StepResult struct {
    Status    StepStatus
    Content   string           // Result content from execution
    Error     error            // Execution error (if any)
    ToolCalls []ToolCallRecord // Tool invocations made during execution
}

// ExecutionResult contains the outcome of the entire execution
type ExecutionResult struct {
    Status        ExecutionStatus
    Context       *ExecutionContext
    Summary       string
    Evidence      []string
    InterruptData map[string]any // For HITL approval
    Reason        string         // For wait/error states
}

// ExecutionStatus tracks overall execution state
type ExecutionStatus string
const (
    ExecutionStatusCompleted   ExecutionStatus = "completed"
    ExecutionStatusInterrupted ExecutionStatus = "interrupted"
    ExecutionStatusWaiting     ExecutionStatus = "waiting"
    ExecutionStatusMaxLoops    ExecutionStatus = "max_loops_reached"
    ExecutionStatusFailed      ExecutionStatus = "failed"
)

// ErrInterrupted is returned when execution is paused for HITL approval
var ErrInterrupted = errors.New("execution interrupted for approval")
```

#### Support Types

```go
// ToolMeta describes a tool for planner prompt generation
type ToolMeta struct {
    Name        string
    Description string
    Category    string    // readonly, write, discovery, etc.
    RiskLevel   RiskLevel
    Parameters  map[string]ParamMeta
}

// ParamMeta describes a tool parameter
type ParamMeta struct {
    Name        string
    Type        string
    Required    bool
    Description string
}

// Change represents a reversible operation for rollback tracking
type Change struct {
    StepID              string
    Description         string
    Reversible          bool
    RollbackTools       []string
    RollbackDescription string
    OriginalState       string // State before change (for restoration)
}

// Alert represents an external alert signal
type Alert struct {
    ID       string
    Severity string
    Message  string
    Labels   map[string]string
}

// SystemEvent represents a cluster/platform state change
type SystemEvent struct {
    Type      string
    Source    string
    Message   string
    Timestamp time.Time
}

// ConditionType defines condition evaluation types
type ConditionType string
const (
    ConditionIfResult     ConditionType = "if_result"
    ConditionIfVariable   ConditionType = "if_variable"
    ConditionIfToolResult ConditionType = "if_tool_result"
)
```

### Step Templates

```go
// StepTemplate provides reusable step patterns
type StepTemplate struct {
    ID          string
    Name        string
    Category    string            // diagnosis, change, rollback, etc.
    Description string
    Steps       []*Step           // Template steps
    Variables   map[string]string // Required variables with descriptions
}

// TemplateRegistry stores reusable step patterns
type TemplateRegistry struct {
    templates map[string]*StepTemplate
    mu        sync.RWMutex
}

func NewTemplateRegistry() *TemplateRegistry {
    r := &TemplateRegistry{templates: make(map[string]*StepTemplate)}

    // Built-in templates
    r.Register(diagnosisCrashloopTemplate())
    r.Register(diagnosisOOMTemplate())
    r.Register(changeScaleTemplate())
    r.Register(rollbackDeployTemplate())

    return r
}

func (r *TemplateRegistry) Register(t *StepTemplate) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.templates[t.ID] = t
}

func (r *TemplateRegistry) Get(id string) *StepTemplate {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.templates[id]
}

func (r *TemplateRegistry) ByCategory(category string) []*StepTemplate {
    r.mu.RLock()
    defer r.mu.RUnlock()

    var result []*StepTemplate
    for _, t := range r.templates {
        if t.Category == category {
            result = append(result, t)
        }
    }
    return result
}
```

#### Example Templates

```go
// diagnosisCrashloopTemplate returns a standard crashloop diagnosis sequence
func diagnosisCrashloopTemplate() *StepTemplate {
    return &StepTemplate{
        ID:          "diagnosis-crashloop",
        Name:        "CrashLoop Diagnosis",
        Category:    "diagnosis",
        Description: "Standard crashloop diagnosis sequence for pods in CrashLoopBackOff state",
        Steps: []*Step{
            {
                ID:            "cc-identify",
                Title:         "Identify crashloop pod",
                Description:   "Use discovery tools to find the pod in crashloop state",
                Priority:      10,
                RiskLevel:     RiskLow,
                RequiredTools: []string{"k8s_list_resources"},
                DependsOn:     []string{},
            },
            {
                ID:            "cc-events",
                Title:         "Fetch pod events",
                Description:   "Get recent Kubernetes events for the crashloop pod",
                Priority:      9,
                RiskLevel:     RiskLow,
                RequiredTools: []string{"k8s_events"},
                DependsOn:     []string{"cc-identify"},
            },
            {
                ID:            "cc-logs",
                Title:         "Fetch container logs",
                Description:   "Get container logs from the crashloop pod",
                Priority:      9,
                RiskLevel:     RiskLow,
                RequiredTools: []string{"k8s_logs"},
                DependsOn:     []string{"cc-identify"},
            },
            {
                ID:            "cc-analyze",
                Title:         "Analyze root cause",
                Description:   "Synthesize events and logs to determine root cause",
                Priority:      8,
                RiskLevel:     RiskLow,
                DependsOn:     []string{"cc-events", "cc-logs"},
            },
        },
        Variables: map[string]string{
            "cluster_id": "Target cluster ID (required)",
            "namespace":  "Pod namespace (optional, defaults to all)",
            "pod_name":   "Pod name pattern (optional)",
        },
    }
}

// rollbackDeployTemplate returns a deployment rollback sequence
func rollbackDeployTemplate() *StepTemplate {
    return &StepTemplate{
        ID:          "rollback-deployment",
        Name:        "Deployment Rollback",
        Category:    "rollback",
        Description: "Rollback a Kubernetes deployment to a previous revision",
        Steps: []*Step{
            {
                ID:            "rb-history",
                Title:         "Check rollout history",
                Description:   "List available revision history for the deployment",
                Priority:      10,
                RiskLevel:     RiskLow,
                RequiredTools: []string{"k8s_rollout_history"},
            },
            {
                ID:            "rb-confirm",
                Title:         "Confirm rollback target",
                Description:   "Verify the target revision exists and is healthy",
                Priority:      9,
                RiskLevel:     RiskMedium,
                RequiredTools: []string{"k8s_describe_deployment"},
                DependsOn:     []string{"rb-history"},
            },
            {
                ID:            "rb-execute",
                Title:         "Execute rollback",
                Description:   "Perform the rollback to the target revision",
                Priority:      8,
                RiskLevel:     RiskHigh,
                RequiredTools: []string{"k8s_rollback_deployment"},
                DependsOn:     []string{"rb-confirm"},
            },
            {
                ID:            "rb-verify",
                Title:         "Verify rollback success",
                Description:   "Confirm the deployment is healthy after rollback",
                Priority:      7,
                RiskLevel:     RiskLow,
                RequiredTools: []string{"k8s_get_deployment", "k8s_list_pods"},
                DependsOn:     []string{"rb-execute"},
            },
        },
        Variables: map[string]string{
            "cluster_id":   "Target cluster ID (required)",
            "namespace":    "Deployment namespace (required)",
            "deployment":   "Deployment name (required)",
            "revision":     "Target revision number (optional, defaults to previous)",
        },
    }
}
```

### Planner

#### Architecture

```go
// Planner generates structured plans from user input
type Planner struct {
    model         *chatmodel.ChatModel
    templates     *TemplateRegistry
    validator     *PlanValidator
    genInputFn    GenPlannerInputFn
}

type PlannerConfig struct {
    Model              *chatmodel.ChatModel
    TemplateRegistry   *TemplateRegistry  // Optional: for step templates
    Validator          *PlanValidator     // Optional: for plan validation
    GenInputFn         GenPlannerInputFn  // Custom prompt generation
}

// GenPlannerInputFn generates planner prompts from user input
type GenPlannerInputFn func(ctx context.Context, input PlannerInput) ([]adk.Message, error)

type PlannerInput struct {
    UserMessage    string
    Context        map[string]any    // Route context, cluster_id, etc.
    SessionHistory []adk.Message     // Prior conversation
    AvailableTools []ToolMeta        // Tools that will be available
}
```

#### Planner Behavior

1. **Prompt Generation**: `GenInputFn` creates structured prompt with user objective, available tools, applicable templates, and planning constraints

2. **Structured Output**: Model outputs JSON matching `Plan` schema

3. **Template Injection**: Planner can inject step templates when patterns match

4. **Plan Validation**: Before returning, validate dependencies, tool availability, and variable requirements

```go
// Plan generates a structured execution plan
func (p *Planner) Plan(ctx context.Context, input []adk.Message) (*Plan, error) {
    // 1. Build prompt with available tools and templates
    plannerInput := p.buildPlannerInput(input)
    msgs, err := p.genInputFn(ctx, plannerInput)
    if err != nil {
        return nil, fmt.Errorf("generate prompt: %w", err)
    }

    // 2. Call model with structured output
    // GenerateWithJSONSchema is a helper method that wraps the model's Generate
    // method with JSON schema validation and parsing
    var plan Plan
    if err := generateWithSchema(ctx, p.model, msgs, &plan, planSchema); err != nil {
        return nil, fmt.Errorf("model generate: %w", err)
    }

    // 3. Apply matching templates
    if p.templates != nil {
        p.applyTemplates(&plan)
    }

    // 4. Validate plan
    if p.validator != nil {
        if err := p.validator.Validate(&plan, plannerInput.AvailableTools); err != nil {
            return nil, fmt.Errorf("plan validation: %w", err)
        }
    }

    // 5. Initialize plan metadata
    plan.ID = generatePlanID()
    plan.CreatedAt = time.Now()
    plan.Status = PlanStatusPending

    return &plan, nil
}

// planSchema defines the JSON schema for structured plan output
// Uses jsonschema from github.com/eino-contrib/jsonschema
// Note: Properties must use orderedmap.OrderedMap for proper JSON schema generation
import (
    "github.com/eino-contrib/jsonschema"
    "github.com/iancoleman/orderedmap"
)

var planSchema = &jsonschema.Schema{
    Type:       "object",
    Properties: newOrderedSchema(map[string]*jsonschema.Schema{
        "goal":      {Type: "string", Description: "User's original objective"},
        "steps":     {Type: "array", Items: stepSchema},
        "variables": {Type: "object"},
    }),
    Required: []string{"goal", "steps"},
}

var stepSchema = &jsonschema.Schema{
    Type:       "object",
    Properties: newOrderedSchema(map[string]*jsonschema.Schema{
        "id":             {Type: "string"},
        "title":          {Type: "string"},
        "description":    {Type: "string"},
        "priority":       {Type: "integer"},
        "risk_level":     {Type: "string", Enum: []string{"low", "medium", "high", "critical"}},
        "timeout":        {Type: "string"}, // Duration string
        "required_tools": {Type: "array", Items: &jsonschema.Schema{Type: "string"}},
        "depends_on":     {Type: "array", Items: &jsonschema.Schema{Type: "string"}},
    }),
    Required: []string{"id", "title"},
}

// newOrderedSchema converts a map to orderedmap for jsonschema.Properties
func newOrderedSchema(m map[string]*jsonschema.Schema) *orderedmap.OrderedMap[string, *jsonschema.Schema] {
    om := orderedmap.New[string, *jsonschema.Schema]()
    for k, v := range m {
        om.Set(k, v)
    }
    return om
}

// buildPlannerInput extracts context from user messages
func (p *Planner) buildPlannerInput(input []adk.Message) PlannerInput {
    // Implementation: extract user message, context, session history
    return PlannerInput{}
}

// applyTemplates injects matching step templates into the plan
func (p *Planner) applyTemplates(plan *Plan) {
    // Implementation: detect template patterns, inject template steps
    // Template matching strategy:
    // 1. Check if user message contains keywords matching template patterns
    // 2. If match found, prepend template steps to plan
    // 3. Resolve template variables from plan variables
}
```

#### Structured Output Helper

```go
// generateWithSchema wraps model.Generate with JSON schema validation
// This helper handles structured output since Eino doesn't have a built-in method
func generateWithSchema(ctx context.Context, model *chatmodel.ChatModel, msgs []adk.Message, target any, schema *jsonschema.Schema) error {
    // Strategy: Use tool-based structured output
    // 1. Create a "submit_result" tool with the target schema
    // 2. Prompt the model to use this tool
    // 3. Parse tool call arguments into target struct

    submitTool := &tool.Info{
        Name: "submit_structured_result",
        Desc: "Submit the result in the required structured format",
        ParamsOneOf: schema, // Bind schema as tool parameters
    }

    // Add tool instruction to the last message
    toolPrompt := fmt.Sprintf("\n\nIMPORTANT: You must call the '%s' tool to submit your result.", submitTool.Name)
    msgs[len(msgs)-1].Content += toolPrompt

    // Generate with tool
    result, err := model.Generate(ctx, msgs, model.WithTools([]*tool.Info{submitTool}))
    if err != nil {
        return err
    }

    // Extract tool call and parse
    for _, tc := range result.ToolCalls {
        if tc.Name == submitTool.Name {
            return json.Unmarshal([]byte(tc.Arguments), target)
        }
    }

    // Fallback: try to parse response as JSON
    return json.Unmarshal([]byte(result.Content), target)
}
```

### Executor

#### Architecture

```go
// Executor runs plan steps with timeout control, dynamic tools, and checkpointing
type Executor struct {
    model           *chatmodel.ChatModel
    toolRegistry    *ToolRegistry
    defaultTimeout  time.Duration
    maxIterations   int
    genInputFn      GenExecutorInputFn
    checkpointStore CheckpointStore
}

type ExecutorConfig struct {
    Model           *chatmodel.ChatModel
    ToolRegistry    *ToolRegistry
    DefaultTimeout  time.Duration      // Default step timeout (e.g., 60s)
    MaxIterations   int                // Max tool calls per step
    GenInputFn      GenExecutorInputFn
    CheckpointStore CheckpointStore    // For resume capability
}

// GenExecutorInputFn generates executor prompts
type GenExecutorInputFn func(ctx context.Context, execCtx *ExecutionContext) ([]adk.Message, error)

// ToolRegistry provides dynamic tool selection
type ToolRegistry struct {
    allTools   map[string]compose.Tool
    byCategory map[string][]string
    byRisk     map[RiskLevel][]string
    mu         sync.RWMutex
}

// NewToolRegistry creates a tool registry from a list of tools
func NewToolRegistry(tools []compose.Tool, categorizer ToolCategorizer) *ToolRegistry {
    r := &ToolRegistry{
        allTools:   make(map[string]compose.Tool),
        byCategory: make(map[string][]string),
        byRisk:     make(map[RiskLevel][]string),
    }
    for _, t := range tools {
        info := t.Info()
        r.allTools[info.Name] = t
        cat := categorizer.Category(info)
        r.byCategory[cat] = append(r.byCategory[cat], info.Name)
        risk := categorizer.RiskLevel(info)
        r.byRisk[risk] = append(r.byRisk[risk], info.Name)
    }
    return r
}

// Get retrieves a tool by name
func (r *ToolRegistry) Get(name string) (compose.Tool, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    t, ok := r.allTools[name]
    return t, ok
}

// ByCategory returns all tools in a category
func (r *ToolRegistry) ByCategory(category string) []compose.Tool {
    r.mu.RLock()
    defer r.mu.RUnlock()
    var tools []compose.Tool
    for _, name := range r.byCategory[category] {
        if tool, ok := r.allTools[name]; ok {
            tools = append(tools, tool)
        }
    }
    return tools
}

// ByRisk returns all tools at a given risk level
func (r *ToolRegistry) ByRisk(level RiskLevel) []compose.Tool {
    r.mu.RLock()
    defer r.mu.RUnlock()
    var tools []compose.Tool
    for _, name := range r.byRisk[level] {
        if tool, ok := r.allTools[name]; ok {
            tools = append(tools, tool)
        }
    }
    return tools
}

// ToolCategorizer determines tool category and risk level
type ToolCategorizer interface {
    Category(info *tool.Info) string
    RiskLevel(info *tool.Info) RiskLevel
}

// CheckpointStore persists execution state for resume
type CheckpointStore interface {
    Save(ctx context.Context, checkpoint *Checkpoint) error
    Load(ctx context.Context, checkpointID string) (*Checkpoint, error)
    Latest(ctx context.Context, checkpointID string) (*Checkpoint, error)
}

// PlanValidator validates plans before execution
type PlanValidator struct {
    availableTools map[string]bool
}

func NewPlanValidator(tools []string) *PlanValidator {
    v := &PlanValidator{availableTools: make(map[string]bool)}
    for _, t := range tools {
        v.availableTools[t] = true
    }
    return v
}

// Validate checks plan for errors (missing dependencies, unavailable tools, etc.)
func (v *PlanValidator) Validate(plan *Plan, availableTools []ToolMeta) error {
    // Implementation: check dependencies, tool availability, variable requirements
    return nil
}
```

#### CheckpointStore Adapter

The existing `internal/ai/checkpoint/store.go` provides a simpler interface. Use an adapter pattern:

Important: checkpoint key must stay aligned with ADK Runner (`adk.WithCheckPointID(...)`),
so custom plan runtime state and interrupt/resume state share the same `checkpoint_id`.

```go
// ErrCheckpointNotFound is returned when a checkpoint doesn't exist
var ErrCheckpointNotFound = errors.New("checkpoint not found")

// PlanCheckpointStore adapts the existing checkpoint.Store for plan execution
type PlanCheckpointStore struct {
    store checkpoint.Store
}

func NewPlanCheckpointStore(store checkpoint.Store) *PlanCheckpointStore {
    return &PlanCheckpointStore{store: store}
}

func (s *PlanCheckpointStore) Save(ctx context.Context, checkpoint *Checkpoint) error {
    data, err := json.Marshal(checkpoint)
    if err != nil {
        return err
    }
    return s.store.Set(ctx, checkpoint.CheckpointID, data)
}

func (s *PlanCheckpointStore) Load(ctx context.Context, checkpointID string) (*Checkpoint, error) {
    data, found, err := s.store.Get(ctx, checkpointID)
    if err != nil {
        return nil, err
    }
    if !found {
        return nil, ErrCheckpointNotFound
    }
    var checkpoint Checkpoint
    if err := json.Unmarshal(data, &checkpoint); err != nil {
        return nil, err
    }
    return &checkpoint, nil
}

func (s *PlanCheckpointStore) Latest(ctx context.Context, checkpointID string) (*Checkpoint, error) {
    // For the existing store, Load and Latest are equivalent
    return s.Load(ctx, checkpointID)
}
```

#### Executor Behavior

1. **Dynamic Tool Selection**: Load only tools needed for current step

2. **Step-Level Timeout Control**: Configurable timeout per step with context cancellation

3. **Checkpoint at Step Boundaries**: Save state before and after each step

4. **Resume from Checkpoint**: Restore execution state and continue from interrupted point

```go
// Run executes the plan step by step
func (e *Executor) Run(ctx context.Context, execCtx *ExecutionContext) error {
    for i, step := range execCtx.Plan.Steps {
        // Skip completed steps (for resume)
        if step.Status == StepStatusCompleted {
            continue
        }

        // Skip steps with unmet dependencies
        if !e.checkDependencies(execCtx, step) {
            step.Status = StepStatusSkipped
            continue
        }

        // Check conditional execution
        if step.Condition != nil && !e.evaluateCondition(execCtx, step.Condition) {
            e.applyBranch(execCtx, step.Condition, false)
            continue
        }

        // Mark step active
        step.Status = StepStatusActive
        step.StartedAt = ptrTime(time.Now())
        execCtx.CurrentStep = i

        // Save checkpoint before execution
        e.saveCheckpoint(ctx, execCtx)

        // Execute step with timeout
        result := e.executeStep(ctx, step, execCtx)

        // Update step state
        step.Status = result.Status
        step.Result = result.Content
        step.Error = result.Error
        step.CompletedAt = ptrTime(time.Now())

        // Record executed step
        execCtx.ExecutedSteps = append(execCtx.ExecutedSteps, &ExecutedStep{
            Step:      step,
            Status:    result.Status,
            Result:    result.Content,
            Error:     result.Error,
            Duration:  time.Since(*step.StartedAt),
            ToolCalls: result.ToolCalls,
        })

        // Save checkpoint after execution
        e.saveCheckpoint(ctx, execCtx)

        // Check for interrupt (HITL approval)
        if execCtx.Interrupted {
            return ErrInterrupted
        }

        // Stop on failure
        if step.Status == StepStatusFailed {
            return fmt.Errorf("step %s failed: %w", step.ID, step.Error)
        }
    }

    return nil
}

// executeStep runs a single step with timeout and dynamic tools
func (e *Executor) executeStep(ctx context.Context, step *Step, execCtx *ExecutionContext) *StepResult {
    timeout := step.Timeout
    if timeout == 0 {
        timeout = e.defaultTimeout
    }

    stepCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    // Select tools for this step
    tools := e.selectTools(step)

    // Build step prompt
    msgs, err := e.genInputFn(stepCtx, execCtx)
    if err != nil {
        return &StepResult{Status: StepStatusFailed, Error: err}
    }

    // Execute with tool calling loop
    result, err := e.runToolLoop(stepCtx, msgs, tools, step, execCtx)

    // Handle timeout
    if errors.Is(stepCtx.Err(), context.DeadlineExceeded) {
        return &StepResult{
            Status: StepStatusTimeout,
            Error:  fmt.Errorf("step %s timed out after %v", step.ID, timeout),
        }
    }

    if err != nil {
        return &StepResult{Status: StepStatusFailed, Error: err}
    }

    return result
}

// selectTools returns tools needed for the current step
func (e *Executor) selectTools(step *Step) []compose.Tool {
    if len(step.RequiredTools) == 0 {
        return e.toolRegistry.ByCategory("readonly")
    }

    tools := make([]compose.Tool, 0, len(step.RequiredTools))
    for _, name := range step.RequiredTools {
        if tool, ok := e.toolRegistry.Get(name); ok {
            tools = append(tools, tool)
        }
    }
    return tools
}

// Resume restores execution from a checkpoint
func (e *Executor) Resume(ctx context.Context, checkpointID string, resumeData map[string]any) (*ExecutionContext, error) {
    checkpoint, err := e.checkpointStore.Latest(ctx, checkpointID)
    if err != nil {
        return nil, err
    }

    execCtx := &ExecutionContext{
        Plan:          checkpoint.Plan,
        ExecutedSteps: checkpoint.ExecutedSteps,
        Variables:     checkpoint.Variables,
        CurrentStep:   checkpoint.StepIndex,
        LoopIteration: checkpoint.LoopIteration,
    }

    // Apply resume data (e.g., approval result)
    if approval, ok := resumeData["approval"]; ok {
        e.handleApprovalResume(execCtx, approval)
    }

    return execCtx, nil
}

// Helper methods for Executor

// checkDependencies verifies all step dependencies are satisfied
func (e *Executor) checkDependencies(execCtx *ExecutionContext, step *Step) bool {
    statusByStepID := make(map[string]StepStatus, len(execCtx.ExecutedSteps))
    for _, executed := range execCtx.ExecutedSteps {
        statusByStepID[executed.Step.ID] = executed.Status
    }

    for _, depID := range step.DependsOn {
        status, ok := statusByStepID[depID]
        if !ok || status != StepStatusCompleted {
            return false
        }
    }
    return true
}

// evaluateCondition evaluates a step's conditional execution rule
func (e *Executor) evaluateCondition(execCtx *ExecutionContext, cond *Condition) bool {
    // Implementation: evaluate condition based on type (if_result, if_variable, if_tool_result)
    return true
}

// applyBranch adds then/else steps based on condition result
func (e *Executor) applyBranch(execCtx *ExecutionContext, cond *Condition, result bool) {
    // Implementation: insert then_steps or else_steps into plan
}

// runToolLoop executes the tool calling loop for a step
func (e *Executor) runToolLoop(ctx context.Context, msgs []adk.Message, tools []compose.Tool, step *Step, execCtx *ExecutionContext) (*StepResult, error) {
    // Implementation: run LLM with tools until completion or max iterations
    return &StepResult{Status: StepStatusCompleted}, nil
}

// handleApprovalResume processes approval result during resume
func (e *Executor) handleApprovalResume(execCtx *ExecutionContext, approval any) {
    // Implementation: apply approval result to interrupted step
}

// saveCheckpoint persists current execution state
func (e *Executor) saveCheckpoint(ctx context.Context, execCtx *ExecutionContext) {
    if e.checkpointStore == nil {
        return
    }
    checkpoint := &Checkpoint{
        Plan:          execCtx.Plan,
        CheckpointID:  checkpointIDFromContext(ctx),
        StepIndex:     execCtx.CurrentStep,
        LoopIteration: execCtx.LoopIteration,
        ExecutedSteps: execCtx.ExecutedSteps,
        Variables:     execCtx.Variables,
        Timestamp:     time.Now(),
        Status:        CheckpointActive,
    }
    e.checkpointStore.Save(ctx, checkpoint)
}

func checkpointIDFromContext(ctx context.Context) string {
    // Implementation: read checkpoint ID from runtime metadata injected by service layer
    // (same ID used by adk.Runner Run/ResumeWithParams)
    return ""
}
```

### Replanner

#### Architecture

```go
// Replanner evaluates execution progress and adjusts the remaining plan
type Replanner struct {
    model              *chatmodel.ChatModel
    convergenceChecker *ConvergenceChecker
    stepOptimizer      *StepOptimizer
    rollbackDetector   *RollbackDetector
    genInputFn         GenReplannerInputFn
}

type ReplannerConfig struct {
    Model              *chatmodel.ChatModel
    ConvergenceChecker *ConvergenceChecker
    StepOptimizer      *StepOptimizer
    RollbackDetector   *RollbackDetector
    GenInputFn         GenReplannerInputFn
}

// ReplannerContext provides full context for replanning decisions
type ReplannerContext struct {
    Plan          *Plan
    ExecutedSteps []*ExecutedStep
    Variables     map[string]any
    ExternalCtx   *ExternalContext  // Alerts, user feedback, etc.
    Iteration     int
    MaxIterations int
}

// ExternalContext carries runtime signals for context-aware replanning
type ExternalContext struct {
    NewAlerts    []Alert
    UserFeedback string
    SystemEvents []SystemEvent
    TimeElapsed  time.Duration
}
```

#### Convergence Detection

```go
// ConvergenceChecker determines if the goal has been achieved
type ConvergenceChecker struct {
    model *chatmodel.ChatModel
}

type ConvergenceResult struct {
    Converged    bool
    Confidence   float64   // 0.0 - 1.0
    Reason       string
    Evidence     []string
}

func (c *ConvergenceChecker) Check(ctx context.Context, rctx *ReplannerContext) (*ConvergenceResult, error) {
    // Build convergence check prompt with goal and executed steps
    prompt := c.buildPrompt(rctx)

    var result ConvergenceResult
    if err := generateWithSchema(ctx, c.model, prompt, &result, convergenceSchema); err != nil {
        return nil, err
    }

    return &result, nil
}

// convergenceSchema defines the JSON schema for convergence results
// Uses jsonschema from github.com/eino-contrib/jsonschema
var convergenceSchema = &jsonschema.Schema{
    Type:       "object",
    Properties: newOrderedSchema(map[string]*jsonschema.Schema{
        "converged":  {Type: "boolean"},
        "confidence": {Type: "number", Minimum: ptrFloat(0), Maximum: ptrFloat(1)},
        "reason":     {Type: "string"},
        "evidence":   {Type: "array", Items: &jsonschema.Schema{Type: "string"}},
    }),
    Required: []string{"converged", "confidence"},
}

func ptrFloat(f float64) *float64 { return &f }

// buildPrompt creates the convergence check prompt
func (c *ConvergenceChecker) buildPrompt(rctx *ReplannerContext) []adk.Message {
    // Implementation: format goal and executed steps into prompt
    return nil
}
```

#### Step Optimization

```go
// StepOptimizer handles merging, splitting, and reordering steps
type StepOptimizer struct {
    model *chatmodel.ChatModel
}

type OptimizationResult struct {
    Action      OptimizationAction
    AffectedIDs []string
    NewSteps    []*Step
    Reason      string
}

type OptimizationAction string
const (
    ActionNone     OptimizationAction = "none"
    ActionMerge    OptimizationAction = "merge"
    ActionSplit    OptimizationAction = "split"
    ActionReorder  OptimizationAction = "reorder"
)

func (o *StepOptimizer) Analyze(ctx context.Context, rctx *ReplannerContext) (*OptimizationResult, error) {
    remaining := o.getRemainingSteps(rctx.Plan)

    // Check for merge candidates (similar tools, no interdependencies)
    if merge := o.findMergeCandidates(remaining); merge != nil {
        return merge, nil
    }

    // Check for split candidates (overly broad steps)
    if split := o.findSplitCandidates(remaining); split != nil {
        return split, nil
    }

    return &OptimizationResult{Action: ActionNone}, nil
}

// canMerge checks if two steps can be combined
func (o *StepOptimizer) canMerge(a, b *Step) bool {
    // Same risk level
    if a.RiskLevel != b.RiskLevel {
        return false
    }

    // Overlapping tool requirements
    if len(intersection(a.RequiredTools, b.RequiredTools)) > 0 {
        return true
    }

    // No dependency between them
    for _, dep := range b.DependsOn {
        if dep == a.ID {
            return false
        }
    }

    return true
}

// shouldSplit checks if a step is too broad
func (o *StepOptimizer) shouldSplit(step *Step) bool {
    // Heuristic: steps with many required tools or long description
    return len(step.RequiredTools) > 4 || len(step.Description) > 500
}

// getRemainingSteps returns steps that haven't been executed
func (o *StepOptimizer) getRemainingSteps(plan *Plan) []*Step {
    // Implementation: filter out completed steps
    return nil
}

// findMergeCandidates finds steps that can be combined
func (o *StepOptimizer) findMergeCandidates(steps []*Step) *OptimizationResult {
    // Implementation: detect mergeable step pairs
    return nil
}

// findSplitCandidates finds steps that should be divided
func (o *StepOptimizer) findSplitCandidates(steps []*Step) *OptimizationResult {
    // Implementation: detect overly broad steps
    return nil
}

// intersection returns common elements between two string slices
func intersection(a, b []string) []string {
    m := make(map[string]bool)
    for _, x := range a {
        m[x] = true
    }
    var result []string
    for _, x := range b {
        if m[x] {
            result = append(result, x)
        }
    }
    return result
}
```

#### Rollback Detection

```go
// RollbackDetector identifies when rollback is needed and generates rollback steps
type RollbackDetector struct {
    model *chatmodel.ChatModel
}

type RollbackDecision struct {
    NeedRollback  bool
    Severity      RollbackSeverity
    Reason        string
    RollbackSteps []*Step
}

type RollbackSeverity string
const (
    RollbackPartial RollbackSeverity = "partial"
    RollbackFull    RollbackSeverity = "full"
)

func (r *RollbackDetector) Analyze(ctx context.Context, rctx *ReplannerContext) (*RollbackDecision, error) {
    // Check for failed steps
    for _, executed := range rctx.ExecutedSteps {
        if executed.Status == StepStatusFailed {
            return r.generateRollbackPlan(ctx, rctx, executed)
        }
    }

    // Check for unexpected results
    if r.hasUnexpectedResults(rctx) {
        return r.generateRollbackPlan(ctx, rctx, nil)
    }

    return &RollbackDecision{NeedRollback: false}, nil
}

func (r *RollbackDetector) generateRollbackPlan(ctx context.Context, rctx *ReplannerContext, failedStep *ExecutedStep) (*RollbackDecision, error) {
    // Identify what was changed
    changes := r.identifyChanges(rctx.ExecutedSteps)

    // Generate compensating steps
    rollbackSteps := make([]*Step, 0)
    for _, change := range changes {
        if change.Reversible {
            rollbackSteps = append(rollbackSteps, &Step{
                ID:            fmt.Sprintf("rollback-%s", change.StepID),
                Title:         fmt.Sprintf("Rollback: %s", change.Description),
                RiskLevel:     RiskHigh,
                RequiredTools: change.RollbackTools,
                Description:   change.RollbackDescription,
            })
        }
    }

    return &RollbackDecision{
        NeedRollback:  len(rollbackSteps) > 0,
        Severity:      r.determineSeverity(rctx),
        Reason:        "Execution failure detected",
        RollbackSteps: rollbackSteps,
    }, nil
}

// identifyChanges finds all reversible changes in executed steps
func (r *RollbackDetector) identifyChanges(executedSteps []*ExecutedStep) []Change {
    // Implementation: scan executed steps for reversible operations
    return nil
}

// hasUnexpectedResults checks for unexpected outcomes
func (r *RollbackDetector) hasUnexpectedResults(rctx *ReplannerContext) bool {
    // Implementation: detect anomalies in execution results
    return false
}

// determineSeverity assesses rollback severity level
func (r *RollbackDetector) determineSeverity(rctx *ReplannerContext) RollbackSeverity {
    // Implementation: determine if partial or full rollback needed
    return RollbackPartial
}
```

#### Replanner Behavior

```go
// Run evaluates execution and adjusts the plan
func (r *Replanner) Run(ctx context.Context, rctx *ReplannerContext) (*ReplanResult, error) {
    // 1. Check convergence
    conv, err := r.convergenceChecker.Check(ctx, rctx)
    if err != nil {
        return nil, err
    }
    if conv.Converged && conv.Confidence > 0.8 {
        return &ReplanResult{
            Action:   ReplanActionComplete,
            Reason:   conv.Reason,
            Evidence: conv.Evidence,
        }, nil
    }

    // 2. Check for rollback needs
    rollback, err := r.rollbackDetector.Analyze(ctx, rctx)
    if err != nil {
        return nil, err
    }
    if rollback.NeedRollback {
        return &ReplanResult{
            Action:       ReplanActionRollback,
            NewSteps:     rollback.RollbackSteps,
            Reason:       rollback.Reason,
            RollbackPlan: rollback,
        }, nil
    }

    // 3. Optimize remaining steps
    opt, err := r.stepOptimizer.Analyze(ctx, rctx)
    if err != nil {
        return nil, err
    }
    if opt.Action != ActionNone {
        rctx.Plan = r.applyOptimization(rctx.Plan, opt)
    }

    // 4. Generate new steps based on execution results
    newSteps, err := r.generateNewSteps(ctx, rctx, conv)
    if err != nil {
        return nil, err
    }

    return &ReplanResult{
        Action:      ReplanActionContinue,
        NewSteps:    newSteps,
        Convergence: conv,
    }, nil
}

// applyOptimization updates the plan based on optimization result
func (r *Replanner) applyOptimization(plan *Plan, opt *OptimizationResult) *Plan {
    // Implementation: apply merge/split/reorder changes
    return plan
}

// generateNewSteps creates additional steps based on execution results
func (r *Replanner) generateNewSteps(ctx context.Context, rctx *ReplannerContext, conv *ConvergenceResult) ([]*Step, error) {
    // Implementation: use LLM to generate new steps
    return nil, nil
}
```

#### Replanner Output

```go
type ReplanResult struct {
    Action        ReplanAction
    NewSteps      []*Step
    RemovedSteps  []string
    ModifiedSteps map[string]*Step
    Reason        string
    Evidence      []string
    Convergence   *ConvergenceResult
    RollbackPlan  *RollbackDecision
}

type ReplanAction string
const (
    ReplanActionContinue ReplanAction = "continue"
    ReplanActionComplete ReplanAction = "complete"
    ReplanActionRollback ReplanAction = "rollback"
    ReplanActionWait     ReplanAction = "wait"
)
```

### Composition

#### Main Agent Builder

```go
// BuildPlanExecuteAgent creates a complete Plan-Execute-Replan agent
func BuildPlanExecuteAgent(ctx context.Context, cfg *PlanExecuteConfig) (adk.ResumableAgent, error) {
    planner, err := NewPlanner(ctx, cfg.PlannerConfig)
    if err != nil {
        return nil, fmt.Errorf("build planner: %w", err)
    }

    executor, err := NewExecutor(ctx, cfg.ExecutorConfig)
    if err != nil {
        return nil, fmt.Errorf("build executor: %w", err)
    }

    replanner, err := NewReplanner(ctx, cfg.ReplannerConfig)
    if err != nil {
        return nil, fmt.Errorf("build replanner: %w", err)
    }

    engine := &PlanExecuteEngine{
        planner:    planner,
        executor:   executor,
        replanner:  replanner,
        maxLoops:   cfg.MaxReplanLoops,
        store:      cfg.CheckpointStore,
    }

    return &PlanExecuteAgent{
        engine: engine,
        config: cfg,
    }, nil
}

type PlanExecuteConfig struct {
    PlannerConfig    *PlannerConfig
    ExecutorConfig   *ExecutorConfig
    ReplannerConfig  *ReplannerConfig
    MaxReplanLoops   int
    CheckpointStore  CheckpointStore
}
```

#### Orchestration Engine

```go
// PlanExecuteEngine orchestrates the Plan-Execute-Replan loop
type PlanExecuteEngine struct {
    planner    *Planner
    executor   *Executor
    replanner  *Replanner
    maxLoops   int
    store      CheckpointStore
}

func (e *PlanExecuteEngine) Run(ctx context.Context, input []adk.Message) (*ExecutionResult, error) {
    // Phase 1: Planning
    plan, err := e.planner.Plan(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("planning phase: %w", err)
    }

    execCtx := &ExecutionContext{
        Plan:      plan,
        Variables: make(map[string]any),
    }

    // Phase 2: Execute-Replan Loop
    for loop := 0; loop < e.maxLoops; loop++ {
        execCtx.LoopIteration = loop
        // Execute current plan
        execErr := e.executor.Run(ctx, execCtx)
        if execErr == ErrInterrupted {
            return &ExecutionResult{
                Status:        ExecutionStatusInterrupted,
                Context:       execCtx,
                InterruptData: execCtx.InterruptData,
            }, nil
        }
        if execErr != nil {
            return nil, fmt.Errorf("execution phase: %w", execErr)
        }

        // Check if all steps completed
        if e.allStepsCompleted(execCtx) {
            break
        }

        // Phase 3: Replan
        replanResult, err := e.replanner.Run(ctx, &ReplannerContext{
            Plan:          execCtx.Plan,
            ExecutedSteps: execCtx.ExecutedSteps,
            Variables:     execCtx.Variables,
            Iteration:     loop,
            MaxIterations: e.maxLoops,
        })
        if err != nil {
            return nil, fmt.Errorf("replan phase: %w", err)
        }

        // Handle replan actions
        switch replanResult.Action {
        case ReplanActionComplete:
            execCtx.Plan.Status = PlanStatusCompleted
            return e.buildResult(execCtx), nil

        case ReplanActionRollback:
            execCtx.Plan.Steps = append(replanResult.NewSteps, execCtx.Plan.Steps...)
            execCtx.Plan.Status = PlanStatusRolledBack

        case ReplanActionContinue:
            execCtx.Plan.Steps = e.applyReplan(execCtx.Plan, replanResult)

        case ReplanActionWait:
            return &ExecutionResult{
                Status:  ExecutionStatusWaiting,
                Context: execCtx,
                Reason:  replanResult.Reason,
            }, nil
        }
    }

    return e.buildResult(execCtx), nil
}

// allStepsCompleted checks if all plan steps are done
func (e *PlanExecuteEngine) allStepsCompleted(execCtx *ExecutionContext) bool {
    for _, step := range execCtx.Plan.Steps {
        if step.Status != StepStatusCompleted && step.Status != StepStatusSkipped {
            return false
        }
    }
    return true
}

// applyReplan updates the plan based on replan result
func (e *PlanExecuteEngine) applyReplan(plan *Plan, result *ReplanResult) []*Step {
    // Implementation: remove completed steps, add new steps, apply modifications
    return plan.Steps
}

// buildResult creates the final execution result
func (e *PlanExecuteEngine) buildResult(execCtx *ExecutionContext) *ExecutionResult {
    return &ExecutionResult{
        Status:   ExecutionStatusCompleted,
        Context:  execCtx,
        Evidence: extractEvidence(execCtx.ExecutedSteps),
    }
}

// extractEvidence gathers key findings from executed steps
func extractEvidence(steps []*ExecutedStep) []string {
    var evidence []string
    for _, s := range steps {
        if s.Result != "" {
            evidence = append(evidence, s.Result)
        }
    }
    return evidence
}
```

#### ResumableAgent Interface

```go
// PlanExecuteAgent implements adk.ResumableAgent
type PlanExecuteAgent struct {
    engine *PlanExecuteEngine
    config *PlanExecuteConfig
}

func (a *PlanExecuteAgent) Run(
    ctx context.Context,
    input *adk.AgentInput,
    opts ...adk.AgentRunOption,
) *adk.AsyncIterator[*adk.AgentEvent] {
    // Implementation detail:
    // 1) Call a.engine.Run(ctx, input.Messages)
    // 2) Convert ExecutionResult to Assistant message / Action events
    // 3) Emit Interrupted action when approval middleware/tool raises interrupt
    return nil
}

// continueFromCheckpoint resumes execution from a checkpoint
func (e *PlanExecuteEngine) continueFromCheckpoint(ctx context.Context, execCtx *ExecutionContext) (*ExecutionResult, error) {
    // Continue the execute-replan loop from where it was interrupted
    // This mirrors the Run() method but starts from the checkpoint state
    for loop := execCtx.LoopIteration; loop < e.maxLoops; loop++ {
        execCtx.LoopIteration = loop
        execErr := e.executor.Run(ctx, execCtx)
        if execErr == ErrInterrupted {
            return &ExecutionResult{
                Status:        ExecutionStatusInterrupted,
                Context:       execCtx,
                InterruptData: execCtx.InterruptData,
            }, nil
        }
        if execErr != nil {
            return nil, fmt.Errorf("execution phase: %w", execErr)
        }

        if e.allStepsCompleted(execCtx) {
            break
        }

        // Replan logic (same as Run)
        replanResult, err := e.replanner.Run(ctx, &ReplannerContext{
            Plan:          execCtx.Plan,
            ExecutedSteps: execCtx.ExecutedSteps,
            Variables:     execCtx.Variables,
            Iteration:     loop,
            MaxIterations: e.maxLoops,
        })
        if err != nil {
            return nil, fmt.Errorf("replan phase: %w", err)
        }

        switch replanResult.Action {
        case ReplanActionComplete:
            execCtx.Plan.Status = PlanStatusCompleted
            return e.buildResult(execCtx), nil
        case ReplanActionRollback:
            execCtx.Plan.Steps = append(replanResult.NewSteps, execCtx.Plan.Steps...)
        case ReplanActionContinue:
            execCtx.Plan.Steps = e.applyReplan(execCtx.Plan, replanResult)
        case ReplanActionWait:
            return &ExecutionResult{
                Status:  ExecutionStatusWaiting,
                Context: execCtx,
                Reason:  replanResult.Reason,
            }, nil
        }
    }

    return e.buildResult(execCtx), nil
}

func (a *PlanExecuteAgent) Resume(
    ctx context.Context,
    info *adk.ResumeInfo,
    opts ...adk.AgentRunOption,
) *adk.AsyncIterator[*adk.AgentEvent] {
    // Resume does NOT define its own "params" protocol.
    // Targeted resume payload is supplied by adk.Runner.ResumeWithParams and consumed
    // by leaf nodes/tools (for this project, tool approval middleware by toolCallID).
    //
    // Here we only restore plan execution state from checkpoint and continue.
    return nil
}
```

### Integration

#### DiagnosisAgent

```go
func NewDiagnosisAgent(ctx context.Context) (adk.ResumableAgent, error) {
    cfg := &PlanExecuteConfig{
        PlannerConfig: &PlannerConfig{
            Model:            chatmodel.NewDiagnosisPlannerModel(ctx),
            GenInputFn:       diagnosisPlannerPrompt,
            TemplateRegistry: diagnosisTemplates(),
        },
        ExecutorConfig: &ExecutorConfig{
            Model:          chatmodel.NewDiagnosisExecutorModel(ctx),
            ToolRegistry:   diagnosisToolRegistry(ctx),
            DefaultTimeout: 60 * time.Second,
            MaxIterations:  24,
        },
        ReplannerConfig: &ReplannerConfig{
            Model: chatmodel.NewDiagnosisReplannerModel(ctx),
        },
        MaxReplanLoops: 20,
    }

    return BuildPlanExecuteAgent(ctx, cfg)
}
```

#### ChangeAgent

```go
func NewChangeAgent(ctx context.Context) (adk.ResumableAgent, error) {
    cfg := &PlanExecuteConfig{
        PlannerConfig: &PlannerConfig{
            Model:            chatmodel.NewChangePlannerModel(ctx),
            GenInputFn:       changePlannerPrompt,
            TemplateRegistry: changeTemplates(),
            Validator:        changePlanValidator(),
        },
        ExecutorConfig: &ExecutorConfig{
            Model:           chatmodel.NewChangeExecutorModel(ctx),
            ToolRegistry:    changeToolRegistry(ctx),
            DefaultTimeout:  120 * time.Second,
            MaxIterations:   24,
            CheckpointStore: dbCheckpointStore(),
        },
        ReplannerConfig: &ReplannerConfig{
            Model:            chatmodel.NewChangeReplannerModel(ctx),
            RollbackDetector: changeRollbackDetector(),
        },
        MaxReplanLoops:  20,
        CheckpointStore: dbCheckpointStore(),
    }

    return BuildPlanExecuteAgent(ctx, cfg)
}
```

## Migration Plan

### Phase 1: Core Types and Planner

1. Create `internal/ai/planexecute/` package structure
2. Implement core types: `Plan`, `Step`, `Condition`, `ExecutionContext`
3. Implement `TemplateRegistry` with built-in templates
4. Implement `Planner` with structured output

### Phase 2: Executor

1. Implement `ToolRegistry` for dynamic tool selection
2. Implement `Executor` with timeout and checkpoint
3. Implement `CheckpointStore` interface (database-backed)
4. Add resume capability

### Phase 3: Replanner

1. Implement `ConvergenceChecker`
2. Implement `StepOptimizer` (merge/split)
3. Implement `RollbackDetector`
4. Integrate into `Replanner`

### Phase 4: Composition and Integration

1. Implement `PlanExecuteEngine` and `PlanExecuteAgent`
2. Migrate `DiagnosisAgent` to use custom implementation
3. Migrate `ChangeAgent` to use custom implementation
4. Update streaming/SSE events for new plan structure
5. Keep interrupt/resume contract unchanged: `Runner.ResumeWithParams(checkpointID, Targets[toolCallID])`

### Phase 5: Cleanup

1. Remove dependency on `github.com/cloudwego/eino/adk/prebuilt/planexecute`
2. Update documentation
3. Add unit tests for all components

## Testing Strategy

### Unit Tests

- `Plan` and `Step` serialization/deserialization
- `TemplateRegistry` operations
- `Planner` structured output parsing
- `Executor` step execution with timeout
- `Executor` checkpoint save/restore
- `Replanner` convergence detection
- `StepOptimizer` merge/split logic
- `RollbackDetector` analysis

### Integration Tests

- Full Plan-Execute-Replan loop
- Resume from interrupt (HITL)
- Template injection and execution
- Rollback flow

### E2E Tests

- DiagnosisAgent end-to-end scenarios
- ChangeAgent end-to-end scenarios with approval

## Open Questions

### Resolved

1. **Checkpoint storage**: Use adapter pattern with existing `internal/ai/checkpoint/store.go`. See `PlanCheckpointStore` adapter above.

2. **Template matching**: Automatic matching based on intent classification, with manual override via planner prompt.

3. **Step parallelism**: Defer to Phase 2 implementation. Focus on sequential execution first.

4. **Rollback templates**: Use template system for common rollback patterns (deployment rollback, scale revert). RollbackDetector can also generate dynamic steps.

## PersistedRuntime Mapping

The existing `runtime.PersistedRuntime` type is used for frontend communication and state persistence. Add conversion methods:

```go
// ToPersistedRuntime converts ExecutionContext to PersistedRuntime for frontend
func (ctx *ExecutionContext) ToPersistedRuntime() *runtime.PersistedRuntime {
    persisted := &runtime.PersistedRuntime{
        Phase:      string(ctx.Plan.Status),
        PhaseLabel: ctx.phaseLabel(),
    }

    // Convert plan steps
    if ctx.Plan != nil {
        persisted.Plan = &runtime.PersistedPlan{
            Steps:           ctx.toPersistedSteps(),
            ActiveStepIndex: ctx.CurrentStep,
        }
    }

    // Convert executed steps to activities
    for _, executed := range ctx.ExecutedSteps {
        persisted.Activities = append(persisted.Activities, runtime.PersistedActivity{
            ID:     executed.Step.ID,
            Kind:   "tool_call",
            Label:  executed.Step.Title,
            Status: string(executed.Status),
            Detail: truncateString(executed.Result, 200),
        })
    }

    return persisted
}

func (ctx *ExecutionContext) phaseLabel() string {
    switch ctx.Plan.Status {
    case PlanStatusPending:
        return "等待执行"
    case PlanStatusExecuting:
        return "正在执行"
    case PlanStatusCompleted:
        return "执行完成"
    case PlanStatusFailed:
        return "执行失败"
    case PlanStatusRolledBack:
        return "已回滚"
    }
    return ""
}

func (ctx *ExecutionContext) toPersistedSteps() []runtime.PersistedStep {
    var steps []runtime.PersistedStep
    for _, s := range ctx.Plan.Steps {
        steps = append(steps, runtime.PersistedStep{
            ID:      s.ID,
            Title:   s.Title,
            Status:  string(s.Status),
            Content: s.Result,
        })
    }
    return steps
}
```

## Feature Flag Integration

Add feature flag for gradual rollout:

```yaml
# configs/config.yaml
feature_flags:
  # Existing flag
  ai_assistant_v2: true

  # New flag for custom PlanExecute (default false for safe rollout)
  custom_planexecute: false
```

Migration strategy:

1. **Phase 1-3**: Develop with `custom_planexecute: false` (using Eino prebuilt)
2. **Phase 4**: Enable `custom_planexecute: true` in staging for testing
3. **Phase 5**: Enable in production, remove feature flag after validation

```go
// In agent factory
func NewDiagnosisAgent(ctx context.Context) (adk.ResumableAgent, error) {
    if featureFlags.CustomPlanExecute {
        return newCustomDiagnosisAgent(ctx)
    }
    return newEinoDiagnosisAgent(ctx) // Existing implementation
}
```

## References

- Existing implementation: `internal/ai/agents/diagnosis/agent.go`, `internal/ai/agents/change/agent.go`
- Eino ADK documentation: https://github.com/cloudwego/eino
- Previous design: `openspec/changes/archive/2026-03-13-ai-module-redesign/design.md`
