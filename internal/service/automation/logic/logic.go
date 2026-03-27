// Package logic 提供自动化运维服务的业务逻辑层。
//
// 本包实现自动化运维的核心业务逻辑，包括：
//   - 清单管理：主机清单的创建和查询
//   - Playbook 管理：自动化脚本的创建和查询
//   - 运行管理：任务预览、执行、状态查询和日志管理
//
// 执行流程：
//  1. PreviewRun 生成预览令牌和风险评估
//  2. ExecuteRun 校验审批令牌，解析主机范围，执行任务
//  3. 记录执行日志和审计信息
//
// 主机范围解析：
//   - 支持 host_ids 和 node_ids 参数
//   - 自动过滤不符合运行条件的主机
//   - 记录跳过原因
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

// Logic 是自动化运维服务的业务逻辑层。
//
// 封装数据库操作和业务规则，为 Handler 层提供服务。
type Logic struct {
	svcCtx *svc.ServiceContext // 服务上下文
}

// NewLogic 创建自动化运维业务逻辑实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库、配置等依赖
//
// 返回: 初始化后的 Logic 实例
func NewLogic(svcCtx *svc.ServiceContext) *Logic {
	return &Logic{svcCtx: svcCtx}
}

// ListInventories 获取所有主机清单。
//
// 参数:
//   - ctx: 上下文
//
// 返回: 清单列表和可能的错误
func (l *Logic) ListInventories(ctx context.Context) ([]model.AutomationInventory, error) {
	rows := make([]model.AutomationInventory, 0, 32)
	err := l.svcCtx.DB.WithContext(ctx).Order("id desc").Find(&rows).Error
	return rows, err
}

// CreateInventory 创建主机清单。
//
// 参数:
//   - ctx: 上下文
//   - actor: 创建者用户 ID
//   - req: 创建请求参数
//
// 返回: 创建的清单实例和可能的错误
//
// 业务规则:
//   - 名称不能为空
//   - 自动记录创建者和时间
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

// ListPlaybooks 获取所有 Playbook。
//
// 参数:
//   - ctx: 上下文
//
// 返回: Playbook 列表和可能的错误
func (l *Logic) ListPlaybooks(ctx context.Context) ([]model.AutomationPlaybook, error) {
	rows := make([]model.AutomationPlaybook, 0, 32)
	err := l.svcCtx.DB.WithContext(ctx).Order("id desc").Find(&rows).Error
	return rows, err
}

// CreatePlaybook 创建 Playbook。
//
// 参数:
//   - ctx: 上下文
//   - actor: 创建者用户 ID
//   - req: 创建请求参数
//
// 返回: 创建的 Playbook 实例和可能的错误
//
// 业务规则:
//   - 名称不能为空
//   - 风险等级默认为 medium
//   - 自动记录创建者和时间
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

// PreviewRun 预览自动化任务执行。
//
// 参数:
//   - ctx: 上下文
//   - req: 预览请求参数
//
// 返回: 预览结果（包含预览令牌、风险等级等）和可能的错误
//
// 业务规则:
//   - action 参数不能为空
//   - 生成唯一的预览令牌用于后续执行
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

// ExecuteRun 执行自动化任务。
//
// 参数:
//   - ctx: 上下文
//   - actor: 执行者用户 ID
//   - req: 执行请求参数
//
// 返回: 执行记录和可能的错误
//
// 业务流程:
//  1. 校验审批令牌
//  2. 创建执行记录，状态为 running
//  3. 解析主机范围，过滤不符合运行条件的主机
//  4. 记录执行日志
//  5. 更新执行状态为 succeeded 或 failed
//  6. 创建审计记录
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

// resolveAutomationHostScope 解析并过滤自动化任务的主机范围。
//
// 参数:
//   - ctx: 上下文
//   - params: 执行参数，包含 host_ids 或 node_ids
//
// 返回: 允许执行的主机 ID 列表、跳过原因列表和可能的错误
//
// 业务规则:
//   - 优先使用 host_ids 参数，其次使用 node_ids
//   - 主机必须存在且满足运行条件（通过 EvaluateOperationalEligibility 校验）
//   - 不符合条件的主机会被跳过，并记录原因
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

// parseHostIDs 从任意类型解析主机 ID 列表。
//
// 参数:
//   - v: 输入值，支持 []uint64、[]int、[]any 等类型
//
// 返回: 解析后的主机 ID 列表
//
// 支持的类型:
//   - []uint64: 直接返回副本
//   - []int: 转换为 uint64
//   - []any: 逐元素解析（支持 float64、int、uint64、string）
//   - 其他: 返回 nil
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

// GetRun 获取任务执行详情。
//
// 参数:
//   - ctx: 上下文
//   - id: 任务 ID
//
// 返回: 执行记录和可能的错误
func (l *Logic) GetRun(ctx context.Context, id string) (*model.AutomationRun, error) {
	var row model.AutomationRun
	err := l.svcCtx.DB.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ListRunLogs 获取任务执行日志列表。
//
// 参数:
//   - ctx: 上下文
//   - id: 任务 ID
//
// 返回: 日志列表和可能的错误
func (l *Logic) ListRunLogs(ctx context.Context, id string) ([]model.AutomationRunLog, error) {
	rows := make([]model.AutomationRunLog, 0, 32)
	err := l.svcCtx.DB.WithContext(ctx).Where("run_id = ?", strings.TrimSpace(id)).Order("id asc").Find(&rows).Error
	return rows, err
}
