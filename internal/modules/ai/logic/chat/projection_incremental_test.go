package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
	aidaochat "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/chat"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	projectionruntime "github.com/cy77cc/OpsPilot/internal/modules/ai/runtime/projection"
)

func TestGetRunProjection_UsesIncrementalProjectionRow(t *testing.T) {
	db := newProjectionRuntimeTestDB(t)
	ctx := context.Background()

	chatDAO := aidaochat.NewAIChatDAO(db)
	runDAO := aidao.NewAIRunDAO(db)
	projectionDAO := aidao.NewAIRunProjectionDAO(db)

	session := &ai.AIChatSession{ID: "sess-1", UserID: 7, Scene: "ops", Title: "projection"}
	if err := chatDAO.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	run := &ai.AIRun{ID: "run-1", SessionID: session.ID, UserMessageID: "msg-1", AssistantMessageID: "msg-2", Status: "running", TraceJSON: "{}"}
	if err := runDAO.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	storedProjection := &ai.AIRunProjection{
		ID:             "proj-1",
		RunID:          run.ID,
		SessionID:      session.ID,
		Version:        7,
		Status:         "running",
		ProjectionJSON: `{"version":7,"run_id":"run-1","session_id":"sess-1","status":"running","blocks":[{"id":"block_executor_1","type":"executor","title":"执行过程","lazy":true,"items":[{"id":"item-1","type":"content","content_id":"content-1"}]}]}`,
	}
	if err := projectionDAO.Upsert(ctx, storedProjection); err != nil {
		t.Fatalf("upsert projection: %v", err)
	}
	if err := aidao.NewAIRunEventDAO(db).Create(ctx, &ai.AIRunEvent{
		ID:          "evt-1",
		RunID:       run.ID,
		SessionID:   session.ID,
		Seq:         1,
		EventType:   "delta",
		PayloadJSON: `{"agent":"executor","content":"rebuild me not"}`,
	}); err != nil {
		t.Fatalf("create event: %v", err)
	}

	l := &Logic{ChatDAO: chatDAO, RunDAO: runDAO, RunEventDAO: aidao.NewAIRunEventDAO(db), RunProjectionDAO: projectionDAO}
	got, err := GetRunProjection(ctx, l, session.UserID, run.ID)
	if err != nil {
		t.Fatalf("get projection: %v", err)
	}
	if got == nil {
		t.Fatal("expected projection row")
	}
	if got.Version != 7 {
		t.Fatalf("expected stored version 7, got %d", got.Version)
	}
	if got.ProjectionJSON != storedProjection.ProjectionJSON {
		t.Fatalf("expected stored projection json to be returned, got %s", got.ProjectionJSON)
	}

	payload, err := GetRunProjectionPayload(ctx, l, session.UserID, run.ID, RunProjectionQuery{})
	if err != nil {
		t.Fatalf("get projection payload: %v", err)
	}
	decoded, ok := payload.(airuntime.RunProjection)
	if !ok {
		t.Fatalf("expected run projection payload, got %T", payload)
	}
	if decoded.Version != 7 || decoded.Status != "running" || len(decoded.Blocks) != 1 {
		t.Fatalf("unexpected payload: %#v", decoded)
	}
}

func TestPersistRunEnhancementsBestEffort_PreservesIncrementalVersion(t *testing.T) {
	db := newProjectionRuntimeTestDB(t)
	ctx := context.Background()

	chatDAO := aidaochat.NewAIChatDAO(db)
	runDAO := aidao.NewAIRunDAO(db)
	projectionDAO := aidao.NewAIRunProjectionDAO(db)
	contentDAO := aidao.NewAIRunContentDAO(db)
	l := &Logic{ChatDAO: chatDAO, RunDAO: runDAO, RunProjectionDAO: projectionDAO, RunContentDAO: contentDAO}

	session := &ai.AIChatSession{ID: "sess-1", UserID: 7, Scene: "ops", Title: "projection"}
	if err := chatDAO.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	run := &ai.AIRun{ID: "run-1", SessionID: session.ID, UserMessageID: "msg-1", AssistantMessageID: "msg-2", Status: "completed", TraceJSON: "{}"}
	if err := runDAO.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	projection := airuntime.RunProjection{
		Version:   9,
		RunID:     run.ID,
		SessionID: session.ID,
		Status:    "running",
		Blocks: []airuntime.ProjectionBlock{{
			ID:    "block_executor_1",
			Type:  "executor",
			Title: "执行过程",
			Items: []airuntime.ProjectionExecutorItem{{ID: "item-1", Type: "content", ContentID: "content-1"}},
		}},
	}
	raw, err := json.Marshal(projection)
	if err != nil {
		t.Fatalf("marshal projection: %v", err)
	}
	if err := projectionDAO.Upsert(ctx, &ai.AIRunProjection{
		ID:             "proj-1",
		RunID:          run.ID,
		SessionID:      session.ID,
		Version:        9,
		Status:         "running",
		ProjectionJSON: string(raw),
	}); err != nil {
		t.Fatalf("seed projection: %v", err)
	}

	if err := PersistRunEnhancementsBestEffort(ctx, l, run.ID, session.ID, "completed", "ignored"); err != nil {
		t.Fatalf("persist run enhancements: %v", err)
	}

	got, err := projectionDAO.GetByRunID(ctx, run.ID)
	if err != nil {
		t.Fatalf("load projection: %v", err)
	}
	if got == nil {
		t.Fatal("expected projection row")
	}
	if got.Version != 9 {
		t.Fatalf("expected version 9 to be preserved, got %d", got.Version)
	}
	var decoded airuntime.RunProjection
	if err := json.Unmarshal([]byte(got.ProjectionJSON), &decoded); err != nil {
		t.Fatalf("unmarshal projection: %v", err)
	}
	if decoded.Version != 9 || decoded.Status != "completed" {
		t.Fatalf("expected stored projection to keep version and update status, got %#v", decoded)
	}
}

func TestPersistTerminalProjectionEvent_AppliesDoneSummary(t *testing.T) {
	db := newProjectionRuntimeTestDB(t)
	ctx := context.Background()

	chatDAO := aidaochat.NewAIChatDAO(db)
	runDAO := aidao.NewAIRunDAO(db)
	projectionDAO := aidao.NewAIRunProjectionDAO(db)
	contentDAO := aidao.NewAIRunContentDAO(db)
	l := &Logic{ChatDAO: chatDAO, RunDAO: runDAO, RunProjectionDAO: projectionDAO, RunContentDAO: contentDAO}

	session := &ai.AIChatSession{ID: "sess-1", UserID: 7, Scene: "ops", Title: "projection"}
	if err := chatDAO.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	run := &ai.AIRun{ID: "run-1", SessionID: session.ID, UserMessageID: "msg-1", AssistantMessageID: "msg-2", Status: "running", TraceJSON: "{}"}
	if err := runDAO.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	projection := airuntime.RunProjection{
		Version:   9,
		RunID:     run.ID,
		SessionID: session.ID,
		Status:    "running",
		Blocks: []airuntime.ProjectionBlock{{
			ID:    "block_executor_1",
			Type:  "executor",
			Title: "执行过程",
		}},
	}
	raw, err := json.Marshal(projection)
	if err != nil {
		t.Fatalf("marshal projection: %v", err)
	}
	if err := projectionDAO.Upsert(ctx, &ai.AIRunProjection{
		ID:             "proj-1",
		RunID:          run.ID,
		SessionID:      session.ID,
		Version:        9,
		Status:         "running",
		ProjectionJSON: string(raw),
	}); err != nil {
		t.Fatalf("seed projection: %v", err)
	}

	done := airuntime.PublicStreamEvent{
		Event: "done",
		Data: map[string]any{
			"run_id":  run.ID,
			"status":  "completed",
			"summary": "all good",
		},
	}
	if err := persistTerminalProjectionEvent(ctx, l, run.ID, session.ID, "evt-done", done); err != nil {
		t.Fatalf("persist terminal projection event: %v", err)
	}

	got, err := projectionDAO.GetByRunID(ctx, run.ID)
	if err != nil {
		t.Fatalf("load projection: %v", err)
	}
	var decoded airuntime.RunProjection
	if err := json.Unmarshal([]byte(got.ProjectionJSON), &decoded); err != nil {
		t.Fatalf("unmarshal projection: %v", err)
	}
	if got.Version != 10 || decoded.Version != 10 {
		t.Fatalf("expected version 10 after done event, got row=%d decoded=%d", got.Version, decoded.Version)
	}
	if decoded.Status != "completed" {
		t.Fatalf("expected completed status, got %#v", decoded)
	}
	if decoded.Summary == nil || decoded.Summary.Content != "all good" {
		t.Fatalf("expected done summary to be applied, got %#v", decoded.Summary)
	}
}

func TestPersistTerminalProjectionEvent_AppendsErrorBlock(t *testing.T) {
	db := newProjectionRuntimeTestDB(t)
	ctx := context.Background()

	chatDAO := aidaochat.NewAIChatDAO(db)
	runDAO := aidao.NewAIRunDAO(db)
	projectionDAO := aidao.NewAIRunProjectionDAO(db)
	contentDAO := aidao.NewAIRunContentDAO(db)
	l := &Logic{ChatDAO: chatDAO, RunDAO: runDAO, RunProjectionDAO: projectionDAO, RunContentDAO: contentDAO}

	session := &ai.AIChatSession{ID: "sess-1", UserID: 7, Scene: "ops", Title: "projection"}
	if err := chatDAO.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	run := &ai.AIRun{ID: "run-1", SessionID: session.ID, UserMessageID: "msg-1", AssistantMessageID: "msg-2", Status: "running", TraceJSON: "{}"}
	if err := runDAO.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	projection := airuntime.RunProjection{
		Version:   3,
		RunID:     run.ID,
		SessionID: session.ID,
		Status:    "running",
		Blocks:    []airuntime.ProjectionBlock{},
	}
	raw, err := json.Marshal(projection)
	if err != nil {
		t.Fatalf("marshal projection: %v", err)
	}
	if err := projectionDAO.Upsert(ctx, &ai.AIRunProjection{
		ID:             "proj-1",
		RunID:          run.ID,
		SessionID:      session.ID,
		Version:        3,
		Status:         "running",
		ProjectionJSON: string(raw),
	}); err != nil {
		t.Fatalf("seed projection: %v", err)
	}

	errEvent := airuntime.PublicStreamEvent{
		Event: "error",
		Data: map[string]any{
			"run_id":  run.ID,
			"message": "fatal",
			"code":    "AI_STREAM_INTERNAL",
		},
	}
	if err := persistTerminalProjectionEvent(ctx, l, run.ID, session.ID, "evt-error", errEvent); err != nil {
		t.Fatalf("persist terminal projection event: %v", err)
	}

	got, err := projectionDAO.GetByRunID(ctx, run.ID)
	if err != nil {
		t.Fatalf("load projection: %v", err)
	}
	var decoded airuntime.RunProjection
	if err := json.Unmarshal([]byte(got.ProjectionJSON), &decoded); err != nil {
		t.Fatalf("unmarshal projection: %v", err)
	}
	if got.Version != 4 || decoded.Version != 4 {
		t.Fatalf("expected version 4 after error event, got row=%d decoded=%d", got.Version, decoded.Version)
	}
	if decoded.Status != "failed_runtime" {
		t.Fatalf("expected failed_runtime status, got %#v", decoded)
	}
	if len(decoded.Blocks) != 1 || decoded.Blocks[0].Type != "error" {
		t.Fatalf("expected appended error block, got %#v", decoded.Blocks)
	}
}

func TestLoadIncrementalProjectionState_HydratesCurrentContentForDeltaCoalescing(t *testing.T) {
	db := newProjectionRuntimeTestDB(t)
	ctx := context.Background()

	chatDAO := aidaochat.NewAIChatDAO(db)
	runDAO := aidao.NewAIRunDAO(db)
	projectionDAO := aidao.NewAIRunProjectionDAO(db)
	contentDAO := aidao.NewAIRunContentDAO(db)
	l := &Logic{ChatDAO: chatDAO, RunDAO: runDAO, RunProjectionDAO: projectionDAO, RunContentDAO: contentDAO}

	session := &ai.AIChatSession{ID: "sess-1", UserID: 7, Scene: "ops", Title: "projection"}
	if err := chatDAO.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	run := &ai.AIRun{ID: "run-1", SessionID: session.ID, UserMessageID: "msg-1", AssistantMessageID: "msg-2", Status: "running", TraceJSON: "{}"}
	if err := runDAO.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	if err := contentDAO.Create(ctx, &ai.AIRunContent{
		ID:          "content-1",
		RunID:       run.ID,
		SessionID:   session.ID,
		ContentKind: "executor_content",
		Encoding:    "text",
		SummaryText: "hello",
		BodyText:    "hello",
		SizeBytes:   5,
	}); err != nil {
		t.Fatalf("seed content: %v", err)
	}
	projection := airuntime.RunProjection{
		Version:   1,
		RunID:     run.ID,
		SessionID: session.ID,
		Status:    "running",
		Blocks: []airuntime.ProjectionBlock{{
			ID:    "block_executor_1",
			Type:  "executor",
			Title: "执行过程",
			Items: []airuntime.ProjectionExecutorItem{{
				ID:           "item-1",
				Type:         "content",
				ContentID:    "content-1",
				StartEventID: "evt-1",
				EndEventID:   "evt-1",
			}},
		}},
	}
	raw, err := json.Marshal(projection)
	if err != nil {
		t.Fatalf("marshal projection: %v", err)
	}
	if err := projectionDAO.Upsert(ctx, &ai.AIRunProjection{
		ID:             "proj-1",
		RunID:          run.ID,
		SessionID:      session.ID,
		Version:        1,
		Status:         "running",
		ProjectionJSON: string(raw),
	}); err != nil {
		t.Fatalf("seed projection: %v", err)
	}

	state, current, err := loadIncrementalProjectionState(ctx, l, run.ID)
	if err != nil {
		t.Fatalf("load incremental state: %v", err)
	}
	state = projectionruntime.ApplyEvent(state, projectionruntime.Event{
		ID:   "evt-2",
		Type: "assistant.delta",
		Text: " world",
		Data: map[string]any{"agent": "executor"},
	})
	if err := persistIncrementalProjection(ctx, l, session.ID, state, current); err != nil {
		t.Fatalf("persist incremental projection: %v", err)
	}

	gotContent, err := contentDAO.Get(ctx, "content-1")
	if err != nil {
		t.Fatalf("get content: %v", err)
	}
	if gotContent == nil || gotContent.BodyText != "hello world" {
		t.Fatalf("expected hydrated content to coalesce across reload, got %#v", gotContent)
	}

	gotProjection, err := projectionDAO.GetByRunID(ctx, run.ID)
	if err != nil {
		t.Fatalf("get projection: %v", err)
	}
	var decoded airuntime.RunProjection
	if err := json.Unmarshal([]byte(gotProjection.ProjectionJSON), &decoded); err != nil {
		t.Fatalf("unmarshal projection: %v", err)
	}
	if len(decoded.Blocks) != 1 || len(decoded.Blocks[0].Items) != 1 {
		t.Fatalf("expected one coalesced content item after reload, got %#v", decoded.Blocks)
	}
	if decoded.Blocks[0].Items[0].StartEventID != "evt-1" || decoded.Blocks[0].Items[0].EndEventID != "evt-2" {
		t.Fatalf("expected content event span to extend across reload, got %#v", decoded.Blocks[0].Items[0])
	}
}

func newProjectionRuntimeTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
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
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}
