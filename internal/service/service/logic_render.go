package service

import (
	"errors"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

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
