package runtimectx

import (
	"context"
	"testing"
	"time"
)

func TestWithContext_PreservesStructuredContextAndCustomData(t *testing.T) {
	base := NewContext()
	base.TraceID = "trace-1"
	base.Set("k", "v")

	ctx := WithContext(context.Background(), base)

	got := FromContext(ctx)
	if got == nil {
		t.Fatal("expected runtime context")
	}
	if got.TraceID != "trace-1" {
		t.Fatalf("expected trace id, got %q", got.TraceID)
	}
	if got.Get("k") != "v" {
		t.Fatalf("expected custom value, got %#v", got.Get("k"))
	}
}

func TestWithServicesAndAIMeta_ReuseSingleRuntimeContext(t *testing.T) {
	services := struct{ Name string }{Name: "svc"}
	ctx := WithServices(context.Background(), services)
	ctx = WithTraceID(ctx, "trace-2")
	ctx = WithAIMetadata(ctx, AIMetadata{
		SessionID: "sess-1",
		RunID:     "run-1",
		UserID:    7,
		Scene:     "cluster",
	})

	if got := Services(ctx); got != services {
		t.Fatalf("expected services in runtime context")
	}
	typed, ok := ServicesAs[struct{ Name string }](ctx)
	if !ok || typed.Name != "svc" {
		t.Fatalf("expected typed services accessor, got %#v, ok=%v", typed, ok)
	}
	if got := TraceID(ctx); got != "trace-2" {
		t.Fatalf("expected trace id, got %q", got)
	}
	meta := AIMetadataFrom(ctx)
	if meta.SessionID != "sess-1" || meta.RunID != "run-1" || meta.UserID != 7 || meta.Scene != "cluster" {
		t.Fatalf("unexpected ai metadata: %#v", meta)
	}
}

func TestWithValue_StoresProjectScopedCustomMetadata(t *testing.T) {
	key := struct{ Name string }{Name: "audit"}
	value := map[string]any{"command_id": "cmd-1"}

	ctx := WithValue(context.Background(), key, value)

	got, ok := Value(ctx, key).(map[string]any)
	if !ok {
		t.Fatalf("expected custom metadata map")
	}
	if got["command_id"] != "cmd-1" {
		t.Fatalf("unexpected custom metadata: %#v", got)
	}
}

func TestDetach_PreservesRuntimeContextButRemovesCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	parent = WithTraceID(parent, "trace-detached")
	parent = WithServices(parent, struct{ Name string }{Name: "svc"})
	parent = WithAIMetadata(parent, AIMetadata{SessionID: "sess-detached"})

	detached := Detach(parent)
	cancel()

	select {
	case <-detached.Done():
		t.Fatal("detached context should not be canceled with parent")
	case <-time.After(10 * time.Millisecond):
	}

	if TraceID(detached) != "trace-detached" {
		t.Fatalf("expected detached trace id")
	}
	if meta := AIMetadataFrom(detached); meta.SessionID != "sess-detached" {
		t.Fatalf("expected detached ai metadata, got %#v", meta)
	}
	typed, ok := ServicesAs[struct{ Name string }](detached)
	if !ok || typed.Name != "svc" {
		t.Fatalf("expected detached services, got %#v ok=%v", typed, ok)
	}
}
