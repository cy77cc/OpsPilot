// Package rag 提供 RAG（检索增强生成）相关的知识检索功能。
//
// 本文件实现知识条目的检索和提示词增强功能。
package rag

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Retriever 知识检索器接口。
type Retriever interface {
	Retrieve(ctx context.Context, namespace, query string, limit int) ([]KnowledgeEntry, error)
}

// NamespaceRetriever 基于命名空间的知识检索器。
type NamespaceRetriever struct {
	indexer Indexer
}

// NewNamespaceRetriever 创建新的命名空间检索器。
func NewNamespaceRetriever(indexer Indexer) *NamespaceRetriever {
	return &NamespaceRetriever{indexer: indexer}
}

// Retrieve 从指定命名空间检索相关知识条目。
//
// 使用简单的词频匹配算法，支持按相关性和时间排序。
func (r *NamespaceRetriever) Retrieve(ctx context.Context, namespace, query string, limit int) ([]KnowledgeEntry, error) {
	if r == nil || r.indexer == nil {
		return nil, fmt.Errorf("retriever is not initialized")
	}
	entries, err := r.indexer.List(ctx, strings.TrimSpace(namespace))
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 4
	}
	query = strings.ToLower(strings.TrimSpace(query))
	// 无查询时返回最新条目
	if query == "" {
		if len(entries) > limit {
			return entries[:limit], nil
		}
		return entries, nil
	}

	// 计算匹配分数并排序
	type scored struct {
		entry KnowledgeEntry
		score int
	}
	scoredEntries := make([]scored, 0, len(entries))
	for _, entry := range entries {
		score := matchScore(query, entry)
		if score == 0 {
			continue
		}
		scoredEntries = append(scoredEntries, scored{entry: entry, score: score})
	}
	sort.Slice(scoredEntries, func(i, j int) bool {
		if scoredEntries[i].score == scoredEntries[j].score {
			return scoredEntries[i].entry.CreatedAt.After(scoredEntries[j].entry.CreatedAt)
		}
		return scoredEntries[i].score > scoredEntries[j].score
	})
	if len(scoredEntries) > limit {
		scoredEntries = scoredEntries[:limit]
	}
	out := make([]KnowledgeEntry, 0, len(scoredEntries))
	for _, item := range scoredEntries {
		out = append(out, item.entry)
	}
	return out, nil
}

// BuildAugmentedPrompt 构建增强提示词，将知识条目注入用户查询。
func BuildAugmentedPrompt(query string, entries []KnowledgeEntry) string {
	query = strings.TrimSpace(query)
	if len(entries) == 0 {
		return query
	}
	var b strings.Builder
	b.WriteString("[Knowledge Context]\n")
	for _, entry := range entries {
		b.WriteString("- Q: ")
		b.WriteString(entry.Question)
		b.WriteString("\n  A: ")
		b.WriteString(entry.Answer)
		b.WriteString("\n")
	}
	b.WriteString("\n[User Query]\n")
	b.WriteString(query)
	return b.String()
}

// matchScore 计算查询与条目的匹配分数。
//
// 问题匹配权重为 2，答案匹配权重为 1。
func matchScore(query string, entry KnowledgeEntry) int {
	score := 0
	for _, token := range strings.Fields(query) {
		if token == "" {
			continue
		}
		if strings.Contains(strings.ToLower(entry.Question), token) {
			score += 2
		}
		if strings.Contains(strings.ToLower(entry.Answer), token) {
			score++
		}
	}
	return score
}
