package runtime

import (
	"testing"
	"time"
)

func TestDeltaBuffer_Append_MinChunkSize(t *testing.T) {
	buf := NewDeltaBuffer(DeltaBufferConfig{MinChunkSize: 10, MaxWaitMs: 1000})

	// Under threshold - no flush
	got := buf.Append("hello", "agent")
	if len(got) != 0 {
		t.Fatalf("expected no flush, got %d events", len(got))
	}

	// Reach threshold - should flush
	got = buf.Append(" world!!!", "agent") // total 14 chars
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].Event != "delta" {
		t.Errorf("expected event delta, got %s", got[0].Event)
	}

	data := got[0].Data.(map[string]any)
	if data["content"] != "hello world!!!" {
		t.Errorf("expected content='hello world!!!', got %v", data["content"])
	}
}

func TestDeltaBuffer_Flush(t *testing.T) {
	buf := NewDeltaBuffer(DeltaBufferConfig{MinChunkSize: 100, MaxWaitMs: 1000})

	buf.Append("some content", "agent")
	got := buf.Flush()

	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].Event != "delta" {
		t.Errorf("expected event delta, got %s", got[0].Event)
	}
}

func TestDeltaBuffer_Flush_Empty(t *testing.T) {
	buf := NewDeltaBuffer(DeltaBufferConfig{})

	got := buf.Flush()
	if len(got) != 0 {
		t.Fatalf("expected 0 events for empty buffer, got %d", len(got))
	}
}

func TestDeltaBuffer_ShouldFlushByTime(t *testing.T) {
	buf := NewDeltaBuffer(DeltaBufferConfig{MinChunkSize: 1000, MaxWaitMs: 50})

	buf.Append("content", "agent")

	// Immediately after append - should not flush
	if buf.ShouldFlushByTime() {
		t.Fatal("expected ShouldFlushByTime to return false immediately")
	}

	time.Sleep(60 * time.Millisecond)

	if !buf.ShouldFlushByTime() {
		t.Fatal("expected ShouldFlushByTime to return true after timeout")
	}
}

func TestDeltaBuffer_DefaultConfig(t *testing.T) {
	buf := NewDeltaBuffer(DeltaBufferConfig{})

	// Default values should be applied
	if buf.config.MinChunkSize != 50 {
		t.Errorf("expected default MinChunkSize=50, got %d", buf.config.MinChunkSize)
	}
	if buf.config.MaxWaitMs != 100 {
		t.Errorf("expected default MaxWaitMs=100, got %d", buf.config.MaxWaitMs)
	}
}
