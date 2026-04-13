package chat

import (
	"github.com/cloudwego/eino/adk"
	chathandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/chat"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

type Service = chathandler.Service
type HTTPHandler = chathandler.HTTPHandler
type RouteMode = chathandler.RouteMode
type TaskAction = chathandler.TaskAction
type RunAction = chathandler.RunAction
type ExecutionShape = chathandler.ExecutionShape
type Domain = chathandler.Domain
type ToolStrategy = chathandler.ToolStrategy
type ContextPlan = chathandler.ContextPlan
type RouteDecision = chathandler.RouteDecision
type RouteInput = chathandler.RouteInput
type RouteService = chathandler.RouteService

const (
	ModeConversation                  = chathandler.ModeConversation
	ModeTask                          = chathandler.ModeTask
	TaskActionCreateTask              = chathandler.TaskActionCreateTask
	TaskActionNone                    = chathandler.TaskActionNone
	RunActionCreateRun                = chathandler.RunActionCreateRun
	RunActionNone                     = chathandler.RunActionNone
	ExecutionShapeSingleAgent         = chathandler.ExecutionShapeSingleAgent
	ExecutionShapeDelegatedSpecialist = chathandler.ExecutionShapeDelegatedSpecialist
	DomainGeneral                     = chathandler.DomainGeneral
	ToolStrategyDirect                = chathandler.ToolStrategyDirect
)

var (
	NewHTTPHandler  = chathandler.NewHTTPHandler
	NewRouteService = chathandler.NewRouteService
)

func NewService(svcCtx *svc.ServiceContext) *Service {
	return chathandler.NewService(svcCtx)
}

func NewServiceWithLogic(l *logic.Logic) *Service {
	return chathandler.NewServiceWithLogic(l)
}

func NewServiceWithLogicAndRouter(l *logic.Logic, routeSvc RouteService) *Service {
	return chathandler.NewServiceWithLogicAndRouter(l, routeSvc)
}

func NewServiceWithDB(db *gorm.DB, agentRouter adk.ResumableAgent) *Service {
	return chathandler.NewServiceWithDB(db, agentRouter)
}
