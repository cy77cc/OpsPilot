package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	hostlogic "github.com/cy77cc/OpsPilot/internal/service/host/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

type Logic struct {
	svcCtx *svc.ServiceContext
}

func NewLogic(svcCtx *svc.ServiceContext) *Logic {
	return &Logic{svcCtx: svcCtx}
}

func (l *Logic) ListInventories(ctx context.Context) ([]model.AutomationInventory, error) {
	rows := make([]model.AutomationInventory, 0, 32)
	err := l.svcCtx.DB.WithContext(ctx).Order("id desc").Find(&rows).Error
	return rows, err
}

func (l *Logic) CreateInventory(ctx context.Context, actor uint, req CreateInventoryReq) (*model.AutomationInventory, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	row := model.AutomationInventory{
		Name:      name,
		HostsJSON: strings.TrimSpace(req.HostsJSON),
		CreatedBy: actor,
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (l *Logic) ListPlaybooks(ctx context.Context) ([]model.AutomationPlaybook, error) {
	rows := make([]model.AutomationPlaybook, 0, 32)
	err := l.svcCtx.DB.WithContext(ctx).Order("id desc").Find(&rows).Error
	return rows, err
}

func (l *Logic) CreatePlaybook(ctx context.Context, actor uint, req CreatePlaybookReq) (*model.AutomationPlaybook, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	risk := strings.ToLower(strings.TrimSpace(req.RiskLevel))
	if risk == "" {
		risk = "medium"
	}
	row := model.AutomationPlaybook{
		Name:       name,
		ContentYML: strings.TrimSpace(req.ContentYML),
		RiskLevel:  risk,
		CreatedBy:  actor,
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (l *Logic) PreviewRun(ctx context.Context, req PreviewRunReq) (map[string]any, error) {
	action := strings.TrimSpace(req.Action)
	if action == "" {
		return nil, fmt.Errorf("action is required")
	}
	return map[string]any{
		"preview_token": fmt.Sprintf("preview-%d", time.Now().UnixNano()),
		"action":        action,
		"risk_level":    "medium",
		"params":        req.Params,
		"status":        "ready",
	}, nil
}

func (l *Logic) ExecuteRun(ctx context.Context, actor uint, req ExecuteRunReq) (*model.AutomationRun, error) {
	if strings.TrimSpace(req.ApprovalToken) == "" {
		return nil, fmt.Errorf("approval_token is required")
	}
	action := strings.TrimSpace(req.Action)
	if action == "" {
		action = "generic"
	}
	buf, _ := json.Marshal(req.Params)
	run := model.AutomationRun{
		ID:         fmt.Sprintf("run-%d", time.Now().UnixNano()),
		Action:     action,
		Status:     "running",
		ParamsJSON: string(buf),
		OperatorID: actor,
		StartedAt:  time.Now(),
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&run).Error; err != nil {
		return nil, err
	}

	hostScope, skippedReasons, err := l.resolveAutomationHostScope(ctx, req.Params)
	if err != nil {
		run.Status = "failed"
		run.ResultJSON = fmt.Sprintf(`{"error":%q}`, err.Error())
		run.FinishedAt = time.Now()
		_ = l.svcCtx.DB.WithContext(ctx).Model(&model.AutomationRun{}).
			Where("id = ?", run.ID).
			Updates(map[string]any{
				"status":      run.Status,
				"result_json": run.ResultJSON,
				"finished_at": run.FinishedAt,
			}).Error
		return &run, err
	}

	_ = l.svcCtx.DB.WithContext(ctx).Create(&model.AutomationRunLog{
		RunID:   run.ID,
		Level:   "info",
		Message: "run queued and started",
	}).Error
	for _, reason := range skippedReasons {
		_ = l.svcCtx.DB.WithContext(ctx).Create(&model.AutomationRunLog{
			RunID:   run.ID,
			Level:   "warning",
			Message: reason,
		}).Error
	}
	if len(hostScope) == 0 {
		run.Status = "succeeded"
		run.ResultJSON = `{"summary":"no eligible hosts, skipped execution","executed_host_ids":[]}`
		run.FinishedAt = time.Now()
		_ = l.svcCtx.DB.WithContext(ctx).Model(&model.AutomationRun{}).
			Where("id = ?", run.ID).
			Updates(map[string]any{
				"status":      run.Status,
				"result_json": run.ResultJSON,
				"finished_at": run.FinishedAt,
			}).Error
		return &run, nil
	}

	run.Status = "succeeded"
	result, _ := json.Marshal(map[string]any{
		"summary":           "skeleton execution completed",
		"executed_host_ids": hostScope,
		"skipped_hosts":     skippedReasons,
	})
	run.ResultJSON = string(result)
	run.FinishedAt = time.Now()
	_ = l.svcCtx.DB.WithContext(ctx).Model(&model.AutomationRun{}).
		Where("id = ?", run.ID).
		Updates(map[string]any{
			"status":      run.Status,
			"result_json": run.ResultJSON,
			"finished_at": run.FinishedAt,
		}).Error
	_ = l.svcCtx.DB.WithContext(ctx).Create(&model.AutomationRunLog{
		RunID:   run.ID,
		Level:   "info",
		Message: "run finished",
	}).Error

	detail, _ := json.Marshal(map[string]any{
		"approval_token": strings.TrimSpace(req.ApprovalToken),
		"params":         req.Params,
	})
	_ = l.svcCtx.DB.WithContext(ctx).Create(&model.AutomationExecutionAudit{
		RunID:      run.ID,
		Action:     run.Action,
		Status:     run.Status,
		ActorID:    actor,
		DetailJSON: string(detail),
	}).Error
	return &run, nil
}

func (l *Logic) resolveAutomationHostScope(ctx context.Context, params map[string]any) ([]uint64, []string, error) {
	if len(params) == 0 {
		return nil, nil, nil
	}
	candidates := parseHostIDs(params["host_ids"])
	if len(candidates) == 0 {
		candidates = parseHostIDs(params["node_ids"])
	}
	if len(candidates) == 0 {
		return nil, nil, nil
	}
	allowed := make([]uint64, 0, len(candidates))
	skipped := make([]string, 0)
	for _, hostID := range candidates {
		var host model.Node
		if err := l.svcCtx.DB.WithContext(ctx).First(&host, hostID).Error; err != nil {
			skipped = append(skipped, fmt.Sprintf("host %d skipped: not found", hostID))
			continue
		}
		if ok, reason := hostlogic.EvaluateOperationalEligibility(&host); !ok {
			skipped = append(skipped, fmt.Sprintf("host %d skipped: %s", hostID, reason))
			continue
		}
		allowed = append(allowed, hostID)
	}
	return allowed, skipped, nil
}

func parseHostIDs(v any) []uint64 {
	switch x := v.(type) {
	case []uint64:
		return append([]uint64{}, x...)
	case []int:
		out := make([]uint64, 0, len(x))
		for _, id := range x {
			if id > 0 {
				out = append(out, uint64(id))
			}
		}
		return out
	case []any:
		out := make([]uint64, 0, len(x))
		for _, item := range x {
			switch v := item.(type) {
			case float64:
				if v > 0 {
					out = append(out, uint64(v))
				}
			case int:
				if v > 0 {
					out = append(out, uint64(v))
				}
			case uint64:
				if v > 0 {
					out = append(out, v)
				}
			case string:
				n, _ := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
				if n > 0 {
					out = append(out, n)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func (l *Logic) GetRun(ctx context.Context, id string) (*model.AutomationRun, error) {
	var row model.AutomationRun
	err := l.svcCtx.DB.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (l *Logic) ListRunLogs(ctx context.Context, id string) ([]model.AutomationRunLog, error) {
	rows := make([]model.AutomationRunLog, 0, 32)
	err := l.svcCtx.DB.WithContext(ctx).Where("run_id = ?", strings.TrimSpace(id)).Order("id asc").Find(&rows).Error
	return rows, err
}
