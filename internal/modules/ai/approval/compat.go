package approval

import (
	approvalhandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/approval"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

type Service = approvalhandler.Service
type HTTPHandler = approvalhandler.HTTPHandler

var NewHTTPHandler = approvalhandler.NewHTTPHandler

func NewService(svcCtx *svc.ServiceContext) *Service {
	return approvalhandler.NewService(svcCtx)
}

func NewServiceWithLogic(l *logic.Logic) *Service {
	return approvalhandler.NewServiceWithLogic(l)
}
