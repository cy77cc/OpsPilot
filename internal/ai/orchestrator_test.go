package ai

import (
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
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
