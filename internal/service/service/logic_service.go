package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cy77cc/k8s-manage/internal/model"
)

func (l *Logic) Create(ctx context.Context, uid uint64, req ServiceCreateReq) (ServiceListItem, error) {
	normalized, rendered, err := l.normalizeAndRender(req)
	if err != nil {
		return ServiceListItem{}, err
	}
	cfg := ensureStandardConfig(normalized.StandardConfig)
	labelsJSON, _ := json.Marshal(normalized.Labels)
	standardJSON, _ := json.Marshal(cfg)
	service := &model.Service{
		ProjectID:             normalized.ProjectID,
		TeamID:                normalized.TeamID,
		OwnerUserID:           uint(uid),
		Owner:                 normalized.Owner,
		Env:                   normalized.Env,
		RuntimeType:           normalized.RuntimeType,
		ConfigMode:            normalized.ConfigMode,
		ServiceKind:           normalized.ServiceKind,
		RenderTarget:          normalized.RenderTarget,
		LabelsJSON:            string(labelsJSON),
		StandardJSON:          string(standardJSON),
		CustomYAML:            normalized.CustomYAML,
		TemplateVer:           defaultIfEmpty(normalized.SourceTemplateV, "v1"),
		TemplateEngineVersion: "v1",
		Status:                defaultIfEmpty(normalized.Status, "draft"),
		Name:                  normalized.Name,
		Type:                  normalized.ServiceType,
		Image:                 cfg.Image,
		Replicas:              cfg.Replicas,
		ServicePort:           cfg.Ports[0].ServicePort,
		ContainerPort:         cfg.Ports[0].ContainerPort,
		EnvVars:               buildLegacyEnvs(cfg),
		Resources:             buildLegacyResources(cfg),
		YamlContent:           rendered,
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(service).Error; err != nil {
		return ServiceListItem{}, err
	}
	if _, err := l.createRevisionRecord(ctx, service, uint(uid), nil); err != nil {
		return ServiceListItem{}, err
	}
	return toServiceListItem(service), nil
}

func (l *Logic) Update(ctx context.Context, id uint, req ServiceCreateReq) (ServiceListItem, error) {
	var existing model.Service
	if err := l.svcCtx.DB.WithContext(ctx).First(&existing, id).Error; err != nil {
		return ServiceListItem{}, err
	}
	if req.ProjectID == 0 {
		req.ProjectID = existing.ProjectID
	}
	if req.TeamID == 0 {
		req.TeamID = existing.TeamID
	}
	if strings.TrimSpace(req.Name) == "" {
		req.Name = existing.Name
	}
	if strings.TrimSpace(req.Env) == "" {
		req.Env = existing.Env
	}
	if strings.TrimSpace(req.Owner) == "" {
		req.Owner = existing.Owner
	}
	if strings.TrimSpace(req.RuntimeType) == "" {
		req.RuntimeType = existing.RuntimeType
	}
	if strings.TrimSpace(req.ConfigMode) == "" {
		req.ConfigMode = existing.ConfigMode
	}
	if strings.TrimSpace(req.ServiceType) == "" {
		req.ServiceType = existing.Type
	}
	if strings.TrimSpace(req.RenderTarget) == "" {
		req.RenderTarget = existing.RenderTarget
	}
	if req.StandardConfig == nil && existing.StandardJSON != "" {
		var c StandardServiceConfig
		if err := json.Unmarshal([]byte(existing.StandardJSON), &c); err == nil {
			req.StandardConfig = &c
		}
	}
	if req.CustomYAML == "" {
		req.CustomYAML = existing.CustomYAML
	}

	normalized, rendered, err := l.normalizeAndRender(req)
	if err != nil {
		return ServiceListItem{}, err
	}
	cfg := ensureStandardConfig(normalized.StandardConfig)
	labelsJSON, _ := json.Marshal(normalized.Labels)
	standardJSON, _ := json.Marshal(cfg)

	existing.ProjectID = normalized.ProjectID
	existing.TeamID = normalized.TeamID
	existing.Owner = normalized.Owner
	existing.Env = normalized.Env
	existing.RuntimeType = normalized.RuntimeType
	existing.ConfigMode = normalized.ConfigMode
	existing.ServiceKind = normalized.ServiceKind
	existing.RenderTarget = normalized.RenderTarget
	existing.LabelsJSON = string(labelsJSON)
	existing.StandardJSON = string(standardJSON)
	existing.CustomYAML = normalized.CustomYAML
	existing.TemplateVer = defaultIfEmpty(normalized.SourceTemplateV, existing.TemplateVer)
	existing.Status = defaultIfEmpty(normalized.Status, existing.Status)
	existing.Name = normalized.Name
	existing.Type = normalized.ServiceType
	existing.Image = cfg.Image
	existing.Replicas = cfg.Replicas
	existing.ServicePort = cfg.Ports[0].ServicePort
	existing.ContainerPort = cfg.Ports[0].ContainerPort
	existing.EnvVars = buildLegacyEnvs(cfg)
	existing.Resources = buildLegacyResources(cfg)
	existing.YamlContent = rendered

	if err := l.svcCtx.DB.WithContext(ctx).Save(&existing).Error; err != nil {
		return ServiceListItem{}, err
	}
	if _, err := l.createRevisionRecord(ctx, &existing, existing.OwnerUserID, nil); err != nil {
		return ServiceListItem{}, err
	}
	return toServiceListItem(&existing), nil
}

func (l *Logic) List(ctx context.Context, filters map[string]string) ([]ServiceListItem, int64, error) {
	query := l.svcCtx.DB.WithContext(ctx).Model(&model.Service{})
	if v := strings.TrimSpace(filters["project_id"]); v != "" {
		query = query.Where("project_id = ?", v)
	}
	if v := strings.TrimSpace(filters["team_id"]); v != "" {
		query = query.Where("team_id = ?", v)
	}
	if v := strings.TrimSpace(filters["runtime_type"]); v != "" {
		query = query.Where("runtime_type = ?", v)
	}
	if v := strings.TrimSpace(filters["env"]); v != "" {
		query = query.Where("env = ?", v)
	}
	if v := strings.TrimSpace(filters["q"]); v != "" {
		query = query.Where("name LIKE ? OR owner LIKE ?", "%"+v+"%", "%"+v+"%")
	}
	if v := strings.TrimSpace(filters["label_selector"]); v != "" {
		parts := strings.Split(v, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			query = query.Where("labels_json LIKE ?", "%"+p+"%")
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Service
	if err := query.Order("updated_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	out := make([]ServiceListItem, 0, len(list))
	for i := range list {
		out = append(out, toServiceListItem(&list[i]))
	}
	return out, total, nil
}

func (l *Logic) Get(ctx context.Context, id uint) (ServiceListItem, error) {
	var service model.Service
	if err := l.svcCtx.DB.WithContext(ctx).First(&service, id).Error; err != nil {
		return ServiceListItem{}, err
	}
	return toServiceListItem(&service), nil
}

func (l *Logic) Delete(ctx context.Context, id uint) error {
	return l.svcCtx.DB.WithContext(ctx).Delete(&model.Service{}, id).Error
}

func toServiceListItem(s *model.Service) ServiceListItem {
	out := ServiceListItem{
		ID:                    s.ID,
		ProjectID:             s.ProjectID,
		TeamID:                s.TeamID,
		Name:                  s.Name,
		Env:                   s.Env,
		Owner:                 s.Owner,
		RuntimeType:           s.RuntimeType,
		ConfigMode:            s.ConfigMode,
		ServiceKind:           s.ServiceKind,
		Status:                s.Status,
		LastRevisionID:        s.LastRevisionID,
		DefaultTargetID:       s.DefaultTargetID,
		TemplateEngineVersion: s.TemplateEngineVersion,
		CustomYAML:            s.CustomYAML,
		RenderedYAML:          s.YamlContent,
		CreatedAt:             s.CreatedAt,
		UpdatedAt:             s.UpdatedAt,
	}
	if strings.TrimSpace(s.LabelsJSON) != "" {
		_ = json.Unmarshal([]byte(s.LabelsJSON), &out.Labels)
	}
	if strings.TrimSpace(s.StandardJSON) != "" {
		var cfg StandardServiceConfig
		if json.Unmarshal([]byte(s.StandardJSON), &cfg) == nil {
			out.StandardConfig = &cfg
		}
	}
	return out
}

func (l *Logic) normalizeAndRender(req ServiceCreateReq) (ServiceCreateReq, string, error) {
	r := req
	if r.ProjectID == 0 {
		if v := strings.TrimSpace(os.Getenv("DEFAULT_PROJECT_ID")); v != "" {
			if n, _ := strconv.Atoi(v); n > 0 {
				r.ProjectID = uint(n)
			}
		}
	}
	if r.ProjectID == 0 {
		r.ProjectID = 1
	}
	r.Env = defaultIfEmpty(r.Env, "staging")
	r.Owner = defaultIfEmpty(r.Owner, "system")
	r.RuntimeType = defaultIfEmpty(r.RuntimeType, "k8s")
	r.ConfigMode = defaultIfEmpty(r.ConfigMode, "standard")
	r.RenderTarget = defaultIfEmpty(r.RenderTarget, r.RuntimeType)
	r.ServiceType = defaultIfEmpty(r.ServiceType, "stateless")
	if r.Name == "" {
		r.Name = "service"
	}

	if r.ConfigMode == "custom" {
		r.CustomYAML = defaultIfEmpty(r.CustomYAML, r.YamlContent)
		if strings.TrimSpace(r.CustomYAML) == "" {
			return r, "", fmt.Errorf("custom_yaml is required when config_mode=custom")
		}
		return r, r.CustomYAML, nil
	}

	if r.StandardConfig == nil {
		r.StandardConfig = &StandardServiceConfig{
			Image:    defaultIfEmpty(r.Image, "nginx:latest"),
			Replicas: maxInt32(r.Replicas, 1),
			Ports: []PortConfig{{
				Name:          "http",
				Protocol:      "TCP",
				ContainerPort: maxInt32(r.ContainerPort, 8080),
				ServicePort:   maxInt32(r.ServicePort, 80),
			}},
			Envs:      r.EnvVars,
			Resources: r.Resources,
		}
	}
	if strings.TrimSpace(r.StandardConfig.Image) == "" {
		r.StandardConfig.Image = defaultIfEmpty(r.Image, "nginx:latest")
	}
	if len(r.StandardConfig.Ports) == 0 {
		r.StandardConfig.Ports = []PortConfig{{
			Name:          "http",
			Protocol:      "TCP",
			ContainerPort: maxInt32(r.ContainerPort, 8080),
			ServicePort:   maxInt32(r.ServicePort, 80),
		}}
	}
	resp, err := renderFromStandard(r.Name, r.ServiceType, r.RenderTarget, r.StandardConfig)
	if err != nil {
		return r, "", err
	}
	r.CustomYAML = resp.RenderedYAML
	return r, resp.RenderedYAML, nil
}
