// Package runtime 提供 AI 运行时的 delta 缓冲组件。
//
// DeltaBuffer 累积 delta 内容并批量发送，减少前端刷新频率。
package runtime

import (
	"strings"
	"sync"
	"time"
)

// DeltaBufferConfig 缓冲配置。
type DeltaBufferConfig struct {
	MinChunkSize int // 最小累积字符数，默认 50
	MaxWaitMs    int // 最大等待毫秒数，默认 100
}

// DeltaBuffer 累积 delta 内容并批量发送。
type DeltaBuffer struct {
	config     DeltaBufferConfig
	mu         sync.Mutex
	content    strings.Builder
	agent      string
	lastAppend time.Time
}

// NewDeltaBuffer 创建 DeltaBuffer 实例。
func NewDeltaBuffer(config DeltaBufferConfig) *DeltaBuffer {
	if config.MinChunkSize <= 0 {
		config.MinChunkSize = 50
	}
	if config.MaxWaitMs <= 0 {
		config.MaxWaitMs = 100
	}
	return &DeltaBuffer{
		config: config,
	}
}

// Append 添加 delta 内容。
// 返回值：需要立即发送的事件（达到 MinChunkSize 阈值时）。
func (b *DeltaBuffer) Append(content, agent string) []PublicStreamEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.content.WriteString(content)
	if b.agent == "" {
		b.agent = agent
	}
	b.lastAppend = time.Now()

	// 只有达到 MinChunkSize 才立即发送
	if b.content.Len() >= b.config.MinChunkSize {
		return b.flush()
	}
	return nil
}

// ShouldFlushByTime 检查是否因超时需要刷新。
func (b *DeltaBuffer) ShouldFlushByTime() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.content.Len() == 0 {
		return false
	}
	elapsed := time.Since(b.lastAppend).Milliseconds()
	return elapsed >= int64(b.config.MaxWaitMs)
}

// Flush 强制刷新剩余内容。
func (b *DeltaBuffer) Flush() []PublicStreamEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.flush()
}

func (b *DeltaBuffer) flush() []PublicStreamEvent {
	content := b.content.String()
	if content == "" {
		return nil
	}

	event := PublicStreamEvent{
		Event: "delta",
		Data: map[string]any{
			"content": content,
			"agent":   b.agent,
		},
	}

	b.content.Reset()
	b.agent = ""

	return []PublicStreamEvent{event}
}
