package chat

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	sharedapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/approval"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestResumeApprovedTask_CompletesRunFromCheckpoint(t *testing.T) {
	db := newResumeChatTestDB(t)
	store := newMemoryCheckpointStore()
	agent := &resumableApprovalAgent{}

	runCtx := context.Background()
	runner := adk.NewRunner(runCtx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: store,
	})
	iter := runner.Run(runCtx, []adk.Message{schema.UserMessage("resume me")}, adk.WithCheckPointID("checkpoint-resume"))

	var interruptID string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Action != nil && event.Action.Interrupted != nil {
			for _, interruptCtx := range event.Action.Interrupted.InterruptContexts {
				if interruptCtx != nil && interruptCtx.IsRootCause {
					interruptID = interruptCtx.ID
					break
				}
			}
		}
	}
	if interruptID == "" {
		t.Fatal("expected interrupt id to be captured from checkpointed run")
	}

	sessionID := "session-resume"
	runID := "run-resume"
	userMessageID := "msg-resume-user"
	assistantMessageID := "msg-resume-assistant"
	if err := db.Create(&ai.AIChatSession{
		ID:     sessionID,
		UserID: 42,
		Scene:  "ai",
		Title:  "resume test",
	}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if err := db.Create(&ai.AIChatMessage{
		ID:           userMessageID,
		SessionID:    sessionID,
		SessionIDNum: 1,
		Role:         "user",
		Content:      "resume me",
		Status:       "done",
	}).Error; err != nil {
		t.Fatalf("seed user message: %v", err)
	}
	if err := db.Create(&ai.AIChatMessage{
		ID:           assistantMessageID,
		SessionID:    sessionID,
		SessionIDNum: 2,
		Role:         "assistant",
		Content:      "",
		Status:       "streaming",
	}).Error; err != nil {
		t.Fatalf("seed assistant message: %v", err)
	}
	if err := db.Create(&ai.AIRun{
		ID:                 runID,
		SessionID:          sessionID,
		ClientRequestID:    "req-resume",
		UserMessageID:      userMessageID,
		AssistantMessageID: assistantMessageID,
		Status:             "waiting_approval",
		TraceJSON:          `{}`,
	}).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	if err := db.Create(&ai.AIApprovalTask{
		ApprovalID:     "approval-resume",
		CheckpointID:   "checkpoint-resume",
		SessionID:      sessionID,
		RunID:          runID,
		UserID:         42,
		ToolName:       "host_exec",
		ToolCallID:     "tool-call-resume",
		ResumeTargetID: interruptID,
		ArgumentsJSON:  `{"command":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "approved",
	}).Error; err != nil {
		t.Fatalf("seed approval task: %v", err)
	}

	logic := New(&svc.ServiceContext{DB: db})
	logic.AIRouter = agent
	logic.CheckpointStore = store

	task := &ai.AIApprovalTask{
		ApprovalID:     "approval-resume",
		CheckpointID:   "checkpoint-resume",
		SessionID:      sessionID,
		RunID:          runID,
		UserID:         42,
		ToolName:       "host_exec",
		ToolCallID:     "tool-call-resume",
		ResumeTargetID: interruptID,
		Comment:        "approved",
	}

	_, err := ResumeApprovedTask(context.Background(), logic, task, &adk.ResumeParams{
		Targets: map[string]any{
			interruptID: &sharedapproval.ApprovalResult{Approved: true, Comment: "approved"},
		},
	})
	if err != nil {
		t.Fatalf("resume approved task: %v", err)
	}

	run, err := logic.RunDAO.GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if run == nil || run.Status != "completed" {
		t.Fatalf("expected completed run, got %#v", run)
	}

	assistant, err := logic.ChatDAO.GetMessage(context.Background(), assistantMessageID)
	if err != nil {
		t.Fatalf("reload assistant: %v", err)
	}
	if assistant == nil || assistant.Content != "resumed ok" {
		t.Fatalf("expected resumed assistant content, got %#v", assistant)
	}
}

type resumableApprovalAgent struct{}

func (a *resumableApprovalAgent) Name(context.Context) string        { return "resumable-approval-agent" }
func (a *resumableApprovalAgent) Description(context.Context) string { return "resumable approval agent" }

func (a *resumableApprovalAgent) Run(ctx context.Context, input *adk.AgentInput, opts ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		gen.Send(adk.StatefulInterrupt(ctx, "approval-resume", "waiting_approval"))
		gen.Close()
	}()
	_ = input
	return iter
}

func (a *resumableApprovalAgent) Resume(ctx context.Context, info *adk.ResumeInfo, opts ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		defer gen.Close()
		result, _ := info.ResumeData.(*sharedapproval.ApprovalResult)
		if !info.IsResumeTarget || result == nil || !result.Approved {
			gen.Send(adk.StatefulInterrupt(ctx, "approval-resume", "waiting_approval"))
			return
		}
		gen.Send(adk.EventFromMessage(schema.AssistantMessage("resumed ok", nil), nil, schema.Assistant, ""))
	}()
	return iter
}

func newResumeChatTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&ai.AIChatSession{},
		&ai.AIChatMessage{},
		&ai.AIRun{},
		&ai.AIRunEvent{},
		&ai.AIRunProjection{},
		&ai.AIRunContent{},
		&ai.AIApprovalTask{},
		&ai.AIApprovalOutboxEvent{},
		&ai.AICheckpoint{},
	); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	return db
}

type memoryCheckpointStore struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newMemoryCheckpointStore() *memoryCheckpointStore {
	return &memoryCheckpointStore{data: make(map[string][]byte)}
}

func (s *memoryCheckpointStore) Get(_ context.Context, checkpointID string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, ok := s.data[checkpointID]
	if !ok {
		return nil, false, nil
	}
	return append([]byte(nil), payload...), true, nil
}

func (s *memoryCheckpointStore) Set(_ context.Context, checkpointID string, checkpoint []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[checkpointID] = append([]byte(nil), checkpoint...)
	return nil
}
