package ai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	aiobs "github.com/cy77cc/OpsPilot/internal/ai/observability"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

type lifecycleObserver struct {
	records []aiobs.ThoughtChainLifecycleRecord
}

func (o *lifecycleObserver) OnThoughtChainLifecycle(record aiobs.ThoughtChainLifecycleRecord) {
	o.records = append(o.records, record)
}

func TestOrchestratorStreamExecution_EmitsThoughtChainLifecycleWithoutLegacyPrimaryEvents(t *testing.T) {
	ctx := context.Background()
	observer := &lifecycleObserver{}
	unregister := aiobs.RegisterObserver(observer)
	defer unregister()

	orchestrator := &Orchestrator{
		executions: airuntime.NewExecutionStore(nil, ""),
		converter:  airuntime.NewSSEConverter(),
	}
	state := airuntime.ExecutionState{
		TraceID:      "trace-1",
		SessionID:    "sess-1",
		PlanID:       "plan-1",
		TurnID:       "turn-1",
		Scene:        "deployment:hosts",
		Status:       airuntime.ExecutionStatusRunning,
		Phase:        string(airuntime.PhasePlanning),
		CheckpointID: "checkpoint-1",
		Steps:        map[string]airuntime.StepState{},
	}

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(assistantEvent("planner", `[{"id":"step-1","title":"获取主机列表"},{"id":"step-2","title":"汇总状态"}]`))
	gen.Send(toolEvent("host_list_inventory", `{"total":1,"list":[{"id":1,"name":"test","status":"online"}]}`))
	gen.Send(assistantEvent("replanner", "1. 汇总主机状态\n2. 输出详细结果"))
	gen.Send(assistantEvent("executor", "## 主机状态\n\n| 名称 | 状态 |\n| --- | --- |\n| test | online |\n"))
	gen.Close()

	var emitted []airuntime.StreamEvent
	if _, err := orchestrator.streamExecution(ctx, iter, &state, func(event airuntime.StreamEvent) bool {
		emitted = append(emitted, event)
		return true
	}); err != nil {
		t.Fatalf("streamExecution returned error: %v", err)
	}

	assertEventTypePresent(t, emitted, airuntime.EventChainStarted)
	assertEventTypePresent(t, emitted, airuntime.EventChainNodeOpen)
	assertEventTypePresent(t, emitted, airuntime.EventChainNodeReplace)
	assertEventTypePresent(t, emitted, airuntime.EventFinalAnswerStart)
	assertEventTypePresent(t, emitted, airuntime.EventFinalAnswerDone)
	assertEventTypePresent(t, emitted, airuntime.EventChainCompleted)
	assertEventTypePresent(t, emitted, airuntime.EventDone)
	assertEventTypeAbsent(t, emitted, airuntime.EventType("turn_started"))
	assertEventTypeAbsent(t, emitted, airuntime.EventType("phase_started"))
	assertEventTypeAbsent(t, emitted, airuntime.EventType("block_open"))

	if state.Status != airuntime.ExecutionStatusCompleted {
		t.Fatalf("expected completed state, got %q", state.Status)
	}
	if len(observer.records) == 0 {
		t.Fatalf("expected lifecycle callbacks to be emitted")
	}
	assertLifecycleEvent(t, observer.records, "chain_started")
	assertLifecycleEvent(t, observer.records, "replan_triggered")
	assertLifecycleEvent(t, observer.records, "chain_completed")
	for _, record := range observer.records {
		if record.TraceID == "" || record.ChainID == "" || record.SessionID == "" {
			t.Fatalf("expected trace/session/chain identifiers on lifecycle record, got %#v", record)
		}
	}
}

func TestOrchestratorApprovalPauseAndResumeEmitLifecycleCallbacks(t *testing.T) {
	ctx := context.Background()
	observer := &lifecycleObserver{}
	unregister := aiobs.RegisterObserver(observer)
	defer unregister()

	executions := airuntime.NewExecutionStore(nil, "")
	checkpoints := airuntime.NewCheckpointStore(nil, "")
	orchestrator := &Orchestrator{
		executions:  executions,
		checkpoints: checkpoints,
		converter:   airuntime.NewSSEConverter(),
	}
	state := airuntime.ExecutionState{
		TraceID:      "trace-2",
		SessionID:    "sess-2",
		PlanID:       "plan-2",
		TurnID:       "turn-2",
		Scene:        "deployment:hosts",
		Status:       airuntime.ExecutionStatusRunning,
		Phase:        string(airuntime.PhaseExecuting),
		CheckpointID: "checkpoint-2",
		Steps:        map[string]airuntime.StepState{},
	}

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(&adk.AgentEvent{
		Action: &adk.AgentAction{
			Interrupted: &adk.InterruptInfo{
				InterruptContexts: []*adk.InterruptCtx{{
					ID: "step-approval",
					Info: map[string]any{
						"step_id":           "step-approval",
						"tool_name":         "host_delete",
						"tool_display_name": "删除主机",
						"mode":              "mutating",
						"risk_level":        "high",
						"summary":           "删除主机前需要审批",
						"params":            map[string]any{"id": 7},
					},
				}},
			},
		},
	})
	gen.Close()

	var pauseEvents []airuntime.StreamEvent
	resumeState, err := orchestrator.streamExecution(ctx, iter, &state, func(event airuntime.StreamEvent) bool {
		pauseEvents = append(pauseEvents, event)
		return true
	})
	if err != nil {
		t.Fatalf("streamExecution returned error: %v", err)
	}
	if resumeState == nil || !resumeState.Interrupted {
		t.Fatalf("expected interrupted resume result, got %#v", resumeState)
	}
	assertEventTypePresent(t, pauseEvents, airuntime.EventChainPaused)
	assertLifecycleEvent(t, observer.records, "chain_paused")
	assertLifecycleEvent(t, observer.records, "approval_pending")

	state, ok, err := executions.Load(ctx, "sess-2", "plan-2")
	if err != nil || !ok {
		t.Fatalf("expected persisted execution state, got ok=%v err=%v", ok, err)
	}
	if state.PendingApproval == nil {
		t.Fatalf("expected pending approval to be stored")
	}

	var resumeEvents []airuntime.StreamEvent
	result, err := orchestrator.resume(ctx, airuntime.ResumeRequest{
		SessionID: "sess-2",
		PlanID:    "plan-2",
		StepID:    "step-approval",
		Approved:  true,
		Reason:    "looks safe",
	}, func(event airuntime.StreamEvent) bool {
		resumeEvents = append(resumeEvents, event)
		return true
	})
	if err != nil {
		t.Fatalf("resume returned error: %v", err)
	}
	if result == nil || result.Status != string(airuntime.ExecutionStatusCompleted) {
		t.Fatalf("expected completed resume result, got %#v", result)
	}
	assertEventTypePresent(t, resumeEvents, airuntime.EventChainMeta)
	assertEventTypePresent(t, resumeEvents, airuntime.EventChainResumed)
	assertEventTypePresent(t, resumeEvents, airuntime.EventDone)
	assertLifecycleEvent(t, observer.records, "approval_resolved")
	assertLifecycleEvent(t, observer.records, "chain_resumed")
}

func assistantEvent(agentName, content string) *adk.AgentEvent {
	return &adk.AgentEvent{
		AgentName: agentName,
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				Message: schema.AssistantMessage(content, nil),
				Role:    schema.Assistant,
			},
		},
	}
}

func toolEvent(toolName, content string) *adk.AgentEvent {
	return &adk.AgentEvent{
		AgentName: "executor",
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				Message:  schema.ToolMessage(content, "call-1", schema.WithToolName(toolName)),
				Role:     schema.Tool,
				ToolName: toolName,
			},
		},
	}
}

func assertEventTypePresent(t *testing.T, events []airuntime.StreamEvent, want airuntime.EventType) {
	t.Helper()
	for _, event := range events {
		if event.Type == want {
			return
		}
	}
	t.Fatalf("expected stream event %q in %#v", want, events)
}

func assertEventTypeAbsent(t *testing.T, events []airuntime.StreamEvent, forbidden airuntime.EventType) {
	t.Helper()
	for _, event := range events {
		if event.Type == forbidden {
			t.Fatalf("did not expect stream event %q in %#v", forbidden, events)
		}
	}
}

func assertLifecycleEvent(t *testing.T, records []aiobs.ThoughtChainLifecycleRecord, want string) {
	t.Helper()
	for _, record := range records {
		if record.Event == want {
			return
		}
	}
	t.Fatalf("expected lifecycle event %q in %#v", want, records)
}

// TestOrchestratorApprovalInterruptIncludesArgumentsAndResumeIdentity 验证审批中断事件
// 被转成的 PendingApproval / SSE approval payload 包含完整的恢复身份和参数信息。
func TestOrchestratorApprovalInterruptIncludesArgumentsAndResumeIdentity(t *testing.T) {
	ctx := context.Background()
	executions := airuntime.NewExecutionStore(nil, "")
	checkpoints := airuntime.NewCheckpointStore(nil, "")
	orchestrator := &Orchestrator{
		executions:  executions,
		checkpoints: checkpoints,
		converter:   airuntime.NewSSEConverter(),
	}
	state := airuntime.ExecutionState{
		TraceID:      "trace-approval-1",
		SessionID:    "sess-approval-1",
		PlanID:       "plan-approval-1",
		TurnID:       "turn-approval-1",
		Scene:        "deployment:hosts",
		Status:       airuntime.ExecutionStatusRunning,
		Phase:        string(airuntime.PhaseExecuting),
		CheckpointID: "checkpoint-approval-1",
		Steps:        map[string]airuntime.StepState{},
	}

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	// 模拟带有完整参数的审批中断事件
	gen.Send(&adk.AgentEvent{
		Action: &adk.AgentAction{
			Interrupted: &adk.InterruptInfo{
				InterruptContexts: []*adk.InterruptCtx{{
					ID: "step-delete-host",
					Info: airuntime.ApprovalInterruptInfo{
						PlanID:          "plan-approval-1",
						StepID:          "step-delete-host",
						CheckpointID:    "checkpoint-approval-1",
						Target:          "step-delete-host",
						ToolName:        "host_delete",
						ToolDisplayName: "删除主机",
						Mode:            "mutating",
						RiskLevel:       "high",
						Summary:         "删除主机前需要审批",
						Params:          map[string]any{"id": 7, "force": false},
						ArgumentsInJSON: `{"id":7,"force":false}`,
					},
				}},
			},
		},
	})
	gen.Close()

	var pauseEvents []airuntime.StreamEvent
	_, err := orchestrator.streamExecution(ctx, iter, &state, func(event airuntime.StreamEvent) bool {
		pauseEvents = append(pauseEvents, event)
		return true
	})
	if err != nil {
		t.Fatalf("streamExecution returned error: %v", err)
	}

	// 验证 ExecutionState 中的 PendingApproval 包含完整字段
	state, ok, err := executions.Load(ctx, "sess-approval-1", "plan-approval-1")
	if err != nil || !ok {
		t.Fatalf("expected persisted execution state, got ok=%v err=%v", ok, err)
	}
	if state.PendingApproval == nil {
		t.Fatalf("expected pending approval to be stored")
	}

	// 验证关键恢复身份字段
	pending := state.PendingApproval
	if pending.CheckpointID != "checkpoint-approval-1" {
		t.Errorf("expected checkpoint_id='checkpoint-approval-1', got %q", pending.CheckpointID)
	}
	if pending.Target != "step-delete-host" {
		t.Errorf("expected target='step-delete-host', got %q", pending.Target)
	}
	if pending.ArgumentsInJSON != `{"id":7,"force":false}` {
		t.Errorf("expected arguments_json='{\"id\":7,\"force\":false}', got %q", pending.ArgumentsInJSON)
	}
	if pending.ToolDisplayName != "删除主机" {
		t.Errorf("expected tool_display_name='删除主机', got %q", pending.ToolDisplayName)
	}

	// 验证 SSE approval payload 包含完整字段
	var approvalNode *airuntime.ChainNodeInfo
	for _, event := range pauseEvents {
		if event.Type == airuntime.EventChainNodeOpen {
			var node airuntime.ChainNodeInfo
			data, _ := json.Marshal(event.Data)
			if err := json.Unmarshal(data, &node); err == nil {
				if node.Kind == airuntime.ChainNodeApproval {
					approvalNode = &node
					break
				}
			}
		}
	}
	if approvalNode == nil {
		t.Fatalf("expected approval chain node to be emitted")
	}

	// 验证 SSE approval payload 包含 canonical 字段
	approval := approvalNode.Approval
	if approval == nil {
		t.Fatalf("expected approval payload in chain node")
	}
	if approval["checkpoint_id"] != "checkpoint-approval-1" {
		t.Errorf("expected approval.checkpoint_id='checkpoint-approval-1', got %v", approval["checkpoint_id"])
	}
	if approval["target"] != "step-delete-host" {
		t.Errorf("expected approval.target='step-delete-host', got %v", approval["target"])
	}
	if approval["arguments_json"] != `{"id":7,"force":false}` {
		t.Errorf("expected approval.arguments_json='{\"id\":7,\"force\":false}', got %v", approval["arguments_json"])
	}
}
