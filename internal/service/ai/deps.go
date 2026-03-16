package ai

import (
	"github.com/cy77cc/OpsPilot/internal/ai/agents/diagnosis"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/intent"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/qa"
	"github.com/cy77cc/OpsPilot/internal/dao"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

type Dependencies struct {
	ChatDAO            *dao.AIChatDAO
	RunDAO             *dao.AIRunDAO
	DiagnosisReportDAO *dao.AIDiagnosisReportDAO
	IntentRouter       *intent.Router
	QAAgent            *qa.Agent
	DiagnosisAgent     *diagnosis.Agent
}

func NewDependencies(svcCtx *svc.ServiceContext) Dependencies {
	if svcCtx == nil || svcCtx.DB == nil {
		return Dependencies{}
	}
	return Dependencies{
		ChatDAO:            dao.NewAIChatDAO(svcCtx.DB),
		RunDAO:             dao.NewAIRunDAO(svcCtx.DB),
		DiagnosisReportDAO: dao.NewAIDiagnosisReportDAO(svcCtx.DB),
		IntentRouter:       intent.NewRouter(),
		QAAgent:            qa.NewAgent(),
		DiagnosisAgent:     diagnosis.NewAgent(),
	}
}
