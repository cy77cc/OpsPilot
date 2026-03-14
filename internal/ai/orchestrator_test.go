package ai

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

func TestMergeTextProgress(t *testing.T) {
	tests := []struct {
		name     string
		previous string
		current  string
		wantText string
	}{
		{
			name:     "first chunk",
			current:  "你",
			wantText: "你",
		},
		{
			name:     "cumulative content",
			previous: "你",
			current:  "你好",
			wantText: "你好",
		},
		{
			name:     "unchanged content",
			previous: "你好",
			current:  "你好",
			wantText: "你好",
		},
		{
			name:     "delta append",
			previous: "你",
			current:  "好",
			wantText: "你好",
		},
		{
			name:     "json content should not be swallowed",
			previous: "结果：",
			current:  "{\"ok\":true}",
			wantText: "结果：{\"ok\":true}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeTextProgress(tt.previous, tt.current)
			if got != tt.wantText {
				t.Fatalf("mergeTextProgress(%q, %q) = %q, want %q", tt.previous, tt.current, got, tt.wantText)
			}
		})
	}
}

func TestEventTextContents(t *testing.T) {
	t.Run("non streaming message", func(t *testing.T) {
		event := &adk.AgentEvent{
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: false,
					Message:     schema.AssistantMessage("你好", nil),
				},
			},
		}

		got, err := eventTextContents(event)
		if err != nil {
			t.Fatalf("eventTextContents returned error: %v", err)
		}
		if len(got) != 1 || got[0] != "你好" {
			t.Fatalf("eventTextContents = %#v, want [\"你好\"]", got)
		}
	})

	t.Run("streaming message keeps chunk granularity", func(t *testing.T) {
		event := &adk.AgentEvent{
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: true,
					MessageStream: schema.StreamReaderFromArray([]*schema.Message{
						schema.AssistantMessage("你", nil),
						schema.AssistantMessage("你好", nil),
					}),
				},
			},
		}

		got, err := eventTextContents(event)
		if err != nil {
			t.Fatalf("eventTextContents returned error: %v", err)
		}
		if len(got) != 2 || got[0] != "你" || got[1] != "你好" {
			t.Fatalf("eventTextContents = %#v, want [\"你\", \"你好\"]", got)
		}
	})
}

func TestStreamExecutionEmitsPlanningPhaseWhenPlannerEventsArrive(t *testing.T) {
	t.Parallel()

	orchestrator := &Orchestrator{
		converter:  airuntime.NewSSEConverter(),
		executions: airuntime.NewExecutionStore(nil, ""),
	}
	state := newExecutionStateForTest()

	events := collectStreamEvents(orchestrator, &state, iteratorFromEvents(
		&adk.AgentEvent{
			AgentName: "planner",
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: false,
					Message:     schema.AssistantMessage("开始规划", nil),
				},
			},
		},
	))

	assertEventType(t, events, "phase_started")
	assertEventPayload(t, events, "phase_started", "phase", "planning")
}

func TestStreamExecutionParsesPlannerJSONIntoPlanGeneratedEvent(t *testing.T) {
	t.Parallel()

	orchestrator := &Orchestrator{
		converter:  airuntime.NewSSEConverter(),
		executions: airuntime.NewExecutionStore(nil, ""),
	}
	state := newExecutionStateForTest()

	events := collectStreamEvents(orchestrator, &state, iteratorFromEvents(
		&adk.AgentEvent{
			AgentName: "planner",
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: false,
					Message:     schema.AssistantMessage("```json\n[{\"id\":\"step-1\",\"content\":\"检查集群状态\",\"toolHint\":\"get_cluster_info\"},{\"id\":\"step-2\",\"content\":\"获取 deployment 列表\",\"toolHint\":\"list_deployments\"}]\n```", nil),
				},
			},
		},
	))

	assertEventType(t, events, "plan_generated")
	assertEventPayload(t, events, "plan_generated", "total", 2)
}

func TestStreamExecutionApprovalInterruptEmitsStepContextBeforeApprovalRequired(t *testing.T) {
	t.Parallel()

	orchestrator := &Orchestrator{
		converter:   airuntime.NewSSEConverter(),
		executions:  airuntime.NewExecutionStore(nil, ""),
		checkpoints: airuntime.NewCheckpointStore(nil, ""),
		summaries:   nil,
	}
	state := newExecutionStateForTest()

	events := collectStreamEvents(orchestrator, &state, iteratorFromEvents(
		&adk.AgentEvent{
			Action: &adk.AgentAction{
				Interrupted: &adk.InterruptInfo{
					InterruptContexts: []*adk.InterruptCtx{
						{
							ID:          "step-1",
							IsRootCause: true,
							Info: airuntime.ApprovalInterruptInfo{
								PlanID:          "plan-test",
								StepID:          "step-1",
								ToolName:        "scale_deployment",
								ToolDisplayName: "扩容 nginx",
								Mode:            "mutating",
								RiskLevel:       "high",
								Summary:         "该步骤会修改工作负载副本数",
								Params:          map[string]any{"replicas": 3},
							},
						},
					},
				},
			},
		},
	))

	assertEventType(t, events, "step_started")
	assertEventPayload(t, events, "approval_required", "tool_name", "scale_deployment")
}

func TestStreamExecutionEmitsReplanTriggeredWhenReplannerEventsArrive(t *testing.T) {
	t.Parallel()

	orchestrator := &Orchestrator{
		converter:  airuntime.NewSSEConverter(),
		executions: airuntime.NewExecutionStore(nil, ""),
	}
	state := newExecutionStateForTest()
	state.Phase = string(airuntime.PhaseExecuting)

	events := collectStreamEvents(orchestrator, &state, iteratorFromEvents(
		&adk.AgentEvent{
			AgentName: "replanner",
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: false,
					Message:     schema.AssistantMessage("需要重新规划后续执行步骤", nil),
				},
			},
		},
	))

	assertEventType(t, events, "replan_triggered")
	assertEventPayload(t, events, "replan_triggered", "reason", "需要重新规划后续执行步骤")
	assertEventPayload(t, events, "phase_started", "phase", "replanning")
}

func TestStreamExecutionEmitsToolLifecycleForToolOutputs(t *testing.T) {
	t.Parallel()

	orchestrator := &Orchestrator{
		converter:  airuntime.NewSSEConverter(),
		executions: airuntime.NewExecutionStore(nil, ""),
	}
	state := newExecutionStateForTest()
	state.Steps["step-1"] = airuntime.StepState{
		StepID:   "step-1",
		Title:    "检查集群状态",
		Status:   airuntime.StepPending,
		ToolName: "get_cluster_info",
	}

	events := collectStreamEvents(orchestrator, &state, iteratorFromEvents(
		&adk.AgentEvent{
			AgentName: "executor",
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: false,
					Role:        schema.Tool,
					ToolName:    "get_cluster_info",
					Message:     schema.ToolMessage("3 pods found", ""),
				},
			},
		},
	))

	assertEventType(t, events, "step_started")
	assertEventType(t, events, "tool_call")
	assertEventType(t, events, "tool_result")
	assertEventType(t, events, "step_complete")
	assertEventPayload(t, events, "tool_result", "tool_name", "get_cluster_info")
	assertEventPayload(t, events, "step_complete", "step_id", "step-1")
}

func TestStreamExecutionEmitsNativeThoughtChainAndFinalAnswerEvents(t *testing.T) {
	t.Parallel()

	orchestrator := &Orchestrator{
		converter:  airuntime.NewSSEConverter(),
		executions: airuntime.NewExecutionStore(nil, ""),
	}
	state := newExecutionStateForTest()

	events := collectStreamEvents(orchestrator, &state, iteratorFromEvents(
		&adk.AgentEvent{
			AgentName: "planner",
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: false,
					Message:     schema.AssistantMessage("```json\n[{\"id\":\"step-1\",\"content\":\"检查集群状态\",\"toolHint\":\"get_cluster_info\"},{\"id\":\"step-2\",\"content\":\"确认 deployment 副本数\",\"toolHint\":\"get_deployment\"}]\n```", nil),
				},
			},
		},
		&adk.AgentEvent{
			AgentName: "executor",
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: false,
					Message:     schema.AssistantMessage("集群状态正常", nil),
				},
			},
		},
	))

	assertEventType(t, events, "chain_started")
	assertEventType(t, events, "chain_node_open")
	assertEventPayload(t, events, "chain_node_open", "kind", "plan")
	assertEventType(t, events, "chain_collapsed")
	assertEventType(t, events, "final_answer_started")
	assertEventType(t, events, "final_answer_delta")
	assertEventPayload(t, events, "final_answer_delta", "chunk", "集群状态正常")

	collapsedIndex := eventIndex(t, events, "chain_collapsed")
	answerStartIndex := eventIndex(t, events, "final_answer_started")
	if collapsedIndex > answerStartIndex {
		t.Fatalf("chain_collapsed index %d should be before final_answer_started index %d", collapsedIndex, answerStartIndex)
	}
}

func TestResumeApprovedStreamEmitsStepCompleteBeforeExecutionCompletion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	executions := airuntime.NewExecutionStore(nil, "")
	orchestrator := &Orchestrator{
		converter:   airuntime.NewSSEConverter(),
		executions:  executions,
		checkpoints: airuntime.NewCheckpointStore(nil, ""),
	}

	state := newExecutionStateForTest()
	state.Status = airuntime.ExecutionStatusWaitingApproval
	state.InterruptTarget = "step-1"
	state.PendingApproval = &airuntime.PendingApproval{
		ID:       "approval-1",
		PlanID:   state.PlanID,
		StepID:   "step-1",
		Status:   "pending",
		Title:    "扩容 nginx",
		Summary:  "该步骤会修改工作负载副本数",
		ToolName: "scale_deployment",
		Params:   map[string]any{"replicas": 3},
	}
	state.Steps["step-1"] = airuntime.StepState{
		StepID:             "step-1",
		Title:              "扩容 nginx",
		Status:             airuntime.StepWaitingApproval,
		ToolName:           "scale_deployment",
		ToolArgs:           map[string]any{"replicas": 3},
		UserVisibleSummary: "该步骤会修改工作负载副本数",
	}
	if err := executions.Save(ctx, state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var events []airuntime.StreamEvent
	_, err := orchestrator.resume(ctx, airuntime.ResumeRequest{
		SessionID: state.SessionID,
		PlanID:    state.PlanID,
		StepID:    "step-1",
		Approved:  true,
	}, func(evt airuntime.StreamEvent) bool {
		events = append(events, evt)
		return true
	})
	if err != nil {
		t.Fatalf("resume() error = %v", err)
	}

	assertEventType(t, events, "step_complete")
	assertEventPayload(t, events, "step_complete", "step_id", "step-1")
}

func collectStreamEvents(orchestrator *Orchestrator, state *airuntime.ExecutionState, iter *adk.AsyncIterator[*adk.AgentEvent]) []airuntime.StreamEvent {
	var events []airuntime.StreamEvent
	_, _ = orchestrator.streamExecution(context.Background(), iter, state, func(evt airuntime.StreamEvent) bool {
		events = append(events, evt)
		return true
	})
	return events
}

func iteratorFromEvents(events ...*adk.AgentEvent) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, generator := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	for _, event := range events {
		generator.Send(event)
	}
	generator.Close()
	return iter
}

func newExecutionStateForTest() airuntime.ExecutionState {
	return airuntime.ExecutionState{
		SessionID: "session-test",
		PlanID:    "plan-test",
		TurnID:    "turn-test",
		Status:    airuntime.ExecutionStatusRunning,
		Phase:     "running",
		Steps:     map[string]airuntime.StepState{},
	}
}

func assertEventType(t *testing.T, events []airuntime.StreamEvent, want string) {
	t.Helper()
	for _, event := range events {
		if string(event.Type) == want {
			return
		}
	}
	t.Fatalf("event %q not found in %#v", want, events)
}

func assertEventPayload(t *testing.T, events []airuntime.StreamEvent, eventType string, key string, want any) {
	t.Helper()
	for _, event := range events {
		if string(event.Type) != eventType {
			continue
		}
		if got := event.Data[key]; got != want {
			t.Fatalf("%s payload %q = %#v, want %#v", eventType, key, got, want)
		}
		return
	}
	t.Fatalf("event %q not found in %#v", eventType, events)
}

func eventIndex(t *testing.T, events []airuntime.StreamEvent, want string) int {
	t.Helper()
	for index, event := range events {
		if string(event.Type) == want {
			return index
		}
	}
	t.Fatalf("event %q not found in %#v", want, events)
	return -1
}
