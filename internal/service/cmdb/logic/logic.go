// Package logic 提供 CMDB 服务的业务逻辑层实现。
//
// 本文件实现 CMDB 的核心业务逻辑，包括：
//   - 资产 (CI) 的 CRUD 操作
//   - 关系管理
//   - 拓扑查询
//   - 数据同步任务
//   - 审计日志记录
package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

// Logic 是 CMDB 业务逻辑层的核心结构。
//
// 封装服务上下文，提供统一的业务处理入口。
type Logic struct {
	svcCtx *svc.ServiceContext
}

// NewLogic 创建 CMDB 业务逻辑实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接等依赖
//
// 返回: 初始化后的 Logic 实例
func NewLogic(svcCtx *svc.ServiceContext) *Logic { return &Logic{svcCtx: svcCtx} }

// CIFilter 是资产列表查询的过滤条件。
type CIFilter struct {
	Type      string // 资产类型
	Status    string // 资产状态
	Keyword   string // 搜索关键字
	ProjectID uint   // 项目ID
	TeamID    uint   // 团队ID
	Page      int    // 页码
	PageSize  int    // 每页数量
}

// syncSummary 是同步任务的统计摘要。
type syncSummary struct {
	Created   int `json:"created"`   // 新创建数量
	Updated   int `json:"updated"`   // 更新数量
	Unchanged int `json:"unchanged"` // 未变更数量
	Failed    int `json:"failed"`    // 失败数量
}

// discoveredCI 是从外部数据源发现的资产信息。
type discoveredCI struct {
	CIType     string // 资产类型
	Source     string // 数据来源
	ExternalID string // 外部系统ID
	Name       string // 资产名称
	Status     string // 资产状态
	ProjectID  uint   // 项目ID
	TeamID     uint   // 团队ID
	Owner      string // 负责人
	AttrsJSON  string // 属性 JSON
}

// normalizePage 规范化页码参数。
//
// 参数:
//   - v: 原始页码值
//   - def: 默认值
//
// 返回: 规范化后的页码，无效值返回默认值
func normalizePage(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

// normalizePageSize 规范化每页数量参数。
//
// 参数:
//   - v: 原始每页数量值
//   - def: 默认值
//
// 返回: 规范化后的每页数量，无效值返回默认值，最大值限制为 200
func normalizePageSize(v, def int) int {
	if v <= 0 {
		return def
	}
	if v > 200 {
		return 200
	}
	return v
}

// ciUID 生成资产的唯一标识符。
//
// 格式: "ciType:externalID"，用于唯一标识一个配置项。
//
// 参数:
//   - ciType: 资产类型
//   - externalID: 外部系统ID
//
// 返回: 格式化的唯一标识符
func ciUID(ciType, externalID string) string {
	return strings.TrimSpace(ciType) + ":" + strings.TrimSpace(externalID)
}

// ListCIs 分页查询资产列表。
//
// 支持按类型、状态、关键字、项目、团队筛选，返回分页结果。
//
// 参数:
//   - ctx: 上下文
//   - f: 过滤条件
//
// 返回: 资产列表、总数、错误信息
func (l *Logic) ListCIs(ctx context.Context, f CIFilter) ([]model.CMDBCI, int64, error) {
	page := normalizePage(f.Page, 1)
	pageSize := normalizePageSize(f.PageSize, 20)

	q := l.svcCtx.DB.WithContext(ctx).Model(&model.CMDBCI{})
	if strings.TrimSpace(f.Type) != "" {
		q = q.Where("ci_type = ?", strings.TrimSpace(f.Type))
	}
	if strings.TrimSpace(f.Status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(f.Status))
	}
	if strings.TrimSpace(f.Keyword) != "" {
		kw := "%" + strings.TrimSpace(f.Keyword) + "%"
		q = q.Where("name LIKE ? OR external_id LIKE ? OR ci_uid LIKE ?", kw, kw, kw)
	}
	if f.ProjectID > 0 {
		q = q.Where("project_id = ?", f.ProjectID)
	}
	if f.TeamID > 0 {
		q = q.Where("team_id = ?", f.TeamID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]model.CMDBCI, 0, pageSize)
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error
	return rows, total, err
}

// CreateCI 创建新资产。
//
// 自动生成 CIUID、设置默认值，并记录创建者信息。
//
// 参数:
//   - ctx: 上下文
//   - uid: 创建者用户ID
//   - in: 资产数据
//
// 返回: 创建成功的资产、错误信息
func (l *Logic) CreateCI(ctx context.Context, uid uint, in model.CMDBCI) (*model.CMDBCI, error) {
	in.CIType = strings.TrimSpace(in.CIType)
	in.Source = defaultIfEmpty(strings.TrimSpace(in.Source), "manual")
	in.Status = defaultIfEmpty(strings.TrimSpace(in.Status), "active")
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || in.CIType == "" {
		return nil, fmt.Errorf("ci_type and name are required")
	}
	if strings.TrimSpace(in.ExternalID) == "" {
		in.ExternalID = fmt.Sprintf("manual-%d", time.Now().UnixNano())
	}
	in.CIUID = ciUID(in.CIType, in.ExternalID)
	in.CreatedBy = uid
	in.UpdatedBy = uid
	if err := l.svcCtx.DB.WithContext(ctx).Create(&in).Error; err != nil {
		return nil, err
	}
	return &in, nil
}

// GetCI 根据ID获取资产详情。
//
// 参数:
//   - ctx: 上下文
//   - id: 资产ID
//
// 返回: 资产详情、错误信息
func (l *Logic) GetCI(ctx context.Context, id uint) (*model.CMDBCI, error) {
	var row model.CMDBCI
	if err := l.svcCtx.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// UpdateCI 更新资产属性。
//
// 支持部分字段更新，自动更新修改者和修改时间。
//
// 参数:
//   - ctx: 上下文
//   - uid: 修改者用户ID
//   - id: 资产ID
//   - updates: 更新字段映射
//
// 返回: 更新后的资产、错误信息
func (l *Logic) UpdateCI(ctx context.Context, uid uint, id uint, updates map[string]any) (*model.CMDBCI, error) {
	updates["updated_by"] = uid
	updates["updated_at"] = time.Now()
	if err := l.svcCtx.DB.WithContext(ctx).Model(&model.CMDBCI{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return l.GetCI(ctx, id)
}

// DeleteCI 删除资产（软删除）。
//
// 参数:
//   - ctx: 上下文
//   - id: 资产ID
//
// 返回: 错误信息
func (l *Logic) DeleteCI(ctx context.Context, id uint) error {
	return l.svcCtx.DB.WithContext(ctx).Delete(&model.CMDBCI{}, id).Error
}

// ListRelations 查询关系列表。
//
// 可按资产ID筛选，返回与该资产相关的所有关系（作为源或目标）。
//
// 参数:
//   - ctx: 上下文
//   - ciID: 资产ID，为0时返回所有关系
//
// 返回: 关系列表、错误信息
func (l *Logic) ListRelations(ctx context.Context, ciID uint) ([]model.CMDBRelation, error) {
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.CMDBRelation{})
	if ciID > 0 {
		q = q.Where("from_ci_id = ? OR to_ci_id = ?", ciID, ciID)
	}
	out := make([]model.CMDBRelation, 0)
	return out, q.Order("id DESC").Find(&out).Error
}

// CreateRelation 创建资产关系。
//
// 不允许自引用关系，需要指定源资产、目标资产和关系类型。
//
// 参数:
//   - ctx: 上下文
//   - uid: 创建者用户ID
//   - in: 关系数据
//
// 返回: 创建成功的关系、错误信息
func (l *Logic) CreateRelation(ctx context.Context, uid uint, in model.CMDBRelation) (*model.CMDBRelation, error) {
	in.RelationType = strings.TrimSpace(in.RelationType)
	if in.FromCIID == 0 || in.ToCIID == 0 || in.RelationType == "" {
		return nil, fmt.Errorf("from_ci_id, to_ci_id, relation_type are required")
	}
	if in.FromCIID == in.ToCIID {
		return nil, fmt.Errorf("self relation is not allowed")
	}
	in.CreatedBy = uid
	if err := l.svcCtx.DB.WithContext(ctx).Create(&in).Error; err != nil {
		return nil, err
	}
	return &in, nil
}

// DeleteRelation 删除关系。
//
// 参数:
//   - ctx: 上下文
//   - id: 关系ID
//
// 返回: 错误信息
func (l *Logic) DeleteRelation(ctx context.Context, id uint) error {
	return l.svcCtx.DB.WithContext(ctx).Delete(&model.CMDBRelation{}, id).Error
}

// ListAudits 查询审计日志列表。
//
// 可按资产ID筛选，最多返回200条记录。
//
// 参数:
//   - ctx: 上下文
//   - ciID: 资产ID，为0时返回所有审计记录
//
// 返回: 审计日志列表、错误信息
func (l *Logic) ListAudits(ctx context.Context, ciID uint) ([]model.CMDBAudit, error) {
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.CMDBAudit{})
	if ciID > 0 {
		q = q.Where("ci_id = ?", ciID)
	}
	out := make([]model.CMDBAudit, 0, 64)
	return out, q.Order("id DESC").Limit(200).Find(&out).Error
}

// WriteAudit 写入审计日志。
//
// 参数:
//   - ctx: 上下文
//   - in: 审计日志数据
func (l *Logic) WriteAudit(ctx context.Context, in model.CMDBAudit) {
	_ = l.svcCtx.DB.WithContext(ctx).Create(&in).Error
}

// Topology 获取资产拓扑图数据。
//
// 返回节点和边的数据结构，用于前端可视化展示。
//
// 参数:
//   - ctx: 上下文
//   - projectID: 项目ID，为0时不筛选
//   - teamID: 团队ID，为0时不筛选
//
// 返回: 包含 nodes 和 edges 的拓扑数据、错误信息
func (l *Logic) Topology(ctx context.Context, projectID uint, teamID uint) (map[string]any, error) {
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.CMDBCI{})
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	if teamID > 0 {
		q = q.Where("team_id = ?", teamID)
	}
	cis := make([]model.CMDBCI, 0)
	if err := q.Find(&cis).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(cis))
	for _, ci := range cis {
		ids = append(ids, ci.ID)
	}
	rels := make([]model.CMDBRelation, 0)
	if len(ids) > 0 {
		if err := l.svcCtx.DB.WithContext(ctx).Where("from_ci_id IN ? AND to_ci_id IN ?", ids, ids).Find(&rels).Error; err != nil {
			return nil, err
		}
	}
	nodes := make([]map[string]any, 0, len(cis))
	for _, ci := range cis {
		nodes = append(nodes, map[string]any{
			"id":         ci.ID,
			"ci_uid":     ci.CIUID,
			"ci_type":    ci.CIType,
			"name":       ci.Name,
			"status":     ci.Status,
			"project_id": ci.ProjectID,
			"team_id":    ci.TeamID,
		})
	}
	edges := make([]map[string]any, 0, len(rels))
	for _, r := range rels {
		edges = append(edges, map[string]any{
			"id":            r.ID,
			"from_ci_id":    r.FromCIID,
			"to_ci_id":      r.ToCIID,
			"relation_type": r.RelationType,
		})
	}
	return map[string]any{"nodes": nodes, "edges": edges}, nil
}

// CreateSyncJob 创建同步任务记录。
//
// 参数:
//   - ctx: 上下文
//   - uid: 操作者用户ID
//   - source: 数据源类型，为空时默认为 "all"
//
// 返回: 创建的同步任务、错误信息
func (l *Logic) CreateSyncJob(ctx context.Context, uid uint, source string) (*model.CMDBSyncJob, error) {
	now := time.Now()
	job := model.CMDBSyncJob{
		ID:         fmt.Sprintf("cmdb-sync-%d", now.UnixNano()),
		Source:     defaultIfEmpty(strings.TrimSpace(source), "all"),
		Status:     "running",
		StartedAt:  now,
		FinishedAt: now,
		OperatorID: uid,
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// GetSyncJob 根据ID获取同步任务详情。
//
// 参数:
//   - ctx: 上下文
//   - id: 同步任务ID
//
// 返回: 同步任务详情、错误信息
func (l *Logic) GetSyncJob(ctx context.Context, id string) (*model.CMDBSyncJob, error) {
	var row model.CMDBSyncJob
	if err := l.svcCtx.DB.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// RunSync 执行数据同步任务。
//
// 从外部数据源（主机、集群、服务、部署目标）发现资产信息，
// 与现有数据进行比对，执行新增、更新操作，并同步服务与集群的关系。
// 整个过程在事务中执行，确保数据一致性。
//
// 参数:
//   - ctx: 上下文
//   - uid: 操作者用户ID
//   - source: 数据源类型
//
// 返回: 同步任务详情（包含执行结果摘要）、错误信息
func (l *Logic) RunSync(ctx context.Context, uid uint, source string) (*model.CMDBSyncJob, error) {
	job, err := l.CreateSyncJob(ctx, uid, source)
	if err != nil {
		return nil, err
	}
	summary := syncSummary{}
	now := time.Now()

	err = l.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		discovered, derr := l.discoverAll(ctx)
		if derr != nil {
			return derr
		}
		for _, d := range discovered {
			uidv := ciUID(d.CIType, d.ExternalID)
			var existing model.CMDBCI
			err := tx.Where("ci_uid = ?", uidv).First(&existing).Error
			action := ""
			recordStatus := "ok"
			diffPayload := map[string]any{"name": d.Name, "status": d.Status}
			switch {
			case err == nil:
				changed := existing.Name != d.Name || existing.Status != d.Status || existing.Owner != d.Owner || existing.AttrsJSON != d.AttrsJSON
				if changed {
					updates := map[string]any{
						"name":           d.Name,
						"status":         d.Status,
						"owner":          d.Owner,
						"project_id":     d.ProjectID,
						"team_id":        d.TeamID,
						"attrs_json":     d.AttrsJSON,
						"updated_by":     uid,
						"last_synced_at": now,
					}
					if uerr := tx.Model(&model.CMDBCI{}).Where("id = ?", existing.ID).Updates(updates).Error; uerr != nil {
						summary.Failed++
						recordStatus = "failed"
					} else {
						action = "updated"
						summary.Updated++
					}
				} else {
					action = "unchanged"
					summary.Unchanged++
				}
			case err == gorm.ErrRecordNotFound:
				newRow := model.CMDBCI{
					CIUID:        uidv,
					CIType:       d.CIType,
					Name:         d.Name,
					Source:       d.Source,
					ExternalID:   d.ExternalID,
					ProjectID:    d.ProjectID,
					TeamID:       d.TeamID,
					Owner:        d.Owner,
					Status:       d.Status,
					AttrsJSON:    d.AttrsJSON,
					CreatedBy:    uid,
					UpdatedBy:    uid,
					LastSyncedAt: &now,
				}
				if cerr := tx.Create(&newRow).Error; cerr != nil {
					summary.Failed++
					recordStatus = "failed"
				} else {
					action = "created"
					summary.Created++
				}
			default:
				summary.Failed++
				action = "failed"
				recordStatus = "failed"
			}

			if action == "" {
				action = "failed"
			}
			diffBytes, _ := json.Marshal(diffPayload)
			rec := model.CMDBSyncRecord{
				JobID:    job.ID,
				CIUID:    uidv,
				Action:   action,
				Status:   recordStatus,
				DiffJSON: string(diffBytes),
			}
			_ = tx.Create(&rec).Error
		}

		_ = l.syncServiceClusterRelationsTx(ctx, tx, uid)
		return nil
	})

	job.FinishedAt = time.Now()
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = err.Error()
	} else {
		job.Status = "succeeded"
	}
	buf, _ := json.Marshal(summary)
	job.SummaryJSON = string(buf)
	_ = l.svcCtx.DB.WithContext(ctx).Model(&model.CMDBSyncJob{}).Where("id = ?", job.ID).Updates(map[string]any{
		"status":        job.Status,
		"summary_json":  job.SummaryJSON,
		"error_message": job.ErrorMessage,
		"finished_at":   job.FinishedAt,
	}).Error
	return job, nil
}

// discoverAll 从所有外部数据源发现资产信息。
//
// 发现来源包括：
//   - 主机 (host)
//   - 集群 (cluster)
//   - 服务 (service)
//   - 部署目标 (deployment)
//
// 参数:
//   - ctx: 上下文
//
// 返回: 发现的资产列表、错误信息
func (l *Logic) discoverAll(ctx context.Context) ([]discoveredCI, error) {
	out := make([]discoveredCI, 0, 256)

	var hosts []model.Node
	if err := l.svcCtx.DB.WithContext(ctx).Find(&hosts).Error; err != nil {
		return nil, err
	}
	for _, h := range hosts {
		attrs, _ := json.Marshal(map[string]any{"ip": h.IP, "role": h.Role, "provider": h.Provider})
		out = append(out, discoveredCI{CIType: "host", Source: "host", ExternalID: fmt.Sprintf("%d", h.ID), Name: defaultIfEmpty(h.Name, h.Hostname), Status: defaultIfEmpty(h.Status, "unknown"), AttrsJSON: string(attrs)})
	}

	var clusters []model.Cluster
	if err := l.svcCtx.DB.WithContext(ctx).Find(&clusters).Error; err != nil {
		return nil, err
	}
	for _, c := range clusters {
		attrs, _ := json.Marshal(map[string]any{"endpoint": c.Endpoint, "version": c.Version, "type": c.Type})
		out = append(out, discoveredCI{CIType: "cluster", Source: "cluster", ExternalID: fmt.Sprintf("%d", c.ID), Name: c.Name, Status: defaultIfEmpty(c.Status, "unknown"), AttrsJSON: string(attrs)})
	}

	var services []model.Service
	if err := l.svcCtx.DB.WithContext(ctx).Find(&services).Error; err != nil {
		return nil, err
	}
	for _, s := range services {
		attrs, _ := json.Marshal(map[string]any{"runtime_type": s.RuntimeType, "env": s.Env, "image": s.Image})
		out = append(out, discoveredCI{CIType: "service", Source: "service", ExternalID: fmt.Sprintf("%d", s.ID), Name: s.Name, Status: defaultIfEmpty(s.Status, "unknown"), ProjectID: s.ProjectID, TeamID: s.TeamID, Owner: s.Owner, AttrsJSON: string(attrs)})
	}

	var targets []model.DeploymentTarget
	if err := l.svcCtx.DB.WithContext(ctx).Find(&targets).Error; err != nil {
		return nil, err
	}
	for _, t := range targets {
		attrs, _ := json.Marshal(map[string]any{"target_type": t.TargetType, "cluster_id": t.ClusterID, "env": t.Env})
		out = append(out, discoveredCI{CIType: "deploy_target", Source: "deployment", ExternalID: fmt.Sprintf("%d", t.ID), Name: t.Name, Status: defaultIfEmpty(t.Status, "unknown"), ProjectID: t.ProjectID, TeamID: t.TeamID, AttrsJSON: string(attrs)})
	}

	return out, nil
}

// syncServiceClusterRelationsTx 同步服务与集群的关系。
//
// 根据服务的部署目标配置，在服务和集群资产之间创建 "runs_on" 关系。
// 该方法在同步事务中调用。
//
// 参数:
//   - ctx: 上下文
//   - tx: 数据库事务
//   - uid: 操作者用户ID
//
// 返回: 错误信息
func (l *Logic) syncServiceClusterRelationsTx(ctx context.Context, tx *gorm.DB, uid uint) error {
	var targets []model.ServiceDeployTarget
	if err := tx.WithContext(ctx).Find(&targets).Error; err != nil {
		return err
	}
	for _, t := range targets {
		if t.ServiceID == 0 || t.ClusterID == 0 {
			continue
		}
		fromUID := ciUID("service", fmt.Sprintf("%d", t.ServiceID))
		toUID := ciUID("cluster", fmt.Sprintf("%d", t.ClusterID))
		var fromCI, toCI model.CMDBCI
		if err := tx.Where("ci_uid = ?", fromUID).First(&fromCI).Error; err != nil {
			continue
		}
		if err := tx.Where("ci_uid = ?", toUID).First(&toCI).Error; err != nil {
			continue
		}
		var existing model.CMDBRelation
		err := tx.Where("from_ci_id = ? AND to_ci_id = ? AND relation_type = ?", fromCI.ID, toCI.ID, "runs_on").First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			rel := model.CMDBRelation{FromCIID: fromCI.ID, ToCIID: toCI.ID, RelationType: "runs_on", CreatedBy: uid}
			if err := tx.Create(&rel).Error; err != nil {
				continue
			}
		}
	}
	return nil
}

// defaultIfEmpty 如果字符串为空则返回默认值。
//
// 参数:
//   - v: 原始字符串
//   - d: 默认值
//
// 返回: 原始字符串（去除空白），若为空则返回默认值
func defaultIfEmpty(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return strings.TrimSpace(v)
}
