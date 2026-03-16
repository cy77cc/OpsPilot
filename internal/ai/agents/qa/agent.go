package qa

import (
	"context"
	"strings"
)

type Request struct {
	Message string
}

type Result struct {
	Text string
}

type Agent struct{}

func NewAgent() *Agent {
	return &Agent{}
}

func (a *Agent) Answer(_ context.Context, req Request) (Result, error) {
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = "your question"
	}
	return Result{
		Text: "Phase 1 QA answer: " + message,
	}, nil
}
