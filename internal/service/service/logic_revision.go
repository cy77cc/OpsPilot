package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cy77cc/k8s-manage/internal/model"
)

func (l *Logic) ListRevisions(ctx context.Context, serviceID uint) ([]ServiceRevisionItem, error) {
	var rows []model.ServiceRevision
	if err := l.svcCtx.DB.WithContext(ctx).Where("service_id = ?", serviceID).Order("revision_no DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ServiceRevisionItem, 0, len(rows))
	for i := range rows {
		item := ServiceRevisionItem{
			ID:           rows[i].ID,
			ServiceID:    rows[i].ServiceID,
			RevisionNo:   rows[i].RevisionNo,
			ConfigMode:   rows[i].ConfigMode,
			RenderTarget: rows[i].RenderTarget,
			CreatedBy:    rows[i].CreatedBy,
			CreatedAt:    rows[i].CreatedAt,
		}
		if strings.TrimSpace(rows[i].VariableSchema) != "" {
			_ = json.Unmarshal([]byte(rows[i].VariableSchema), &item.VariableSchema)
		}
		out = append(out, item)
	}
	return out, nil
}

func (l *Logic) CreateRevision(ctx context.Context, serviceID uint, uid uint64, req RevisionCreateReq) (ServiceRevisionItem, error) {
	var service model.Service
	if err := l.svcCtx.DB.WithContext(ctx).First(&service, serviceID).Error; err != nil {
		return ServiceRevisionItem{}, err
	}
	if strings.TrimSpace(req.ConfigMode) != "" {
		service.ConfigMode = req.ConfigMode
	}
	if strings.TrimSpace(req.RenderTarget) != "" {
		service.RenderTarget = req.RenderTarget
	}
	if req.StandardConfig != nil {
		b, _ := json.Marshal(req.StandardConfig)
		service.StandardJSON = string(b)
	}
	if strings.TrimSpace(req.CustomYAML) != "" {
		service.CustomYAML = req.CustomYAML
		service.YamlContent = req.CustomYAML
	}
	rev, err := l.createRevisionRecord(ctx, &service, uint(uid), req.VariableSchema)
	if err != nil {
		return ServiceRevisionItem{}, err
	}
	out := ServiceRevisionItem{
		ID:           rev.ID,
		ServiceID:    rev.ServiceID,
		RevisionNo:   rev.RevisionNo,
		ConfigMode:   rev.ConfigMode,
		RenderTarget: rev.RenderTarget,
		CreatedBy:    rev.CreatedBy,
		CreatedAt:    rev.CreatedAt,
	}
	if strings.TrimSpace(rev.VariableSchema) != "" {
		_ = json.Unmarshal([]byte(rev.VariableSchema), &out.VariableSchema)
	}
	return out, nil
}

func (l *Logic) createRevisionRecord(ctx context.Context, service *model.Service, createdBy uint, override []TemplateVar) (*model.ServiceRevision, error) {
	var maxRevision uint
	_ = l.svcCtx.DB.WithContext(ctx).Model(&model.ServiceRevision{}).Where("service_id = ?", service.ID).Select("COALESCE(MAX(revision_no),0)").Scan(&maxRevision).Error
	schema := override
	if len(schema) == 0 {
		schema = detectTemplateVars(defaultIfEmpty(service.CustomYAML, service.YamlContent))
	}
	schemaJSON := mustJSON(schema)
	rev := &model.ServiceRevision{
		ServiceID:      service.ID,
		RevisionNo:     maxRevision + 1,
		ConfigMode:     service.ConfigMode,
		RenderTarget:   service.RenderTarget,
		StandardConfig: service.StandardJSON,
		CustomYAML:     service.CustomYAML,
		VariableSchema: schemaJSON,
		CreatedBy:      createdBy,
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(rev).Error; err != nil {
		return nil, err
	}
	service.LastRevisionID = rev.ID
	if err := l.svcCtx.DB.WithContext(ctx).Model(service).Updates(map[string]any{
		"last_revision_id":        rev.ID,
		"template_engine_version": defaultIfEmpty(service.TemplateEngineVersion, "v1"),
	}).Error; err != nil {
		return nil, err
	}
	return rev, nil
}
