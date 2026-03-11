// Package rag 提供 RAG（检索增强生成）相关的知识索引功能。
//
// 本文件实现知识条目的索引、存储和查询功能。
package rag

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// KnowledgeSource 知识来源类型。
type KnowledgeSource string

const (
	SourceUserInput KnowledgeSource = "user_input" // 用户输入
	SourceFeedback  KnowledgeSource = "feedback"   // 反馈收集
)

// KnowledgeEntry 知识条目结构。
type KnowledgeEntry struct {
	ID        string          `json:"id"`                  // 条目 ID
	Source    KnowledgeSource `json:"source"`              // 来源类型
	Namespace string          `json:"namespace"`           // 命名空间（隔离不同场景）
	Question  string          `json:"question"`            // 问题
	Answer    string          `json:"answer"`              // 答案
	CreatedAt time.Time       `json:"created_at"`          // 创建时间
}

// Indexer 知识索引器接口。
type Indexer interface {
	// Index 批量索引知识条目
	Index(ctx context.Context, entries []KnowledgeEntry) error
	// AddUserKnowledge 添加用户知识
	AddUserKnowledge(ctx context.Context, namespace, question, answer string) (KnowledgeEntry, error)
	// List 列出指定命名空间的知识条目
	List(ctx context.Context, namespace string) ([]KnowledgeEntry, error)
}

// MilvusBackend Milvus 向量数据库后端接口。
type MilvusBackend interface {
	Upsert(ctx context.Context, entries []KnowledgeEntry) error
}

// MilvusIndexer 基于 Milvus 的知识索引器实现。
type MilvusIndexer struct {
	backend MilvusBackend        // Milvus 后端
	nowFn   func() time.Time     // 时间函数（便于测试）

	mu      sync.RWMutex         // 读写锁
	entries map[string][]KnowledgeEntry // 内存缓存
}

// NewMilvusIndexer 创建新的 Milvus 索引器。
func NewMilvusIndexer(backend MilvusBackend) *MilvusIndexer {
	return &MilvusIndexer{
		backend: backend,
		nowFn:   time.Now,
		entries: make(map[string][]KnowledgeEntry),
	}
}

// Index 批量索引知识条目。
func (i *MilvusIndexer) Index(ctx context.Context, entries []KnowledgeEntry) error {
	if i == nil {
		return fmt.Errorf("indexer is nil")
	}
	// 预处理和验证条目
	prepared := make([]KnowledgeEntry, 0, len(entries))
	for idx, entry := range entries {
		entry.Namespace = strings.TrimSpace(entry.Namespace)
		if entry.Namespace == "" {
			return fmt.Errorf("knowledge entry namespace is required")
		}
		entry.Question = strings.TrimSpace(entry.Question)
		entry.Answer = strings.TrimSpace(entry.Answer)
		if entry.Question == "" && entry.Answer == "" {
			return fmt.Errorf("knowledge entry content is empty")
		}
		if strings.TrimSpace(entry.ID) == "" {
			entry.ID = fmt.Sprintf("%s-%d", entry.Namespace, idx+1)
		}
		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = i.nowFn()
		}
		prepared = append(prepared, entry)
	}
	if len(prepared) == 0 {
		return nil
	}
	// 写入 Milvus 后端
	if i.backend != nil {
		if err := i.backend.Upsert(ctx, prepared); err != nil {
			return err
		}
	}
	// 更新内存缓存
	i.mu.Lock()
	defer i.mu.Unlock()
	for _, entry := range prepared {
		items := append(i.entries[entry.Namespace], entry)
		sort.Slice(items, func(a, b int) bool { return items[a].CreatedAt.After(items[b].CreatedAt) })
		i.entries[entry.Namespace] = items
	}
	return nil
}

// AddUserKnowledge 添加用户知识条目。
func (i *MilvusIndexer) AddUserKnowledge(ctx context.Context, namespace, question, answer string) (KnowledgeEntry, error) {
	entry := KnowledgeEntry{
		ID:        fmt.Sprintf("user-%d", i.nowFn().UnixNano()),
		Source:    SourceUserInput,
		Namespace: strings.TrimSpace(namespace),
		Question:  strings.TrimSpace(question),
		Answer:    strings.TrimSpace(answer),
		CreatedAt: i.nowFn(),
	}
	return entry, i.Index(ctx, []KnowledgeEntry{entry})
}

// List 列出指定命名空间的知识条目。
func (i *MilvusIndexer) List(_ context.Context, namespace string) ([]KnowledgeEntry, error) {
	if i == nil {
		return nil, fmt.Errorf("indexer is nil")
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	items := append([]KnowledgeEntry(nil), i.entries[strings.TrimSpace(namespace)]...)
	return items, nil
}
