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

type Logic struct {
	svcCtx *svc.ServiceContext
}

func NewLogic(svcCtx *svc.ServiceContext) *Logic { return &Logic{svcCtx: svcCtx} }

type CIFilter struct {
	Type      string
	Status    string
	Keyword   string
	ProjectID uint
	TeamID    uint
	Page      int
	PageSize  int
}

type syncSummary struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
	Failed    int `json:"failed"`
}

type discoveredCI struct {
	CIType     string
	Source     string
	ExternalID string
	Name       string
	Status     string
	ProjectID  uint
	TeamID     uint
	Owner      string
	AttrsJSON  string
}

func normalizePage(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func normalizePageSize(v, def int) int {
	if v <= 0 {
		return def
	}
	if v > 200 {
		return 200
	}
	return v
}

func ciUID(ciType, externalID string) string {
	return strings.TrimSpace(ciType) + ":" + strings.TrimSpace(externalID)
}

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

func (l *Logic) GetCI(ctx context.Context, id uint) (*model.CMDBCI, error) {
	var row model.CMDBCI
	if err := l.svcCtx.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (l *Logic) UpdateCI(ctx context.Context, uid uint, id uint, updates map[string]any) (*model.CMDBCI, error) {
	updates["updated_by"] = uid
	updates["updated_at"] = time.Now()
	if err := l.svcCtx.DB.WithContext(ctx).Model(&model.CMDBCI{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return l.GetCI(ctx, id)
}

func (l *Logic) DeleteCI(ctx context.Context, id uint) error {
	return l.svcCtx.DB.WithContext(ctx).Delete(&model.CMDBCI{}, id).Error
}

func (l *Logic) ListRelations(ctx context.Context, ciID uint) ([]model.CMDBRelation, error) {
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.CMDBRelation{})
	if ciID > 0 {
		q = q.Where("from_ci_id = ? OR to_ci_id = ?", ciID, ciID)
	}
	out := make([]model.CMDBRelation, 0)
	return out, q.Order("id DESC").Find(&out).Error
}

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

func (l *Logic) DeleteRelation(ctx context.Context, id uint) error {
	return l.svcCtx.DB.WithContext(ctx).Delete(&model.CMDBRelation{}, id).Error
}

func (l *Logic) ListAudits(ctx context.Context, ciID uint) ([]model.CMDBAudit, error) {
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.CMDBAudit{})
	if ciID > 0 {
		q = q.Where("ci_id = ?", ciID)
	}
	out := make([]model.CMDBAudit, 0, 64)
	return out, q.Order("id DESC").Limit(200).Find(&out).Error
}

func (l *Logic) WriteAudit(ctx context.Context, in model.CMDBAudit) {
	_ = l.svcCtx.DB.WithContext(ctx).Create(&in).Error
}

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

func (l *Logic) GetSyncJob(ctx context.Context, id string) (*model.CMDBSyncJob, error) {
	var row model.CMDBSyncJob
	if err := l.svcCtx.DB.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

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

func defaultIfEmpty(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return strings.TrimSpace(v)
}
