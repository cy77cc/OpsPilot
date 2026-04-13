package approvalhandler_test

import (
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/testsupport"
	"github.com/gin-gonic/gin"
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

func registerAIHandlersForTest(v1 *gin.RouterGroup) {
	testsupport.RegisterAIHandlersForTest(v1)
}
