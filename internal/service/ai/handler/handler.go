package handler

import (
	"github.com/cy77cc/OpsPilot/internal/service/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

type Decision = logic.Decision
type QARequest = logic.QARequest
type QAResult = logic.QAResult
type DiagnosisRequest = logic.DiagnosisRequest
type DiagnosisReport = logic.DiagnosisReport
type DiagnosisResult = logic.DiagnosisResult

const (
	IntentTypeQA        = logic.IntentTypeQA
	IntentTypeDiagnosis = logic.IntentTypeDiagnosis
	IntentTypeChange    = logic.IntentTypeChange
)

type Handler struct {
	svcCtx *svc.ServiceContext
	logic  *logic.Logic
	deps   Dependencies
}

func NewAIHandler(svcCtx *svc.ServiceContext) *Handler {
	l := logic.NewAILogic(svcCtx)
	return &Handler{
		svcCtx: svcCtx,
		logic:  l,
		deps: Dependencies{
			ChatDAO:            l.ChatDAO,
			RunDAO:             l.RunDAO,
			DiagnosisReportDAO: l.DiagnosisReportDAO,
			IntentRouter:       l.IntentRouter,
			QAAgent:            l.QAAgent,
			DiagnosisAgent:     l.DiagnosisAgent,
		},
	}
}

func New(deps Dependencies) *Handler {
	l := logic.NewWithDeps(deps.ChatDAO, deps.RunDAO, deps.DiagnosisReportDAO, deps.IntentRouter, deps.QAAgent, deps.DiagnosisAgent)
	return &Handler{logic: l, deps: deps}
}
