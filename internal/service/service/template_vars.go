// Package service 提供模板变量检测和解析功能。
//
// 本文件实现模板变量的正则匹配、检测和替换功能。
package service

import (
	"regexp"
	"sort"
	"strings"
)

// templateVarPattern 匹配模板变量的正则表达式。
//
// 支持格式: {{ var_name }} 或 {{ var_name|default:value }}
var templateVarPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_\.\-]*)(?:\|default:([^}]+))?\s*\}\}`)

// detectTemplateVars 检测内容中的模板变量。
//
// 使用正则表达式匹配所有模板变量，返回去重后的变量列表。
//
// 参数:
//   - content: 模板内容
//
// 返回: 检测到的模板变量列表
func detectTemplateVars(content string) []TemplateVar {
	matches := templateVarPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	uniq := make(map[string]TemplateVar)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		name := strings.TrimSpace(m[1])
		if name == "" {
			continue
		}
		def := ""
		if len(m) > 2 {
			def = strings.TrimSpace(m[2])
		}
		item := TemplateVar{
			Name:       name,
			Required:   def == "",
			Default:    def,
			SourcePath: "template",
		}
		uniq[name] = item
	}
	out := make([]TemplateVar, 0, len(uniq))
	for _, v := range uniq {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// resolveTemplateVars 替换模板变量。
//
// 按优先级顺序替换变量: 请求变量 > 环境变量 > 默认值。
//
// 参数:
//   - content: 模板内容
//   - reqValues: 请求变量映射
//   - envValues: 环境变量映射
//
// 返回: 替换后的内容和未解析变量列表
func resolveTemplateVars(content string, reqValues map[string]string, envValues map[string]string) (string, []string) {
	out := templateVarPattern.ReplaceAllStringFunc(content, func(token string) string {
		m := templateVarPattern.FindStringSubmatch(token)
		if len(m) < 2 {
			return token
		}
		name := strings.TrimSpace(m[1])
		def := ""
		if len(m) > 2 {
			def = strings.TrimSpace(m[2])
		}
		if v, ok := reqValues[name]; ok && strings.TrimSpace(v) != "" {
			return v
		}
		if v, ok := envValues[name]; ok && strings.TrimSpace(v) != "" {
			return v
		}
		if def != "" {
			return def
		}
		return token
	})
	var unresolved []string
	for _, v := range detectTemplateVars(out) {
		if v.Required {
			unresolved = append(unresolved, v.Name)
		}
	}
	return out, unresolved
}
