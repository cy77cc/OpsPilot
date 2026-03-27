package todo

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"
)

func TestWriteOpsTodosMiddleware_StoresSnapshotInSession(t *testing.T) {
	mw, err := NewWriteOpsTodosMiddleware()
	if err != nil {
		t.Fatalf("new middleware: %v", err)
	}
	ctx := context.Background()
	runCtx := &adk.ChatModelAgentContext{}
	_, nextCtx, err := mw.BeforeAgent(ctx, runCtx)
	if err != nil {
		t.Fatalf("before agent: %v", err)
	}
	if len(nextCtx.Tools) != 1 {
		t.Fatalf("expected write_ops_todos tool to be injected, got %d tools", len(nextCtx.Tools))
	}
}
