package orchestrator

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// DeterministicTransferAgent performs one explicit transfer and then stays silent
// once downstream agents have produced assistant output.
type DeterministicTransferAgent struct {
	name        string
	description string
	targetAgent string
}

func NewDeterministicTransferAgent(name, description, targetAgent string) *DeterministicTransferAgent {
	return &DeterministicTransferAgent{
		name:        strings.TrimSpace(name),
		description: strings.TrimSpace(description),
		targetAgent: strings.TrimSpace(targetAgent),
	}
}

func (a *DeterministicTransferAgent) Name(_ context.Context) string {
	if a == nil || strings.TrimSpace(a.name) == "" {
		return "supervisor"
	}
	return a.name
}

func (a *DeterministicTransferAgent) Description(_ context.Context) string {
	return a.description
}

func (a *DeterministicTransferAgent) Run(ctx context.Context, input *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		defer gen.Close()
		if a == nil || strings.TrimSpace(a.targetAgent) == "" {
			return
		}
		if hasDownstreamAssistantOutput(input) {
			return
		}

		assistantMessage, toolMessage := adk.GenTransferMessages(ctx, a.targetAgent)

		assistantEvent := adk.EventFromMessage(assistantMessage, nil, schema.Assistant, "")
		assistantEvent.AgentName = a.Name(ctx)
		gen.Send(assistantEvent)

		toolEvent := adk.EventFromMessage(toolMessage, nil, schema.Tool, toolMessage.ToolName)
		toolEvent.AgentName = a.Name(ctx)
		toolEvent.Action = adk.NewTransferToAgentAction(a.targetAgent)
		gen.Send(toolEvent)
	}()
	return iter
}

func (a *DeterministicTransferAgent) Resume(_ context.Context, _ *adk.ResumeInfo, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Close()
	return iter
}

// hasDownstreamAssistantOutput checks if downstream agent(s) have produced output.
// Note: flowAgent rewrites other agents' assistant messages as user messages for context,
// so we must check for rewritten context messages (starting with "For context:")
// or messages from any agent other than the original user input.
func hasDownstreamAssistantOutput(input *adk.AgentInput) bool {
	if input == nil {
		return false
	}
	for _, message := range input.Messages {
		if message == nil {
			continue
		}
		// Check for rewritten downstream output: flowAgent converts other agents'
		// assistant messages to user messages starting with "For context:"
		if message.Role == schema.User {
			content := strings.TrimSpace(message.Content)
			if strings.HasPrefix(content, "For context:") {
				return true
			}
		}
		// Also check for original assistant output (if not rewritten yet)
		if message.Role == schema.Assistant {
			if strings.TrimSpace(message.Content) == "" && len(message.ToolCalls) == 0 {
				continue
			}
			if len(message.ToolCalls) > 0 {
				continue
			}
			return true
		}
	}
	return false
}
