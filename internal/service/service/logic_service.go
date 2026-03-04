package service

import (
	"context"
	"encoding/json"
	"fmt"
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
		Visibility:            normalized.Visibility,
		GrantedTeams:          marshalUintSlice(normalized.GrantedTeams),
		Icon:                  normalized.Icon,
		Tags:                  marshalStringSlice(normalized.Tags),
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
	if strings.TrimSpace(req.ServiceKind) == "" {
		req.ServiceKind = existing.ServiceKind
	}
	if strings.TrimSpace(req.Visibility) == "" {
		req.Visibility = existing.Visibility
	}
	if req.GrantedTeams == nil {
		_ = json.Unmarshal([]byte(existing.GrantedTeams), &req.GrantedTeams)
	}
	if strings.TrimSpace(req.Icon) == "" {
		req.Icon = existing.Icon
	}
	if req.Tags == nil {
		_ = json.Unmarshal([]byte(existing.Tags), &req.Tags)
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
	existing.Visibility = normalized.Visibility
	existing.GrantedTeams = marshalUintSlice(normalized.GrantedTeams)
	existing.Icon = normalized.Icon
	existing.Tags = marshalStringSlice(normalized.Tags)
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
	if v := strings.TrimSpace(filters["service_kind"]); v != "" {
		query = query.Where("service_kind = ?", v)
	}
	if v := strings.TrimSpace(filters["visibility"]); v != "" {
		query = query.Where("visibility = ?", v)
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

	var list []model.Service
	if err := query.Order("updated_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	uid := parseUint64(filters["viewer_uid"])
	teamID := uint(parseUint64(filters["viewer_team_id"]))
	isAdmin := strings.EqualFold(strings.TrimSpace(filters["viewer_is_admin"]), "true")
	out := make([]ServiceListItem, 0, len(list))
	for i := range list {
		if !l.CheckViewPermission(&list[i], uid, teamID, isAdmin) {
			continue
		}
		out = append(out, toServiceListItem(&list[i]))
	}
	return out, int64(len(out)), nil
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

func (l *Logic) UpdateVisibility(ctx context.Context, id uint, req VisibilityUpdateReq) (ServiceListItem, error) {
	var service model.Service
	if err := l.svcCtx.DB.WithContext(ctx).First(&service, id).Error; err != nil {
		return ServiceListItem{}, err
	}
	visibility := normalizeVisibility(req.Visibility)
	if err := l.svcCtx.DB.WithContext(ctx).Model(&model.Service{}).Where("id = ?", id).Update("visibility", visibility).Error; err != nil {
		return ServiceListItem{}, err
	}
	service.Visibility = visibility
	return toServiceListItem(&service), nil
}

func (l *Logic) UpdateGrantedTeams(ctx context.Context, id uint, req GrantTeamsReq) (ServiceListItem, error) {
	var service model.Service
	if err := l.svcCtx.DB.WithContext(ctx).First(&service, id).Error; err != nil {
		return ServiceListItem{}, err
	}
	if err := l.svcCtx.DB.WithContext(ctx).Model(&model.Service{}).Where("id = ?", id).Update("granted_teams", marshalUintSlice(req.GrantedTeams)).Error; err != nil {
		return ServiceListItem{}, err
	}
	service.GrantedTeams = marshalUintSlice(req.GrantedTeams)
	return toServiceListItem(&service), nil
}

func (l *Logic) CheckViewPermission(s *model.Service, uid uint64, viewerTeamID uint, isAdmin bool) bool {
	if s == nil {
		return false
	}
	if isAdmin {
		return true
	}
	if uint(uid) == s.OwnerUserID {
		return true
	}
	visibility := normalizeVisibility(defaultIfEmpty(s.Visibility, "team"))
	switch visibility {
	case "private":
		return false
	case "team":
		return viewerTeamID > 0 && viewerTeamID == s.TeamID
	case "team-granted":
		if viewerTeamID > 0 && viewerTeamID == s.TeamID {
			return true
		}
		teams := parseUintSlice(s.GrantedTeams)
		for _, teamID := range teams {
			if teamID == viewerTeamID {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func (l *Logic) CheckEditPermission(s *model.Service, uid uint64, viewerTeamID uint, isAdmin bool) bool {
	if s == nil {
		return false
	}
	if isAdmin || uint(uid) == s.OwnerUserID {
		return true
	}
	return viewerTeamID > 0 && viewerTeamID == s.TeamID
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
		Visibility:            s.Visibility,
		Icon:                  s.Icon,
		DeployCount:           s.DeployCount,
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
	if strings.TrimSpace(s.GrantedTeams) != "" {
		_ = json.Unmarshal([]byte(s.GrantedTeams), &out.GrantedTeams)
	}
	if strings.TrimSpace(s.Tags) != "" {
		_ = json.Unmarshal([]byte(s.Tags), &out.Tags)
	}
	return out
}

func (l *Logic) normalizeAndRender(req ServiceCreateReq) (ServiceCreateReq, string, error) {
	r := req
	if r.ProjectID == 0 {
		return r, "", fmt.Errorf("project_id is required from request or X-Project-ID context")
	}
	r.Env = defaultIfEmpty(r.Env, "staging")
	r.Owner = defaultIfEmpty(r.Owner, "system")
	r.RuntimeType = defaultIfEmpty(r.RuntimeType, "k8s")
	r.ConfigMode = defaultIfEmpty(r.ConfigMode, "standard")
	r.ServiceKind = defaultIfEmpty(r.ServiceKind, "business")
	r.Visibility = normalizeVisibility(defaultIfEmpty(r.Visibility, defaultVisibilityByKind(r.ServiceKind)))
	r.RenderTarget = defaultIfEmpty(r.RenderTarget, r.RuntimeType)
	r.ServiceType = defaultIfEmpty(r.ServiceType, "stateless")
	r.Icon = strings.TrimSpace(r.Icon)
	if r.Tags == nil {
		r.Tags = make([]string, 0)
	}
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

func marshalUintSlice(list []uint) string {
	if list == nil {
		list = make([]uint, 0)
	}
	b, _ := json.Marshal(list)
	return string(b)
}

func marshalStringSlice(list []string) string {
	if list == nil {
		list = make([]string, 0)
	}
	b, _ := json.Marshal(list)
	return string(b)
}

func parseUintSlice(raw string) []uint {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []uint
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func parseUint64(v string) uint64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	var n uint64
	_, _ = fmt.Sscanf(v, "%d", &n)
	return n
}

func normalizeVisibility(v string) string {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "private", "team", "team-granted", "public":
		return s
	default:
		return "team"
	}
}

func defaultVisibilityByKind(kind string) string {
	if strings.EqualFold(strings.TrimSpace(kind), "middleware") {
		return "public"
	}
	return "team"
}
