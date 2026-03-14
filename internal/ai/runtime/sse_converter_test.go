package runtime

import "testing"

func TestSSEConverterApprovalRequiredIncludesCheckpointIdentity(t *testing.T) {
	converter := NewSSEConverter()
	events := converter.OnApprovalRequired(&PendingApproval{
		ID:       "approval-1",
		PlanID:   "plan-1",
		StepID:   "step-1",
		ToolName: "scale_deployment",
		Summary:  "scale nginx",
	}, "cp-1")

	if len(events) != 2 {
		t.Fatalf("event count = %d", len(events))
	}
	if got := events[1].Data["checkpoint_id"]; got != "cp-1" {
		t.Fatalf("checkpoint_id = %#v", got)
	}
}

func TestSSEConverterPlannerStartEmitsTurnLifecycle(t *testing.T) {
	converter := NewSSEConverter()
	events := converter.OnPlannerStart("sess-1", "plan-1", "turn-1")
	if len(events) != 3 {
		t.Fatalf("event count = %d", len(events))
	}
	if events[0].Type != EventTurnStarted {
		t.Fatalf("first event type = %s, want %s", events[0].Type, EventTurnStarted)
	}
	if string(events[1].Type) != "phase_started" {
		t.Fatalf("second event type = %s, want phase_started", events[1].Type)
	}
	if got := events[1].Data["phase"]; got != "planning" {
		t.Fatalf("phase = %#v, want planning", got)
	}
	if events[2].Type != EventTurnState {
		t.Fatalf("third event type = %s, want %s", events[2].Type, EventTurnState)
	}
	if got := events[2].Data["status"]; got != "running" {
		t.Fatalf("status = %#v, want running", got)
	}
}

func TestSSEConverterExecuteCompleteEmitsPhaseCompleteBeforeTurnCompletion(t *testing.T) {
	converter := NewSSEConverter()
	events := converter.OnExecuteComplete()
	if len(events) != 2 {
		t.Fatalf("event count = %d", len(events))
	}
	if string(events[0].Type) != "phase_complete" {
		t.Fatalf("first event type = %s, want phase_complete", events[0].Type)
	}
	if got := events[0].Data["phase"]; got != "executing" {
		t.Fatalf("phase = %#v, want executing", got)
	}
	if events[1].Type != EventTurnState {
		t.Fatalf("second event type = %s, want %s", events[1].Type, EventTurnState)
	}
	if got := events[1].Data["status"]; got != "completed" {
		t.Fatalf("status = %#v, want completed", got)
	}
}

func TestSSEConverterNativeThoughtChainEvents(t *testing.T) {
	converter := NewSSEConverter()

	if event := converter.OnChainStarted("turn-1"); event.Type != EventChainStarted {
		t.Fatalf("chain started type = %s", event.Type)
	}

	open := converter.OnChainNodeOpen(ChainNodeInfo{
		TurnID:  "turn-1",
		NodeID:  "plan-1",
		Kind:    ChainNodePlan,
		Title:   "正在整理执行计划",
		Status:  "loading",
		Summary: "准备检查集群状态",
	})
	if open.Type != EventChainNodeOpen {
		t.Fatalf("chain node open type = %s", open.Type)
	}
	if got := open.Data["node_id"]; got != "plan-1" {
		t.Fatalf("node_id = %#v, want plan-1", got)
	}
	if got := open.Data["kind"]; got != "plan" {
		t.Fatalf("kind = %#v, want plan", got)
	}

	patch := converter.OnChainNodePatch(ChainNodeInfo{
		TurnID:  "turn-1",
		NodeID:  "plan-1",
		Summary: "已整理出 2 个执行步骤",
	})
	if patch.Type != EventChainNodePatch {
		t.Fatalf("chain node patch type = %s", patch.Type)
	}
	if got := patch.Data["summary"]; got != "已整理出 2 个执行步骤" {
		t.Fatalf("summary = %#v", got)
	}

	if event := converter.OnChainCollapsed("turn-1"); event.Type != EventChainCollapsed {
		t.Fatalf("chain collapsed type = %s", event.Type)
	}
	if event := converter.OnFinalAnswerStarted("turn-1"); event.Type != EventFinalAnswerStart {
		t.Fatalf("final answer start type = %s", event.Type)
	}
	if event := converter.OnFinalAnswerDelta("turn-1", "nginx 当前状态正常"); event.Type != EventFinalAnswerDelta {
		t.Fatalf("final answer delta type = %s", event.Type)
	}
	if event := converter.OnFinalAnswerDone("turn-1"); event.Type != EventFinalAnswerDone {
		t.Fatalf("final answer done type = %s", event.Type)
	}
}
