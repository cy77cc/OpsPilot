// Package service 提供服务变量管理相关的业务逻辑。
//
// 本文件实现服务变量提取、Schema 获取、值管理等业务逻辑。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// ExtractVariables 提取服务模板变量。
//
// 从标准配置或自定义 YAML 中检测模板变量定义。
//
// 参数:
//   - ctx: 上下文
//   - req: 变量提取请求
//
// 返回: 变量提取响应和错误信息
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

// GetVariableSchema 获取服务变量 Schema。
//
// 从最新版本记录获取变量 Schema，若无版本则从服务配置中检测。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//
// 返回: 变量 Schema 列表和错误信息
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

// GetVariableValues 获取服务变量值。
//
// 查询指定服务在指定环境下的变量值集合。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//   - env: 环境名称
//
// 返回: 变量值响应和错误信息
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

// UpsertVariableValues 创建或更新服务变量值。
//
// 保存指定服务在指定环境下的变量值集合。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//   - uid: 操作用户 ID
//   - req: 变量值更新请求
//
// 返回: 变量值响应和错误信息
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
