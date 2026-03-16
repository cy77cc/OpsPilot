package ai

import (
	"github.com/cy77cc/OpsPilot/internal/dao"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// Deps centralizes the AI service dependencies that handlers can consume.
type Deps struct {
	SvcCtx       *svc.ServiceContext
	ChatDAO      *dao.AIChatDAO
	RunDAO       *dao.AIRunDAO
	DiagnosisDAO *dao.AIDiagnosisReportDAO
}

// NewDeps builds a dependency bag using the provided service context.
func NewDeps(svcCtx *svc.ServiceContext) *Deps {
	return &Deps{
		SvcCtx:       svcCtx,
		ChatDAO:      dao.NewAIChatDAO(svcCtx.DB),
		RunDAO:       dao.NewAIRunDAO(svcCtx.DB),
		DiagnosisDAO: dao.NewAIDiagnosisReportDAO(svcCtx.DB),
	}
}
