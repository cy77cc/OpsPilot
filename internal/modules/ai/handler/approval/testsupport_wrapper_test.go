package approvalhandler_test

import (
	approvalhandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/approval"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type aiHandlerTestHarness struct {
	approval *approvalhandler.HTTPHandler
	logic    *logic.Logic
}

func newAIHandlerTestHarness(db *gorm.DB) *aiHandlerTestHarness {
	l := logic.NewLogicWithDB(db, nil)
	approvalSvc := approvalhandler.NewServiceWithLogic(l)
	return &aiHandlerTestHarness{
		approval: approvalhandler.NewHTTPHandler(approvalSvc),
		logic:    l,
	}
}

func (h *aiHandlerTestHarness) SubmitApproval(c *gin.Context) {
	h.approval.SubmitApproval(c)
}

func (h *aiHandlerTestHarness) RetryResumeApproval(c *gin.Context) {
	h.approval.RetryResumeApproval(c)
}

func registerAIHandlersForTest(v1 *gin.RouterGroup) {
	h := newAIHandlerTestHarness(nil)
	g := v1.Group("/ai")
	{
		g.POST("/approvals/:id/submit", h.SubmitApproval)
		g.POST("/approvals/:id/retry-resume", h.RetryResumeApproval)
	}
}
