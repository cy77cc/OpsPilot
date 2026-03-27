package todo

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

func TestOpsTODO_JSONContract(t *testing.T) {
	todo := OpsTODO{
		Content:    "inspect cluster",
		ActiveForm: "inspect cluster",
		Status:     "pending",
	}

	raw, err := json.Marshal(todo)
	if err != nil {
		t.Fatalf("marshal todo: %v", err)
	}
	if got, want := string(raw), `{"content":"inspect cluster","active_form":"inspect cluster","status":"pending"}`; got != want {
		t.Fatalf("unexpected todo json, got %s want %s", got, want)
	}

	var decoded OpsTODO
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal todo: %v", err)
	}
	if decoded != todo {
		t.Fatalf("unexpected decoded todo: %#v", decoded)
	}
}

func TestWriteOpsTodosMiddleware_ReturnsFormattedSummary(t *testing.T) {
	mw, err := NewWriteOpsTodosMiddleware()
	if err != nil {
		t.Fatalf("new middleware: %v", err)
	}

	ctx := context.Background()
	runtimeCtx := &adk.ChatModelAgentContext{}
	_, nextCtx, err := mw.BeforeAgent(ctx, runtimeCtx)
	if err != nil {
		t.Fatalf("before agent: %v", err)
	}
	if len(nextCtx.Tools) != 1 {
		t.Fatalf("expected one tool to be appended, got %d", len(nextCtx.Tools))
	}

	invokable, ok := nextCtx.Tools[0].(tool.InvokableTool)
	if !ok {
		t.Fatalf("expected invokable tool, got %T", nextCtx.Tools[0])
	}

	raw, err := json.Marshal([]OpsTODO{{Content: "inspect cluster", ActiveForm: "inspect cluster", Status: "pending"}})
	if err != nil {
		t.Fatalf("marshal todos: %v", err)
	}
	got, err := invokable.InvokableRun(ctx, `{"todos":`+string(raw)+`}`)
	if err != nil {
		t.Fatalf("invokable run: %v", err)
	}
	if want := `Updated ops todo list to [{"content":"inspect cluster","active_form":"inspect cluster","status":"pending"}]`; got != want {
		t.Fatalf("unexpected summary, got %q want %q", got, want)
	}
}

func TestWriteOpsTodosMiddleware_WritesSnapshotIntoRunningSession(t *testing.T) {
	mw, err := NewWriteOpsTodosMiddleware()
	if err != nil {
		t.Fatalf("new middleware: %v", err)
	}

	fakeModel := &sessionAwareModel{t: t}
	agent, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:          "ops-writer",
		Description:   "test agent",
		Model:         fakeModel,
		Middlewares:   []adk.AgentMiddleware{},
		Handlers:      []adk.ChatModelAgentMiddleware{mw},
		MaxIterations: 2,
	})
	if err != nil {
		t.Fatalf("new chat model agent: %v", err)
	}

	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{Agent: agent})
	iter := runner.Run(context.Background(), []adk.Message{schema.UserMessage("start")})
	for {
		if _, ok := iter.Next(); !ok {
			break
		}
	}

	if fakeModel.snapshot == nil {
		t.Fatal("expected session snapshot to be captured")
	}
	if got, ok := fakeModel.snapshot[SessionKeyOpsTodos].([]OpsTODO); !ok || len(got) != 1 || got[0].Content != "inspect cluster" {
		t.Fatalf("unexpected session snapshot: %#v", fakeModel.snapshot)
	}
}

type sessionAwareModel struct {
	t        *testing.T
	call     int
	snapshot map[string]any
}

func (m *sessionAwareModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func (m *sessionAwareModel) Generate(ctx context.Context, input []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	m.call++
	if m.call == 1 {
		return schema.AssistantMessage("", []schema.ToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: schema.FunctionCall{
					Name:      "write_ops_todos",
					Arguments: `{"todos":[{"content":"inspect cluster","active_form":"inspect cluster","status":"pending"}]}`,
				},
			},
		}), nil
	}
	m.snapshot = adk.GetSessionValues(ctx)
	return schema.AssistantMessage("done", nil), nil
}

func (m *sessionAwareModel) Stream(ctx context.Context, input []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, err := m.Generate(ctx, input)
	if err != nil {
		return nil, err
	}
	return schema.StreamReaderFromArray([]*schema.Message{msg}), nil
}
