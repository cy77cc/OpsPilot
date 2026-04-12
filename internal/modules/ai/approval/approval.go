package approval

import (
	approvalapp "github.com/cy77cc/OpsPilot/internal/modules/ai/approval/app"
	approvalhttp "github.com/cy77cc/OpsPilot/internal/modules/ai/approval/http"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

type Service = approvalapp.Service
type HTTPHandler = approvalhttp.HTTPHandler

var NewService = approvalapp.NewService
var NewServiceWithLogic = approvalapp.NewServiceWithLogic
var NewHTTPHandler = approvalhttp.NewHTTPHandler

var _ = logic.SubmitApprovalInput{}
var _ = svc.ServiceContext{}
