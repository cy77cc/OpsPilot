package logic

import (
	"github.com/cloudwego/eino/adk"
	aidaoapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/approval"
	aidaochat "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/chat"
	aidaodiagnosis "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/diagnosis"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic/event"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// Deps defines the dependencies required by Logic.
type Deps struct {
	ServiceContext     *svc.ServiceContext
	ChatDAO            *aidaochat.AIChatDAO
	RunDAO             *aidao.AIRunDAO
	DiagnosisReportDAO *aidaodiagnosis.AIDiagnosisReportDAO
	ApprovalDAO        *aidaoapproval.AIApprovalTaskDAO
	RunEventDAO        *aidao.AIRunEventDAO
	RunProjectionDAO   *aidao.AIRunProjectionDAO
	RunContentDAO      *aidao.AIRunContentDAO
	CheckpointStore    adk.CheckPointStore
	AIRouter           adk.ResumableAgent
	MigrationFlags     event.ApprovalEventMigrationFlags
}

// New creates a Logic instance from explicit dependencies.
func New(deps Deps) *Logic {
	return &Logic{
		svcCtx:             deps.ServiceContext,
		ChatDAO:            deps.ChatDAO,
		RunDAO:             deps.RunDAO,
		DiagnosisReportDAO: deps.DiagnosisReportDAO,
		ApprovalDAO:        deps.ApprovalDAO,
		RunEventDAO:        deps.RunEventDAO,
		RunProjectionDAO:   deps.RunProjectionDAO,
		RunContentDAO:      deps.RunContentDAO,
		CheckpointStore:    deps.CheckpointStore,
		AIRouter:           deps.AIRouter,
		MigrationFlags:     deps.MigrationFlags,
	}
}
