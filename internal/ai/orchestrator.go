package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/ai/agents"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

type Orchestrator struct {
	runner *adk.Runner
}

type memoryCheckPointStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func newMemoryCheckPointStore() *memoryCheckPointStore {
	return &memoryCheckPointStore{data: make(map[string][]byte)}
}

func (s *memoryCheckPointStore) Get(_ context.Context, checkPointID string) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[checkPointID]
	if !ok {
		return nil, false, nil
	}
	copied := make([]byte, len(v))
	copy(copied, v)
	return copied, true, nil
}

func (s *memoryCheckPointStore) Set(_ context.Context, checkPointID string, checkPoint []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := make([]byte, len(checkPoint))
	copy(copied, checkPoint)
	s.data[checkPointID] = copied
	return nil
}

func NewOrchestrator(ctx context.Context, deps common.PlatformDeps) (*Orchestrator, error) {
	return NewOrchestratorWithCheckPointStore(ctx, deps, newMemoryCheckPointStore())
}

func NewOrchestratorWithCheckPointStore(ctx context.Context, deps common.PlatformDeps, store adk.CheckPointStore) (*Orchestrator, error) {
	agent, err := agents.NewAgent(ctx, deps)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}
	return &Orchestrator{
		runner: adk.NewRunner(ctx, adk.RunnerConfig{
			Agent:           agent,
			CheckPointStore: store,
			EnableStreaming: true,
		}),
	}, nil
}

func (o *Orchestrator) Run(ctx context.Context, input string) (string, error) {
	text, interrupted, err := o.RunWithCheckPoint(ctx, input, "")
	if err != nil {
		return "", err
	}
	if interrupted {
		return "", fmt.Errorf("agent execution interrupted, resume is required")
	}
	return text, nil
}

func (o *Orchestrator) RunWithCheckPoint(ctx context.Context, input, checkPointID string) (string, bool, error) {
	if o == nil || o.runner == nil {
		return "", false, fmt.Errorf("orchestrator runner is nil")
	}

	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", false, fmt.Errorf("input is empty")
	}

	var iter *adk.AsyncIterator[*adk.AgentEvent]
	if strings.TrimSpace(checkPointID) == "" {
		iter = o.runner.Query(ctx, trimmed)
	} else {
		iter = o.runner.Query(ctx, trimmed, adk.WithCheckPointID(checkPointID))
	}

	return consumeEvents(iter)
}

func (o *Orchestrator) Resume(ctx context.Context, checkPointID string) (string, bool, error) {
	if o == nil || o.runner == nil {
		return "", false, fmt.Errorf("orchestrator runner is nil")
	}
	if strings.TrimSpace(checkPointID) == "" {
		return "", false, fmt.Errorf("checkpoint id is empty")
	}

	iter, err := o.runner.Resume(ctx, checkPointID)
	if err != nil {
		return "", false, err
	}
	return consumeEvents(iter)
}

func (o *Orchestrator) ResumeWithParams(ctx context.Context, checkPointID string, params *adk.ResumeParams) (string, bool, error) {
	if o == nil || o.runner == nil {
		return "", false, fmt.Errorf("orchestrator runner is nil")
	}
	if strings.TrimSpace(checkPointID) == "" {
		return "", false, fmt.Errorf("checkpoint id is empty")
	}

	iter, err := o.runner.ResumeWithParams(ctx, checkPointID, params)
	if err != nil {
		return "", false, err
	}
	return consumeEvents(iter)
}

func consumeEvents(iter *adk.AsyncIterator[*adk.AgentEvent]) (string, bool, error) {
	if iter == nil {
		return "", false, fmt.Errorf("event iterator is nil")
	}

	var (
		lastText    string
		interrupted bool
	)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		
		if event.Err != nil {
			return "", false, event.Err
		}

		msg, _, err := adk.GetMessage(event)
		if err != nil {
			return "", false, fmt.Errorf("failed to parse agent message: %w", err)
		}
		if msg != nil && strings.TrimSpace(msg.Content) != "" {
			lastText = msg.Content
		}

		if event.Action != nil && event.Action.Interrupted != nil {
			interrupted = true
		}
	}

	if strings.TrimSpace(lastText) == "" {
		if interrupted {
			return "", true, nil
		}
		return "", false, fmt.Errorf("agent returned empty response")
	}

	return lastText, interrupted, nil
}
