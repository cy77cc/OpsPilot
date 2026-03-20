package ai

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

func TestUpdateRunStatus_SetsFinishedAtForTerminalStates(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAIRunDAO(db)
	ctx := context.Background()

	run := &model.AIRun{
		ID:            "run-1",
		SessionID:     "session-1",
		UserMessageID: "msg-1",
		Status:        "running",
		TraceJSON:     "{}",
	}
	if err := dao.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	if err := dao.UpdateRunStatus(ctx, run.ID, AIRunStatusUpdate{
		Status:          "completed",
		ProgressSummary: "done",
	}); err != nil {
		t.Fatalf("update run status: %v", err)
	}

	refreshed, err := dao.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if refreshed == nil || refreshed.FinishedAt == nil || refreshed.FinishedAt.IsZero() {
		t.Fatalf("expected finished_at to be set for completed run, got %#v", refreshed)
	}
}

func TestUpdateRunStatus_DoesNotBlankExistingFieldsOnPartialUpdate(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAIRunDAO(db)
	ctx := context.Background()

	run := &model.AIRun{
		ID:            "run-2",
		SessionID:     "session-2",
		UserMessageID: "msg-2",
		Status:        "running",
		TraceJSON:     "{}",
	}
	if err := dao.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	if err := dao.UpdateRunStatus(ctx, run.ID, AIRunStatusUpdate{
		IntentType:    "diagnosis",
		AssistantType: "DiagnosisAgent",
	}); err != nil {
		t.Fatalf("partial update run status: %v", err)
	}

	refreshed, err := dao.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if refreshed == nil {
		t.Fatal("expected run to exist")
	}
	if refreshed.Status != "running" {
		t.Fatalf("expected status to remain running, got %q", refreshed.Status)
	}
	if refreshed.IntentType != "diagnosis" || refreshed.AssistantType != "DiagnosisAgent" {
		t.Fatalf("expected partial fields to be updated, got %#v", refreshed)
	}
}

func TestListBySessionIDs_ReturnsRunsAcrossSessions(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAIRunDAO(db)
	ctx := context.Background()

	for _, run := range []*model.AIRun{
		{ID: "run-a", SessionID: "session-a", UserMessageID: "msg-a", Status: "completed", TraceJSON: "{}"},
		{ID: "run-b", SessionID: "session-b", UserMessageID: "msg-b", Status: "completed", TraceJSON: "{}"},
		{ID: "run-c", SessionID: "session-c", UserMessageID: "msg-c", Status: "completed", TraceJSON: "{}"},
	} {
		if err := dao.CreateRun(ctx, run); err != nil {
			t.Fatalf("create run %s: %v", run.ID, err)
		}
	}

	runs, err := dao.ListBySessionIDs(ctx, []string{"session-a", "session-c"})
	if err != nil {
		t.Fatalf("list runs by session ids: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
	if runs[0].SessionID != "session-a" || runs[1].SessionID != "session-c" {
		t.Fatalf("unexpected runs returned: %#v", runs)
	}
}

func TestAIRunDAO_FindByClientRequestID(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAIRunDAO(db)
	ctx := context.Background()

	run := &model.AIRun{
		ID:              "run-find",
		SessionID:       "session-find",
		UserMessageID:   "msg-find",
		ClientRequestID: "req-find",
		Status:          "running",
		TraceJSON:       "{}",
	}
	if err := dao.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	found, err := dao.FindByClientRequestID(ctx, run.SessionID, run.ClientRequestID)
	if err != nil {
		t.Fatalf("find run by client request id: %v", err)
	}
	if found == nil {
		t.Fatal("expected run to be found")
	}
	if found.ID != run.ID {
		t.Fatalf("expected run %q, got %q", run.ID, found.ID)
	}

	missing, err := dao.FindByClientRequestID(ctx, run.SessionID, "req-missing")
	if err != nil {
		t.Fatalf("find missing run by client request id: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected no run for missing client request id, got %#v", missing)
	}
}

func TestAIRunDAO_CreateOrReuseRunShell(t *testing.T) {
	db := newAIDAOTestDB(t)
	enableSQLiteBusyTimeout(t, db)

	chatDAO := NewAIChatDAO(db)
	dao := NewAIRunDAO(db)
	ctx := context.Background()

	session := &model.AIChatSession{
		ID:     "session-shell",
		UserID: 42,
		Scene:  "ai",
		Title:  "shell",
	}
	if err := chatDAO.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	const attempts = 6
	start := make(chan struct{})
	results := make(chan createOrReuseRunShellResult, attempts)
	var wg sync.WaitGroup
	var buildCalls atomic.Int32

	for i := range attempts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start

			run, created, err := dao.CreateOrReuseRunShell(ctx, session.UserID, session.ID, "req-shell", func() (*model.AIRun, *model.AIChatMessage, *model.AIChatMessage) {
				buildCalls.Add(1)
				runID := fmt.Sprintf("run-shell-%d", i)
				userMessageID := fmt.Sprintf("msg-user-%d", i)
				assistantMessageID := fmt.Sprintf("msg-assistant-%d", i)
				return &model.AIRun{
						ID:                 runID,
						SessionID:          session.ID,
						UserMessageID:      userMessageID,
						AssistantMessageID: assistantMessageID,
						ClientRequestID:    "req-shell",
						Status:             "running",
						TraceJSON:          "{}",
					}, &model.AIChatMessage{
						ID:        userMessageID,
						SessionID: session.ID,
						Role:      "user",
						Content:   "hello",
						Status:    "done",
					}, &model.AIChatMessage{
						ID:        assistantMessageID,
						SessionID: session.ID,
						Role:      "assistant",
						Content:   "",
						Status:    "streaming",
					}
			})
			results <- createOrReuseRunShellResult{
				run:     run,
				created: created,
				err:     err,
			}
		}(i)
	}

	close(start)
	wg.Wait()
	close(results)

	var (
		createdCount int
		reusedCount  int
		survivingRun *model.AIRun
	)
	for result := range results {
		if result.err != nil {
			t.Fatalf("create or reuse run shell: %v", result.err)
		}
		if result.run == nil {
			t.Fatal("expected run to be returned")
		}
		if survivingRun == nil {
			survivingRun = result.run
		}
		if result.run.ID != survivingRun.ID {
			t.Fatalf("expected all callers to receive the same run id, got %q and %q", survivingRun.ID, result.run.ID)
		}
		if result.created {
			createdCount++
		} else {
			reusedCount++
		}
	}

	if createdCount != 1 {
		t.Fatalf("expected exactly one created shell, got %d", createdCount)
	}
	if reusedCount != attempts-1 {
		t.Fatalf("expected %d reused shells, got %d", attempts-1, reusedCount)
	}
	if buildCalls.Load() == 0 {
		t.Fatal("expected build callback to run at least once")
	}

	runs, err := dao.ListBySession(ctx, session.ID)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected one surviving run shell, got %d", len(runs))
	}
	if runs[0].ClientRequestID != "req-shell" {
		t.Fatalf("expected surviving run to keep client request id, got %#v", runs[0])
	}

	messages, err := chatDAO.ListMessagesBySession(ctx, session.ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected only one user/assistant message pair, got %d messages", len(messages))
	}
	if messages[0].Role != "user" || messages[1].Role != "assistant" {
		t.Fatalf("unexpected messages returned: %#v", messages)
	}
}

type createOrReuseRunShellResult struct {
	run     *model.AIRun
	created bool
	err     error
}

func enableSQLiteBusyTimeout(t *testing.T, db *gorm.DB) {
	t.Helper()

	if execErr := db.Exec("PRAGMA busy_timeout = 5000").Error; execErr != nil {
		t.Fatalf("set sqlite busy timeout: %v", execErr)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
}
