package streaming

import (
	"errors"
	"testing"
)

func TestMapStreamError_DefaultRuntimeError(t *testing.T) {
	out := MapStreamError(errors.New("db timeout"))

	if out.Code != "AI_STREAM_INTERNAL" {
		t.Fatalf("unexpected code: %s", out.Code)
	}
	if out.Message == "db timeout" {
		t.Fatal("raw error message leaked")
	}
	if !out.Retryable {
		t.Fatal("unexpected retryable=false for default runtime error")
	}
}
