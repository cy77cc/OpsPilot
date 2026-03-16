package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

type Logic struct {
	svcCtx *svc.ServiceContext
}

func NewLogic(svcCtx *svc.ServiceContext) *Logic {
	return &Logic{svcCtx: svcCtx}
}

func (l *Logic) GetServiceTopology(ctx context.Context, serviceID uint) (GraphResponse, error) {
	out := GraphResponse{
		Nodes: make([]GraphNode, 0, 8),
		Edges: make([]GraphEdge, 0, 8),
	}
	var svc model.Service
	if err := l.svcCtx.DB.WithContext(ctx).Where("id = ?", serviceID).First(&svc).Error; err != nil {
		return out, err
	}
	serviceNodeID := fmt.Sprintf("service:%d", svc.ID)
	out.Nodes = append(out.Nodes, GraphNode{
		ID:   serviceNodeID,
		Type: "service",
		Name: svc.Name,
		Metadata: map[string]any{
			"project_id": svc.ProjectID,
		},
	})

	targets := make([]model.DeploymentTarget, 0, 8)
	_ = l.svcCtx.DB.WithContext(ctx).
		Joins("JOIN deployment_releases ON deployment_releases.target_id = deployment_targets.id").
		Where("deployment_releases.service_id = ?", serviceID).
		Group("deployment_targets.id").
		Find(&targets).Error

	for _, t := range targets {
		targetNodeID := fmt.Sprintf("target:%d", t.ID)
		out.Nodes = append(out.Nodes, GraphNode{
			ID:   targetNodeID,
			Type: "deployment_target",
			Name: t.Name,
			Metadata: map[string]any{
				"runtime_type": t.RuntimeType,
				"cluster_id":   t.ClusterID,
				"env":          t.Env,
			},
		})
		out.Edges = append(out.Edges, GraphEdge{
			ID:   fmt.Sprintf("edge:%s->%s", serviceNodeID, targetNodeID),
			From: serviceNodeID,
			To:   targetNodeID,
			Type: "deploys_to",
		})
	}
	return out, nil
}

func (l *Logic) GetHostServices(ctx context.Context, hostID uint) ([]map[string]any, error) {
	type row struct {
		ServiceID   uint   `gorm:"column:service_id"`
		TargetID    uint   `gorm:"column:target_id"`
		RuntimeType string `gorm:"column:runtime_type"`
	}
	rows := make([]row, 0, 16)
	err := l.svcCtx.DB.WithContext(ctx).
		Table("deployment_target_nodes").
		Select("deployment_releases.service_id, deployment_releases.target_id, deployment_releases.runtime_type").
		Joins("JOIN deployment_releases ON deployment_releases.target_id = deployment_target_nodes.target_id").
		Where("deployment_target_nodes.host_id = ?", hostID).
		Group("deployment_releases.service_id, deployment_releases.target_id, deployment_releases.runtime_type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"service_id":   r.ServiceID,
			"target_id":    r.TargetID,
			"runtime_type": r.RuntimeType,
		})
	}
	return out, nil
}

func (l *Logic) GetClusterServices(ctx context.Context, clusterID uint) ([]map[string]any, error) {
	type row struct {
		ServiceID   uint   `gorm:"column:service_id"`
		TargetID    uint   `gorm:"column:target_id"`
		RuntimeType string `gorm:"column:runtime_type"`
	}
	rows := make([]row, 0, 16)
	err := l.svcCtx.DB.WithContext(ctx).
		Table("deployment_targets").
		Select("deployment_releases.service_id, deployment_releases.target_id, deployment_releases.runtime_type").
		Joins("JOIN deployment_releases ON deployment_releases.target_id = deployment_targets.id").
		Where("deployment_targets.cluster_id = ?", clusterID).
		Group("deployment_releases.service_id, deployment_releases.target_id, deployment_releases.runtime_type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"service_id":   r.ServiceID,
			"target_id":    r.TargetID,
			"runtime_type": r.RuntimeType,
		})
	}
	return out, nil
}

func (l *Logic) QueryGraph(ctx context.Context, filter QueryFilter) (GraphResponse, error) {
	out := GraphResponse{
		Nodes: make([]GraphNode, 0, 64),
		Edges: make([]GraphEdge, 0, 64),
	}
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.CMDBCI{})
	if filter.ProjectID > 0 {
		q = q.Where("project_id = ?", filter.ProjectID)
	}
	if filter.ResourceType != "" {
		q = q.Where("ci_type = ?", filter.ResourceType)
	}
	if filter.Keyword != "" {
		kw := "%" + strings.TrimSpace(filter.Keyword) + "%"
		q = q.Where("name LIKE ?", kw)
	}
	cis := make([]model.CMDBCI, 0, 64)
	if err := q.Order("id asc").Find(&cis).Error; err != nil {
		return out, err
	}
	allowCI := make(map[uint]struct{}, len(cis))
	for _, ci := range cis {
		allowCI[ci.ID] = struct{}{}
		out.Nodes = append(out.Nodes, GraphNode{
			ID:   fmt.Sprintf("ci:%d", ci.ID),
			Type: ci.CIType,
			Name: ci.Name,
			Metadata: map[string]any{
				"project_id": ci.ProjectID,
				"team_id":    ci.TeamID,
				"status":     ci.Status,
			},
		})
	}
	rels := make([]model.CMDBRelation, 0, 128)
	if err := l.svcCtx.DB.WithContext(ctx).Order("id asc").Find(&rels).Error; err != nil {
		return out, err
	}
	for _, rel := range rels {
		if _, ok := allowCI[rel.FromCIID]; !ok {
			continue
		}
		if _, ok := allowCI[rel.ToCIID]; !ok {
			continue
		}
		out.Edges = append(out.Edges, GraphEdge{
			ID:   fmt.Sprintf("rel:%d", rel.ID),
			From: fmt.Sprintf("ci:%d", rel.FromCIID),
			To:   fmt.Sprintf("ci:%d", rel.ToCIID),
			Type: rel.RelationType,
		})
	}
	if filter.ClusterID > 0 {
		filtered := out.Nodes[:0]
		for _, n := range out.Nodes {
			if n.Metadata["cluster_id"] == filter.ClusterID {
				filtered = append(filtered, n)
				continue
			}
			if n.Type == "cluster" && strings.HasSuffix(n.ID, fmt.Sprintf(":%d", filter.ClusterID)) {
				filtered = append(filtered, n)
			}
		}
		out.Nodes = filtered
	}
	return out, nil
}

func (l *Logic) WriteAccessAudit(ctx context.Context, actor uint, action, scope string, filter any) {
	buf, _ := json.Marshal(filter)
	_ = l.svcCtx.DB.WithContext(ctx).Create(&model.TopologyAccessAudit{
		ActorID:    actor,
		Action:     action,
		Scope:      scope,
		FilterJSON: string(buf),
	}).Error
}
