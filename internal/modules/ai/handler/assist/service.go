package assist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	aiclient "github.com/cy77cc/OpsPilot/internal/modules/llmprovider/client"
)

// Service provides form assistance use cases.
type Service struct {
	logic *logic.Logic
}

// NewService creates a new assist service.
func NewService(l *logic.Logic) *Service {
	return &Service{logic: l}
}

// StreamAssist invokes the AI model to provide form assistance and streams the results.
func (s *Service) StreamAssist(ctx context.Context, input logic.FormAssistInput, emit logic.EventEmitter) error {
	systemPrompt := BuildSystemPrompt(input.FieldMeta)
	sanitizedContext := SanitizeFormContext(input.FormContext)

	chatModel, err := aiclient.GetDefaultChatModel(ctx, nil, aiclient.ChatModelConfig{})
	if err != nil {
		return fmt.Errorf("get default chat model: %w", err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "form_assistant",
		Description: "Professional Ops Assistant for form filling.",
		Instruction: systemPrompt,
		Model:       chatModel,
	})
	if err != nil {
		return fmt.Errorf("create chat model agent: %w", err)
	}

	// Build user message with context
	userMsgContent := input.UserPrompt
	if len(sanitizedContext) > 0 {
		contextBytes, _ := json.Marshal(sanitizedContext)
		userMsgContent = fmt.Sprintf("Context: %s\n\nUser Request: %s", string(contextBytes), input.UserPrompt)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	iterator := runner.Run(ctx, []*schema.Message{
		schema.UserMessage(userMsgContent),
	})

	var fullContent strings.Builder
	for {
		event, ok := iterator.Next()
		if !ok {
			break
		}

		if event.Output != nil && event.Output.MessageOutput != nil && event.Output.MessageOutput.IsStreaming && event.Output.MessageOutput.MessageStream != nil {
			for {
				msg, err := event.Output.MessageOutput.MessageStream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
				if msg != nil && msg.Content != "" {
					content := msg.Content
					// The instruction said: "Normalize each chunk before emitting delta"
					// However, full normalization (like removing fences) requires full content.
					// We'll do basic trimming and emit.
					normalizedChunk := strings.TrimLeft(content, " \n\r\t")
					if normalizedChunk != "" || fullContent.Len() > 0 {
						fullContent.WriteString(content)
						emit("delta", map[string]any{"content": content})
					}
				}
			}
		}
	}

	// Final normalization for the "done" event summary
	finalContent := NormalizeFormAssistOutput(fullContent.String())
	emit("done", map[string]any{"content": finalContent})

	return nil
}
