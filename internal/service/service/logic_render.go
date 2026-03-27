// Package service 提供服务渲染相关的业务逻辑。
//
// 本文件实现服务配置渲染预览和转换功能。
package service

import (
	"errors"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Preview 预览服务配置渲染结果。
//
// 根据模式处理标准配置或自定义 YAML，执行变量替换和语法校验。
//
// 参数:
//   - req: 渲染预览请求
//
// 返回: 渲染预览响应和错误信息
func (l *Logic) Preview(req RenderPreviewReq) (RenderPreviewResp, error) {
	req.Variables = normalizeStringMap(req.Variables)
	if req.Mode == "custom" {
		diagnostics := validateCustomYAML(req.Target, req.CustomYAML)
		resolved, unresolved := resolveTemplateVars(req.CustomYAML, req.Variables, nil)
		return RenderPreviewResp{
			RenderedYAML:   req.CustomYAML,
			ResolvedYAML:   resolved,
			Diagnostics:    diagnostics,
			UnresolvedVars: unresolved,
			DetectedVars:   detectTemplateVars(req.CustomYAML),
		}, nil
	}
	resp, err := renderFromStandard(req.ServiceName, req.ServiceType, req.Target, req.StandardConfig)
	if err != nil {
		return RenderPreviewResp{}, err
	}
	resp.DetectedVars = detectTemplateVars(resp.RenderedYAML)
	resp.ResolvedYAML, resp.UnresolvedVars = resolveTemplateVars(resp.RenderedYAML, req.Variables, nil)
	resp.ASTSummary = map[string]any{
		"target": req.Target,
		"docs":   strings.Count(resp.RenderedYAML, "\n---\n") + 1,
	}
	return resp, nil
}

// Transform 将标准服务配置转换为自定义 YAML。
//
// 根据标准配置生成目标平台的 YAML 配置。
//
// 参数:
//   - req: 转换请求
//
// 返回: 转换响应和错误信息
func (l *Logic) Transform(req TransformReq) (TransformResp, error) {
	res, err := renderFromStandard(req.ServiceName, req.ServiceType, req.Target, req.StandardConfig)
	if err != nil {
		return TransformResp{}, err
	}
	return TransformResp{
		CustomYAML:   res.RenderedYAML,
		SourceHash:   sourceHash(res.RenderedYAML),
		DetectedVars: detectTemplateVars(res.RenderedYAML),
	}, nil
}

// validateCustomYAML 校验自定义 YAML 内容。
//
// 检查 YAML 语法和必要的字段（如 K8s 的 kind，Compose 的 services）。
//
// 参数:
//   - target: 目标平台 (k8s/compose)
//   - content: YAML 内容
//
// 返回: 诊断信息列表
func validateCustomYAML(target, content string) []RenderDiagnostic {
	diags := make([]RenderDiagnostic, 0)
	if strings.TrimSpace(content) == "" {
		return []RenderDiagnostic{{Level: "warning", Code: "empty_yaml", Message: "custom_yaml is empty"}}
	}
	if target == "compose" {
		var obj map[string]any
		if err := yaml.Unmarshal([]byte(content), &obj); err != nil {
			diags = append(diags, RenderDiagnostic{Level: "error", Code: "invalid_compose_yaml", Message: err.Error()})
			return diags
		}
		if _, ok := obj["services"]; !ok {
			diags = append(diags, RenderDiagnostic{Level: "warning", Code: "compose_services_missing", Message: "compose yaml missing services"})
		}
		return diags
	}
	dec := yaml.NewDecoder(strings.NewReader(content))
	for {
		var obj map[string]any
		if err := dec.Decode(&obj); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			diags = append(diags, RenderDiagnostic{Level: "error", Code: "invalid_k8s_yaml", Message: err.Error()})
			break
		}
		if len(obj) == 0 {
			continue
		}
		if _, ok := obj["kind"]; !ok {
			diags = append(diags, RenderDiagnostic{Level: "warning", Code: "k8s_kind_missing", Message: "yaml doc missing kind"})
		}
	}
	return diags
}
