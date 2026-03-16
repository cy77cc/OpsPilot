package dao

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/testutil"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestAIChatDAO_SessionCRUD(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)

	ctx := context.Background()
	dao := NewAIChatDAO(suite.DB)

	session := &model.AIChatSession{
		ID:     uuid.NewString(),
		UserID: 1001,
		Scene:  "test-scene",
		Title:  "Test Session",
	}
	testutil.RequireNoError(t, dao.CreateSession(ctx, session))

	loaded, err := dao.GetSession(ctx, session.ID)
	testutil.RequireNoError(t, err)
	testutil.AssertEqual(t, session.UserID, loaded.UserID)
	testutil.AssertEqual(t, session.Scene, loaded.Scene)

	sessions, err := dao.ListSessions(ctx, session.UserID, "")
	testutil.RequireNoError(t, err)
	testutil.AssertLen[model.AIChatSession](t, 1, sessions)
	testutil.AssertEqual(t, session.ID, sessions[0].ID)

	second := &model.AIChatSession{
		ID:     uuid.NewString(),
		UserID: session.UserID,
		Scene:  "other-scene",
		Title:  "Other",
	}
	testutil.RequireNoError(t, dao.CreateSession(ctx, second))

	filtered, err := dao.ListSessions(ctx, session.UserID, second.Scene)
	testutil.RequireNoError(t, err)
	testutil.AssertLen[model.AIChatSession](t, 1, filtered)
	testutil.AssertEqual(t, second.ID, filtered[0].ID)

	testutil.RequireNoError(t, dao.DeleteSession(ctx, session.ID))
	_, err = dao.GetSession(ctx, session.ID)
	testutil.RequireError(t, err)
	testutil.AssertTrue(t, errors.Is(err, gorm.ErrRecordNotFound), "expected deleted session to be missing")

	remaining, err := dao.ListSessions(ctx, session.UserID, "")
	testutil.RequireNoError(t, err)
	testutil.AssertLen[model.AIChatSession](t, 1, remaining)
	testutil.AssertEqual(t, second.ID, remaining[0].ID)
}

func TestAIChatDAO_MessageOrdering(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)

	ctx := context.Background()
	dao := NewAIChatDAO(suite.DB)

	session := &model.AIChatSession{
		ID:     uuid.NewString(),
		UserID: 2002,
		Scene:  "ordering",
		Title:  "Message Order",
	}
	testutil.RequireNoError(t, dao.CreateSession(ctx, session))

	now := time.Now().UTC()
	earlier := now.Add(-time.Minute)
	later := now.Add(time.Minute)

	laterMessage := &model.AIChatMessage{
		ID:        uuid.NewString(),
		SessionID: session.ID,
		Role:      "user",
		Content:   "later",
		CreatedAt: later,
	}
	earlierMessage := &model.AIChatMessage{
		ID:        uuid.NewString(),
		SessionID: session.ID,
		Role:      "assistant",
		Content:   "earlier",
		CreatedAt: earlier,
	}

	// insert out of order to ensure ordering by CreatedAt is preserved.
	testutil.RequireNoError(t, dao.CreateMessage(ctx, laterMessage))
	testutil.RequireNoError(t, dao.CreateMessage(ctx, earlierMessage))

	messages, err := dao.ListMessagesBySession(ctx, session.ID)
	testutil.RequireNoError(t, err)
	testutil.AssertLen[model.AIChatMessage](t, 2, messages)
	testutil.AssertEqual(t, earlierMessage.ID, messages[0].ID)
	testutil.AssertEqual(t, laterMessage.ID, messages[1].ID)
}
