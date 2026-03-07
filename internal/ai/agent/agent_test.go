package agent

import (
	"context"
	"testing"
)

func TestNewPlatformAgent_NilModel(t *testing.T) {
	a, err := newPlatformAgent(context.Background(), nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil model")
	}
	if a != nil {
		t.Fatalf("expected nil agent when model is nil")
	}
}
