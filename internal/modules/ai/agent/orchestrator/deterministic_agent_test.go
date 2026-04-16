package orchestrator

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"
	adksupervisor "github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/schema"
)

func TestDeterministicTransferAgent_TransfersOnFirstRun(t *testing.T) {
	agent := NewDeterministicTransferAgent("supervisor", "routes work", "monitor")
	iter := agent.Run(context.Background(), &adk.AgentInput{
		Messages: []*schema.Message{schema.UserMessage("inspect p95")},
	})

	first, ok := iter.Next()
	if !ok || first == nil || first.Action != nil {
		t.Fatalf("expected assistant transfer prelude event, got %#v", first)
	}

	second, ok := iter.Next()
	if !ok || second == nil || second.Action == nil || second.Action.TransferToAgent == nil {
		t.Fatalf("expected transfer action event, got %#v", second)
	}
	if second.Action.TransferToAgent.DestAgentName != "monitor" {
		t.Fatalf("expected transfer target monitor, got %#v", second.Action.TransferToAgent)
	}
}

func TestDeterministicTransferAgent_SilentAfterAssistantOutput(t *testing.T) {
	agent := NewDeterministicTransferAgent("supervisor", "routes work", "monitor")
	iter := agent.Run(context.Background(), &adk.AgentInput{
		Messages: []*schema.Message{
			schema.UserMessage("inspect p95"),
			schema.AssistantMessage("monitor summary", nil),
		},
	})

	if event, ok := iter.Next(); ok {
		t.Fatalf("expected no follow-up events after assistant output, got %#v", event)
	}
}

func TestDeterministicTransferAgent_WithSupervisorRoutesToSubAgent(t *testing.T) {
	ctx := context.Background()
	root := NewDeterministicTransferAgent("supervisor", "routes work", "monitor")
	subAgent := &scriptedResumableAgent{name: "monitor", content: "monitor handled the request"}

	system, err := adksupervisor.New(ctx, &adksupervisor.Config{
		Supervisor: root,
		SubAgents:  []adk.Agent{subAgent},
	})
	if err != nil {
		t.Fatalf("build supervisor system: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: system})
	iter := runner.Run(ctx, []*schema.Message{schema.UserMessage("inspect p95")})

	var sawTransfer bool
	var sawSpecialistOutput bool
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Action != nil && event.Action.TransferToAgent != nil && event.Action.TransferToAgent.DestAgentName == "monitor" {
			sawTransfer = true
		}
		if event.AgentName == "monitor" && event.Output != nil && event.Output.MessageOutput != nil {
			message, err := event.Output.MessageOutput.GetMessage()
			if err == nil && message != nil && message.Content == "monitor handled the request" {
				sawSpecialistOutput = true
			}
		}
	}

	if !sawTransfer {
		t.Fatal("expected transfer to monitor specialist")
	}
	if !sawSpecialistOutput {
		t.Fatal("expected monitor specialist output to be emitted")
	}
}

type scriptedResumableAgent struct {
	name    string
	content string
}

func (s *scriptedResumableAgent) Name(_ context.Context) string { return s.name }

func (s *scriptedResumableAgent) Description(_ context.Context) string { return s.name }

func (s *scriptedResumableAgent) Run(_ context.Context, _ *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	event := adk.EventFromMessage(schema.AssistantMessage(s.content, nil), nil, schema.Assistant, "")
	event.AgentName = s.name
	gen.Send(event)
	gen.Close()
	return iter
}

func (s *scriptedResumableAgent) Resume(_ context.Context, _ *adk.ResumeInfo, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	return s.Run(context.Background(), nil)
}
