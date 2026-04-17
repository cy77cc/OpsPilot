package workers

import (
	"context"
	"time"
)

type Runner struct {
	tick  func(context.Context)
	every time.Duration
}

func NewRunner(tick func(context.Context), every time.Duration) *Runner {
	return &Runner{tick: tick, every: every}
}

func (r *Runner) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		if ctx == nil || r == nil {
			return
		}

		tick := r.tick
		if ctx.Err() != nil {
			return
		}
		if tick != nil {
			tick(ctx)
		}

		every := r.every
		if every <= 0 {
			every = time.Second
		}
		ticker := time.NewTicker(every)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			if tick != nil {
				tick(ctx)
			}
		}
	}()

	return done
}
