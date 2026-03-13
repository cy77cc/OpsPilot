package runtime

import (
	"context"
	"testing"
)

func TestCheckpointStoreBindsAndResolvesIdentity(t *testing.T) {
	store := NewCheckpointStore(nil, "")
	ctx := context.Background()
	if err := store.Set(ctx, "cp-1", []byte("state")); err != nil {
		t.Fatalf("Set error = %v", err)
	}
	if err := store.BindIdentity(ctx, "session-1", "plan-1", "step-1", "cp-1", "interrupt-1"); err != nil {
		t.Fatalf("BindIdentity error = %v", err)
	}
	checkpointID, target, ok, err := store.Resolve(ctx, "session-1", "plan-1", "step-1", "")
	if err != nil {
		t.Fatalf("Resolve error = %v", err)
	}
	if !ok || checkpointID != "cp-1" || target != "interrupt-1" {
		t.Fatalf("resolved = (%q, %q, %v)", checkpointID, target, ok)
	}
}
