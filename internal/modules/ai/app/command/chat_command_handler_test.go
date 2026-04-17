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
		TraceID:         "trace-1",
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
	if stub.chatInput.TraceID != "trace-1" {
		t.Fatalf("expected trace id to be propagated, got %#v", stub.chatInput)
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

func TestChatHandlerUsesExplicitContextBudget(t *testing.T) {
	stub := &stubChatUseCase{}
	h := NewChatCommandHandler(stub)

	req := &ChatRequest{
		Message: "hello",
	}

	if err := h.Handle(context.Background(), req, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stub.chatInput.Budget != runtimecontext.DefaultBudget {
		t.Fatalf("expected explicit budget to be passed through, got %#v", stub.chatInput.Budget)
	}
	if stub.chatInput.Context != nil {
		t.Fatalf("expected empty context to remain nil, got %#v", stub.chatInput.Context)
	}
	if stub.chatInput.TraceID == "" {
		t.Fatal("expected trace id to be generated")
	}
}
