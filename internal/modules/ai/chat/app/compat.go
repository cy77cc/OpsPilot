package app

import chathandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/chat"

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

var NewRouteService = chathandler.NewRouteService
