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
