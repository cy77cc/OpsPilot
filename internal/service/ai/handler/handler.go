// Package handler 实现 AI 模块的 HTTP 处理器。
//
// 提供以下接口:
//   - POST /ai/chat        - SSE 流式对话
//   - GET  /ai/sessions    - 列出会话
//   - POST /ai/sessions    - 创建会话
//   - GET  /ai/sessions/:id - 获取会话详情
//   - DELETE /ai/sessions/:id - 删除会话
//   - GET  /ai/runs/:runId - 获取运行状态
//   - GET  /ai/diagnosis/:reportId - 获取诊断报告
package handler

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/adk"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/service/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

// Handler 封装 AI 模块的 HTTP 处理逻辑。
type Handler struct {
	svcCtx *svc.ServiceContext
	logic  *logic.Logic

	workerMu     sync.Mutex
	workerStart  bool
	workerCancel context.CancelFunc
}

// NewAIHandler 创建 Handler 实例。
func NewAIHandler(svcCtx *svc.ServiceContext) *Handler {
	l := logic.NewAILogic(svcCtx)
	return &Handler{
		svcCtx: svcCtx,
		logic:  l,
	}
}

// NewAIHandlerWithDB 创建用于测试的 Handler 实例。
//
// 注意: 此构造函数仅用于测试，不会初始化 AIRouter。
// 生产环境请使用 NewAIHandler。
func NewAIHandlerWithDB(db *gorm.DB) *Handler {
	return &Handler{
		svcCtx: &svc.ServiceContext{DB: db},
		logic: &logic.Logic{
			ChatDAO:            aidao.NewAIChatDAO(db),
			RunDAO:             aidao.NewAIRunDAO(db),
			DiagnosisReportDAO: aidao.NewAIDiagnosisReportDAO(db),
			ApprovalDAO:        aidao.NewAIApprovalTaskDAO(db),
			RunEventDAO:        aidao.NewAIRunEventDAO(db),
			RunProjectionDAO:   aidao.NewAIRunProjectionDAO(db),
			RunContentDAO:      aidao.NewAIRunContentDAO(db),
			AIRouter:           &noopAgent{},
		},
	}
}

// noopAgent 是一个空操作的 Agent，用于测试。
type noopAgent struct{}

func (n *noopAgent) Name(_ context.Context) string        { return "NoopAgent" }
func (n *noopAgent) Description(_ context.Context) string { return "No-op agent for testing" }
func (n *noopAgent) Run(_ context.Context, _ *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Close()
	return iter
}
func (n *noopAgent) Resume(_ context.Context, _ *adk.ResumeInfo, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Close()
	return iter
}
