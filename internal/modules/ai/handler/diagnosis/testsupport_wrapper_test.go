package diagnosis_test

import (
	"fmt"
	"testing"

	chathandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/chat"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type aiHandlerTestHarness struct {
	chat  *chathandler.HTTPHandler
	logic *logic.Logic
}

func newAIHandlerTestHarness(db *gorm.DB) *aiHandlerTestHarness {
	l := logic.NewLogicWithDB(db, nil)
	chatSvc := chathandler.NewServiceWithLogic(l)
	return &aiHandlerTestHarness{
		chat:  chathandler.NewHTTPHandler(chatSvc),
		logic: l,
	}
}

func (h *aiHandlerTestHarness) GetDiagnosisReport(c *gin.Context) {
	h.chat.GetDiagnosisReport(c)
}

func newAIHandlerTestDB(t *testing.T) *gorm.DB {
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
		&ai.AIDiagnosisReport{},
		&ai.AIScenePrompt{},
		&ai.AISceneConfig{},
	); err != nil {
		t.Fatalf("auto migrate ai handler tables: %v", err)
	}
	return db
}
