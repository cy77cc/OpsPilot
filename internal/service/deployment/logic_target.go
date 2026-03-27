// Package deployment 提供部署目标管理的业务逻辑实现。
//
// 本文件包含部署目标的 CRUD 操作和节点管理逻辑。
package deployment

import (
	"context"
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/model"
	hostlogic "github.com/cy77cc/OpsPilot/internal/service/host/logic"
	"gorm.io/gorm"
)

// ListTargets 获取部署目标列表。
//
// 参数:
//   - ctx: 上下文
//   - projectID: 项目 ID (可选筛选)
//   - teamID: 团队 ID (可选筛选)
//
// 返回: 目标响应列表
func (l *Logic) ListTargets(ctx context.Context, projectID, teamID uint) ([]TargetResp, error) {
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.DeploymentTarget{})
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	if teamID > 0 {
		q = q.Where("team_id = ?", teamID)
	}
	var rows []model.DeploymentTarget
	if err := q.Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]TargetResp, 0, len(rows))
	for i := range rows {
		item, err := l.GetTarget(ctx, rows[i].ID)
		if err != nil {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

// GetTarget 获取部署目标详情。
//
// 参数:
//   - ctx: 上下文
//   - id: 目标 ID
//
// 返回: 目标响应
func (l *Logic) GetTarget(ctx context.Context, id uint) (TargetResp, error) {
	var row model.DeploymentTarget
	if err := l.svcCtx.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return TargetResp{}, err
	}
	resp := TargetResp{
		ID:              row.ID,
		Name:            row.Name,
		TargetType:      row.TargetType,
		RuntimeType:     defaultIfEmpty(row.RuntimeType, row.TargetType),
		ClusterID:       row.ClusterID,
		ClusterSource:   l.compatClusterSource(row.ClusterSource, row.ClusterID, row.CredentialID),
		CredentialID:    row.CredentialID,
		BootstrapJobID:  row.BootstrapJobID,
		ProjectID:       row.ProjectID,
		TeamID:          row.TeamID,
		Env:             row.Env,
		Status:          row.Status,
		ReadinessStatus: defaultIfEmpty(row.ReadinessStatus, "unknown"),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	var nodes []model.DeploymentTargetNode
	if err := l.svcCtx.DB.WithContext(ctx).Where("target_id = ?", row.ID).Find(&nodes).Error; err == nil {
		resp.Nodes = make([]TargetNodeResp, 0, len(nodes))
		for _, n := range nodes {
			item := TargetNodeResp{HostID: n.HostID, Role: n.Role, Weight: n.Weight, Status: n.Status}
			var host model.Node
			if err := l.svcCtx.DB.WithContext(ctx).First(&host, n.HostID).Error; err == nil {
				item.Name = host.Name
				item.IP = host.IP
				item.Status = host.Status
			}
			resp.Nodes = append(resp.Nodes, item)
		}
	}
	return resp, nil
}

// CreateTarget 创建部署目标。
//
// 参数:
//   - ctx: 上下文
//   - uid: 用户 ID
//   - req: 创建请求
//
// 返回: 创建的目标响应
func (l *Logic) CreateTarget(ctx context.Context, uid uint64, req TargetUpsertReq) (TargetResp, error) {
	runtimeType := normalizedRuntime(req.TargetType, req.RuntimeType)
	row := model.DeploymentTarget{
		Name:            strings.TrimSpace(req.Name),
		TargetType:      runtimeType,
		RuntimeType:     runtimeType,
		ClusterID:       req.ClusterID,
		ClusterSource:   l.compatClusterSource(strings.TrimSpace(req.ClusterSource), req.ClusterID, req.CredentialID),
		CredentialID:    req.CredentialID,
		BootstrapJobID:  strings.TrimSpace(req.BootstrapJobID),
		ProjectID:       req.ProjectID,
		TeamID:          req.TeamID,
		Env:             defaultIfEmpty(req.Env, "staging"),
		Status:          "active",
		ReadinessStatus: "unknown",
		CreatedBy:       uint(uid),
	}
	if strings.TrimSpace(row.BootstrapJobID) != "" {
		var job model.EnvironmentInstallJob
		if err := l.svcCtx.DB.WithContext(ctx).Select("id,status").Where("id = ?", row.BootstrapJobID).First(&job).Error; err != nil {
			return TargetResp{}, fmt.Errorf("bootstrap job not found")
		}
		if strings.EqualFold(strings.TrimSpace(job.Status), "succeeded") {
			row.ReadinessStatus = "ready"
		} else {
			row.ReadinessStatus = "bootstrap_pending"
		}
	}
	if row.TargetType != "k8s" && row.TargetType != "compose" {
		return TargetResp{}, fmt.Errorf("unsupported target_type: %s", row.TargetType)
	}
	if err := l.validateTargetUpsert(ctx, row.TargetType, row.ClusterID, row.ClusterSource, row.CredentialID, req.Nodes); err != nil {
		return TargetResp{}, err
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&row).Error; err != nil {
		return TargetResp{}, err
	}
	if len(req.Nodes) > 0 {
		if err := l.ReplaceTargetNodes(ctx, row.ID, req.Nodes); err != nil {
			return TargetResp{}, err
		}
	}
	return l.GetTarget(ctx, row.ID)
}

// UpdateTarget 更新部署目标。
//
// 参数:
//   - ctx: 上下文
//   - id: 目标 ID
//   - req: 更新请求
//
// 返回: 更新后的目标响应
func (l *Logic) UpdateTarget(ctx context.Context, id uint, req TargetUpsertReq) (TargetResp, error) {
	var row model.DeploymentTarget
	if err := l.svcCtx.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return TargetResp{}, err
	}
	if strings.TrimSpace(req.Name) != "" {
		row.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.TargetType) != "" {
		row.TargetType = normalizedRuntime(req.TargetType, req.RuntimeType)
		row.RuntimeType = row.TargetType
	} else if strings.TrimSpace(req.RuntimeType) != "" {
		row.RuntimeType = normalizedRuntime(row.TargetType, req.RuntimeType)
		row.TargetType = row.RuntimeType
	}
	if req.ClusterID > 0 || row.TargetType == "k8s" {
		row.ClusterID = req.ClusterID
	}
	if strings.TrimSpace(req.ClusterSource) != "" {
		row.ClusterSource = strings.TrimSpace(req.ClusterSource)
	}
	if req.CredentialID > 0 || row.TargetType == "k8s" {
		row.CredentialID = req.CredentialID
	}
	if strings.TrimSpace(req.BootstrapJobID) != "" {
		row.BootstrapJobID = strings.TrimSpace(req.BootstrapJobID)
		var job model.EnvironmentInstallJob
		if err := l.svcCtx.DB.WithContext(ctx).Select("id,status").Where("id = ?", row.BootstrapJobID).First(&job).Error; err != nil {
			return TargetResp{}, fmt.Errorf("bootstrap job not found")
		}
		if strings.EqualFold(strings.TrimSpace(job.Status), "succeeded") {
			row.ReadinessStatus = "ready"
		} else {
			row.ReadinessStatus = "bootstrap_pending"
		}
	}
	if req.ProjectID > 0 {
		row.ProjectID = req.ProjectID
	}
	if req.TeamID > 0 {
		row.TeamID = req.TeamID
	}
	if strings.TrimSpace(req.Env) != "" {
		row.Env = req.Env
	}
	row.ClusterSource = l.compatClusterSource(row.ClusterSource, row.ClusterID, row.CredentialID)
	if err := l.validateTargetUpsert(ctx, row.TargetType, row.ClusterID, row.ClusterSource, row.CredentialID, req.Nodes); err != nil {
		return TargetResp{}, err
	}
	if err := l.svcCtx.DB.WithContext(ctx).Save(&row).Error; err != nil {
		return TargetResp{}, err
	}
	if req.Nodes != nil {
		if err := l.ReplaceTargetNodes(ctx, row.ID, req.Nodes); err != nil {
			return TargetResp{}, err
		}
	}
	return l.GetTarget(ctx, row.ID)
}

// DeleteTarget 删除部署目标。
//
// 参数:
//   - ctx: 上下文
//   - id: 目标 ID
//
// 返回: 错误信息
func (l *Logic) DeleteTarget(ctx context.Context, id uint) error {
	return l.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("target_id = ?", id).Delete(&model.DeploymentTargetNode{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.DeploymentTarget{}, id).Error
	})
}

// ReplaceTargetNodes 替换部署目标的节点列表。
//
// 参数:
//   - ctx: 上下文
//   - targetID: 目标 ID
//   - nodes: 节点请求列表
//
// 返回: 错误信息
func (l *Logic) ReplaceTargetNodes(ctx context.Context, targetID uint, nodes []TargetNodeReq) error {
	var target model.DeploymentTarget
	if err := l.svcCtx.DB.WithContext(ctx).Select("id,target_type").First(&target, targetID).Error; err != nil {
		return err
	}
	if target.TargetType == "compose" && len(nodes) == 0 {
		return fmt.Errorf("compose target requires at least one host node")
	}
	return l.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("target_id = ?", targetID).Delete(&model.DeploymentTargetNode{}).Error; err != nil {
			return err
		}
		for _, n := range nodes {
			if n.HostID == 0 {
				continue
			}
			var host model.Node
			if err := tx.Select("id,ip,status").First(&host, n.HostID).Error; err != nil {
				return fmt.Errorf("host node %d not found", n.HostID)
			}
			if target.TargetType == "compose" {
				if ok, reason := hostlogic.EvaluateOperationalEligibility(&host); !ok {
					return fmt.Errorf("host node %d is unavailable: %s", n.HostID, reason)
				}
			}
			row := model.DeploymentTargetNode{TargetID: targetID, HostID: n.HostID, Role: defaultIfEmpty(n.Role, "worker"), Weight: defaultInt(n.Weight, 100), Status: "active"}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// validateTargetUpsert 验证部署目标的创建/更新参数。
//
// 参数:
//   - ctx: 上下文
//   - targetType: 目标类型
//   - clusterID: 集群 ID
//   - clusterSource: 集群来源
//   - credentialID: 凭证 ID
//   - nodes: 节点列表
//
// 返回: 验证错误
func (l *Logic) validateTargetUpsert(ctx context.Context, targetType string, clusterID uint, clusterSource string, credentialID uint, nodes []TargetNodeReq) error {
	switch targetType {
	case "k8s":
		if clusterID == 0 && credentialID == 0 {
			return fmt.Errorf("cluster_id or credential_id is required for k8s target")
		}
		if clusterID > 0 {
			var cluster model.Cluster
			if err := l.svcCtx.DB.WithContext(ctx).Select("id,status").First(&cluster, clusterID).Error; err != nil {
				return fmt.Errorf("cluster binding not found: %w", err)
			}
		}
		if credentialID > 0 {
			var cred model.ClusterCredential
			if err := l.svcCtx.DB.WithContext(ctx).Select("id,runtime_type,status").First(&cred, credentialID).Error; err != nil {
				return fmt.Errorf("cluster credential not found: %w", err)
			}
			if !strings.EqualFold(strings.TrimSpace(cred.Status), "active") {
				return fmt.Errorf("cluster credential is not active")
			}
		}
		if clusterSource != "" && clusterSource != "platform_managed" && clusterSource != "external_managed" {
			return fmt.Errorf("unsupported cluster_source: %s", clusterSource)
		}
		return nil
	case "compose":
		if clusterID != 0 {
			return fmt.Errorf("compose target must not bind cluster_id")
		}
		if credentialID != 0 {
			return fmt.Errorf("compose target must not bind credential_id")
		}
		if nodes != nil && len(nodes) == 0 {
			return fmt.Errorf("compose target requires at least one host node")
		}
		return nil
	default:
		return fmt.Errorf("unsupported target_type: %s", targetType)
	}
}

// compatClusterSource 兼容处理集群来源字段。
//
// 参数:
//   - clusterSource: 原始集群来源
//   - clusterID: 集群 ID
//   - credentialID: 凭证 ID
//
// 返回: 兼容后的集群来源
func (l *Logic) compatClusterSource(clusterSource string, clusterID, credentialID uint) string {
	source := strings.TrimSpace(clusterSource)
	if source != "" {
		return source
	}
	if credentialID > 0 {
		return "external_managed"
	}
	if clusterID > 0 {
		return "platform_managed"
	}
	return "platform_managed"
}

// normalizedRuntime 规范化运行时类型。
//
// 参数:
//   - targetType: 目标类型
//   - runtimeType: 运行时类型
//
// 返回: 规范化后的运行时类型
func normalizedRuntime(targetType, runtimeType string) string {
	target := strings.TrimSpace(targetType)
	if target == "" {
		target = strings.TrimSpace(runtimeType)
	}
	if target == "" {
		return "k8s"
	}
	return target
}
