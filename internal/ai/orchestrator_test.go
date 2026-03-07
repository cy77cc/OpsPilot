package ai

import (
	"context"
	"strings"
	"testing"
	"time"

	adkcore "github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	aitools "github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/service/ai/logic"
)

type fakeSessionStore struct {
	current *logic.AISession
	appends []map[string]any
}

func (f *fakeSessionStore) CurrentSession(_ uint64, _ string) (*logic.AISession, bool) {
	if f.current == nil {
		return nil, false
	}
	return f.current, true
}

func (f *fakeSessionStore) AppendMessage(_ uint64, scene, sessionID string, message map[string]any) (*logic.AISession, error) {
	f.appends = append(f.appends, message)
	if f.current == nil {
		f.current = &logic.AISession{
			ID:        sessionID,
			Scene:     scene,
			Title:     logic.DefaultAISessionTitle,
			Messages:  []map[string]any{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}
	f.current.Messages = append(f.current.Messages, message)
	return f.current, nil
}

type fakeRunner struct {
	metas         []aitools.ToolMeta
	queryMessage  string
	queryIter     *adkcore.AsyncIterator[*adkcore.AgentEvent]
	resumeIter    *adkcore.AsyncIterator[*adkcore.AgentEvent]
	resumeErr     error
	generateReply string
}

func (f *fakeRunner) ToolMetas() []aitools.ToolMeta { return f.metas }

func (f *fakeRunner) Query(_ context.Context, _ string, message string, _ ...adkcore.AgentRunOption) *adkcore.AsyncIterator[*adkcore.AgentEvent] {
	f.queryMessage = message
	return f.queryIter
}

func (f *fakeRunner) Resume(_ context.Context, _ string, _ map[string]any, _ ...adkcore.AgentRunOption) (*adkcore.AsyncIterator[*adkcore.AgentEvent], error) {
	return f.resumeIter, f.resumeErr
}

func (f *fakeRunner) Generate(_ context.Context, _ []*schema.Message) (*schema.Message, error) {
	return schema.AssistantMessage(f.generateReply, nil), nil
}

func eventIterator(events ...*adkcore.AgentEvent) *adkcore.AsyncIterator[*adkcore.AgentEvent] {
	iter, gen := adkcore.NewAsyncIteratorPair[*adkcore.AgentEvent]()
	go func() {
		defer gen.Close()
		for _, event := range events {
			gen.Send(event)
		}
	}()
	return iter
}

func TestOrchestratorChatStreamEmitsMetaAndDone(t *testing.T) {
	sessions := &fakeSessionStore{}
	runtime := logic.NewRuntimeStore(nil)
	runner := &fakeRunner{
		metas: []aitools.ToolMeta{{Name: "host_list_inventory"}},
		queryIter: eventIterator(&adkcore.AgentEvent{
			Output: &adkcore.AgentOutput{
				MessageOutput: &adkcore.MessageVariant{
					Message: schema.AssistantMessage("diagnosis complete", nil),
				},
			},
		}),
		generateReply: "检查建议|先做健康检查|0.8|降低风险",
	}
	control := NewControlPlane(nil, runtime, runner)
	orch := NewOrchestrator(runner, sessions, runtime, control)

	var events []string
	var donePayload map[string]any
	err := orch.ChatStream(context.Background(), ChatStreamRequest{
		UserID:  7,
		Message: "帮我查下磁盘",
		Context: map[string]any{"scene": "global"},
	}, func(event string, payload map[string]any) bool {
		events = append(events, event)
		if event == "done" {
			donePayload = payload
		}
		return true
	})
	if err != nil {
		t.Fatalf("chat stream failed: %v", err)
	}
	if len(events) < 2 || events[0] != "meta" {
		t.Fatalf("expected meta first, got %#v", events)
	}
	if events[len(events)-1] != "done" {
		t.Fatalf("expected done last, got %#v", events)
	}
	if len(sessions.appends) != 2 {
		t.Fatalf("expected 2 session appends, got %d", len(sessions.appends))
	}
	if !strings.Contains(runner.queryMessage, "帮我查下磁盘") {
		t.Fatalf("expected query message to contain original prompt, got %q", runner.queryMessage)
	}
	if donePayload == nil || donePayload["session"] == nil {
		t.Fatalf("expected done payload with session, got %#v", donePayload)
	}
}

func TestOrchestratorResumePayloadHandlesInterrupt(t *testing.T) {
	runner := &fakeRunner{
		resumeIter: eventIterator(&adkcore.AgentEvent{
			Action: &adkcore.AgentAction{
				Interrupted: &adkcore.InterruptInfo{
					Data: &aitools.ApprovalInfo{
						ToolName:        "host_batch_exec_apply",
						ArgumentsInJSON: "{\"host_ids\":[1]}",
						Risk:            aitools.ToolRiskHigh,
						Preview:         map[string]any{"target_count": 1},
					},
					InterruptContexts: []*adkcore.InterruptCtx{{ID: "call-1", IsRootCause: true}},
				},
			},
		}),
	}
	orch := NewOrchestrator(runner, &fakeSessionStore{}, logic.NewRuntimeStore(nil), NewControlPlane(nil, logic.NewRuntimeStore(nil), runner))

	payload, err := orch.ResumePayload(context.Background(), "sess-1", map[string]any{"call-1": &aitools.ApprovalResult{Approved: true}})
	if err != nil {
		t.Fatalf("resume payload failed: %v", err)
	}
	if payload["approval_required"] != true {
		t.Fatalf("expected approval interrupt payload, got %#v", payload)
	}
	if payload["sessionId"] != "sess-1" {
		t.Fatalf("unexpected session id: %#v", payload["sessionId"])
	}
}
