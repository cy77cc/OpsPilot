package command

import (
	"context"
	"errors"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	runtimecontext "github.com/cy77cc/OpsPilot/internal/modules/ai/runtime/context"
)

type stubChatUseCase struct {
	validateArgs []string
	chatInput    logic.ChatInput
	emitCalled   bool
	err          error
}

func (s *stubChatUseCase) ValidateReplayCursor(_ context.Context, sessionID, clientRequestID, lastEventID string) error {
	s.validateArgs = []string{sessionID, clientRequestID, lastEventID}
	return nil
}

func (s *stubChatUseCase) Chat(_ context.Context, input logic.ChatInput, emit logic.EventEmitter) error {
	s.chatInput = input
	s.emitCalled = emit != nil
	if emit != nil {
		emit("meta", map[string]any{"session_id": input.SessionID})
	}
	return s.err
}

func TestChatHandlerDelegatesTransportFieldsToUseCase(t *testing.T) {
	stub := &stubChatUseCase{}
	h := NewChatCommandHandler(stub)

	req := &ChatRequest{
		SessionID:       "sess-1",
		ClientRequestID: "req-1",
		LastEventID:     "evt-1",
		Message:         "hello",
		Scene:           "ops",
		Context:         map[string]any{"team": "platform"},
		UserID:          7,
	}

	if err := h.Handle(context.Background(), req, func(string, any) {}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := stub.validateArgs; len(got) != 3 || got[0] != "sess-1" || got[1] != "req-1" || got[2] != "evt-1" {
		t.Fatalf("unexpected replay validation args: %#v", got)
	}
	if stub.chatInput.SessionID != "sess-1" || stub.chatInput.LastEventID != "evt-1" || stub.chatInput.UserID != 7 {
		t.Fatalf("unexpected chat input: %#v", stub.chatInput)
	}
	if !stub.emitCalled {
		t.Fatal("expected emit callback to be passed through")
	}
}

func TestChatHandlerSurfacesUseCaseErrors(t *testing.T) {
	stub := &stubChatUseCase{err: errors.New("boom")}
	h := NewChatCommandHandler(stub)

	err := h.Handle(context.Background(), &ChatRequest{Message: "hello"}, nil)
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected error to be returned, got %v", err)
	}
}

func TestChatHandlerNormalizesRuntimeContextMessages(t *testing.T) {
	stub := &stubChatUseCase{}
	h := NewChatCommandHandler(stub)

	req := &ChatRequest{
		Message: "hello",
		Context: map[string]any{
			"messages": []any{
				map[string]any{"role": "system", "content": "pinned-1", "pinned": true},
				map[string]any{"role": "user", "content": "h1"},
				map[string]any{"role": "assistant", "content": "h2"},
				map[string]any{"role": "user", "content": "recent-1"},
				map[string]any{"role": "assistant", "content": "recent-2"},
			},
		},
	}

	if err := h.Handle(context.Background(), req, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctxMap := stub.chatInput.Context
	if _, ok := ctxMap["context_budget"].(runtimecontext.Budget); !ok {
		t.Fatalf("expected context budget to be attached, got %#v", ctxMap["context_budget"])
	}

	messages, ok := ctxMap["messages"].([]runtimecontext.Message)
	if !ok {
		t.Fatalf("expected normalized runtime messages, got %#v", ctxMap["messages"])
	}
	if len(messages) != 5 {
		t.Fatalf("expected 5 normalized messages, got %d", len(messages))
	}
	if messages[0].Content != "pinned-1" {
		t.Fatalf("expected pinned message preserved, got %#v", messages)
	}
	if messages[4].Content != "recent-2" {
		t.Fatalf("expected tail message preserved, got %#v", messages)
	}
}
