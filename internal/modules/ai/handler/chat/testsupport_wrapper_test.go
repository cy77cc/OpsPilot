package chathandler_test

import (
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/testsupport"
	"gorm.io/gorm"
)

type aiHandlerTestHarness struct {
	*testsupport.HandlerHarness
	logic *logic.Logic
}

func newAIHandlerTestHarness(db *gorm.DB) *aiHandlerTestHarness {
	h := testsupport.NewAIHandlerTestHarness(db)
	return &aiHandlerTestHarness{HandlerHarness: h, logic: h.Logic}
}
