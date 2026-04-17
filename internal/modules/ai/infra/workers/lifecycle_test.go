package workers

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerStopsOnContextCancel(t *testing.T) {
	var ticks atomic.Int32

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := NewRunner(func(context.Context) { ticks.Add(1) }, 5*time.Millisecond)
	done := r.Start(ctx)

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("runner did not stop after cancel")
	}

	if ticks.Load() == 0 {
		t.Fatal("runner never ticked before cancel")
	}
}

func TestRunnerTicksImmediately(t *testing.T) {
	ticked := make(chan struct{}, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := NewRunner(func(context.Context) {
		select {
		case ticked <- struct{}{}:
		default:
		}
	}, time.Hour)
	done := r.Start(ctx)

	select {
	case <-ticked:
	case <-time.After(50 * time.Millisecond):
		t.Fatal("runner did not tick immediately")
	}

	select {
	case <-done:
		t.Fatal("runner exited before cancel")
	default:
	}

	cancel()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("runner did not stop after cancel")
	}
}

func TestRunnerClosesDoneImmediatelyForNilContext(t *testing.T) {
	r := NewRunner(func(context.Context) {}, 5*time.Millisecond)

	select {
	case <-r.Start(nil):
	case <-time.After(50 * time.Millisecond):
		t.Fatal("runner did not stop immediately for nil context")
	}
}

func TestRunnerTicksAgainAfterIntervalWhenCallbackReturns(t *testing.T) {
	var ticks atomic.Int32

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := NewRunner(func(context.Context) { ticks.Add(1) }, 5*time.Millisecond)
	done := r.Start(ctx)

	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("runner did not stop after cancel")
	}

	if got := ticks.Load(); got < 2 {
		t.Fatalf("expected runner to tick more than once, got %d", got)
	}
}

func TestRunnerDoesNotTickWhenContextAlreadyCanceled(t *testing.T) {
	var ticks atomic.Int32

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := NewRunner(func(context.Context) { ticks.Add(1) }, 5*time.Millisecond)
	done := r.Start(ctx)

	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatal("runner did not stop for canceled context")
	}

	if got := ticks.Load(); got != 0 {
		t.Fatalf("expected no ticks for canceled context, got %d", got)
	}
}
