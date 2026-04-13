package bootstrap

import (
	"context"

	approvalhandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/approval"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// StartBackgroundProcessors wires AI background lifecycle responsibilities.
//
// HTTP route registration must remain transport-only; background processor
// startup belongs to module bootstrap/server assembly.
func StartBackgroundProcessors(ctx context.Context, svcCtx *svc.ServiceContext) {
	if svcCtx == nil {
		return
	}
	service := approvalhandler.NewService(svcCtx)
	service.StartWorker(ctx)
	service.StartExpirer(ctx)
}
