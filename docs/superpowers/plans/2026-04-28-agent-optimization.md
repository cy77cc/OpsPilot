# Agent System Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Optimize the OpsPilot Agent system for context efficiency, observability, and correctness based on frontier agent design principles (context engineering, dynamic tool discovery, verification subagents).

**Architecture:** Incremental refactoring — unify duplicated logic, introduce dynamic tool discovery to reduce context token consumption, add observability middleware, and strengthen the approval pipeline with consistent AST-based command classification.

**Tech Stack:** Go, Eino ADK v0.8.4, mvdan.cc/sh/v3, GORM

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `shared/sceneutil/sceneutil.go` | Shared scene normalization + allowed-tool-set |
| Create | `shared/sceneutil/sceneutil_test.go` | Tests for scene utilities |
| Create | `tools/discovery.go` | `search_tools` dynamic tool discovery |
| Create | `tools/discovery_test.go` | Tests for tool discovery |
| Create | `shared/middleware/observability.go` | Tracing + metrics middleware |
| Create | `shared/middleware/observability_test.go` | Tests for observability |
| Modify | `shared/middleware/scene_router.go` | Use `sceneutil.AllowedToolSet` |
| Modify | `shared/middleware/approval.go` | Unify `commandClassForTool` with `hostpolicy` AST |
| Modify | `shared/approval/events.go` | Unify 7 duplicate Input structs |
| Modify | `tools/factory.go` | Use `sceneutil.NormalizeScene` |
| Modify | `orchestrator/registry.go` | Use `sceneutil.NormalizeScene` |
| Modify | `shared/skill/backend.go` | Add `sync.RWMutex` cache |
| Modify | `orchestrator/factory.go` | Wire `search_tools` into agent creation |
| Modify | `tools/catalog.go` | Ensure all entries have descriptions |

---

### Task 1: Extract Shared Scene Utilities

**Goal:** Eliminate 3 duplicate `normalizeScene` functions and create an `O(1)` tool-allowance checker.

**Files:**
- Create: `internal/modules/ai/agent/shared/sceneutil/sceneutil.go`
- Create: `internal/modules/ai/agent/shared/sceneutil/sceneutil_test.go`

- [ ] **Step 1: Write tests for NormalizeScene**

```go
// shared/sceneutil/sceneutil_test.go
package sceneutil

import "testing"

func TestNormalizeScene(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "ai"},
		{"  ", "ai"},
		{"Kubernetes", "kubernetes"},
		{"  MONITORING  ", "monitoring"},
		{"ai", "ai"},
	}
	for _, tt := range tests {
		got := NormalizeScene(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeScene(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAllowedToolSet(t *testing.T) {
	set := NewAllowedToolSet([]string{"tool_a", "tool_b", "tool_c"})

	if !set.IsAllowed("tool_a") {
		t.Error("expected tool_a to be allowed")
	}
	if set.IsAllowed("tool_x") {
		t.Error("expected tool_x to be disallowed")
	}
	if set.Len() != 3 {
		t.Errorf("expected len 3, got %d", set.Len())
	}
}

func TestAllowedToolSet_Empty(t *testing.T) {
	set := NewAllowedToolSet(nil)
	if set.IsAllowed("anything") {
		t.Error("empty set should disallow everything")
	}
	if set.Len() != 0 {
		t.Errorf("expected len 0, got %d", set.Len())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/shared/sceneutil/... -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Implement sceneutil**

```go
// shared/sceneutil/sceneutil.go
// Package sceneutil provides shared scene normalization and tool-allowance utilities.
package sceneutil

import "strings"

const DefaultScene = "ai"

// NormalizeScene normalizes a scene name to lowercase trimmed form.
// Returns "ai" for empty input.
func NormalizeScene(scene string) string {
	scene = strings.TrimSpace(scene)
	if scene == "" {
		return DefaultScene
	}
	return strings.ToLower(scene)
}

// AllowedToolSet provides O(1) tool-name membership checks.
type AllowedToolSet struct {
	allowed map[string]struct{}
}

// NewAllowedToolSet creates a set from a slice of tool names.
func NewAllowedToolSet(names []string) *AllowedToolSet {
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}
	return &AllowedToolSet{allowed: set}
}

// IsAllowed returns true if the tool name is in the set.
func (s *AllowedToolSet) IsAllowed(name string) bool {
	if s == nil || s.allowed == nil {
		return false
	}
	_, ok := s.allowed[name]
	return ok
}

// Len returns the number of allowed tools.
func (s *AllowedToolSet) Len() int {
	if s == nil || s.allowed == nil {
		return 0
	}
	return len(s.allowed)
}

// Names returns the allowed tool names in no guaranteed order.
func (s *AllowedToolSet) Names() []string {
	if s == nil || s.allowed == nil {
		return nil
	}
	names := make([]string, 0, len(s.allowed))
	for n := range s.allowed {
		names = append(names, n)
	}
	return names
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/shared/sceneutil/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/modules/ai/agent/shared/sceneutil/
git commit -m "feat(agent): add shared scene normalization and allowed-tool-set utilities"
```

---

### Task 2: Deduplicate NormalizeScene Across Codebase

**Goal:** Replace 3 local `normalizeScene` functions with `sceneutil.NormalizeScene`.

**Files:**
- Modify: `internal/modules/ai/agent/tools/factory.go`
- Modify: `internal/modules/ai/agent/orchestrator/registry.go`
- Modify: `internal/modules/ai/agent/shared/middleware/scene_router.go`

- [ ] **Step 1: Update tools/factory.go**

Remove the local `normalizeScene` function and import `sceneutil`:

```go
// tools/factory.go — change import and usage
import (
	// ... existing imports ...
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/sceneutil"
)

// In BuildToolsForSceneWithMode, replace normalizeScene(scene) with sceneutil.NormalizeScene(scene)
// Delete the local normalizeScene function at the bottom of the file.
```

Specifically, replace line `switch normalizeScene(scene) {` with `switch sceneutil.NormalizeScene(scene) {` and delete the function at line 124-126.

- [ ] **Step 2: Update orchestrator/registry.go**

Replace `normalizeScene` calls with `sceneutil.NormalizeScene`:

```go
// orchestrator/registry.go
import (
	// ... existing imports ...
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/sceneutil"
)

// In Register(): r.byScene[normalizeScene(scene)] → r.byScene[sceneutil.NormalizeScene(scene)]
// In Lookup(): r.byScene[normalizeScene(scene)] → r.byScene[sceneutil.NormalizeScene(scene)]
// Delete the local normalizeScene function.
```

- [ ] **Step 3: Update shared/middleware/scene_router.go**

Replace `NormalizeScene` with `sceneutil.NormalizeScene` and delete the local `NormalizeScene` function:

```go
// shared/middleware/scene_router.go
import (
	// ... existing imports ...
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/sceneutil"
)

// In NewSceneRouter: currentScene := sceneutil.NormalizeScene(sceneMeta.Scene)
// In resolveScene: sceneutil.NormalizeScene(...) for both calls
// In buildSceneToolMapForAgent (handlers.go): sceneutil.NormalizeScene(...)
// Delete the local NormalizeScene function (lines 134-140).
// Keep DefaultSceneToolMap and DefaultScenePromptMap as-is.
```

- [ ] **Step 4: Run full build to verify no breakage**

Run: `cd /root/project/OpsPilot && go build ./internal/modules/ai/agent/...`
Expected: Build succeeds

- [ ] **Step 5: Run existing tests**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/... -v -count=1`
Expected: All existing tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/modules/ai/agent/tools/factory.go internal/modules/ai/agent/orchestrator/registry.go internal/modules/ai/agent/shared/middleware/scene_router.go
git commit -m "refactor(agent): deduplicate normalizeScene using shared sceneutil package"
```

---

### Task 3: Scene Router — Use AllowedToolSet for O(1) Lookup

**Goal:** Replace linear `isToolAllowed` scan with `sceneutil.AllowedToolSet`.

**Files:**
- Modify: `internal/modules/ai/agent/shared/middleware/scene_router.go`

- [ ] **Step 1: Update SceneRouterMiddleware to use AllowedToolSet**

```go
// scene_router.go — modify the struct
type SceneRouterMiddleware struct {
	*adk.BaseChatModelAgentMiddleware

	// sceneToolSets maps scene name to an O(1) allowed-tool set
	sceneToolSets map[string]*sceneutil.AllowedToolSet

	// currentScene is the runtime scene
	currentScene string
}
```

- [ ] **Step 2: Update NewSceneRouter to build AllowedToolSet map**

```go
func NewSceneRouter(ctx context.Context, cfg *SceneRouterConfig) (*SceneRouterMiddleware, error) {
	if cfg == nil {
		cfg = &SceneRouterConfig{}
	}
	if cfg.SceneToolMap == nil {
		cfg.SceneToolMap = DefaultSceneToolMap()
	}

	sceneToolSets := make(map[string]*sceneutil.AllowedToolSet, len(cfg.SceneToolMap))
	for scene, tools := range cfg.SceneToolMap {
		sceneToolSets[sceneutil.NormalizeScene(scene)] = sceneutil.NewAllowedToolSet(tools)
	}

	sceneMeta := runtimectx.AIMetadataFrom(ctx)
	currentScene := sceneutil.NormalizeScene(sceneMeta.Scene)

	return &SceneRouterMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		sceneToolSets:                sceneToolSets,
		currentScene:                 currentScene,
	}, nil
}
```

- [ ] **Step 3: Update WrapInvokableToolCall and WrapStreamableToolCall**

Replace `allowedTools := m.sceneToolMap[scene]` + `isToolAllowed(...)` with:

```go
func (m *SceneRouterMiddleware) WrapInvokableToolCall(
	ctx context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	if tCtx == nil {
		return endpoint, nil
	}
	scene := m.resolveScene(ctx)
	toolSet := m.sceneToolSets[scene]
	if toolSet != nil && !toolSet.IsAllowed(tCtx.Name) {
		return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
			return "", fmt.Errorf("tool '%s' is not available in scene '%s'", tCtx.Name, scene)
		}, nil
	}
	return endpoint, nil
}

// Same pattern for WrapStreamableToolCall
```

- [ ] **Step 4: Delete the old isToolAllowed function**

Remove the `isToolAllowed` function (lines 121-128).

- [ ] **Step 5: Run tests**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/shared/middleware/... -v -count=1`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/ai/agent/shared/middleware/scene_router.go
git commit -m "perf(agent): use O(1) AllowedToolSet in scene router instead of linear scan"
```

---

### Task 4: Unify Host Command Classification with AST Engine

**Goal:** Eliminate the string-matching `classifyHostCommand` and `isReadonlyHostCommand` functions in `approval.go`, replacing them with the existing `hostpolicy` AST engine. This fixes the security issue where `isReadonlyHostCommand` has only 5 hardcoded entries while `DefaultReadonlyAllowlist` has 40+.

**Files:**
- Modify: `internal/modules/ai/agent/shared/middleware/approval.go`

- [ ] **Step 1: Write a test for the unified commandClassForTool**

Create or extend `internal/modules/ai/agent/shared/middleware/approval_test.go`:

```go
package middleware

import "testing"

func TestCommandClassForTool_HostExec(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		args     string
		want     string
	}{
		{
			name:     "readonly command via AST",
			toolName: "host_exec",
			args:     `{"command": "df -h"}`,
			want:     "readonly",
		},
		{
			name:     "non-readonly command",
			toolName: "host_exec",
			args:     `{"command": "systemctl restart nginx"}`,
			want:     "service_control",
		},
		{
			name:     "empty command",
			toolName: "host_exec",
			args:     `{"command": ""}`,
			want:     "unknown",
		},
		{
			name:     "delete tool name",
			toolName: "k8s_delete_pod",
			args:     `{}`,
			want:     "write",
		},
		{
			name:     "read tool name",
			toolName: "k8s_query",
			args:     `{}`,
			want:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commandClassForTool(tt.toolName, tt.args)
			if got != tt.want {
				t.Errorf("commandClassForTool(%q, %q) = %q, want %q", tt.toolName, tt.args, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to see current behavior**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/shared/middleware/... -run TestCommandClassForTool_HostExec -v`
Expected: Some cases may fail — this documents the current behavior delta.

- [ ] **Step 3: Replace commandClassForTool to use hostpolicy AST**

Replace the `commandClassForTool`, `hostExecCommandClass`, `hostCommandClassFromMap`, `classifyHostCommand`, `isReadonlyHostCommand`, and `normalizeWhitespace` functions (lines 502-583) with:

```go
func commandClassForTool(toolName, args string) string {
	toolName = strings.ToLower(strings.TrimSpace(toolName))
	switch toolName {
	case "host_exec":
		return hostExecCommandClassFromAST(args)
	case "host_batch_status_update":
		return "service_control"
	default:
		return unknownWhenEmpty(defaultCommandClass(toolName))
	}
}

// hostExecCommandClassFromAST uses the hostpolicy AST engine to classify commands.
// This replaces the old string-matching classifyHostCommand function.
func hostExecCommandClassFromAST(args string) string {
	cmd := extractCommandFromArgs(args)
	if cmd == "" {
		return "unknown"
	}

	parsed, err := host.ParseCommand(cmd)
	if err != nil {
		return "unknown"
	}

	engine := getHostPolicyEngine()
	violations := engine.Validator().Validate(parsed)
	if len(violations) == 0 {
		return "readonly"
	}
	return "service_control"
}

// extractCommandFromArgs extracts the command string from tool arguments JSON.
func extractCommandFromArgs(args string) string {
	var params map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(args)), &params); err != nil {
		return ""
	}
	if cmd, _ := params["command"].(string); strings.TrimSpace(cmd) != "" {
		return strings.TrimSpace(cmd)
	}
	if script, _ := params["script"].(string); strings.TrimSpace(script) != "" {
		return strings.TrimSpace(script)
	}
	return ""
}
```

Note: The `hostpolicy.HostCommandPolicyEngine` needs to expose its validator. Add a `Validator()` method:

In `shared/hostpolicy/engine.go`, add:
```go
// Validator returns the underlying command validator.
func (e *HostCommandPolicyEngine) Validator() *HostCommandValidator {
	return e.validator
}
```

- [ ] **Step 4: Delete old functions**

Delete `classifyHostCommand`, `isReadonlyHostCommand`, `normalizeWhitespace`, and `hostCommandClassFromMap` from `approval.go`.

- [ ] **Step 5: Run the test to verify**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/shared/middleware/... -run TestCommandClassForTool_HostExec -v`
Expected: PASS

- [ ] **Step 6: Run full build**

Run: `cd /root/project/OpsPilot && go build ./internal/modules/ai/agent/...`
Expected: Build succeeds

- [ ] **Step 7: Commit**

```bash
git add internal/modules/ai/agent/shared/middleware/approval.go internal/modules/ai/agent/shared/middleware/approval_test.go internal/modules/ai/agent/shared/hostpolicy/engine.go
git commit -m "fix(agent): unify host command classification with hostpolicy AST engine"
```

---

### Task 5: Unify Approval Event Input Types

**Goal:** Replace 7 nearly identical Input structs in `events.go` with a single `ApprovalEventInput` struct.

**Files:**
- Modify: `internal/modules/ai/agent/shared/approval/events.go`

- [ ] **Step 1: Write tests for the unified input type**

Create `internal/modules/ai/agent/shared/approval/events_test.go`:

```go
package approval

import (
	"testing"
	"time"
)

func TestNewApprovalEventEnvelope_Unified(t *testing.T) {
	input := ApprovalEventInput{
		EventID:     "evt-1",
		OccurredAt:  time.Now(),
		Sequence:    1,
		Version:     1,
		RunID:       "run-1",
		SessionID:    "sess-1",
		ApprovalID:   "appr-1",
		ToolCallID:   "call-1",
		AggregateID:  "run-1",
		Payload:      map[string]string{"key": "value"},
	}

	env, err := NewApprovalRequestedEnvelope(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.EventType != ApprovalEventTypeRequested {
		t.Errorf("expected %q, got %q", ApprovalEventTypeRequested, env.EventType)
	}
	if env.ApprovalID != "appr-1" {
		t.Errorf("expected approval_id appr-1, got %q", env.ApprovalID)
	}
}

func TestNewApprovalEventEnvelope_Validation(t *testing.T) {
	// Missing required fields
	input := ApprovalEventInput{}
	_, err := NewApprovalRequestedEnvelope(input)
	if err == nil {
		t.Error("expected error for missing required fields")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/shared/approval/... -run TestNewApprovalEventEnvelope_Unified -v`
Expected: FAIL — `ApprovalEventInput` does not exist yet

- [ ] **Step 3: Refactor events.go**

Replace the 7 input structs and their constructor functions with:

```go
// ApprovalEventInput is the unified input for creating approval event envelopes.
// Replaces ApprovalRequestedInput, ApprovalDecidedInput, ApprovalExpiredInput,
// RunResumingInput, RunResumedInput, RunResumeFailedInput, RunCompletedInput.
type ApprovalEventInput struct {
	EventID     string
	OccurredAt  time.Time
	Sequence    int64
	Version     int
	RunID       string
	SessionID   string
	ApprovalID  string
	ToolCallID  string
	AggregateID string
	Payload     any
}

func NewApprovalRequestedEnvelope(input ApprovalEventInput) (*ApprovalEventEnvelope, error) {
	return newApprovalEventEnvelope(ApprovalEventTypeRequested, input.EventID, input.OccurredAt,
		input.Sequence, input.Version, input.RunID, input.SessionID,
		input.ApprovalID, input.ToolCallID, input.AggregateID, input.Payload)
}

func NewApprovalDecidedEnvelope(input ApprovalEventInput) (*ApprovalEventEnvelope, error) {
	return newApprovalEventEnvelope(ApprovalEventTypeDecided, input.EventID, input.OccurredAt,
		input.Sequence, input.Version, input.RunID, input.SessionID,
		input.ApprovalID, input.ToolCallID, input.AggregateID, input.Payload)
}

func NewApprovalExpiredEnvelope(input ApprovalEventInput) (*ApprovalEventEnvelope, error) {
	return newApprovalEventEnvelope(ApprovalEventTypeExpired, input.EventID, input.OccurredAt,
		input.Sequence, input.Version, input.RunID, input.SessionID,
		input.ApprovalID, input.ToolCallID, input.AggregateID, input.Payload)
}

func NewRunResumingEnvelope(input ApprovalEventInput) (*ApprovalEventEnvelope, error) {
	return newApprovalEventEnvelope(RunEventTypeResuming, input.EventID, input.OccurredAt,
		input.Sequence, input.Version, input.RunID, input.SessionID,
		input.ApprovalID, input.ToolCallID, input.AggregateID, input.Payload)
}

func NewRunResumedEnvelope(input ApprovalEventInput) (*ApprovalEventEnvelope, error) {
	return newApprovalEventEnvelope(RunEventTypeResumed, input.EventID, input.OccurredAt,
		input.Sequence, input.Version, input.RunID, input.SessionID,
		input.ApprovalID, input.ToolCallID, input.AggregateID, input.Payload)
}

func NewRunResumeFailedEnvelope(input ApprovalEventInput) (*ApprovalEventEnvelope, error) {
	return newApprovalEventEnvelope(RunEventTypeResumeFailed, input.EventID, input.OccurredAt,
		input.Sequence, input.Version, input.RunID, input.SessionID,
		input.ApprovalID, input.ToolCallID, input.AggregateID, input.Payload)
}

func NewRunCompletedEnvelope(input ApprovalEventInput) (*ApprovalEventEnvelope, error) {
	return newApprovalEventEnvelope(RunEventTypeCompleted, input.EventID, input.OccurredAt,
		input.Sequence, input.Version, input.RunID, input.SessionID,
		input.ApprovalID, input.ToolCallID, input.AggregateID, input.Payload)
}
```

- [ ] **Step 4: Find and update all callers of the old Input types**

Run: `cd /root/project/OpsPilot && grep -rn "ApprovalRequestedInput\|ApprovalDecidedInput\|ApprovalExpiredInput\|RunResumingInput\|RunResumedInput\|RunResumeFailedInput\|RunCompletedInput" --include="*.go" | grep -v "_test.go" | grep -v "events.go"`

Update each caller to use `ApprovalEventInput` instead. The field names are identical, so this is a type-name-only change.

- [ ] **Step 5: Run tests**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/shared/approval/... -v`
Expected: PASS

- [ ] **Step 6: Run full build**

Run: `cd /root/project/OpsPilot && go build ./internal/modules/ai/agent/...`
Expected: Build succeeds

- [ ] **Step 7: Commit**

```bash
git add internal/modules/ai/agent/shared/approval/events.go internal/modules/ai/agent/shared/approval/events_test.go
git commit -m "refactor(agent): unify 7 duplicate approval event input structs into single ApprovalEventInput"
```

---

### Task 6: Dynamic Tool Discovery (search_tools)

**Goal:** Implement a `search_tools` tool that lets the agent discover tools on-demand, reducing context token consumption by up to 85% (per Anthropic data).

**Files:**
- Create: `internal/modules/ai/agent/tools/discovery.go`
- Create: `internal/modules/ai/agent/tools/discovery_test.go`
- Modify: `internal/modules/ai/agent/tools/catalog.go` (ensure descriptions exist)

- [ ] **Step 1: Write tests for search_tools**

```go
// tools/discovery_test.go
package tools

import (
	"context"
	"strings"
	"testing"
)

func TestSearchToolsTool_FindsByName(t *testing.T) {
	catalog := NewCatalog([]CatalogEntry{
		{Name: "k8s_query", Description: "Query Kubernetes resources with SQL-like syntax", Scene: "kubernetes"},
		{Name: "k8s_list_resources", Description: "List Kubernetes resources in a namespace", Scene: "kubernetes"},
		{Name: "monitor_metric", Description: "Query Prometheus metrics", Scene: "monitoring"},
	})

	tool := NewSearchToolsTool(catalog)
	info, _ := tool.Info(context.Background())
	if info.Name != "search_tools" {
		t.Errorf("expected tool name search_tools, got %q", info.Name)
	}

	result, err := tool.InvokableRun(context.Background(), `{"query": "kubernetes"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "k8s_query") {
		t.Errorf("expected result to contain k8s_query, got %q", result)
	}
	if strings.Contains(result, "monitor_metric") {
		t.Errorf("kubernetes query should not return monitoring tools")
	}
}

func TestSearchToolsTool_FindsByDescription(t *testing.T) {
	catalog := NewCatalog([]CatalogEntry{
		{Name: "host_exec", Description: "Execute shell commands on remote hosts", Scene: "host"},
		{Name: "cicd_pipeline_trigger", Description: "Trigger a CI/CD pipeline run", Scene: "cicd"},
	})

	tool := NewSearchToolsTool(catalog)
	result, err := tool.InvokableRun(context.Background(), `{"query": "pipeline"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "cicd_pipeline_trigger") {
		t.Errorf("expected cicd_pipeline_trigger in results")
	}
	if strings.Contains(result, "host_exec") {
		t.Errorf("pipeline query should not return host_exec")
	}
}

func TestSearchToolsTool_EmptyQuery(t *testing.T) {
	catalog := NewCatalog([]CatalogEntry{
		{Name: "k8s_query", Description: "Query Kubernetes resources", Scene: "kubernetes"},
	})

	tool := NewSearchToolsTool(catalog)
	result, err := tool.InvokableRun(context.Background(), `{"query": ""}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty query returns all tools (up to limit)
	if !strings.Contains(result, "k8s_query") {
		t.Errorf("empty query should return all tools")
	}
}

func TestSearchToolsTool_NoMatch(t *testing.T) {
	catalog := NewCatalog([]CatalogEntry{
		{Name: "k8s_query", Description: "Query Kubernetes resources", Scene: "kubernetes"},
	})

	tool := NewSearchToolsTool(catalog)
	result, err := tool.InvokableRun(context.Background(), `{"query": "nonexistent_xyz"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "k8s_query") {
		t.Errorf("non-matching query should not return results")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/tools/... -run TestSearchToolsTool -v`
Expected: FAIL — `NewSearchToolsTool` does not exist

- [ ] **Step 3: Implement search_tools**

```go
// tools/discovery.go
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool/utils"
)

// SearchToolsInput is the input for the search_tools dynamic discovery tool.
type SearchToolsInput struct {
	Query string `json:"query" jsonschema:"description=Search query to find relevant tools by name, description, or domain keywords"`
}

// SearchToolsResult represents a single tool search result.
type SearchToolsResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Scene       string `json:"scene"`
}

const searchToolsMaxResults = 10

// NewSearchToolsTool creates a tool that lets the agent discover tools on-demand.
// This reduces context token usage by up to 85% while improving tool selection accuracy.
// Reference: Anthropic "Building Multi-Agent Systems" — Tool Discovery pattern.
func NewSearchToolsTool(catalog *Catalog) (*utils.Tool[SearchToolsInput, string], error) {
	return utils.InferTool(
		"search_tools",
		"Search available tools by keyword or capability. Use this to discover tools before calling them. Returns matching tool names, descriptions, and domains.",
		func(ctx context.Context, input SearchToolsInput) (string, error) {
			query := strings.TrimSpace(input.Query)
			results := catalog.Search(query, searchToolsMaxResults)

			if len(results) == 0 {
				return "No tools found matching the query.", nil
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Found %d tool(s):\n", len(results)))
			for _, r := range results {
				sb.WriteString(fmt.Sprintf("- %s: %s [%s]\n", r.Name, r.Description, r.Scene))
			}
			return sb.String(), nil
		},
	)
}
```

- [ ] **Step 4: Add Search method to Catalog**

In `tools/catalog.go`, add:

```go
// Search finds tools matching the query across name, description, and scene fields.
// Returns up to maxResults entries sorted by relevance.
func (c *Catalog) Search(query string, maxResults int) []CatalogEntry {
	if c == nil {
		return nil
	}
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		// Return all up to limit
		if maxResults <= 0 || maxResults > len(c.entries) {
			maxResults = len(c.entries)
		}
		out := make([]CatalogEntry, maxResults)
		copy(out, c.entries[:maxResults])
		return out
	}

	queryTerms := strings.Fields(query)
	var scored []struct {
		entry CatalogEntry
		score int
	}

	for _, entry := range c.entries {
		score := scoreEntry(entry, queryTerms)
		if score > 0 {
			scored = append(scored, struct {
				entry CatalogEntry
				score int
			}{entry, score})
		}
	}

	// Sort by score descending
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > len(scored) {
		maxResults = len(scored)
	}

	results := make([]CatalogEntry, maxResults)
	for i := 0; i < maxResults; i++ {
		results[i] = scored[i].entry
	}
	return results
}

func scoreEntry(entry CatalogEntry, queryTerms []string) int {
	score := 0
	name := strings.ToLower(entry.Name)
	desc := strings.ToLower(entry.Description)
	scene := strings.ToLower(entry.Scene)

	for _, term := range queryTerms {
		if strings.Contains(name, term) {
			score += 3 // Name match is highest priority
		}
		if strings.Contains(desc, term) {
			score += 2
		}
		if strings.Contains(scene, term) {
			score += 1
		}
	}
	return score
}
```

- [ ] **Step 5: Run tests**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/tools/... -run TestSearchToolsTool -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/ai/agent/tools/discovery.go internal/modules/ai/agent/tools/discovery_test.go internal/modules/ai/agent/tools/catalog.go
git commit -m "feat(agent): add search_tools dynamic tool discovery to reduce context token consumption"
```

---

### Task 7: Wire search_tools into Agent Creation

**Goal:** Register `search_tools` alongside core scene tools in the agent factory, so agents can discover tools on-demand instead of having all 34 tools in context.

**Files:**
- Modify: `internal/modules/ai/agent/orchestrator/factory.go`

- [ ] **Step 1: Update createDeepAgent to include search_tools**

In `createDeepAgent`, after building `sceneTools`, add the search_tools tool:

```go
func createDeepAgent(ctx context.Context, registry *Registry, scene string) (adk.ResumableAgent, error) {
	// ... existing code up to sceneTools := tools.BuildToolsForSceneWithMode(...) ...

	// Register search_tools for dynamic tool discovery
	searchTool, err := tools.NewSearchToolsTool(tools.GlobalCatalog())
	if err == nil {
		sceneTools = append([]tool.BaseTool{searchTool}, sceneTools...)
	}

	// ... rest of function unchanged ...
}
```

- [ ] **Step 2: Add GlobalCatalog accessor to tools package**

In `tools/catalog.go`, add:

```go
var globalCatalog *Catalog

// GlobalCatalog returns the global tool catalog.
func GlobalCatalog() *Catalog {
	if globalCatalog == nil {
		globalCatalog = NewCatalog(DefaultCatalogEntries())
	}
	return globalCatalog
}
```

- [ ] **Step 3: Run build**

Run: `cd /root/project/OpsPilot && go build ./internal/modules/ai/agent/...`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add internal/modules/ai/agent/orchestrator/factory.go internal/modules/ai/agent/tools/catalog.go
git commit -m "feat(agent): wire search_tools into deep agent creation for on-demand tool discovery"
```

---

### Task 8: Add Skill Backend Cache

**Goal:** Cache loaded skills to avoid re-reading from disk on every `List`/`Get` call.

**Files:**
- Modify: `internal/modules/ai/agent/shared/skill/backend.go`

- [ ] **Step 1: Write test for caching behavior**

Create `internal/modules/ai/agent/shared/skill/backend_test.go`:

```go
package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBackend_CachesSkills(t *testing.T) {
	// Create a temp skill directory
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: test-skill
description: A test skill
---
Skill content here.
`), 0644)

	backend, err := New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	// First call — loads from disk
	skills1, err := backend.List(context.Background())
	if err != nil {
		t.Fatalf("first List failed: %v", err)
	}
	if len(skills1) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills1))
	}

	// Remove the file — cache should still return it
	os.RemoveAll(skillDir)

	// Second call — should return cached result
	skills2, err := backend.List(context.Background())
	if err != nil {
		t.Fatalf("second List failed: %v", err)
	}
	if len(skills2) != 1 {
		t.Errorf("expected cached 1 skill after file deletion, got %d", len(skills2))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/shared/skill/... -run TestBackend_CachesSkills -v`
Expected: FAIL — second List returns 0 because no cache

- [ ] **Step 3: Add cache to Backend**

```go
// shared/skill/backend.go — modify Backend struct
type Backend struct {
	baseDir string

	cacheMu sync.RWMutex
	cached  []skill.Skill
	loaded  bool
}

func (b *Backend) loadAll() ([]skill.Skill, error) {
	// Fast path: read from cache
	b.cacheMu.RLock()
	if b.loaded {
		defer b.cacheMu.RUnlock()
		return b.cached, nil
	}
	b.cacheMu.RUnlock()

	// Slow path: load from disk
	b.cacheMu.Lock()
	defer b.cacheMu.Unlock()

	// Double-check after acquiring write lock
	if b.loaded {
		return b.cached, nil
	}

	skills, err := b.loadAllFromDisk()
	if err != nil {
		return nil, err
	}
	b.cached = skills
	b.loaded = true
	return skills, nil
}

// loadAllFromDisk reads skills from the filesystem.
func (b *Backend) loadAllFromDisk() ([]skill.Skill, error) {
	entries, err := os.ReadDir(b.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill directory %q: %w", b.baseDir, err)
	}

	var skills []skill.Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(b.baseDir, entry.Name(), skillFileName)
		s, err := b.loadSkill(skillPath)
		if err != nil {
			continue
		}
		skills = append(skills, s)
	}
	return skills, nil
}
```

Add `"sync"` to the import list.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/shared/skill/... -run TestBackend_CachesSkills -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/modules/ai/agent/shared/skill/backend.go internal/modules/ai/agent/shared/skill/backend_test.go
git commit -m "perf(agent): add sync.RWMutex cache to skill backend to avoid repeated disk reads"
```

---

### Task 9: Add Observability Middleware

**Goal:** Add tracing and metrics to the agent middleware pipeline for operational visibility.

**Files:**
- Create: `internal/modules/ai/agent/shared/middleware/observability.go`
- Create: `internal/modules/ai/agent/shared/middleware/observability_test.go`
- Modify: `internal/modules/ai/agent/shared/middleware/handlers.go`

- [ ] **Step 1: Write tests for observability middleware**

```go
// shared/middleware/observability_test.go
package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/middleware/metricsnoop"
)

func TestObservabilityMiddleware_RecordsToolCall(t *testing.T) {
	noop := metricsnoop.New()
	mw := NewObservabilityMiddleware(noop)

	called := false
	endpoint := func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		called = true
		return "ok", nil
	}

	wrapped, err := mw.WrapInvokableToolCall(context.Background(), endpoint, &adk.ToolContext{
		Name: "test_tool",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _ = wrapped(context.Background(), `{}`)

	if !called {
		t.Error("expected endpoint to be called")
	}

	metrics := noop.Snapshot()
	if metrics.ToolCallCount != 1 {
		t.Errorf("expected 1 tool call, got %d", metrics.ToolCallCount)
	}
	if metrics.ToolCallErrors != 0 {
		t.Errorf("expected 0 errors, got %d", metrics.ToolCallErrors)
	}
}

func TestObservabilityMiddleware_RecordsError(t *testing.T) {
	noop := metricsnoop.New()
	mw := NewObservabilityMiddleware(noop)

	endpoint := func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		return "", fmt.Errorf("tool failed")
	}

	wrapped, _ := mw.WrapInvokableToolCall(context.Background(), endpoint, &adk.ToolContext{
		Name: "failing_tool",
	})
	_, _ = wrapped(context.Background(), `{}`)

	metrics := noop.Snapshot()
	if metrics.ToolCallErrors != 1 {
		t.Errorf("expected 1 error, got %d", metrics.ToolCallErrors)
	}
}
```

- [ ] **Step 2: Create metricsnoop test helper**

Create `internal/modules/ai/agent/shared/middleware/metricsnoop/metricsnoop.go`:

```go
// Package metricsnoop provides a no-op metrics collector for testing.
package metricsnoop

import "sync"

// Metrics holds observed metric values.
type Metrics struct {
	ToolCallCount  int
	ToolCallErrors int
	ToolCallDurationSum time.Duration
}

// Collector implements ObservabilityMetrics for testing.
type Collector struct {
	mu      sync.Mutex
	metrics Metrics
}

// New creates a new noop metrics collector.
func New() *Collector {
	return &Collector{}
}

func (c *Collector) RecordToolCall(toolName string, duration time.Duration, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics.ToolCallCount++
	c.metrics.ToolCallDurationSum += duration
	if err != nil {
		c.metrics.ToolCallErrors++
	}
}

func (c *Collector) Snapshot() Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.metrics
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/shared/middleware/... -run TestObservabilityMiddleware -v`
Expected: FAIL — `ObservabilityMiddleware` does not exist

- [ ] **Step 4: Implement observability middleware**

```go
// shared/middleware/observability.go
package middleware

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
)

// ObservabilityMetrics defines the metrics collection interface.
// Implementations can export to Prometheus, OpenTelemetry, or in-memory for testing.
type ObservabilityMetrics interface {
	RecordToolCall(toolName string, duration time.Duration, err error)
}

// ObservabilityMiddleware provides tracing and metrics for agent tool calls.
type ObservabilityMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	metrics ObservabilityMetrics
}

// NewObservabilityMiddleware creates a new observability middleware.
func NewObservabilityMiddleware(metrics ObservabilityMetrics) *ObservabilityMiddleware {
	return &ObservabilityMiddleware{metrics: metrics}
}

func (m *ObservabilityMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	toolName := ""
	if tCtx != nil {
		toolName = tCtx.Name
	}

	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		start := time.Now()
		result, err := endpoint(ctx, args, opts...)
		if m.metrics != nil {
			m.metrics.RecordToolCall(toolName, time.Since(start), err)
		}
		return result, err
	}, nil
}

func (m *ObservabilityMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	toolName := ""
	if tCtx != nil {
		toolName = tCtx.Name
	}

	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		start := time.Now()
		result, err := endpoint(ctx, args, opts...)
		if m.metrics != nil {
			m.metrics.RecordToolCall(toolName, time.Since(start), err)
		}
		return result, err
	}, nil
}
```

- [ ] **Step 5: Run tests**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/shared/middleware/... -run TestObservabilityMiddleware -v`
Expected: PASS

- [ ] **Step 6: Wire into BuildAgentHandlers**

In `handlers.go`, add observability middleware to the chain:

```go
func BuildAgentHandlers(ctx context.Context, scene string, tools []tool.BaseTool) ([]adk.ChatModelAgentMiddleware, error) {
	var middlewares []adk.ChatModelAgentMiddleware

	// 1. Observability (outermost — sees everything)
	noopMetrics := &noopObservabilityMetrics{} // Replace with real metrics in production
	middlewares = append(middlewares, NewObservabilityMiddleware(noopMetrics))

	// ... rest of existing middleware chain unchanged ...
}

// noopObservabilityMetrics is a no-op implementation for when metrics are not configured.
type noopObservabilityMetrics struct{}

func (n *noopObservabilityMetrics) RecordToolCall(toolName string, duration time.Duration, err error) {}
```

- [ ] **Step 7: Run full build**

Run: `cd /root/project/OpsPilot && go build ./internal/modules/ai/agent/...`
Expected: Build succeeds

- [ ] **Step 8: Commit**

```bash
git add internal/modules/ai/agent/shared/middleware/observability.go internal/modules/ai/agent/shared/middleware/observability_test.go internal/modules/ai/agent/shared/middleware/metricsnoop/ internal/modules/ai/agent/shared/middleware/handlers.go
git commit -m "feat(agent): add observability middleware for tool call tracing and metrics"
```

---

### Task 10: Ensure All Catalog Entries Have Descriptions

**Goal:** Every tool in the catalog should have a meaningful description for `search_tools` to work effectively and for the LLM to select tools accurately.

**Files:**
- Modify: `internal/modules/ai/agent/tools/catalog.go`

- [ ] **Step 1: Write a test that validates catalog descriptions**

```go
// tools/catalog_test.go
package tools

import (
	"strings"
	"testing"
)

func TestCatalog_AllEntriesHaveDescriptions(t *testing.T) {
	catalog := NewCatalog(DefaultCatalogEntries())
	for _, entry := range catalog.All() {
		desc := strings.TrimSpace(entry.Description)
		if desc == "" {
			t.Errorf("tool %q has empty description", entry.Name)
		}
		if len(desc) < 10 {
			t.Errorf("tool %q description too short (%d chars): %q", entry.Name, len(desc), desc)
		}
	}
}

func TestCatalog_NoDuplicateNames(t *testing.T) {
	catalog := NewCatalog(DefaultCatalogEntries())
	seen := make(map[string]bool)
	for _, entry := range catalog.All() {
		if seen[entry.Name] {
			t.Errorf("duplicate tool name: %q", entry.Name)
		}
		seen[entry.Name] = true
	}
}
```

- [ ] **Step 2: Run the test**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/tools/... -run TestCatalog_AllEntriesHaveDescriptions -v`
Expected: May fail if any entries lack descriptions.

- [ ] **Step 3: Fill in missing descriptions**

Update catalog entries that lack descriptions. Each description should be 1-2 sentences explaining what the tool does AND when to use it (ACI principle).

- [ ] **Step 4: Run test to verify all pass**

Run: `cd /root/project/OpsPilot && go test ./internal/modules/ai/agent/tools/... -run TestCatalog -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/modules/ai/agent/tools/catalog.go internal/modules/ai/agent/tools/catalog_test.go
git commit -m "docs(agent): ensure all tool catalog entries have meaningful descriptions for ACI compliance"
```

---

## Execution Summary

| Task | Priority | Effort | Dependencies |
|------|----------|--------|-------------|
| 1. Extract sceneutil | P0 | 0.5d | — |
| 2. Deduplicate NormalizeScene | P0 | 0.5d | Task 1 |
| 3. AllowedToolSet in scene router | P0 | 0.5d | Task 1 |
| 4. Unify host command classification | P0 | 1d | — |
| 5. Unify approval event types | P1 | 1d | — |
| 6. search_tools discovery | P1 | 2d | — |
| 7. Wire search_tools into factory | P1 | 0.5d | Task 6 |
| 8. Skill backend cache | P1 | 0.5d | — |
| 9. Observability middleware | P1 | 1.5d | — |
| 10. Catalog descriptions | P1 | 0.5d | Task 6 |

**Total estimated effort:** ~9 days

**Parallelizable tracks:**
- Track A: Tasks 1 → 2 → 3 (sceneutil refactor)
- Track B: Task 4 (host policy unification)
- Track C: Task 5 (approval events)
- Track D: Tasks 6 → 7 → 10 (tool discovery)
- Track E: Task 8 (skill cache)
- Track F: Task 9 (observability)

Tracks B, C, E, F can all run in parallel with Track A and D.
