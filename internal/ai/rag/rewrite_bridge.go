// Package rag 提供 RAG（检索增强生成）与改写模块的桥接功能。
//
// 本文件实现从改写输出构建检索查询信封的功能。
package rag

import (
	"strings"

	"github.com/cy77cc/OpsPilot/internal/ai/rewrite"
)

// RewriteQueryEnvelope 改写查询信封，包含检索所需的所有信息。
type RewriteQueryEnvelope struct {
	Intent         string                `json:"intent,omitempty"`          // 检索意图
	Goal           string                `json:"goal,omitempty"`            // 归一化目标
	Queries        []string              `json:"queries,omitempty"`         // 检索查询列表
	Keywords       []string              `json:"keywords,omitempty"`        // 关键词列表
	KnowledgeScope []string              `json:"knowledge_scope,omitempty"` // 知识范围
	ResourceHints  rewrite.ResourceHints `json:"resource_hints,omitempty"`  // 资源提示
	RequiresRAG    bool                  `json:"requires_rag,omitempty"`    // 是否需要 RAG
}

// BuildRewriteQueryEnvelope 从改写输出构建查询信封。
//
// 提取检索查询、关键词和资源提示，用于后续知识检索。
func BuildRewriteQueryEnvelope(out rewrite.Output) RewriteQueryEnvelope {
	semantic := out.SemanticContract()
	// 构建查询列表
	queries := append([]string(nil), semantic.RetrievalQueries...)
	if len(queries) == 0 {
		if goal := strings.TrimSpace(semantic.NormalizedGoal); goal != "" {
			queries = append(queries, goal)
		} else if raw := strings.TrimSpace(semantic.RawUserInput); raw != "" {
			queries = append(queries, raw)
		}
	}
	// 构建关键词列表
	keywords := append([]string(nil), semantic.RetrievalKeywords...)
	if len(keywords) == 0 {
		keywords = appendKeywords(keywords, semantic.DomainHints...)
		keywords = appendKeywords(keywords, semantic.KnowledgeScope...)
	}
	return RewriteQueryEnvelope{
		Intent:         strings.TrimSpace(semantic.RetrievalIntent),
		Goal:           strings.TrimSpace(semantic.NormalizedGoal),
		Queries:        dedupeStrings(queries),
		Keywords:       dedupeStrings(keywords),
		KnowledgeScope: dedupeStrings(semantic.KnowledgeScope),
		ResourceHints:  semantic.ResourceHints,
		RequiresRAG:    semantic.RequiresRAG || len(semantic.RetrievalQueries) > 0,
	}
}

// appendKeywords 追加关键词到列表。
func appendKeywords(base []string, values ...string) []string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		base = append(base, value)
	}
	return base
}

// dedupeStrings 去重字符串列表。
func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
