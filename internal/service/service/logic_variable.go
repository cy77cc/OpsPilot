package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/cy77cc/k8s-manage/internal/model"
	"gorm.io/gorm"
)

func (l *Logic) ExtractVariables(ctx context.Context, req VariableExtractReq) (VariableExtractResp, error) {
	if strings.TrimSpace(req.CustomYAML) != "" {
		return VariableExtractResp{Vars: detectTemplateVars(req.CustomYAML)}, nil
	}
	resp, err := renderFromStandard(req.ServiceName, req.ServiceType, req.RenderTarget, req.StandardConfig)
	if err != nil {
		return VariableExtractResp{}, err
	}
	return VariableExtractResp{Vars: detectTemplateVars(resp.RenderedYAML)}, nil
}

func (l *Logic) GetVariableSchema(ctx context.Context, serviceID uint) ([]TemplateVar, error) {
	var rev model.ServiceRevision
	if err := l.svcCtx.DB.WithContext(ctx).Where("service_id = ?", serviceID).Order("revision_no DESC").First(&rev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var service model.Service
			if err := l.svcCtx.DB.WithContext(ctx).First(&service, serviceID).Error; err != nil {
				return nil, err
			}
			content := defaultIfEmpty(service.CustomYAML, service.YamlContent)
			return detectTemplateVars(content), nil
		}
		return nil, err
	}
	var vars []TemplateVar
	if strings.TrimSpace(rev.VariableSchema) != "" {
		_ = json.Unmarshal([]byte(rev.VariableSchema), &vars)
	}
	return vars, nil
}

func (l *Logic) GetVariableValues(ctx context.Context, serviceID uint, env string) (VariableValuesResp, error) {
	var set model.ServiceVariableSet
	err := l.svcCtx.DB.WithContext(ctx).Where("service_id = ? AND env = ?", serviceID, defaultIfEmpty(env, "staging")).First(&set).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return VariableValuesResp{ServiceID: serviceID, Env: defaultIfEmpty(env, "staging"), Values: map[string]string{}}, nil
		}
		return VariableValuesResp{}, err
	}
	out := VariableValuesResp{
		ServiceID: serviceID,
		Env:       set.Env,
		Values:    map[string]string{},
		UpdatedAt: set.UpdatedAt,
	}
	_ = json.Unmarshal([]byte(set.ValuesJSON), &out.Values)
	_ = json.Unmarshal([]byte(set.SecretKeys), &out.SecretKeys)
	return out, nil
}

func (l *Logic) UpsertVariableValues(ctx context.Context, serviceID uint, uid uint64, req VariableValuesUpsertReq) (VariableValuesResp, error) {
	env := defaultIfEmpty(req.Env, "staging")
	req.Values = normalizeStringMap(req.Values)
	valuesJSON := mustJSON(req.Values)
	secretJSON := mustJSON(req.SecretKeys)
	var set model.ServiceVariableSet
	err := l.svcCtx.DB.WithContext(ctx).Where("service_id = ? AND env = ?", serviceID, env).First(&set).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return VariableValuesResp{}, err
		}
		set = model.ServiceVariableSet{
			ServiceID:  serviceID,
			Env:        env,
			ValuesJSON: valuesJSON,
			SecretKeys: secretJSON,
			UpdatedBy:  uint(uid),
		}
		if err := l.svcCtx.DB.WithContext(ctx).Create(&set).Error; err != nil {
			return VariableValuesResp{}, err
		}
	} else {
		set.ValuesJSON = valuesJSON
		set.SecretKeys = secretJSON
		set.UpdatedBy = uint(uid)
		if err := l.svcCtx.DB.WithContext(ctx).Save(&set).Error; err != nil {
			return VariableValuesResp{}, err
		}
	}
	return l.GetVariableValues(ctx, serviceID, env)
}
