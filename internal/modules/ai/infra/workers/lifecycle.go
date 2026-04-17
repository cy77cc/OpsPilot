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

		if ctx == nil {
			ctx = context.Background()
		}
		if r == nil {
			return
		}

		if r.tick != nil {
			r.tick(ctx)
		}

		<-ctx.Done()
	}()

	return done
}
