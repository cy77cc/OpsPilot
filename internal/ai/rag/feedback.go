// Package rag 提供 RAG（检索增强生成）相关的功能实现。
//
// 本文件实现反馈收集和 QA 提取功能，用于从会话中提取知识并索引。
package rag

import (
	"context"
	"fmt"
	"strings"

	aistate "github.com/cy77cc/OpsPilot/internal/ai/state"
)

// Feedback 用户反馈结构。
type Feedback struct {
	IsEffective bool   `json:"is_effective"`    // 是否有效反馈
	Comment     string `json:"comment,omitempty"` // 反馈评论
}

// QAExtractor QA 提取器接口，从会话中提取问答对。
type QAExtractor interface {
	Extract(ctx context.Context, sessionID string) (KnowledgeEntry, error)
}

// FeedbackCollector 反馈收集器接口。
type FeedbackCollector interface {
	Collect(ctx context.Context, sessionID, namespace string, feedback Feedback) (*KnowledgeEntry, error)
}

// SessionFeedbackCollector 会话反馈收集器，基于会话提取知识。
type SessionFeedbackCollector struct {
	indexer   Indexer     // 知识索引器
	extractor QAExtractor // QA 提取器
}

// NewFeedbackCollector 创建新的反馈收集器。
func NewFeedbackCollector(indexer Indexer, extractor QAExtractor) *SessionFeedbackCollector {
	return &SessionFeedbackCollector{indexer: indexer, extractor: extractor}
}

// Collect 收集反馈并提取知识条目。
func (c *SessionFeedbackCollector) Collect(ctx context.Context, sessionID, namespace string, feedback Feedback) (*KnowledgeEntry, error) {
	if c == nil || c.indexer == nil || c.extractor == nil {
		return nil, fmt.Errorf("feedback collector is not initialized")
	}
	if !feedback.IsEffective {
		return nil, nil
	}
	entry, err := c.extractor.Extract(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, err
	}
	entry.Source = SourceFeedback
	entry.Namespace = strings.TrimSpace(namespace)
	if err := c.indexer.Index(ctx, []KnowledgeEntry{entry}); err != nil {
		return nil, err
	}
	return &entry, nil
}

// SessionSnapshotLoader 会话快照加载器接口。
type SessionSnapshotLoader interface {
	Load(ctx context.Context, sessionID string) (*aistate.SessionSnapshot, error)
}

// SessionQAExtractor 会话 QA 提取器，从会话消息中提取问答对。
type SessionQAExtractor struct {
	loader SessionSnapshotLoader
}

// NewSessionQAExtractor 创建新的会话 QA 提取器。
func NewSessionQAExtractor(loader SessionSnapshotLoader) *SessionQAExtractor {
	return &SessionQAExtractor{loader: loader}
}

// Extract 从会话中提取最新的问答对。
func (e *SessionQAExtractor) Extract(ctx context.Context, sessionID string) (KnowledgeEntry, error) {
	if e == nil || e.loader == nil {
		return KnowledgeEntry{}, fmt.Errorf("session qa extractor is not initialized")
	}
	snapshot, err := e.loader.Load(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return KnowledgeEntry{}, err
	}
	if snapshot == nil || len(snapshot.Messages) == 0 {
		return KnowledgeEntry{}, fmt.Errorf("session snapshot not found")
	}
	// 从后向前遍历，提取最近的问答对
	var question, answer string
	for i := len(snapshot.Messages) - 1; i >= 0; i-- {
		msg := snapshot.Messages[i]
		switch msg.Role {
		case "assistant":
			if answer == "" {
				answer = strings.TrimSpace(msg.Content)
			}
		case "user":
			if question == "" {
				question = strings.TrimSpace(msg.Content)
			}
		}
		if question != "" && answer != "" {
			break
		}
	}
	if question == "" || answer == "" {
		return KnowledgeEntry{}, fmt.Errorf("session does not contain a complete qa pair")
	}
	return KnowledgeEntry{Question: question, Answer: answer}, nil
}
